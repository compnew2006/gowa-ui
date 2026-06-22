package commentdata

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
)

func TestApplySettingsRequestNormalizesUserValues(t *testing.T) {
	replyText := "  public reply  "
	privateText := "  private reply  "
	postLimit := -1
	commentsPerPost := 999
	settings := &models.FacebookCommentSettings{}

	ApplySettingsRequest(settings, SettingsRequest{
		AutoCommentReplyText:       &replyText,
		AutoPrivateMessageText:     &privateText,
		DefaultSyncPostLimit:       &postLimit,
		DefaultSyncCommentsPerPost: &commentsPerPost,
	})

	if settings.AutoCommentReplyText != "public reply" {
		t.Fatalf("AutoCommentReplyText = %q", settings.AutoCommentReplyText)
	}
	if settings.AutoPrivateMessageText != "private reply" {
		t.Fatalf("AutoPrivateMessageText = %q", settings.AutoPrivateMessageText)
	}
	if settings.DefaultSyncPostLimit != DefaultPostLimit {
		t.Fatalf("DefaultSyncPostLimit = %d, want %d", settings.DefaultSyncPostLimit, DefaultPostLimit)
	}
	if settings.DefaultSyncCommentsPerPost != MaxCommentsPerPost {
		t.Fatalf("DefaultSyncCommentsPerPost = %d, want %d", settings.DefaultSyncCommentsPerPost, MaxCommentsPerPost)
	}
}
