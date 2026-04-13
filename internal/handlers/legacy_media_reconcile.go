package handlers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const legacyMediaReconcileBatchSize = 500

type LegacyMediaReconcileOptions struct {
	OlderThan time.Duration
	Limit     int
	Apply     bool
}

type LegacyMediaReconcileSummary struct {
	DryRun         bool
	Cutoff         time.Time
	CandidateCount int
	MissingCount   int
	UpdatedCount   int
	SampleIDs      []string
}

type legacyMediaReconcileCandidate struct {
	ID        uuid.UUID `gorm:"column:id"`
	MediaURL  string    `gorm:"column:media_url"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func ReconcileMissingLegacyMedia(
	ctx context.Context,
	db *gorm.DB,
	mediaStoragePath string,
	options LegacyMediaReconcileOptions,
) (LegacyMediaReconcileSummary, error) {
	summary := LegacyMediaReconcileSummary{
		DryRun: true,
	}
	if db == nil {
		return summary, errors.New("db is required")
	}

	basePath, err := filepath.Abs(strings.TrimSpace(mediaStoragePath))
	if err != nil {
		return summary, err
	}

	olderThan := options.OlderThan
	if olderThan <= 0 {
		olderThan = time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	summary.Cutoff = cutoff
	summary.DryRun = !options.Apply

	query := db.WithContext(ctx).
		Model(&models.Message{}).
		Select("id", "media_url", "created_at").
		Where("media_asset_id IS NULL").
		Where("media_deleted_at IS NULL").
		Where("TRIM(COALESCE(media_url, '')) <> ''").
		Where("created_at <= ?", cutoff).
		Order("created_at ASC")
	if options.Limit > 0 {
		query = query.Limit(options.Limit)
	}

	var candidates []legacyMediaReconcileCandidate
	if err := query.Find(&candidates).Error; err != nil {
		return summary, err
	}
	summary.CandidateCount = len(candidates)
	if len(candidates) == 0 {
		return summary, nil
	}

	now := time.Now().UTC()
	pendingIDs := make([]uuid.UUID, 0, legacyMediaReconcileBatchSize)
	flush := func() error {
		if len(pendingIDs) == 0 {
			return nil
		}
		summary.MissingCount += len(pendingIDs)
		if options.Apply {
			result := db.WithContext(ctx).
				Model(&models.Message{}).
				Where("id IN ?", pendingIDs).
				Where("media_deleted_at IS NULL").
				Updates(map[string]any{
					"media_deleted_at": now,
					"updated_at":       now,
				})
			if result.Error != nil {
				return result.Error
			}
			summary.UpdatedCount += int(result.RowsAffected)
		}
		pendingIDs = pendingIDs[:0]
		return nil
	}

	for _, candidate := range candidates {
		missing, resolveErr := isMissingLegacyMediaPath(basePath, candidate.MediaURL)
		if resolveErr != nil || !missing {
			continue
		}
		if len(summary.SampleIDs) < 20 {
			summary.SampleIDs = append(summary.SampleIDs, candidate.ID.String())
		}
		pendingIDs = append(pendingIDs, candidate.ID)
		if len(pendingIDs) >= legacyMediaReconcileBatchSize {
			if err := flush(); err != nil {
				return summary, err
			}
		}
	}

	if err := flush(); err != nil {
		return summary, err
	}

	return summary, nil
}

func isMissingLegacyMediaPath(basePath, relativePath string) (bool, error) {
	fullPath, err := resolveLegacyMediaPath(basePath, relativePath)
	if err != nil {
		return true, err
	}
	_, statErr := os.Stat(fullPath)
	if statErr == nil {
		return false, nil
	}
	if errors.Is(statErr, os.ErrNotExist) {
		return true, nil
	}
	return false, statErr
}

func resolveLegacyMediaPath(basePath, relativePath string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(relativePath))
	if cleanPath == "" || cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid media file path")
	}

	fullPath, err := filepath.Abs(filepath.Join(basePath, cleanPath))
	if err != nil {
		return "", err
	}
	if fullPath != basePath && !strings.HasPrefix(fullPath, basePath+string(os.PathSeparator)) {
		return "", errors.New("invalid media file path")
	}

	return fullPath, nil
}
