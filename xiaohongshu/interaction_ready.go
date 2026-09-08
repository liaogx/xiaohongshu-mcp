// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

type NoteReadiness struct {
	NoteID       string `json:"note_id"`
	UserID       string `json:"user_id"`
	Liked        *bool  `json:"liked"`
	Collected    *bool  `json:"collected"`
	InputVisible bool   `json:"input_visible"`
	Blocked      bool   `json:"blocked"`
}

func readNoteReadiness(page *rod.Page, feedID string) (*NoteReadiness, error) {
	r, err := page.Eval(`(id) => {
		const unwrap = v => v?.value !== undefined ? v.value : v?._value !== undefined ? v._value : v;
		const s=window.__INITIAL_STATE__, note=unwrap(unwrap(s?.note?.noteDetailMap)?.[id])?.note;
		const user=unwrap(s?.user?.userInfo), info=note?.interactInfo;
		const visible=e=>!!e && e.getBoundingClientRect().height>0 && getComputedStyle(e).visibility!=='hidden';
		return {note_id:note?.noteId||note?.id||'', user_id:user && user.guest!==true?(user.userId||user.user_id||''):'',
			liked: typeof info?.liked==='boolean'?info.liked:null, collected:typeof info?.collected==='boolean'?info.collected:null,
			input_visible:[...document.querySelectorAll('div.input-box div.content-edit')].some(visible),
			blocked:[...document.querySelectorAll('.access-wrapper,.error-wrapper,.not-found-wrapper,.blocked-wrapper')].some(e=>visible(e)&&e.innerText.trim())};
	}`, feedID)
	if err != nil {
		return nil, err
	}
	var v NoteReadiness
	if err := r.Value.Unmarshal(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func waitNoteReady(page *rod.Page, feedID string, needsInput bool, timeout time.Duration) (*NoteReadiness, error) {
	ctx, cancel := context.WithTimeout(page.GetContext(), timeout)
	defer cancel()
	p := page.Context(ctx)
	var last *NoteReadiness
	for {
		if probe, err := readSearchPageProbe(p.Timeout(time.Second)); err == nil {
			if err := probe.accessError(); err != nil {
				return nil, err
			}
		}
		if state, err := readNoteReadiness(p.Timeout(time.Second), feedID); err == nil {
			last = state
			if state.Blocked {
				return nil, fmt.Errorf("NOTE_UNAVAILABLE: 页面明确显示笔记不可访问，未执行互动")
			}
			if state.NoteID == feedID && state.Liked != nil && state.Collected != nil && (!needsInput || state.InputVisible && state.UserID != "") {
				return state, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("NOTE_NOT_READY: 目标笔记数据/操作区域未就绪（目标数据=%t, 账号=%t, 输入区域=%t），未发送: %w", last != nil && last.NoteID == feedID, last != nil && last.UserID != "", last != nil && last.InputVisible, ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// Navigate at most once and wait for the target note, not global DOM stability.
func prepareNote(ctx context.Context, page *rod.Page, feedID, token string, needsInput, navigate bool) (*NoteReadiness, error) {
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	p := page.Context(ctx)
	if navigate {
		if err := p.Navigate(makeFeedDetailURL(feedID, token)); err != nil {
			return nil, &InteractionError{Stage: "navigate", State: "not_sent", Cause: err}
		}
	}
	state, err := waitNoteReady(p, feedID, needsInput, 25*time.Second)
	if err != nil {
		return nil, &InteractionError{Stage: "ready", State: "not_sent", Cause: err}
	}
	return state, nil
}
