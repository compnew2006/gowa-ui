package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

const (
	OutgoingRetryZSetKey       = "whatomate:outgoing_retry:queue"
	OutgoingRetryPayloadPrefix = "whatomate:outgoing_retry:payload:"
	OutgoingRetryCounterKey    = "whatomate:outgoing_retry:metrics:total_enqueued"
	OutgoingRetryPermFailKey   = "whatomate:outgoing_retry:metrics:permanent_failures"
	OutgoingRetrySuccessKey    = "whatomate:outgoing_retry:metrics:successes"
	MaxOutgoingRetries         = 3
	outgoingRetryPayloadTTL    = 48 * time.Hour
)

type OutgoingRetryEntry struct {
	ID            string    `json:"id"`
	MessageID     string    `json:"message_id"`
	OrgID         string    `json:"org_id"`
	Attempt       int       `json:"attempt"`
	FirstFailedAt time.Time `json:"first_failed_at"`
	LastError     string    `json:"last_error"`
	EnqueuedAt    time.Time `json:"enqueued_at"`
}

type OutgoingRetryQueue struct {
	client *redis.Client
	log    logf.Logger
}

func NewOutgoingRetryQueue(client *redis.Client, log logf.Logger) *OutgoingRetryQueue {
	return &OutgoingRetryQueue{client: client, log: log}
}

func (q *OutgoingRetryQueue) Push(ctx context.Context, entry *OutgoingRetryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.EnqueuedAt.IsZero() {
		entry.EnqueuedAt = time.Now()
	}
	if entry.FirstFailedAt.IsZero() {
		entry.FirstFailedAt = time.Now()
	}

	backoff := outgoingRetryBackoff(entry.Attempt)
	nextRetry := time.Now().Add(backoff)

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("outgoing_retry: marshal entry: %w", err)
	}

	pipe := q.client.Pipeline()
	pipe.Set(ctx, OutgoingRetryPayloadPrefix+entry.ID, payload, outgoingRetryPayloadTTL)
	pipe.ZAdd(ctx, OutgoingRetryZSetKey, redis.Z{
		Score:  float64(nextRetry.UnixNano()),
		Member: entry.ID,
	})
	pipe.Incr(ctx, OutgoingRetryCounterKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("outgoing_retry: push entry: %w", err)
	}

	q.log.Warn("Outgoing message queued for retry",
		"entry_id", entry.ID,
		"message_id", entry.MessageID,
		"attempt", entry.Attempt,
		"next_retry", nextRetry,
		"org_id", entry.OrgID,
	)
	return nil
}

func (q *OutgoingRetryQueue) PopReady(ctx context.Context, limit int) ([]*OutgoingRetryEntry, error) {
	now := float64(time.Now().UnixNano())

	ids, err := q.client.ZRangeByScore(ctx, OutgoingRetryZSetKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%f", now),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("outgoing_retry: range by score: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	entries := make([]*OutgoingRetryEntry, 0, len(ids))
	for _, id := range ids {
		payload, err := q.client.Get(ctx, OutgoingRetryPayloadPrefix+id).Bytes()
		if err != nil {
			if err == redis.Nil {
				q.client.ZRem(ctx, OutgoingRetryZSetKey, id)
				continue
			}
			return entries, fmt.Errorf("outgoing_retry: get payload %s: %w", id, err)
		}

		var entry OutgoingRetryEntry
		if err := json.Unmarshal(payload, &entry); err != nil {
			q.client.ZRem(ctx, OutgoingRetryZSetKey, id)
			q.client.Del(ctx, OutgoingRetryPayloadPrefix+id)
			continue
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

func (q *OutgoingRetryQueue) Ack(ctx context.Context, entryID string) error {
	pipe := q.client.Pipeline()
	pipe.ZRem(ctx, OutgoingRetryZSetKey, entryID)
	pipe.Del(ctx, OutgoingRetryPayloadPrefix+entryID)
	pipe.Incr(ctx, OutgoingRetrySuccessKey)
	_, err := pipe.Exec(ctx)
	return err
}

func (q *OutgoingRetryQueue) Requeue(ctx context.Context, entry *OutgoingRetryEntry) error {
	entry.Attempt++

	if entry.Attempt >= MaxOutgoingRetries {
		return q.markPermanentFailure(ctx, entry)
	}

	backoff := outgoingRetryBackoff(entry.Attempt)
	nextRetry := time.Now().Add(backoff)

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("outgoing_retry: requeue marshal: %w", err)
	}

	pipe := q.client.Pipeline()
	pipe.Set(ctx, OutgoingRetryPayloadPrefix+entry.ID, payload, outgoingRetryPayloadTTL)
	pipe.ZAdd(ctx, OutgoingRetryZSetKey, redis.Z{
		Score:  float64(nextRetry.UnixNano()),
		Member: entry.ID,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("outgoing_retry: requeue: %w", err)
	}

	q.log.Warn("Outgoing retry entry requeued",
		"entry_id", entry.ID,
		"message_id", entry.MessageID,
		"attempt", entry.Attempt,
		"next_retry", nextRetry,
	)
	return nil
}

func (q *OutgoingRetryQueue) markPermanentFailure(ctx context.Context, entry *OutgoingRetryEntry) error {
	pipe := q.client.Pipeline()
	pipe.ZRem(ctx, OutgoingRetryZSetKey, entry.ID)
	pipe.Del(ctx, OutgoingRetryPayloadPrefix+entry.ID)
	pipe.Incr(ctx, OutgoingRetryPermFailKey)
	_, err := pipe.Exec(ctx)

	q.log.Error("Outgoing message permanently failed after max retries",
		"entry_id", entry.ID,
		"message_id", entry.MessageID,
		"attempts", entry.Attempt,
		"org_id", entry.OrgID,
		"first_failed_at", entry.FirstFailedAt,
		"last_error", entry.LastError,
	)
	return err
}

func (q *OutgoingRetryQueue) Size(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, OutgoingRetryZSetKey).Result()
}

func outgoingRetryBackoff(attempt int) time.Duration {
	if attempt >= 0 && attempt < len(DLQBackoffSchedule) {
		return DLQBackoffSchedule[attempt]
	}
	return DLQBackoffSchedule[len(DLQBackoffSchedule)-1]
}
