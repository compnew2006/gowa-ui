package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

// sweepExpiredUploads is the legacy scheduled sweep entry point.
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
