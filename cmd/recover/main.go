// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// Legacy source-only entry point. Releases use xiaohongshu-mcp recover.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/liaogx/xiaohongshu-mcp/internal/sessioncmd"
	"github.com/liaogx/xiaohongshu-mcp/pkg/buildinfo"
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
	if err := sessioncmd.Recover(sessioncmd.RecoverOptions{Keyword: *keyword, Fresh: *fresh}, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
