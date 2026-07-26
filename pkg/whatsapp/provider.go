package whatsapp

import (
	"context"
	"errors"
)

// ErrNotSupported is returned when a method is called on a provider that does
// not support the operation (e.g. SendInteractiveButtons on GOWA).
var ErrNotSupported = errors.New("operation not supported by this provider")

// Capabilities advertises which optional features a provider implements.
// Handlers consult these flags before calling provider-specific methods and
// before rendering provider-specific UI sections.
type Capabilities struct {
	MediaUpload bool // two-step upload-then-send vs inline multipart (GOWA)
	Interactive bool // interactive buttons, CTA URL, list messages
}

// Provider is the unified interface for WhatsApp messaging backends.
// GOWA (Go WhatsApp Web Multi-Device) is the sole implementation; methods
// that GOWA cannot support natively return ErrNotSupported.
type Provider interface {
	// Capabilities returns the feature set supported by this provider.
	Capabilities() Capabilities

	// --- Core messaging ---

	SendTextMessage(ctx context.Context, account *Account, rcpt Recipient, text string, replyToMsgID ...string) (string, error)
	SendImageMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, caption, replyMessageID string) (string, error)
	SendVideoMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, caption, replyMessageID string) (string, error)
	SendAudioMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, replyMessageID string) (string, error)
	SendDocumentMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, filename, caption, replyMessageID string) (string, error)
	UploadMedia(ctx context.Context, account *Account, data []byte, mimeType, filename string) (string, error)
	MarkMessageRead(ctx context.Context, account *Account, messageID string) error

	// --- Interactive messages ---

	SendInteractiveButtons(ctx context.Context, account *Account, rcpt Recipient, bodyText string, buttons []Button) (string, error)
	SendCTAURLButton(ctx context.Context, account *Account, rcpt Recipient, bodyText, buttonText, url string) (string, error)

	// --- Media retrieval ---

	GetMediaURL(ctx context.Context, mediaID string, account *Account) (string, error)
	DownloadMedia(ctx context.Context, mediaURL string, accessToken string) ([]byte, error)
}
