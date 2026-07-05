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

import "time"

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
type SendTextRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
	ReplyTo string `json:"reply_to,omitempty"` // GOWA supports replying when set
}

// SendMediaRequest is the shared body shape for /send/image, /send/file,
// /send/video, /send/audio. GOWA expects a publicly fetchable URL for media.
type SendMediaRequest struct {
	Phone    string `json:"phone"`
	Caption  string `json:"caption,omitempty"`
	URL      string `json:"url"`
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

// MessageReaction mirrors a single reaction embedded in MessageInfo.
type MessageReaction struct {
	MessageID  string    `json:"message_id"`
	ReactorJID string    `json:"reactor_jid"`
	Emoji      string    `json:"emoji"`
	IsFromMe   bool      `json:"is_from_me"`
	Timestamp  time.Time `json:"timestamp"`
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
