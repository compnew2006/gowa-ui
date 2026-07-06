package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/gowa"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// GOWA inbound webhook receiver.
//
// GOWA POSTs events to /api/gowa/webhook/{instanceID} for every device that
// has a webhook_url configured. The instanceID in the path is the whatomate
// UUID we encoded into the per-device webhook_url at provisioning time
// (see gowaWebhookURLForInstance). The body is a JSON envelope:
//
//	{
//	  "event":     "message" | "receipt" | "message.reaction" | ...,
//	  "device_id": "<jid of connected account, e.g. 556283088170@s.whatsapp.net>",
//	  "payload":   { ...event-specific fields... }
//	}
//
// Every request is HMAC-authenticated against cfg.Gowa.WebhookSecret via the
// X-Hub-Signature-256 header (sha256=<hex>) — exactly the Meta convention,
// which GOWA's submitWebhook reuses verbatim.
//
// On successful auth + dispatch we ACK with 200 so GOWA stops retrying. Any
// downstream persistence failure is logged and swallowed: returning non-2xx
// would cause GOWA to retry 5 times with exponential backoff, which is the
// wrong policy for "payload unparseable" or "contact row missing" — those
// will not self-heal. Stage 7's polling reconciler is the safety net for
// genuinely transient delivery failures.

// gowaEnvelope is the top-level GOWA webhook body. The Payload is kept as a
// raw json.RawMessage so each event-type handler can decode its own shape
// lazily without needing a giant tagged union.
type gowaEnvelope struct {
	Event    string          `json:"event"`
	DeviceID string          `json:"device_id"`
	Payload  json.RawMessage `json:"payload"`
}

// GowaWebhook is the inbound receiver for GOWA device events. It is
// registered OUTSIDE the auth middleware chain (in main.go before auth is
// applied) because GOWA cannot authenticate as a whatomate user; it
// authenticates via HMAC instead.
//
// POST /api/gowa/webhook/{instanceID}
func (a *App) GowaWebhook(r *fastglue.Request) error {
	if !a.isGowaProvider() {
		// Silently 404 in non-gowa deployments so probes/scanners get nothing.
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Not Found", nil, "")
	}

	instanceIDStr, _ := r.RequestCtx.UserValue("instanceID").(string)
	// If no instanceID in path (global webhook URL fallback), we'll resolve
	// it from the device_id in the payload after HMAC verification.
	var instanceID uuid.UUID
	if instanceIDStr != "" {
		var parseErr error
		instanceID, parseErr = uuid.Parse(instanceIDStr)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instanceID", nil, "")
		}
	}

	body := append([]byte(nil), r.RequestCtx.PostBody()...)
	if len(body) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty body", nil, "")
	}

	// Verify HMAC before doing any work. Constant-time compare.
	sigHeader := r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256")
	if !a.verifyGowaSignature(body, sigHeader) {
		a.Log.Warn("GOWA webhook signature verification failed",
			"instance_id", instanceID, "bytes", len(body))
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid signature", nil, "")
	}

	var env gowaEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		a.Log.Warn("GOWA webhook body unparseable", "error", err)
		// ACK so GOWA doesn't retry; we can't recover from a malformed payload.
		return r.SendEnvelope(map[string]string{"status": "accepted", "note": "unparseable payload"})
	}

	// If no instanceID from path (global webhook URL), resolve from the
	// device_id in the payload. GOWA's device_id is the WhatsApp JID of the
	// connected account (e.g. "201007181781@s.whatsapp.net"). We match it
	// against the instance's phone_number or jid field.
	if instanceID == uuid.Nil {
		if env.DeviceID == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot resolve instance: no instanceID in path and no device_id in payload", nil, "")
		}
		// The device_id from GOWA is the JID. Extract phone digits and find
		// the instance by phone_number.
		phone := jidToPhone(env.DeviceID)
		var inst models.WhatsAppInstance
		if err := a.DB.Where("phone_number = ? OR jid = ?", phone, env.DeviceID).First(&inst).Error; err != nil {
			a.Log.Warn("GOWA webhook: cannot resolve instance from device_id",
				"device_id", env.DeviceID, "phone", phone, "error", err)
			// ACK to stop retries — a stale device_id won't self-heal.
			return r.SendEnvelope(map[string]string{"status": "accepted", "note": "unresolved device"})
		}
		instanceID = inst.ID
	}

	// Resolve the instance from the path UUID. We scope by ID only (not org)
	// because GOWA doesn't carry org context; the path UUID is unforgeable
	// behind the HMAC check.
	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Instance was deleted locally but GOWA is still delivering.
			// ACK so GOWA stops retrying against a dead endpoint.
			a.Log.Warn("GOWA webhook for unknown instance; ACKing to stop retries",
				"instance_id", instanceID)
			return r.SendEnvelope(map[string]string{"status": "accepted", "note": "unknown instance"})
		}
		a.Log.Error("GOWA webhook: failed to load instance", "error", err, "instance_id", instanceID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "DB error", nil, "")
	}

	// Dispatch on event type. Each handler is best-effort: failures are
	// logged but never bubble up to the HTTP response, because GOWA's retry
	// policy (5 attempts, exponential backoff) is unsuitable for these
	// failures — they will not resolve on retry.
	ctx, cancel := context.WithTimeout(r.RequestCtx, 10*time.Second)
	defer cancel()

	switch {
	case env.Event == "message":
		a.handleGowaMessage(ctx, &instance, env.Payload)
	case env.Event == "receipt":
		a.handleGowaReceipt(ctx, &instance, env.Payload)
	case env.Event == "message.ack":
		// GOWA sends message.ack for delivery/read confirmations. These
		// carry the same shape as receipt events (message_id + type).
		a.handleGowaReceipt(ctx, &instance, env.Payload)
	case env.Event == "message.reaction":
		a.handleGowaReaction(ctx, &instance, env.Payload)
	case env.Event == "message.revoked":
		a.handleGowaRevoked(ctx, &instance, env.Payload)
	case env.Event == "message.edited":
		// Edited messages: update content in place if the row exists.
		a.handleGowaEdited(ctx, &instance, env.Payload)
	case env.Event == "message.deleted":
		// Deletion: soft-delete or mark the message.
		a.handleGowaDeleted(ctx, &instance, env.Payload)
	case strings.HasPrefix(env.Event, "group."), strings.HasPrefix(env.Event, "newsletter."):
		// Forward to WS but don't persist — Stage 6 scope is 1:1 messaging.
		a.broadcastGowaEvent(instance.OrganizationID, "gowa_"+env.Event, env.Payload)
	default:
		a.Log.Debug("GOWA webhook: unhandled event type", "event", env.Event, "instance_id", instanceID)
	}

	return r.SendEnvelope(map[string]string{"status": "accepted"})
}

// ----- HMAC verification -----

// verifyGowaSignature validates the X-Hub-Signature-256 header against the
// shared secret in cfg.Gowa.WebhookSecret. The header format is
// "sha256=<hex>" — identical to Meta and to GOWA's outbound signing.
//
// Constant-time comparison via hmac.Equal prevents timing oracles.
func (a *App) verifyGowaSignature(body, signature []byte) bool {
	secret := a.Config.Gowa.WebhookSecret
	if secret == "" {
		// No secret configured is a hard fail — never accept unsigned events.
		return false
	}
	prefix := []byte("sha256=")
	if !bytes.HasPrefix(signature, prefix) {
		return false
	}
	expected := bytes.TrimPrefix(signature, prefix)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	computed := make([]byte, hex.EncodedLen(mac.Size()))
	hex.Encode(computed, mac.Sum(nil))
	return hmac.Equal(expected, computed)
}

// ----- Event: message -----

// gowaMessagePayload is the per-event payload shape GOWA emits for `event=message`.
// Source: GOWA event_message.go buildEventPayload + buildFromFields. Only fields
// we persist are decoded; the rest stay in the raw JSON and ride along via WS.
//
// Field semantics (verified against GOWA source):
//   - chat_id: the chat JID — for a 1:1 conversation this is the counterparty
//     regardless of direction. For an outgoing echo (IsFromMe=true) the
//     counterparty is the recipient; for an incoming message it's the sender.
//     This is the canonical key for resolving the Contact row in both cases.
//   - from: the sender's JID — for an outgoing echo this is the connected
//     account's OWN jid; for incoming it's the remote party.
type gowaMessagePayload struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	IsFromMe  bool   `json:"is_from_me"`
	From      string `json:"from"`
	FromName  string `json:"from_name,omitempty"`
	FromLID   string `json:"from_lid,omitempty"`
	// ChatID is the chat JID — the counterparty for a 1:1 conversation,
	// regardless of direction. Used as the contact key (replaces the old,
	// wrong "resolve from `from`" logic that put outgoing echoes on a
	// "self" contact).
	ChatID    string `json:"chat_id,omitempty"`
	ChatLID   string `json:"chat_lid,omitempty"`
	Text      string `json:"text,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	URL       string `json:"url,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Caption   string `json:"caption,omitempty"`
}

func (a *App) handleGowaMessage(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p gowaMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		a.Log.Warn("GOWA message payload unparseable", "error", err, "instance_id", instance.ID)
		return
	}
	if p.From == "" && p.ChatID == "" {
		a.Log.Warn("GOWA message missing both 'from' and 'chat_id'",
			"instance_id", instance.ID, "msg_id", p.ID)
		return
	}

	// Resolve the counterparty JID. GOWA's chat_id is the chat JID — for a
	// 1:1 conversation this is the counterparty regardless of direction, so
	// prefer it whenever present. Fall back to "from" only when chat_id is
	// absent (older GOWA builds). Using "from" for an outgoing echo would
	// resolve to the connected account's own contact (a "self" contact),
	// which is exactly the bug that previously hid phone-sent messages.
	counterpartyJID := p.ChatID
	if counterpartyJID == "" {
		counterpartyJID = p.From
		a.Log.Warn("GOWA message missing 'chat_id'; falling back to 'from' for contact resolution",
			"instance_id", instance.ID, "msg_id", p.ID, "from", p.From)
	}
	// Resolve or create the Contact for the counterparty JID. We use the phone
	// number (digits before @) as the canonical phone_number field.
	phone := jidToPhone(counterpartyJID)
	contact, err := a.upsertGowaContact(ctx, instance, phone, p.FromName)
	if err != nil {
		a.Log.Error("GOWA webhook: upsert contact failed", "error", err, "instance_id", instance.ID, "phone", phone)
		return
	}

	// Map GOWA media_type -> whatomate MessageType.
	msgType := gowaMediaTypeToWhatomase(p.MediaType, p.Text)

	// Persist the message. Idempotent on WhatsAppMessageID so GOWA retries
	// (and Stage 7 polling) don't create duplicates.
	msg := models.Message{
		OrganizationID:    instance.OrganizationID,
		InstanceID:        &instance.ID,
		WhatsAppAccount:   instance.Name,
		ContactID:         contact.ID,
		WhatsAppMessageID: p.ID,
		Direction:         directionFromGowa(p.IsFromMe),
		MessageType:       msgType,
		Content:           firstNonEmpty(p.Text, p.Caption),
		MediaURL:          p.URL,
		MediaMimeType:     p.MimeType,
		MediaFilename:     p.Filename,
		Status:            messageStatusForDirection(p.IsFromMe),
		// Store the authoritative chat JID GOWA persists the message under
		// (p.ChatID, with the documented "from" fallback). Without this, the
		// row has an empty conversation_id and downstream JID resolution in
		// ServeMedia/RetryMediaDownload collapses to contact.phone_number,
		// which can disagree with GOWA's stored chat_jid (status broadcasts,
		// groups, LID-routed chats, repaired phones) and GOWA then rejects
		// the download with "<wamid> does not belong to chat <jid>".
		ConversationID: counterpartyJID,
	}
	// Bug fix: use the GOWA message timestamp rather than GORM's auto-insert
	// time, so messages order correctly even when delivered/batched late.
	// GORM only honors CreatedAt on the insert path, so this won't overwrite
	// an existing row's timestamp (we also guard explicitly in the patch path
	// below).
	if p.Timestamp != "" {
		if ts, tsErr := time.Parse(time.RFC3339, p.Timestamp); tsErr == nil {
			msg.CreatedAt = ts
		} else {
			a.Log.Debug("GOWA message timestamp unparseable; falling back to now",
				"error", tsErr, "instance_id", instance.ID, "msg_id", p.ID, "timestamp", p.Timestamp)
			msg.CreatedAt = time.Now()
		}
	} else {
		msg.CreatedAt = time.Now()
	}
	// FirstOrCreate on whats_app_message_id gives idempotent inserts. But a
	// row created during a stale/backfill sweep may have empty content or
	// type=ignore because GOWA hadn't fully synced the message yet. When a
	// later delivery (poll or webhook) brings richer data, patch the row so
	// the UI doesn't render an ignore/empty message forever.
	newContent := firstNonEmpty(p.Text, p.Caption)
	result := a.DB.WithContext(ctx).Where("whats_app_message_id = ?", p.ID).
		FirstOrCreate(&msg)
	if result.Error != nil {
		a.Log.Error("GOWA webhook: persist message failed", "error", result.Error, "instance_id", instance.ID, "msg_id", p.ID)
		return
	}
	if result.RowsAffected == 0 {
		// Row already existed. If the new payload carries content or a more
		// specific type than what we stored, refresh those fields in place.
		// We never overwrite non-empty existing content (the first write
		// wins) — only fill empties and promote ignore→concrete type.
		patch := map[string]any{}
		if msg.Content == "" && newContent != "" {
			patch["content"] = newContent
		}
		if msg.MessageType == models.MessageTypeIgnore && msgType != models.MessageTypeIgnore {
			patch["message_type"] = msgType
		}
		if msg.MediaURL == "" && p.URL != "" {
			patch["media_url"] = p.URL
			patch["media_mime_type"] = p.MimeType
			patch["media_filename"] = p.Filename
		}
		// Backfill conversation_id for rows created before this field was
		// persisted on the GOWA path. First-write-wins, mirroring the fields
		// above — never overwrite a non-empty conversation_id.
		if msg.ConversationID == "" && counterpartyJID != "" {
			patch["conversation_id"] = counterpartyJID
		}
		// First-write-wins for ordering stability: never overwrite an existing
		// non-zero created_at, but if the original row was inserted without a
		// timestamp, fill it from the GOWA payload now.
		if msg.CreatedAt.IsZero() && p.Timestamp != "" {
			if ts, tsErr := time.Parse(time.RFC3339, p.Timestamp); tsErr == nil {
				patch["created_at"] = ts
			}
		}
		if len(patch) > 0 {
			if err := a.DB.Model(&msg).Updates(patch).Error; err != nil {
				a.Log.Warn("GOWA webhook: patch existing message failed",
					"error", err, "instance_id", instance.ID, "msg_id", p.ID)
			}
			// Reflect the patch on the in-memory msg so the WS broadcast the
			// frontend receives carries the freshly-filled fields.
			for k, v := range patch {
				switch k {
				case "content":
					msg.Content = v.(string)
				case "message_type":
					msg.MessageType = v.(models.MessageType)
				case "media_url":
					msg.MediaURL = v.(string)
				case "media_mime_type":
					msg.MediaMimeType = v.(string)
				case "media_filename":
					msg.MediaFilename = v.(string)
				case "conversation_id":
					msg.ConversationID = v.(string)
				case "created_at":
					msg.CreatedAt = v.(time.Time)
				}
			}
			// Broadcast the patched message so the UI refreshes in real time
			// instead of holding the stale [Message] / ignore placeholder.
			// Only broadcast when a UI-rendered field actually changed —
			// idempotent re-deliveries (no diff) stay silent to avoid WS spam.
			a.Log.Info("GOWA: broadcasting patched inbound message",
				"message_id", msg.ID, "contact_id", contact.ID,
				"contact_phone", contact.PhoneNumber, "patched_fields", len(patch))
			a.broadcastNewMessage(instance.OrganizationID, &msg, contact)
		}
		// Already existed — skip contact touch (avoid last-message churn on
		// re-delivery) but the WS broadcast above already refreshed the UI.
		return
	}

	// Update contact's last-message metadata so chat lists stay sorted.
	now := time.Now()
	a.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      &now,
		"last_message_preview": truncateForPreview(firstNonEmpty(p.Text, p.Caption, string(msgType))),
		"last_inbound_at":      &now,
	})

	// If this is a content-less marker (GOWA's first push, before the real
	// text arrives), the second push can take 7-30s to arrive — far too
	// long to show "Loading message…" in the UI. So spawn a background
	// fetch that polls GOWA directly for this message's content and
	// patches+broadcasts as soon as it lands (~1-3s in practice). The
	// second push (when it arrives) becomes a no-op because the patch is
	// idempotent on whats_app_message_id and only fills empties.
	if msg.MessageType == models.MessageTypeIgnore && msg.Content == "" && msg.MediaURL == "" {
		a.fetchGowaMessageContentAsync(instance, contact, &msg)
	}

	// Broadcast via the same broadcastNewMessage the outgoing path uses, so
	// the frontend receives the exact same WS payload shape it already knows
	// how to render (id, contact_id, direction, content, message_type, etc.).
	a.Log.Info("GOWA: broadcasting new inbound message",
		"message_id", msg.ID, "contact_id", contact.ID,
		"contact_phone", contact.PhoneNumber, "direction", msg.Direction,
		"content_preview", truncateForPreview(msg.Content))
	a.broadcastNewMessage(instance.OrganizationID, &msg, contact)
}

// fetchGowaMessageContentAsync polls GOWA for the real content of a content-less
// marker message and patches the row + broadcasts as soon as it lands. Used to
// beat GOWA's unreliable second-push latency (which can be 7-30s or get dropped).
//
// Best-effort: failures are logged and swallowed. The webhook's own second push
// and the polling reconciler remain as backstops.
func (a *App) fetchGowaMessageContentAsync(instance *models.WhatsAppInstance, contact *models.Contact, msg *models.Message) {
	if a.GowaClient == nil {
		return
	}
	deviceID := gowaDeviceID(instance)
	if deviceID == "" || msg.WhatsAppMessageID == "" {
		return
	}
	// The chat JID for the counterparty. We stored the phone on the contact;
	// reconstruct the individual JID GOWA expects.
	chatJID := strings.TrimSpace(contact.PhoneNumber) + "@s.whatsapp.net"
	wamid := msg.WhatsAppMessageID
	msgID := msg.ID
	orgID := instance.OrganizationID

	go func() {
		// Tight poll: try every 1s for up to ~10s. GOWA usually surfaces the
		// content within 1-3s of the marker; we want to broadcast it as soon
		// as it's available. GOWA's `search` filters on message text content,
		// not message ID, so we fetch the most recent messages for this chat
		// and match by wamid ourselves.
		for attempt := 1; attempt <= 15; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := a.GowaClient.GetChatMessages(ctx, deviceID, chatJID, gowa.GetMessagesFilter{
				Limit: 50,
			})
			cancel()
			if err != nil {
				a.Log.Debug("GOWA content-fetch: query failed",
					"attempt", attempt, "wamid", wamid, "error", err)
				time.Sleep(time.Second)
				continue
			}
			// Find the message with this wamid in the recent history.
			for _, m := range resp.Data {
				if m.ID != wamid {
					continue
				}
				text := strings.TrimSpace(m.Content)
				if text == "" && m.URL == "" {
					// GOWA knows the message but content still empty — try again.
					break
				}
				// Got it. Patch + broadcast synchronously.
				a.applyGowaContentPatch(instance, contact, msgID, wamid, text, m.MediaType, m.URL, m.Filename)
				return
			}
			time.Sleep(time.Second)
		}
		a.Log.Warn("GOWA content-fetch: gave up after retries",
			"wamid", wamid, "msg_id", msgID,
			"note", "second webhook push and poller remain as backstops")
		_ = orgID
	}()
}

// applyGowaContentPatch applies the fetched content/type/media to the row and
// broadcasts the patched message via WS. Idempotent: skips when the row already
// carries real content (the second webhook push may have landed first).
func (a *App) applyGowaContentPatch(instance *models.WhatsAppInstance, contact *models.Contact, msgID uuid.UUID, wamid, text, mediaType, mediaURL, mediaFilename string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msg models.Message
	if err := a.DB.WithContext(ctx).First(&msg, msgID).Error; err != nil {
		a.Log.Warn("GOWA content-fetch: row vanished before patch",
			"msg_id", msgID, "wamid", wamid, "error", err)
		return
	}
	// Idempotency: skip if the row already has real content (e.g. the second
	// webhook push beat us).
	if msg.Content != "" || msg.MediaURL != "" {
		return
	}
	resolvedType := gowaMediaTypeToWhatomase(mediaType, text)
	patch := map[string]any{
		"content":      text,
		"message_type": resolvedType,
	}
	if mediaURL != "" {
		patch["media_url"] = mediaURL
		patch["media_filename"] = mediaFilename
	}
	if err := a.DB.Model(&msg).Updates(patch).Error; err != nil {
		a.Log.Warn("GOWA content-fetch: patch failed",
			"msg_id", msgID, "wamid", wamid, "error", err)
		return
	}
	msg.Content = text
	msg.MessageType = resolvedType
	if mediaURL != "" {
		msg.MediaURL = mediaURL
		msg.MediaFilename = mediaFilename
	}
	a.Log.Info("GOWA content-fetch: patched from GOWA directly",
		"message_id", msg.ID, "contact_id", contact.ID, "wamid", wamid,
		"content_preview", truncateForPreview(text))
	a.broadcastNewMessage(instance.OrganizationID, &msg, contact)
}

// ----- Event: receipt -----

// gowaReceiptPayload covers delivery/read receipts.
type gowaReceiptPayload struct {
	MessageID string `json:"message_id"`
	From      string `json:"from"`
	Type      string `json:"type"` // delivered | read
	Timestamp string `json:"timestamp"`
}

func (a *App) handleGowaReceipt(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p gowaReceiptPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return
	}
	if p.MessageID == "" {
		return
	}
	status := models.MessageStatusDelivered
	if p.Type == "read" {
		status = models.MessageStatusRead
	}
	res := a.DB.WithContext(ctx).Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", p.MessageID, instance.OrganizationID).
		Update("status", status)
	if res.Error != nil {
		a.Log.Warn("GOWA receipt: status update failed", "error", res.Error, "msg_id", p.MessageID)
		return
	}
	a.broadcastGowaEvent(instance.OrganizationID, websocket.TypeStatusUpdate, map[string]any{
		"message_id":  p.MessageID,
		"status":      string(status),
		"instance_id": instance.ID,
		"source":      "gowa",
	})
}

// ----- Event: message.reaction -----

// gowaReactionPayload covers reactions added/removed.
type gowaReactionPayload struct {
	ID               string `json:"id"`
	ChatID           string `json:"chat_id"`
	From             string `json:"from"`
	FromName         string `json:"from_name"`
	Timestamp        string `json:"timestamp"`
	IsFromMe         bool   `json:"is_from_me"`
	Reaction         string `json:"reaction"`
	ReactedMessageID string `json:"reacted_message_id"`
}

func (a *App) handleGowaReaction(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p gowaReactionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return
	}
	if p.ReactedMessageID == "" {
		return
	}

	// Find the original message
	var msg models.Message
	if err := a.DB.WithContext(ctx).Where("whats_app_message_id = ? AND organization_id = ?", p.ReactedMessageID, instance.OrganizationID).First(&msg).Error; err != nil {
		a.Log.Warn("GOWA reaction: original message not found", "whats_app_message_id", p.ReactedMessageID, "org_id", instance.OrganizationID)
		return
	}

	// Update the metadata
	var metadata map[string]interface{}
	if msg.Metadata != nil {
		metadata = msg.Metadata
	} else {
		metadata = make(map[string]interface{})
	}

	// Retrieve reactions list
	var reactions []map[string]any
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]interface{}); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]interface{}); ok {
					reactions = append(reactions, rMap)
				}
			}
		}
	}

	// Clean phone number format for from JID/phone
	fromPhone := jidToPhone(p.From)

	// Remove existing reaction from this phone
	newReactions := make([]map[string]any, 0)
	for _, r := range reactions {
		if rPhone, _ := r["from_phone"].(string); rPhone != fromPhone {
			newReactions = append(newReactions, r)
		}
	}

	// Add new reaction if emoji is not empty
	if p.Reaction != "" {
		newReactions = append(newReactions, map[string]any{
			"emoji":      p.Reaction,
			"from_phone": fromPhone,
		})
	}

	metadata["reactions"] = newReactions
	if err := a.DB.WithContext(ctx).Model(&msg).Update("metadata", metadata).Error; err != nil {
		a.Log.Error("GOWA reaction: failed to update message metadata", "error", err)
		return
	}

	a.broadcastGowaEvent(instance.OrganizationID, websocket.TypeReactionUpdate, map[string]any{
		"message_id": msg.ID.String(),
		"contact_id": msg.ContactID.String(),
		"reactions":  newReactions,
	})
}

// ----- Event: message.revoked -----

func (a *App) handleGowaRevoked(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.MessageID == "" {
		return
	}
	a.DB.WithContext(ctx).Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", p.MessageID, instance.OrganizationID).
		Update("status", "revoked")
	a.broadcastGowaEvent(instance.OrganizationID, websocket.TypeStatusUpdate, map[string]any{
		"message_id":  p.MessageID,
		"status":      "revoked",
		"instance_id": instance.ID,
		"source":      "gowa",
	})
}

// ----- Event: message.edited -----

func (a *App) handleGowaEdited(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p struct {
		MessageID string `json:"message_id"`
		Text      string `json:"text"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return
	}
	if p.MessageID == "" {
		return
	}
	newContent := firstNonEmpty(p.Text, p.Content)
	if newContent == "" {
		return
	}
	a.DB.WithContext(ctx).Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", p.MessageID, instance.OrganizationID).
		Updates(map[string]any{"content": newContent, "updated_at": time.Now()})
}

// ----- Event: message.deleted -----

func (a *App) handleGowaDeleted(ctx context.Context, instance *models.WhatsAppInstance, raw json.RawMessage) {
	var p struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.MessageID == "" {
		return
	}
	a.DB.WithContext(ctx).Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", p.MessageID, instance.OrganizationID).
		Update("status", "revoked")
	a.broadcastGowaEvent(instance.OrganizationID, websocket.TypeStatusUpdate, map[string]any{
		"message_id":  p.MessageID,
		"status":      "deleted",
		"instance_id": instance.ID,
		"source":      "gowa",
	})
}

// ----- Helpers -----

// upsertGowaContact resolves an existing Contact by (org, instance, phone) or
// creates one. GOWA doesn't surface contact lists with stable IDs, so the
// phone number is the canonical key.
//
// IMPORTANT: for group chats (phone is a group ID, detected via
// isLikelyGroupID), the displayName passed in is the SENDER's push name
// (the person who sent the message in the group), NOT the group name.
// Overwriting contact.ProfileName with that sender name is what made the
// group chat header flip to "sabrin abdaluity Mohamed" instead of showing
// the group name. For groups we:
//   - never overwrite an existing profile_name with a sender push name
//   - on creation, resolve the real group name from GOWA's /chats
//     (best-effort; falls back to a stable placeholder if unreachable)
func (a *App) upsertGowaContact(ctx context.Context, instance *models.WhatsAppInstance, phone, displayName string) (*models.Contact, error) {
	if phone == "" {
		return nil, errors.New("empty phone")
	}
	isGroup := isLikelyGroupID(phone) || phone == "status" ||
		strings.HasSuffix(phone, "@g.us") || strings.HasSuffix(phone, "@newsletter")

	var contact models.Contact
	err := a.DB.WithContext(ctx).
		Where("organization_id = ? AND instance_id = ? AND phone_number = ?",
			instance.OrganizationID, instance.ID, phone).
		First(&contact).Error
	if err == nil {
		// Refresh display name — but never with the volatile WhatsApp push
		// name (p.FromName) for an existing contact. The push name is the
		// sender's self-set WhatsApp profile name, which they can change at
		// any time and which frequently disagrees with the chat name GOWA
		// stored (e.g. GOWA has "وجآته البشري" but the push name is
		// "مكتبة الأركان المثالية" — the account's business display name).
		// Overwriting on every incoming message caused the chat header to
		// flip to whatever push name WhatsApp reported that day.
		//
		// For 1:1 contacts we only set the name from the push name when the
		// row has NO name yet (first contact). For groups we never touch it
		// here — group names are resolved once at creation.
		if !isGroup && displayName != "" && strings.TrimSpace(contact.ProfileName) == "" {
			a.DB.Model(&contact).Update("profile_name", displayName)
			contact.ProfileName = displayName
		}
		return &contact, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Creating a new contact. For groups, resolve the real group name
	// rather than using the sender's push name.
	resolvedName := displayName
	if isGroup {
		resolvedName = a.resolveGowaGroupName(ctx, instance, phone)
	}
	contact = models.Contact{
		OrganizationID:  instance.OrganizationID,
		InstanceID:      &instance.ID,
		PhoneNumber:     phone,
		ProfileName:     resolvedName,
		WhatsAppAccount: instance.Name,
		Status:          models.ChatStatusPending,
	}
	if err := a.DB.WithContext(ctx).Create(&contact).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

// resolveGowaGroupName fetches the human-readable name of a group chat from
// GOWA's /chats endpoint. Best-effort: on any error or empty result, falls
// back to a stable placeholder ("Group <id>") so the UI never shows a
// sender's push name as the group identity.
func (a *App) resolveGowaGroupName(ctx context.Context, instance *models.WhatsAppInstance, phone string) string {
	// Build the full JID GOWA expects (handles both group ID shapes).
	chatJID := resolveGowaChatJID("", phone)
	if chatJID == "" {
		chatJID = phone
	}
	if a.GowaClient == nil {
		return "Group " + phone
	}
	deviceID := gowaDeviceID(instance)
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// GOWA's /chats supports a Search filter — use the JID to narrow results.
	resp, err := a.GowaClient.ListChats(listCtx, deviceID, gowa.ListChatsFilter{
		Search: chatJID,
		Limit:  5,
	})
	if err != nil {
		a.Log.Debug("GOWA group name lookup failed; using placeholder",
			"error", err, "phone", phone, "chat_jid", chatJID)
		return "Group " + phone
	}
	for _, chat := range resp.Data {
		if chat.JID == chatJID && strings.TrimSpace(chat.Name) != "" {
			return chat.Name
		}
	}
	return "Group " + phone
}

// broadcastGowaEvent pushes an event to all WS clients in the org. Failures
// are best-effort (hub handles its own backpressure).
func (a *App) broadcastGowaEvent(orgID uuid.UUID, msgType string, payload any) {
	if a.WSHub == nil {
		return
	}
	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    msgType,
		Payload: payload,
	})
}

// jidToPhone extracts the phone digits from a JID like
// "12025550100@s.whatsapp.net" → "12025550100". For group/newsletter JIDs
// it returns the raw identifier (groups have no phone).
func jidToPhone(jid string) string {
	if at := strings.IndexByte(jid, '@'); at > 0 {
		return jid[:at]
	}
	return jid
}

// gowaMediaTypeToWhatomate maps GOWA's media_type string (image, video,
// audio, document, sticker) to whatomate's MessageType. Text with no media
// returns text.
func gowaMediaTypeToWhatomase(mediaType, text string) models.MessageType {
	switch strings.ToLower(mediaType) {
	case "image":
		return models.MessageTypeImage
	case "video":
		return models.MessageTypeVideo
	case "audio", "ptv":
		return models.MessageTypeAudio
	case "document":
		return models.MessageTypeDocument
	case "sticker":
		return models.MessageTypeSticker
	default:
		if text == "" {
			return models.MessageTypeIgnore
		}
		return models.MessageTypeText
	}
}

// directionFromGowa flips IsFromMe (GOWA's flag) into whatomate's Direction.
// GOWA's IsFromMe=true means the message was sent FROM the connected account;
// false means it was received.
func directionFromGowa(isFromMe bool) models.Direction {
	if isFromMe {
		return models.DirectionOutgoing
	}
	return models.DirectionIncoming
}

// messageStatusForDirection: incoming messages are already "received";
// outgoing ones (echoed back by GOWA) start at "sent".
func messageStatusForDirection(isFromMe bool) models.MessageStatus {
	if isFromMe {
		return models.MessageStatusSent
	}
	return models.MessageStatusReceived
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncateForPreview(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
