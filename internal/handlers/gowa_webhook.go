package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/contactutil"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm/clause"
)

// GowaWebhookHandler processes incoming webhook events from a GOWA instance.
// GOWA sends POST requests with a JSON body containing event, device_id, and
// a payload object. The X-Hub-Signature-256 header carries an HMAC-SHA256
// signature computed over the raw body.
//
// There is no GET challenge flow for GOWA — verification is HMAC-only.
func (a *App) GowaWebhookHandler(r *fastglue.Request) error {
	return a.handleGowaWebhook(r, "")
}

// GowaWebhookHandlerDevice is the per-device variant. When GOWA is configured
// with per-device webhook URLs (v8.10.0), each device points to a unique path
// /api/gowa/webhook/{device_id}. The path device ID overrides the payload's
// device_id field, which is more reliable since the payload field can be
// omitted when the JID can't be mapped.
func (a *App) GowaWebhookHandlerDevice(r *fastglue.Request) error {
	deviceID, _ := r.RequestCtx.UserValue("device_id").(string)
	return a.handleGowaWebhook(r, deviceID)
}

// handleGowaWebhook is the shared logic for both the single-endpoint and
// per-device webhook routes. When pathDeviceID is non-empty it overrides
// the payload's device_id.
func (a *App) handleGowaWebhook(r *fastglue.Request, pathDeviceID string) error {
	rawBody := r.RequestCtx.PostBody()

	// Parse the envelope first so we can resolve the account by device_id.
	var envelope gowa.WebhookPayload
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		a.Log.Error("Failed to parse GOWA webhook payload", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid payload", nil, "")
	}

	// Per-device routing: prefer the path device_id, falling back to the
	// payload device_id. GOWA's per-device webhook URL embeds an identifier in
	// the path. A previous platform registered per-device webhooks with its OWN
	// account UUID in the path; those registrations persist on the GOWA side
	// and cannot always be cleared via the API (notably for devices whose ids
	// contain non-ASCII characters, where GOWA's path router 404s). To stay
	// forward-compatible, try the path id first, and if it does not resolve to
	// an account, retry with the payload device_id — which GOWA populates with
	// the device JID and getGowaAccountByDeviceID resolves via gowa_jid. The
	// HMAC check below still authenticates the event against the resolved
	// account's secret, so the fallback only changes routing, not trust.
	deviceID := envelope.DeviceID
	if pathDeviceID != "" {
		deviceID = pathDeviceID
	}

	if deviceID == "" {
		a.Log.Warn("GOWA webhook missing device_id")
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing device_id", nil, "")
	}

	// Resolve the WhatsApp account for this GOWA device.
	account, err := a.getGowaAccountByDeviceID(deviceID)
	if err != nil && pathDeviceID != "" && envelope.DeviceID != "" && envelope.DeviceID != pathDeviceID {
		// Legacy path id (e.g. a prior platform's UUID) did not resolve; retry
		// with the payload device_id (the device JID).
		if acct, err2 := a.getGowaAccountByDeviceID(envelope.DeviceID); err2 == nil {
			account = acct
			err = nil
			deviceID = envelope.DeviceID
		}
	}
	if err != nil {
		a.Log.Warn("Unknown GOWA device", "device_id", deviceID, "error", err)
		// Return the same generic rejection as a signature failure so an
		// attacker cannot distinguish an unconfigured device from a bad
		// signature (FR-023: indistinguishable responses).
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Webhook verification failed", nil, "")
	}
	// Normalize so downstream (inbox, event-key, broadcast) uses the resolved
	// device id, not a stale path UUID.
	envelope.DeviceID = deviceID

	// Verify HMAC signature using the account's webhook secret (FAIL-CLOSED).
	// All rejection paths return the exact same error message so that an
	// attacker cannot enumerate valid devices (FR-023). Never log the secret.
	sigHeader := string(r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256"))
	if account.GowaWebhookSecret == "" {
		a.Log.Warn("GOWA account has no webhook secret configured", "device_id", envelope.DeviceID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Webhook verification failed", nil, "")
	}
	if sigHeader == "" {
		a.Log.Warn("GOWA webhook verification failed", "device_id", envelope.DeviceID, "reason", "signature")
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Webhook verification failed", nil, "")
	}
	if !gowa.VerifyWebhookSignature(rawBody, sigHeader, account.GowaWebhookSecret) {
		a.Log.Warn("GOWA webhook verification failed", "device_id", envelope.DeviceID, "reason", "signature")
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Webhook verification failed", nil, "")
	}

	// Replay protection (FR-005): reject webhooks older than 5 minutes.
	// For message events, GOWA puts the timestamp inside the payload object
	// (not at the envelope level), so we peek into the payload to extract it
	// before the replay check. For non-message events GOWA sometimes sends a
	// top-level timestamp and sometimes omits it entirely — the production
	// server has been observed omitting it for message.reaction / message.revoked
	// / message.edited. When no timestamp is available at all, we skip the
	// replay check rather than reject: the HMAC signature above already
	// authenticates the sender, and these events are idempotent (a duplicate
	// reaction / revoke / edit produces the same final state), so the residual
	// replay risk is acceptable.
	replayTS := envelope.Timestamp
	if replayTS == "" && envelope.Event == "message" {
		var msgPeek struct {
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(envelope.Payload, &msgPeek); err == nil {
			replayTS = msgPeek.Timestamp
		}
	}
	if replayTS == "" {
		// No timestamp available — HMAC already authenticated the payload; accept it.
		a.Log.Debug("GOWA webhook has no timestamp; accepting after HMAC verification",
			"device_id", envelope.DeviceID, "event", envelope.Event)
	} else if !gowa.CheckReplay(replayTS, 5*time.Minute) {
		a.Log.Warn("Stale GOWA webhook rejected (replay)", "device_id", envelope.DeviceID, "timestamp", replayTS)
		return r.SendEnvelope(map[string]string{"status": "ok"}) // 200 to prevent GOWA retries
	}

	// DURABLE INGRESS (gap #1): persist the RAW event to the webhook inbox and
	// return 2xx ONLY after the row is durable. A background processor
	// (GowaWebhookProcessor) then dispatches it with retries + dead-lettering,
	// so a crash, DB outage, or panic AFTER this 200 can never silently drop an
	// inbound event — the row survives and is processed on restart. The
	// idempotency key + partial unique index dedupes concurrent/replayed
	// deliveries of the same logical event (gap #8), including call.offer.
	eventKey := deriveGowaEventKey(&envelope)
	evt := models.GowaWebhookEvent{
		OrganizationID: account.OrganizationID,
		AccountID:      account.ID,
		DeviceID:       envelope.DeviceID,
		Event:          envelope.Event,
		EventKey:       eventKey,
		// Copy the body: fasthttp reuses PostBody()'s underlying buffer across
		// requests, so a live reference would be overwritten by the next
		// webhook before the worker reads it.
		RawBody: append([]byte(nil), rawBody...),
		Status:  models.GowaWebhookEventPending,
	}
	res := a.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&evt)
	if res.Error != nil {
		a.Log.Error("Failed to persist GOWA webhook event to inbox",
			"error", res.Error, "event", envelope.Event, "device_id", envelope.DeviceID)
		// 500 → GOWA retries, because we did NOT durably accept the event.
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "accept failed", nil, "")
	}
	// Wake the worker for immediate (near-real-time) processing. Falls back to
	// the worker's poll interval when no processor is wired (tests / worker cmd).
	if a.GowaWebhookNotify != nil {
		a.GowaWebhookNotify()
	}
	return r.SendEnvelope(map[string]string{"status": "ok"})
}

// deriveGowaEventKey returns the idempotency key for a GOWA webhook event —
// the value that makes a redelivery of the SAME logical event collide with the
// original at the inbox boundary. Combined with the partial unique index
// idx_gowa_webhook_events_idempotency, this deduplicates concurrent or replayed
// deliveries (gap #8) so every event is processed at most once.
//
// Keys are namespaced by type to avoid cross-event collisions (a wamid reused
// as a call_id, etc.). Events with no natural identity fall back to a stable
// hash of the payload; chat_presence/connection additionally collapse into a
// per-minute window so bursts don't pile up while still refreshing state.
func deriveGowaEventKey(envelope *gowa.WebhookPayload) string {
	switch envelope.Event {
	case "message":
		var p struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && p.ID != "" {
			return "wamid:" + p.ID
		}
	case "message.ack":
		var p struct {
			IDs         []string `json:"ids"`
			ReceiptType string   `json:"receipt_type"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && len(p.IDs) > 0 {
			return "ack:" + p.ReceiptType + ":" + p.IDs[0]
		}
	case "message.reaction":
		var p struct {
			ReactedMessageID string `json:"reacted_message_id"`
			Reaction         string `json:"reaction"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && p.ReactedMessageID != "" {
			return "react:" + p.ReactedMessageID + ":" + p.Reaction
		}
	case "message.revoked":
		var p struct {
			RevokedMessageID string `json:"revoked_message_id"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && p.RevokedMessageID != "" {
			return "revoke:" + p.RevokedMessageID
		}
	case "message.edited":
		var p struct {
			OriginalMessageID string `json:"original_message_id"`
			Body              string `json:"body"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && p.OriginalMessageID != "" {
			// Include the body hash so a SECOND edit (different body) to the same
			// message is a distinct event, while a redelivery of the same edit
			// dedupes.
			return "edit:" + p.OriginalMessageID + ":" + hashShort(p.Body)
		}
	case "call.offer":
		var p struct {
			CallID string `json:"call_id"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && p.CallID != "" {
			return "call:" + p.CallID
		}
	case "chat_presence":
		var p struct {
			From   string `json:"from"`
			ChatID string `json:"chat_id"`
		}
		if json.Unmarshal(envelope.Payload, &p) == nil && (p.From != "" || p.ChatID != "") {
			// Per-minute window: rapid typing/idle toggles collapse, but a
			// sustained state still refreshes once a minute.
			return fmt.Sprintf("presence:%s:%s:%d", p.From, p.ChatID, time.Now().Unix()/60)
		}
	case "connection":
		var p struct {
			Event string `json:"event"`
		}
		_ = json.Unmarshal(envelope.Payload, &p)
		return fmt.Sprintf("conn:%s:%s:%d", envelope.DeviceID, p.Event, time.Now().Unix()/60)
	}
	// No natural key — stable hash of the payload (redelivery → same hash →
	// deduped). Empty payload collapses to a single sentinel.
	if len(envelope.Payload) == 0 {
		return "empty"
	}
	return "hash:" + hashShort(string(envelope.Payload))
}

// hashShort returns the first 16 hex chars of the SHA-256 of s — a compact,
// collision-resistant identity for fallback event keys.
func hashShort(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// getGowaAccountByDeviceID looks up a WhatsAppAccount by its GOWA device ID.
// The device_id in the webhook is the GOWA session identifier (typically the
// account's phone JID, e.g. "628123456789@s.whatsapp.net").
//
// Resolution is deterministic and priority-ordered: exact gowa_device_id first
// (the reliable GOWA session id), then gowa_jid, then the phone-only fallbacks.
// The partial UNIQUE indexes idx_wa_accounts_gowa_device / idx_wa_accounts_gowa_jid
// (see internal/database/postgres.go) guarantee each value maps to at most one
// account globally, so a duplicated device_id/jid across orgs can no longer make
// First() pick an arbitrary (possibly cross-tenant) account — closing the
// webhook-routing hole where a valid webhook could be HMAC-checked against the
// wrong account's secret or routed into the wrong org.
//
// NOTE: the previous implementation had (a) a single OR query across four
// columns that could match different accounts for device_id vs phone and let
// First() pick arbitrarily, and (b) a fallback that iterated ALL GOWA accounts
// across every org with outbound GetAppStatus calls on each unauthenticated
// request — an abuse vector since removed. Resolution is now a direct indexed
// lookup only.
func (a *App) getGowaAccountByDeviceID(deviceID string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount
	phone := gowa.PhoneFromJID(deviceID)

	// Priority 1: exact gowa_device_id (indexed, unique). Happy path.
	err := a.DB.Where("gowa_device_id = ?", deviceID).First(&account).Error
	// Priority 2: exact gowa_jid (indexed, unique) — GOWA v8 sends the connected JID.
	if err != nil {
		err = a.DB.Where("gowa_jid = ?", deviceID).First(&account).Error
	}
	// Priority 3/4: phone-only fallbacks for legacy rows where the stored id is
	// the bare phone digits. Skipped when phone equals the device_id (no point
	// re-querying the same value) or is empty.
	if err != nil && phone != "" && phone != deviceID {
		err = a.DB.Where("gowa_device_id = ?", phone).First(&account).Error
		if err != nil {
			err = a.DB.Where("gowa_jid = ?", phone).First(&account).Error
		}
	}
	if err != nil {
		return nil, fmt.Errorf("gowa account not found for device %s: %w", deviceID, err)
	}

	// Cache the connected JID as gowa_jid for faster future exact-match lookups.
	if phone != "" && phone != deviceID && account.GowaJID != deviceID {
		a.DB.Model(&account).Update("gowa_jid", deviceID)
	}

	a.decryptAccountSecrets(&account)
	return &account, nil
}

// processGowaMessage normalizes a GOWA message event and feeds it into the
// existing incoming-message pipeline.
func (a *App) processGowaMessage(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaMessage", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var msg gowa.MessagePayload
	if err := json.Unmarshal(envelope.Payload, &msg); err != nil {
		a.Log.Error("Failed to parse GOWA message payload", "error", err)
		return
	}

	// Messages from the connected account itself (is_from_me) are outgoing
	// messages sent from the phone — process them as sent messages, not echoes.
	if msg.IsFromMe {
		a.processGowaOutgoingMessage(account, &msg, envelope.DeviceID)
		return
	}

	// Build the gowa-ui IncomingTextMessage from the GOWA payload.
	// For group/newsletter messages, route to the GROUP/CHANNEL contact (keyed
	// by the @g.us / @newsletter JID), not the individual sender — mirroring
	// processGowaOutgoingMessage which prefers ChatID first. The actual sender
	// is carried separately so the bubble can show who sent it inside the group.
	// Groups (@g.us) and newsletters (@newsletter) are tracked as separate
	// categories: isGroup vs isNewsletter. A newsletter is NOT a group.
	var isGroup, isNewsletter bool
	var senderName, senderJID string
	if msg.IsGroup() {
		isGroup = true
	}
	if msg.IsNewsletter() {
		isNewsletter = true
	}
	if isGroup || isNewsletter {
		senderName = msg.FromName
		senderJID = msg.From
	}
	fromPhone := gowa.PhoneFromJID(msg.From)
	if isGroup || isNewsletter {
		fromPhone = gowa.PhoneFromJID(msg.ChatID)
	}

	incoming := IncomingTextMessage{
		From:      fromPhone,
		To:        gowa.PhoneFromJID(envelope.DeviceID),
		ID:        msg.ID,
		Timestamp: msg.Timestamp,
	}

	// Reply context.
	if msg.RepliedToID != "" {
		incoming.Context = &struct {
			From string `json:"from"`
			ID   string `json:"id"`
		}{
			From: msg.From,
			ID:   msg.RepliedToID,
		}
	}

	// Determine message type and populate the appropriate struct.
	switch {
	case len(msg.Body) > 0 && len(msg.Image) == 0 && len(msg.Video) == 0 &&
		len(msg.Audio) == 0 && len(msg.Document) == 0 && len(msg.Sticker) == 0:
		// Plain text message.
		incoming.Type = "text"
		incoming.Text = &struct {
			Body string `json:"body"`
		}{Body: msg.Body}

	case len(msg.Image) > 0:
		mf := gowa.ResolveMediaField(msg.Image)
		incoming.Type = "image"
		incoming.Image = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
			SHA256   string `json:"sha256"`
			Caption  string `json:"caption,omitempty"`
		}{
			ID:       mf.URL,
			MimeType: "image/jpeg",
			Caption:  mf.Caption,
		}

	case len(msg.Video) > 0:
		mf := gowa.ResolveMediaField(msg.Video)
		incoming.Type = "video"
		incoming.Video = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
			SHA256   string `json:"sha256"`
			Caption  string `json:"caption,omitempty"`
		}{
			ID:       mf.URL,
			MimeType: "video/mp4",
			Caption:  mf.Caption,
		}

	case len(msg.Audio) > 0:
		mf := gowa.ResolveMediaField(msg.Audio)
		incoming.Type = "audio"
		incoming.Audio = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
		}{
			ID:       mf.URL,
			MimeType: "audio/mpeg",
		}

	case len(msg.Document) > 0:
		mf := gowa.ResolveMediaField(msg.Document)
		incoming.Type = "document"
		// Derive a filename from the URL path if GOWA didn't provide one.
		docFilename := mf.Filename
		if docFilename == "" {
			docFilename = filenameFromPath(mf.URL)
		}
		incoming.Document = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
			SHA256   string `json:"sha256"`
			Filename string `json:"filename,omitempty"`
			Caption  string `json:"caption,omitempty"`
		}{
			ID:       mf.URL,
			MimeType: "application/octet-stream",
			Filename: docFilename,
			Caption:  mf.Caption,
		}

	case len(msg.Sticker) > 0:
		mf := gowa.ResolveMediaField(msg.Sticker)
		incoming.Type = "sticker"
		incoming.Sticker = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
			SHA256   string `json:"sha256"`
			Animated bool   `json:"animated,omitempty"`
		}{
			ID:       mf.URL,
			MimeType: "image/webp",
		}

	case len(msg.VideoNote) > 0:
		// Video note (round, muted) — render as a video. Previously this fell
		// through to the default and was silently dropped as "Unhandled GOWA
		// message type" because it carries no body (gap #10).
		vnf := gowa.ResolveMediaField(msg.VideoNote)
		incoming.Type = "video"
		incoming.Video = &struct {
			ID       string `json:"id"`
			MimeType string `json:"mime_type"`
			SHA256   string `json:"sha256"`
			Caption  string `json:"caption,omitempty"`
		}{
			ID:       vnf.URL,
			MimeType: "video/mp4",
			Caption:  vnf.Caption,
		}

	case msg.Location != nil:
		// Location message. Direct assign: MessagePayload.Location and
		// IncomingTextMessage.Location share an identical anonymous struct
		// shape by construction — keep them in sync if either changes.
		incoming.Type = "location"
		incoming.Location = msg.Location

	case len(msg.Contacts) > 0:
		// Contact-card message. Direct assign (see Location note above).
		incoming.Type = "contacts"
		incoming.Contacts = msg.Contacts

	default:
		// Unknown message type — treat body as text if present.
		if msg.Body != "" {
			incoming.Type = "text"
			incoming.Text = &struct {
				Body string `json:"body"`
			}{Body: msg.Body}
		} else {
			a.Log.Debug("Unhandled GOWA message type", "message_id", msg.ID)
			return
		}
	}

	// Feed into the existing pipeline. For group/newsletter messages, the
	// sender's name (msg.FromName) must NOT be used as the group/newsletter
	// contact's profileName — that would overwrite the group/channel name with
	// whoever sent the latest message. GetOrCreateContact updates profile_name
	// whenever a non-empty value is passed and differs from the stored value, so
	// for groups/newsletters we pass "" to leave the existing name untouched (it
	// was set when the contact was first created, or via a separate name
	// lookup). The actual per-message sender is carried via senderName/senderJID.
	profileName := msg.FromName
	if isGroup || isNewsletter {
		profileName = ""
	}
	a.processIncomingMessage(account, envelope.DeviceID, incoming, profileName, isGroup, isNewsletter, senderName, senderJID)
}

// processGowaOutgoingMessage handles messages sent from the connected phone
// (is_from_me=true). These are outgoing messages that should appear in the chat
// as sent by the user, similar to how Meta echoes work.
func (a *App) processGowaOutgoingMessage(account *models.WhatsAppAccount, msg *gowa.MessagePayload, deviceID string) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaOutgoingMessage", "panic", rv, "device_id", deviceID)
		}
	}()

	// Create or find the contact for the recipient.
	recipientPhone := gowa.PhoneFromJID(msg.ChatID)
	if recipientPhone == "" {
		recipientPhone = gowa.PhoneFromJID(msg.From)
	}

	orgID := account.OrganizationID
	contact, _, err := contactutil.GetOrCreateContact(a.DB, orgID, recipientPhone, "")
	if err != nil {
		a.Log.Error("Failed to find/create contact for outgoing GOWA message", "error", err, "phone", recipientPhone)
		return
	}

	// Mark group/newsletter contacts consistently with the incoming path so the
	// contact list and info panel display the correct badge regardless of which
	// path created the contact. Groups and newsletters are distinct categories:
	// a @newsletter JID sets is_newsletter, NOT is_group_chat.
	if err := contactutil.StampChatCategory(a.DB, contact, msg.IsGroup(), msg.IsNewsletter()); err != nil {
		a.Log.Error("Failed to set chat metadata for outgoing GOWA message", "error", err, "chat_id", msg.ChatID)
	}

	// Determine message type, content, and media from the GOWA payload.
	// Media messages carry the file URL in the polymorphic fields; we download
	// it via the GOWA client and store locally (same as incoming media).

	// Dedup: if this message was already recorded (e.g. sent from the gowa-ui
	// UI, which created a local row with the GOWA-returned wamid), update its
	// reply context in place and skip creating a duplicate. The GOWA echo and
	// the local row share the same WhatsAppMessageID. Scoped to this account:
	// when two org accounts message each other, the recipient account stores
	// its own incoming copy under the same wamid — that copy must not suppress
	// this account's outgoing echo (and vice versa).
	if msg.ID != "" {
		var existing models.Message
		if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			msg.ID, orgID, account.Name).First(&existing).Error; err == nil {
			// Patch the reply context onto the existing row if the echo carries one
			// and the local row doesn't already have it.
			if msg.RepliedToID != "" && !existing.IsReply {
				var replyToMsg models.Message
				if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
					msg.RepliedToID, orgID, account.Name).First(&replyToMsg).Error; err == nil {
					a.DB.Model(&existing).Updates(map[string]any{
						"is_reply":            true,
						"reply_to_message_id": replyToMsg.ID,
					})
				}
			}
			return
		}
	}

	msgType := models.MessageTypeText
	content := msg.Body
	var mediaURL, mediaMime, mediaFilename string

	// A message sent from the connected number's phone is customer-side
	// activity just like a received message: an unassigned conversation must
	// stay claimable (pending), and a closed one reopens as pending. Without
	// this, outgoing-only chats default past the claim gate and any agent can
	// read them without claiming. Runs after the dedup above so echoes of
	// UI-sent messages (already lifecycle-managed at send time) are excluded.
	a.ensureClaimableChatStatus(orgID, contact,
		"🔔 Conversation reopened by a message sent from the phone")

	ctx := context.Background()
	waAccount := account.ToWAAccount()

	switch {
	case len(msg.Image) > 0:
		mf := gowa.ResolveMediaField(msg.Image)
		msgType = models.MessageTypeImage
		content = mf.Caption
		mediaMime = "image/jpeg"
		if localPath, err := a.DownloadAndSaveMedia(ctx, mf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing image", "error", err)
		} else {
			mediaURL = localPath
		}
	case len(msg.Video) > 0:
		mf := gowa.ResolveMediaField(msg.Video)
		msgType = models.MessageTypeVideo
		content = mf.Caption
		mediaMime = "video/mp4"
		if localPath, err := a.DownloadAndSaveMedia(ctx, mf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing video", "error", err)
		} else {
			mediaURL = localPath
		}
	case len(msg.Audio) > 0:
		mf := gowa.ResolveMediaField(msg.Audio)
		msgType = models.MessageTypeAudio
		mediaMime = "audio/mpeg"
		if localPath, err := a.DownloadAndSaveMedia(ctx, mf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing audio", "error", err)
		} else {
			mediaURL = localPath
		}
	case len(msg.Document) > 0:
		mf := gowa.ResolveMediaField(msg.Document)
		msgType = models.MessageTypeDocument
		content = mf.Caption
		mediaFilename = mf.Filename
		mediaMime = "application/octet-stream"
		if localPath, err := a.DownloadAndSaveMedia(ctx, mf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing document", "error", err)
		} else {
			mediaURL = localPath
		}
	case len(msg.Sticker) > 0:
		mf := gowa.ResolveMediaField(msg.Sticker)
		msgType = models.MessageTypeImage // stickers render as images
		mediaMime = "image/webp"
		if localPath, err := a.DownloadAndSaveMedia(ctx, mf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing sticker", "error", err)
		} else {
			mediaURL = localPath
		}
	case len(msg.VideoNote) > 0:
		// Video note echoed from the phone — render as a video (gap #10).
		vnf := gowa.ResolveMediaField(msg.VideoNote)
		msgType = models.MessageTypeVideo
		content = vnf.Caption
		mediaMime = "video/mp4"
		if localPath, err := a.DownloadAndSaveMedia(ctx, vnf.URL, mediaMime, waAccount); err != nil {
			a.Log.Error("Failed to download outgoing video note", "error", err)
		} else {
			mediaURL = localPath
		}
	}

	// Fallback: if there's no body and no media, use a placeholder.
	if content == "" && mediaURL == "" {
		content = ""
	}

	// Create the outgoing message record
	outgoing := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    orgID,
		WhatsAppAccount:   account.Name,
		ContactID:         contact.ID,
		WhatsAppMessageID: msg.ID,
		Direction:         models.DirectionOutgoing,
		MessageType:       msgType,
		Content:           content,
		MediaURL:          mediaURL,
		MediaMimeType:     mediaMime,
		MediaFilename:     mediaFilename,
		Status:            models.MessageStatusSent,
	}

	// Reply context. When a message is sent from the phone as a reply to
	// another message, GOWA echoes it back with RepliedToID set to the
	// original message's WhatsApp ID. Resolve it to the local message row so
	// the chat bubble renders as a quote (mirrors the incoming path). Without
	// this, text replies sent from the phone show as plain messages instead
	// of quoted replies.
	if msg.RepliedToID != "" {
		var replyToMsg models.Message
		if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			msg.RepliedToID, orgID, account.Name).First(&replyToMsg).Error; err == nil {
			outgoing.IsReply = true
			outgoing.ReplyToMessageID = &replyToMsg.ID
		} else {
			a.Log.Debug("Outgoing reply-to message not found locally", "reply_to_wamid", msg.RepliedToID)
		}
	}

	// Create the outgoing message record. The partial unique index
	// idx_messages_org_account_wamid is the race backstop: if a concurrent
	// echo/webhook already inserted this (org, account, wamid), ON CONFLICT DO
	// NOTHING skips the insert and RowsAffected is 0 — in that case skip the
	// preview/broadcast side effects so we don't double-emit.
	outgResult := a.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(outgoing)
	if outgResult.Error != nil {
		a.Log.Error("Failed to save outgoing GOWA message", "error", outgResult.Error)
		return
	}
	if outgResult.RowsAffected == 0 {
		a.Log.Debug("Outgoing GOWA message race-lost to a concurrent insert; skipping side effects",
			"message_id", outgoing.ID, "wamid", msg.ID)
		return
	}

	// Build a preview text for the contact list.
	preview := content
	if preview == "" {
		switch msgType {
		case models.MessageTypeImage:
			preview = "📷 Photo"
		case models.MessageTypeVideo:
			preview = "🎬 Video"
		case models.MessageTypeAudio:
			preview = "🎵 Audio"
		case models.MessageTypeDocument:
			preview = "📎 Document"
		}
	}

	// Update contact's last message preview
	a.DB.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"last_message_at":      gowa.ParseTimestamp(msg.Timestamp),
		"last_message_preview": preview,
		"whats_app_account":    account.Name,
	})

	// Broadcast via WebSocket so the chat UI updates in real-time.
	a.broadcastNewMessage(orgID, outgoing, contact)

	a.Log.Info("Saved outgoing GOWA message from phone", "message_id", outgoing.ID, "contact_id", contact.ID, "type", msgType, "has_media", mediaURL != "")
}

// processGowaAck converts GOWA receipt events into status updates.
// GOWA sends "delivered" and "read" receipts with an array of message IDs.
// The resolved `account` is passed in (like every other GOWA sub-handler) so
// the status update is scoped to the acking device's org + account WITHOUT a
// second, non-write-back lookup via getWhatsAppAccountCached — which was the
// root cause of delivered/read ticks never advancing (the scoped WHERE found
// nothing and silently no-op'd).
func (a *App) processGowaAck(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaAck", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	if account == nil {
		a.Log.Warn("processGowaAck called with nil account; skipping", "device_id", envelope.DeviceID)
		return
	}

	var ack gowa.AckPayload
	if err := json.Unmarshal(envelope.Payload, &ack); err != nil {
		a.Log.Error("Failed to parse GOWA ack payload", "error", err)
		return
	}

	// Map GOWA receipt types to gowa-ui status values.
	var status string
	switch ack.ReceiptType {
	case "read":
		status = "read"
	case "delivered":
		status = "delivered"
	default:
		a.Log.Debug("Unhandled GOWA receipt type", "receipt_type", ack.ReceiptType)
		return
	}

	// Process each message ID in the batch. Use the already-resolved account's
	// org + name directly (the same scalars the other handlers pass through)
	// instead of re-resolving via getWhatsAppAccountCached, which does not
	// write back the JID and would scope the lookup to (uuid.Nil, "").
	webhookStatus := WebhookStatus{
		Status:    status,
		Timestamp: envelope.Timestamp, // top-level for ack events
	}
	for _, msgID := range ack.IDs {
		webhookStatus.ID = msgID
		a.updateMessageStatus(account.OrganizationID, account.Name, msgID, status, webhookStatus.Errors)
	}
}

// processGowaChatPresence handles typing/recording presence events and
// broadcasts them to connected clients so the UI can show a live typing
// indicator (gap #9 — previously parsed and only logged, with an explicit TODO).
func (a *App) processGowaChatPresence(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaChatPresence", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var presence gowa.ChatPresencePayload
	if err := json.Unmarshal(envelope.Payload, &presence); err != nil {
		a.Log.Error("Failed to parse GOWA chat_presence payload", "error", err)
		return
	}

	// Determine the human-readable presence.
	var activity string
	switch {
	case presence.State == "composing" && presence.Media == "audio":
		activity = "recording"
	case presence.State == "composing":
		activity = "typing"
	default:
		activity = "idle"
	}

	a.Log.Debug("GOWA chat presence",
		"device_id", envelope.DeviceID,
		"from", presence.From,
		"chat_id", presence.ChatID,
		"activity", activity,
		"is_group", presence.IsGroup,
	)

	// Broadcast so the frontend can render a typing/recording indicator for the
	// matching contact. chat_id is the conversation JID (group @g.us or 1:1
	// @s.whatsapp.net); the frontend matches it against the open contact.
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeChatPresence,
			Payload: map[string]any{
				"chat_id":  presence.ChatID,
				"from":     presence.From,
				"activity": activity,
				"is_group": presence.IsGroup,
			},
		})
	}
}

// processGowaConnection handles device connection state changes.
// When a GOWA device connects or disconnects, we update the account status
// so the frontend can reflect the current connection state.
func (a *App) processGowaConnection(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaConnection", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var conn gowa.ConnectionPayload
	if err := json.Unmarshal(envelope.Payload, &conn); err != nil {
		a.Log.Error("Failed to parse GOWA connection payload", "error", err)
		return
	}

	// Map GOWA connection events to account status.
	var newStatus string
	switch conn.Event {
	case gowa.ConnectionConnected:
		newStatus = "active"
	case gowa.ConnectionDisconnected, gowa.ConnectionConnecting:
		newStatus = "reconnecting"
	case gowa.ConnectionLogout:
		newStatus = "disconnected"
	default:
		a.Log.Debug("Unhandled GOWA connection sub-event", "event", conn.Event, "device_id", envelope.DeviceID)
		return
	}

	// Update the account status in the database.
	if err := a.DB.Model(&models.WhatsAppAccount{}).
		Where("id = ?", account.ID).
		Update("status", newStatus).Error; err != nil {
		a.Log.Error("Failed to update GOWA account status",
			"account_id", account.ID,
			"device_id", envelope.DeviceID,
			"new_status", newStatus,
			"error", err,
		)
		return
	}

	a.Log.Info("GOWA device connection state updated",
		"device_id", envelope.DeviceID,
		"connection_event", conn.Event,
		"new_status", newStatus,
		"reason", conn.Reason,
	)

	// Broadcast to connected frontend clients via WebSocket.
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: "gowa_connection",
			Payload: map[string]any{
				"account_id": account.ID,
				"device_id":  envelope.DeviceID,
				"status":     newStatus,
				"connected":  conn.Event == gowa.ConnectionConnected,
			},
		})
	}

	// A (re)connected device has just completed GOWA's own history sync, and
	// GOWA never replays that history via webhook — pull it into the messages
	// table now so conversations appear without any manual action. Cooldown
	// inside AutoSyncGowaHistory dedupes bursts of connection events.
	if conn.Event == gowa.ConnectionConnected {
		go a.AutoSyncGowaHistory(account)
	}
}

// processGowaReaction handles incoming reaction events.
// GOWA sends this when a contact reacts to a message with an emoji.
func (a *App) processGowaReaction(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaReaction", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var react gowa.ReactionPayload
	if err := json.Unmarshal(envelope.Payload, &react); err != nil {
		a.Log.Error("Failed to parse GOWA reaction payload", "error", err)
		return
	}

	a.Log.Info("GOWA message reaction received",
		"device_id", envelope.DeviceID,
		"reacted_message_id", react.ReactedMessageID,
		"emoji", react.Reaction,
		"from", react.From,
	)

	// Update the original message with the reaction via the existing handler.
	if react.Reaction != "" {
		a.handleIncomingReaction(account, gowa.PhoneFromJID(react.From), react.ReactedMessageID, react.Reaction, "")
	} else {
		a.handleIncomingReaction(account, gowa.PhoneFromJID(react.From), react.ReactedMessageID, "", "")
	}
}

// processGowaRevoked handles message revocation (unsend) events.
func (a *App) processGowaRevoked(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaRevoked", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var revoked gowa.RevokedPayload
	if err := json.Unmarshal(envelope.Payload, &revoked); err != nil {
		a.Log.Error("Failed to parse GOWA revoked payload", "error", err)
		return
	}

	a.Log.Info("GOWA message revoked", "revoked_message_id", revoked.RevokedMessageID)

	// Mark the message as revoked in the database. We use a dedicated
	// MessageStatusRevoked value (not "failed") so the UI can render a
	// distinct "[message revoked]" placeholder instead of an error state,
	// and so the outbound revoke handler can reuse the exact same status.
	// We flip ONLY status — the original content/media stays in the DB so the
	// UI can render it under a "deleted" overlay. The frontend keys the
	// revoked render entirely off status === "revoked".
	var msg models.Message
	if err := a.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			revoked.RevokedMessageID, account.OrganizationID, account.Name).
		Updates(map[string]any{
			"status": models.MessageStatusRevoked,
		}).Error; err != nil {
		a.Log.Error("Failed to mark GOWA message as revoked",
			"revoked_message_id", revoked.RevokedMessageID, "error", err)
		return
	}

	// Broadcast the revoked status over WebSocket so every open client
	// updates the bubble in real time. The existing status_update handler
	// on the frontend keys off message_id, so we resolve the row first.
	if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
		revoked.RevokedMessageID, account.OrganizationID, account.Name).
		First(&msg).Error; err != nil {
		return
	}

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeStatusUpdate,
			Payload: map[string]any{
				"message_id": msg.ID,
				"contact_id": msg.ContactID,
				"status":     models.MessageStatusRevoked,
			},
		})
	}
}

// processGowaEdited handles message edit events.
// filenameFromPath extracts the last path segment from a URL or file path,
// used as a fallback filename when GOWA doesn't provide one explicitly.
func filenameFromPath(path string) string {
	// Strip query string
	if idx := strings.Index(path, "?"); idx > 0 {
		path = path[:idx]
	}
	// Get last segment
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if path == "" {
		path = "document"
	}
	return path
}

func (a *App) processGowaEdited(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaEdited", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var edited gowa.EditedPayload
	if err := json.Unmarshal(envelope.Payload, &edited); err != nil {
		a.Log.Error("Failed to parse GOWA edited payload", "error", err)
		return
	}

	a.Log.Info("GOWA message edited",
		"original_message_id", edited.OriginalMessageID,
		"new_body_length", len(edited.Body),
	)

	// Update the message content in the database. Scoped to the owning account
	// (mirrors ack/reaction/revoke): two org accounts can share a WAMID, so an
	// edit from one device must only touch THAT account's copy.
	res := a.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			edited.OriginalMessageID, account.OrganizationID, account.Name).
		Update("content", edited.Body)
	if res.Error != nil {
		a.Log.Error("Failed to update GOWA edited message content",
			"original_message_id", edited.OriginalMessageID, "error", res.Error)
		return
	}
	if res.RowsAffected == 0 {
		// Unknown wamid locally (e.g. a message we never stored). Nothing to
		// patch or broadcast.
		a.Log.Debug("GOWA edit target not found locally; skipping side effects",
			"original_message_id", edited.OriginalMessageID)
		return
	}

	// Re-fetch the row to drive side effects with the resolved local ids.
	var msg models.Message
	if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
		edited.OriginalMessageID, account.OrganizationID, account.Name).
		First(&msg).Error; err != nil {
		a.Log.Error("Edited message vanished after update", "error", err,
			"original_message_id", edited.OriginalMessageID)
		return
	}

	// Refresh the contact-list preview ONLY when the edit targets the
	// conversation's newest message (the only row whose preview the list
	// shows). Avoids clobbering a newer message's preview with an older edit.
	var lastMsg models.Message
	if err := a.DB.Where("contact_id = ? AND organization_id = ?", msg.ContactID, account.OrganizationID).
		Order("created_at DESC").Limit(1).First(&lastMsg).Error; err == nil && lastMsg.ID == msg.ID {
		a.DB.Model(&models.Contact{}).Where("id = ?", msg.ContactID).
			Update("last_message_preview", getMessagePreviewFromContent(msg.MessageType, edited.Body))
	}

	// Real-time patch: open chat bubbles re-render the new body without a refetch (gap #11).
	a.broadcastMessageEdited(account.OrganizationID, msg.ID, msg.ContactID, edited.Body)

	// Outbound application webhook so integrations see the edit.
	a.DispatchWebhook(account.OrganizationID, models.WebhookEventMessageEdited, MessageEventData{
		MessageID:       msg.ID.String(),
		ContactID:       msg.ContactID.String(),
		MessageType:     msg.MessageType,
		Content:         edited.Body,
		WhatsAppAccount: account.Name,
		Direction:       msg.Direction,
	})
}
