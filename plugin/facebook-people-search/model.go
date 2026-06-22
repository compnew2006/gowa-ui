package facebookpeoplesearch

import (
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

type PeopleSearch struct {
	models.BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	CampaignID     string    `gorm:"size:255;not null;index" json:"campaign_id"`
	PageID         string    `gorm:"size:255" json:"page_id"`
	Name           string    `gorm:"size:255" json:"name"`
	FollowersCount string    `gorm:"type:text" json:"followers_count"`

	Organization *models.Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (PeopleSearch) TableName() string {
	return "fb_people_searches"
}
