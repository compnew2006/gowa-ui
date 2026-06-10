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
	"github.com/zerodha/logf"
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
	if !w.app.isReady() {
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
	if !w.app.isReady() {
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

	// Per-instance sweep: each instance gets its own retention.
	rootPath, err := w.app.resolveUploadsRootPath()
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker failed to resolve root path for instance sweep", "error", err)
		return
	}
	instanceDeleted, err := w.deleteExpiredInstanceUploadFiles(ctx, rootPath, now.UTC(), dueOrganizations)
	if err != nil {
		w.app.Log.Error("Uploads cleanup worker instance sweep failed", "error", err)
	} else if instanceDeleted > 0 {
		w.app.Log.Info(
			"Uploads cleanup worker completed instance sweep",
			"deleted_files", instanceDeleted,
		)
	}
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
	if w == nil || !w.app.isReady() {
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
	if w == nil || !w.app.isReady() {
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
	rootPath, err := w.app.resolveUploadsRootPath()
	if err != nil {
		return 0, fmt.Errorf("resolve uploads root: %w", err)
	}

	cutoff := uploadsCutoffTime(now, retentionDays)
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

// walkOptions configures a file-deletion sweep. When DB is non-nil, each deleted
// file is synced to the database (marking media_deleted_at and removing media_assets).
// When InstanceID is also set, DB queries are scoped to that specific instance.
type walkOptions struct {
	RootPath   string
	DirPath    string
	Cutoff     time.Time
	DB         *gorm.DB       // nil = disk-only deletion
	Log        logf.Logger    // zero-value = no logging
	InstanceID *uuid.UUID     // nil = org-level DB sync; set = instance-scoped
}

func walkAndDeleteExpiredFiles(opts walkOptions) (int, error) {
	info, err := os.Stat(opts.DirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("stat uploads directory %q: %w", opts.DirPath, err)
	}
	if !info.IsDir() {
		return 0, nil
	}

	deletedFiles := 0
	err = filepath.WalkDir(opts.DirPath, func(path string, entry fs.DirEntry, walkErr error) error {
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
		if fileInfo.ModTime().After(opts.Cutoff) {
			return nil
		}

		relPath, err := filepath.Rel(opts.RootPath, path)
		if err != nil {
			return fmt.Errorf("resolve upload relative path %q: %w", path, err)
		}

		if err := syncFileDeletion(opts, path, relPath); err != nil {
			return err
		}

		deletedFiles++
		opts.Log.Info("Uploads cleanup deleted file", "path", path)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deletedFiles, nil
}

// syncFileDeletion handles DB sync + disk removal for a single expired file.
func syncFileDeletion(opts walkOptions, path, relPath string) error {
	if opts.DB == nil {
		return os.Remove(path)
	}

	now := time.Now().UTC()
	return opts.DB.Transaction(func(tx *gorm.DB) error {
		// Mark legacy local file messages as deleted, optionally scoped to instance.
		msgQuery := tx.Model(&models.Message{}).
			Where("media_url = ? AND media_deleted_at IS NULL", relPath)
		if opts.InstanceID != nil {
			msgQuery = msgQuery.Where("instance_id = ?", *opts.InstanceID)
		}
		if err := msgQuery.Updates(map[string]any{
			"media_deleted_at": now,
			"updated_at":       now,
		}).Error; err != nil {
			return fmt.Errorf("update legacy local media_deleted_at: %w", err)
		}

		// Mark object-storage assets with matching S3Key as deleted.
		var asset models.MediaAsset
		if err := tx.Where("s3_key = ?", relPath).First(&asset).Error; err == nil {
			assetMsgQuery := tx.Model(&models.Message{}).
				Where("media_asset_id = ? AND media_deleted_at IS NULL", asset.ID)
			if opts.InstanceID != nil {
				assetMsgQuery = assetMsgQuery.Where("instance_id = ?", *opts.InstanceID)
			}
			if err := assetMsgQuery.Updates(map[string]any{
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
}

func (w *UploadsCleanupWorker) deleteExpiredFilesFromDir(rootPath, dirPath string, cutoff time.Time, retentionDays int) (int, error) {
	return walkAndDeleteExpiredFiles(walkOptions{
		RootPath: rootPath,
		DirPath:  dirPath,
		Cutoff:   cutoff,
		DB:       w.app.DB,
		Log:      w.app.Log,
	})
}
