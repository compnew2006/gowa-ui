package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

const (
	CannedResponseAttachmentTypeImage = "image"
	CannedResponseAttachmentTypeVideo = "video"
)

// CannedResponseAttachment stores one media file linked to a canned response.
type CannedResponseAttachment struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // image or video
	MimeType  string `json:"mime_type"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CannedResponseAttachments is a typed JSONB list for canned response media attachments.
type CannedResponseAttachments []CannedResponseAttachment

func (c CannedResponseAttachments) Value() (driver.Value, error) {
	if c == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(c)
}

func (c *CannedResponseAttachments) Scan(value interface{}) error {
	if value == nil {
		*c = CannedResponseAttachments{}
		return nil
	}

	switch typed := value.(type) {
	case []byte:
		if len(typed) == 0 {
			*c = CannedResponseAttachments{}
			return nil
		}
		return json.Unmarshal(typed, c)
	case string:
		if typed == "" {
			*c = CannedResponseAttachments{}
			return nil
		}
		return json.Unmarshal([]byte(typed), c)
	default:
		return errors.New("type assertion to []byte/string failed")
	}
}

// CannedResponse represents a pre-defined response text for quick insertion in chat
type CannedResponse struct {
	BaseModel
	OrganizationID uuid.UUID                 `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string                    `gorm:"size:100;not null" json:"name"`
	Shortcut       string                    `gorm:"size:50;index" json:"shortcut"`
	Content        string                    `gorm:"type:text;not null" json:"content"`
	Attachments    CannedResponseAttachments `gorm:"type:jsonb;default:'[]'" json:"attachments"`
	Category       string                    `gorm:"size:50" json:"category"`
	IsActive       bool                      `gorm:"default:true" json:"is_active"`
	UsageCount     int                       `gorm:"default:0" json:"usage_count"`
	CreatedByID    uuid.UUID                 `gorm:"type:uuid" json:"created_by_id"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	CreatedBy    *User         `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
}

func (CannedResponse) TableName() string {
	return "canned_responses"
}
