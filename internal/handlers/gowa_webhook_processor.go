package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GowaWebhookProcessor drains the durable GOWA webhook inbox
// (gowa_webhook_events). It claims pending rows atomically (pending→processing
// UPDATE ... RETURNING with FOR UPDATE SKIP LOCKED, so concurrent replicas never
// double-process), dispatches each to the SAME process* handler the old inline
// path used, and records the outcome with bounded retries + dead-lettering.
//
// This is the consumption side of the gap-#1 fix: because the webhook handler
// now persists every event BEFORE returning 2xx, a crash / DB outage / panic
// after the ACK no longer silently drops the event — the pending row survives
// and is processed here on restart or the next tick.
//
// Lifecycle mirrors ScheduledMessageProcessor (Start/Stop with context +
// stopCh). The extra wakeCh lets the handler trigger an immediate drain right
// after an enqueue, so inbound message latency stays near-real-time instead of
// waiting for the poll interval.
type GowaWebhookProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
	wakeCh   chan struct{}
}

const (
	// gowaWebhookBatchSize bounds how many events one claim grabs. Small enough
	// that a slow batch (media downloads) doesn't block the worker long, large
	// enough to amortize the claim query under bursts.
	gowaWebhookBatchSize = 20

	// gowaWebhookMaxAttempts is the per-event retry ceiling before dead-letter.
	// Generous enough to ride out a transient GOWA / DB blip, bounded so a
	// permanently-bad event stops consuming a claim slot.
	gowaWebhookMaxAttempts = 5

	// gowaWebhookStaleAge: a row sitting in "processing" longer than this is
	// assumed to belong to a crashed worker and returned to pending. Matches
	// the scheduled-message processor's tolerance.
	gowaWebhookStaleAge = 10 * time.Minute

	// gowaWebhookErrMaxLen bounds the stored last_error so a huge error string
	// can't bloat the row.
	gowaWebhookErrMaxLen = 1000
)

// NewGowaWebhookProcessor creates the inbox drain processor. interval is the
// safety-net poll frequency (the wakeCh handles the common low-latency path).
func NewGowaWebhookProcessor(app *App, interval time.Duration) *GowaWebhookProcessor {
	return &GowaWebhookProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
		// Buffered size 1 + the Notify() non-blocking send coalesce bursts into
		// a single pending wakeup — many enqueues in quick succession drain in
		// one batch instead of queuing one wake per event.
		wakeCh: make(chan struct{}, 1),
	}
}

// Start begins the drain loop. Blocks until ctx is cancelled or Stop is called.
func (p *GowaWebhookProcessor) Start(ctx context.Context) {
	p.app.Log.Info("GOWA webhook inbox processor started", "interval", p.interval)

	// Catch up shortly after startup: recover rows stranded in "processing" by
	// a previous crash, then drain anything enqueued while the server was down.
	time.AfterFunc(5*time.Second, func() {
		p.RecoverStale(time.Now())
		p.ProcessBatch()
	})

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("GOWA webhook inbox processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("GOWA webhook inbox processor stopped")
			return
		case <-ticker.C:
			p.ProcessBatch()
		case <-p.wakeCh:
			// Immediate drain requested by the handler after an enqueue. Drain
			// fully: keep looping until no pending rows remain so a burst
			// triggered by one wake is fully cleared.
			p.drainAll()
		}
	}
}

// Stop signals the processor to exit. Idempotent.
func (p *GowaWebhookProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// Notify requests an immediate drain. Non-blocking and coalescing: if a drain
// is already pending the call is a no-op, so the handler can call it on every
// enqueue without queueing a wake per event.
func (p *GowaWebhookProcessor) Notify() {
	select {
	case p.wakeCh <- struct{}{}:
	default:
	}
}

// drainAll repeatedly processes batches until a claim returns nothing, so a
// single wake fully clears a burst instead of processing one batch per tick.
func (p *GowaWebhookProcessor) drainAll() {
	for {
		if p.ProcessBatch() == 0 {
			return
		}
	}
}

// RecoverStale returns rows stuck in "processing" longer than gowaWebhookStaleAge
// to pending so they are retried. Expected only after a crash between claim and
// the final status update.
func (p *GowaWebhookProcessor) RecoverStale(now time.Time) {
	res := p.app.DB.Model(&models.GowaWebhookEvent{}).
		Where("status = ? AND updated_at < ?", models.GowaWebhookEventProcessing, now.Add(-gowaWebhookStaleAge)).
		Update("status", models.GowaWebhookEventPending)
	if res.Error != nil {
		p.app.Log.Error("Failed to recover stale GOWA webhook events", "error", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		p.app.Log.Warn("Recovered stale GOWA webhook events", "count", res.RowsAffected)
	}
}

// ProcessBatch atomically claims up to gowaWebhookBatchSize pending rows and
// dispatches each. Returns the number of rows claimed (0 = nothing pending).
// The claim is the idempotency barrier: FOR UPDATE SKIP LOCKED lets multiple
// workers/replicas claim disjoint rows without blocking.
func (p *GowaWebhookProcessor) ProcessBatch() int {
	var batch []models.GowaWebhookEvent
	res := p.app.DB.Model(&batch).Clauses(clause.Returning{}).
		Where(`id IN (SELECT id FROM gowa_webhook_events
			WHERE status = ? AND deleted_at IS NULL
			ORDER BY created_at LIMIT ? FOR UPDATE SKIP LOCKED)`,
			models.GowaWebhookEventPending, gowaWebhookBatchSize).
		Updates(map[string]any{
			"status":   models.GowaWebhookEventProcessing,
			"attempts": gorm.Expr("attempts + 1"),
		})
	if res.Error != nil {
		p.app.Log.Error("Failed to claim GOWA webhook events", "error", res.Error)
		return 0
	}
	if len(batch) == 0 {
		return 0
	}
	for i := range batch {
		p.processOne(&batch[i])
	}
	return len(batch)
}

// processOne re-parses the stored raw envelope, loads the owning account, and
// dispatches to the appropriate process* handler. A panic in any handler is
// recovered so a single bad event can never crash the worker loop; the event
// is then marked processed (a panic is deterministic — retrying would panic
// again), with the panic captured in last_error for diagnosis.
func (p *GowaWebhookProcessor) processOne(evt *models.GowaWebhookEvent) {
	defer func() {
		if rv := recover(); rv != nil {
			p.finish(evt, fmt.Errorf("panic: %v", rv))
		}
	}()

	var envelope gowa.WebhookPayload
	if err := json.Unmarshal(evt.RawBody, &envelope); err != nil {
		p.finish(evt, fmt.Errorf("unmarshal stored envelope: %w", err))
		return
	}

	// Load + decrypt the owning account resolved at enqueue time. Stored on the
	// row so the worker doesn't re-resolve the device (which may since have been
	// reconfigured) — the event belongs to the account that owned it when it
	// arrived and was HMAC-verified.
	var account models.WhatsAppAccount
	if err := p.app.DB.First(&account, "id = ?", evt.AccountID).Error; err != nil {
		p.finish(evt, fmt.Errorf("load account %s: %w", evt.AccountID, err))
		return
	}
	p.app.decryptAccountSecrets(&account)

	p.dispatch(&account, &envelope)
	p.finish(evt, nil)
}

// dispatch routes a parsed envelope to its handler. Each process* recovers its
// own panics and logs its own errors internally (best-effort, as before); the
// only failures processOne surfaces are unmarshal/account-load errors, which
// warrant a retry. Unhandled event types are a successful no-op (the event was
// durably captured for later support — gap #9 acknowledges them here).
func (p *GowaWebhookProcessor) dispatch(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	switch envelope.Event {
	case "message":
		p.app.processGowaMessage(account, envelope)
	case "message.ack":
		p.app.processGowaAck(account, envelope)
	case "chat_presence":
		p.app.processGowaChatPresence(account, envelope)
	case "connection":
		p.app.processGowaConnection(account, envelope)
	case "message.reaction":
		p.app.processGowaReaction(account, envelope)
	case "message.revoked":
		p.app.processGowaRevoked(account, envelope)
	case "message.edited":
		p.app.processGowaEdited(account, envelope)
	case "call.offer":
		p.app.processGowaCallOffer(account, envelope)
	default:
		// Known-but-not-yet-implemented GOWA events — message.deleted,
		// group.participants, group.joined, label.edit, label.association,
		// newsletter.{joined,left,message,mute} — are persisted to the inbox
		// (so nothing is lost) and acknowledged here. The row is marked
		// processed. Full business logic for each is a separate capability: add
		// a case above to implement one (gap #9 coverage). Info-level so
		// operators can see what is flowing through unimplemented.
		p.app.Log.Info("GOWA event acknowledged but not yet implemented",
			"event", envelope.Event, "device_id", envelope.DeviceID)
	}
}

// finish records the terminal state of a claimed row: processed on success,
// retried (pending) on failure until MaxAttempts, then dead-lettered.
// evt.Attempts already reflects the post-claim increment (RETURNING *).
func (p *GowaWebhookProcessor) finish(evt *models.GowaWebhookEvent, processErr error) {
	if processErr == nil {
		now := time.Now()
		if err := p.app.DB.Model(&models.GowaWebhookEvent{}).Where("id = ?", evt.ID).Updates(map[string]any{
			"status":       models.GowaWebhookEventProcessed,
			"processed_at": &now,
			"last_error":   "",
		}).Error; err != nil {
			p.app.Log.Error("Failed to mark GOWA webhook event processed",
				"error", err, "event_id", evt.ID)
		}
		return
	}

	errStr := truncateErr(processErr.Error())
	if evt.Attempts >= gowaWebhookMaxAttempts {
		if err := p.app.DB.Model(&models.GowaWebhookEvent{}).Where("id = ?", evt.ID).Updates(map[string]any{
			"status":     models.GowaWebhookEventDead,
			"last_error": errStr,
		}).Error; err != nil {
			p.app.Log.Error("Failed to dead-letter GOWA webhook event",
				"error", err, "event_id", evt.ID)
		}
		p.app.Log.Error("GOWA webhook event dead-lettered (exhausted retries)",
			"event_id", evt.ID, "event", evt.Event, "attempts", evt.Attempts, "error", processErr)
		return
	}

	if err := p.app.DB.Model(&models.GowaWebhookEvent{}).Where("id = ?", evt.ID).Updates(map[string]any{
		"status":     models.GowaWebhookEventPending, // re-claimable on the next batch
		"last_error": errStr,
	}).Error; err != nil {
		p.app.Log.Error("Failed to requeue GOWA webhook event",
			"error", err, "event_id", evt.ID)
	}
	p.app.Log.Warn("GOWA webhook event processing failed; requeued for retry",
		"event_id", evt.ID, "event", evt.Event, "attempts", evt.Attempts, "error", processErr)
}

// truncateErr bounds an error string to gowaWebhookErrMaxLen for storage.
func truncateErr(s string) string {
	if len(s) <= gowaWebhookErrMaxLen {
		return s
	}
	return s[:gowaWebhookErrMaxLen-3] + "..."
}
