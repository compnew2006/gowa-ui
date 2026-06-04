package testutil

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
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

// runMigrations runs all model migrations.
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		// Core models
		&models.Organization{},
		&models.OrganizationConfig{},
		&models.Permission{},
		&models.CustomRole{},
		&models.User{},
		&models.UserOrganization{},
		&models.Team{},
		&models.TeamMember{},
		&models.APIKey{},
		&models.SSOProvider{},
		&models.Webhook{},
		&models.CustomAction{},
		&models.UserAvailabilityLog{},
		// WhatsApp models
		&models.WhatsAppAccount{},
		&models.Contact{},
		&models.MediaAsset{},
		&models.Tag{},
		&models.Message{},
		&models.ChatClosureRating{},
		&models.ConversationNote{},
		&models.Template{},
		&models.WhatsAppFlow{},
		&models.WhatsAppInstance{},
		&models.InstanceNotification{},
		// Chatbot models
		&models.ChatbotSettings{},
		&models.KeywordRule{},
		&models.ChatbotFlow{},
		&models.ChatbotFlowStep{},
		&models.ChatbotSession{},
		&models.ChatbotSessionMessage{},
		&models.AIContext{},
		&models.AgentTransfer{},
		&models.ContactCollaborator{},
		// Bulk message models
		&models.BulkMessageCampaign{},
		&models.BulkMessageRecipient{},
		&models.NotificationRule{},
		// Catalog models
		&models.Catalog{},
		&models.CatalogProduct{},
		// Canned responses
		&models.CannedResponse{},
		// Dashboard
		&models.Widget{},
		// Agent selection (customer routing)
		&models.AgentSelectionSettings{},
		&models.AgentSelectionParticipant{},
		&models.AgentSelectionOption{},
		&models.AgentSelectionSession{},
		&models.AgentSelectionAuditEvent{},
		// Facebook models
		&models.FacebookAccount{},
		&models.FacebookComment{},
		&models.FacebookCommentReply{},
		&models.FacebookCommentSettings{},
	)
	// Mirror the production pre-migration fix so tests use the partial unique
	// index and can reproduce the "delete then re-add" scenario.
	if err := db.Exec(`
		DROP INDEX IF EXISTS idx_agent_selection_participant_user;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_selection_participant_user
		ON agent_selection_participants (organization_id, settings_id, user_id)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}
	return nil
}

// cleanupTables removes all data from tables (for PostgreSQL cleanup).
// Uses TRUNCATE CASCADE to handle foreign key constraints properly.
func cleanupTables(db *gorm.DB) {
	tables := []string{
		// Dashboard tables
		"widgets",
		"contact_collaborators",
		// Catalog tables
		"catalog_products",
		"catalogs",
		// Canned responses
		"canned_responses",
		// Bulk message tables
		"bulk_message_recipients",
		"bulk_message_campaigns",
		"notification_rules",
		// Chatbot tables
		"chatbot_session_messages",
		"chatbot_sessions",
		"chatbot_flow_steps",
		"chatbot_flows",
		"keyword_rules",
		"chatbot_settings",
		"ai_contexts",
		"agent_transfers",
		// WhatsApp tables
		"chat_closure_ratings",
		"conversation_notes",
		"media_assets",
		"messages",
		"tags",
		"contacts",
		"templates",
		"instance_notifications",
		"whatsapp_instances",
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
		"user_availability_logs",
		"user_organizations",
		"users",
		"organization_configs",
		"organizations",
		// Agent selection (customer routing)
		"agent_selection_audit_events",
		"agent_selection_sessions",
		"agent_selection_options",
		"agent_selection_participants",
		"agent_selection_settings",
		// Facebook tables
		"facebook_comment_replies",
		"facebook_comments",
		"facebook_comment_settings",
		"facebook_accounts",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}

// TruncateTables truncates all tables (PostgreSQL only, faster than DELETE).
func TruncateTables(db *gorm.DB) {
	tables := []string{
		"widgets",
		"catalog_products",
		"catalogs",
		"canned_responses",
		"bulk_message_recipients",
		"bulk_message_campaigns",
		"notification_rules",
		"chatbot_session_messages",
		"chatbot_sessions",
		"chatbot_flow_steps",
		"chatbot_flows",
		"keyword_rules",
		"chatbot_settings",
		"ai_contexts",
		"agent_transfers",
		"chat_closure_ratings",
		"conversation_notes",
		"media_assets",
		"messages",
		"tags",
		"instance_notifications",
		"whatsapp_instances",
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
		"user_availability_logs",
		"user_organizations",
		"users",
		"organization_configs",
		"organizations",
		// Agent selection (customer routing)
		"agent_selection_audit_events",
		"agent_selection_sessions",
		"agent_selection_options",
		"agent_selection_participants",
		"agent_selection_settings",
		// Facebook tables
		"facebook_comment_replies",
		"facebook_comments",
		"facebook_comment_settings",
		"facebook_accounts",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}
