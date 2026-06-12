package handlers

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunManualCleanupForInstance_DeletesOnlyExpiredFiles(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	orgID := uuid.New()
	instanceID := uuid.New()
	otherInstanceID := uuid.New()

	expired := time.Now().UTC().Add(-48 * time.Hour)
	fresh := time.Now().UTC().Add(-1 * time.Hour)

	writeAgedUploadFixture(t, uploadRoot, filepath.Join("orgs", orgID.String(), "images", instanceID.String(), "old.png"), expired)
	writeAgedUploadFixture(t, uploadRoot, filepath.Join("orgs", orgID.String(), "images", instanceID.String(), "fresh.png"), fresh)
	writeAgedUploadFixture(t, uploadRoot, filepath.Join("orgs", orgID.String(), "images", otherInstanceID.String(), "other.png"), expired)

	days := 1
	deleted, err := RunManualCleanupForInstance(context.Background(), app, orgID, instanceID, &days)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	assert.NoFileExists(t, filepath.Join(uploadRoot, "orgs", orgID.String(), "images", instanceID.String(), "old.png"))
	assert.FileExists(t, filepath.Join(uploadRoot, "orgs", orgID.String(), "images", instanceID.String(), "fresh.png"))
	assert.FileExists(t, filepath.Join(uploadRoot, "orgs", orgID.String(), "images", otherInstanceID.String(), "other.png"))
}

func TestRunManualCleanupForInstance_DatabaseSyncOnDeletion(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	orgID := uuid.New()
	instanceID := uuid.New()

	expired := time.Now().UTC().Add(-48 * time.Hour)
	relPath := filepath.Join("orgs", orgID.String(), "documents", instanceID.String(), "expired.pdf")
	writeAgedUploadFixture(t, uploadRoot, relPath, expired)

	msgID := uuid.New()
	require.NoError(t, app.DB.Exec(`INSERT INTO messages (id, organization_id, instance_id, media_url, message_type) VALUES (?, ?, ?, ?, ?)`,
		msgID.String(), orgID.String(), instanceID.String(), relPath, "document").Error)

	days := 1
	deleted, err := RunManualCleanupForInstance(context.Background(), app, orgID, instanceID, &days)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	var deletedAt sql.NullTime
	require.NoError(t, app.DB.Raw("SELECT media_deleted_at FROM messages WHERE id = ?", msgID.String()).Scan(&deletedAt).Error)
	assert.True(t, deletedAt.Valid, "media_deleted_at should be set after instance cleanup")
}

func TestRunManualCleanupForInstance_OnlyAffectsTargetInstance(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	orgID := uuid.New()
	instanceA := uuid.New()
	instanceB := uuid.New()

	expired := time.Now().UTC().Add(-72 * time.Hour)

	relPathA := filepath.Join("orgs", orgID.String(), "images", instanceA.String(), "old.jpg")
	relPathB := filepath.Join("orgs", orgID.String(), "images", instanceB.String(), "old.jpg")
	writeAgedUploadFixture(t, uploadRoot, relPathA, expired)
	writeAgedUploadFixture(t, uploadRoot, relPathB, expired)

	msgIDA := uuid.New()
	msgIDB := uuid.New()
	require.NoError(t, app.DB.Exec(`INSERT INTO messages (id, organization_id, media_url, instance_id, message_type) VALUES (?, ?, ?, ?, ?)`,
		msgIDA.String(), orgID.String(), relPathA, instanceA.String(), "image").Error)
	require.NoError(t, app.DB.Exec(`INSERT INTO messages (id, organization_id, media_url, instance_id, message_type) VALUES (?, ?, ?, ?, ?)`,
		msgIDB.String(), orgID.String(), relPathB, instanceB.String(), "image").Error)

	days := 1
	deleted, err := RunManualCleanupForInstance(context.Background(), app, orgID, instanceA, &days)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	var deletedAtA, deletedAtB sql.NullTime
	require.NoError(t, app.DB.Raw("SELECT media_deleted_at FROM messages WHERE id = ?", msgIDA.String()).Scan(&deletedAtA).Error)
	require.NoError(t, app.DB.Raw("SELECT media_deleted_at FROM messages WHERE id = ?", msgIDB.String()).Scan(&deletedAtB).Error)
	assert.True(t, deletedAtA.Valid, "instance A message should be marked deleted")
	assert.False(t, deletedAtB.Valid, "instance B message should NOT be marked deleted")
}

func TestRunManualCleanupForInstance_DisabledWhenRetentionZero(t *testing.T) {
	t.Parallel()

	app, _ := newUploadsCleanupTestApp(t)
	orgID := uuid.New()
	instanceID := uuid.New()

	_, err := RunManualCleanupForInstance(context.Background(), app, orgID, instanceID, intPtr(0))
	assert.Error(t, err)

	_, err = RunManualCleanupForInstance(context.Background(), app, orgID, instanceID, nil)
	assert.Error(t, err)
}

func TestResolveInstanceRetention_CustomRetention(t *testing.T) {
	days, source := ResolveInstanceRetention(models.JSONB{
		"uploads_cleanup": map[string]interface{}{
			"inherit":        false,
			"retention_days": float64(14),
		},
	}, 30)
	assert.Equal(t, 14, days)
	assert.Equal(t, "custom", source)
}

func TestResolveInstanceRetention_InheritWorkspaceDefault(t *testing.T) {
	days, source := ResolveInstanceRetention(models.JSONB{
		"uploads_cleanup": map[string]interface{}{
			"inherit":        true,
			"retention_days": float64(14),
		},
	}, 30)
	assert.Equal(t, 30, days)
	assert.Equal(t, "default", source)
}

func TestResolveInstanceRetention_Disabled(t *testing.T) {
	days, source := ResolveInstanceRetention(models.JSONB{
		"uploads_cleanup": map[string]interface{}{
			"inherit":        false,
			"retention_days": float64(0),
		},
	}, 30)
	assert.Equal(t, 0, days)
	assert.Equal(t, "disabled", source)
}

func TestResolveInstanceRetention_NoConfig(t *testing.T) {
	days, source := ResolveInstanceRetention(models.JSONB{}, 30)
	assert.Equal(t, 30, days)
	assert.Equal(t, "default", source)

	days, source = ResolveInstanceRetention(models.JSONB{}, 0)
	assert.Equal(t, 0, days)
	assert.Equal(t, "disabled", source)
}
