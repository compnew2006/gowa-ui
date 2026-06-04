package models

import (
	"time"

	"github.com/google/uuid"
)

// BulkMessageCampaign represents a bulk message campaign
type BulkMessageCampaign struct {
	BaseModel
	OrganizationID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount      string         `gorm:"size:100;index;not null" json:"whatsapp_account"` // References WhatsAppAccount.Name
	Name                 string         `gorm:"size:255;not null" json:"name"`
	TemplateID           uuid.UUID      `gorm:"type:uuid;not null" json:"template_id"`
	HeaderMediaID        string         `gorm:"type:text" json:"header_media_id"`         // Meta media ID (from uploaded media)
	HeaderMediaFilename  string         `gorm:"type:text" json:"header_media_filename"`   // Original filename
	HeaderMediaMimeType  string         `gorm:"type:text" json:"header_media_mime_type"`  // MIME type (image/jpeg, video/mp4, etc.)
	HeaderMediaLocalPath string         `gorm:"type:text" json:"header_media_local_path"` // Local file path for preview
	MinDelaySeconds      int            `gorm:"default:0" json:"min_delay_seconds"`       // Minimum delay between recipients in seconds
	MaxDelaySeconds      int            `gorm:"default:0" json:"max_delay_seconds"`       // Maximum delay between recipients in seconds
	Status               CampaignStatus `gorm:"size:20;default:'draft'" json:"status"`    // draft, queued, processing, completed, failed
	TotalRecipients      int            `gorm:"default:0" json:"total_recipients"`
	SentCount            int            `gorm:"default:0" json:"sent_count"`
	DeliveredCount       int            `gorm:"default:0" json:"delivered_count"`
	ReadCount            int            `gorm:"default:0" json:"read_count"`
	FailedCount          int            `gorm:"default:0" json:"failed_count"`
	ScheduledAt          *time.Time     `json:"scheduled_at,omitempty"`
	StartedAt            *time.Time     `json:"started_at,omitempty"`
	CompletedAt          *time.Time     `json:"completed_at,omitempty"`
	CreatedBy            uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`

	// Relations
	Organization *Organization          `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Template     *Template              `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Creator      *User                  `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Recipients   []BulkMessageRecipient `gorm:"foreignKey:CampaignID" json:"recipients,omitempty"`
}

func (BulkMessageCampaign) TableName() string {
	return "bulk_message_campaigns"
}

// RecipientType distinguishes individual vs group campaign targets.
const (
	RecipientTypeIndividual = "individual"
	RecipientTypeGroup      = "group"
)

// BulkMessageRecipient represents a recipient in a bulk message campaign
type BulkMessageRecipient struct {
	BaseModel
	CampaignID        uuid.UUID     `gorm:"type:uuid;index;not null;uniqueIndex:idx_bulk_recipients_campaign_phone_normalized,priority:1" json:"campaign_id"`
	PhoneNumber       string        `gorm:"size:50;not null" json:"phone_number"`
	PhoneNormalized   string        `gorm:"size:32;default:'';uniqueIndex:idx_bulk_recipients_campaign_phone_normalized,priority:2" json:"phone_normalized,omitempty"`
	RecipientName     string        `gorm:"size:255" json:"recipient_name"`
	TemplateParams    JSONB         `gorm:"type:jsonb;default:'{}'" json:"template_params"`
	Status            MessageStatus `gorm:"size:20;default:'pending'" json:"status"` // pending, sent, delivered, read, failed
	WhatsAppMessageID string        `gorm:"column:whats_app_message_id;size:100;index" json:"whatsapp_message_id,omitempty"`
	MessageID         *uuid.UUID    `gorm:"type:uuid" json:"message_id,omitempty"`
	ErrorMessage      string        `gorm:"type:text" json:"error_message"`
	SentAt            *time.Time    `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time    `json:"delivered_at,omitempty"`
	ReadAt            *time.Time    `json:"read_at,omitempty"`
	RecipientType     string        `gorm:"size:20;index;default:'individual'" json:"recipient_type"` // individual, group
	GroupJID          string        `gorm:"size:100;index" json:"group_jid,omitempty"`               // e.g. "1203631234567-1601234567@g.us"
	GroupName         string        `gorm:"size:255" json:"group_name,omitempty"`
	ParticipantCount  int           `json:"participant_count"`
	SendAttemptID     string        `gorm:"size:36;index;default:''" json:"send_attempt_id,omitempty"`

	// Relations
	Campaign *BulkMessageCampaign `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
	Message  *Message             `gorm:"foreignKey:MessageID" json:"message,omitempty"`
}

func (BulkMessageRecipient) TableName() string {
	return "bulk_message_recipients"
}

// NotificationRule defines automated notification rules
type NotificationRule struct {
	BaseModel
	OrganizationID   uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount  string    `gorm:"size:100;index;not null" json:"whatsapp_account"` // References WhatsAppAccount.Name
	Name             string    `gorm:"size:255;not null" json:"name"`
	IsEnabled        bool      `gorm:"default:true" json:"is_enabled"`
	TriggerType      string    `gorm:"size:50;not null" json:"trigger_type"` // webhook, scheduler, api
	TriggerConfig    JSONB     `gorm:"type:jsonb;not null" json:"trigger_config"`
	TemplateID       uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`
	FieldMappings    JSONB     `gorm:"type:jsonb;default:'{}'" json:"field_mappings"`
	Conditions       JSONB     `gorm:"type:jsonb;default:'{}'" json:"conditions"`
	AttachmentConfig JSONB     `gorm:"type:jsonb" json:"attachment_config"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Template     *Template     `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
}

func (NotificationRule) TableName() string {
	return "notification_rules"
}
