// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// Legacy source-only entry point. Releases use xiaohongshu-mcp login.
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
	showVersion := flag.Bool("version", false, "显示版本和构建信息，不打开登录窗口")
	flag.Parse()
	if *showVersion {
		fmt.Print(buildinfo.Summary("xiaohongshu-login", version))
		return
	}
	if err := sessioncmd.Login(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
