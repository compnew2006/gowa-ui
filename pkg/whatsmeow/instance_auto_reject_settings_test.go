package whatsmeow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAutoRejectCallSettings_Defaults(t *testing.T) {
	normalized := NormalizeAutoRejectCallSettings(nil)

	assert.False(t, normalized.Enabled)
	assert.Equal(t, AutoRejectCallModeWithoutMessage, normalized.Mode)
	assert.Equal(t, AutoRejectScheduleAlways, normalized.Schedule.Type)
	assert.Equal(t, "09:00", normalized.Schedule.Start)
	assert.Equal(t, "18:00", normalized.Schedule.End)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, normalized.Schedule.Days)
	assert.Equal(t, "UTC", normalized.Schedule.Timezone)
	assert.True(t, normalized.RejectIndividualCalls)
	assert.True(t, normalized.RejectGroupCalls)
	assert.Empty(t, normalized.BypassContacts)
}

func TestNormalizeAutoRejectCallSettings_CleansValues(t *testing.T) {
	normalized := NormalizeAutoRejectCallSettings(map[string]any{
		"enabled":                 true,
		"mode":                    AutoRejectCallModeWithMessage,
		"message":                 "  Busy right now  ",
		"reject_individual_calls": "true",
		"reject_group_calls":      0,
		"bypass_contacts": []any{
			" +1 (555) 123-4567 ",
			"15551234567",
			"+44 7777 888999",
		},
		"schedule": map[string]any{
			"type":     AutoRejectScheduleCustomHours,
			"start":    "08:30",
			"end":      "17:45",
			"days":     []any{1, "2", 6, 99},
			"timezone": "America/New_York",
		},
	})

	assert.True(t, normalized.Enabled)
	assert.Equal(t, AutoRejectCallModeWithMessage, normalized.Mode)
	assert.Equal(t, "Busy right now", normalized.Message)
	assert.True(t, normalized.RejectIndividualCalls)
	assert.False(t, normalized.RejectGroupCalls)
	assert.Equal(t, []string{"15551234567", "447777888999"}, normalized.BypassContacts)
	assert.Equal(t, AutoRejectScheduleCustomHours, normalized.Schedule.Type)
	assert.Equal(t, "08:30", normalized.Schedule.Start)
	assert.Equal(t, "17:45", normalized.Schedule.End)
	assert.Equal(t, []int{1, 2, 6}, normalized.Schedule.Days)
	assert.Equal(t, "America/New_York", normalized.Schedule.Timezone)
}

func TestValidateAutoRejectCallSettings(t *testing.T) {
	err := ValidateAutoRejectCallSettings(map[string]any{
		"enabled": true,
		"mode":    AutoRejectCallModeWithMessage,
		"message": "",
		"schedule": map[string]any{
			"type": AutoRejectScheduleAlways,
		},
	})
	require.Error(t, err)

	err = ValidateAutoRejectCallSettings(map[string]any{
		"enabled": true,
		"mode":    AutoRejectCallModeWithoutMessage,
		"schedule": map[string]any{
			"type":     AutoRejectScheduleCustomHours,
			"start":    "bad",
			"end":      "17:00",
			"days":     []int{1},
			"timezone": "UTC",
		},
	})
	require.Error(t, err)

	err = ValidateAutoRejectCallSettings(map[string]any{
		"enabled": true,
		"mode":    AutoRejectCallModeWithMessage,
		"message": "Away",
		"schedule": map[string]any{
			"type":     AutoRejectScheduleCustomHours,
			"start":    "09:00",
			"end":      "17:00",
			"days":     []int{1, 2, 3},
			"timezone": "UTC",
		},
	})
	require.NoError(t, err)
}

func TestEvaluateAutoRejectCall(t *testing.T) {
	settings := DefaultAutoRejectCallSettings()
	settings.Enabled = true
	settings.Mode = AutoRejectCallModeWithMessage
	settings.Message = "Unavailable"
	settings.Schedule.Type = AutoRejectScheduleCustomHours
	settings.Schedule.Start = "09:00"
	settings.Schedule.End = "17:00"
	settings.Schedule.Days = []int{1}
	settings.Schedule.Timezone = "UTC"
	settings.BypassContacts = []string{"15550000001"}

	mondayMorning := time.Date(2026, time.January, 5, 10, 0, 0, 0, time.UTC) // Monday
	decision := EvaluateAutoRejectCall(settings, mondayMorning, false, 0, "15552223333")
	assert.True(t, decision.ShouldReject)
	assert.Equal(t, "Unavailable", decision.ReplyMessage)

	decision = EvaluateAutoRejectCall(settings, mondayMorning, false, 0, "15550000001")
	assert.False(t, decision.ShouldReject)
	assert.Equal(t, "bypass_contact", decision.Reason)

	night := time.Date(2026, time.January, 5, 20, 0, 0, 0, time.UTC)
	decision = EvaluateAutoRejectCall(settings, night, false, 0, "15552223333")
	assert.False(t, decision.ShouldReject)
	assert.Equal(t, "outside_schedule", decision.Reason)

	busyOnly := DefaultAutoRejectCallSettings()
	busyOnly.Enabled = true
	busyOnly.Schedule.Type = AutoRejectScheduleWhileInOtherCall
	decision = EvaluateAutoRejectCall(busyOnly, mondayMorning, false, 0, "15552223333")
	assert.False(t, decision.ShouldReject)
	decision = EvaluateAutoRejectCall(busyOnly, mondayMorning, false, 1, "15552223333")
	assert.True(t, decision.ShouldReject)
}
