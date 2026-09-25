// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/sites"
)

func ValidateSiteConfig() error {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("XHS_SITE")))
	if value == "" || value == "auto" {
		return nil
	}
	if _, ok := sites.Parse(value); !ok {
		return fmt.Errorf("XHS_SITE must be auto, xiaohongshu or rednote")
	}
	return nil
}

func configuredSite() (sites.Site, bool) { return sites.Parse(os.Getenv("XHS_SITE")) }

// ActiveSite is session-local configuration, not a mutable process-wide flag.
// An old cookie jar with sessions for both hosts is resolved by a real login
// check, never by treating a web_session cookie alone as authentication.
func ActiveSite() sites.Site {
	if site, ok := configuredSite(); ok {
		return site
	}
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	if site := store.LoadSite(); site != "" {
		return site
	}
	if candidates := cookieSites(store); len(candidates) == 1 {
		return candidates[0]
	}
	return sites.Xiaohongshu
}

func cookieSites(store cookies.Cookier) []sites.Site {
	data, err := store.LoadCookies()
	if err != nil {
		return nil
	}
	var jar []struct {
		Name    string  `json:"name"`
		Domain  string  `json:"domain"`
		Expires float64 `json:"expires"`
	}
	if json.Unmarshal(data, &jar) != nil {
		return nil
	}
	found := map[sites.Site]bool{}
	for _, c := range jar {
		if c.Name != "web_session" || (c.Expires > 0 && c.Expires < float64(time.Now().Unix())) {
			continue
		}
		if site, ok := sites.FromURL("https://" + strings.TrimPrefix(c.Domain, ".")); ok {
			found[site] = true
		}
	}
	result := []sites.Site{}
	for _, site := range []sites.Site{sites.Xiaohongshu, sites.RedNote} {
		if found[site] {
			result = append(result, site)
		}
	}
	return result
}

func loginSites() []sites.Site {
	if site, ok := configuredSite(); ok {
		return []sites.Site{site}
	}
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	if site := store.LoadSite(); site != "" {
		return []sites.Site{site}
	}
	if candidates := cookieSites(store); len(candidates) > 0 {
		return candidates
	}
	return []sites.Site{sites.Xiaohongshu}
}

// Prefer the current page after a legitimate cross-site redirect. Blank pages
// use the persisted site, so newly created browser sessions stay consistent.
func pageSite(page *rod.Page) sites.Site {
	if site, ok := pageOriginSite(page); ok {
		return site
	}
	return ActiveSite()
}

func pageOriginSite(page *rod.Page) (sites.Site, bool) {
	if page != nil {
		info, err := page.Timeout(2 * time.Second).Info()
		if err == nil {
			return sites.FromURL(info.URL)
		}
	}
	return "", false
}

// SaveBrowserSession persists the actual verified origin along with the cookie
// jar. It does not copy cookies between domains or use a normal browser profile.
func SaveBrowserSession(page *rod.Page) error {
	site, ok := pageOriginSite(page)
	if !ok {
		return fmt.Errorf("LOGIN_STATUS_UNCONFIRMED: login origin is not a supported site")
	}
	if configured, explicit := configuredSite(); explicit && configured != site {
		return fmt.Errorf("LOGIN_SITE_MISMATCH: 扫码落地站点与 XHS_SITE 不符，请使用 XHS_SITE=auto 或实际站点")
	}
	if _, err := NewLogin(page).CurrentUser(page.GetContext()); err != nil {
		return err
	}
	jar, err := page.Browser().GetCookies()
	if err != nil {
		return err
	}
	data, err := json.Marshal(jar)
	if err != nil {
		return err
	}
	return cookies.NewLoadCookie(cookies.GetCookiesFilePath()).SaveSession(data, site)
}
