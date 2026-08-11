package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GowaWebhookEventStatus is the lifecycle state of a durable inbox row.
type GowaWebhookEventStatus string

const (
	// GowaWebhookEventPending: enqueued, waiting for the worker to claim it.
	GowaWebhookEventPending GowaWebhookEventStatus = "pending"
	// GowaWebhookEventProcessing: claimed by a worker (atomic UPDATE); a crash
	// leaves the row here until RecoverStale resets it to pending.
	GowaWebhookEventProcessing GowaWebhookEventStatus = "processing"
	// GowaWebhookEventProcessed: the event was dispatched to its handler
	// successfully. Kept for diagnostics / idempotency until GC.
	GowaWebhookEventProcessed GowaWebhookEventStatus = "processed"
	// GowaWebhookEventDead: exhausted MaxAttempts; last_error holds the cause.
	// Excluded from the active idempotency window so a later genuine retry can
	// get a fresh attempt (the failure may have been transient).
	GowaWebhookEventDead GowaWebhookEventStatus = "dead"
)

// GowaWebhookEvent is one row in the durable webhook inbox. The webhook
// handler verifies HMAC, resolves the account, derives an idempotency key,
// persists the RAW event body here, and returns 2xx to GOWA ONLY after the row
// is durable — closing the silent event-loss window where the old code 200'd
// before saving and processed in an untracked goroutine (gap #1).
//
// A background processor (GowaWebhookProcessor) claims pending rows, dispatches
// each to the existing process* handler, and marks the outcome with retries +
// dead-lettering. The partial unique index idx_gowa_webhook_events_idempotency
// on (device_id, event, event_key) for active rows deduplicates concurrent or
// replayed deliveries at the inbox boundary (gap #8), so every event is
// processed at most once.
type GowaWebhookEvent struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	// AccountID is the resolved owning WhatsAppAccount; the worker re-loads it
	// (decrypted) to dispatch. Stored at enqueue time so the worker does not
	// need to re-resolve the device (which could have been reconfigured).
	AccountID uuid.UUID `gorm:"type:uuid;index" json:"account_id"`
	// DeviceID is the GOWA session id from the webhook envelope / per-device
	// route. Paired with Event+EventKey it forms the idempotency identity.
	DeviceID string `gorm:"size:100;not null" json:"device_id"`
	// Event is the GOWA event name ("message", "message.ack", "call.offer", …).
	Event string `gorm:"size:50;not null" json:"event"`
	// EventKey is the per-event idempotency key (see deriveGowaEventKey):
	// wamid for messages, call_id for call.offer, target wamid for
	// revoke/edit/reaction, etc. Empty only when no natural key exists (then a
	// payload hash is used).
	EventKey string `gorm:"size:255" json:"event_key"`
	// RawBody is the FULL webhook envelope exactly as received (so the worker
	// re-parses identically, including the top-level timestamp some events
	// carry outside payload). jsonb so it is inspectable.
	RawBody json.RawMessage `gorm:"type:jsonb" json:"raw_body"`
	// Status drives the worker claim/retry/dead-letter state machine.
	Status GowaWebhookEventStatus `gorm:"size:20;default:pending;index" json:"status"`
	// Attempts increments on each claim; compared against MaxAttempts.
	Attempts int `gorm:"default:0" json:"attempts"`
	// LastError captures the most recent processing failure (dead-letter cause).
	LastError string `gorm:"type:text" json:"last_error,omitempty"`
	// ProcessedAt is set when Status flips to processed.
	ProcessedAt *time.Time `gorm:"index" json:"processed_at,omitempty"`
}

// TableName pins the table name (convention: model struct does not embed a
// TableName method, but the inbox is referenced by raw SQL in migrations, so
// pinning avoids any GORM pluralization surprise).
func (GowaWebhookEvent) TableName() string { return "gowa_webhook_events" }
