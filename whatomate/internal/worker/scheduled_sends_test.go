package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPollerTest(t *testing.T) (*Worker, *miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	w := &Worker{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{
				Provider: "meta",
			},
		},
		Redis:     client,
		Log:       testutil.NopLogger(),
		Publisher: queue.NewPublisher(client, testutil.NopLogger()),
	}

	return w, mr, client
}

func readyScheduledSend(t *testing.T) *scheduledSend {
	t.Helper()
	return &scheduledSend{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: uuid.New(),
		PhoneNumber:    "+1234567890",
		RecipientName:  "Test User",
		RecipientType:  models.RecipientTypeIndividual,
		EnqueuedAt:     time.Now().Add(-1 * time.Minute),
		SendAt:         time.Now().Add(-30 * time.Second),
	}
}

func insertReadyZSETEntry(t *testing.T, client *redis.Client, ss *scheduledSend) {
	t.Helper()
	ctx := context.Background()
	payload, err := json.Marshal(ss)
	require.NoError(t, err)
	score := float64(time.Now().Add(-1 * time.Second).UnixMilli())
	require.NoError(t, client.ZAdd(ctx, scheduledSendsKey, redis.Z{Score: score, Member: string(payload)}).Err())
}

func TestClaimScheduledSend_AcquiresLock(t *testing.T) {
	_, _, client := setupPollerTest(t)
	w := &Worker{Redis: client, Log: testutil.NopLogger()}
	ctx := context.Background()

	recipientID := uuid.New()

	token, acquired := w.claimScheduledSend(ctx, recipientID)
	assert.True(t, acquired, "first claim should succeed")
	assert.NotEmpty(t, token, "token should be non-empty on successful claim")

	assert.NoError(t, client.Get(ctx, fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)).Err(),
		"lock key should exist in Redis")

	token2, acquired2 := w.claimScheduledSend(ctx, recipientID)
	assert.False(t, acquired2, "second claim for same recipient should fail")
	assert.Empty(t, token2, "token should be empty on failed claim")
}

func TestClaimScheduledSend_NilRedis(t *testing.T) {
	w := &Worker{Redis: nil, Log: testutil.NopLogger()}
	ctx := context.Background()

	token, acquired := w.claimScheduledSend(ctx, uuid.New())
	assert.True(t, acquired, "nil Redis should return true (fallback)")
	assert.Equal(t, "local", token, "nil Redis should return 'local' token")
}

func TestReleaseScheduledSendLock_Ownership(t *testing.T) {
	_, _, client := setupPollerTest(t)
	w := &Worker{Redis: client, Log: testutil.NopLogger()}
	ctx := context.Background()

	recipientID := uuid.New()
	token, acquired := w.claimScheduledSend(ctx, recipientID)
	require.True(t, acquired)

	w.releaseScheduledSendLock(ctx, recipientID, "wrong-token")
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	_, err := client.Get(ctx, key).Result()
	assert.NoError(t, err, "lock should not be released with wrong token")

	w.releaseScheduledSendLock(ctx, recipientID, token)
	_, err = client.Get(ctx, key).Result()
	assert.Equal(t, redis.Nil, err, "lock should be released with correct token")
}

func TestVerifyScheduledSendLock(t *testing.T) {
	_, _, client := setupPollerTest(t)
	w := &Worker{Redis: client, Log: testutil.NopLogger()}
	ctx := context.Background()

	recipientID := uuid.New()
	token, acquired := w.claimScheduledSend(ctx, recipientID)
	require.True(t, acquired)

	assert.True(t, w.verifyScheduledSendLock(ctx, recipientID, token), "should verify own token")
	assert.False(t, w.verifyScheduledSendLock(ctx, recipientID, "wrong"), "should reject wrong token")
	assert.False(t, w.verifyScheduledSendLock(ctx, uuid.New(), token), "should reject missing lock")
}

func TestPollScheduledSends_DoesNotRemoveIfLockHeldByAnother(t *testing.T) {
	w, _, client := setupPollerTest(t)
	ctx := context.Background()

	ss := readyScheduledSend(t)
	insertReadyZSETEntry(t, client, ss)

	lockKey := fmt.Sprintf("whatomate:scheduled_send_lock:%s", ss.RecipientID)
	ok, err := client.SetArgs(ctx, lockKey, "other-worker", redis.SetArgs{Mode: "NX", TTL: 5 * time.Minute}).Result()
	require.NoError(t, err)
	require.Equal(t, "OK", ok)

	err = w.pollScheduledSends(ctx)
	assert.NoError(t, err)

	results, err := client.ZRange(ctx, scheduledSendsKey, 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, results, 1, "ZSET entry should remain when lock is held by another worker")
}

func TestPollScheduledSends_RetriesOnDBError(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL required for DB-error retry test")
	}

	db := testutil.SetupTestDB(t)
	w, _, client := setupPollerTest(t)
	ctx := context.Background()

	ss := readyScheduledSend(t)
	insertReadyZSETEntry(t, client, ss)

	w.DB = db

	err := w.pollScheduledSends(ctx)
	assert.NoError(t, err)

	results, err := client.ZRange(ctx, scheduledSendsKey, 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, results, 1, "ZSET entry should remain when recipient not found in DB (will retry)")
}

func TestPollScheduledSends_RemovesCorruptEntry(t *testing.T) {
	w, _, client := setupPollerTest(t)
	ctx := context.Background()

	score := float64(time.Now().Add(-1 * time.Second).UnixMilli())
	require.NoError(t, client.ZAdd(ctx, scheduledSendsKey, redis.Z{Score: score, Member: "not-valid-json{{{"}).Err())

	err := w.pollScheduledSends(ctx)
	assert.NoError(t, err)

	results, err := client.ZRange(ctx, scheduledSendsKey, 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, results, 0, "corrupt ZSET entry should be removed")
}

func TestIsScheduledSendLockHeld(t *testing.T) {
	w, _, client := setupPollerTest(t)
	ctx := context.Background()

	lockedID := uuid.New()
	lockKey := fmt.Sprintf("whatomate:scheduled_send_lock:%s", lockedID)
	ok, err := client.SetArgs(ctx, lockKey, "1", redis.SetArgs{Mode: "NX", TTL: 5 * time.Minute}).Result()
	require.NoError(t, err)
	require.Equal(t, "OK", ok)

	assert.True(t, w.isScheduledSendLockHeld(ctx, lockedID), "should detect held lock")

	unlockedID := uuid.New()
	assert.False(t, w.isScheduledSendLockHeld(ctx, unlockedID), "should detect absent lock")
}

func TestPollScheduledSends_FullE2EWithDB(t *testing.T) {
	db := testutil.SetupTestDB(t)

	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	org := testutil.CreateTestOrganization(t, db)
	instance := testutil.CreateTestWhatsAppAccount(t, db, org.ID)

	template := &models.Template{
		OrganizationID: org.ID,
		Name:           "poller-e2e-" + uuid.New().String()[:8],
		BodyContent:    "Hello {{name}}!",
		DisplayName:    "Poller Template",
	}
	require.NoError(t, db.Create(template).Error)

	testUser := testutil.CreateTestUser(t, db, org.ID)

	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		WhatsAppAccount: instance.Name,
		Name:            "Poller E2E Campaign",
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		MinDelaySeconds: 0,
		MaxDelaySeconds: 0,
		CreatedBy:       testUser.ID,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		CampaignID:    campaign.ID,
		PhoneNumber:   "+1234567890",
		RecipientName: "Test User",
		RecipientType: models.RecipientTypeIndividual,
		Status:        models.MessageStatusSending,
	}
	require.NoError(t, db.Create(recipient).Error)

	var sendCalled bool
	mockProvider := &mockMessageProvider{
		sendTextFunc: func(_ context.Context, _, _, _ string) (string, error) {
			sendCalled = true
			return "wamid.test.123", nil
		},
	}

	w := &Worker{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:              db,
		Redis:           client,
		Log:             testutil.NopLogger(),
		MessageProvider: mockProvider,
		Publisher:       queue.NewPublisher(client, testutil.NopLogger()),
	}

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		RecipientType:  models.RecipientTypeIndividual,
		EnqueuedAt:     time.Now().Add(-1 * time.Minute),
	}

	ss := scheduledSendFromRecipientJob(job, time.Now().Add(-30*time.Second))
	insertReadyZSETEntry(t, client, ss)

	err := w.pollScheduledSends(ctx)
	assert.NoError(t, err)

	assert.True(t, sendCalled, "provider SendText should have been called")

	var updated models.BulkMessageRecipient
	require.NoError(t, db.Where("id = ?", recipient.ID).First(&updated).Error)
	assert.Equal(t, models.MessageStatusSent, updated.Status, "recipient should be marked as sent")

	results, err := client.ZRange(ctx, scheduledSendsKey, 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, results, 0, "ZSET should be empty after successful send")
}

func TestPollScheduledSends_ProcessesPendingRecipientFromCrashWindow(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL required for pending crash-window test")
	}

	db := testutil.SetupTestDB(t)
	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	org := testutil.CreateTestOrganization(t, db)
	instance := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	template := &models.Template{
		OrganizationID: org.ID,
		Name:           "pending-crash-" + uuid.New().String()[:8],
		BodyContent:    "Hello!",
		DisplayName:    "Pending Crash Template",
	}
	require.NoError(t, db.Create(template).Error)
	testUser := testutil.CreateTestUser(t, db, org.ID)

	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		WhatsAppAccount: instance.Name,
		Name:            "Pending Crash Campaign",
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		CreatedBy:       testUser.ID,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		CampaignID:    campaign.ID,
		PhoneNumber:   "+1234567890",
		RecipientName: "Pending User",
		RecipientType: models.RecipientTypeIndividual,
		Status:        models.MessageStatusPending,
	}
	require.NoError(t, db.Create(recipient).Error)

	var sendCalled bool
	w := &Worker{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:    db,
		Redis: client,
		Log:   testutil.NopLogger(),
		MessageProvider: &mockMessageProvider{
			sendTextFunc: func(_ context.Context, _, _, _ string) (string, error) {
				sendCalled = true
				return "wamid.pending-crash", nil
			},
		},
		Publisher: queue.NewPublisher(client, testutil.NopLogger()),
	}

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		RecipientType:  models.RecipientTypeIndividual,
		EnqueuedAt:     time.Now(),
	}
	ss := scheduledSendFromRecipientJob(job, time.Now().Add(-30*time.Second))
	insertReadyZSETEntry(t, client, ss)

	require.NoError(t, w.pollScheduledSends(ctx))
	assert.True(t, sendCalled, "pending recipient from ZSET should still be sent")

	var updated models.BulkMessageRecipient
	require.NoError(t, db.Where("id = ?", recipient.ID).First(&updated).Error)
	assert.Equal(t, models.MessageStatusSent, updated.Status)
}

type mockMessageProvider struct {
	sendTextFunc func(ctx context.Context, instanceID, to, text string) (string, error)
}

func (m *mockMessageProvider) SendText(ctx context.Context, instanceID, to, text string) (string, error) {
	if m.sendTextFunc != nil {
		return m.sendTextFunc(ctx, instanceID, to, text)
	}
	return "wamid.mock-" + uuid.New().String()[:8], nil
}
func (m *mockMessageProvider) SendImage(ctx context.Context, instanceID, to, imageURL, caption string) (string, error) {
	return "", nil
}
func (m *mockMessageProvider) SendDocument(ctx context.Context, instanceID, to, docURL, filename, caption string) (string, error) {
	return "", nil
}
func (m *mockMessageProvider) SendVideo(ctx context.Context, instanceID, to, videoURL, caption string) (string, error) {
	return "", nil
}
func (m *mockMessageProvider) SendAudio(ctx context.Context, instanceID, to, audioURL string) (string, error) {
	return "", nil
}
func (m *mockMessageProvider) MarkRead(ctx context.Context, instanceID, messageID string) error {
	return nil
}
func (m *mockMessageProvider) SendReaction(ctx context.Context, instanceID, messageID, emoji string) error {
	return nil
}
func (m *mockMessageProvider) RevokeMessage(ctx context.Context, instanceID, messageID string) error {
	return nil
}
func (m *mockMessageProvider) GetMediaURL(ctx context.Context, instanceID, mediaID string) (string, error) {
	return "", nil
}
func (m *mockMessageProvider) DownloadMedia(ctx context.Context, instanceID, mediaURL string) ([]byte, error) {
	return nil, nil
}
func (m *mockMessageProvider) UploadMedia(ctx context.Context, instanceID, mediaType string, data []byte) (string, error) {
	return "", nil
}

func TestRecoverStuckSendingRecipients_ReEnqueues(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL required for recovery re-enqueue test")
	}

	db := testutil.SetupTestDB(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	org := testutil.CreateTestOrganization(t, db)
	instance := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	template := &models.Template{
		OrganizationID: org.ID,
		Name:           "recovery-test-" + uuid.New().String()[:8],
		BodyContent:    "Hello!",
		DisplayName:    "Recovery Template",
	}
	require.NoError(t, db.Create(template).Error)

	testUser := testutil.CreateTestUser(t, db, org.ID)

	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		WhatsAppAccount: instance.Name,
		Name:            "Recovery Test Campaign",
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		CreatedBy:       testUser.ID,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		CampaignID:    campaign.ID,
		PhoneNumber:   "+1234567890",
		RecipientName: "Stuck User",
		RecipientType: models.RecipientTypeIndividual,
		Status:        models.MessageStatusSending,
		SendAttemptID: "stale-attempt-123",
	}
	require.NoError(t, db.Create(recipient).Error)

	require.NoError(t, db.Model(recipient).Update("updated_at", time.Now().Add(-1*time.Hour)).Error)

	w := &Worker{
		Config:    &config.Config{WhatsApp: config.WhatsAppConfig{Provider: "meta"}},
		DB:        db,
		Redis:     client,
		Log:       testutil.NopLogger(),
		Publisher: queue.NewPublisher(client, testutil.NopLogger()),
	}

	err := w.recoverStuckSendingRecipients(context.Background())
	assert.NoError(t, err)

	var updated models.BulkMessageRecipient
	require.NoError(t, db.Where("id = ?", recipient.ID).First(&updated).Error)
	assert.Equal(t, models.MessageStatusPending, updated.Status, "recipient should be reset to pending")
	assert.Empty(t, updated.SendAttemptID, "send_attempt_id should be cleared on recovery")

	streamName := queue.CampaignStreamName(org.ID)
	streams, err := client.XRange(context.Background(), streamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, streams, 1, "recovered recipient should be re-enqueued in the org's campaign stream")
}

func TestExecuteRecipientSendWithAttempt_MismatchSkips(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL required for attempt ID guard test")
	}

	db := testutil.SetupTestDB(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	org := testutil.CreateTestOrganization(t, db)
	instance := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	template := &models.Template{
		OrganizationID: org.ID,
		Name:           "attempt-guard-" + uuid.New().String()[:8],
		BodyContent:    "Hello!",
		DisplayName:    "Attempt Guard Template",
	}
	require.NoError(t, db.Create(template).Error)
	testUser := testutil.CreateTestUser(t, db, org.ID)

	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		WhatsAppAccount: instance.Name,
		Name:            "Attempt Guard Campaign",
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		CreatedBy:       testUser.ID,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		CampaignID:    campaign.ID,
		PhoneNumber:   "+1234567890",
		RecipientName: "Test User",
		RecipientType: models.RecipientTypeIndividual,
		Status:        models.MessageStatusSending,
	}
	require.NoError(t, db.Create(recipient).Error)

	require.NoError(t, db.Model(recipient).Update("send_attempt_id", "original-attempt-123").Error)

	var sendCalled bool
	mockProvider := &mockMessageProvider{
		sendTextFunc: func(_ context.Context, _, _, _ string) (string, error) {
			sendCalled = true
			return "wamid.test.456", nil
		},
	}

	w := &Worker{
		Config:          &config.Config{WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"}},
		DB:              db,
		Redis:           client,
		Log:             testutil.NopLogger(),
		MessageProvider: mockProvider,
		Publisher:       queue.NewPublisher(client, testutil.NopLogger()),
	}

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		RecipientType:  models.RecipientTypeIndividual,
		EnqueuedAt:     time.Now(),
	}

	err := w.executeRecipientSendWithAttempt(context.Background(), job, "different-attempt-456")
	assert.NoError(t, err)
	assert.False(t, sendCalled, "provider should NOT be called when attempt ID mismatches")

	var updated models.BulkMessageRecipient
	require.NoError(t, db.Where("id = ?", recipient.ID).First(&updated).Error)
	assert.Equal(t, models.MessageStatusSending, updated.Status, "recipient should remain in sending state")
}

var _ provider.MessageProvider = (*mockMessageProvider)(nil)
