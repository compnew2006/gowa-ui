package models

import (
	"github.com/google/uuid"
)

type FBPeopleSearch struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	CampaignID     string    `gorm:"size:255;not null;index" json:"campaign_id"`
	PageID         string    `gorm:"size:255" json:"page_id"`
	Name           string    `gorm:"size:255" json:"name"`
	FollowersCount string    `gorm:"type:text" json:"followers_count"`

	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (FBPeopleSearch) TableName() string {
	return "fb_people_searches"
}
