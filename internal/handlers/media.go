package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/storage"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type safeRequestCtx struct {
	*fasthttp.RequestCtx
}

func (s safeRequestCtx) Done() (ch <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			ch = nil
		}
	}()
	return s.RequestCtx.Done()
}

func (s safeRequestCtx) Err() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = nil
		}
	}()
	return s.RequestCtx.Err()
}

func (s safeRequestCtx) Deadline() (deadline time.Time, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			deadline = time.Time{}
			ok = false
		}
	}()
	return s.RequestCtx.Deadline()
}

func (s safeRequestCtx) Value(key any) (val any) {
	defer func() {
		if r := recover(); r != nil {
			val = nil
		}
	}()
	return s.RequestCtx.Value(key)
}

// getMediaStoragePath returns the base path for media storage
func (a *App) getMediaStoragePath() string {
	basePath := a.Config.Storage.LocalPath
	if basePath == "" {
		basePath = "./media"
	}
	return basePath
}

// ensureMediaDir ensures the media directory exists
func (a *App) ensureMediaDir(subdir string) error {
	path := filepath.Join(a.getMediaStoragePath(), subdir)
	return os.MkdirAll(path, 0750)
}

func organizationMediaSubdir(orgID uuid.UUID, parts ...string) string {
	segments := []string{"orgs", orgID.String()}
	segments = append(segments, parts...)
	return filepath.Join(segments...)
}

// getExtensionFromMimeType returns file extension based on mime type
func getExtensionFromMimeType(mimeType string) string {
	// Prefix matching as implemented before
	switch {
	case strings.HasPrefix(mimeType, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mimeType, "image/png"):
		return ".png"
	case strings.HasPrefix(mimeType, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mimeType, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mimeType, "video/mp4"):
		return ".mp4"
	case strings.HasPrefix(mimeType, "video/3gpp"):
		return ".3gp"
	case strings.HasPrefix(mimeType, "audio/aac"):
		return ".aac"
	case strings.HasPrefix(mimeType, "audio/mp4"):
		return ".m4a"
	case strings.HasPrefix(mimeType, "audio/mpeg"):
		return ".mp3"
	case strings.HasPrefix(mimeType, "audio/amr"):
		return ".amr"
	case strings.HasPrefix(mimeType, "audio/ogg"):
		return ".ogg"
	case strings.HasPrefix(mimeType, "application/pdf"):
		return ".pdf"
	case strings.HasPrefix(mimeType, "application/vnd.ms-powerpoint"):
		return ".ppt"
	case strings.HasPrefix(mimeType, "application/msword"):
		return ".doc"
	case strings.HasPrefix(mimeType, "application/vnd.ms-excel"):
		return ".xls"
	case strings.HasPrefix(mimeType, "application/vnd.openxmlformats-officedocument.wordprocessingml"):
		return ".docx"
	case strings.HasPrefix(mimeType, "application/vnd.openxmlformats-officedocument.spreadsheetml"):
		return ".xlsx"
	case strings.HasPrefix(mimeType, "application/vnd.openxmlformats-officedocument.presentationml"):
		return ".pptx"
	case strings.HasPrefix(mimeType, "text/plain"):
		return ".txt"
	default:
		return ""
	}
}

// getContentTypeFromExt maps a file extension to its corresponding MIME content type.
func getContentTypeFromExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".3gp":
		return "video/3gpp"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".oga":
		return "audio/ogg"
	case ".amr":
		return "audio/amr"
	case ".opus":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/vnd.rar"
	case ".7z":
		return "application/x-7z-compressed"
	case ".txt":
		return "text/plain"
	default:
		return ""
	}
}

// DownloadAndSaveMedia downloads media from Meta and saves it locally
// Returns the local file path (relative to media storage) or error
func (a *App) DownloadAndSaveMedia(ctx context.Context, mediaID string, mimeType string, account *whatsapp.Account) (string, error) {
	savedFile, err := a.downloadAndSaveLegacyMedia(
		ctx,
		mediaID,
		mimeType,
		legacyMediaMessageTypeFromMIME(mimeType),
		"",
		account,
	)
	if err != nil {
		return "", err
	}
	return savedFile.RelativePath, nil
}

// ServeMedia serves media files from local storage
// Only authorized users who have access to the message can view the media
func (a *App) ServeMedia(r *fastglue.Request) error {
	ctx := safeRequestCtx{r.RequestCtx}
	requestDB :=
		// Get auth context
		a.requestDB(r)

	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionRead); err != nil {
		return nil
	}

	// Get the message ID from URL parameter
	messageIDValue := r.RequestCtx.UserValue("message_id")
	messageIDStr, ok := messageIDValue.(string)
	if !ok || messageIDStr == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	var message models.Message
	if err := requestDB.WithContext(ctx).
		Preload("MediaAsset").
		Where("id = ? AND organization_id = ?", messageID, orgID).
		First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	// Active team transfers remain a fallback for the transfer workflow.
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		var contact models.Contact
		contactQuery := requestDB.Where("id = ? AND organization_id = ?", message.ContactID, orgID)
		contactQuery = applyAgentVisibleChatAccessFilter(contactQuery, userID)
		if err := contactQuery.First(&contact).Error; err != nil {
			// Not directly assigned — check team membership via active transfer
			var transfer models.AgentTransfer
			if err := requestDB.Where("contact_id = ? AND organization_id = ? AND status = ? AND team_id IS NOT NULL",
				message.ContactID, orgID, models.TransferStatusActive).First(&transfer).Error; err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
			}
			var count int64
			requestDB.
				Model(&models.TeamMember{}).Where("team_id = ? AND user_id = ?", transfer.TeamID, userID).Count(&count)
			if count == 0 {
				return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
			}
		}
	}

	// Check if message has media
	if message.MediaDeletedAt != nil {
		return r.SendErrorEnvelope(fasthttp.StatusGone, "Media expired", nil, "")
	}
	if message.MediaAssetID == nil || message.MediaAsset == nil || strings.TrimSpace(message.MediaAsset.S3Key) == "" {
		if relativePath := strings.TrimSpace(message.MediaURL); relativePath != "" {
			if missing, missingErr := isMissingLegacyMediaPath(a.getMediaStoragePath(), relativePath); missingErr == nil && missing {
				if restoredMessage, restored := a.maybeRestoreLegacyMedia(ctx, requestDB, &message); restored && restoredMessage != nil {
					message = *restoredMessage
					relativePath = strings.TrimSpace(message.MediaURL)
				} else {
					// File is gone and restore failed — mark media as deleted
					// so the API stops returning media_url for this message.
					_ = requestDB.Model(&models.Message{}).
						Where("id = ? AND media_deleted_at IS NULL", message.ID).
						Update("media_deleted_at", time.Now().UTC()).Error
				}
			}
			if filename := strings.TrimSpace(message.MediaFilename); filename != "" {
				r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
			}
			return a.serveLocalMediaFile(r, relativePath, message.MediaMimeType)
		}
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No media found", nil, "")
	}
	if a.ObjectStorage == nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Media storage is not configured", nil, "")
	}

	reader, objectInfo, err := a.ObjectStorage.GetObject(ctx, message.MediaAsset.S3Key)
	if err != nil {
		if errors.Is(err, storage.ErrCircuitGetOpen) {
			a.Log.Warn("ServeMedia: object storage GetObject circuit open", "media_asset_id", message.MediaAssetID, "error", err)
			r.RequestCtx.Response.Header.Set("Retry-After", "30")
			return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Object storage temporarily unavailable", nil, "")
		}
		if errors.Is(err, storage.ErrObjectNotFound) {
			now := time.Now().UTC()
			// File is missing from Object Storage/local disk — self-heal by updating all matching messages and the asset.
			errDb := requestDB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&models.Message{}).
					Where("media_asset_id = ? AND media_deleted_at IS NULL", message.MediaAssetID).
					Updates(map[string]any{
						"media_deleted_at": now,
						"updated_at":       now,
					}).Error; err != nil {
					return fmt.Errorf("update messages media_deleted_at for asset %s: %w", message.MediaAssetID, err)
				}
				if err := tx.Delete(message.MediaAsset).Error; err != nil {
					return fmt.Errorf("delete media asset %s: %w", message.MediaAssetID, err)
				}
				return nil
			})
			if errDb != nil {
				a.Log.Error("ServeMedia: failed to sync missing media asset state in database", "media_asset_id", message.MediaAssetID, "error", errDb)
			}
		}
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Media not found", nil, "")
	}

	contentType := strings.TrimSpace(message.MediaAsset.MimeType)
	if contentType == "" {
		contentType = strings.TrimSpace(message.MediaMimeType)
	}
	if contentType == "" {
		contentType = strings.TrimSpace(objectInfo.ContentType)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
	r.RequestCtx.Response.Header.SetContentType(contentType)
	if filename := strings.TrimSpace(message.MediaFilename); filename != "" {
		r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	}
	r.RequestCtx.SetBodyStream(reader, int(objectInfo.Size))
	return nil
}

// RetryMediaDownload attempts to re-download media for a message whose file
// is missing. It supports two recovery paths:
//  1. Legacy (Meta Cloud API): re-downloads synchronously from Meta CDN within 30-day TTL.
//  2. Whatsmeow async: re-enqueues the stored protobuf payload for worker-driven recovery.
func (a *App) RetryMediaDownload(r *fastglue.Request) error {
	ctx := safeRequestCtx{r.RequestCtx}
	requestDB := a.requestDB(r)

	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionRead); err != nil {
		return nil
	}

	messageIDValue := r.RequestCtx.UserValue("message_id")
	messageIDStr, ok := messageIDValue.(string)
	if !ok || messageIDStr == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	var message models.Message
	if err := requestDB.WithContext(ctx).
		Preload("MediaAsset").
		Where("id = ? AND organization_id = ?", messageID, orgID).
		First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
		}
		a.Log.Error("Failed to load message for media retry", "message_id", messageID, "organization_id", orgID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load message", nil, "")
	}

	if strings.TrimSpace(message.MediaURL) != "" && message.MediaDeletedAt == nil {
		mediaAvailable := true
		if message.MediaAssetID != nil {
			mediaAvailable = false
			if message.MediaAsset != nil && strings.TrimSpace(message.MediaAsset.S3Key) != "" && a.ObjectStorage != nil {
				reader, _, err := a.ObjectStorage.GetObject(ctx, message.MediaAsset.S3Key)
				if err == nil {
					if reader != nil {
						_ = reader.Close()
					}
					mediaAvailable = true
				} else if errors.Is(err, storage.ErrCircuitGetOpen) {
					a.Log.Warn("RetryMediaDownload: object storage GetObject circuit open; treating as unavailable", "message_id", message.ID, "organization_id", orgID, "error", err)
				} else {
					a.Log.Warn("Object-backed media is missing during retry", "message_id", message.ID, "organization_id", orgID, "media_asset_id", message.MediaAssetID, "s3_key", message.MediaAsset.S3Key, "error", err)
				}
			}
		}
		if mediaAvailable {
			return r.SendEnvelope(map[string]any{
				"media_url":       message.MediaURL,
				"media_filename":  message.MediaFilename,
				"media_mimetype":  message.MediaMimeType,
				"media_mime_type": message.MediaMimeType,
			})
		}
	}

	_, eligible, reason := inspectLegacyMediaRecovery(&message, time.Now().UTC())
	if eligible {
		return a.retryLegacyMediaDownload(r, requestDB, message)
	}

	if a.retryWhatsmeowMediaRecovery(ctx, requestDB, &message) {
		return r.SendEnvelope(map[string]any{
			"status":  "queued",
			"message": "Media recovery re-queued",
		})
	}

	status := fasthttp.StatusGone
	msg := "Media download link has expired"
	if reason == "missing_metadata" {
		status = fasthttp.StatusNotFound
		msg = "No recovery information available for this media"
	}
	return r.SendErrorEnvelope(status, msg, nil, "")
}

func (a *App) retryLegacyMediaDownload(r *fastglue.Request, requestDB *gorm.DB, message models.Message) error {
	ctx := safeRequestCtx{r.RequestCtx}
	result, err, _ := a.legacyMediaRestoreGroup.Do(message.ID.String(), func() (any, error) {
		recoveryInfo, _, _ := inspectLegacyMediaRecovery(&message, time.Now().UTC())
		msg, _, err := a.performLegacyMediaRestore(ctx, requestDB, message, recoveryInfo)
		return msg, err
	})
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to download media", nil, "")
	}

	restoreResult, ok := result.(*models.Message)
	if !ok || restoreResult == nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to restore media", nil, "")
	}

	_ = requestDB.Model(&models.Message{}).
		Where("id = ?", message.ID).
		Update("media_deleted_at", nil).Error

	return r.SendEnvelope(map[string]any{
		"media_url":       restoreResult.MediaURL,
		"media_filename":  restoreResult.MediaFilename,
		"media_mimetype":  restoreResult.MediaMimeType,
		"media_mime_type": restoreResult.MediaMimeType,
	})
}

const inboundMediaAsyncJobMetaKey = "inbound_media_async_job"

func (a *App) retryWhatsmeowMediaRecovery(ctx context.Context, db *gorm.DB, message *models.Message) bool {
	if message == nil || message.Metadata == nil {
		return false
	}

	rawJob, ok := message.Metadata[inboundMediaAsyncJobMetaKey]
	if !ok || rawJob == nil {
		return false
	}

	jobBytes, err := json.Marshal(rawJob)
	if err != nil {
		a.Log.Warn("retryWhatsmeowMediaRecovery: failed to marshal stored job", "message_id", message.ID, "error", err)
		return false
	}

	var job queue.InboundMediaJob
	if err := json.Unmarshal(jobBytes, &job); err != nil {
		a.Log.Warn("retryWhatsmeowMediaRecovery: failed to decode stored job", "message_id", message.ID, "error", err)
		return false
	}

	if strings.TrimSpace(job.MediaPayloadBase64) == "" {
		a.Log.Warn("retryWhatsmeowMediaRecovery: stored job missing payload", "message_id", message.ID)
		return false
	}

	if a.Queue == nil {
		a.Log.Warn("retryWhatsmeowMediaRecovery: job queue not available")
		return false
	}

	job.EnqueuedAt = time.Now().UTC()
	job.LastError = ""

	if err := a.Queue.EnqueueInboundMedia(ctx, &job); err != nil {
		a.Log.Warn("retryWhatsmeowMediaRecovery: failed to enqueue job", "message_id", message.ID, "error", err)
		return false
	}

	nextMetadata := make(models.JSONB, len(message.Metadata))
	for k, v := range message.Metadata {
		nextMetadata[k] = v
	}
	nextMetadata["inbound_media_async_status"] = "queued"
	nextMetadata["inbound_media_async_enqueued_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	delete(nextMetadata, "inbound_media_async_enqueue_error")
	nextMetadata["inbound_media_async_last_error"] = ""

	_ = db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ? AND organization_id = ?", message.ID, message.OrganizationID).
		Updates(map[string]any{
			"media_deleted_at": nil,
			"metadata":         nextMetadata,
			"error_message":    "",
		}).Error

	a.Log.Info("retryWhatsmeowMediaRecovery: re-queued inbound media recovery job",
		"message_id", message.ID,
		"instance_id", job.InstanceID,
	)
	return true
}

func (a *App) serveLocalMediaFile(r *fastglue.Request, relativePath, mimeHint string) error {
	// Security: prevent directory traversal and symlink attacks
	filePath := filepath.Clean(relativePath)
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Storage configuration error", nil, "")
	}
	resolvedBaseDir := baseDir
	if realBaseDir, err := filepath.EvalSymlinks(baseDir); err == nil {
		resolvedBaseDir = realBaseDir
	}
	fullPath, err := filepath.Abs(filepath.Join(resolvedBaseDir, filePath))
	if err != nil || !strings.HasPrefix(fullPath, resolvedBaseDir+string(os.PathSeparator)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}
	resolvedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "File not found", nil, "")
	}
	if !strings.HasPrefix(resolvedPath, resolvedBaseDir+string(os.PathSeparator)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	// Reject symlinks
	info, err := os.Lstat(fullPath)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "File not found", nil, "")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	// Read file
	// #nosec G304 -- fullPath is sanitized and enforced under baseDir with symlink rejection.
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		a.Log.Error("Failed to read media file", "path", resolvedPath, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}

	// Determine content type from message metadata first, then extension fallback.
	contentType := "application/octet-stream"
	if mimeHint != "" {
		contentType = strings.TrimSpace(strings.Split(strings.ToLower(mimeHint), ";")[0])
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if mappedType := getContentTypeFromExt(ext); mappedType != "" && contentType == "application/octet-stream" {
		contentType = mappedType
	}

	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, max-age=3600") // Cache for 1 hour, private
	r.RequestCtx.SetBody(data)

	return nil
}
