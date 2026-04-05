package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
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
	if w.app == nil || !w.app.isWhatsmeowProvider() {
		return
	}

	var instances []models.WhatsAppInstance
	if err := w.app.DB.Select("id", "organization_id", "settings").Find(&instances).Error; err != nil {
		w.app.Log.Error("Assigned chat reset worker failed to load instances", "error", err)
		return
	}
	if len(instances) == 0 {
		return
	}

	orgTimezones := w.loadOrganizationTimezones(instances)
	for idx := range instances {
		instance := instances[idx]
		timezone := orgTimezones[instance.OrganizationID]
		if timezone == "" {
			timezone = "UTC"
		}

		if err := w.processInstance(nowUTC, instance, timezone); err != nil {
			w.app.Log.Error(
				"Assigned chat reset worker failed to process instance",
				"error", err,
				"org_id", instance.OrganizationID,
				"instance_id", instance.ID,
			)
		}
	}
}

func (w *ChatAssignmentResetWorker) loadOrganizationTimezones(instances []models.WhatsAppInstance) map[uuid.UUID]string {
	orgIDSet := make(map[uuid.UUID]struct{}, len(instances))
	for _, instance := range instances {
		orgIDSet[instance.OrganizationID] = struct{}{}
	}
	orgIDs := make([]uuid.UUID, 0, len(orgIDSet))
	for orgID := range orgIDSet {
		orgIDs = append(orgIDs, orgID)
	}

	timezones := make(map[uuid.UUID]string, len(orgIDs))
	if len(orgIDs) == 0 {
		return timezones
	}

	var organizations []models.Organization
	if err := w.app.DB.Select("id", "settings").Where("id IN ?", orgIDs).Find(&organizations).Error; err != nil {
		w.app.Log.Warn("Assigned chat reset worker failed to load organization timezones; defaulting to UTC", "error", err)
		return timezones
	}
	for _, org := range organizations {
		timezones[org.ID] = parseOrganizationTimezone(org.Settings)
	}

	return timezones
}

func (w *ChatAssignmentResetWorker) processInstance(nowUTC time.Time, instance models.WhatsAppInstance, timezone string) error {
	schedule := waManager.AssignedChatResetSettingsFromSettings(instance.Settings)
	if !schedule.Enabled {
		return nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
		timezone = "UTC"
	}

	localNow := nowUTC.In(location)
	today := localNow.Format("2006-01-02")

	if schedule.LastResetDate == "" && schedule.Mode == waManager.AssignedChatResetModeMidnight {
		return w.persistInstanceResetDate(instance.ID, today)
	}

	if schedule.LastResetDate == today {
		return nil
	}

	if localNow.Hour() < schedule.Hour {
		return nil
	}

	resetCount, resetCandidates, err := w.resetAssignedChats(instance.OrganizationID, instance.ID, nowUTC)
	if err != nil {
		return err
	}

	if err := w.persistInstanceResetDate(instance.ID, today); err != nil {
		return err
	}

	if resetCount > 0 {
		contactIDs := resetCandidateIDs(resetCandidates)
		w.broadcastResetContacts(instance.OrganizationID, contactIDs)
		w.appendResetSystemMessages(resetCandidates, schedule, timezone, today)
		w.app.Log.Info(
			"Assigned chat reset completed",
			"org_id", instance.OrganizationID,
			"instance_id", instance.ID,
			"reset_count", resetCount,
			"mode", schedule.Mode,
			"scheduled_hour", schedule.Hour,
			"timezone", timezone,
		)
	}

	return nil
}

func (w *ChatAssignmentResetWorker) resetAssignedChats(orgID, instanceID uuid.UUID, nowUTC time.Time) (int64, []chatAssignmentResetCandidate, error) {
	query := w.app.DB.Model(&models.Contact{}).
		Where("organization_id = ? AND instance_id = ? AND assigned_user_id IS NOT NULL AND (status IS NULL OR status = '' OR status <> ?)", orgID, instanceID, models.ChatStatusClosed)

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

func (w *ChatAssignmentResetWorker) persistInstanceResetDate(instanceID uuid.UUID, resetDate string) error {
	expr := fmt.Sprintf(
		"jsonb_set(COALESCE(settings, '{}'::jsonb), '{%s}', to_jsonb(?::text), true)",
		waManager.InstanceSettingAssignedChatResetLastDate,
	)

	return w.app.DB.Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
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
	schedule waManager.AssignedChatResetSettings,
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
