// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/sites"
	"github.com/pkg/errors"
)

type LoginAction struct {
	page *rod.Page
}

func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	if err := ValidateSiteConfig(); err != nil {
		return false, err
	}
	for _, site := range loginSites() {
		logged, err := a.checkLoginAt(ctx, site)
		if err != nil {
			return false, err
		}
		if logged {
			if err := SaveBrowserSession(a.page.Context(ctx)); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (a *LoginAction) checkLoginAt(ctx context.Context, site sites.Site) (bool, error) {
	// 加超时保护：只是查登录态的快速检查，不应无限挂（登录扫码的等待在 Login/WaitForLogin 里）
	pp := a.page.Context(ctx).Timeout(30 * time.Second)
	if err := pp.Navigate(site.Home()); err != nil {
		return false, errors.Wrap(err, "open login status page failed")
	}
	// A sidebar element can exist in a guest page. Require a non-guest account
	// in page state; distinguish a security challenge from an expired login.
	deadline := time.Now().Add(25 * time.Second)
	var loginVisibleSince time.Time
	for time.Now().Before(deadline) {
		if err := pp.GetContext().Err(); err != nil {
			return false, errors.Wrap(err, "check login status interrupted")
		}
		probe, err := readSearchPageProbe(pp.Timeout(2 * time.Second))
		if err == nil {
			if probe.SecurityPage {
				return false, probe.accessError()
			}
			if probe.LoginVisible {
				// QR login may redirect from the domestic site to RedNote.
				// Allow transient overlays to settle before declaring logout.
				if loginVisibleSince.IsZero() {
					loginVisibleSince = time.Now()
				}
				if time.Since(loginVisibleSince) >= time.Second {
					return false, nil
				}
			} else {
				loginVisibleSince = time.Time{}
			}
			if probe.Authenticated && !probe.LoginVisible {
				if _, ok := pageOriginSite(pp); !ok {
					return false, errors.New("LOGIN_STATUS_UNCONFIRMED: unsupported login origin")
				}
				return true, nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false, errors.New("LOGIN_STATUS_UNCONFIRMED: 未读取到真实登录账户，不能仅凭侧边栏或旧 Cookie 判定已登录")
}

// CurrentUser 当前登录用户的基础信息。
type CurrentUser struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"userId"`
}

// CurrentUser 从当前页面的 __INITIAL_STATE__ 读取登录用户信息。
// 需在 CheckLoginStatus 之后调用：复用已加载的 explore 页，不做额外导航。
func (a *LoginAction) CurrentUser(ctx context.Context) (*CurrentUser, error) {
	pp := a.page.Context(ctx).Timeout(10 * time.Second)

	res, err := pp.Eval(`() => {
		const u = window.__INITIAL_STATE__ && window.__INITIAL_STATE__.user;
		const ref = u && u.userInfo;
		const info = ref && (ref.value !== undefined ? ref.value : ref._value !== undefined ? ref._value : ref);
		if (!info || info.guest) return "";
		return JSON.stringify({nickname: info.nickname, userId: info.userId || info.user_id});
	}`)
	if err != nil {
		return nil, errors.Wrap(err, "read current user state failed")
	}

	raw := res.Value.String()
	if raw == "" {
		return nil, errors.New("current user not found in page state")
	}

	var user CurrentUser
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		return nil, errors.Wrap(err, "unmarshal current user failed")
	}
	if user.UserID == "" {
		return nil, errors.New("LOGIN_STATUS_UNCONFIRMED: current user ID is empty")
	}

	return &user, nil
}

func (a *LoginAction) Login(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	_, loggedIn, err := a.FetchQrcodeImage(ctx)
	if err != nil {
		return err
	}
	if loggedIn || a.WaitForLogin(ctx) {
		return nil
	}
	return errors.Wrap(ctx.Err(), "LOGIN_STATUS_UNCONFIRMED: login was not confirmed")
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	pp := a.page.Context(ctx)
	if err := ValidateSiteConfig(); err != nil {
		return "", false, err
	}
	if err := pp.Navigate(pageSite(pp).Home()); err != nil {
		return "", false, errors.Wrap(err, "LOGIN_QRCODE_UNCONFIRMED: open login page failed")
	}
	image, logged, err := waitLoginQrcode(ctx, pp)
	if err == nil && logged {
		err = SaveBrowserSession(pp)
	}
	return image, logged && err == nil, err
}

// A slow page may become authenticated without ever rendering a QR image.
// Recheck both outcomes on every poll; a sidebar alone is not login evidence.
func waitLoginQrcode(ctx context.Context, page *rod.Page) (string, bool, error) {
	pp := page.Context(ctx)
	for {
		if probe, err := readSearchPageProbe(pp.Timeout(2 * time.Second)); err == nil {
			if probe.SecurityPage {
				return "", false, probe.accessError()
			}
			if probe.Authenticated && !probe.LoginVisible {
				return "", true, nil
			}
			if probe.LoginVisible {
				result, err := pp.Timeout(time.Second).Eval(`()=>[...document.querySelectorAll('.login-container .qrcode-img')].filter(e=>e.getClientRects().length && getComputedStyle(e).visibility!=='hidden').map(e=>e.getAttribute('src')).find(Boolean)||''`)
				if err == nil && result.Value.Str() != "" {
					return result.Value.Str(), false, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", false, errors.Wrap(ctx.Err(), "LOGIN_QRCODE_UNCONFIRMED: neither authenticated account nor visible QR became ready")
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	pp := a.page.Context(ctx)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			probe, err := readSearchPageProbe(pp.Timeout(2 * time.Second))
			if err == nil && probe.Authenticated && !probe.SecurityPage && !probe.LoginVisible {
				return true
			}
		}
	}
}
