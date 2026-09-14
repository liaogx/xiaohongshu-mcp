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
	require.Equal(t, "NOTE_NOT_READY", CleanupErrorCode(fmt.Errorf("INTERACTION_NOT_SENT: stage=ready; NOTE_NOT_READY")))
	require.Equal(t, "NOTE_UNAVAILABLE", CleanupErrorCode(fmt.Errorf("NOTE_UNAVAILABLE: unavailable note")))
	require.Equal(t, "AUTH_REQUIRED", CleanupErrorCode(fmt.Errorf("AUTH_REQUIRED; NOTE_NOT_READY")))
	require.Equal(t, "LOGIN_STATUS_UNCONFIRMED", CleanupErrorCode(fmt.Errorf("LOGIN_STATUS_UNCONFIRMED: account data missing")))
	require.True(t, CleanupMustStop(cleanupLoginStatusError(fmt.Errorf("current user not found in page state"))))
	require.Equal(t, "RATE_LIMITED", CleanupErrorCode(cleanupLoginStatusError(fmt.Errorf("RATE_LIMITED: unavailable"))))
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
