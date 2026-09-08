// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// InteractionError distinguishes a failure before dispatch from an ambiguous
// submission. In particular, missing DOM text is NOT proof of rejection.
type InteractionError struct {
	Stage string
	State string // not_sent, rejected, unknown
	Cause error
}

func (e *InteractionError) Error() string {
	return fmt.Sprintf("INTERACTION_%s: stage=%s; %v", strings.ToUpper(e.State), e.Stage, e.Cause)
}
func (e *InteractionError) Unwrap() error { return e.Cause }

type interactionReceipt struct {
	State        string `json:"state"`
	CommentID    string `json:"comment_id,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
	Code         string `json:"code,omitempty"`
	Reason       string `json:"reason,omitempty"`
	RequestField string `json:"request_field,omitempty"`
}

// Observe the request made by the normal page UI. Never replay requests or
// save request headers, signed URLs, cookies, or the response body.
type submissionObserver struct {
	mu       sync.Mutex
	result   interactionReceipt
	cancel   context.CancelFunc
	done     chan struct{}
	setupErr error
}

func observeSubmission(page *rod.Page, endpoint, feedID, content string) *submissionObserver {
	ctx, cancel := context.WithCancel(page.GetContext())
	p := page.Context(ctx)
	o := &submissionObserver{cancel: cancel, done: make(chan struct{})}
	// Rod restores domains using the listener's canceled context. Chrome may
	// process Network.disable while Rod's cached state still says "enabled".
	// Explicitly enable on the LIVE context before every listener, and keep
	// the domain enabled until the page closes. This matters when a new MCP
	// request reuses the comment page for a like.
	if err := (proto.NetworkEnable{}).Call(p); err != nil {
		o.setupErr = err
		cancel()
		close(o.done)
		return o
	}
	type pendingResponse struct {
		status int
		field  string
	}
	requests := map[proto.NetworkRequestID]pendingResponse{}
	wait := p.EachEvent(
		func(e *proto.NetworkRequestWillBeSent) {
			if e.Request.Method != "POST" {
				return
			}
			u, err := url.Parse(e.Request.URL)
			if err != nil || !strings.HasSuffix(u.Path, endpoint) {
				return
			}
			body := e.Request.PostData
			if body == "" {
				r, err := (proto.NetworkGetRequestPostData{RequestID: e.RequestID}).Call(p)
				if err != nil {
					return
				}
				body = r.PostData
			}
			if !matchesSubmission(body, feedID, content) {
				return
			}
			requests[e.RequestID] = pendingResponse{field: submissionTargetField(body, feedID)}
			o.mu.Lock()
			o.result.State = "unknown"
			o.mu.Unlock()
		},
		func(e *proto.NetworkResponseReceived) {
			if req, ok := requests[e.RequestID]; ok {
				req.status = e.Response.Status
				requests[e.RequestID] = req
			}
		},
		func(e *proto.NetworkLoadingFinished) {
			req, ok := requests[e.RequestID]
			if !ok {
				return
			}
			delete(requests, e.RequestID)
			body, err := (proto.NetworkGetResponseBody{RequestID: e.RequestID}).Call(p)
			r := interactionReceipt{State: "unknown", HTTPStatus: req.status}
			if err == nil {
				data := []byte(body.Body)
				if body.Base64Encoded {
					data, err = base64.StdEncoding.DecodeString(body.Body)
				}
				if err == nil {
					r = parseSubmissionResponse(req.status, data, feedID, content)
				}
			}
			r.RequestField = req.field
			o.mu.Lock()
			o.result = r
			o.mu.Unlock()
		},
		func(e *proto.NetworkLoadingFailed) {
			if _, ok := requests[e.RequestID]; !ok {
				return
			}
			delete(requests, e.RequestID)
			o.mu.Lock()
			o.result = interactionReceipt{State: "unknown", Reason: "网络请求未完成，不能据此重发"}
			o.mu.Unlock()
		},
	)
	go func() { defer close(o.done); wait() }()
	return o
}

func (o *submissionObserver) close() { o.cancel(); <-o.done }
func (o *submissionObserver) snapshot() interactionReceipt {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.result
}

func matchesSubmission(body, feedID, content string) bool {
	var v map[string]any
	if json.Unmarshal([]byte(body), &v) != nil {
		return false
	}
	if submissionTargetField(body, feedID) == "" {
		return false
	}
	if content != "" {
		text, _ := v["content"].(string)
		return text == content
	}
	return true
}

func submissionTargetField(body, feedID string) string {
	var v map[string]any
	if json.Unmarshal([]byte(body), &v) != nil {
		return ""
	}
	for _, key := range []string{"note_id", "noteId", "note_oid"} {
		if id, ok := v[key].(string); ok && id == feedID {
			return key
		}
	}
	return ""
}

func parseSubmissionResponse(status int, body []byte, feedID, content string) interactionReceipt {
	r := interactionReceipt{State: "unknown", HTTPStatus: status}
	var v struct {
		Success *bool           `json:"success"`
		Code    json.RawMessage `json:"code"`
		Msg     string          `json:"msg"`
		Message string          `json:"message"`
		Data    struct {
			ID      string `json:"id"`
			Comment struct {
				ID      string `json:"id"`
				Content string `json:"content"`
				NoteID  string `json:"note_id"`
			} `json:"comment"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &v) != nil {
		return r
	}
	// Only expose short numeric business codes, never arbitrary server text.
	code := strings.Trim(string(v.Code), `"`)
	if len(code) <= 12 && code != "" && strings.Trim(code, "-0123456789") == "" {
		r.Code = code
	}
	msg := v.Msg + " " + v.Message
	for _, label := range []string{"安全验证", "登录", "操作频繁", "评论过于频繁", "禁止评论", "评论已关闭", "内容违规"} {
		if strings.Contains(msg, label) {
			r.Reason = label
			break
		}
	}
	if v.Success != nil && !*v.Success || r.Code != "" && r.Code != "0" {
		r.State = "rejected"
		return r
	}
	if status < 200 || status >= 300 {
		return r
	}
	if v.Success == nil && r.Code != "0" {
		return r
	}
	if content != "" {
		c := v.Data.Comment
		if c.Content != "" && c.Content != content || c.NoteID != "" && c.NoteID != feedID {
			return r
		}
		r.CommentID = c.ID
		if r.CommentID == "" {
			r.CommentID = v.Data.ID
		}
		// code=0 alone without a success flag or a comment identifier is not
		// enough to interpret an unfamiliar payload as a posted comment.
		if v.Success == nil && r.CommentID == "" {
			return r
		}
	}
	r.State = "confirmed"
	return r
}

func awaitSubmission(page *rod.Page, observer *submissionObserver, timeout time.Duration) (interactionReceipt, error) {
	ctx, cancel := context.WithTimeout(page.GetContext(), timeout)
	defer cancel()
	p := page.Context(ctx)
	for {
		r := observer.snapshot()
		if r.State == "confirmed" {
			return r, nil
		}
		if probe, err := readSearchPageProbe(p.Timeout(time.Second)); err == nil {
			if err := probe.accessError(); err != nil {
				return r, &InteractionError{Stage: "confirm", State: "unknown", Cause: err}
			}
		}
		if r.State == "rejected" {
			cause := fmt.Errorf("平台拒绝提交（HTTP=%d, code=%s, reason=%s）；不要自动重复提交", r.HTTPStatus, r.Code, r.Reason)
			if r.Reason == "安全验证" {
				return r, &InteractionError{Stage: "confirm", State: "rejected", Cause: &PageAccessError{Code: "SECURITY_VERIFICATION_REQUIRED"}}
			}
			if r.Reason == "登录" {
				return r, &InteractionError{Stage: "confirm", State: "rejected", Cause: &PageAccessError{Code: "AUTH_REQUIRED"}}
			}
			return r, &InteractionError{Stage: "confirm", State: "rejected", Cause: cause}
		}
		select {
		case <-ctx.Done():
			return r, &InteractionError{Stage: "confirm", State: "unknown", Cause: fmt.Errorf("未取得可靠的业务回执；不要重发、不要点赞，需先核验: %w", ctx.Err())}
		case <-time.After(200 * time.Millisecond):
		}
	}
}
