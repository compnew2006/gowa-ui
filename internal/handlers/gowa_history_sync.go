package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/contactutil"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GOWA performs its own history synchronization when a device (re)connects,
// but it only delivers NEW messages via webhook — the synced history is never
// replayed. This file makes gowa-ui pull that history automatically:
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
		identity, isGroup, isNewsletter := gowaChatIdentity(jid)
		if identity == "" {
			continue
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

		// Stamp the owning account name and group/newsletter metadata (mirrors
		// /sync-contacts — shared helpers, see contactutil).
		if err := contactutil.StampAccountName(a.DB, contact, account.Name); err != nil {
			a.Log.Error("Failed to stamp whats_app_account during GOWA message sync", "error", err, "jid", jid)
		}
		if err := contactutil.StampChatCategory(a.DB, contact, isGroup, isNewsletter); err != nil {
			a.Log.Error("Failed to set chat metadata during GOWA message sync", "error", err, "jid", jid)
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
			msgType := gowaMsgTypeToGowaUI(m.MediaType)
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
				// MediaURL is populated below for media messages by downloading
				// the bytes via the GOWA message-ID endpoint (which proxies the
				// WhatsApp CDN fetch through the connected device). This must
				// happen at sync time, BEFORE the WhatsApp CDN link expires —
				// once expired, the bytes are gone and lazy recovery on view
				// (ServeMedia in media.go) cannot fetch them either. A download
				// failure is non-fatal: media_url stays empty and the row falls
				// back to ServeMedia's recovery path as a safety net.
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
				// Keep GOWA's original WhatsApp CDN URL as a hint for diagnostics
				// and potential future recovery. Not used as a local path.
				msg.Metadata["gowa_media_url"] = m.URL
			}

			// Eagerly download media bytes at sync time, before the WhatsApp CDN
			// link expires. Reuses the same DownloadMessageMedia + saveMediaBytes
			// helpers as ServeMedia's recovery path (media.go) — no logic fork.
			// Only attempt for real media types with a WA message ID; skip for
			// status/newsletter JIDs where GOWA rejects the download anyway.
			//
			// Gated by storage.eager_history_media (default false): eager download
			// of every chat's history across many devices can fill the disk. With
			// the flag off (the fresh-start default), media_url metadata is still
			// stored above and ServeMedia's lazy recovery fetches bytes on first
			// view — so no functionality is lost, only the disk-amplifying burst.
			if a.Config.Storage.EagerHistoryMedia && msgType != models.MessageTypeText && m.ID != "" {
				chatJID := gowaChatJID(contact)
				if chatJID != "" {
					dlCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					if data, mediaType, derr := client.DownloadMessageMedia(dlCtx, account.ToWAAccount(), m.ID, chatJID); derr == nil && len(data) > 0 {
						if relPath, serr := a.saveMediaBytes(data, mediaType); serr == nil {
							msg.MediaURL = relPath
							// Sniff the real MIME type from the bytes (GOWA's
							// mediaType is generic like "image", not a valid MIME).
							sniffLen := 512
							if len(data) < sniffLen {
								sniffLen = len(data)
							}
							msg.MediaMimeType = http.DetectContentType(data[:sniffLen])
						} else {
							a.Log.Warn("history-sync media save failed",
								"wamid", m.ID, "error", serr)
						}
					} else if derr != nil {
						// Non-fatal: leave media_url empty. ServeMedia's lazy
						// recovery will retry on first view (and the frontend
						// shows a filename fallback if that also fails).
						a.Log.Warn("history-sync media download failed",
							"wamid", m.ID, "jid", chatJID, "error", derr)
					}
					cancel()
				}
			}

			// INSERT against the partial unique index idx_messages_org_account_wamid
			// as the race backstop: a concurrent webhook may have stored the same
			// wamid between our pre-filter and this insert. ON CONFLICT DO NOTHING
			// skips silently; RowsAffected==0 means it already exists, so don't
			// count it or treat it as the chat's newest (would clobber the real
			// newest with an older timestamp).
			createRes := a.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&msg)
			if createRes.Error != nil {
				a.Log.Error("Failed to store GOWA history message", "error", createRes.Error, "msg_id", m.ID)
				continue
			}
			if createRes.RowsAffected == 0 {
				continue
			}
			stats.MessagesStored++

			// Track the newest message (largest timestamp) for the contact stamp.
			// GOWA returns newest-first, so the first non-skipped msg is newest.
			if newest.ID == uuid.Nil {
				newest = msg
			}
		}

		// Always reconcile the contact's last_message_at from the GOWA chat's
		// authoritative last_message_time, EVEN when no new messages were stored.
		// Previously this ran only inside `if newest.ID != uuid.Nil`, so a re-sync
		// of a chat whose messages all already existed left last_message_at frozen
		// at the time of the FIRST sync forever — the conversation then got buried
		// off the visible list (sorted by last_message_at). Forward-only via
		// GREATEST so a clock skew or stale GOWA timestamp can never rewind a
		// newer local value. The preview is only refreshed when we actually saw a
		// new (newest) message this run, to avoid clobbering it with stale text.
		lastAt := gowa.ParseTimestamp(ch.LastMessageTime)
		if lastAt.IsZero() && newest.ID != uuid.Nil {
			lastAt = newest.CreatedAt
		}
		if !lastAt.IsZero() {
			updates := map[string]any{
				"last_message_at": gorm.Expr("GREATEST(COALESCE(last_message_at, '1970-01-01'::timestamptz), ?)", lastAt),
			}
			if newest.ID != uuid.Nil {
				updates["last_message_preview"] = getMessagePreviewFromContent(newest.MessageType, newest.Content)
			}
			a.DB.Model(contact).Updates(updates)
		}
		if newest.ID != uuid.Nil {
			stats.ChatsWithMsgs++
		}
	}

	return stats, nil
}

// gowaChatIdentity derives the contact identity for a GOWA chat JID, matching
// the webhook convention: group/newsletter chats are keyed by their full JID
// (the @g.us/@newsletter suffix is part of the identity), while 1:1 chats use
// the bare phone digits. identity is "" when the JID is unusable. Shared by
// the contact-sync and history-sync chat loops.
func gowaChatIdentity(jid string) (identity string, isGroup, isNewsletter bool) {
	isGroup = strings.HasSuffix(jid, "@g.us")
	isNewsletter = strings.HasSuffix(jid, "@newsletter")
	identity = jid
	if !isGroup && !isNewsletter {
		identity = gowa.PhoneFromJID(jid)
	}
	return identity, isGroup, isNewsletter
}

// ResolveGowaCreds resolves the Basic Auth credentials for a GOWA server by
// its (organization, base URL) pair. It prefers the DB-managed instance
// (created via the UI; credentials are encrypted at rest) — scoped to the
// owning org so two orgs that register the same GOWA base URL resolve their
// OWN credentials, never each other's — and falls back to the config-file
// [[gowa_instances]] section for backward compatibility, then to empty
// credentials. This is the single source of truth for GOWA Basic Auth: the
// provider-registry factory (main) and gowaClientForAccount both call it.
//
// A zero orgID (uuid.Nil) preserves the legacy cross-org lookup for callers
// that don't yet know the owning org; new callers should always pass it.
func ResolveGowaCreds(db *gorm.DB, cfg *config.Config, orgID uuid.UUID, baseURL string) (username, password string) {
	// 1. DB-managed instance (UI-created), org-scoped. Credentials are
	//    encrypted at rest. Org scoping is the tenant-isolation boundary: it
	//    prevents org B from resolving org A's credentials for the same URL.
	q := db.Where("base_url = ? AND is_active = ?", baseURL, true)
	if orgID != uuid.Nil {
		q = q.Where("organization_id = ?", orgID)
	}
	var inst models.GowaInstance
	err := q.Order("created_at DESC").First(&inst).Error
	if err == nil {
		inst.DecryptCredentials(cfg.App.EncryptionKey)
		if inst.HasCredentials() {
			return inst.Username, inst.Password
		}
	}

	// 2. Config-file fallback (legacy/manual provisioning), org-scoped when
	//    possible. FindGOWAInstance takes a variadic orgID for this.
	if c := cfg.FindGOWAInstance(baseURL, orgIDString(orgID)); c != nil {
		return c.Username, c.Password
	}
	return "", ""
}

// orgIDString renders a uuid.UUID for the variadic orgID parameter of
// FindGOWAInstance, returning "" for uuid.Nil so the config lookup stays
// org-agnostic (legacy behavior) when the org is unknown.
func orgIDString(orgID uuid.UUID) string {
	if orgID == uuid.Nil {
		return ""
	}
	return orgID.String()
}

// gowaClientForAccount returns a gowa.Client for the account's GOWA base URL.
// It prefers the shared provider registry (cached client, invalidated when
// credentials change) and falls back to building one via ResolveGowaCreds
// when the registry isn't wired (e.g. tests).
func (a *App) gowaClientForAccount(account *models.WhatsAppAccount) *gowa.Client {
	if a.WARegistry != nil {
		if c, ok := a.WARegistry.Get(account.ToWAAccount()).(*gowa.Client); ok && c != nil {
			return c
		}
	}
	baseURL := account.GowaBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	user, pass := ResolveGowaCreds(a.DB, a.Config, account.OrganizationID, baseURL)
	return gowa.New(baseURL, user, pass)
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
// history GOWA synced while gowa-ui was down) and then re-syncs on the
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
	// gowa-ui restart) without waiting a full interval.
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
