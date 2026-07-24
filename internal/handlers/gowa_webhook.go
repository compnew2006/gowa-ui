package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/contactutil"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
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

	// Per-device routing: the path device_id overrides the payload device_id.
	if pathDeviceID != "" {
		envelope.DeviceID = pathDeviceID
	}

	if envelope.DeviceID == "" {
		a.Log.Warn("GOWA webhook missing device_id")
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing device_id", nil, "")
	}

	// Resolve the WhatsApp account for this GOWA device.
	account, err := a.getGowaAccountByDeviceID(envelope.DeviceID)
	if err != nil {
		a.Log.Warn("Unknown GOWA device", "device_id", envelope.DeviceID, "error", err)
		// Return the same generic rejection as a signature failure so an
		// attacker cannot distinguish an unconfigured device from a bad
		// signature (FR-023: indistinguishable responses).
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Webhook verification failed", nil, "")
	}

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
	// For non-message events, the timestamp is in envelope.Timestamp (top-level).
	// For message events, GOWA puts it inside the payload object, so we extract
	// it from there before the replay check.
	replayTS := envelope.Timestamp
	if replayTS == "" && envelope.Event == "message" {
		var msgPeek struct {
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(envelope.Payload, &msgPeek); err == nil {
			replayTS = msgPeek.Timestamp
		}
	}
	if !gowa.CheckReplay(replayTS, 5*time.Minute) {
		a.Log.Warn("Stale GOWA webhook rejected (replay)", "device_id", envelope.DeviceID, "timestamp", replayTS)
		return r.SendEnvelope(map[string]string{"status": "ok"}) // 200 to prevent GOWA retries
	}

	// Route based on event type. Processing is async so we respond 200
	// within GOWA's 10-second timeout.
	switch envelope.Event {
	case "message":
		go a.processGowaMessage(account, &envelope)
	case "message.ack":
		go a.processGowaAck(&envelope)
	case "chat_presence":
		go a.processGowaChatPresence(&envelope)
	case "connection":
		go a.processGowaConnection(account, &envelope)
	case "message.reaction":
		go a.processGowaReaction(account, &envelope)
	case "message.revoked":
		go a.processGowaRevoked(account, &envelope)
	case "message.edited":
		go a.processGowaEdited(account, &envelope)
	default:
		a.Log.Debug("Unhandled GOWA event type", "event", envelope.Event)
	}

	return r.SendEnvelope(map[string]string{"status": "ok"})
}

// getGowaAccountByDeviceID looks up a WhatsAppAccount by its GOWA device ID.
// The device_id in the webhook is the GOWA session identifier (typically the
// account's phone JID, e.g. "628123456789@s.whatsapp.net").
func (a *App) getGowaAccountByDeviceID(deviceID string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount
	// GOWA v8 webhooks send the connected JID (e.g. "201007181781@s.whatsapp.net")
	// as device_id, but whatomate stores the custom device ID assigned during
	// device creation (e.g. "test-account-d9768a03"). We try multiple match
	// strategies: exact device_id, phone portion of JID, phone_id field.
	//
	// NOTE: the previous implementation had a fallback that iterated ALL GOWA
	// accounts across every org and made outbound GetAppStatus calls on every
	// unauthenticated request. That fallback was an abuse vector (M5) and has
	// been removed. The account must resolve via the direct query below.
	phone := gowa.PhoneFromJID(deviceID)
	if err := a.DB.Where(
		"provider_type = ? AND (gowa_device_id = ? OR gowa_device_id = ? OR phone_id = ? OR phone_id = ?)",
		"gowa", deviceID, phone, deviceID, phone,
	).First(&account).Error; err != nil {
		return nil, fmt.Errorf("gowa account not found for device %s: %w", deviceID, err)
	}

	// Cache the JID as phone_id for faster future lookups.
	if phone != "" && phone != deviceID && account.PhoneID != deviceID {
		a.DB.Model(&account).Update("phone_id", deviceID)
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

	// Build the whatomate IncomingTextMessage from the GOWA payload.
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
	a.processIncomingMessage(envelope.DeviceID, incoming, profileName, isGroup, isNewsletter, senderName, senderJID)
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
	var metaKey string
	if msg.IsGroup() {
		metaKey = "is_group_chat"
	} else if msg.IsNewsletter() {
		metaKey = "is_newsletter"
	}
	if metaKey != "" {
		if contact.Metadata == nil {
			contact.Metadata = models.JSONB{}
		}
		// Groups and newsletters are mutually exclusive. Setting one clears the
		// other so legacy contacts that carry both flags self-heal on the next
		// incoming message.
		otherKey := ""
		if metaKey == "is_group_chat" {
			otherKey = "is_newsletter"
		} else if metaKey == "is_newsletter" {
			otherKey = "is_group_chat"
		}
		_, hasOther := contact.Metadata[otherKey]
		if contact.Metadata[metaKey] != true || hasOther {
			contact.Metadata[metaKey] = true
			delete(contact.Metadata, otherKey)
			a.DB.Model(contact).Update("metadata", contact.Metadata)
		}
	}

	// Determine message type, content, and media from the GOWA payload.
	// Media messages carry the file URL in the polymorphic fields; we download
	// it via the GOWA client and store locally (same as incoming media).

	// Dedup: if this message was already recorded (e.g. sent from the whatomate
	// UI, which created a local row with the GOWA-returned wamid), update its
	// reply context in place and skip creating a duplicate. The GOWA echo and
	// the local row share the same WhatsAppMessageID.
	if msg.ID != "" {
		var existing models.Message
		if err := a.DB.Where("whats_app_message_id = ?", msg.ID).First(&existing).Error; err == nil {
			// Patch the reply context onto the existing row if the echo carries one
			// and the local row doesn't already have it.
			if msg.RepliedToID != "" && !existing.IsReply {
				var replyToMsg models.Message
				if err := a.DB.Where("whats_app_message_id = ?", msg.RepliedToID).First(&replyToMsg).Error; err == nil {
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
		if err := a.DB.Where("whats_app_message_id = ?", msg.RepliedToID).First(&replyToMsg).Error; err == nil {
			outgoing.IsReply = true
			outgoing.ReplyToMessageID = &replyToMsg.ID
		} else {
			a.Log.Debug("Outgoing reply-to message not found locally", "reply_to_wamid", msg.RepliedToID)
		}
	}

	if err := a.DB.Create(outgoing).Error; err != nil {
		a.Log.Error("Failed to save outgoing GOWA message", "error", err)
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
func (a *App) processGowaAck(envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaAck", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	var ack gowa.AckPayload
	if err := json.Unmarshal(envelope.Payload, &ack); err != nil {
		a.Log.Error("Failed to parse GOWA ack payload", "error", err)
		return
	}

	// Map GOWA receipt types to whatomate status values.
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

	// Process each message ID in the batch.
	webhookStatus := WebhookStatus{
		Status:    status,
		Timestamp: envelope.Timestamp, // top-level for ack events
	}
	for _, msgID := range ack.IDs {
		webhookStatus.ID = msgID
		a.processStatusUpdate(envelope.DeviceID, webhookStatus)
	}
}

// processGowaChatPresence handles typing/recording presence events.
// Currently this only logs; it can be extended to broadcast typing
// indicators via the WebSocket hub.
func (a *App) processGowaChatPresence(envelope *gowa.WebhookPayload) {
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

	// TODO: broadcast typing indicator via WebSocket hub for real-time UI.
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
	var msg models.Message
	if err := a.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", revoked.RevokedMessageID, account.OrganizationID).
		Updates(map[string]any{
			"status":  models.MessageStatusRevoked,
			"content": "[message revoked]",
		}).Error; err != nil {
		a.Log.Error("Failed to mark GOWA message as revoked",
			"revoked_message_id", revoked.RevokedMessageID, "error", err)
		return
	}

	// Broadcast the revoked status over WebSocket so every open client
	// updates the bubble in real time. The existing status_update handler
	// on the frontend keys off message_id, so we resolve the row first.
	if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ?",
		revoked.RevokedMessageID, account.OrganizationID).
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

	// Update the message content in the database.
	if err := a.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", edited.OriginalMessageID, account.OrganizationID).
		Update("content", edited.Body).Error; err != nil {
		a.Log.Error("Failed to update GOWA edited message content",
			"original_message_id", edited.OriginalMessageID, "error", err)
	}
}
