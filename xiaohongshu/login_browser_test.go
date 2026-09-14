//go:build browser

package xiaohongshu

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/stretchr/testify/require"
)

// Synthetic page state only; no real account or QR code is used.
func TestLoginQrcodeReadinessFixtures(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	for _, tc := range []struct {
		name, html, image, code string
		logged                  bool
	}{
		{"delayed account without QR", `<script>setTimeout(()=>window.__INITIAL_STATE__={user:{userInfo:{value:{userId:'fixture-account',guest:false}}}},300)</script>`, "", "", true},
		{"guest sidebar is not a session", `<div class="main-container"><div class="user"><div class="link-wrapper"><div class="channel">我</div></div></div></div><script>window.__INITIAL_STATE__={user:{userInfo:{value:{userId:'fixture-guest',guest:true}}}}</script>`, "", "LOGIN_QRCODE_UNCONFIRMED", false},
		{"visible QR", `<div class="login-container"><div class="qrcode-img" src="synthetic-qr-placeholder" style="width:10px;height:10px"></div></div>`, "synthetic-qr-placeholder", "", false},
		{"hidden QR does not count", `<div class="login-container" style="display:none"><div class="qrcode-img" src="synthetic-qr-placeholder"></div></div>`, "", "LOGIN_QRCODE_UNCONFIRMED", false},
		{"security challenge wins over cached account", `<title>安全验证</title><script>window.__INITIAL_STATE__={user:{userInfo:{value:{userId:'fixture-account',guest:false}}}}</script>`, "", "SECURITY_VERIFICATION_REQUIRED", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := b.NewPage()
			defer p.Close()
			require.NoError(t, p.SetDocumentContent(tc.html))
			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			defer cancel()
			image, logged, err := waitLoginQrcode(ctx, p)
			if tc.code == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.code)
			}
			require.Equal(t, tc.image, image)
			require.Equal(t, tc.logged, logged)
		})
	}
	t.Run("scan wait requires a real account", func(t *testing.T) {
		p := b.NewPage()
		defer p.Close()
		require.NoError(t, p.SetDocumentContent(`<div class="main-container"><div class="user"><div class="link-wrapper"><div class="channel">我</div></div></div></div>`))
		ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
		defer cancel()
		require.False(t, NewLogin(p).WaitForLogin(ctx))
		_, err := p.Eval(`()=>{window.__INITIAL_STATE__={user:{userInfo:{value:{userId:'fixture-account',guest:false}}}}}`)
		require.NoError(t, err)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		require.True(t, NewLogin(p).WaitForLogin(ctx2))
	})
}
