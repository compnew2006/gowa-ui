package testutil

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB        *gorm.DB
	testDBOnce    sync.Once
	testDBInitErr error
)

// SetupTestDB creates a connection to a test PostgreSQL database.
// Requires TEST_DATABASE_URL environment variable to be set.
// If not set, the test will be skipped.
// Migrations are run only once across all tests to avoid conflicts.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database test")
	}

	// Initialize database and run migrations only once
	testDBOnce.Do(func() {
		var err error
		testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			testDBInitErr = fmt.Errorf("failed to connect to test postgres: %w", err)
			return
		}

		// Run migrations once
		if err := runMigrations(testDB); err != nil {
			testDBInitErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		// Clean up any existing data before tests start
		cleanupTables(testDB)
	})

	if testDBInitErr != nil {
		t.Fatalf("failed to initialize test database: %v", testDBInitErr)
	}

	// Return a new session for this test to avoid connection conflicts
	return testDB.Session(&gorm.Session{})
}

// SetupTestDBWithCleanup is like SetupTestDB but allows controlling cleanup behavior.
func SetupTestDBWithCleanup(t *testing.T, cleanup bool) *gorm.DB {
	t.Helper()

	db := SetupTestDB(t)

	if cleanup {
		t.Cleanup(func() {
			// Clean up only the data created by this test
			// Note: In parallel tests, this may affect other tests
			// Consider using unique identifiers instead
		})
	}

	return db
}

// runMigrations mirrors production schema setup so the test DB is faithful to
// prod: AutoMigrate every registered model (single source of truth:
// database.GetMigrationModels) followed by the raw-SQL index/healing
// statements (database.ApplyRawIndexes) — including the partial unique indexes
// AutoMigrate cannot express. No seeding runs here (no default admin /
// permissions / widgets), keeping the test DB data-empty and fast.
//
// Using GetMigrationModels instead of a duplicated inline list prevents drift:
// registering a new model once in GetMigrationModels covers both prod and
// tests, so a uniqueness/dedup index added for a feature is automatically
// exercised by the test suite.
func runMigrations(db *gorm.DB) error {
	for _, m := range database.GetMigrationModels() {
		if err := db.AutoMigrate(m.Model); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", m.Name, err)
		}
	}
	if err := database.ApplyRawIndexes(db); err != nil {
		return fmt.Errorf("failed to apply raw indexes: %w", err)
	}
	return nil
}

// cleanupTables removes all data from tables (for PostgreSQL cleanup).
// Uses TRUNCATE CASCADE to handle foreign key constraints properly.
func cleanupTables(db *gorm.DB) {
	tables := []string{
		// Dashboard tables
		"widgets",
		// Canned responses
		"canned_responses",
		// Bulk message tables
		"bulk_message_recipients",
		"bulk_message_campaigns",
		"notification_rules",
		// WhatsApp tables
		"scheduled_messages",
		"messages",
		"tags",
		"contacts",
		"templates",
		"whatsapp_flows",
		"whatsapp_accounts",
		// Roles and permissions
		"role_permissions",
		"custom_roles",
		"permissions",
		// Core tables
		"team_members",
		"teams",
		"api_keys",
		"sso_providers",
		"webhooks",
		"custom_actions",
		"gowa_instances",
		"user_availability_logs",
		"user_whatsapp_accounts",
		"user_organizations",
		"users",
		"organizations",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}

// TruncateTables truncates all tables (PostgreSQL only, faster than DELETE).
func TruncateTables(db *gorm.DB) {
	tables := []string{
		"widgets",
		"canned_responses",
		"bulk_message_recipients",
		"bulk_message_campaigns",
		"notification_rules",
		"messages",
		"tags",
		"contacts",
		"templates",
		"whatsapp_flows",
		"whatsapp_accounts",
		"role_permissions",
		"custom_roles",
		"permissions",
		"team_members",
		"teams",
		"api_keys",
		"sso_providers",
		"webhooks",
		"custom_actions",
		"gowa_instances",
		"user_availability_logs",
		"user_whatsapp_accounts",
		"user_organizations",
		"users",
		"organizations",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}
