package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/models"
)

// ChatResetProcessor runs the daily assigned-chat reset. It ticks every minute
// and, for each WhatsApp account whose daily_reset schedule has fired, returns
// all assigned (open) conversations to the pending pool via
// chatlifecycle.Service.ResetAssignedChats.
//
// The processor mirrors SLAProcessor's lifecycle (Start/Stop with a context
// + stopCh). "Already run today" is tracked in-memory per account — the reset
// itself is idempotent (a second pass finds no assigned chats), so the
// in-memory guard is purely an optimization to avoid redundant DB work within
// the same process.
type ChatResetProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}

	mu      sync.Mutex
	lastRun map[uuid.UUID]string // account ID → "2006-01-02" of the last reset
}

// NewChatResetProcessor creates a new daily-reset processor. The interval
// controls how often the scheduler is polled; the default is one minute.
func NewChatResetProcessor(app *App, interval time.Duration) *ChatResetProcessor {
	return &ChatResetProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
		lastRun:  make(map[uuid.UUID]string),
	}
}

// Start begins the daily-reset processing loop. Blocks until the context is
// cancelled or Stop is called.
func (p *ChatResetProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Chat reset processor started", "interval", p.interval)

	// Run once shortly after startup so a server that starts after the
	// configured reset time still catches up the same day.
	time.AfterFunc(10*time.Second, func() {
		p.processDueAccounts(time.Now())
	})

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Chat reset processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("Chat reset processor stopped")
			return
		case now := <-ticker.C:
			p.processDueAccounts(now)
		}
	}
}

// Stop signals the processor to exit.
func (p *ChatResetProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// processDueAccounts loads every account with daily_reset enabled and resets
// those whose configured wall-clock time has been reached today and that have
// not yet been reset today.
func (p *ChatResetProcessor) processDueAccounts(now time.Time) {
	var accounts []models.WhatsAppAccount
	// settings->'daily_reset'->>'enabled' = 'true' — JSONB path query. Accounts
	// without the block (or with enabled=false) are skipped by the DB filter.
	if err := p.app.DB.Where(
		`settings->'daily_reset'->>'enabled' = 'true'`,
	).Find(&accounts).Error; err != nil {
		p.app.Log.Error("Failed to load daily-reset accounts", "error", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	today := now.Format("2006-01-02")

	for i := range accounts {
		account := &accounts[i]
		p.processAccount(account, now, today)
	}
}

// processAccount evaluates one account's schedule and runs the reset if due.
func (p *ChatResetProcessor) processAccount(account *models.WhatsAppAccount, now time.Time, today string) {
	settings := chatResetSettingsForAccount(account)

	// Resolve the account's wall-clock "now" in the configured timezone.
	loc := time.Local
	if settings.Timezone != "" {
		if loaded, err := time.LoadLocation(settings.Timezone); err == nil {
			loc = loaded
		}
	}
	localNow := now.In(loc)

	// Parse the configured reset time (HH:MM).
	resetTime, err := time.Parse("15:04", settings.Time)
	if err != nil {
		// Invalid time format — skip and log. The PUT handler validates, so
		// this should only happen with manually-corrupted data.
		p.app.Log.Warn("Skipping account with invalid daily-reset time",
			"account_id", account.ID, "account", account.Name, "time", settings.Time)
		return
	}

	// Build today's reset instant in the account's timezone.
	scheduledAt := time.Date(
		localNow.Year(), localNow.Month(), localNow.Day(),
		resetTime.Hour(), resetTime.Minute(), 0, 0, loc,
	)

	// Not yet time today.
	if localNow.Before(scheduledAt) {
		return
	}

	// Already run today (in-memory guard).
	if p.alreadyRunToday(account.ID, today) {
		return
	}

	// Fire the reset.
	summary, err := p.app.ChatLifecycle.ResetAssignedChats(
		context.Background(), account.OrganizationID, account.Name, "System",
	)
	if err != nil {
		p.app.Log.Error("Daily chat reset failed",
			"error", err, "account_id", account.ID, "account", account.Name)
		return
	}

	p.markRun(account.ID, today)

	p.app.Log.Info("Daily chat reset fired",
		"account_id", account.ID, "account", account.Name,
		"scheduled_at", scheduledAt.Format(time.RFC3339),
		"reset_count", summary.ResetCount, "skipped", summary.Skipped)
}

// alreadyRunToday returns true if the account was already reset on the given
// calendar date (process-local).
func (p *ChatResetProcessor) alreadyRunToday(accountID uuid.UUID, today string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastRun[accountID] == today
}

// markRun records that the account was reset on the given calendar date.
func (p *ChatResetProcessor) markRun(accountID uuid.UUID, today string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRun[accountID] = today
}
