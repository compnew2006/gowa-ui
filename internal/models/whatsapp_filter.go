package models

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppFilterBatchStatus represents the status of a filter batch
type WhatsAppFilterBatchStatus string

const (
	FilterStatusPending    WhatsAppFilterBatchStatus = "pending"
	FilterStatusProcessing WhatsAppFilterBatchStatus = "processing"
	FilterStatusCompleted  WhatsAppFilterBatchStatus = "completed"
	FilterStatusFailed     WhatsAppFilterBatchStatus = "failed"
)

// WhatsAppFilterBatch represents a single phone number filtering campaign/batch
type WhatsAppFilterBatch struct {
	BaseModel
	OrganizationID   uuid.UUID                 `gorm:"type:uuid;index;not null" json:"organization_id"`
	CreatedBy        uuid.UUID                 `gorm:"type:uuid;not null" json:"created_by"`
	
	// Verification Account/Instance Reference
	// Supports whatsmeow (InstanceID) or Meta (WhatsAppAccount)
	WhatsAppAccount  string                    `gorm:"size:100;index" json:"whatsapp_account"` // References WhatsAppAccount.Name or WhatsAppInstance.Name
	InstanceID       *uuid.UUID                `gorm:"type:uuid;index" json:"instance_id,omitempty"` // References WhatsAppInstance.ID
	
	Status           WhatsAppFilterBatchStatus `gorm:"size:20;default:'pending';index" json:"status"`
	TotalNumbers     int                       `gorm:"default:0" json:"total_numbers"`
	ValidNumbers     int                       `gorm:"default:0" json:"valid_numbers"`
	InvalidNumbers   int                       `gorm:"default:0" json:"invalid_numbers"`
	ErrorMessage     string                    `gorm:"type:text" json:"error_message,omitempty"`
	CompletedAt      *time.Time                `json:"completed_at,omitempty"`

	// Relations
	Organization     *Organization             `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Creator          *User                     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Instance         *WhatsAppInstance         `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	Results          []WhatsAppFilterResult    `gorm:"foreignKey:BatchID;constraint:OnDelete:CASCADE" json:"results,omitempty"`
}

func (WhatsAppFilterBatch) TableName() string {
	return "whatsapp_filter_batches"
}

// WhatsAppFilterResult represents a single verified phone number inside a batch
type WhatsAppFilterResult struct {
	BaseModel
	BatchID          uuid.UUID  `gorm:"type:uuid;index;not null" json:"batch_id"`
	PhoneNumber      string     `gorm:"size:50;not null" json:"phone_number"`
	ContactName      string     `gorm:"size:255" json:"contact_name,omitempty"`
	IsValid          bool       `gorm:"default:false;index" json:"is_valid"`
	CheckedAt        *time.Time `json:"checked_at,omitempty"`
	ErrorMessage     string     `gorm:"type:text" json:"error_message,omitempty"`

	// Relations
	Batch            *WhatsAppFilterBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"`
}

func (WhatsAppFilterResult) TableName() string {
	return "whatsapp_filter_results"
}
