// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/liaogx/xiaohongshu-mcp/humanize"
	"github.com/sirupsen/logrus"
)

// PreparedComment has filled and checked a draft, but has NOT clicked submit.
type PreparedComment struct {
	action                  *CommentFeedAction
	feedID, content, userID string
	Ready                   NoteReadiness
}

func (f *CommentFeedAction) PrepareComment(ctx context.Context, feedID, xsecToken, content string) (*PreparedComment, error) {
	if strings.TrimSpace(content) == "" {
		return nil, &InteractionError{Stage: "validate", State: "not_sent", Cause: fmt.Errorf("评论内容不能为空")}
	}
	state, err := prepareNote(ctx, f.page, feedID, xsecToken, true, true)
	if err != nil {
		return nil, err
	}
	// Separate phase deadlines: slow navigation cannot exhaust typing/submit.
	inputCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p := f.page.Context(inputCtx)
	// A contenteditable can exist and accept focus while the editor is still
	// collapsed. Activate the editor with its normal click handler first.
	editor, err := p.Timeout(8 * time.Second).Element("div.input-box div.content-edit")
	if err != nil {
		return nil, &InteractionError{Stage: "editor", State: "not_sent", Cause: err}
	}
	if err = editor.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, &InteractionError{Stage: "editor_activate", State: "not_sent", Cause: err}
	}
	input, err := p.Timeout(time.Second).Element("div.input-box div.content-edit p.content-input")
	if err == nil {
		if visible, e := input.Visible(); e != nil || !visible {
			err = fmt.Errorf("input not visible")
		}
	}
	if err != nil {
		placeholder, e := p.Timeout(8 * time.Second).Element("div.input-box div.content-edit span")
		if e != nil {
			return nil, &InteractionError{Stage: "input", State: "not_sent", Cause: e}
		}
		if e = humanize.Click(placeholder); e != nil {
			return nil, &InteractionError{Stage: "focus", State: "not_sent", Cause: e}
		}
		input, err = p.Timeout(8 * time.Second).Element("div.input-box div.content-edit p.content-input")
	}
	if err != nil {
		return nil, &InteractionError{Stage: "input", State: "not_sent", Cause: err}
	}
	input = input.Context(inputCtx)
	if err = humanize.Type(inputCtx, input, content); err != nil {
		return nil, &InteractionError{Stage: "type", State: "not_sent", Cause: err}
	}
	match, err := input.Eval(`(text)=>this.innerText.replace(/\u00a0/g,' ').trim()===text.trim()`, content)
	if err != nil {
		return nil, &InteractionError{Stage: "draft_check", State: "not_sent", Cause: err}
	}
	if !match.Value.Bool() {
		return nil, &InteractionError{Stage: "draft_check", State: "not_sent", Cause: fmt.Errorf("输入内容与拟发评论不一致，未提交")}
	}
	if _, err = commentSubmitButton(p.Timeout(10 * time.Second)); err != nil {
		return nil, err
	}
	return &PreparedComment{action: f, feedID: feedID, content: content, userID: state.UserID, Ready: *state}, nil
}

func (p *PreparedComment) Submit(ctx context.Context) error {
	f := p.action
	state, err := prepareNote(ctx, f.page, p.feedID, "", true, false)
	if err != nil {
		return err
	}
	if state.UserID != p.userID {
		return &InteractionError{Stage: "account_check", State: "not_sent", Cause: fmt.Errorf("当前账号发生变化，未提交")}
	}
	clickCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	page := f.page.Context(clickCtx)
	button, err := commentSubmitButton(page)
	if err != nil {
		return err
	}
	check, err := page.Eval(`text=>document.querySelector('div.input-box div.content-edit p.content-input')?.innerText.replace(/\u00a0/g,' ').trim()===text.trim()`, p.content)
	if err != nil || !check.Value.Bool() {
		return &InteractionError{Stage: "draft_check", State: "not_sent", Cause: fmt.Errorf("提交前草稿已改变，未发送")}
	}
	observer := observeSubmission(f.page.Context(ctx), "/comment/post", p.feedID, p.content)
	defer observer.close()
	if observer.setupErr != nil {
		return &InteractionError{Stage: "response_observer", State: "not_sent", Cause: observer.setupErr}
	}
	receiptPath := commentReceiptPath(p.userID, p.feedID, p.content)
	if err = beginCommentReceipt(receiptPath); err != nil {
		return &InteractionError{Stage: "deduplicate", State: "not_sent", Cause: err}
	}
	result := interactionReceipt{State: "unknown"}
	defer func() {
		if e := finishCommentReceipt(receiptPath, result); e != nil {
			logrus.WithField("feed_id", p.feedID).Error("评论回执写入失败，保留原待确认记录")
		}
	}()
	if err = button.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return &InteractionError{Stage: "click", State: "unknown", Cause: err}
	}
	result, err = awaitSubmission(f.page.Context(ctx), observer, 20*time.Second)
	f.CommentID = result.CommentID
	return err
}

// Retry ONLY activating the draft editor, never the submit click. The editable
// remains present even when collapsed; its send button can sit under the modal
// mask. Do not force-click through the mask or modify page CSS.
func commentSubmitButton(page *rod.Page) (*rod.Element, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if probe, err := readSearchPageProbe(page.Timeout(time.Second)); err == nil {
			if err := probe.accessError(); err != nil {
				return nil, err
			}
		}
		button, err := page.Element("div.bottom button.submit")
		if err != nil {
			return nil, &InteractionError{Stage: "submit_button", State: "not_sent", Cause: err}
		}
		if err = button.WaitEnabled(); err != nil {
			return nil, &InteractionError{Stage: "submit_enabled", State: "not_sent", Cause: err}
		}
		if _, err = button.Interactable(); err == nil {
			return button, nil
		}
		if attempt == 2 {
			break
		}
		input, err := page.Element("div.input-box div.content-edit p.content-input")
		if err != nil {
			return nil, &InteractionError{Stage: "editor_reactivate", State: "not_sent", Cause: err}
		}
		if err = input.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, &InteractionError{Stage: "editor_reactivate", State: "not_sent", Cause: err}
		}
		select {
		case <-page.GetContext().Done():
			return nil, page.GetContext().Err()
		case <-time.After(350 * time.Millisecond):
		}
	}
	// Do not include a Rod CoveredError dump: it can contain private DOM text.
	return nil, &InteractionError{Stage: "submit_visible", State: "not_sent", Cause: fmt.Errorf("发送按钮仍被遮挡；未点击发送")}
}
