package gowa

import (
	"context"

	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
)

// SendTextMessage sends a plain-text message via GOWA.
func (c *Client) SendTextMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, text string, replyToMsgID ...string) (string, error) {
	body := map[string]any{
		"phone":   toJID(rcpt.Phone),
		"message": text,
	}
	if len(replyToMsgID) > 0 && replyToMsgID[0] != "" {
		body["reply_message_id"] = replyToMsgID[0]
	}
	return c.doJSON(ctx, "POST", "/send/message", deviceID(account), body)
}

// resolveMediaData returns the raw bytes for a mediaID produced by
// UploadMedia, or treats mediaID as a URL if it is not cached.
func (c *Client) resolveMediaData(mediaID string) ([]byte, string, string, error) {
	if item, ok := c.popMedia(mediaID); ok {
		return item.Data, item.MimeType, item.Filename, nil
	}
	// Not in cache — treat as a URL.
	return nil, "", mediaID, nil
}

// SendImageMessage sends an image. Because GOWA has no separate upload
// endpoint, mediaID is either a key returned by UploadMedia (consumed
// inline) or a URL passed as the image_url field. replyMessageID, when
// non-empty, quotes the referenced message (GOWA reply_message_id).
func (c *Client) SendImageMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption, replyMessageID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "image", "/send/image", replyMessageID)
}

// SendVideoMessage sends a video message.
func (c *Client) SendVideoMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption, replyMessageID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "video", "/send/video", replyMessageID)
}

// SendAudioMessage sends an audio/voice message.
func (c *Client) SendAudioMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, replyMessageID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, "", "audio", "/send/audio", replyMessageID)
}

// SendDocumentMessage sends a document/file.
func (c *Client) SendDocumentMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, filename, caption, replyMessageID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "file", "/send/file", replyMessageID)
}

// sendMedia is the shared multipart/JSON sender for all media types.
// If the mediaID was produced by UploadMedia it is in the in-memory cache
// and is sent as a binary multipart field. Otherwise mediaID is treated as
// a URL and sent as the *_url JSON field. replyMessageID, when non-empty,
// is forwarded as reply_message_id so the recipient sees the quoted
// context (mirrors SendTextMessage's empty-omit behavior).
func (c *Client) sendMedia(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption, fileType, path, replyMessageID string) (string, error) {
	phone := toJID(rcpt.Phone)
	data, _, cachedFilename, err := c.resolveMediaData(mediaID)
	if err != nil {
		return "", err
	}

	// If data is non-nil, we have bytes from the cache → send as multipart.
	if data != nil {
		fields := map[string]string{"phone": phone}
		if caption != "" {
			fields["caption"] = caption
		}
		if replyMessageID != "" {
			fields["reply_message_id"] = replyMessageID
		}
		fileField := fileType // "image", "video", "audio", "file"
		fileName := cachedFilename
		if fileName == "" || !hasExtension(fileName) {
			// GOWA requires a filename with a valid extension (e.g. .jpg, .png).
			// Derive one from the file type when the caller didn't provide one.
			fileName = fileType + defaultExtension(fileType)
		}
		return c.doMultipart(ctx, "POST", path, deviceID(account), fields, fileField, fileName, data)
	}

	// No cached bytes — send as URL.
	urlField := fileType + "_url" // "image_url", "video_url", etc.
	body := map[string]any{
		"phone":  phone,
		urlField: mediaID,
	}
	if caption != "" {
		body["caption"] = caption
	}
	if replyMessageID != "" {
		body["reply_message_id"] = replyMessageID
	}
	return c.doJSON(ctx, "POST", path, deviceID(account), body)
}

// SendInteractiveButtons is not supported by GOWA v8.10.0.
func (c *Client) SendInteractiveButtons(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, bodyText string, buttons []whatsapp.Button) (string, error) {
	return "", whatsapp.ErrNotSupported
}

// SendCTAURLButton sends a link-preview card via /send/link.
// GOWA does not support native CTA URL buttons, so we approximate with a
// link preview message.
func (c *Client) SendCTAURLButton(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, bodyText, buttonText, url string) (string, error) {
	body := map[string]any{
		"phone": toJID(rcpt.Phone),
		"link":  url,
	}
	if bodyText != "" {
		body["caption"] = bodyText
	}
	return c.doJSON(ctx, "POST", "/send/link", deviceID(account), body)
}

// StatusBroadcastJID is WhatsApp's well-known JID for the Status (story)
// service. Unlike a phone-based JID it is constant — every status post is
// addressed to this single recipient, and the account's contacts see it as a
// story. GOWA's /send/{message,image,video} accept this value verbatim as the
// `phone` field because its pipeline (SanitizePhone → ParseJID →
// ValidateJidWithLogin) preserves any "@"-bearing JID and skips the
// "is not on whatsapp" check (which only applies to @s.whatsapp.net).
const StatusBroadcastJID = "status@broadcast"

// PostStatusText posts a text WhatsApp Status (story) from the connected
// account. It reuses the plain-message send path with the well-known
// status@broadcast JID — GOWA treats it as any other recipient.
func (c *Client) PostStatusText(ctx context.Context, account *whatsapp.Account, text string) (string, error) {
	return c.SendTextMessage(ctx, account, whatsapp.Recipient{Phone: StatusBroadcastJID}, text)
}

// PostStatusImage posts an image WhatsApp Status. mediaID follows the same
// rules as SendImageMessage (UploadMedia key or URL).
func (c *Client) PostStatusImage(ctx context.Context, account *whatsapp.Account, mediaID, caption string) (string, error) {
	return c.sendMedia(ctx, account, whatsapp.Recipient{Phone: StatusBroadcastJID}, mediaID, caption, "image", "/send/image", "")
}

// PostStatusVideo posts a video WhatsApp Status. mediaID follows the same
// rules as SendVideoMessage (UploadMedia key or URL).
func (c *Client) PostStatusVideo(ctx context.Context, account *whatsapp.Account, mediaID, caption string) (string, error) {
	return c.sendMedia(ctx, account, whatsapp.Recipient{Phone: StatusBroadcastJID}, mediaID, caption, "video", "/send/video", "")
}

// MarkMessageRead marks a message as read via /message/{id}/read.
//
// The whatsapp.Provider interface mandates this signature, but GOWA's
// /message/{id}/read REQUIRES the chat JID (`phone`) that this signature does
// not carry. The previous implementation sent an empty phone, which GOWA
// rejects with 400 — a latent API defect (gap #12). It now fails fast with
// ErrNotSupported and directs callers to the GOWA-specific method that carries
// the JID. The production read path already uses MarkMessageReadWithJID.
func (c *Client) MarkMessageRead(ctx context.Context, account *whatsapp.Account, messageID string) error {
	_ = ctx
	_ = account
	_ = messageID
	return whatsapp.ErrNotSupported
}

// hasExtension reports whether the filename has a file extension (contains a dot
// in the last path segment).
func hasExtension(name string) bool {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return true
		}
		if name[i] == '/' || name[i] == '\\' {
			break
		}
	}
	return false
}

// defaultExtension returns the conventional file extension for a GOWA file type
// so the multipart upload filename satisfies GOWA's extension validation.
func defaultExtension(fileType string) string {
	switch fileType {
	case "image":
		return ".jpg"
	case "video":
		return ".mp4"
	case "audio":
		return ".mp3"
	case "file":
		return ".pdf"
	case "sticker":
		return ".webp"
	default:
		return ".bin"
	}
}
