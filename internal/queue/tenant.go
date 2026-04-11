package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

const (
	// TenantCampaignStreamPrefix is the shared prefix for per-organization campaign streams.
	TenantCampaignStreamPrefix = "whatomate:org:"

	// TenantCampaignStreamSuffix is the suffix for per-organization campaign streams.
	TenantCampaignStreamSuffix = ":campaigns"

	// CampaignStreamPattern discovers per-organization campaign streams.
	CampaignStreamPattern = TenantCampaignStreamPrefix + "*" + TenantCampaignStreamSuffix

	tenantStreamIdleDelay = time.Second
)

// CampaignStreamName returns the Redis stream name for an organization's campaign jobs.
func CampaignStreamName(orgID uuid.UUID) string {
	return TenantCampaignStreamPrefix + orgID.String() + TenantCampaignStreamSuffix
}

// CampaignDeadLetterStreamName returns the dead-letter stream for an organization's campaign jobs.
func CampaignDeadLetterStreamName(orgID uuid.UUID) string {
	return CampaignStreamName(orgID) + ":dlq"
}

// TenantQueueManager routes organization-scoped campaign jobs into per-tenant streams.
type TenantQueueManager struct {
	client *redis.Client
	log    logf.Logger
}

var _ Queue = (*TenantQueueManager)(nil)
var _ Consumer = (*TenantCampaignConsumer)(nil)

// NewTenantQueueManager creates a queue manager that routes campaign work by organization.
func NewTenantQueueManager(client *redis.Client, log logf.Logger) *TenantQueueManager {
	return &TenantQueueManager{
		client: client,
		log:    log,
	}
}

// EnqueueRecipient adds a single recipient job to the organization's campaign stream.
func (q *TenantQueueManager) EnqueueRecipient(ctx context.Context, job *RecipientJob) error {
	if job == nil {
		return fmt.Errorf("recipient job is nil")
	}
	stream, err := campaignStreamForOrg(job.OrganizationID)
	if err != nil {
		return err
	}
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now().UTC()
	}
	return q.enqueueJSONJob(ctx, stream, JobTypeRecipient, job)
}

// EnqueueRecipients adds multiple recipient jobs, grouping them by organization.
func (q *TenantQueueManager) EnqueueRecipients(ctx context.Context, jobs []*RecipientJob) error {
	if len(jobs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	pipe := q.client.Pipeline()

	for _, job := range jobs {
		if job == nil {
			return fmt.Errorf("recipient job is nil")
		}
		stream, err := campaignStreamForOrg(job.OrganizationID)
		if err != nil {
			return err
		}
		if job.EnqueuedAt.IsZero() {
			job.EnqueuedAt = now
		}

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("failed to marshal recipient job: %w", err)
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{
				"type":    string(JobTypeRecipient),
				"payload": string(payload),
			},
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to enqueue recipient jobs: %w", err)
	}

	q.log.Info("Tenant recipient jobs enqueued", "count", len(jobs))
	return nil
}

// EnqueueInboundMedia keeps inbound-media recovery on the existing global stream in this phase.
func (q *TenantQueueManager) EnqueueInboundMedia(ctx context.Context, job *InboundMediaJob) error {
	if job == nil {
		return fmt.Errorf("inbound media job is nil")
	}
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now().UTC()
	}
	return q.enqueueJSONJob(ctx, InboundMediaStreamName, JobTypeInboundMedia, job)
}

// EnqueueContactRepair routes direct-contact repair jobs into the organization's campaign stream.
func (q *TenantQueueManager) EnqueueContactRepair(ctx context.Context, job *ContactRepairJob) error {
	if job == nil {
		return fmt.Errorf("contact repair job is nil")
	}
	stream, err := campaignStreamForOrg(job.OrganizationID)
	if err != nil {
		return err
	}
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now().UTC()
	}
	return q.enqueueJSONJob(ctx, stream, JobTypeContactRepair, job)
}

// Close closes the queue connection.
func (q *TenantQueueManager) Close() error {
	return nil
}

func (q *TenantQueueManager) enqueueJSONJob(ctx context.Context, stream string, jobType JobType, payloadValue interface{}) error {
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return fmt.Errorf("failed to marshal %s job: %w", jobType, err)
	}

	if _, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"type":    string(jobType),
			"payload": string(payload),
		},
	}).Result(); err != nil {
		return fmt.Errorf("failed to enqueue %s job: %w", jobType, err)
	}

	return nil
}

func campaignStreamForOrg(orgID uuid.UUID) (string, error) {
	if orgID == uuid.Nil {
		return "", fmt.Errorf("organization id is required for tenant campaign queue")
	}
	return CampaignStreamName(orgID), nil
}

// TenantCampaignConsumer consumes campaign/contact-repair jobs from all organization streams.
type TenantCampaignConsumer struct {
	client              *redis.Client
	log                 logf.Logger
	consumerID          string
	consumerGroup       string
	maxDeliveryAttempts int64
}

// NewTenantCampaignConsumer creates a tenant-aware consumer for organization-scoped campaign streams.
func NewTenantCampaignConsumer(client *redis.Client, log logf.Logger) *TenantCampaignConsumer {
	hostname, _ := os.Hostname()
	return &TenantCampaignConsumer{
		client:              client,
		log:                 log,
		consumerID:          fmt.Sprintf("worker-%s-%d", hostname, os.Getpid()),
		consumerGroup:       ConsumerGroup,
		maxDeliveryAttempts: MaxDeliveryAttempts,
	}
}

// Consume starts consuming jobs across all tenant campaign streams.
func (c *TenantCampaignConsumer) Consume(ctx context.Context, handler JobHandler) error {
	c.log.Info("Starting tenant campaign consumer", "consumer_id", c.consumerID, "group", c.consumerGroup, "pattern", CampaignStreamPattern)

	if err := c.claimPendingMessages(ctx, handler); err != nil {
		c.log.Warn("Failed initial tenant pending-claim cycle", "group", c.consumerGroup, "error", err)
	}

	lastPendingClaim := time.Now()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Tenant campaign consumer shutting down", "group", c.consumerGroup)
			return ctx.Err()
		default:
		}

		if gate, ok := handler.(ReadinessGate); ok {
			if err := gate.WaitUntilOperational(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				c.log.Warn("Tenant campaign consumer readiness gate blocked dequeue", "group", c.consumerGroup, "error", err)
				time.Sleep(time.Second)
				continue
			}
		}

		if time.Since(lastPendingClaim) >= PendingClaimInterval {
			if err := c.claimPendingMessages(ctx, handler); err != nil {
				c.log.Warn("Failed tenant pending-claim cycle", "group", c.consumerGroup, "error", err)
			}
			lastPendingClaim = time.Now()
		}

		streams, err := c.discoverCampaignStreams(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.log.Error("Failed to discover tenant campaign streams", "error", err)
			time.Sleep(tenantStreamIdleDelay)
			continue
		}
		if len(streams) == 0 {
			time.Sleep(tenantStreamIdleDelay)
			continue
		}

		readArgs := make([]string, 0, len(streams)*2)
		readArgs = append(readArgs, streams...)
		for range streams {
			readArgs = append(readArgs, ">")
		}

		results, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.consumerGroup,
			Consumer: c.consumerID,
			Streams:  readArgs,
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
			c.log.Error("Failed to read tenant campaign streams", "group", c.consumerGroup, "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, streamResult := range results {
			for _, msg := range streamResult.Messages {
				if err := c.processMessage(ctx, streamResult.Stream, msg, handler); err != nil {
					c.log.Error("Failed to process tenant campaign message", "stream", streamResult.Stream, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
					if isPermanentProcessError(err) {
						if dlqErr := c.moveToDeadLetter(ctx, streamResult.Stream, msg, err.Error(), 0); dlqErr != nil {
							c.log.Error("Failed to dead-letter permanently invalid tenant message", "stream", streamResult.Stream, "group", c.consumerGroup, "message_id", msg.ID, "error", dlqErr)
						}
					}
					continue
				}

				if err := c.client.XAck(ctx, streamResult.Stream, c.consumerGroup, msg.ID).Err(); err != nil {
					c.log.Error("Failed to ACK tenant campaign message", "stream", streamResult.Stream, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
				}
			}
		}
	}
}

func (c *TenantCampaignConsumer) discoverCampaignStreams(ctx context.Context) ([]string, error) {
	iter := c.client.Scan(ctx, 0, CampaignStreamPattern, 0).Iterator()
	streams := make([]string, 0)
	for iter.Next(ctx) {
		streams = append(streams, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan tenant campaign streams: %w", err)
	}

	sort.Strings(streams)
	for _, stream := range streams {
		if err := c.ensureConsumerGroup(ctx, stream); err != nil {
			return nil, err
		}
	}

	return streams, nil
}

func (c *TenantCampaignConsumer) ensureConsumerGroup(ctx context.Context, stream string) error {
	err := c.client.XGroupCreateMkStream(ctx, stream, c.consumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group %s on stream %s: %w", c.consumerGroup, stream, err)
	}
	return nil
}

func (c *TenantCampaignConsumer) claimPendingMessages(ctx context.Context, handler JobHandler) error {
	streams, err := c.discoverCampaignStreams(ctx)
	if err != nil {
		return err
	}

	for _, stream := range streams {
		if err := c.claimPendingMessagesForStream(ctx, stream, handler); err != nil {
			return err
		}
	}

	return nil
}

func (c *TenantCampaignConsumer) claimPendingMessagesForStream(ctx context.Context, stream string, handler JobHandler) error {
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  c.consumerGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
		Idle:   ClaimMinIdleTime,
	}).Result()
	if err != nil {
		return fmt.Errorf("get pending messages for %s: %w", stream, err)
	}
	if len(pending) == 0 {
		return nil
	}

	for _, entry := range pending {
		if entry.RetryCount >= c.maxDeliveryAttempts {
			if err := c.movePendingToDeadLetter(ctx, stream, entry.ID, fmt.Sprintf("max delivery attempts exceeded (%d)", entry.RetryCount), entry.RetryCount); err != nil {
				c.log.Error("Failed to dead-letter stale tenant message", "stream", stream, "group", c.consumerGroup, "message_id", entry.ID, "error", err)
			}
			continue
		}

		messages, err := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   stream,
			Group:    c.consumerGroup,
			Consumer: c.consumerID,
			MinIdle:  ClaimMinIdleTime,
			Messages: []string{entry.ID},
		}).Result()
		if err != nil {
			c.log.Error("Failed to claim tenant pending message", "stream", stream, "group", c.consumerGroup, "message_id", entry.ID, "error", err)
			continue
		}

		for _, msg := range messages {
			if err := c.processMessage(ctx, stream, msg, handler); err != nil {
				c.log.Error("Failed to process claimed tenant message", "stream", stream, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
				if isPermanentProcessError(err) {
					if dlqErr := c.moveToDeadLetter(ctx, stream, msg, err.Error(), entry.RetryCount); dlqErr != nil {
						c.log.Error("Failed to dead-letter claimed tenant message", "stream", stream, "group", c.consumerGroup, "message_id", msg.ID, "error", dlqErr)
					}
				}
				continue
			}

			if err := c.client.XAck(ctx, stream, c.consumerGroup, msg.ID).Err(); err != nil {
				c.log.Error("Failed to ACK claimed tenant message", "stream", stream, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
			}
		}
	}

	return nil
}

func (c *TenantCampaignConsumer) processMessage(ctx context.Context, stream string, msg redis.XMessage, handler JobHandler) error {
	jobType, ok := streamStringValue(msg.Values["type"])
	if !ok {
		return newPermanentProcessError(fmt.Errorf("invalid message: missing type"))
	}

	payload, ok := streamStringValue(msg.Values["payload"])
	if !ok {
		return newPermanentProcessError(fmt.Errorf("invalid message: missing payload"))
	}

	switch JobType(jobType) {
	case JobTypeRecipient:
		var job RecipientJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return newPermanentProcessError(fmt.Errorf("failed to unmarshal recipient job: %w", err))
		}
		c.log.Debug("Processing tenant recipient job", "stream", stream, "campaign_id", job.CampaignID, "recipient_id", job.RecipientID, "message_id", msg.ID)
		return handler.HandleRecipientJob(ctx, &job)

	case JobTypeContactRepair:
		var job ContactRepairJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return newPermanentProcessError(fmt.Errorf("failed to unmarshal contact repair job: %w", err))
		}
		c.log.Debug("Processing tenant contact repair job", "stream", stream, "contact_id", job.ContactID, "organization_id", job.OrganizationID, "message_id", msg.ID)
		return handler.HandleContactRepairJob(ctx, &job)

	default:
		return newPermanentProcessError(fmt.Errorf("unknown job type: %s", jobType))
	}
}

func (c *TenantCampaignConsumer) movePendingToDeadLetter(ctx context.Context, stream, messageID, reason string, attempts int64) error {
	messages, err := c.client.XRangeN(ctx, stream, messageID, messageID, 1).Result()
	if err != nil {
		return fmt.Errorf("load pending message for DLQ: %w", err)
	}
	if len(messages) == 0 {
		if ackErr := c.client.XAck(ctx, stream, c.consumerGroup, messageID).Err(); ackErr != nil {
			return fmt.Errorf("ack missing pending message %s: %w", messageID, ackErr)
		}
		return nil
	}

	return c.moveToDeadLetter(ctx, stream, messages[0], reason, attempts)
}

func (c *TenantCampaignConsumer) moveToDeadLetter(ctx context.Context, stream string, msg redis.XMessage, reason string, attempts int64) error {
	values := map[string]interface{}{
		"original_stream": stream,
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

	dlqStream := campaignDeadLetterStreamFromSource(stream)
	if _, err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: values,
	}).Result(); err != nil {
		return fmt.Errorf("write dead-letter message: %w", err)
	}

	if err := c.client.XAck(ctx, stream, c.consumerGroup, msg.ID).Err(); err != nil {
		return fmt.Errorf("ack dead-lettered message: %w", err)
	}
	if err := c.client.XDel(ctx, stream, msg.ID).Err(); err != nil {
		c.log.Warn("Failed to delete dead-lettered tenant message from source stream", "stream", stream, "group", c.consumerGroup, "message_id", msg.ID, "error", err)
	}

	c.log.Warn("Tenant campaign message moved to dead-letter stream", "stream", stream, "dlq_stream", dlqStream, "message_id", msg.ID, "reason", reason, "attempts", attempts)
	return nil
}

// Close closes the consumer connection.
func (c *TenantCampaignConsumer) Close() error {
	return nil
}

func campaignDeadLetterStreamFromSource(stream string) string {
	if stream == StreamName {
		return DeadLetterStreamName
	}
	return stream + ":dlq"
}

func streamStringValue(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, strings.TrimSpace(v) != ""
	case []byte:
		text := string(v)
		return text, strings.TrimSpace(text) != ""
	default:
		return "", false
	}
}
