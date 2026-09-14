// Account cleanup operates through the website UI and reports coverage explicitly.
// It does not replay private APIs or treat inaccessible lists as empty.
package xiaohongshu

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

type CleanupScope string

const (
	CleanupNotes            CleanupScope = "notes"
	CleanupSentComments     CleanupScope = "sent_comments"
	CleanupFavorites        CleanupScope = "favorites"
	CleanupLikes            CleanupScope = "likes"
	CleanupReceivedComments CleanupScope = "received_comments"
	CleanupGroups           CleanupScope = "groups"
	CleanupFollowing        CleanupScope = "following"
	CleanupMessages         CleanupScope = "messages"
)

var CleanupScopes = []CleanupScope{CleanupNotes, CleanupSentComments, CleanupFavorites, CleanupLikes, CleanupReceivedComments, CleanupGroups, CleanupFollowing, CleanupMessages}

// Tokens are private execution inputs, never included in a public plan/report.
type CleanupTarget struct {
	Scope          CleanupScope `json:"scope" jsonschema:"notes|sent_comments|favorites|likes|received_comments|groups|following|messages"`
	FeedID         string       `json:"feed_id,omitempty"`
	CommentID      string       `json:"comment_id,omitempty"`
	UserID         string       `json:"user_id,omitempty"`
	ConversationID string       `json:"conversation_id,omitempty"`
	XsecToken      string       `json:"xsec_token,omitempty"`
	Title          string       `json:"title,omitempty"`
}

func (t CleanupTarget) ID() string {
	switch t.Scope {
	case CleanupSentComments, CleanupReceivedComments:
		return t.FeedID + "/" + t.CommentID
	case CleanupFollowing:
		return t.UserID
	case CleanupGroups, CleanupMessages:
		return t.ConversationID
	default:
		return t.FeedID
	}
}

var cleanupIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)

func (t CleanupTarget) Validate() error {
	if len(t.XsecToken) > 4096 || len(t.Title) > 4096 {
		return fmt.Errorf("INVALID_TARGET: target metadata exceeds limit")
	}
	valid := false
	for _, s := range CleanupScopes {
		if s == t.Scope {
			valid = true
		}
	}
	if !valid {
		return fmt.Errorf("INVALID_SCOPE: unknown cleanup scope")
	}
	ids := []string{t.FeedID}
	switch t.Scope {
	case CleanupSentComments, CleanupReceivedComments:
		ids = append(ids, t.CommentID)
	case CleanupFollowing:
		ids = []string{t.UserID}
	case CleanupGroups, CleanupMessages:
		ids = []string{t.ConversationID}
	}
	for _, id := range ids {
		if !cleanupIDPattern.MatchString(id) {
			return fmt.Errorf("INVALID_TARGET: missing or invalid target ID")
		}
	}
	return nil
}

type CleanupCoverage struct {
	Scope    CleanupScope `json:"scope"`
	Status   string       `json:"status"` // inventoried, partial, blocked, manual_required
	Count    int          `json:"count"`  // -1 = unknown, never silently zero
	Complete bool         `json:"complete"`
	Reason   string       `json:"reason,omitempty"`
}

type CleanupInventory struct {
	AccountID string            `json:"account_id"`
	Coverage  []CleanupCoverage `json:"coverage"`
	Targets   []CleanupTarget   `json:"-"`
}

type CleanupAction struct{ page *rod.Page }

func NewCleanupAction(page *rod.Page) *CleanupAction { return &CleanupAction{page: page} }

func cleanupPause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Only inspect platform chrome/errors here, never search private message text
// for error keywords (a chat message may legitimately contain those words).
func cleanupPageGuard(page *rod.Page) error {
	probe, err := readSearchPageProbe(page)
	if err != nil {
		return fmt.Errorf("PAGE_UNCONFIRMED: cannot inspect access state")
	}
	if err := probe.accessError(); err != nil {
		return err
	}
	r, err := page.Eval(`() => {
	 const visible=e=>e&&e.getClientRects().length&&getComputedStyle(e).visibility!=='hidden';
	 const roots=[...document.querySelectorAll('.access-wrapper,.error-wrapper,.blocked-wrapper,.reds-alert,.reds-toast,.d-message,.no-content')].filter(visible);
	 const text=roots.map(e=>e.innerText||'').join(' ');
	 if (/访问频繁|请求太频繁|操作频繁|300013/.test(text)) return 'RATE_LIMITED';
	 if (/该账号疑似存在风险|账号已被封禁|账号异常/.test(text)) return 'ACCOUNT_RESTRICTED';
	 if (document.querySelector('.user-page') && /该账号疑似存在风险/.test(document.querySelector('.user-page').innerText)) return 'ACCOUNT_RESTRICTED';
	 return '';
	}`)
	if err != nil {
		return fmt.Errorf("PAGE_UNCONFIRMED: cannot inspect restriction")
	}
	if code := r.Value.Str(); code != "" {
		return fmt.Errorf("%s: 清理已暂停，页面受平台限制", code)
	}
	return nil
}

func CleanupErrorCode(err error) string {
	if err == nil {
		return ""
	}
	for _, c := range []string{"SECURITY_VERIFICATION_REQUIRED", "RATE_LIMITED", "AUTH_REQUIRED", "ACCOUNT_RESTRICTED", "ACCOUNT_MISMATCH", "MANUAL_REQUIRED", "TARGET_NOT_FOUND", "NOT_AUTHORIZED", "PAGE_UNCONFIRMED"} {
		if strings.Contains(err.Error(), c) {
			return c
		}
	}
	if strings.Contains(err.Error(), "code=300013") || strings.Contains(err.Error(), "reason=访问频繁") || strings.Contains(err.Error(), "reason=请求太频繁") || strings.Contains(err.Error(), "reason=操作频繁") {
		return "RATE_LIMITED"
	}
	return "CLEANUP_UNCONFIRMED"
}

func CleanupMustStop(err error) bool {
	switch CleanupErrorCode(err) {
	case "SECURITY_VERIFICATION_REQUIRED", "RATE_LIMITED", "AUTH_REQUIRED", "ACCOUNT_MISMATCH":
		return true
	}
	return false
}

func (a *CleanupAction) account(ctx context.Context) (string, error) {
	login := NewLogin(a.page)
	ok, err := login.CheckLoginStatus(ctx)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("AUTH_REQUIRED: 请先在 MCP 登录")
	}
	u, err := login.CurrentUser(ctx)
	if err != nil {
		return "", err
	}
	return u.UserID, nil
}

func (a *CleanupAction) Inventory(ctx context.Context, scopes []CleanupScope, maxItems int) (*CleanupInventory, error) {
	account, err := a.account(ctx)
	if err != nil {
		return nil, err
	}
	out := &CleanupInventory{AccountID: account, Coverage: []CleanupCoverage{}, Targets: []CleanupTarget{}}
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 200
	}
	for _, scope := range scopes {
		err = nil
		cov := CleanupCoverage{Scope: scope, Count: -1}
		var targets []CleanupTarget
		switch scope {
		case CleanupNotes:
			targets, cov.Complete, err = a.creatorNotes(ctx, account, maxItems)
		case CleanupFavorites, CleanupLikes:
			tab := TabFavorites
			if scope == CleanupLikes {
				tab = TabLiked
			}
			targets, cov.Complete, err = a.profileItems(ctx, account, tab, maxItems)
			if scope == CleanupFavorites && err == nil {
				cov.Complete, cov.Status = false, "partial"
				cov.Reason = "仅覆盖网页收藏标签可读笔记；未验证所有收藏夹的覆盖情况，不能声称全部取消收藏"
			}
		case CleanupSentComments:
			cov.Status = "manual_required"
			cov.Reason = "网页没有账号全部已发评论列表；可通过 discover_cleanup_comments 按已知笔记查找，覆盖范围不能视为全部历史"
		case CleanupReceivedComments:
			cov.Status = "manual_required"
			cov.Reason = "可删除自有笔记下的评论；其他笔记收到的回复不属于本账号，网页通知页没有已验证的删除通知入口"
		case CleanupFollowing:
			cov.Count, err = a.followingCount(ctx, account)
			cov.Status = "manual_required"
			cov.Reason = "网页个人主页未提供完整关注名单；已知 user_id 可单独准备取消关注"
		case CleanupGroups, CleanupMessages:
			targets, err = a.conversations(ctx, scope, maxItems)
			cov.Status = "manual_required"
			cov.Reason = "当前网页聊天没有已验证的清空/删除/退出入口，需手机 App 操作；清除浏览器本地缓存不能删除平台聊天记录"
		default:
			err = fmt.Errorf("INVALID_SCOPE: unknown cleanup scope")
		}
		if err != nil {
			cov.Status, cov.Reason = "blocked", CleanupErrorCode(err)
			out.Coverage = append(out.Coverage, cov)
			if CleanupMustStop(err) {
				return out, err
			}
			continue
		}
		if cov.Count < 0 && targets != nil {
			cov.Count = len(targets)
		}
		if cov.Status == "" {
			cov.Status = "inventoried"
			if !cov.Complete {
				cov.Status, cov.Reason = "partial", "列表未确认到末尾或达到读取上限；不能声称覆盖全部"
			}
		}
		if scope != CleanupGroups && scope != CleanupMessages {
			out.Targets = append(out.Targets, targets...)
		}
		out.Coverage = append(out.Coverage, cov)
	}
	return out, nil
}

func (a *CleanupAction) openProfile(ctx context.Context, account string) (*rod.Page, error) {
	p := a.page.Context(ctx).Timeout(40 * time.Second)
	if err := p.Navigate("https://www.xiaohongshu.com/user/profile/" + account); err != nil {
		return nil, err
	}
	// The profile header arrives before either tabs or an access error. Do
	// not mistake that intermediate skeleton for an empty/missing list.
	if err := p.Wait(rod.Eval(`() => !!document.querySelector('.reds-tab-item.sub-tab-list') || !!document.querySelector('.login-container')?.getClientRects().length || /该账号疑似存在风险|暂时无法查看笔记/.test(document.querySelector('.main-content')?.innerText||'') || /\/website-login\//.test(location.pathname)`)); err != nil {
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: profile did not load")
	}
	if err := cleanupPageGuard(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (a *CleanupAction) profileItems(ctx context.Context, account string, tab ProfileTab, limit int) ([]CleanupTarget, bool, error) {
	p, err := a.openProfile(ctx, account)
	if err != nil {
		return nil, false, err
	}
	restricted, err := p.Eval(`() => document.querySelector('.main-content')?.innerText.includes('该账号疑似存在风险')===true`)
	if err != nil {
		return nil, false, err
	}
	if restricted.Value.Bool() {
		return nil, false, fmt.Errorf("ACCOUNT_RESTRICTED: 个人主页隐藏了笔记标签，空数据不是清空成功")
	}
	if err := NewUserProfileAction(p).selectTab(ctx, p, tab); err != nil {
		return nil, false, fmt.Errorf("PAGE_UNCONFIRMED: requested profile tab unavailable")
	}
	scope := CleanupFavorites
	if tab == TabLiked {
		scope = CleanupLikes
	}
	items := []CleanupTarget{}
	seen := map[string]bool{}
	for round := 0; round < 40; round++ {
		if err := cleanupPageGuard(p); err != nil {
			return nil, false, err
		}
		var v struct {
			Current  string `json:"current"`
			Path     string `json:"path"`
			Query    string `json:"query"`
			Feeds    []Feed `json:"feeds"`
			HasMore  *bool  `json:"has_more"`
			Fetching bool   `json:"fetching"`
		}
		r, err := p.Eval(`() => {const u=window.__INITIAL_STATE__?.user,unwrap=x=>x?.value??x?._value??x;const tab=unwrap(u?.activeTab);const i=tab?.index;const q=unwrap(u?.noteQueries)?.[i];return {current:unwrap(u?.userInfo)?.userId,path:location.pathname,query:tab?.query,feeds:unwrap(u?.notes)?.[i]||[],has_more:typeof q?.hasMore==='boolean'?q.hasMore:null,fetching:!!unwrap(u?.isFetchingNotes)}}`)
		if err != nil || r.Value.Unmarshal(&v) != nil || v.Query != string(tab) {
			return nil, false, fmt.Errorf("PAGE_UNCONFIRMED: profile tab changed")
		}
		if v.Current != account || v.Path != "/user/profile/"+account {
			return nil, false, fmt.Errorf("ACCOUNT_MISMATCH: profile identity changed")
		}
		for _, f := range v.Feeds {
			if cleanupIDPattern.MatchString(f.ID) && !seen[f.ID] {
				seen[f.ID] = true
				items = append(items, CleanupTarget{Scope: scope, FeedID: f.ID, XsecToken: f.XsecToken, Title: f.NoteCard.DisplayTitle})
			}
		}
		if len(items) > limit {
			return items[:limit], false, nil
		}
		if v.HasMore != nil && !*v.HasMore && !v.Fetching {
			return items, true, nil
		}
		if len(items) >= limit {
			return items[:limit], false, nil
		}
		_, err = p.Eval(`() => {const e=document.querySelector('.main-content'); if(e&&e.scrollHeight>e.clientHeight)e.scrollTop=e.scrollHeight; window.scrollTo(0,document.body.scrollHeight);return true}`)
		if err != nil {
			return items, false, err
		}
		if err := cleanupPause(ctx, 1200*time.Millisecond); err != nil {
			return items, false, err
		}
	}
	return items, false, nil
}

func (a *CleanupAction) followingCount(ctx context.Context, account string) (int, error) {
	// The count may remain visible even when the profile's note lists are blocked.
	p := a.page.Context(ctx).Timeout(35 * time.Second)
	if err := p.Navigate("https://www.xiaohongshu.com/user/profile/" + account); err != nil {
		return -1, err
	}
	if err := p.Wait(rod.Eval(`()=>!!document.querySelector('.user-interactions')`)); err != nil {
		return -1, fmt.Errorf("PAGE_UNCONFIRMED: follow count unavailable")
	}
	r, err := p.Eval(`()=>{const e=[...document.querySelectorAll('.user-interactions>div')].find(e=>e.querySelector('.shows')?.innerText==='关注');const v=e?.querySelector('.count')?.innerText;return /^\d+$/.test(v||'')?Number(v):-1}`)
	if err != nil {
		return -1, err
	}
	return r.Value.Int(), nil
}

const creatorCleanupURL = "https://creator.xiaohongshu.com/new/note-manager"

// Tracker metadata belongs to the actual rendered card and carries noteId.
// Never match destructive targets by title, position, or an unscoped button.
const creatorCardsJS = `() => {
 const idOf=e=>{try { const find=(v)=>{if(!v||typeof v!=='object')return '';if(v.noteId)return typeof v.noteId==='string'?v.noteId:v.noteId.value||'';for(const x of Object.values(v)){const s=find(x);if(s)return s}return ''};return find(JSON.parse(e.getAttribute('data-impression')||'{}'))}catch{return ''}};
 const panel=document.querySelector('.notes-container'),label=panel?.querySelector('.tab-item--active')?.innerText||'';
 const all=/^全部\s*(\d+)$/.exec(label),query=panel?.querySelector('input')?.value;
 return {all:!!all,total:all?Number(all[1]):-1,search:query||'',empty:!!panel?.querySelector('.no-content')&&panel.querySelector('.no-content').getClientRects().length>0,
 cards:[...document.querySelectorAll('.notes-container .note-card')].map(e=>({id:idOf(e),title:e.querySelector('.note-card__title')?.innerText||''}))};
}`

type creatorSnapshot struct {
	All    bool   `json:"all"`
	Total  int    `json:"total"`
	Search string `json:"search"`
	Empty  bool   `json:"empty"`
	Cards  []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"cards"`
}

func (a *CleanupAction) creatorNotes(ctx context.Context, account string, limit int) ([]CleanupTarget, bool, error) {
	p := a.page.Context(ctx).Timeout(60 * time.Second)
	if err := p.Navigate(creatorCleanupURL); err != nil {
		return nil, false, err
	}
	if err := p.Wait(rod.Eval(`()=>!!document.querySelector('.notes-container .tab-item--active')`)); err != nil {
		return nil, false, fmt.Errorf("PAGE_UNCONFIRMED: creator manager unavailable or requires login")
	}
	identity, err := p.Eval(`()=>document.querySelector('#app')?.__vue_app__?.config?.globalProperties?.$store?.state?.Auth?.userInfo?.userId||''`)
	if err != nil || account == "" || identity.Value.Str() != account {
		return nil, false, fmt.Errorf("ACCOUNT_MISMATCH: creator identity does not match cleanup account")
	}
	items := []CleanupTarget{}
	seen := map[string]bool{}
	for round := 0; round < 35; round++ {
		if err := cleanupPageGuard(p); err != nil {
			return nil, false, err
		}
		r, err := p.Eval(creatorCardsJS)
		var v creatorSnapshot
		if err != nil || r.Value.Unmarshal(&v) != nil || !v.All || v.Search != "" {
			return nil, false, fmt.Errorf("PAGE_UNCONFIRMED: creator all-notes view not confirmed")
		}
		for _, card := range v.Cards {
			if !cleanupIDPattern.MatchString(card.ID) {
				return nil, false, fmt.Errorf("PAGE_UNCONFIRMED: note card has no reliable ID")
			}
			if !seen[card.ID] {
				seen[card.ID] = true
				items = append(items, CleanupTarget{Scope: CleanupNotes, FeedID: card.ID, Title: card.Title})
			}
		}
		if len(items) > limit {
			return items[:limit], false, nil
		}
		if v.Total == 0 && v.Empty || v.Total > 0 && len(items) == v.Total {
			return items, true, nil
		}
		if len(items) >= limit {
			return items[:limit], false, nil
		}
		_, err = p.Eval(`()=>{document.querySelector('#notes-request')?.scrollIntoView();return true}`)
		if err != nil {
			return items, false, err
		}
		if err := cleanupPause(ctx, time.Second); err != nil {
			return items, false, err
		}
	}
	return items, false, nil
}

func (a *CleanupAction) conversations(ctx context.Context, scope CleanupScope, limit int) ([]CleanupTarget, error) {
	p := a.page.Context(ctx).Timeout(35 * time.Second)
	if err := p.Navigate("https://www.xiaohongshu.com/chat"); err != nil {
		return nil, err
	}
	if err := p.Wait(rod.Eval(`()=>!!document.querySelector('.xhs-im-conv-list__scroll')`)); err != nil {
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: chat list unavailable")
	}
	if err := cleanupPageGuard(p); err != nil {
		return nil, err
	}
	// A mounted sidebar is only a loading shell. With no real rows and no
	// verified empty-list signal, report unknown rather than zero sessions.
	if err := p.Wait(rod.Eval(conversationRowsReadyJS)); err != nil {
		if gate := cleanupPageGuard(a.page.Context(ctx)); gate != nil {
			return nil, gate
		}
		return nil, fmt.Errorf("PAGE_UNCONFIRMED: chat rows have not loaded; zero conversations is not confirmed")
	}
	kind := "c2c"
	if scope == CleanupGroups {
		kind = "group"
	}
	r, err := p.Eval(`kind=>[...document.querySelectorAll('.xhs-im-conv-item[data-conv-id]')].filter(e=>e.dataset.convKind===kind).map(e=>({conversation_id:e.dataset.convId}))`, kind)
	if err != nil {
		return nil, err
	}
	var items []CleanupTarget
	if err = r.Value.Unmarshal(&items); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Scope = scope
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

const conversationRowsReadyJS = `()=>document.querySelectorAll('.xhs-im-conv-item[data-conv-id]').length>0`

type CleanupDiscovery struct {
	AccountID string          `json:"account_id"`
	Complete  bool            `json:"complete"`
	Targets   []CleanupTarget `json:"targets"`
	Reason    string          `json:"reason"`
}

// Discovers comments only on explicitly supplied notes. Complete is always
// false for account history: web does not expose an all-authored-comments list.
func (a *CleanupAction) DiscoverComments(ctx context.Context, feedID, token string, received bool, limit int) (*CleanupDiscovery, error) {
	if !cleanupIDPattern.MatchString(feedID) {
		return nil, fmt.Errorf("INVALID_TARGET")
	}
	account, err := a.account(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	detail, err := NewFeedDetailAction(a.page).GetFeedDetail(ctx, feedID, token, true, CommentLoadConfig{ClickMoreReplies: true, MaxRepliesThreshold: limit, MaxCommentItems: limit, ScrollSpeed: "slow"})
	if err != nil {
		return nil, err
	}
	if received && detail.Note.User.UserID != account {
		return nil, fmt.Errorf("NOT_AUTHORIZED: 只能删除自有笔记下收到的评论")
	}
	out := &CleanupDiscovery{AccountID: account, Targets: []CleanupTarget{}, Reason: "只覆盖所提供笔记已加载的评论；不是账号全部历史"}
	var walk func([]Comment)
	walk = func(cs []Comment) {
		for _, c := range cs {
			if c.UserInfo.UserID == account && !received || received && c.UserInfo.UserID != "" && c.UserInfo.UserID != account {
				scope := CleanupSentComments
				if received {
					scope = CleanupReceivedComments
				}
				out.Targets = append(out.Targets, CleanupTarget{Scope: scope, FeedID: feedID, CommentID: c.ID, Title: detail.Note.Title})
			}
			walk(c.SubComments)
		}
	}
	walk(detail.Comments.List)
	return out, nil
}
