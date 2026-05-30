package models

import (
	"github.com/google/uuid"
)

type FacebookAccountStatus string

const (
	FBAccountStatusActive   FacebookAccountStatus = "active"
	FBAccountStatusInactive FacebookAccountStatus = "inactive"
	FBAccountStatusClosed   FacebookAccountStatus = "closed"
)

type FacebookAccountMethod string

const (
	FBAccountMethodCookies     FacebookAccountMethod = "cookies"
	FBAccountMethodCredentials FacebookAccountMethod = "credentials"
)

type FacebookAccount struct {
	BaseModel
	OrganizationID uuid.UUID             `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string                `gorm:"not null" json:"name"`
	AccountUID     string                `gorm:"size:255" json:"account_uid"`
	Status         FacebookAccountStatus `gorm:"type:varchar(20);default:'inactive'" json:"status"`
	Method         FacebookAccountMethod `gorm:"type:varchar(20);default:'cookies'" json:"method"`
	CookiesText    string                `gorm:"type:text" json:"-"` // encrypted at rest
	Data           JSONB                 `gorm:"type:jsonb;default:'{}'" json:"data"`

	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (FacebookAccount) TableName() string {
	return "facebook_accounts"
}
