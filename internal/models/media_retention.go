package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaRetentionProgress is a single-row (id = 1) cursor for the
// media-retention sweeper. It records which account was served last and when
// the last pass completed, so that:
//
//   - consecutive capped passes resume rotation across process restarts
//     instead of always restarting at the first account (fairness), and
//   - the startup catch-up pass can skip itself when a recent pass already
//     ran (crash-loop protection: without this, every restart would trigger
//     a full cleanup pass minutes after boot).
//
// Written best-effort at the end of each lock-holding pass; a missing or
// stale row simply degrades to start-at-first-account, never to data loss.
type MediaRetentionProgress struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	LastAccountID *uuid.UUID `gorm:"type:uuid" json:"last_account_id,omitempty"`
	LastPassAt    *time.Time `json:"last_pass_at,omitempty"`
}

func (MediaRetentionProgress) TableName() string {
	return "media_retention_progress"
}
