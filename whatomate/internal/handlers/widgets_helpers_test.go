package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWidgetGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "nil map returns empty",
			m:        nil,
			key:      "test",
			expected: "",
		},
		{
			name:     "key not present returns empty",
			m:        map[string]interface{}{"other": "value"},
			key:      "test",
			expected: "",
		},
		{
			name:     "key present with string value returns value",
			m:        map[string]interface{}{"test": "hello"},
			key:      "test",
			expected: "hello",
		},
		{
			name:     "key present with non-string value returns empty",
			m:        map[string]interface{}{"test": 123},
			key:      "test",
			expected: "",
		},
		{
			name:     "key present with bool value returns empty",
			m:        map[string]interface{}{"test": true},
			key:      "test",
			expected: "",
		},
		{
			name:     "key present with nil value returns empty",
			m:        map[string]interface{}{"test": nil},
			key:      "test",
			expected: "",
		},
		{
			name:     "key present with map value returns empty",
			m:        map[string]interface{}{"test": map[string]string{}},
			key:      "test",
			expected: "",
		},
		{
			name:     "key present with slice value returns empty",
			m:        map[string]interface{}{"test": []string{}},
			key:      "test",
			expected: "",
		},
		{
			name:     "empty string value returned",
			m:        map[string]interface{}{"test": ""},
			key:      "test",
			expected: "",
		},
		{
			name:     "multiple keys, correct one returned",
			m:        map[string]interface{}{"a": "1", "b": "2", "c": "3"},
			key:      "b",
			expected: "2",
		},
		{
			name:     "key with special characters",
			m:        map[string]interface{}{"test-key": "value"},
			key:      "test-key",
			expected: "value",
		},
		{
			name:     "empty key lookup in non-empty map",
			m:        map[string]interface{}{"": "empty-key-value"},
			key:      "",
			expected: "empty-key-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := widgetGetString(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "empty slice returns false",
			slice:    []string{},
			item:     "test",
			expected: false,
		},
		{
			name:     "nil slice returns false",
			slice:    nil,
			item:     "test",
			expected: false,
		},
		{
			name:     "item present returns true",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not present returns false",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "single element slice with match",
			slice:    []string{"test"},
			item:     "test",
			expected: true,
		},
		{
			name:     "single element slice without match",
			slice:    []string{"other"},
			item:     "test",
			expected: false,
		},
		{
			name:     "case sensitive match",
			slice:    []string{"Test", "TEST", "test"},
			item:     "Test",
			expected: true,
		},
		{
			name:     "case sensitive no match",
			slice:    []string{"TEST"},
			item:     "test",
			expected: false,
		},
		{
			name:     "empty string item present",
			slice:    []string{"a", "", "c"},
			item:     "",
			expected: true,
		},
		{
			name:     "duplicate items still returns true",
			slice:    []string{"a", "b", "a"},
			item:     "a",
			expected: true,
		},
		{
			name:     "special characters in item",
			slice:    []string{"test@example.com", "other"},
			item:     "test@example.com",
			expected: true,
		},
		{
			name:     "unicode characters",
			slice:    []string{"test", "тест"},
			item:     "тест",
			expected: true,
		},
		{
			name:     "long slice",
			slice:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			item:     "f",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "single word capitalized",
			input:    "test",
			expected: "Test",
		},
		{
			name:     "single uppercase word unchanged",
			input:    "TEST",
			expected: "TEST",
		},
		{
			name:     "snake_case converted to words and capitalized",
			input:    "hello_world",
			expected: "Hello world",
		},
		{
			name:     "multiple underscores",
			input:    "one_two_three_four",
			expected: "One two three four",
		},
		{
			name:     "leading underscore",
			input:    "_private_field",
			expected: " private field", // Space is first char, gets uppercased but stays space
		},
		{
			name:     "trailing underscore",
			input:    "private_field_",
			expected: "Private field ",
		},
		{
			name:     "consecutive underscores",
			input:    "field__name",
			expected: "Field  name",
		},
		{
			name:     "already capitalized",
			input:    "HelloWorld",
			expected: "HelloWorld",
		},
		{
			name:     "numbers preserved",
			input:    "field_123_name",
			expected: "Field 123 name",
		},
		{
			name:     "single underscore",
			input:    "_",
			expected: " ",
		},
		{
			name:     "camelCase preserved",
			input:    "camelCase",
			expected: "CamelCase",
		},
		{
			name:     "mixed case with underscores",
			input:    "my_variable_Name",
			expected: "My variable Name",
		},
		{
			name:     "all caps with underscores",
			input:    "USER_ID",
			expected: "USER ID",
		},
		{
			name:     "special characters preserved",
			input:    "field@name",
			expected: "Field@name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveDataSourceTable(t *testing.T) {
	tests := []struct {
		name              string
		dataSource        string
		expectedTableName string
		expectedDateField string
		expectedOk        bool
	}{
		{
			name:              "messages data source",
			dataSource:        "messages",
			expectedTableName: "messages",
			expectedDateField: "created_at",
			expectedOk:        true,
		},
		{
			name:              "contacts data source",
			dataSource:        "contacts",
			expectedTableName: "contacts",
			expectedDateField: "last_message_at",
			expectedOk:        true,
		},
		{
			name:              "campaigns data source",
			dataSource:        "campaigns",
			expectedTableName: "bulk_message_campaigns",
			expectedDateField: "created_at",
			expectedOk:        true,
		},
		{
			name:              "transfers data source",
			dataSource:        "transfers",
			expectedTableName: "agent_transfers",
			expectedDateField: "transferred_at",
			expectedOk:        true,
		},
		{
			name:              "sessions data source",
			dataSource:        "sessions",
			expectedTableName: "chatbot_sessions",
			expectedDateField: "created_at",
			expectedOk:        true,
		},
		{
			name:              "unknown data source",
			dataSource:        "unknown",
			expectedTableName: "",
			expectedDateField: "",
			expectedOk:        false,
		},
		{
			name:              "empty data source",
			dataSource:        "",
			expectedTableName: "",
			expectedDateField: "",
			expectedOk:        false,
		},
		{
			name:              "case sensitive messages",
			dataSource:        "Messages",
			expectedTableName: "",
			expectedDateField: "",
			expectedOk:        false,
		},
		{
			name:              "partial match not enough",
			dataSource:        "message",
			expectedTableName: "",
			expectedDateField: "",
			expectedOk:        false,
		},
		{
			name:              "extra characters not matched",
			dataSource:        "messages_extra",
			expectedTableName: "",
			expectedDateField: "",
			expectedOk:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableName, dateField, ok := resolveDataSourceTable(tt.dataSource)
			assert.Equal(t, tt.expectedTableName, tableName)
			assert.Equal(t, tt.expectedDateField, dateField)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestBuildFilterSQL(t *testing.T) {
	tests := []struct {
		name          string
		filter        FilterInput
		expectedSQL   string
		expectedValue interface{}
		expectErr     bool
	}{
		{
			name: "equals operator",
			filter: FilterInput{
				Field:    "status",
				Operator: "equals",
				Value:    "active",
			},
			expectedSQL:   "status = ?",
			expectedValue: "active",
		},
		{
			name: "not_equals operator",
			filter: FilterInput{
				Field:    "status",
				Operator: "not_equals",
				Value:    "inactive",
			},
			expectedSQL:   "status != ?",
			expectedValue: "inactive",
		},
		{
			name: "contains operator adds wildcards",
			filter: FilterInput{
				Field:    "name",
				Operator: "contains",
				Value:    "test",
			},
			expectedSQL:   "name ILIKE ?",
			expectedValue: "%test%",
		},
		{
			name: "gt operator",
			filter: FilterInput{
				Field:    "count",
				Operator: "gt",
				Value:    "10",
			},
			expectedSQL:   "count > ?",
			expectedValue: "10",
		},
		{
			name: "lt operator",
			filter: FilterInput{
				Field:    "count",
				Operator: "lt",
				Value:    "5",
			},
			expectedSQL:   "count < ?",
			expectedValue: "5",
		},
		{
			name: "gte operator",
			filter: FilterInput{
				Field:    "created_at",
				Operator: "gte",
				Value:    "2024-01-01",
			},
			expectedSQL:   "created_at >= ?",
			expectedValue: "2024-01-01",
		},
		{
			name: "lte operator",
			filter: FilterInput{
				Field:    "created_at",
				Operator: "lte",
				Value:    "2024-12-31",
			},
			expectedSQL:   "created_at <= ?",
			expectedValue: "2024-12-31",
		},
		{
			name: "unknown operator rejected",
			filter: FilterInput{
				Field:    "status",
				Operator: "unknown",
				Value:    "active",
			},
			expectErr: true,
		},
		{
			name: "invalid field name rejected",
			filter: FilterInput{
				Field:    "field;DROP TABLE--",
				Operator: "equals",
				Value:    "test",
			},
			expectErr: true,
		},
		{
			name: "field with spaces rejected",
			filter: FilterInput{
				Field:    "field name",
				Operator: "equals",
				Value:    "test",
			},
			expectErr: true,
		},
		{
			name: "field with SQL injection attempt rejected",
			filter: FilterInput{
				Field:    "status' OR '1'='1",
				Operator: "equals",
				Value:    "active",
			},
			expectErr: true,
		},
		{
			name: "valid field with underscore",
			filter: FilterInput{
				Field:    "created_at",
				Operator: "equals",
				Value:    "2024-01-01",
			},
			expectedSQL:   "created_at = ?",
			expectedValue: "2024-01-01",
		},
		{
			name: "valid field with numbers",
			filter: FilterInput{
				Field:    "field123",
				Operator: "equals",
				Value:    "test",
			},
			expectedSQL:   "field123 = ?",
			expectedValue: "test",
		},
		{
			name: "empty field name rejected",
			filter: FilterInput{
				Field:    "",
				Operator: "equals",
				Value:    "test",
			},
			expectErr: true,
		},
	}

	columns := map[string]string{
		"status":     "status",
		"name":       "name",
		"count":      "count",
		"created_at": "created_at",
		"field123":   "field123",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, value, err := buildFilterSQL(columns, tt.filter)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}
