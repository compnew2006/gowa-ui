package handlers

import (
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
)

type MediaInfo struct {
	MediaURL         string
	MediaMimeType    string
	MediaFilename    string
	RecoveryProvider string
	RecoveryMediaID  string
	RecoveryPhoneID  string
}

func (a *App) shouldSkipClosedChatAutoReopenForIncomingMessage(orgID uuid.UUID, contact *models.Contact, msgType, content string) bool {
	if a == nil || contact == nil {
		return false
	}
	_ = content
	if normalizeContactStatus(contact) != models.ChatStatusClosed {
		return false
	}
	if strings.TrimSpace(msgType) != "text" {
		return false
	}

	cycle, _, err := a.findActiveChatCloseRatingCycle(orgID, contact, time.Now().UTC())
	if err != nil {
		a.Log.Error("Failed to resolve pending close rating cycle before auto-reopen", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	return cycle != nil
}

func (a *App) saveIncomingMessage(account *models.WhatsAppAccount, contact *models.Contact, whatsappMsgID, msgType, content string, mediaInfo *MediaInfo, replyToWAMID string) *models.Message {
	now := time.Now()

	if a.shouldSkipClosedChatAutoReopenForIncomingMessage(account.OrganizationID, contact, msgType, content) {
		a.Log.Info("Skipping auto-reopen for inbound rating response", "contact_id", contact.ID)
	} else {
		if reopened, err := a.reopenClosedChatToPending(contact); err != nil {
			a.Log.Error("Failed to auto-reopen closed chat on incoming message", "error", err, "contact_id", contact.ID)
		} else if reopened {
			a.Log.Info("Auto-reopened closed chat after inbound message", "contact_id", contact.ID)
		}
	}

	message := models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    account.OrganizationID,
		WhatsAppAccount:   account.Name,
		ContactID:         contact.ID,
		WhatsAppMessageID: whatsappMsgID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageType(msgType),
		Content:           content,
		Status:            models.MessageStatusReceived,
	}

	if replyToWAMID != "" {
		var replyToMsg models.Message
		if err := a.DB.Where("whats_app_message_id = ?", replyToWAMID).First(&replyToMsg).Error; err == nil {
			message.IsReply = true
			message.ReplyToMessageID = &replyToMsg.ID
		} else {
			a.Log.Warn("Reply-to message not found", "reply_to_wamid", replyToWAMID)
		}
	}

	if mediaInfo != nil {
		message.MediaURL = mediaInfo.MediaURL
		message.MediaMimeType = mediaInfo.MediaMimeType
		message.MediaFilename = mediaInfo.MediaFilename
		if strings.TrimSpace(mediaInfo.RecoveryProvider) != "" && strings.TrimSpace(mediaInfo.RecoveryMediaID) != "" {
			message.Metadata = cloneMessageMetadata(message.Metadata)
			if message.Metadata == nil {
				message.Metadata = models.JSONB{}
			}
			message.Metadata[legacyMediaRecoveryProviderKey] = strings.TrimSpace(mediaInfo.RecoveryProvider)
			message.Metadata[legacyMediaRecoveryMediaIDKey] = strings.TrimSpace(mediaInfo.RecoveryMediaID)
			if phoneID := strings.TrimSpace(mediaInfo.RecoveryPhoneID); phoneID != "" {
				message.Metadata[legacyMediaRecoveryPhoneIDKey] = phoneID
			}
			message.Metadata[legacyMediaRecoveryExpiresAtKey] = now.UTC().Add(legacyMediaRecoveryTTL).Format(time.RFC3339Nano)
		}
	}

	if err := a.DB.Create(&message).Error; err != nil {
		a.Log.Error("Failed to save incoming message", "error", err)
		return nil
	}

	preview := content
	if len(preview) > 100 {
		preview = preview[:97] + "..."
	}
	if msgType != "text" {
		preview = "[" + msgType + "]"
	}

	a.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      now,
		"last_message_preview": preview,
		"is_read":              false,
		"whats_app_account":    account.Name,
		"last_inbound_at":      now,
	})

	a.Log.Info("Saved incoming message", "message_id", message.ID, "contact_id", contact.ID, "media_url", message.MediaURL)

	if a.WSHub != nil && !a.licenseBlocksValueDelivery() {
		var assignedUserIDStr string
		if contact.AssignedUserID != nil {
			assignedUserIDStr = contact.AssignedUserID.String()
		}
		profileName := contact.ProfileName
		if a.ShouldMaskPhoneNumbers(account.OrganizationID) {
			profileName = MaskIfPhoneNumber(profileName)
		}
		wsPayload := map[string]any{
			"id":               message.ID.String(),
			"contact_id":       contact.ID.String(),
			"assigned_user_id": assignedUserIDStr,
			"contact_status":   normalizeContactStatus(contact).String(),
			"profile_name":     profileName,
			"direction":        message.Direction,
			"message_type":     message.MessageType,
			"content":          map[string]string{"body": message.Content},
			"media_url":        message.MediaURL,
			"media_mime_type":  message.MediaMimeType,
			"media_filename":   message.MediaFilename,
			"status":           message.Status,
			"wamid":            message.WhatsAppMessageID,
			"created_at":       message.CreatedAt,
			"updated_at":       message.UpdatedAt,
			"metadata":         message.Metadata,
			"is_reply":         message.IsReply,
		}
		if message.IsReply && message.ReplyToMessageID != nil {
			wsPayload["reply_to_message_id"] = message.ReplyToMessageID.String()
			var replyToMsg models.Message
			if err := a.DB.First(&replyToMsg, message.ReplyToMessageID).Error; err == nil {
				wsPayload["reply_to_message"] = map[string]any{
					"id":              replyToMsg.ID.String(),
					"content":         map[string]string{"body": replyToMsg.Content},
					"message_type":    replyToMsg.MessageType,
					"direction":       replyToMsg.Direction,
					"media_url":       replyToMsg.MediaURL,
					"media_mime_type": replyToMsg.MediaMimeType,
					"media_filename":  replyToMsg.MediaFilename,
				}
			}
		}
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type:    websocket.TypeNewMessage,
			Payload: wsPayload,
		})
	}

	if !a.licenseBlocksValueDelivery() {
		a.DispatchWebhook(account.OrganizationID, models.WebhookEventMessageIncoming, MessageEventData{
			MessageID:       message.ID.String(),
			ContactID:       contact.ID.String(),
			ContactPhone:    contact.PhoneNumber,
			ContactName:     contact.ProfileName,
			MessageType:     models.MessageType(msgType),
			Content:         content,
			WhatsAppAccount: account.Name,
			Direction:       models.DirectionIncoming,
		})
	}

	return &message
}
