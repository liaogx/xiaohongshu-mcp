//go:build browser

// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

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

// Local synthetic fixtures only. No real cookies, QR codes or XHS requests.
func TestAccessPageFixtures(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	for _, tc := range []struct {
		name, path, html, code string
		authenticated          bool
	}{
		{"title-only challenge", "/search_result", "<title>安全验证</title><p>请使用 App</p>", "SECURITY_VERIFICATION_REQUIRED", false},
		{"redirect path without body", "/website-login/captcha", "<title>Loading</title>", "SECURITY_VERIFICATION_REQUIRED", false},
		{"detail access error redirect", "/website-login/error", "<title>错误</title>", "SECURITY_VERIFICATION_REQUIRED", false},
		{"actual challenge text", "/other", "<p>为保护账号安全，请使用已登录该账号的小红书APP扫码验证身份</p>", "SECURITY_VERIFICATION_REQUIRED", false},
		{"visible login", "/explore", "<div class='login-container'>登录</div>", "AUTH_REQUIRED", false},
		{"hidden login not a gate", "/explore", "<div style='display:none' class='login-container'>登录</div>", "", false},
		{"reactive user and raw feed array", "/search_result", `<script>window.__INITIAL_STATE__={user:{userInfo:{_value:{userId:'fixture',guest:false}}},search:{feeds:[{id:'fixture'}]}}</script>`, "", true},
		{"guest sidebar must not imply login", "/explore", `<div class='main-container'><div class='user'><div class='link-wrapper'><div class='channel'>我</div></div></div></div><script>window.__INITIAL_STATE__={user:{userInfo:{value:{guest:true,userId:'guest'}}}}</script>`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprint(w, tc.html)
			}))
			defer srv.Close()
			p := b.NewPage().Timeout(10 * time.Second)
			defer p.Close()
			require.NoError(t, p.Navigate(srv.URL+tc.path))
			require.NoError(t, p.WaitLoad())
			probe, err := readSearchPageProbe(p)
			require.NoError(t, err)
			require.Equal(t, tc.authenticated, probe.Authenticated)
			if tc.code != "" {
				require.ErrorContains(t, probe.accessError(), tc.code)
				require.ErrorContains(t, checkPageAccessible(p), tc.code)
				started := time.Now()
				require.ErrorContains(t, waitForSearchData(p, 5*time.Second), tc.code)
				require.Less(t, time.Since(started), 2*time.Second)
			} else {
				require.NoError(t, probe.accessError())
			}
			if tc.authenticated {
				require.NoError(t, waitForSearchData(p, time.Second))
			}
		})
	}
}

func TestFilterIgnoresHiddenDuplicateLabels(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	p := b.NewPage()
	defer p.Close()
	require.NoError(t, p.SetDocumentContent(`<div class="filter-panel"><div class="filters"><span>排序依据</span><div class="tags active" aria-hidden="true" onclick="window.wrongClick=true">最多评论</div><div class="tags" onclick="this.classList.add('active');window.realClick=true">最多评论</div></div></div>`))
	selected, err := p.Eval(filterSelectedJS, "排序依据", "最多评论")
	require.NoError(t, err)
	require.False(t, selected.Value.Bool())
	option, err := findFilterOption(p, pendingFilter{group: "排序依据", option: "最多评论"})
	require.NoError(t, err)
	isSelected, err := filterOptionSelected(option)
	require.NoError(t, err)
	require.False(t, isSelected)
	clicked, err := p.Eval(filterClickJS, "排序依据", "最多评论")
	require.NoError(t, err)
	require.True(t, clicked.Value.Bool())
	state, err := p.Eval(`()=>({wrong:!!window.wrongClick,real:!!window.realClick})`)
	require.NoError(t, err)
	require.False(t, state.Value.Get("wrong").Bool())
	require.True(t, state.Value.Get("real").Bool())
	selected, err = p.Eval(filterSelectedJS, "排序依据", "最多评论")
	require.NoError(t, err)
	require.True(t, selected.Value.Bool())
}

func TestProfileReadinessDoesNotAcceptInitialEmptyList(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	p := b.NewPage()
	defer p.Close()
	require.NoError(t, p.SetDocumentContent(`<script>window.__INITIAL_STATE__={user:{userPageData:{basicInfo:{}},activeTab:{query:'note',index:0},notes:[[]],noteQueries:[{hasMore:true}],isFetchingNotes:false}}</script>`))
	r, err := p.Eval(profileDataReadyJS, "note")
	require.NoError(t, err)
	require.False(t, r.Value.Bool())
	_, err = p.Eval(`()=>{window.__INITIAL_STATE__.user.noteQueries[0].hasMore=false}`)
	require.NoError(t, err)
	r, err = p.Eval(profileDataReadyJS, "note")
	require.NoError(t, err)
	require.True(t, r.Value.Bool(), "a confirmed empty tab is a valid result")
	r, err = p.Eval(profileDataReadyJS, "liked")
	require.NoError(t, err)
	require.False(t, r.Value.Bool(), "the wrong tab is never a match")
	_, err = p.Eval(`()=>{window.__INITIAL_STATE__.user.isFetchingNotes=true}`)
	require.NoError(t, err)
	r, err = p.Eval(profileDataReadyJS, "note")
	require.NoError(t, err)
	require.False(t, r.Value.Bool(), "wait while the new tab request is in flight")
	_, err = p.Eval(`()=>{window.__INITIAL_STATE__.user.isFetchingNotes=[false,true,false]}`)
	require.NoError(t, err)
	r, err = p.Eval(profileDataReadyJS, "note")
	require.NoError(t, err)
	require.True(t, r.Value.Bool(), "per-tab fetching arrays must use the active index")
}

func TestFilterLeafSelection(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	b := browser.NewBrowser(true)
	defer b.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<div class="filter-panel"><div class="filters"><span>排序依据</span><div class="tags active"><div class="tags">综合</div></div><div class="tags"><div class="tags active">最多评论</div></div></div><div class="filters"><span>笔记类型</span><div class="tags active">不限</div></div></div>`)
	}))
	defer srv.Close()
	p := b.NewPage().Timeout(10 * time.Second)
	defer p.Close()
	require.NoError(t, p.Navigate(srv.URL))
	require.NoError(t, p.WaitLoad())
	for _, tc := range []struct {
		group, option string
		selected      bool
	}{
		{"排序依据", "综合", false},
		{"排序依据", "最多评论", true},
		{"笔记类型", "不限", true},
	} {
		opt, err := findFilterOption(p, pendingFilter{group: tc.group, option: tc.option})
		require.NoError(t, err)
		selected, err := filterOptionSelected(opt)
		require.NoError(t, err)
		require.Equal(t, tc.selected, selected)
	}
}
