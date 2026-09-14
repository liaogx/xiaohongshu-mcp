package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

// Read the current website, never an operations-history directory. The same
// snapshot is shared by the sent/received scopes within a single inventory.
func (a *CleanupAction) notificationCommentTargets(ctx context.Context, account string, scope CleanupScope, limit int) ([]CleanupTarget, error) {
	if a.notificationSnapshot == nil {
		p := a.page.Context(ctx).Timeout(90 * time.Second)
		if err := p.Navigate("https://www.xiaohongshu.com/notification"); err != nil {
			return nil, err
		}
		if err := p.Timeout(20 * time.Second).Wait(rod.Eval(`()=>window.__INITIAL_STATE__?.notification?.notificationMap?.mentions?.messageList?.length>0 || !!document.querySelector('.login-container')?.getClientRects().length || /\/website-login\//.test(location.pathname)`)); err != nil {
			if gate := cleanupPageGuard(p); gate != nil {
				return nil, gate
			}
			return nil, fmt.Errorf("PAGE_UNCONFIRMED: notification rows not readable; no history coverage inferred")
		}
		if err := cleanupPageGuard(p); err != nil {
			return nil, err
		}
		me, err := NewLogin(p).CurrentUser(ctx)
		if err != nil || me.UserID != account {
			return nil, fmt.Errorf("ACCOUNT_MISMATCH: notification account not confirmed")
		}
		n := NewNotificationAction(p)
		if err := n.loadUntil(ctx, p, TabMentions, limit); err != nil {
			return nil, err
		}
		if err := cleanupPageGuard(p); err != nil {
			return nil, err
		}
		payload, err := n.readTab(p, TabMentions)
		if err != nil {
			return nil, err
		}
		a.notificationSnapshot = payload
	}
	return notificationCleanupTargets(account, a.notificationSnapshot.MessageList, scope, limit), nil
}

func notificationCleanupTargets(account string, rows []rawNotification, scope CleanupScope, limit int) []CleanupTarget {
	items := []CleanupTarget{}
	seen := map[string]bool{}
	if account == "" || limit <= 0 {
		return items
	}
	for _, row := range rows {
		if row.Item.Type != itemTypeNote || row.Item.Illegal.Status != statusNormal {
			continue
		}
		var comment *rawComment
		switch scope {
		case CleanupSentComments:
			// Even a removed reply can still quote a live original comment.
			// Check the original's own visibility and author, not the replier.
			comment = row.Comment.TargetComment
			if comment == nil || comment.UserInfo.UserID != account {
				continue
			}
		case CleanupReceivedComments:
			if row.Item.UserInfo.UserID != account || row.from().UserID == "" || row.from().UserID == account {
				continue
			}
			comment = &row.Comment
		default:
			return items
		}
		if comment.Illegal.Status != statusNormal {
			continue
		}
		target := CleanupTarget{Scope: scope, FeedID: row.Item.ID, CommentID: comment.ID, XsecToken: row.Item.XsecToken, Title: row.Item.Content}
		if target.Validate() != nil || seen[target.ID()] {
			continue
		}
		seen[target.ID()] = true
		items = append(items, target)
		if len(items) >= limit {
			break
		}
	}
	return items
}
