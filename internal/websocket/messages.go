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
	TypeMessageEdited = "message_edited"
	TypeChatPresence  = "chat_presence"
	TypeContactUpdate = "contact_update"
	TypeSetContact    = "set_contact"
	TypePing          = "ping"
	TypePong          = "pong"

	// Campaign types
	TypeCampaignStatsUpdate = "campaign_stats_update"

	// Permission types
	TypePermissionsUpdated = "permissions_updated"

	// Conversation note types
	TypeConversationNoteCreated = "conversation_note_created"
	TypeConversationNoteUpdated = "conversation_note_updated"
	TypeConversationNoteDeleted = "conversation_note_deleted"

	// Call types
	TypeCallIncoming = "call_incoming"
	TypeCallAnswered = "call_answered"
	TypeCallEnded    = "call_ended"

	TypeChatClaimed        = "chat_claimed"
	TypeChatClosed         = "chat_closed"
	TypeChatReopened       = "chat_reopened"
	TypeChatReleased       = "chat_released"
	TypeCollaboratorJoined = "collaborator_joined"
	TypeCollaboratorLeft   = "collaborator_left"

	// Scheduled maintenance types
	TypeDailyChatReset = "daily_chat_reset"

	// Scheduled message types
	TypeScheduledMessageCreated = "scheduled_message_created"
	TypeScheduledMessageUpdated = "scheduled_message_updated"
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
