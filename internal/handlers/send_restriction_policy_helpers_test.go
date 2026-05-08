package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseOptionalUUID(t *testing.T) {
	validUUID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		input    interface{}
		expected *uuid.UUID
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "uuid.UUID type",
			input:    validUUID,
			expected: &validUUID,
		},
		{
			name:     "pointer to uuid.UUID",
			input:    &validUUID,
			expected: &validUUID,
		},
		{
			name:     "nil pointer to uuid.UUID",
			input:    (*uuid.UUID)(nil),
			expected: nil,
		},
		{
			name:     "valid UUID string",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: &validUUID,
		},
		{
			name:     "valid UUID string with spaces",
			input:    "  550e8400-e29b-41d4-a716-446655440000  ",
			expected: &validUUID,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace-only string",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "invalid UUID string",
			input:    "not-a-uuid",
			expected: nil,
		},
		{
			name:     "valid UUID byte slice",
			input:    []byte("550e8400-e29b-41d4-a716-446655440000"),
			expected: &validUUID,
		},
		{
			name:     "byte slice with spaces",
			input:    []byte("  550e8400-e29b-41d4-a716-446655440000  "),
			expected: &validUUID,
		},
		{
			name:     "empty byte slice",
			input:    []byte(""),
			expected: nil,
		},
		{
			name:     "whitespace-only byte slice",
			input:    []byte("   "),
			expected: nil,
		},
		{
			name:     "invalid byte slice",
			input:    []byte("not-a-uuid"),
			expected: nil,
		},
		{
			name:     "integer type",
			input:    123,
			expected: nil,
		},
		{
			name:     "bool type",
			input:    true,
			expected: nil,
		},
		{
			name:     "map type",
			input:    map[string]string{"key": "value"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOptionalUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringifyOptionalUUID(t *testing.T) {
	validUUID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	formatted := validUUID.String()

	tests := []struct {
		name     string
		input    *uuid.UUID
		expected *string
	}{
		{
			name:     "nil UUID",
			input:    nil,
			expected: nil,
		},
		{
			name:     "valid UUID",
			input:    &validUUID,
			expected: &formatted,
		},
		{
			name:     "nil UUID pointer",
			input:    (*uuid.UUID)(nil),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringifyOptionalUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringifyUUIDs(t *testing.T) {
	uuid1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	uuid2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440001")
	uuid3, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440002")

	tests := []struct {
		name     string
		input    []uuid.UUID
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []uuid.UUID{},
			expected: []string{},
		},
		{
			name:     "nil UUID in slice",
			input:    []uuid.UUID{uuid.Nil},
			expected: []string{},
		},
		{
			name:     "single valid UUID",
			input:    []uuid.UUID{uuid1},
			expected: []string{uuid1.String()},
		},
		{
			name:     "multiple valid UUIDs",
			input:    []uuid.UUID{uuid1, uuid2, uuid3},
			expected: []string{uuid1.String(), uuid2.String(), uuid3.String()},
		},
		{
			name:     "mix of nil and valid UUIDs",
			input:    []uuid.UUID{uuid.Nil, uuid1, uuid.Nil, uuid2},
			expected: []string{uuid1.String(), uuid2.String()},
		},
		{
			name:     "all nil UUIDs",
			input:    []uuid.UUID{uuid.Nil, uuid.Nil, uuid.Nil},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringifyUUIDs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseUUIDSlice(t *testing.T) {
	uuid1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	uuid2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440001")
	uuid3, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440002")

	tests := []struct {
		name     string
		input    interface{}
		expected []uuid.UUID
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "[]uuid.UUID type",
			input:    []uuid.UUID{uuid1, uuid2, uuid3},
			expected: []uuid.UUID{uuid1, uuid2, uuid3},
		},
		{
			name:     "empty []uuid.UUID",
			input:    []uuid.UUID{},
			expected: []uuid.UUID{},
		},
		{
			name:     "[]string with valid UUIDs",
			input:    []string{uuid1.String(), uuid2.String(), uuid3.String()},
			expected: []uuid.UUID{uuid1, uuid2, uuid3},
		},
		{
			name:     "[]string with invalid UUIDs",
			input:    []string{"not-a-uuid", "also-not-a-uuid"},
			expected: []uuid.UUID{},
		},
		{
			name:     "[]string with mix of valid and invalid",
			input:    []string{uuid1.String(), "invalid", uuid2.String()},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "models.StringArray with valid UUIDs",
			input:    models.StringArray{uuid1.String(), uuid2.String()},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "empty models.StringArray",
			input:    models.StringArray{},
			expected: []uuid.UUID{},
		},
		{
			name:     "[]interface{} with UUID objects",
			input:    []interface{}{uuid1, uuid2},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "[]interface{} with UUID strings",
			input:    []interface{}{uuid1.String(), uuid2.String()},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "[]interface{} with mixed types",
			input:    []interface{}{uuid1.String(), 123, uuid2.String()},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "[]interface{} with non-UUID types",
			input:    []interface{}{123, "invalid", true},
			expected: []uuid.UUID{},
		},
		{
			name:     "string type",
			input:    "not-a-slice",
			expected: nil,
		},
		{
			name:     "integer type",
			input:    123,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUUIDSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeRestrictedUUIDs(t *testing.T) {
	uuid1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	uuid2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440001")
	uuid3, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440002")

	tests := []struct {
		name     string
		input    []uuid.UUID
		expected []uuid.UUID
	}{
		{
			name:     "empty slice",
			input:    []uuid.UUID{},
			expected: []uuid.UUID{},
		},
		{
			name:     "slice with only nil UUIDs",
			input:    []uuid.UUID{uuid.Nil, uuid.Nil},
			expected: []uuid.UUID{},
		},
		{
			name:     "slice with single valid UUID",
			input:    []uuid.UUID{uuid1},
			expected: []uuid.UUID{uuid1},
		},
		{
			name:     "slice with no duplicates",
			input:    []uuid.UUID{uuid1, uuid2, uuid3},
			expected: []uuid.UUID{uuid1, uuid2, uuid3},
		},
		{
			name:     "slice with duplicates",
			input:    []uuid.UUID{uuid1, uuid2, uuid1, uuid3, uuid2},
			expected: []uuid.UUID{uuid1, uuid2, uuid3},
		},
		{
			name:     "slice with nil UUIDs and duplicates",
			input:    []uuid.UUID{uuid.Nil, uuid1, uuid.Nil, uuid2, uuid1},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name:     "all same UUID",
			input:    []uuid.UUID{uuid1, uuid1, uuid1, uuid1},
			expected: []uuid.UUID{uuid1},
		},
		{
			name:     "preserves order",
			input:    []uuid.UUID{uuid3, uuid1, uuid2},
			expected: []uuid.UUID{uuid3, uuid1, uuid2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRestrictedUUIDs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFirstRestrictedUUID(t *testing.T) {
	uuid1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	uuid2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440001")
	uuid3, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440002")

	tests := []struct {
		name     string
		input    []uuid.UUID
		expected *uuid.UUID
	}{
		{
			name:     "empty slice",
			input:    []uuid.UUID{},
			expected: nil,
		},
		{
			name:     "slice with only nil UUIDs",
			input:    []uuid.UUID{uuid.Nil, uuid.Nil},
			expected: nil,
		},
		{
			name:     "slice with single valid UUID",
			input:    []uuid.UUID{uuid1},
			expected: &uuid1,
		},
		{
			name:     "slice with multiple UUIDs",
			input:    []uuid.UUID{uuid1, uuid2, uuid3},
			expected: &uuid1,
		},
		{
			name:     "nil UUIDs followed by valid UUID",
			input:    []uuid.UUID{uuid.Nil, uuid.Nil, uuid1},
			expected: &uuid1,
		},
		{
			name:     "valid UUID followed by nil UUIDs",
			input:    []uuid.UUID{uuid1, uuid.Nil, uuid.Nil},
			expected: &uuid1,
		},
		{
			name:     "mix of nil and valid UUIDs",
			input:    []uuid.UUID{uuid.Nil, uuid1, uuid.Nil, uuid2},
			expected: &uuid1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstRestrictedUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsRestrictedUUID(t *testing.T) {
	uuid1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	uuid2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440001")
	uuid3, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440002")

	tests := []struct {
		name     string
		values   []uuid.UUID
		needle   uuid.UUID
		expected bool
	}{
		{
			name:     "empty slice, non-nil needle",
			values:   []uuid.UUID{},
			needle:   uuid1,
			expected: false,
		},
		{
			name:     "slice contains the needle",
			values:   []uuid.UUID{uuid1, uuid2, uuid3},
			needle:   uuid2,
			expected: true,
		},
		{
			name:     "slice does not contain the needle",
			values:   []uuid.UUID{uuid1, uuid3},
			needle:   uuid2,
			expected: false,
		},
		{
			name:     "nil needle",
			values:   []uuid.UUID{uuid1, uuid2},
			needle:   uuid.Nil,
			expected: false,
		},
		{
			name:     "nil needle with empty slice",
			values:   []uuid.UUID{},
			needle:   uuid.Nil,
			expected: false,
		},
		{
			name:     "slice with only nil UUIDs",
			values:   []uuid.UUID{uuid.Nil, uuid.Nil},
			needle:   uuid1,
			expected: false,
		},
		{
			name:     "needle at beginning",
			values:   []uuid.UUID{uuid1, uuid2, uuid3},
			needle:   uuid1,
			expected: true,
		},
		{
			name:     "needle at end",
			values:   []uuid.UUID{uuid1, uuid2, uuid3},
			needle:   uuid3,
			expected: true,
		},
		{
			name:     "needle in middle",
			values:   []uuid.UUID{uuid1, uuid2, uuid3},
			needle:   uuid2,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsRestrictedUUID(tt.values, tt.needle)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeRestrictedPhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "already normalized digits",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:     "phone number with spaces",
			input:    "123 456 7890",
			expected: "1234567890",
		},
		{
			name:     "phone number with plus",
			input:    "+1234567890",
			expected: "1234567890",
		},
		{
			name:     "phone number with plus and spaces",
			input:    "+123 456 7890",
			expected: "1234567890",
		},
		{
			name:     "phone number with dashes",
			input:    "123-456-7890",
			expected: "1234567890",
		},
		{
			name:     "phone number with mixed formatting",
			input:    "+123-456 7890",
			expected: "1234567890",
		},
		{
			name:     "phone number with leading/trailing spaces",
			input:    "  1234567890  ",
			expected: "1234567890",
		},
		{
			name:     "phone number with non-digit characters",
			input:    "(123) 456-7890",
			expected: "1234567890",
		},
		{
			name:     "returns only digits",
			input:    "+1 (234) 567-890 x123",
			expected: "1234567890123",
		},
		{
			name:     "handles letters",
			input:    "phone: 123-456-7890",
			expected: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRestrictedPhoneNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsRestrictedNumber(t *testing.T) {
	tests := []struct {
		name     string
		numbers  []string
		target   string
		expected bool
	}{
		{
			name:     "empty numbers slice",
			numbers:  []string{},
			target:   "1234567890",
			expected: false,
		},
		{
			name:     "empty target",
			numbers:  []string{"1234567890"},
			target:   "",
			expected: false,
		},
		{
			name:     "both empty",
			numbers:  []string{},
			target:   "",
			expected: false,
		},
		{
			name:     "number exists in slice",
			numbers:  []string{"1234567890", "9876543210"},
			target:   "1234567890",
			expected: true,
		},
		{
			name:     "number does not exist",
			numbers:  []string{"1234567890", "9876543210"},
			target:   "5555555555",
			expected: false,
		},
		{
			name:     "single element slice, found",
			numbers:  []string{"1234567890"},
			target:   "1234567890",
			expected: true,
		},
		{
			name:     "single element slice, not found",
			numbers:  []string{"1234567890"},
			target:   "9876543210",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsRestrictedNumber(tt.numbers, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeRestrictedNumbers(t *testing.T) {
	tests := []struct {
		name            string
		existing        []string
		additions       []string
		expectedMerged  []string
		expectedChanged bool
	}{
		{
			name:            "both empty",
			existing:        []string{},
			additions:       []string{},
			expectedMerged:  []string{},
			expectedChanged: false,
		},
		{
			name:            "existing empty, adding one",
			existing:        []string{},
			additions:       []string{"1234567890"},
			expectedMerged:  []string{"1234567890"},
			expectedChanged: true,
		},
		{
			name:            "adding to existing, no overlap",
			existing:        []string{"1234567890"},
			additions:       []string{"9876543210"},
			expectedMerged:  []string{"1234567890", "9876543210"},
			expectedChanged: true,
		},
		{
			name:            "adding duplicate, no change",
			existing:        []string{"1234567890", "9876543210"},
			additions:       []string{"1234567890"},
			expectedMerged:  []string{"1234567890", "9876543210"},
			expectedChanged: false,
		},
		{
			name:            "adding multiple with some duplicates",
			existing:        []string{"1111111111", "2222222222"},
			additions:       []string{"2222222222", "3333333333"},
			expectedMerged:  []string{"1111111111", "2222222222", "3333333333"},
			expectedChanged: true,
		},
		{
			name:            "both have duplicates internally",
			existing:        []string{"111", "111", "222"},
			additions:       []string{"222", "333"},
			expectedMerged:  []string{"111", "222", "333"},
			expectedChanged: true,
		},
		{
			name:            "all duplicates",
			existing:        []string{"1234567890", "1234567890"},
			additions:       []string{"1234567890", "1234567890"},
			expectedMerged:  []string{"1234567890"},
			expectedChanged: false,
		},
		{
			name:            "empty additions, no change",
			existing:        []string{"1234567890", "9876543210"},
			additions:       []string{},
			expectedMerged:  []string{"1234567890", "9876543210"},
			expectedChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := mergeRestrictedNumbers(tt.existing, tt.additions)
			assert.Equal(t, tt.expectedMerged, result)
			assert.Equal(t, tt.expectedChanged, changed)
		})
	}
}

func TestParseOrganizationBoolSetting(t *testing.T) {
	tests := []struct {
		name     string
		settings models.JSONB
		key      string
		fallback bool
		expected bool
	}{
		{
			name:     "nil settings, fallback true",
			settings: nil,
			key:      "test_key",
			fallback: true,
			expected: true,
		},
		{
			name:     "nil settings, fallback false",
			settings: nil,
			key:      "test_key",
			fallback: false,
			expected: false,
		},
		{
			name:     "empty settings, fallback true",
			settings: models.JSONB{},
			key:      "test_key",
			fallback: true,
			expected: true,
		},
		{
			name:     "key not present, fallback true",
			settings: models.JSONB{"other_key": true},
			key:      "test_key",
			fallback: true,
			expected: true,
		},
		{
			name:     "key not present, fallback false",
			settings: models.JSONB{"other_key": true},
			key:      "test_key",
			fallback: false,
			expected: false,
		},
		{
			name:     "boolean true value",
			settings: models.JSONB{"test_key": true},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "boolean false value",
			settings: models.JSONB{"test_key": false},
			key:      "test_key",
			fallback: true,
			expected: false,
		},
		{
			name:     "string 'true' case insensitive",
			settings: models.JSONB{"test_key": "TrUe"},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "string '1'",
			settings: models.JSONB{"test_key": "1"},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "string 'yes' case insensitive",
			settings: models.JSONB{"test_key": "YeS"},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "string 'on' case insensitive",
			settings: models.JSONB{"test_key": "On"},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "string 'false' case insensitive",
			settings: models.JSONB{"test_key": "FaLsE"},
			key:      "test_key",
			fallback: true,
			expected: false,
		},
		{
			name:     "string '0'",
			settings: models.JSONB{"test_key": "0"},
			key:      "test_key",
			fallback: true,
			expected: false,
		},
		{
			name:     "string 'no' case insensitive",
			settings: models.JSONB{"test_key": "nO"},
			key:      "test_key",
			fallback: true,
			expected: false,
		},
		{
			name:     "string 'off' case insensitive",
			settings: models.JSONB{"test_key": "oFf"},
			key:      "test_key",
			fallback: true,
			expected: false,
		},
		{
			name:     "invalid string, fallback true",
			settings: models.JSONB{"test_key": "invalid"},
			key:      "test_key",
			fallback: true,
			expected: true,
		},
		{
			name:     "invalid string, fallback false",
			settings: models.JSONB{"test_key": "invalid"},
			key:      "test_key",
			fallback: false,
			expected: false,
		},
		{
			name:     "string with spaces, 'true'",
			settings: models.JSONB{"test_key": "  true  "},
			key:      "test_key",
			fallback: false,
			expected: true,
		},
		{
			name:     "integer value",
			settings: models.JSONB{"test_key": 123},
			key:      "test_key",
			fallback: true,
			expected: true,
		},
		{
			name:     "nil value",
			settings: models.JSONB{"test_key": nil},
			key:      "test_key",
			fallback: false,
			expected: false,
		},
		{
			name:     "object value",
			settings: models.JSONB{"test_key": map[string]string{"key": "value"}},
			key:      "test_key",
			fallback: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOrganizationBoolSetting(tt.settings, tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadSendRestrictionsSettings(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	uuid2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	tests := []struct {
		name     string
		settings models.JSONB
		expected sendRestrictionsSettings
	}{
		{
			name:     "nil settings",
			settings: nil,
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{},
				AllowedInstanceID:      nil,
				AllowedInstanceIDs:     []uuid.UUID{},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: false,
				AllowUnclaimedChatSend: false,
			},
		},
		{
			name:     "nil userSettingSendRestrictions key",
			settings: models.JSONB{"send_restrictions": nil},
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{},
				AllowedInstanceID:      nil,
				AllowedInstanceIDs:     []uuid.UUID{},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: false,
				AllowUnclaimedChatSend: false,
			},
		},
		{
			name:     "empty payload",
			settings: models.JSONB{"send_restrictions": map[string]any{}},
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{},
				AllowedInstanceID:      nil,
				AllowedInstanceIDs:     []uuid.UUID{},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: false,
				AllowUnclaimedChatSend: false,
			},
		},
		{
			name: "all fields populated",
			settings: models.JSONB{
				"send_restrictions": map[string]any{
					"enabled":                   true,
					"include_all_contacts":      true,
					"authorized_numbers":        []interface{}{"1234567890", "9876543210"},
					"allowed_instance_ids":      []interface{}{uuid1.String(), uuid2.String()},
					"prefix_agent_name":         false,
					"allow_unclaimed_chat_view": true,
					"allow_unclaimed_chat_send": true,
				},
			},
			expected: sendRestrictionsSettings{
				Enabled:                true,
				IncludeAllContacts:     true,
				AuthorizedNumbers:      []string{"1234567890", "9876543210"},
				AllowedInstanceID:      &uuid1,
				AllowedInstanceIDs:     []uuid.UUID{uuid1, uuid2},
				PrefixAgentName:        false,
				AllowUnclaimedChatView: true,
				AllowUnclaimedChatSend: true,
			},
		},
		{
			name: "authorized_numbers as string array",
			settings: models.JSONB{
				"send_restrictions": map[string]any{
					"authorized_numbers": []string{"111", "222"},
				},
			},
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{"111", "222"},
				AllowedInstanceID:      nil,
				AllowedInstanceIDs:     []uuid.UUID{},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: false,
				AllowUnclaimedChatSend: false,
			},
		},
		{
			name: "allowed_instance_ids as legacy + new format",
			settings: models.JSONB{
				"send_restrictions": map[string]any{
					"allowed_instance_ids": []interface{}{uuid1.String()},
					"allowed_instance_id":  uuid2.String(),
				},
			},
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{},
				AllowedInstanceID:      &uuid1,
				AllowedInstanceIDs:     []uuid.UUID{uuid1},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: false,
				AllowUnclaimedChatSend: false,
			},
		},
		{
			name: "AllowUnclaimedChatSend overrides AllowUnclaimedChatView",
			settings: models.JSONB{
				"send_restrictions": map[string]any{
					"allow_unclaimed_chat_send": true,
					"allow_unclaimed_chat_view": false,
				},
			},
			expected: sendRestrictionsSettings{
				Enabled:                false,
				IncludeAllContacts:     false,
				AuthorizedNumbers:      []string{},
				AllowedInstanceID:      nil,
				AllowedInstanceIDs:     []uuid.UUID{},
				PrefixAgentName:        true,
				AllowUnclaimedChatView: true,
				AllowUnclaimedChatSend: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readSendRestrictionsSettings(tt.settings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteSendRestrictionsSettings(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	uuid2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	t.Run("roundtrip with readSendRestrictionsSettings", func(t *testing.T) {
		original := models.JSONB{
			"send_restrictions": map[string]any{
				"enabled":                   true,
				"include_all_contacts":      true,
				"authorized_numbers":        []interface{}{"1234567890"},
				"allowed_instance_ids":      []interface{}{uuid1.String(), uuid2.String()},
				"prefix_agent_name":         false,
				"allow_unclaimed_chat_view": true,
				"allow_unclaimed_chat_send": true,
			},
		}
		cfg := readSendRestrictionsSettings(original)
		written := writeSendRestrictionsSettings(nil, cfg)
		readBack := readSendRestrictionsSettings(written)
		assert.Equal(t, cfg, readBack)
	})

	t.Run("nil settings creates new map", func(t *testing.T) {
		cfg := sendRestrictionsSettings{Enabled: true}
		result := writeSendRestrictionsSettings(nil, cfg)
		assert.NotNil(t, result)
		assert.Contains(t, result, userSettingSendRestrictions)
	})

	t.Run("preserves existing unrelated keys", func(t *testing.T) {
		settings := models.JSONB{"some_other_key": "preserved"}
		cfg := sendRestrictionsSettings{Enabled: true}
		result := writeSendRestrictionsSettings(settings, cfg)
		assert.Equal(t, "preserved", result["some_other_key"])
		assert.Contains(t, result, userSettingSendRestrictions)
	})
}

func TestParseOrganizationTimeSetting(t *testing.T) {
	pastTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		settings models.JSONB
		key      string
		expected *time.Time
	}{
		{
			name:     "nil settings",
			settings: nil,
			key:      "ts",
			expected: nil,
		},
		{
			name:     "time.Time value",
			settings: models.JSONB{"ts": pastTime},
			key:      "ts",
			expected: &pastTime,
		},
		{
			name: "RFC3339 string",
			settings: models.JSONB{
				"ts": "2023-01-01T00:00:00Z",
			},
			key:      "ts",
			expected: &pastTime,
		},
		{
			name: "RFC3339Nano string",
			settings: models.JSONB{
				"ts": "2023-01-01T00:00:00.000000000Z",
			},
			key:      "ts",
			expected: &pastTime,
		},
		{
			name: "string layout 2006-01-02 15:04:05",
			settings: models.JSONB{
				"ts": "2023-01-01 00:00:00",
			},
			key:      "ts",
			expected: &pastTime,
		},
		{
			name:     "missing key",
			settings: models.JSONB{"other": "value"},
			key:      "ts",
			expected: nil,
		},
		{
			name:     "invalid string",
			settings: models.JSONB{"ts": "not-a-time"},
			key:      "ts",
			expected: nil,
		},
		{
			name: "[]byte value",
			settings: models.JSONB{
				"ts": []byte("2023-01-01T00:00:00Z"),
			},
			key:      "ts",
			expected: &pastTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOrganizationTimeSetting(tt.settings, tt.key)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.True(t, tt.expected.Equal(*result), "expected %v, got %v", *tt.expected, *result)
			}
		})
	}
}

func TestParseOrganizationStringSetting(t *testing.T) {
	tests := []struct {
		name     string
		settings models.JSONB
		key      string
		fallback string
		expected string
	}{
		{
			name:     "nil settings",
			settings: nil,
			key:      "k",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "string value",
			settings: models.JSONB{"k": "hello"},
			key:      "k",
			fallback: "default",
			expected: "hello",
		},
		{
			name:     "empty string returns fallback",
			settings: models.JSONB{"k": ""},
			key:      "k",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "[]byte value",
			settings: models.JSONB{"k": []byte("world")},
			key:      "k",
			fallback: "default",
			expected: "world",
		},
		{
			name:     "missing key",
			settings: models.JSONB{"other": "val"},
			key:      "k",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "non-string type",
			settings: models.JSONB{"k": 123},
			key:      "k",
			fallback: "default",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOrganizationStringSetting(tt.settings, tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeOutboundMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "mixed returns mixed", input: "mixed", expected: "mixed"},
		{name: "inbound_only returns inbound_only", input: "inbound_only", expected: "inbound_only"},
		{name: "MIXED returns mixed", input: "MIXED", expected: "mixed"},
		{name: "random returns inbound_only", input: "random", expected: "inbound_only"},
		{name: "empty returns inbound_only", input: "", expected: "inbound_only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeOutboundMode(tt.input))
		})
	}
}

func TestNormalizeRolloutMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "audit returns audit", input: "audit", expected: "audit"},
		{name: "enforce returns enforce", input: "enforce", expected: "enforce"},
		{name: "AUDIT returns audit", input: "AUDIT", expected: "audit"},
		{name: "random returns enforce", input: "random", expected: "enforce"},
		{name: "empty returns enforce", input: "", expected: "enforce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRolloutMode(tt.input))
		})
	}
}

func TestAllowedInstanceIDsForRestrictions(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	uuid2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	tests := []struct {
		name     string
		cfg      sendRestrictionsSettings
		expected []uuid.UUID
	}{
		{
			name: "with AllowedInstanceIDs",
			cfg: sendRestrictionsSettings{
				AllowedInstanceIDs: []uuid.UUID{uuid1, uuid2},
			},
			expected: []uuid.UUID{uuid1, uuid2},
		},
		{
			name: "empty AllowedInstanceIDs with AllowedInstanceID fallback",
			cfg: sendRestrictionsSettings{
				AllowedInstanceIDs: []uuid.UUID{},
				AllowedInstanceID:  &uuid1,
			},
			expected: []uuid.UUID{uuid1},
		},
		{
			name: "both empty",
			cfg: sendRestrictionsSettings{
				AllowedInstanceIDs: []uuid.UUID{},
				AllowedInstanceID:  nil,
			},
			expected: []uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, allowedInstanceIDsForRestrictions(tt.cfg))
		})
	}
}

func TestResolveOutgoingInstanceID(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	uuid2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	tests := []struct {
		name     string
		req      OutgoingMessageRequest
		expected *uuid.UUID
	}{
		{
			name: "req.InstanceID takes priority",
			req: OutgoingMessageRequest{
				InstanceID: &uuid1,
				Contact:    &models.Contact{InstanceID: &uuid2},
			},
			expected: &uuid1,
		},
		{
			name: "falls back to Contact.InstanceID",
			req: OutgoingMessageRequest{
				InstanceID: nil,
				Contact:    &models.Contact{InstanceID: &uuid2},
			},
			expected: &uuid2,
		},
		{
			name: "both nil returns nil",
			req: OutgoingMessageRequest{
				InstanceID: nil,
				Contact:    nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveOutgoingInstanceID(tt.req))
		})
	}
}

func TestShouldEnforceStrictPolicy(t *testing.T) {
	pastTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	futureTime := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		settings organizationStrictPolicySettings
		now      time.Time
		expected bool
	}{
		{
			name: "enforce mode always true",
			settings: organizationStrictPolicySettings{
				StrictRolloutMode: organizationStrictRolloutModeEnforce,
			},
			now:      now,
			expected: true,
		},
		{
			name: "audit mode with nil StrictRolloutAfter false",
			settings: organizationStrictPolicySettings{
				StrictRolloutMode:  organizationStrictRolloutModeAudit,
				StrictRolloutAfter: nil,
			},
			now:      now,
			expected: false,
		},
		{
			name: "audit mode with past StrictRolloutAfter true",
			settings: organizationStrictPolicySettings{
				StrictRolloutMode:  organizationStrictRolloutModeAudit,
				StrictRolloutAfter: &pastTime,
			},
			now:      now,
			expected: true,
		},
		{
			name: "audit mode with future StrictRolloutAfter false",
			settings: organizationStrictPolicySettings{
				StrictRolloutMode:  organizationStrictRolloutModeAudit,
				StrictRolloutAfter: &futureTime,
			},
			now:      now,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.settings.shouldEnforceStrictPolicy(tt.now))
		})
	}
}

func TestAsStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "[]string",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "models.StringArray",
			input:    models.StringArray{"x", "y"},
			expected: []string{"x", "y"},
		},
		{
			name:     "[]interface{} with mixed types",
			input:    []interface{}{"hello", 123, "world", true},
			expected: []string{"hello", "world"},
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string not slice",
			input:    "single",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := asStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
