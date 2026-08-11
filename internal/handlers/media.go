package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
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
	return os.MkdirAll(path, 0755)
}

// getExtensionFromMimeType returns file extension based on mime type
func getExtensionFromMimeType(mimeType string) string {
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
	case strings.HasPrefix(mimeType, "text/html"):
		return ".html"
	default:
		return ""
	}
}

// DownloadAndSaveMedia downloads media from GOWA and saves it locally.
// mediaID may be a full URL, a relative server path, or a GOWA message ID.
// Returns the local file path (relative to media storage) or error
func (a *App) DownloadAndSaveMedia(ctx context.Context, mediaID string, mimeType string, account *whatsapp.Account) (string, error) {
	var provider whatsapp.Provider
	if a.WARegistry != nil {
		provider = a.WARegistry.Get(account)
	}

	// GOWA media handling: GOWA webhooks send either a full URL, a relative
	// server path, or just the media field as-is. We try multiple strategies:
	//   1. Full HTTP URL → download directly
	//   2. Relative path → prepend GOWA base URL, download directly
	//   3. GOWA message ID → use DownloadMessageMedia (calls /message/{id}/download)
	var data []byte
	var err error

	gowaClient, ok := provider.(*gowa.Client)
	if ok {
		if strings.HasPrefix(mediaID, "http") {
			// SECURITY (gap #7): only fetch absolute media URLs that belong to
			// the GOWA instance itself. A signed webhook can carry an arbitrary
			// URL; fetching an external one would (a) be an SSRF vector into
			// internal services and (b) risk leaking Basic Auth. Reject any URL
			// not on the account's GOWA base origin. (DownloadMedia is also
			// hardened as defense-in-depth: no cross-origin auth, no cross-host
			// redirects, size-capped.)
			if !gowa.URLMatchesBase(mediaID, account.GowaBaseURL) {
				return "", fmt.Errorf("refusing media URL not on the GOWA base host")
			}
			data, err = gowaClient.DownloadMedia(ctx, mediaID, "")
		} else if strings.Contains(mediaID, "/") {
			// Relative path — prepend base URL
			baseURL := account.GowaBaseURL
			if baseURL == "" {
				baseURL = "http://localhost:3000"
			}
			baseURL = strings.TrimSuffix(baseURL, "/")
			gowaURL := baseURL + "/" + strings.TrimPrefix(mediaID, "/")
			data, err = gowaClient.DownloadMedia(ctx, gowaURL, "")
		} else {
			// GOWA message ID or media field — try DownloadMedia first,
			// then fall back to treating as message ID with download endpoint
			data, err = gowaClient.DownloadMedia(ctx, mediaID, "")
		}
	} else {
		err = fmt.Errorf("GOWA provider not available")
	}
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	return a.saveMediaBytes(data, mimeType)
}

// writeMediaFile picks the media subdirectory from mimeType, ensures it
// exists, writes data under a fresh uuid+ext filename, and returns the path
// relative to the media storage root (suitable for Message.MediaURL). Shared
// by saveMediaBytes (downloaded bytes) and saveMediaLocally (uploaded bytes)
// so the subdir/write rule lives in one place.
func (a *App) writeMediaFile(data []byte, mimeType, ext string) (string, error) {
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

	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	filename := uuid.New().String() + ext
	filePath := filepath.Join(a.getMediaStoragePath(), subdir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	return filepath.Join(subdir, filename), nil
}

// saveMediaBytes sniffs the content type of already-downloaded bytes, writes
// them to the appropriate media subdirectory, and returns the relative path
// suitable for Message.MediaURL. Extracted from DownloadAndSaveMedia so the
// redownload handler (and any future caller that already has bytes in hand)
// can reuse the exact same save logic.
func (a *App) saveMediaBytes(data []byte, mimeType string) (string, error) {
	// Sniff the actual content type from the first 512 bytes. This catches
	// wrong/missing MIME types from GOWA and uses the sniffed type instead.
	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}
	sniffType := http.DetectContentType(data[:sniffLen])
	// Use the sniffed type if the caller didn't provide a useful one. GOWA
	// returns generic types ("image", "audio", "video", "document") without
	// the slash subtype, which are NOT valid MIME types and break the frontend
	// (it checks media_mime_type.startsWith("image/")). Treat those as unknown
	// so the sniffed type wins.
	if mimeType == "" || mimeType == "application/octet-stream" || !strings.Contains(mimeType, "/") {
		mimeType = sniffType
	}
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}

	relativePath, err := a.writeMediaFile(data, mimeType, ext)
	if err != nil {
		return "", err
	}
	a.Log.Info("Media saved", "path", relativePath, "size", len(data))

	return relativePath, nil
}

// readLocalMedia resolves a storage-relative path against the media root with
// directory-traversal and symlink protection, then reads and returns the file
// bytes. ok is false when relPath is empty/invalid or the file is missing,
// unreadable, or a symlink. Shared by media/avatar serving handlers so the
// path-safety rules live in one place.
func (a *App) readLocalMedia(relPath string) ([]byte, bool) {
	if relPath == "" {
		return nil, false
	}
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		a.Log.Error("Storage configuration error", "error", err)
		return nil, false
	}
	fullPath, err := filepath.Abs(filepath.Join(baseDir, filepath.Clean(relPath)))
	if err != nil || !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return nil, false
	}
	info, err := os.Lstat(fullPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return nil, false
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, false
	}
	return data, true
}

// removeLocalMedia best-effort deletes a storage-relative file using the same
// traversal/symlink guards as readLocalMedia. Errors are ignored — used to
// clean up superseded cached files (e.g. a replaced avatar).
func (a *App) removeLocalMedia(relPath string) {
	if relPath == "" {
		return
	}
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return
	}
	fullPath, err := filepath.Abs(filepath.Join(baseDir, filepath.Clean(relPath)))
	if err != nil || !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return
	}
	if info, err := os.Lstat(fullPath); err != nil || info.Mode()&os.ModeSymlink != 0 {
		return
	}
	_ = os.Remove(fullPath)
}

// isRecoverableMediaType reports whether a message type can have its media
// lazily fetched from the provider on first view. Non-media types (text,
// location, contacts) and rendering-only types (template, interactive) have no
// downloadable bytes, so ServeMedia should not attempt recovery for them when
// MediaURL is empty. Sticker is included because WhatsApp delivers stickers as
// image-like media even though the app overlays them.
func isRecoverableMediaType(t models.MessageType) bool {
	switch t {
	case models.MessageTypeImage, models.MessageTypeVideo,
		models.MessageTypeAudio, models.MessageTypeDocument:
		return true
	case "sticker":
		return true
	default:
		return false
	}
}

// ServeMedia serves media files from local storage
// Only authorized users who have access to the message can view the media
func (a *App) ServeMedia(r *fastglue.Request) error {
	// Get auth context
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	// Get the message ID from URL parameter
	messageIDStr := r.RequestCtx.UserValue("message_id").(string)
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	// Find the message and verify access
	message, err := findByIDAndOrg[models.Message](a.DB, r, messageID, orgID, "Message")
	if err != nil {
		return nil
	}

	// Users without contacts:read permission can only access media from contacts
	// assigned to them (the persistent owner or a collaborator, via
	// scopeAssignedContact).
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		var contact models.Contact
		q := a.scopeAssignedContact(a.DB.Where("id = ? AND organization_id = ?", message.ContactID, orgID), userID, orgID)
		if err := q.First(&contact).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
		}
	}

	// Resolve the media storage root once — used both for path-traversal guards
	// and as the parent for any lazily recovered file.
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		a.Log.Error("Storage configuration error", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Storage configuration error", nil, "")
	}

	var filePath, fullPath string
	var info os.FileInfo

	if message.MediaURL == "" {
		// No local media path. History-synced messages intentionally store
		// MediaURL="" (see gowa_history_sync.go) because the bytes were never
		// downloaded to disk; recovery is expected to fetch them on first view
		// via WhatsAppMessageID. Only attempt recovery for genuine media types
		// that carry a WhatsApp message ID — otherwise there is nothing to fetch.
		if message.WhatsAppMessageID == "" || !isRecoverableMediaType(message.MessageType) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No media found", nil, "")
		}
		// Fall through to the recovery block below. Mark info as missing by
		// leaving err non-nil so the Lstat error branch runs the recovery.
		err = os.ErrNotExist
	} else {
		// Security: prevent directory traversal and symlink attacks
		filePath = filepath.Clean(message.MediaURL)
		fullPath, err = filepath.Abs(filepath.Join(baseDir, filePath))
		if err != nil || !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
		}

		// Reject symlinks
		info, err = os.Lstat(fullPath)
	}

	if err != nil {
		// File is missing from disk (or never downloaded). If it's a GOWA
		// message, try to auto-recover/download it.
		// The contact row is loaded first because it's needed for both the chat
		// JID and as an account-name fallback: a message may reference an account
		// that was renamed/deleted (e.g. legacy GOWA history-sync rows), while the
		// contact row still points at the live account that currently owns the chat.
		var contact models.Contact
		hasContact := a.DB.Where("id = ? AND organization_id = ?", message.ContactID, orgID).First(&contact).Error == nil

		var account models.WhatsAppAccount
		acctName := message.WhatsAppAccount
		accountErr := a.DB.Where("organization_id = ? AND name = ?", orgID, acctName).First(&account).Error
		if accountErr != nil && hasContact && contact.WhatsAppAccount != "" && contact.WhatsAppAccount != acctName {
			// The message's account no longer exists, but the contact carries a
			// (possibly different) account name that may be the renamed successor.
			// Try it rather than silently giving up. Log the mismatch so the
			// drift between message.whats_app_account and contact.whats_app_account
			// is visible for later cleanup.
			fallbackName := contact.WhatsAppAccount
			a.Log.Warn("Message references a non-existent account; falling back to the contact's current account",
				"message_id", message.ID, "msg_account", acctName, "contact_account", fallbackName)
			accountErr = a.DB.Where("organization_id = ? AND name = ?", orgID, fallbackName).First(&account).Error
		}

		if accountErr != nil {
			// No usable account — make the failure explicit instead of a silent 404.
			a.Log.Warn("Media missing from disk and no recoverable account for message",
				"message_id", message.ID, "wa_message_id", message.WhatsAppMessageID,
				"msg_account", acctName, "error", err)
		} else if message.WhatsAppMessageID != "" && hasContact {
			a.decryptAccountSecrets(&account)
			waAccount := account.ToWAAccount()
			provider := a.resolveProvider(&account)
			gowaClient, ok := provider.(*gowa.Client)
			if ok {
				a.Log.Info("Media missing from disk, attempting auto-recovery", "message_id", message.ID, "path", fullPath)
				// Build the chat JID (handles group @g.us vs 1:1 suffix).
				chatJID := gowaChatJID(&contact)
				ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
				data, mediaType, derr := gowaClient.DownloadMessageMedia(ctx, waAccount, message.WhatsAppMessageID, chatJID)
				cancel()
				if derr != nil {
					a.Log.Warn("GOWA media recovery failed", "message_id", message.ID, "wa_message_id", message.WhatsAppMessageID, "error", derr)
				}
				if derr == nil && len(data) > 0 {
					relativePath, serr := a.saveMediaBytes(data, mediaType)
					if serr == nil {
						// Update the message in place. Sniff the real MIME
						// type from the bytes (GOWA's mediaType is generic
						// like "image", not a valid MIME type).
						sniffLen := 512
						if len(data) < sniffLen {
							sniffLen = len(data)
						}
						sniffedType := http.DetectContentType(data[:sniffLen])
						updates := map[string]any{
							"media_url":       relativePath,
							"media_mime_type": sniffedType,
							// Re-link the message to the account that actually
							// owns it, so future fetches find the account on the
							// first lookup and ServeMedia no longer has to fall
							// back. Only set when the recovery account differs.
							"whats_app_account": account.Name,
						}
						a.DB.Model(&models.Message{}).Where("id = ?", message.ID).Updates(updates)

						// Re-evaluate full path
						filePath = filepath.Clean(relativePath)
						fullPath, err = filepath.Abs(filepath.Join(baseDir, filePath))
						if err == nil {
							info, err = os.Lstat(fullPath)
						}
					}
				}
			}
		}
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "File not found", nil, "")
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	// Read file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		a.Log.Error("Failed to read media file", "path", fullPath, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}

	// Determine content type: first try the file extension, then sniff the
	// actual bytes if the extension is unknown (e.g. .bin from GOWA downloads).
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".mp4":
		contentType = "video/mp4"
	case ".3gp":
		contentType = "video/3gpp"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".aac":
		contentType = "audio/aac"
	case ".m4a":
		contentType = "audio/mp4"
	case ".ogg":
		contentType = "audio/ogg"
	case ".amr":
		contentType = "audio/amr"
	case ".pdf":
		contentType = "application/pdf"
	case ".doc":
		contentType = "application/msword"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		contentType = "application/vnd.ms-excel"
	case ".xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		contentType = "text/plain"
	default:
		// Unknown extension (.bin, no extension, etc.) — sniff the first 512 bytes.
		sniffLen := len(data)
		if sniffLen > 512 {
			sniffLen = 512
		}
		contentType = http.DetectContentType(data[:sniffLen])
	}

	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, max-age=3600") // Cache for 1 hour, private
	r.RequestCtx.SetBody(data)

	return nil
}
