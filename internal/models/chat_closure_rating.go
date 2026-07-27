package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatClosureRating cycle states
const (
	RatingStatusPending = "pending"
	RatingStatusRated   = "rated"
	RatingStatusExpired = "expired"
)

// Rating sources — an explicit rating (digits/stars/lexicon) is not the same
// weight as one inferred later from free text; analytics must be able to
// separate them.
const (
	RatingSourceExplicit = "explicit" // parsed deterministically from the reply
	RatingSourceInferred = "inferred" // reserved: async LLM classification of the free-text tail
)

// Rating prompt kinds. Poll stays dormant until the WhatsApp provider can
// report poll votes (stock GOWA cannot) — kept now so the upgrade is a config
// flip, not a migration.
const (
	RatingPromptText = "text"
	RatingPromptPoll = "poll"
)

// ChatClosureRating tracks one CSAT cycle: a prompt sent to the customer
// after their conversation was closed, and the reply captured within the
// window. At most one pending cycle may exist per contact — enforced by the
// partial unique index idx_chat_closure_ratings_pending (see database.getIndexes).
type ChatClosureRating struct {
	BaseModel
	OrganizationID  uuid.UUID  `gorm:"type:uuid;index;not null" json:"organization_id"`
	ContactID       uuid.UUID  `gorm:"type:uuid;index;not null" json:"contact_id"`
	WhatsAppAccount string     `gorm:"column:whats_app_account;size:100;index" json:"whatsapp_account"` // References WhatsAppAccount.Name
	ClosedByUserID  *uuid.UUID `gorm:"type:uuid" json:"closed_by_user_id,omitempty"`
	Status          string     `gorm:"size:20;default:'pending';index" json:"status"` // pending | rated | expired
	PromptKind      string     `gorm:"size:10;default:'text'" json:"prompt_kind"`     // text | poll
	Rating          *int       `json:"rating,omitempty"`                              // 1..5, validated in code
	RatingSource    string     `gorm:"size:10" json:"rating_source,omitempty"`        // explicit | inferred
	RawReply        string     `gorm:"type:text" json:"raw_reply,omitempty"`          // verbatim customer reply (also kept for unmatched replies)
	RatedAt         *time.Time `json:"rated_at,omitempty"`
	ExpiresAt       time.Time  `gorm:"index" json:"expires_at"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Contact      *Contact      `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
}

func (ChatClosureRating) TableName() string {
	return "chat_closure_ratings"
}
