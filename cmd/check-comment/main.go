// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

// check-comment tests page readiness and draft input without clicking submit.
// Read a JSON array of {feed_id,xsec_token,content} from stdin; credentials and
// draft text are never printed. It uses the service's cookie/proxy settings.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/liaogx/xiaohongshu-mcp/browser"
	"github.com/liaogx/xiaohongshu-mcp/configs"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
	"github.com/sirupsen/logrus"
)

func main() {
	rounds := flag.Int("rounds", 1, "只读/草稿检查轮数（1—5）；永不提交")
	diagnostics := flag.String("diagnostics-dir", "", "可选：把失败截图保存到私有目录（不应指向源码仓库）")
	flag.Parse()
	if *rounds < 1 || *rounds > 5 {
		logrus.Fatal("rounds must be 1..5")
	}
	var inputs []struct {
		FeedID  string `json:"feed_id"`
		Token   string `json:"xsec_token"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&inputs); err != nil {
		logrus.Fatal("invalid input JSON")
	}
	store := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	seed := configs.ResolveFingerprintSeed(store)
	for round := 1; round <= *rounds; round++ {
		for i, in := range inputs {
			started := time.Now()
			b := browser.NewBrowser(true, browser.WithFingerprintSeed(seed), browser.WithProxy(configs.ProxyFromEnv()))
			p := b.NewPage()
			draft, err := xiaohongshu.NewCommentFeedAction(p).PrepareComment(context.Background(), in.FeedID, in.Token, in.Content)
			result := map[string]any{"round": round, "item": i + 1, "ready": draft != nil, "elapsed_ms": time.Since(started).Milliseconds(), "submitted": false}
			if err != nil {
				result["error"] = err.Error()
				if v, e := p.Timeout(2 * time.Second).Eval(`()=>[...document.querySelectorAll('div.bottom button.submit')].map(e=>{const r=e.getBoundingClientRect(),hit=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);return {class:e.className,disabled:e.disabled,rect:{x:r.x,y:r.y,width:r.width,height:r.height},hit:hit?.tagName,hitClass:hit?.className,inside:hit===e||e.contains(hit),display:getComputedStyle(e).display}})`); e == nil {
					result["submit_buttons"] = v.Value.Val()
				}
				if *diagnostics != "" {
					if os.MkdirAll(*diagnostics, 0700) == nil {
						if data, e := p.Timeout(3*time.Second).Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.PageCaptureScreenshotFormatPng}); e == nil {
							_ = os.WriteFile(filepath.Join(*diagnostics, "check-comment.png"), data, 0600)
						}
					}
				}
			}
			p.Close()
			b.Close()
			json.NewEncoder(os.Stdout).Encode(result)
			if err != nil && (strings.Contains(err.Error(), "SECURITY_VERIFICATION_REQUIRED") || strings.Contains(err.Error(), "AUTH_REQUIRED")) {
				return
			}
		}
	}
}
