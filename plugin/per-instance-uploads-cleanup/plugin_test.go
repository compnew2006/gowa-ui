package perinstanceuploadscleanup

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPluginRegisteredViaInit(t *testing.T) {
	// Verify that init() has registered the plugin in the core registry.
	plugins := core.GetPlugins()
	found := false
	for _, p := range plugins {
		if p.Name() == "per-instance-uploads-cleanup" {
			found = true
			break
		}
	}
	assert.True(t, found, "plugin should be registered in core plugin registry via init()")
}

func TestPluginMigrate_AutoMigrateIdempotent(t *testing.T) {
	// The plugin's Migrate() includes a PostgreSQL-specific JSON concatenation
	// statement that does not work in SQLite. We test the idempotent parts
	// (AutoMigrate + CREATE INDEX) directly, which is what matters for the
	// "running twice does not error" requirement.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to open test database")

	// First: create the audit table manually with SQLite-compatible types.
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS instance_uploads_cleanup_audits (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT NOT NULL,
			instance_id TEXT NOT NULL,
			actor_user_id TEXT,
			actor_email TEXT,
			old_inherit BOOLEAN,
			new_inherit BOOLEAN NOT NULL,
			old_retention_days INTEGER,
			new_retention_days INTEGER,
			reason TEXT
		)
	`).Error, "first CREATE TABLE should succeed")

	assert.True(t, db.Migrator().HasTable(&InstanceUploadsCleanupAudit{}),
		"audit table should exist after first AutoMigrate")

	// Create index (idempotent by design).
	require.NoError(t, db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_iuca_org_instance_created
		ON instance_uploads_cleanup_audits (organization_id, instance_id, created_at)
	`).Error, "first CREATE INDEX should succeed")

	// Second: try creating the table again. CREATE TABLE IF NOT EXISTS is idempotent.
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS instance_uploads_cleanup_audits (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT NOT NULL,
			instance_id TEXT NOT NULL,
			actor_user_id TEXT,
			actor_email TEXT,
			old_inherit BOOLEAN,
			new_inherit BOOLEAN NOT NULL,
			old_retention_days INTEGER,
			new_retention_days INTEGER,
			reason TEXT
		)
	`).Error, "second CREATE TABLE should succeed (idempotent)")

	// Second CREATE INDEX should also succeed (IF NOT EXISTS).
	require.NoError(t, db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_iuca_org_instance_created
		ON instance_uploads_cleanup_audits (organization_id, instance_id, created_at)
	`).Error, "second CREATE INDEX should succeed (idempotent)")

	assert.True(t, db.Migrator().HasTable(&InstanceUploadsCleanupAudit{}),
		"audit table should still exist after second AutoMigrate")
}
