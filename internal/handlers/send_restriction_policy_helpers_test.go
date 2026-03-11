package handlers

import (
	"testing"

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
		name         string
		existing     []string
		additions    []string
		expectedMerged []string
		expectedChanged bool
	}{
		{
			name:         "both empty",
			existing:     []string{},
			additions:    []string{},
			expectedMerged: []string{},
			expectedChanged: false,
		},
		{
			name:         "existing empty, adding one",
			existing:     []string{},
			additions:    []string{"1234567890"},
			expectedMerged: []string{"1234567890"},
			expectedChanged: true,
		},
		{
			name:         "adding to existing, no overlap",
			existing:     []string{"1234567890"},
			additions:    []string{"9876543210"},
			expectedMerged: []string{"1234567890", "9876543210"},
			expectedChanged: true,
		},
		{
			name:         "adding duplicate, no change",
			existing:     []string{"1234567890", "9876543210"},
			additions:    []string{"1234567890"},
			expectedMerged: []string{"1234567890", "9876543210"},
			expectedChanged: false,
		},
		{
			name:         "adding multiple with some duplicates",
			existing:     []string{"1111111111", "2222222222"},
			additions:    []string{"2222222222", "3333333333"},
			expectedMerged: []string{"1111111111", "2222222222", "3333333333"},
			expectedChanged: true,
		},
		{
			name:         "both have duplicates internally",
			existing:     []string{"111", "111", "222"},
			additions:    []string{"222", "333"},
			expectedMerged: []string{"111", "222", "333"},
			expectedChanged: true,
		},
		{
			name:         "all duplicates",
			existing:     []string{"1234567890", "1234567890"},
			additions:    []string{"1234567890", "1234567890"},
			expectedMerged: []string{"1234567890"},
			expectedChanged: false,
		},
		{
			name:         "empty additions, no change",
			existing:     []string{"1234567890", "9876543210"},
			additions:    []string{},
			expectedMerged: []string{"1234567890", "9876543210"},
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
