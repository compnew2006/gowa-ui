package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"gorm.io/gorm"
)

// GowaMediaBackfillProcessor proactively downloads pending message media
// (media_url = "" but a provider message id exists) so files are on local
// disk before the provider expires them.
//
// Why this exists: webhook-time downloads cap at the in-memory 50MiB guard
// and can also fail transiently, leaving the row with media_url="". Until
// now the only recovery path was a user's first VIEW of the bubble (lazy
// recovery in ServeMedia) — but WhatsApp expires media on the provider side,
// so a file nobody opened in time is lost forever. This worker closes that
// gap: every interval it claims the oldest pending rows and streams them to
// disk through the same recoverMediaToDisk path the lazy recovery uses.
type GowaMediaBackfillProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

const (
	// gowaBackfillBatchSize bounds one pass. Old-first ordering means the
	// rows closest to provider expiry are always claimed first.
	gowaBackfillBatchSize = 50

	// gowaBackfillMaxAttempts caps retries for transient failures before the
	// row is marked permanently failed (stops the scan from re-reading it
	// forever and the logs from filling with the same error).
	gowaBackfillMaxAttempts = 10

	// gowaBackfillInterItemDelay spaces out downloads so a large backlog
	// doesn't hammer GOWA with back-to-back requests.
	gowaBackfillInterItemDelay = 500 * time.Millisecond

	// Metadata keys track per-message backfill state. JSONB text values.
	gowaBackfillAttemptsKey  = "mbf_attempts"
	gowaBackfillPermanentKey = "mbf_permanent"
)

// NewGowaMediaBackfillProcessor creates the pending-media backfill processor.
func NewGowaMediaBackfillProcessor(app *App, interval time.Duration) *GowaMediaBackfillProcessor {
	return &GowaMediaBackfillProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the backfill loop with an immediate initial pass — a fresh
// deploy should drain the existing backlog right away, not one interval in.
func (p *GowaMediaBackfillProcessor) Start(ctx context.Context) {
	p.app.Log.Info("GOWA media backfill processor started", "interval", p.interval)

	p.runPass(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("GOWA media backfill processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("GOWA media backfill processor stopped")
			return
		case <-ticker.C:
			p.runPass(ctx)
		}
	}
}

// Stop stops the processor. An in-flight download finishes first (bounded by
// recoverMediaToDisk's transfer budget).
func (p *GowaMediaBackfillProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// gowaBackfillPassStats summarizes one backfill pass for logging.
type gowaBackfillPassStats struct {
	scanned   int
	recovered int
	transient int
	permanent int
	skipped   int
}

func (s gowaBackfillPassStats) logFields() []any {
	return []any{
		"scanned", s.scanned,
		"recovered", s.recovered,
		"transient_failed", s.transient,
		"permanent_failed", s.permanent,
		"skipped", s.skipped,
	}
}

// runPass claims one batch of pending media messages and recovers them.
func (p *GowaMediaBackfillProcessor) runPass(ctx context.Context) {
	var msgs []models.Message
	if err := p.app.pendingBackfillQuery().
		Order("created_at ASC").
		Limit(gowaBackfillBatchSize).
		Find(&msgs).Error; err != nil {
		p.app.Log.Error("GOWA media backfill scan failed", "error", err)
		return
	}
	if len(msgs) == 0 {
		return
	}

	stats := gowaBackfillPassStats{scanned: len(msgs)}
	for i := range msgs {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		default:
		}

		switch p.app.backfillMessageMedia(&msgs[i]) {
		case gowaBackfillRecovered:
			stats.recovered++
		case gowaBackfillTransient:
			stats.transient++
		case gowaBackfillPermanent:
			stats.permanent++
		default:
			stats.skipped++
		}

		if i < len(msgs)-1 {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-time.After(gowaBackfillInterItemDelay):
			}
		}
	}
	p.app.Log.Info("GOWA media backfill pass", stats.logFields()...)
}

type gowaBackfillResult int

const (
	gowaBackfillSkipped gowaBackfillResult = iota
	gowaBackfillRecovered
	gowaBackfillTransient
	gowaBackfillPermanent
)

// pendingBackfillQuery is the single source of truth for backfill
// eligibility: media-bearing rows that never got bytes on disk (media_url
// empty but a provider message id exists) and are still worth retrying —
// i.e. not yet exhausted (mbf_attempts) and not permanently gone
// (mbf_permanent). Callers add ordering/limit.
func (a *App) pendingBackfillQuery() *gorm.DB {
	return a.DB.Model(&models.Message{}).
		Where("media_url = '' AND whats_app_message_id <> ''").
		Where("message_type IN ?", []string{
			string(models.MessageTypeImage), string(models.MessageTypeVideo),
			string(models.MessageTypeAudio), string(models.MessageTypeDocument), "sticker",
		}).
		Where("COALESCE((metadata ->> 'mbf_permanent')::boolean, false) = false").
		Where("COALESCE((metadata ->> 'mbf_attempts')::int, 0) < ?", gowaBackfillMaxAttempts)
}

// backfillMessageMedia recovers one pending message's media. Skipped (no
// account / no provider) rows stay eligible and are retried on a later pass;
// permanent failures are flagged in metadata so the scan stops selecting them.
func (a *App) backfillMessageMedia(msg *models.Message) gowaBackfillResult {
	var contact models.Contact
	hasContact := a.DB.Where("id = ? AND organization_id = ?", msg.ContactID, msg.OrganizationID).
		First(&contact).Error == nil
	if !hasContact {
		// Contact gone (deleted) — nothing to build a chat JID from. This
		// can't succeed later; mark permanent so the scan stops re-reading it.
		a.markBackfillResult(msg, true, "contact deleted")
		return gowaBackfillPermanent
	}
	if contactIsStatusLike(&contact) {
		// Status feed / broadcast / newsletter media is rejected by GOWA's
		// download endpoint — retrying can never succeed.
		a.markBackfillResult(msg, true, "status/newsletter media not downloadable")
		return gowaBackfillPermanent
	}

	account := a.resolveGowaAccountForRecovery(msg.OrganizationID, msg.WhatsAppAccount, contact.WhatsAppAccount)
	if account == nil {
		// No usable GOWA account right now (org has none / renamed). Cheap to
		// retry next pass; may become recoverable when an account is added.
		return gowaBackfillSkipped
	}
	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return gowaBackfillSkipped
	}

	return a.backfillWithClient(gowaClient, account.ToWAAccount(), account.Name, msg, &contact)
}

// backfillWithClient streams the message's media to disk through the given
// GOWA client and claims the row. Split from backfillMessageMedia so unit
// tests can drive the real recovery+claim path against a fake GOWA server
// without wiring a WARegistry. The claim is guarded to still-pending rows —
// ServeMedia's lazy recovery may have recovered the same message
// concurrently; losing that race leaves our freshly streamed file orphaned,
// so it is removed again.
func (a *App) backfillWithClient(client *gowa.Client, waAccount *whatsapp.Account, accountName string, msg *models.Message, contact *models.Contact) gowaBackfillResult {
	relativePath, sniffedType, err := a.recoverMediaToDisk(
		client, waAccount, *msg, gowaChatJID(contact))
	if err != nil {
		permanent := isMediaGoneError(err)
		a.markBackfillResult(msg, permanent, err.Error())
		if permanent {
			return gowaBackfillPermanent
		}
		return gowaBackfillTransient
	}

	res := a.DB.Model(&models.Message{}).
		Where("id = ? AND media_url = ''", msg.ID).
		Updates(map[string]any{
			"media_url":         relativePath,
			"media_mime_type":   sniffedType,
			"whats_app_account": accountName,
		})
	if res.Error != nil {
		a.Log.Error("GOWA media backfill: failed to update message", "message_id", msg.ID, "error", res.Error)
		a.removeLocalMedia(relativePath)
		return gowaBackfillTransient
	}
	if res.RowsAffected == 0 {
		// Another path (lazy recovery) claimed the row first — media is on
		// disk either way; drop our duplicate.
		a.removeLocalMedia(relativePath)
		return gowaBackfillRecovered
	}
	return gowaBackfillRecovered
}

// markBackfillResult records the attempt in the message's metadata JSONB:
// mbf_attempts counts tries, mbf_permanent stops future scans. Attempts also
// hard-stop at gowaBackfillMaxAttempts (checked in the scan query).
func (a *App) markBackfillResult(msg *models.Message, permanent bool, detail string) {
	meta := models.JSONB{}
	for k, v := range msg.Metadata {
		meta[k] = v
	}
	attempts := 1
	if n, err := strconv.Atoi(asString(meta[gowaBackfillAttemptsKey])); err == nil {
		attempts = n + 1
	}
	meta[gowaBackfillAttemptsKey] = attempts
	if permanent || attempts >= gowaBackfillMaxAttempts {
		meta[gowaBackfillPermanentKey] = true
	}
	if err := a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).
		Update("metadata", meta).Error; err != nil {
		a.Log.Error("GOWA media backfill: failed to mark message", "message_id", msg.ID, "error", err)
		return
	}
	if permanent {
		a.Log.Warn("GOWA media backfill: media permanently unavailable",
			"message_id", msg.ID, "wamid", msg.WhatsAppMessageID, "detail", truncateString(detail, 200))
	}
}

// asString coerces a JSONB scalar to string for attempt counting.
func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// contactIsStatusLike reports whether a contact's media can never be fetched
// via GOWA's chat download endpoint (status feed, broadcast, newsletters).
// Mirrors the frontend's shouldRenderMedia exclusions and the is_newsletter
// metadata convention in contacts.go.
func contactIsStatusLike(contact *models.Contact) bool {
	phone := strings.ToLower(strings.TrimSpace(contact.PhoneNumber))
	return phone == "status" ||
		phone == "broadcast" ||
		strings.HasSuffix(phone, "@newsletter") ||
		(contact.Metadata != nil && contact.Metadata["is_newsletter"] == true)
}

// isMediaGoneError reports whether a provider error means the media is
// permanently unavailable (expired/purged/not held). Shared by the
// redownload handler's classification and the backfill worker.
func isMediaGoneError(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "does not belong") ||
		strings.Contains(low, "not found") ||
		strings.Contains(low, "no longer available") ||
		strings.Contains(low, "expired") ||
		strings.Contains(low, "deleted") ||
		strings.Contains(low, "status 404")
}
