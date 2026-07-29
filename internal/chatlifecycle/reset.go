package chatlifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/audit"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
)

// ResetSummary captures the outcome of a batch reset for logging, audit, and
// the response envelope returned to callers (the scheduled processor).
type ResetSummary struct {
	OrgID           uuid.UUID   `json:"org_id"`
	WhatsAppAccount string      `json:"whatsapp_account"`
	ResetCount      int         `json:"reset_count"`
	ContactIDs      []uuid.UUID `json:"contact_ids"`
	Skipped         int         `json:"skipped"`
}

// ResetAssignedChats returns every assigned (status = open, assigned_user_id
// set) conversation for one WhatsApp account back to the pending pool. This is
// the batch body of the daily scheduled reset.
//
// For each contact it mirrors Service.Release's side effects so the
// conversation timeline, audit trail, and WebSocket clients stay consistent
// with a manual release:
//   - assigned_user_id cleared
//   - metadata.chat_status set to "pending"
//   - collaborators cleared
//   - last_message_at bumped (released chats re-sort to the top of Pending)
//   - a system message recorded in the timeline
//
// A single summary audit-log entry is written for the whole batch (per-contact
// audit would flood the log on a large reset). The batch is idempotent: an
// account with no assigned-open chats resets zero rows and writes no audit
// entry.
//
// actorName is "System" for the scheduled reset and may be overridden when a
// manual "reset all" action is wired in the future.
func (s *Service) ResetAssignedChats(_ context.Context, orgID uuid.UUID, accountName, actorName string) (ResetSummary, error) {
	summary := ResetSummary{
		OrgID:           orgID,
		WhatsAppAccount: accountName,
		ContactIDs:      []uuid.UUID{},
	}

	if actorName == "" {
		actorName = "System"
	}

	// Target set: contacts assigned to an agent whose effective status is open.
	// Status lives in the JSONB metadata map; an assigned contact with no
	// explicit chat_status key defaults to open (see EffectiveStatus). We
	// therefore match assigned contacts whose chat_status is absent OR "open",
	// excluding explicitly-pending and explicitly-closed rows.
	var contacts []models.Contact
	if err := s.db.Where(
		`organization_id = ? AND whats_app_account = ? AND assigned_user_id IS NOT NULL
		   AND (metadata->>'chat_status' IS NULL OR metadata->>'chat_status' = ?)`,
		orgID, accountName, string(models.ChatStatusOpen),
	).Find(&contacts).Error; err != nil {
		s.log.Error("Failed to load assigned chats for daily reset",
			"error", err, "org_id", orgID, "account", accountName)
		return summary, fmt.Errorf("chat: failed to load assigned chats: %w", err)
	}

	if len(contacts) == 0 {
		return summary, nil
	}

	now := time.Now()
	failed := 0

	for i := range contacts {
		c := &contacts[i]

		c.AssignedUserID = nil
		c.SetStatus(models.ChatStatusPending)
		c.ClearCollaborators()

		if err := s.db.Model(&models.Contact{}).Where("id = ?", c.ID).Updates(map[string]any{
			"assigned_user_id": nil,
			"metadata":         c.Metadata,
			"last_message_at":  &now,
		}).Error; err != nil {
			s.log.Error("Failed to reset assigned chat",
				"error", err, "contact_id", c.ID)
			failed++
			continue
		}

		s.CreateSystemMessage(orgID, c.ID,
			"🔔 Conversation returned to the pending queue (daily reset schedule)",
			models.JSONB{
				"system_type":      "chat_daily_reset",
				"reset_by":         "schedule",
				"whatsapp_account": accountName,
			})

		summary.ResetCount++
		summary.ContactIDs = append(summary.ContactIDs, c.ID)
	}
	summary.Skipped = failed

	// Single audit entry for the batch. uuid.Nil marks a system-initiated
	// action; the summary diff records how many chats were affected.
	if summary.ResetCount > 0 {
		audit.LogAudit(s.db, orgID, uuid.Nil, actorName,
			models.ResourceSettingsChatReset, uuid.Nil, models.AuditActionUpdated, nil, nil,
			map[string]any{
				"whatsapp_account": accountName,
				"reset_count":      summary.ResetCount,
				"skipped":          summary.Skipped,
				"contact_ids":      contactIDStrings(summary.ContactIDs),
				"trigger":          "daily_schedule",
			})
	}

	// Broadcast a summary notification so connected clients can refresh their
	// chat lists. Individual chat_released broadcasts would be noisy for a
	// large batch; one summary + the list of affected contact IDs lets the
	// frontend update the affected rows in place.
	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeDailyChatReset,
		Payload: map[string]any{
			"org_id":           orgID.String(),
			"whatsapp_account": accountName,
			"reset_count":      summary.ResetCount,
			"contact_ids":      contactIDStrings(summary.ContactIDs),
			"chat_status":      string(models.ChatStatusPending),
			"last_message_at":  now.Format(time.RFC3339Nano),
		},
	})

	s.log.Info("Daily chat reset completed",
		"org_id", orgID, "account", accountName,
		"reset_count", summary.ResetCount, "skipped", summary.Skipped)

	return summary, nil
}

func contactIDStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
