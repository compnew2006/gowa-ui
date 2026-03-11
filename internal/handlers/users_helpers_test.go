package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeNotificationSound(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "empty string returns default",
			raw:      "",
			expected: "notification1",
		},
		{
			name:     "whitespace only returns default",
			raw:      "   ",
			expected: "notification1",
		},
		{
			name:     "exact match notification1",
			raw:      "notification1",
			expected: "notification1",
		},
		{
			name:     "exact match notification2",
			raw:      "notification2",
			expected: "notification2",
		},
		{
			name:     "exact match notification",
			raw:      "notification",
			expected: "notification",
		},
		{
			name:     "uppercase notification1 normalized",
			raw:      "NOTIFICATION1",
			expected: "notification1",
		},
		{
			name:     "uppercase notification2 normalized",
			raw:      "NOTIFICATION2",
			expected: "notification2",
		},
		{
			name:     "uppercase notification normalized",
			raw:      "NOTIFICATION",
			expected: "notification",
		},
		{
			name:     "mixed case normalized",
			raw:      "Notification1",
			expected: "notification1",
		},
		{
			name:     "with leading/trailing spaces",
			raw:      "  notification1  ",
			expected: "notification1",
		},
		{
			name:     "unknown sound returns default",
			raw:      "unknown_sound",
			expected: "notification1",
		},
		{
			name:     "partial match returns default",
			raw:      "notification12",
			expected: "notification1",
		},
		{
			name:     "case insensitive with spaces",
			raw:      "  NOTIFICATION2  ",
			expected: "notification2",
		},
		{
			name:     "extra characters in valid name",
			raw:      "notification2_extra",
			expected: "notification1",
		},
		{
			name:     "notification1 with extra spaces",
			raw:      "notification 1",
			expected: "notification1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeNotificationSound(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPermission(t *testing.T) {
	tests := []struct {
		name     string
		perm     string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			perm:     "",
			expected: nil,
		},
		{
			name:     "no colon returns nil",
			perm:     "nocolon",
			expected: nil,
		},
		{
			name:     "single colon splits",
			perm:     "resource:action",
			expected: []string{"resource", "action"},
		},
		{
			name:     "colon at beginning splits",
			perm:     ":action",
			expected: []string{"", "action"},
		},
		{
			name:     "colon at end splits",
			perm:     "resource:",
			expected: []string{"resource", ""},
		},
		{
			name:     "only colon splits",
			perm:     ":",
			expected: []string{"", ""},
		},
		{
			name:     "multiple colons splits on last",
			perm:     "resource:action:extra",
			expected: []string{"resource:action", "extra"}, // Last colon is after "action"
		},
		{
			name:     "colon in middle with spaces",
			perm:     "contacts : read",
			expected: []string{"contacts ", " read"},
		},
		{
			name:     "resource with hyphens",
			perm:     "api-keys:manage",
			expected: []string{"api-keys", "manage"},
		},
		{
			name:     "resource with underscores",
			perm:     "bulk_messages:create",
			expected: []string{"bulk_messages", "create"},
		},
		{
			name:     "resource with dots",
			perm:     "api.endpoint:update",
			expected: []string{"api.endpoint", "update"},
		},
		{
			name:     "action with special characters",
			perm:     "users:delete-all",
			expected: []string{"users", "delete-all"},
		},
		{
			name:     "long resource name",
			perm:     "very_long_resource_name_with_underscores:action",
			expected: []string{"very_long_resource_name_with_underscores", "action"},
		},
		{
			name:     "unicode characters",
			perm:     "ресурс:действие",
			expected: []string{"ресурс", "действие"},
		},
		{
			name:     "numbers in permission",
			perm:     "api2:read3",
			expected: []string{"api2", "read3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPermission(tt.perm)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserToResponse(t *testing.T) {
	roleID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()
	permID := uuid.New()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		user     models.User
	 validate func(*testing.T, UserResponse)
	}{
		{
			name: "basic user without role",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "test@example.com",
				FullName:       "Test User",
				RoleID:         nil,
				IsActive:       true,
				IsAvailable:    false,
				IsSuperAdmin:   false,
				OrganizationID: orgID,
				Settings:       nil,
				Role:           nil,
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.Equal(t, userID, resp.ID)
				assert.Equal(t, "test@example.com", resp.Email)
				assert.Equal(t, "Test User", resp.FullName)
				assert.Nil(t, resp.RoleID)
				assert.True(t, resp.IsActive)
				assert.False(t, resp.IsAvailable)
				assert.False(t, resp.IsSuperAdmin)
				assert.Equal(t, orgID, resp.OrganizationID)
				assert.Equal(t, now.Format("2006-01-02T15:04:05Z"), resp.CreatedAt)
				assert.Equal(t, now.Format("2006-01-02T15:04:05Z"), resp.UpdatedAt)
				assert.Nil(t, resp.Role)
			},
		},
		{
			name: "user with role but no permissions",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "admin@example.com",
				FullName:       "Admin User",
				RoleID:         &roleID,
				IsActive:       true,
				IsAvailable:    true,
				IsSuperAdmin:   false,
				OrganizationID: orgID,
				Settings:       models.JSONB{"key": "value"},
				Role: &models.CustomRole{
					BaseModel:    models.BaseModel{ID: roleID},
					Name:         "Admin",
					Description:  "Administrator role",
					IsSystem:     true,
					Permissions:  []models.Permission{},
				},
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.NotNil(t, resp.Role)
				assert.Equal(t, roleID, resp.Role.ID)
				assert.Equal(t, "Admin", resp.Role.Name)
				assert.Equal(t, "Administrator role", resp.Role.Description)
				assert.True(t, resp.Role.IsSystem)
				assert.Empty(t, resp.Role.Permissions)
				assert.Equal(t, models.JSONB{"key": "value"}, resp.Settings)
			},
		},
		{
			name: "user with role and permissions",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "user@example.com",
				FullName:       "Regular User",
				RoleID:         &roleID,
				IsActive:       true,
				IsAvailable:    true,
				IsSuperAdmin:   false,
				OrganizationID: orgID,
				Settings:       models.JSONB{},
				Role: &models.CustomRole{
					BaseModel:   models.BaseModel{ID: roleID},
					Name:        "Editor",
					Description: "Can edit content",
					IsSystem:    false,
					Permissions: []models.Permission{
						{
							BaseModel:    models.BaseModel{ID: permID},
							Resource:     "contacts",
							Action:       "read",
							Description:  "Read contacts",
						},
						{
							BaseModel:    models.BaseModel{ID: uuid.New()},
							Resource:     "campaigns",
							Action:       "write",
							Description:  "Write campaigns",
						},
					},
				},
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.NotNil(t, resp.Role)
				assert.Equal(t, "Editor", resp.Role.Name)
				assert.False(t, resp.Role.IsSystem)
				assert.Len(t, resp.Role.Permissions, 2)

				// Check first permission
				assert.Equal(t, permID, resp.Role.Permissions[0].ID)
				assert.Equal(t, "contacts", resp.Role.Permissions[0].Resource)
				assert.Equal(t, "read", resp.Role.Permissions[0].Action)
				assert.Equal(t, "Read contacts", resp.Role.Permissions[0].Description)

				// Check second permission
				assert.Equal(t, "campaigns", resp.Role.Permissions[1].Resource)
				assert.Equal(t, "write", resp.Role.Permissions[1].Action)
				assert.Equal(t, "Write campaigns", resp.Role.Permissions[1].Description)
			},
		},
		{
			name: "super admin user",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "super@example.com",
				FullName:       "Super Admin",
				RoleID:         nil,
				IsActive:       true,
				IsAvailable:    true,
				IsSuperAdmin:   true,
				OrganizationID: orgID,
				Settings:       models.JSONB{},
				Role:           nil,
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.True(t, resp.IsSuperAdmin)
				assert.Nil(t, resp.Role)
				assert.Nil(t, resp.RoleID)
			},
		},
		{
			name: "inactive user",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "inactive@example.com",
				FullName:       "Inactive User",
				RoleID:         nil,
				IsActive:       false,
				IsAvailable:    false,
				IsSuperAdmin:   false,
				OrganizationID: orgID,
				Settings:       nil,
				Role:           nil,
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.False(t, resp.IsActive)
				assert.False(t, resp.IsAvailable)
			},
		},
		{
			name: "user with empty settings",
			user: models.User{
				BaseModel:      models.BaseModel{ID: userID, CreatedAt: now, UpdatedAt: now},
				Email:          "user@example.com",
				FullName:       "User",
				RoleID:         nil,
				IsActive:       true,
				IsAvailable:    true,
				IsSuperAdmin:   false,
				OrganizationID: orgID,
				Settings:       models.JSONB{},
				Role:           nil,
			},
			validate: func(t *testing.T, resp UserResponse) {
				assert.NotNil(t, resp.Settings)
				assert.Empty(t, resp.Settings)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := userToResponse(tt.user)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
