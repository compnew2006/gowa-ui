package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/compnew2006/gowa-ui/internal/worker"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// setupMiddleware registers the four global Before middlewares in the exact
// order they were inlined previously: SecurityHeaders → RequestLogger →
// Recovery → CSRFProtection. fastglue applies Before handlers in registration
// order, so reordering is an observable behavior change. CORS is handled at
// the fasthttp layer (corsWrapper), not here.
func setupMiddleware(g *fastglue.Fastglue, lo logf.Logger) {
	g.Before(middleware.SecurityHeaders())
	g.Before(middleware.RequestLogger(lo))
	g.Before(middleware.Recovery(lo))
	g.Before(middleware.CSRFProtection())
}

// processorHandles bundles the three background processors and their cancel
// funcs started by startProcessors, so gracefulShutdown can address each in
// the same order they were started.
type processorHandles struct {
	chatReset     *handlers.ChatResetProcessor
	chatResetStop context.CancelFunc

	scheduledMsg     *handlers.ScheduledMessageProcessor
	scheduledMsgStop context.CancelFunc

	gowaHistory     *handlers.GowaHistorySyncProcessor
	gowaHistoryStop context.CancelFunc

	gowaWebhook     *handlers.GowaWebhookProcessor
	gowaWebhookStop context.CancelFunc
}

// startProcessors starts the three periodic background processors (chat-reset
// every minute, scheduled-message every minute, GOWA-history-sync every 15
// minutes). Each gets its own cancellable context so shutdown can stop them
// independently. All three "started" log lines are preserved verbatim.
func startProcessors(app *handlers.App, lo logf.Logger) *processorHandles {
	// Start daily chat-reset processor (polls every minute, resets assigned
	// chats to pending per account schedule).
	chatResetProcessor := handlers.NewChatResetProcessor(app, time.Minute)
	chatResetCtx, chatResetCancel := context.WithCancel(context.Background())
	go chatResetProcessor.Start(chatResetCtx)
	lo.Info("Chat reset processor started")

	// Start scheduled-message processor (polls every minute, fires due
	// scheduled messages through the unified sender).
	scheduledMsgProcessor := handlers.NewScheduledMessageProcessor(app, time.Minute)
	scheduledMsgCtx, scheduledMsgCancel := context.WithCancel(context.Background())
	go scheduledMsgProcessor.Start(scheduledMsgCtx)
	lo.Info("Scheduled message processor started")

	// Start GOWA history-sync processor (initial pass at startup, then periodic
	// re-sync). GOWA syncs message history itself but never replays it via
	// webhook, so this pulls synced history into the messages table
	// automatically. A device "connected" webhook also triggers an immediate
	// sync; the per-account cooldown keeps overlapping triggers cheap.
	gowaHistoryProcessor := handlers.NewGowaHistorySyncProcessor(app, 15*time.Minute)
	gowaHistoryCtx, gowaHistoryCancel := context.WithCancel(context.Background())
	go gowaHistoryProcessor.Start(gowaHistoryCtx)
	lo.Info("GOWA history sync processor started")

	// Start the durable GOWA webhook inbox processor. The webhook handler
	// persists every inbound event before 2xx and calls app.GowaWebhookNotify
	// (wired here) to wake this processor for near-real-time dispatch; the
	// 5s poll is the safety net (crash recovery + events enqueued while down).
	gowaWebhookProcessor := handlers.NewGowaWebhookProcessor(app, 5*time.Second)
	gowaWebhookCtx, gowaWebhookCancel := context.WithCancel(context.Background())
	app.GowaWebhookNotify = gowaWebhookProcessor.Notify
	go gowaWebhookProcessor.Start(gowaWebhookCtx)
	lo.Info("GOWA webhook inbox processor started")

	return &processorHandles{
		chatReset:     chatResetProcessor,
		chatResetStop: chatResetCancel,

		scheduledMsg:     scheduledMsgProcessor,
		scheduledMsgStop: scheduledMsgCancel,

		gowaHistory:     gowaHistoryProcessor,
		gowaHistoryStop: gowaHistoryCancel,

		gowaWebhook:     gowaWebhookProcessor,
		gowaWebhookStop: gowaWebhookCancel,
	}
}

// startEmbeddedWorkers starts the workers embedded in the server command when
// numWorkers > 0. It preserves the per-worker worker.New + fatal-on-error +
// goroutine spawn with the workerNum closure capture, and the numWorkers == 0
// "disabled" log. Returns started=false when numWorkers == 0 so the caller
// leaves workerCancel nil and gracefulShutdown skips worker teardown.
func startEmbeddedWorkers(
	cfg *config.Config,
	db *gorm.DB,
	rdb *redis.Client,
	lo logf.Logger,
	waRegistry *whatsapp.Registry,
	numWorkers int,
) (workers []*worker.Worker, workerCancel context.CancelFunc, started bool) {
	if numWorkers <= 0 {
		lo.Info("Embedded workers disabled, run workers separately")
		return nil, nil, false
	}

	var workerCtx context.Context
	workerCtx, workerCancel = context.WithCancel(context.Background())

	for i := 0; i < numWorkers; i++ {
		w, err := worker.New(cfg, db, rdb, lo, waRegistry)
		if err != nil {
			lo.Fatal("Failed to create worker", "error", err, "worker_num", i+1)
		}
		workers = append(workers, w)

		workerNum := i + 1
		go func() {
			lo.Info("Worker started", "worker_num", workerNum)
			if err := w.Run(workerCtx); err != nil && err != context.Canceled {
				lo.Error("Worker error", "error", err, "worker_num", workerNum)
			}
		}()
	}
	lo.Info("Embedded workers started", "count", numWorkers)
	return workers, workerCancel, true
}

// gracefulShutdown performs the exact reverse order of start: campaign-stats
// subscriber stop; for each of the three processors — cancel ctx then Stop();
// if workerCancel != nil — cancel + Close() each worker; finally
// server.Shutdown(). Every "Stopping…/stopped" log line is preserved.
func gracefulShutdown(
	lo logf.Logger,
	app *handlers.App,
	procs *processorHandles,
	workers []*worker.Worker,
	workerCancel context.CancelFunc,
	server *fasthttp.Server,
) {
	lo.Info("Shutting down...")

	// Stop campaign stats subscriber
	lo.Info("Stopping campaign stats subscriber...")
	app.StopCampaignStatsSubscriber()
	lo.Info("Campaign stats subscriber stopped")

	// Stop chat reset processor
	lo.Info("Stopping chat reset processor...")
	procs.chatResetStop()
	procs.chatReset.Stop()
	lo.Info("Chat reset processor stopped")

	// Stop scheduled message processor
	lo.Info("Stopping scheduled message processor...")
	procs.scheduledMsgStop()
	procs.scheduledMsg.Stop()
	lo.Info("Scheduled message processor stopped")

	// Stop GOWA history sync processor
	lo.Info("Stopping GOWA history sync processor...")
	procs.gowaHistoryStop()
	procs.gowaHistory.Stop()
	lo.Info("GOWA history sync processor stopped")

	// Stop GOWA webhook inbox processor
	lo.Info("Stopping GOWA webhook inbox processor...")
	procs.gowaWebhookStop()
	procs.gowaWebhook.Stop()
	lo.Info("GOWA webhook inbox processor stopped")

	// Stop workers first
	if workerCancel != nil {
		lo.Info("Stopping workers...", "count", len(workers))
		workerCancel()
		for _, w := range workers {
			_ = w.Close()
		}
		lo.Info("Workers stopped")
	}

	// Then stop server
	lo.Info("Stopping server...")
	if err := server.Shutdown(); err != nil {
		lo.Error("Server shutdown error", "error", err)
	}
	lo.Info("Server stopped")
}

// waitForShutdownSignal blocks until SIGINT/SIGTERM is received and returns
// the signal. Used by the server command (via waitForShutdown) and the worker
// command (via its select on the same signals).
func waitForShutdownSignal() os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return <-quit
}

// spawnStandaloneWorkers builds workerCount workers for the worker command,
// starts each in its own goroutine (sending Run's result to errCh), and
// returns the slice. Fatal-on-creation-error preserves the prior inline
// behavior. Unlike startEmbeddedWorkers (server-only), these run until the
// caller cancels ctx or errCh delivers a non-Canceled error.
func spawnStandaloneWorkers(
	cfg *config.Config,
	db *gorm.DB,
	rdb *redis.Client,
	lo logf.Logger,
	waRegistry *whatsapp.Registry,
	workerCount int,
	ctx context.Context,
	errCh chan<- error,
) []*worker.Worker {
	workers := make([]*worker.Worker, workerCount)
	for i := 0; i < workerCount; i++ {
		w, err := worker.New(cfg, db, rdb, lo, waRegistry)
		if err != nil {
			lo.Fatal("Failed to create worker", "error", err, "worker_num", i+1)
		}
		workers[i] = w

		go func(workerNum int) {
			lo.Info("Worker started", "worker_num", workerNum)
			errCh <- w.Run(ctx)
		}(i + 1)
	}
	return workers
}

// stopWorkers closes each worker in ws, logging any close error. Used by the
// worker command's cleanup path. (The server command's worker teardown is part
// of gracefulShutdown.)
func stopWorkers(ws []*worker.Worker, lo logf.Logger) {
	lo.Info("Shutting down workers...")
	for _, w := range ws {
		if w == nil {
			continue
		}
		if err := w.Close(); err != nil {
			lo.Error("Error closing worker", "error", err)
		}
	}
	lo.Info("Workers stopped")
}
