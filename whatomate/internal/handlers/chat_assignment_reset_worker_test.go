package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChatAssignmentResetTestApp(t *testing.T) *App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	return &App{
		Config: &config.Config{
			App:      config.AppConfig{EncryptionKey: testutil.TestEncryptionKey},
			JWT:      config.JWTConfig{Secret: testutil.TestJWTSecret},
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:  db,
		Log: testutil.NopLogger(),
	}
}

func createChatAssignmentResetInstance(
	t *testing.T,
	app *App,
	orgID uuid.UUID,
	name string,
	settings models.JSONB,
) *models.WhatsAppInstance {
	t.Helper()

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           name,
		Status:         models.InstanceStatusConnected,
		Settings:       settings,
	}
	require.NoError(t, app.DB.Create(instance).Error)
	return instance
}

func TestChatAssignmentResetWorker_ProcessInstance_ResetsAssignedChatsWhenDueAndScopesToInstance(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	org.Settings = models.JSONB{"timezone": "UTC"}
	require.NoError(t, app.DB.Save(org).Error)

	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	now := time.Now().UTC()

	targetInstance := createChatAssignmentResetInstance(t, app, org.ID, "Support", models.JSONB{
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Add(-24 * time.Hour).Format("2006-01-02"),
	})
	otherInstance := createChatAssignmentResetInstance(t, app, org.ID, "Sales", models.JSONB{
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Format("2006-01-02"),
	})

	assignedTarget := testutil.CreateTestContact(t, app.DB, org.ID)
	closedTarget := testutil.CreateTestContact(t, app.DB, org.ID)
	assignedOther := testutil.CreateTestContact(t, app.DB, org.ID)
	assignedWithoutInstance := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedTarget.ID).Updates(map[string]any{
		"instance_id":       targetInstance.ID,
		"assigned_user_id":  assignee.ID,
		"status":            models.ChatStatusOpen,
		"closed_at":         nil,
		"closed_by_user_id": nil,
	}).Error)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", closedTarget.ID).Updates(map[string]any{
		"instance_id":      targetInstance.ID,
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusClosed,
	}).Error)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedOther.ID).Updates(map[string]any{
		"instance_id":      otherInstance.ID,
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedWithoutInstance.ID).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	var storedInstance models.WhatsAppInstance
	require.NoError(t, app.DB.Where("id = ?", targetInstance.ID).First(&storedInstance).Error)
	require.NoError(t, worker.processInstance(now, storedInstance, "UTC"))

	var refreshedAssignedTarget models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedTarget.ID).First(&refreshedAssignedTarget).Error)
	assert.Nil(t, refreshedAssignedTarget.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, refreshedAssignedTarget.Status)

	var refreshedClosedTarget models.Contact
	require.NoError(t, app.DB.Where("id = ?", closedTarget.ID).First(&refreshedClosedTarget).Error)
	require.NotNil(t, refreshedClosedTarget.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedClosedTarget.AssignedUserID)
	assert.Equal(t, models.ChatStatusClosed, refreshedClosedTarget.Status)

	var refreshedAssignedOther models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedOther.ID).First(&refreshedAssignedOther).Error)
	require.NotNil(t, refreshedAssignedOther.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedAssignedOther.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedAssignedOther.Status)

	var refreshedAssignedWithoutInstance models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedWithoutInstance.ID).First(&refreshedAssignedWithoutInstance).Error)
	require.NotNil(t, refreshedAssignedWithoutInstance.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedAssignedWithoutInstance.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedAssignedWithoutInstance.Status)

	var resetSystemMessage models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND metadata->>'event_type' = ?", assignedTarget.ID, "chat_assignment_reset").
		Order("created_at DESC").
		First(&resetSystemMessage).Error)
	assert.Equal(t, models.DirectionOutgoing, resetSystemMessage.Direction)
	assert.Equal(t, true, resetSystemMessage.Metadata["system_event"])
	assert.Contains(t, resetSystemMessage.Content, "Assigned Chat Reset schedule")

	for _, untouchedContactID := range []uuid.UUID{closedTarget.ID, assignedOther.ID, assignedWithoutInstance.ID} {
		var count int64
		require.NoError(t, app.DB.Model(&models.Message{}).
			Where("contact_id = ? AND metadata->>'event_type' = ?", untouchedContactID, "chat_assignment_reset").
			Count(&count).Error)
		assert.Equal(t, int64(0), count)
	}

	require.NoError(t, app.DB.Where("id = ?", targetInstance.ID).First(&storedInstance).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedInstance.Settings[organizationSettingAssignedChatResetLastDate])

	var storedOtherInstance models.WhatsAppInstance
	require.NoError(t, app.DB.Where("id = ?", otherInstance.ID).First(&storedOtherInstance).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedOtherInstance.Settings[organizationSettingAssignedChatResetLastDate])
}

func TestChatAssignmentResetWorker_ProcessInstance_BootstrapsMidnightWithoutImmediateReset(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	instance := createChatAssignmentResetInstance(t, app, org.ID, "Support", models.JSONB{
		organizationSettingAssignedChatResetMode: string(ChatAssignmentResetModeMidnight),
		organizationSettingAssignedChatResetHour: 0,
	})
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"instance_id":      instance.ID,
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	now := time.Now().UTC()
	var storedInstance models.WhatsAppInstance
	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	require.NoError(t, worker.processInstance(now, storedInstance, "UTC"))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)

	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedInstance.Settings[organizationSettingAssignedChatResetLastDate])

	var messageCount int64
	require.NoError(t, app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'event_type' = ?", assignedContact.ID, "chat_assignment_reset").
		Count(&messageCount).Error)
	assert.Equal(t, int64(0), messageCount)
}

func TestChatAssignmentResetWorker_ProcessInstance_SkipsWhenAlreadyRunToday(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	now := time.Now().UTC()
	instance := createChatAssignmentResetInstance(t, app, org.ID, "Support", models.JSONB{
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Format("2006-01-02"),
	})
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"instance_id":      instance.ID,
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	var storedInstance models.WhatsAppInstance
	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	require.NoError(t, worker.processInstance(now, storedInstance, "UTC"))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)

	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	assert.Equal(t, now.Format("2006-01-02"), storedInstance.Settings[organizationSettingAssignedChatResetLastDate])
}

func TestChatAssignmentResetWorker_ProcessInstance_SkipsWhenDisabled(t *testing.T) {
	app := newChatAssignmentResetTestApp(t)
	worker := NewChatAssignmentResetWorker(app, time.Minute)

	org := testutil.CreateTestOrganization(t, app.DB)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	now := time.Now().UTC()
	instance := createChatAssignmentResetInstance(t, app, org.ID, "Support", models.JSONB{
		organizationSettingAssignedChatResetEnabled:  false,
		organizationSettingAssignedChatResetMode:     string(ChatAssignmentResetModeCustomHour),
		organizationSettingAssignedChatResetHour:     0,
		organizationSettingAssignedChatResetLastDate: now.Add(-24 * time.Hour).Format("2006-01-02"),
	})
	assignedContact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", assignedContact.ID).Updates(map[string]any{
		"instance_id":      instance.ID,
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	var storedInstance models.WhatsAppInstance
	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	require.NoError(t, worker.processInstance(now, storedInstance, "UTC"))

	var refreshedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", assignedContact.ID).First(&refreshedContact).Error)
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshedContact.Status)

	require.NoError(t, app.DB.Where("id = ?", instance.ID).First(&storedInstance).Error)
	assert.Equal(t, now.Add(-24*time.Hour).Format("2006-01-02"), storedInstance.Settings[organizationSettingAssignedChatResetLastDate])

	var messageCount int64
	require.NoError(t, app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'event_type' = ?", assignedContact.ID, "chat_assignment_reset").
		Count(&messageCount).Error)
	assert.Equal(t, int64(0), messageCount)
}
