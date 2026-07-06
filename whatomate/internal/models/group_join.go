package models

import (
	"time"

	"github.com/google/uuid"
)

// GroupJoinSpeed controls the delay between join attempts.
type GroupJoinSpeed string

const (
	GroupJoinSpeedSlow GroupJoinSpeed = "slow" // Free, longer delays
	GroupJoinSpeedFast GroupJoinSpeed = "fast" // Costs points, shorter delays
)

// GroupJoinCampaignStatus represents group join campaign states.
type GroupJoinCampaignStatus string

const (
	GroupJoinStatusDraft      GroupJoinCampaignStatus = "draft"
	GroupJoinStatusProcessing GroupJoinCampaignStatus = "processing"
	GroupJoinStatusPaused     GroupJoinCampaignStatus = "paused"
	GroupJoinStatusCompleted  GroupJoinCampaignStatus = "completed"
	GroupJoinStatusFailed     GroupJoinCampaignStatus = "failed"
	GroupJoinStatusCancelled  GroupJoinCampaignStatus = "cancelled"
)

// GroupJoinRecipientStatus represents the processing state of a single invite link.
type GroupJoinRecipientStatus string

const (
	GroupJoinRecipientPending   GroupJoinRecipientStatus = "pending"
	GroupJoinRecipientJoined    GroupJoinRecipientStatus = "joined"
	GroupJoinRecipientFailed    GroupJoinRecipientStatus = "failed"
	GroupJoinRecipientSkipped   GroupJoinRecipientStatus = "skipped" // Already joined or rate-limited
	GroupJoinRecipientDuplicate GroupJoinRecipientStatus = "duplicate"
)

// GroupJoinCampaign represents a WhatsApp group join campaign.
type GroupJoinCampaign struct {
	BaseModel
	OrganizationID uuid.UUID               `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string                  `gorm:"size:255;not null" json:"name"`
	Accounts       JSONBArray              `gorm:"type:jsonb;not null;default:'[]'" json:"accounts"` // JSON array of WhatsApp account names
	Speed          GroupJoinSpeed           `gorm:"size:10;default:'slow'" json:"speed"`
	Status         GroupJoinCampaignStatus  `gorm:"size:20;default:'draft'" json:"status"`
	TotalRecipients int                     `gorm:"default:0" json:"total_recipients"`
	JoinedCount    int                     `gorm:"default:0" json:"joined_count"`
	FailedCount    int                     `gorm:"default:0" json:"failed_count"`
	SkippedCount   int                     `gorm:"default:0" json:"skipped_count"`
	StartedAt      *time.Time              `json:"started_at,omitempty"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`
	CreatedBy      uuid.UUID               `gorm:"type:uuid;not null" json:"created_by"`

	// Relations
	Organization *Organization           `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Creator      *User                   `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Recipients   []GroupJoinRecipient     `gorm:"foreignKey:CampaignID" json:"recipients,omitempty"`
}

func (GroupJoinCampaign) TableName() string {
	return "group_join_campaigns"
}

// GroupJoinRecipient represents a single group invite link to be processed.
type GroupJoinRecipient struct {
	BaseModel
	CampaignID      uuid.UUID               `gorm:"type:uuid;index;not null" json:"campaign_id"`
	InviteLink      string                  `gorm:"type:text;not null" json:"invite_link"`
	GroupName       string                  `gorm:"size:255" json:"group_name"`
	GroupJID        string                  `gorm:"size:100" json:"group_jid,omitempty"` // Set after successful join
	ParticipantCount int                    `json:"participant_count"`
	Status          GroupJoinRecipientStatus `gorm:"size:20;default:'pending'" json:"status"`
	ErrorMessage    string                  `gorm:"type:text" json:"error_message"`
	ProcessedAt     *time.Time             `json:"processed_at,omitempty"`

	// Relations
	Campaign *GroupJoinCampaign `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
}

func (GroupJoinRecipient) TableName() string {
	return "group_join_recipients"
}
