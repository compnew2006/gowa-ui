package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParseClockMinutes(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"09:00", 540, true},
		{"23:59", 1439, true},
		{"00:00", 0, true},
		{"17:30", 1050, true},
		{"24:00", 0, false},
		{"12:60", 0, false},
		{"9am", 0, false},
		{"", 0, false},
		{"12:30:45", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseClockMinutes(tt.in)
		assert.Equal(t, tt.ok, ok, tt.in)
		if ok {
			assert.Equal(t, tt.want, got, tt.in)
		}
	}
}

func TestIsWithinBusinessHours(t *testing.T) {
	// Local = UTC+120min (Egypt standard); windows and instants are chosen
	// so each case exercises one rule.
	base := businessHoursSettings{
		Enabled:      true,
		StartTime:    "09:00",
		EndTime:      "17:00",
		Days:         []int{0, 1, 2, 3, 4}, // Sun..Thu (Egyptian work week)
		UtcOffsetMin: 120,
	}
	// 2026-08-17 is a Monday; 10:00 local = 08:00 UTC.
	mondayLocal10 := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	// 2026-08-17 20:00 local = 18:00 UTC (after close).
	mondayLocal20 := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)
	// 2026-08-21 is a Friday — closed day.
	fridayLocal10 := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		bh   businessHoursSettings
		now  time.Time
		want bool
	}{
		{"inside the window", base, mondayLocal10, true},
		{"after close", base, mondayLocal20, false},
		{"before open (07:00 local)", base, time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC), false},
		{"closed weekday", base, fridayLocal10, false},
		{"no day filter = every day", func() businessHoursSettings { b := base; b.Days = nil; return b }(), fridayLocal10, true},
		{"disabled never in-hours", func() businessHoursSettings { b := base; b.Enabled = false; return b }(), mondayLocal10, false},
		{"overnight window: 20:00 local is inside 18:00→06:00", func() businessHoursSettings {
			b := base
			b.StartTime, b.EndTime = "18:00", "06:00"
			return b
		}(), mondayLocal20, true},
		{"overnight window: 10:00 local is outside", func() businessHoursSettings {
			b := base
			b.StartTime, b.EndTime = "18:00", "06:00"
			return b
		}(), mondayLocal10, false},
		{"invalid clock values fail silent (treated in-hours)", func() businessHoursSettings {
			b := base
			b.StartTime = "bananas"
			return b
		}(), mondayLocal20, true},
		{"start==end fails silent", func() businessHoursSettings {
			b := base
			b.EndTime = b.StartTime
			return b
		}(), mondayLocal10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isWithinBusinessHours(tt.bh, tt.now))
		})
	}
}

func TestBusinessHoursForAccount(t *testing.T) {
	t.Run("absent block disables", func(t *testing.T) {
		bh := businessHoursForAccount(testAccountWithSettings(nil))
		assert.False(t, bh.Enabled)
	})
	t.Run("fields parsed from the settings block", func(t *testing.T) {
		bh := businessHoursForAccount(testAccountWithSettings(map[string]any{
			"business_hours": map[string]any{
				"enabled":        true,
				"start_time":     "09:30",
				"end_time":       "15:45",
				"days":           []any{float64(1), float64(3), float64(9)}, // 9 is out of range → dropped
				"utc_offset_min": float64(180),
				"away_message":   "  نرد بعد الافتتاح  ",
			},
		}))
		assert.True(t, bh.Enabled)
		assert.Equal(t, "09:30", bh.StartTime)
		assert.Equal(t, "15:45", bh.EndTime)
		assert.Equal(t, []int{1, 3}, bh.Days)
		assert.Equal(t, 180, bh.UtcOffsetMin)
		assert.Equal(t, "نرد بعد الافتتاح", bh.AwayMessage)
	})
}

// testAccountWithSettings builds a minimal account carrying only a settings
// map — business-hours parsing never touches other fields.
func testAccountWithSettings(settings map[string]any) *models.WhatsAppAccount {
	return &models.WhatsAppAccount{Settings: settings}
}
