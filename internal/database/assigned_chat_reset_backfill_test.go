package database

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillInstanceAssignedChatResetSettings_CopiesLegacyOrgSettingsOnce(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := testutil.CreateTestOrganization(t, db)
	org.Settings = models.JSONB{
		waManager.InstanceSettingAssignedChatResetEnabled:  true,
		waManager.InstanceSettingAssignedChatResetMode:     "custom_hour",
		waManager.InstanceSettingAssignedChatResetHour:     9,
		waManager.InstanceSettingAssignedChatResetLastDate: "2026-04-04",
	}
	require.NoError(t, db.Save(org).Error)

	instanceWithoutSettings := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Support",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{"custom_existing_setting": "keep-me"},
	}
	require.NoError(t, db.Create(instanceWithoutSettings).Error)

	instanceWithOwnSettings := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Sales",
		Status:         models.InstanceStatusDisconnected,
		Settings: models.JSONB{
			waManager.InstanceSettingAssignedChatResetEnabled:  false,
			waManager.InstanceSettingAssignedChatResetMode:     "custom_hour",
			waManager.InstanceSettingAssignedChatResetHour:     17,
			waManager.InstanceSettingAssignedChatResetLastDate: "2026-04-03",
		},
	}
	require.NoError(t, db.Create(instanceWithOwnSettings).Error)

	require.NoError(t, BackfillInstanceAssignedChatResetSettings(db))
	require.NoError(t, BackfillInstanceAssignedChatResetSettings(db))

	var refreshedOrg models.Organization
	require.NoError(t, db.Where("id = ?", org.ID).First(&refreshedOrg).Error)
	assert.NotContains(t, refreshedOrg.Settings, waManager.InstanceSettingAssignedChatResetEnabled)
	assert.NotContains(t, refreshedOrg.Settings, waManager.InstanceSettingAssignedChatResetMode)
	assert.NotContains(t, refreshedOrg.Settings, waManager.InstanceSettingAssignedChatResetHour)
	assert.NotContains(t, refreshedOrg.Settings, waManager.InstanceSettingAssignedChatResetLastDate)

	var refreshedInherited models.WhatsAppInstance
	require.NoError(t, db.Where("id = ?", instanceWithoutSettings.ID).First(&refreshedInherited).Error)
	assert.Equal(t, "keep-me", refreshedInherited.Settings["custom_existing_setting"])
	assert.Equal(t, true, refreshedInherited.Settings[waManager.InstanceSettingAssignedChatResetEnabled])
	assert.Equal(t, "custom_hour", refreshedInherited.Settings[waManager.InstanceSettingAssignedChatResetMode])
	assert.Equal(t, 9.0, refreshedInherited.Settings[waManager.InstanceSettingAssignedChatResetHour])
	assert.Equal(t, "2026-04-04", refreshedInherited.Settings[waManager.InstanceSettingAssignedChatResetLastDate])

	var refreshedOwn models.WhatsAppInstance
	require.NoError(t, db.Where("id = ?", instanceWithOwnSettings.ID).First(&refreshedOwn).Error)
	assert.Equal(t, false, refreshedOwn.Settings[waManager.InstanceSettingAssignedChatResetEnabled])
	assert.Equal(t, "custom_hour", refreshedOwn.Settings[waManager.InstanceSettingAssignedChatResetMode])
	assert.Equal(t, 17.0, refreshedOwn.Settings[waManager.InstanceSettingAssignedChatResetHour])
	assert.Equal(t, "2026-04-03", refreshedOwn.Settings[waManager.InstanceSettingAssignedChatResetLastDate])
}

func TestBackfillInstanceAssignedChatResetSettings_NilDatabase(t *testing.T) {
	t.Parallel()

	err := BackfillInstanceAssignedChatResetSettings(nil)
	require.Error(t, err)
}
