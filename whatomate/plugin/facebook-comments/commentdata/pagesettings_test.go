package commentdata

import (
	"testing"

	"github.com/google/uuid"

	"github.com/compnew2006/whatomate/internal/models"
)

func TestApplyPageSettingsRequestMapsFields(t *testing.T) {
	settings := &models.FacebookPageCommentSettings{}
	autoReply := true
	commentReply := true
	privateReply := true
	unanswered := false
	waEnabled := true
	waInstance := uuid.New().String()
	waPhone := "+966500000000"

	ApplyPageSettingsRequest(settings, PageSettingsRequest{
		AutoReplyEnabled:        &autoReply,
		AutoCommentReplyEnabled: &commentReply,
		AutoPrivateReplyEnabled: &privateReply,
		AutoCommentReplyTexts:   []string{"reply-a", "reply-b"},
		AutoPrivateMessageTexts: []string{"pm-a"},
		OnlyAutoReplyUnanswered: &unanswered,
		WhatsAppNotifyEnabled:   &waEnabled,
		WhatsAppInstanceID:      &waInstance,
		WhatsAppNotifyPhone:     &waPhone,
	})

	if !settings.AutoReplyEnabled || !settings.AutoCommentReplyEnabled || !settings.AutoPrivateReplyEnabled {
		t.Fatalf("bool flags not applied: %+v", settings)
	}
	if settings.AutoCommentReplyTexts["0"] != "reply-a" || settings.AutoCommentReplyTexts["1"] != "reply-b" {
		t.Fatalf("AutoCommentReplyTexts = %v", settings.AutoCommentReplyTexts)
	}
	if settings.AutoPrivateMessageTexts["0"] != "pm-a" {
		t.Fatalf("AutoPrivateMessageTexts = %v", settings.AutoPrivateMessageTexts)
	}
	if settings.OnlyAutoReplyUnanswered {
		t.Fatal("OnlyAutoReplyUnanswered should be false")
	}
	if !settings.WhatsAppNotifyEnabled {
		t.Fatal("WhatsAppNotifyEnabled should be true")
	}
	if settings.WhatsAppInstanceID == nil || settings.WhatsAppInstanceID.String() != waInstance {
		t.Fatalf("WhatsAppInstanceID = %v", settings.WhatsAppInstanceID)
	}
	if settings.WhatsAppNotifyPhone != waPhone {
		t.Fatalf("WhatsAppNotifyPhone = %q", settings.WhatsAppNotifyPhone)
	}
}

func TestApplyPageSettingsRequestClearsInstanceOnEmptyString(t *testing.T) {
	settings := &models.FacebookPageCommentSettings{}
	id := uuid.New()
	settings.WhatsAppInstanceID = &id
	empty := ""

	ApplyPageSettingsRequest(settings, PageSettingsRequest{WhatsAppInstanceID: &empty})

	if settings.WhatsAppInstanceID != nil {
		t.Fatalf("WhatsAppInstanceID should be nil, got %v", settings.WhatsAppInstanceID)
	}
}
