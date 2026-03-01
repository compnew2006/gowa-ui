package whatsmeow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAutoCampaignSettings_Defaults(t *testing.T) {
	normalized := NormalizeAutoCampaignSettings(nil)

	assert.False(t, normalized.Enabled)
	assert.Equal(t, "", normalized.NamePrefix)
	assert.Equal(t, "", normalized.Message)
	assert.Equal(t, 7, normalized.IntervalDays)
	assert.Equal(t, 1, normalized.MinDelayMinutes)
	assert.Equal(t, 3, normalized.MaxDelayMinutes)
	assert.Equal(t, AutoCampaignTargetStatusDraft, normalized.TargetStatus)
	assert.Equal(t, "", normalized.MediaLocalPath)
	assert.Nil(t, normalized.LastGeneratedAt)
}

func TestNormalizeAutoCampaignSettings_CleansValues(t *testing.T) {
	normalized := NormalizeAutoCampaignSettings(map[string]any{
		"enabled":           "true",
		"name_prefix":       "  promo-  ",
		"message":           "  Welcome back  ",
		"interval_days":     float64(3),
		"min_delay_minutes": float64(15),
		"max_delay_minutes": float64(25),
		"target_status":     "RUN",
		"media_local_path":  "campaigns/media.jpg",
		"media_mime_type":   "image/jpeg",
		"media_filename":    "media.jpg",
		"last_generated_at": "2026-02-24T10:30:00Z",
	})

	require.NotNil(t, normalized.LastGeneratedAt)
	assert.True(t, normalized.Enabled)
	assert.Equal(t, "promo-", normalized.NamePrefix)
	assert.Equal(t, "Welcome back", normalized.Message)
	assert.Equal(t, 3, normalized.IntervalDays)
	assert.Equal(t, 15, normalized.MinDelayMinutes)
	assert.Equal(t, 25, normalized.MaxDelayMinutes)
	assert.Equal(t, AutoCampaignTargetStatusRun, normalized.TargetStatus)
	assert.Equal(t, "campaigns/media.jpg", normalized.MediaLocalPath)
	assert.Equal(t, "image/jpeg", normalized.MediaMimeType)
	assert.Equal(t, "media.jpg", normalized.MediaFilename)
	assert.Equal(t, "2026-02-24T10:30:00Z", normalized.LastGeneratedAt.UTC().Format(time.RFC3339))
}

func TestValidateAutoCampaignSettings(t *testing.T) {
	err := ValidateAutoCampaignSettings(map[string]any{
		"enabled": true,
	})
	require.Error(t, err)

	err = ValidateAutoCampaignSettings(map[string]any{
		"enabled":       true,
		"message":       "hello",
		"interval_days": 0,
	})
	require.Error(t, err)

	err = ValidateAutoCampaignSettings(map[string]any{
		"enabled":       true,
		"message":       "hello",
		"interval_days": 5,
		"target_status": "queued",
	})
	require.Error(t, err)

	err = ValidateAutoCampaignSettings(map[string]any{
		"enabled":           true,
		"message":           "hello",
		"interval_days":     5,
		"min_delay_minutes": 30,
		"max_delay_minutes": 10,
	})
	require.Error(t, err)

	err = ValidateAutoCampaignSettings(map[string]any{
		"enabled":          true,
		"media_local_path": "../etc/passwd",
		"interval_days":    5,
	})
	require.Error(t, err)

	err = ValidateAutoCampaignSettings(map[string]any{
		"enabled":           true,
		"message":           "hello",
		"interval_days":     5,
		"min_delay_minutes": 15,
		"max_delay_minutes": 25,
		"target_status":     AutoCampaignTargetStatusRun,
	})
	require.NoError(t, err)
}

func TestEnsureInstanceSettingsDefaults_InjectsAutoCampaign(t *testing.T) {
	normalized := EnsureInstanceSettingsDefaults(nil)
	autoCampaignRaw, ok := normalized[InstanceSettingAutoCampaign]
	require.True(t, ok)
	autoCampaign := NormalizeAutoCampaignSettings(autoCampaignRaw)
	assert.Equal(t, 7, autoCampaign.IntervalDays)
	assert.Equal(t, 1, autoCampaign.MinDelayMinutes)
	assert.Equal(t, 3, autoCampaign.MaxDelayMinutes)
	assert.Equal(t, AutoCampaignTargetStatusDraft, autoCampaign.TargetStatus)
}
