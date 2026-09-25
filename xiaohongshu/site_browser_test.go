//go:build browser

package xiaohongshu

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/sites"
	"github.com/stretchr/testify/require"
)

// Every request is intercepted with synthetic data; these tests never reach
// either platform, read real credentials, or send a real interaction.
func TestCrossSiteLoginBrowserFixtures(t *testing.T) {
	t.Setenv("XHS_SITE", "auto")
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	require.NoError(t, store.SaveCookies([]byte(`[{"name":"web_session","domain":".xiaohongshu.com","path":"/","value":"synthetic"},{"name":"web_session","domain":".rednote.com","path":"/","value":"synthetic"}]`)))
	require.NoError(t, store.SaveSeed(12345))
	b := browser.NewBrowser(true)
	defer b.Close()
	p := b.NewPage()
	defer p.Close()
	var mu sync.Mutex
	domesticVisits := 0
	redirect := ""
	router := p.Browser().HijackRequests()
	require.NoError(t, router.Add("*", "", func(h *rod.Hijack) {
		h.Response.SetHeader("Content-Type", "text/html; charset=utf-8")
		mu.Lock()
		defer mu.Unlock()
		if h.Request.URL().Hostname() == "www.xiaohongshu.com" {
			if h.Request.URL().Path == "/explore" {
				domesticVisits++
			}
			if redirect != "" {
				h.Response.Payload().ResponseCode = 302
				h.Response.SetHeader("Location", redirect)
				h.Response.SetBody("")
				return
			}
			h.Response.SetBody(`<div class="login-container">Login</div>`)
			return
		}
		h.Response.SetBody(`<script>window.__INITIAL_STATE__={user:{userInfo:{value:{guest:false,userId:'fixture-user',nickname:'Fixture'}}}}</script>`)
	}))
	go router.Run()
	defer router.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logged, err := NewLogin(p).CheckLoginStatus(ctx)
	require.NoError(t, err)
	require.True(t, logged)
	require.Equal(t, sites.RedNote, store.LoadSite())
	require.Equal(t, 12345, store.LoadSeed())
	t.Setenv("XHS_SITE", "xiaohongshu")
	require.ErrorContains(t, SaveBrowserSession(p), "LOGIN_SITE_MISMATCH")
	require.Equal(t, sites.RedNote, store.LoadSite(), "a mismatched override must preserve the previous session")
	t.Setenv("XHS_SITE", "auto")
	mu.Lock()
	visited := domesticVisits
	mu.Unlock()
	require.Equal(t, 1, visited)
	// A fresh action/page uses the verified site instead of probing domestic again.
	second := b.NewPage()
	defer second.Close()
	logged, err = NewLogin(second).CheckLoginStatus(ctx)
	require.NoError(t, err)
	require.True(t, logged)
	mu.Lock()
	visited = domesticVisits
	mu.Unlock()
	require.Equal(t, 1, visited)

	// A fresh QR flow records its final origin, not its initial login URL.
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	mu.Lock()
	redirect = sites.RedNote.Home()
	mu.Unlock()
	third := b.NewPage()
	defer third.Close()
	_, logged, err = NewLogin(third).FetchQrcodeImage(ctx)
	require.NoError(t, err)
	require.True(t, logged)
	require.Equal(t, sites.RedNote, cookies.NewLoadCookie(cookies.GetCookiesFilePath()).LoadSite())
	require.Equal(t, sites.RedNote, pageSite(third))

	// A lookalike/foreign redirect must not persist a claimed authenticated state.
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	mu.Lock()
	redirect = "https://example.invalid/"
	mu.Unlock()
	fourth := b.NewPage()
	defer fourth.Close()
	_, logged, err = NewLogin(fourth).FetchQrcodeImage(ctx)
	require.ErrorContains(t, err, "login origin")
	require.False(t, logged)
	require.NoFileExists(t, cookies.GetCookiesFilePath())
}
