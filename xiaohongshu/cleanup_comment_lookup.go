package xiaohongshu

import (
	"context"
	"time"

	"github.com/go-rod/rod"
)

// Look through loaded top-level rows before expanding unrelated reply trees.
// This is only a locator: deletion still requires the exact comment ID and
// current-account ownership proof in prepareDeleteComment.
func findCleanupCommentElement(ctx context.Context, page *rod.Page, commentID string) (*rod.Element, error) {
	deadline := time.Now().Add(45 * time.Second)
	previous, stagnant := -1, 0
	for round := 0; round < 80 && time.Now().Before(deadline); round++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := page.GetContext().Err(); err != nil {
			return nil, err
		}
		if err := cleanupPageGuard(page); err != nil {
			return nil, err
		}
		if el := lookupComment(page, commentID, ""); el != nil {
			return el, nil
		}
		parents, err := page.Elements(".parent-comment")
		if err != nil {
			return nil, err
		}
		if checkEndContainer(page) {
			break
		}
		if len(parents) == previous {
			stagnant++
		} else {
			previous, stagnant = len(parents), 0
		}
		if stagnant >= 5 {
			break
		}
		if len(parents) > 0 {
			if err := parents[len(parents)-1].ScrollIntoView(); err != nil {
				return nil, err
			}
		}
		humanScroll(ctx, page, "normal", false, 1)
		if err := cleanupPause(ctx, 700*time.Millisecond); err != nil {
			return nil, err
		}
	}
	// Nested comments still use the existing bounded expansion search.
	return findCommentElement(ctx, page, commentID, "")
}
