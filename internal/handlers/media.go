package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

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
	// Get the media URL from Meta
	mediaURL, err := a.WhatsApp.GetMediaURL(ctx, mediaID, account)
	if err != nil {
		return "", fmt.Errorf("failed to get media URL: %w", err)
	}

	// Download the media content
	data, err := a.WhatsApp.DownloadMedia(ctx, mediaURL, account.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	// Determine file extension
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}

	// Generate unique filename
	filename := uuid.New().String() + ext

	// Determine subdirectory based on media type
	var subdir string
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		subdir = "images"
	case strings.HasPrefix(mimeType, "video/"):
		subdir = "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		subdir = "audio"
	default:
		subdir = "documents"
	}

	// Ensure directory exists
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Save file
	filePath := filepath.Join(a.getMediaStoragePath(), subdir, filename)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	// Return relative path for storage in database
	relativePath := filepath.Join(subdir, filename)
	a.Log.Info("Media saved", "path", relativePath, "size", len(data))

	return relativePath, nil
}

// ServeMedia serves media files from local storage
// Only authorized users who have access to the message can view the media
func (a *App) ServeMedia(r *fastglue.Request) error {
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
	if err := requestDB.WithContext(r.RequestCtx).
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

	reader, objectInfo, err := a.ObjectStorage.GetObject(r.RequestCtx, message.MediaAsset.S3Key)
	if err != nil {
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
