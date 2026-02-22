package whatsmeow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
)

type persistMessageOptions struct {
	AllowFromMe   bool
	Broadcast     bool
	HistorySync   bool
	UpdateMetrics bool
}

func (cm *ConnectionManager) persistParsedMessage(
	ctx context.Context,
	client *waClient.Client,
	evt *events.Message,
	instanceID, orgID uuid.UUID,
	opts persistMessageOptions,
) (*models.Message, error) {
	if evt == nil || evt.Message == nil {
		return nil, nil
	}
	if !opts.AllowFromMe && evt.Info.IsFromMe {
		return nil, nil
	}
	if evt.Info.Chat == types.StatusBroadcastJID {
		return nil, nil
	}

	if revokedWAMID, isRevoke := incomingRevokeTargetID(evt.Message); isRevoke {
		cm.applyIncomingRevoke(ctx, orgID, instanceID, revokedWAMID, evt.Info.Timestamp)
		return nil, nil
	}

	chatJID := evt.Info.Chat
	isGroup := evt.Info.IsGroup
	isChannel := chatJID.Server == types.NewsletterServer

	senderPhone := cm.resolveSenderPhone(ctx, client, evt.Info)
	if !isGroup && !isChannel && chatJID.Server == types.DefaultUserServer && chatJID.User != "" {
		// PN JID on direct chats is the canonical contact identity, even when sender is LID-addressed.
		senderPhone = chatJID.User
	}
	if evt.Info.IsFromMe && !isGroup && !isChannel && chatJID.User != "" {
		senderPhone = chatJID.User
	}
	if senderPhone == "" {
		senderPhone = evt.Info.Sender.User
	}
	if !evt.Info.IsFromMe && senderPhone != "" {
		if evt.Info.Sender.User != "" && senderPhone != evt.Info.Sender.User {
			cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, evt.Info.Sender.User, senderPhone)
		}
		if !isGroup && !isChannel && chatJID.Server == types.HiddenUserServer && chatJID.User != "" && senderPhone != chatJID.User {
			cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, chatJID.User, senderPhone)
		}
	}

	contactPushName := evt.Info.PushName
	if evt.Info.IsFromMe && !isGroup && !isChannel {
		contactPushName = ""
	}

	contactDetails := cm.resolveInboundContactDetails(ctx, client, chatJID, senderPhone, contactPushName, isGroup)

	var contactMetadata models.JSONB
	if isGroup {
		contactMetadata = groupContactMetadata(chatJID, contactDetails)
	} else if isChannel {
		contactMetadata = channelContactMetadata(chatJID, contactDetails)
	}

	contact, err := cm.findOrCreateContact(ctx, orgID, instanceID, contactDetails.PhoneNumber, contactDetails.ProfileName, contactMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create contact: %w", err)
	}
	if !evt.Info.IsFromMe && !opts.HistorySync {
		if err := cm.reopenClosedContactOnIncoming(ctx, contact); err != nil {
			cm.logger.Warn("Failed to auto-reopen closed chat on incoming message", "contact_id", contact.ID, "error", err)
		}
	}
	cm.scheduleContactAvatarRefresh(instanceID, contact)

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContent(ctx, client, evt.Message)
	if msgType == models.MessageTypeIgnore {
		return nil, nil
	}
	if msgType == models.MessageTypeText && strings.TrimSpace(content) == "[Unsupported message type]" {
		kinds := strings.Join(incomingMessageKinds(evt.Message), ",")
		if kinds == "" && evt.RawMessage != nil {
			kinds = strings.Join(incomingMessageKinds(evt.RawMessage), ",")
		}
		cm.logger.Warn(
			"Persisting unsupported inbound message payload",
			"instance_id", instanceID,
			"wa_message_id", evt.Info.ID,
			"chat", evt.Info.Chat.String(),
			"kinds", kinds,
		)
	}
	myJID := ""
	myAccount := ""
	if client != nil && client.Store != nil && client.Store.ID != nil {
		myJID = client.Store.ID.String()
		myAccount = client.Store.ID.User
	}
	if myAccount == "" {
		myAccount = "whatsmeow"
	}

	replyCtx := cm.resolveIncomingReplyContext(ctx, orgID, instanceID, chatJID.String(), myJID, evt.Message)

	metadata := models.JSONB{
		"push_name":  evt.Info.PushName,
		"is_group":   isGroup,
		"is_channel": isChannel,
	}
	if isGroup {
		metadata["is_group_chat"] = true
		metadata["group_jid"] = chatJID.String()
		metadata["sender_phone"] = senderPhone
		metadata["sender_push_name"] = evt.Info.PushName

		if contactDetails.GroupName != "" {
			metadata["group_name"] = contactDetails.GroupName
		}
		if contactDetails.GroupTopic != "" {
			metadata["group_topic"] = contactDetails.GroupTopic
		}
	} else if isChannel {
		metadata["is_channel_chat"] = true
		metadata["channel_jid"] = chatJID.String()
		if contactDetails.ChannelName != "" {
			metadata["channel_name"] = contactDetails.ChannelName
		}
		if contactDetails.ChannelDescription != "" {
			metadata["channel_description"] = contactDetails.ChannelDescription
		}
	}
	replyCtx.applyMetadata(metadata)

	direction := models.DirectionIncoming
	status := models.MessageStatusReceived
	if evt.Info.IsFromMe {
		direction = models.DirectionOutgoing
		status = models.MessageStatusSent
	} else if opts.HistorySync {
		status = models.MessageStatusRead
	}

	createdAt := evt.Info.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	waMessageID := strings.TrimSpace(evt.Info.ID)
	if waMessageID != "" {
		var existing models.Message
		err := cm.db.WithContext(ctx).
			Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?", orgID, instanceID, waMessageID).
			Order("created_at DESC").
			First(&existing).Error
		if err == nil {
			return &existing, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed duplicate check for wa_message_id %s: %w", waMessageID, err)
		}
	}

	// Outgoing messages created by this runtime are inserted first as "pending" and
	// may receive their self event before finalizeMessageSend sets wamid. For these,
	// reconcile the event into the existing pending row instead of creating a duplicate.
	if evt.Info.IsFromMe && evt.Info.DeviceSentMeta == nil && !opts.HistorySync && strings.TrimSpace(content) != "" {
		reconciled, err := cm.reconcilePendingOutgoingMessage(ctx, orgID, instanceID, contact.ID, chatJID.String(), msgType, content, waMessageID, createdAt)
		if err != nil {
			return nil, err
		}
		if reconciled != nil {
			reconciled.Contact = contact
			return reconciled, nil
		}
	}

	message := models.Message{
		BaseModel:         models.BaseModel{CreatedAt: createdAt},
		OrganizationID:    orgID,
		InstanceID:        &instanceID,
		WhatsAppAccount:   myAccount,
		ContactID:         contact.ID,
		WhatsAppMessageID: waMessageID,
		ConversationID:    chatJID.String(),
		Direction:         direction,
		MessageType:       msgType,
		Content:           content,
		MediaURL:          mediaURL,
		MediaMimeType:     mimeType,
		MediaFilename:     filename,
		Status:            status,
		IsReply:           replyCtx.IsReply,
		ReplyToMessageID:  replyCtx.ReplyToMessageID,
		Metadata:          metadata,
	}

	if err := cm.db.WithContext(ctx).Create(&message).Error; err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	if opts.UpdateMetrics && direction == models.DirectionIncoming {
		cm.MarkMessageReceived(instanceID)
	}

	preview := content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	if preview == "" {
		preview = "[" + string(msgType) + "]"
	}

	if opts.HistorySync {
		updates := map[string]any{
			"last_message_at":      message.CreatedAt,
			"last_message_preview": preview,
		}
		if direction == models.DirectionIncoming {
			updates["is_read"] = true
		}

		if err := cm.db.WithContext(ctx).
			Model(&models.Contact{}).
			Where("id = ? AND (last_message_at IS NULL OR last_message_at <= ?)", contact.ID, message.CreatedAt).
			Updates(updates).Error; err != nil {
			cm.logger.Warn("Failed to update contact history preview", "contact_id", contact.ID, "error", err)
		}
	} else {
		now := time.Now()
		updates := map[string]any{
			"last_message_at":      &now,
			"last_message_preview": preview,
		}
		if direction == models.DirectionIncoming {
			updates["is_read"] = false
		}
		if err := cm.db.WithContext(ctx).Model(contact).Updates(updates).Error; err != nil {
			cm.logger.Warn("Failed to update contact message preview", "contact_id", contact.ID, "error", err)
		}
	}

	message.Contact = contact

	if opts.Broadcast {
		cm.broadcastPersistedMessage(orgID, &message, contact, isGroup, isChannel, senderPhone, evt.Info.PushName, replyCtx)
	}

	logFields := []interface{}{
		"message_id", message.ID,
		"wa_message_id", message.WhatsAppMessageID,
		"type", msgType,
		"direction", direction,
	}
	if isGroup {
		logFields = append(logFields, "group", chatJID.String())
	} else if isChannel {
		logFields = append(logFields, "channel", chatJID.String())
	}
	cm.logger.Info("Message persisted", logFields...)

	return &message, nil
}

func (cm *ConnectionManager) reconcilePendingOutgoingMessage(
	ctx context.Context,
	orgID, instanceID uuid.UUID,
	contactID uuid.UUID,
	conversationID string,
	msgType models.MessageType,
	content string,
	waMessageID string,
	eventTime time.Time,
) (*models.Message, error) {
	windowStart := eventTime.Add(-2 * time.Minute)

	var pending models.Message
	err := cm.db.WithContext(ctx).
		Where(
			"organization_id = ? AND instance_id = ? AND contact_id = ? AND direction = ? AND message_type = ? AND content = ? AND status = ? AND COALESCE(whats_app_message_id, '') = '' AND created_at >= ?",
			orgID,
			instanceID,
			contactID,
			models.DirectionOutgoing,
			msgType,
			content,
			models.MessageStatusPending,
			windowStart,
		).
		Order("created_at DESC").
		First(&pending).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed pending outgoing reconciliation lookup: %w", err)
	}

	updates := map[string]any{
		"status": models.MessageStatusSent,
	}
	if waMessageID != "" {
		updates["whats_app_message_id"] = waMessageID
		pending.WhatsAppMessageID = waMessageID
	}
	if pending.ConversationID == "" && strings.TrimSpace(conversationID) != "" {
		updates["conversation_id"] = conversationID
		pending.ConversationID = conversationID
	}

	if err := cm.db.WithContext(ctx).Model(&pending).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to reconcile pending outgoing message: %w", err)
	}

	pending.Status = models.MessageStatusSent
	return &pending, nil
}

func (cm *ConnectionManager) broadcastPersistedMessage(
	orgID uuid.UUID,
	message *models.Message,
	contact *models.Contact,
	isGroup bool,
	isChannel bool,
	senderPhone string,
	senderPushName string,
	replyCtx incomingReplyContext,
) {
	if cm.hub == nil || message == nil {
		return
	}

	wsPayload := map[string]any{
		"id":              message.ID,
		"contact_id":      message.ContactID.String(),
		"instance_id":     message.InstanceID.String(),
		"conversation_id": message.ConversationID,
		"is_group_chat":   isGroup,
		"is_channel_chat": isChannel,
		"direction":       message.Direction,
		"message_type":    message.MessageType,
		"content":         map[string]string{"body": message.Content},
		"media_url":       message.MediaURL,
		"media_mime_type": message.MediaMimeType,
		"media_filename":  message.MediaFilename,
		"status":          message.Status,
		"created_at":      message.CreatedAt,
		"updated_at":      message.UpdatedAt,
		"metadata":        message.Metadata,
	}
	if isGroup {
		wsPayload["sender_phone"] = senderPhone
		wsPayload["sender_push_name"] = senderPushName
	}
	if replyCtx.IsReply {
		wsPayload["is_reply"] = true
		if message.ReplyToMessageID != nil {
			wsPayload["reply_to_message_id"] = message.ReplyToMessageID.String()
		}
		if replyCtx.hasPreview() {
			replyType := replyCtx.ReplyPreviewType
			if replyType == "" {
				replyType = models.MessageTypeText
			}
			replyPreviewPayload := map[string]any{
				"content":      map[string]string{"body": replyCtx.ReplyPreviewBody},
				"message_type": replyType,
				"direction":    replyCtx.ReplyDirection,
			}
			if replyCtx.ReplyToMessageID != nil {
				replyPreviewPayload["id"] = replyCtx.ReplyToMessageID.String()
			} else if replyCtx.ReplyToWAMID != "" {
				replyPreviewPayload["id"] = replyCtx.ReplyToWAMID
			}
			if replyCtx.ReplySenderPhone != "" {
				replyPreviewPayload["sender_phone"] = replyCtx.ReplySenderPhone
			}
			wsPayload["reply_to_message"] = replyPreviewPayload
		}
	}
	if contact != nil {
		assignedUserID := ""
		if contact.AssignedUserID != nil {
			assignedUserID = contact.AssignedUserID.String()
		}
		wsPayload["assigned_user_id"] = assignedUserID
		wsPayload["contact_status"] = contact.EffectiveStatus().String()
		wsPayload["profile_name"] = contact.ProfileName
	}

	cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeNewMessage,
		Payload: wsPayload,
	})
}
