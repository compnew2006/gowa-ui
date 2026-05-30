package models

import (
	"time"

	"github.com/google/uuid"
)

type MemberExtractionStatus string

const (
	MemberExtractionStatusDraft      MemberExtractionStatus = "draft"
	MemberExtractionStatusProcessing MemberExtractionStatus = "processing"
	MemberExtractionStatusPaused     MemberExtractionStatus = "paused"
	MemberExtractionStatusCompleted  MemberExtractionStatus = "completed"
	MemberExtractionStatusFailed     MemberExtractionStatus = "failed"
	MemberExtractionStatusCancelled  MemberExtractionStatus = "cancelled"
)

type MemberExtractionResultStatus string

const (
	MemberExtractionResultPending   MemberExtractionResultStatus = "pending"
	MemberExtractionResultExtracted MemberExtractionResultStatus = "extracted"
	MemberExtractionResultFailed    MemberExtractionResultStatus = "failed"
)

type MemberExtractionCampaign struct {
	BaseModel
	OrganizationID uuid.UUID              `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string                 `gorm:"size:255;not null" json:"name"`
	InstanceID     uuid.UUID              `gorm:"type:uuid;index;not null" json:"instance_id"`
	InstanceName   string                 `gorm:"size:255" json:"instance_name"`
	GroupJID       string                 `gorm:"size:100;not null" json:"group_jid"`
	GroupName      string                 `gorm:"size:255" json:"group_name"`
	Status         MemberExtractionStatus `gorm:"size:20;default:'draft'" json:"status"`
	TotalMembers   int                    `gorm:"default:0" json:"total_members"`
	ExtractedCount int                    `gorm:"default:0" json:"extracted_count"`
	FailedCount    int                    `gorm:"default:0" json:"failed_count"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedBy      uuid.UUID              `gorm:"type:uuid;not null" json:"created_by"`

	Organization *Organization              `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Creator      *User                      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Results      []MemberExtractionResult   `gorm:"foreignKey:CampaignID" json:"results,omitempty"`
}

func (MemberExtractionCampaign) TableName() string { return "member_extraction_campaigns" }

type MemberExtractionResult struct {
	BaseModel
	CampaignID     uuid.UUID                    `gorm:"type:uuid;index;not null" json:"campaign_id"`
	ParticipantJID string                       `gorm:"size:100;index" json:"participant_jid"`
	PhoneNumber    string                       `gorm:"size:50" json:"phone_number"`
	PushName       string                       `gorm:"size:255" json:"push_name"`
	IsAdmin        bool                         `gorm:"default:false" json:"is_admin"`
	IsSuperAdmin   bool                         `gorm:"default:false" json:"is_super_admin"`
	Status         MemberExtractionResultStatus `gorm:"size:20;default:'pending'" json:"status"`
	ErrorMessage   string                       `gorm:"type:text" json:"error_message"`

	Campaign *MemberExtractionCampaign `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
}

func (MemberExtractionResult) TableName() string { return "member_extraction_results" }
