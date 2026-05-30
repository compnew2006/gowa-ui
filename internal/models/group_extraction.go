package models

import (
	"time"

	"github.com/google/uuid"
)

type GroupExtractionStatus string

const (
	GroupExtractionStatusDraft      GroupExtractionStatus = "draft"
	GroupExtractionStatusProcessing GroupExtractionStatus = "processing"
	GroupExtractionStatusPaused     GroupExtractionStatus = "paused"
	GroupExtractionStatusCompleted  GroupExtractionStatus = "completed"
	GroupExtractionStatusFailed     GroupExtractionStatus = "failed"
	GroupExtractionStatusCancelled  GroupExtractionStatus = "cancelled"
)

type GroupExtractionResultStatus string

const (
	GroupExtractionResultPending   GroupExtractionResultStatus = "pending"
	GroupExtractionResultExtracted GroupExtractionResultStatus = "extracted"
	GroupExtractionResultFailed    GroupExtractionResultStatus = "failed"
)

type GroupExtractionCampaign struct {
	BaseModel
	OrganizationID uuid.UUID            `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string               `gorm:"size:255;not null" json:"name"`
	InstanceID     uuid.UUID            `gorm:"type:uuid;index;not null" json:"instance_id"`
	InstanceName   string               `gorm:"size:255" json:"instance_name"`
	Status         GroupExtractionStatus `gorm:"size:20;default:'draft'" json:"status"`
	TotalGroups    int                  `gorm:"default:0" json:"total_groups"`
	ExtractedCount int                  `gorm:"default:0" json:"extracted_count"`
	FailedCount    int                  `gorm:"default:0" json:"failed_count"`
	StartedAt      *time.Time           `json:"started_at,omitempty"`
	CompletedAt    *time.Time           `json:"completed_at,omitempty"`
	CreatedBy      uuid.UUID            `gorm:"type:uuid;not null" json:"created_by"`

	Organization *Organization             `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Creator      *User                     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Results      []GroupExtractionResult   `gorm:"foreignKey:CampaignID" json:"results,omitempty"`
}

func (GroupExtractionCampaign) TableName() string { return "group_extraction_campaigns" }

type GroupExtractionResult struct {
	BaseModel
	CampaignID        uuid.UUID                   `gorm:"type:uuid;index;not null" json:"campaign_id"`
	GroupJID          string                      `gorm:"size:100;index" json:"group_jid"`
	GroupName         string                      `gorm:"size:255" json:"group_name"`
	ParticipantCount  int                         `gorm:"default:0" json:"participant_count"`
	IsAdmin           bool                        `gorm:"default:false" json:"is_admin"`
	Description       string                      `gorm:"type:text" json:"description"`
	Status            GroupExtractionResultStatus `gorm:"size:20;default:'pending'" json:"status"`
	ErrorMessage      string                      `gorm:"type:text" json:"error_message"`

	Campaign *GroupExtractionCampaign `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
}

func (GroupExtractionResult) TableName() string { return "group_extraction_results" }
