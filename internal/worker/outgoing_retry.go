package worker

import (
	"context"
	"errors"
	"time"

	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	outgoingRetryPollInterval = 10 * time.Second
	outgoingRetryBatchSize    = 50
)

type RetryOutgoingMessageFunc func(ctx context.Context, msgID uuid.UUID) error

type OutgoingRetryWorker struct {
	db          *gorm.DB
	redis       *redis.Client
	rq          *queue.OutgoingRetryQueue
	log         logf.Logger
	processFunc RetryOutgoingMessageFunc
}

func NewOutgoingRetryWorker(
	db *gorm.DB,
	rdb *redis.Client,
	log logf.Logger,
	processFunc RetryOutgoingMessageFunc,
) *OutgoingRetryWorker {
	return &OutgoingRetryWorker{
		db:          db,
		redis:       rdb,
		rq:          queue.NewOutgoingRetryQueue(rdb, log),
		log:         log,
		processFunc: processFunc,
	}
}

func (w *OutgoingRetryWorker) Run(ctx context.Context) {
	w.log.Info("Outgoing retry worker starting",
		"poll_interval", outgoingRetryPollInterval,
		"max_retries", queue.MaxOutgoingRetries,
	)

	w.processBatch(ctx)

	ticker := time.NewTicker(outgoingRetryPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Outgoing retry worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutgoingRetryWorker) processBatch(ctx context.Context) {
	entries, err := w.rq.PopReady(ctx, outgoingRetryBatchSize)
	if err != nil {
		w.log.Error("Outgoing retry: failed to pop ready entries", "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	w.log.Debug("Outgoing retry: processing entries", "count", len(entries))

	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}

		msgID, err := uuid.Parse(entry.MessageID)
		if err != nil {
			w.log.Error("Outgoing retry: invalid message ID, discarding entry",
				"entry_id", entry.ID,
				"message_id", entry.MessageID,
			)
			_ = w.rq.Ack(ctx, entry.ID)
			continue
		}

		if err := w.processFunc(ctx, msgID); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			entry.LastError = err.Error()

			if queue.IsPermanentError(err) {
				_ = w.rq.Ack(ctx, entry.ID)
				w.log.Error("Outgoing retry: permanent error, discarding entry",
					"entry_id", entry.ID,
					"message_id", entry.MessageID,
					"error", err,
				)
				continue
			}

			if requeueErr := w.rq.Requeue(ctx, entry); requeueErr != nil {
				w.log.Error("Outgoing retry: failed to requeue entry",
					"entry_id", entry.ID,
					"message_id", entry.MessageID,
					"error", requeueErr,
				)
			}
			continue
		}

		if ackErr := w.rq.Ack(ctx, entry.ID); ackErr != nil {
			w.log.Error("Outgoing retry: failed to ack successful retry",
				"entry_id", entry.ID,
				"message_id", entry.MessageID,
				"error", ackErr,
			)
		}

		w.log.Info("Outgoing retry: message retry succeeded",
			"entry_id", entry.ID,
			"message_id", entry.MessageID,
			"attempt", entry.Attempt,
		)
	}
}
