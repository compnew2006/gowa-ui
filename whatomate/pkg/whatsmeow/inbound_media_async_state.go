package whatsmeow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
)

const inboundMediaAsyncJobKey = "inbound_media_async_job"

func setInboundMediaAsyncJobMetadata(metadata models.JSONB, job *queue.InboundMediaJob) {
	if metadata == nil || job == nil {
		return
	}

	metadata[inboundMediaAsyncJobKey] = models.JSONB{
		"message_id":           job.MessageID.String(),
		"organization_id":      job.OrganizationID.String(),
		"instance_id":          job.InstanceID.String(),
		"whatsapp_message_id":  strings.TrimSpace(job.WhatsAppMessageID),
		"message_type":         string(job.MessageType),
		"media_kind":           strings.TrimSpace(job.MediaKind),
		"mime_type":            strings.TrimSpace(job.MimeType),
		"fallback_filename":    strings.TrimSpace(job.FallbackFilename),
		"media_payload_base64": strings.TrimSpace(job.MediaPayloadBase64),
		"last_error":           strings.TrimSpace(job.LastError),
		"enqueued_at":          job.EnqueuedAt,
	}
}

func clearInboundMediaAsyncJobMetadata(metadata models.JSONB) {
	if metadata == nil {
		return
	}
	delete(metadata, inboundMediaAsyncJobKey)
}

func updateInboundMediaAsyncJobLastError(metadata models.JSONB, reason string) {
	if metadata == nil {
		return
	}

	job, err := decodeInboundMediaAsyncJobMetadata(metadata[inboundMediaAsyncJobKey])
	if err != nil || job == nil {
		return
	}
	job.LastError = strings.TrimSpace(reason)
	setInboundMediaAsyncJobMetadata(metadata, job)
}

func decodeInboundMediaAsyncJobMetadata(value any) (*queue.InboundMediaJob, error) {
	if value == nil {
		return nil, fmt.Errorf("missing inbound media async job metadata")
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal inbound media async job metadata: %w", err)
	}

	var job queue.InboundMediaJob
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, fmt.Errorf("decode inbound media async job metadata: %w", err)
	}

	if strings.TrimSpace(job.MediaPayloadBase64) == "" {
		return nil, fmt.Errorf("inbound media async job metadata missing media_payload_base64")
	}
	if strings.TrimSpace(job.MediaKind) == "" {
		return nil, fmt.Errorf("inbound media async job metadata missing media_kind")
	}

	return &job, nil
}
