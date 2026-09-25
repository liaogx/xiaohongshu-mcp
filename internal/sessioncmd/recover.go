// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package sessioncmd

import (
	"fmt"
	"io"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/configs"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
)

type RecoverOptions struct {
	Keyword string
	Fresh   bool
}

func Recover(options RecoverOptions, output io.Writer) error {
	if err := xiaohongshu.ValidateSiteConfig(); err != nil {
		return err
	}
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	// Match explicit service configuration without writing the saved session
	// before a fresh login has actually been confirmed.
	seed := configs.FingerprintSeedFromEnv()
	if seed <= 0 {
		seed = store.LoadSeed()
	}
	b := browser.NewBrowser(false, browser.WithFingerprintSeed(seed), browser.WithProxy(configs.ProxyFromEnv()))
	defer b.Close()
	p := b.NewPage().Timeout(8 * time.Minute)
	defer p.Close()
	target := xiaohongshu.ActiveSite().Search(options.Keyword)
	if options.Fresh {
		// Clear only the new isolated browser, never the saved login file.
		if err := (proto.NetworkClearBrowserCookies{}).Call(p); err != nil {
			return fmt.Errorf("无法建立新的登录会话；原登录文件保持不变")
		}
		target = xiaohongshu.ActiveSite().Home()
	}
	if err := p.Navigate(target); err != nil {
		return fmt.Errorf("打开验证页面失败；未修改登录文件")
	}
	fmt.Fprintln(output, "请在专用窗口中手动完成扫码或安全验证。")
	return waitForRecovery(p, options.Fresh, output, 7*time.Minute)
}

func recoveryPageState(page *rod.Page, fresh bool) (string, error) {
	probe, err := xiaohongshu.ProbePageAccess(page.Timeout(3 * time.Second))
	if err != nil {
		return "", err
	}
	if probe.SecurityVerification {
		return "challenge", nil
	}
	if probe.LoginVisible {
		return "login", nil
	}
	if !probe.Authenticated {
		return "loading", nil
	}
	if fresh {
		return "ready", nil
	}
	result, err := page.Timeout(3 * time.Second).Eval(`() => !!(window.__INITIAL_STATE__ && window.__INITIAL_STATE__.search)`)
	if err != nil {
		return "", err
	}
	if result.Value.Bool() {
		return "ready", nil
	}
	return "loading", nil
}

func waitForRecovery(page *rod.Page, fresh bool, output io.Writer, timeout time.Duration) error {
	last := ""
	readFailures := 0
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := recoveryPageState(page, fresh)
		// A QR refresh can replace the execution context; retain this window.
		if err != nil {
			readFailures++
			if readFailures == 1 {
				fmt.Fprintln(output, "验证页正在切换，保留当前窗口。")
			}
			if readFailures >= 40 {
				return fmt.Errorf("验证窗口持续不可用；未确认恢复")
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		readFailures = 0
		if state != last {
			fmt.Fprintln(output, "verification_state="+state)
			last = state
		}
		if state == "ready" {
			if err := xiaohongshu.SaveBrowserSession(page); err != nil {
				return fmt.Errorf("保存登录状态失败: %w", err)
			}
			fmt.Fprintln(output, "验证已完成，MCP 登录状态已保存；请重新检查登录并做一次只读搜索。")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("等待扫码或验证超时；未确认恢复，未替换登录文件")
}
