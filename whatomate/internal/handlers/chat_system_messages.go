package handlers

import (
	"context"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	chatSystemEventMetadataKey = "system_event"
)

// appendSystemChatMessage stores and broadcasts a system message inside a chat thread.
func (a *App) appendSystemChatMessage(contact *models.Contact, content string, metadata models.JSONB) {
	if a == nil || contact == nil {
		return
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return
	}

	if metadata == nil {
		metadata = models.JSONB{}
	}
	metadata[chatSystemEventMetadataKey] = true

	conversationContext := a.resolveContactConversationContext(context.Background(), contact.OrganizationID, *contact)
	if conversationContext.IsGroupChat {
		if _, ok := metadata["is_group_chat"]; !ok {
			metadata["is_group_chat"] = true
		}
		if conversationContext.ConversationID != "" {
			if _, ok := metadata["group_jid"]; !ok {
				metadata["group_jid"] = conversationContext.ConversationID
			}
		}
	}
	if conversationContext.IsChannelChat {
		if _, ok := metadata["is_channel_chat"]; !ok {
			metadata["is_channel_chat"] = true
		}
		if conversationContext.ConversationID != "" {
			if _, ok := metadata["channel_jid"]; !ok {
				metadata["channel_jid"] = conversationContext.ConversationID
			}
		}
	}

	systemMessage := models.Message{
		OrganizationID:  contact.OrganizationID,
		InstanceID:      contact.InstanceID,
		ConversationID:  conversationContext.ConversationID,
		WhatsAppAccount: a.resolveContactMessageAccount(contact),
		ContactID:       contact.ID,
		Direction:       models.DirectionOutgoing,
		MessageType:     models.MessageTypeText,
		Content:         trimmedContent,
		Status:          models.MessageStatusSent,
		Metadata:        metadata,
	}

	if err := a.DB.Create(&systemMessage).Error; err != nil {
		a.Log.Error("Failed to create system chat message", "error", err, "contact_id", contact.ID)
		return
	}

	a.updateContactLastMessage(contact, trimmedContent)
	a.broadcastNewMessage(contact.OrganizationID, &systemMessage, contact)
}
