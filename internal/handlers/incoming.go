package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
)

// WebhookStatusError represents an error in a status update
type WebhookStatusError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

// WebhookStatus represents a message status update
type WebhookStatus struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	Conversation *struct {
		ID string `json:"id"`
	} `json:"conversation,omitempty"`
	Pricing *struct {
		Billable     bool   `json:"billable"`
		PricingModel string `json:"pricing_model"`
		Category     string `json:"category"`
	} `json:"pricing,omitempty"`
	Errors []WebhookStatusError `json:"errors,omitempty"`
}

func (a *App) processIncomingMessage(phoneNumberID string, msg IncomingTextMessage, profileName string, isGroup, isNewsletter bool, senderName, senderJID string) {
	defer func() {
		if r := recover(); r != nil {
			a.Log.Error("Panic recovered in processIncomingMessage", "panic", r, "phone_id", phoneNumberID, "message_id", msg.ID)
		}
	}()

	// Check for duplicate message - the same message can be delivered multiple times
	if msg.ID != "" {
		var existingMsg models.Message
		if err := a.DB.Where("whats_app_message_id = ?", msg.ID).First(&existingMsg).Error; err == nil {
			a.Log.Debug("Duplicate message detected, skipping", "message_id", msg.ID)
			return
		}
	}

	// Process the message with chatbot logic
	a.processIncomingMessageFull(phoneNumberID, msg, profileName, isGroup, isNewsletter, senderName, senderJID)
}

func (a *App) processStatusUpdate(phoneNumberID string, status WebhookStatus) {
	defer func() {
		if r := recover(); r != nil {
			a.Log.Error("Panic recovered in processStatusUpdate", "panic", r, "phone_id", phoneNumberID, "status_id", status.ID)
		}
	}()

	messageID := status.ID
	statusValue := status.Status

	a.Log.Info("Processing status update", "message_id", messageID, "status", statusValue, "phone_number_id", phoneNumberID)

	// Resolve the account to scope the status update to its org. Fall back to a
	// zero UUID so the org-scoped lookup simply finds nothing rather than
	// matching messages in another org.
	var orgID uuid.UUID
	if account, err := a.getWhatsAppAccountCached(phoneNumberID); err == nil {
		orgID = account.OrganizationID
	}

	// Update messages table - this also handles campaign stats via incrementCampaignStat
	a.updateMessageStatus(orgID, messageID, statusValue, status.Errors)
}

// statusPriority returns the priority of a status (higher = more progressed)
func statusPriority(status models.MessageStatus) int {
	switch status {
	case models.MessageStatusPending:
		return 0
	case models.MessageStatusSent:
		return 1
	case models.MessageStatusDelivered:
		return 2
	case models.MessageStatusRead:
		return 3
	case models.MessageStatusFailed:
		return 4 // Failed can override any status
	default:
		return -1
	}
}

// updateMessageStatus updates the status of a regular message in the messages table
func (a *App) updateMessageStatus(orgID uuid.UUID, whatsappMsgID, statusValue string, errors []WebhookStatusError) {
	// Find the message by WhatsApp message ID, scoped to the owning org.
	var message models.Message
	result := a.DB.Where("whats_app_message_id = ? AND organization_id = ?", whatsappMsgID, orgID).First(&message)
	if result.Error != nil {
		a.Log.Debug("No message found for status update", "whats_app_message_id", whatsappMsgID)
		return
	}

	newStatus := models.MessageStatus(statusValue)
	currentPriority := statusPriority(message.Status)
	newPriority := statusPriority(newStatus)

	// Only update if new status is a progression (higher priority) or if it's failed
	if newPriority <= currentPriority && newStatus != models.MessageStatusFailed {
		a.Log.Debug("Ignoring status update - not a progression",
			"message_id", message.ID,
			"current_status", message.Status,
			"new_status", statusValue)
		return
	}

	updates := map[string]any{}

	switch newStatus {
	case models.MessageStatusSent:
		updates["status"] = models.MessageStatusSent
	case models.MessageStatusDelivered:
		updates["status"] = models.MessageStatusDelivered
	case models.MessageStatusRead:
		updates["status"] = models.MessageStatusRead
	case models.MessageStatusFailed:
		updates["status"] = models.MessageStatusFailed
		if len(errors) > 0 {
			// Prefer error_data.details (most descriptive), then Message, then Title.
			errText := errors[0].ErrorData.Details
			if errText == "" {
				errText = errors[0].Message
			}
			if errText == "" || errText == errors[0].Title {
				errText = errors[0].Title
			}

			updates["error_message"] = errText
		}
	default:
		a.Log.Debug("Ignoring message status update", "status", statusValue)
		return
	}

	if err := a.DB.Model(&message).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update message status", "error", err, "message_id", message.ID)
		return
	}

	a.Log.Info("Updated message status", "message_id", message.ID, "status", statusValue)

	// Update campaign stats and recipient status if this is a campaign message
	if message.Metadata != nil {
		if campaignID, ok := message.Metadata["campaign_id"].(string); ok && campaignID != "" {
			a.incrementCampaignStat(campaignID, statusValue)

			// Update the BulkMessageRecipient status and timestamps
			recipientUpdates := map[string]any{
				"status": newStatus,
			}
			switch newStatus {
			case models.MessageStatusDelivered:
				recipientUpdates["delivered_at"] = time.Now()
			case models.MessageStatusRead:
				recipientUpdates["read_at"] = time.Now()
			case models.MessageStatusFailed:
				if errMsg, ok := updates["error_message"].(string); ok && errMsg != "" {
					recipientUpdates["error_message"] = errMsg
				}
			}
			a.DB.Model(&models.BulkMessageRecipient{}).
				Where("whats_app_message_id = ?", whatsappMsgID).
				Updates(recipientUpdates)
		}
	}

	// Broadcast status update via WebSocket
	if a.WSHub != nil {
		wsPayload := map[string]any{
			"message_id": message.ID.String(),
			"status":     statusValue,
		}
		if errMsg, ok := updates["error_message"].(string); ok && errMsg != "" {
			wsPayload["error_message"] = errMsg
		}
		a.WSHub.BroadcastToOrg(message.OrganizationID, websocket.WSMessage{
			Type:    websocket.TypeStatusUpdate,
			Payload: wsPayload,
		})
	}
}
