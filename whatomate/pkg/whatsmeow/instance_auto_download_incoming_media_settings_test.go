package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureInstanceSettingsDefaults_InjectsAutoDownloadIncomingMedia(t *testing.T) {
	normalized := EnsureInstanceSettingsDefaults(nil)
	raw, ok := normalized[InstanceSettingAutoDownloadIncomingMedia]
	require.True(t, ok)
	assert.Equal(t, false, raw)
	assert.False(t, IsAutoDownloadIncomingMediaEnabled(normalized))
}

func TestIsAutoDownloadIncomingMediaEnabled_ParsesBooleanInputs(t *testing.T) {
	assert.True(t, IsAutoDownloadIncomingMediaEnabled(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: true,
	}))
	assert.True(t, IsAutoDownloadIncomingMediaEnabled(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: "true",
	}))
	assert.False(t, IsAutoDownloadIncomingMediaEnabled(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: "false",
	}))
	assert.True(t, IsAutoDownloadIncomingMediaEnabled(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: 1,
	}))
	assert.False(t, IsAutoDownloadIncomingMediaEnabled(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: 0,
	}))
}

func TestValidateInstanceSettings_AutoDownloadIncomingMediaRejectsObjects(t *testing.T) {
	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: models.JSONB{"enabled": true},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), InstanceSettingAutoDownloadIncomingMedia)
}

func TestValidateInstanceSettings_AutoDownloadIncomingMediaNoRegression(t *testing.T) {
	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingAutoDownloadIncomingMedia: true,
		InstanceSettingAutoRejectCalls: models.JSONB{
			"enabled": true,
			"mode":    AutoRejectCallModeWithMessage,
			"message": "Call declined",
		},
		InstanceSettingAutoCampaign: models.JSONB{
			"enabled":       true,
			"message":       "hello",
			"interval_days": 3,
			"target_status": AutoCampaignTargetStatusDraft,
		},
	})
	require.NoError(t, err)
}
