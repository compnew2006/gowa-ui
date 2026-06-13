package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewLeavesDisabledCampaignConsumerNil(t *testing.T) {
	t.Parallel()

	client := setupScalerRedis(t)
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			BaseURL: "https://graph.facebook.com",
		},
	}

	w, err := New(cfg, nil, client, testutil.NopLogger(), nil, nil, WorkerOptions{
		EnableCampaignConsumer: false,
		EnableInboundMedia:     true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.Consumer != nil {
		t.Fatal("Consumer should be nil when campaign consumer is disabled")
	}
	if len(w.InboundConsumers) == 0 {
		t.Fatal("InboundConsumers should be initialized when inbound media is enabled")
	}
}

func TestNewLeavesDisabledInboundConsumerNil(t *testing.T) {
	t.Parallel()

	client := setupScalerRedis(t)
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			BaseURL: "https://graph.facebook.com",
		},
	}

	w, err := New(cfg, nil, client, testutil.NopLogger(), nil, nil, WorkerOptions{
		EnableCampaignConsumer: true,
		EnableInboundMedia:     false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.Consumer == nil {
		t.Fatal("Consumer should be initialized when campaign consumer is enabled")
	}
	if len(w.InboundConsumers) != 0 {
		t.Fatal("InboundConsumers should be empty when inbound media is disabled")
	}
}

func TestHandleRecipientJobSkipsPausedAndCancelledCampaigns(t *testing.T) {
	for _, status := range []models.CampaignStatus{models.CampaignStatusPaused, models.CampaignStatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			db := testutil.SetupTestDB(t)
			testutil.TruncateTables(db)
			org, campaign, recipient := seedWorkerCampaign(t, db, status, models.MessageStatusPending)
			w := &Worker{DB: db, Log: testutil.NopLogger(), Config: &config.Config{}}

			err := w.HandleRecipientJob(context.Background(), &queue.RecipientJob{
				CampaignID:     campaign.ID,
				RecipientID:    recipient.ID,
				OrganizationID: org.ID,
				PhoneNumber:    recipient.PhoneNumber,
				RecipientName:  recipient.RecipientName,
			})
			require.NoError(t, err)

			var updated models.BulkMessageRecipient
			require.NoError(t, db.First(&updated, "id = ?", recipient.ID).Error)
			assert.Equal(t, models.MessageStatusPending, updated.Status)
		})
	}
}

func TestHandleRecipientJobSkipsDuplicateInFlightJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set")
	}
	org, campaign, recipient := seedWorkerCampaign(t, db, models.CampaignStatusProcessing, models.MessageStatusPending)
	require.NoError(t, redisClient.Set(context.Background(), recipientLockKey(recipient.ID), "1", recipientLockTTL).Err())
	t.Cleanup(func() { _ = redisClient.Del(context.Background(), recipientLockKey(recipient.ID)).Err() })
	w := &Worker{DB: db, Redis: redisClient, Log: testutil.NopLogger(), Config: &config.Config{}}

	err := w.HandleRecipientJob(context.Background(), &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
	})
	require.NoError(t, err)

	var updated models.BulkMessageRecipient
	require.NoError(t, db.First(&updated, "id = ?", recipient.ID).Error)
	assert.Equal(t, models.MessageStatusPending, updated.Status)
}

func TestCampaignTemplatePlaceholdersRender(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	org := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Agent Smith"), testutil.WithEmail(testutil.UniqueEmail("worker-agent")))
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "201000000000",
		ProfileName:    "Customer One",
		AssignedUserID: &user.ID,
	}
	require.NoError(t, db.Create(contact).Error)
	w := &Worker{DB: db}

	params := w.resolveCampaignTemplateParams(context.Background(), org.ID, contact, "201", "", "Hello {{contact_name}} from {{agent_name}} at {{organization_name}}", models.JSONB{})
	rendered := renderCampaignTemplateBody("Hello {{contact_name}} from {{agent_name}} at {{organization_name}}", params)

	assert.Contains(t, rendered, "Customer One")
	assert.Contains(t, rendered, "Agent Smith")
	assert.Contains(t, rendered, org.Name)
}

func TestCheckCampaignCompletionUsesCompareAndSwap(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set")
	}
	org, campaign, recipient := seedWorkerCampaign(t, db, models.CampaignStatusProcessing, models.MessageStatusSent)
	require.NoError(t, db.Model(recipient).Update("status", models.MessageStatusSent).Error)
	w := &Worker{DB: db, Redis: redisClient, Log: testutil.NopLogger(), Publisher: queue.NewPublisher(redisClient, testutil.NopLogger())}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.checkCampaignCompletion(context.Background(), campaign.ID, org.ID)
		}()
	}
	wg.Wait()

	var updated models.BulkMessageCampaign
	require.NoError(t, db.First(&updated, "id = ?", campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusCompleted, updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

func TestIncrementCampaignCountIsAtomic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	_, campaign, _ := seedWorkerCampaign(t, db, models.CampaignStatusProcessing, models.MessageStatusPending)
	w := &Worker{DB: db, Log: testutil.NopLogger()}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.incrementCampaignCount(campaign.ID, "sent_count")
		}()
	}
	wg.Wait()

	var updated models.BulkMessageCampaign
	require.NoError(t, db.First(&updated, "id = ?", campaign.ID).Error)
	assert.Equal(t, 10, updated.SentCount)
}

func TestClassifyCampaignMediaType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		filename string
		want     string
	}{
		{name: "image mime", mimeType: "image/png", filename: "file.bin", want: "image"},
		{name: "video extension", filename: "clip.mp4", want: "video"},
		{name: "audio mime", mimeType: "audio/ogg", filename: "voice.bin", want: "audio"},
		{name: "document fallback", filename: "file.pdf", want: "document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyCampaignMediaType(tt.mimeType, tt.filename))
		})
	}
}

func seedWorkerCampaign(t *testing.T, db *gorm.DB, campaignStatus models.CampaignStatus, recipientStatus models.MessageStatus) (*models.Organization, *models.BulkMessageCampaign, *models.BulkMessageRecipient) {
	t.Helper()

	org := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, org.ID, testutil.WithEmail(testutil.UniqueEmail("worker-user")))
	template := testutil.CreateTestTemplate(t, db, org.ID, "worker-account")
	startedAt := time.Now().UTC()
	campaign := &models.BulkMessageCampaign{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "worker-account",
		Name:            "Worker Campaign",
		TemplateID:      template.ID,
		Status:          campaignStatus,
		TotalRecipients: 1,
		MinDelaySeconds: 10,
		MaxDelaySeconds: 10,
		CreatedBy:       user.ID,
		ScheduledAt:     nil,
		StartedAt:       &startedAt,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		CampaignID:      campaign.ID,
		PhoneNumber:     "201000000000",
		PhoneNormalized: "201000000000",
		RecipientName:   "Worker Recipient",
		TemplateParams:  models.JSONB{},
		Status:          recipientStatus,
	}
	require.NoError(t, db.Create(recipient).Error)

	return org, campaign, recipient
}
