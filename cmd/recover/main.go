// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// recover opens the MCP-owned browser for manual login or security verification.
// It never solves challenges or submits comments. Credentials stay on this device.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/sirupsen/logrus"
)

func main() {
	keyword := flag.String("keyword", "咖啡", "用于检查搜索页可用性的关键词")
	flag.Parse()
	logrus.SetLevel(logrus.ErrorLevel)
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	b := browser.NewBrowser(false, browser.WithFingerprintSeed(store.LoadSeed()), browser.WithProxy(os.Getenv("XHS_PROXY")))
	defer b.Close()
	p := b.NewPage().Timeout(8 * time.Minute)
	defer p.Close()
	if err := p.Navigate("https://www.xiaohongshu.com/search_result?keyword=" + url.QueryEscape(*keyword) + "&source=web_explore_feed"); err != nil {
		fmt.Println("打开验证页面失败；未修改登录文件")
		return
	}
	last := ""
	readFailures := 0
	deadline := time.Now().Add(7 * time.Minute)
	for time.Now().Before(deadline) {
		r, err := p.Eval(`() => {
			if (/captcha|verify/.test(location.pathname) || /安全验证/.test(document.title)) return 'challenge';
			const s=window.__INITIAL_STATE__, u=s&&s.user, ref=u&&u.userInfo;
			const info=ref&&(ref.value!==undefined?ref.value:ref._value!==undefined?ref._value:ref);
			if(info && !info.guest && (info.userId||info.user_id) && s.search) return 'ready';
			if(document.querySelector('.login-container')) return 'login';
			return 'loading';
		}`)
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
			cks, err := p.Browser().GetCookies()
			if err != nil {
				fmt.Println("读取登录状态失败")
				return
			}
			data, err := json.Marshal(cks)
			if err != nil {
				fmt.Println("编码登录状态失败")
				return
			}
			if err = store.SaveCookies(data); err != nil {
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
