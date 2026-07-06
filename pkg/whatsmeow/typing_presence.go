package whatsmeow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	// ErrTypingPresenceUnsupportedChat is returned when typing presence is requested for
	// non 1:1 chats (groups/channels).
	ErrTypingPresenceUnsupportedChat = errors.New("typing presence is only supported for direct chats")
	// ErrTypingPresenceInstanceUnavailable is returned when the target instance is not connected.
	ErrTypingPresenceInstanceUnavailable = errors.New("typing presence instance is unavailable")
	// ErrTypingPresenceInvalidRecipient is returned when recipient JID/phone cannot be resolved.
	ErrTypingPresenceInvalidRecipient = errors.New("typing presence recipient is invalid")
)

// SendTypingPresence sends composing/paused presence for a direct chat.
// It is best-effort and intended for live chat UX (agent typing state).
func (cm *ConnectionManager) SendTypingPresence(
	ctx context.Context,
	instanceID uuid.UUID,
	recipient string,
	state types.ChatPresence,
) error {
	if cm == nil {
		return ErrTypingPresenceInstanceUnavailable
	}

	client := cm.GetClient(instanceID)
	if client == nil || !client.IsConnected() {
		return ErrTypingPresenceInstanceUnavailable
	}

	chatJID, err := parseTypingPresenceRecipient(recipient)
	if err != nil {
		return err
	}
	if !isDirectChatJID(chatJID) {
		return ErrTypingPresenceUnsupportedChat
	}

	media := types.ChatPresenceMediaText
	if state == types.ChatPresencePaused {
		media = ""
	}

	if err := client.SendChatPresence(ctx, chatJID, state, media); err != nil {
		return fmt.Errorf("send chat presence: %w", err)
	}
	return nil
}

func parseTypingPresenceRecipient(recipient string) (types.JID, error) {
	trimmed := strings.TrimSpace(recipient)
	if trimmed == "" {
		return types.JID{}, ErrTypingPresenceInvalidRecipient
	}

	if strings.Contains(trimmed, "@") {
		jid, err := types.ParseJID(trimmed)
		if err != nil {
			return types.JID{}, fmt.Errorf("%w: %v", ErrTypingPresenceInvalidRecipient, err)
		}
		return jid, nil
	}

	normalized := normalizeTypingPresencePhone(trimmed)
	if normalized == "" {
		return types.JID{}, ErrTypingPresenceInvalidRecipient
	}

	jid, err := types.ParseJID(normalized + "@s.whatsapp.net")
	if err != nil {
		return types.JID{}, fmt.Errorf("%w: %v", ErrTypingPresenceInvalidRecipient, err)
	}
	return jid, nil
}

func normalizeTypingPresencePhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
		}
	}
	return builder.String()
}

// SubscribeChatPresence subscribes the instance to a contact's chat presence so
// WhatsApp starts delivering *events.ChatPresence (typing) updates for that chat.
// WhatsApp only emits typing events for direct chats you have subscribed to.
// Best-effort: errors are logged and returned but do not interrupt chat UX.
func (cm *ConnectionManager) SubscribeChatPresence(ctx context.Context, instanceID uuid.UUID, recipient string) error {
	if cm == nil {
		return ErrTypingPresenceInstanceUnavailable
	}

	client := cm.GetClient(instanceID)
	if client == nil || !client.IsConnected() {
		return ErrTypingPresenceInstanceUnavailable
	}

	chatJID, err := parseTypingPresenceRecipient(recipient)
	if err != nil {
		return err
	}
	if !isDirectChatJID(chatJID) {
		return ErrTypingPresenceUnsupportedChat
	}

	if err := client.SubscribePresence(ctx, chatJID); err != nil {
		return fmt.Errorf("subscribe chat presence: %w", err)
	}
	return nil
}

// HandleChatPresence processes an inbound *events.ChatPresence event from a
// contact and broadcasts a typing WebSocket event to the organization so the
// agent UI can show "typing…". It is best-effort: unknown contacts or non-direct
// chats are silently ignored.
func (cm *ConnectionManager) HandleChatPresence(evt *events.ChatPresence, instanceID, orgID uuid.UUID) {
	if cm == nil || evt == nil || cm.hub == nil {
		return
	}

	// ChatPresence embeds MessageSource; for 1:1 chats the sender is the contact.
	senderJID := evt.Sender.ToNonAD()
	if !isDirectChatJID(senderJID) || senderJID.User == "" {
		return
	}

	// Resolve the contact by phone + instance — typing from an unknown contact
	// has no chat to display in, so we skip silently rather than create a row.
	var contact models.Contact
	if err := cm.db.WithContext(context.Background()).
		Where("organization_id = ? AND phone_number = ? AND instance_id = ?",
			orgID, senderJID.User, instanceID).
		First(&contact).Error; err != nil {
		return
	}

	state := strings.ToLower(string(evt.State))
	if state != string(types.ChatPresenceComposing) && state != string(types.ChatPresencePaused) {
		return
	}

	cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeTyping,
		Payload: websocket.TypingPayload{
			ContactID: contact.ID.String(),
			State:     state,
		},
	})
}
