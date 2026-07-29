package models

import (
	"time"

	"github.com/google/uuid"
)

// ScheduledMessageStatus represents the lifecycle state of a scheduled message.
type ScheduledMessageStatus string

const (
	ScheduledMessageStatusPending    ScheduledMessageStatus = "pending"
	ScheduledMessageStatusProcessing ScheduledMessageStatus = "processing"
	ScheduledMessageStatusSent       ScheduledMessageStatus = "sent"
	ScheduledMessageStatusFailed     ScheduledMessageStatus = "failed"
	ScheduledMessageStatusCancelled  ScheduledMessageStatus = "cancelled"
)

// ScheduledMessage represents an outgoing message queued to be sent at a
// future time. Rows are claimed by the ScheduledMessageProcessor via an
// atomic pending→processing UPDATE and dispatched through the unified
// SendOutgoingMessage path, so all message types behave exactly like an
// immediate send. Media is stored as a local path (MediaURL) only — the
// sender re-reads the file from disk at fire time.
type ScheduledMessage struct {
	BaseModel
	OrganizationID  uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount string    `gorm:"size:100;index;not null" json:"whatsapp_account"` // References WhatsAppAccount.Name
	ContactID       uuid.UUID `gorm:"type:uuid;index;not null" json:"contact_id"`

	// Message payload — mirrors the fields SendOutgoingMessage consumes.
	MessageType    MessageType `gorm:"size:20;not null" json:"message_type"`
	Content        string      `gorm:"type:text" json:"content"` // Body text, or caption for media
	MediaURL       string      `gorm:"type:text" json:"media_url"`
	MediaMimeType  string      `gorm:"size:100" json:"media_mime_type"`
	MediaFilename  string      `gorm:"size:255" json:"media_filename"`
	TemplateID     *uuid.UUID  `gorm:"type:uuid" json:"template_id,omitempty"`
	TemplateParams JSONB       `gorm:"type:jsonb;default:'{}'" json:"template_params"`

	// Scheduling state. ScheduledAt is stored in UTC.
	ScheduledAt   time.Time              `gorm:"index;not null" json:"scheduled_at"`
	Status        ScheduledMessageStatus `gorm:"size:20;default:'pending';index" json:"status"`
	SentMessageID *uuid.UUID             `gorm:"type:uuid" json:"sent_message_id,omitempty"` // Message row created at fire time
	ErrorMessage  string                 `gorm:"type:text" json:"error_message"`
	CreatedBy     uuid.UUID              `gorm:"type:uuid;not null" json:"created_by"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Contact      *Contact      `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
	Template     *Template     `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Creator      *User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (ScheduledMessage) TableName() string {
	return "scheduled_messages"
}
