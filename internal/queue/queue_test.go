package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoRedis returns a Redis client or skips the test if Redis is unavailable.
func skipIfNoRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := testutil.SetupTestRedis(t)
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	return client
}

// cleanCampaignStream deletes the org-scoped campaign stream used by tests.
func cleanCampaignStream(t *testing.T, client *redis.Client, orgID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	streamName := queue.CampaignStreamName(orgID)
	groupName := queue.CampaignConsumerGroup(orgID)
	deadLetterStream := queue.CampaignDeadLetterStreamName(orgID)
	client.Del(ctx, streamName)
	client.Del(ctx, deadLetterStream)
	t.Cleanup(func() {
		client.Del(ctx, streamName)
		client.Del(ctx, deadLetterStream)
		client.XGroupDestroy(ctx, streamName, groupName)
	})
}

// cleanInboundMediaStream deletes inbound-media streams/groups used by tests.
func cleanInboundMediaStream(t *testing.T, client *redis.Client) {
	t.Helper()
	ctx := context.Background()
	client.Del(ctx, queue.InboundMediaStreamName)
	client.Del(ctx, queue.InboundMediaDeadLetterStreamName)
	t.Cleanup(func() {
		client.Del(ctx, queue.InboundMediaStreamName)
		client.Del(ctx, queue.InboundMediaDeadLetterStreamName)
		client.XGroupDestroy(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup)
	})
}

// makeRecipientJob creates a RecipientJob with random IDs for testing.
func makeRecipientJob() *queue.RecipientJob {
	return &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: uuid.New(),
		PhoneNumber:    "1234567890",
		RecipientName:  "Test User",
		TemplateParams: models.JSONB{"1": "Hello", "2": "World"},
	}
}

func makeRecipientJobForOrg(orgID uuid.UUID) *queue.RecipientJob {
	job := makeRecipientJob()
	job.OrganizationID = orgID
	return job
}

// makeInboundMediaJob creates an InboundMediaJob with random IDs for testing.
func makeInboundMediaJob() *queue.InboundMediaJob {
	return &queue.InboundMediaJob{
		MessageID:          uuid.New(),
		OrganizationID:     uuid.New(),
		InstanceID:         uuid.New(),
		WhatsAppMessageID:  "wamid.inbound.media.test",
		MessageType:        models.MessageTypeDocument,
		MediaKind:          "document",
		MimeType:           "application/pdf",
		FallbackFilename:   "test.pdf",
		MediaPayloadBase64: "dGVzdA==",
		LastError:          "hash of media ciphertext doesn't match",
	}
}

func makeContactRepairJob(orgID uuid.UUID) *queue.ContactRepairJob {
	return &queue.ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgID,
		ConversationID: "201234567890@s.whatsapp.net",
	}
}

// mockHandler implements queue.JobHandler for testing.
type mockHandler struct {
	mu               sync.Mutex
	jobs             []*queue.RecipientJob
	inboundMediaJobs []*queue.InboundMediaJob
	contactJobs      []*queue.ContactRepairJob
	filterJobs       []*queue.WhatsAppFilterJob
	groupJoinJobs    []*queue.GroupJoinJob
	err              error // if set, handler returns this error
}

func (h *mockHandler) HandleRecipientJob(_ context.Context, job *queue.RecipientJob) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.jobs = append(h.jobs, job)
	return h.err
}

func (h *mockHandler) HandleInboundMediaJob(_ context.Context, job *queue.InboundMediaJob) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.inboundMediaJobs = append(h.inboundMediaJobs, job)
	return h.err
}

func (h *mockHandler) HandleContactRepairJob(_ context.Context, job *queue.ContactRepairJob) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.contactJobs = append(h.contactJobs, job)
	return h.err
}

func (h *mockHandler) HandleWhatsAppFilterJob(_ context.Context, job *queue.WhatsAppFilterJob) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.filterJobs = append(h.filterJobs, job)
	return h.err
}

func (h *mockHandler) HandleGroupJoinJob(_ context.Context, job *queue.GroupJoinJob) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.groupJoinJobs = append(h.groupJoinJobs, job)
	return h.err
}

func (h *mockHandler) HandleMessageExtractionJob(_ context.Context, job *queue.MessageExtractionJob) error {
	return h.err
}

func (h *mockHandler) HandleGroupExtractionJob(_ context.Context, job *queue.GroupExtractionJob) error {
	return h.err
}

func (h *mockHandler) HandleMemberExtractionJob(_ context.Context, job *queue.MemberExtractionJob) error {
	return h.err
}

func (h *mockHandler) getJobs() []*queue.RecipientJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	dst := make([]*queue.RecipientJob, len(h.jobs))
	copy(dst, h.jobs)
	return dst
}

func (h *mockHandler) getInboundMediaJobs() []*queue.InboundMediaJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	dst := make([]*queue.InboundMediaJob, len(h.inboundMediaJobs))
	copy(dst, h.inboundMediaJobs)
	return dst
}

// --- NewRedisQueue tests ---

func TestNewRedisQueue(t *testing.T) {
	t.Parallel()
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()

	q := queue.NewRedisQueue(client, log)
	require.NotNil(t, q)

	err := q.Close()
	assert.NoError(t, err)
}

// --- EnqueueRecipient tests ---

func TestEnqueueRecipient_Single(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()
	cleanCampaignStream(t, client, job.OrganizationID)

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// Verify the job landed in the stream.
	msgs, err := client.XRange(ctx, queue.CampaignStreamName(job.OrganizationID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	assert.Equal(t, string(queue.JobTypeRecipient), msgs[0].Values["type"])

	var decoded queue.RecipientJob
	err = json.Unmarshal([]byte(msgs[0].Values["payload"].(string)), &decoded)
	require.NoError(t, err)
	assert.Equal(t, job.CampaignID, decoded.CampaignID)
	assert.Equal(t, job.RecipientID, decoded.RecipientID)
	assert.Equal(t, job.PhoneNumber, decoded.PhoneNumber)
}

func TestEnqueueRecipient_SetsEnqueuedAt(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()
	cleanCampaignStream(t, client, job.OrganizationID)
	// Leave EnqueuedAt as zero so the queue sets it.
	assert.True(t, job.EnqueuedAt.IsZero())

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// The job should now have a non-zero EnqueuedAt.
	assert.False(t, job.EnqueuedAt.IsZero())
}

func TestEnqueueRecipient_PreservesExistingEnqueuedAt(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()
	cleanCampaignStream(t, client, job.OrganizationID)
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	job.EnqueuedAt = fixedTime

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// EnqueuedAt should remain unchanged.
	assert.Equal(t, fixedTime, job.EnqueuedAt)

	// Verify in Redis payload as well.
	msgs, err := client.XRange(ctx, queue.CampaignStreamName(job.OrganizationID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var decoded queue.RecipientJob
	err = json.Unmarshal([]byte(msgs[0].Values["payload"].(string)), &decoded)
	require.NoError(t, err)
	assert.True(t, fixedTime.Equal(decoded.EnqueuedAt))
}

// --- EnqueueRecipients (batch) tests ---

func TestEnqueueRecipients_Batch(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	q := queue.NewRedisQueue(client, log)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	jobs := make([]*queue.RecipientJob, 5)
	for i := range jobs {
		jobs[i] = makeRecipientJobForOrg(orgID)
	}

	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	// All 5 jobs should be in the stream.
	msgs, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, msgs, 5)

	// Verify each message has the correct type.
	for _, msg := range msgs {
		assert.Equal(t, string(queue.JobTypeRecipient), msg.Values["type"])
	}
}

func TestEnqueueRecipients_Empty(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	q := queue.NewRedisQueue(client, log)

	// Enqueuing an empty slice should be a no-op.
	err := q.EnqueueRecipients(ctx, []*queue.RecipientJob{})
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestEnqueueRecipients_SetsEnqueuedAt(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	q := queue.NewRedisQueue(client, log)

	jobs := []*queue.RecipientJob{makeRecipientJobForOrg(orgID), makeRecipientJobForOrg(orgID)}
	for _, j := range jobs {
		assert.True(t, j.EnqueuedAt.IsZero())
	}

	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	// All jobs should now have EnqueuedAt set.
	for _, j := range jobs {
		assert.False(t, j.EnqueuedAt.IsZero())
	}
}

func TestEnqueueContactRepair_UsesOrganizationStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	q := queue.NewRedisQueue(client, log)
	job := makeContactRepairJob(orgID)

	err := q.EnqueueContactRepair(ctx, job)
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, string(queue.JobTypeContactRepair), msgs[0].Values["type"])
}

func TestCampaignStreams_IsolatedPerOrganization(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()
	orgA := uuid.New()
	orgB := uuid.New()

	q := queue.NewRedisQueue(client, log)
	require.NoError(t, q.EnqueueRecipient(ctx, makeRecipientJobForOrg(orgA)))
	require.NoError(t, q.EnqueueRecipient(ctx, makeRecipientJobForOrg(orgA)))
	require.NoError(t, q.EnqueueRecipient(ctx, makeRecipientJobForOrg(orgB)))

	depthA, err := client.XLen(ctx, queue.CampaignStreamName(orgA)).Result()
	require.NoError(t, err)
	depthB, err := client.XLen(ctx, queue.CampaignStreamName(orgB)).Result()
	require.NoError(t, err)

	assert.Equal(t, int64(2), depthA)
	assert.Equal(t, int64(1), depthB)
}

func TestEnqueueInboundMedia_Single(t *testing.T) {
	client := skipIfNoRedis(t)
	cleanInboundMediaStream(t, client)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	q := queue.NewRedisQueue(client, log)
	job := makeInboundMediaJob()

	err := q.EnqueueInboundMedia(ctx, job)
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.InboundMediaStreamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	assert.Equal(t, string(queue.JobTypeInboundMedia), msgs[0].Values["type"])

	var decoded queue.InboundMediaJob
	err = json.Unmarshal([]byte(msgs[0].Values["payload"].(string)), &decoded)
	require.NoError(t, err)
	assert.Equal(t, job.MessageID, decoded.MessageID)
	assert.Equal(t, job.OrganizationID, decoded.OrganizationID)
	assert.Equal(t, job.InstanceID, decoded.InstanceID)
	assert.Equal(t, job.MediaKind, decoded.MediaKind)
	assert.Equal(t, job.MediaPayloadBase64, decoded.MediaPayloadBase64)
	assert.False(t, decoded.EnqueuedAt.IsZero())
}

// --- Consumer tests ---

func TestNewRedisConsumer(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	require.NotNil(t, consumer)

	err = consumer.Close()
	assert.NoError(t, err)
}

func TestConsume_ProcessesJob(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 10*time.Second)

	q := queue.NewRedisQueue(client, log)

	// Enqueue a job first.
	job := makeRecipientJob()
	cleanCampaignStream(t, client, job.OrganizationID)
	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// Create consumer.
	consumer, err := queue.NewOrganizationRedisConsumer(client, log, job.OrganizationID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	// Run the consumer in a goroutine with a cancellable context.
	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	// Wait for the handler to receive the job.
	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) >= 1
	}, 8*time.Second, "handler should have received at least 1 job")

	cancel()

	received := handler.getJobs()
	require.Len(t, received, 1)
	assert.Equal(t, job.CampaignID, received[0].CampaignID)
	assert.Equal(t, job.RecipientID, received[0].RecipientID)
	assert.Equal(t, job.PhoneNumber, received[0].PhoneNumber)
	assert.Equal(t, job.RecipientName, received[0].RecipientName)
}

func TestConsume_RemovesProcessedMessageFromStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 10*time.Second)

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()
	cleanCampaignStream(t, client, job.OrganizationID)

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, job.OrganizationID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}
	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) == 1
	}, 8*time.Second, "handler should process the queued message")

	testutil.AssertEventually(t, func() bool {
		depth, depthErr := client.XLen(ctx, queue.CampaignStreamName(job.OrganizationID)).Result()
		return depthErr == nil && depth == 0
	}, 8*time.Second, "processed message should be removed from the org stream")

	pending, err := client.XPending(ctx, queue.CampaignStreamName(job.OrganizationID), queue.CampaignConsumerGroup(job.OrganizationID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

func TestConsume_EmptyQueue(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 10*time.Second)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	// Cancel quickly -- the consumer should handle an empty queue gracefully.
	consumeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = consumer.Consume(consumeCtx, handler)
	// Should return context error (deadline exceeded or cancelled), not a crash.
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// No jobs should have been processed.
	assert.Empty(t, handler.getJobs())
}

func TestConsume_MultipleJobs(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 15*time.Second)

	q := queue.NewRedisQueue(client, log)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	// Enqueue 3 jobs.
	jobs := make([]*queue.RecipientJob, 3)
	for i := range jobs {
		jobs[i] = makeRecipientJobForOrg(orgID)
	}
	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) >= 3
	}, 12*time.Second, "handler should have received all 3 jobs")

	cancel()

	received := handler.getJobs()
	assert.Len(t, received, 3)

	// Verify all campaign IDs were received (order may vary).
	receivedIDs := make(map[uuid.UUID]bool)
	for _, r := range received {
		receivedIDs[r.CampaignID] = true
	}
	for _, j := range jobs {
		assert.True(t, receivedIDs[j.CampaignID], "expected campaign ID %s to be received", j.CampaignID)
	}
}

func TestConsume_PermanentFailureMovesToDLQ(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 15*time.Second)
	orgID := uuid.New()
	cleanCampaignStream(t, client, orgID)

	dlqStream := queue.CampaignDeadLetterStreamName(orgID)
	streamName := queue.CampaignStreamName(orgID)
	groupName := queue.CampaignConsumerGroup(orgID)

	// Push a malformed job type that cannot be processed and should be dead-lettered.
	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"type":    "unknown_job_type",
			"payload": "{}",
		},
	}).Result()
	require.NoError(t, err)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}
	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		msgs, dlqErr := client.XRange(ctx, dlqStream, "-", "+").Result()
		return dlqErr == nil && len(msgs) >= 1
	}, 10*time.Second, "invalid job should be moved to dead-letter stream")

	cancel()

	// Invalid message should not remain pending indefinitely.
	pending, err := client.XPending(ctx, streamName, groupName).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

func TestConsumeInboundMedia_ProcessesJob(t *testing.T) {
	client := skipIfNoRedis(t)
	cleanInboundMediaStream(t, client)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 10*time.Second)

	q := queue.NewRedisQueue(client, log)
	job := makeInboundMediaJob()
	err := q.EnqueueInboundMedia(ctx, job)
	require.NoError(t, err)

	consumer, err := queue.NewRedisInboundMediaConsumer(client, log, 0)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getInboundMediaJobs()) >= 1
	}, 8*time.Second, "handler should have received inbound media job")

	cancel()

	received := handler.getInboundMediaJobs()
	require.Len(t, received, 1)
	assert.Equal(t, job.MessageID, received[0].MessageID)
	assert.Equal(t, job.InstanceID, received[0].InstanceID)
	assert.Equal(t, job.WhatsAppMessageID, received[0].WhatsAppMessageID)
}

func TestConsumeInboundMedia_PermanentFailureMovesToDLQ(t *testing.T) {
	client := skipIfNoRedis(t)
	cleanInboundMediaStream(t, client)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 15*time.Second)

	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: queue.InboundMediaStreamName,
		Values: map[string]interface{}{
			"type":    string(queue.JobTypeInboundMedia),
			"payload": "{not-json",
		},
	}).Result()
	require.NoError(t, err)

	consumer, err := queue.NewRedisInboundMediaConsumer(client, log, 0)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}
	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		msgs, dlqErr := client.XRange(ctx, queue.InboundMediaDeadLetterStreamName, "-", "+").Result()
		return dlqErr == nil && len(msgs) >= 1
	}, 10*time.Second, "invalid inbound media job should be moved to dead-letter stream")

	cancel()

	pending, err := client.XPending(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

// --- Pub/Sub tests ---

func TestPublishCampaignStats(t *testing.T) {
	t.Parallel()
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)

	pub := queue.NewPublisher(client, log)

	update := &queue.CampaignStatsUpdate{
		CampaignID:     uuid.New().String(),
		OrganizationID: uuid.New(),
		Status:         models.CampaignStatusProcessing,
		SentCount:      10,
		DeliveredCount: 8,
		ReadCount:      5,
		FailedCount:    2,
	}

	// Publishing without any subscriber should not error.
	err := pub.PublishCampaignStats(ctx, update)
	assert.NoError(t, err)
}

func TestSubscribeCampaignStats_ReceivesUpdate(t *testing.T) {
	t.Parallel()
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContextWithTimeout(t, 10*time.Second)

	pub := queue.NewPublisher(client, log)
	sub := queue.NewSubscriber(client, log)
	defer func() { _ = sub.Close() }()

	// Use a unique campaign ID to filter out messages from parallel tests
	// sharing the same pub/sub channel.
	targetCampaignID := uuid.New().String()

	var mu sync.Mutex
	var matched *queue.CampaignStatsUpdate

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := sub.SubscribeCampaignStats(subCtx, func(update *queue.CampaignStatsUpdate) {
		mu.Lock()
		defer mu.Unlock()
		if update.CampaignID == targetCampaignID {
			matched = update
		}
	})
	require.NoError(t, err)

	// Give the subscriber a moment to fully establish.
	time.Sleep(100 * time.Millisecond)

	update := &queue.CampaignStatsUpdate{
		CampaignID:     targetCampaignID,
		OrganizationID: uuid.New(),
		Status:         models.CampaignStatusCompleted,
		SentCount:      50,
		DeliveredCount: 45,
		ReadCount:      30,
		FailedCount:    5,
	}

	err = pub.PublishCampaignStats(ctx, update)
	require.NoError(t, err)

	testutil.AssertEventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return matched != nil
	}, 5*time.Second, "subscriber should have received the stats update")

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, matched)
	assert.Equal(t, update.CampaignID, matched.CampaignID)
	assert.Equal(t, update.OrganizationID, matched.OrganizationID)
	assert.Equal(t, update.Status, matched.Status)
	assert.Equal(t, update.SentCount, matched.SentCount)
	assert.Equal(t, update.DeliveredCount, matched.DeliveredCount)
	assert.Equal(t, update.ReadCount, matched.ReadCount)
	assert.Equal(t, update.FailedCount, matched.FailedCount)
}

func TestSubscriber_Close(t *testing.T) {
	t.Parallel()
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()

	sub := queue.NewSubscriber(client, log)

	// Close before subscribing should not error.
	err := sub.Close()
	assert.NoError(t, err)
}

// --- Error handling tests ---

func TestEnqueueRecipient_InvalidRedis(t *testing.T) {
	t.Parallel()
	log := testutil.NopLogger()

	// Create a client pointing to an invalid address.
	badClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:1", // Invalid port
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = badClient.Close() }()

	q := queue.NewRedisQueue(badClient, log)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	job := makeRecipientJob()
	err := q.EnqueueRecipient(ctx, job)
	assert.Error(t, err)
}

func TestEnqueueRecipients_InvalidRedis(t *testing.T) {
	t.Parallel()
	log := testutil.NopLogger()

	badClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = badClient.Close() }()

	q := queue.NewRedisQueue(badClient, log)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	jobs := []*queue.RecipientJob{makeRecipientJob()}
	err := q.EnqueueRecipients(ctx, jobs)
	assert.Error(t, err)
}

func TestNewRedisConsumer_InvalidRedis(t *testing.T) {
	t.Parallel()
	log := testutil.NopLogger()

	badClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = badClient.Close() }()

	_, err := queue.NewRedisConsumer(badClient, log)
	assert.Error(t, err)
}

func TestPublishCampaignStats_InvalidRedis(t *testing.T) {
	t.Parallel()
	log := testutil.NopLogger()

	badClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = badClient.Close() }()

	pub := queue.NewPublisher(badClient, log)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	update := &queue.CampaignStatsUpdate{
		CampaignID: uuid.New().String(),
		Status:     models.CampaignStatusProcessing,
	}
	err := pub.PublishCampaignStats(ctx, update)
	assert.Error(t, err)
}

// --- Permanent error tests (unit tests, no Redis required) ---

func TestNewPermanentError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	perr := queue.NewPermanentError(baseErr)

	require.NotNil(t, perr)
	assert.Error(t, perr)
	assert.Equal(t, "base error", perr.Error())
}

func TestNewPermanentError_Nil(t *testing.T) {
	t.Parallel()

	perr := queue.NewPermanentError(nil)
	assert.Nil(t, perr)
}

func TestIsPermanentError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("permanent failure")
	perr := queue.NewPermanentError(baseErr)

	assert.True(t, queue.IsPermanentError(perr))
}

func TestIsPermanentError_NonPermanent(t *testing.T) {
	t.Parallel()

	err := errors.New("regular error")
	assert.False(t, queue.IsPermanentError(err))
}

func TestIsPermanentError_Nil(t *testing.T) {
	t.Parallel()

	assert.False(t, queue.IsPermanentError(nil))
}

func TestIsPermanentError_Wrapped(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("wrapped permanent error")
	perr := queue.NewPermanentError(baseErr)
	wrapped := fmt.Errorf("wrapped: %w", perr)

	assert.True(t, queue.IsPermanentError(wrapped))
	assert.True(t, queue.IsPermanentError(perr))
}

// --- RecipientJob validation tests ---

func TestRecipientJob_Validation(t *testing.T) {
	t.Parallel()

	job := makeRecipientJob()

	// Valid job should have required fields
	assert.NotEqual(t, uuid.UUID{}, job.CampaignID)
	assert.NotEqual(t, uuid.UUID{}, job.RecipientID)
	assert.NotEqual(t, uuid.UUID{}, job.OrganizationID)
	assert.NotEmpty(t, job.PhoneNumber)
	assert.NotEmpty(t, job.RecipientName)
}

func TestInboundMediaJob_Validation(t *testing.T) {
	t.Parallel()

	job := makeInboundMediaJob()

	// Valid job should have required fields
	assert.NotEqual(t, uuid.UUID{}, job.MessageID)
	assert.NotEqual(t, uuid.UUID{}, job.OrganizationID)
	assert.NotEqual(t, uuid.UUID{}, job.InstanceID)
	assert.NotEmpty(t, job.WhatsAppMessageID)
	assert.NotEmpty(t, job.MediaKind)
	assert.NotEmpty(t, job.MimeType)
	assert.NotEmpty(t, job.MediaPayloadBase64)
}

// --- JobType tests ---

func TestJobType_Values(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jobType  queue.JobType
		expected string
	}{
		{"Recipient", queue.JobTypeRecipient, "recipient"},
		{"InboundMedia", queue.JobTypeInboundMedia, "inbound_media"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.jobType))
		})
	}
}

// --- Constants tests ---

func TestStreamNames(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	tests := []struct {
		name  string
		value string
	}{
		{"StreamName", queue.StreamName},
		{"ConsumerGroup", queue.ConsumerGroup},
		{"DeadLetterStreamName", queue.DeadLetterStreamName},
		{"CampaignStreamName", queue.CampaignStreamName(orgID)},
		{"CampaignConsumerGroup", queue.CampaignConsumerGroup(orgID)},
		{"CampaignDeadLetterStreamName", queue.CampaignDeadLetterStreamName(orgID)},
		{"InboundMediaStreamName", queue.InboundMediaStreamName},
		{"InboundMediaConsumerGroup", queue.InboundMediaConsumerGroup},
		{"InboundMediaDeadLetterStreamName", queue.InboundMediaDeadLetterStreamName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestDeadLetterStreamName_Derived(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	// DeadLetterStreamName should be StreamName + ":dlq"
	assert.Equal(t, queue.StreamName+":dlq", queue.DeadLetterStreamName)
	assert.Equal(t, queue.CampaignStreamName(orgID)+":dlq", queue.CampaignDeadLetterStreamName(orgID))
	assert.Equal(t, queue.InboundMediaStreamName+":dlq", queue.InboundMediaDeadLetterStreamName)
}

func TestMaxDeliveryAttempts(t *testing.T) {
	t.Parallel()

	// MaxDeliveryAttempts should be positive
	assert.Greater(t, queue.MaxDeliveryAttempts, int64(0))
}

func TestBlockTimeout_Positive(t *testing.T) {
	t.Parallel()

	// BlockTimeout should be positive
	assert.Greater(t, queue.BlockTimeout, time.Duration(0))
}

func TestClaimMinIdleTime_Positive(t *testing.T) {
	t.Parallel()

	// ClaimMinIdleTime should be positive
	assert.Greater(t, queue.ClaimMinIdleTime, time.Duration(0))
}

func TestPendingClaimInterval_Positive(t *testing.T) {
	t.Parallel()

	// PendingClaimInterval should be positive
	assert.Greater(t, queue.PendingClaimInterval, time.Duration(0))
}

// --- Miniredis-based tests (no Redis server required) ---

// setupMiniRedis creates a miniredis instance and returns a Redis client connected to it.
func setupMiniRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestRedisQueue_WithMiniRedis_EnqueueRecipient(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// Verify the job landed in the stream.
	msgs, err := client.XRange(ctx, queue.CampaignStreamName(job.OrganizationID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	assert.Equal(t, string(queue.JobTypeRecipient), msgs[0].Values["type"])

	var decoded queue.RecipientJob
	err = json.Unmarshal([]byte(msgs[0].Values["payload"].(string)), &decoded)
	require.NoError(t, err)
	assert.Equal(t, job.CampaignID, decoded.CampaignID)
	assert.Equal(t, job.RecipientID, decoded.RecipientID)
	assert.Equal(t, job.PhoneNumber, decoded.PhoneNumber)
}

func TestRedisQueue_WithMiniRedis_EnqueueRecipients_Batch(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	q := queue.NewRedisQueue(client, log)
	orgID := uuid.New()

	jobs := make([]*queue.RecipientJob, 5)
	for i := range jobs {
		jobs[i] = makeRecipientJobForOrg(orgID)
	}

	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	// All 5 jobs should be in the stream.
	msgs, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, msgs, 5)

	// Verify each message has the correct type.
	for _, msg := range msgs {
		assert.Equal(t, string(queue.JobTypeRecipient), msg.Values["type"])
	}
}

func TestRedisQueue_WithMiniRedis_EnqueueRecipients_Empty(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	q := queue.NewRedisQueue(client, log)

	err := q.EnqueueRecipients(ctx, []*queue.RecipientJob{})
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.CampaignStreamName(uuid.New()), "-", "+").Result()
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestRedisQueue_WithMiniRedis_EnqueueInboundMedia(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	q := queue.NewRedisQueue(client, log)
	job := makeInboundMediaJob()

	err := q.EnqueueInboundMedia(ctx, job)
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.InboundMediaStreamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	assert.Equal(t, string(queue.JobTypeInboundMedia), msgs[0].Values["type"])

	var decoded queue.InboundMediaJob
	err = json.Unmarshal([]byte(msgs[0].Values["payload"].(string)), &decoded)
	require.NoError(t, err)
	assert.Equal(t, job.MessageID, decoded.MessageID)
	assert.Equal(t, job.OrganizationID, decoded.OrganizationID)
	assert.Equal(t, job.InstanceID, decoded.InstanceID)
	assert.Equal(t, job.MediaKind, decoded.MediaKind)
}

func TestRedisConsumer_WithMiniRedis_Consume_ProcessesJob(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := queue.NewRedisQueue(client, log)

	// Enqueue a job first.
	job := makeRecipientJob()
	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	// Create consumer.
	consumer, err := queue.NewOrganizationRedisConsumer(client, log, job.OrganizationID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	// Run the consumer in a goroutine with a cancellable context.
	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	// Wait for the handler to receive the job.
	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) >= 1
	}, 5*time.Second, "handler should have received at least 1 job")

	consumeCancel()

	received := handler.getJobs()
	require.Len(t, received, 1)
	assert.Equal(t, job.CampaignID, received[0].CampaignID)
	assert.Equal(t, job.RecipientID, received[0].RecipientID)
	assert.Equal(t, job.PhoneNumber, received[0].PhoneNumber)

	testutil.AssertEventually(t, func() bool {
		pending, pendingErr := client.XPending(ctx, queue.CampaignStreamName(job.OrganizationID), queue.CampaignConsumerGroup(job.OrganizationID)).Result()
		length, lenErr := client.XLen(ctx, queue.CampaignStreamName(job.OrganizationID)).Result()
		return pendingErr == nil && lenErr == nil && pending.Count == 0 && length == 0
	}, 5*time.Second, "successfully processed job should be ACKed and deleted")
}

func TestRedisConsumer_WithMiniRedis_Consume_MultipleJobs(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	q := queue.NewRedisQueue(client, log)
	orgID := uuid.New()

	// Enqueue 3 jobs.
	jobs := make([]*queue.RecipientJob, 3)
	for i := range jobs {
		jobs[i] = makeRecipientJobForOrg(orgID)
	}
	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) >= 3
	}, 10*time.Second, "handler should have received all 3 jobs")

	consumeCancel()

	received := handler.getJobs()
	assert.Len(t, received, 3)

	// Verify all campaign IDs were received (order may vary).
	receivedIDs := make(map[uuid.UUID]bool)
	for _, r := range received {
		receivedIDs[r.CampaignID] = true
	}
	for _, j := range jobs {
		assert.True(t, receivedIDs[j.CampaignID], "expected campaign ID %s to be received", j.CampaignID)
	}
}

func TestRedisConsumer_WithMiniRedis_PermanentFailureMovesToDLQ(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	orgID := uuid.New()
	streamName := queue.CampaignStreamName(orgID)
	groupName := queue.CampaignConsumerGroup(orgID)
	deadLetterStream := queue.CampaignDeadLetterStreamName(orgID)

	// Push a malformed job type that cannot be processed and should be dead-lettered.
	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"type":    "unknown_job_type",
			"payload": "{}",
		},
	}).Result()
	require.NoError(t, err)

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}
	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		msgs, dlqErr := client.XRange(ctx, deadLetterStream, "-", "+").Result()
		return dlqErr == nil && len(msgs) >= 1
	}, 8*time.Second, "invalid job should be moved to dead-letter stream")

	consumeCancel()

	// Invalid message should not remain pending indefinitely.
	pending, err := client.XPending(ctx, streamName, groupName).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

func TestRedisConsumer_WithMiniRedis_HandlerErrorDoesNotAck(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := queue.NewRedisQueue(client, log)
	job := makeRecipientJob()
	require.NoError(t, q.EnqueueRecipient(ctx, job))

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, job.OrganizationID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{err: errors.New("temporary handler failure")}
	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getJobs()) >= 1
	}, 5*time.Second, "handler should receive the job before returning an error")

	consumeCancel()

	pending, err := client.XPending(ctx, queue.CampaignStreamName(job.OrganizationID), queue.CampaignConsumerGroup(job.OrganizationID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), pending.Count)

	deadLetters, err := client.XRange(ctx, queue.CampaignDeadLetterStreamName(job.OrganizationID), "-", "+").Result()
	require.NoError(t, err)
	assert.Empty(t, deadLetters)
}

func TestRedisConsumer_Consume_CanceledContextExits(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	orgID := uuid.New()

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = consumer.Consume(ctx, &mockHandler{})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRedisInboundMediaConsumer_WithMiniRedis_Consume_ProcessesJob(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := queue.NewRedisQueue(client, log)
	job := makeInboundMediaJob()
	err := q.EnqueueInboundMedia(ctx, job)
	require.NoError(t, err)

	consumer, err := queue.NewRedisInboundMediaConsumer(client, log, 0)
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	handler := &mockHandler{}

	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		_ = consumer.Consume(consumeCtx, handler)
	}()

	testutil.AssertEventually(t, func() bool {
		return len(handler.getInboundMediaJobs()) >= 1
	}, 5*time.Second, "handler should have received inbound media job")

	consumeCancel()

	received := handler.getInboundMediaJobs()
	require.Len(t, received, 1)
	assert.Equal(t, job.MessageID, received[0].MessageID)
	assert.Equal(t, job.InstanceID, received[0].InstanceID)
	assert.Equal(t, job.WhatsAppMessageID, received[0].WhatsAppMessageID)
}

func TestRedisQueue_WithMiniRedis_Close(t *testing.T) {
	t.Parallel()

	client := setupMiniRedis(t)
	log := testutil.NopLogger()

	q := queue.NewRedisQueue(client, log)
	err := q.Close()
	assert.NoError(t, err)
}

func TestRedisConsumer_WithMiniRedis_Close(t *testing.T) {
	client := setupMiniRedis(t)
	log := testutil.NopLogger()
	orgID := uuid.New()

	consumer, err := queue.NewOrganizationRedisConsumer(client, log, orgID)
	require.NoError(t, err)

	err = consumer.Close()
	assert.NoError(t, err)
}
