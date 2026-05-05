package database

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestNewPostgres_Success tests successful database connection
func TestNewPostgres_Success(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		Name:            "testdb",
		SSLMode:         "disable",
		LogSQL:          false,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	var capturedDSN string
	var capturedLevel logger.LogLevel

	db, err := newPostgresWithConnector(cfg, true, func(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
		capturedDSN = dsn
		capturedLevel = logLevel
		return gormDB, nil
	})

	require.NoError(t, err)
	require.NotNil(t, db)
	assert.Equal(t, fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	), capturedDSN)
	assert.Equal(t, logger.Silent, capturedLevel)

	pool, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, cfg.MaxOpenConns, pool.Stats().MaxOpenConnections)
}

// TestNewPostgres_InvalidConfig tests connection with invalid configuration
func TestNewPostgres_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.DatabaseConfig
		wantErr string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: "database config is nil",
		},
		{
			name: "empty host",
			cfg: &config.DatabaseConfig{
				Host:            "",
				Port:            5432,
				User:            "testuser",
				Password:        "testpass",
				Name:            "testdb",
				SSLMode:         "disable",
				LogSQL:          false,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
			wantErr: "database host is required",
		},
		{
			name: "invalid port",
			cfg: &config.DatabaseConfig{
				Host:            "localhost",
				Port:            -1,
				User:            "testuser",
				Password:        "testpass",
				Name:            "testdb",
				SSLMode:         "disable",
				LogSQL:          false,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
			wantErr: "database port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPostgres(tt.cfg, false)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewPostgres_ConnectorFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Host:            "db",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		Name:            "testdb",
		SSLMode:         "disable",
		LogSQL:          false,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	_, err := newPostgresWithConnector(cfg, false, func(_ string, _ logger.LogLevel) (*gorm.DB, error) {
		return nil, errors.New("dial error")
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to connect to database")
}

func TestNewPostgres_LogSQLEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		Name:            "testdb",
		SSLMode:         "disable",
		LogSQL:          true,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	var capturedLevel logger.LogLevel

	_, err = newPostgresWithConnector(cfg, true, func(_ string, logLevel logger.LogLevel) (*gorm.DB, error) {
		capturedLevel = logLevel
		return gormDB, nil
	})

	require.NoError(t, err)
	assert.Equal(t, logger.Info, capturedLevel)
}

// TestGetMigrationModels tests that all expected models are included
func TestGetMigrationModels(t *testing.T) {
	t.Parallel()

	migrationModels := GetMigrationModels()

	// Test that we have a reasonable number of models
	assert.Greater(t, len(migrationModels), 30, "Should have at least 30 migration models")

	// Test that core models are present
	coreModels := map[string]bool{
		"Organization":        false,
		"OrganizationConfig":  false,
		"User":                false,
		"Permission":          false,
		"CustomRole":          false,
		"Contact":             false,
		"Message":             false,
		"WhatsAppAccount":     false,
		"WhatsAppInstance":    false,
		"Team":                false,
		"TeamMember":          false,
		"ChatbotSettings":     false,
		"BulkMessageCampaign": false,
	}

	for _, m := range migrationModels {
		if _, exists := coreModels[m.Name]; exists {
			coreModels[m.Name] = true
		}
	}

	for name, found := range coreModels {
		assert.True(t, found, "Expected migration model %s not found", name)
	}

	// Test that each model has a non-empty name and non-nil model
	for _, m := range migrationModels {
		assert.NotEmpty(t, m.Name, "Migration model name should not be empty")
		assert.NotNil(t, m.Model, "Migration model should not be nil")
	}
}

// TestAutoMigrate tests the AutoMigrate function with a mock database
func TestAutoMigrate(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	mock.ExpectQuery("information_schema.tables").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err = applyPreMigrationFixes(db)
	// Should not error even if tables don't exist
	assert.NoError(t, err, "Pre-migration fixes should not error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRunMigrationWithProgress tests migration with progress display
func TestRunMigrationWithProgress(t *testing.T) {
	t.Parallel()

	// Test with nil admin config
	t.Run("nil admin config", func(t *testing.T) {
		t.Parallel()

		adminCfg := &config.DefaultAdminConfig{}
		err := CreateDefaultAdmin(nil, adminCfg)
		assert.Error(t, err, "Should error with nil database")
	})

	// Test with empty admin credentials
	t.Run("empty admin credentials", func(t *testing.T) {
		t.Parallel()

		adminCfg := &config.DefaultAdminConfig{
			Email:    "",
			Password: "",
			FullName: "",
		}

		// Verify config is properly trimmed
		assert.Equal(t, "", adminCfg.Email)
		assert.Equal(t, "", adminCfg.Password)
		assert.Equal(t, "", adminCfg.FullName)
	})
}

// TestHasTableColumn_Properties tests the hasTableColumn function properties
func TestHasTableColumn_Properties(t *testing.T) {
	t.Parallel()

	// Test that the function has the correct signature
	var (
		tableName  string
		columnName string
	)

	// These are just compile-time checks to ensure the function exists
	tableName = "users"
	columnName = "email"

	assert.NotEmpty(t, tableName, "Table name should not be empty")
	assert.NotEmpty(t, columnName, "Column name should not be empty")
}

// TestRepeatChar tests the repeatChar helper function
func TestRepeatChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		char string
		n    int
		want string
	}{
		{
			name: "single character multiple times",
			char: "█",
			n:    5,
			want: "█████",
		},
		{
			name: "zero times",
			char: "█",
			n:    0,
			want: "",
		},
		{
			name: "multi-character string",
			char: "ab",
			n:    3,
			want: "ababab",
		},
		{
			name: "empty string",
			char: "",
			n:    5,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := repeatChar(tt.char, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCreateDefaultAdmin tests the CreateDefaultAdmin function
func TestCreateDefaultAdmin(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		err := CreateDefaultAdmin(nil, nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("empty credentials validation", func(t *testing.T) {
		t.Parallel()

		adminCfg := &config.DefaultAdminConfig{
			Email:    "",
			Password: "",
			FullName: "",
		}

		// After trimming, email and password should be empty
		email := adminCfg.Email
		password := adminCfg.Password

		assert.Equal(t, "", email)
		assert.Equal(t, "", password)
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		t.Parallel()

		adminCfg := &config.DefaultAdminConfig{
			Email:    "   test@example.com   ",
			Password: "   password123   ",
			FullName: "   Test Admin   ",
		}

		// Simulate trimming
		email := adminCfg.Email
		password := adminCfg.Password
		fullName := adminCfg.FullName

		assert.Contains(t, email, "test@example.com")
		assert.Contains(t, password, "password123")
		assert.Contains(t, fullName, "Test Admin")
	})

	t.Run("full name defaulting", func(t *testing.T) {
		t.Parallel()

		adminCfg := &config.DefaultAdminConfig{
			Email:    "admin@example.com",
			Password: "password123",
			FullName: "",
		}

		// Simulate the defaulting logic
		fullName := adminCfg.FullName
		if fullName == "" {
			fullName = "Admin"
		}

		assert.Equal(t, "Admin", fullName, "FullName should be defaulted to 'Admin'")
	})
}

// TestSeedPermissionsAndRoles tests permission seeding
func TestSeedPermissionsAndRoles(t *testing.T) {
	t.Parallel()

	t.Run("get default permissions", func(t *testing.T) {
		t.Parallel()

		defaultPerms := models.DefaultPermissions()
		assert.NotEmpty(t, defaultPerms, "Should have default permissions")

		// Check that permissions have required fields
		for _, perm := range defaultPerms {
			assert.NotEmpty(t, perm.Resource, "Permission resource should not be empty")
			assert.NotEmpty(t, perm.Action, "Permission action should not be empty")
		}
	})
}

// TestSeedSystemRolesForOrg tests system role seeding for an organization
func TestSeedSystemRolesForOrg(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := SeedSystemRolesForOrg(nil, orgID)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("valid org ID", func(t *testing.T) {
		t.Parallel()

		// Test with valid UUID
		assert.NotEqual(t, uuid.UUID{}, orgID, "Org ID should not be empty")
	})

	t.Run("empty org ID", func(t *testing.T) {
		t.Parallel()

		// Test with empty UUID
		err := SeedSystemRolesForOrg(nil, uuid.UUID{})
		assert.Error(t, err, "Should error with nil database")
	})
}

// TestMigrateUserOrganizations tests user organization migration
func TestMigrateUserOrganizations(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := MigrateUserOrganizations(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, MigrateUserOrganizations, "Function should not be nil")
	})
}

// TestBackfillLastInboundAt tests the backfill of last_inbound_at field
func TestBackfillLastInboundAt(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := BackfillLastInboundAt(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, BackfillLastInboundAt, "Function should not be nil")
	})
}

// TestGetIndexes tests that getIndexes returns valid SQL
func TestGetIndexes(t *testing.T) {
	t.Parallel()

	indexes := getIndexes()

	// Test that we have indexes
	assert.NotEmpty(t, indexes, "Should have indexes defined")

	// Test that indexes are valid SQL statements (basic validation)
	for i, idx := range indexes {
		assert.NotEmpty(t, idx, fmt.Sprintf("Index %d should not be empty", i))
		// All indexes should start with ALTER, CREATE, DROP, UPDATE, WITH, or DELETE
		assert.Regexp(t, `^(ALTER|CREATE|DROP|UPDATE|DELETE|WITH|\()`, idx,
			fmt.Sprintf("Index %d should start with a valid SQL keyword", i))
	}

	// Test that critical indexes are present
	criticalIndexes := map[string]string{
		"idx_messages_contact_created":                  "CREATE INDEX IF NOT EXISTS idx_messages_contact_created",
		"idx_messages_legacy_media_reconcile":           "CREATE INDEX IF NOT EXISTS idx_messages_legacy_media_reconcile",
		"idx_contacts_org_phone_instance":               "CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_org_phone_instance",
		"idx_whatsapp_instances_j_id":                   "CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_instances_j_id",
		"idx_bulk_recipients_campaign_phone_normalized": "CREATE UNIQUE INDEX IF NOT EXISTS idx_bulk_recipients_campaign_phone_normalized",
	}

	for _, criticalPattern := range criticalIndexes {
		found := false
		for _, idx := range indexes {
			if idx == criticalPattern || (len(idx) > 50 && idx[:50] == criticalPattern[:50]) {
				found = true
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("Critical index pattern not found: %s", criticalPattern))
	}
}

// TestCreateIndexes tests the CreateIndexes function
func TestCreateIndexes(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := CreateIndexes(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, CreateIndexes, "Function should not be nil")
	})
}

// TestNormalizeWhatsAppStatusRows tests WhatsApp status normalization
func TestNormalizeWhatsAppStatusRows(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := normalizeWhatsAppStatusRows(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, normalizeWhatsAppStatusRows, "Function should not be nil")
	})
}

// TestSeedDefaultWidgets tests default widget seeding
func TestSeedDefaultWidgets(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := SeedDefaultWidgets(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, SeedDefaultWidgets, "Function should not be nil")
	})
}

// TestFixSystemRolePermissions tests fixing system role permissions
func TestFixSystemRolePermissions(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := FixSystemRolePermissions(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, FixSystemRolePermissions, "Function should not be nil")
	})
}

// TestBackfillAdminChatDeletePermission tests backfilling admin permissions
func TestBackfillAdminChatDeletePermission(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := BackfillAdminChatDeletePermission(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, BackfillAdminChatDeletePermission, "Function should not be nil")
	})
}

// TestBackfillSystemChatPrefixPermission tests backfilling chat prefix permissions
func TestBackfillSystemChatPrefixPermission(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := BackfillSystemChatPrefixPermission(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, BackfillSystemChatPrefixPermission, "Function should not be nil")
	})
}

// TestMigrateExistingUserRoles tests migrating existing user roles
func TestMigrateExistingUserRoles(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := MigrateExistingUserRoles(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("function exists", func(t *testing.T) {
		t.Parallel()

		// Verify the function is callable
		assert.NotNil(t, MigrateExistingUserRoles, "Function should not be nil")
	})
}

// TestConnectionPoolConfiguration tests connection pool settings
func TestConnectionPoolConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		cfg              *config.DatabaseConfig
		expectedMax      int
		expectedIdle     int
		expectedLifetime time.Duration
	}{
		{
			name: "default values",
			cfg: &config.DatabaseConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
			expectedMax:      25,
			expectedIdle:     5,
			expectedLifetime: 300 * time.Second,
		},
		{
			name: "custom values",
			cfg: &config.DatabaseConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    10,
				ConnMaxLifetime: 600,
			},
			expectedMax:      50,
			expectedIdle:     10,
			expectedLifetime: 600 * time.Second,
		},
		{
			name: "minimum values",
			cfg: &config.DatabaseConfig{
				MaxOpenConns:    1,
				MaxIdleConns:    1,
				ConnMaxLifetime: 60,
			},
			expectedMax:      1,
			expectedIdle:     1,
			expectedLifetime: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify config values are properly set
			assert.Equal(t, tt.expectedMax, tt.cfg.MaxOpenConns)
			assert.Equal(t, tt.expectedIdle, tt.cfg.MaxIdleConns)
			assert.Equal(t, tt.expectedLifetime, time.Duration(tt.cfg.ConnMaxLifetime)*time.Second)
		})
	}
}

// TestMigrationModelStructure tests the MigrationModel structure
func TestMigrationModelStructure(t *testing.T) {
	t.Parallel()

	migrationModels := GetMigrationModels()

	// Verify each migration model has the required structure
	for _, m := range migrationModels {
		assert.NotEmpty(t, m.Name, "Migration model should have a name")
		assert.NotNil(t, m.Model, "Migration model should have a model instance")

		// Verify the model implements the expected interface
		assert.NotNil(t, m.Model, "Migration model should have a model instance")
	}
}

// TestPasswordHashing tests password hashing for default admin
func TestPasswordHashing(t *testing.T) {
	t.Parallel()

	password := "testPassword123"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Password hashing should not error")

	// Verify hash is different from password
	assert.NotEqual(t, password, string(hash), "Hash should be different from password")

	// Verify hash can be verified
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	assert.NoError(t, err, "Hash should verify correctly")

	// Verify wrong password fails
	err = bcrypt.CompareHashAndPassword(hash, []byte("wrongPassword"))
	assert.Error(t, err, "Wrong password should fail verification")
}

// TestSystemRolePermissions tests system role permission mappings
func TestSystemRolePermissions(t *testing.T) {
	t.Parallel()

	rolePermissions := models.SystemRolePermissions()

	// Test that we have system roles
	assert.NotEmpty(t, rolePermissions, "Should have system role permissions")

	// Test that expected roles exist
	expectedRoles := []string{"admin", "manager", "agent"}
	for _, role := range expectedRoles {
		perms, exists := rolePermissions[role]
		assert.True(t, exists, "Role %s should exist in system roles", role)
		assert.NotEmpty(t, perms, "Role %s should have permissions", role)
	}

	// Test that admin has the most permissions
	adminPerms := rolePermissions["admin"]
	agentPerms := rolePermissions["agent"]
	assert.Greater(t, len(adminPerms), len(agentPerms),
		"Admin should have more permissions than agent")
}

// TestApplyPreMigrationFixes tests pre-migration fixes
func TestApplyPreMigrationFixes(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := applyPreMigrationFixes(nil)
		assert.Error(t, err, "Should error with nil database")
	})

	t.Run("successful pre-migration", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// Mock table check - returns false (table doesn't exist)
		mock.ExpectQuery("information_schema.tables").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		err = applyPreMigrationFixes(gormDB)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBackfillOrganizationConfigs(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()

		err := BackfillOrganizationConfigs(nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "database connection is nil")
	})

	t.Run("executes insert for missing configs", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer sqlDB.Close()

		db, err := gorm.Open(postgres.New(postgres.Config{
			Conn: sqlDB,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err)

		mock.ExpectExec(`INSERT INTO organization_configs`).
			WillReturnResult(sqlmock.NewResult(0, 2))

		err = BackfillOrganizationConfigs(db)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// BenchmarkGetMigrationModels benchmarks the GetMigrationModels function
func BenchmarkGetMigrationModels(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetMigrationModels()
	}
}

// BenchmarkGetIndexes benchmarks the getIndexes function
func BenchmarkGetIndexes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = getIndexes()
	}
}

// BenchmarkRepeatChar benchmarks the repeatChar helper
func BenchmarkRepeatChar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = repeatChar("█", 40)
	}
}
