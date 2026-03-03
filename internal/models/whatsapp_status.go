package models

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppStatus represents a WhatsApp Status (story) message.
// Statuses expire after 24 hours and are scoped to an organization and instance.
type WhatsAppStatus struct {
	BaseModel
	OrganizationID    uuid.UUID          `gorm:"type:uuid;index;not null" json:"organization_id"`
	InstanceID        uuid.UUID          `gorm:"type:uuid;index;not null" json:"instance_id"`
	WhatsAppAccount   string             `gorm:"column:whats_app_account;size:100;not null;default:'';index" json:"-"`
	SenderJID         string             `gorm:"column:sender_jid;size:255;index;not null" json:"sender_jid"`
	SenderName        string             `gorm:"size:255" json:"sender_name"`
	WhatsAppMessageID string             `gorm:"column:whats_app_message_id;size:255;index:idx_status_instance_wamid,unique,where:whats_app_message_id <> ''" json:"whatsapp_message_id"`
	StatusType        WhatsAppStatusType `gorm:"size:20;index;not null" json:"status_type"`
	Content           string             `gorm:"type:text" json:"content"`
	MediaURL          string             `gorm:"type:text" json:"media_url"`
	MediaMimeType     string             `gorm:"size:128" json:"media_mime_type"`
	MediaFilename     string             `gorm:"size:255" json:"media_filename"`
	TextARGB          *int64             `gorm:"column:text_argb" json:"text_argb,omitempty"`
	BackgroundARGB    *int64             `gorm:"column:background_argb" json:"background_argb,omitempty"`
	Font              string             `gorm:"size:64" json:"font"`
	SeenAt            *time.Time         `gorm:"type:timestamptz" json:"seen_at,omitempty"`
	ExpiresAt         time.Time          `gorm:"type:timestamptz;index;not null" json:"expires_at"`
	Metadata          JSONB              `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relations
	Organization *Organization     `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Instance     *WhatsAppInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

func (WhatsAppStatus) TableName() string {
	return "whatsapp_statuses"
}
