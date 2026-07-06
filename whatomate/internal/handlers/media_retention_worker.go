package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	mediaRetentionWorkerLockKey int64 = 2026041005
	mediaRetentionDeletedNote         = "*(Media deleted automatically to save space)*"
)

type mediaRetentionCandidate struct {
	ID                   uuid.UUID    `gorm:"column:id"`
	OrganizationID       uuid.UUID    `gorm:"column:organization_id"`
	MediaAssetID         uuid.UUID    `gorm:"column:media_asset_id"`
	Content              string       `gorm:"column:content"`
	CreatedAt            time.Time    `gorm:"column:created_at"`
	OrganizationSettings models.JSONB `gorm:"column:organization_settings"`
}

// MediaRetentionWorker deletes expired inbound media objects based on organization policy tiers.
type MediaRetentionWorker struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

func NewMediaRetentionWorker(app *App, interval time.Duration) *MediaRetentionWorker {
	return &MediaRetentionWorker{
		app:      app,
		interval: interval,
	}
}

func (w *MediaRetentionWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.ticker = time.NewTicker(w.interval)
	ticker := w.ticker
	w.mu.Unlock()
	defer ticker.Stop()

	w.runOnce(ctx, time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			w.runOnce(ctx, tick.UTC())
		}
	}
}

func (w *MediaRetentionWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

func (w *MediaRetentionWorker) runOnce(ctx context.Context, now time.Time) {
	if w.app == nil || w.app.DB == nil || w.app.ObjectStorage == nil || !w.app.isWhatsmeowProvider() {
		return
	}

	acquired, err := w.tryAcquireLock(ctx)
	if err != nil {
		w.app.Log.Error("Media retention worker failed to acquire advisory lock", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer w.releaseLock(context.Background())

	if err := w.sweepExpiredMedia(ctx, now); err != nil {
		w.app.Log.Error("Media retention worker sweep failed", "error", err)
	}
}

func (w *MediaRetentionWorker) tryAcquireLock(ctx context.Context) (bool, error) {
	var acquired bool
	if err := w.app.DB.WithContext(ctx).
		Raw("SELECT pg_try_advisory_lock(?)", mediaRetentionWorkerLockKey).
		Scan(&acquired).Error; err != nil {
		return false, err
	}
	return acquired, nil
}

func (w *MediaRetentionWorker) releaseLock(ctx context.Context) {
	if w == nil || w.app == nil || w.app.DB == nil {
		return
	}
	if err := w.app.DB.WithContext(ctx).
		Exec("SELECT pg_advisory_unlock(?)", mediaRetentionWorkerLockKey).
		Error; err != nil {
		w.app.Log.Warn("Media retention worker failed to release advisory lock", "error", err)
	}
}

func (w *MediaRetentionWorker) sweepExpiredMedia(ctx context.Context, now time.Time) error {
	candidates, err := w.loadCandidates(ctx)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	touchedAssets := make(map[uuid.UUID]struct{})
	for _, candidate := range candidates {
		if !isMediaRetentionExpired(candidate.CreatedAt, mediaRetentionDays(candidate.OrganizationSettings), now) {
			continue
		}
		if err := w.markMessageMediaExpired(ctx, candidate, now); err != nil {
			return err
		}
		touchedAssets[candidate.MediaAssetID] = struct{}{}
	}

	for assetID := range touchedAssets {
		if err := w.deleteAssetIfUnused(ctx, assetID, now); err != nil {
			return err
		}
	}

	return nil
}

func (w *MediaRetentionWorker) loadCandidates(ctx context.Context) ([]mediaRetentionCandidate, error) {
	var candidates []mediaRetentionCandidate
	err := w.app.DB.WithContext(ctx).
		Table("messages").
		Select(
			"messages.id",
			"messages.organization_id",
			"messages.media_asset_id",
			"messages.content",
			"messages.created_at",
			"organizations.settings AS organization_settings",
		).
		Joins("JOIN organizations ON organizations.id = messages.organization_id").
		Where("messages.media_asset_id IS NOT NULL").
		Where("messages.media_deleted_at IS NULL").
		Find(&candidates).Error
	return candidates, err
}

func (w *MediaRetentionWorker) markMessageMediaExpired(ctx context.Context, candidate mediaRetentionCandidate, now time.Time) error {
	updates := map[string]any{
		"media_deleted_at": now,
		"updated_at":       now,
	}
	nextContent := appendMediaRetentionNote(candidate.Content)
	if nextContent != candidate.Content {
		updates["content"] = nextContent
	}

	return w.app.DB.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ? AND media_deleted_at IS NULL", candidate.ID).
		Updates(updates).Error
}

func (w *MediaRetentionWorker) deleteAssetIfUnused(ctx context.Context, assetID uuid.UUID, now time.Time) error {
	var activeRefs int64
	if err := w.app.DB.WithContext(ctx).
		Model(&models.Message{}).
		Where("media_asset_id = ? AND media_deleted_at IS NULL", assetID).
		Count(&activeRefs).Error; err != nil {
		return fmt.Errorf("count active media references: %w", err)
	}
	if activeRefs > 0 {
		return nil
	}

	var asset models.MediaAsset
	if err := w.app.DB.WithContext(ctx).
		Where("id = ?", assetID).
		First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("load media asset: %w", err)
	}

	if err := w.app.ObjectStorage.DeleteObject(ctx, asset.S3Key); err != nil {
		if errors.Is(err, objectstorage.ErrCircuitDeleteOpen) {
			w.app.Log.Warn("MediaRetention: object storage DeleteObject circuit open; deferring delete", "media_asset_id", asset.ID, "s3_key", asset.S3Key, "error", err)
			return nil
		}
		if !errors.Is(err, objectstorage.ErrObjectNotFound) {
			return fmt.Errorf("delete media asset from object storage: %w", err)
		}
	}

	return w.app.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Message{}).
			Where("media_asset_id = ?", asset.ID).
			Updates(map[string]any{
				"media_asset_id": nil,
				"updated_at":     now,
			}).Error; err != nil {
			return fmt.Errorf("null media asset references: %w", err)
		}
		if err := tx.Delete(&models.MediaAsset{}, "id = ?", asset.ID).Error; err != nil {
			return fmt.Errorf("soft delete media asset row: %w", err)
		}
		return nil
	})
}

func mediaRetentionDays(settings models.JSONB) int {
	tierValue, _ := settings["media_retention_tier"].(string)
	switch strings.ToLower(strings.TrimSpace(tierValue)) {
	case "pro":
		return 180
	default:
		return 30
	}
}

func isMediaRetentionExpired(createdAt time.Time, retentionDays int, now time.Time) bool {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	return !createdAt.Add(time.Duration(retentionDays) * 24 * time.Hour).After(now)
}

func appendMediaRetentionNote(content string) string {
	if strings.Contains(content, mediaRetentionDeletedNote) {
		return content
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return mediaRetentionDeletedNote
	}
	return trimmed + "\n\n" + mediaRetentionDeletedNote
}
