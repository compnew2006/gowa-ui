package facebookoauth

import (
	"time"

	"github.com/google/uuid"

	"github.com/compnew2006/whatomate/internal/models"
)

type OAuthState struct {
	models.BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID      uuid.UUID `gorm:"type:uuid;index" json:"account_id"`
	StateToken     string    `gorm:"size:128;uniqueIndex;not null" json:"state_token"`
	Action         string    `gorm:"size:50;default:'connect'" json:"action"`
	ExpiresAt      time.Time `gorm:"index;not null" json:"expires_at"`
}

func (OAuthState) TableName() string {
	return "facebook_oauth_states"
}
