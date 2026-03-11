package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSnakeToPascal(t *testing.T) {
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
			name:     "single word",
			input:    "hello",
			expected: "Hello",
		},
		{
			name:     "single word uppercase",
			input:    "HELLO",
			expected: "HELLO", // No underscore, stays as-is
		},
		{
			name:     "two words",
			input:    "hello_world",
			expected: "HelloWorld",
		},
		{
			name:     "three words",
			input:    "hello_world_again",
			expected: "HelloWorldAgain",
		},
		{
			name:     "id acronym (lowercase)",
			input:    "user_id",
			expected: "UserID",
		},
		{
			name:     "id acronym (uppercase)",
			input:    "user_ID",
			expected: "UserID",
		},
		{
			name:     "url acronym (lowercase)",
			input:    "image_url",
			expected: "ImageURL",
		},
		{
			name:     "url acronym (uppercase)",
			input:    "image_URL",
			expected: "ImageURL",
		},
		{
			name:     "api acronym (lowercase)",
			input:     "api_key",
			expected:    "APIKey",
		},
		{
			name:     "api acronym (uppercase)",
			input:     "API_KEY",
			expected:    "APIKEY", // API is acronym, KEY stays uppercase as-is
		},
		{
			name:     "uuid acronym (lowercase)",
			input:     "user_uuid",
			expected:    "UserUUID",
		},
		{
			name:     "uuid acronym (uppercase)",
			input:     "user_UUID",
			expected:    "UserUUID",
		},
		{
			name:     "http acronym (lowercase)",
			input:     "http_url",
			expected:    "HTTPURL",
		},
		{
			name:     "sql acronym (lowercase)",
			input:     "sql_query",
			expected:    "SQLQuery",
		},
		{
			name:     "json acronym (lowercase)",
			input:     "json_data",
			expected:    "JSONData",
		},
		{
			name:     "mixed acronyms",
			input:     "user_api_url_id",
			expected:    "UserAPIURLID",
		},
		{
			name:     "multiple underscores",
			input:     "one_two_three_four",
			expected:    "OneTwoThreeFour",
		},
		{
			name:     "leading underscore",
			input:     "_private_field",
			expected:    "PrivateField", // Leading empty part dropped by Join
		},
		{
			name:     "trailing underscore",
			input:     "private_field_",
			expected:     "PrivateField",
		},
		{
			name:     "consecutive underscores",
			input:     "field__name",
			expected:     "FieldName", // Empty string part is dropped
		},
		{
			name:     "all uppercase with underscores",
			input:     "USER_ID",
			expected:     "USERID", // USER not in acronyms, ID is
		},
		{
			name:     "already PascalCase",
			input:     "UserName",
			expected:    "UserName",
		},
		{
			name:     "camelCase input",
			input:     "userName",
			expected:    "UserName",
		},
		{
			name:     "numbers in field",
			input:     "field_1_name",
			expected:    "Field1Name",
		},
		{
			name:     "acronym at end",
			input:     "image_id",
			expected:    "ImageID",
		},
		{
			name:     "multiple acronyms in sequence",
			input:     "api_url_id",
			expected:    "APIURLID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snakeToPascal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatExportValue(t *testing.T) {
	now := time.Now()
	validUUID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		input    interface{}
		colType  interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			colType:  nil,
			expected: "",
		},
		{
			name:     "string value",
			input:    "hello world",
			colType:  nil,
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			colType:  nil,
			expected: "",
		},
		{
			name:     "byte slice",
			input:    []byte("test"),
			colType:  nil,
			expected: "test",
		},
		{
			name:     "empty byte slice",
			input:    []byte(""),
			colType:  nil,
			expected: "",
		},
		{
			name:     "int value",
			input:    42,
			colType:  nil,
			expected: "42",
		},
		{
			name:     "int32 value",
			input:    int32(42),
			colType:  nil,
			expected: "42",
		},
		{
			name:     "int64 value",
			input:    int64(42),
			colType:  nil,
			expected: "42",
		},
		{
			name:     "negative int",
			input:    -42,
			colType:  nil,
			expected: "-42",
		},
		{
			name:     "float32 value",
			input:    float32(3.14),
			colType:  nil,
			expected: "3.140000",
		},
		{
			name:     "float64 value",
			input:    3.14159,
			colType:  nil,
			expected: "3.141590",
		},
		{
			name:     "boolean true",
			input:    true,
			colType:  nil,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			colType:  nil,
			expected: "false",
		},
		{
			name:     "time.Time value",
			input:    now,
			colType:  nil,
			expected: now.Format(time.RFC3339),
		},
		{
			name:     "uuid.UUID value",
			input:    validUUID,
			colType:  nil,
			expected: validUUID.String(),
		},
		{
			name:     "pointer to time.Time, non-nil",
			input:    &now,
			colType:  nil,
			expected: now.Format(time.RFC3339),
		},
		{
			name:     "pointer to time.Time, nil",
			input:    (*time.Time)(nil),
			colType:  nil,
			expected: "",
		},
		{
			name:     "pointer to uuid.UUID, non-nil",
			input:    &validUUID,
			colType:  nil,
			expected: validUUID.String(),
		},
		{
			name:     "pointer to uuid.UUID, nil",
			input:    (*uuid.UUID)(nil),
			colType:  nil,
			expected: "",
		},
		{
			name:     "map value",
			input:    map[string]string{"key": "value"},
			colType:  nil,
			expected: `{"key":"value"}`,
		},
		{
			name:     "slice value",
			input:    []int{1, 2, 3},
			colType:  nil,
			expected: `[1,2,3]`,
		},
		{
			name:     "complex struct",
			input:    struct{Field string}{Field: "test"},
			colType:  nil,
			expected: `{"Field":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatExportValue(tt.input, tt.colType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
