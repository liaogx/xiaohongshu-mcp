// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/configs"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
)

const securityVerificationTimeout = 10 * time.Minute

// securityVerificationManager 同时只保留一个可见的人工验证窗口，避免定时
// 任务连续报错时反复创建窗口，也避免多个窗口争用同一个 cookies 文件。
type securityVerificationManager struct {
	mu      sync.Mutex
	opening bool
	session *securityVerificationSession
}

type securityVerificationSession struct {
	browser *headless_browser.Browser
	page    *rod.Page
	done    chan struct{}
	cancel  context.CancelFunc
}

func (s *XiaohongshuService) handleSecurityVerification(err error, keyword string) {
	var accessErr *xiaohongshu.PageAccessError
	if errors.As(err, &accessErr) && accessErr.Code == "SECURITY_VERIFICATION_REQUIRED" {
		s.openSecurityVerification(keyword)
		return
	}
	// Keep the popup behavior for wrapped or translated errors returned by an
	// action handler. The typed error is preferred, but the stable code is
	// still useful at the service boundary.
	if !strings.Contains(err.Error(), "SECURITY_VERIFICATION_REQUIRED") {
		return
	}

	s.openSecurityVerification(keyword)
}

// openSecurityVerification 在检测到安全验证时启动一个真正可见的浏览器窗口。
// 该窗口仍然由 MCP 的浏览器工厂创建，因此会读取同一份 cookies、指纹和代理
// 配置；不会打开用户日常 Chrome，也不会把账号资料复制到其他浏览器。
func (s *XiaohongshuService) openSecurityVerification(keyword string) {
	s.securityVerification.mu.Lock()
	if s.securityVerification.opening || s.securityVerification.session != nil {
		s.securityVerification.mu.Unlock()
		logrus.Warn("小红书安全验证窗口已存在，复用当前 MCP 会话")
		return
	}
	s.securityVerification.opening = true
	s.securityVerification.mu.Unlock()

	b := browser.NewBrowser(false,
		browser.WithFingerprintSeed(configs.FingerprintSeed()),
		browser.WithProxy(configs.Proxy()),
	)
	page := b.NewPage()
	closeBrowser := func() {
		_ = page.Close()
		b.Close()
	}

	if err := page.Timeout(60 * time.Second).Navigate(securityVerificationURL(keyword)); err != nil {
		closeBrowser()
		s.securityVerification.mu.Lock()
		s.securityVerification.opening = false
		s.securityVerification.mu.Unlock()
		logrus.Errorf("打开小红书安全验证窗口失败: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), securityVerificationTimeout)
	session := &securityVerificationSession{
		browser: b,
		page:    page,
		done:    make(chan struct{}),
		cancel:  cancel,
	}

	s.securityVerification.mu.Lock()
	s.securityVerification.opening = false
	if s.securityVerification.session != nil {
		// 极小的并发窗口：另一请求已经先完成了创建，关闭当前多余窗口。
		s.securityVerification.mu.Unlock()
		cancel()
		closeBrowser()
		logrus.Warn("检测到并发的安全验证窗口，已关闭重复窗口")
		return
	}
	s.securityVerification.session = session
	s.securityVerification.mu.Unlock()

	logrus.Warnf("已打开小红书安全验证窗口，请在该窗口中用小红书 App 完成人工验证；窗口最多保留 %s", securityVerificationTimeout)
	go s.watchSecurityVerification(ctx, session, keyword)
}

func securityVerificationURL(keyword string) string {
	if keyword == "" {
		return "https://www.xiaohongshu.com/explore"
	}

	values := url.Values{}
	values.Set("keyword", keyword)
	values.Set("source", "web_explore_feed")
	return fmt.Sprintf("https://www.xiaohongshu.com/search_result?%s", values.Encode())
}

func (s *XiaohongshuService) watchSecurityVerification(ctx context.Context, session *securityVerificationSession, keyword string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer session.cancel()
	defer func() {
		_ = session.page.Close()
		session.browser.Close()

		s.securityVerification.mu.Lock()
		if s.securityVerification.session == session {
			s.securityVerification.session = nil
		}
		s.securityVerification.mu.Unlock()
		close(session.done)
	}()

	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("小红书安全验证窗口结束，未确认验证完成（关键词=%q）", keyword)
			return
		case <-ticker.C:
			verified, err := securityVerificationCompleted(session.page)
			if err != nil {
				// 页面在人工验证过程中可能短暂重载；只记录安全的状态错误，
				// 保留窗口继续等待，不把它误报成验证成功。
				logrus.Debugf("等待小红书安全验证完成: %v", err)
				continue
			}
			if !verified {
				continue
			}

			if err := saveCookies(session.page); err != nil {
				logrus.Errorf("小红书安全验证已通过，但保存 cookies 失败: %v", err)
				return
			}
			logrus.Infof("小红书安全验证已完成，MCP cookies 已保存（关键词=%q）", keyword)
			return
		}
	}
}

func securityVerificationCompleted(page *rod.Page) (bool, error) {
	state, err := xiaohongshu.ProbePageAccess(page.Timeout(3 * time.Second))
	if err != nil {
		return false, err
	}
	if state.SecurityVerification {
		return false, nil
	}
	if state.Authenticated {
		return true, nil
	}
	if state.LoginVisible {
		return false, nil
	}

	// 验证页面有时不会自动回到搜索页。验证页消失后，用同一个可见页面
	// 检查 explore 的真实登录态，再把该页面的 cookies 保存回 MCP。
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	loggedIn, err := xiaohongshu.NewLogin(page).CheckLoginStatus(ctx)
	if err != nil {
		return false, err
	}
	return loggedIn, nil
}
