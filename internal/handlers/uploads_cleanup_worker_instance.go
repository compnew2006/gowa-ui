package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func RunManualCleanupForInstance(ctx context.Context, app *App, orgID, instanceID uuid.UUID, retentionDays *int) (int, error) {
	if !app.isReady() {
		return 0, nil
	}

	days := 0
	if retentionDays != nil {
		days = *retentionDays
	}
	if days <= 0 {
		return 0, fmt.Errorf("uploads cleanup disabled for this instance")
	}

	rootPath, err := app.resolveUploadsRootPath()
	if err != nil {
		return 0, fmt.Errorf("resolve uploads root: %w", err)
	}

	cutoff := uploadsCutoffTime(time.Now().UTC(), days)
	deletedFiles := 0

	orgDir := filepath.Join(rootPath, "orgs", orgID.String())
	for _, relativeDir := range uploadsCleanupTargetDirs {
		instanceDir := filepath.Join(orgDir, relativeDir, instanceID.String())
		count, err := walkAndDeleteExpiredFiles(walkOptions{
			RootPath:   rootPath,
			DirPath:    instanceDir,
			Cutoff:     cutoff,
			DB:         app.DB,
			Log:        app.Log,
			InstanceID: &instanceID,
		})
		if err != nil {
			return deletedFiles, err
		}
		deletedFiles += count
	}

	return deletedFiles, nil
}

// deleteExpiredInstanceUploadFiles sweeps per-instance upload directories for all due orgs.
// Each instance's own retention setting is resolved individually.
func (w *UploadsCleanupWorker) deleteExpiredInstanceUploadFiles(ctx context.Context, rootPath string, now time.Time, dueSchedules []uploadsCleanupSchedule) (int, error) {
	totalDeleted := 0

	for _, schedule := range dueSchedules {
		var instances []models.WhatsAppInstance
		if err := w.app.DB.WithContext(ctx).
			Where("organization_id = ?", schedule.OrganizationID).
			Find(&instances).Error; err != nil {
			return totalDeleted, fmt.Errorf("load instances for org %s: %w", schedule.OrganizationID, err)
		}

		orgDefault := schedule.RetentionDays
		orgDir := filepath.Join(rootPath, "orgs", schedule.OrganizationID.String())

		for _, inst := range instances {
			days, _ := ResolveInstanceRetention(inst.Settings, orgDefault)
			if days <= 0 {
				continue
			}

			cutoff := uploadsCutoffTime(now, days)
			for _, relativeDir := range uploadsCleanupTargetDirs {
				instanceDir := filepath.Join(orgDir, relativeDir, inst.ID.String())
				count, err := walkAndDeleteExpiredFiles(walkOptions{
					RootPath:   rootPath,
					DirPath:    instanceDir,
					Cutoff:     cutoff,
					DB:         w.app.DB,
					Log:        w.app.Log,
					InstanceID: &inst.ID,
				})
				if err != nil {
					w.app.Log.Error("Instance uploads cleanup failed",
						"org_id", schedule.OrganizationID,
						"instance_id", inst.ID,
						"error", err,
					)
					continue
				}
				totalDeleted += count
			}
		}
	}

	return totalDeleted, nil
}

func (w *UploadsCleanupWorker) deleteExpiredUploadFiles(now time.Time, retentionDays int) (int, error) {
	rootPath, err := w.app.resolveUploadsRootPath()
	if err != nil {
		return 0, fmt.Errorf("resolve uploads root: %w", err)
	}

	rootDeleted, err := w.deleteRootLevelExpiredFiles(rootPath, now, retentionDays)
	if err != nil {
		return 0, err
	}

	// Legacy: sweep all org dirs with the same retention.
	cutoff := uploadsCutoffTime(now, retentionDays)
	orgsRoot := filepath.Join(rootPath, "orgs")
	entries, err := os.ReadDir(orgsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return rootDeleted, nil
		}
		return 0, fmt.Errorf("read organization uploads directory %q: %w", orgsRoot, err)
	}

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
			rootDeleted += count
		}
	}

	return rootDeleted, nil
}
