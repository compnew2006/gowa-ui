package handlers

import (
	"testing"

	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParseChatResetSettings(t *testing.T) {
	tests := []struct {
		name string
		in   models.JSONB
		check func(t *testing.T, s chatResetSettings)
	}{
		{
			name: "nil settings → defaults (disabled, 02:00)",
			in:   nil,
			check: func(t *testing.T, s chatResetSettings) {
				assert.False(t, s.Enabled)
				assert.Equal(t, "02:00", s.Time)
				assert.Empty(t, s.Timezone)
			},
		},
		{
			name: "empty settings → defaults",
			in:   models.JSONB{},
			check: func(t *testing.T, s chatResetSettings) {
				assert.False(t, s.Enabled)
				assert.Equal(t, "02:00", s.Time)
			},
		},
		{
			name: "other key present → defaults",
			in:   models.JSONB{"close_rating": map[string]any{"enabled": true}},
			check: func(t *testing.T, s chatResetSettings) {
				assert.False(t, s.Enabled)
			},
		},
		{
			name: "enabled with time and timezone",
			in: models.JSONB{"daily_reset": map[string]any{
				"enabled":  true,
				"time":     "09:30",
				"timezone": "Asia/Dubai",
			}},
			check: func(t *testing.T, s chatResetSettings) {
				assert.True(t, s.Enabled)
				assert.Equal(t, "09:30", s.Time)
				assert.Equal(t, "Asia/Dubai", s.Timezone)
			},
		},
		{
			name: "enabled but no time → defaults to 02:00",
			in: models.JSONB{"daily_reset": map[string]any{
				"enabled": true,
			}},
			check: func(t *testing.T, s chatResetSettings) {
				assert.True(t, s.Enabled)
				assert.Equal(t, "02:00", s.Time)
			},
		},
		{
			name: "whitespace in time/timezone trimmed",
			in: models.JSONB{"daily_reset": map[string]any{
				"enabled":  true,
				"time":     "  08:00  ",
				"timezone": "  America/New_York  ",
			}},
			check: func(t *testing.T, s chatResetSettings) {
				assert.Equal(t, "08:00", s.Time)
				assert.Equal(t, "America/New_York", s.Timezone)
			},
		},
		{
			name: "wrong types ignored",
			in: models.JSONB{"daily_reset": map[string]any{
				"enabled":  "yes",
				"time":     123,
				"timezone": 42,
			}},
			check: func(t *testing.T, s chatResetSettings) {
				assert.False(t, s.Enabled, "non-bool enabled should not flip the flag")
				assert.Equal(t, "02:00", s.Time, "non-string time should keep default")
				assert.Empty(t, s.Timezone, "non-string timezone should stay empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, parseChatResetSettings(tt.in))
		})
	}
}
