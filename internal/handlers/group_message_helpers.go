package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	groupJIDSuffix       = "@g.us"
	newsletterJIDSuffix  = "@newsletter"
	defaultUserJIDSuffix = "@s.whatsapp.net"
)

func isGroupConversationID(conversationID string) bool {
	return strings.HasSuffix(strings.TrimSpace(conversationID), groupJIDSuffix)
}

func isGroupMessage(message models.Message) bool {
	if isGroupConversationID(message.ConversationID) {
		return true
	}
	if message.Metadata == nil {
		return false
	}
	if isGroup, ok := message.Metadata["is_group_chat"].(bool); ok && isGroup {
		return true
	}
	if isGroup, ok := message.Metadata["is_group"].(bool); ok && isGroup {
		return true
	}
	return false
}

func isChannelConversationID(conversationID string) bool {
	return strings.HasSuffix(strings.TrimSpace(conversationID), newsletterJIDSuffix)
}

func directUserFromConversationID(conversationID string) string {
	normalized := strings.TrimSpace(conversationID)
	if normalized == "" {
		return ""
	}
	if strings.HasSuffix(normalized, defaultUserJIDSuffix) {
		return strings.TrimSuffix(normalized, defaultUserJIDSuffix)
	}
	return ""
}

func isChannelMessage(message models.Message) bool {
	if isChannelConversationID(message.ConversationID) {
		return true
	}
	if message.Metadata == nil {
		return false
	}
	if isChannel, ok := message.Metadata["is_channel_chat"].(bool); ok && isChannel {
		return true
	}
	if isChannel, ok := message.Metadata["is_channel"].(bool); ok && isChannel {
		return true
	}
	return false
}

func messageMetadataString(metadata models.JSONB, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	parsed, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(parsed)
}

func extractMessageSenderPhone(metadata models.JSONB) string {
	return messageMetadataString(metadata, "sender_phone")
}

func extractMessageSenderPushName(metadata models.JSONB) string {
	return messageMetadataString(metadata, "sender_push_name")
}

func isGroupContact(contact *models.Contact) bool {
	if contact == nil {
		return false
	}
	if isGroupConversationID(contact.PhoneNumber) {
		return true
	}
	if contact.Metadata == nil {
		return false
	}
	isGroup, ok := contact.Metadata["is_group_chat"].(bool)
	return ok && isGroup
}

func isChannelContact(contact *models.Contact) bool {
	if contact == nil {
		return false
	}
	if isChannelConversationID(contact.PhoneNumber) {
		return true
	}
	if contact.Metadata == nil {
		return false
	}
	isChannel, ok := contact.Metadata["is_channel_chat"].(bool)
	return ok && isChannel
}
