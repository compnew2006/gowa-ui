package worker

import (
	"context"
	"errors"
	"time"

	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	dlqPollInterval = 10 * time.Second
	dlqBatchSize    = 50
)

type RetryInboundMessageFunc func(ctx context.Context, entry *queue.InboundDLQEntry) error

type DLQRetyWorker struct {
	db          *gorm.DB
	redis       *redis.Client
	dlq         *queue.InboundDLQ
	log         logf.Logger
	processFunc RetryInboundMessageFunc
}

func NewDLQRetyWorker(
	db *gorm.DB,
	rdb *redis.Client,
	log logf.Logger,
	processFunc RetryInboundMessageFunc,
) *DLQRetyWorker {
	return &DLQRetyWorker{
		db:          db,
		redis:       rdb,
		dlq:         queue.NewInboundDLQ(rdb, log),
		log:         log,
		processFunc: processFunc,
	}
}

func (w *DLQRetyWorker) Run(ctx context.Context) {
	w.log.Info("DLQ retry worker starting", "poll_interval", dlqPollInterval, "max_retries", queue.MaxDLQRetries)

	w.processBatch(ctx)

	ticker := time.NewTicker(dlqPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("DLQ retry worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *DLQRetyWorker) processBatch(ctx context.Context) {
	entries, err := w.dlq.PopReady(ctx, dlqBatchSize)
	if err != nil {
		w.log.Error("DLQ: failed to pop ready entries", "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	w.log.Debug("DLQ: processing entries", "count", len(entries))

	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}

		if err := w.processFunc(ctx, entry); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			entry.LastError = err.Error()

			if queue.IsPermanentError(err) {
				_ = w.dlq.Ack(ctx, entry.ID)
				w.log.Error("DLQ: permanent error, discarding entry",
					"entry_id", entry.ID,
					"error", err,
				)
				continue
			}

			if requeueErr := w.dlq.Requeue(ctx, entry); requeueErr != nil {
				w.log.Error("DLQ: failed to requeue entry",
					"entry_id", entry.ID,
					"error", requeueErr,
				)
			}
			continue
		}

		if ackErr := w.dlq.Ack(ctx, entry.ID); ackErr != nil {
			w.log.Error("DLQ: failed to ack successful retry",
				"entry_id", entry.ID,
				"error", ackErr,
			)
		}

		w.log.Info("DLQ: message retry succeeded",
			"entry_id", entry.ID,
			"attempt", entry.Attempt,
			"phone_number_id", entry.PhoneNumberID,
		)
	}
}
