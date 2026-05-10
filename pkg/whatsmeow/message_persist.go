package whatsmeow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type persistMessageOptions struct {
	AllowFromMe   bool
	Broadcast     bool
	HistorySync   bool
	UpdateMetrics bool
}

const (
	inboundMediaAsyncStatusKey         = "inbound_media_async_status"
	inboundMediaAsyncEnqueuedAtKey     = "inbound_media_async_enqueued_at"
	inboundMediaAsyncLastErrorKey      = "inbound_media_async_last_error"
	inboundMediaAsyncEnqueueErrorKey   = "inbound_media_async_enqueue_error"
	inboundMediaAsyncRecoveredAtKey    = "inbound_media_async_recovered_at"
	inboundMediaAsyncStatusQueued      = "queued"
	inboundMediaAsyncStatusEnqueueFail = "enqueue_failed"
	inboundMediaAsyncStatusSucceeded   = "succeeded"
	inboundMediaAsyncStatusFailed      = "failed"
)

var wamidPhonePattern = regexp.MustCompile(`\d{10,15}`)

func shouldMigrateLIDContact(senderIdentity string) bool {
	senderIdentity = strings.TrimSpace(senderIdentity)
	return senderIdentity != "" && !strings.Contains(senderIdentity, "@")
}

func normalizeDirectSenderIdentity(senderIdentity string, chatJID, senderJID types.JID) string {
	senderIdentity = strings.TrimSpace(senderIdentity)

	// For direct chats, a PN chat JID is always canonical.
	if chatJID.Server == types.DefaultUserServer && chatJID.User != "" {
		return chatJID.User
	}

	if senderIdentity != "" {
		if strings.Contains(senderIdentity, "@") {
			return senderIdentity
		}
		// Hidden IDs can look like phone numbers; preserve their JID server suffix
		// so they aren't treated as canonical phone numbers.
		if chatJID.Server == types.HiddenUserServer && chatJID.User == senderIdentity {
			return chatJID.String()
		}
		if senderJID.Server == types.HiddenUserServer && senderJID.User == senderIdentity {
			return senderJID.String()
		}
		return senderIdentity
	}

	if senderJID.Server == types.DefaultUserServer && senderJID.User != "" {
		return senderJID.User
	}
	if chatJID.Server == types.HiddenUserServer && chatJID.User != "" {
		return chatJID.String()
	}
	if senderJID.Server == types.HiddenUserServer && senderJID.User != "" {
		return senderJID.String()
	}
	return ""
}

func inferPhoneFromWAMID(wamid string) string {
	wamid = strings.TrimSpace(wamid)
	const prefix = "wamid."
	if wamid == "" || !strings.HasPrefix(strings.ToLower(wamid), prefix) {
		return ""
	}

	payload := strings.TrimSpace(wamid[len(prefix):])
	if payload == "" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return ""
		}
	}

	matches := wamidPhonePattern.FindAllString(string(decoded), -1)
	longest := ""
	for _, candidate := range matches {
		if len(candidate) > len(longest) {
			longest = candidate
		}
	}
	return longest
}

func directConversationID(chatJID types.JID, peerIdentity string) string {
	peerIdentity = strings.TrimSpace(peerIdentity)
	if peerIdentity == "" {
		return chatJID.String()
	}
	if strings.Contains(peerIdentity, "@") {
		return peerIdentity
	}
	return peerIdentity + "@s.whatsapp.net"
}

func (cm *ConnectionManager) resolveInstanceWhatsAppAccount(ctx context.Context, instanceID uuid.UUID) string {
	if cm == nil || cm.db == nil || instanceID == uuid.Nil {
		return ""
	}

	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).
		Select("phone_number", "jid").
		Where("id = ?", instanceID).
		First(&instance).Error; err != nil {
		return ""
	}

	if account := strings.TrimSpace(instance.PhoneNumber); account != "" {
		return account
	}
	if jid := strings.TrimSpace(instance.JID); jid != "" {
		if parsed, err := types.ParseJID(jid); err == nil && parsed.User != "" {
			return parsed.User
		}
	}
	return ""
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
	if isStatusMessageInfo(evt.Info) {
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
	if evt.Info.IsFromMe && !isGroup && !isChannel {
		recipientAlt := evt.Info.RecipientAlt.ToNonAD()
		if recipientAlt.Server == types.DefaultUserServer && recipientAlt.User != "" {
			senderPhone = recipientAlt.User
		} else if chatJID.User != "" {
			senderPhone = chatJID.User
		}
	}
	if senderPhone == "" {
		senderJID := evt.Info.Sender.ToNonAD()
		recipientAlt := evt.Info.RecipientAlt.ToNonAD()
		switch {
		case !isGroup && !isChannel && chatJID.Server == types.DefaultUserServer && chatJID.User != "":
			senderPhone = chatJID.User
		case evt.Info.IsFromMe && recipientAlt.Server == types.DefaultUserServer && recipientAlt.User != "":
			senderPhone = recipientAlt.User
		case senderJID.Server == types.DefaultUserServer && senderJID.User != "":
			senderPhone = senderJID.User
		case !isGroup && !isChannel && chatJID.Server == types.HiddenUserServer && chatJID.User != "":
			senderPhone = chatJID.String()
		case senderJID.Server == types.HiddenUserServer && senderJID.User != "":
			senderPhone = senderJID.String()
		}
	}
	if !isGroup && !isChannel {
		senderPhone = normalizeDirectSenderIdentity(senderPhone, chatJID, evt.Info.Sender.ToNonAD())
	}
	if evt.Info.IsFromMe && !isGroup && !isChannel && strings.HasSuffix(senderPhone, "@"+string(types.HiddenUserServer)) {
		if resolvedPN := cm.lookupInstancePhoneByJID(ctx, orgID, senderPhone); resolvedPN != "" {
			senderPhone = resolvedPN
		}
	}
	lidSenderIdentity := ""
	if !isGroup && !isChannel && strings.HasSuffix(senderPhone, "@"+string(types.HiddenUserServer)) {
		lidSenderIdentity = senderPhone
		if inferredPhone := inferPhoneFromWAMID(evt.Info.ID); inferredPhone != "" {
			senderPhone = inferredPhone
		}
	}
	if !evt.Info.IsFromMe {
		if lidSenderIdentity != "" && senderPhone != "" && senderPhone != lidSenderIdentity {
			cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, lidSenderIdentity, senderPhone)
		}
	}
	if evt.Info.IsFromMe && !isGroup && !isChannel && chatJID.Server == types.HiddenUserServer && chatJID.User != "" && shouldMigrateLIDContact(senderPhone) {
		cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, chatJID.String(), senderPhone)
	}
	if !evt.Info.IsFromMe && shouldMigrateLIDContact(senderPhone) {
		if evt.Info.Sender.User != "" && senderPhone != evt.Info.Sender.User {
			cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, evt.Info.Sender.User, senderPhone)
		}
		if !isGroup && !isChannel && chatJID.Server == types.HiddenUserServer && chatJID.User != "" && senderPhone != chatJID.User {
			cm.migrateContactPhoneFromLID(ctx, orgID, instanceID, chatJID.User, senderPhone)
		}
	}
	if !isGroup && !isChannel {
		cm.logger.Debug(
			"Resolved direct chat peer identity",
			"instance_id", instanceID,
			"from_me", evt.Info.IsFromMe,
			"chat_jid", chatJID.String(),
			"sender_jid", evt.Info.Sender.ToNonAD().String(),
			"sender_alt", evt.Info.SenderAlt.ToNonAD().String(),
			"recipient_alt", evt.Info.RecipientAlt.ToNonAD().String(),
			"resolved_peer", senderPhone,
		)
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
	cm.scheduleContactAvatarRefresh(instanceID, contact)

	msgType, content, mimeType, filename, downloadable := cm.extractMessageContentMetadata(ctx, client, evt.Message)
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
		myAccount = cm.resolveInstanceWhatsAppAccount(ctx, instanceID)
	}
	if myAccount == "" {
		myAccount = "whatsmeow"
	}
	if strings.TrimSpace(contact.WhatsAppAccount) != myAccount {
		if err := cm.db.WithContext(ctx).
			Model(&models.Contact{}).
			Where("id = ?", contact.ID).
			Update("whats_app_account", myAccount).Error; err != nil {
			cm.logger.Warn("Failed to backfill contact account from instance", "contact_id", contact.ID, "error", err)
		} else {
			contact.WhatsAppAccount = myAccount
		}
	}
	if !evt.Info.IsFromMe && !opts.HistorySync {
		if err := cm.reopenClosedContactOnIncoming(ctx, contact, msgType, content); err != nil {
			cm.logger.Warn("Failed to auto-reopen closed chat on incoming message", "contact_id", contact.ID, "error", err)
		}
	}

	replyCtx := cm.resolveIncomingReplyContext(ctx, orgID, instanceID, chatJID.String(), myJID, evt.Message)
	conversationID := chatJID.String()
	if !isGroup && !isChannel {
		conversationID = directConversationID(chatJID, contactDetails.PhoneNumber)
	}

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

	messageID := uuid.New()
	mediaURL := ""
	var mediaAssetID *uuid.UUID
	var mediaRetryArtifact *inboundMediaRetryArtifact
	if downloadable != nil {
		if cm.mediaService == nil {
			lastErr := "media service is not configured"
			if client == nil {
				lastErr = waClient.ErrClientIsNil.Error()
			}
			mediaRetryArtifact = cm.buildInboundMediaRetryArtifact(msgType, downloadable, mimeType, filename, lastErr)
		} else {
			handledMedia, mediaErr := cm.mediaService.HandleIncomingMedia(WithMediaInstanceID(ctx, instanceID), evt)
			if mediaErr != nil {
				cm.logger.Warn(
					"Inline inbound media handling failed",
					"instance_id", instanceID,
					"wa_message_id", waMessageID,
					"error", mediaErr,
				)
				mediaRetryArtifact = cm.buildInboundMediaRetryArtifact(msgType, downloadable, mimeType, filename, mediaErr.Error())
			} else if handledMedia != nil {
				mediaAssetID = &handledMedia.MediaAssetID
				mediaURL = buildMessageMediaURL(messageID)
				mimeType = coalesceMediaValue(handledMedia.MimeType, mimeType)
				filename = coalesceMediaValue(handledMedia.Filename, filename)
			}
		}
	}

	message := models.Message{
		BaseModel:         models.BaseModel{ID: messageID, CreatedAt: createdAt},
		OrganizationID:    orgID,
		InstanceID:        &instanceID,
		WhatsAppAccount:   myAccount,
		ContactID:         contact.ID,
		WhatsAppMessageID: waMessageID,
		ConversationID:    conversationID,
		Direction:         direction,
		MessageType:       msgType,
		Content:           content,
		MediaAssetID:      mediaAssetID,
		MediaURL:          mediaURL,
		MediaMimeType:     mimeType,
		MediaFilename:     filename,
		Status:            status,
		IsReply:           replyCtx.IsReply,
		ReplyToMessageID:  replyCtx.ReplyToMessageID,
		Metadata:          metadata,
	}

	result := cm.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "whats_app_message_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{
			clause.Neq{Column: clause.Column{Name: "whats_app_message_id"}, Value: ""},
		}},
		DoNothing: true,
	}).Create(&message)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		cm.logger.Info("Duplicate message ignored (ON CONFLICT)", "whats_app_message_id", waMessageID)
		return nil, nil
	}

	if direction == models.DirectionIncoming && mediaRetryArtifact != nil {
		recoveryJob, enqueueErr := buildInboundMediaRecoveryJob(&message, mediaRetryArtifact)
		if enqueueErr == nil {
			enqueueErr = cm.enqueueInboundMediaRecovery(ctx, recoveryJob)
		}

		nextMetadata := cloneJSONBMap(message.Metadata)
		if nextMetadata == nil {
			nextMetadata = models.JSONB{}
		}
		if recoveryJob != nil {
			setInboundMediaAsyncJobMetadata(nextMetadata, recoveryJob)
		}
		nextMetadata[inboundMediaAsyncLastErrorKey] = strings.TrimSpace(mediaRetryArtifact.LastError)
		nextMetadata[inboundMediaAsyncRecoveredAtKey] = nil

		var nextErrorMessage string
		if enqueueErr == nil {
			nextMetadata[inboundMediaAsyncStatusKey] = inboundMediaAsyncStatusQueued
			nextMetadata[inboundMediaAsyncEnqueuedAtKey] = time.Now().UTC().Format(time.RFC3339Nano)
			delete(nextMetadata, inboundMediaAsyncEnqueueErrorKey)
			nextErrorMessage = "Inbound media download failed inline; queued for async recovery"
		} else {
			nextMetadata[inboundMediaAsyncStatusKey] = inboundMediaAsyncStatusEnqueueFail
			nextMetadata[inboundMediaAsyncEnqueueErrorKey] = enqueueErr.Error()
			nextErrorMessage = fmt.Sprintf("Inbound media download failed inline and async enqueue failed: %v", enqueueErr)
		}

		if err := cm.db.WithContext(ctx).
			Model(&models.Message{}).
			Where("id = ?", message.ID).
			Updates(map[string]any{
				"metadata":      nextMetadata,
				"error_message": nextErrorMessage,
			}).Error; err != nil {
			cm.logger.Warn("Failed to persist inbound media async recovery marker", "message_id", message.ID, "error", err)
		} else {
			message.Metadata = nextMetadata
			message.ErrorMessage = nextErrorMessage
		}
	}

	if direction == models.DirectionIncoming {
		cm.maybeCaptureChatCloseRating(ctx, orgID, contact, &message)
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
			"whats_app_account":    myAccount,
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
			"whats_app_account":    myAccount,
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

func buildInboundMediaRecoveryJob(message *models.Message, artifact *inboundMediaRetryArtifact) (*queue.InboundMediaJob, error) {
	if message == nil {
		return nil, fmt.Errorf("message is nil")
	}
	if artifact == nil {
		return nil, fmt.Errorf("inbound media retry artifact is nil")
	}
	if message.InstanceID == nil || *message.InstanceID == uuid.Nil {
		return nil, fmt.Errorf("message %s has no instance id for inbound media recovery", message.ID)
	}
	if strings.TrimSpace(artifact.MediaPayloadBase64) == "" {
		return nil, fmt.Errorf("inbound media retry artifact is missing media payload")
	}
	if strings.TrimSpace(artifact.MediaKind) == "" {
		return nil, fmt.Errorf("inbound media retry artifact is missing media kind")
	}

	return &queue.InboundMediaJob{
		MessageID:          message.ID,
		OrganizationID:     message.OrganizationID,
		InstanceID:         *message.InstanceID,
		WhatsAppMessageID:  strings.TrimSpace(message.WhatsAppMessageID),
		MessageType:        message.MessageType,
		MediaKind:          strings.TrimSpace(artifact.MediaKind),
		MimeType:           strings.TrimSpace(artifact.MimeType),
		FallbackFilename:   strings.TrimSpace(artifact.FallbackFilename),
		MediaPayloadBase64: strings.TrimSpace(artifact.MediaPayloadBase64),
		LastError:          strings.TrimSpace(artifact.LastError),
	}, nil
}

func (cm *ConnectionManager) enqueueInboundMediaRecovery(ctx context.Context, job *queue.InboundMediaJob) error {
	if cm == nil {
		return fmt.Errorf("connection manager is nil")
	}
	if cm.inboundMediaQueue == nil {
		return fmt.Errorf("inbound media queue is not configured")
	}
	if job == nil {
		return fmt.Errorf("inbound media recovery job is nil")
	}

	if err := cm.inboundMediaQueue.EnqueueInboundMedia(ctx, job); err != nil {
		return fmt.Errorf("failed to enqueue inbound media recovery job for message %s: %w", job.MessageID, err)
	}

	return nil
}

func buildMessageMediaURL(messageID uuid.UUID) string {
	return "/api/media/" + messageID.String()
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
		"id":               message.ID,
		"contact_id":       message.ContactID.String(),
		"instance_id":      message.InstanceID.String(),
		"conversation_id":  message.ConversationID,
		"is_group_chat":    isGroup,
		"is_channel_chat":  isChannel,
		"direction":        message.Direction,
		"message_type":     message.MessageType,
		"content":          map[string]string{"body": message.Content},
		"media_url":        message.MediaURL,
		"media_mime_type":  message.MediaMimeType,
		"media_filename":   message.MediaFilename,
		"status":           message.Status,
		"whatsapp_account": message.WhatsAppAccount,
		"created_at":       message.CreatedAt,
		"updated_at":       message.UpdatedAt,
		"metadata":         message.Metadata,
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
			if replyCtx.ReplyPreviewMediaURL != "" {
				replyPreviewPayload["media_url"] = replyCtx.ReplyPreviewMediaURL
			}
			if replyCtx.ReplyPreviewMediaMimeType != "" {
				replyPreviewPayload["media_mime_type"] = replyCtx.ReplyPreviewMediaMimeType
			}
			if replyCtx.ReplyPreviewMediaFilename != "" {
				replyPreviewPayload["media_filename"] = replyCtx.ReplyPreviewMediaFilename
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
