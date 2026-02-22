package handlers

import (
	"context"
	"sync"
	"time"
)

// ActivityRetentionWorker periodically purges old activity logs.
type ActivityRetentionWorker struct {
	app       *App
	interval  time.Duration
	retention time.Duration
	mu        sync.Mutex
	ticker    *time.Ticker
}

func NewActivityRetentionWorker(app *App, interval, retention time.Duration) *ActivityRetentionWorker {
	return &ActivityRetentionWorker{
		app:       app,
		interval:  interval,
		retention: retention,
	}
}

func (w *ActivityRetentionWorker) purgeOnce() {
	cutoff := time.Now().Add(-w.retention)
	deleted, err := w.app.PurgeOlderThan(cutoff)
	if err != nil {
		w.app.Log.Error("Activity retention purge failed", "error", err, "cutoff", cutoff)
		return
	}
	if deleted > 0 {
		w.app.Log.Info("Activity retention purge completed", "deleted_rows", deleted, "cutoff", cutoff)
	}
}

// Start begins periodic retention cleanup until ctx is cancelled.
func (w *ActivityRetentionWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.ticker = time.NewTicker(w.interval)
	ticker := w.ticker
	w.mu.Unlock()
	defer ticker.Stop()

	w.purgeOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.purgeOnce()
		}
	}
}

// Stop stops the periodic ticker.
func (w *ActivityRetentionWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}
