package whatsmeow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
)

func (cm *ConnectionManager) createInstanceNotification(ctx context.Context, orgID, instanceID uuid.UUID, eventType, message string) (*models.InstanceNotification, error) {
	notification := &models.InstanceNotification{
		OrganizationID: orgID,
		InstanceID:     instanceID,
		EventType:      eventType,
		Message:        message,
		IsDismissed:    false,
	}
	if err := cm.db.WithContext(ctx).Create(notification).Error; err != nil {
		return nil, err
	}
	return notification, nil
}

func (cm *ConnectionManager) broadcastInstanceNotification(orgID uuid.UUID, notification *models.InstanceNotification) {
	if cm.hub == nil || notification == nil {
		return
	}
	cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeInstanceNotification,
		Payload: websocket.InstanceNotificationPayload{
			ID:         notification.ID.String(),
			InstanceID: notification.InstanceID.String(),
			EventType:  notification.EventType,
			Message:    notification.Message,
			CreatedAt:  notification.CreatedAt.Format(time.RFC3339),
		},
	})
}
