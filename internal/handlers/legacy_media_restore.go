package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	legacyMediaRecoveryProviderKey  = "legacy_media_recovery_provider"
	legacyMediaRecoveryMediaIDKey   = "legacy_media_recovery_media_id"
	legacyMediaRecoveryPhoneIDKey   = "legacy_media_recovery_phone_id"
	legacyMediaRecoveryExpiresAtKey = "legacy_media_recovery_expires_at"
	legacyMediaRestoredAtKey        = "legacy_media_restored_at"

	legacyMediaRecoveryProviderMeta = "meta"
	legacyMediaRecoveryTTL          = 30 * 24 * time.Hour

	legacyMediaRestoreMaxConcurrent = 4
	legacyMediaRestoreWaitTimeout   = 15 * time.Second
	legacyMediaRestoreWaitPoll      = 150 * time.Millisecond
)

var (
	errLegacyMediaRestoreNotNeeded     = errors.New("legacy media restore not needed")
	errLegacyMediaRestoreWaitingTimed  = errors.New("legacy media restore wait timed out")
	errLegacyMediaRestoreInvalidMedia  = errors.New("legacy media restore invalid media payload")
	errLegacyMediaRestoreTransaction   = errors.New("legacy media restore transaction failed")
	errLegacyMediaRestoreAccountLookup = errors.New("legacy media restore account lookup failed")
)

type legacyMediaRestoreMetrics struct {
	attempts        atomic.Uint64
	successes       atomic.Uint64
	failures        atomic.Uint64
	inFlightHits    atomic.Uint64
	budgetExhausted atomic.Uint64
	ttlExpiredSkips atomic.Uint64
}

type legacyMediaRecoveryInfo struct {
	Provider  string
	MediaID   string
	PhoneID   string
	ExpiresAt time.Time
}

type legacyMediaRestoreResult struct {
	Message   models.Message
	Available bool
	Restored  bool
	Reason    string
}

type savedLegacyMedia struct {
	RelativePath string
	MIMEType     string
	Filename     string
	Size         int64
}

func (a *App) maybeRestoreLegacyMedia(ctx context.Context, db *gorm.DB, message *models.Message) (*models.Message, bool) {
	if a == nil || db == nil || message == nil {
		return message, false
	}

	recoveryInfo, eligible, reason := inspectLegacyMediaRecovery(message, time.Now().UTC())
	if !eligible {
		if reason == "ttl_expired" {
			a.legacyMediaRestoreMetrics.ttlExpiredSkips.Add(1)
			a.Log.Info("Skipping legacy media restore because recovery TTL expired", "message_id", message.ID, "organization_id", message.OrganizationID)
		}
		return message, false
	}

	resultAny, err, shared := a.legacyMediaRestoreGroup.Do(message.ID.String(), func() (any, error) {
		a.legacyMediaRestoreMetrics.attempts.Add(1)
		return a.restoreLegacyMediaCoordinated(ctx, db, *message, recoveryInfo)
	})
	if shared {
		a.legacyMediaRestoreMetrics.inFlightHits.Add(1)
	}
	if err != nil {
		a.legacyMediaRestoreMetrics.failures.Add(1)
		a.Log.Error("Legacy media restore failed", "message_id", message.ID, "organization_id", message.OrganizationID, "error", err)
		return message, false
	}

	result, ok := resultAny.(legacyMediaRestoreResult)
	if !ok {
		return message, false
	}
	if result.Restored {
		a.legacyMediaRestoreMetrics.successes.Add(1)
	}
	return &result.Message, result.Available
}

func (a *App) restoreLegacyMediaCoordinated(
	ctx context.Context,
	db *gorm.DB,
	message models.Message,
	recoveryInfo legacyMediaRecoveryInfo,
) (legacyMediaRestoreResult, error) {
	result := legacyMediaRestoreResult{
		Message: message,
		Reason:  "missing",
	}

	if exists, err := a.legacyMediaFileExists(message.MediaURL); err == nil && exists {
		result.Available = true
		result.Reason = "already_present"
		return result, nil
	}

	if db.Dialector.Name() == "postgres" {
		tx := db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return result, fmt.Errorf("%w: %v", errLegacyMediaRestoreTransaction, tx.Error)
		}

		acquired, err := tryAcquireLegacyMediaRestoreTxLock(tx, message.ID)
		if err != nil {
			_ = tx.Rollback()
			return result, err
		}
		if !acquired {
			_ = tx.Rollback()
			waited, waitErr := a.waitForLegacyMediaAvailability(ctx, db, message.ID)
			if waitErr == nil {
				return waited, nil
			}
			if errors.Is(waitErr, errLegacyMediaRestoreWaitingTimed) {
				return result, nil
			}
			return result, waitErr
		}

		if exists, err := a.legacyMediaFileExists(message.MediaURL); err == nil && exists {
			if err := tx.Commit().Error; err != nil {
				return result, fmt.Errorf("%w: %v", errLegacyMediaRestoreTransaction, err)
			}
			result.Available = true
			result.Reason = "already_present"
			return result, nil
		}

		if !a.acquireLegacyMediaRestoreSlot() {
			a.legacyMediaRestoreMetrics.budgetExhausted.Add(1)
			a.Log.Warn("Skipping legacy media restore because restore budget is exhausted", "message_id", message.ID, "organization_id", message.OrganizationID)
			_ = tx.Rollback()
			return result, nil
		}
		defer a.releaseLegacyMediaRestoreSlot()

		restoredMessage, savedFile, err := a.performLegacyMediaRestore(ctx, tx, message, recoveryInfo)
		if err != nil {
			_ = tx.Rollback()
			if savedFile != nil {
				a.removeLegacyMediaFile(savedFile.RelativePath)
			}
			if errors.Is(err, errLegacyMediaRestoreNotNeeded) {
				return a.reloadLegacyMediaRestoreResult(ctx, db, message.ID)
			}
			return result, err
		}
		if err := tx.Commit().Error; err != nil {
			if savedFile != nil {
				a.removeLegacyMediaFile(savedFile.RelativePath)
			}
			return result, fmt.Errorf("%w: %v", errLegacyMediaRestoreTransaction, err)
		}

		result.Message = *restoredMessage
		result.Available = true
		result.Restored = true
		result.Reason = "restored"
		a.broadcastLegacyMediaUpdated(restoredMessage)
		a.Log.Info("Legacy media restored successfully", "message_id", restoredMessage.ID, "organization_id", restoredMessage.OrganizationID, "path", restoredMessage.MediaURL, "mime_type", restoredMessage.MediaMimeType)
		return result, nil
	}

	if !a.acquireLegacyMediaRestoreSlot() {
		a.legacyMediaRestoreMetrics.budgetExhausted.Add(1)
		a.Log.Warn("Skipping legacy media restore because restore budget is exhausted", "message_id", message.ID, "organization_id", message.OrganizationID)
		return result, nil
	}
	defer a.releaseLegacyMediaRestoreSlot()

	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		restoredMessage, savedFile, err := a.performLegacyMediaRestore(ctx, tx, message, recoveryInfo)
		if err != nil {
			if savedFile != nil {
				a.removeLegacyMediaFile(savedFile.RelativePath)
			}
			if errors.Is(err, errLegacyMediaRestoreNotNeeded) {
				reloaded, reloadErr := a.reloadLegacyMediaRestoreResult(ctx, db, message.ID)
				if reloadErr == nil {
					result = reloaded
					return nil
				}
			}
			return err
		}
		result.Message = *restoredMessage
		result.Available = true
		result.Restored = true
		result.Reason = "restored"
		return nil
	})
	if txErr != nil {
		return result, txErr
	}

	a.broadcastLegacyMediaUpdated(&result.Message)
	a.Log.Info("Legacy media restored successfully", "message_id", result.Message.ID, "organization_id", result.Message.OrganizationID, "path", result.Message.MediaURL, "mime_type", result.Message.MediaMimeType)
	return result, nil
}

func (a *App) reloadLegacyMediaRestoreResult(ctx context.Context, db *gorm.DB, messageID uuid.UUID) (legacyMediaRestoreResult, error) {
	var current models.Message
	if err := db.WithContext(ctx).
		Where("id = ?", messageID).
		First(&current).Error; err != nil {
		return legacyMediaRestoreResult{}, err
	}

	available := false
	if exists, err := a.legacyMediaFileExists(current.MediaURL); err == nil && exists {
		available = true
	}

	reason := "missing"
	switch {
	case current.MediaDeletedAt != nil:
		reason = "expired"
	case available:
		reason = "already_present"
	}

	return legacyMediaRestoreResult{
		Message:   current,
		Available: available,
		Reason:    reason,
	}, nil
}

func (a *App) performLegacyMediaRestore(
	ctx context.Context,
	tx *gorm.DB,
	message models.Message,
	recoveryInfo legacyMediaRecoveryInfo,
) (*models.Message, *savedLegacyMedia, error) {
	account, err := a.resolveLegacyMediaRecoveryAccount(message.OrganizationID, message.WhatsAppAccount, recoveryInfo.PhoneID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", errLegacyMediaRestoreAccountLookup, err)
	}

	savedFile, err := a.downloadAndSaveLegacyMedia(
		ctx,
		recoveryInfo.MediaID,
		message.MediaMimeType,
		message.MessageType,
		legacyMediaFilenameHint(&message),
		a.toWhatsAppAccount(account),
	)
	if err != nil {
		return nil, nil, err
	}

	refreshed, err := a.persistLegacyMediaRestore(tx, &message, savedFile)
	if err != nil {
		return nil, savedFile, err
	}

	return refreshed, savedFile, nil
}

func (a *App) persistLegacyMediaRestore(tx *gorm.DB, message *models.Message, savedFile *savedLegacyMedia) (*models.Message, error) {
	if tx == nil || message == nil || savedFile == nil {
		return nil, errors.New("legacy media restore persistence requires transaction, message, and saved file")
	}

	var current models.Message
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND organization_id = ?", message.ID, message.OrganizationID).
		First(&current).Error; err != nil {
		return nil, err
	}
	if current.MediaDeletedAt != nil {
		return nil, errLegacyMediaRestoreNotNeeded
	}
	if current.MediaAssetID != nil {
		return nil, errLegacyMediaRestoreNotNeeded
	}
	if exists, err := a.legacyMediaFileExists(current.MediaURL); err == nil && exists {
		return &current, errLegacyMediaRestoreNotNeeded
	}

	nextMetadata := cloneMessageMetadata(current.Metadata)
	if nextMetadata == nil {
		nextMetadata = models.JSONB{}
	}
	now := time.Now().UTC()
	nextMetadata[legacyMediaRestoredAtKey] = now.Format(time.RFC3339Nano)

	nextFilename := coalesceLegacyMediaValue(savedFile.Filename, current.MediaFilename)
	updates := map[string]any{
		"media_url":       savedFile.RelativePath,
		"media_mime_type": coalesceLegacyMediaValue(savedFile.MIMEType, current.MediaMimeType),
		"media_filename":  nextFilename,
		"metadata":        nextMetadata,
		"updated_at":      now,
	}
	if err := tx.Model(&models.Message{}).
		Where("id = ? AND organization_id = ?", current.ID, current.OrganizationID).
		Updates(updates).Error; err != nil {
		return nil, err
	}

	current.MediaURL = savedFile.RelativePath
	current.MediaMimeType = coalesceLegacyMediaValue(savedFile.MIMEType, current.MediaMimeType)
	current.MediaFilename = nextFilename
	current.Metadata = nextMetadata
	current.UpdatedAt = now
	return &current, nil
}

func (a *App) waitForLegacyMediaAvailability(
	ctx context.Context,
	db *gorm.DB,
	messageID uuid.UUID,
) (legacyMediaRestoreResult, error) {
	deadline := time.Now().Add(legacyMediaRestoreWaitTimeout)
	ticker := time.NewTicker(legacyMediaRestoreWaitPoll)
	defer ticker.Stop()

	for {
		var current models.Message
		if err := db.WithContext(ctx).
			Where("id = ?", messageID).
			First(&current).Error; err != nil {
			return legacyMediaRestoreResult{}, err
		}

		if current.MediaDeletedAt != nil {
			return legacyMediaRestoreResult{Message: current, Reason: "expired"}, nil
		}
		if exists, err := a.legacyMediaFileExists(current.MediaURL); err == nil && exists {
			return legacyMediaRestoreResult{
				Message:   current,
				Available: true,
				Reason:    "wait_completed",
			}, nil
		}

		if time.Now().After(deadline) {
			return legacyMediaRestoreResult{
				Message: current,
				Reason:  "wait_timeout",
			}, errLegacyMediaRestoreWaitingTimed
		}

		select {
		case <-ctx.Done():
			return legacyMediaRestoreResult{Message: current}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *App) resolveLegacyMediaRecoveryAccount(orgID uuid.UUID, fallbackAccountName, phoneID string) (*models.WhatsAppAccount, error) {
	phoneID = strings.TrimSpace(phoneID)
	if phoneID != "" {
		account, err := a.getWhatsAppAccountCached(phoneID)
		if err == nil && account != nil && account.OrganizationID == orgID {
			return account, nil
		}
	}
	return a.resolveWhatsAppAccount(orgID, fallbackAccountName)
}

func (a *App) legacyMediaRestoreLimiterChan() chan struct{} {
	if a == nil {
		return nil
	}
	a.legacyMediaRestoreLimiterOnce.Do(func() {
		a.legacyMediaRestoreLimiter = make(chan struct{}, legacyMediaRestoreMaxConcurrent)
	})
	return a.legacyMediaRestoreLimiter
}

func (a *App) acquireLegacyMediaRestoreSlot() bool {
	limiter := a.legacyMediaRestoreLimiterChan()
	if limiter == nil {
		return false
	}
	select {
	case limiter <- struct{}{}:
		return true
	default:
		return false
	}
}

func (a *App) releaseLegacyMediaRestoreSlot() {
	limiter := a.legacyMediaRestoreLimiterChan()
	if limiter == nil {
		return
	}
	select {
	case <-limiter:
	default:
	}
}

func tryAcquireLegacyMediaRestoreTxLock(tx *gorm.DB, messageID uuid.UUID) (bool, error) {
	var acquired bool
	if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", legacyMediaRestoreLockKey(messageID)).
		Scan(&acquired).Error; err != nil {
		return false, err
	}
	return acquired, nil
}

func legacyMediaRestoreLockKey(messageID uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(messageID[:8]))
}

func inspectLegacyMediaRecovery(message *models.Message, now time.Time) (legacyMediaRecoveryInfo, bool, string) {
	if message == nil || message.Metadata == nil {
		return legacyMediaRecoveryInfo{}, false, "missing_metadata"
	}

	provider := strings.ToLower(strings.TrimSpace(getStringFromMap(message.Metadata, legacyMediaRecoveryProviderKey)))
	mediaID := strings.TrimSpace(getStringFromMap(message.Metadata, legacyMediaRecoveryMediaIDKey))
	if provider != legacyMediaRecoveryProviderMeta || mediaID == "" {
		return legacyMediaRecoveryInfo{}, false, "missing_metadata"
	}

	info := legacyMediaRecoveryInfo{
		Provider: provider,
		MediaID:  mediaID,
		PhoneID:  strings.TrimSpace(getStringFromMap(message.Metadata, legacyMediaRecoveryPhoneIDKey)),
	}
	info.ExpiresAt = legacyMediaRecoveryExpiresAt(message)
	if !info.ExpiresAt.IsZero() && now.After(info.ExpiresAt) {
		return info, false, "ttl_expired"
	}

	return info, true, "eligible"
}

func legacyMediaRecoveryExpiresAt(message *models.Message) time.Time {
	if message == nil || message.Metadata == nil {
		return time.Time{}
	}
	if raw, ok := message.Metadata[legacyMediaRecoveryExpiresAtKey]; ok {
		switch typed := raw.(type) {
		case string:
			if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(typed)); err == nil {
				return parsed
			}
			if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(typed)); err == nil {
				return parsed
			}
		case time.Time:
			return typed
		}
	}

	provider := strings.ToLower(strings.TrimSpace(getStringFromMap(message.Metadata, legacyMediaRecoveryProviderKey)))
	mediaID := strings.TrimSpace(getStringFromMap(message.Metadata, legacyMediaRecoveryMediaIDKey))
	if provider == legacyMediaRecoveryProviderMeta && mediaID != "" {
		return message.CreatedAt.UTC().Add(legacyMediaRecoveryTTL)
	}
	return time.Time{}
}

func cloneMessageMetadata(src models.JSONB) models.JSONB {
	if src == nil {
		return nil
	}
	dst := make(models.JSONB, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func legacyMediaFilenameHint(message *models.Message) string {
	if message == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(message.MediaFilename); trimmed != "" {
		return filepath.Base(trimmed)
	}
	if trimmed := strings.TrimSpace(message.MediaURL); trimmed != "" {
		return filepath.Base(trimmed)
	}
	return ""
}

func (a *App) downloadAndSaveLegacyMedia(
	ctx context.Context,
	mediaID,
	metaMimeType string,
	messageType models.MessageType,
	filenameHint string,
	account *whatsapp.Account,
) (*savedLegacyMedia, error) {
	if a == nil || a.WhatsApp == nil || account == nil {
		return nil, fmt.Errorf("%w: media restore requires whatsapp client and account", errLegacyMediaRestoreInvalidMedia)
	}

	mediaURL, err := a.WhatsApp.GetMediaURL(ctx, mediaID, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get media URL: %w", err)
	}

	data, err := a.WhatsApp.DownloadMedia(ctx, mediaURL, account.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: media download was empty", errLegacyMediaRestoreInvalidMedia)
	}

	resolvedMimeType := resolveLegacyMediaMimeType(metaMimeType, data)
	if !legacyMediaMimeMatchesMessageType(messageType, resolvedMimeType) {
		return nil, fmt.Errorf("%w: restored MIME %q does not match message type %q", errLegacyMediaRestoreInvalidMedia, resolvedMimeType, messageType)
	}
	if err := validateDownloadedLegacyMedia(data, resolvedMimeType, messageType); err != nil {
		return nil, err
	}

	relativePath, err := a.writeLegacyMediaData(data, resolvedMimeType)
	if err != nil {
		return nil, err
	}

	return &savedLegacyMedia{
		RelativePath: relativePath,
		MIMEType:     resolvedMimeType,
		Filename:     normalizeLegacyMediaFilename(filenameHint, resolvedMimeType),
		Size:         int64(len(data)),
	}, nil
}

func resolveLegacyMediaMimeType(metaMimeType string, data []byte) string {
	meta := normalizeLegacyMediaMime(metaMimeType)
	sniffed := normalizeLegacyMediaMime(http.DetectContentType(data[:minInt(len(data), 512)]))
	switch {
	case meta != "" && meta != "application/octet-stream" && sniffed == "":
		return meta
	case meta != "" && meta != "application/octet-stream" && (sniffed == "" || sniffed == "application/octet-stream" || legacyMediaSameMIMEFamily(meta, sniffed)):
		return meta
	case sniffed != "" && sniffed != "application/octet-stream":
		return sniffed
	case meta != "":
		return meta
	default:
		return "application/octet-stream"
	}
}

func normalizeLegacyMediaMime(mimeType string) string {
	trimmed := strings.TrimSpace(strings.ToLower(mimeType))
	if trimmed == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(trimmed, ";")[0])
}

func legacyMediaSameMIMEFamily(a, b string) bool {
	a = normalizeLegacyMediaMime(a)
	b = normalizeLegacyMediaMime(b)
	if a == "" || b == "" {
		return false
	}
	aPrefix, _, aFound := strings.Cut(a, "/")
	bPrefix, _, bFound := strings.Cut(b, "/")
	return aFound && bFound && aPrefix == bPrefix
}

func legacyMediaMimeMatchesMessageType(messageType models.MessageType, mimeType string) bool {
	mimeType = normalizeLegacyMediaMime(mimeType)
	switch messageType {
	case models.MessageTypeImage:
		return strings.HasPrefix(mimeType, "image/")
	case models.MessageTypeSticker:
		return strings.HasPrefix(mimeType, "image/")
	case models.MessageTypeVideo:
		return strings.HasPrefix(mimeType, "video/")
	case models.MessageTypeAudio:
		return strings.HasPrefix(mimeType, "audio/")
	case models.MessageTypeDocument:
		return !strings.HasPrefix(mimeType, "image/") && !strings.HasPrefix(mimeType, "video/") && !strings.HasPrefix(mimeType, "audio/")
	default:
		return true
	}
}

func validateDownloadedLegacyMedia(data []byte, mimeType string, messageType models.MessageType) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: media download was empty", errLegacyMediaRestoreInvalidMedia)
	}

	switch {
	case messageType == models.MessageTypeSticker || mimeType == "image/webp":
		if !isValidWEBP(data) {
			return fmt.Errorf("%w: invalid WEBP payload", errLegacyMediaRestoreInvalidMedia)
		}
		return nil
	case strings.HasPrefix(mimeType, "image/"):
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("%w: invalid image payload", errLegacyMediaRestoreInvalidMedia)
		}
		return nil
	case mimeType == "application/pdf":
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return fmt.Errorf("%w: invalid PDF payload", errLegacyMediaRestoreInvalidMedia)
		}
		return nil
	default:
		return nil
	}
}

func isValidWEBP(data []byte) bool {
	return len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

func (a *App) writeLegacyMediaData(data []byte, mimeType string) (string, error) {
	subdir := legacyMediaSubdirForMime(mimeType)
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}

	dirPath := filepath.Join(a.getMediaStoragePath(), subdir)
	tempFile, err := os.CreateTemp(dirPath, ".restore-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp media file: %w", err)
	}

	tempPath := tempFile.Name()
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return "", fmt.Errorf("failed to write temp media file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp media file: %w", err)
	}

	filename := uuid.New().String() + ext
	finalPath := filepath.Join(dirPath, filename)
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", fmt.Errorf("failed to finalize media file: %w", err)
	}
	cleanupTemp = false

	relativePath := filepath.Join(subdir, filename)
	a.Log.Info("Media saved", "path", relativePath, "size", len(data))
	return relativePath, nil
}

func legacyMediaSubdirForMime(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "images"
	case strings.HasPrefix(mimeType, "video/"):
		return "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "documents"
	}
}

func legacyMediaMessageTypeFromMIME(mimeType string) models.MessageType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return models.MessageTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return models.MessageTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return models.MessageTypeAudio
	default:
		return models.MessageTypeDocument
	}
}

func normalizeLegacyMediaFilename(filenameHint, mimeType string) string {
	filenameHint = filepath.Base(strings.TrimSpace(filenameHint))
	if filenameHint == "" {
		return ""
	}

	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		return filenameHint
	}

	currentExt := filepath.Ext(filenameHint)
	if strings.EqualFold(currentExt, ext) {
		return filenameHint
	}
	if currentExt == "" {
		return filenameHint + ext
	}

	base := strings.TrimSuffix(filenameHint, currentExt)
	if base == "" {
		base = "file"
	}
	return base + ext
}

func (a *App) legacyMediaFileExists(relativePath string) (bool, error) {
	if strings.TrimSpace(relativePath) == "" {
		return false, nil
	}

	fullPath, err := resolveLegacyMediaPath(a.getMediaStoragePath(), relativePath)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(fullPath)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (a *App) removeLegacyMediaFile(relativePath string) {
	fullPath, err := resolveLegacyMediaPath(a.getMediaStoragePath(), relativePath)
	if err != nil {
		return
	}
	if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		a.Log.Warn("Failed to remove restored legacy media file during rollback", "path", fullPath, "error", removeErr)
	}
}

func (a *App) broadcastLegacyMediaUpdated(message *models.Message) {
	if a == nil || a.WSHub == nil || message == nil {
		return
	}

	a.WSHub.BroadcastToOrg(message.OrganizationID, websocket.WSMessage{
		Type: websocket.TypeMessageMediaUpdated,
		Payload: map[string]any{
			"id":              message.ID,
			"contact_id":      message.ContactID.String(),
			"media_url":       message.MediaURL,
			"media_mime_type": message.MediaMimeType,
			"media_filename":  message.MediaFilename,
			"error_message":   message.ErrorMessage,
			"metadata":        message.Metadata,
			"updated_at":      message.UpdatedAt,
		},
	})
}

func coalesceLegacyMediaValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
