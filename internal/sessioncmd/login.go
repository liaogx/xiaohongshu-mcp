// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// Package sessioncmd provides manual session entry points shared by the main
// executable and the legacy source-only commands. It never posts content.
package sessioncmd

import (
	"context"
	"fmt"
	"io"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/configs"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
)

func Login(output io.Writer) error {
	if err := xiaohongshu.ValidateSiteConfig(); err != nil {
		return err
	}
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	b := browser.NewBrowser(false,
		browser.WithFingerprintSeed(configs.ResolveFingerprintSeed(store)),
		browser.WithProxy(configs.ProxyFromEnv()),
	)
	defer b.Close()
	page := b.NewPage()
	defer page.Close()
	action := xiaohongshu.NewLogin(page)
	ctx := context.Background()
	loggedIn, err := action.CheckLoginStatus(ctx)
	if err != nil {
		return fmt.Errorf("检查登录失败，可使用 recover 打开人工验证: %w", err)
	}
	if loggedIn {
		fmt.Fprintln(output, "已登录；需要重新扫码时使用 recover -fresh。")
		return nil
	}
	fmt.Fprintln(output, "请在专用窗口中用小红书 App 扫码登录。")
	if err := action.Login(ctx); err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}
	if err := xiaohongshu.SaveBrowserSession(page); err != nil {
		return fmt.Errorf("保存登录状态失败: %w", err)
	}
	loggedIn, err = action.CheckLoginStatus(ctx)
	if err != nil {
		return fmt.Errorf("登录后核验失败: %w", err)
	}
	if !loggedIn {
		return fmt.Errorf("登录尚未确认，请重新检查")
	}
	fmt.Fprintln(output, "登录成功，登录状态已保存。")
	return nil
}
