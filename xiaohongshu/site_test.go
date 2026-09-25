package xiaohongshu

import (
	"path/filepath"
	"testing"

	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/sites"
	"github.com/stretchr/testify/require"
)

func TestSessionSiteSelection(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	t.Setenv("XHS_SITE", "auto")
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	require.Equal(t, sites.Xiaohongshu, ActiveSite())
	require.NoError(t, store.SaveCookies([]byte(`[{"name":"web_session","domain":".rednote.com","value":"synthetic"}]`)))
	require.Equal(t, sites.RedNote, ActiveSite())
	require.NoError(t, store.SaveCookies([]byte(`[{"name":"web_session","domain":".rednote.com","value":"synthetic"},{"name":"web_session","domain":".xiaohongshu.com","value":"synthetic"}]`)))
	require.Equal(t, []sites.Site{sites.Xiaohongshu, sites.RedNote}, loginSites())
	require.NoError(t, store.SaveSite(sites.RedNote))
	require.Equal(t, sites.RedNote, ActiveSite())
	require.Equal(t, []sites.Site{sites.RedNote}, loginSites(), "a verified site must not silently fall back to a different account")
	t.Setenv("XHS_SITE", "xiaohongshu")
	require.Equal(t, sites.Xiaohongshu, ActiveSite())
	require.Equal(t, []sites.Site{sites.Xiaohongshu}, loginSites())
	require.NoError(t, ValidateSiteConfig())
	t.Setenv("XHS_SITE", "invalid")
	require.Error(t, ValidateSiteConfig())
}

func TestExpiredAndUntrustedCookieSitesAreNotCandidates(t *testing.T) {
	t.Setenv("COOKIES_PATH", filepath.Join(t.TempDir(), "cookies.json"))
	t.Setenv("XHS_SITE", "")
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	require.NoError(t, store.SaveCookies([]byte(`[{"name":"web_session","domain":".rednote.com","expires":1,"value":"synthetic"},{"name":"web_session","domain":".rednote.com.evil.invalid","value":"synthetic"},{"name":"unrelated","domain":".rednote.com","value":"synthetic"}]`)))
	require.Empty(t, cookieSites(store))
	require.Equal(t, sites.Xiaohongshu, ActiveSite())
}
