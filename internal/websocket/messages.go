package websocket

import "github.com/google/uuid"

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Message types
const (
	TypeAuth          = "auth"
	TypeNewMessage    = "new_message"
	TypeStatusUpdate  = "status_update"
	TypeContactUpdate = "contact_update"
	TypeSetContact    = "set_contact"
	TypePing          = "ping"
	TypePong          = "pong"

	// Agent transfer types
	TypeAgentTransfer       = "agent_transfer"
	TypeAgentTransferResume = "agent_transfer_resume"
	TypeAgentTransferAssign = "agent_transfer_assign"

	// Campaign types
	TypeCampaignStatsUpdate = "campaign_stats_update"

	// Permission types
	TypePermissionsUpdated = "permissions_updated"

	// Conversation note types
	TypeConversationNoteCreated = "conversation_note_created"
	TypeConversationNoteUpdated = "conversation_note_updated"
	TypeConversationNoteDeleted = "conversation_note_deleted"

	// Instance types
	TypeInstanceQRCode          = "instance_qr_code"
	TypeInstanceConnected       = "instance_connected"
	TypeInstanceDisconnected    = "instance_disconnected"
	TypeInstanceBanned          = "instance_banned"
	TypeInstanceLoggedOut       = "instance_logged_out"
	TypeInstanceQRTimeout       = "instance_qr_timeout"
	TypeInstanceReconnectFailed = "instance_reconnect_failed"
	TypeInstanceNotification    = "instance_notification"
)

// BroadcastMessage represents a message to be broadcast to clients
type BroadcastMessage struct {
	OrgID     uuid.UUID
	UserID    uuid.UUID // Optional: only send to specific user
	ContactID uuid.UUID // Optional: only send to users viewing this contact
	Message   WSMessage
}

// AuthPayload is the payload for auth messages from client
type AuthPayload struct {
	Token string `json:"token"`
}

// SetContactPayload is the payload for set_contact messages from client
type SetContactPayload struct {
	ContactID string `json:"contact_id"`
}

// StatusUpdatePayload is the payload for status_update messages
type StatusUpdatePayload struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// QRCodePayload is the payload for instance_qr_code messages
type QRCodePayload struct {
	InstanceID string `json:"instance_id"`
	QRCode     string `json:"qr_code"` // base64
	TimeoutSec int    `json:"timeout_seconds"`
}

// InstancePayload is the payload for instance status events
type InstancePayload struct {
	InstanceID  string `json:"instance_id"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Status      string `json:"status"`
}

// InstanceNotificationPayload is the payload for instance_notification events.
type InstanceNotificationPayload struct {
	ID         string `json:"id"`
	InstanceID string `json:"instance_id"`
	EventType  string `json:"event_type"`
	Message    string `json:"message"`
	CreatedAt  string `json:"created_at"`
}

// InstanceReconnectFailedPayload is the payload for instance_reconnect_failed events.
type InstanceReconnectFailedPayload struct {
	InstanceID string `json:"instance_id"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
}
