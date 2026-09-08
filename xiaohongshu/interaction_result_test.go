// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubmissionResponseEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body, state, id string
		status                int
	}{
		{"business success", `{"success":true,"code":0,"data":{"comment":{"id":"fixture-comment","content":"fixture text","note_id":"fixture-note"}}}`, "confirmed", "fixture-comment", 200},
		{"flag sufficient", `{"success":true,"code":0}`, "confirmed", "", 200},
		{"id with zero code", `{"code":0,"data":{"comment":{"id":"fixture-comment"}}}`, "confirmed", "fixture-comment", 200},
		{"HTTP alone", `{}`, "unknown", "", 200},
		{"zero code only", `{"code":0}`, "unknown", "", 200},
		{"explicit rejection", `{"success":false,"code":-1,"msg":"操作频繁"}`, "rejected", "", 200},
		{"contradictory flags", `{"success":true,"code":-1}`, "rejected", "", 200},
		{"wrong note", `{"success":true,"data":{"comment":{"id":"fixture-comment","note_id":"another-note"}}}`, "unknown", "", 200},
		{"wrong text", `{"success":true,"data":{"comment":{"id":"fixture-comment","content":"another text"}}}`, "unknown", "", 200},
		{"server error", `{"success":true}`, "unknown", "", 503},
		{"html challenge", `<title>安全验证</title>`, "unknown", "", 200},
		{"no arbitrary echo", `{"success":false,"code":"untrusted server text","msg":"untrusted server text"}`, "rejected", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := parseSubmissionResponse(tc.status, []byte(tc.body), "fixture-note", "fixture text")
			require.Equal(t, tc.state, r.State)
			require.Equal(t, tc.id, r.CommentID)
			require.NotContains(t, r.Code+r.Reason, "untrusted server text")
		})
	}
}

func TestSubmissionRequestCorrelation(t *testing.T) {
	require.True(t, matchesSubmission(`{"note_id":"fixture-note","content":"fixture text"}`, "fixture-note", "fixture text"))
	require.False(t, matchesSubmission(`{"note_id":"different-note","content":"fixture text"}`, "fixture-note", "fixture text"))
	require.False(t, matchesSubmission(`{"note_id":"fixture-note","content":"different text"}`, "fixture-note", "fixture text"))
	require.True(t, matchesSubmission(`{"note_id":"fixture-note"}`, "fixture-note", ""))
	require.True(t, matchesSubmission(`{"note_oid":"fixture-note"}`, "fixture-note", ""))
	require.False(t, matchesSubmission(`{"note_oid":"other-note"}`, "fixture-note", ""))
	require.False(t, matchesSubmission(`{"success":true}`, "fixture-note", ""))
}

func TestCommentReceiptGuardsConcurrentAndRestartedCalls(t *testing.T) {
	t.Setenv("XHS_INTERACTION_STATE_DIR", t.TempDir())
	p := commentReceiptPath("fixture-user", "fixture-note", "fixture text")
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if beginCommentReceipt(p) == nil {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, winners.Load())
	require.ErrorContains(t, beginCommentReceipt(p), "DUPLICATE_OR_UNCERTAIN_COMMENT")
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Contains(t, string(data), `"state":"unknown"`)
	require.NotContains(t, string(data), "fixture text")
	require.NoError(t, finishCommentReceipt(p, interactionReceipt{State: "confirmed", CommentID: "fixture-comment"}))
	require.Error(t, beginCommentReceipt(p))
	info, err := os.Stat(p)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.NotEqual(t, p, commentReceiptPath("other-user", "fixture-note", "fixture text"))
	require.NotEqual(t, p, commentReceiptPath("fixture-user", "other-note", "fixture text"))
	require.Equal(t, filepath.Dir(p), os.Getenv("XHS_INTERACTION_STATE_DIR"))
}
