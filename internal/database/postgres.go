package database

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type postgresConnector func(dsn string, logLevel logger.LogLevel) (*gorm.DB, error)

func defaultPostgresConnector(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
}

func validateDatabaseConfig(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	return nil
}

func buildPostgresDSN(cfg *config.DatabaseConfig) (string, error) {
	if err := validateDatabaseConfig(cfg); err != nil {
		return "", err
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		Path:     "/" + cfg.Name,
		RawQuery: "sslmode=" + cfg.SSLMode,
	}
	return u.String(), nil
}

// NewPostgres creates a new PostgreSQL connection
func NewPostgres(cfg *config.DatabaseConfig, debug bool) (*gorm.DB, error) {
	return newPostgresWithConnector(cfg, debug, defaultPostgresConnector)
}

func newPostgresWithConnector(cfg *config.DatabaseConfig, debug bool, connector postgresConnector) (*gorm.DB, error) {
	_ = debug
	dsn, err := buildPostgresDSN(cfg)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		connector = defaultPostgresConnector
	}

	logLevel := logger.Silent
	if cfg.LogSQL {
		logLevel = logger.Info
	}

	db, err := connector(dsn, logLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	return db, nil
}

// MigrationModel holds model info for migration progress
type MigrationModel struct {
	Name  string
	Model interface{}
}

// GetMigrationModels returns all models to migrate with their names
func GetMigrationModels() []MigrationModel {
	return []MigrationModel{
		// Core models
		{"Organization", &models.Organization{}},
		{"OrganizationConfig", &models.OrganizationConfig{}},
		{"Permission", &models.Permission{}},
		{"CustomRole", &models.CustomRole{}},
		{"User", &models.User{}},
		{"UserOrganization", &models.UserOrganization{}},
		{"Team", &models.Team{}},
		{"TeamMember", &models.TeamMember{}},
		{"APIKey", &models.APIKey{}},
		{"LicenseRecord", &models.LicenseRecord{}},
		{"LicenseEvent", &models.LicenseEvent{}},
		{"SSOProvider", &models.SSOProvider{}},
		{"Webhook", &models.Webhook{}},
		{"CustomAction", &models.CustomAction{}},
		{"WhatsAppAccount", &models.WhatsAppAccount{}},
		{"WhatsAppInstance", &models.WhatsAppInstance{}},
		{"FacebookAccount", &models.FacebookAccount{}},
		{"FacebookComment", &models.FacebookComment{}},
		{"FacebookCommentReply", &models.FacebookCommentReply{}},
		{"FacebookCommentSettings", &models.FacebookCommentSettings{}},
		{"FacebookPageCommentSettings", &models.FacebookPageCommentSettings{}},
		{"InstanceNotification", &models.InstanceNotification{}},
		{"Contact", &models.Contact{}},
		{"MediaAsset", &models.MediaAsset{}},
		{"ContactUserDeletion", &models.ContactUserDeletion{}},
		{"Tag", &models.Tag{}},
		{"Message", &models.Message{}},
		{"WhatsAppStatus", &models.WhatsAppStatus{}},
		{"ChatClosureRating", &models.ChatClosureRating{}},
		{"Template", &models.Template{}},
		{"WhatsAppFlow", &models.WhatsAppFlow{}},

		// Bulk & Notifications
		{"BulkMessageCampaign", &models.BulkMessageCampaign{}},
		{"BulkMessageRecipient", &models.BulkMessageRecipient{}},
		{"NotificationRule", &models.NotificationRule{}},

		// Chatbot models
		{"ChatbotSettings", &models.ChatbotSettings{}},
		{"KeywordRule", &models.KeywordRule{}},
		{"ChatbotFlow", &models.ChatbotFlow{}},
		{"ChatbotFlowStep", &models.ChatbotFlowStep{}},
		{"ChatbotSession", &models.ChatbotSession{}},
		{"ChatbotSessionMessage", &models.ChatbotSessionMessage{}},
		{"AIContext", &models.AIContext{}},
		{"AgentTransfer", &models.AgentTransfer{}},
		{"AgentSelectionSettings", &models.AgentSelectionSettings{}},
		{"AgentSelectionParticipant", &models.AgentSelectionParticipant{}},
		{"AgentSelectionOption", &models.AgentSelectionOption{}},
		{"AgentSelectionSession", &models.AgentSelectionSession{}},
		{"AgentSelectionAuditEvent", &models.AgentSelectionAuditEvent{}},

		// User tracking
		{"UserAvailabilityLog", &models.UserAvailabilityLog{}},

		// Canned responses
		{"CannedResponse", &models.CannedResponse{}},

		// Catalogs
		{"Catalog", &models.Catalog{}},
		{"CatalogProduct", &models.CatalogProduct{}},

		// Dashboard
		{"Widget", &models.Widget{}},

		// Conversation Notes
		{"ConversationNote", &models.ConversationNote{}},
		{"ContactCollaborator", &models.ContactCollaborator{}},
		{"WhatsAppFilterBatch", &models.WhatsAppFilterBatch{}},
		{"WhatsAppFilterResult", &models.WhatsAppFilterResult{}},

		// Group Directory
		{"GroupDirectory", &models.GroupDirectory{}},

		// Group Join Campaigns
		{"GroupJoinCampaign", &models.GroupJoinCampaign{}},
		{"GroupJoinRecipient", &models.GroupJoinRecipient{}},

		// Message Extraction Campaigns
		{"MessageExtractionCampaign", &models.MessageExtractionCampaign{}},
		{"MessageExtractionResult", &models.MessageExtractionResult{}},

		// Group Extraction Campaigns
		{"GroupExtractionCampaign", &models.GroupExtractionCampaign{}},
		{"GroupExtractionResult", &models.GroupExtractionResult{}},

		// Member Extraction Campaigns
		{"MemberExtractionCampaign", &models.MemberExtractionCampaign{}},
		{"MemberExtractionResult", &models.MemberExtractionResult{}},

		// Content Library
		{"SavedContent", &models.SavedContent{}},
	}
}

// AutoMigrate runs auto migration for all models (silent mode)
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := applyPreMigrationFixes(db); err != nil {
		return err
	}

	migrationModels := GetMigrationModels()
	for _, m := range migrationModels {
		if err := db.AutoMigrate(m.Model); err != nil {
			return err
		}
	}
	if err := BackfillOrganizationConfigs(db); err != nil {
		return err
	}
	return nil
}

// RunMigrationWithProgress runs migrations with a progress bar display
func RunMigrationWithProgress(db *gorm.DB, adminCfg *config.DefaultAdminConfig) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Silence GORM logging during migration
	silentDB := db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})

	if err := applyPreMigrationFixes(silentDB); err != nil {
		return fmt.Errorf("failed pre-migration fixes: %w", err)
	}

	migrationModels := GetMigrationModels()
	indexes := getIndexes()

	// Total steps: models + indexes + default admin check
	totalSteps := len(migrationModels) + len(indexes) + 1
	currentStep := 0
	barWidth := 40

	printProgress := func(step int, total int) {
		percent := float64(step) / float64(total)
		filled := int(percent * float64(barWidth))
		empty := barWidth - filled

		bar := repeatChar("█", filled) + "\033[90m" + repeatChar("░", empty) + "\033[0m"
		fmt.Printf("\r  Running migrations  %s %3d%%", bar, int(percent*100))
		_ = os.Stdout.Sync()
	}

	fmt.Println()

	// Migrate models
	for _, m := range migrationModels {
		printProgress(currentStep, totalSteps)
		if err := silentDB.AutoMigrate(m.Model); err != nil {
			fmt.Printf("\n  \033[31m✗ Migration failed: %s\033[0m\n\n", m.Name)
			return fmt.Errorf("failed to migrate %s: %w", m.Name, err)
		}
		currentStep++
	}

	// Create indexes
	for _, idx := range indexes {
		printProgress(currentStep, totalSteps)
		if err := silentDB.Exec(idx).Error; err != nil {
			fmt.Printf("\n  \033[31m✗ Index creation failed\033[0m\n\n")
			return fmt.Errorf("failed to create index: %w", err)
		}
		currentStep++
	}

	// Seed permissions (always run, will skip if already seeded)
	printProgress(currentStep, totalSteps)
	if err := SeedPermissionsAndRoles(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to seed permissions\033[0m\n\n")
		return err
	}

	// Fix existing organizations - link permissions to system roles if missing
	if err := SeedSystemRolesForAllOrgs(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to fix existing role permissions\033[0m\n\n")
		return err
	}

	// Backfill user_organizations from existing users
	if err := MigrateUserOrganizations(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to backfill user organizations\033[0m\n\n")
		return err
	}

	// Create default admin (only runs if no users exist)
	printProgress(currentStep, totalSteps)
	if err := CreateDefaultAdmin(silentDB, adminCfg); err != nil {
		fmt.Printf("\n  \033[31m✗ Setup failed\033[0m\n\n")
		return err
	}
	currentStep++

	if err := BackfillOrganizationConfigs(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to backfill organization configs\033[0m\n\n")
		return err
	}

	// Seed default widgets for all organizations
	printProgress(currentStep, totalSteps)
	if err := SeedDefaultWidgets(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to seed widgets\033[0m\n\n")
		return err
	}

	// Backfill last_inbound_at from existing messages
	if err := BackfillLastInboundAt(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to backfill last_inbound_at\033[0m\n\n")
		return err
	}

	if err := BackfillInstanceAssignedChatResetSettings(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to backfill per-instance assigned chat reset settings\033[0m\n\n")
		return err
	}

	printProgress(currentStep, totalSteps)
	fmt.Printf("\n  \033[32m✓ Migration completed\033[0m\n\n")

	return nil
}

// repeatChar repeats a character n times
func repeatChar(char string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += char
	}
	return result
}

func applyPreMigrationFixes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := normalizeWhatsAppStatusRows(db); err != nil {
		return fmt.Errorf("failed to normalize whatsapp statuses: %w", err)
	}
	if err := fixFacebookCommentsAdminReplyColumn(db); err != nil {
		return fmt.Errorf("failed to fix facebook comments admin reply column: %w", err)
	}
	if err := fixSavedContentsUniqueIndex(db); err != nil {
		return fmt.Errorf("failed to fix saved contents unique index: %w", err)
	}
	if err := fixAgentSelectionParticipantUniqueIndex(db); err != nil {
		return fmt.Errorf("failed to fix agent selection participant unique index: %w", err)
	}
	return nil
}

func fixSavedContentsUniqueIndex(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.SavedContent{}) {
		return nil
	}
	// Drop existing single-column unique indexes on the saved_contents table.
	indexRows, err := db.Raw(`SELECT indexname FROM pg_indexes WHERE tablename = 'saved_contents' AND indexname LIKE '%saved_content%'`).Rows()
	if err != nil {
		return err
	}
	indexNames := make([]string, 0)
	for indexRows.Next() {
		var idxName string
		if err := indexRows.Scan(&idxName); err != nil {
			indexRows.Close()
			return err
		}
		indexNames = append(indexNames, idxName)
	}
	indexRows.Close()

	for _, idxName := range indexNames {
		_ = db.Exec("DROP INDEX IF EXISTS " + idxName).Error
	}
	_ = db.Exec("ALTER TABLE saved_contents DROP CONSTRAINT IF EXISTS idx_saved_contents_org_name").Error

	if err := db.Exec("CREATE UNIQUE INDEX idx_saved_contents_org_name ON saved_contents(organization_id, name) WHERE deleted_at IS NULL").Error; err != nil {
		return err
	}
	return nil
}

// fixAgentSelectionParticipantUniqueIndex replaces any full (non-partial)
// unique index on (organization_id, settings_id, user_id) with a partial
// unique index that ignores soft-deleted rows. Without this, deleting a
// participant and re-adding the same agent to the same settings row fails
// with 23505 "duplicate key" because the soft-deleted row still satisfies
// the index.
//
// Looks for the index by definition (matching column set + uniqueness +
// no WHERE clause) rather than by name, because GORM's auto-migration can
// generate different index names depending on the schema path.
func fixAgentSelectionParticipantUniqueIndex(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.AgentSelectionParticipant{}) {
		return nil
	}

	// Find any non-partial unique index that covers the three columns.
	// pg_indexes.indexdef includes the WHERE clause for partial indexes, so
	// the absence of "WHERE" + the column set is a reliable fingerprint.
	rows, err := db.Raw(`
		SELECT indexname, indexdef FROM pg_indexes
		WHERE tablename = 'agent_selection_participants'
		  AND indexdef LIKE '%UNIQUE%'
		  AND indexdef ILIKE '%organization_id%'
		  AND indexdef ILIKE '%settings_id%'
		  AND indexdef ILIKE '%user_id%'
	`).Rows()
	if err != nil {
		return err
	}
	type indexInfo struct {
		name string
		def  string
	}
	var toDrop []indexInfo
	for rows.Next() {
		var info indexInfo
		if err := rows.Scan(&info.name, &info.def); err != nil {
			rows.Close()
			return err
		}
		// Skip if already partial (has WHERE clause in the indexdef).
		if strings.Contains(strings.ToUpper(info.def), " WHERE ") {
			continue
		}
		toDrop = append(toDrop, info)
	}
	rows.Close()

	for _, idx := range toDrop {
		if err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", pgQuoteIdent(idx.name))).Error; err != nil {
			return fmt.Errorf("failed to drop non-partial index %s: %w", idx.name, err)
		}
	}

	// Ensure the canonical partial index exists. CREATE UNIQUE INDEX IF NOT
	// EXISTS is a no-op when the index is already there, so this is safe to
	// run on every startup.
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_selection_participant_user
		ON agent_selection_participants (organization_id, settings_id, user_id)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Log a single line so the operator can confirm the fix is in place.
	// If dropped indexes were found, the user previously hit the
	// "Agent is already in this routing list" 23505 regression.
	if len(toDrop) > 0 {
		fmt.Printf("[migrate] agent_selection_participants: replaced %d non-partial unique index(es) with partial unique index on (organization_id, settings_id, user_id) WHERE deleted_at IS NULL\n", len(toDrop))
	}
	return nil
}

func fixFacebookCommentsAdminReplyColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.FacebookComment{}) {
		return nil
	}
	hasColumn, err := hasTableColumn(db, "facebook_comments", "is_admin_reply")
	if err != nil {
		return err
	}
	if !hasColumn {
		if err := db.Exec(`
			ALTER TABLE facebook_comments
			ADD COLUMN is_admin_reply boolean NOT NULL DEFAULT false
		`).Error; err != nil {
			return err
		}
	}
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_facebook_comments_is_admin_reply
		ON facebook_comments (is_admin_reply)
	`).Error; err != nil {
		return err
	}
	return nil
}

// pgQuoteIdent quotes a Postgres identifier (index/table/column name) so
// it can be safely interpolated into a DDL statement. Doubles any embedded
// double-quotes per the SQL standard.
func pgQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func normalizeWhatsAppStatusRows(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if !db.Migrator().HasTable(&models.WhatsAppStatus{}) {
		return nil
	}

	if err := db.Exec(`
		UPDATE whatsapp_statuses AS ws
		SET organization_id = wi.organization_id
		FROM whatsapp_instances AS wi
		WHERE ws.organization_id IS NULL
		  AND ws.instance_id = wi.id
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		DELETE FROM whatsapp_statuses
		WHERE organization_id IS NULL OR instance_id IS NULL
	`).Error; err != nil {
		return err
	}

	senderColumn := ""
	if hasSenderJID, err := hasTableColumn(db, "whatsapp_statuses", "sender_jid"); err != nil {
		return err
	} else if hasSenderJID {
		senderColumn = "sender_jid"
	}
	if senderColumn == "" {
		if hasSenderJIDLegacy, err := hasTableColumn(db, "whatsapp_statuses", "sender_j_id"); err != nil {
			return err
		} else if hasSenderJIDLegacy {
			senderColumn = "sender_j_id"
		}
	}

	if senderColumn != "" {
		query := fmt.Sprintf(`
			UPDATE whatsapp_statuses
			SET %s = 'unknown@s.whatsapp.net'
			WHERE %s IS NULL OR btrim(%s) = ''
		`, senderColumn, senderColumn, senderColumn)
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}

	if hasStatusType, err := hasTableColumn(db, "whatsapp_statuses", "status_type"); err != nil {
		return err
	} else if hasStatusType {
		if err := db.Exec(`
			UPDATE whatsapp_statuses
			SET status_type = 'text'
			WHERE status_type IS NULL OR btrim(status_type) = ''
		`).Error; err != nil {
			return err
		}
	}

	if hasExpiresAt, err := hasTableColumn(db, "whatsapp_statuses", "expires_at"); err != nil {
		return err
	} else if hasExpiresAt {
		if err := db.Exec(`
			UPDATE whatsapp_statuses
			SET expires_at = COALESCE(created_at + interval '24 hours', NOW() + interval '24 hours')
			WHERE expires_at IS NULL
		`).Error; err != nil {
			return err
		}
	}

	if hasMetadata, err := hasTableColumn(db, "whatsapp_statuses", "metadata"); err != nil {
		return err
	} else if hasMetadata {
		if err := db.Exec(`
			UPDATE whatsapp_statuses
			SET metadata = '{}'::jsonb
			WHERE metadata IS NULL
		`).Error; err != nil {
			return err
		}
	}

	if hasWhatsAppAccount, err := hasTableColumn(db, "whatsapp_statuses", "whats_app_account"); err != nil {
		return err
	} else if hasWhatsAppAccount {
		if err := db.Exec(`
			UPDATE whatsapp_statuses
			SET whats_app_account = ''
			WHERE whats_app_account IS NULL
		`).Error; err != nil {
			return err
		}
	}

	return nil
}

func hasTableColumn(db *gorm.DB, tableName, columnName string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("database connection is nil")
	}
	var exists bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = ?
			  AND column_name = ?
		)
	`, tableName, columnName).Scan(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

// getIndexes returns all index creation SQL statements
func getIndexes() []string {
	return []string{
		// Expand phone_number columns to support group JIDs (e.g., 120363422675615917@g.us)
		`ALTER TABLE contacts ALTER COLUMN phone_number TYPE varchar(50)`,
		`ALTER TABLE contacts ADD COLUMN IF NOT EXISTS status varchar(20) DEFAULT 'pending'`,
		`ALTER TABLE contacts ADD COLUMN IF NOT EXISTS closed_at timestamptz`,
		`ALTER TABLE contacts ADD COLUMN IF NOT EXISTS closed_by_user_id uuid`,
		`ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS send_blocked_until timestamptz`,
		`ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS send_block_reason text`,
		`ALTER TABLE whatsapp_instances ALTER COLUMN send_block_reason SET DEFAULT ''`,
		`UPDATE whatsapp_instances SET send_block_reason = '' WHERE send_block_reason IS NULL`,
		`ALTER TABLE whatsapp_instances ALTER COLUMN send_block_reason SET NOT NULL`,
		`UPDATE contacts SET status = 'pending' WHERE status IS NULL OR status = ''`,
		`ALTER TABLE chatbot_sessions ALTER COLUMN phone_number TYPE varchar(50)`,
		`ALTER TABLE agent_transfers ALTER COLUMN phone_number TYPE varchar(50)`,
		`ALTER TABLE bulk_message_recipients ALTER COLUMN phone_number TYPE varchar(50)`,
		`ALTER TABLE bulk_message_recipients ADD COLUMN IF NOT EXISTS phone_normalized varchar(32)`,
		`WITH normalized AS (
			SELECT id,
				campaign_id,
				COALESCE(NULLIF(regexp_replace(COALESCE(phone_number, ''), '[^0-9]', '', 'g'), ''), '') AS normalized_phone,
				ROW_NUMBER() OVER (
					PARTITION BY campaign_id, COALESCE(NULLIF(regexp_replace(COALESCE(phone_number, ''), '[^0-9]', '', 'g'), ''), '')
					ORDER BY created_at ASC, id ASC
				) AS row_num
			FROM bulk_message_recipients
		)
		UPDATE bulk_message_recipients AS r
		SET phone_normalized = CASE
			WHEN normalized.row_num = 1 THEN normalized.normalized_phone
			ELSE ''
		END
		FROM normalized
		WHERE normalized.id = r.id`,
		`UPDATE bulk_message_recipients SET phone_normalized = '' WHERE phone_normalized IS NULL`,
		`ALTER TABLE bulk_message_campaigns ADD COLUMN IF NOT EXISTS min_delay_seconds integer DEFAULT 0`,
		`ALTER TABLE bulk_message_campaigns ADD COLUMN IF NOT EXISTS max_delay_seconds integer DEFAULT 0`,
		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_messages_contact_created ON messages(contact_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_legacy_media_reconcile ON messages(created_at ASC) WHERE media_asset_id IS NULL AND media_deleted_at IS NULL AND COALESCE(BTRIM(media_url), '') <> ''`,
		`DROP INDEX IF EXISTS idx_contacts_org_phone`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_org_phone_instance ON contacts(organization_id, phone_number, instance_id) WHERE instance_id IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_org_phone_no_instance ON contacts(organization_id, phone_number) WHERE instance_id IS NULL`,
		// Allow multiple unpaired instances (jid='') while preserving uniqueness for real JIDs.
		`DROP INDEX IF EXISTS idx_whatsapp_instances_jid`,
		`DROP INDEX IF EXISTS idx_whatsapp_instances_j_id`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_instances_j_id ON whatsapp_instances(jid) WHERE jid <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_send_blocked_until ON whatsapp_instances(send_blocked_until) WHERE send_blocked_until IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_bulk_recipients_campaign_phone_normalized ON bulk_message_recipients(campaign_id, phone_normalized) WHERE phone_normalized <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_assigned_read ON contacts(assigned_user_id, is_read)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_status_assignee ON contacts(status, assigned_user_id, last_message_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_closed_at ON contacts(closed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_phone_status ON chatbot_sessions(organization_id, phone_number, status)`,
		`CREATE INDEX IF NOT EXISTS idx_keyword_rules_priority ON keyword_rules(organization_id, is_enabled, priority DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_transfers_active ON agent_transfers(organization_id, phone_number, status)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_transfers_org_contact ON agent_transfers(organization_id, contact_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_transfers_agent_active ON agent_transfers(agent_id, status) WHERE status = 'active'`,
		`CREATE INDEX IF NOT EXISTS idx_agent_transfers_team ON agent_transfers(team_id, status) WHERE team_id IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_accounts_org_phone ON whatsapp_accounts(organization_id, phone_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_account_name_lang ON templates(whats_app_account, name, language)`,
		`CREATE INDEX IF NOT EXISTS idx_whatsapp_statuses_org_instance_expires ON whatsapp_statuses(organization_id, instance_id, expires_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_whatsapp_statuses_sender_created ON whatsapp_statuses(organization_id, sender_jid, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_keyword_rules_account ON keyword_rules(whats_app_account, is_enabled, priority DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chatbot_flows_account ON chatbot_flows(whats_app_account, is_enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_contexts_account ON ai_contexts(whats_app_account, is_enabled, priority DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bulk_campaigns_account ON bulk_message_campaigns(whats_app_account, status)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_rules_account ON notification_rules(whats_app_account, is_enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(whats_app_account, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_account ON contacts(whats_app_account)`,
		`ALTER TABLE agent_selection_participants ADD COLUMN IF NOT EXISTS settings_id uuid`,
		`ALTER TABLE agent_selection_options ADD COLUMN IF NOT EXISTS settings_id uuid`,
		`UPDATE agent_selection_participants AS p
		 SET settings_id = s.id
		 FROM agent_selection_settings AS s
		 WHERE p.settings_id IS NULL
		   AND p.organization_id = s.organization_id
		   AND s.instance_id IS NULL`,
		`UPDATE agent_selection_options AS o
		 SET settings_id = s.id
		 FROM agent_selection_settings AS s
		 WHERE o.settings_id IS NULL
		   AND o.organization_id = s.organization_id
		   AND s.instance_id IS NULL`,
		`ALTER TABLE agent_selection_settings ADD COLUMN IF NOT EXISTS prompt_delay_min_minutes integer DEFAULT 3`,
		`ALTER TABLE agent_selection_settings ADD COLUMN IF NOT EXISTS prompt_delay_max_minutes integer DEFAULT 3`,
		`UPDATE agent_selection_settings
		 SET prompt_delay_min_minutes = COALESCE(NULLIF(prompt_delay_min_minutes, 0), prompt_delay_minutes, 3),
		     prompt_delay_max_minutes = COALESCE(NULLIF(prompt_delay_max_minutes, 0), prompt_delay_minutes, 3)
		 WHERE COALESCE(prompt_delay_minutes, 0) > 0
		   AND (COALESCE(prompt_delay_min_minutes, 0) = 0 OR COALESCE(prompt_delay_max_minutes, 0) = 0)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_selection_participants_settings_enabled ON agent_selection_participants(settings_id, is_enabled, sort_order)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_selection_options_settings_enabled ON agent_selection_options(settings_id, is_enabled, sort_order)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_canned_responses_org_name ON canned_responses(organization_id, name)`,
		`CREATE INDEX IF NOT EXISTS idx_canned_responses_active ON canned_responses(organization_id, is_active, usage_count DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_org_active ON webhooks(organization_id, is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_availability_logs_user_time ON user_availability_logs(user_id, started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_availability_logs_org_time ON user_availability_logs(organization_id, started_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_providers_org_provider ON sso_providers(organization_id, provider)`,
		// Teams indexes
		`CREATE INDEX IF NOT EXISTS idx_teams_org_active ON teams(organization_id, is_active)`,
		// Create partial unique index (soft-deleted members)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_unique ON team_members(team_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id)`,
		// Custom roles indexes
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_roles_org_name ON custom_roles(organization_id, name)`,
		`CREATE INDEX IF NOT EXISTS idx_custom_roles_org_system ON custom_roles(organization_id, is_system)`,
		`CREATE INDEX IF NOT EXISTS idx_custom_roles_org_default ON custom_roles(organization_id, is_default) WHERE is_default = true`,
		// GIN index for JSONB tag filtering
		`CREATE INDEX IF NOT EXISTS idx_contacts_tags ON contacts USING GIN (tags)`,
		// User organizations
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_unique ON user_organizations(user_id, organization_id) WHERE deleted_at IS NULL`,
		// Conversation notes
		`CREATE INDEX IF NOT EXISTS idx_conversation_notes_contact ON conversation_notes(organization_id, contact_id, created_at DESC)`,
		// Contact collaborators
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_collaborators_unique ON contact_collaborators(contact_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_contact_collaborators_user_status ON contact_collaborators(user_id, status) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_contact_collaborators_contact_status ON contact_collaborators(contact_id, status) WHERE deleted_at IS NULL`,
		// Chat closure ratings
		`CREATE INDEX IF NOT EXISTS idx_chat_closure_ratings_org_closed ON chat_closure_ratings(organization_id, closed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_closure_ratings_contact_state ON chat_closure_ratings(contact_id, state, closed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_closure_ratings_agent_rated ON chat_closure_ratings(agent_user_id, rated_at DESC) WHERE state = 'rated'`,
		`CREATE INDEX IF NOT EXISTS idx_wa_filter_results_batch_phone ON whatsapp_filter_results(batch_id, phone_number)`,
		`CREATE INDEX IF NOT EXISTS idx_wa_filter_results_batch_is_valid ON whatsapp_filter_results(batch_id, is_valid)`,
		`CREATE INDEX IF NOT EXISTS idx_wa_filter_batches_org ON whatsapp_filter_batches(organization_id)`,
		// Group Directory trigram index for name search
		`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
		`CREATE INDEX IF NOT EXISTS idx_gd_name_trgm ON group_directories USING GIN (name gin_trgm_ops)`,
	}
}

// CreateIndexes creates additional indexes not handled by GORM tags
func CreateIndexes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	for _, idx := range getIndexes() {
		if err := db.Exec(idx).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	return nil
}

// CreateDefaultAdmin creates a default admin user if no users exist
// This should only be called once during initial setup
func CreateDefaultAdmin(db *gorm.DB, cfg *config.DefaultAdminConfig) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if cfg == nil {
		return nil
	}
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.FullName = strings.TrimSpace(cfg.FullName)
	if cfg.Email == "" || cfg.Password == "" {
		// Bootstrap admin creation is optional; skip when credentials are not provided.
		return nil
	}
	if cfg.FullName == "" {
		cfg.FullName = "Admin"
	}

	// Check if admin already exists (using email from config)
	var existingAdmin models.User
	if err := db.Where("email = ?", cfg.Email).First(&existingAdmin).Error; err == nil {
		// Admin already exists, skip
		return nil
	}

	// Find any existing organization, or create "Default Organization" if none exist
	var org models.Organization
	if err := db.First(&org).Error; err != nil {
		// No organizations exist, create default one
		org = models.Organization{
			BaseModel: models.BaseModel{ID: uuid.New()},
			Name:      "Default Organization",
			Settings:  models.JSONB{},
		}
		if err := db.Create(&org).Error; err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
	}

	// Hash the default password from config
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Seed permissions if not exist
	if err := SeedPermissionsAndRoles(db); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	// Seed system roles for this organization if not exist
	if err := SeedSystemRolesForOrg(db, org.ID); err != nil {
		return fmt.Errorf("failed to seed system roles: %w", err)
	}

	// Get admin system role for the organization
	var adminRole models.CustomRole
	if err := db.Where("organization_id = ? AND name = ? AND is_system = ?", org.ID, "admin", true).First(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	// Create default admin user (super admin for cross-organization access)
	admin := models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Email:          cfg.Email,
		PasswordHash:   string(passwordHash),
		FullName:       cfg.FullName,
		RoleID:         &adminRole.ID,
		IsActive:       true,
		IsAvailable:    true,
		IsSuperAdmin:   true,
		Settings:       models.JSONB{},
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	// Create UserOrganization entry for the default admin
	userOrg := models.UserOrganization{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		UserID:         admin.ID,
		OrganizationID: org.ID,
		RoleID:         &adminRole.ID,
		IsDefault:      true,
	}
	if err := db.Create(&userOrg).Error; err != nil {
		return fmt.Errorf("failed to create user organization entry: %w", err)
	}

	return nil
}

// MigrateUserOrganizations backfills user_organizations from existing users
func MigrateUserOrganizations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.Exec(`
		INSERT INTO user_organizations (id, user_id, organization_id, role_id, is_default, created_at, updated_at)
		SELECT gen_random_uuid(), u.id, u.organization_id, u.role_id, true, NOW(), NOW()
		FROM users u
		LEFT JOIN user_organizations uo ON uo.user_id = u.id AND uo.organization_id = u.organization_id AND uo.deleted_at IS NULL
		WHERE uo.id IS NULL AND u.deleted_at IS NULL
	`).Error
}

// BackfillOrganizationConfigs ensures every organization has a config row.
func BackfillOrganizationConfigs(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.Exec(`
		INSERT INTO organization_configs (
			id,
			organization_id,
			worker_count,
			max_queue_size,
			max_whatsapp_instances,
			created_at,
			updated_at
		)
		SELECT
			gen_random_uuid(),
			o.id,
			0,
			0,
			0,
			NOW(),
			NOW()
		FROM organizations o
		LEFT JOIN organization_configs oc
			ON oc.organization_id = o.id
			AND oc.deleted_at IS NULL
		WHERE o.deleted_at IS NULL
			AND oc.id IS NULL
	`).Error
}

// BackfillLastInboundAt sets last_inbound_at for existing contacts from their
// most recent incoming message. Only updates contacts where the field is NULL.
func BackfillLastInboundAt(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.Exec(`
		UPDATE contacts c
		SET last_inbound_at = sub.max_created
		FROM (
			SELECT contact_id, MAX(created_at) AS max_created
			FROM messages
			WHERE direction = 'incoming' AND deleted_at IS NULL
			GROUP BY contact_id
		) sub
		WHERE c.id = sub.contact_id AND c.last_inbound_at IS NULL AND c.deleted_at IS NULL
	`).Error
}

// SeedPermissionsAndRoles seeds the default permissions and system roles
func SeedPermissionsAndRoles(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Get all default permissions
	defaultPerms := models.DefaultPermissions()

	// Add any missing permissions
	for _, perm := range defaultPerms {
		var existing models.Permission
		if err := db.Where("resource = ? AND action = ?", perm.Resource, perm.Action).First(&existing).Error; err != nil {
			// Permission doesn't exist, create it
			perm.ID = uuid.New()
			if err := db.Create(&perm).Error; err != nil {
				return fmt.Errorf("failed to create permission %s:%s: %w", perm.Resource, perm.Action, err)
			}
		}
	}

	return nil
}

// SeedSystemRolesForAllOrgs creates system roles for all existing organizations
// This is idempotent - it skips organizations that already have system roles
func SeedSystemRolesForAllOrgs(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var orgs []models.Organization
	if err := db.Find(&orgs).Error; err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	for _, org := range orgs {
		if err := SeedSystemRolesForOrg(db, org.ID); err != nil {
			return fmt.Errorf("failed to seed roles for org %s: %w", org.ID, err)
		}
	}

	// Fix any system roles that don't have permissions linked
	if err := FixSystemRolePermissions(db); err != nil {
		return fmt.Errorf("failed to fix role permissions: %w", err)
	}

	if err := BackfillAdminChatDeletePermission(db); err != nil {
		return fmt.Errorf("failed to backfill admin chat delete permission: %w", err)
	}
	if err := BackfillSystemChatPrefixPermission(db); err != nil {
		return fmt.Errorf("failed to backfill system chat prefix permission: %w", err)
	}
	if err := BackfillSystemContactSoftDeletePermission(db); err != nil {
		return fmt.Errorf("failed to backfill system contact soft delete permission: %w", err)
	}
	if err := BackfillAdminUploadsCleanupPermissions(db); err != nil {
		return fmt.Errorf("failed to backfill admin uploads cleanup permissions: %w", err)
	}
	if err := BackfillSystemChatBypassClaimPermission(db); err != nil {
		return fmt.Errorf("failed to backfill system chat bypass claim permission: %w", err)
	}

	// Migrate existing users from old role column to new role_id
	if err := MigrateExistingUserRoles(db); err != nil {
		return fmt.Errorf("failed to migrate user roles: %w", err)
	}

	// Make admin@admin.com a super admin if exists
	if err := db.Exec("UPDATE users SET is_super_admin = true WHERE email = 'admin@admin.com'").Error; err != nil {
		return fmt.Errorf("failed to set super admin: %w", err)
	}

	return nil
}

// FixSystemRolePermissions links permissions to existing system roles that have no permissions
func FixSystemRolePermissions(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Get all permissions from database
	var permissions []models.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return fmt.Errorf("failed to fetch permissions: %w", err)
	}

	if len(permissions) == 0 {
		return nil // No permissions to link
	}

	// Create permission map for lookup
	permMap := make(map[string]models.Permission)
	for _, p := range permissions {
		permMap[p.Resource+":"+p.Action] = p
	}

	// Get system role permission mappings
	rolePermissions := models.SystemRolePermissions()

	// Find system roles without permissions
	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ?", true).Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to fetch system roles: %w", err)
	}

	for _, role := range systemRoles {
		// Check if role has permissions
		var permCount int64
		db.Table("role_permissions").Where("custom_role_id = ?", role.ID).Count(&permCount)

		if permCount > 0 {
			continue // Already has permissions, don't overwrite customizations
		}

		// Get the permission keys for this role
		permKeys, ok := rolePermissions[role.Name]
		if !ok {
			continue // Unknown role name
		}

		// Link permissions to role
		var permsToAdd []models.Permission
		for _, key := range permKeys {
			if perm, ok := permMap[key]; ok {
				permsToAdd = append(permsToAdd, perm)
			}
		}

		if len(permsToAdd) > 0 {
			if err := db.Model(&role).Association("Permissions").Replace(permsToAdd); err != nil {
				return fmt.Errorf("failed to link permissions to role %s: %w", role.Name, err)
			}
		}
	}

	return nil
}

// BackfillAdminChatDeletePermission ensures existing system admin roles include chat:delete.
// This is idempotent and does not change manager/agent roles.
func BackfillAdminChatDeletePermission(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var permission models.Permission
	if err := db.Where("resource = ? AND action = ?", models.ResourceChat, models.ActionDelete).
		First(&permission).Error; err != nil {
		return fmt.Errorf("failed to resolve chat:delete permission: %w", err)
	}

	var adminRoles []models.CustomRole
	if err := db.Where("is_system = ? AND LOWER(name) = ?", true, "admin").Find(&adminRoles).Error; err != nil {
		return fmt.Errorf("failed to list admin roles: %w", err)
	}

	for _, role := range adminRoles {
		var count int64
		if err := db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", role.ID, permission.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to inspect admin role permissions: %w", err)
		}
		if count > 0 {
			continue
		}

		if err := db.Exec(
			"INSERT INTO role_permissions (custom_role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			role.ID,
			permission.ID,
		).Error; err != nil {
			return fmt.Errorf("failed to backfill admin role %s: %w", role.ID, err)
		}
	}

	return nil
}

// BackfillSystemChatPrefixPermission ensures system admin/manager/agent roles
// include chat:prefix so outgoing message prefix behavior remains available by default.
// This is idempotent and only affects system roles.
func BackfillSystemChatPrefixPermission(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var permission models.Permission
	if err := db.Where("resource = ? AND action = ?", models.ResourceChat, models.ActionPrefix).
		First(&permission).Error; err != nil {
		return fmt.Errorf("failed to resolve chat:prefix permission: %w", err)
	}

	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ? AND LOWER(name) IN ?", true, []string{"admin", "manager", "agent"}).
		Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to list system roles: %w", err)
	}

	for _, role := range systemRoles {
		var count int64
		if err := db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", role.ID, permission.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to inspect role permissions: %w", err)
		}
		if count > 0 {
			continue
		}

		if err := db.Exec(
			"INSERT INTO role_permissions (custom_role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			role.ID,
			permission.ID,
		).Error; err != nil {
			return fmt.Errorf("failed to backfill role %s: %w", role.ID, err)
		}
	}

	return nil
}

func BackfillSystemChatBypassClaimPermission(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var permission models.Permission
	if err := db.Where("resource = ? AND action = ?", models.ResourceChatBypassClaim, models.ActionRead).
		First(&permission).Error; err != nil {
		return fmt.Errorf("failed to resolve chat.bypass_claim:read permission: %w", err)
	}

	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ? AND LOWER(name) IN ?", true, []string{"admin", "manager"}).
		Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to list system roles: %w", err)
	}

	for _, role := range systemRoles {
		var count int64
		if err := db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", role.ID, permission.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to inspect role permissions: %w", err)
		}
		if count > 0 {
			continue
		}

		if err := db.Exec(
			"INSERT INTO role_permissions (custom_role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			role.ID,
			permission.ID,
		).Error; err != nil {
			return fmt.Errorf("failed to backfill role %s: %w", role.ID, err)
		}
	}

	return nil
}

// BackfillSystemContactSoftDeletePermission ensures system admin/manager/agent roles
// include contacts:soft_delete so per-user chat hiding is available by default.
// This is idempotent and only affects system roles.
func BackfillSystemContactSoftDeletePermission(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var permission models.Permission
	if err := db.Where("resource = ? AND action = ?", models.ResourceContacts, models.ActionSoftDelete).
		First(&permission).Error; err != nil {
		return fmt.Errorf("failed to resolve contacts:soft_delete permission: %w", err)
	}

	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ? AND LOWER(name) IN ?", true, []string{"admin", "manager", "agent"}).
		Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to list system roles: %w", err)
	}

	for _, role := range systemRoles {
		var count int64
		if err := db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", role.ID, permission.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to inspect role permissions: %w", err)
		}
		if count > 0 {
			continue
		}

		if err := db.Exec(
			"INSERT INTO role_permissions (custom_role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			role.ID,
			permission.ID,
		).Error; err != nil {
			return fmt.Errorf("failed to backfill role %s: %w", role.ID, err)
		}
	}

	return nil
}

// BackfillAdminUploadsCleanupPermissions ensures existing system admin roles
// include uploads cleanup permissions added after the original role seed.
// This is idempotent and does not change manager/agent roles.
func BackfillAdminUploadsCleanupPermissions(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	requiredPermissions := []struct {
		resource string
		action   string
	}{
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionRead},
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionWrite},
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionExecute},
	}

	permissions := make([]models.Permission, 0, len(requiredPermissions))
	for _, required := range requiredPermissions {
		var permission models.Permission
		if err := db.Where("resource = ? AND action = ?", required.resource, required.action).
			First(&permission).Error; err != nil {
			return fmt.Errorf("failed to resolve %s:%s permission: %w", required.resource, required.action, err)
		}
		permissions = append(permissions, permission)
	}

	var adminRoles []models.CustomRole
	if err := db.Where("is_system = ? AND LOWER(name) = ?", true, "admin").Find(&adminRoles).Error; err != nil {
		return fmt.Errorf("failed to list admin roles: %w", err)
	}

	for _, role := range adminRoles {
		for _, permission := range permissions {
			var count int64
			if err := db.Table("role_permissions").
				Where("custom_role_id = ? AND permission_id = ?", role.ID, permission.ID).
				Count(&count).Error; err != nil {
				return fmt.Errorf("failed to inspect admin role permissions: %w", err)
			}
			if count > 0 {
				continue
			}

			if err := db.Exec(
				"INSERT INTO role_permissions (custom_role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				role.ID,
				permission.ID,
			).Error; err != nil {
				return fmt.Errorf("failed to backfill admin role %s: %w", role.ID, err)
			}
		}
	}

	return nil
}

// MigrateExistingUserRoles migrates users from the old role column to the new role_id
// This is safe to run on fresh installs - it will simply do nothing if the column doesn't exist
func MigrateExistingUserRoles(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Check if the old 'role' column exists in the users table
	var columnExists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'users' AND column_name = 'role'
		)
	`).Scan(&columnExists).Error
	if err != nil {
		return fmt.Errorf("failed to check for role column: %w", err)
	}

	if !columnExists {
		return nil // Fresh install, no old role column
	}

	// Get users who have old role but no role_id assigned
	type UserWithLegacyRole struct {
		ID             uuid.UUID
		OrganizationID uuid.UUID
		LegacyRole     string
	}

	var usersToMigrate []UserWithLegacyRole
	err = db.Raw(`
		SELECT id, organization_id, role as legacy_role
		FROM users
		WHERE role_id IS NULL AND role IS NOT NULL AND role != ''
	`).Scan(&usersToMigrate).Error
	if err != nil {
		return fmt.Errorf("failed to fetch users with legacy roles: %w", err)
	}

	if len(usersToMigrate) == 0 {
		return nil // No users to migrate
	}

	// Get all system roles grouped by organization
	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ?", true).Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to fetch system roles: %w", err)
	}

	// Create lookup: orgID -> roleName -> roleID
	roleMap := make(map[uuid.UUID]map[string]uuid.UUID)
	for _, role := range systemRoles {
		if roleMap[role.OrganizationID] == nil {
			roleMap[role.OrganizationID] = make(map[string]uuid.UUID)
		}
		roleMap[role.OrganizationID][role.Name] = role.ID
	}

	// Migrate each user
	for _, user := range usersToMigrate {
		orgRoles, ok := roleMap[user.OrganizationID]
		if !ok {
			continue // Organization doesn't have system roles yet
		}

		roleID, ok := orgRoles[user.LegacyRole]
		if !ok {
			continue // Role not found (shouldn't happen for admin/manager/agent)
		}

		// Update user's role_id
		if err := db.Exec("UPDATE users SET role_id = ? WHERE id = ?", roleID, user.ID).Error; err != nil {
			return fmt.Errorf("failed to update user %s role: %w", user.ID, err)
		}
	}

	return nil
}

// SeedSystemRolesForOrg creates system roles for an organization
func SeedSystemRolesForOrg(db *gorm.DB, orgID uuid.UUID) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Check if system roles exist for this org
	var roleCount int64
	if err := db.Model(&models.CustomRole{}).Where("organization_id = ? AND is_system = ?", orgID, true).Count(&roleCount).Error; err != nil {
		return fmt.Errorf("failed to count roles: %w", err)
	}

	if roleCount > 0 {
		return nil // Already seeded
	}

	// Get all permissions from database
	var permissions []models.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return fmt.Errorf("failed to fetch permissions: %w", err)
	}

	// Create permission map for lookup
	permMap := make(map[string]models.Permission)
	for _, p := range permissions {
		permMap[p.Resource+":"+p.Action] = p
	}

	// Get system role permission mappings
	rolePermissions := models.SystemRolePermissions()

	// Create system roles
	systemRoles := []struct {
		Name        string
		Description string
		IsDefault   bool
	}{
		{"admin", "Full system access", false},
		{"manager", "Manage chatbot, campaigns, and team operations", false},
		{"agent", "Handle customer conversations", true},
	}

	for _, sr := range systemRoles {
		role := models.CustomRole{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: orgID,
			Name:           sr.Name,
			Description:    sr.Description,
			IsSystem:       true,
			IsDefault:      sr.IsDefault,
		}

		// Add permissions
		permKeys := rolePermissions[sr.Name]
		for _, key := range permKeys {
			if perm, ok := permMap[key]; ok {
				role.Permissions = append(role.Permissions, perm)
			}
		}

		if err := db.Create(&role).Error; err != nil {
			return fmt.Errorf("failed to create %s role: %w", sr.Name, err)
		}
	}

	return nil
}

// SeedDefaultWidgets creates default dashboard widgets for all organizations
func SeedDefaultWidgets(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	// Find the super admin user (admin@admin.com)
	var superAdmin models.User
	if err := db.Where("email = ?", "admin@admin.com").First(&superAdmin).Error; err != nil {
		// No super admin exists yet, skip widget creation
		return nil
	}

	// Get all organizations
	var orgs []models.Organization
	if err := db.Find(&orgs).Error; err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	// Default widget definitions
	defaultWidgetsData := []struct {
		Name         string
		Description  string
		DataSource   string
		DisplayType  string
		Color        string
		Config       models.JSONB
		DisplayOrder int
		GridX        int
		GridY        int
		GridW        int
		GridH        int
	}{
		{"Total Messages", "Total number of messages sent and received", "messages", "number", "blue", nil, 1, 0, 0, 3, 3},
		{"Active Contacts", "Number of contacts with recent activity", "contacts", "number", "green", nil, 2, 3, 0, 3, 3},
		{"Chatbot Sessions", "Active chatbot conversation sessions", "sessions", "number", "purple", nil, 3, 6, 0, 3, 3},
		{"Total Campaigns", "Number of bulk message campaigns", "campaigns", "number", "orange", nil, 4, 9, 0, 3, 3},
		{"Recent Messages", "Latest conversations from your contacts", "messages", "table", "", nil, 5, 0, 3, 6, 8},
		{"Quick Actions", "Common tasks and shortcuts", "shortcuts", "shortcuts", "", models.JSONB{"shortcuts": []interface{}{"chat", "campaigns", "templates", "chatbot"}}, 6, 6, 3, 6, 8},
	}

	for _, org := range orgs {
		// Create default widgets owned by super admin
		for _, wd := range defaultWidgetsData {
			// Skip if a widget with this name already exists for the org
			var exists int64
			db.Model(&models.Widget{}).Where("organization_id = ? AND name = ?", org.ID, wd.Name).Count(&exists)
			if exists > 0 {
				continue
			}

			displayType := wd.DisplayType
			if displayType == "" {
				displayType = "number"
			}
			widget := models.Widget{
				BaseModel:      models.BaseModel{ID: uuid.New()},
				OrganizationID: org.ID,
				UserID:         &superAdmin.ID,
				Name:           wd.Name,
				Description:    wd.Description,
				DataSource:     wd.DataSource,
				Metric:         "count",
				DisplayType:    displayType,
				ShowChange:     displayType == "number",
				Color:          wd.Color,
				Size:           "small",
				Config:         wd.Config,
				DisplayOrder:   wd.DisplayOrder,
				GridX:          wd.GridX,
				GridY:          wd.GridY,
				GridW:          wd.GridW,
				GridH:          wd.GridH,
				IsShared:       true,
				IsDefault:      true,
			}
			db.Create(&widget)
		}
	}

	return nil
}
