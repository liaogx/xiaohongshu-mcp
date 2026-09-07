// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
)

type LoginAction struct {
	page *rod.Page
}

func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	// 加超时保护：只是查登录态的快速检查，不应无限挂（登录扫码的等待在 Login/WaitForLogin 里）
	pp := a.page.Context(ctx).Timeout(30 * time.Second)
	if err := pp.Navigate("https://www.xiaohongshu.com/explore"); err != nil {
		return false, errors.Wrap(err, "open login status page failed")
	}
	// A sidebar element can exist in a guest page. Require a non-guest account
	// in page state; distinguish a security challenge from an expired login.
	deadline := time.Now().Add(15 * time.Second)
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
				return false, nil
			}
			if probe.Authenticated {
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
	pp := a.page.Context(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	pp.MustNavigate("https://www.xiaohongshu.com/explore")
	// 注意 pp 只做了 Context(ctx) 没设 Timeout，裸 MustWaitLoad 在此处没有任何上限。
	softWaitLoad(pp, "登录-explore 页")

	time.Sleep(2 * time.Second)

	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		return nil
	}

	pp.MustElement(".main-container .user .link-wrapper .channel")

	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := a.page.Context(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	pp.MustNavigate("https://www.xiaohongshu.com/explore")
	// 注意 pp 只做了 Context(ctx) 没设 Timeout，裸 MustWaitLoad 在此处没有任何上限。
	softWaitLoad(pp, "登录-explore 页")

	time.Sleep(2 * time.Second)

	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		return "", true, nil
	}

	src, err := pp.MustElement(".login-container .qrcode-img").Attribute("src")
	if err != nil {
		return "", false, errors.Wrap(err, "get qrcode src failed")
	}
	if src == nil || len(*src) == 0 {
		return "", false, errors.New("qrcode src is empty")
	}

	return *src, false, nil
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
			el, err := pp.Element(".main-container .user .link-wrapper .channel")
			if err == nil && el != nil {
				return true
			}
		}
	}
}
