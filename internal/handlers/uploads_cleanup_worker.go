package handlers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const uploadsCleanupWorkerLockKey int64 = 2026041301

var errUploadsCleanupDisabled = errors.New("uploads cleanup retention is disabled")

// UploadsCleanupRunResult summarizes a cleanup execution.
type UploadsCleanupRunResult struct {
	DeletedFiles           int
	RetentionDays          int
	ScheduledOrganizations int
}

type uploadsCleanupSchedule struct {
	OrganizationID uuid.UUID
	RetentionDays  int
	ScheduleHour   int
	Timezone       string
	LocalDate      string
	LastRunDate    string
}

// UploadsCleanupWorker deletes transient local uploads older than the configured retention window.
type UploadsCleanupWorker struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

func NewUploadsCleanupWorker(app *App, interval time.Duration) *UploadsCleanupWorker {
	return &UploadsCleanupWorker{
		app:      app,
		interval: interval,
	}
}

func (w *UploadsCleanupWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.ticker = time.NewTicker(w.interval)
	ticker := w.ticker
	w.mu.Unlock()
	defer ticker.Stop()

	w.runDueOrganizations(ctx, time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			w.runDueOrganizations(ctx, tick.UTC())
		}
	}
}

func (w *UploadsCleanupWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

func (w *UploadsCleanupWorker) RunManualCleanup(ctx context.Context, orgID uuid.UUID, now time.Time) (UploadsCleanupRunResult, error) {
	if w.app == nil || w.app.DB == nil || w.app.Config == nil {
		return UploadsCleanupRunResult{}, nil
	}

	retentionDays, err := w.retentionDaysForOrganization(ctx, orgID)
	if err != nil {
		return UploadsCleanupRunResult{}, err
	}
	if retentionDays <= 0 {
		return UploadsCleanupRunResult{}, errUploadsCleanupDisabled
	}

	acquired, err := w.tryAcquireLock(ctx)
	if err != nil {
		return UploadsCleanupRunResult{}, err
	}
	if !acquired {
		return UploadsCleanupRunResult{}, fmt.Errorf("uploads cleanup is already running")
	}
	defer w.releaseLock(context.Background())

	deletedFiles, err := w.deleteExpiredUploadFiles(now.UTC(), retentionDays)
	if err != nil {
		return UploadsCleanupRunResult{}, err
	}

	return UploadsCleanupRunResult{
		DeletedFiles:  deletedFiles,
		RetentionDays: retentionDays,
	}, nil
}

func (w *UploadsCleanupWorker) runDueOrganizations(ctx context.Context, now time.Time) {
	if w.app == nil || w.app.DB == nil || w.app.Config == nil {
		return
	}

	dueOrganizations, effectiveRetention, err := w.dueOrganizations(ctx, now)
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker failed to evaluate schedule", "error", err)
		return
	}
	if len(dueOrganizations) == 0 || effectiveRetention <= 0 {
		return
	}

	acquired, err := w.tryAcquireLock(ctx)
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker failed to acquire advisory lock", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer w.releaseLock(context.Background())

	dueOrganizations, effectiveRetention, err = w.dueOrganizations(ctx, now)
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker failed to re-evaluate schedule", "error", err)
		return
	}
	if len(dueOrganizations) == 0 || effectiveRetention <= 0 {
		return
	}

	deletedFiles, err := w.deleteExpiredUploadFiles(now.UTC(), effectiveRetention)
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker sweep failed", "error", err)
		return
	}

	if err := w.markOrganizationsAsRan(ctx, dueOrganizations); err != nil {
		w.app.Log.Error("Uploads cleanup worker failed to persist last run date", "error", err)
		return
	}

	w.app.Log.Info(
		"Uploads cleanup worker completed scheduled sweep",
		"deleted_files", deletedFiles,
		"retention_days", effectiveRetention,
		"scheduled_organizations", len(dueOrganizations),
	)
}

func (w *UploadsCleanupWorker) tryAcquireLock(ctx context.Context) (bool, error) {
	if w.app.DB.Dialector.Name() != "postgres" {
		return true, nil
	}

	var acquired bool
	if err := w.app.DB.WithContext(ctx).
		Raw("SELECT pg_try_advisory_lock(?)", uploadsCleanupWorkerLockKey).
		Scan(&acquired).Error; err != nil {
		return false, err
	}
	return acquired, nil
}

func (w *UploadsCleanupWorker) releaseLock(ctx context.Context) {
	if w == nil || w.app == nil || w.app.DB == nil {
		return
	}
	if w.app.DB.Dialector.Name() != "postgres" {
		return
	}

	if err := w.app.DB.WithContext(ctx).
		Exec("SELECT pg_advisory_unlock(?)", uploadsCleanupWorkerLockKey).
		Error; err != nil {
		w.app.Log.Warn("Uploads cleanup worker failed to release advisory lock", "error", err)
	}
}

func (w *UploadsCleanupWorker) sweepExpiredUploads(ctx context.Context, now time.Time) (UploadsCleanupRunResult, error) {
	if w == nil || w.app == nil || w.app.DB == nil {
		return UploadsCleanupRunResult{}, nil
	}

	retentionDays, err := w.effectiveRetentionDays(ctx)
	if err != nil {
		return UploadsCleanupRunResult{}, err
	}
	if retentionDays <= 0 {
		return UploadsCleanupRunResult{}, errUploadsCleanupDisabled
	}

	deletedFiles, err := w.deleteExpiredUploadFiles(now.UTC(), retentionDays)
	if err != nil {
		return UploadsCleanupRunResult{}, err
	}

	return UploadsCleanupRunResult{
		DeletedFiles:  deletedFiles,
		RetentionDays: retentionDays,
	}, nil
}

func (w *UploadsCleanupWorker) dueOrganizations(ctx context.Context, now time.Time) ([]uploadsCleanupSchedule, int, error) {
	schedules, err := w.loadOrganizationSchedules(ctx, now)
	if err != nil {
		return nil, 0, err
	}

	due := make([]uploadsCleanupSchedule, 0, len(schedules))
	effectiveRetention := 0
	for _, schedule := range schedules {
		if schedule.RetentionDays <= 0 {
			continue
		}

		if effectiveRetention == 0 || schedule.RetentionDays < effectiveRetention {
			effectiveRetention = schedule.RetentionDays
		}

		if schedule.LocalDate == "" || schedule.LastRunDate == schedule.LocalDate {
			continue
		}

		location, err := time.LoadLocation(schedule.Timezone)
		if err != nil {
			location = time.UTC
		}
		if now.In(location).Hour() < schedule.ScheduleHour {
			continue
		}

		due = append(due, schedule)
	}

	return due, effectiveRetention, nil
}

func (w *UploadsCleanupWorker) loadOrganizationSchedules(ctx context.Context, now time.Time) ([]uploadsCleanupSchedule, error) {
	var organizations []models.Organization
	if err := w.app.DB.WithContext(ctx).
		Select("id", "settings").
		Find(&organizations).Error; err != nil {
		return nil, fmt.Errorf("load organization settings: %w", err)
	}

	schedules := make([]uploadsCleanupSchedule, 0, len(organizations))
	for _, org := range organizations {
		timezone := parseOrganizationTimezone(org.Settings)
		location, err := time.LoadLocation(timezone)
		if err != nil {
			timezone = "UTC"
			location = time.UTC
		}

		schedules = append(schedules, uploadsCleanupSchedule{
			OrganizationID: org.ID,
			RetentionDays:  parseUploadsCleanupRetentionDays(org.Settings),
			ScheduleHour:   parseUploadsCleanupScheduleHour(org.Settings),
			Timezone:       timezone,
			LocalDate:      now.In(location).Format("2006-01-02"),
			LastRunDate:    parseUploadsCleanupLastRunDate(org.Settings),
		})
	}

	return schedules, nil
}

func (w *UploadsCleanupWorker) markOrganizationsAsRan(ctx context.Context, schedules []uploadsCleanupSchedule) error {
	for _, schedule := range schedules {
		var org models.Organization
		if err := w.app.DB.WithContext(ctx).
			Select("id", "settings").
			Where("id = ?", schedule.OrganizationID).
			First(&org).Error; err != nil {
			return fmt.Errorf("load organization for uploads cleanup last run date: %w", err)
		}

		if org.Settings == nil {
			org.Settings = models.JSONB{}
		}
		org.Settings[organizationSettingUploadsCleanupLastRunDate] = schedule.LocalDate

		if err := w.app.DB.WithContext(ctx).
			Model(&org).
			Update("settings", org.Settings).
			Error; err != nil {
			return fmt.Errorf("persist uploads cleanup last run date: %w", err)
		}
	}

	return nil
}

func (w *UploadsCleanupWorker) effectiveRetentionDays(ctx context.Context) (int, error) {
	var organizations []models.Organization
	if err := w.app.DB.WithContext(ctx).
		Select("id", "settings").
		Find(&organizations).Error; err != nil {
		return 0, fmt.Errorf("load organization settings: %w", err)
	}

	effective := 0
	for _, org := range organizations {
		days := parseUploadsCleanupRetentionDays(org.Settings)
		if days <= 0 {
			continue
		}
		if effective == 0 || days < effective {
			effective = days
		}
	}

	return effective, nil
}

func (w *UploadsCleanupWorker) retentionDaysForOrganization(ctx context.Context, orgID uuid.UUID) (int, error) {
	var org models.Organization
	if err := w.app.DB.WithContext(ctx).
		Select("id", "settings").
		Where("id = ?", orgID).
		First(&org).Error; err != nil {
		return 0, fmt.Errorf("load organization cleanup settings: %w", err)
	}

	return parseUploadsCleanupRetentionDays(org.Settings), nil
}

func (w *UploadsCleanupWorker) deleteExpiredUploadFiles(now time.Time, retentionDays int) (int, error) {
	rootPath, err := filepath.Abs(w.app.getMediaStoragePath())
	if err != nil {
		return 0, fmt.Errorf("resolve uploads root: %w", err)
	}

	cutoff := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	deletedFiles := 0
	for _, relativeDir := range uploadsCleanupTargetDirs {
		dirPath := filepath.Join(rootPath, relativeDir)
		count, err := w.deleteExpiredFilesFromDir(rootPath, dirPath, cutoff, retentionDays)
		if err != nil {
			return 0, err
		}
		deletedFiles += count
	}

	orgScopedDeletedFiles, err := w.deleteExpiredOrganizationUploadFiles(rootPath, cutoff, retentionDays)
	if err != nil {
		return 0, err
	}
	deletedFiles += orgScopedDeletedFiles

	return deletedFiles, nil
}

func (w *UploadsCleanupWorker) deleteExpiredOrganizationUploadFiles(rootPath string, cutoff time.Time, retentionDays int) (int, error) {
	orgsRoot := filepath.Join(rootPath, "orgs")
	entries, err := os.ReadDir(orgsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read organization uploads directory %q: %w", orgsRoot, err)
	}

	deletedFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		for _, relativeDir := range uploadsCleanupTargetDirs {
			dirPath := filepath.Join(orgsRoot, entry.Name(), relativeDir)
			count, err := w.deleteExpiredFilesFromDir(rootPath, dirPath, cutoff, retentionDays)
			if err != nil {
				return 0, err
			}
			deletedFiles += count
		}
	}

	return deletedFiles, nil
}

func (w *UploadsCleanupWorker) deleteExpiredFilesFromDir(rootPath, dirPath string, cutoff time.Time, retentionDays int) (int, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("stat uploads directory %q: %w", dirPath, err)
	}
	if !info.IsDir() {
		return 0, nil
	}

	deletedFiles := 0
	err = filepath.WalkDir(dirPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk uploads directory %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat upload file %q: %w", path, err)
		}
		if fileInfo.ModTime().After(cutoff) {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return fmt.Errorf("resolve upload relative path %q: %w", path, err)
		}

		now := time.Now().UTC()
		errDb := w.app.DB.Transaction(func(tx *gorm.DB) error {
			// 1. Mark legacy local file messages as deleted.
			if err := tx.Model(&models.Message{}).
				Where("media_url = ? AND media_deleted_at IS NULL", relPath).
				Updates(map[string]any{
					"media_deleted_at": now,
					"updated_at":       now,
				}).Error; err != nil {
				return fmt.Errorf("update legacy local media_deleted_at: %w", err)
			}

			// 2. Mark object-storage/minio assets with the matching S3Key as deleted.
			var asset models.MediaAsset
			if err := tx.Where("s3_key = ?", relPath).First(&asset).Error; err == nil {
				if err := tx.Model(&models.Message{}).
					Where("media_asset_id = ? AND media_deleted_at IS NULL", asset.ID).
					Updates(map[string]any{
						"media_deleted_at": now,
						"updated_at":       now,
					}).Error; err != nil {
					return fmt.Errorf("update messages media_deleted_at for asset %s: %w", asset.ID, err)
				}

				if err := tx.Delete(&asset).Error; err != nil {
					return fmt.Errorf("delete media asset %s: %w", asset.ID, err)
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("lookup media asset for s3_key %s: %w", relPath, err)
			}

			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("delete upload file %q: %w", path, err)
			}

			return nil
		})
		if errDb != nil {
			w.app.Log.Error("Uploads cleanup worker failed to sync database on file deletion", "path", path, "error", errDb)
			return errDb
		}

		deletedFiles++
		w.app.Log.Info("Uploads cleanup worker deleted file", "path", path, "retention_days", retentionDays)

		return nil
	})
	if err != nil {
		return 0, err
	}

	return deletedFiles, nil
}
