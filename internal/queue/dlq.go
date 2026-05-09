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
	InboundDLQZSetKey       = "whatomate:inbound_dlq:queue"
	InboundDLQPayloadPrefix = "whatomate:inbound_dlq:payload:"
	InboundDLQCounterKey    = "whatomate:inbound_dlq:metrics:total_enqueued"
	InboundDLQPermFailKey   = "whatomate:inbound_dlq:metrics:permanent_failures"
	InboundDLQRetryOKKey    = "whatomate:inbound_dlq:metrics:retry_successes"
	MaxDLQRetries           = 3
	dlqPayloadTTL           = 48 * time.Hour
)

var DLQBackoffSchedule = [3]time.Duration{
	30 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
}

type InboundDLQEntry struct {
	ID            string          `json:"id"`
	PhoneNumberID string          `json:"phone_number_id"`
	ProfileName   string          `json:"profile_name"`
	RawMessage    json.RawMessage `json:"raw_message"`
	Attempt       int             `json:"attempt"`
	FirstFailedAt time.Time       `json:"first_failed_at"`
	LastError     string          `json:"last_error"`
	EnqueuedAt    time.Time       `json:"enqueued_at"`
}

type InboundDLQ struct {
	client *redis.Client
	log    logf.Logger
}

func NewInboundDLQ(client *redis.Client, log logf.Logger) *InboundDLQ {
	return &InboundDLQ{client: client, log: log}
}

func (d *InboundDLQ) Push(ctx context.Context, entry *InboundDLQEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.EnqueuedAt.IsZero() {
		entry.EnqueuedAt = time.Now()
	}
	if entry.FirstFailedAt.IsZero() {
		entry.FirstFailedAt = time.Now()
	}

	backoff := dlqBackoff(entry.Attempt)
	nextRetry := time.Now().Add(backoff)

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("dlq: marshal entry: %w", err)
	}

	pipe := d.client.Pipeline()
	pipe.Set(ctx, InboundDLQPayloadPrefix+entry.ID, payload, dlqPayloadTTL)
	pipe.ZAdd(ctx, InboundDLQZSetKey, redis.Z{
		Score:  float64(nextRetry.UnixNano()),
		Member: entry.ID,
	})
	pipe.Incr(ctx, InboundDLQCounterKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("dlq: push entry: %w", err)
	}

	d.log.Warn("Message pushed to inbound DLQ",
		"entry_id", entry.ID,
		"attempt", entry.Attempt,
		"next_retry", nextRetry,
		"phone_number_id", entry.PhoneNumberID,
	)
	return nil
}

func (d *InboundDLQ) PopReady(ctx context.Context, limit int) ([]*InboundDLQEntry, error) {
	now := float64(time.Now().UnixNano())

	ids, err := d.client.ZRangeByScore(ctx, InboundDLQZSetKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%f", now),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("dlq: range by score: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	entries := make([]*InboundDLQEntry, 0, len(ids))
	for _, id := range ids {
		payload, err := d.client.Get(ctx, InboundDLQPayloadPrefix+id).Bytes()
		if err != nil {
			if err == redis.Nil {
				d.client.ZRem(ctx, InboundDLQZSetKey, id)
				continue
			}
			return entries, fmt.Errorf("dlq: get payload %s: %w", id, err)
		}

		var entry InboundDLQEntry
		if err := json.Unmarshal(payload, &entry); err != nil {
			d.client.ZRem(ctx, InboundDLQZSetKey, id)
			d.client.Del(ctx, InboundDLQPayloadPrefix+id)
			continue
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

func (d *InboundDLQ) Ack(ctx context.Context, entryID string) error {
	pipe := d.client.Pipeline()
	pipe.ZRem(ctx, InboundDLQZSetKey, entryID)
	pipe.Del(ctx, InboundDLQPayloadPrefix+entryID)
	pipe.Incr(ctx, InboundDLQRetryOKKey)
	_, err := pipe.Exec(ctx)
	return err
}

func (d *InboundDLQ) Requeue(ctx context.Context, entry *InboundDLQEntry) error {
	entry.Attempt++

	if entry.Attempt >= MaxDLQRetries {
		return d.markPermanentFailure(ctx, entry)
	}

	backoff := dlqBackoff(entry.Attempt)
	nextRetry := time.Now().Add(backoff)

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("dlq: requeue marshal: %w", err)
	}

	pipe := d.client.Pipeline()
	pipe.Set(ctx, InboundDLQPayloadPrefix+entry.ID, payload, dlqPayloadTTL)
	pipe.ZAdd(ctx, InboundDLQZSetKey, redis.Z{
		Score:  float64(nextRetry.UnixNano()),
		Member: entry.ID,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("dlq: requeue: %w", err)
	}

	d.log.Warn("DLQ entry requeued for retry",
		"entry_id", entry.ID,
		"attempt", entry.Attempt,
		"next_retry", nextRetry,
	)
	return nil
}

func (d *InboundDLQ) markPermanentFailure(ctx context.Context, entry *InboundDLQEntry) error {
	pipe := d.client.Pipeline()
	pipe.ZRem(ctx, InboundDLQZSetKey, entry.ID)
	pipe.Del(ctx, InboundDLQPayloadPrefix+entry.ID)
	pipe.Incr(ctx, InboundDLQPermFailKey)
	_, err := pipe.Exec(ctx)

	d.log.Error("DLQ entry permanently failed after max retries",
		"entry_id", entry.ID,
		"attempts", entry.Attempt,
		"phone_number_id", entry.PhoneNumberID,
		"first_failed_at", entry.FirstFailedAt,
		"last_error", entry.LastError,
		"raw_message_length", len(entry.RawMessage),
	)
	return err
}

func (d *InboundDLQ) Size(ctx context.Context) (int64, error) {
	return d.client.ZCard(ctx, InboundDLQZSetKey).Result()
}

type DLQMetrics struct {
	QueueSize         int64 `json:"queue_size"`
	TotalEnqueued     int64 `json:"total_enqueued"`
	RetrySuccesses    int64 `json:"retry_successes"`
	PermanentFailures int64 `json:"permanent_failures"`
}

func (d *InboundDLQ) Metrics(ctx context.Context) DLQMetrics {
	pipe := d.client.Pipeline()
	eCmd := pipe.Get(ctx, InboundDLQCounterKey)
	rCmd := pipe.Get(ctx, InboundDLQRetryOKKey)
	pCmd := pipe.Get(ctx, InboundDLQPermFailKey)
	_, _ = pipe.Exec(ctx)

	var m DLQMetrics
	m.QueueSize, _ = d.Size(ctx)
	if v, e := eCmd.Int64(); e == nil {
		m.TotalEnqueued = v
	}
	if v, e := rCmd.Int64(); e == nil {
		m.RetrySuccesses = v
	}
	if v, e := pCmd.Int64(); e == nil {
		m.PermanentFailures = v
	}
	return m
}

func dlqBackoff(attempt int) time.Duration {
	if attempt >= 0 && attempt < len(DLQBackoffSchedule) {
		return DLQBackoffSchedule[attempt]
	}
	return DLQBackoffSchedule[len(DLQBackoffSchedule)-1]
}
