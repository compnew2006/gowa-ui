package handlers

import (
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

	systemMessage := models.Message{
		OrganizationID:  contact.OrganizationID,
		InstanceID:      contact.InstanceID,
		WhatsAppAccount: contact.WhatsAppAccount,
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
