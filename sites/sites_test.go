package sites

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOriginsAndRoutes(t *testing.T) {
	for _, site := range []Site{Xiaohongshu, RedNote} {
		t.Run(string(site), func(t *testing.T) {
			homePath, detailPath := "/explore", "/explore/fixture-note"
			if site == RedNote {
				homePath, detailPath = "/", "/discovery/item/fixture-note"
			}
			require.Equal(t, site.Origin()+homePath, site.Home())
			routes := []struct{ raw, path, origin string }{
				{site.Search("咖啡 + tea & 100%"), "/search_result", site.Origin()},
				{site.Detail("fixture-note", "synthetic+/=& token"), detailPath, site.Origin()},
				{site.Profile("fixture-user", "synthetic+/=& token", "liked"), "/user/profile/fixture-user", site.Origin()},
				{site.Notifications(), "/notification", site.Origin()},
				{site.Chat(), "/chat", site.Origin()},
				{site.Publish(), "/publish/publish", site.CreatorOrigin()},
				{site.ManageNotes(), "/new/note-manager", site.CreatorOrigin()},
			}
			for _, route := range routes {
				u, err := url.Parse(route.raw)
				require.NoError(t, err)
				require.Equal(t, route.origin, u.Scheme+"://"+u.Host)
				require.Equal(t, route.path, u.Path)
				got, ok := FromURL(route.raw)
				require.True(t, ok)
				require.Equal(t, site, got)
			}
			search, _ := url.Parse(routes[0].raw)
			require.Equal(t, "咖啡 + tea & 100%", search.Query().Get("keyword"))
			detail, _ := url.Parse(routes[1].raw)
			require.Equal(t, "synthetic+/=& token", detail.Query().Get("xsec_token"))
			require.Equal(t, "pc_search", detail.Query().Get("xsec_source"))
			profile, _ := url.Parse(routes[2].raw)
			require.Equal(t, "synthetic+/=& token", profile.Query().Get("xsec_token"))
			require.Equal(t, "liked", profile.Query().Get("tab"))
			require.Equal(t, site.Origin()+"/user/profile/fixture-user", site.Profile("fixture-user", "", "note"))
		})
	}
}

func TestRejectUntrustedOrigins(t *testing.T) {
	for _, raw := range []string{"http://www.rednote.com/", "https://rednote.com.example.org/", "https://notrednote.com/", "https://www.rednote.com:1234/", "https://www.rednote.com@example.org/", "https://user@www.rednote.com/", "https://www.xiaohongshu.com.evil.invalid/", "about:blank", "https://edith.xiaohongshu.com/"} {
		_, ok := FromURL(raw)
		require.False(t, ok, raw)
	}
}
