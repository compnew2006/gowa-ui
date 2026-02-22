package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatAssignmentResetWorker periodically resets assigned active chats back to pending.
type ChatAssignmentResetWorker struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

type chatAssignmentResetCandidate struct {
	ID              uuid.UUID  `gorm:"column:id"`
	OrganizationID  uuid.UUID  `gorm:"column:organization_id"`
	PhoneNumber     string     `gorm:"column:phone_number"`
	ProfileName     string     `gorm:"column:profile_name"`
	InstanceID      *uuid.UUID `gorm:"column:instance_id"`
	WhatsAppAccount string     `gorm:"column:whats_app_account"`
	AssignedUserID  *uuid.UUID `gorm:"column:assigned_user_id"`
}

func NewChatAssignmentResetWorker(app *App, interval time.Duration) *ChatAssignmentResetWorker {
	return &ChatAssignmentResetWorker{
		app:      app,
		interval: interval,
	}
}

func (w *ChatAssignmentResetWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.ticker = time.NewTicker(w.interval)
	ticker := w.ticker
	w.mu.Unlock()
	defer ticker.Stop()

	w.runOnce(time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(time.Now().UTC())
		}
	}
}

func (w *ChatAssignmentResetWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

func (w *ChatAssignmentResetWorker) runOnce(nowUTC time.Time) {
	var organizations []models.Organization
	if err := w.app.DB.Select("id", "settings").Find(&organizations).Error; err != nil {
		w.app.Log.Error("Assigned chat reset worker failed to load organizations", "error", err)
		return
	}

	for idx := range organizations {
		if err := w.processOrganization(nowUTC, organizations[idx]); err != nil {
			w.app.Log.Error("Assigned chat reset worker failed to process organization", "error", err, "org_id", organizations[idx].ID)
		}
	}
}

func (w *ChatAssignmentResetWorker) processOrganization(nowUTC time.Time, organization models.Organization) error {
	schedule := readChatAssignmentResetSettings(organization.Settings)
	if !schedule.Enabled {
		return nil
	}
	tzName := parseOrganizationTimezone(organization.Settings)
	location, err := time.LoadLocation(tzName)
	if err != nil {
		location = time.UTC
		tzName = "UTC"
	}

	localNow := nowUTC.In(location)
	today := localNow.Format("2006-01-02")

	if schedule.LastResetDate == "" && schedule.Mode == ChatAssignmentResetModeMidnight {
		return w.persistOrganizationResetDate(organization.ID, today)
	}

	if schedule.LastResetDate == today {
		return nil
	}

	if localNow.Hour() < schedule.Hour {
		return nil
	}

	resetCount, resetCandidates, err := w.resetAssignedChats(organization.ID, nowUTC)
	if err != nil {
		return err
	}

	if err := w.persistOrganizationResetDate(organization.ID, today); err != nil {
		return err
	}

	if resetCount > 0 {
		contactIDs := resetCandidateIDs(resetCandidates)
		w.broadcastResetContacts(organization.ID, contactIDs)
		w.appendResetSystemMessages(resetCandidates, schedule, tzName, today)
		w.app.Log.Info(
			"Assigned chat reset completed",
			"org_id", organization.ID,
			"reset_count", resetCount,
			"mode", schedule.Mode,
			"scheduled_hour", schedule.Hour,
			"timezone", tzName,
		)
	}

	return nil
}

func (w *ChatAssignmentResetWorker) resetAssignedChats(orgID uuid.UUID, nowUTC time.Time) (int64, []chatAssignmentResetCandidate, error) {
	query := w.app.DB.Model(&models.Contact{}).
		Where("organization_id = ? AND assigned_user_id IS NOT NULL AND (status IS NULL OR status = '' OR status <> ?)", orgID, models.ChatStatusClosed)

	var resetCandidates []chatAssignmentResetCandidate
	if err := query.Select("id", "organization_id", "phone_number", "profile_name", "instance_id", "whats_app_account", "assigned_user_id").Find(&resetCandidates).Error; err != nil {
		return 0, nil, err
	}
	if len(resetCandidates) == 0 {
		return 0, nil, nil
	}

	updates := map[string]any{
		"assigned_user_id":  nil,
		"status":            models.ChatStatusPending,
		"closed_at":         nil,
		"closed_by_user_id": nil,
		"updated_at":        nowUTC,
	}

	result := w.app.DB.Model(&models.Contact{}).
		Where("id IN ?", resetCandidateIDs(resetCandidates)).
		Updates(updates)
	if result.Error != nil {
		return 0, nil, result.Error
	}

	return result.RowsAffected, resetCandidates, nil
}

func (w *ChatAssignmentResetWorker) persistOrganizationResetDate(orgID uuid.UUID, resetDate string) error {
	expr := fmt.Sprintf(
		"jsonb_set(COALESCE(settings, '{}'::jsonb), '{%s}', to_jsonb(?::text), true)",
		organizationSettingAssignedChatResetLastDate,
	)

	return w.app.DB.Model(&models.Organization{}).
		Where("id = ?", orgID).
		Update("settings", gorm.Expr(expr, resetDate)).
		Error
}

func (w *ChatAssignmentResetWorker) broadcastResetContacts(orgID uuid.UUID, contactIDs []uuid.UUID) {
	if w.app.WSHub == nil {
		return
	}

	for _, contactID := range contactIDs {
		w.app.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeContactUpdate,
			Payload: map[string]any{
				"id":               contactID.String(),
				"assigned_user_id": "",
				"status":           models.ChatStatusPending.String(),
			},
		})
	}
}

func resetCandidateIDs(resetCandidates []chatAssignmentResetCandidate) []uuid.UUID {
	contactIDs := make([]uuid.UUID, 0, len(resetCandidates))
	for _, candidate := range resetCandidates {
		contactIDs = append(contactIDs, candidate.ID)
	}
	return contactIDs
}

func (w *ChatAssignmentResetWorker) appendResetSystemMessages(
	resetCandidates []chatAssignmentResetCandidate,
	schedule ChatAssignmentResetSettings,
	timezone string,
	resetDate string,
) {
	if len(resetCandidates) == 0 {
		return
	}

	for _, candidate := range resetCandidates {
		metadata := models.JSONB{
			"event_type":    "chat_assignment_reset",
			"reason":        "assigned_chat_reset_schedule",
			"schedule_mode": string(schedule.Mode),
			"schedule_hour": schedule.Hour,
			"timezone":      timezone,
			"reset_date":    resetDate,
		}
		if candidate.AssignedUserID != nil && *candidate.AssignedUserID != uuid.Nil {
			metadata["previous_assigned_user_id"] = candidate.AssignedUserID.String()
		}
		resetContact := models.Contact{
			BaseModel: models.BaseModel{
				ID: candidate.ID,
			},
			OrganizationID:  candidate.OrganizationID,
			PhoneNumber:     candidate.PhoneNumber,
			ProfileName:     candidate.ProfileName,
			InstanceID:      candidate.InstanceID,
			WhatsAppAccount: candidate.WhatsAppAccount,
			Status:          models.ChatStatusPending,
			AssignedUserID:  nil,
		}
		w.app.appendSystemChatMessage(
			&resetContact,
			"System: Assigned Chat Reset schedule moved this chat back to pending queue.",
			metadata,
		)
	}
}
