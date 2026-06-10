package models_test

import (
	"sync"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestJSONB_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    models.JSONB
		wantJSON string
		wantNil  bool
	}{
		{
			name:    "nil JSONB returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "empty JSONB returns empty object",
			input:    models.JSONB{},
			wantJSON: "{}",
		},
		{
			name: "JSONB with values",
			input: models.JSONB{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			wantJSON: `{"key1":"value1","key2":123,"key3":true}`,
		},
		{
			name: "nested JSONB",
			input: models.JSONB{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			wantJSON: `{"outer":{"inner":"value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			val, err := tt.input.Value()
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, val)
				return
			}

			// Value returns []byte from json.Marshal
			bytes, ok := val.([]byte)
			require.True(t, ok, "expected []byte, got %T", val)
			assert.JSONEq(t, tt.wantJSON, string(bytes))
		})
	}
}

func TestJSONB_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    models.JSONB
		wantErr bool
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty object bytes",
			input: []byte("{}"),
			want:  models.JSONB{},
		},
		{
			name:  "object with values",
			input: []byte(`{"key":"value","num":42}`),
			want: models.JSONB{
				"key": "value",
				"num": float64(42), // JSON numbers decode as float64
			},
		},
		{
			name:    "invalid type",
			input:   "not bytes",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte("not json"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var j models.JSONB
			err := j.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, j)
		})
	}
}

func TestStringArray_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    models.StringArray
		wantJSON string
		wantNil  bool
	}{
		{
			name:    "nil StringArray returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "empty StringArray returns empty array",
			input:    models.StringArray{},
			wantJSON: "[]",
		},
		{
			name:     "StringArray with values",
			input:    models.StringArray{"a", "b", "c"},
			wantJSON: `["a","b","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			val, err := tt.input.Value()
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, val)
				return
			}

			bytes, ok := val.([]byte)
			require.True(t, ok, "expected []byte, got %T", val)
			assert.JSONEq(t, tt.wantJSON, string(bytes))
		})
	}
}

func TestStringArray_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    models.StringArray
		wantErr bool
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty array bytes",
			input: []byte("[]"),
			want:  models.StringArray{},
		},
		{
			name:  "array with values",
			input: []byte(`["one","two","three"]`),
			want:  models.StringArray{"one", "two", "three"},
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var s models.StringArray
			err := s.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestJSONBArray_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    models.JSONBArray
		wantJSON string
		wantNil  bool
	}{
		{
			name:    "nil JSONBArray returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "empty JSONBArray returns empty array",
			input:    models.JSONBArray{},
			wantJSON: "[]",
		},
		{
			name: "JSONBArray with values",
			input: models.JSONBArray{
				map[string]interface{}{"id": "1", "title": "Button 1"},
				map[string]interface{}{"id": "2", "title": "Button 2"},
			},
			wantJSON: `[{"id":"1","title":"Button 1"},{"id":"2","title":"Button 2"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			val, err := tt.input.Value()
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, val)
				return
			}

			bytes, ok := val.([]byte)
			require.True(t, ok, "expected []byte, got %T", val)
			assert.JSONEq(t, tt.wantJSON, string(bytes))
		})
	}
}

func TestJSONBArray_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantLen: 0,
		},
		{
			name:    "empty array bytes",
			input:   []byte("[]"),
			wantLen: 0,
		},
		{
			name:    "array with objects",
			input:   []byte(`[{"id":"1"},{"id":"2"}]`),
			wantLen: 2,
		},
		{
			name:    "invalid type",
			input:   "not bytes",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var j models.JSONBArray
			err := j.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantLen == 0 && tt.input == nil {
				assert.Nil(t, j)
			} else {
				assert.Len(t, j, tt.wantLen)
			}
		})
	}
}

func TestOrganizationSchema_HasConfigRelation(t *testing.T) {
	t.Parallel()

	s, err := schema.Parse(&models.Organization{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	rel, ok := s.Relationships.Relations["Config"]
	require.True(t, ok, "Organization schema should expose the Config relation")
	require.NotNil(t, rel.FieldSchema)

	assert.Equal(t, "OrganizationConfig", rel.FieldSchema.Name)
	assert.Equal(t, "organization_configs", rel.FieldSchema.Table)
	require.NotEmpty(t, rel.References)
	assert.Equal(t, "OrganizationID", rel.References[0].ForeignKey.Name)
}

func TestOrganizationConfigSchema_UsesExpectedColumnNames(t *testing.T) {
	t.Parallel()

	s, err := schema.Parse(&models.OrganizationConfig{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	require.NotNil(t, s.LookUpField("OrganizationID"))
	assert.Equal(t, "organization_id", s.LookUpField("OrganizationID").DBName)
	assert.Equal(t, "worker_count", s.LookUpField("WorkerCount").DBName)
	assert.Equal(t, "max_queue_size", s.LookUpField("MaxQueueSize").DBName)
	assert.Equal(t, "max_whatsapp_instances", s.LookUpField("MaxWhatsAppInstances").DBName)
}

func TestCustomJSONTypes_AutoMigrateOnSQLite(t *testing.T) {
	t.Parallel()

	type jsonHolder struct {
		ID       uint
		Metadata models.JSONB
		Items    models.JSONBArray
		Keywords models.StringArray
	}

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&jsonHolder{}))
}

func TestJSONB_Bool(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			result := tt.settings.Bool(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONB_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings models.JSONB
		key      string
		fallback string
		expected string
	}{
		{
			name:     "nil settings, fallback",
			settings: nil,
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "empty settings, fallback",
			settings: models.JSONB{},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "key not present, fallback",
			settings: models.JSONB{"other_key": "value"},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "string value exists",
			settings: models.JSONB{"test_key": "actual"},
			key:      "test_key",
			fallback: "fallback",
			expected: "actual",
		},
		{
			name:     "string value with spaces is trimmed",
			settings: models.JSONB{"test_key": "  actual  "},
			key:      "test_key",
			fallback: "fallback",
			expected: "actual",
		},
		{
			name:     "string value empty, fallback",
			settings: models.JSONB{"test_key": ""},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "string value spaces only, fallback",
			settings: models.JSONB{"test_key": "   "},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "non-string value, fallback",
			settings: models.JSONB{"test_key": 123},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "nil value, fallback",
			settings: models.JSONB{"test_key": nil},
			key:      "test_key",
			fallback: "fallback",
			expected: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.settings.String(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

