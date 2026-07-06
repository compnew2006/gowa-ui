package models

import (
	"time"

	"github.com/google/uuid"
)

type CollaboratorRole string

const (
	CollaboratorRoleViewer    CollaboratorRole = "viewer"
	CollaboratorRoleAssistant CollaboratorRole = "assistant"
)

type CollaboratorStatus string

const (
	CollaboratorStatusInvited  CollaboratorStatus = "invited"
	CollaboratorStatusAccepted CollaboratorStatus = "accepted"
	CollaboratorStatusDeclined CollaboratorStatus = "declined"
)

// ContactCollaborator represents a user invited to collaborate on a contact chat.
type ContactCollaborator struct {
	BaseModel
	OrganizationID   uuid.UUID          `gorm:"type:uuid;index;not null" json:"organization_id"`
	ContactID        uuid.UUID          `gorm:"type:uuid;index;not null" json:"contact_id"`
	UserID           uuid.UUID          `gorm:"type:uuid;index;not null" json:"user_id"`
	Role             CollaboratorRole   `gorm:"size:20;not null;default:'assistant'" json:"role"`
	Status           CollaboratorStatus `gorm:"size:20;not null;default:'invited'" json:"status"`
	InvitedByUserID  uuid.UUID          `gorm:"type:uuid;index;not null" json:"invited_by_user_id"`
	AcceptedAt       *time.Time         `json:"accepted_at,omitempty"`
	DeclinedAt       *time.Time         `json:"declined_at,omitempty"`
}

func (ContactCollaborator) TableName() string {
	return "contact_collaborators"
}
