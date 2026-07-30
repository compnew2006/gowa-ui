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

	// Call transfer types
	TypeCallTransferWaiting    = "call_transfer_waiting"
	TypeCallTransferConnected  = "call_transfer_connected"
	TypeCallTransferCompleted  = "call_transfer_completed"
	TypeCallTransferAbandoned  = "call_transfer_abandoned"
	TypeCallTransferNoAnswer   = "call_transfer_no_answer"
	TypeCallTransferReassigned = "call_transfer_reassigned"

	// Call hold types
	TypeCallHold    = "call_hold"
	TypeCallResumed = "call_resumed"

	// Outgoing call types
	TypeOutgoingCallInitiated = "outgoing_call_initiated"
	TypeOutgoingCallRinging   = "outgoing_call_ringing"
	TypeOutgoingCallAnswered  = "outgoing_call_answered"
	TypeOutgoingCallRejected  = "outgoing_call_rejected"
	TypeOutgoingCallEnded     = "outgoing_call_ended"

	// Call permission types
	TypeCallPermissionUpdate = "call_permission_update"

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
