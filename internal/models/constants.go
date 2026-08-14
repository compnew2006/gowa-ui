package models

// AuditAction represents the type of audit action
type AuditAction string

const (
	AuditActionCreated AuditAction = "created"
	AuditActionUpdated AuditAction = "updated"
	AuditActionDeleted AuditAction = "deleted"
)

// TeamRole represents a user's role within a specific team (not organizational role)
type TeamRole string

const (
	TeamRoleManager TeamRole = "manager"
	TeamRoleAgent   TeamRole = "agent"
)

// Direction represents message direction
type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// MessageType represents the type of WhatsApp message
type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeVideo       MessageType = "video"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeDocument    MessageType = "document"
	MessageTypeTemplate    MessageType = "template"
	MessageTypeInteractive MessageType = "interactive"
)

// MessageStatus represents the delivery status of a message
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
	MessageStatusReceived  MessageStatus = "received"

	// MessageStatusSending marks a campaign recipient claimed by a worker
	// (pending→sending claim in HandleRecipientJob). Used only on
	// BulkMessageRecipient.Status — chat messages go straight to a terminal
	// status in finalizeMessageSend.
	MessageStatusSending MessageStatus = "sending"

	// MessageStatusRevoked marks a message that was unsent/deleted for
	// everyone (GOWA revoke, "delete for everyone"). Both the inbound
	// message.revoked webhook and the outbound revoke handler set this so
	// the UI renders a consistent "[message revoked]" placeholder.
	MessageStatusRevoked MessageStatus = "revoked"
)

// CampaignStatus represents bulk message campaign states
type CampaignStatus string

const (
	CampaignStatusDraft      CampaignStatus = "draft"
	CampaignStatusScheduled  CampaignStatus = "scheduled"
	CampaignStatusQueued     CampaignStatus = "queued"
	CampaignStatusProcessing CampaignStatus = "processing"
	CampaignStatusPaused     CampaignStatus = "paused"
	CampaignStatusCompleted  CampaignStatus = "completed"
	CampaignStatusCancelled  CampaignStatus = "cancelled"
	CampaignStatusFailed     CampaignStatus = "failed"
)

// AssignmentStrategy represents team assignment strategies
type AssignmentStrategy string

const (
	AssignmentStrategyRoundRobin   AssignmentStrategy = "round_robin"
	AssignmentStrategyLoadBalanced AssignmentStrategy = "load_balanced"
	AssignmentStrategyManual       AssignmentStrategy = "manual"
)

// WebhookEvent represents webhook event types
type WebhookEvent string

const (
	WebhookEventMessageIncoming WebhookEvent = "message.incoming"
	WebhookEventMessageOutgoing WebhookEvent = "message.outgoing"
	WebhookEventMessageSent     WebhookEvent = "message.sent"
	WebhookEventMessageEdited   WebhookEvent = "message.edited"
	WebhookEventContactCreated  WebhookEvent = "contact.created"
)

// ActionType represents custom action types
type ActionType string

const (
	ActionTypeWebhook    ActionType = "webhook"
	ActionTypeURL        ActionType = "url"
	ActionTypeJavascript ActionType = "javascript"
)
