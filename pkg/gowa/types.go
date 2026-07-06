// Package gowa implements a MessageProvider backed by an external GOWA HTTP
// server (github.com/aldinokemal/go-whatsapp-web-multidevice).
//
// When the operator sets whatsapp.provider="gowa", whatomate stops driving Meta
// Cloud API or whatsmeow directly and instead proxies every messaging and
// device-lifecycle operation to a remote GOWA REST API. GOWA owns the
// whatsmeow sessions; whatomate owns the multi-tenant org/user/RBAC model and
// the WebSocket UI.
//
// The package is organised as:
//
//   - types.go        DTOs mirroring GOWA's request/response envelopes
//   - client.go       Thin HTTP client (no business logic, owns *http.Client)
//   - adapter.go      MessageProvider implementation using the client
//   - errors.go       Typed errors with Retryable() classification
//
// The adapter is wired in cmd/whatomate/main.go alongside the meta and
// whatsmeow cases. It implements the full base provider.MessageProvider
// interface and compile-time asserts this with a top-level var.
package gowa

import (
	"encoding/json"
	"time"
)

// ----- Envelope ---------------------------------------------------------------
//
// Envelope wraps every GOWA REST response. GOWA always returns:
//
//	{ "status": 200, "code": "SUCCESS", "message": "...", "results": <T> }
//
// Failure responses use the same shape with a non-2xx status and a non-empty
// message; the client unwraps results on 2xx and converts non-2xx to *Error.
type Envelope[T any] struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Results T      `json:"results"`
}

// ----- Device lifecycle -------------------------------------------------------

// Device mirrors GOWA's GET /devices and POST /devices response shape.
// Source: GOWA ui/rest/device.go Device struct + Results fields.
type Device struct {
	ID          string    `json:"id"`
	PhoneNumber string    `json:"phone_number,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	State       string    `json:"state"` // disconnected|connecting|connected|logged_in
	JID         string    `json:"jid,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateDeviceRequest is the POST /devices body. The webhook_* fields are
// optional but critical: webhook_url tells GOWA where to push inbound events
// (this is how whatomate receives messages). WebhookSecret is the HMAC key.
type CreateDeviceRequest struct {
	DeviceID                  string `json:"device_id,omitempty"`
	WebhookURL                string `json:"webhook_url,omitempty"`
	WebhookSecret             string `json:"webhook_secret,omitempty"`
	WebhookEvents             string `json:"webhook_events,omitempty"`
	WebhookInsecureSkipVerify bool   `json:"webhook_insecure_skip_verify,omitempty"`
}

// UpdateDeviceWebhookRequest is the PATCH /devices/:device_id/webhook body.
type UpdateDeviceWebhookRequest struct {
	WebhookURL                string `json:"webhook_url"`
	WebhookSecret             string `json:"webhook_secret,omitempty"`
	WebhookEvents             string `json:"webhook_events,omitempty"`
	WebhookInsecureSkipVerify bool   `json:"webhook_insecure_skip_verify,omitempty"`
}

// DeviceStatus is the GET /devices/:device_id/status response.
type DeviceStatus struct {
	DeviceID    string `json:"device_id"`
	IsConnected bool   `json:"is_connected"`
	IsLoggedIn  bool   `json:"is_logged_in"`
}

// LoginResponse is the GET /devices/:device_id/login (QR) response.
// QRLink is a URL whatomate can render or proxy to the UI.
type LoginResponse struct {
	DeviceID   string `json:"device_id"`
	QRLink     string `json:"qr_link"`
	QRDuration int    `json:"qr_duration"`
}

// LoginWithCodeResponse is the POST /devices/:device_id/login/code response.
type LoginWithCodeResponse struct {
	DeviceID string `json:"device_id"`
	PairCode string `json:"pair_code"`
}

// ----- Send -------------------------------------------------------------------

// SendTextRequest is the POST /send/message body.
//
// ReplyMessageID mirrors GOWA's `reply_message_id` field (a *string). When set
// to a non-empty WhatsApp message ID (wamid), GOWA renders the send as a quoted
// reply to that message. The field is a pointer so an empty reply is encoded
// as `null`/omitted, not as the empty string (which GOWA would still treat as
// "no reply" but we keep the typed distinction for clarity).
//
// NOTE: previous versions of this struct used the JSON tag `reply_to`, which
// GOWA silently ignored — the message went out as a plain text with no quote
// context. The correct upstream field is `reply_message_id`.
type SendTextRequest struct {
	Phone          string  `json:"phone"`
	Message        string  `json:"message"`
	ReplyMessageID *string `json:"reply_message_id,omitempty"`
}

// SendImageRequest is the payload structure for GOWA's /send/image.
type SendImageRequest struct {
	Phone    string `json:"phone"`
	Caption  string `json:"caption,omitempty"`
	ImageURL string `json:"image_url"`
}

// SendVideoRequest is the payload structure for GOWA's /send/video.
type SendVideoRequest struct {
	Phone    string `json:"phone"`
	Caption  string `json:"caption,omitempty"`
	VideoURL string `json:"video_url"`
}

// SendAudioRequest is the payload structure for GOWA's /send/audio.
type SendAudioRequest struct {
	Phone    string `json:"phone"`
	AudioURL string `json:"audio_url"`
}

// SendFileRequest is the payload structure for GOWA's /send/file.
type SendFileRequest struct {
	Phone    string `json:"phone"`
	Caption  string `json:"caption,omitempty"`
	FileURL  string `json:"file_url"`
	Filename string `json:"filename,omitempty"`
}

// MessageActionRequest covers /message/:id/react, /revoke, /read, /delete etc.
// Phone scopes the action to a chat on the target device.
type MessageActionRequest struct {
	MessageID string `json:"message_id"`
	Phone     string `json:"phone,omitempty"`
	Emoji     string `json:"emoji,omitempty"`
}

// MessageActionResponse is returned by message mutation endpoints.
type MessageActionResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// ----- Media download ---------------------------------------------------------

// MediaDownloadResponse is the GET /message/:id/download response.
// FilePath and FileURL are relative/absolute references into GOWA's static
// storage. FileBytes is populated by a follow-up GET on FileURL when the
// adapter's DownloadMedia method is invoked.
type MediaDownloadResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename"`
	FilePath  string `json:"file_path"`
	FileURL   string `json:"file_url"`
	FileSize  int64  `json:"file_size"`
}

// ----- Chat / message read ----------------------------------------------------

// ChatInfo mirrors a single chat from GOWA's ListChats response.
type ChatInfo struct {
	JID                 string     `json:"jid"`
	Name                string     `json:"name"`
	LastMessageTime     *time.Time `json:"last_message_time,omitempty"`
	EphemeralExpiration int        `json:"ephemeral_expiration,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Archived            bool       `json:"archived"`
}

// UnmarshalJSON implements json.Unmarshaler for ChatInfo. GOWA sometimes
// returns last_message_time, created_at, or updated_at as "" (empty string)
// instead of null or omitting the field; the standard library rejects this
// for time.Time. We handle it by treating empty-string as a zero time or nil pointer.
func (c *ChatInfo) UnmarshalJSON(data []byte) error {
	type Alias ChatInfo
	aux := &struct {
		LastMessageTime string `json:"last_message_time"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.LastMessageTime != "" {
		t, err := parseGowaTime(aux.LastMessageTime)
		if err == nil {
			c.LastMessageTime = &t
		}
	} else {
		c.LastMessageTime = nil
	}
	if aux.CreatedAt != "" {
		t, err := parseGowaTime(aux.CreatedAt)
		if err == nil {
			c.CreatedAt = t
		}
	}
	if aux.UpdatedAt != "" {
		t, err := parseGowaTime(aux.UpdatedAt)
		if err == nil {
			c.UpdatedAt = t
		}
	}
	return nil
}

// MessageReaction mirrors a single reaction embedded in MessageInfo.
type MessageReaction struct {
	MessageID  string    `json:"message_id"`
	ReactorJID string    `json:"reactor_jid"`
	Emoji      string    `json:"emoji"`
	IsFromMe   bool      `json:"is_from_me"`
	Timestamp  time.Time `json:"timestamp"`
}

// UnmarshalJSON implements json.Unmarshaler for MessageReaction to handle
// GOWA sometimes returning timestamp as an empty string.
func (mr *MessageReaction) UnmarshalJSON(data []byte) error {
	type Alias MessageReaction
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(mr),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		t, err := parseGowaTime(aux.Timestamp)
		if err == nil {
			mr.Timestamp = t
		}
	}
	return nil
}

// MessageInfo mirrors a single message from GOWA's GetChatMessages response.
type MessageInfo struct {
	ID         string            `json:"id"`
	ChatJID    string            `json:"chat_jid"`
	SenderJID  string            `json:"sender_jid"`
	Content    string            `json:"content"`
	Timestamp  time.Time         `json:"timestamp"`
	IsFromMe   bool              `json:"is_from_me"`
	MediaType  string            `json:"media_type,omitempty"`
	Reactions  []MessageReaction `json:"reactions,omitempty"`
	Filename   string            `json:"filename,omitempty"`
	URL        string            `json:"url,omitempty"`
	FileLength int64             `json:"file_length,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// UnmarshalJSON implements json.Unmarshaler for MessageInfo to handle
// GOWA sometimes returning timestamp, created_at, or updated_at as an empty string.
func (m *MessageInfo) UnmarshalJSON(data []byte) error {
	type Alias MessageInfo
	aux := &struct {
		Timestamp string `json:"timestamp"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		t, err := parseGowaTime(aux.Timestamp)
		if err == nil {
			m.Timestamp = t
		}
	}
	if aux.CreatedAt != "" {
		t, err := parseGowaTime(aux.CreatedAt)
		if err == nil {
			m.CreatedAt = t
		}
	}
	if aux.UpdatedAt != "" {
		t, err := parseGowaTime(aux.UpdatedAt)
		if err == nil {
			m.UpdatedAt = t
		}
	}
	return nil
}

// Pagination mirrors GOWA's pagination block.
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// ListChatsResponse is the GOWA /chats response.
type ListChatsResponse struct {
	Data       []ChatInfo `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListChatsFilter carries the query parameters GOWA /chats accepts.
type ListChatsFilter struct {
	Limit    int
	Offset   int
	Search   string
	HasMedia bool
	Archived *bool // nil = omit
}

// GetMessagesResponse is the GOWA /chat/:jid/messages response.
type GetMessagesResponse struct {
	Data       []MessageInfo `json:"data"`
	Pagination Pagination    `json:"pagination"`
	ChatInfo   *ChatInfo     `json:"chat_info,omitempty"`
}

// GetMessagesFilter carries the query parameters GOWA /chat/:jid/messages accepts.
type GetMessagesFilter struct {
	Limit     int
	Offset    int
	Search    string
	StartTime string // RFC3339
	EndTime   string // RFC3339
	MediaOnly bool
	IsFromMe  *bool // nil = omit
}

// ----- Webhook ----------------------------------------------------------------
//
// WebhookEvent is the inbound event envelope GOWA POSTs to whatomate's
// /api/gowa/webhook. GOWA's webhook_forward.go populates this with the raw
// whatsmeow event payload plus device scoping.
//
// EventType is one of: message, receipt, reaction, call, presence,
// chat_presence, archive, pin, delete, group, newsletter, etc. The receiver
// dispatches on EventType and parses Data lazily per type.
//
// NOTE: This is a legacy/unused DTO kept for documentation parity with GOWA's
// webhook_forward.go shape. The live inbound payload parsed by whatomate is
// gowaMessagePayload in internal/handlers/webhook_gowa.go, which is a separate
// struct and not a collision.
type WebhookEvent struct {
	DeviceID  string         `json:"device_id"`
	EventType string         `json:"event_type"`
	Data      map[string]any `json:"data"`
}
