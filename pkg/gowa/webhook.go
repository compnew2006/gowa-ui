package gowa

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// WebhookPayload is the top-level envelope for all GOWA webhook events.
// The timestamp field is at the top level for ack, presence, group, call,
// label, and newsletter events — but NOT for message events (where it
// lives inside payload). We define both locations and pick the right one
// in the handler.
type WebhookPayload struct {
	Event     string          `json:"event"`
	DeviceID  string          `json:"device_id"`
	SessionID string          `json:"session_id,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"` // top-level for non-message events
	Payload   json.RawMessage `json:"payload"`
}

// MessagePayload is decoded from WebhookPayload.Payload when event == "message".
type MessagePayload struct {
	ID          string `json:"id"`
	ChatID      string `json:"chat_id"`
	From        string `json:"from"`
	FromLID     string `json:"from_lid,omitempty"`
	FromName    string `json:"from_name,omitempty"` // push name / display name
	Timestamp   string `json:"timestamp"`           // RFC3339, INSIDE payload for message events
	IsFromMe    bool   `json:"is_from_me"`
	Body        string `json:"body,omitempty"` // text body, or caption for media
	RepliedToID string `json:"replied_to_id,omitempty"`
	QuotedBody  string `json:"quoted_body,omitempty"`

	// Polymorphic media fields. GOWA encodes these as either a plain string
	// (auto-download path or URL) or an object ({path/url, caption/filename}).
	// We decode them as json.RawMessage and resolve with resolveMediaField().
	Image     json.RawMessage `json:"image,omitempty"`
	Video     json.RawMessage `json:"video,omitempty"`
	Audio     json.RawMessage `json:"audio,omitempty"`
	Document  json.RawMessage `json:"document,omitempty"`
	Sticker   json.RawMessage `json:"sticker,omitempty"`
	VideoNote json.RawMessage `json:"video_note,omitempty"`

	ViewOnce  bool `json:"view_once,omitempty"`
	Forwarded bool `json:"forwarded,omitempty"`
}

// AckPayload is decoded from WebhookPayload.Payload when event == "message.ack".
type AckPayload struct {
	IDs         []string `json:"ids"`
	ChatID      string   `json:"chat_id"`
	From        string   `json:"from"`
	FromLID     string   `json:"from_lid,omitempty"`
	ReceiptType string   `json:"receipt_type"` // "delivered" or "read"
}

// ChatPresencePayload is decoded when event == "chat_presence".
type ChatPresencePayload struct {
	From    string `json:"from"`
	ChatID  string `json:"chat_id"`
	State   string `json:"state"` // "composing" or "paused"
	Media   string `json:"media"` // "" = text, "audio" = voice recording
	IsGroup bool   `json:"is_group"`
}

// ReactionPayload is decoded when event == "message.reaction".
type ReactionPayload struct {
	Reaction         string `json:"reaction"`           // emoji
	ReactedMessageID string `json:"reacted_message_id"` // ID of the message being reacted to
	From             string `json:"from,omitempty"`
	ChatID           string `json:"chat_id,omitempty"`
}

// RevokedPayload is decoded when event == "message.revoked".
type RevokedPayload struct {
	RevokedMessageID string `json:"revoked_message_id"`
	From             string `json:"from,omitempty"`
	ChatID           string `json:"chat_id,omitempty"`
}

// EditedPayload is decoded when event == "message.edited".
type EditedPayload struct {
	OriginalMessageID string `json:"original_message_id"`
	Body              string `json:"body"`
	From              string `json:"from,omitempty"`
	ChatID            string `json:"chat_id,omitempty"`
}

// ConnectionPayload is decoded when event == "connection".
// GOWA sends connection events when a device connects, disconnects,
// or its connection state changes.
type ConnectionPayload struct {
	Event       string `json:"event"` // "connected", "disconnected", "connecting", "logout"
	Reason      string `json:"reason,omitempty"`
	IsConnected bool   `json:"is_connected"`
}

// ConnectionState constants for the payload.Event field.
const (
	ConnectionConnected    = "connected"
	ConnectionDisconnected = "disconnected"
	ConnectionConnecting   = "connecting"
	ConnectionLogout       = "logout"
)

// MediaField is the resolved form of a polymorphic media value.
type MediaField struct {
	URL      string // download URL or server path
	Caption  string
	Filename string
}

// ResolveMediaField decodes a polymorphic media json.RawMessage into a
// MediaField. The field can be:
//   - a plain string (auto-download path or URL)
//   - an object with "path" and optional "caption" (auto-download ON)
//   - an object with "url" and optional "caption" or "filename" (auto-download OFF)
func ResolveMediaField(raw json.RawMessage) MediaField {
	if len(raw) == 0 {
		return MediaField{}
	}

	// Try string form first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return MediaField{URL: s}
	}

	// Try object form.
	var obj struct {
		Path     string `json:"path"`
		URL      string `json:"url"`
		Caption  string `json:"caption"`
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		mf := MediaField{
			Caption:  obj.Caption,
			Filename: obj.Filename,
		}
		if obj.URL != "" {
			mf.URL = obj.URL
		} else {
			mf.URL = obj.Path
		}
		return mf
	}

	return MediaField{}
}

// IsGroup reports whether the chat is a group (chat_id ends with @g.us).
func (m *MessagePayload) IsGroup() bool {
	return strings.HasSuffix(m.ChatID, "@g.us")
}

// IsNewsletter reports whether the chat is a newsletter/channel (chat_id ends
// with @newsletter). WhatsApp newsletters are broadcast channels — they share
// the "120363..." ID prefix with groups but use a distinct JID suffix. We treat
// them like groups for routing/badge purposes (one contact per channel, content
// visible without a 1:1 relationship) so the UI shows the group badge.
func (m *MessagePayload) IsNewsletter() bool {
	return strings.HasSuffix(m.ChatID, "@newsletter")
}

// PhoneFromJID extracts the plain phone number digits from a JID.
// "16505551234@s.whatsapp.net" → "16505551234"
func PhoneFromJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx > 0 {
		return jid[:idx]
	}
	return jid
}

// ParseTimestamp parses the RFC3339 timestamp, returning zero time on error.
func ParseTimestamp(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

// CheckReplay validates that a webhook timestamp is within the allowed
// freshness window. It returns true if the timestamp is fresh (within ±maxAge
// of the current time), and false if it is stale, missing (zero), or
// unparseable. The window is symmetric to tolerate clock drift between the
// GOWA server and whatomate in either direction.
//
// timestampStr is the raw string from the webhook payload (RFC3339 or Unix
// epoch seconds as a string). If parsing fails, the webhook is rejected
// (fail-closed).
func CheckReplay(timestampStr string, maxAge time.Duration) bool {
	if timestampStr == "" {
		return false // missing timestamp = reject
	}

	// Try RFC3339 first (message events use this format inside payload).
	t, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		// Try Unix epoch seconds (top-level envelope timestamp).
		var sec int64
		if _, err := fmt.Sscanf(timestampStr, "%d", &sec); err != nil {
			return false // unparseable = reject
		}
		t = time.Unix(sec, 0)
	}

	age := time.Since(t)
	return age <= maxAge && age >= -maxAge
}
