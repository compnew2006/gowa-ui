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
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name: "enabled_without_message",
			input: map[string]any{
				"enabled": true,
			},
			wantErr: true,
		},
		{
			name: "enabled_message_empty_interval_zero",
			input: map[string]any{
				"enabled":       true,
				"message":       "hello",
				"interval_days": 0,
			},
			wantErr: true,
		},
		{
			name: "invalid_target_status",
			input: map[string]any{
				"enabled":       true,
				"message":       "hello",
				"interval_days": 5,
				"target_status": "queued",
			},
			wantErr: true,
		},
		{
			name: "min_delay_greater_than_max_delay",
			input: map[string]any{
				"enabled":           true,
				"message":           "hello",
				"interval_days":     5,
				"min_delay_minutes": 30,
				"max_delay_minutes": 10,
			},
			wantErr: true,
		},
		{
			name: "media_path_traversal",
			input: map[string]any{
				"enabled":          true,
				"media_local_path": "../etc/passwd",
				"interval_days":    5,
			},
			wantErr: true,
		},
		{
			name: "valid_full_config",
			input: map[string]any{
				"enabled":           true,
				"message":           "hello",
				"interval_days":     5,
				"min_delay_minutes": 15,
				"max_delay_minutes": 25,
				"target_status":     AutoCampaignTargetStatusRun,
			},
			wantErr: false,
		},
		{
			name: "disabled_no_message_needed",
			input: map[string]any{
				"enabled":       false,
				"interval_days": 3,
			},
			wantErr: false,
		},
		{
			name: "interval_days_zero",
			input: map[string]any{
				"enabled":       false,
				"interval_days": 0,
			},
			wantErr: true,
		},
		{
			name: "interval_days_negative",
			input: map[string]any{
				"enabled":       false,
				"interval_days": -1,
			},
			wantErr: true,
		},
		{
			name: "interval_days_365_allowed",
			input: map[string]any{
				"enabled":       false,
				"interval_days": 365,
			},
			wantErr: false,
		},
		{
			name: "interval_days_366_rejected",
			input: map[string]any{
				"enabled":       false,
				"interval_days": 366,
			},
			wantErr: true,
		},
		{
			name: "interval_days_boundary_1",
			input: map[string]any{
				"enabled":       false,
				"interval_days": 1,
			},
			wantErr: false,
		},
		{
			name: "min_delay_negative",
			input: map[string]any{
				"enabled":           false,
				"interval_days":     5,
				"min_delay_minutes": -5,
			},
			wantErr: true,
		},
		{
			name: "max_delay_negative",
			input: map[string]any{
				"enabled":           false,
				"interval_days":     5,
				"max_delay_minutes": -1,
			},
			wantErr: true,
		},
		{
			name: "min_max_delay_equal",
			input: map[string]any{
				"enabled":           true,
				"message":           "promo",
				"interval_days":     7,
				"min_delay_minutes": 10,
				"max_delay_minutes": 10,
				"target_status":     AutoCampaignTargetStatusDraft,
			},
			wantErr: false,
		},
		{
			name: "min_max_delay_zero",
			input: map[string]any{
				"enabled":           true,
				"message":           "promo",
				"interval_days":     7,
				"min_delay_minutes": 0,
				"max_delay_minutes": 0,
				"target_status":     AutoCampaignTargetStatusDraft,
			},
			wantErr: false,
		},
		{
			name: "target_status_draft",
			input: map[string]any{
				"enabled":       true,
				"message":       "hi",
				"interval_days": 3,
				"target_status": AutoCampaignTargetStatusDraft,
			},
			wantErr: false,
		},
		{
			name: "target_status_run",
			input: map[string]any{
				"enabled":       true,
				"message":       "hi",
				"interval_days": 3,
				"target_status": AutoCampaignTargetStatusRun,
			},
			wantErr: false,
		},
		{
			name: "target_status_case_insensitive",
			input: map[string]any{
				"enabled":       true,
				"message":       "hi",
				"interval_days": 3,
				"target_status": "DRAFT",
			},
			wantErr: false,
		},
		{
			name: "media_path_absolute",
			input: map[string]any{
				"enabled":          false,
				"interval_days":    5,
				"media_local_path": "/etc/passwd",
			},
			wantErr: true,
		},
		{
			name: "media_path_dot",
			input: map[string]any{
				"enabled":          false,
				"interval_days":    5,
				"media_local_path": ".",
			},
			wantErr: true,
		},
		{
			name: "media_path_valid_relative",
			input: map[string]any{
				"enabled":          false,
				"interval_days":    5,
				"media_local_path": "campaigns/photo.jpg",
			},
			wantErr: false,
		},
		{
			name: "last_generated_at_invalid",
			input: map[string]any{
				"enabled":           false,
				"interval_days":     5,
				"last_generated_at": "not-a-date",
			},
			wantErr: true,
		},
		{
			name: "last_generated_at_valid_rfc3339",
			input: map[string]any{
				"enabled":           false,
				"interval_days":     5,
				"last_generated_at": "2026-02-24T10:30:00Z",
			},
			wantErr: false,
		},
		{
			name:    "nil_input_uses_defaults",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "empty_map_uses_defaults",
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name: "valid_minimal_enabled",
			input: map[string]any{
				"enabled":       true,
				"message":       "hi",
				"interval_days": 1,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAutoCampaignSettings(tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
