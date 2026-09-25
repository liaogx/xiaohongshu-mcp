// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// recover opens the MCP-owned browser for manual login or security verification.
// It never solves challenges or submits comments. Credentials stay on this device.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/pkg/buildinfo"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
	"github.com/sirupsen/logrus"
)

var version = buildinfo.DefaultVersion

func main() {
	showVersion := flag.Bool("version", false, "显示版本和构建信息，不打开验证窗口")
	keyword := flag.String("keyword", "咖啡", "用于检查搜索页可用性的关键词")
	fresh := flag.Bool("fresh", false, "在新的专用浏览器会话中重新扫码；成功前保留原登录文件")
	flag.Parse()
	if *showVersion {
		fmt.Print(buildinfo.Summary("xiaohongshu-recover", version))
		return
	}
	if err := xiaohongshu.ValidateSiteConfig(); err != nil {
		fmt.Println(err)
		return
	}
	logrus.SetLevel(logrus.ErrorLevel)
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	b := browser.NewBrowser(false, browser.WithFingerprintSeed(store.LoadSeed()), browser.WithProxy(os.Getenv("XHS_PROXY")))
	defer b.Close()
	p := b.NewPage().Timeout(8 * time.Minute)
	defer p.Close()
	target := xiaohongshu.ActiveSite().Search(*keyword)
	if *fresh {
		// Only clear the new isolated browser, never delete the saved session.
		if err := (proto.NetworkClearBrowserCookies{}).Call(p); err != nil {
			fmt.Println("无法建立新的登录会话；原登录文件保持不变")
			return
		}
		target = xiaohongshu.ActiveSite().Home()
	}
	if err := p.Navigate(target); err != nil {
		fmt.Println("打开验证页面失败；未修改登录文件")
		return
	}
	last := ""
	readFailures := 0
	deadline := time.Now().Add(7 * time.Minute)
	for time.Now().Before(deadline) {
		r, err := p.Eval(`(fresh) => {
			if (/\/website-login\/(captcha|verify|error)/.test(location.pathname) || /安全验证|身份验证/.test(document.title)) return 'challenge';
			const s=window.__INITIAL_STATE__, u=s&&s.user, ref=u&&u.userInfo;
			const info=ref&&(ref.value!==undefined?ref.value:ref._value!==undefined?ref._value:ref);
			if(info && !info.guest && (info.userId||info.user_id) && (fresh || s.search)) return 'ready';
			if(document.querySelector('.login-container')) return 'login';
			return 'loading';
		}`, *fresh)
		// Navigation can destroy the execution context during a QR-page refresh.
		// Keep the same session alive and retry briefly instead of closing it.
		if err != nil {
			readFailures++
			if readFailures == 1 {
				fmt.Printf("验证页正在切换（%T），保留当前窗口\n", err)
			}
			if readFailures >= 40 {
				fmt.Println("验证窗口持续不可用；未确认恢复")
				return
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		readFailures = 0
		state := r.Value.Str()
		if state != last {
			fmt.Println("verification_state=" + state)
			last = state
		}
		if state == "ready" {
			if err = xiaohongshu.SaveBrowserSession(p); err != nil {
				fmt.Println("保存登录状态失败")
				return
			}
			fmt.Println("验证已完成，MCP 登录状态已保存")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("等待扫码超时；未确认恢复")
}
