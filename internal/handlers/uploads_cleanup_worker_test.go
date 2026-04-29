package handlers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUploadsCleanupTestApp(t *testing.T) (*App, string) {
	t.Helper()

	uploadRoot := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "uploads-cleanup.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			settings BLOB
		)
	`).Error)

	app := &App{
		Config: &config.Config{
			Storage: config.StorageConfig{
				Type:      "local",
				LocalPath: uploadRoot,
			},
		},
		DB:  db,
		Log: logf.New(logf.Opts{}),
	}

	return app, uploadRoot
}

func writeAgedUploadFixture(t *testing.T, root, relativePath string, modifiedAt time.Time) string {
	t.Helper()

	fullPath := filepath.Join(root, relativePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte("fixture"), 0o600))
	require.NoError(t, os.Chtimes(fullPath, modifiedAt, modifiedAt))
	return fullPath
}

func TestUploadsCleanupWorker_SweepExpiredUploads_DeletesOnlyExpiredTransientFiles(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	worker := NewUploadsCleanupWorker(app, 24*time.Hour)
	now := time.Now().UTC()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Cleanup Org",
		Slug:      "cleanup-" + uuid.NewString(),
		Settings: models.JSONB{
			"uploads_cleanup_retention_days": 5,
		},
	}
	require.NoError(t, app.DB.Create(&org).Error)

	oldDocument := writeAgedUploadFixture(t, uploadRoot, filepath.Join("documents", "expired.pdf"), now.Add(-6*24*time.Hour))
	freshVideo := writeAgedUploadFixture(t, uploadRoot, filepath.Join("videos", "fresh.mp4"), now.Add(-2*24*time.Hour))
	excludedBackground := writeAgedUploadFixture(t, uploadRoot, filepath.Join("chat-backgrounds", "user-1", "bg.png"), now.Add(-10*24*time.Hour))

	result, err := worker.sweepExpiredUploads(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedFiles)
	assert.Equal(t, 5, result.RetentionDays)

	_, oldErr := os.Stat(oldDocument)
	assert.ErrorIs(t, oldErr, os.ErrNotExist)

	_, freshErr := os.Stat(freshVideo)
	assert.NoError(t, freshErr)

	_, excludedErr := os.Stat(excludedBackground)
	assert.NoError(t, excludedErr)
}

func TestUploadsCleanupWorker_SweepExpiredUploads_KeepsFreshlyRestoredFiles(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	worker := NewUploadsCleanupWorker(app, 24*time.Hour)
	now := time.Now().UTC()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Cleanup Org",
		Slug:      "cleanup-" + uuid.NewString(),
		Settings: models.JSONB{
			"uploads_cleanup_retention_days": 5,
		},
	}
	require.NoError(t, app.DB.Create(&org).Error)

	restoredDocument := writeAgedUploadFixture(t, uploadRoot, filepath.Join("documents", "restored.pdf"), now)

	result, err := worker.sweepExpiredUploads(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, 0, result.DeletedFiles)
	assert.Equal(t, 5, result.RetentionDays)

	_, statErr := os.Stat(restoredDocument)
	assert.NoError(t, statErr)
}

func TestUploadsCleanupWorker_DueOrganizations_UsesFixedDailyHourAndLastRunDate(t *testing.T) {
	t.Parallel()

	app, _ := newUploadsCleanupTestApp(t)
	worker := NewUploadsCleanupWorker(app, time.Minute)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Cleanup Org",
		Slug:      "cleanup-" + uuid.NewString(),
		Settings: models.JSONB{
			organizationSettingUploadsCleanupRetentionDays: 5,
			organizationSettingUploadsCleanupScheduleHour:  3,
			"timezone": "UTC",
		},
	}
	require.NoError(t, app.DB.Create(&org).Error)

	beforeWindow := time.Date(2026, time.April, 13, 2, 59, 0, 0, time.UTC)
	due, effectiveRetention, err := worker.dueOrganizations(context.Background(), beforeWindow)
	require.NoError(t, err)
	assert.Empty(t, due)
	assert.Equal(t, 5, effectiveRetention)

	atWindow := time.Date(2026, time.April, 13, 3, 0, 0, 0, time.UTC)
	due, effectiveRetention, err = worker.dueOrganizations(context.Background(), atWindow)
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, 5, effectiveRetention)
	assert.Equal(t, "2026-04-13", due[0].LocalDate)

	require.NoError(t, worker.markOrganizationsAsRan(context.Background(), due))

	due, effectiveRetention, err = worker.dueOrganizations(context.Background(), atWindow.Add(8*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, due)
	assert.Equal(t, 5, effectiveRetention)
}

func TestUploadsCleanupWorker_RunManualCleanup_UsesOrganizationRetention(t *testing.T) {
	t.Parallel()

	app, uploadRoot := newUploadsCleanupTestApp(t)
	worker := NewUploadsCleanupWorker(app, time.Minute)
	now := time.Now().UTC()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Cleanup Org",
		Slug:      "cleanup-" + uuid.NewString(),
		Settings: models.JSONB{
			organizationSettingUploadsCleanupRetentionDays: 7,
			organizationSettingUploadsCleanupScheduleHour:  3,
		},
	}
	require.NoError(t, app.DB.Create(&org).Error)

	oldImage := writeAgedUploadFixture(t, uploadRoot, filepath.Join("images", "expired.png"), now.Add(-8*24*time.Hour))
	freshImage := writeAgedUploadFixture(t, uploadRoot, filepath.Join("images", "fresh.png"), now.Add(-2*24*time.Hour))

	result, err := worker.RunManualCleanup(context.Background(), org.ID, now)
	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedFiles)
	assert.Equal(t, 7, result.RetentionDays)

	_, oldErr := os.Stat(oldImage)
	assert.ErrorIs(t, oldErr, os.ErrNotExist)

	_, freshErr := os.Stat(freshImage)
	assert.NoError(t, freshErr)
}

func TestUploadsCleanupWorker_RunManualCleanup_DeletesOrganizationScopedUploads(t *testing.T) {
	app, uploadRoot := newUploadsCleanupTestApp(t)
	worker := NewUploadsCleanupWorker(app, time.Minute)
	now := time.Now().UTC()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Cleanup Org",
		Slug:      "cleanup-" + uuid.NewString(),
		Settings: models.JSONB{
			organizationSettingUploadsCleanupRetentionDays: 5,
			organizationSettingUploadsCleanupScheduleHour:  3,
		},
	}
	require.NoError(t, app.DB.Create(&org).Error)

	expiredDocument := writeAgedUploadFixture(t, uploadRoot, organizationMediaSubdir(org.ID, "documents", "expired.pdf"), now.Add(-6*24*time.Hour))
	freshImage := writeAgedUploadFixture(t, uploadRoot, organizationMediaSubdir(org.ID, "images", "fresh.png"), now.Add(-2*24*time.Hour))
	excludedBackground := writeAgedUploadFixture(t, uploadRoot, organizationMediaSubdir(org.ID, "chat-backgrounds", "bg.png"), now.Add(-10*24*time.Hour))

	result, err := worker.RunManualCleanup(context.Background(), org.ID, now)
	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedFiles)
	assert.Equal(t, 5, result.RetentionDays)

	_, expiredErr := os.Stat(expiredDocument)
	assert.ErrorIs(t, expiredErr, os.ErrNotExist)

	_, freshErr := os.Stat(freshImage)
	assert.NoError(t, freshErr)

	_, excludedErr := os.Stat(excludedBackground)
	assert.NoError(t, excludedErr)
}
