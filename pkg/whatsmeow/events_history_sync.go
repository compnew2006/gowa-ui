package whatsmeow

import (
	"context"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (cm *ConnectionManager) handleHistorySync(ctx context.Context, evt *events.HistorySync, instanceID, orgID uuid.UUID) {
	if evt == nil || evt.Data == nil {
		return
	}
	if !cm.isHistorySyncEnabled(ctx, instanceID) {
		cm.logger.Info("History sync skipped by instance settings", "instance_id", instanceID)
		return
	}

	client := cm.GetClient(instanceID)
	if client == nil {
		cm.logger.Warn("History sync received for non-connected instance", "instance_id", instanceID)
		return
	}

	persistedCount := 0
	for _, conversation := range evt.Data.GetConversations() {
		if conversation == nil {
			continue
		}

		chatID := conversation.GetID()
		if chatID == "" {
			continue
		}

		chatJID, err := types.ParseJID(chatID)
		if err != nil {
			cm.logger.Debug("Skipping history conversation with invalid JID", "instance_id", instanceID, "conversation_id", chatID, "error", err)
			continue
		}

		for _, historyMsg := range conversation.GetMessages() {
			if historyMsg == nil || historyMsg.GetMessage() == nil {
				continue
			}

			parsedMessage, err := client.ParseWebMessage(chatJID, historyMsg.GetMessage())
			if err != nil {
				cm.logger.Debug("Failed to parse history message", "instance_id", instanceID, "conversation_id", chatID, "error", err)
				continue
			}

			if _, err := cm.persistParsedMessage(ctx, client, parsedMessage, instanceID, orgID, persistMessageOptions{
				AllowFromMe:   true,
				Broadcast:     false,
				HistorySync:   true,
				UpdateMetrics: false,
			}); err != nil {
				cm.logger.Warn("Failed to persist history message", "instance_id", instanceID, "conversation_id", chatID, "error", err)
				continue
			}
			persistedCount++
		}
	}

	cm.logger.Info("History sync processed", "instance_id", instanceID, "persisted_messages", persistedCount)
}
