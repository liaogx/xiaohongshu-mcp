// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"crypto/sha256"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/cookies"
)

// Only one short-lived, exclusively owned page is kept for comment -> like.
// A cookie change invalidates it. The timer is synchronized with operations,
// so it cannot close a page while a request is using it.
type interactionPageCache struct {
	mu    sync.Mutex
	entry *interactionPage
}
type interactionPage struct {
	feedID      string
	credentials [32]byte
	page        *rod.Page
	close       func()
	timer       *time.Timer
}

func credentialDigest() ([32]byte, error) {
	b, err := os.ReadFile(cookies.GetCookiesFilePath())
	return sha256.Sum256(b), err
}

// Call these helpers while holding mu.
func (c *interactionPageCache) clear() {
	if c.entry != nil {
		c.entry.timer.Stop()
		c.entry.close()
		c.entry = nil
	}
}
func (c *interactionPageCache) keep(feedID string, p *rod.Page, closePage func(), digest [32]byte) {
	c.clear()
	e := &interactionPage{feedID: feedID, page: p, close: closePage, credentials: digest}
	e.timer = time.AfterFunc(90*time.Second, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.entry == e {
			c.clear()
		}
	})
	c.entry = e
}
func (c *interactionPageCache) take(feedID string) *interactionPage {
	digest, err := credentialDigest()
	if c.entry == nil {
		return nil
	}
	if err != nil || c.entry.feedID != feedID || c.entry.credentials != digest {
		c.clear()
		return nil
	}
	e := c.entry
	c.entry = nil
	e.timer.Stop()
	return e
}
