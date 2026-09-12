//go:build browser

// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/stretchr/testify/require"
)

func TestDetailPageReadyRequiresHydratedExplorePage(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/explore/feed-1" {
			fmt.Fprint(w, `<script>window.__INITIAL_STATE__={note:{noteDetailMap:{"feed-1":{note:{}}}}}</script>`)
			return
		}
		fmt.Fprint(w, `<title>错误</title>`)
	}))
	defer srv.Close()

	readyPage := b.NewPage().Timeout(10 * time.Second)
	defer readyPage.Close()
	require.NoError(t, readyPage.Navigate(srv.URL+"/explore/feed-1"))
	require.NoError(t, readyPage.WaitLoad())
	ready, err := detailPageReady(readyPage)
	require.NoError(t, err)
	require.True(t, ready)

	blockedPage := b.NewPage().Timeout(10 * time.Second)
	defer blockedPage.Close()
	require.NoError(t, blockedPage.Navigate(srv.URL+"/website-login/error"))
	require.NoError(t, blockedPage.WaitLoad())
	ready, err = detailPageReady(blockedPage)
	require.NoError(t, err)
	require.False(t, ready)
}
