package database

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgres creates a new PostgreSQL connection
func NewPostgres(cfg *config.DatabaseConfig, debug bool) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
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
	Model any
}

// GetMigrationModels returns all models to migrate with their names
func GetMigrationModels() []MigrationModel {
	return []MigrationModel{
		// Core models
		{"Organization", &models.Organization{}},
		{"Permission", &models.Permission{}},
		{"CustomRole", &models.CustomRole{}},
		{"User", &models.User{}},
		{"UserOrganization", &models.UserOrganization{}},
		{"UserWhatsAppAccount", &models.UserWhatsAppAccount{}},
		{"Team", &models.Team{}},
		{"TeamMember", &models.TeamMember{}},
		{"APIKey", &models.APIKey{}},
		{"SSOProvider", &models.SSOProvider{}},
		{"Webhook", &models.Webhook{}},
		{"CustomAction", &models.CustomAction{}},
		{"WhatsAppAccount", &models.WhatsAppAccount{}},
		{"Contact", &models.Contact{}},
		{"Tag", &models.Tag{}},
		{"Message", &models.Message{}},
		{"Template", &models.Template{}},

		// Bulk & Notifications
		{"BulkMessageCampaign", &models.BulkMessageCampaign{}},
		{"BulkMessageRecipient", &models.BulkMessageRecipient{}},
		{"NotificationRule", &models.NotificationRule{}},

		// User tracking
		{"UserAvailabilityLog", &models.UserAvailabilityLog{}},

		// Canned responses
		{"CannedResponse", &models.CannedResponse{}},

		// Dashboard
		{"Widget", &models.Widget{}},

		// Conversation Notes
		{"ConversationNote", &models.ConversationNote{}},

		{"AuditLog", &models.AuditLog{}},

		// GOWA instances (DB-managed GOWA servers)
		{"GowaInstance", &models.GowaInstance{}},

		// Chat closure CSAT rating cycles
		{"ChatClosureRating", &models.ChatClosureRating{}},

		// Scheduled outgoing messages
		{"ScheduledMessage", &models.ScheduledMessage{}},
	}
}

// RunMigrationWithProgress runs migrations with a progress bar display
func RunMigrationWithProgress(db *gorm.DB, adminCfg *config.DefaultAdminConfig) error {
	// Silence GORM logging during migration
	silentDB := db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})

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

	// One-time destructive cleanup of the removed chatbot / agent-transfer /
	// SLA subsystem. Idempotent (IF EXISTS) — a no-op once the tables and
	// columns are gone. See the chatbot feature removal.
	for _, stmt := range dropChatbotArtifacts() {
		if err := silentDB.Exec(stmt).Error; err != nil {
			fmt.Printf("\n  \033[31m✗ Chatbot cleanup failed\033[0m\n\n")
			return fmt.Errorf("failed to drop chatbot artifacts: %w", err)
		}
	}

	// Seed permissions (always run, will skip if already seeded)
	printProgress(currentStep, totalSteps)
	if err := SeedPermissionsAndRoles(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to seed permissions\033[0m\n\n")
		return err
	}

	// Purge stale permissions left over from removed features (chatbot, calling,
	// flows, IVR, ...) so they no longer appear in the roles UI.
	if err := PurgeStalePermissions(silentDB); err != nil {
		fmt.Printf("\n  \033[31m✗ Failed to purge stale permissions\033[0m\n\n")
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

// dropChatbotArtifacts returns the one-time destructive SQL that removes the
// chatbot, keyword, flow, AI-context, session, and agent-transfer tables plus
// the chatbot tracking columns on contacts. All statements use IF EXISTS so
// re-running is a harmless no-op.
func dropChatbotArtifacts() []string {
	return []string{
		`DROP TABLE IF EXISTS chatbot_session_messages, chatbot_sessions, chatbot_flow_steps, chatbot_flows, keyword_rules, ai_contexts, agent_transfers, chatbot_settings CASCADE`,
		`ALTER TABLE contacts DROP COLUMN IF EXISTS chatbot_last_message_at, DROP COLUMN IF EXISTS chatbot_reminder_sent`,
	}
}

// getIndexes returns all index creation SQL statements
func getIndexes() []string {
	return []string{
		// Expand phone_number columns to support group JIDs (e.g., 120363422675615917@g.us)
		`ALTER TABLE contacts ALTER COLUMN phone_number TYPE varchar(50)`,
		// Heal the mis-named gowa_j_id column created by AutoMigrate before the
		// GowaJID field had an explicit column tag. Raw SQL always used gowa_jid.
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'whatsapp_accounts' AND column_name = 'gowa_j_id') THEN
				IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'whatsapp_accounts' AND column_name = 'gowa_jid') THEN
					UPDATE whatsapp_accounts SET gowa_jid = gowa_j_id WHERE (gowa_jid IS NULL OR gowa_jid = '') AND gowa_j_id IS NOT NULL AND gowa_j_id <> '';
					ALTER TABLE whatsapp_accounts DROP COLUMN gowa_j_id;
				ELSE
					ALTER TABLE whatsapp_accounts RENAME COLUMN gowa_j_id TO gowa_jid;
				END IF;
			END IF;
		END $$`,
		// Meta-era rows stored the connected JID in phone_id; the GOWA-only
		// refactor dropped the PhoneID field and moved webhook resolution to
		// gowa_jid without copying the value across, so webhooks from those
		// devices fail account resolution ("Unknown GOWA device") until a manual
		// re-sync. Backfill gowa_jid from phone_id where it holds a JID.
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'whatsapp_accounts' AND column_name = 'phone_id') THEN
				UPDATE whatsapp_accounts SET gowa_jid = phone_id WHERE (gowa_jid IS NULL OR gowa_jid = '') AND phone_id LIKE '%@s.whatsapp.net';
			END IF;
		END $$`,
		`ALTER TABLE bulk_message_recipients ALTER COLUMN phone_number TYPE varchar(50)`,
		// An earlier prototype of the CSAT feature created chat_closure_ratings
		// with NOT NULL columns (chat_id, closing_agent_id, closed_at, ...) the
		// current model never fills — every cycle insert violated them, so no
		// rating prompt was ever sent. AutoMigrate never drops columns; do it here.
		`ALTER TABLE chat_closure_ratings
			DROP COLUMN IF EXISTS chat_id,
			DROP COLUMN IF EXISTS agent_user_id,
			DROP COLUMN IF EXISTS closing_agent_id,
			DROP COLUMN IF EXISTS closed_at,
			DROP COLUMN IF EXISTS state,
			DROP COLUMN IF EXISTS rating_message,
			DROP COLUMN IF EXISTS rating_message_id,
			DROP COLUMN IF EXISTS close_message,
			DROP COLUMN IF EXISTS close_message_language,
			DROP COLUMN IF EXISTS close_message_id,
			DROP COLUMN IF EXISTS context_messages`,
		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_messages_contact_created ON messages(contact_id, created_at DESC)`,
		// One pending rating cycle per contact — closes the check-then-insert race
		// between concurrent chat closes at the database level.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_closure_ratings_pending ON chat_closure_ratings(contact_id) WHERE status = 'pending' AND deleted_at IS NULL`,
		// Due-scan index for the ScheduledMessageProcessor poller: only pending
		// rows matter, so a partial index keeps it small and hot.
		`CREATE INDEX IF NOT EXISTS idx_scheduled_messages_due ON scheduled_messages(scheduled_at) WHERE status = 'pending' AND deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_messages_contact ON scheduled_messages(contact_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_org_phone ON contacts(organization_id, phone_number)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_assigned_read ON contacts(assigned_user_id, is_read)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_account_name_lang ON templates(whats_app_account, name, language)`,
		`CREATE INDEX IF NOT EXISTS idx_bulk_campaigns_account ON bulk_message_campaigns(whats_app_account, status)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_rules_account ON notification_rules(whats_app_account, is_enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(whats_app_account, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_account ON contacts(whats_app_account)`,
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
	}
}

// CreateDefaultAdmin creates a default admin user if no users exist
// This should only be called once during initial setup
func CreateDefaultAdmin(db *gorm.DB, cfg *config.DefaultAdminConfig) error {
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
	return db.Exec(`
		INSERT INTO user_organizations (id, user_id, organization_id, role_id, is_default, created_at, updated_at)
		SELECT gen_random_uuid(), u.id, u.organization_id, u.role_id, true, NOW(), NOW()
		FROM users u
		LEFT JOIN user_organizations uo ON uo.user_id = u.id AND uo.organization_id = u.organization_id AND uo.deleted_at IS NULL
		WHERE uo.id IS NULL AND u.deleted_at IS NULL
	`).Error
}

// BackfillLastInboundAt sets last_inbound_at for existing contacts from their
// most recent incoming message. Only updates contacts where the field is NULL.
func BackfillLastInboundAt(db *gorm.DB) error {
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

// PurgeStalePermissions removes permissions that are no longer part of the
// current DefaultPermissions() set (e.g. leftovers from removed features such
// as chatbot, calling, flows and IVR) together with their role junction rows.
// It is idempotent and safe to run on every migration.
func PurgeStalePermissions(db *gorm.DB) error {
	valid := make(map[string]bool)
	for _, perm := range models.DefaultPermissions() {
		valid[perm.Resource+":"+perm.Action] = true
	}

	var stale []models.Permission
	if err := db.Find(&stale).Error; err != nil {
		return fmt.Errorf("failed to list permissions: %w", err)
	}

	for _, perm := range stale {
		if valid[perm.Resource+":"+perm.Action] {
			continue
		}
		tx := db.Begin()
		if err := tx.Where("permission_id = ?", perm.ID).Delete(&models.RolePermission{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to purge role_permissions for %s:%s: %w", perm.Resource, perm.Action, err)
		}
		if err := tx.Delete(&models.Permission{}, "id = ?", perm.ID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to purge permission %s:%s: %w", perm.Resource, perm.Action, err)
		}
		tx.Commit()
	}

	return nil
}

// SeedSystemRolesForAllOrgs creates system roles for all existing organizations
// This is idempotent - it skips organizations that already have system roles
func SeedSystemRolesForAllOrgs(db *gorm.DB) error {
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

	// Additively grant newly-mapped permissions to existing system roles
	if err := SyncSystemRolePermissions(db); err != nil {
		return fmt.Errorf("failed to sync role permissions: %w", err)
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

// SyncSystemRolePermissions additively grants any permissions from
// SystemRolePermissions that an existing system role is missing. It never
// removes permissions, so super-admin customizations of system roles are
// preserved while new default grants (e.g. permissions added in an upgrade)
// still reach already-seeded organizations.
func SyncSystemRolePermissions(db *gorm.DB) error {
	var permissions []models.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return fmt.Errorf("failed to fetch permissions: %w", err)
	}
	if len(permissions) == 0 {
		return nil
	}

	permMap := make(map[string]models.Permission)
	for _, p := range permissions {
		permMap[p.Resource+":"+p.Action] = p
	}

	rolePermissions := models.SystemRolePermissions()

	var systemRoles []models.CustomRole
	if err := db.Where("is_system = ?", true).Find(&systemRoles).Error; err != nil {
		return fmt.Errorf("failed to fetch system roles: %w", err)
	}

	for _, role := range systemRoles {
		permKeys, ok := rolePermissions[role.Name]
		if !ok {
			continue // Unknown role name
		}

		// Collect the permission IDs the role already holds
		var existingIDs []uuid.UUID
		if err := db.Table("role_permissions").
			Where("custom_role_id = ?", role.ID).
			Pluck("permission_id", &existingIDs).Error; err != nil {
			return fmt.Errorf("failed to fetch role permissions for %s: %w", role.Name, err)
		}
		existing := make(map[uuid.UUID]bool, len(existingIDs))
		for _, id := range existingIDs {
			existing[id] = true
		}

		var permsToAdd []models.Permission
		for _, key := range permKeys {
			if perm, ok := permMap[key]; ok && !existing[perm.ID] {
				permsToAdd = append(permsToAdd, perm)
			}
		}

		if len(permsToAdd) > 0 {
			if err := db.Model(&role).Association("Permissions").Append(permsToAdd); err != nil {
				return fmt.Errorf("failed to grant permissions to role %s: %w", role.Name, err)
			}
		}
	}

	return nil
}

// MigrateExistingUserRoles migrates users from the old role column to the new role_id
// This is safe to run on fresh installs - it will simply do nothing if the column doesn't exist
func MigrateExistingUserRoles(db *gorm.DB) error {
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
		{"manager", "Manage campaigns and team operations", false},
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

	for _, org := range orgs {
		// Skip orgs that already have widgets
		var exists int64
		db.Model(&models.Widget{}).Where("organization_id = ?", org.ID).Count(&exists)
		if exists > 0 {
			continue
		}

		if err := SeedDefaultWidgetsForOrg(db, org.ID, superAdmin.ID); err != nil {
			return err
		}
	}

	return nil
}

// SeedDefaultWidgetsForOrg creates default dashboard widgets for a single organization.
// Used when a new organization is created at runtime.
func SeedDefaultWidgetsForOrg(db *gorm.DB, orgID, userID uuid.UUID) error {
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
		{"Total Campaigns", "Number of bulk message campaigns", "campaigns", "number", "orange", nil, 4, 9, 0, 3, 3},
		{"Recent Messages", "Latest conversations from your contacts", "messages", "table", "", nil, 5, 0, 3, 6, 8},
		{"Quick Actions", "Common tasks and shortcuts", "shortcuts", "shortcuts", "", models.JSONB{"shortcuts": []any{"chat", "campaigns", "templates"}}, 6, 6, 3, 6, 8},
	}

	for _, wd := range defaultWidgetsData {
		displayType := wd.DisplayType
		if displayType == "" {
			displayType = "number"
		}
		widget := models.Widget{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: orgID,
			UserID:         &userID,
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
		if err := db.Create(&widget).Error; err != nil {
			return fmt.Errorf("failed to create widget %s: %w", wd.Name, err)
		}
	}

	return nil
}
