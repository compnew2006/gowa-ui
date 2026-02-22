package whatsmeow

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
)

func (cm *ConnectionManager) broadcastInstanceConnected(orgID, instanceID uuid.UUID, phoneNumber string) {
	if cm.hub == nil {
		return
	}

	cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeInstanceConnected,
		Payload: websocket.InstancePayload{
			InstanceID:  instanceID.String(),
			PhoneNumber: strings.TrimSpace(phoneNumber),
			Status:      string(models.InstanceStatusConnected),
		},
	})
}
