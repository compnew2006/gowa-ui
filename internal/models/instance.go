package models

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppInstance represents a whatsmeow WhatsApp connection
type WhatsAppInstance struct {
	BaseModel
	OrganizationID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name             string         `gorm:"not null" json:"name"`
	PhoneNumber      string         `json:"phone_number"`
	JID              string         `gorm:"column:jid;index:idx_whatsapp_instances_j_id,unique,where:jid <> ''" json:"jid"`
	Status           InstanceStatus `gorm:"type:varchar(50);default:'disconnected'" json:"status"`
	IsDefault        bool           `gorm:"default:false" json:"is_default"`
	SessionID        string         `json:"session_id"`
	AutoReadReceipt  bool           `gorm:"default:false" json:"auto_read_receipt"`
	Settings         JSONB          `gorm:"type:jsonb;default:'{}'" json:"settings"`
	LastConnectedAt  *time.Time     `json:"last_connected_at"`
	SendBlockedUntil *time.Time     `gorm:"type:timestamptz" json:"send_blocked_until,omitempty"`
	SendBlockReason  string         `gorm:"type:text;not null;default:''" json:"send_block_reason,omitempty"`

	// Relations
	Organization Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (WhatsAppInstance) TableName() string {
	return "whatsapp_instances"
}

// InstanceNotification represents a notification for an instance event
type InstanceNotification struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	InstanceID     uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	EventType      string    `gorm:"not null" json:"event_type"` // e.g., "ban", "logout"
	Message        string    `json:"message"`
	IsDismissed    bool      `gorm:"default:false" json:"is_dismissed"`
	ContactID      *uuid.UUID `gorm:"type:uuid;index" json:"contact_id,omitempty"`
	Metadata       JSONB      `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Relations
	Organization Organization     `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Instance     WhatsAppInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

func (InstanceNotification) TableName() string {
	return "instance_notifications"
}
