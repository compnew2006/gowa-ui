package whatsmeow

import (
	"context"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
)

// handleMessage processes incoming messages.
func (cm *ConnectionManager) handleMessage(ctx context.Context, evt *events.Message, instanceID, orgID uuid.UUID) {
	allowFromMe := false
	if evt.Info.IsFromMe {
		// Keep ignoring self-origin messages emitted by this runtime to avoid
		// duplicate persistence for dashboard/API sends. DeviceSentMeta indicates
		// the message came from another linked device (e.g. mobile), which should
		// be synchronized into the chat thread.
		if evt.Info.DeviceSentMeta == nil {
			return
		}
		allowFromMe = true
	}
	if evt.Info.Chat == types.StatusBroadcastJID {
		return
	}

	client := cm.GetClient(instanceID)
	if client == nil || client.Store == nil {
		cm.logger.Error("Client not found or bot not logged in", "instance_id", instanceID)
		cm.MarkError(instanceID)
		return
	}
	normalizedEvt := cm.normalizeIncomingEventMessage(ctx, client, evt, instanceID)
	if normalizedEvt == nil {
		return
	}
	if _, err := cm.persistParsedMessage(ctx, client, normalizedEvt, instanceID, orgID, persistMessageOptions{
		AllowFromMe:   allowFromMe,
		Broadcast:     true,
		HistorySync:   false,
		UpdateMetrics: true,
	}); err != nil {
		cm.logger.Error("Failed to persist incoming message", "error", err, "instance_id", instanceID)
		cm.MarkMessageFailed(instanceID)
	}
}

func (cm *ConnectionManager) normalizeIncomingEventMessage(
	ctx context.Context,
	client *waClient.Client,
	evt *events.Message,
	instanceID uuid.UUID,
) *events.Message {
	if evt == nil {
		return nil
	}

	baseMessage := evt.Message
	if baseMessage == nil {
		baseMessage = evt.RawMessage
	}
	if baseMessage == nil {
		return nil
	}

	normalizedMessage := baseMessage
	if unwrapped := unwrapIncomingMessage(normalizedMessage); unwrapped != nil {
		normalizedMessage = unwrapped
	}
	if normalizedMessage.GetEncCommentMessage() == nil && evt.RawMessage != nil && evt.RawMessage.GetEncCommentMessage() != nil {
		normalizedMessage = evt.RawMessage
		if unwrapped := unwrapIncomingMessage(normalizedMessage); unwrapped != nil {
			normalizedMessage = unwrapped
		}
	}

	decryptSource := *evt
	decryptSource.Message = normalizedMessage

	if normalizedMessage.GetEncCommentMessage() != nil {
		if client == nil {
			cm.logger.Warn("Skipping encrypted comment: client unavailable", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
			return nil
		}
		decryptedComment, err := client.DecryptComment(ctx, &decryptSource)
		if err != nil {
			cm.logger.Warn("Failed to decrypt encrypted comment", "instance_id", instanceID, "wa_message_id", evt.Info.ID, "error", err)
			return nil
		}
		normalizedMessage = decryptedComment
		decryptSource.Message = normalizedMessage
	}

	if normalizedMessage.GetSecretEncryptedMessage() != nil {
		if client == nil {
			cm.logger.Warn("Skipping secret-encrypted message: client unavailable", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
			return nil
		}
		decryptedMessage, err := client.DecryptSecretEncryptedMessage(ctx, &decryptSource)
		if err != nil {
			cm.logger.Warn("Failed to decrypt secret-encrypted message", "instance_id", instanceID, "wa_message_id", evt.Info.ID, "error", err)
			return nil
		}
		normalizedMessage = decryptedMessage
	}

	if normalizedMessage.GetPollUpdateMessage() != nil {
		cm.logger.Debug("Skipping poll-update event", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
		return nil
	}
	if normalizedMessage.GetKeepInChatMessage() != nil {
		cm.logger.Debug("Skipping keep-in-chat control event", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
		return nil
	}

	normalized := *evt
	normalized.Message = normalizedMessage
	return &normalized
}

func (cm *ConnectionManager) findOrCreateContact(
	ctx context.Context,
	orgID, instanceID uuid.UUID,
	phoneNumber, profileName string,
	metadata models.JSONB,
) (*models.Contact, error) {
	var contact models.Contact
	err := cm.db.WithContext(ctx).
		Where("organization_id = ? AND phone_number = ? AND instance_id = ?", orgID, phoneNumber, instanceID).
		First(&contact).Error
	if err == nil {
		updates := map[string]any{}

		if profileName != "" {
			shouldUpdateProfileName := contact.ProfileName == "" ||
				contact.ProfileName == contact.PhoneNumber ||
				isGroupContactMetadata(metadata) ||
				isChannelContactMetadata(metadata)
			if shouldUpdateProfileName && contact.ProfileName != profileName {
				updates["profile_name"] = profileName
				contact.ProfileName = profileName
			}
		}

		if mergedMetadata, changed := mergeJSONB(contact.Metadata, metadata); changed {
			updates["metadata"] = mergedMetadata
			contact.Metadata = mergedMetadata
		}

		if len(updates) > 0 {
			if err := cm.db.WithContext(ctx).Model(&contact).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		return &contact, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Backward-compatibility path:
	// adopt legacy records created before per-instance contact mapping (instance_id IS NULL).
	err = cm.db.WithContext(ctx).
		Where("organization_id = ? AND phone_number = ? AND instance_id IS NULL", orgID, phoneNumber).
		First(&contact).Error
	if err == nil {
		updates := map[string]any{
			"instance_id": instanceID,
		}

		instID := instanceID
		contact.InstanceID = &instID

		if profileName != "" {
			shouldUpdateProfileName := contact.ProfileName == "" ||
				contact.ProfileName == contact.PhoneNumber ||
				isGroupContactMetadata(metadata) ||
				isChannelContactMetadata(metadata)
			if shouldUpdateProfileName && contact.ProfileName != profileName {
				updates["profile_name"] = profileName
				contact.ProfileName = profileName
			}
		}

		if mergedMetadata, changed := mergeJSONB(contact.Metadata, metadata); changed {
			updates["metadata"] = mergedMetadata
			contact.Metadata = mergedMetadata
		}

		if err := cm.db.WithContext(ctx).Model(&contact).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &contact, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	instID := instanceID
	contact = models.Contact{
		OrganizationID: orgID,
		InstanceID:     &instID,
		ProfileName:    profileName,
		PhoneNumber:    phoneNumber,
		Metadata:       metadata,
	}
	if contact.ProfileName == "" {
		contact.ProfileName = phoneNumber
	}

	if err := cm.db.WithContext(ctx).Create(&contact).Error; err != nil {
		// Concurrent message delivery can race on contact creation.
		// Re-fetch the winner to keep inbound processing idempotent.
		if fetchErr := cm.db.WithContext(ctx).
			Where("organization_id = ? AND phone_number = ? AND instance_id = ?", orgID, phoneNumber, instanceID).
			First(&contact).Error; fetchErr == nil {
			return &contact, nil
		}
		return nil, err
	}

	return &contact, nil
}

func (cm *ConnectionManager) extractMessageContent(ctx context.Context, client *waClient.Client, msg *waE2E.Message) (models.MessageType, string, string, string, string) {
	return cm.extractMessageContentWithMedia(ctx, client, msg)
}
