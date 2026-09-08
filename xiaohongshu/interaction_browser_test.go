//go:build browser

// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

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

	"github.com/go-rod/rod/lib/proto"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/stretchr/testify/require"
)

const fixtureNoteState = `window.__INITIAL_STATE__={user:{userInfo:{value:{guest:false,userId:'fixture-user'}}},note:{noteDetailMap:{'fixture-note':{note:{noteId:'fixture-note',interactInfo:{liked:false,collected:false}}}}}};`

// A real browser with local synthetic pages only. No platform interactions.
func TestInteractionBrowserFixtures(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	t.Setenv("XHS_INTERACTION_STATE_DIR", t.TempDir())
	b := browser.NewBrowser(true)
	defer b.Close()
	t.Run("reactivate collapsed editor without submitting", func(t *testing.T) {
		p := b.NewPage().Timeout(5 * time.Second)
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="input-box"><div class="content-edit"><p class="content-input" contenteditable="true" onclick="document.querySelector('button').style.display='block'">fixture text</p></div></div><div class="bottom"><button class="submit" style="display:none" onclick="window.submitted=true">Send</button></div>`))
		button, err := commentSubmitButton(p)
		require.NoError(t, err)
		_, err = button.Interactable()
		require.NoError(t, err)
		v, err := p.Eval(`()=>window.submitted===true`)
		require.NoError(t, err)
		require.False(t, v.Value.Bool())
	})
	t.Run("canceled observer fails before dispatch", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		o := observeSubmission(p.Context(ctx), "/note/like", "fixture-note", "")
		defer o.close()
		require.Error(t, o.setupErr)
	})
	t.Run("missing interaction state is not false", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<script>window.__INITIAL_STATE__={note:{noteDetailMap:{'fixture-note':{note:{noteId:'fixture-note'}}}}}</script>`))
		_, _, err := newInteractAction(p).getInteractState(p, "fixture-note")
		require.ErrorContains(t, err, "禁止盲点")
	})
	t.Run("wait for target data rather than a blank shell", func(t *testing.T) {
		p := b.NewPage().Timeout(10 * time.Second)
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="input-box"><div class="content-edit">input</div></div><script>window.__INITIAL_STATE__={};setTimeout(()=>{`+fixtureNoteState+`},1200);setInterval(()=>document.body.dataset.tick=Date.now(),50)</script>`))
		start := time.Now()
		state, err := waitNoteReady(p, "fixture-note", true, 5*time.Second)
		require.NoError(t, err)
		require.Equal(t, "fixture-note", state.NoteID)
		require.GreaterOrEqual(t, time.Since(start), time.Second)
	})
	t.Run("slow server confirmation and repeated submit", func(t *testing.T) {
		var posts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				posts.Add(1)
				time.Sleep(5 * time.Second)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"success":true,"code":0,"data":{"comment":{"id":"fixture-comment"}}}`)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<div class="input-box"><div class="content-edit"><p contenteditable="true" class="content-input">fixture text</p></div></div><div class="comments-container">fixture text</div><div class="bottom"><button class="submit" onclick="fetch('/api/sns/web/v1/comment/post',{method:'POST',body:JSON.stringify({note_id:'fixture-note',content:'fixture text'})})">Send</button></div><script>`+fixtureNoteState+`</script>`)
		}))
		defer srv.Close()
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.Navigate(srv.URL))
		require.NoError(t, p.Timeout(5*time.Second).WaitLoad())
		draft := &PreparedComment{action: NewCommentFeedAction(p), feedID: "fixture-note", content: "fixture text", userID: "fixture-user"}
		start := time.Now()
		require.NoError(t, draft.Submit(context.Background()))
		require.GreaterOrEqual(t, time.Since(start), 5*time.Second)
		require.ErrorContains(t, draft.Submit(context.Background()), "DUPLICATE_OR_UNCERTAIN_COMMENT")
		require.EqualValues(t, 1, posts.Load())
	})
	for _, tc := range []struct{ name, body, want string }{
		{"HTTP 200 rejection", `{"success":false,"code":-1}`, "rejected"},
		{"unrecognized body is ambiguous", `{}`, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tc.body) }))
			defer srv.Close()
			p := b.NewPage()
			defer p.Close()
			require.NoError(t, p.Navigate(srv.URL))
			obs := observeSubmission(p, "/comment/post", "fixture-note", "fixture text")
			defer obs.close()
			_, err := p.Eval(`()=>fetch('/comment/post',{method:'POST',body:JSON.stringify({note_id:'fixture-note',content:'fixture text'})}).then(()=>true)`)
			require.NoError(t, err)
			r, err := awaitSubmission(p, obs, 1500*time.Millisecond)
			require.Error(t, err)
			var outcome *InteractionError
			require.ErrorAs(t, err, &outcome)
			require.Equal(t, tc.want, outcome.State)
			if tc.want == "rejected" {
				require.Equal(t, tc.want, r.State)
			}
		})
	}
	t.Run("like once with stale initial state", func(t *testing.T) {
		var clicks atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				clicks.Add(1)
				fmt.Fprint(w, `{"success":true,"code":0}`)
				return
			}
			fmt.Fprint(w, `<div class="interact-container"><div class="left"><button class="like-lottie" onclick="fetch('/api/sns/web/v1/note/like',{method:'POST',body:JSON.stringify({note_oid:'fixture-note'})})">like</button></div></div><script>`+fixtureNoteState+`</script>`)
		}))
		defer srv.Close()
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.Navigate(srv.URL))
		require.NoError(t, NewLikeAction(p).LikeOnCurrentPage(context.Background(), "fixture-note"))
		require.EqualValues(t, 1, clicks.Load())
	})
	t.Run("manual verification after submit", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<button onclick="document.title='安全验证'">Send</button>`))
		obs := observeSubmission(p, "/comment/post", "fixture-note", "fixture text")
		defer obs.close()
		button, err := p.Element("button")
		require.NoError(t, err)
		require.NoError(t, button.Click(proto.InputMouseButtonLeft, 1))
		_, err = awaitSubmission(p, obs, time.Second)
		var gate *PageAccessError
		require.ErrorAs(t, err, &gate)
		require.Equal(t, "SECURITY_VERIFICATION_REQUIRED", gate.Code)
	})
	t.Run("successive observers across request contexts on same page", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"success":true,"code":0}`) }))
		defer srv.Close()
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.Navigate(srv.URL))
		for round := 0; round < 3; round++ {
			ctx, cancel := context.WithCancel(context.Background())
			obs := observeSubmission(p.Context(ctx), "/note/like", "fixture-note", "")
			_, err := p.Eval(`()=>fetch('/note/like',{method:'POST',body:JSON.stringify({note_oid:'fixture-note'})}).then(()=>true)`)
			require.NoError(t, err)
			_, err = awaitSubmission(p, obs, time.Second)
			obs.close()
			cancel()
			require.NoError(t, err)
		}
	})
}
