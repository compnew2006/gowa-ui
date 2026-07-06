package perinstanceuploadscleanup

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database with the schema needed for
// the uploads-cleanup service tests. It creates minimal table definitions that
// match the GORM models used by the service.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to open SQLite in-memory database")

	// Create the tables required by the service.
	require.NoError(t, db.Exec(`
		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT,
			slug TEXT,
			settings JSON
		)
	`).Error, "failed to create organizations table")

	require.NoError(t, db.Exec(`
		CREATE TABLE whatsapp_instances (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT,
			name TEXT,
			phone_number TEXT,
			jid TEXT,
			status TEXT DEFAULT 'disconnected',
			is_default BOOLEAN DEFAULT FALSE,
			session_id TEXT,
			auto_read_receipt BOOLEAN DEFAULT FALSE,
			settings JSON DEFAULT '{}',
			last_connected_at DATETIME,
			send_blocked_until DATETIME,
			send_block_reason TEXT DEFAULT ''
		)
	`).Error, "failed to create whatsapp_instances table")

	// Create the audit table manually with SQLite-compatible types.
	// GORM AutoMigrate fails because the model uses `type:uuid` which SQLite does not support.
	require.NoError(t, db.Exec(`
		CREATE TABLE instance_uploads_cleanup_audits (
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
	`).Error, "failed to create audit table")

	// Create indexes to match the GORM model tags.
	require.NoError(t, db.Exec(`CREATE INDEX idx_iuca_org_id ON instance_uploads_cleanup_audits (organization_id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_iuca_inst_id ON instance_uploads_cleanup_audits (instance_id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_iuca_actor_id ON instance_uploads_cleanup_audits (actor_user_id)`).Error)

	return db
}

// seedOrganization inserts an organization row and returns its UUID.
func seedOrganization(t *testing.T, db *gorm.DB, settings models.JSONB) uuid.UUID {
	t.Helper()

	orgID := uuid.New()
	settingsJSON := "{}"
	if settings != nil {
		b, err := settings.Value()
		require.NoError(t, err)
		settingsJSON = string(b.([]byte))
	}

	require.NoError(t, db.Exec(
		`INSERT INTO organizations (id, created_at, updated_at, name, slug, settings) VALUES (?, ?, ?, ?, ?, ?)`,
		orgID.String(), time.Now().UTC(), time.Now().UTC(), "Test Org", "test-org", settingsJSON,
	).Error, "failed to seed organization")

	return orgID
}

// seedInstance inserts a WhatsAppInstance row and returns its UUID.
func seedInstance(t *testing.T, db *gorm.DB, orgID uuid.UUID, settings models.JSONB) uuid.UUID {
	t.Helper()

	instanceID := uuid.New()
	settingsJSON := "{}"
	if settings != nil {
		b, err := settings.Value()
		require.NoError(t, err)
		settingsJSON = string(b.([]byte))
	}

	require.NoError(t, db.Exec(
		`INSERT INTO whatsapp_instances (id, created_at, updated_at, organization_id, name, settings) VALUES (?, ?, ?, ?, ?, ?)`,
		instanceID.String(), time.Now().UTC(), time.Now().UTC(), orgID.String(), "Test Instance", settingsJSON,
	).Error, "failed to seed instance")

	return instanceID
}

func TestResolveEffectiveRetention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		instanceSettings models.JSONB
		orgSettings      models.JSONB
		wantDays         int
		wantSource       string
		wantErr          bool
		skipInstance     bool
		skipOrg          bool
	}{
		{
			name: "inherit true resolves to workspace default",
			instanceSettings: models.JSONB{
				"uploads_cleanup": map[string]interface{}{
					"inherit":        true,
					"retention_days": float64(5),
				},
			},
			orgSettings: models.JSONB{
				"uploads_cleanup_retention_days": float64(90),
			},
			wantDays:   90,
			wantSource: "default",
		},
		{
			name: "inherit false with custom retention_days returns custom",
			instanceSettings: models.JSONB{
				"uploads_cleanup": map[string]interface{}{
					"inherit":        false,
					"retention_days": float64(5),
				},
			},
			orgSettings: models.JSONB{
				"uploads_cleanup_retention_days": float64(90),
			},
			wantDays:   5,
			wantSource: "custom",
		},
		{
			name: "inherit false with retention_days 0 returns disabled",
			instanceSettings: models.JSONB{
				"uploads_cleanup": map[string]interface{}{
					"inherit":        false,
					"retention_days": float64(0),
				},
			},
			orgSettings: models.JSONB{},
			wantDays:    0,
			wantSource:  "disabled",
		},
		{
			name:         "instance not found returns error",
			skipInstance: true,
			orgSettings:  models.JSONB{},
			wantErr:      true,
		},
		{
			name: "no uploads_cleanup key resolves to workspace default",
			instanceSettings: models.JSONB{
				"some_other_key": "value",
			},
			orgSettings: models.JSONB{
				"uploads_cleanup_retention_days": float64(60),
			},
			wantDays:   60,
			wantSource: "default",
		},
		{
			name: "workspace default is 0 returns disabled",
			instanceSettings: models.JSONB{
				"uploads_cleanup": map[string]interface{}{
					"inherit": true,
				},
			},
			orgSettings: models.JSONB{
				"uploads_cleanup_retention_days": float64(0),
			},
			wantDays:   0,
			wantSource: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			orgID := seedOrganization(t, db, tt.orgSettings)

			srv := newService(db, slog.Default())

			var instanceID uuid.UUID
			if !tt.skipInstance {
				instanceID = seedInstance(t, db, orgID, tt.instanceSettings)
			} else {
				instanceID = uuid.New()
			}

			days, source, err := srv.ResolveEffectiveRetention(context.Background(), orgID, instanceID, time.Now())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDays, days, "days mismatch")
			assert.Equal(t, tt.wantSource, source, "source mismatch")
		})
	}
}

func TestWriteAuditRow(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	srv := newService(db, slog.Default())

	orgID := uuid.New()
	instanceID := uuid.New()
	userID := uuid.New()
	email := "test@example.com"
	reason := "policy change"
	oldInherit := true
	newInherit := false
	oldDays := 30
	newDays := 60

	err := srv.WriteAuditRow(
		context.Background(),
		orgID,
		instanceID,
		&userID,
		&email,
		RetentionSnapshot{Inherit: oldInherit, RetentionDays: &oldDays},
		RetentionSnapshot{Inherit: newInherit, RetentionDays: &newDays},
		&reason,
	)
	require.NoError(t, err, "WriteAuditRow should succeed")

	// Verify the row was written with correct fields.
	var audit InstanceUploadsCleanupAudit
	require.NoError(t, db.First(&audit).Error, "audit row should exist")

	assert.Equal(t, orgID, audit.OrganizationID, "organization_id mismatch")
	assert.Equal(t, instanceID, audit.InstanceID, "instance_id mismatch")
	assert.Equal(t, userID, *audit.ActorUserID, "actor_user_id mismatch")
	assert.Equal(t, email, *audit.ActorEmail, "actor_email mismatch")
	assert.Equal(t, oldInherit, *audit.OldInherit, "old_inherit mismatch")
	assert.Equal(t, newInherit, audit.NewInherit, "new_inherit mismatch")
	assert.Equal(t, oldDays, *audit.OldRetentionDays, "old_retention_days mismatch")
	assert.Equal(t, newDays, *audit.NewRetentionDays, "new_retention_days mismatch")
	assert.Equal(t, reason, *audit.Reason, "reason mismatch")
}

func TestWriteAuditRow_NilOptionalFields(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	srv := newService(db, slog.Default())

	orgID := uuid.New()
	instanceID := uuid.New()

	err := srv.WriteAuditRow(
		context.Background(),
		orgID,
		instanceID,
		nil, // no actor user ID
		nil, // no actor email
		RetentionSnapshot{Inherit: true, RetentionDays: nil},
		RetentionSnapshot{Inherit: false, RetentionDays: intPtr(10)},
		nil, // no reason
	)
	require.NoError(t, err, "WriteAuditRow with nil optional fields should succeed")

	var audit InstanceUploadsCleanupAudit
	require.NoError(t, db.First(&audit).Error, "audit row should exist")

	assert.Equal(t, orgID, audit.OrganizationID)
	assert.Nil(t, audit.ActorUserID, "actor_user_id should be nil")
	assert.Nil(t, audit.ActorEmail, "actor_email should be nil")
	assert.Nil(t, audit.Reason, "reason should be nil")
}

func TestTryAcquireInstanceRun(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	srv := newService(db, slog.Default())

	// First call should succeed.
	release, ok := srv.tryAcquireInstanceRun()
	assert.True(t, ok, "first acquire should succeed")

	// After releasing, a second acquire should also succeed.
	release()
	release2, ok2 := srv.tryAcquireInstanceRun()
	assert.True(t, ok2, "second acquire after release should succeed")
	release2()
}
