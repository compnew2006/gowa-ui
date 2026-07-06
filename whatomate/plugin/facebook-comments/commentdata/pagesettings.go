package commentdata

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/compnew2006/whatomate/internal/models"
)

// AccountResolver resolves a Facebook account ID for a given page within an org.
// It is supplied by callers that have access to the App's account lookup helpers
// (findFacebookAccountByPageID and the active-account fallback). Returning
// uuid.Nil instructs GetOrCreatePageSettings to persist AccountID as uuid.Nil.
type AccountResolver func(db *gorm.DB, orgID uuid.UUID, pageID string) uuid.UUID

// PageSettingsRequest mirrors the historical PUT body for page comment settings.
type PageSettingsRequest struct {
	AutoReplyEnabled        *bool    `json:"auto_reply_enabled"`
	AutoCommentReplyEnabled *bool    `json:"auto_comment_reply_enabled"`
	AutoPrivateReplyEnabled *bool    `json:"auto_private_reply_enabled"`
	AutoCommentReplyTexts   []string `json:"auto_comment_reply_texts"`
	AutoPrivateMessageTexts []string `json:"auto_private_message_texts"`
	OnlyAutoReplyUnanswered *bool    `json:"only_auto_reply_unanswered"`
	WhatsAppNotifyEnabled   *bool    `json:"whatsapp_notify_enabled"`
	WhatsAppInstanceID      *string  `json:"whatsapp_instance_id"`
	WhatsAppNotifyPhone     *string  `json:"whatsapp_notify_phone"`
}

// GetOrCreatePageSettings loads or creates the per-page comment settings row.
// The resolver callback preserves the original account-resolution behavior
// (findFacebookAccountByPageID, then first active OAuth account in the org).
func GetOrCreatePageSettings(db *gorm.DB, orgID uuid.UUID, pageID string, resolveAccount AccountResolver) (*models.FacebookPageCommentSettings, error) {
	var settings models.FacebookPageCommentSettings
	err := db.Where("organization_id = ? AND page_id = ?", orgID, pageID).First(&settings).Error
	if err == nil {
		return &settings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	accountID := uuid.Nil
	if resolveAccount != nil {
		accountID = resolveAccount(db, orgID, pageID)
	}

	settings = models.FacebookPageCommentSettings{
		OrganizationID:          orgID,
		AccountID:               accountID,
		PageID:                  pageID,
		AutoReplyEnabled:        false,
		AutoCommentReplyEnabled: true,
		AutoPrivateReplyEnabled: false,
		AutoCommentReplyTexts:   models.JSONB{"0": DefaultCommentReply},
		AutoPrivateMessageTexts: models.JSONB{"0": DefaultPrivateMessage},
		OnlyAutoReplyUnanswered: true,
		Metadata:                models.JSONB{},
	}
	if err := db.Create(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

// ApplyPageSettingsRequest mutates the supplied settings in place, mirroring the
// historical PUT handler field-by-field semantics.
func ApplyPageSettingsRequest(settings *models.FacebookPageCommentSettings, req PageSettingsRequest) {
	if req.AutoReplyEnabled != nil {
		settings.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	if req.AutoCommentReplyEnabled != nil {
		settings.AutoCommentReplyEnabled = *req.AutoCommentReplyEnabled
	}
	if req.AutoPrivateReplyEnabled != nil {
		settings.AutoPrivateReplyEnabled = *req.AutoPrivateReplyEnabled
	}
	if req.AutoCommentReplyTexts != nil {
		settings.AutoCommentReplyTexts = models.JSONB{}
		for i, t := range req.AutoCommentReplyTexts {
			settings.AutoCommentReplyTexts[fmt.Sprintf("%d", i)] = t
		}
	}
	if req.AutoPrivateMessageTexts != nil {
		settings.AutoPrivateMessageTexts = models.JSONB{}
		for i, t := range req.AutoPrivateMessageTexts {
			settings.AutoPrivateMessageTexts[fmt.Sprintf("%d", i)] = t
		}
	}
	if req.OnlyAutoReplyUnanswered != nil {
		settings.OnlyAutoReplyUnanswered = *req.OnlyAutoReplyUnanswered
	}
	if req.WhatsAppNotifyEnabled != nil {
		settings.WhatsAppNotifyEnabled = *req.WhatsAppNotifyEnabled
	}
	if req.WhatsAppInstanceID != nil {
		if *req.WhatsAppInstanceID == "" {
			settings.WhatsAppInstanceID = nil
		} else if id, err := uuid.Parse(*req.WhatsAppInstanceID); err == nil {
			settings.WhatsAppInstanceID = &id
		}
	}
	if req.WhatsAppNotifyPhone != nil {
		settings.WhatsAppNotifyPhone = *req.WhatsAppNotifyPhone
	}
}
