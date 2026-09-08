// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/liaogx/xiaohongshu-mcp/humanize"
	"github.com/sirupsen/logrus"
)

// ActionResult 通用动作响应（点赞/收藏等）
type ActionResult struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

const (
	SelectorLikeButton    = ".interact-container .left .like-lottie"
	SelectorCollectButton = ".interact-container .left .reds-icon.collect-icon"
)

// interactActionType 交互动作类型
type interactActionType string

const (
	actionLike       interactActionType = "点赞"
	actionFavorite   interactActionType = "收藏"
	actionUnlike     interactActionType = "取消点赞"
	actionUnfavorite interactActionType = "取消收藏"
)

type interactAction struct {
	page *rod.Page
}

func newInteractAction(page *rod.Page) *interactAction {
	return &interactAction{page: page}
}

func (a *interactAction) preparePage(ctx context.Context, actionType interactActionType, feedID, xsecToken string) (*rod.Page, error) {
	if _, err := prepareNote(ctx, a.page, feedID, xsecToken, false, true); err != nil {
		return nil, err
	}
	return a.page.Context(ctx), nil
}

func (a *interactAction) performClick(page *rod.Page, selector string) error {
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("未找到交互元素 %s: %w", selector, err)
	}
	return humanize.Click(element)
}

// stateOf 从交互状态里取目标字段（点赞取 liked，收藏取 collected）。
type stateOf func(liked, collected bool) bool

// toggleInteract clicks once. A stale snapshot must never cause a second
// toggle, which could undo a successful like.
func (a *interactAction) toggleInteract(ctx context.Context, page *rod.Page, feedID, selector string, want bool, actionType interactActionType, pick stateOf) error {
	if actionType == actionLike || actionType == actionUnlike {
		endpoint := "/note/like"
		if !want {
			endpoint = "/note/dislike"
		}
		observer := observeSubmission(page, endpoint, feedID, "")
		defer observer.close()
		if observer.setupErr != nil {
			return &InteractionError{Stage: "response_observer", State: "not_sent", Cause: observer.setupErr}
		}
		if err := a.performClick(page.Timeout(12*time.Second), selector); err != nil {
			return &InteractionError{Stage: "like_click", State: "unknown", Cause: err}
		}
		r, err := awaitSubmission(page, observer, 15*time.Second)
		logrus.WithFields(logrus.Fields{"feed_id": feedID, "state": r.State, "http_status": r.HTTPStatus, "code": r.Code, "request_field": r.RequestField}).Info("点赞业务回执检查")
		return err
	}
	if err := a.performClick(page.Timeout(12*time.Second), selector); err != nil {
		return &InteractionError{Stage: "interact_click", State: "unknown", Cause: err}
	}
	ok, err := a.waitInteractState(page.Timeout(12*time.Second), feedID, want, pick, 10*time.Second)
	if err != nil {
		return &InteractionError{Stage: "interact_confirm", State: "unknown", Cause: err}
	}
	if !ok {
		return &InteractionError{Stage: "interact_confirm", State: "unknown", Cause: fmt.Errorf("状态尚未确认，不再重复点击")}
	}
	return nil
}

// waitInteractState 轮询 __INITIAL_STATE__ 的交互状态，直到 pick()==want 或超时。
// 状态回写快则立即返回、慢则等满 timeout。
func (a *interactAction) waitInteractState(page *rod.Page, feedID string, want bool, pick stateOf, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		liked, collected, err := a.getInteractState(page, feedID)
		if err != nil {
			return false, err
		}
		if pick(liked, collected) == want {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// LikeAction 负责处理点赞相关交互
type LikeAction struct {
	*interactAction
}

func NewLikeAction(page *rod.Page) *LikeAction {
	return &LikeAction{interactAction: newInteractAction(page)}
}

// Like 点赞指定笔记，如果已点赞则直接返回
func (a *LikeAction) Like(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, true)
}

// LikeOnCurrentPage reuses the page whose comment was just confirmed.
func (a *LikeAction) LikeOnCurrentPage(ctx context.Context, feedID string) error {
	state, err := prepareNote(ctx, a.page, feedID, "", false, false)
	if err != nil {
		return err
	}
	if *state.Liked {
		return nil
	}
	return a.toggleInteract(ctx, a.page.Context(ctx), feedID, SelectorLikeButton, true, actionLike, func(liked, collected bool) bool { return liked })
}

// Unlike 取消点赞指定笔记，如果未点赞则直接返回
func (a *LikeAction) Unlike(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, false)
}

func (a *LikeAction) perform(ctx context.Context, feedID, xsecToken string, targetLiked bool) error {
	actionType := actionLike
	if !targetLiked {
		actionType = actionUnlike
	}

	page, err := a.preparePage(ctx, actionType, feedID, xsecToken)
	if err != nil {
		return err
	}

	liked, _, err := a.getInteractState(page, feedID)
	if err != nil {
		return err
	}

	if targetLiked && liked {
		logrus.Infof("feed %s already liked, skip clicking", feedID)
		return nil
	}
	if !targetLiked && !liked {
		logrus.Infof("feed %s not liked yet, skip clicking", feedID)
		return nil
	}

	return a.toggleInteract(ctx, page, feedID, SelectorLikeButton, targetLiked, actionType,
		func(liked, collected bool) bool { return liked })
}

// FavoriteAction 负责处理收藏相关交互
type FavoriteAction struct {
	*interactAction
}

func NewFavoriteAction(page *rod.Page) *FavoriteAction {
	return &FavoriteAction{interactAction: newInteractAction(page)}
}

// Favorite 收藏指定笔记，如果已收藏则直接返回
func (a *FavoriteAction) Favorite(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, true)
}

// Unfavorite 取消收藏指定笔记，如果未收藏则直接返回
func (a *FavoriteAction) Unfavorite(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, false)
}

func (a *FavoriteAction) perform(ctx context.Context, feedID, xsecToken string, targetCollected bool) error {
	actionType := actionFavorite
	if !targetCollected {
		actionType = actionUnfavorite
	}

	page, err := a.preparePage(ctx, actionType, feedID, xsecToken)
	if err != nil {
		return err
	}

	_, collected, err := a.getInteractState(page, feedID)
	if err != nil {
		return err
	}

	if targetCollected && collected {
		logrus.Infof("feed %s already favorited, skip clicking", feedID)
		return nil
	}
	if !targetCollected && !collected {
		logrus.Infof("feed %s not favorited yet, skip clicking", feedID)
		return nil
	}

	return a.toggleInteract(ctx, page, feedID, SelectorCollectButton, targetCollected, actionType,
		func(liked, collected bool) bool { return collected })
}

// getInteractState requires explicit booleans on the matching note; missing
// or stale data cannot be interpreted as "not yet liked".
func (a *interactAction) getInteractState(page *rod.Page, feedID string) (bool, bool, error) {
	state, err := readNoteReadiness(page, feedID)
	if err != nil {
		return false, false, err
	}
	if state.NoteID != feedID || state.Liked == nil || state.Collected == nil {
		return false, false, fmt.Errorf("目标笔记互动状态未确认，禁止盲点")
	}
	return *state.Liked, *state.Collected, nil
}
