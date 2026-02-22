package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChatAssignmentResetTestApp(t *testing.T) *App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set, skipping test")
	}

	return &App{
		Config: &config.Config{
			App: config.AppConfig{EncryptionKey: "test-encryption-key"},
			JWT: config.JWTConfig{Secret: testutil.TestJWTSecret},
		},
		DB:    db,
		Redis: redisClient,
		Log:   testutil.NopLogger(),
	}
}

func TestChatAssignmentResetWorker_ProcessOrganization_ResetsAssignedChatsWhenDue(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)

	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	closedContact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", closedContact.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusClosed,
	}).Error)

	now := time.Now().UTC()
	org.Settings = models.JSONB{
		"timezone":                                   "UTC",
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Add(-24 * time.Hour).Format("2006-01-02"),
	}
	require.NoError(t, app.DB.Save(org).Error)

	var storedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	require.NoError(t, worker.processOrganization(now, storedOrg))

	var refreshedAssigned models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedAssigned).Error)
	assert.Nil(t, refreshedAssigned.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, refreshedAssigned.Status)

	var refreshedClosed models.Contact
	require.NoError(t, app.DB.Where("id = ?", closedContact.ID).First(&refreshedClosed).Error)
	require.NotNil(t, refreshedClosed.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedClosed.AssignedUserID)
	assert.Equal(t, models.ChatStatusClosed, refreshedClosed.Status)

	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedOrg.Settings[organizationSettingAssignedChatResetLastDate])
}

func TestChatAssignmentResetWorker_ProcessOrganization_BootstrapsMidnightWithoutImmediateReset(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	now := time.Now().UTC()
	org.Settings = models.JSONB{
		"timezone":                               "UTC",
		organizationSettingAssignedChatResetMode: string(ChatAssignmentResetModeMidnight),
		organizationSettingAssignedChatResetHour: 0,
	}
	require.NoError(t, app.DB.Save(org).Error)

	var storedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	require.NoError(t, worker.processOrganization(now, storedOrg))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)

	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedOrg.Settings[organizationSettingAssignedChatResetLastDate])
}

func TestChatAssignmentResetWorker_ProcessOrganization_SkipsWhenAlreadyRunToday(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	now := time.Now().UTC()
	org.Settings = models.JSONB{
		"timezone":                                   "UTC",
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Format("2006-01-02"),
	}
	require.NoError(t, app.DB.Save(org).Error)

	var storedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	require.NoError(t, worker.processOrganization(now, storedOrg))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)
}

func TestChatAssignmentResetWorker_ProcessOrganization_SkipsWhenDisabled(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	now := time.Now().UTC()
	org.Settings = models.JSONB{
		"timezone": "UTC",
		organizationSettingAssignedChatResetEnabled:  false,
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Add(-24 * time.Hour).Format("2006-01-02"),
	}
	require.NoError(t, app.DB.Save(org).Error)

	var storedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	require.NoError(t, worker.processOrganization(now, storedOrg))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)

	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&storedOrg).Error)
	assert.Equal(t, now.Add(-24*time.Hour).Format("2006-01-02"), storedOrg.Settings[organizationSettingAssignedChatResetLastDate])
}
