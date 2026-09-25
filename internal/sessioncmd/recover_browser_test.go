//go:build browser

package sessioncmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/stretchr/testify/require"
)

func TestManualRecoveryUsesVerifiedPageState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cookies.json")
	t.Setenv("COOKIES_PATH", path)
	t.Setenv("XHS_SITE", "auto")
	b := browser.NewBrowser(true)
	defer b.Close()
	const account = `<script>window.__INITIAL_STATE__={user:{userInfo:{value:{userId:'fixture-account',guest:false}}}};</script>`
	for _, tc := range []struct {
		name, html, want string
		fresh            bool
	}{
		{"empty page", `<div>Loading</div>`, "loading", false},
		{"homepage is not search readiness", account, "loading", false},
		{"fresh login", account, "ready", true},
		{"search hydrated", account + `<script>window.__INITIAL_STATE__.search={feeds:[]};</script>`, "ready", false},
		{"guest", `<script>window.__INITIAL_STATE__={user:{userInfo:{userId:'fixture-guest',guest:true}},search:{}};</script>`, "loading", true},
		{"challenge before account", account + `<title>安全验证</title>`, "challenge", true},
		{"visible login before account", account + `<div class="login-container" style="width:100px;height:100px">Login</div>`, "login", true},
		{"hidden login", account + `<div class="login-container" style="display:none">Login</div>`, "ready", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := b.NewPage()
			defer p.Close()
			require.NoError(t, p.SetDocumentContent(tc.html))
			state, err := recoveryPageState(p, tc.fresh)
			require.NoError(t, err)
			require.Equal(t, tc.want, state)
		})
	}
	p := b.NewPage()
	defer p.Close()
	require.NoError(t, p.SetDocumentContent(`<title>安全验证</title>`))
	require.Error(t, waitForRecovery(p, true, io.Discard, time.Millisecond))
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "pending verification must not save a session")
}
