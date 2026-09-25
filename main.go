// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"fmt"
	"os"

	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/configs"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/internal/sessioncmd"
	"github.com/liaogx/xiaohongshu-mcp/pkg/buildinfo"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
	"github.com/sirupsen/logrus"
)

// version is also used by /health and MCP serverInfo; releases stamp all three.
var version = buildinfo.DefaultVersion

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, cliActions{
		serve: startService,
		login: func() error { return sessioncmd.Login(os.Stdout) },
		recover: func(options sessioncmd.RecoverOptions) error {
			return sessioncmd.Recover(options, os.Stdout)
		},
	}))
}

func startService(options serverOptions) error {
	if err := xiaohongshu.ValidateSiteConfig(); err != nil {
		return err
	}
	token := options.token
	if token == "" {
		token = os.Getenv("AUTH_TOKEN")
	}

	logrus.Infof("xiaohongshu-mcp version: %s", version)

	// 只用内置浏览器。启动时就备好，缺它直接退出，不拖到第一个请求才失败。
	binPath, err := browser.EnsureBrowser()
	if err != nil {
		return err
	}
	logrus.Infof("using browser binary: %s", binPath)

	configs.InitHeadless(options.headless)
	// 入口层解析出 seed 和代理，经 configs 透传给浏览器工厂。
	// seed 取值：环境变量 > 会话文件 > 新生成并写回，保证同一账号每次启动一致。
	configs.SetFingerprintSeed(configs.ResolveFingerprintSeed(
		cookies.NewLoadCookie(cookies.GetCookiesFilePath())))
	configs.SetProxy(configs.ProxyFromEnv())

	// 初始化服务
	xiaohongshuService := NewXiaohongshuService()

	// 创建并启动应用服务器
	appServer := NewAppServer(xiaohongshuService, token)
	if err := appServer.Start(options.port); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}
	return nil
}
