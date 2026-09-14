package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/cookies"
	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
)

var cleanupPlanID = regexp.MustCompile(`^[a-f0-9]{32}$`)

type cleanupManager struct{ mu sync.Mutex }

type CleanupPrepareArgs struct {
	Scopes   []xiaohongshu.CleanupScope  `json:"scopes,omitempty" jsonschema:"清理范围：notes|sent_comments|favorites|likes|received_comments|groups|following|messages；省略为盘点全部八类"`
	Targets  []xiaohongshu.CleanupTarget `json:"targets,omitempty" jsonschema:"补充已知目标 ID；仅作为候选，执行前核验当前状态和所有权。不要把历史清单当成账号全部内容"`
	MaxItems int                         `json:"max_items,omitempty" jsonschema:"单类盘点上限，默认200，最大1000；达到上限标记覆盖不完整"`
}

type CleanupExecuteArgs struct {
	PlanID  string `json:"plan_id" jsonschema:"prepare_account_cleanup 返回的计划 ID"`
	Confirm bool   `json:"confirm" jsonschema:"明确授权后传 true。每次最多执行一个目标，结果明确后可继续下一项，无固定本地等待；验证、限流或不确定结果时停止"`
}

type CleanupStatusArgs struct {
	PlanID string `json:"plan_id"`
}

type CleanupDiscoverArgs struct {
	FeedID    string `json:"feed_id"`
	XsecToken string `json:"xsec_token"`
	Received  bool   `json:"received,omitempty" jsonschema:"false 查找本账号发表的评论；true 查找本账号笔记收到的评论，不包括其他笔记的回复通知"`
	Limit     int    `json:"limit,omitempty" jsonschema:"单篇加载上限，默认200，最大1000；不代表账号全部历史"`
}

type cleanupEntry struct {
	Target    xiaohongshu.CleanupTarget `json:"target"`
	State     string                    `json:"state"`
	Code      string                    `json:"code,omitempty"`
	UpdatedAt time.Time                 `json:"updated_at,omitempty"`
}

type cleanupPlan struct {
	Version     int                           `json:"version"`
	ID          string                        `json:"plan_id"`
	AccountID   string                        `json:"account_id"`
	CreatedAt   time.Time                     `json:"created_at"`
	Coverage    []xiaohongshu.CleanupCoverage `json:"coverage"`
	Entries     []cleanupEntry                `json:"entries"`
	BlockedCode string                        `json:"blocked_code,omitempty"`
}

type cleanupAccountState struct {
	Receipts map[string]cleanupReceipt `json:"receipts"`
}

type cleanupReceipt struct {
	State string    `json:"state"`
	Code  string    `json:"code,omitempty"`
	At    time.Time `json:"at"`
}

// Reports contain IDs and outcomes, never access tokens or private chat text.
type CleanupReport struct {
	PlanID    string                        `json:"plan_id"`
	AccountID string                        `json:"account_id"`
	Status    string                        `json:"status"`
	Coverage  []xiaohongshu.CleanupCoverage `json:"coverage"`
	Entries   []CleanupReportEntry          `json:"entries"`
	ErrorCode string                        `json:"error_code,omitempty"`
}
type CleanupReportEntry struct {
	Scope xiaohongshu.CleanupScope `json:"scope"`
	ID    string                   `json:"id"`
	Title string                   `json:"title,omitempty"`
	State string                   `json:"state"`
	Code  string                   `json:"code,omitempty"`
}

func cleanupDir() string {
	if p := os.Getenv("XHS_CLEANUP_STATE_DIR"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(cookies.GetCookiesFilePath()), "state", "account-cleanup")
}

// A separate process sharing the same private state must not dispatch a
// second cleanup. A crash leaves the lock in place for manual inspection.
func lockCleanupState() (func(), error) {
	dir := cleanupDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "execution.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("CLEANUP_BUSY: state is locked; inspect a stale lock only after all cleanup processes stop")
	}
	fi, err := f.Stat()
	f.Close()
	if err != nil {
		return nil, err
	}
	return func() {
		if current, err := os.Lstat(path); err == nil && os.SameFile(fi, current) {
			os.Remove(path)
		}
	}, nil
}

func writeCleanupJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if fi, err := os.Lstat(path); err == nil && (!fi.Mode().IsRegular() || fi.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("invalid cleanup state file")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".cleanup-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = json.NewEncoder(f).Encode(v); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func readCleanupJSON(path string, v any) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() || fi.Size() > 16<<20 {
		return fmt.Errorf("invalid cleanup state file")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(io.LimitReader(f, 16<<20)).Decode(v)
}

func planPath(id string) (string, error) {
	if !cleanupPlanID.MatchString(id) {
		return "", fmt.Errorf("INVALID_PLAN_ID")
	}
	return filepath.Join(cleanupDir(), "plan-"+id+".json"), nil
}
func accountCleanupPath(id string) string {
	h := sha256.Sum256([]byte(id))
	return filepath.Join(cleanupDir(), "account-"+hex.EncodeToString(h[:])+".json")
}
func cleanupKey(t xiaohongshu.CleanupTarget) string {
	scope := string(t.Scope)
	if t.Scope == xiaohongshu.CleanupReceivedComments {
		scope = string(xiaohongshu.CleanupSentComments)
	}
	h := sha256.Sum256([]byte(scope + "\x00" + t.ID()))
	return hex.EncodeToString(h[:])
}

func reportCleanup(p *cleanupPlan) *CleanupReport {
	out := &CleanupReport{PlanID: p.ID, AccountID: p.AccountID, Status: "finished", Coverage: p.Coverage, Entries: []CleanupReportEntry{}}
	incomplete := false
	for _, c := range p.Coverage {
		if !c.Complete {
			incomplete = true
		}
	}
	for _, e := range p.Entries {
		out.Entries = append(out.Entries, CleanupReportEntry{Scope: e.Target.Scope, ID: e.Target.ID(), Title: e.Target.Title, State: e.State, Code: e.Code})
		if e.State == "pending" {
			out.Status = "pending"
		}
		if e.State != "confirmed" && e.State != "already_clear" {
			incomplete = true
		}
	}
	if out.Status == "finished" && incomplete {
		out.Status = "incomplete"
	}
	if p.BlockedCode != "" {
		out.Status, out.ErrorCode = "blocked", p.BlockedCode
	}
	return out
}

func normalizedCleanupScopes(in []xiaohongshu.CleanupScope) ([]xiaohongshu.CleanupScope, error) {
	if len(in) == 0 {
		return append([]xiaohongshu.CleanupScope(nil), xiaohongshu.CleanupScopes...), nil
	}
	out := []xiaohongshu.CleanupScope{}
	seen := map[xiaohongshu.CleanupScope]bool{}
	for _, s := range in {
		found := false
		for _, known := range xiaohongshu.CleanupScopes {
			if s == known {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("INVALID_SCOPE")
		}
		if !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	return out, nil
}

func (s *XiaohongshuService) PrepareAccountCleanup(ctx context.Context, args CleanupPrepareArgs) (*CleanupReport, error) {
	if !s.cleanup.mu.TryLock() {
		return nil, fmt.Errorf("CLEANUP_BUSY")
	}
	defer s.cleanup.mu.Unlock()
	scopes, err := normalizedCleanupScopes(args.Scopes)
	if err != nil {
		return nil, err
	}
	if len(args.Targets) > 1000 {
		return nil, fmt.Errorf("TOO_MANY_TARGETS: maximum 1000")
	}
	allowed := map[xiaohongshu.CleanupScope]bool{}
	for _, scope := range scopes {
		allowed[scope] = true
	}
	for _, t := range args.Targets {
		if err := t.Validate(); err != nil {
			return nil, err
		}
		if !allowed[t.Scope] {
			return nil, fmt.Errorf("TARGET_SCOPE_MISMATCH")
		}
	}
	b := newBrowser()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()
	inv, err := xiaohongshu.NewCleanupAction(page).Inventory(ctx, scopes, args.MaxItems)
	blockedCode := ""
	if err != nil {
		s.handleSecurityVerification(err, "")
		if inv == nil || inv.AccountID == "" {
			return nil, err
		}
		blockedCode = xiaohongshu.CleanupErrorCode(err)
		for _, scope := range scopes[len(inv.Coverage):] {
			inv.Coverage = append(inv.Coverage, xiaohongshu.CleanupCoverage{Scope: scope, Status: "blocked", Count: -1, Reason: blockedCode})
		}
	}
	var raw [16]byte
	if _, err = rand.Read(raw[:]); err != nil {
		return nil, err
	}
	plan := &cleanupPlan{Version: 1, ID: hex.EncodeToString(raw[:]), AccountID: inv.AccountID, CreatedAt: time.Now(), Coverage: inv.Coverage, Entries: []cleanupEntry{}}
	plan.BlockedCode = blockedCode
	seen := map[string]bool{}
	for _, t := range append(inv.Targets, args.Targets...) {
		if err = t.Validate(); err != nil {
			return nil, err
		}
		key := cleanupKey(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		state, code := "pending", ""
		if t.Scope == xiaohongshu.CleanupGroups || t.Scope == xiaohongshu.CleanupMessages {
			state, code = "manual_required", "MANUAL_REQUIRED"
		}
		plan.Entries = append(plan.Entries, cleanupEntry{Target: t, State: state, Code: code})
	}
	path, _ := planPath(plan.ID)
	if err = writeCleanupJSON(path, plan); err != nil {
		return nil, err
	}
	return reportCleanup(plan), nil
}

func loadCleanupPlan(id string) (*cleanupPlan, string, error) {
	path, err := planPath(id)
	if err != nil {
		return nil, "", err
	}
	var p cleanupPlan
	if err = readCleanupJSON(path, &p); err != nil {
		return nil, "", fmt.Errorf("PLAN_NOT_FOUND_OR_INVALID")
	}
	if p.Version != 1 || p.ID != id || p.AccountID == "" {
		return nil, "", fmt.Errorf("PLAN_NOT_FOUND_OR_INVALID")
	}
	return &p, path, nil
}

func (s *XiaohongshuService) GetAccountCleanupStatus(id string) (*CleanupReport, error) {
	s.cleanup.mu.Lock()
	defer s.cleanup.mu.Unlock()
	p, _, err := loadCleanupPlan(id)
	if err != nil {
		return nil, err
	}
	state, err := loadAccountCleanupState(p.AccountID)
	if err != nil {
		return nil, err
	}
	for i := range p.Entries {
		if r, ok := state.Receipts[cleanupKey(p.Entries[i].Target)]; ok {
			p.Entries[i].State = r.State
			p.Entries[i].Code = r.Code
		}
	}
	return reportCleanup(p), nil
}

func loadAccountCleanupState(account string) (*cleanupAccountState, error) {
	v := &cleanupAccountState{Receipts: map[string]cleanupReceipt{}}
	err := readCleanupJSON(accountCleanupPath(account), v)
	if os.IsNotExist(err) {
		return v, nil
	}
	if err != nil {
		return nil, fmt.Errorf("CLEANUP_STATE_UNREADABLE")
	}
	if v.Receipts == nil {
		return nil, fmt.Errorf("CLEANUP_STATE_UNREADABLE")
	}
	return v, nil
}

func (s *XiaohongshuService) ExecuteAccountCleanup(ctx context.Context, args CleanupExecuteArgs) (*CleanupReport, error) {
	if !args.Confirm {
		return nil, fmt.Errorf("CONFIRM_REQUIRED")
	}
	if !s.cleanup.mu.TryLock() {
		return nil, fmt.Errorf("CLEANUP_BUSY")
	}
	defer s.cleanup.mu.Unlock()
	unlock, err := lockCleanupState()
	if err != nil {
		return nil, err
	}
	defer unlock()
	p, path, err := loadCleanupPlan(args.PlanID)
	if err != nil {
		return nil, err
	}
	ledger, err := loadAccountCleanupState(p.AccountID)
	if err != nil {
		return nil, err
	}
	for i := range p.Entries {
		if r, ok := ledger.Receipts[cleanupKey(p.Entries[i].Target)]; ok {
			p.Entries[i].State = r.State
			p.Entries[i].Code = r.Code
		}
	}
	out := reportCleanup(p)
	if p.BlockedCode != "" {
		return out, nil
	}
	// A timeout, rejection, or account access gate needs a new inspection,
	// not a loop that silently advances to the next destructive target.
	for _, receipt := range ledger.Receipts {
		if receipt.State == "unknown" || receipt.State == "rejected" {
			out.Status = "blocked"
			out.ErrorCode = "PREVIOUS_RESULT_REQUIRES_REVIEW"
			return out, nil
		}
	}
	for _, e := range p.Entries {
		if e.State == "not_sent" && xiaohongshu.CleanupMustStop(fmt.Errorf("%s", e.Code)) {
			out.Status = "blocked"
			out.ErrorCode = e.Code
			return out, nil
		}
	}
	index := -1
	for i, e := range p.Entries {
		if e.State == "pending" {
			index = i
			break
		}
	}
	if index < 0 {
		return out, nil
	}
	b := newBrowser()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()
	entry := &p.Entries[index]
	prepared, err := xiaohongshu.NewCleanupAction(page).Prepare(ctx, p.AccountID, entry.Target)
	if err != nil {
		entry.State = "not_sent"
		entry.Code = xiaohongshu.CleanupErrorCode(err)
		entry.UpdatedAt = time.Now()
		if entry.Code == "MANUAL_REQUIRED" {
			entry.State = "manual_required"
		}
		if saveErr := writeCleanupJSON(path, p); saveErr != nil {
			return nil, saveErr
		}
		s.handleSecurityVerificationTarget(err, cleanupTargetURL(entry.Target), "", entry.Target.FeedID != "" && entry.Target.Scope != xiaohongshu.CleanupNotes)
		out = reportCleanup(p)
		out.Status = "blocked"
		out.ErrorCode = entry.Code
		return out, nil
	}
	key := cleanupKey(entry.Target)
	if prepared.AlreadyClear {
		entry.State = "already_clear"
		entry.Code = ""
	} else {
		// This durable marker is written before the first destructive click.
		// Unknown/rejected operations are never automatically replayed by a new plan.
		ledger.Receipts[key] = cleanupReceipt{State: "unknown", At: time.Now()}
		if err = writeCleanupJSON(accountCleanupPath(p.AccountID), ledger); err != nil {
			return nil, err
		}
		err = prepared.Execute(ctx)
		entry.State = xiaohongshu.CleanupResultState(err)
		entry.Code = xiaohongshu.CleanupErrorCode(err)
		if err != nil {
			s.handleSecurityVerificationTarget(err, cleanupTargetURL(entry.Target), "", entry.Target.FeedID != "" && entry.Target.Scope != xiaohongshu.CleanupNotes)
		}
	}
	entry.UpdatedAt = time.Now()
	ledger.Receipts[key] = cleanupReceipt{State: entry.State, Code: entry.Code, At: entry.UpdatedAt}
	if saveErr := writeCleanupJSON(accountCleanupPath(p.AccountID), ledger); saveErr != nil {
		return nil, saveErr
	}
	if saveErr := writeCleanupJSON(path, p); saveErr != nil {
		return nil, saveErr
	}
	out = reportCleanup(p)
	if err != nil {
		out.Status = "blocked"
		out.ErrorCode = entry.Code
	}
	return out, nil
}

func cleanupTargetURL(t xiaohongshu.CleanupTarget) string {
	if t.Scope == xiaohongshu.CleanupNotes {
		return "https://creator.xiaohongshu.com/new/note-manager"
	}
	if t.Scope == xiaohongshu.CleanupFollowing {
		return "https://www.xiaohongshu.com/user/profile/" + t.UserID
	}
	// Tokens stay inside the private plan/browser, never in the output report.
	return xiaohongshu.FeedDetailURL(t.FeedID, t.XsecToken)
}

func (s *XiaohongshuService) DiscoverCleanupComments(ctx context.Context, args CleanupDiscoverArgs) (*xiaohongshu.CleanupDiscovery, error) {
	if !s.cleanup.mu.TryLock() {
		return nil, fmt.Errorf("CLEANUP_BUSY")
	}
	defer s.cleanup.mu.Unlock()
	b := newBrowser()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()
	r, err := xiaohongshu.NewCleanupAction(page).DiscoverComments(ctx, args.FeedID, args.XsecToken, args.Received, args.Limit)
	if err != nil {
		s.handleSecurityVerificationTarget(err, xiaohongshu.FeedDetailURL(args.FeedID, args.XsecToken), "", true)
	}
	return r, err
}
