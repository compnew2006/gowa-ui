package models

import (
	"time"

	"github.com/google/uuid"
)

type FacebookAccountStatus string

const (
	FBAccountStatusActive   FacebookAccountStatus = "active"
	FBAccountStatusInactive FacebookAccountStatus = "inactive"
	FBAccountStatusClosed   FacebookAccountStatus = "closed"
	FBAccountStatusExpired  FacebookAccountStatus = "expired"
	FBAccountStatusRevoked  FacebookAccountStatus = "revoked"
)

type FacebookAccountMethod string

const (
	FBAccountMethodCookies     FacebookAccountMethod = "cookies"
	FBAccountMethodCredentials FacebookAccountMethod = "credentials"
	FBAccountMethodOAuth       FacebookAccountMethod = "oauth"
)

type FacebookAccount struct {
	BaseModel
	OrganizationID uuid.UUID             `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID             `gorm:"type:uuid;index" json:"user_id"`
	Platform       string                `gorm:"size:50;default:'facebook';index" json:"platform"`
	Name           string                `gorm:"not null" json:"name"`
	AccountUID     string                `gorm:"size:255" json:"account_uid"`
	Email          string                `gorm:"size:255" json:"email,omitempty"`
	AvatarURL      string                `gorm:"type:text" json:"avatar_url,omitempty"`
	Status         FacebookAccountStatus `gorm:"type:varchar(20);default:'inactive'" json:"status"`
	Method         FacebookAccountMethod `gorm:"type:varchar(20);default:'cookies'" json:"method"`
	CookiesText    string                `gorm:"type:text" json:"-"` // encrypted at rest
	AccessToken    string                `gorm:"type:text" json:"-"` // encrypted OAuth user token
	PageTokens     string                `gorm:"type:text" json:"-"` // encrypted page_id -> page token JSON
	TokenExpiresAt *time.Time            `gorm:"index" json:"token_expires_at,omitempty"`
	ConnectedAt    *time.Time            `json:"connected_at,omitempty"`
	LastRenewedAt  *time.Time            `json:"last_renewed_at,omitempty"`
	Data           JSONB                 `gorm:"type:jsonb;default:'{}'" json:"data"`

	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (FacebookAccount) TableName() string {
	return "facebook_accounts"
}

type FacebookOAuthState struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID      uuid.UUID `gorm:"type:uuid;index" json:"account_id"`
	StateToken     string    `gorm:"size:128;uniqueIndex;not null" json:"state_token"`
	Action         string    `gorm:"size:50;default:'connect'" json:"action"`
	ExpiresAt      time.Time `gorm:"index;not null" json:"expires_at"`
}

func (FacebookOAuthState) TableName() string {
	return "facebook_oauth_states"
}
