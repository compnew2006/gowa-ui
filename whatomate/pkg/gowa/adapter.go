package gowa

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zerodha/logf"
	"gorm.io/gorm"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/provider"
)

// Compile-time assertion: GowaAdapter implements the full base MessageProvider.
// If the interface changes upstream, this fails to compile here rather than at
// runtime.
var _ provider.MessageProvider = (*GowaAdapter)(nil)

// ErrGowaNotConnected is returned when an adapter method is invoked against a
// GOWA server that is unreachable on every retry attempt.
var ErrGowaNotConnected = errors.New("gowa: backend unreachable after retries")

// GowaAdapter implements provider.MessageProvider by proxying every operation
// to an external GOWA HTTP server via a *Client.
//
// The adapter is intentionally thin at this stage of the rollout (Stage 1):
// device provisioning and lifecycle wiring land in Stage 4 (instances.go),
// and the full MessageProvider method bodies land in Stage 3. Stage 1's scope
// is config + bootstrap + a compile-safe skeleton that proves the wiring
// works end-to-end.
//
// Methods that are not yet implemented return a typed error so callers can
// distinguish "intentionally stubbed" from "GOWA failure" while we build out
// the rest of the surface. Boot-time wiring still works because the adapter
// satisfies the interface; the stubs only fire if a caller actually invokes
// them, which won't happen until Stage 3 lands.
type GowaAdapter struct {
	client *Client
	db     *gorm.DB
	log    logf.Logger

	// retry configuration (mirrors cfg.Gowa.MaxRetries). Cached at construction
	// so retries do not need to re-read config on every call.
	maxRetries int
}

// NewAdapter constructs a GowaAdapter. db is reserved for Stage 3/4 (device
// lookup, message persistence); it is not yet used but kept in the signature
// so the wiring site in main.go does not need to change between stages.
func NewAdapter(client *Client, db *gorm.DB, log logf.Logger) *GowaAdapter {
	a := &GowaAdapter{
		client: client,
		db:     db,
		log:    log,
	}
	return a
}

// Client returns the underlying GOWA HTTP client. Exposed so Stage 4/5/6/7
// code (instance lifecycle, read-side proxy handlers, webhook receiver,
// polling reconciler) can call device/chat/message methods without going
// through the MessageProvider interface.
func (a *GowaAdapter) Client() *Client { return a.client }

// resolveDeviceID maps a whatomate instance UUID to the GOWA device_id string.
// The provider.MessageProvider interface passes instanceID as a string (UUID),
// but GOWA addresses devices by the operator-chosen Name we used at
// provisioning time (see gowaCreateDevice). We look up the instance by ID and
// return its Name; if the lookup fails or the ID is not a UUID, we fall back
// to passing the string through verbatim (covers callers that already pass a
// device_id directly, e.g. tests).
func (a *GowaAdapter) resolveDeviceID(ctx context.Context, instanceID string) string {
	if a.db == nil || instanceID == "" {
		return instanceID
	}
	var name string
	if err := a.db.WithContext(ctx).
		Table("whatsapp_instances").
		Where("id = ?", instanceID).
		Limit(1).
		Pluck("name", &name).Error; err != nil || name == "" {
		return instanceID
	}
	return name
}

// ----- provider.MessageProvider (Stage 3 — full impl) -----

// SendText sends a text message via GOWA POST /send/message.
//
// instanceID is the whatomate UUID; we resolve it to the GOWA device_id
// (instance.Name) before calling GOWA. "to" is a phone number.
func (a *GowaAdapter) SendText(ctx context.Context, instanceID, to, text string) (string, error) {
	deviceID := a.resolveDeviceID(ctx, instanceID)
	req := SendTextRequest{Phone: to, Message: text}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendText(ctx, deviceID, req)
	})
}

// SendTextReply sends a text message as a quoted reply to a specific message.
// Implements provider.ReplyProvider so sendViaProvider routes replies through
// this method when the GOWA backend is active — without it, the type assertion
// in sendViaProvider fails and replies silently degrade to plain SendText.
//
// GOWA builds the WhatsApp ContextInfo (StanzaID/Participant/QuotedMessage)
// server-side from the reply_message_id we pass here; the value must be the
// WhatsApp message ID (wamid) of the message being quoted, not the whatomate
// row UUID. The caller (sendViaProvider) already resolves this for us.
func (a *GowaAdapter) SendTextReply(ctx context.Context, instanceID, to, text, replyToMsgID string) (string, error) {
	if strings.TrimSpace(replyToMsgID) == "" {
		// No quote target — degrade to a normal send instead of erroring, so
		// the message still goes out. This matches the whatsmeow adapter's
		// behaviour when reply context can't be resolved.
		return a.SendText(ctx, instanceID, to, text)
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	replyID := replyToMsgID
	req := SendTextRequest{Phone: to, Message: text, ReplyMessageID: &replyID}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendText(ctx, deviceID, req)
	})
}

// Compile-time guard: GowaAdapter satisfies provider.ReplyProvider. Without
// this, sendViaProvider's MessageProvider.(ReplyProvider) assertion silently
// fails for GOWA and replies are sent as plain text (the bug we just fixed).
var _ provider.ReplyProvider = (*GowaAdapter)(nil)

// SendImage sends an image message via GOWA POST /send/image. imageURL must be
// publicly fetchable by GOWA. If whatomate stored the file locally, the
// caller is responsible for producing a public URL first.
// SendImage sends an image via GOWA POST /send/image.
// GOWA does not support binary uploads on this endpoint; the
// caller is responsible for producing a public URL first.
func (a *GowaAdapter) SendImage(ctx context.Context, instanceID, to, imageURL, caption string) (string, error) {
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return "", fmt.Errorf("gowa: SendImage requires a publicly accessible URL starting with http:// or https://, got: %q", imageURL)
	}
	if !strings.Contains(to, "@") {
		to = to + "@s.whatsapp.net"
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	req := SendImageRequest{Phone: to, ImageURL: imageURL, Caption: caption}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendImage(ctx, deviceID, req)
	})
}

// SendDocument sends a document via GOWA POST /send/file.
func (a *GowaAdapter) SendDocument(ctx context.Context, instanceID, to, docURL, filename, caption string) (string, error) {
	if !strings.HasPrefix(docURL, "http://") && !strings.HasPrefix(docURL, "https://") {
		return "", fmt.Errorf("gowa: SendDocument requires a publicly accessible URL starting with http:// or https://, got: %q", docURL)
	}
	if !strings.Contains(to, "@") {
		to = to + "@s.whatsapp.net"
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	req := SendFileRequest{Phone: to, FileURL: docURL, Filename: filename, Caption: caption}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendFile(ctx, deviceID, req)
	})
}

// SendVideo sends a video via GOWA POST /send/video.
func (a *GowaAdapter) SendVideo(ctx context.Context, instanceID, to, videoURL, caption string) (string, error) {
	if !strings.HasPrefix(videoURL, "http://") && !strings.HasPrefix(videoURL, "https://") {
		return "", fmt.Errorf("gowa: SendVideo requires a publicly accessible URL starting with http:// or https://, got: %q", videoURL)
	}
	if !strings.Contains(to, "@") {
		to = to + "@s.whatsapp.net"
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	req := SendVideoRequest{Phone: to, VideoURL: videoURL, Caption: caption}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendVideo(ctx, deviceID, req)
	})
}

// SendAudio sends an audio via GOWA POST /send/audio.
func (a *GowaAdapter) SendAudio(ctx context.Context, instanceID, to, audioURL string) (string, error) {
	if !strings.HasPrefix(audioURL, "http://") && !strings.HasPrefix(audioURL, "https://") {
		return "", fmt.Errorf("gowa: SendAudio requires a publicly accessible URL starting with http:// or https://, got: %q", audioURL)
	}
	if !strings.Contains(to, "@") {
		to = to + "@s.whatsapp.net"
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	req := SendAudioRequest{Phone: to, AudioURL: audioURL}
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendAudio(ctx, deviceID, req)
	})
}

// ----- provider.MediaUploader (multipart direct upload) -----
//
// These methods upload the raw bytes directly to GOWA as multipart/form-data,
// bypassing the URL round-trip used by SendImage/SendDocument/etc. This is the
// preferred path for outgoing media in GOWA mode because it means whatomate
// never has to persist the outgoing file to disk — GOWA is the single source
// of truth, mirroring how incoming media already works.
//
// All four methods resolve the device_id from instanceID, normalise the
// recipient to a JID, and delegate to the corresponding Client.*Multipart
// helper. Empty MIME types are passed through; GOWA sniffs them when needed.

// compile-time guard: GowaAdapter satisfies provider.MediaUploader.
var _ provider.MediaUploader = (*GowaAdapter)(nil)

// SendImageBytes uploads image bytes directly to GOWA's /send/image.
func (a *GowaAdapter) SendImageBytes(ctx context.Context, instanceID, to string, data []byte, filename, mimeType, caption string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("gowa: SendImageBytes received empty data")
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	recipient := normaliseGowaRecipient(to)
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendImageMultipart(ctx, deviceID, recipient, caption, filename, mimeType, data)
	})
}

// SendDocumentBytes uploads document bytes directly to GOWA's /send/file.
func (a *GowaAdapter) SendDocumentBytes(ctx context.Context, instanceID, to string, data []byte, filename, mimeType, caption string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("gowa: SendDocumentBytes received empty data")
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	recipient := normaliseGowaRecipient(to)
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendFileMultipart(ctx, deviceID, recipient, caption, filename, mimeType, data)
	})
}

// SendVideoBytes uploads video bytes directly to GOWA's /send/video.
func (a *GowaAdapter) SendVideoBytes(ctx context.Context, instanceID, to string, data []byte, filename, mimeType, caption string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("gowa: SendVideoBytes received empty data")
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	recipient := normaliseGowaRecipient(to)
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendVideoMultipart(ctx, deviceID, recipient, caption, filename, mimeType, data)
	})
}

// SendAudioBytes uploads audio bytes directly to GOWA's /send/audio.
func (a *GowaAdapter) SendAudioBytes(ctx context.Context, instanceID, to string, data []byte, filename, mimeType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("gowa: SendAudioBytes received empty data")
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	recipient := normaliseGowaRecipient(to)
	return a.withRetry(ctx, func(ctx context.Context) (string, error) {
		return a.client.SendAudioMultipart(ctx, deviceID, recipient, filename, mimeType, data)
	})
}

// normaliseGowaRecipient turns a bare phone number into a WhatsApp JID. JIDs
// are passed through unchanged. This mirrors the heuristic in SendAudio.
func normaliseGowaRecipient(to string) string {
	if strings.Contains(to, "@") {
		return to
	}
	return to + "@s.whatsapp.net"
}

// MarkRead marks a message as read via GOWA POST /message/:id/read.
//
// NOTE: provider.MessageProvider.MarkRead does not take a "to"/"phone"
// argument, but GOWA's API requires one to scope the action. Until Stage 3
// refines the contract, we send an empty phone; GOWA's handler treats phone
// as optional and falls back to the message_id alone.
func (a *GowaAdapter) MarkRead(ctx context.Context, instanceID, messageID string) error {
	deviceID := a.resolveDeviceID(ctx, instanceID)
	return a.withRetryErr(ctx, func(ctx context.Context) error {
		return a.client.MarkRead(ctx, deviceID, messageID, "")
	})
}

// SendReaction reacts to a message via GOWA POST /message/:id/reaction.
func (a *GowaAdapter) SendReaction(ctx context.Context, instanceID, messageID, emoji string) error {
	deviceID := a.resolveDeviceID(ctx, instanceID)
	phone := provider.GetRecipientPhone(ctx)
	if phone == "" && a.db != nil {
		// Fallback to database lookup if context doesn't carry it (mirroring MetaAdapter)
		var msg models.Message
		if err := a.db.Where("whats_app_message_id = ?", messageID).First(&msg).Error; err == nil {
			var contact models.Contact
			if err := a.db.Where("id = ?", msg.ContactID).First(&contact).Error; err == nil {
				phone = contact.PhoneNumber
			}
		}
	}
	if phone != "" && !strings.Contains(phone, "@") {
		phone = phone + "@s.whatsapp.net"
	}
	return a.withRetryErr(ctx, func(ctx context.Context) error {
		return a.client.ReactMessage(ctx, deviceID, messageID, phone, emoji)
	})
}

// RevokeMessage revokes an outgoing message via GOWA POST /message/:id/revoke.
// Unlike the Meta adapter, GOWA actually supports this through whatsmeow.
func (a *GowaAdapter) RevokeMessage(ctx context.Context, instanceID, messageID string) error {
	deviceID := a.resolveDeviceID(ctx, instanceID)
	return a.withRetryErr(ctx, func(ctx context.Context) error {
		return a.client.RevokeMessage(ctx, deviceID, messageID, "")
	})
}

// GetMediaURL returns a temporary URL for a media ID. GOWA does not expose a
// "get media URL by ID" endpoint like Meta does; instead, media is addressed
// by message_id via /message/:id/download. This method therefore returns
// ErrUnsupportedForGowa — callers should use DownloadMedia by message_id
// instead. Stage 3 will refine this contract.
func (a *GowaAdapter) GetMediaURL(ctx context.Context, instanceID, mediaID string) (string, error) {
	return "", fmt.Errorf("gowa: GetMediaURL is unsupported on the GOWA backend (use DownloadMedia by message_id instead): media_id=%s", mediaID)
}

// DownloadMedia downloads media bytes. GOWA's /message/:id/download returns
// metadata with a FileURL; we then fetch the bytes. The mediaURL argument
// from the interface is interpreted as the message_id for GOWA (the upstream
// caller in whatomate uses this method for both URL-based and ID-based fetch).
//
// Stage 3 will reconcile the contract with the rest of the codebase.
func (a *GowaAdapter) DownloadMedia(ctx context.Context, instanceID, mediaURL string) ([]byte, error) {
	// Treat mediaURL as an absolute URL first; fall back to GOWA message_id.
	if isAbsoluteHTTPURL(mediaURL) {
		// Pass the resolved device ID so FetchBytes can attach X-Device-Id for
		// GOWA-hosted URLs (required by the device-scoping middleware in v8).
		return a.client.FetchBytes(ctx, mediaURL, a.resolveDeviceID(ctx, instanceID))
	}
	deviceID := a.resolveDeviceID(ctx, instanceID)
	resp, err := a.client.DownloadMedia(ctx, deviceID, mediaURL, "")
	if err != nil {
		return nil, err
	}
	if resp.FileURL == "" {
		return nil, fmt.Errorf("gowa: GOWA returned no file_url for message_id=%s", mediaURL)
	}
	return a.client.FetchBytes(ctx, resp.FileURL, deviceID)
}

// UploadMedia uploads media bytes. GOWA's send endpoints expect a public URL,
// not raw bytes, so this method returns ErrUnsupportedForGowa at this stage.
// Stage 3 will introduce a whatomate-side static host (or signed S3 URL) so
// the adapter can persist the bytes and return a URL consumable by GOWA's
// /send/image, /send/file, etc.
func (a *GowaAdapter) UploadMedia(ctx context.Context, instanceID, mediaType string, data []byte) (string, error) {
	return "", fmt.Errorf("gowa: UploadMedia not yet implemented on the GOWA backend (Stage 3 will host bytes and return a public URL): media_type=%s bytes=%d", mediaType, len(data))
}
