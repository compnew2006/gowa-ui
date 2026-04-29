package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveAutoCampaignWindowUsesExactLastGeneratedAt(t *testing.T) {
	location := time.FixedZone("UTC+2", 2*60*60)
	localNow := time.Date(2026, time.April, 8, 12, 1, 0, 0, location)
	lastGeneratedAt := time.Date(2026, time.April, 1, 10, 0, 30, 0, time.UTC)

	windowStart, windowEnd := resolveAutoCampaignWindow(localNow, &lastGeneratedAt, 7)

	assert.Equal(t, lastGeneratedAt.In(location), windowStart)
	assert.Equal(t, localNow, windowEnd)
}

func TestResolveAutoCampaignWindowFallsBackToIntervalOnFirstRun(t *testing.T) {
	location := time.FixedZone("UTC+2", 2*60*60)
	localNow := time.Date(2026, time.April, 8, 12, 1, 0, 0, location)

	windowStart, windowEnd := resolveAutoCampaignWindow(localNow, nil, 7)

	assert.Equal(t, localNow.AddDate(0, 0, -7), windowStart)
	assert.Equal(t, localNow, windowEnd)
}

func TestAutoCampaignHelpers(t *testing.T) {
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -7)

	assert.True(t, isAutoCampaignDue(now, nil, 7))
	assert.True(t, isAutoCampaignDue(now, &last, 7))
	assert.False(t, isAutoCampaignDue(now.Add(-time.Minute), &last, 7))
	assert.Equal(t, "prefix-week15-3/4-10/4", buildAutoCampaignName("prefix-", last, now))
	assert.Equal(t, "Hello {{contact_name}} {{phone_number}}", normalizeAutoCampaignMessageTemplate("Hello {contact_name} {{phone_number}}"))
}

func TestAutoCampaignWorkerPersistLastGeneratedAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	app := autoCampaignTestApp(db, testutil.NewMockQueue())
	org := testutil.CreateTestOrganization(t, db)
	instance := seedAutoCampaignInstance(t, db, org.ID, waManager.AutoCampaignSettings{Enabled: true, Message: "Hello", IntervalDays: 7})
	worker := NewInstanceAutoCampaignWorker(app, time.Minute)
	now := time.Now().UTC()

	require.NoError(t, worker.persistLastGeneratedAt(instance.ID, now))

	var updated models.WhatsAppInstance
	require.NoError(t, db.First(&updated, "id = ?", instance.ID).Error)
	settings := waManager.AutoCampaignSettingsFromSettings(updated.Settings)
	require.NotNil(t, settings.LastGeneratedAt)
	assert.WithinDuration(t, now, *settings.LastGeneratedAt, time.Second)
}

func TestAutoCampaignWorkerContactWindowAndDuplicatePrevention(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	app := autoCampaignTestApp(db, testutil.NewMockQueue())
	org := testutil.CreateTestOrganization(t, db)
	instance := seedAutoCampaignInstance(t, db, org.ID, waManager.AutoCampaignSettings{Enabled: true, Message: "Hello {contact_name}", IntervalDays: 7, TargetStatus: waManager.AutoCampaignTargetStatusDraft})
	worker := NewInstanceAutoCampaignWorker(app, time.Minute)
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	inside := now.Add(-24 * time.Hour)
	outside := now.Add(-10 * 24 * time.Hour)
	seedAutoCampaignContact(t, db, org.ID, instance.ID, "201000000001", "Inside", inside)
	seedAutoCampaignContact(t, db, org.ID, instance.ID, "201000000002", "Outside", outside)

	contacts, err := worker.loadInstanceContactsInWindow(org.ID, instance.ID, now.AddDate(0, 0, -7), now)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	assert.Equal(t, "201000000001", contacts[0].PhoneNumber)

	campaignName := buildAutoCampaignName("", now.AddDate(0, 0, -7), now)
	assert.False(t, mustCampaignNameExists(t, worker, org.ID, instance.ID.String(), campaignName, now.AddDate(0, 0, -7)))
	seedAutoCampaignCampaign(t, db, org.ID, instance.ID.String(), campaignName, now)
	assert.True(t, mustCampaignNameExists(t, worker, org.ID, instance.ID.String(), campaignName, now.AddDate(0, 0, -7)))
}

func TestAutoCampaignWorkerCreatesDraftWithMedia(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	queue := testutil.NewMockQueue()
	app := autoCampaignTestApp(db, queue)
	org := testutil.CreateTestOrganization(t, db)
	testutil.CreateTestUser(t, db, org.ID, testutil.WithEmail(testutil.UniqueEmail("auto-draft-user")))
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	instance := seedAutoCampaignInstance(t, db, org.ID, waManager.AutoCampaignSettings{
		Enabled:         true,
		Message:         "Hello {contact_name}",
		IntervalDays:    7,
		TargetStatus:    waManager.AutoCampaignTargetStatusDraft,
		MediaLocalPath:  "org/campaigns/image.jpg",
		MediaMimeType:   "image/jpeg",
		MediaFilename:   "image.jpg",
		MinDelayMinutes: 1,
		MaxDelayMinutes: 1,
	})
	seedAutoCampaignContact(t, db, org.ID, instance.ID, "201000000001", "Inside", now.Add(-24*time.Hour))
	worker := NewInstanceAutoCampaignWorker(app, time.Minute)

	require.NoError(t, worker.processInstance(now, *instance, "UTC"))

	var campaign models.BulkMessageCampaign
	require.NoError(t, db.Where("organization_id = ? AND whats_app_account = ?", org.ID, instance.ID.String()).First(&campaign).Error)
	assert.Equal(t, models.CampaignStatusDraft, campaign.Status)
	assert.Equal(t, "org/campaigns/image.jpg", campaign.HeaderMediaLocalPath)
	assert.Equal(t, "image/jpeg", campaign.HeaderMediaMimeType)
	assert.Equal(t, 0, queue.JobCount())
}

func TestAutoCampaignWorkerRunStartsOnlyWhenPolicyAllows(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	queue := testutil.NewMockQueue()
	app := autoCampaignTestApp(db, queue)
	org := testutil.CreateTestOrganization(t, db)
	testutil.CreateTestUser(t, db, org.ID, testutil.WithEmail(testutil.UniqueEmail("auto-run-user")))
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	instance := seedAutoCampaignInstance(t, db, org.ID, waManager.AutoCampaignSettings{
		Enabled:         true,
		Message:         "Hello {contact_name}",
		IntervalDays:    7,
		TargetStatus:    waManager.AutoCampaignTargetStatusRun,
		MinDelayMinutes: 1,
		MaxDelayMinutes: 1,
	})
	seedAutoCampaignContact(t, db, org.ID, instance.ID, "201000000001", "Inside", now.Add(-24*time.Hour))
	worker := NewInstanceAutoCampaignWorker(app, time.Minute)

	require.NoError(t, worker.processInstance(now, *instance, "UTC"))
	var campaign models.BulkMessageCampaign
	require.NoError(t, db.Where("organization_id = ? AND whats_app_account = ?", org.ID, instance.ID.String()).First(&campaign).Error)
	assert.Equal(t, models.CampaignStatusProcessing, campaign.Status)
	assert.Equal(t, 1, queue.JobCount())

	blockedInstance := seedAutoCampaignInstance(t, db, org.ID, waManager.AutoCampaignSettings{
		Enabled:         true,
		Message:         "Hello",
		IntervalDays:    7,
		TargetStatus:    waManager.AutoCampaignTargetStatusRun,
		MinDelayMinutes: 1,
		MaxDelayMinutes: 1,
	})
	require.NoError(t, db.Model(blockedInstance).Update("status", models.InstanceStatusDisconnected).Error)
	seedAutoCampaignContact(t, db, org.ID, blockedInstance.ID, "201000000002", "Blocked", now.Add(-24*time.Hour))
	require.NoError(t, worker.processInstance(now, *blockedInstance, "UTC"))
	var blockedCampaign models.BulkMessageCampaign
	require.NoError(t, db.Where("organization_id = ? AND whats_app_account = ?", org.ID, blockedInstance.ID.String()).First(&blockedCampaign).Error)
	assert.Equal(t, models.CampaignStatusDraft, blockedCampaign.Status)
}

func autoCampaignTestApp(db *gorm.DB, queue *testutil.MockQueue) *App {
	return &App{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:    db,
		Queue: queue,
		Log:   testutil.NopLogger(),
	}
}

func seedAutoCampaignInstance(t *testing.T, db *gorm.DB, orgID uuid.UUID, settings waManager.AutoCampaignSettings) *models.WhatsAppInstance {
	t.Helper()
	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "Auto Instance",
		PhoneNumber:    "15550001111",
		Status:         models.InstanceStatusConnected,
		Settings:       models.JSONB{waManager.InstanceSettingAutoCampaign: settings.ToJSONB()},
	}
	require.NoError(t, db.Create(instance).Error)
	return instance
}

func seedAutoCampaignContact(t *testing.T, db *gorm.DB, orgID, instanceID uuid.UUID, phone, name string, lastInboundAt time.Time) {
	t.Helper()
	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		InstanceID:     &instanceID,
		PhoneNumber:    phone,
		ProfileName:    name,
		LastInboundAt:  &lastInboundAt,
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&contact).Error)
}

func seedAutoCampaignCampaign(t *testing.T, db *gorm.DB, orgID uuid.UUID, account, name string, createdAt time.Time) {
	t.Helper()
	user := testutil.CreateTestUser(t, db, orgID, testutil.WithEmail(testutil.UniqueEmail("auto-dup-user")))
	template := testutil.CreateTestTemplate(t, db, orgID, account)
	campaign := models.BulkMessageCampaign{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt},
		OrganizationID:  orgID,
		WhatsAppAccount: account,
		Name:            name,
		TemplateID:      template.ID,
		Status:          models.CampaignStatusDraft,
		CreatedBy:       user.ID,
		MinDelaySeconds: 10,
		MaxDelaySeconds: 10,
	}
	require.NoError(t, db.Create(&campaign).Error)
}

func mustCampaignNameExists(t *testing.T, worker *InstanceAutoCampaignWorker, orgID uuid.UUID, account, name string, createdAfter time.Time) bool {
	t.Helper()
	exists, err := worker.campaignNameExists(orgID, account, name, createdAfter)
	require.NoError(t, err)
	return exists
}
