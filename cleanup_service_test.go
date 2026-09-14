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
	ledger := cleanupAccountState{NextAllowedAt: time.Now().Add(time.Minute), Receipts: map[string]cleanupReceipt{}}
	require.NoError(t, writeCleanupJSON(accountCleanupPath(p.AccountID), ledger))
	r, err := s.ExecuteAccountCleanup(context.Background(), CleanupExecuteArgs{PlanID: p.ID, Confirm: true})
	require.NoError(t, err)
	require.Equal(t, "waiting_interval", r.Status)
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
