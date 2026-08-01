package gowa

import (
	"context"
	"fmt"

	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
)

// MessageExtensions defines GOWA-native message operations that Meta Cloud API
// does not support (or supports differently). Callers type-assert the Provider
// to this interface to access these features:
//
//	if ext, ok := provider.(*gowa.Client); ok {
//	    ext.SendReaction(ctx, account, messageID, chatJID, "👍")
//	}
type MessageExtensions interface {
	// SendReaction reacts to a message with an emoji.
	// GOWA endpoint: POST /message/{message_id}/reaction
	// Body: { phone: chatJID, emoji: "👍" }
	SendReaction(ctx context.Context, account *whatsapp.Account, messageID, chatJID, emoji string) error

	// RevokeMessage unsends/deletes a message for everyone in the chat.
	// GOWA endpoint: POST /message/{message_id}/revoke
	// Body: { phone: chatJID }
	RevokeMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error

	// SendChatPresence sends a typing ("composing") indicator to a chat.
	// action is "start" or "stop". The indicator is outbound-only: it renders
	// on the recipient's WhatsApp client, not in the GOWA UI.
	// GOWA endpoint: POST /send/chat-presence
	// Body: { phone: chatJID, action: "start"|"stop" }
	SendChatPresence(ctx context.Context, account *whatsapp.Account, chatJID, action string) error

	// UnstarMessage removes a star from a message.
	// GOWA endpoint: POST /message/{message_id}/unstar
	// Body: { phone: chatJID }
	UnstarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error

	// MarkMessageReadWithJID marks a message as read, providing the chat JID
	// that GOWA requires. This is the preferred read-receipt method for GOWA
	// since the Provider interface's MarkMessageRead lacks the JID parameter.
	// GOWA endpoint: POST /message/{message_id}/read
	// Body: { phone: chatJID }
	MarkMessageReadWithJID(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error
}

// --- Implementation ---

// SendReaction reacts to a message with an emoji.
func (c *Client) SendReaction(ctx context.Context, account *whatsapp.Account, messageID, chatJID, emoji string) error {
	path := fmt.Sprintf("/message/%s/reaction", messageID)
	body := map[string]any{
		"phone": toJID(chatJID),
		"emoji": emoji,
	}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// RevokeMessage unsends a message for everyone in the chat.
func (c *Client) RevokeMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/revoke", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// SendChatPresence sends a typing ("composing") indicator to a chat. action is
// "start" or "stop"; the caller is responsible for validating the value.
func (c *Client) SendChatPresence(ctx context.Context, account *whatsapp.Account, chatJID, action string) error {
	body := map[string]any{
		"phone":  toJID(chatJID),
		"action": action,
	}
	_, err := c.doJSON(ctx, "POST", "/send/chat-presence", deviceID(account), body)
	return err
}

// UnstarMessage removes a star from a message.
func (c *Client) UnstarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/unstar", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// MarkMessageReadWithJID marks a message as read with the chat JID that GOWA requires.
// This is the GOWA-preferred read-receipt method.
func (c *Client) MarkMessageReadWithJID(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/read", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// Compile-time assertion that Client implements MessageExtensions.
var _ MessageExtensions = (*Client)(nil)
