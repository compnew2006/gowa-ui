package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureInstanceSettingsDefaults_InjectsChatCloseRatingDefaults(t *testing.T) {
	t.Parallel()

	normalized := EnsureInstanceSettingsDefaults(nil)
	assert.Equal(t, true, normalized[InstanceSettingChatCloseRatingEnabled])
	assert.Equal(t, defaultChatCloseRatingFollowupWindowMinutes, normalized[InstanceSettingChatCloseRatingFollowupWindowMinutes])
}

func TestValidateInstanceSettings_ChatCloseRatingAcceptsValidValues(t *testing.T) {
	t.Parallel()

	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingChatCloseRatingEnabled:               false,
		InstanceSettingChatCloseRatingFollowupWindowMinutes: 30,
		InstanceSettingChatCloseRatingTemplates: models.JSONB{
			"en": "Rate us",
			"ar": "قيمنا",
		},
	})
	require.NoError(t, err)
}

func TestValidateInstanceSettings_ChatCloseRatingRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	err := ValidateInstanceSettings(models.JSONB{
		InstanceSettingChatCloseRatingEnabled:               models.JSONB{},
		InstanceSettingChatCloseRatingFollowupWindowMinutes: 0,
		InstanceSettingChatCloseRatingTemplates: models.JSONB{
			"en": 99,
		},
	})
	require.Error(t, err)
}
