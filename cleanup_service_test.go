package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/liaogx/xiaohongshu-mcp/xiaohongshu"
	"github.com/stretchr/testify/require"
)

func TestCleanupPlanStoreAndRedactedReport(t *testing.T) {
	t.Setenv("XHS_CLEANUP_STATE_DIR", t.TempDir())
	p := &cleanupPlan{Version: 1, ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AccountID: "fixture-account", Coverage: []xiaohongshu.CleanupCoverage{{Scope: xiaohongshu.CleanupLikes, Status: "partial", Count: 1}}, Entries: []cleanupEntry{{Target: xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupLikes, FeedID: "fixture-note", XsecToken: "fixture-private-token"}, State: "pending"}}}
	path, err := planPath(p.ID)
	require.NoError(t, err)
	require.NoError(t, writeCleanupJSON(path, p))
	fi, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), fi.Mode().Perm())
	loaded, _, err := loadCleanupPlan(p.ID)
	require.NoError(t, err)
	require.Equal(t, p.AccountID, loaded.AccountID)
	raw, err := json.Marshal(reportCleanup(loaded))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "fixture-private-token")
	require.NotContains(t, string(raw), "xsec_token")
	_, err = planPath("../../outside")
	require.Error(t, err)
	loaded.Entries[0].State = "confirmed"
	require.Equal(t, "incomplete", reportCleanup(loaded).Status, "processed visible items is not all history")
	loaded.Coverage[0].Complete = true
	require.Equal(t, "finished", reportCleanup(loaded).Status)
}

func TestCleanupReplayProtectionWithoutBrowser(t *testing.T) {
	t.Setenv("XHS_CLEANUP_STATE_DIR", t.TempDir())
	s := NewXiaohongshuService()
	_, err := s.ExecuteAccountCleanup(context.Background(), CleanupExecuteArgs{PlanID: "invalid"})
	require.ErrorContains(t, err, "CONFIRM_REQUIRED")
	target := xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupSentComments, FeedID: "fixture-note", CommentID: "fixture-comment"}
	p := &cleanupPlan{Version: 1, ID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AccountID: "fixture-account", Entries: []cleanupEntry{{Target: target, State: "pending"}}}
	path, _ := planPath(p.ID)
	require.NoError(t, writeCleanupJSON(path, p))
	for _, state := range []string{"unknown", "rejected"} {
		require.NoError(t, writeCleanupJSON(accountCleanupPath(p.AccountID), cleanupAccountState{Receipts: map[string]cleanupReceipt{cleanupKey(target): {State: state, At: time.Now()}}}))
		r, err := s.ExecuteAccountCleanup(context.Background(), CleanupExecuteArgs{PlanID: p.ID, Confirm: true})
		require.NoError(t, err)
		require.Equal(t, "blocked", r.Status)
		require.Equal(t, state, r.Entries[0].State)
	}
}

func TestCleanupIgnoresLegacyCooldownWithoutBrowser(t *testing.T) {
	t.Setenv("XHS_CLEANUP_STATE_DIR", t.TempDir())
	target := xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupLikes, FeedID: "fixture-note"}
	p := &cleanupPlan{Version: 1, ID: "dddddddddddddddddddddddddddddddd", AccountID: "fixture-account", Coverage: []xiaohongshu.CleanupCoverage{{Scope: xiaohongshu.CleanupLikes, Complete: true}}, Entries: []cleanupEntry{{Target: target, State: "pending"}}}
	path, err := planPath(p.ID)
	require.NoError(t, err)
	require.NoError(t, writeCleanupJSON(path, p))
	// An old ledger may retain a future deadline. Ignore that field while
	// retaining its confirmed receipt, so no browser or duplicate action starts.
	legacy := map[string]any{
		"next_allowed_at": time.Now().Add(time.Hour),
		"receipts":        map[string]cleanupReceipt{cleanupKey(target): {State: "confirmed", At: time.Now()}},
	}
	require.NoError(t, writeCleanupJSON(accountCleanupPath(p.AccountID), legacy))
	s := NewXiaohongshuService()
	status, err := s.GetAccountCleanupStatus(p.ID)
	require.NoError(t, err)
	result, err := s.ExecuteAccountCleanup(context.Background(), CleanupExecuteArgs{PlanID: p.ID, Confirm: true})
	require.NoError(t, err)
	for _, report := range []*CleanupReport{status, result} {
		require.Equal(t, "finished", report.Status)
		require.Equal(t, "confirmed", report.Entries[0].State)
		raw, err := json.Marshal(report)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "next_allowed_at")
	}
	ledger, err := loadAccountCleanupState(p.AccountID)
	require.NoError(t, err)
	raw, err := json.Marshal(ledger)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "next_allowed_at")
	require.Equal(t, "confirmed", ledger.Receipts[cleanupKey(target)].State)
}

func TestCleanupRejectsExtraScopesAndMalformedTargets(t *testing.T) {
	s := NewXiaohongshuService()
	_, err := s.PrepareAccountCleanup(context.Background(), CleanupPrepareArgs{Scopes: []xiaohongshu.CleanupScope{"unknown"}})
	require.ErrorContains(t, err, "INVALID_SCOPE")
	_, err = s.PrepareAccountCleanup(context.Background(), CleanupPrepareArgs{Scopes: []xiaohongshu.CleanupScope{xiaohongshu.CleanupLikes}, Targets: []xiaohongshu.CleanupTarget{{Scope: xiaohongshu.CleanupNotes, FeedID: "fixture-note"}}})
	require.ErrorContains(t, err, "TARGET_SCOPE_MISMATCH")
}

func TestCleanupLockAndCanonicalCommentKey(t *testing.T) {
	t.Setenv("XHS_CLEANUP_STATE_DIR", t.TempDir())
	unlock, err := lockCleanupState()
	require.NoError(t, err)
	_, err = lockCleanupState()
	require.ErrorContains(t, err, "CLEANUP_BUSY")
	unlock()
	unlock, err = lockCleanupState()
	require.NoError(t, err)
	unlock()
	target := xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupSentComments, FeedID: "fixture-note", CommentID: "fixture-comment"}
	key := cleanupKey(target)
	target.Scope = xiaohongshu.CleanupReceivedComments
	require.Equal(t, key, cleanupKey(target), "one comment deletion cannot replay under another scope")
}

func TestCleanupUnconfirmedLoginBlocksNextTarget(t *testing.T) {
	t.Setenv("XHS_CLEANUP_STATE_DIR", t.TempDir())
	p := &cleanupPlan{Version: 1, ID: "cccccccccccccccccccccccccccccccc", AccountID: "fixture-account", Entries: []cleanupEntry{
		{Target: xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupLikes, FeedID: "fixture-first"}, State: "not_sent", Code: "LOGIN_STATUS_UNCONFIRMED"},
		{Target: xiaohongshu.CleanupTarget{Scope: xiaohongshu.CleanupLikes, FeedID: "fixture-next"}, State: "pending"},
	}}
	path, err := planPath(p.ID)
	require.NoError(t, err)
	require.NoError(t, writeCleanupJSON(path, p))
	r, err := NewXiaohongshuService().ExecuteAccountCleanup(context.Background(), CleanupExecuteArgs{PlanID: p.ID, Confirm: true})
	require.NoError(t, err)
	require.Equal(t, "blocked", r.Status)
	require.Equal(t, "LOGIN_STATUS_UNCONFIRMED", r.ErrorCode)
	require.Equal(t, "pending", r.Entries[1].State)
}
