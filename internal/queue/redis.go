package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

const (
	// StreamName is the Redis stream for campaign jobs.
	StreamName = "whatomate:campaigns"

	// ConsumerGroup is the consumer group name for campaign workers.
	ConsumerGroup = "campaign-workers"

	// DeadLetterStreamName stores campaign jobs that exceeded retry limits.
	DeadLetterStreamName = StreamName + ":dlq"

	// InboundMediaStreamName is the Redis stream for inbound media recovery jobs.
	InboundMediaStreamName = "whatomate:inbound_media"

	// InboundMediaConsumerGroup is the consumer group name for inbound media workers.
	InboundMediaConsumerGroup = "inbound-media-workers"

	// InboundMediaDeadLetterStreamName stores inbound-media jobs that exceeded retry limits.
	InboundMediaDeadLetterStreamName = InboundMediaStreamName + ":dlq"

	// BlockTimeout is how long to block waiting for new messages.
	BlockTimeout = 5 * time.Second

	// ClaimMinIdleTime is the minimum idle time before claiming a pending message.
	ClaimMinIdleTime = 5 * time.Minute

	// PendingClaimInterval controls how often the consumer checks stale pending jobs.
	PendingClaimInterval = 30 * time.Second

	// MaxDeliveryAttempts is the number of retries before moving a message to DLQ.
	MaxDeliveryAttempts = int64(5)
)

// RedisQueue implements the Queue interface using Redis Streams.
type RedisQueue struct {
	client *redis.Client
	log    logf.Logger
}

// NewRedisQueue creates a new Redis queue.
func NewRedisQueue(client *redis.Client, log logf.Logger) *RedisQueue {
	return &RedisQueue{
		client: client,
		log:    log,
	}
}

// EnqueueRecipient adds a single recipient job to the queue.
func (q *RedisQueue) EnqueueRecipient(ctx context.Context, job *RecipientJob) error {
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now()
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal recipient job: %w", err)
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{
			"type":    string(JobTypeRecipient),
			"payload": string(payload),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to enqueue recipient job: %w", err)
	}

	return nil
}

// EnqueueRecipients adds multiple recipient jobs to the queue using pipeline.
func (q *RedisQueue) EnqueueRecipients(ctx context.Context, jobs []*RecipientJob) error {
	if len(jobs) == 0 {
		return nil
	}

	pipe := q.client.Pipeline()
	now := time.Now()

	for _, job := range jobs {
		if job.EnqueuedAt.IsZero() {
			job.EnqueuedAt = now
		}

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("failed to marshal recipient job: %w", err)
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: StreamName,
			Values: map[string]interface{}{
				"type":    string(JobTypeRecipient),
				"payload": string(payload),
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue recipient jobs: %w", err)
	}

	q.log.Info("Recipient jobs enqueued", "count", len(jobs), "campaign_id", jobs[0].CampaignID)
	return nil
}

// EnqueueInboundMedia adds a single inbound-media recovery job to the queue.
func (q *RedisQueue) EnqueueInboundMedia(ctx context.Context, job *InboundMediaJob) error {
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now()
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal inbound media job: %w", err)
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: InboundMediaStreamName,
		Values: map[string]interface{}{
			"type":    string(JobTypeInboundMedia),
			"payload": string(payload),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to enqueue inbound media job: %w", err)
	}

	return nil
}

// EnqueueContactRepair adds a single direct-contact repair job to the queue.
func (q *RedisQueue) EnqueueContactRepair(ctx context.Context, job *ContactRepairJob) error {
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now()
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal contact repair job: %w", err)
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{
			"type":    string(JobTypeContactRepair),
			"payload": string(payload),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to enqueue contact repair job: %w", err)
	}

	return nil
}

// Close closes the queue connection.
func (q *RedisQueue) Close() error {
	return nil // Redis client is managed externally.
}

// RedisConsumer implements the Consumer interface using Redis Streams.
type RedisConsumer struct {
	client               *redis.Client
	log                  logf.Logger
	consumerID           string
	streamName           string
	consumerGroup        string
	deadLetterStreamName string
	maxDeliveryAttempts  int64
}

type processMessageError struct {
	err       error
	permanent bool
}

func (e *processMessageError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *processMessageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newPermanentProcessError(err error) error {
	if err == nil {
		return nil
	}
	return &processMessageError{err: err, permanent: true}
}

func isPermanentProcessError(err error) bool {
	var perr *processMessageError
	return errors.As(err, &perr) && perr.permanent
}

// NewPermanentError marks an error as permanently non-retryable for Redis consumers.
func NewPermanentError(err error) error {
	return newPermanentProcessError(err)
}

// IsPermanentError reports whether err is marked as permanently non-retryable.
func IsPermanentError(err error) bool {
	return isPermanentProcessError(err)
}

type consumerOptions struct {
	streamName           string
	consumerGroup        string
	deadLetterStreamName string
}

func newRedisConsumer(client *redis.Client, log logf.Logger, opts consumerOptions) (*RedisConsumer, error) {
	hostname, _ := os.Hostname()
	consumerID := fmt.Sprintf("worker-%s-%d", hostname, os.Getpid())

	consumer := &RedisConsumer{
		client:               client,
		log:                  log,
		consumerID:           consumerID,
		streamName:           opts.streamName,
		consumerGroup:        opts.consumerGroup,
		deadLetterStreamName: opts.deadLetterStreamName,
		maxDeliveryAttempts:  MaxDeliveryAttempts,
	}

	ctx := context.Background()
	err := client.XGroupCreateMkStream(ctx, consumer.streamName, consumer.consumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return nil, fmt.Errorf("failed to create consumer group %s on stream %s: %w", consumer.consumerGroup, consumer.streamName, err)
	}

	log.Info("Redis consumer initialized", "consumer_id", consumerID, "stream", consumer.streamName, "group", consumer.consumerGroup)
	return consumer, nil
}

// NewRedisConsumer creates a consumer for campaign recipient jobs.
func NewRedisConsumer(client *redis.Client, log logf.Logger) (*RedisConsumer, error) {
	return newRedisConsumer(client, log, consumerOptions{
		streamName:           StreamName,
		consumerGroup:        ConsumerGroup,
		deadLetterStreamName: DeadLetterStreamName,
	})
}

// NewRedisInboundMediaConsumer creates a consumer for inbound-media recovery jobs.
func NewRedisInboundMediaConsumer(client *redis.Client, log logf.Logger) (*RedisConsumer, error) {
	return newRedisConsumer(client, log, consumerOptions{
		streamName:           InboundMediaStreamName,
		consumerGroup:        InboundMediaConsumerGroup,
		deadLetterStreamName: InboundMediaDeadLetterStreamName,
	})
}

// Consume starts consuming jobs from the queue.
func (c *RedisConsumer) Consume(ctx context.Context, handler JobHandler) error {
	c.log.Info("Starting to consume jobs", "consumer_id", c.consumerID, "stream", c.streamName, "group", c.consumerGroup)

	if err := c.claimPendingMessages(ctx, handler); err != nil {
		c.log.Warn("Failed to claim pending messages", "stream", c.streamName, "group", c.consumerGroup, "error", err)
	}

	lastPendingClaim := time.Now()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Consumer shutting down", "stream", c.streamName, "group", c.consumerGroup)
			return ctx.Err()
		default:
		}

		if time.Since(lastPendingClaim) >= PendingClaimInterval {
			if err := c.claimPendingMessages(ctx, handler); err != nil {
				c.log.Warn("Failed periodic pending-claim cycle", "stream", c.streamName, "group", c.consumerGroup, "error", err)
			}
			lastPendingClaim = time.Now()
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.consumerGroup,
			Consumer: c.consumerID,
			Streams:  []string{c.streamName, ">"},
			Count:    1,
			Block:    BlockTimeout,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.log.Error("Failed to read from stream", "stream", c.streamName, "group", c.consumerGroup, "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				if err := c.processMessage(ctx, msg, handler); err != nil {
					c.log.Error("Failed to process message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", msg.ID)
					if isPermanentProcessError(err) {
						if dlqErr := c.moveToDeadLetter(ctx, msg, err.Error(), 0); dlqErr != nil {
							c.log.Error("Failed to move permanently invalid message to DLQ", "stream", c.streamName, "group", c.consumerGroup, "error", dlqErr, "message_id", msg.ID)
						}
					}
					continue
				}

				if err := c.client.XAck(ctx, c.streamName, c.consumerGroup, msg.ID).Err(); err != nil {
					c.log.Error("Failed to ACK message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", msg.ID)
				}
			}
		}
	}
}

// claimPendingMessages claims stale pending messages from crashed workers.
func (c *RedisConsumer) claimPendingMessages(ctx context.Context, handler JobHandler) error {
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.streamName,
		Group:  c.consumerGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
		Idle:   ClaimMinIdleTime,
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	if len(pending) == 0 {
		return nil
	}

	c.log.Info("Found stale pending messages to claim", "stream", c.streamName, "group", c.consumerGroup, "count", len(pending))

	for _, p := range pending {
		if p.RetryCount >= c.maxDeliveryAttempts {
			if err := c.movePendingToDeadLetter(ctx, p.ID, fmt.Sprintf("max delivery attempts exceeded (%d)", p.RetryCount), p.RetryCount); err != nil {
				c.log.Error("Failed to dead-letter stale message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", p.ID)
			}
			continue
		}

		messages, err := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   c.streamName,
			Group:    c.consumerGroup,
			Consumer: c.consumerID,
			MinIdle:  ClaimMinIdleTime,
			Messages: []string{p.ID},
		}).Result()
		if err != nil {
			c.log.Error("Failed to claim message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", p.ID)
			continue
		}

		for _, msg := range messages {
			if err := c.processMessage(ctx, msg, handler); err != nil {
				c.log.Error("Failed to process claimed message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", msg.ID)
				if isPermanentProcessError(err) {
					if dlqErr := c.moveToDeadLetter(ctx, msg, err.Error(), p.RetryCount); dlqErr != nil {
						c.log.Error("Failed to move permanently invalid claimed message to DLQ", "stream", c.streamName, "group", c.consumerGroup, "error", dlqErr, "message_id", msg.ID)
					}
				}
				continue
			}

			if err := c.client.XAck(ctx, c.streamName, c.consumerGroup, msg.ID).Err(); err != nil {
				c.log.Error("Failed to ACK claimed message", "stream", c.streamName, "group", c.consumerGroup, "error", err, "message_id", msg.ID)
			}
		}
	}

	return nil
}

// processMessage processes a single message from the stream.
func (c *RedisConsumer) processMessage(ctx context.Context, msg redis.XMessage, handler JobHandler) error {
	jobType, ok := msg.Values["type"].(string)
	if !ok {
		return newPermanentProcessError(fmt.Errorf("invalid message: missing type"))
	}

	payload, ok := msg.Values["payload"].(string)
	if !ok {
		return newPermanentProcessError(fmt.Errorf("invalid message: missing payload"))
	}

	switch JobType(jobType) {
	case JobTypeRecipient:
		var job RecipientJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return newPermanentProcessError(fmt.Errorf("failed to unmarshal recipient job: %w", err))
		}
		c.log.Debug("Processing recipient job", "stream", c.streamName, "campaign_id", job.CampaignID, "recipient_id", job.RecipientID, "message_id", msg.ID)
		return handler.HandleRecipientJob(ctx, &job)

	case JobTypeInboundMedia:
		var job InboundMediaJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return newPermanentProcessError(fmt.Errorf("failed to unmarshal inbound media job: %w", err))
		}
		c.log.Debug("Processing inbound media job", "stream", c.streamName, "message_id", job.MessageID, "wa_message_id", job.WhatsAppMessageID, "redis_message_id", msg.ID)
		return handler.HandleInboundMediaJob(ctx, &job)

	case JobTypeContactRepair:
		var job ContactRepairJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return newPermanentProcessError(fmt.Errorf("failed to unmarshal contact repair job: %w", err))
		}
		c.log.Debug("Processing contact repair job", "stream", c.streamName, "contact_id", job.ContactID, "organization_id", job.OrganizationID, "redis_message_id", msg.ID)
		return handler.HandleContactRepairJob(ctx, &job)

	default:
		return newPermanentProcessError(fmt.Errorf("unknown job type: %s", jobType))
	}
}

func (c *RedisConsumer) movePendingToDeadLetter(ctx context.Context, messageID, reason string, attempts int64) error {
	messages, err := c.client.XRangeN(ctx, c.streamName, messageID, messageID, 1).Result()
	if err != nil {
		return fmt.Errorf("failed to load pending message for DLQ: %w", err)
	}

	if len(messages) == 0 {
		if ackErr := c.client.XAck(ctx, c.streamName, c.consumerGroup, messageID).Err(); ackErr != nil {
			return fmt.Errorf("failed to ack missing pending message %s: %w", messageID, ackErr)
		}
		return nil
	}

	return c.moveToDeadLetter(ctx, messages[0], reason, attempts)
}

func (c *RedisConsumer) moveToDeadLetter(ctx context.Context, msg redis.XMessage, reason string, attempts int64) error {
	values := map[string]interface{}{
		"original_stream": c.streamName,
		"original_id":     msg.ID,
		"reason":          reason,
		"attempts":        attempts,
		"failed_at":       time.Now().UTC().Format(time.RFC3339Nano),
	}

	if jobType, ok := msg.Values["type"]; ok {
		values["type"] = jobType
	}
	if payload, ok := msg.Values["payload"]; ok {
		values["payload"] = payload
	}

	if _, err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: c.deadLetterStreamName,
		Values: values,
	}).Result(); err != nil {
		return fmt.Errorf("failed to write dead-letter message: %w", err)
	}

	if err := c.client.XAck(ctx, c.streamName, c.consumerGroup, msg.ID).Err(); err != nil {
		return fmt.Errorf("failed to ack dead-lettered message: %w", err)
	}
	if err := c.client.XDel(ctx, c.streamName, msg.ID).Err(); err != nil {
		c.log.Warn("Failed to delete dead-lettered message from source stream", "stream", c.streamName, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
	}

	c.log.Warn("Message moved to dead-letter stream", "stream", c.streamName, "dlq_stream", c.deadLetterStreamName, "message_id", msg.ID, "reason", reason, "attempts", attempts)
	return nil
}

// Close closes the consumer connection.
func (c *RedisConsumer) Close() error {
	return nil // Redis client is managed externally.
}
