package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/contactutil"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
)

// GOWA performs its own history synchronization when a device (re)connects,
// but it only delivers NEW messages via webhook — the synced history is never
// replayed. This file makes whatomate pull that history automatically:
//
//   - syncGowaHistory is the reusable backfill core (also behind the manual
//     POST .../sync-messages endpoint).
//   - AutoSyncGowaHistory wraps it with a per-account cooldown so it can be
//     fired from webhooks/tickers without hammering GOWA.
//   - GowaHistorySyncProcessor runs an initial pass at startup and a periodic
//     re-sync, mirroring the SLA/chat-reset/scheduled-message processors.
//   - A device "connected" webhook additionally triggers an immediate sync
//     (see processGowaConnection).

const (
	// gowaHistorySyncCooldown is the minimum gap between automatic history
	// syncs for the same account. Connection webhooks can fire in bursts and
	// the periodic processor overlaps with them; the cooldown keeps a single
	// bounded pull per account per window.
	gowaHistorySyncCooldown = 10 * time.Minute

	// gowaAutoSyncPerChatLimit caps messages pulled per chat during automatic
	// syncs (newest-first tail), matching the manual endpoint's default.
	gowaAutoSyncPerChatLimit = 50
)

// gowaHistorySyncStats summarizes a history backfill run.
type gowaHistorySyncStats struct {
	ChatsSeen      int
	ChatsWithMsgs  int
	MessagesStored int
}

// syncGowaHistory backfills a device's message history from GOWA into the
// messages table. It iterates the device's chat list (GET /chats), and for
// each chat pulls recent messages (GET /chat/{jid}/messages), upserting them
// as messages rows keyed by whats_app_message_id (idempotent), and stamps each
// contact's last_message_at/preview from its newest message.
//
// perChatLimit caps messages fetched per chat (default 50, newest-first) so
// the operation stays bounded for large histories. maxChats limits how many
// chats are processed (0 = all).
func (a *App) syncGowaHistory(ctx context.Context, client *gowa.Client, account *models.WhatsAppAccount, deviceID string, perChatLimit, maxChats int) (gowaHistorySyncStats, error) {
	if perChatLimit <= 0 || perChatLimit > 100 {
		perChatLimit = gowaAutoSyncPerChatLimit
	}
	orgID := account.OrganizationID

	chats, totalChats, err := client.ListChats(ctx, deviceID, gowa.ListChatsOptions{Limit: 100})
	if err != nil {
		return gowaHistorySyncStats{}, fmt.Errorf("list gowa chats: %w", err)
	}
	if maxChats > 0 && len(chats) > maxChats {
		chats = chats[:maxChats]
	}

	stats := gowaHistorySyncStats{ChatsSeen: totalChats}
	for _, ch := range chats {
		jid := strings.TrimSpace(ch.JID)
		if jid == "" {
			continue
		}

		isGroup := strings.HasSuffix(jid, "@g.us")
		isNewsletter := strings.HasSuffix(jid, "@newsletter")
		identity := jid
		if !isGroup && !isNewsletter {
			identity = gowa.PhoneFromJID(jid)
			if identity == "" {
				continue
			}
		}

		contact, _, err := contactutil.GetOrCreateContact(a.DB, orgID, identity, strings.TrimSpace(ch.Name))
		if err != nil {
			a.Log.Error("Failed to upsert contact during GOWA message sync", "error", err, "jid", jid)
			continue
		}

		// Fetch recent history (newest-first). GOWA returns the newest page
		// first at offset 0, so per_chat_limit gives us the tail of the convo.
		msgs, _, err := client.GetChatMessages(ctx, deviceID, jid, gowa.ChatMessagesOptions{Limit: perChatLimit})
		if err != nil {
			a.Log.Error("Failed to fetch GOWA chat messages", "error", err, "device", deviceID, "jid", jid)
			continue
		}
		if len(msgs) == 0 {
			continue
		}

		// Stamp the owning account name (mirrors /sync-contacts): always
		// overwrite so an empty or stale value self-heals. The DB column is
		// whats_app_account (GORM's mapping of the WhatsAppAccount field),
		// not whatsapp_account.
		if contact.WhatsAppAccount != account.Name {
			if err := a.DB.Model(contact).Update("whats_app_account", account.Name).Error; err != nil {
				a.Log.Error("Failed to stamp whats_app_account during GOWA message sync", "error", err, "jid", jid)
			} else {
				contact.WhatsAppAccount = account.Name
			}
		}

		// Stamp group/newsletter metadata if not already set (mirrors
		// /sync-contacts). Groups and newsletters are distinct categories.
		metaKey := ""
		if isGroup {
			metaKey = "is_group_chat"
		} else if isNewsletter {
			metaKey = "is_newsletter"
		}
		if metaKey != "" {
			needsMetaUpdate := false
			if contact.Metadata == nil {
				contact.Metadata = models.JSONB{}
				needsMetaUpdate = true
			}
			// Groups and newsletters are mutually exclusive. Setting one clears
			// the other so legacy contacts that carry both flags self-heal.
			otherKey := ""
			if metaKey == "is_group_chat" {
				otherKey = "is_newsletter"
			} else if metaKey == "is_newsletter" {
				otherKey = "is_group_chat"
			}
			_, hasOther := contact.Metadata[otherKey]
			if contact.Metadata[metaKey] != true {
				contact.Metadata[metaKey] = true
				needsMetaUpdate = true
			}
			if hasOther {
				delete(contact.Metadata, otherKey)
				needsMetaUpdate = true
			}
			if needsMetaUpdate {
				a.DB.Model(contact).Update("metadata", contact.Metadata)
			}
		}

		// Bulk-insert messages, skipping any whose whats_app_message_id already
		// exists (idempotent re-sync). GORM Clause OnConflict would need a
		// unique constraint on the message id; we instead pre-filter by querying
		// existing ids for this chat to avoid a schema change. Scoped to this
		// account: two org accounts chatting with each other share wamids across
		// their copies, and syncing one device must not skip messages that only
		// exist as the other account's copy.
		existing := make(map[string]bool, len(msgs))
		{
			ids := make([]string, 0, len(msgs))
			for _, m := range msgs {
				if m.ID != "" {
					ids = append(ids, m.ID)
				}
			}
			if len(ids) > 0 {
				var found []string
				a.DB.Model(&models.Message{}).
					Where("whats_app_message_id IN ? AND organization_id = ? AND whats_app_account = ?",
						ids, account.OrganizationID, account.Name).
					Pluck("whats_app_message_id", &found)
				for _, id := range found {
					existing[id] = true
				}
			}
		}

		var newest models.Message
		for _, m := range msgs {
			if m.ID == "" || existing[m.ID] {
				continue
			}
			ts := gowa.ParseTimestamp(m.Timestamp)
			if ts.IsZero() {
				ts = time.Now()
			}
			direction := models.DirectionIncoming
			status := models.MessageStatusReceived
			if m.IsFromMe {
				direction = models.DirectionOutgoing
				status = models.MessageStatusSent
			}
			msgType := gowaMsgTypeToWhatomate(m.MediaType)
			// For media messages, prefer the stored content as caption only if
			// it's a text body; otherwise leave content empty (media lives in
			// MediaURL/Filename).
			content := m.Content
			if msgType != models.MessageTypeText {
				content = "" // caption is not reliably separated by GOWA history
			}

			msg := models.Message{
				// Set CreatedAt to the message's real timestamp so historical
				// messages render in chronological order in the chat view
				// (GORM's autoCreateTime honors an explicitly-set value).
				BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: ts},
				OrganizationID:    orgID,
				WhatsAppAccount:   account.Name,
				ContactID:         contact.ID,
				WhatsAppMessageID: m.ID,
				Direction:         direction,
				MessageType:       msgType,
				Content:           content,
				// NOTE: MediaURL is intentionally left empty here. History sync
				// gives us GOWA's server-side URL (m.URL), NOT a local file path —
				// the bytes were never downloaded to disk. Writing m.URL into
				// media_url would create a lying row: ServeMedia would try to serve
				// a non-existent local file and 404. Instead, leave media_url empty
				// so the row is honest ("no local media yet"), and let ServeMedia's
				// auto-recovery lazily download the bytes via WhatsAppMessageID on
				// first view. See internal/handlers/media.go ServeMedia for the
				// recovery path. Stash the original GOWA URL in metadata as a
				// fallback in case the message-ID-based recovery is unavailable.
				MediaURL:      "",
				MediaFilename: m.Filename,
				Status:        status,
			}
			// Preserve the original timestamp via Metadata so the UI can render
			// historical order even though created_at is now.
			if msg.Metadata == nil {
				msg.Metadata = models.JSONB{}
			}
			msg.Metadata["synced_from_history"] = true
			msg.Metadata["gowa_timestamp"] = m.Timestamp
			if m.URL != "" {
				// Keep GOWA's original URL for potential lazy recovery. Not used
				// as a local path — only as a hint for future download attempts.
				msg.Metadata["gowa_media_url"] = m.URL
			}

			if err := a.DB.Create(&msg).Error; err != nil {
				a.Log.Error("Failed to store GOWA history message", "error", err, "msg_id", m.ID)
				continue
			}
			stats.MessagesStored++

			// Track the newest message (largest timestamp) for the contact stamp.
			// GOWA returns newest-first, so the first non-skipped msg is newest.
			if newest.ID == uuid.Nil {
				newest = msg
			}
		}

		// Stamp the contact's last_message_at/preview from the newest message.
		if newest.ID != uuid.Nil {
			preview := getMessagePreviewFromContent(newest.MessageType, newest.Content)
			// Use the chat's last_message_time (authoritative from GOWA) when available.
			lastAt := gowa.ParseTimestamp(ch.LastMessageTime)
			if lastAt.IsZero() {
				lastAt = time.Now()
			}
			a.DB.Model(contact).Updates(map[string]any{
				"last_message_at":      lastAt,
				"last_message_preview": preview,
			})
			stats.ChatsWithMsgs++
		}
	}

	return stats, nil
}

// gowaClientForAccount builds a gowa.Client for the account's GOWA base URL,
// resolving Basic Auth credentials the same way as the provider registry
// (resolveGowaCreds in main): DB-managed instance first, config-file fallback,
// empty credentials last.
func (a *App) gowaClientForAccount(account *models.WhatsAppAccount) *gowa.Client {
	baseURL := account.GowaBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	var inst models.GowaInstance
	err := a.DB.Where("base_url = ? AND is_active = ?", baseURL, true).
		Order("created_at DESC").
		First(&inst).Error
	if err == nil {
		inst.DecryptCredentials(a.Config.App.EncryptionKey)
		if inst.HasCredentials() {
			return gowa.New(baseURL, inst.Username, inst.Password)
		}
	}

	if c := a.Config.FindGOWAInstance(baseURL); c != nil {
		return gowa.New(baseURL, c.Username, c.Password)
	}
	return gowa.New(baseURL, "", "")
}

// tryAcquireGowaHistorySync records a sync attempt for the account and reports
// whether it may proceed. Attempts within gowaHistorySyncCooldown of the last
// one are rejected, deduplicating bursts of connection webhooks and overlap
// with the periodic processor.
func (a *App) tryAcquireGowaHistorySync(accountID uuid.UUID) bool {
	a.gowaHistorySyncMu.Lock()
	defer a.gowaHistorySyncMu.Unlock()
	if a.gowaHistoryLastSync == nil {
		a.gowaHistoryLastSync = make(map[uuid.UUID]time.Time)
	}
	if last, ok := a.gowaHistoryLastSync[accountID]; ok && time.Since(last) < gowaHistorySyncCooldown {
		return false
	}
	a.gowaHistoryLastSync[accountID] = time.Now()
	return true
}

// AutoSyncGowaHistory backfills the account's message history from GOWA
// without any user action. Safe to call from goroutines; no-ops when the
// account has no GOWA device or when a sync ran within the cooldown window.
func (a *App) AutoSyncGowaHistory(account *models.WhatsAppAccount) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in AutoSyncGowaHistory", "panic", rv)
		}
	}()

	if account == nil || account.GowaDeviceID == "" {
		return
	}
	if !a.tryAcquireGowaHistorySync(account.ID) {
		return
	}

	client := a.gowaClientForAccount(account)
	stats, err := a.syncGowaHistory(context.Background(), client, account, account.GowaDeviceID, gowaAutoSyncPerChatLimit, 0)
	if err != nil {
		// Warn, not error: GOWA being briefly unreachable during (re)connects
		// is expected; the periodic processor retries after the cooldown.
		a.Log.Warn("GOWA history auto-sync failed", "error", err,
			"account", account.Name, "device_id", account.GowaDeviceID)
		return
	}
	if stats.MessagesStored > 0 {
		a.Log.Info("GOWA history auto-sync completed",
			"account", account.Name,
			"device_id", account.GowaDeviceID,
			"chats_seen", stats.ChatsSeen,
			"chats_with_msgs", stats.ChatsWithMsgs,
			"messages_stored", stats.MessagesStored)
	} else {
		a.Log.Debug("GOWA history auto-sync found nothing new",
			"account", account.Name, "device_id", account.GowaDeviceID,
			"chats_seen", stats.ChatsSeen)
	}
}

// GowaHistorySyncProcessor periodically backfills GOWA message history for
// every GOWA-backed account. It runs an initial pass at startup (covering
// history GOWA synced while whatomate was down) and then re-syncs on the
// ticker; AutoSyncGowaHistory's cooldown keeps overlapping triggers cheap.
type GowaHistorySyncProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

// NewGowaHistorySyncProcessor creates a new GOWA history sync processor.
func NewGowaHistorySyncProcessor(app *App, interval time.Duration) *GowaHistorySyncProcessor {
	return &GowaHistorySyncProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the history sync loop with an immediate initial pass.
func (p *GowaHistorySyncProcessor) Start(ctx context.Context) {
	p.app.Log.Info("GOWA history sync processor started", "interval", p.interval)

	// Initial pass: pick up history GOWA already holds (e.g. after a GOWA or
	// whatomate restart) without waiting a full interval.
	p.syncAll()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("GOWA history sync processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("GOWA history sync processor stopped")
			return
		case <-ticker.C:
			p.syncAll()
		}
	}
}

// Stop stops the processor.
func (p *GowaHistorySyncProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// syncAll runs an auto-sync for every GOWA-backed account that is not logged
// out. Accounts are processed sequentially so a large fleet doesn't hammer
// GOWA instances in parallel; per-account cooldown handles re-entry.
func (p *GowaHistorySyncProcessor) syncAll() {
	var accounts []models.WhatsAppAccount
	if err := p.app.DB.
		Where("gowa_device_id <> '' AND status <> 'disconnected'").
		Find(&accounts).Error; err != nil {
		p.app.Log.Error("Failed to load GOWA accounts for history sync", "error", err)
		return
	}
	for i := range accounts {
		p.app.AutoSyncGowaHistory(&accounts[i])
	}
}
