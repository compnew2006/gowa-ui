package models

import (
	"time"

	"github.com/google/uuid"
)

type MessageExtractionStatus string

const (
	MsgExtractionStatusDraft      MessageExtractionStatus = "draft"
	MsgExtractionStatusProcessing MessageExtractionStatus = "processing"
	MsgExtractionStatusPaused     MessageExtractionStatus = "paused"
	MsgExtractionStatusCompleted  MessageExtractionStatus = "completed"
	MsgExtractionStatusFailed     MessageExtractionStatus = "failed"
	MsgExtractionStatusCancelled  MessageExtractionStatus = "cancelled"
)

type MessageExtractionResultStatus string

const (
	MsgExtractionResultPending   MessageExtractionResultStatus = "pending"
	MsgExtractionResultExtracted MessageExtractionResultStatus = "extracted"
	MsgExtractionResultFailed    MessageExtractionResultStatus = "failed"
)

type MessageExtractionCampaign struct {
	BaseModel
	OrganizationID uuid.UUID               `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string                  `gorm:"size:255;not null" json:"name"`
	InstanceID     uuid.UUID               `gorm:"type:uuid;index;not null" json:"instance_id"`
	InstanceName   string                  `gorm:"size:255" json:"instance_name"`
	Status         MessageExtractionStatus `gorm:"size:20;default:'draft'" json:"status"`
	TotalChats     int                     `gorm:"default:0" json:"total_chats"`
	ExtractedCount int                     `gorm:"default:0" json:"extracted_count"`
	FailedCount    int                     `gorm:"default:0" json:"failed_count"`
	StartedAt      *time.Time              `json:"started_at,omitempty"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`
	CreatedBy      uuid.UUID               `gorm:"type:uuid;not null" json:"created_by"`

	Organization *Organization               `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Creator      *User                       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Results      []MessageExtractionResult   `gorm:"foreignKey:CampaignID" json:"results,omitempty"`
}

func (MessageExtractionCampaign) TableName() string { return "message_extraction_campaigns" }

type MessageExtractionResult struct {
	BaseModel
	CampaignID   uuid.UUID                     `gorm:"type:uuid;index;not null" json:"campaign_id"`
	ChatJID      string                        `gorm:"size:100;index" json:"chat_jid"`
	PhoneNumber  string                        `gorm:"size:50" json:"phone_number"`
	ProfileName  string                        `gorm:"size:255" json:"profile_name"`
	PushName     string                        `gorm:"size:255" json:"push_name"`
	IsGroup      bool                          `gorm:"default:false" json:"is_group"`
	GroupName    string                        `gorm:"size:255" json:"group_name"`
	GroupJID     string                        `gorm:"size:100" json:"group_jid"`
	UnreadCount  int                           `gorm:"default:0" json:"unread_count"`
	IsMe         bool                          `gorm:"default:false" json:"is_me"`
	LastMessageAt *time.Time                   `json:"last_message_at,omitempty"`
	Status       MessageExtractionResultStatus `gorm:"size:20;default:'pending'" json:"status"`
	ErrorMessage string                        `gorm:"type:text" json:"error_message"`

	Campaign *MessageExtractionCampaign `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
}

func (MessageExtractionResult) TableName() string { return "message_extraction_results" }
