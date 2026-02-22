package whatsmeow

import (
	"context"

	"github.com/compnew2006/whatomate/internal/models"
)

// reopenClosedContactOnIncoming moves a closed conversation back to pending queue on inbound activity.
func (cm *ConnectionManager) reopenClosedContactOnIncoming(ctx context.Context, contact *models.Contact) error {
	if contact == nil {
		return nil
	}
	if contact.EffectiveStatus() != models.ChatStatusClosed {
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

	contact.Status = models.ChatStatusPending
	contact.AssignedUserID = nil
	contact.ClosedAt = nil
	contact.ClosedByUserID = nil
	contact.ClosedByUser = nil
	return nil
}
