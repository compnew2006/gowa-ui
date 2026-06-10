package handlers

import (
	"context"
	"fmt"
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
