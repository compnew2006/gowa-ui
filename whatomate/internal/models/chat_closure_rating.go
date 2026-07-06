package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatClosureRatingState tracks the lifecycle of a close-cycle rating request.
type ChatClosureRatingState string

const (
	ChatClosureRatingStatePending ChatClosureRatingState = "pending"
	ChatClosureRatingStateRated   ChatClosureRatingState = "rated"
	ChatClosureRatingStateExpired ChatClosureRatingState = "expired"
)

// ChatClosureRating stores one feedback cycle created each time a chat is manually closed.
type ChatClosureRating struct {
	BaseModel
	OrganizationID uuid.UUID              `gorm:"type:uuid;index;not null" json:"organization_id"`
	ContactID      uuid.UUID              `gorm:"type:uuid;index;not null" json:"contact_id"`
	ChatID         uuid.UUID              `gorm:"type:uuid;index;not null" json:"chat_id"`
	AgentUserID    *uuid.UUID             `gorm:"type:uuid;index" json:"agent_user_id,omitempty"`
	ClosingAgentID uuid.UUID              `gorm:"type:uuid;index;not null" json:"closing_agent_id"`
	ClosedAt       time.Time              `gorm:"index;not null" json:"closed_at"`
	State          ChatClosureRatingState `gorm:"size:20;default:'pending';index;not null" json:"state"`

	Rating          *int       `gorm:"check:rating >= 1 AND rating <= 10" json:"rating,omitempty"`
	RatedAt         *time.Time `gorm:"index" json:"rated_at,omitempty"`
	RatingMessage   string     `gorm:"type:text" json:"rating_message"`
	RatingMessageID *uuid.UUID `gorm:"type:uuid;index" json:"rating_message_id,omitempty"`

	CloseMessage         string     `gorm:"type:text" json:"close_message"`
	CloseMessageLanguage string     `gorm:"size:16" json:"close_message_language"`
	CloseMessageID       *uuid.UUID `gorm:"type:uuid;index" json:"close_message_id,omitempty"`

	ContextMessages JSONB `gorm:"type:jsonb;default:'{}'" json:"context_messages"`

	Contact          *Contact `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
	AgentUser        *User    `gorm:"foreignKey:AgentUserID" json:"agent_user,omitempty"`
	ClosingAgent     *User    `gorm:"foreignKey:ClosingAgentID" json:"closing_agent,omitempty"`
	RatingMessageRef *Message `gorm:"foreignKey:RatingMessageID" json:"rating_message_ref,omitempty"`
	CloseMessageRef  *Message `gorm:"foreignKey:CloseMessageID" json:"close_message_ref,omitempty"`
}

func (ChatClosureRating) TableName() string {
	return "chat_closure_ratings"
}
