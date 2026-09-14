package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/humanize"
)

// PreparedCleanup holds a verified target and a single dispatch button.
// Preparation may open a confirmation dialog, but cannot confirm it.
type PreparedCleanup struct {
	Target       CleanupTarget
	AlreadyClear bool
	page         *rod.Page
	button       *rod.Element
	endpoint     string
	match        func(string) bool
	dispatched   atomic.Bool
}

func cleanupRequestMatches(body string, fields map[string]string) bool {
	var v map[string]any
	if json.Unmarshal([]byte(body), &v) != nil || len(fields) == 0 {
		return false
	}
	for k, want := range fields {
		got, ok := v[k].(string)
		if !ok || want == "" || got != want {
			return false
		}
	}
	return true
}

// The web request adapter can serialize camelCase parameters as snake_case.
// Bind both IDs, and reject contradictory aliases rather than accepting an
// unrelated successful request merely because one pair of fields matches.
func cleanupCommentRequestMatches(body, feedID, commentID string) bool {
	var v map[string]any
	if feedID == "" || commentID == "" || json.Unmarshal([]byte(body), &v) != nil {
		return false
	}
	complete := false
	for _, pair := range [][2]string{{"noteId", "commentId"}, {"note_id", "comment_id"}} {
		note, hasNote := v[pair[0]]
		comment, hasComment := v[pair[1]]
		if hasNote && note != feedID || hasComment && comment != commentID {
			return false
		}
		complete = complete || hasNote && hasComment
	}
	return complete
}

func (a *CleanupAction) Prepare(ctx context.Context, account string, t CleanupTarget) (*PreparedCleanup, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if t.Scope == CleanupGroups || t.Scope == CleanupMessages {
		return nil, fmt.Errorf("MANUAL_REQUIRED: 当前网页无已验证的清空/删除/退出聊天入口")
	}
	if account == "" {
		return nil, fmt.Errorf("ACCOUNT_MISMATCH: 当前账号与清理计划不一致")
	}
	// Note scopes verify the authenticated account on the exact target page
	// below. Opening explore first adds a second, unrelated readiness failure.
	if t.Scope == CleanupNotes || t.Scope == CleanupFollowing {
		id, err := a.account(ctx)
		if err != nil {
			return nil, err
		}
		if id != account {
			return nil, fmt.Errorf("ACCOUNT_MISMATCH: 当前账号与清理计划不一致")
		}
	}
	p := a.page.Context(ctx).Timeout(100 * time.Second)
	out := &PreparedCleanup{Target: t, page: p}
	switch t.Scope {
	case CleanupNotes:
		if err := a.prepareDeleteNote(ctx, account, out); err != nil {
			return nil, err
		}
	case CleanupFollowing:
		if t.UserID == account {
			return nil, fmt.Errorf("INVALID_TARGET: cannot unfollow self")
		}
		if err := p.Navigate(makeUserProfileURL(t.UserID, t.XsecToken, TabNotes)); err != nil {
			return nil, err
		}
		if err := p.Wait(rod.Eval(`()=>!!document.querySelector('.user-info')`)); err != nil {
			return nil, fmt.Errorf("PAGE_UNCONFIRMED: user profile unavailable")
		}
		if err := cleanupPageGuard(p); err != nil {
			return nil, err
		}
		var v struct {
			Following *bool  `json:"following"`
			Current   string `json:"current"`
			Path      string `json:"path"`
		}
		r, err := p.Eval(`()=>{const u=window.__INITIAL_STATE__?.user,f=u?.follow?.value??u?.follow?._value,me=u?.userInfo?.value??u?.userInfo?._value;return {following:typeof f==='boolean'?f:null,current:me?.userId,path:location.pathname}}`)
		if err != nil || r.Value.Unmarshal(&v) != nil || v.Following == nil || v.Current != account || v.Path != "/user/profile/"+t.UserID {
			return nil, fmt.Errorf("PAGE_UNCONFIRMED: follow state or account not confirmed")
		}
		if !*v.Following {
			out.AlreadyClear = true
			break
		}
		out.button, err = p.ElementR(".user-info .follow-button.followed", `^(已关注|互相关注)$`)
		if err != nil {
			return nil, fmt.Errorf("MANUAL_REQUIRED: target has no verified unfollow control")
		}
		out.endpoint = "/user/unfollow"
		out.match = func(body string) bool {
			return cleanupRequestMatches(body, map[string]string{"targetUserId": t.UserID})
		}
	case CleanupFavorites, CleanupLikes, CleanupSentComments, CleanupReceivedComments:
		state, err := prepareNote(ctx, p, t.FeedID, t.XsecToken, false, true, true)
		if err != nil {
			return nil, err
		}
		if state.UserID != account {
			return nil, fmt.Errorf("ACCOUNT_MISMATCH: 笔记页登录账号发生变化")
		}
		if err := cleanupPageGuard(p); err != nil {
			return nil, err
		}
		switch t.Scope {
		case CleanupLikes:
			if state.Liked == nil {
				return nil, fmt.Errorf("PAGE_UNCONFIRMED: like state missing")
			}
			if !*state.Liked {
				out.AlreadyClear = true
				break
			}
			out.button, err = p.Element(SelectorLikeButton)
			out.endpoint = "/note/dislike"
			out.match = func(body string) bool { return submissionTargetField(body, t.FeedID) != "" }
		case CleanupFavorites:
			if state.Collected == nil {
				return nil, fmt.Errorf("PAGE_UNCONFIRMED: favorite state missing")
			}
			if !*state.Collected {
				out.AlreadyClear = true
				break
			}
			out.button, err = p.Element(SelectorCollectButton)
			out.endpoint = "/note/uncollect"
			out.match = func(body string) bool {
				return cleanupRequestMatches(body, map[string]string{"noteIds": t.FeedID}) || cleanupRequestMatches(body, map[string]string{"note_ids": t.FeedID})
			}
		default:
			err = a.prepareDeleteComment(ctx, account, out)
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *CleanupAction) prepareDeleteComment(ctx context.Context, account string, out *PreparedCleanup) error {
	p, t := out.page, out.Target
	// Exact comment IDs only. Author-name/text matching must not delete a
	// different reply, even if several comments have identical content.
	el, err := findCleanupCommentElement(ctx, p, t.CommentID)
	if err != nil {
		if CleanupMustStop(err) || CleanupErrorCode(err) == "ACCOUNT_RESTRICTED" {
			return err
		}
		return fmt.Errorf("TARGET_NOT_FOUND: 未找到指定评论；不等于已删除")
	}
	var proof struct {
		NoteOwner    string `json:"note_owner"`
		CommentOwner string `json:"comment_owner"`
		CommentID    string `json:"comment_id"`
	}
	r, err := p.Eval(`(feed,id)=>{const unwrap=x=>x?.value??x?._value??x;const d=unwrap(unwrap(window.__INITIAL_STATE__?.note?.noteDetailMap)?.[feed]);const walk=cs=>{for(const c of cs||[]){if(c.id===id)return c;const found=walk(c.subComments);if(found)return found}};const c=walk(d?.comments?.list);return {note_owner:d?.note?.user?.userId,comment_owner:c?.userInfo?.userId,comment_id:c?.id}}`, t.FeedID, t.CommentID)
	if err != nil || r.Value.Unmarshal(&proof) != nil {
		return fmt.Errorf("PAGE_UNCONFIRMED: cannot establish comment ownership")
	}
	if proof.CommentID != t.CommentID || proof.CommentOwner == "" {
		return fmt.Errorf("PAGE_UNCONFIRMED: target comment identity is not in page state")
	}
	if t.Scope == CleanupSentComments && proof.CommentOwner != account || t.Scope == CleanupReceivedComments && proof.NoteOwner != account {
		return fmt.Errorf("NOT_AUTHORIZED: 评论不属于本账号或不在本账号笔记下")
	}
	more, err := el.Timeout(5 * time.Second).Element(".more")
	if err != nil {
		return fmt.Errorf("MANUAL_REQUIRED: comment menu unavailable")
	}
	if err = humanize.Click(more); err != nil {
		return err
	}
	menu, err := p.Timeout(5*time.Second).ElementR(".menu-wrapper .menu-item", `^删除评论$`)
	if err != nil {
		return fmt.Errorf("NOT_AUTHORIZED: 页面未提供删除此评论入口")
	}
	if err = humanize.Click(menu); err != nil {
		return err
	}
	out.button, err = cleanupCommentConfirm(p)
	out.endpoint = "/comment/delete"
	out.match = func(body string) bool {
		return cleanupCommentRequestMatches(body, t.FeedID, t.CommentID)
	}
	return err
}

func cleanupCommentConfirm(p *rod.Page) (*rod.Element, error) {
	dialog, err := p.Timeout(5*time.Second).ElementR(".reds-alert", `确认删除此评论`)
	if err != nil {
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: deletion confirmation missing")
	}
	// The website renders this control as a div, not a <button>. Scope it
	// to the verified deletion dialog and never click a generic “确定”.
	button, err := dialog.ElementR(".reds-alert-footer .foot-btn.strong", `^确定$`)
	if err != nil {
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: comment deletion confirm control unavailable")
	}
	// The dialog can still be animating when the control becomes visible.
	// Resolve its final position before pointer movement; a stale point can
	// dismiss the dialog without sending a deletion request.
	stable := button.Timeout(3 * time.Second)
	err = stable.Wait(rod.Eval(`function(){
		for(let node=this;node;node=node.parentElement){
			for(const animation of node.getAnimations?.()||[]){
				if(animation.pending||animation.playState==='running')return false;
			}
		}
		return true;
	}`))
	if err == nil {
		err = stable.WaitStable(200 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: comment deletion confirmation did not settle")
	}
	return button, nil
}

func (a *CleanupAction) prepareDeleteNote(ctx context.Context, account string, out *PreparedCleanup) error {
	items, complete, err := a.creatorNotes(ctx, account, 1000)
	if err != nil {
		return err
	}
	p := out.page
	r, err := p.Eval(`()=>document.querySelector('#app')?.__vue_app__?.config?.globalProperties?.$store?.state?.Auth?.userInfo?.userId||''`)
	if err != nil || r.Value.Str() != account {
		return fmt.Errorf("ACCOUNT_MISMATCH: creator identity does not match cleanup account")
	}
	found := false
	for _, t := range items {
		if t.FeedID == out.Target.FeedID {
			found = true
			out.Target.Title = t.Title
		}
	}
	if !found {
		if complete {
			out.AlreadyClear = true
			return nil
		}
		return fmt.Errorf("TARGET_NOT_FOUND: manager list is incomplete")
	}
	// Bind to the card's platform-provided noteId, never a title substring.
	r, err = p.Eval(`id=>{const find=v=>{if(!v||typeof v!=='object')return '';if(v.noteId)return typeof v.noteId==='string'?v.noteId:v.noteId.value||'';for(const x of Object.values(v)){const r=find(x);if(r)return r}return ''};let matches=[];for(const e of document.querySelectorAll('.notes-container .note-card')){try{if(find(JSON.parse(e.getAttribute('data-impression')||'{}'))===id)matches.push(e)}catch{}}if(matches.length!==1)return false;matches[0].setAttribute('data-cleanup-target',id);return true}`, out.Target.FeedID)
	if err != nil || !r.Value.Bool() {
		return fmt.Errorf("PAGE_UNCONFIRMED: note card target not unique")
	}
	card, err := p.Element(`[data-cleanup-target="` + out.Target.FeedID + `"]`)
	if err != nil {
		return err
	}
	if err = card.Hover(); err != nil {
		return err
	}
	button, err := card.Timeout(5 * time.Second).Element(".note-card__action-btn--del:not(.note-card__action-btn--disabled)")
	if err != nil {
		return fmt.Errorf("NOT_AUTHORIZED: creator does not allow deletion of this note")
	}
	if err = humanize.Click(button); err != nil {
		return err
	}
	dialog, err := p.Timeout(8*time.Second).ElementR(".modal-container", `删除后将无法恢复`)
	if err != nil {
		return fmt.Errorf("PAGE_UNCONFIRMED: permanent deletion confirmation not available")
	}
	out.button, err = dialog.ElementR(".confirm-button", `^确定$`)
	out.endpoint = "/note/delete"
	out.match = func(body string) bool {
		return cleanupRequestMatches(body, map[string]string{"note_id": out.Target.FeedID})
	}
	return err
}

// Execute is deliberately single-shot. The caller must persist an unknown
// receipt BEFORE calling it so disconnects and process crashes cannot resend.
func (p *PreparedCleanup) Execute(ctx context.Context) error {
	if p.AlreadyClear {
		return nil
	}
	if p.button == nil || p.match == nil || p.endpoint == "" {
		return fmt.Errorf("MANUAL_REQUIRED: no verified cleanup action")
	}
	if err := ctx.Err(); err != nil {
		return &InteractionError{Stage: "cleanup_context", State: "not_sent", Cause: err}
	}
	if !p.dispatched.CompareAndSwap(false, true) {
		return &InteractionError{Stage: "cleanup_reuse", State: "not_sent", Cause: fmt.Errorf("already dispatched; inspect the first result")}
	}
	page := p.page.Context(ctx)
	if err := cleanupPageGuard(page); err != nil {
		return &InteractionError{Stage: "cleanup_access", State: "not_sent", Cause: err}
	}
	observer := observeSubmissionMatching(page, p.endpoint, p.Target.FeedID, "", p.match)
	defer observer.close()
	if observer.setupErr != nil {
		return &InteractionError{Stage: "cleanup_observer", State: "not_sent", Cause: observer.setupErr}
	}
	if err := humanize.Click(p.button.Context(ctx)); err != nil {
		return &InteractionError{Stage: "cleanup_click", State: "unknown", Cause: err}
	}
	_, err := awaitSubmission(page, observer, 20*time.Second)
	return err
}

func CleanupResultState(err error) string {
	if err == nil {
		return "confirmed"
	}
	if strings.Contains(err.Error(), "INTERACTION_REJECTED") {
		return "rejected"
	}
	if strings.Contains(err.Error(), "INTERACTION_NOT_SENT") {
		return "not_sent"
	}
	return "unknown"
}
