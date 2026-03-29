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

	// Close closes the queue connection
	Close() error
}

// JobHandler handles different job types
type JobHandler interface {
	HandleRecipientJob(ctx context.Context, job *RecipientJob) error
	HandleInboundMediaJob(ctx context.Context, job *InboundMediaJob) error
	HandleContactRepairJob(ctx context.Context, job *ContactRepairJob) error
}

// Consumer defines the interface for consuming jobs from the queue
type Consumer interface {
	// Consume starts consuming jobs from the queue
	// Returns when context is cancelled
	Consume(ctx context.Context, handler JobHandler) error

	// Close closes the consumer connection
	Close() error
}
