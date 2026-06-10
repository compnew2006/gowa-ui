package whatsmeow

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
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

	var vote *waE2E.PollVoteMessage
	var err error
	if evt.Info.Chat.Server == waTypes.HiddenUserServer && client != nil && client.Store != nil && !client.Store.LID.IsEmpty() {
		originalID := client.Store.ID
		ownLID := client.Store.LID.ToNonAD()
		client.Store.ID = &ownLID
		vote, err = client.DecryptPollVote(ctx, evt)
		client.Store.ID = originalID
	} else {
		vote, err = client.DecryptPollVote(ctx, evt)
	}
	if err != nil {
		cm.logger.Warn("Failed to decrypt poll vote", "instance_id", instanceID, "wa_message_id", evt.Info.ID, "error", err)
		return
	}

	selectedNames := cm.resolveSelectedOptionNames(ctx, orgID, instanceID, originalWAMID, vote.GetSelectedOptions())

	var originalMsg models.Message
	if err := cm.db.WithContext(ctx).
		Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?", orgID, instanceID, originalWAMID).
		First(&originalMsg).Error; err != nil {
		cm.logger.Warn("Poll vote original message not found", "instance_id", instanceID, "poll_wa_id", originalWAMID, "error", err)
		return
	}

	voter := cm.resolveSenderPhone(ctx, client, evt.Info)
	if voter == "" {
		voter = evt.Info.Sender.String()
	}
	if voter == "" {
		voter = evt.Info.ID
	}

	updatedInteractive := applyPollVoteToInteractive(originalMsg.InteractiveData, voter, selectedNames)
	now := time.Now()
	if err := cm.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", originalMsg.ID).
		Updates(map[string]any{
			"interactive_data": updatedInteractive,
			"updated_at":       now,
		}).Error; err != nil {
		cm.logger.Error("Failed to update poll vote on original message", "error", err, "message_id", originalMsg.ID)
		return
	}

	originalMsg.InteractiveData = updatedInteractive
	originalMsg.UpdatedAt = now
	cm.broadcastPollVoteUpdate(orgID, &originalMsg)

	// Capture poll vote as close rating if it matches an active pending rating cycle
	var cycle models.ChatClosureRating
	if err := cm.db.WithContext(ctx).Where(
		"contact_id = ? AND close_message_id = ? AND state = ?",
		originalMsg.ContactID,
		originalMsg.ID,
		models.ChatClosureRatingStatePending,
	).First(&cycle).Error; err == nil {
		if len(selectedNames) > 0 {
			var optsSlice []interface{}
			for _, n := range selectedNames {
				optsSlice = append(optsSlice, n)
			}
			incomingMsg := models.Message{
				BaseModel:         models.BaseModel{ID: uuid.New()},
				OrganizationID:    orgID,
				InstanceID:        &instanceID,
				WhatsAppAccount:   originalMsg.WhatsAppAccount,
				ContactID:         originalMsg.ContactID,
				WhatsAppMessageID: evt.Info.ID,
				Direction:         models.DirectionIncoming,
				MessageType:       models.MessageTypePoll,
				Content:           strings.Join(selectedNames, ", "),
				Status:            models.MessageStatusReceived,
				InteractiveData: models.JSONB{
					"type":             "poll_vote",
					"selected_options": optsSlice,
				},
			}
			if err := cm.db.WithContext(ctx).Create(&incomingMsg).Error; err != nil {
				cm.logger.Error("Failed to persist incoming poll vote message for rating", "error", err)
			} else {
				var contact models.Contact
				if err := cm.db.WithContext(ctx).Where("id = ?", originalMsg.ContactID).First(&contact).Error; err == nil {
					cm.maybeCaptureChatCloseRating(ctx, orgID, &contact, &incomingMsg)
				}
			}
		}
	}

	cm.logger.Debug("Updated poll vote on original message",
		"instance_id", instanceID,
		"poll_wa_id", originalWAMID,
		"selected", selectedNames,
	)
}

func applyPollVoteToInteractive(existing models.JSONB, voter string, selectedNames []string) models.JSONB {
	updated := cloneJSONB(existing)
	if updated == nil {
		updated = models.JSONB{}
	}
	if _, ok := updated["type"].(string); !ok {
		updated["type"] = "poll"
	}

	voters := pollVoteVoters(updated["voters"])
	if len(selectedNames) == 0 {
		delete(voters, voter)
	} else {
		voters[voter] = append([]string(nil), selectedNames...)
	}
	updated["voters"] = voters
	updated["votes"] = pollVoteCounts(voters)
	updated["total_votes"] = len(voters)
	updated["last_selected_options"] = append([]string(nil), selectedNames...)
	updated["last_voter"] = voter
	return updated
}

func cloneJSONB(existing models.JSONB) models.JSONB {
	if len(existing) == 0 {
		return models.JSONB{}
	}
	cloned := make(models.JSONB, len(existing))
	for key, value := range existing {
		cloned[key] = value
	}
	return cloned
}

func pollVoteVoters(raw interface{}) map[string][]string {
	voters := map[string][]string{}
	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		if typed, ok := raw.(map[string][]string); ok {
			for voter, selected := range typed {
				voters[voter] = append([]string(nil), selected...)
			}
		}
		return voters
	}
	for voter, selected := range rawMap {
		voters[voter] = stringSliceValue(selected)
	}
	return voters
}

func stringSliceValue(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func pollVoteCounts(voters map[string][]string) map[string]int {
	counts := map[string]int{}
	for _, selected := range voters {
		for _, option := range selected {
			option = strings.TrimSpace(option)
			if option != "" {
				counts[option]++
			}
		}
	}
	return counts
}

func (cm *ConnectionManager) broadcastPollVoteUpdate(orgID uuid.UUID, message *models.Message) {
	if cm.hub == nil || message == nil {
		return
	}
	cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeMessageMediaUpdated,
		Payload: map[string]any{
			"id":               message.ID.String(),
			"contact_id":       message.ContactID.String(),
			"message_type":     message.MessageType,
			"content":          map[string]string{"body": message.Content},
			"interactive_data": message.InteractiveData,
			"updated_at":       message.UpdatedAt,
		},
	})
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
