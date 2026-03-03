package whatsmeow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/websocket"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	defaultInboundMediaAsyncMaxAttempts = 4
	defaultInboundMediaAsyncBaseBackoff = 5 * time.Second
	defaultInboundMediaAsyncMaxBackoff  = 60 * time.Second
)

// ProcessInboundMediaRecoveryJob retries failed inbound media downloads asynchronously.
func (cm *ConnectionManager) ProcessInboundMediaRecoveryJob(ctx context.Context, job *queue.InboundMediaJob) error {
	if cm == nil {
		return queue.NewPermanentError(fmt.Errorf("connection manager is nil"))
	}
	if cm.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if job == nil {
		return queue.NewPermanentError(fmt.Errorf("inbound media job is nil"))
	}
	if strings.TrimSpace(job.MediaPayloadBase64) == "" {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing media payload"))
	}

	var message models.Message
	if err := cm.db.WithContext(ctx).
		Select("id", "organization_id", "contact_id", "instance_id", "message_type", "media_url", "media_mime_type", "media_filename", "metadata", "error_message", "updated_at").
		Where("id = ? AND organization_id = ?", job.MessageID, job.OrganizationID).
		First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return queue.NewPermanentError(fmt.Errorf("message %s not found in organization %s", job.MessageID, job.OrganizationID))
		}
		return fmt.Errorf("failed to load inbound media target message: %w", err)
	}
	if message.InstanceID == nil {
		return queue.NewPermanentError(fmt.Errorf("message %s has no instance_id", message.ID))
	}
	if strings.TrimSpace(message.MediaURL) != "" {
		return nil
	}

	downloadable, msgType, err := decodeInboundMediaPayload(job)
	if err != nil {
		return queue.NewPermanentError(err)
	}

	client := cm.GetClient(job.InstanceID)
	if client == nil {
		return fmt.Errorf("whatsmeow client is unavailable for instance %s", job.InstanceID)
	}

	maxAttempts, baseBackoff, maxBackoff := cm.inboundMediaAsyncRetrySettings()
	var (
		data            []byte
		lastFailureText = strings.TrimSpace(job.LastError)
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		data, err = client.Download(ctx, downloadable)
		if err == nil && len(data) > 0 {
			break
		}

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			lastFailureText = err.Error()
			if attempt >= maxAttempts || !shouldRetryInboundMediaDownload(err) {
				cm.markInboundMediaRecoveryFailed(ctx, &message, lastFailureText)
				return nil
			}

			backoff := inboundMediaDownloadBackoff(attempt, baseBackoff, maxBackoff)
			if waitErr := sleepWithInboundMediaContext(ctx, backoff); waitErr != nil {
				if errors.Is(waitErr, context.Canceled) || errors.Is(waitErr, context.DeadlineExceeded) {
					return waitErr
				}
				lastFailureText = waitErr.Error()
				cm.markInboundMediaRecoveryFailed(ctx, &message, lastFailureText)
				return nil
			}
			continue
		}

		lastFailureText = "inbound media download returned empty data"
		if attempt >= maxAttempts {
			cm.markInboundMediaRecoveryFailed(ctx, &message, lastFailureText)
			return nil
		}

		backoff := inboundMediaDownloadBackoff(attempt, baseBackoff, maxBackoff)
		if waitErr := sleepWithInboundMediaContext(ctx, backoff); waitErr != nil {
			if errors.Is(waitErr, context.Canceled) || errors.Is(waitErr, context.DeadlineExceeded) {
				return waitErr
			}
			lastFailureText = waitErr.Error()
			cm.markInboundMediaRecoveryFailed(ctx, &message, lastFailureText)
			return nil
		}
	}

	resolvedMimeType := strings.TrimSpace(job.MimeType)
	if resolvedMimeType == "" {
		resolvedMimeType = strings.TrimSpace(message.MediaMimeType)
	}
	resolvedFilename := strings.TrimSpace(job.FallbackFilename)
	if resolvedFilename == "" {
		resolvedFilename = strings.TrimSpace(message.MediaFilename)
	}

	relPath, err := cm.persistInboundMedia(data, msgType, resolvedMimeType, resolvedFilename)
	if err != nil {
		return fmt.Errorf("failed to persist recovered inbound media: %w", err)
	}

	if err := cm.applyInboundMediaRecoverySuccess(ctx, &message, relPath, resolvedMimeType, resolvedFilename); err != nil {
		return err
	}

	return nil
}

func (cm *ConnectionManager) inboundMediaAsyncRetrySettings() (int, time.Duration, time.Duration) {
	maxAttempts := defaultInboundMediaAsyncMaxAttempts
	baseBackoff := defaultInboundMediaAsyncBaseBackoff
	maxBackoff := defaultInboundMediaAsyncMaxBackoff

	if cm != nil && cm.cfg != nil {
		if cm.cfg.InboundMediaAsyncRetryCount > 0 {
			maxAttempts = cm.cfg.InboundMediaAsyncRetryCount
		}
		if cm.cfg.InboundMediaAsyncRetryDelayMs > 0 {
			baseBackoff = time.Duration(cm.cfg.InboundMediaAsyncRetryDelayMs) * time.Millisecond
		}
		if cm.cfg.InboundMediaAsyncRetryMaxDelayMs > 0 {
			maxBackoff = time.Duration(cm.cfg.InboundMediaAsyncRetryMaxDelayMs) * time.Millisecond
		}
	}

	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if baseBackoff < 0 {
		baseBackoff = 0
	}
	if maxBackoff < baseBackoff {
		maxBackoff = baseBackoff
	}

	return maxAttempts, baseBackoff, maxBackoff
}

func decodeInboundMediaPayload(job *queue.InboundMediaJob) (waClient.DownloadableMessage, models.MessageType, error) {
	rawPayload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(job.MediaPayloadBase64))
	if err != nil {
		return nil, "", fmt.Errorf("invalid inbound media payload encoding: %w", err)
	}
	if len(rawPayload) == 0 {
		return nil, "", fmt.Errorf("decoded inbound media payload is empty")
	}

	kind := strings.ToLower(strings.TrimSpace(job.MediaKind))
	switch kind {
	case "image":
		var media waE2E.ImageMessage
		if err := proto.Unmarshal(rawPayload, &media); err != nil {
			return nil, "", fmt.Errorf("invalid image media payload: %w", err)
		}
		return &media, models.MessageTypeImage, nil
	case "sticker":
		var media waE2E.StickerMessage
		if err := proto.Unmarshal(rawPayload, &media); err != nil {
			return nil, "", fmt.Errorf("invalid sticker media payload: %w", err)
		}
		return &media, models.MessageTypeSticker, nil
	case "video":
		var media waE2E.VideoMessage
		if err := proto.Unmarshal(rawPayload, &media); err != nil {
			return nil, "", fmt.Errorf("invalid video media payload: %w", err)
		}
		return &media, models.MessageTypeVideo, nil
	case "audio":
		var media waE2E.AudioMessage
		if err := proto.Unmarshal(rawPayload, &media); err != nil {
			return nil, "", fmt.Errorf("invalid audio media payload: %w", err)
		}
		return &media, models.MessageTypeAudio, nil
	case "document":
		var media waE2E.DocumentMessage
		if err := proto.Unmarshal(rawPayload, &media); err != nil {
			return nil, "", fmt.Errorf("invalid document media payload: %w", err)
		}
		return &media, models.MessageTypeDocument, nil
	default:
		return nil, "", fmt.Errorf("unsupported inbound media kind %q", job.MediaKind)
	}
}

func (cm *ConnectionManager) applyInboundMediaRecoverySuccess(
	ctx context.Context,
	message *models.Message,
	mediaURL string,
	mimeType string,
	filename string,
) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	nextMetadata := cloneJSONBMap(message.Metadata)
	if nextMetadata == nil {
		nextMetadata = models.JSONB{}
	}
	nextMetadata[inboundMediaAsyncStatusKey] = inboundMediaAsyncStatusSucceeded
	nextMetadata[inboundMediaAsyncRecoveredAtKey] = time.Now().UTC().Format(time.RFC3339Nano)
	nextMetadata[inboundMediaAsyncLastErrorKey] = nil
	delete(nextMetadata, inboundMediaAsyncEnqueueErrorKey)

	now := time.Now().UTC()
	if err := cm.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", message.ID).
		Updates(map[string]any{
			"media_url":       mediaURL,
			"media_mime_type": mimeType,
			"media_filename":  filename,
			"metadata":        nextMetadata,
			"error_message":   "",
			"updated_at":      now,
		}).Error; err != nil {
		return fmt.Errorf("failed to update recovered inbound media message: %w", err)
	}

	message.MediaURL = mediaURL
	message.MediaMimeType = mimeType
	message.MediaFilename = filename
	message.Metadata = nextMetadata
	message.ErrorMessage = ""
	message.UpdatedAt = now

	cm.broadcastInboundMediaRecovered(message)
	return nil
}

func (cm *ConnectionManager) markInboundMediaRecoveryFailed(ctx context.Context, message *models.Message, reason string) {
	if cm == nil || cm.db == nil || message == nil {
		return
	}

	nextMetadata := cloneJSONBMap(message.Metadata)
	if nextMetadata == nil {
		nextMetadata = models.JSONB{}
	}
	nextMetadata[inboundMediaAsyncStatusKey] = inboundMediaAsyncStatusFailed
	nextMetadata[inboundMediaAsyncLastErrorKey] = strings.TrimSpace(reason)
	nextMetadata[inboundMediaAsyncRecoveredAtKey] = nil

	nextErrorMessage := fmt.Sprintf("Inbound media async recovery failed: %s", strings.TrimSpace(reason))
	if strings.TrimSpace(reason) == "" {
		nextErrorMessage = "Inbound media async recovery failed"
	}

	if err := cm.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", message.ID).
		Updates(map[string]any{
			"metadata":      nextMetadata,
			"error_message": nextErrorMessage,
		}).Error; err != nil {
		cm.logger.Warn("Failed to persist inbound media async failure marker", "message_id", message.ID, "error", err)
		return
	}

	message.Metadata = nextMetadata
	message.ErrorMessage = nextErrorMessage
}

func (cm *ConnectionManager) broadcastInboundMediaRecovered(message *models.Message) {
	if cm == nil || cm.hub == nil || message == nil {
		return
	}

	cm.hub.BroadcastToOrg(message.OrganizationID, websocket.WSMessage{
		Type: websocket.TypeMessageMediaUpdated,
		Payload: map[string]any{
			"id":              message.ID,
			"contact_id":      message.ContactID.String(),
			"media_url":       message.MediaURL,
			"media_mime_type": message.MediaMimeType,
			"media_filename":  message.MediaFilename,
			"error_message":   message.ErrorMessage,
			"metadata":        message.Metadata,
			"updated_at":      message.UpdatedAt,
		},
	})
}
