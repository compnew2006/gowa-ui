package provider

import "context"

// MessageProvider defines the interface for sending messages and managing interactions
// across different WhatsApp providers (Meta Cloud API, whatsmeow, etc.)
type MessageProvider interface {
	// SendText sends a text message
	SendText(ctx context.Context, instanceID string, to string, text string) (string, error)

	// SendImage sends an image message
	SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error)

	// SendDocument sends a document message
	SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string, caption string) (string, error)

	// SendVideo sends a video message
	SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error)

	// SendAudio sends an audio message
	SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error)

	// MarkRead marks a message as read
	MarkRead(ctx context.Context, instanceID string, messageID string) error

	// SendReaction sends an emoji reaction to a message
	SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error

	// RevokeMessage deletes an outgoing message from WhatsApp
	RevokeMessage(ctx context.Context, instanceID string, messageID string) error

	// GetMediaURL retrieves a temporary URL for a media ID (Meta specific usually, but useful abstraction)
	GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error)

	// DownloadMedia downloads media bytes from a URL
	DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error)

	// UploadMedia uploads media bytes and returns a handle/ID/URL
	UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error)
}

// ReplyProvider is an optional extension to MessageProvider for adapters that
// support quoted replies (reply-to-message context). Callers should type-assert
// the MessageProvider to check if this is supported.
type ReplyProvider interface {
	// SendTextReply sends a text message as a reply to a specific message
	SendTextReply(ctx context.Context, instanceID string, to string, text string, replyToMsgID string) (string, error)
}
