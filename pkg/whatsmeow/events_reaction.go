package whatsmeow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleAnyReaction consumes plain and encrypted reaction events.
// Returns true when the incoming message event was handled as a reaction.
func (cm *ConnectionManager) handleAnyReaction(ctx context.Context, evt *events.Message, instanceID, orgID uuid.UUID) bool {
	if evt == nil {
		return false
	}

	msg := evt.Message
	if msg == nil {
		msg = evt.RawMessage
	}
	if unwrapped := unwrapIncomingMessage(msg); unwrapped != nil {
		msg = unwrapped
	}
	if msg == nil {
		return false
	}

	if reaction := msg.GetReactionMessage(); reaction != nil {
		cm.handleReactionPayload(ctx, evt.Info, reaction, instanceID, orgID)
		return true
	}

	if msg.GetEncReactionMessage() == nil && evt.RawMessage != nil && evt.RawMessage.GetEncReactionMessage() != nil {
		msg = evt.RawMessage
	}
	if unwrapped := unwrapIncomingMessage(msg); unwrapped != nil {
		msg = unwrapped
	}

	if msg.GetEncReactionMessage() == nil {
		return false
	}

	client := cm.GetClient(instanceID)
	if client == nil {
		cm.logger.Warn("Skipping encrypted reaction: client unavailable", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
		return true
	}

	decryptSource := *evt
	decryptSource.Message = msg

	reaction, err := client.DecryptReaction(ctx, &decryptSource)
	if err != nil {
		cm.logger.Warn("Failed to decrypt encrypted reaction", "instance_id", instanceID, "wa_message_id", evt.Info.ID, "error", err)
		return true
	}

	cm.handleReactionPayload(ctx, evt.Info, reaction, instanceID, orgID)
	return true
}

func (cm *ConnectionManager) handleReactionPayload(
	ctx context.Context,
	info types.MessageInfo,
	reaction *waE2E.ReactionMessage,
	instanceID, orgID uuid.UUID,
) {
	if reaction == nil || reaction.Key == nil {
		return
	}

	targetMsgID := ""
	if reaction.Key.ID != nil {
		targetMsgID = *reaction.Key.ID
	}
	if targetMsgID == "" {
		cm.logger.Warn("Reaction received without target message ID")
		return
	}

	emoji := ""
	if reaction.Text != nil {
		emoji = *reaction.Text
	}

	reactionClient := cm.GetClient(instanceID)
	senderPhone := cm.resolveSenderPhone(ctx, reactionClient, info)

	var message models.Message
	if err := cm.db.WithContext(ctx).
		Where("whats_app_message_id = ? AND instance_id = ?", targetMsgID, instanceID).
		First(&message).Error; err != nil {
		cm.logger.Warn("Reaction target message not found",
			"target_msg_id", targetMsgID,
			"sender", senderPhone)
		cm.MarkError(instanceID)
		return
	}

	metadata := message.Metadata
	if metadata == nil {
		metadata = make(models.JSONB)
	}

	type reactionEntry struct {
		Emoji     string `json:"emoji"`
		FromPhone string `json:"from_phone,omitempty"`
	}

	var reactions []reactionEntry
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if arr, ok := reactionsRaw.([]interface{}); ok {
			for _, r := range arr {
				if rMap, ok := r.(map[string]interface{}); ok {
					e, _ := rMap["emoji"].(string)
					fp, _ := rMap["from_phone"].(string)
					reactions = append(reactions, reactionEntry{Emoji: e, FromPhone: fp})
				}
			}
		}
	}

	var newReactions []reactionEntry
	for _, r := range reactions {
		if r.FromPhone != senderPhone {
			newReactions = append(newReactions, r)
		}
	}

	if emoji != "" {
		newReactions = append(newReactions, reactionEntry{
			Emoji:     emoji,
			FromPhone: senderPhone,
		})
	}

	metadata["reactions"] = newReactions
	if err := cm.db.WithContext(ctx).
		Model(&message).
		Update("metadata", metadata).Error; err != nil {
		cm.logger.Error("Failed to update reaction metadata", "error", err)
		cm.MarkError(instanceID)
		return
	}

	if cm.hub != nil {
		cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: "reaction_update",
			Payload: map[string]any{
				"message_id": message.ID.String(),
				"contact_id": message.ContactID.String(),
				"reactions":  newReactions,
				"timestamp":  time.Now(),
			},
		})
	}

	cm.logger.Info("Reaction processed",
		"target_msg_id", targetMsgID,
		"emoji", emoji,
		"from", senderPhone)
}
