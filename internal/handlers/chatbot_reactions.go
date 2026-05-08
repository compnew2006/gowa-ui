package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/contactutil"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
)

type Reaction struct {
	Emoji     string `json:"emoji"`
	FromPhone string `json:"from_phone,omitempty"`
	FromUser  string `json:"from_user,omitempty"`
}

func (a *App) handleIncomingReaction(account *models.WhatsAppAccount, fromPhone, messageWAMID, emoji, profileName string) {
	a.Log.Info("Handling incoming reaction",
		"from", fromPhone,
		"message_wamid", messageWAMID,
		"emoji", emoji,
	)

	var message models.Message
	if err := a.DB.Where("whats_app_message_id = ?", messageWAMID).First(&message).Error; err != nil {
		if idx := strings.Index(messageWAMID, "FQIA"); idx != -1 {
			suffixStart := idx + 8
			if suffixStart < len(messageWAMID) {
				suffix := messageWAMID[suffixStart:]
				if err := a.DB.Where("whats_app_message_id LIKE ?", "%"+suffix).First(&message).Error; err != nil {
					a.Log.Warn("Message not found for reaction", "wamid", messageWAMID, "suffix", suffix)
					return
				}
			} else {
				a.Log.Warn("Message not found for reaction - invalid WAMID format", "wamid", messageWAMID)
				return
			}
		} else {
			a.Log.Warn("Message not found for reaction - no FQIA pattern", "wamid", messageWAMID)
			return
		}
	}

	contact, _, _ := contactutil.GetOrCreateContact(a.DB, account.OrganizationID, fromPhone, profileName)

	var metadata map[string]any
	if message.Metadata != nil {
		metadata = message.Metadata
	} else {
		metadata = make(map[string]any)
	}

	var reactions []Reaction
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]interface{}); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]any); ok {
					emoji, _ := rMap["emoji"].(string)
					reactions = append(reactions, Reaction{
						Emoji:     emoji,
						FromPhone: getStringFromMap(rMap, "from_phone"),
						FromUser:  getStringFromMap(rMap, "from_user"),
					})
				}
			}
		}
	}

	var newReactions []Reaction
	for _, r := range reactions {
		if r.FromPhone != fromPhone {
			newReactions = append(newReactions, r)
		}
	}

	if emoji != "" {
		newReactions = append(newReactions, Reaction{
			Emoji:     emoji,
			FromPhone: fromPhone,
		})
	}

	metadata["reactions"] = newReactions

	if err := a.DB.Model(&message).Update("metadata", metadata).Error; err != nil {
		a.Log.Error("Failed to update message reactions", "error", err)
		return
	}

	a.Log.Info("Updated message reaction", "message_id", message.ID, "reactions_count", len(newReactions))

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: "reaction_update",
			Payload: map[string]any{
				"message_id": message.ID.String(),
				"contact_id": contact.ID.String(),
				"reactions":  newReactions,
			},
		})
	}
}

func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
