package models

import (
	"time"

	"github.com/google/uuid"
)

// ContactReadState is one user's read position in a conversation, so each
// user's unread badge is their own: opening a chat clears it for the reader
// only, not for every agent who can see the conversation.
//
// Positions are messages.read_seq values (a sequence assigned in INSERT
// order, see database.getIndexes) rather than created_at: history-synced and
// late-delivered messages carry an older created_at than messages already
// read, and must still count as unread. A message is unread for a user when
// its read_seq is above both LastReadSeq and the conversation's shared
// contacts.read_watermark_seq (automated replies and the pre-per-user read
// state). read_seq and read_watermark_seq are raw-SQL columns on purpose:
// declaring them on Message/Contact would let GORM write zero values over
// the sequence default and the watermark.
type ContactReadState struct {
	ContactID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"contact_id"`
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	LastReadSeq    int64     `gorm:"not null;default:0" json:"last_read_seq"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ContactReadState) TableName() string {
	return "contact_read_states"
}
