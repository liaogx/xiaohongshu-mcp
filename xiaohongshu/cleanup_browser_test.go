//go:build browser

package xiaohongshu

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/stretchr/testify/require"
)

func TestCleanupBrowserFixtures(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	t.Run("exact ID lookup does not wait for absent rows", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="parent-comment"><div class="comment-item" id="comment-fixture-target"></div><button class="show-more" onclick="window.expanded=true">展开</button></div>`))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		probe := p.Context(ctx)
		require.Nil(t, lookupComment(probe, "fixture-missing", ""))
		require.False(t, checkEndContainer(probe))
		require.NotNil(t, lookupComment(probe, "fixture-target", ""))
		require.NoError(t, ctx.Err(), "an absent row must not consume the polling budget")
		el, err := findCleanupCommentElement(context.Background(), p, "fixture-target")
		require.NoError(t, err)
		require.NotNil(t, el)
		r, err := p.Eval(`()=>!!window.expanded`)
		require.NoError(t, err)
		require.False(t, r.Value.Bool(), "do not expand unrelated replies before checking the exact ID")
	})
	t.Run("restricted profile is not zero notes", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="main-content">该账号疑似存在风险，暂时无法查看笔记。</div>`))
		require.ErrorContains(t, checkProfileReadable(p), "ACCOUNT_RESTRICTED")
	})
	t.Run("creator zero requires all tab and empty marker", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="notes-container"><div class="tab-item--active">全部 0</div><input value=""><div class="no-content">没有找到相关笔记</div></div>`))
		r, err := p.Eval(creatorCardsJS)
		require.NoError(t, err)
		var v creatorSnapshot
		require.NoError(t, r.Value.Unmarshal(&v))
		require.True(t, v.All)
		require.True(t, v.Empty)
		require.Zero(t, v.Total)
		require.NoError(t, p.SetDocumentContent(`<div class="notes-container"><div class="tab-item--active">已发布 0</div><input value="filtered"></div>`))
		r, err = p.Eval(creatorCardsJS)
		require.NoError(t, err)
		require.NoError(t, r.Value.Unmarshal(&v))
		require.False(t, v.All)
		require.False(t, v.Empty)
	})
	t.Run("rate marker in a chat is not a restriction", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="xhs-im-message">访问频繁，请稍后再试</div>`))
		require.NoError(t, cleanupPageGuard(p))
		require.NoError(t, p.SetDocumentContent(`<div class="access-wrapper">访问频繁，请稍后再试 300013</div>`))
		require.ErrorContains(t, cleanupPageGuard(p), "RATE_LIMITED")
	})
	t.Run("chat skeleton is not zero conversations", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="xhs-im-conv-list__scroll"></div>`))
		r, err := p.Eval(conversationRowsReadyJS, "c2c")
		require.NoError(t, err)
		require.False(t, r.Value.Bool())
		require.NoError(t, p.SetDocumentContent(`<div class="xhs-im-conv-item" data-conv-id="fixture-group" data-conv-kind="group"></div>`))
		r, err = p.Eval(conversationRowsReadyJS, "c2c")
		require.NoError(t, err)
		require.False(t, r.Value.Bool(), "a group batch is not a loaded direct-chat list")
		require.NoError(t, p.SetDocumentContent(`<div class="xhs-im-conv-item" data-conv-id="fixture-conversation" data-conv-kind="c2c"></div>`))
		r, err = p.Eval(conversationRowsReadyJS, "c2c")
		require.NoError(t, err)
		require.True(t, r.Value.Bool())
	})
	t.Run("moving deletion confirmation must settle before preparation finishes", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="reds-alert"><div>确认删除此评论？</div><div class="reds-alert-footer"><div id="confirm" class="foot-btn strong" style="position:fixed;top:350px;left:100px;width:300px;height:40px" onclick="window.clicks++">确定</div></div></div><script>window.clicks=0;window.settled=false;document.querySelector('#confirm').animate([{transform:'translateY(-200px)'},{transform:'translateY(0px)'}],{duration:600,fill:'forwards'}).finished.then(()=>window.settled=true)</script>`))
		_, err := cleanupCommentConfirm(p)
		require.NoError(t, err)
		r, err := p.Eval(`()=>({settled:window.settled,clicks:window.clicks})`)
		require.NoError(t, err)
		require.True(t, r.Value.Get("settled").Bool(), "a click point captured during dialog motion can miss after pointer movement")
		require.Zero(t, r.Value.Get("clicks").Int(), "readiness checks must not dispatch deletion")
	})
	for _, tc := range []struct {
		name, body, request string
		ok                  bool
	}{
		{"confirmed camelCase", `{"success":true,"code":0}`, `{noteId:'fixture-note',commentId:'fixture-comment'}`, true},
		{"confirmed snake_case", `{"success":true,"code":0}`, `{note_id:'fixture-note',comment_id:'fixture-comment'}`, true},
		{"HTTP200 business rejection", `{"success":false,"code":-1}`, `{note_id:'fixture-note',comment_id:'fixture-comment'}`, false},
		{"unrecognized response", `{}`, `{note_id:'fixture-note',comment_id:'fixture-comment'}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					w.Header().Set("Content-Type", "application/json")
					calls.Add(1)
					fmt.Fprint(w, tc.body)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, `<button onclick="throw Error('unrelated confirmation')">确定</button><div class="reds-alert"><div class="reds-alert-title">确认删除此评论？</div><div class="reds-alert-footer"><div class="foot-btn strong" id="confirm" onclick="fetch('/comment/delete',{method:'POST',body:JSON.stringify(%s)})">确定</div><div class="foot-btn">取消</div></div></div>`, tc.request)
			}))
			defer srv.Close()
			p := b.NewPage()
			defer p.Close()
			require.NoError(t, p.Navigate(srv.URL))
			require.NoError(t, p.Timeout(5*time.Second).Wait(rod.Eval(`()=>!!document.querySelector('#confirm')`)))
			btn, err := cleanupCommentConfirm(p)
			require.NoError(t, err)
			prepared := PreparedCleanup{Target: CleanupTarget{Scope: CleanupSentComments, FeedID: "fixture-note", CommentID: "fixture-comment"}, page: p, button: btn, endpoint: "/comment/delete", match: func(body string) bool {
				return cleanupCommentRequestMatches(body, "fixture-note", "fixture-comment")
			}}
			require.Zero(t, calls.Load(), "prepare must not send")
			err = prepared.Execute(context.Background())
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.EqualValues(t, 1, calls.Load(), "never retry a deletion toggle")
			require.ErrorContains(t, prepared.Execute(context.Background()), "already dispatched")
			require.EqualValues(t, 1, calls.Load(), "a second Execute must not dispatch")
		})
	}
	t.Run("unrelated success does not confirm target deletion", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"success":true,"code":0}`) }))
		defer srv.Close()
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.Navigate(srv.URL))
		obs := observeSubmissionMatching(p, "/comment/delete", "fixture-note", "", func(body string) bool {
			return cleanupCommentRequestMatches(body, "fixture-note", "fixture-comment")
		})
		defer obs.close()
		_, err := p.Eval(`()=>fetch('/comment/delete',{method:'POST',body:JSON.stringify({noteId:'fixture-note',commentId:'fixture-other'})}).then(()=>true)`)
		require.NoError(t, err)
		r, err := awaitSubmission(p, obs, time.Second)
		require.Error(t, err)
		require.NotEqual(t, "confirmed", r.State)
	})
}
