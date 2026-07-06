package commentdata

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	DefaultPostLimit       = 25
	DefaultCommentsPerPost = 50
	MaxPostLimit           = 100
	MaxCommentsPerPost     = 100
	DefaultCommentReply    = "تم الرد خاص"
	DefaultPrivateMessage  = "اهلا كيف اقدر اساعدك"
)

type SettingsRequest struct {
	Enabled                    *bool   `json:"enabled"`
	SyncEnabled                *bool   `json:"sync_enabled"`
	AutoReplyEnabled           *bool   `json:"auto_reply_enabled"`
	AutoCommentReplyEnabled    *bool   `json:"auto_comment_reply_enabled"`
	AutoPrivateReplyEnabled    *bool   `json:"auto_private_reply_enabled"`
	AutoCommentReplyText       *string `json:"auto_comment_reply_text"`
	AutoPrivateMessageText     *string `json:"auto_private_message_text"`
	OnlyAutoReplyUnanswered    *bool   `json:"only_auto_reply_unanswered"`
	IgnorePageAdminComments    *bool   `json:"ignore_page_admin_comments"`
	DefaultSyncPostLimit       *int    `json:"default_sync_post_limit"`
	DefaultSyncCommentsPerPost *int    `json:"default_sync_comments_per_post"`
}

func GetOrCreateSettings(db *gorm.DB, orgID uuid.UUID) (*models.FacebookCommentSettings, error) {
	var settings models.FacebookCommentSettings
	err := db.Where("organization_id = ?", orgID).First(&settings).Error
	if err == nil {
		NormalizeSettings(&settings)
		return &settings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	settings = models.FacebookCommentSettings{
		OrganizationID:             orgID,
		Enabled:                    true,
		SyncEnabled:                true,
		AutoReplyEnabled:           false,
		AutoCommentReplyEnabled:    true,
		AutoPrivateReplyEnabled:    true,
		AutoCommentReplyText:       DefaultCommentReply,
		AutoPrivateMessageText:     DefaultPrivateMessage,
		OnlyAutoReplyUnanswered:    true,
		IgnorePageAdminComments:    true,
		DefaultSyncPostLimit:       DefaultPostLimit,
		DefaultSyncCommentsPerPost: DefaultCommentsPerPost,
		Metadata:                   models.JSONB{},
	}
	if err := db.Create(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func ApplySettingsRequest(settings *models.FacebookCommentSettings, req SettingsRequest) {
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.SyncEnabled != nil {
		settings.SyncEnabled = *req.SyncEnabled
	}
	if req.AutoReplyEnabled != nil {
		settings.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	if req.AutoCommentReplyEnabled != nil {
		settings.AutoCommentReplyEnabled = *req.AutoCommentReplyEnabled
	}
	if req.AutoPrivateReplyEnabled != nil {
		settings.AutoPrivateReplyEnabled = *req.AutoPrivateReplyEnabled
	}
	if req.AutoCommentReplyText != nil {
		settings.AutoCommentReplyText = strings.TrimSpace(*req.AutoCommentReplyText)
	}
	if req.AutoPrivateMessageText != nil {
		settings.AutoPrivateMessageText = strings.TrimSpace(*req.AutoPrivateMessageText)
	}
	if req.OnlyAutoReplyUnanswered != nil {
		settings.OnlyAutoReplyUnanswered = *req.OnlyAutoReplyUnanswered
	}
	if req.IgnorePageAdminComments != nil {
		settings.IgnorePageAdminComments = *req.IgnorePageAdminComments
	}
	if req.DefaultSyncPostLimit != nil {
		settings.DefaultSyncPostLimit = clampPositive(*req.DefaultSyncPostLimit, DefaultPostLimit, MaxPostLimit)
	}
	if req.DefaultSyncCommentsPerPost != nil {
		settings.DefaultSyncCommentsPerPost = clampPositive(*req.DefaultSyncCommentsPerPost, DefaultCommentsPerPost, MaxCommentsPerPost)
	}
	NormalizeSettings(settings)
}

func NormalizeSettings(settings *models.FacebookCommentSettings) {
	if strings.TrimSpace(settings.AutoCommentReplyText) == "" {
		settings.AutoCommentReplyText = DefaultCommentReply
	}
	if strings.TrimSpace(settings.AutoPrivateMessageText) == "" {
		settings.AutoPrivateMessageText = DefaultPrivateMessage
	}
	settings.DefaultSyncPostLimit = clampPositive(settings.DefaultSyncPostLimit, DefaultPostLimit, MaxPostLimit)
	settings.DefaultSyncCommentsPerPost = clampPositive(settings.DefaultSyncCommentsPerPost, DefaultCommentsPerPost, MaxCommentsPerPost)
}

func clampPositive(value, fallback, maxValue int) int {
	if value <= 0 {
		return fallback
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
