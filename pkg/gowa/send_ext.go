package gowa

import (
	"context"
	"fmt"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
)

// SendSticker sends a sticker message. The mediaID is either a key from
// UploadMedia (sent as multipart) or a URL (sent as sticker_url).
// GOWA endpoint: POST /send/sticker (multipart/form-data)
func (c *Client) SendSticker(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, mediaID string) (string, error) {
	return c.sendMedia(ctx, account, rcpt, mediaID, "", "sticker", "/send/sticker", "")
}

// SendContact sends a contact card.
// GOWA endpoint: POST /send/contact (JSON)
func (c *Client) SendContact(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, contactName, contactPhone string) (string, error) {
	body := map[string]any{
		"phone":         toJID(rcpt.Phone),
		"contact_name":  contactName,
		"contact_phone": contactPhone,
	}
	return c.doJSON(ctx, "POST", "/send/contact", deviceID(account), body)
}

// SendLocation sends a location message.
// GOWA endpoint: POST /send/location (JSON)
func (c *Client) SendLocation(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, latitude, longitude string) (string, error) {
	body := map[string]any{
		"phone":     toJID(rcpt.Phone),
		"latitude":  latitude,
		"longitude": longitude,
	}
	return c.doJSON(ctx, "POST", "/send/location", deviceID(account), body)
}

// SendPoll sends a native WhatsApp poll.
// GOWA endpoint: POST /send/poll (JSON)
func (c *Client) SendPoll(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, question string, options []string, maxAnswers int) (string, error) {
	body := map[string]any{
		"phone":      toJID(rcpt.Phone),
		"question":   question,
		"options":    options,
		"max_answer": maxAnswers,
	}
	return c.doJSON(ctx, "POST", "/send/poll", deviceID(account), body)
}

// SendPresence sets the connected account's presence (available/unavailable).
// GOWA endpoint: POST /send/presence (JSON)
func (c *Client) SendPresence(ctx context.Context, account *whatsapp.Account, presenceType string) error {
	body := map[string]any{"type": presenceType}
	_, err := c.doJSON(ctx, "POST", "/send/presence", deviceID(account), body)
	return err
}

// SendChatPresence sends a typing/recording indicator to a chat.
// action = "start" or "stop".
// GOWA endpoint: POST /send/chat-presence (JSON)
func (c *Client) SendChatPresence(ctx context.Context, account *whatsapp.Account, chatJID, action string) error {
	body := map[string]any{
		"phone":  toJID(chatJID),
		"action": action,
	}
	_, err := c.doJSON(ctx, "POST", "/send/chat-presence", deviceID(account), body)
	return err
}

// UpdateMessage edits the text of a previously sent message.
// GOWA endpoint: POST /message/{message_id}/update (JSON)
func (c *Client) UpdateMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID, newText string) error {
	path := fmt.Sprintf("/message/%s/update", messageID)
	body := map[string]any{
		"phone":   toJID(chatJID),
		"message": newText,
	}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}
