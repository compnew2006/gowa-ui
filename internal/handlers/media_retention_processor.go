package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaRetentionProcessor deletes local media files older than the
// per-account retention window (settings.media_retention.retention_days).
//
// Per message the sequence is:
//  1. Atomic claim: clear media_url AND merge a metadata tombstone in one
//     UPDATE guarded by "media_url still the old value AND no retention
//     state yet". Losers of the guard never touch the file. The claim is
//     the resumable point — a crash after it leaves the row in 'purging'
//     with retention_original_path, which the next pass's stuck-sweep
//     finishes.
//  2. Reference check on the old path, GLOBAL (all orgs — one storage
//     root): other message rows (soft-deleted included), pending/
//     processing scheduled messages, draft/queued/processing campaign
//     header media, contact avatars. Referenced → terminal 'purged', the
//     file stays for its remaining owners.
//  3. Unreferenced → delete the file (error-returning helper). Missing
//     file counts as success. Failure → 'delete_failed' with the claim
//     timestamp as a lease; the stuck-sweep retries it after an hour.
//
// The sweep never races ServeMedia's lazy recovery or the backfill worker:
// purged rows are excluded from recovery (ServeMedia 410) and from
// pendingBackfillQuery by the same retention_state marker.

const (
	// mediaRetentionAdvisoryKey identifies the pass lock. pg_try_advisory_
	// xact_lock on a dedicated transaction held open for the pass — released
	// by the deferred rollback, so a crashed process cannot leave it stuck.
	mediaRetentionAdvisoryKey int64 = 728451

	mediaRetentionBatchSize         = 50
	mediaRetentionInterItemDelay    = 500 * time.Millisecond
	mediaRetentionMaxFilesPerPass   = 1000
	mediaRetentionMaxBytesPerPass   = 2 << 30 // 2 GiB
	mediaRetentionMaxPassDuration   = 30 * time.Minute
	mediaRetentionStuckRetryLease   = time.Hour
	mediaRetentionStuckSweepLimit   = 200
	mediaRetentionMaxDays           = 3650
	mediaRetentionStatePurging      = "purging"
	mediaRetentionStatePurged       = "purged"
	mediaRetentionStateDeleteFailed = "delete_failed"
)

// Retention metadata keys (JSONB, text values). retention_state is the single
// marker every consumer checks: ” / absent = normal message.
const (
	retentionStateKey        = "retention_state"
	retentionPurgedAtKey     = "retention_purged_at"
	retentionOriginalPathKey = "retention_original_path"
	retentionRestoredAtKey   = "retention_restored_at"
	retentionKeepUntilKey    = "retention_keep_until"
)

// MediaRetentionProcessor runs the daily media retention pass.
type MediaRetentionProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

// NewMediaRetentionProcessor creates the media-retention processor. The
// interval is the pass cadence; a jittered catch-up pass also runs 1–5
// minutes after Start so daily restarts can't starve the sweep forever.
func NewMediaRetentionProcessor(app *App, interval time.Duration) *MediaRetentionProcessor {
	return &MediaRetentionProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the retention loop: a jittered catch-up pass shortly after
// startup, then every interval. The catch-up is required because the 24h
// cadence would otherwise never fire on hosts that restart daily — each
// restart would reset the timer. The advisory lock + per-pass caps make the
// early pass safe. Blocks until the context is cancelled or Stop is called.
func (p *MediaRetentionProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Media retention processor started", "interval", p.interval)

	catchUpDelay := time.Minute + time.Duration(rand.Int63n(int64(4*time.Minute)))
	time.AfterFunc(catchUpDelay, func() {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-p.stopCh:
			return
		default:
		}
		p.runPass(ctx)
	})

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Media retention processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("Media retention processor stopped")
			return
		case <-ticker.C:
			p.runPass(ctx)
		}
	}
}

// Stop signals the processor to exit. An in-flight item finishes first.
func (p *MediaRetentionProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// mediaRetentionPassStats summarizes one pass.
type mediaRetentionPassStats struct {
	accounts     int
	scanned      int
	purged       int // reference cleared + file deleted
	purgedShared int // reference cleared, file kept for other owners
	deleteFailed int
	lostRace     int
	sweptStuck   int
	bytesFreed   int64
}

// runPass executes one full retention pass under the advisory lock: finish
// stuck claims from a previous crashed pass first, then scan every account
// with retention enabled.
func (p *MediaRetentionProcessor) runPass(ctx context.Context) {
	// Pass mutex. A dedicated transaction holds an xact-scoped advisory lock:
	// it is released by the rollback below no matter how this function ends,
	// so a second instance (or a crashed one) can never strand it. All data
	// writes below use their own autocommit statements, not this tx.
	lockTx := p.app.DB.Begin()
	if lockTx.Error != nil {
		p.app.Log.Error("Media retention: failed to open lock transaction", "error", lockTx.Error)
		return
	}
	var locked bool
	if err := lockTx.Raw("SELECT pg_try_advisory_xact_lock(?)", mediaRetentionAdvisoryKey).Scan(&locked).Error; err != nil || !locked {
		rollback(lockTx)
		if err != nil {
			p.app.Log.Error("Media retention: lock query failed", "error", err)
		} else {
			p.app.Log.Info("Media retention: another pass holds the lock, skipping")
		}
		return
	}
	defer rollback(lockTx)

	stats := mediaRetentionPassStats{}
	deadline := time.Now().Add(mediaRetentionMaxPassDuration)

	stats.sweptStuck = p.app.retentionSweepStuck(time.Now(), mediaRetentionStuckSweepLimit, &stats)

	var accounts []models.WhatsAppAccount
	if err := p.app.DB.Where(
		`settings->'media_retention'->>'enabled' = 'true'`,
	).Find(&accounts).Error; err != nil {
		p.app.Log.Error("Media retention: failed to load enabled accounts", "error", err)
		return
	}

	stats.accounts = len(accounts)

	// Round-robin: one batch per account per round. Draining the first
	// account fully before touching the second would let one high-volume
	// account consume the whole global cap every pass and starve the rest
	// forever — each account advances one batch per round instead.
	cursors := make(map[uuid.UUID]*retentionAccountCursor, len(accounts))
	for i := range accounts {
		cursors[accounts[i].ID] = &retentionAccountCursor{}
	}
	for {
		select {
		case <-ctx.Done():
			p.logPass(stats)
			return
		case <-p.stopCh:
			p.logPass(stats)
			return
		default:
		}
		if p.capsExceeded(&stats, deadline) {
			break
		}
		progress := false
		for i := range accounts {
			if p.capsExceeded(&stats, deadline) {
				break
			}
			if p.processAccountBatch(&accounts[i], cursors[accounts[i].ID], &stats, deadline) {
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	p.logPass(stats)
}

// capsExceeded reports whether the pass budget (file count, freed bytes,
// wall clock) is spent.
func (p *MediaRetentionProcessor) capsExceeded(stats *mediaRetentionPassStats, deadline time.Time) bool {
	return stats.purged >= mediaRetentionMaxFilesPerPass ||
		stats.bytesFreed >= mediaRetentionMaxBytesPerPass ||
		time.Now().After(deadline)
}

func (p *MediaRetentionProcessor) logPass(stats mediaRetentionPassStats) {
	p.app.Log.Info("Media retention pass",
		"accounts", stats.accounts,
		"scanned", stats.scanned,
		"purged", stats.purged,
		"purged_shared", stats.purgedShared,
		"delete_failed", stats.deleteFailed,
		"lost_race", stats.lostRace,
		"stuck_swept", stats.sweptStuck,
		"bytes_freed", stats.bytesFreed)
}

// retentionAccountCursor is the keyset position of one account's scan
// ((0, nil) = start). Shared by the round-robin pass and drained fully by
// processAccount.
type retentionAccountCursor struct {
	createdAt time.Time
	id        uuid.UUID
	drained   bool
}

// processAccount purges one account's expired media in keyset batches until
// the account is drained or a cap trips. Kept as the single-account drain
// primitive (and test entry point); the pass itself round-robins via
// processAccountBatch for fairness across accounts.
func (p *MediaRetentionProcessor) processAccount(account *models.WhatsAppAccount, stats *mediaRetentionPassStats, deadline time.Time) {
	stats.accounts++
	cur := &retentionAccountCursor{}
	for !cur.drained {
		if p.capsExceeded(stats, deadline) {
			return
		}
		p.processAccountBatch(account, cur, stats, deadline)
	}
}

// processAccountBatch purges a single keyset batch for one account. Returns
// true when the batch did work (the account may hold more); false when the
// account is drained, misconfigured, or the batch query failed.
func (p *MediaRetentionProcessor) processAccountBatch(account *models.WhatsAppAccount, cur *retentionAccountCursor, stats *mediaRetentionPassStats, deadline time.Time) bool {
	if cur.drained {
		return false
	}
	days := mediaRetentionDays(account)
	if days < 1 {
		p.app.Log.Warn("Media retention: skipping account with invalid retention_days",
			"account_id", account.ID, "account", account.Name, "retention_days", days)
		cur.drained = true
		return false
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	batch, err := p.fetchExpiredBatch(account, cutoff, cur.createdAt, cur.id)
	if err != nil {
		p.app.Log.Error("Media retention: batch query failed",
			"account_id", account.ID, "account", account.Name, "error", err)
		return false
	}
	if len(batch) == 0 {
		cur.drained = true
		return false
	}

	for i := range batch {
		select {
		case <-p.stopCh:
			return true
		default:
		}
		if p.capsExceeded(stats, deadline) {
			return true
		}
		stats.scanned++
		p.app.retentionPurgeMessage(&batch[i], time.Now(), stats)
		if i < len(batch)-1 {
			select {
			case <-p.stopCh:
				return true
			case <-time.After(mediaRetentionInterItemDelay):
			}
		}
	}

	last := batch[len(batch)-1]
	cur.createdAt, cur.id = last.CreatedAt, last.ID
	if len(batch) < mediaRetentionBatchSize {
		cur.drained = true
	}
	return true
}

// fetchExpiredBatch returns one keyset-ordered batch of purge candidates for
// an account: media-bearing types, still holding a local file, older than
// the cutoff, and outside any redownload keep-until window.
func (p *MediaRetentionProcessor) fetchExpiredBatch(account *models.WhatsAppAccount, cutoff, afterCreatedAt time.Time, afterID uuid.UUID) ([]models.Message, error) {
	var batch []models.Message
	// Unscoped ON PURPOSE: soft-deleted message rows are invisible but
	// their files still occupy disk — they are the best deletion
	// candidates. The row itself stays soft-deleted.
	q := p.app.DB.Unscoped().
		Where("organization_id = ? AND whats_app_account = ?", account.OrganizationID, account.Name).
		Where("message_type IN ?", []string{
			string(models.MessageTypeImage), string(models.MessageTypeVideo),
			string(models.MessageTypeAudio), string(models.MessageTypeDocument), "sticker",
		}).
		Where("media_url <> '' AND created_at < ?", cutoff).
		// Re-downloaded (restored) media lives out its keep-until window
		// before becoming eligible again.
		Where("metadata ->> 'retention_keep_until' IS NULL OR (metadata ->> 'retention_keep_until')::timestamptz <= now()").
		Where("(created_at, id) > (?, ?)", afterCreatedAt, afterID).
		Order("created_at ASC, id ASC").
		Limit(mediaRetentionBatchSize)
	if err := q.Find(&batch).Error; err != nil {
		return nil, err
	}
	return batch, nil
}

// rollback silently rolls back a lock transaction (best-effort mutex release).
func rollback(tx *gorm.DB) {
	_ = tx.Rollback().Error
}

// mediaRetentionDays reads and validates the account's retention_days.
func mediaRetentionDays(account *models.WhatsAppAccount) int {
	block, ok := account.Settings["media_retention"].(map[string]any)
	if !ok {
		return 0
	}
	enabled, _ := block["enabled"].(bool)
	days := 0
	if v, ok := block["retention_days"].(float64); ok {
		days = int(v)
	}
	if !enabled || days < 1 || days > mediaRetentionMaxDays {
		return 0
	}
	return days
}

// retentionTombstone builds the JSONB fragment merged into metadata by the
// claim UPDATE.
func retentionTombstone(state, purgedAt, originalPath string) string {
	b, _ := json.Marshal(map[string]string{
		retentionStateKey:        state,
		retentionPurgedAtKey:     purgedAt,
		retentionOriginalPathKey: originalPath,
	})
	return string(b)
}

// retentionPurgeMessage runs the claim → refcheck → delete sequence for one
// expired message. See the processor comment for the state machine.
func (a *App) retentionPurgeMessage(msg *models.Message, now time.Time, stats *mediaRetentionPassStats) {
	originalPath := msg.MediaURL

	// 1) Atomic claim: clear the reference and write the tombstone in one
	// guarded UPDATE. The guard (old media_url + no prior retention state)
	// makes a concurrent redownload or a second cleaner lose safely.
	res := a.DB.Exec(
		`UPDATE messages SET
			media_url = '',
			metadata = COALESCE(metadata, '{}'::jsonb) || ?::jsonb
		WHERE id = ? AND media_url = ? AND COALESCE(metadata ->> 'retention_state', '') = ''`,
		retentionTombstone(mediaRetentionStatePurging, now.UTC().Format(time.RFC3339), originalPath),
		msg.ID, originalPath)
	if res.Error != nil {
		a.Log.Error("Media retention: claim failed", "message_id", msg.ID, "error", res.Error)
		return
	}
	if res.RowsAffected == 0 {
		stats.lostRace++
		return
	}

	a.retentionFinishClaim(msg.ID, originalPath, stats)
}

// retentionFinishClaim decides the file's fate for an already-claimed row
// (state 'purging'): refcheck → delete or keep, then finalize the state.
// Split from retentionPurgeMessage so the stuck-sweep reuses it.
func (a *App) retentionFinishClaim(messageID uuid.UUID, originalPath string, stats *mediaRetentionPassStats) {
	referenced, err := a.mediaPathStillReferenced(originalPath)
	if err != nil {
		a.Log.Error("Media retention: refcheck failed", "message_id", messageID, "error", err)
		return // row stays 'purging'; the stuck-sweep retries after the lease
	}
	if referenced {
		// Terminal purged, file kept — another owner still needs it.
		if err := a.retentionSetState(messageID, mediaRetentionStatePurged); err != nil {
			a.Log.Error("Media retention: finalize (shared) failed", "message_id", messageID, "error", err)
			return
		}
		stats.purgedShared++
		return
	}

	freed, err := a.deleteLocalMediaChecked(originalPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		a.Log.Warn("Media retention: file delete failed, will retry after lease",
			"message_id", messageID, "path", originalPath, "error", err)
		if serr := a.retentionSetState(messageID, mediaRetentionStateDeleteFailed); serr != nil {
			a.Log.Error("Media retention: finalize (failed) errored", "message_id", messageID, "error", serr)
		}
		stats.deleteFailed++
		return
	}

	if err := a.retentionSetState(messageID, mediaRetentionStatePurged); err != nil {
		a.Log.Error("Media retention: finalize failed", "message_id", messageID, "error", err)
		return
	}
	stats.purged++
	stats.bytesFreed += freed
}

// retentionSetState overwrites only the state key inside metadata (atomic
// JSONB merge, guarded by the current state being 'purging' or
// 'delete_failed' so a concurrent redownload win is never clobbered).
func (a *App) retentionSetState(messageID uuid.UUID, state string) error {
	frag, _ := json.Marshal(map[string]string{retentionStateKey: state})
	return a.DB.Exec(
		`UPDATE messages SET metadata = COALESCE(metadata, '{}'::jsonb) || ?::jsonb
		WHERE id = ? AND metadata ->> 'retention_state' IN (?, ?)`,
		string(frag), messageID, mediaRetentionStatePurging, mediaRetentionStateDeleteFailed,
	).Error
}

// retentionSweepStuck finishes claims stranded by a crash or a failed file
// delete: rows in 'purging'/'delete_failed' whose lease (retention_purged_at)
// is at least an hour old. Returns the number of rows processed.
func (a *App) retentionSweepStuck(now time.Time, limit int, stats *mediaRetentionPassStats) int {
	var stuck []models.Message
	if err := a.DB.
		Where("media_url = '' AND COALESCE(metadata ->> 'retention_state', '') IN (?, ?)",
			mediaRetentionStatePurging, mediaRetentionStateDeleteFailed).
		Where("COALESCE((metadata ->> 'retention_purged_at')::timestamptz, to_timestamp(0)) < ?", now.Add(-mediaRetentionStuckRetryLease)).
		Order("created_at ASC, id ASC").
		Limit(limit).
		Find(&stuck).Error; err != nil {
		a.Log.Error("Media retention: stuck sweep query failed", "error", err)
		return 0
	}

	for i := range stuck {
		originalPath, _ := stuck[i].Metadata[retentionOriginalPathKey].(string)
		if originalPath == "" {
			// Nothing to finish — terminal purged, defensive.
			_ = a.retentionSetState(stuck[i].ID, mediaRetentionStatePurged)
			continue
		}
		// Refresh the lease first so a crash mid-retry doesn't hammer the
		// same row every pass. Guarded by the current state: a concurrent
		// redownload may have restored the row (clearing retention_state)
		// between the sweep query above and this write — without the guard
		// we'd resurrect a 'purging' marker on a live file and delete it.
		frag, _ := json.Marshal(map[string]string{
			retentionStateKey:    mediaRetentionStatePurging,
			retentionPurgedAtKey: now.UTC().Format(time.RFC3339),
		})
		leaseRes := a.DB.Exec(
			`UPDATE messages SET metadata = COALESCE(metadata, '{}'::jsonb) || ?::jsonb WHERE id = ? AND metadata ->> 'retention_state' IN (?, ?)`,
			string(frag), stuck[i].ID, mediaRetentionStatePurging, mediaRetentionStateDeleteFailed)
		if leaseRes.Error != nil {
			a.Log.Error("Media retention: lease refresh failed", "message_id", stuck[i].ID, "error", leaseRes.Error)
			continue
		}
		if leaseRes.RowsAffected == 0 {
			// Lost the race (row restored or finalized concurrently) — skip.
			continue
		}
		a.retentionFinishClaim(stuck[i].ID, originalPath, stats)
	}
	return len(stuck)
}

// mediaPathStillReferenced reports whether any row still points at the given
// storage-relative path. Deliberately GLOBAL across organizations: the media
// storage root is shared, so a same-path collision in another org must still
// protect the file. Message rows include soft-deleted ones — the rows still
// carry the reference even though nobody can view them.
func (a *App) mediaPathStillReferenced(path string) (bool, error) {
	var n int64
	if err := a.DB.Unscoped().Model(&models.Message{}).
		Where("media_url = ?", path).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := a.DB.Model(&models.ScheduledMessage{}).
		Where("media_url = ? AND status IN ?", path,
			[]string{string(models.ScheduledMessageStatusPending), string(models.ScheduledMessageStatusProcessing)}).
		Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	// Any campaign state protects the file — scheduled/paused campaigns
	// still need it, in-flight ones are sending it, and even completed/
	// failed ones can be retried or duplicated. Filtering by status here
	// deleted files out from under live campaigns.
	if err := a.DB.Model(&models.BulkMessageCampaign{}).
		Where("header_media_local_path = ?", path).
		Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := a.DB.Model(&models.Contact{}).
		Where("avatar_local_path = ?", path).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// deleteLocalMediaChecked deletes a storage-relative file with the same
// traversal/symlink guards as removeLocalMedia, but reports the outcome: the
// freed byte count (0 when the file was already gone) and any error. Unlike
// removeLocalMedia (best-effort, for superseded caches) retention must know
// whether the disk actually lost the file.
func (a *App) deleteLocalMediaChecked(relPath string) (int64, error) {
	if relPath == "" {
		return 0, nil
	}
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return 0, err
	}
	fullPath, ok := resolveMediaPath(baseDir, relPath)
	if !ok {
		// Unresolvable (traversal/symlink/missing) — treat as already gone.
		return 0, os.ErrNotExist
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	if err := os.Remove(fullPath); err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// messageRetentionState returns the row's retention_state (” when normal).
func messageRetentionState(msg *models.Message) string {
	if msg.Metadata == nil {
		return ""
	}
	return asString(msg.Metadata[retentionStateKey])
}

// retentionKeepUntilFor computes the keep-until instant a successful
// redownload should record: one full retention window of the account that
// owns the message by name (the same account the purge scan matches on).
// Zero time when no effective retention applies — nothing will purge the
// message, so no protection is needed.
func (a *App) retentionKeepUntilFor(msg *models.Message, now time.Time) time.Time {
	var account models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND name = ?",
		msg.OrganizationID, msg.WhatsAppAccount).First(&account).Error; err != nil {
		return time.Time{}
	}
	days := mediaRetentionDays(&account)
	if days < 1 {
		return time.Time{}
	}
	return now.AddDate(0, 0, days)
}

// applyRetentionRestore is the metadata side of a successful explicit
// redownload: drop the tombstone keys (retention_state included, so ServeMedia
// and the frontend card stop treating the message as purged) and record
// restored-at plus keep-until when an effective retention window exists.
// Executed as one UPDATE alongside the media_url write.
func (a *App) applyRetentionRestore(msg *models.Message, relativePath, sniffedType string, now time.Time) error {
	// Postgres refuses multiple assignments to the same column, so the key
	// removals and the merge compose into ONE jsonb expression.
	metadataExpr := `(COALESCE(metadata, '{}'::jsonb) - 'retention_state' - 'retention_purged_at' - 'retention_original_path')`
	args := []any{relativePath}

	merge := map[string]string{}
	if !now.IsZero() {
		merge[retentionRestoredAtKey] = now.UTC().Format(time.RFC3339)
		if keepUntil := a.retentionKeepUntilFor(msg, now); !keepUntil.IsZero() {
			merge[retentionKeepUntilKey] = keepUntil.UTC().Format(time.RFC3339)
		}
	}
	if len(merge) > 0 {
		frag, _ := json.Marshal(merge)
		metadataExpr += ` || ?::jsonb`
		args = append(args, string(frag))
	}

	assignments := []string{
		"media_url = ?",
		"metadata = " + metadataExpr,
	}
	if sniffedType != "" {
		assignments = append(assignments, "media_mime_type = ?")
		args = append(args, sniffedType)
	}

	return a.DB.Exec(
		fmt.Sprintf("UPDATE messages SET %s WHERE id = ?", joinStrings(assignments, ", ")),
		append(args, msg.ID)...,
	).Error
}

// joinStrings joins with sep (tiny helper to keep SQL building readable).
func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
