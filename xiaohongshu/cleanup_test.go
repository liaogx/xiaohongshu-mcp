package xiaohongshu

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupTargetAndSubmissionIdentity(t *testing.T) {
	for _, bad := range []CleanupTarget{{Scope: CleanupNotes}, {Scope: CleanupNotes, FeedID: `fixture"] button`}, {Scope: CleanupSentComments, FeedID: "fixture-note"}, {Scope: "unknown", FeedID: "fixture-note"}} {
		require.Error(t, bad.Validate())
	}
	for _, scope := range CleanupScopes {
		require.NoError(t, (CleanupTarget{Scope: scope, FeedID: "fixture-note", CommentID: "fixture-comment", UserID: "fixture-user", ConversationID: "fixture-conversation"}).Validate())
	}
	fields := map[string]string{"noteId": "fixture-note", "commentId": "fixture-comment"}
	require.True(t, cleanupRequestMatches(`{"noteId":"fixture-note","commentId":"fixture-comment"}`, fields))
	for _, body := range []string{`{}`, `{"noteId":"fixture-note"}`, `{"noteId":"fixture-other","commentId":"fixture-comment"}`, `{"noteId":"fixture-note","commentId":"fixture-other"}`} {
		require.False(t, cleanupRequestMatches(body, fields))
	}
	require.False(t, cleanupRequestMatches(`{"noteIds":["fixture-note","fixture-other"]}`, map[string]string{"noteIds": "fixture-note"}))
}

func TestCleanupResultNeverAssumesSuccessfulDeletion(t *testing.T) {
	require.Equal(t, "unknown", CleanupResultState(fmt.Errorf("timeout")))
	require.Equal(t, "rejected", CleanupResultState(&InteractionError{State: "rejected", Cause: fmt.Errorf("denied")}))
	require.True(t, CleanupMustStop(fmt.Errorf("RATE_LIMITED")))
	require.Equal(t, "ACCOUNT_RESTRICTED", CleanupErrorCode(fmt.Errorf("ACCOUNT_RESTRICTED: unavailable")))
}

func TestCleanupCommentRequestAliases(t *testing.T) {
	for _, body := range []string{`{"noteId":"fixture-note","commentId":"fixture-comment"}`, `{"note_id":"fixture-note","comment_id":"fixture-comment"}`} {
		require.True(t, cleanupCommentRequestMatches(body, "fixture-note", "fixture-comment"))
	}
	for _, body := range []string{
		`{}`, `{"note_id":"fixture-note"}`, `{"noteId":"fixture-note","comment_id":"fixture-comment"}`,
		`{"note_id":"fixture-note","comment_id":"other"}`,
		`{"noteId":"fixture-note","commentId":"fixture-comment","comment_id":"other"}`,
		`{"note_id":"fixture-note","comment_id":["fixture-comment"]}`,
	} {
		require.False(t, cleanupCommentRequestMatches(body, "fixture-note", "fixture-comment"))
	}
}
