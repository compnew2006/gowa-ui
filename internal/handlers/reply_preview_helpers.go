package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	replyMetadataWAMIDKey                = "reply_to_wamid"
	replyMetadataSenderPhoneKey          = "reply_sender_phone"
	replyMetadataPreviewBodyKey          = "reply_preview_body"
	replyMetadataPreviewTypeKey          = "reply_preview_type"
	replyMetadataDirectionKey            = "reply_direction"
	replyMetadataPreviewMediaURLKey      = "reply_preview_media_url"
	replyMetadataPreviewMediaMimeTypeKey = "reply_preview_media_mime_type"
	replyMetadataPreviewMediaFilenameKey = "reply_preview_media_filename"
)

func buildReplyPreviewFromMetadata(db *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID, metadata models.JSONB) *ReplyPreview {
	if metadata == nil {
		return nil
	}

	previewTypeRaw := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewTypeKey))
	previewBody := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewBodyKey))
	senderPhone := strings.TrimSpace(messageMetadataString(metadata, replyMetadataSenderPhoneKey))
	replyWAMID := strings.TrimSpace(messageMetadataString(metadata, replyMetadataWAMIDKey))
	directionRaw := strings.TrimSpace(messageMetadataString(metadata, replyMetadataDirectionKey))
	previewMediaURL := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewMediaURLKey))
	previewMediaMimeType := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewMediaMimeTypeKey))
	previewMediaFilename := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewMediaFilenameKey))

	if previewTypeRaw == "" && previewBody == "" && senderPhone == "" && previewMediaURL == "" && replyWAMID == "" {
		return nil
	}

	previewType := models.MessageType(previewTypeRaw)
	needsStatusLookup := previewTypeRaw == "" || (previewMediaURL == "" && previewType == models.MessageTypeImage)
	if needsStatusLookup {
		if status := findStatusByWAMID(db, orgID, instanceID, replyWAMID); status != nil {
			if previewTypeRaw == "" {
				previewType = mapStatusTypeToMessageType(status.StatusType)
			}
			if previewBody == "" {
				previewBody = strings.TrimSpace(status.Content)
			}
			if previewMediaURL == "" {
				previewMediaURL = resolveStatusMediaURL(*status)
			}
			if previewMediaMimeType == "" {
				previewMediaMimeType = strings.TrimSpace(status.MediaMimeType)
			}
			if previewMediaFilename == "" {
				previewMediaFilename = strings.TrimSpace(status.MediaFilename)
			}
		}
	}
	if previewType == "" {
		previewType = models.MessageTypeText
	}

	direction := models.DirectionIncoming
	if directionRaw == string(models.DirectionOutgoing) {
		direction = models.DirectionOutgoing
	}

	content := map[string]string{"body": normalizeDeletedMessageBody(previewBody)}
	return &ReplyPreview{
		ID:            replyWAMID,
		Content:       content,
		MessageType:   previewType,
		Direction:     direction,
		SenderPhone:   senderPhone,
		MediaURL:      previewMediaURL,
		MediaMimeType: previewMediaMimeType,
		MediaFilename: previewMediaFilename,
	}
}

func findStatusByWAMID(db *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID, wamid string) *models.WhatsAppStatus {
	if db == nil {
		return nil
	}
	wamid = strings.TrimSpace(wamid)
	if wamid == "" {
		return nil
	}

	query := db.Where("organization_id = ? AND whats_app_message_id = ?", orgID, wamid)
	if instanceID != nil {
		query = query.Where("instance_id = ?", *instanceID)
	}

	var status models.WhatsAppStatus
	if err := query.Order("created_at DESC").First(&status).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return nil
	}
	return &status
}

func mapStatusTypeToMessageType(statusType models.WhatsAppStatusType) models.MessageType {
	switch statusType {
	case models.WhatsAppStatusTypeImage:
		return models.MessageTypeImage
	case models.WhatsAppStatusTypeVideo:
		return models.MessageTypeVideo
	default:
		return models.MessageTypeText
	}
}
