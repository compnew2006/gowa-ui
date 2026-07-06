package models

import "github.com/google/uuid"

// GroupDirectory represents a searchable WhatsApp group listing.
type GroupDirectory struct {
	BaseModel
	OrganizationID   uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	GroupJID         string    `gorm:"size:100;not null;index" json:"group_jid"`
	Name             string    `gorm:"size:255;not null;index" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	Country          string    `gorm:"size:100;index" json:"country"`
	Language         string    `gorm:"size:100" json:"language"`
	Category         string    `gorm:"size:255;index" json:"category"`
	ImageURL         string    `gorm:"type:text" json:"image_url"`
	JoinLink         string    `gorm:"type:text" json:"join_link"`
	ParticipantCount int       `json:"participant_count"`
}

func (GroupDirectory) TableName() string {
	return "group_directories"
}
