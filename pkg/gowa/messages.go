package gowa

import (
	"context"
	"fmt"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
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
// inline) or a URL passed as the image_url field.
func (c *Client) SendImageMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "image", "/send/image")
}

// SendVideoMessage sends a video message.
func (c *Client) SendVideoMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "video", "/send/video")
}

// SendAudioMessage sends an audio/voice message.
func (c *Client) SendAudioMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, "", "audio", "/send/audio")
}

// SendDocumentMessage sends a document/file.
func (c *Client) SendDocumentMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, filename, caption string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, caption, "file", "/send/file")
}

// sendMedia is the shared multipart/JSON sender for all media types.
// If the mediaID was produced by UploadMedia it is in the in-memory cache
// and is sent as a binary multipart field. Otherwise mediaID is treated as
// a URL and sent as the *_url JSON field.
func (c *Client) sendMedia(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID, caption, fileType, path string) (string, error) {
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

// SendVoiceCallButton is not supported by GOWA.
func (c *Client) SendVoiceCallButton(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, bodyText, displayText string, ttlMinutes int, payload string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

// MarkMessageRead marks a message as read via /message/{id}/read.
func (c *Client) MarkMessageRead(ctx context.Context, account *whatsapp.Account, messageID string) error {
	path := fmt.Sprintf("/message/%s/read", messageID)
	body := map[string]any{
		"phone": toJID(""), // phone is required but unknown at this point
	}
	// We cannot know the sender JID from just a messageID in the current
	// interface signature. GOWA requires it. This is a known limitation;
	// callers that have the phone should use a GOWA-specific method.
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
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
