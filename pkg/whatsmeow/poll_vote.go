package whatsmeow

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// handlePollVote decrypts and persists a poll vote event as a reply message
// linked to the original poll message.
func (cm *ConnectionManager) handlePollVote(
	ctx context.Context,
	client *waClient.Client,
	evt *events.Message,
	pollUpdate *waE2E.PollUpdateMessage,
	instanceID, orgID uuid.UUID,
) {
	pollCreationKey := pollUpdate.GetPollCreationMessageKey()
	if pollCreationKey == nil {
		cm.logger.Debug("Poll vote missing creation message key", "instance_id", instanceID)
		return
	}

	originalWAMID := pollCreationKey.GetID()
	if originalWAMID == "" {
		return
	}

	vote, err := client.DecryptPollVote(ctx, evt)
	if err != nil {
		cm.logger.Warn("Failed to decrypt poll vote", "instance_id", instanceID, "wa_message_id", evt.Info.ID, "error", err)
		return
	}

	// Find the original poll message to resolve selected option names.
	selectedNames := cm.resolveSelectedOptionNames(ctx, orgID, instanceID, originalWAMID, vote.GetSelectedOptions())

	// Find or create the contact for the voter.
	senderPhone := cm.resolveSenderPhone(ctx, client, evt.Info)
	if senderPhone == "" {
		senderPhone = evt.Info.Sender.User
	}
	contact, err := cm.findOrCreateContact(ctx, orgID, instanceID, senderPhone, evt.Info.PushName, nil)
	if err != nil {
		cm.logger.Warn("Failed to find/create contact for poll vote", "error", err)
		return
	}

	// Find the original poll message to link as reply.
	var originalMsg models.Message
	err = cm.db.WithContext(ctx).
		Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?", orgID, instanceID, originalWAMID).
		First(&originalMsg).Error

	isReply := err == nil
	var replyToID *uuid.UUID
	if isReply {
		replyToID = &originalMsg.ID
	}

	content := "Voted: " + strings.Join(selectedNames, ", ")
	if content == "Voted: " {
		content = "Voted"
	}

	myAccount := "whatsmeow"
	if client.Store != nil && client.Store.ID != nil && client.Store.ID.User != "" {
		myAccount = client.Store.ID.User
	}

	chatJID := evt.Info.Chat
	conversationID := chatJID.String()
	if !evt.Info.IsGroup && chatJID.Server != "newsletter" {
		conversationID = directConversationID(chatJID, senderPhone)
	}

	createdAt := evt.Info.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	metadata := models.JSONB{
		"is_group":  evt.Info.IsGroup,
		"push_name": evt.Info.PushName,
		"poll_vote": true,
	}
	if evt.Info.IsGroup {
		metadata["group_jid"] = chatJID.String()
		metadata["sender_phone"] = senderPhone
	}

	message := models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: createdAt},
		OrganizationID:    orgID,
		InstanceID:        &instanceID,
		WhatsAppAccount:   myAccount,
		ContactID:         contact.ID,
		WhatsAppMessageID: strings.TrimSpace(evt.Info.ID),
		ConversationID:    conversationID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypePoll,
		Content:           content,
		Status:            models.MessageStatusReceived,
		IsReply:           isReply,
		ReplyToMessageID:  replyToID,
		Metadata:          metadata,
		InteractiveData: models.JSONB{
			"type":             "poll_vote",
			"selected_options": selectedNames,
			"poll_message_id":  originalWAMID,
		},
	}

	if err := cm.db.WithContext(ctx).Create(&message).Error; err != nil {
		cm.logger.Error("Failed to persist poll vote message", "error", err, "instance_id", instanceID)
	}

	cm.logger.Debug("Persisted poll vote",
		"instance_id", instanceID,
		"poll_wa_id", originalWAMID,
		"selected", selectedNames,
	)
}

// resolveSelectedOptionNames matches hashed selected options back to original
// poll option names by looking up the original poll's InteractiveData.
func (cm *ConnectionManager) resolveSelectedOptionNames(
	ctx context.Context,
	orgID, instanceID uuid.UUID,
	originalWAMID string,
	selectedHashes [][]byte,
) []string {
	if len(selectedHashes) == 0 {
		return nil
	}

	var originalMsg models.Message
	err := cm.db.WithContext(ctx).
		Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?", orgID, instanceID, originalWAMID).
		First(&originalMsg).Error
	if err != nil {
		// Can't resolve names without original poll; return hex hashes as fallback.
		names := make([]string, len(selectedHashes))
		for i, h := range selectedHashes {
			names[i] = hex.EncodeToString(h)[:12]
		}
		return names
	}

	optionsRaw, ok := originalMsg.InteractiveData["options"]
	if !ok {
		return nil
	}
	optionsSlice, ok := optionsRaw.([]interface{})
	if !ok {
		return nil
	}

	selectedSet := make(map[string]bool, len(selectedHashes))
	for _, h := range selectedHashes {
		selectedSet[fmt.Sprintf("%x", h)] = true
	}

	var names []string
	for _, opt := range optionsSlice {
		optStr, ok := opt.(string)
		if !ok {
			continue
		}
		hashes := waClient.HashPollOptions([]string{optStr})
		for _, h := range hashes {
			hexHash := fmt.Sprintf("%x", h)
			if selectedSet[hexHash] {
				names = append(names, optStr)
				break
			}
		}
	}
	return names
}
