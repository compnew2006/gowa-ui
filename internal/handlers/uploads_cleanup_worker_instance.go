package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
			continue
		}
		count, err := deleteExpiredFilesFromDirStatic(rootPath, instanceDir, cutoff)
		if err != nil {
			return deletedFiles, err
		}
		deletedFiles += count
	}

	return deletedFiles, nil
}

func deleteExpiredFilesFromDirStatic(rootPath, dirPath string, cutoff time.Time) (int, error) {
	deletedFiles := 0
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read directory %q: %w", dirPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dirPath, entry.Name())
			count, err := deleteExpiredFilesFromDirStatic(rootPath, subDir, cutoff)
			if err != nil {
				return deletedFiles, err
			}
			deletedFiles += count
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			fullPath := filepath.Join(dirPath, entry.Name())
			if err := os.Remove(fullPath); err != nil {
				continue
			}
			deletedFiles++
		}
	}
	return deletedFiles, nil
}
