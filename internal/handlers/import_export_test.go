package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetColumnLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   ImportConfig
		col      string
		expected string
	}{
		{
			name:     "contacts phone_number label from export config",
			config:   importConfigs["contacts"],
			col:      "phone_number",
			expected: "Phone Number",
		},
		{
			name:     "contacts profile_name label from export config",
			config:   importConfigs["contacts"],
			col:      "profile_name",
			expected: "Name",
		},
		{
			name:     "contacts tags label from export config",
			config:   importConfigs["contacts"],
			col:      "tags",
			expected: "Tags",
		},
		{
			name:     "contacts whats_app_account label from export config",
			config:   importConfigs["contacts"],
			col:      "whats_app_account",
			expected: "WhatsApp Account",
		},
		{
			name:     "contacts assigned_user_id label from export config",
			config:   importConfigs["contacts"],
			col:      "assigned_user_id",
			expected: "Assigned User ID",
		},
		{
			name:     "tags name label from export config",
			config:   importConfigs["tags"],
			col:      "name",
			expected: "Name",
		},
		{
			name:     "tags color label from export config",
			config:   importConfigs["tags"],
			col:      "color",
			expected: "Color",
		},
		{
			name:     "tags description label from export config",
			config:   importConfigs["tags"],
			col:      "description",
			expected: "Description",
		},
		{
			name:     "unknown column returns column name",
			config:   importConfigs["contacts"],
			col:      "nonexistent",
			expected: "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.config.getColumnLabel(tt.col)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateExportColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested []string
		allowed   []string
		expected  []string
		wantErr   bool
	}{
		{
			name:      "all columns valid",
			requested: []string{"phone_number", "profile_name"},
			allowed:   []string{"phone_number", "profile_name", "tags"},
			expected:  []string{"phone_number", "profile_name"},
			wantErr:   false,
		},
		{
			name:      "empty requested returns empty",
			requested: []string{},
			allowed:   []string{"phone_number"},
			expected:  []string{},
			wantErr:   false,
		},
		{
			name:      "invalid column rejected",
			requested: []string{"phone_number", "evil_column"},
			allowed:   []string{"phone_number", "profile_name"},
			expected:  nil,
			wantErr:   true,
		},
		{
			name:      "single valid column",
			requested: []string{"tags"},
			allowed:   []string{"phone_number", "tags"},
			expected:  []string{"tags"},
			wantErr:   false,
		},
		{
			name:      "all allowed columns requested",
			requested: []string{"a", "b", "c"},
			allowed:   []string{"a", "b", "c"},
			expected:  []string{"a", "b", "c"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := validateExportColumns(tt.requested, tt.allowed)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateRequiredColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		colIndex map[string]int
		required []string
		wantErr  bool
	}{
		{
			name:     "all required columns present",
			colIndex: map[string]int{"phone_number": 0, "profile_name": 1},
			required: []string{"phone_number"},
			wantErr:  false,
		},
		{
			name:     "required column missing",
			colIndex: map[string]int{"profile_name": 1},
			required: []string{"phone_number"},
			wantErr:  true,
		},
		{
			name:     "empty required list",
			colIndex: map[string]int{"phone_number": 0},
			required: []string{},
			wantErr:  false,
		},
		{
			name:     "case insensitive match",
			colIndex: map[string]int{"Phone_Number": 0},
			required: []string{"phone_number"},
			wantErr:  false,
		},
		{
			name:     "underscore space normalization",
			colIndex: map[string]int{"phone number": 0},
			required: []string{"phone_number"},
			wantErr:  false,
		},
		{
			name:     "multiple required all present",
			colIndex: map[string]int{"phone_number": 0, "profile_name": 1},
			required: []string{"phone_number", "profile_name"},
			wantErr:  false,
		},
		{
			name:     "multiple required one missing",
			colIndex: map[string]int{"phone_number": 0},
			required: []string{"phone_number", "profile_name"},
			wantErr:  true,
		},
		{
			name:     "empty col index",
			colIndex: map[string]int{},
			required: []string{"phone_number"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRequiredColumns(tt.colIndex, tt.required)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExportConfig_Structures(t *testing.T) {
	t.Parallel()

	t.Run("contacts export config", func(t *testing.T) {
		cfg := exportConfigs["contacts"]
		assert.NotNil(t, cfg.Model)
		assert.Equal(t, "contacts", cfg.Resource)
		assert.Contains(t, cfg.AllowedColumns, "phone_number")
		assert.Contains(t, cfg.AllowedColumns, "profile_name")
		assert.Contains(t, cfg.AllowedColumns, "tags")
		assert.Contains(t, cfg.DefaultColumns, "phone_number")
		assert.NotNil(t, cfg.ColumnLabels)
		assert.NotNil(t, cfg.ColumnTransform)
		assert.Contains(t, cfg.ColumnTransform, "tags")
		assert.Contains(t, cfg.ColumnTransform, "last_message_at")
		assert.Contains(t, cfg.ColumnTransform, "created_at")
		assert.Contains(t, cfg.ColumnTransform, "updated_at")
		assert.Contains(t, cfg.ColumnTransform, "assigned_user_id")
	})

	t.Run("tags export config", func(t *testing.T) {
		cfg := exportConfigs["tags"]
		assert.NotNil(t, cfg.Model)
		assert.Equal(t, "tags", cfg.Resource)
		assert.Contains(t, cfg.AllowedColumns, "name")
		assert.Contains(t, cfg.AllowedColumns, "color")
		assert.Contains(t, cfg.AllowedColumns, "description")
		assert.Contains(t, cfg.DefaultColumns, "name")
		assert.NotNil(t, cfg.ColumnLabels)
	})

	t.Run("invalid table not in export configs", func(t *testing.T) {
		_, ok := exportConfigs["nonexistent"]
		assert.False(t, ok)
	})
}

func TestImportConfig_Structures(t *testing.T) {
	t.Parallel()

	t.Run("contacts import config", func(t *testing.T) {
		cfg := importConfigs["contacts"]
		assert.NotNil(t, cfg.Model)
		assert.Equal(t, "contacts", cfg.Resource)
		assert.Contains(t, cfg.RequiredColumns, "phone_number")
		assert.Contains(t, cfg.OptionalColumns, "profile_name")
		assert.Contains(t, cfg.OptionalColumns, "tags")
		assert.Equal(t, "phone_number", cfg.UniqueColumn)
		assert.NotNil(t, cfg.ColumnTransform)
	})

	t.Run("tags import config", func(t *testing.T) {
		cfg := importConfigs["tags"]
		assert.NotNil(t, cfg.Model)
		assert.Equal(t, "tags", cfg.Resource)
		assert.Contains(t, cfg.RequiredColumns, "name")
		assert.Contains(t, cfg.OptionalColumns, "color")
		assert.Contains(t, cfg.OptionalColumns, "description")
		assert.Equal(t, "name", cfg.UniqueColumn)
	})

	t.Run("invalid table not in import configs", func(t *testing.T) {
		_, ok := importConfigs["nonexistent"]
		assert.False(t, ok)
	})
}

func TestExportConfig_ColumnTransforms(t *testing.T) {
	t.Parallel()

	t.Run("tags transform nil value", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["tags"]
		result := transform(nil)
		assert.Equal(t, "", result)
	})

	t.Run("tags transform valid JSONBArray", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["tags"]
		tags := models.JSONBArray{"vip", "premium"}
		result := transform(tags)
		assert.Equal(t, "vip,premium", result)
	})

	t.Run("tags transform empty JSONBArray", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["tags"]
		tags := models.JSONBArray{}
		result := transform(tags)
		assert.Equal(t, "", result)
	})

	t.Run("last_message_at transform nil", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["last_message_at"]
		result := transform(nil)
		assert.Equal(t, "", result)
	})

	t.Run("last_message_at transform valid time", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["last_message_at"]
		now := time.Now()
		result := transform(&now)
		assert.Equal(t, now.Format(time.RFC3339), result)
	})

	t.Run("last_message_at transform nil pointer", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["last_message_at"]
		result := transform((*time.Time)(nil))
		assert.Equal(t, "", result)
	})

	t.Run("assigned_user_id transform nil", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["assigned_user_id"]
		result := transform(nil)
		assert.Equal(t, "", result)
	})

	t.Run("assigned_user_id transform valid uuid", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["assigned_user_id"]
		id := uuid.New()
		result := transform(&id)
		assert.Equal(t, id.String(), result)
	})

	t.Run("assigned_user_id transform nil uuid pointer", func(t *testing.T) {
		transform := exportConfigs["contacts"].ColumnTransform["assigned_user_id"]
		result := transform((*uuid.UUID)(nil))
		assert.Equal(t, "", result)
	})
}

func TestImportConfig_ColumnTransforms(t *testing.T) {
	t.Parallel()

	t.Run("phone_number transform normalizes plus prefix", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["phone_number"]
		result, err := transform("+1234567890")
		assert.NoError(t, err)
		assert.Equal(t, "1234567890", result)
	})

	t.Run("phone_number transform trims whitespace", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["phone_number"]
		result, err := transform("  1234567890  ")
		assert.NoError(t, err)
		assert.Equal(t, "1234567890", result)
	})

	t.Run("phone_number transform rejects empty", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["phone_number"]
		_, err := transform("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("phone_number transform rejects only plus", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["phone_number"]
		_, err := transform("+")
		assert.Error(t, err)
	})

	t.Run("tags transform empty string returns nil", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["tags"]
		result, err := transform("")
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("tags transform single tag", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["tags"]
		result, err := transform("vip")
		assert.NoError(t, err)
		tags, ok := result.(models.JSONBArray)
		assert.True(t, ok)
		assert.Equal(t, "vip", tags[0])
	})

	t.Run("tags transform multiple tags", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["tags"]
		result, err := transform("vip, premium, important")
		assert.NoError(t, err)
		tags, ok := result.(models.JSONBArray)
		assert.True(t, ok)
		assert.Len(t, tags, 3)
		assert.Equal(t, "vip", tags[0])
		assert.Equal(t, "premium", tags[1])
		assert.Equal(t, "important", tags[2])
	})

	t.Run("tags transform trims whitespace", func(t *testing.T) {
		transform := importConfigs["contacts"].ColumnTransform["tags"]
		result, err := transform("  vip , premium ")
		assert.NoError(t, err)
		tags, ok := result.(models.JSONBArray)
		assert.True(t, ok)
		assert.Len(t, tags, 2)
		assert.Equal(t, "vip", tags[0])
		assert.Equal(t, "premium", tags[1])
	})
}

func TestExportRequest_Struct(t *testing.T) {
	t.Parallel()

	req := ExportRequest{
		Table:   "contacts",
		Columns: []string{"phone_number", "profile_name"},
		Filters: map[string]string{"search": "john"},
		Format:  "csv",
	}

	assert.Equal(t, "contacts", req.Table)
	assert.Len(t, req.Columns, 2)
	assert.Equal(t, "john", req.Filters["search"])
	assert.Equal(t, "csv", req.Format)
}

func TestImportDataRequest_Struct(t *testing.T) {
	t.Parallel()

	req := ImportDataRequest{
		Table:         "contacts",
		ColumnMapping: map[string]string{"Phone Number": "phone_number"},
		UpdateOnDup:   true,
	}

	assert.Equal(t, "contacts", req.Table)
	assert.Equal(t, "phone_number", req.ColumnMapping["Phone Number"])
	assert.True(t, req.UpdateOnDup)
}
