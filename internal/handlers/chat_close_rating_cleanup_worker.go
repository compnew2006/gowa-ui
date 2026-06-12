package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/chat_close_ratings"
)

// ChatCloseRatingCleanupWorker periodically cleans up expired unanswered rating cycles.
type ChatCloseRatingCleanupWorker struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

func NewChatCloseRatingCleanupWorker(app *App, interval time.Duration) *ChatCloseRatingCleanupWorker {
	return &ChatCloseRatingCleanupWorker{
		app:      app,
		interval: interval,
	}
}

func (w *ChatCloseRatingCleanupWorker) Start(ctx context.Context) {
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

func (w *ChatCloseRatingCleanupWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

func (w *ChatCloseRatingCleanupWorker) runOnce(nowUTC time.Time) {
	if w.app == nil {
		return
	}

	var pendingCycles []models.ChatClosureRating
	if err := w.app.DB.Where("state = ?", models.ChatClosureRatingStatePending).Find(&pendingCycles).Error; err != nil {
		w.app.Log.Error("Chat close rating cleanup worker failed to load pending cycles", "error", err)
		return
	}

	for _, cycle := range pendingCycles {
		var instanceSettings models.JSONB
		if cycle.Contact != nil && cycle.Contact.InstanceID != nil {
			var instance models.WhatsAppInstance
			if err := w.app.DB.Select("settings").Where("id = ?", *cycle.Contact.InstanceID).First(&instance).Error; err == nil {
				instanceSettings = instance.Settings
			}
		} else {
			var contact models.Contact
			if err := w.app.DB.Select("instance_id").Where("id = ?", cycle.ContactID).First(&contact).Error; err == nil && contact.InstanceID != nil {
				var instance models.WhatsAppInstance
				if err := w.app.DB.Select("settings").Where("id = ?", *contact.InstanceID).First(&instance).Error; err == nil {
					instanceSettings = instance.Settings
				}
			}
		}

		settings := readInstanceChatCloseRatingSettings(instanceSettings)

		var closedAt time.Time
		if !cycle.ClosedAt.IsZero() {
			closedAt = cycle.ClosedAt
		} else {
			closedAt = nowUTC
		}

		followup := chat_close_ratings.ReadFollowupState(closedAt, cycle.ContextMessages, settings.FollowupWindowMinutes)
		if !followup.IsActive(nowUTC) {
			if err := w.app.DB.Model(&models.ChatClosureRating{}).
				Where("id = ? AND state = ?", cycle.ID, models.ChatClosureRatingStatePending).
				Update("state", models.ChatClosureRatingStateExpired).Error; err != nil {
				w.app.Log.Error("Failed to transition expired rating cycle to expired", "cycle_id", cycle.ID, "error", err)
			} else {
				w.app.Log.Info("Automatically expired unanswered chat close rating cycle", "cycle_id", cycle.ID, "contact_id", cycle.ContactID)
			}
		}
	}
}
