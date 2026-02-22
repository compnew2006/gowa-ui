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

	resetCount, contactIDs, err := w.resetAssignedChats(organization.ID, nowUTC)
	if err != nil {
		return err
	}

	if err := w.persistOrganizationResetDate(organization.ID, today); err != nil {
		return err
	}

	if resetCount > 0 {
		w.broadcastResetContacts(organization.ID, contactIDs)
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

func (w *ChatAssignmentResetWorker) resetAssignedChats(orgID uuid.UUID, nowUTC time.Time) (int64, []uuid.UUID, error) {
	query := w.app.DB.Model(&models.Contact{}).
		Where("organization_id = ? AND assigned_user_id IS NOT NULL AND (status IS NULL OR status = '' OR status <> ?)", orgID, models.ChatStatusClosed)

	var contactIDs []uuid.UUID
	if err := query.Pluck("id", &contactIDs).Error; err != nil {
		return 0, nil, err
	}
	if len(contactIDs) == 0 {
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
		Where("id IN ?", contactIDs).
		Updates(updates)
	if result.Error != nil {
		return 0, nil, result.Error
	}

	return result.RowsAffected, contactIDs, nil
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
