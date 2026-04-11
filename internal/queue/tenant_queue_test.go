package queue_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanTenantCampaignStreams(t *testing.T, client *redis.Client, orgIDs ...uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	for _, orgID := range orgIDs {
		if orgID == uuid.Nil {
			continue
		}
		stream := queue.CampaignStreamName(orgID)
		dlq := queue.CampaignDeadLetterStreamName(orgID)
		client.Del(ctx, stream, dlq)
		client.XGroupDestroy(ctx, stream, queue.ConsumerGroup)

		t.Cleanup(func() {
			client.Del(ctx, stream, dlq)
			client.XGroupDestroy(ctx, stream, queue.ConsumerGroup)
		})
	}
}

func enqueueLegacyMessage(t *testing.T, client *redis.Client, jobType queue.JobType, payload interface{}) string {
	t.Helper()
	ctx := context.Background()

	payloadText, err := json.Marshal(payload)
	require.NoError(t, err)

	id, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: queue.StreamName,
		Values: map[string]interface{}{
			"type":    string(jobType),
			"payload": string(payloadText),
		},
	}).Result()
	require.NoError(t, err)
	return id
}

func TestEnqueueRecipient_TenantQueueUsesOrganizationStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgID)

	q := queue.NewTenantQueueManager(client, log)
	job := makeRecipientJob()
	job.OrganizationID = orgID

	err := q.EnqueueRecipient(ctx, job)
	require.NoError(t, err)

	tenantMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, tenantMessages, 1)

	globalMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, globalMessages, 0)
}

func TestEnqueueRecipient_TenantQueueRejectsMissingOrganizationID(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()

	q := queue.NewTenantQueueManager(client, log)
	job := makeRecipientJob()
	job.OrganizationID = uuid.Nil

	err := q.EnqueueRecipient(testutil.TestContext(t), job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization id is required")
}

func TestEnqueueInboundMedia_TenantQueueKeepsGlobalStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	cleanInboundMediaStream(t, client)

	q := queue.NewTenantQueueManager(client, log)
	job := makeInboundMediaJob()

	err := q.EnqueueInboundMedia(ctx, job)
	require.NoError(t, err)

	msgs, err := client.XRange(ctx, queue.InboundMediaStreamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}

func TestEnqueueRecipients_TenantQueueFansOutMixedOrganizations(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgA := uuid.New()
	orgB := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgA, orgB)

	jobs := []*queue.RecipientJob{
		makeRecipientJob(),
		makeRecipientJob(),
		makeRecipientJob(),
	}
	jobs[0].OrganizationID = orgA
	jobs[1].OrganizationID = orgB
	jobs[2].OrganizationID = orgA

	q := queue.NewTenantQueueManager(client, log)
	err := q.EnqueueRecipients(ctx, jobs)
	require.NoError(t, err)

	orgAMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgA), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, orgAMessages, 2)

	orgBMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgB), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, orgBMessages, 1)

	globalMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, globalMessages, 0)
}

func TestEnqueueContactRepair_TenantQueueUsesOrganizationStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgID)

	q := queue.NewTenantQueueManager(client, log)
	job := &queue.ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgID,
		ConversationID: "12345@s.whatsapp.net",
	}

	err := q.EnqueueContactRepair(ctx, job)
	require.NoError(t, err)

	tenantMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, tenantMessages, 1)
	assert.Equal(t, string(queue.JobTypeContactRepair), tenantMessages[0].Values["type"])

	globalMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, globalMessages, 0)
}

func TestConsume_TenantCampaignConsumerConsumesMultipleStreams(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgA := uuid.New()
	orgB := uuid.New()
	cleanTenantCampaignStreams(t, client, orgA, orgB)

	queueManager := queue.NewTenantQueueManager(client, log)
	consumer := queue.NewTenantCampaignConsumer(client, log)
	handler := &mockHandler{}

	consumeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = consumer.Consume(consumeCtx, handler)
	}()

	err := queueManager.EnqueueRecipient(ctx, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgA,
		PhoneNumber:    "201001234567",
		RecipientName:  "Org A",
		TemplateParams: models.JSONB{"name": "Org A"},
	})
	require.NoError(t, err)

	err = queueManager.EnqueueContactRepair(ctx, &queue.ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgB,
		ConversationID: "20105551234@s.whatsapp.net",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(handler.getJobs()) == 1 && len(handler.getContactJobs()) == 1
	}, 8*time.Second, 100*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tenant consumer did not stop after cancellation")
	}
}

func TestConsume_TenantCampaignConsumerDeadLettersInvalidTenantMessage(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgA := uuid.New()
	orgB := uuid.New()
	cleanTenantCampaignStreams(t, client, orgA, orgB)

	queueManager := queue.NewTenantQueueManager(client, log)
	consumer := queue.NewTenantCampaignConsumer(client, log)
	handler := &mockHandler{}

	consumeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = consumer.Consume(consumeCtx, handler)
	}()

	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: queue.CampaignStreamName(orgA),
		Values: map[string]interface{}{
			"type":    "unknown",
			"payload": "{}",
		},
	}).Result()
	require.NoError(t, err)

	err = queueManager.EnqueueRecipient(ctx, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgB,
		PhoneNumber:    "201044444444",
		RecipientName:  "Valid",
		TemplateParams: models.JSONB{"name": "Valid"},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		validJobs := len(handler.getJobs()) == 1
		dlqMessages, dlqErr := client.XRange(ctx, queue.CampaignDeadLetterStreamName(orgA), "-", "+").Result()
		if dlqErr != nil {
			return false
		}
		return validJobs && len(dlqMessages) == 1
	}, 8*time.Second, 100*time.Millisecond)

	sourceMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgA), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, sourceMessages, 0)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tenant consumer did not stop after cancellation")
	}
}

func TestMigrateLegacyCampaignStream_NoLegacyStream(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	cleanStream(t, client)

	summary, err := queue.MigrateLegacyCampaignStream(testutil.TestContext(t), client, log, queue.CampaignMigrationOptions{})
	require.NoError(t, err)
	assert.False(t, summary.LegacyStreamExists)
	assert.Zero(t, summary.Migrated)
}

func TestMigrateLegacyCampaignStream_NoConsumerGroupMovesAllValidJobs(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgA := uuid.New()
	orgB := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgA, orgB)

	enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgA,
		PhoneNumber:    "201001234567",
		RecipientName:  "Org A",
		TemplateParams: models.JSONB{"name": "Org A"},
	})
	enqueueLegacyMessage(t, client, queue.JobTypeContactRepair, &queue.ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgB,
		ConversationID: "20105551234@s.whatsapp.net",
	})

	summary, err := queue.MigrateLegacyCampaignStream(ctx, client, log, queue.CampaignMigrationOptions{
		Apply: true,
	})
	require.NoError(t, err)
	assert.True(t, summary.TemporaryGroupUsed)
	assert.EqualValues(t, 2, summary.Migrated)

	orgAMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgA), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, orgAMessages, 1)

	orgBMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgB), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, orgBMessages, 1)

	legacyMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, legacyMessages, 0)
}

func TestMigrateLegacyCampaignStream_ExistingConsumerGroupMovesUnreadAndPendingOnly(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgID)

	ackedID := enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgID,
		PhoneNumber:    "201000000001",
		RecipientName:  "Acked",
	})
	pendingID := enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgID,
		PhoneNumber:    "201000000002",
		RecipientName:  "Pending",
	})
	unreadID := enqueueLegacyMessage(t, client, queue.JobTypeContactRepair, &queue.ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgID,
		ConversationID: "201000000003@s.whatsapp.net",
	})

	err := client.XGroupCreateMkStream(ctx, queue.StreamName, queue.ConsumerGroup, "0").Err()
	require.NoError(t, err)

	readResults, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    queue.ConsumerGroup,
		Consumer: "legacy-worker",
		Streams:  []string{queue.StreamName, ">"},
		Count:    2,
	}).Result()
	require.NoError(t, err)
	require.Len(t, readResults, 1)
	require.Len(t, readResults[0].Messages, 2)

	require.NoError(t, client.XAck(ctx, queue.StreamName, queue.ConsumerGroup, ackedID).Err())

	summary, err := queue.MigrateLegacyCampaignStream(ctx, client, log, queue.CampaignMigrationOptions{
		Apply: true,
	})
	require.NoError(t, err)
	assert.True(t, summary.ConsumerGroupFound)
	assert.EqualValues(t, 1, summary.Pending)
	assert.EqualValues(t, 1, summary.Unread)
	assert.EqualValues(t, 2, summary.Migrated)

	tenantMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, tenantMessages, 2)

	legacyMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, legacyMessages, 1)
	assert.Equal(t, ackedID, legacyMessages[0].ID)
	assert.NotEqual(t, pendingID, legacyMessages[0].ID)
	assert.NotEqual(t, unreadID, legacyMessages[0].ID)
}

func TestMigrateLegacyCampaignStream_InvalidMessagePreserved(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgID)

	validID := enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgID,
		PhoneNumber:    "201011111111",
		RecipientName:  "Valid",
		TemplateParams: models.JSONB{"name": "Valid"},
	})
	invalidID := enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: uuid.Nil,
		PhoneNumber:    "201022222222",
		RecipientName:  "Invalid",
	})

	summary, err := queue.MigrateLegacyCampaignStream(ctx, client, log, queue.CampaignMigrationOptions{
		Apply: true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.Migrated)
	assert.EqualValues(t, 1, summary.Invalid)
	assert.Contains(t, summary.InvalidMessageIDs, invalidID)
	assert.Contains(t, summary.MigratedMessageIDs, validID)

	tenantMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, tenantMessages, 1)

	legacyMessages, err := client.XRange(ctx, queue.StreamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, legacyMessages, 1)
	assert.Equal(t, invalidID, legacyMessages[0].ID)
}

func TestMigrateLegacyCampaignStream_RerunIsIdempotentForMovedJobs(t *testing.T) {
	client := skipIfNoRedis(t)
	log := testutil.NopLogger()
	ctx := testutil.TestContext(t)
	orgID := uuid.New()
	cleanStream(t, client)
	cleanTenantCampaignStreams(t, client, orgID)

	enqueueLegacyMessage(t, client, queue.JobTypeRecipient, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgID,
		PhoneNumber:    "201033333333",
		RecipientName:  "First pass",
	})

	firstSummary, err := queue.MigrateLegacyCampaignStream(ctx, client, log, queue.CampaignMigrationOptions{
		Apply: true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, firstSummary.Migrated)

	secondSummary, err := queue.MigrateLegacyCampaignStream(ctx, client, log, queue.CampaignMigrationOptions{
		Apply: true,
	})
	require.NoError(t, err)
	assert.Zero(t, secondSummary.Migrated)

	tenantMessages, err := client.XRange(ctx, queue.CampaignStreamName(orgID), "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, tenantMessages, 1)
}
