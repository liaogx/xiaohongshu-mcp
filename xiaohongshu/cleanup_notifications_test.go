package xiaohongshu

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func notificationReplyFixture(replyID, targetID, targetOwner string) rawNotification {
	return rawNotification{
		Type: "comment/comment", UserInfo: rawUser{UserID: "fixture-replier"},
		Item:    rawItem{ID: "fixture-note", Type: itemTypeNote, Content: "Synthetic note", XsecToken: "fixture-access", Illegal: rawIllegal{Status: statusNormal}, UserInfo: rawUser{UserID: "fixture-note-owner"}},
		Comment: rawComment{ID: replyID, Content: "A reply", Illegal: rawIllegal{Status: statusNormal}, TargetComment: &rawComment{ID: targetID, Content: "Original comment", UserInfo: rawUser{UserID: targetOwner}, Illegal: rawIllegal{Status: statusNormal}}},
	}
}

func TestNotificationCleanupUsesOwnedOriginalNotReceivedReply(t *testing.T) {
	a := notificationReplyFixture("reply-one", "own-original", "fixture-account")
	b := notificationReplyFixture("reply-two", "own-original", "fixture-account")
	removedReply := notificationReplyFixture("reply-three", "own-second", "fixture-account")
	removedReply.Comment.Illegal.Status = "DELETE"
	otherOwner := notificationReplyFixture("reply-four", "other-original", "someone-else")
	unknownOwner := notificationReplyFixture("reply-five", "unknown-original", "")
	removedOriginal := notificationReplyFixture("reply-six", "removed-original", "fixture-account")
	removedOriginal.Comment.TargetComment.Illegal.Status = "DELETE"
	removedNote := notificationReplyFixture("reply-seven", "unreadable-original", "fixture-account")
	removedNote.Item.Illegal.Status = "DELETE"
	items := notificationCleanupTargets("fixture-account", []rawNotification{a, b, removedReply, otherOwner, unknownOwner, removedOriginal, removedNote}, CleanupSentComments, 20)
	require.Len(t, items, 2)
	require.Equal(t, "own-original", items[0].CommentID)
	require.Equal(t, "own-second", items[1].CommentID)
	require.Equal(t, "fixture-access", items[0].XsecToken)
	require.Equal(t, "fixture-note", items[0].FeedID)
	require.Len(t, notificationCleanupTargets("fixture-account", []rawNotification{a, removedReply}, CleanupSentComments, 1), 1)
}

func TestReceivedCleanupRequiresOwnNoteNotOnlyNotification(t *testing.T) {
	elsewhere := notificationReplyFixture("reply-elsewhere", "own-original", "fixture-account")
	owned := notificationReplyFixture("received-own-note", "own-original", "fixture-account")
	owned.Item.UserInfo.UserID = "fixture-account"
	items := notificationCleanupTargets("fixture-account", []rawNotification{elsewhere, owned}, CleanupReceivedComments, 20)
	require.Len(t, items, 1)
	require.Equal(t, "received-own-note", items[0].CommentID)
	require.Empty(t, notificationCleanupTargets("", []rawNotification{owned}, CleanupReceivedComments, 20))
}

func TestNotificationQuotedCommentMapping(t *testing.T) {
	const fixture = `{"type":"comment/comment","commentInfo":{"id":"fixture-reply","content":"A reply","illegalInfo":{"illegalStatus":"NORMAL"},"targetComment":{"id":"fixture-original","content":"Original comment","userInfo":{"userid":"fixture-account","nickname":"Fixture"},"illegalInfo":{"illegalStatus":"NORMAL"}}},"itemInfo":{"id":"fixture-note","type":"note_info","illegalInfo":{"illegalStatus":"NORMAL"}}}`
	var raw rawNotification
	require.NoError(t, json.Unmarshal([]byte(fixture), &raw))
	items, _ := convertNotifications([]rawNotification{raw}, 20)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].TargetComment)
	require.Equal(t, "fixture-original", items[0].TargetComment.ID)
	require.Equal(t, "fixture-account", items[0].TargetComment.Author.UserID)
	require.Equal(t, "fixture-reply", items[0].CommentID)
	raw.Comment.TargetComment.Illegal.Status = "DELETE"
	items, _ = convertNotifications([]rawNotification{raw}, 20)
	require.Nil(t, items[0].TargetComment, "do not expose deleted quoted text")
}
