package handlers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMessageHasVisibleMedia(t *testing.T) {
	now := time.Now().UTC()
	assert.True(t, messageHasVisibleMedia(&models.Message{MediaURL: "images/file.jpg"}))
	assert.False(t, messageHasVisibleMedia(&models.Message{MediaURL: ""}))
	assert.False(t, messageHasVisibleMedia(&models.Message{MediaURL: "images/file.jpg", MediaDeletedAt: &now}))
	assert.False(t, messageHasVisibleMedia(nil))
}

func TestReconcileMissingLegacyMediaMarksOnlyMissingOldFiles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			media_url TEXT,
			media_asset_id TEXT,
			media_deleted_at DATETIME,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)

	tempDir := t.TempDir()
	require.NoError(t, db.Exec(
		`INSERT INTO messages (id, media_url, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), "images/present.jpg", time.Now().UTC().Add(-2*time.Hour), time.Now().UTC(),
	).Error)
	missingID := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO messages (id, media_url, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		missingID, "images/missing.jpg", time.Now().UTC().Add(-2*time.Hour), time.Now().UTC(),
	).Error)
	recentMissingID := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO messages (id, media_url, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		recentMissingID, "images/recent-missing.jpg", time.Now().UTC().Add(-5*time.Minute), time.Now().UTC(),
	).Error)
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "images"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "images", "present.jpg"), []byte("ok"), 0o644))

	dryRun, err := ReconcileMissingLegacyMedia(context.Background(), db, tempDir, LegacyMediaReconcileOptions{
		OlderThan: time.Hour,
	})
	require.NoError(t, err)
	assert.True(t, dryRun.DryRun)
	assert.Equal(t, 2, dryRun.CandidateCount)
	assert.Equal(t, 1, dryRun.MissingCount)
	assert.Equal(t, 0, dryRun.UpdatedCount)
	assert.Contains(t, dryRun.SampleIDs, missingID)

	applied, err := ReconcileMissingLegacyMedia(context.Background(), db, tempDir, LegacyMediaReconcileOptions{
		OlderThan: time.Hour,
		Apply:     true,
	})
	require.NoError(t, err)
	assert.False(t, applied.DryRun)
	assert.Equal(t, 1, applied.MissingCount)
	assert.Equal(t, 1, applied.UpdatedCount)

	var deletedAt sql.NullTime
	require.NoError(t, db.Raw(`SELECT media_deleted_at FROM messages WHERE id = ?`, missingID).Scan(&deletedAt).Error)
	assert.True(t, deletedAt.Valid)
	require.NoError(t, db.Raw(`SELECT media_deleted_at FROM messages WHERE id = ?`, recentMissingID).Scan(&deletedAt).Error)
	assert.False(t, deletedAt.Valid)
}
