package gowa

import (
	"context"
	"fmt"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
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

	// DeleteMessage deletes a message locally (only for the connected account).
	// GOWA endpoint: POST /message/{message_id}/delete
	// Body: { phone: chatJID }
	DeleteMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error

	// StarMessage stars a message.
	// GOWA endpoint: POST /message/{message_id}/star
	// Body: { phone: chatJID }
	StarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error

	// UnstarMessage removes a star from a message.
	// GOWA endpoint: POST /message/{message_id}/unstar
	// Body: { phone: chatJID }
	UnstarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error

	// ForwardMessage forwards a message to another chat.
	// GOWA endpoint: POST /message/{message_id}/forward
	// Body: { phone: destJID }
	ForwardMessage(ctx context.Context, account *whatsapp.Account, messageID, destJID string) error

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

// RevokeMessage unsends a message for everyone.
func (c *Client) RevokeMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/revoke", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// DeleteMessage deletes a message locally.
func (c *Client) DeleteMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/delete", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// StarMessage stars a message.
func (c *Client) StarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/star", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// UnstarMessage removes a star from a message.
func (c *Client) UnstarMessage(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) error {
	path := fmt.Sprintf("/message/%s/unstar", messageID)
	body := map[string]any{"phone": toJID(chatJID)}
	_, err := c.doJSON(ctx, "POST", path, deviceID(account), body)
	return err
}

// ForwardMessage forwards a message to another chat.
func (c *Client) ForwardMessage(ctx context.Context, account *whatsapp.Account, messageID, destJID string) error {
	path := fmt.Sprintf("/message/%s/forward", messageID)
	body := map[string]any{"phone": toJID(destJID)}
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
