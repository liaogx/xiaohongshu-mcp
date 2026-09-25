// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/liaogx/xiaohongshu-mcp/internal/sessioncmd"
	"github.com/liaogx/xiaohongshu-mcp/pkg/buildinfo"
)

type serverOptions struct {
	headless bool
	port     string
	token    string
}

type cliActions struct {
	serve   func(serverOptions) error
	login   func() error
	recover func(sessioncmd.RecoverOptions) error
}

// runCLI parses before any browser, session, or server initialization. Keeping
// dispatch separate also lets offline tests prove help/errors have no effects.
func runCLI(args []string, stdout, stderr io.Writer, actions cliActions) int {
	command := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}
	if command != "serve" && command != "login" && command != "recover" {
		fmt.Fprintln(stderr, "未知子命令；使用 -help 查看用法。")
		return 2
	}
	flags := flag.NewFlagSet("xiaohongshu-mcp "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintln(stderr, "用法: xiaohongshu-mcp [serve|login|recover] [选项]")
		fmt.Fprintln(stderr, "不指定子命令时启动 MCP 服务；所有入口共用 COOKIES_PATH 和 XHS_SITE。")
		fmt.Fprintln(stderr, "  login             打开扫码登录，完成后退出")
		fmt.Fprintln(stderr, "  recover           打开人工安全验证，完成后退出")
		fmt.Fprintln(stderr, "  recover -fresh    重新扫码，确认前保留原登录文件")
		fmt.Fprintf(stderr, "\n当前 %s 选项:\n", command)
		flags.PrintDefaults()
	}
	showVersion := flags.Bool("version", false, "显示版本信息，不启动浏览器或服务")
	var server serverOptions
	var recovery sessioncmd.RecoverOptions
	switch command {
	case "serve":
		flags.BoolVar(&server.headless, "headless", true, "是否无头模式")
		flags.StringVar(&server.port, "port", ":18060", "监听地址或端口")
		flags.StringVar(&server.token, "token", "", "鉴权 Token，留空则读取 AUTH_TOKEN")
	case "recover":
		flags.StringVar(&recovery.Keyword, "keyword", "咖啡", "用于检查搜索页可用性的关键词")
		flags.BoolVar(&recovery.Fresh, "fresh", false, "在新的专用浏览器会话重新扫码；成功前保留原登录文件")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "存在多余参数；子命令必须放在选项之前。")
		return 2
	}
	if *showVersion {
		fmt.Fprint(stdout, buildinfo.Summary("xiaohongshu-mcp", version))
		return 0
	}
	var err error
	switch command {
	case "serve":
		err = actions.serve(server)
	case "login":
		err = actions.login()
	case "recover":
		err = actions.recover(recovery)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
