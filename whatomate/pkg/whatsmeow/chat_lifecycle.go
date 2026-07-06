package whatsmeow

import (
	"context"

	"github.com/compnew2006/whatomate/internal/models"
)

// reopenClosedContactOnIncoming moves a closed conversation back to pending queue on inbound activity.
func (cm *ConnectionManager) reopenClosedContactOnIncoming(
	ctx context.Context,
	contact *models.Contact,
	msgType models.MessageType,
	content string,
) error {
	if contact == nil {
		return nil
	}
	if contact.EffectiveStatus() != models.ChatStatusClosed {
		return nil
	}
	if cm.shouldSkipClosedChatAutoReopenForIncomingMessage(ctx, contact.OrganizationID, contact, msgType, content) {
		return nil
	}

	if err := cm.db.WithContext(ctx).Model(contact).Updates(map[string]any{
		"status":            models.ChatStatusPending,
		"assigned_user_id":  nil,
		"closed_at":         nil,
		"closed_by_user_id": nil,
	}).Error; err != nil {
		return err
	}

	if err := cm.db.WithContext(ctx).Model(&models.ChatClosureRating{}).
		Where("contact_id = ? AND state = ?", contact.ID, models.ChatClosureRatingStatePending).
		Update("state", models.ChatClosureRatingStateExpired).Error; err != nil {
		cm.logger.Error("Failed to expire pending close rating cycles on contact reopen", "error", err, "contact_id", contact.ID)
	}

	contact.Status = models.ChatStatusPending
	contact.AssignedUserID = nil
	contact.ClosedAt = nil
	contact.ClosedByUserID = nil
	contact.ClosedByUser = nil
	return nil
}
