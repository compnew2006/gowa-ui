package handlers_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestDefaultChatAssignmentResetSettings tests the default settings
func TestDefaultChatAssignmentResetSettings(t *testing.T) {
	t.Parallel()

	result := handlers.DefaultChatAssignmentResetSettings()

	assert.True(t, result.Enabled, "Default should be enabled")
	assert.Equal(t, handlers.ChatAssignmentResetModeMidnight, result.Mode, "Default mode should be midnight")
	assert.Equal(t, 0, result.Hour, "Default hour should be 0")
	assert.Empty(t, result.LastResetDate, "Last reset date should be empty")
}

// TestIsValidChatAssignmentResetMode tests mode validation
func TestIsValidChatAssignmentResetMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid midnight lowercase",
			input:    "midnight",
			expected: true,
		},
		{
			name:     "valid midnight uppercase",
			input:    "MIDNIGHT",
			expected: true,
		},
		{
			name:     "valid midnight mixed case",
			input:    "MidNight",
			expected: true,
		},
		{
			name:     "valid midnight with spaces",
			input:    "  midnight  ",
			expected: true,
		},
		{
			name:     "valid custom_hour lowercase",
			input:    "custom_hour",
			expected: true,
		},
		{
			name:     "valid custom_hour uppercase",
			input:    "CUSTOM_HOUR",
			expected: true,
		},
		{
			name:     "invalid mode",
			input:    "invalid",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
		{
			name:     "similar but invalid",
			input:    "customhour",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.IsValidChatAssignmentResetMode(tt.input)
			assert.Equal(t, tt.expected, result, "isValidChatAssignmentResetMode should return correct result")
		})
	}
}

// TestNormalizeChatAssignmentResetMode tests mode normalization
func TestNormalizeChatAssignmentResetMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected handlers.ChatAssignmentResetMode
	}{
		{
			name:     "custom_hour lowercase",
			input:    "custom_hour",
			expected: handlers.ChatAssignmentResetModeCustomHour,
		},
		{
			name:     "custom_hour uppercase",
			input:    "CUSTOM_HOUR",
			expected: handlers.ChatAssignmentResetModeCustomHour,
		},
		{
			name:     "custom_hour mixed case",
			input:    "CuStom_HoUr",
			expected: handlers.ChatAssignmentResetModeCustomHour,
		},
		{
			name:     "custom_hour with spaces",
			input:    "  custom_hour  ",
			expected: handlers.ChatAssignmentResetModeCustomHour,
		},
		{
			name:     "midnight",
			input:    "midnight",
			expected: handlers.ChatAssignmentResetModeMidnight,
		},
		{
			name:     "MIDNIGHT",
			input:    "MIDNIGHT",
			expected: handlers.ChatAssignmentResetModeMidnight,
		},
		{
			name:     "invalid defaults to midnight",
			input:    "invalid",
			expected: handlers.ChatAssignmentResetModeMidnight,
		},
		{
			name:     "empty defaults to midnight",
			input:    "",
			expected: handlers.ChatAssignmentResetModeMidnight,
		},
		{
			name:     "whitespace defaults to midnight",
			input:    "   ",
			expected: handlers.ChatAssignmentResetModeMidnight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.NormalizeChatAssignmentResetMode(tt.input)
			assert.Equal(t, tt.expected, result, "normalizeChatAssignmentResetMode should return correct mode")
		})
	}
}

// TestIsValidChatAssignmentResetHour tests hour validation
func TestIsValidChatAssignmentResetHour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{
			name:     "valid hour 0",
			input:    0,
			expected: true,
		},
		{
			name:     "valid hour 12",
			input:    12,
			expected: true,
		},
		{
			name:     "valid hour 23",
			input:    23,
			expected: true,
		},
		{
			name:     "invalid hour -1",
			input:    -1,
			expected: false,
		},
		{
			name:     "invalid hour 24",
			input:    24,
			expected: false,
		},
		{
			name:     "invalid hour 100",
			input:    100,
			expected: false,
		},
		{
			name:     "invalid negative hour",
			input:    -10,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.IsValidChatAssignmentResetHour(tt.input)
			assert.Equal(t, tt.expected, result, "isValidChatAssignmentResetHour should return correct result")
		})
	}
}

// TestParseChatAssignmentResetHour tests hour parsing from various types
func TestParseChatAssignmentResetHour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		expected  int
		wantValid bool
	}{
		{
			name:      "valid int",
			input:     15,
			expected:  15,
			wantValid: true,
		},
		{
			name:      "valid int8",
			input:     int8(12),
			expected:  12,
			wantValid: true,
		},
		{
			name:      "valid int16",
			input:     int16(8),
			expected:  8,
			wantValid: true,
		},
		{
			name:      "valid int32",
			input:     int32(20),
			expected:  20,
			wantValid: true,
		},
		{
			name:      "valid int64",
			input:     int64(10),
			expected:  10,
			wantValid: true,
		},
		{
			name:      "valid uint",
			input:     uint(5),
			expected:  5,
			wantValid: true,
		},
		{
			name:      "valid uint8",
			input:     uint8(7),
			expected:  7,
			wantValid: true,
		},
		{
			name:      "valid uint16",
			input:     uint16(14),
			expected:  14,
			wantValid: true,
		},
		{
			name:      "valid uint32",
			input:     uint32(18),
			expected:  18,
			wantValid: true,
		},
		{
			name:      "valid uint64",
			input:     uint64(22),
			expected:  22,
			wantValid: true,
		},
		{
			name:      "valid float64 whole number",
			input:     float64(16),
			expected:  16,
			wantValid: true,
		},
		{
			name:      "valid float32 whole number",
			input:     float32(11),
			expected:  11,
			wantValid: true,
		},
		{
			name:      "valid json number",
			input:     json.Number("9"),
			expected:  9,
			wantValid: true,
		},
		{
			name:      "invalid float64 with decimal",
			input:     float64(16.5),
			expected:  0,
			wantValid: false,
		},
		{
			name:      "invalid float32 with decimal",
			input:     float32(11.7),
			expected:  0,
			wantValid: false,
		},
		{
			name:      "invalid string",
			input:     "15",
			expected:  0,
			wantValid: false,
		},
		{
			name:      "invalid bool",
			input:     true,
			expected:  0,
			wantValid: false,
		},
		{
			name:      "invalid nil",
			input:     nil,
			expected:  0,
			wantValid: false,
		},
		{
			name:      "negative int parses but should be validated separately",
			input:     -5,
			expected:  -5,
			wantValid: true, // Parsing succeeds, range validation is separate
		},
		{
			name:      "overflow float64 max",
			input:     float64(math.MaxFloat64),
			expected:  0,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, valid := handlers.ParseChatAssignmentResetHour(tt.input)
			assert.Equal(t, tt.wantValid, valid, "parseChatAssignmentResetHour validity should match")
			assert.Equal(t, tt.expected, result, "parseChatAssignmentResetHour should return correct value")
		})
	}
}

// TestParseJSONBBool tests boolean parsing from various types
func TestParseJSONBBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		expected  bool
		wantValid bool
	}{
		{
			name:      "true bool",
			input:     true,
			expected:  true,
			wantValid: true,
		},
		{
			name:      "false bool",
			input:     false,
			expected:  false,
			wantValid: true,
		},
		{
			name:      "string true lowercase",
			input:     "true",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string true uppercase",
			input:     "TRUE",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string true mixed case",
			input:     "TrUe",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string true with spaces",
			input:     "  true  ",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string 1",
			input:     "1",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string yes",
			input:     "yes",
			expected:  true,
			wantValid: true,
		},
		{
			name:      "string false",
			input:     "false",
			expected:  false,
			wantValid: true,
		},
		{
			name:      "string FALSE",
			input:     "FALSE",
			expected:  false,
			wantValid: true,
		},
		{
			name:      "string 0",
			input:     "0",
			expected:  false,
			wantValid: true,
		},
		{
			name:      "string no",
			input:     "no",
			expected:  false,
			wantValid: true,
		},
		{
			name:      "invalid string",
			input:     "invalid",
			expected:  false,
			wantValid: false,
		},
		{
			name:      "invalid int",
			input:     15,
			expected:  false,
			wantValid: false,
		},
		{
			name:      "invalid nil",
			input:     nil,
			expected:  false,
			wantValid: false,
		},
		{
			name:      "empty string",
			input:     "",
			expected:  false,
			wantValid: false,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			expected:  false,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, valid := handlers.ParseJSONBBool(tt.input)
			assert.Equal(t, tt.wantValid, valid, "parseJSONBBool validity should match")
			assert.Equal(t, tt.expected, result, "parseJSONBBool should return correct value")
		})
	}
}

// TestParseOrganizationTimezone tests timezone parsing
func TestParseOrganizationTimezone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    models.JSONB
		expected string
	}{
		{
			name:     "nil settings",
			input:    nil,
			expected: "UTC",
		},
		{
			name:     "empty JSONB",
			input:    models.JSONB{},
			expected: "UTC",
		},
		{
			name: "valid timezone",
			input: models.JSONB{
				"timezone": "America/New_York",
			},
			expected: "America/New_York",
		},
		{
			name: "valid timezone with spaces",
			input: models.JSONB{
				"timezone": "  America/Los_Angeles  ",
			},
			expected: "America/Los_Angeles",
		},
		{
			name: "empty timezone string",
			input: models.JSONB{
				"timezone": "",
			},
			expected: "UTC",
		},
		{
			name: "whitespace timezone",
			input: models.JSONB{
				"timezone": "   ",
			},
			expected: "UTC",
		},
		{
			name: "non-string timezone",
			input: models.JSONB{
				"timezone": 123,
			},
			expected: "UTC",
		},
		{
			name: "missing timezone key",
			input: models.JSONB{
				"other_key": "value",
			},
			expected: "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.ParseOrganizationTimezone(tt.input)
			assert.Equal(t, tt.expected, result, "parseOrganizationTimezone should return correct timezone")
		})
	}
}

// TestValidateChatAssignmentResetInputs tests input validation
func TestValidateChatAssignmentResetInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      *string
		hour      *int
		wantError bool
	}{
		{
			name:      "both nil",
			mode:      nil,
			hour:      nil,
			wantError: false,
		},
		{
			name:      "valid mode only",
			mode:      strPtr("midnight"),
			hour:      nil,
			wantError: false,
		},
		{
			name:      "valid hour only",
			mode:      nil,
			hour:      intPtr(12),
			wantError: false,
		},
		{
			name:      "valid both",
			mode:      strPtr("custom_hour"),
			hour:      intPtr(15),
			wantError: false,
		},
		{
			name:      "invalid mode",
			mode:      strPtr("invalid"),
			hour:      nil,
			wantError: true,
		},
		{
			name:      "invalid hour negative",
			mode:      nil,
			hour:      intPtr(-1),
			wantError: true,
		},
		{
			name:      "invalid hour too high",
			mode:      nil,
			hour:      intPtr(24),
			wantError: true,
		},
		{
			name:      "valid mode with invalid hour",
			mode:      strPtr("midnight"),
			hour:      intPtr(25),
			wantError: true,
		},
		{
			name:      "invalid mode with valid hour",
			mode:      strPtr("bad_mode"),
			hour:      intPtr(10),
			wantError: true,
		},
		{
			name:      "empty string mode",
			mode:      strPtr(""),
			hour:      nil,
			wantError: true,
		},
		{
			name:      "mode with spaces valid",
			mode:      strPtr("  midnight  "),
			hour:      nil,
			wantError: false,
		},
		{
			name:      "boundary hour 0",
			mode:      nil,
			hour:      intPtr(0),
			wantError: false,
		},
		{
			name:      "boundary hour 23",
			mode:      nil,
			hour:      intPtr(23),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := handlers.ValidateChatAssignmentResetInputs(tt.mode, tt.hour)
			if tt.wantError {
				assert.Error(t, err, "validateChatAssignmentResetInputs should return error")
			} else {
				assert.NoError(t, err, "validateChatAssignmentResetInputs should succeed")
			}
		})
	}
}

// TestReadChatAssignmentResetSettings tests reading settings from JSONB
func TestReadChatAssignmentResetSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    models.JSONB
		expected handlers.ChatAssignmentResetSettings
	}{
		{
			name:  "nil settings returns defaults",
			input: nil,
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name:  "empty JSONB returns defaults",
			input: models.JSONB{},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name: "full settings",
			input: models.JSONB{
				"assigned_chat_reset_enabled":  true,
				"assigned_chat_reset_mode":     "custom_hour",
				"assigned_chat_reset_hour":     15,
				"assigned_chat_reset_last_date": "2024-01-15",
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled:       true,
				Mode:          handlers.ChatAssignmentResetModeCustomHour,
				Hour:          15,
				LastResetDate: "2024-01-15",
			},
		},
		{
			name: "disabled settings",
			input: models.JSONB{
				"assigned_chat_reset_enabled": false,
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: false,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name: "midnight mode ignores hour",
			input: models.JSONB{
				"assigned_chat_reset_mode": "midnight",
				"assigned_chat_reset_hour": 15,
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0, // Should be 0 for midnight
			},
		},
		{
			name: "invalid hour defaults to 0",
			input: models.JSONB{
				"assigned_chat_reset_hour": 25,
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name: "string bool true",
			input: models.JSONB{
				"assigned_chat_reset_enabled": "true",
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name: "string bool false",
			input: models.JSONB{
				"assigned_chat_reset_enabled": "false",
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: false,
				Mode:    handlers.ChatAssignmentResetModeMidnight,
				Hour:    0,
			},
		},
		{
			name: "mode normalizes to custom_hour",
			input: models.JSONB{
				"assigned_chat_reset_mode": "CUSTOM_HOUR",
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled: true,
				Mode:    handlers.ChatAssignmentResetModeCustomHour,
				Hour:    0,
			},
		},
		{
			name: "last reset date with spaces",
			input: models.JSONB{
				"assigned_chat_reset_last_date": "  2024-01-15  ",
			},
			expected: handlers.ChatAssignmentResetSettings{
				Enabled:       true,
				Mode:          handlers.ChatAssignmentResetModeMidnight,
				Hour:          0,
				LastResetDate: "2024-01-15",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.ReadChatAssignmentResetSettings(tt.input)
			assert.Equal(t, tt.expected.Enabled, result.Enabled, "Enabled should match")
			assert.Equal(t, tt.expected.Mode, result.Mode, "Mode should match")
			assert.Equal(t, tt.expected.Hour, result.Hour, "Hour should match")
			assert.Equal(t, tt.expected.LastResetDate, result.LastResetDate, "LastResetDate should match")
		})
	}
}

// Helper functions for tests
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
