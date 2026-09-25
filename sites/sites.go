// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// Package sites contains the supported web origins and route differences.
// Cookie domains must never be rewritten to move a session between sites.
package sites

import (
	"net/url"
	"strings"
)

type Site string

const (
	Xiaohongshu Site = "xiaohongshu"
	RedNote     Site = "rednote"
)

func Parse(value string) (Site, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(Xiaohongshu):
		return Xiaohongshu, true
	case string(RedNote):
		return RedNote, true
	default:
		return "", false
	}
}

// FromURL accepts exact official HTTPS origins, not lookalike suffixes or
// arbitrary redirect destinations. Both web and creator origins have a site.
func FromURL(raw string) (Site, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return "", false
	}
	switch strings.ToLower(u.Hostname()) {
	case "xiaohongshu.com", "www.xiaohongshu.com", "creator.xiaohongshu.com":
		return Xiaohongshu, true
	case "rednote.com", "www.rednote.com", "creator.rednote.com":
		return RedNote, true
	default:
		return "", false
	}
}

func (s Site) domain() string {
	if s == RedNote {
		return "rednote.com"
	}
	return "xiaohongshu.com"
}

func (s Site) Origin() string        { return "https://www." + s.domain() }
func (s Site) CreatorOrigin() string { return "https://creator." + s.domain() }
func (s Site) Home() string {
	if s == RedNote {
		return s.Origin() + "/"
	}
	return s.Origin() + "/explore"
}
func (s Site) Search(keyword string) string {
	q := url.Values{"keyword": {keyword}, "source": {"web_explore_feed"}}
	return s.Origin() + "/search_result?" + q.Encode()
}
func (s Site) Detail(id, token string) string {
	path := "/explore/"
	if s == RedNote {
		path = "/discovery/item/"
	}
	q := url.Values{"xsec_token": {token}, "xsec_source": {"pc_search"}, "source": {"web_explore_feed"}}
	return s.Origin() + path + url.PathEscape(id) + "?" + q.Encode()
}
func (s Site) Profile(id, token, tab string) string {
	q := url.Values{}
	if token != "" {
		q.Set("xsec_token", token)
		q.Set("xsec_source", "pc_note")
	}
	if tab != "" && tab != "note" {
		q.Set("tab", tab)
		q.Set("subTab", "note")
	}
	target := s.Origin() + "/user/profile/" + url.PathEscape(id)
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	return target
}
func (s Site) Notifications() string { return s.Origin() + "/notification" }
func (s Site) Chat() string          { return s.Origin() + "/chat" }
func (s Site) Publish() string       { return s.CreatorOrigin() + "/publish/publish?source=official" }
func (s Site) ManageNotes() string   { return s.CreatorOrigin() + "/new/note-manager" }
