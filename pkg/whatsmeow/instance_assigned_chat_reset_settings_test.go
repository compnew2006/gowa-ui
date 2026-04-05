package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureInstanceSettingsDefaults_InjectsAssignedChatResetDefaults(t *testing.T) {
	t.Parallel()

	normalized := EnsureInstanceSettingsDefaults(nil)
	assert.Equal(t, true, normalized[InstanceSettingAssignedChatResetEnabled])
	assert.Equal(t, string(AssignedChatResetModeMidnight), normalized[InstanceSettingAssignedChatResetMode])
	assert.Equal(t, 0, normalized[InstanceSettingAssignedChatResetHour])
}

func TestAssignedChatResetSettingsFromSettings_NormalizesValues(t *testing.T) {
	t.Parallel()

	settings := AssignedChatResetSettingsFromSettings(models.JSONB{
		InstanceSettingAssignedChatResetEnabled:  "false",
		InstanceSettingAssignedChatResetMode:     "custom_hour",
		InstanceSettingAssignedChatResetHour:     9,
		InstanceSettingAssignedChatResetLastDate: "2026-04-05",
	})

	assert.False(t, settings.Enabled)
	assert.Equal(t, AssignedChatResetModeCustomHour, settings.Mode)
	assert.Equal(t, 9, settings.Hour)
	assert.Equal(t, "2026-04-05", settings.LastResetDate)
}

func TestAssignedChatResetSettingsFromSettings_ForcesMidnightHourToZero(t *testing.T) {
	t.Parallel()

	settings := AssignedChatResetSettingsFromSettings(models.JSONB{
		InstanceSettingAssignedChatResetMode: "midnight",
		InstanceSettingAssignedChatResetHour: 17,
	})

	assert.Equal(t, AssignedChatResetModeMidnight, settings.Mode)
	assert.Equal(t, 0, settings.Hour)
}

func TestValidateInstanceSettings_AssignedChatResetAcceptsValidValues(t *testing.T) {
	t.Parallel()

	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingAssignedChatResetEnabled: true,
		InstanceSettingAssignedChatResetMode:    "custom_hour",
		InstanceSettingAssignedChatResetHour:    23,
	})
	require.NoError(t, err)
}

func TestValidateInstanceSettings_AssignedChatResetRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingAssignedChatResetEnabled: models.JSONB{},
		InstanceSettingAssignedChatResetMode:    "every_hour",
		InstanceSettingAssignedChatResetHour:    24,
	})
	require.Error(t, err)
}
