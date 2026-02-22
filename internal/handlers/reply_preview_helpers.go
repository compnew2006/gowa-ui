package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	replyMetadataWAMIDKey       = "reply_to_wamid"
	replyMetadataSenderPhoneKey = "reply_sender_phone"
	replyMetadataPreviewBodyKey = "reply_preview_body"
	replyMetadataPreviewTypeKey = "reply_preview_type"
	replyMetadataDirectionKey   = "reply_direction"
)

func buildReplyPreviewFromMetadata(metadata models.JSONB) *ReplyPreview {
	if metadata == nil {
		return nil
	}

	previewTypeRaw := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewTypeKey))
	previewBody := strings.TrimSpace(messageMetadataString(metadata, replyMetadataPreviewBodyKey))
	senderPhone := strings.TrimSpace(messageMetadataString(metadata, replyMetadataSenderPhoneKey))
	replyWAMID := strings.TrimSpace(messageMetadataString(metadata, replyMetadataWAMIDKey))
	directionRaw := strings.TrimSpace(messageMetadataString(metadata, replyMetadataDirectionKey))

	if previewTypeRaw == "" && previewBody == "" && senderPhone == "" {
		return nil
	}

	previewType := models.MessageType(previewTypeRaw)
	if previewType == "" {
		previewType = models.MessageTypeText
	}

	direction := models.DirectionIncoming
	if directionRaw == string(models.DirectionOutgoing) {
		direction = models.DirectionOutgoing
	}

	content := map[string]string{"body": normalizeDeletedMessageBody(previewBody)}
	return &ReplyPreview{
		ID:          replyWAMID,
		Content:     content,
		MessageType: previewType,
		Direction:   direction,
		SenderPhone: senderPhone,
	}
}
