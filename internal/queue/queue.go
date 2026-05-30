package queue

import (
	"context"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

// JobType represents the type of job
type JobType string

const (
	// JobTypeRecipient is for processing a single recipient message
	JobTypeRecipient JobType = "recipient"

	// JobTypeInboundMedia is for async inbound media recovery processing
	JobTypeInboundMedia JobType = "inbound_media"

	// JobTypeContactRepair is for async direct-contact phone repair processing
	JobTypeContactRepair JobType = "contact_repair"

	// JobTypeWhatsAppFilter is for async WhatsApp phone validation filtering
	JobTypeWhatsAppFilter JobType = "whatsapp_filter"

	// JobTypeGroupJoin is for processing a single group join invite link
	JobTypeGroupJoin JobType = "group_join"

	// JobTypeMessageExtraction is for extracting messages from a WhatsApp instance
	JobTypeMessageExtraction JobType = "message_extraction"

	// JobTypeGroupExtraction is for extracting groups from a WhatsApp instance
	JobTypeGroupExtraction JobType = "group_extraction"

	// JobTypeMemberExtraction is for extracting members from a WhatsApp group
	JobTypeMemberExtraction JobType = "member_extraction"
)

// RecipientJob represents a single recipient message job
type RecipientJob struct {
	CampaignID     uuid.UUID    `json:"campaign_id"`
	RecipientID    uuid.UUID    `json:"recipient_id"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	PhoneNumber    string       `json:"phone_number"`
	RecipientName  string       `json:"recipient_name"`
	TemplateParams models.JSONB `json:"template_params"`
	EnqueuedAt     time.Time    `json:"enqueued_at"`
	RecipientType  string       `json:"recipient_type,omitempty"` // "individual" | "group"
	GroupJID       string       `json:"group_jid,omitempty"`      // for group recipients
}

// InboundMediaJob represents an async inbound-media recovery job.
type InboundMediaJob struct {
	MessageID          uuid.UUID          `json:"message_id"`
	OrganizationID     uuid.UUID          `json:"organization_id"`
	InstanceID         uuid.UUID          `json:"instance_id"`
	WhatsAppMessageID  string             `json:"whatsapp_message_id,omitempty"`
	MessageType        models.MessageType `json:"message_type"`
	MediaKind          string             `json:"media_kind"`
	MimeType           string             `json:"mime_type"`
	FallbackFilename   string             `json:"fallback_filename"`
	MediaPayloadBase64 string             `json:"media_payload_base64"`
	LastError          string             `json:"last_error,omitempty"`
	EnqueuedAt         time.Time          `json:"enqueued_at"`
}

// ContactRepairJob represents a background repair for direct contact phone numbers.
type ContactRepairJob struct {
	ContactID      uuid.UUID `json:"contact_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ConversationID string    `json:"conversation_id"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
}

// WhatsAppFilterJob represents a background WhatsApp number validation campaign/batch.
type WhatsAppFilterJob struct {
	BatchID           uuid.UUID  `json:"batch_id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	WhatsAppAccountID uuid.UUID  `json:"whatsapp_account_id"`
	InstanceID        *uuid.UUID `json:"instance_id,omitempty"`
	EnqueuedAt        time.Time  `json:"enqueued_at"`
}

// GroupJoinJob represents a single group join attempt from a campaign.
type GroupJoinJob struct {
	CampaignID      uuid.UUID `json:"campaign_id"`
	RecipientID     uuid.UUID `json:"recipient_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	InstanceID      string    `json:"instance_id"`       // WhatsApp account name to use for joining
	InviteLink      string    `json:"invite_link"`        // The invite link/code to join
	EnqueuedAt      time.Time `json:"enqueued_at"`
}

// MessageExtractionJob represents a message extraction campaign job.
type MessageExtractionJob struct {
	CampaignID     uuid.UUID `json:"campaign_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	InstanceID     uuid.UUID `json:"instance_id"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
}

// GroupExtractionJob represents a group extraction campaign job.
type GroupExtractionJob struct {
	CampaignID     uuid.UUID `json:"campaign_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	InstanceID     uuid.UUID `json:"instance_id"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
}

// MemberExtractionJob represents a member extraction campaign job.
type MemberExtractionJob struct {
	CampaignID     uuid.UUID `json:"campaign_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	InstanceID     uuid.UUID `json:"instance_id"`
	GroupJID       string    `json:"group_jid"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
}

// Queue defines the interface for job queue operations
type Queue interface {
	// EnqueueRecipient adds a single recipient job to the queue
	EnqueueRecipient(ctx context.Context, job *RecipientJob) error

	// EnqueueRecipients adds multiple recipient jobs to the queue
	EnqueueRecipients(ctx context.Context, jobs []*RecipientJob) error

	// EnqueueInboundMedia adds a single inbound-media recovery job to the queue.
	EnqueueInboundMedia(ctx context.Context, job *InboundMediaJob) error

	// EnqueueContactRepair adds a single direct-contact repair job to the queue.
	EnqueueContactRepair(ctx context.Context, job *ContactRepairJob) error

	// EnqueueWhatsAppFilter adds a single WhatsApp filter job to the queue.
	EnqueueWhatsAppFilter(ctx context.Context, job *WhatsAppFilterJob) error

	// EnqueueGroupJoin adds a single group join job to the queue.
	EnqueueGroupJoin(ctx context.Context, job *GroupJoinJob) error

	// EnqueueGroupJoins adds multiple group join jobs to the queue.
	EnqueueGroupJoins(ctx context.Context, jobs []*GroupJoinJob) error

	// EnqueueMessageExtraction adds a message extraction job to the queue.
	EnqueueMessageExtraction(ctx context.Context, job *MessageExtractionJob) error

	// EnqueueGroupExtraction adds a group extraction job to the queue.
	EnqueueGroupExtraction(ctx context.Context, job *GroupExtractionJob) error

	// EnqueueMemberExtraction adds a member extraction job to the queue.
	EnqueueMemberExtraction(ctx context.Context, job *MemberExtractionJob) error

	// Close closes the queue connection
	Close() error
}

// JobHandler handles different job types
type JobHandler interface {
	HandleRecipientJob(ctx context.Context, job *RecipientJob) error
	HandleInboundMediaJob(ctx context.Context, job *InboundMediaJob) error
	HandleContactRepairJob(ctx context.Context, job *ContactRepairJob) error
	HandleWhatsAppFilterJob(ctx context.Context, job *WhatsAppFilterJob) error
	HandleGroupJoinJob(ctx context.Context, job *GroupJoinJob) error
	HandleMessageExtractionJob(ctx context.Context, job *MessageExtractionJob) error
	HandleGroupExtractionJob(ctx context.Context, job *GroupExtractionJob) error
	HandleMemberExtractionJob(ctx context.Context, job *MemberExtractionJob) error
}

// ReadinessGate lets consumers pause before dequeuing the next job.
type ReadinessGate interface {
	WaitUntilOperational(ctx context.Context) error
}

// Consumer defines the interface for consuming jobs from the queue
type Consumer interface {
	// Consume starts consuming jobs from the queue
	// Returns when context is cancelled
	Consume(ctx context.Context, handler JobHandler) error

	// Close closes the consumer connection
	Close() error
}
