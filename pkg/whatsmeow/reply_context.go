package whatsmeow

import (
	"context"
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
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

type incomingReplyContext struct {
	IsReply                   bool
	ReplyToWAMID              string
	ReplyToMessageID          *uuid.UUID
	ReplySenderPhone          string
	ReplyPreviewType          models.MessageType
	ReplyPreviewBody          string
	ReplyDirection            models.Direction
	ReplyPreviewMediaURL      string
	ReplyPreviewMediaMimeType string
	ReplyPreviewMediaFilename string
}

func (rc incomingReplyContext) hasPreview() bool {
	return rc.ReplyPreviewType != "" ||
		strings.TrimSpace(rc.ReplyPreviewBody) != "" ||
		strings.TrimSpace(rc.ReplySenderPhone) != "" ||
		strings.TrimSpace(rc.ReplyPreviewMediaURL) != ""
}

func (rc incomingReplyContext) applyMetadata(metadata models.JSONB) {
	if !rc.IsReply || metadata == nil {
		return
	}
	if rc.ReplyToWAMID != "" {
		metadata[replyMetadataWAMIDKey] = rc.ReplyToWAMID
	}
	if rc.ReplySenderPhone != "" {
		metadata[replyMetadataSenderPhoneKey] = rc.ReplySenderPhone
	}
	if rc.ReplyPreviewType != "" {
		metadata[replyMetadataPreviewTypeKey] = string(rc.ReplyPreviewType)
	}
	if strings.TrimSpace(rc.ReplyPreviewBody) != "" {
		metadata[replyMetadataPreviewBodyKey] = rc.ReplyPreviewBody
	}
	if rc.ReplyDirection != "" {
		metadata[replyMetadataDirectionKey] = string(rc.ReplyDirection)
	}
	if rc.ReplyPreviewMediaURL != "" {
		metadata[replyMetadataPreviewMediaURLKey] = rc.ReplyPreviewMediaURL
	}
	if rc.ReplyPreviewMediaMimeType != "" {
		metadata[replyMetadataPreviewMediaMimeTypeKey] = rc.ReplyPreviewMediaMimeType
	}
	if rc.ReplyPreviewMediaFilename != "" {
		metadata[replyMetadataPreviewMediaFilenameKey] = rc.ReplyPreviewMediaFilename
	}
}

func (cm *ConnectionManager) resolveIncomingReplyContext(
	ctx context.Context,
	orgID, instanceID uuid.UUID,
	conversationID string,
	myJID string,
	msg *waE2E.Message,
) incomingReplyContext {
	contextInfo := incomingReplyContextInfo(msg)
	if contextInfo == nil {
		return incomingReplyContext{}
	}

	replyCtx := incomingReplyContext{
		ReplyDirection: models.DirectionIncoming,
		ReplyToWAMID:   strings.TrimSpace(contextInfo.GetStanzaID()),
	}
	replyCtx.ReplySenderPhone = parseJIDUser(contextInfo.GetParticipant())
	replyCtx.ReplyDirection = inferReplyDirection(replyCtx.ReplySenderPhone, myJID)
	if replyCtx.ReplyDirection == "" {
		replyCtx.ReplyDirection = models.DirectionIncoming
	}

	if quoted := contextInfo.GetQuotedMessage(); quoted != nil {
		replyCtx.ReplyPreviewType, replyCtx.ReplyPreviewBody = cm.extractQuotedMessagePreview(ctx, quoted)
	}

	replyCtx.IsReply = replyCtx.ReplyToWAMID != "" || replyCtx.hasPreview()
	if !replyCtx.IsReply {
		return incomingReplyContext{}
	}

	if cm.db == nil || replyCtx.ReplyToWAMID == "" {
		return replyCtx
	}

	var replyMessage models.Message
	query := cm.db.WithContext(ctx).
		Where("organization_id = ? AND whats_app_message_id = ?", orgID, replyCtx.ReplyToWAMID)

	if instanceID != uuid.Nil {
		query = query.Where("(instance_id = ? OR instance_id IS NULL)", instanceID)
	}
	if strings.TrimSpace(conversationID) != "" {
		query = query.Where("(conversation_id = ? OR conversation_id = '')", strings.TrimSpace(conversationID))
	}

	// Prefer the real companion message over synthetic placeholders when duplicates share one WAMID.
	query = query.Order(`
		CASE
			WHEN message_type = 'text' AND TRIM(content) IN ('[Unsupported message type]', '(This message was deleted)', 'This message was deleted') THEN 1
			ELSE 0
		END ASC
	`).Order("created_at DESC")

	err := query.First(&replyMessage).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			cm.logger.Warn("Failed to resolve quoted message", "error", err, "reply_to_wamid", replyCtx.ReplyToWAMID)
		}
		status, statusErr := cm.resolveReplyStatusByWAMID(ctx, orgID, instanceID, replyCtx.ReplyToWAMID)
		if statusErr != nil {
			cm.logger.Warn("Failed to resolve quoted status", "error", statusErr, "reply_to_wamid", replyCtx.ReplyToWAMID)
			return replyCtx
		}
		if status != nil {
			if replyCtx.ReplyPreviewType == "" {
				replyCtx.ReplyPreviewType = mapStatusTypeToReplyMessageType(status.StatusType)
			}
			if strings.TrimSpace(replyCtx.ReplyPreviewBody) == "" {
				replyCtx.ReplyPreviewBody = strings.TrimSpace(status.Content)
			}
			replyCtx.ReplyPreviewMediaURL = resolveReplyStatusMediaURL(*status)
			replyCtx.ReplyPreviewMediaMimeType = strings.TrimSpace(status.MediaMimeType)
			replyCtx.ReplyPreviewMediaFilename = strings.TrimSpace(status.MediaFilename)
		}
		return replyCtx
	}

	replyCtx.ReplyToMessageID = &replyMessage.ID
	replyCtx.ReplyDirection = replyMessage.Direction
	if replyCtx.ReplyPreviewType == "" {
		replyCtx.ReplyPreviewType = replyMessage.MessageType
	}
	if strings.TrimSpace(replyCtx.ReplyPreviewBody) == "" {
		replyCtx.ReplyPreviewBody = replyMessage.Content
	}
	if replyCtx.ReplySenderPhone == "" {
		replyCtx.ReplySenderPhone = metadataString(replyMessage.Metadata, "sender_phone")
	}
	replyCtx.ReplyPreviewMediaURL = strings.TrimSpace(replyMessage.MediaURL)
	replyCtx.ReplyPreviewMediaMimeType = strings.TrimSpace(replyMessage.MediaMimeType)
	replyCtx.ReplyPreviewMediaFilename = strings.TrimSpace(replyMessage.MediaFilename)

	return replyCtx
}

func (cm *ConnectionManager) resolveReplyStatusByWAMID(ctx context.Context, orgID, instanceID uuid.UUID, wamid string) (*models.WhatsAppStatus, error) {
	if cm == nil || cm.db == nil {
		return nil, nil
	}
	wamid = strings.TrimSpace(wamid)
	if wamid == "" {
		return nil, nil
	}

	query := cm.db.WithContext(ctx).Where("organization_id = ? AND whats_app_message_id = ?", orgID, wamid)
	if instanceID != uuid.Nil {
		query = query.Where("instance_id = ?", instanceID)
	}

	var status models.WhatsAppStatus
	err := query.Order("created_at DESC").First(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func resolveReplyStatusMediaURL(status models.WhatsAppStatus) string {
	mediaURL := strings.TrimSpace(status.MediaURL)
	if mediaURL == "" {
		return ""
	}

	lowerMediaURL := strings.ToLower(mediaURL)
	if strings.HasPrefix(lowerMediaURL, "http://") ||
		strings.HasPrefix(lowerMediaURL, "https://") ||
		strings.HasPrefix(lowerMediaURL, "data:") {
		return mediaURL
	}
	if strings.HasPrefix(mediaURL, "/api/statuses/") {
		return mediaURL
	}
	return "/api/statuses/" + status.ID.String() + "/media"
}

func mapStatusTypeToReplyMessageType(statusType models.WhatsAppStatusType) models.MessageType {
	switch statusType {
	case models.WhatsAppStatusTypeImage:
		return models.MessageTypeImage
	case models.WhatsAppStatusTypeVideo:
		return models.MessageTypeVideo
	default:
		return models.MessageTypeText
	}
}

func incomingReplyContextInfo(msg *waE2E.Message) *waE2E.ContextInfo {
	unwrapped := unwrapIncomingMessage(msg)
	if unwrapped == nil {
		return nil
	}

	switch {
	case unwrapped.GetExtendedTextMessage() != nil:
		return unwrapped.GetExtendedTextMessage().GetContextInfo()
	case unwrapped.GetImageMessage() != nil:
		return unwrapped.GetImageMessage().GetContextInfo()
	case unwrapped.GetVideoMessage() != nil:
		return unwrapped.GetVideoMessage().GetContextInfo()
	case unwrapped.GetAudioMessage() != nil:
		return unwrapped.GetAudioMessage().GetContextInfo()
	case unwrapped.GetDocumentMessage() != nil:
		return unwrapped.GetDocumentMessage().GetContextInfo()
	case unwrapped.GetStickerMessage() != nil:
		return unwrapped.GetStickerMessage().GetContextInfo()
	case unwrapped.GetLocationMessage() != nil:
		return unwrapped.GetLocationMessage().GetContextInfo()
	case unwrapped.GetLiveLocationMessage() != nil:
		return unwrapped.GetLiveLocationMessage().GetContextInfo()
	case unwrapped.GetContactMessage() != nil:
		return unwrapped.GetContactMessage().GetContextInfo()
	case unwrapped.GetContactsArrayMessage() != nil:
		return unwrapped.GetContactsArrayMessage().GetContextInfo()
	case unwrapped.GetTemplateButtonReplyMessage() != nil:
		return unwrapped.GetTemplateButtonReplyMessage().GetContextInfo()
	case unwrapped.GetButtonsResponseMessage() != nil:
		return unwrapped.GetButtonsResponseMessage().GetContextInfo()
	case unwrapped.GetListResponseMessage() != nil:
		return unwrapped.GetListResponseMessage().GetContextInfo()
	case unwrapped.GetListMessage() != nil:
		return unwrapped.GetListMessage().GetContextInfo()
	case unwrapped.GetInteractiveResponseMessage() != nil:
		return unwrapped.GetInteractiveResponseMessage().GetContextInfo()
	case unwrapped.GetPollCreationMessage() != nil:
		return unwrapped.GetPollCreationMessage().GetContextInfo()
	case unwrapped.GetPollCreationMessageV2() != nil:
		return unwrapped.GetPollCreationMessageV2().GetContextInfo()
	case unwrapped.GetPollCreationMessageV3() != nil:
		return unwrapped.GetPollCreationMessageV3().GetContextInfo()
	case unwrapped.GetPollCreationMessageV5() != nil:
		return unwrapped.GetPollCreationMessageV5().GetContextInfo()
	default:
		return nil
	}
}

func (cm *ConnectionManager) extractQuotedMessagePreview(ctx context.Context, msg *waE2E.Message) (models.MessageType, string) {
	unwrapped := unwrapIncomingMessage(msg)
	if unwrapped == nil {
		return models.MessageTypeText, ""
	}

	if msgType, content, ok := cm.extractTextualIncomingMessage(ctx, nil, unwrapped); ok {
		return msgType, content
	}

	switch {
	case unwrapped.GetImageMessage() != nil:
		return models.MessageTypeImage, strings.TrimSpace(unwrapped.GetImageMessage().GetCaption())
	case unwrapped.GetVideoMessage() != nil:
		return models.MessageTypeVideo, strings.TrimSpace(unwrapped.GetVideoMessage().GetCaption())
	case unwrapped.GetAudioMessage() != nil:
		return models.MessageTypeAudio, ""
	case unwrapped.GetDocumentMessage() != nil:
		return models.MessageTypeDocument, strings.TrimSpace(unwrapped.GetDocumentMessage().GetCaption())
	case unwrapped.GetStickerMessage() != nil:
		return models.MessageTypeSticker, ""
	default:
		return models.MessageTypeText, "[Unsupported message type]"
	}
}

func parseJIDUser(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if at := strings.Index(trimmed, "@"); at > 0 {
		trimmed = trimmed[:at]
	}
	if colon := strings.Index(trimmed, ":"); colon > 0 {
		trimmed = trimmed[:colon]
	}
	return strings.TrimSpace(trimmed)
}

func inferReplyDirection(senderJID string, myJID string) models.Direction {
	myUser := parseJIDUser(myJID)
	senderUser := parseJIDUser(senderJID)
	if myUser != "" && senderUser != "" && myUser == senderUser {
		return models.DirectionOutgoing
	}
	return models.DirectionIncoming
}
