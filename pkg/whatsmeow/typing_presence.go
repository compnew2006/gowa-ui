package whatsmeow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
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
