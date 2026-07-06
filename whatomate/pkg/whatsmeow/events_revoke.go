package whatsmeow

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const legacyDeletedMessageCaption = "This message was deleted"

func appendDeletedCaption(content string) string {
	trimmed := strings.TrimSpace(content)
	switch trimmed {
	case "":
		return deletedMessageCaption
	case deletedMessageCaption, legacyDeletedMessageCaption:
		return deletedMessageCaption
	case "[Unsupported message type]":
		return deletedMessageCaption
	}

	if strings.Contains(content, deletedMessageCaption) {
		return content
	}

	return strings.TrimRight(content, "\n") + "\n" + deletedMessageCaption
}

func cloneJSONBMap(src models.JSONB) models.JSONB {
	dst := make(models.JSONB, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// applyIncomingRevoke updates the originally sent message instead of creating a new revoke row.
// Returns true when the revoke event has been consumed and should not continue normal handling.
func (cm *ConnectionManager) applyIncomingRevoke(ctx context.Context, orgID, instanceID uuid.UUID, revokedMessageID string, eventTime time.Time) bool {
	revokedMessageID = strings.TrimSpace(revokedMessageID)
	if revokedMessageID == "" {
		cm.logger.Info("Skipping revoke event without target message ID", "instance_id", instanceID)
		return true
	}

	var target models.Message
	err := cm.db.WithContext(ctx).
		Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?", orgID, instanceID, revokedMessageID).
		Order("created_at DESC").
		First(&target).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cm.logger.Info("Revoke target message not found; ignoring revoke event", "instance_id", instanceID, "revoked_wamid", revokedMessageID)
			return true
		}
		cm.logger.Error("Failed to find revoke target message", "error", err, "instance_id", instanceID, "revoked_wamid", revokedMessageID)
		return true
	}

	updatedContent := appendDeletedCaption(target.Content)
	updatedMetadata := cloneJSONBMap(target.Metadata)
	if updatedMetadata == nil {
		updatedMetadata = models.JSONB{}
	}
	updatedMetadata["revoked"] = true
	updatedMetadata["revoked_at"] = eventTime.UTC().Format(time.RFC3339)

	now := time.Now()
	if err := cm.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", target.ID).
		Updates(map[string]any{
			"content":    updatedContent,
			"metadata":   updatedMetadata,
			"updated_at": now,
		}).Error; err != nil {
		cm.logger.Error("Failed to apply revoke update", "error", err, "message_id", target.ID, "revoked_wamid", revokedMessageID)
		return true
	}

	target.Content = updatedContent
	target.Metadata = updatedMetadata
	target.UpdatedAt = now

	preview := updatedContent
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	cm.db.WithContext(ctx).
		Model(&models.Contact{}).
		Where("id = ? AND organization_id = ? AND (last_message_at IS NULL OR last_message_at <= ?)", target.ContactID, orgID, target.CreatedAt).
		Update("last_message_preview", preview)

	if cm.hub != nil {
		wsPayload := map[string]any{
			"id":              target.ID,
			"contact_id":      target.ContactID.String(),
			"direction":       target.Direction,
			"message_type":    target.MessageType,
			"content":         map[string]string{"body": target.Content},
			"media_url":       target.MediaURL,
			"media_mime_type": target.MediaMimeType,
			"media_filename":  target.MediaFilename,
			"status":          target.Status,
			"created_at":      target.CreatedAt,
			"updated_at":      target.UpdatedAt,
			"metadata":        target.Metadata,
		}
		cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type:    websocket.TypeNewMessage,
			Payload: wsPayload,
		})
	}

	cm.logger.Info("Applied revoke update to existing message", "message_id", target.ID, "revoked_wamid", revokedMessageID, "contact_id", target.ContactID)
	return true
}
