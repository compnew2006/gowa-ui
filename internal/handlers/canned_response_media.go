package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

const (
	maxCannedAttachmentSize = 16 * 1024 * 1024 // 16 MB
)

type parsedCannedResponseRequest struct {
	Request              CannedResponseRequest
	AttachmentFiles      []*multipart.FileHeader
	HasAttachmentChanges bool
}

func (a *App) parseCannedResponseRequest(r *fastglue.Request) (parsedCannedResponseRequest, error) {
	if isMultipartFormRequest(r) {
		return a.parseMultipartCannedResponseRequest(r)
	}

	var req CannedResponseRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return parsedCannedResponseRequest{}, err
	}

	return parsedCannedResponseRequest{
		Request:              req,
		HasAttachmentChanges: req.KeepAttachmentIDs != nil,
	}, nil
}

func isMultipartFormRequest(r *fastglue.Request) bool {
	contentType := strings.ToLower(string(r.RequestCtx.Request.Header.ContentType()))
	return strings.HasPrefix(contentType, "multipart/form-data")
}

func (a *App) parseMultipartCannedResponseRequest(r *fastglue.Request) (parsedCannedResponseRequest, error) {
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return parsedCannedResponseRequest{}, fmt.Errorf("invalid multipart form")
	}

	req := CannedResponseRequest{
		Name:     strings.TrimSpace(firstFormValue(form.Value, "name")),
		Shortcut: strings.TrimSpace(firstFormValue(form.Value, "shortcut")),
		Content:  firstFormValue(form.Value, "content"),
		Category: strings.TrimSpace(firstFormValue(form.Value, "category")),
	}

	isActive, err := parseOptionalBool(firstFormValue(form.Value, "is_active"))
	if err != nil {
		return parsedCannedResponseRequest{}, fmt.Errorf("invalid is_active value")
	}
	req.IsActive = isActive

	keepIDsRaw := strings.TrimSpace(firstFormValue(form.Value, "keep_attachment_ids"))
	hasAttachmentChanges := false
	if keepIDsRaw != "" {
		if err := json.Unmarshal([]byte(keepIDsRaw), &req.KeepAttachmentIDs); err != nil {
			return parsedCannedResponseRequest{}, fmt.Errorf("invalid keep_attachment_ids payload")
		}
		hasAttachmentChanges = true
	}

	files := form.File["attachments"]
	if len(files) > 0 {
		hasAttachmentChanges = true
	}

	return parsedCannedResponseRequest{
		Request:              req,
		AttachmentFiles:      files,
		HasAttachmentChanges: hasAttachmentChanges,
	}, nil
}

func firstFormValue(values map[string][]string, key string) string {
	if values == nil {
		return ""
	}
	items, ok := values[key]
	if !ok || len(items) == 0 {
		return ""
	}
	return items[0]
}

func parseOptionalBool(raw string) (*bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (a *App) mergeCannedResponseAttachments(
	orgID uuid.UUID,
	existing models.CannedResponseAttachments,
	keepIDs []string,
	files []*multipart.FileHeader,
	hasAttachmentChanges bool,
) (models.CannedResponseAttachments, models.CannedResponseAttachments, error) {
	if !hasAttachmentChanges {
		return existing, nil, nil
	}

	existingByID := make(map[string]models.CannedResponseAttachment, len(existing))
	for _, attachment := range existing {
		existingByID[attachment.ID] = attachment
	}

	next := make(models.CannedResponseAttachments, 0, len(keepIDs)+len(files))
	kept := make(map[string]struct{}, len(keepIDs))
	for _, rawID := range keepIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, dup := kept[id]; dup {
			continue
		}
		attachment, ok := existingByID[id]
		if !ok {
			continue
		}
		next = append(next, attachment)
		kept[id] = struct{}{}
	}

	for _, fileHeader := range files {
		attachment, err := a.persistCannedResponseAttachment(orgID, fileHeader)
		if err != nil {
			return nil, nil, err
		}
		next = append(next, attachment)
	}

	removed := make(models.CannedResponseAttachments, 0, len(existing))
	for _, attachment := range existing {
		if _, ok := kept[attachment.ID]; !ok {
			removed = append(removed, attachment)
		}
	}

	return next, removed, nil
}

func (a *App) persistCannedResponseAttachment(orgID uuid.UUID, fileHeader *multipart.FileHeader) (models.CannedResponseAttachment, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return models.CannedResponseAttachment{}, fmt.Errorf("failed to read attachment")
	}
	defer func() { _ = file.Close() }()

	data, err := readBoundedFile(file, int64(maxCannedAttachmentSize))
	if err != nil {
		return models.CannedResponseAttachment{}, err
	}

	mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(data)
	}
	mediaType, ok := normalizeCannedAttachmentType(mimeType)
	if !ok {
		return models.CannedResponseAttachment{}, fmt.Errorf("only image and video attachments are supported")
	}

	if a.License != nil {
		check, err := a.License.CheckQuotaWithDelta(context.Background(), license.ResourceStorage, orgID, int64(len(data)))
		if err != nil {
			return models.CannedResponseAttachment{}, fmt.Errorf("failed to evaluate storage quota")
		}
		if !check.Allowed {
			return models.CannedResponseAttachment{}, fmt.Errorf("licensed storage quota exceeded")
		}
	}

	relativePath, err := a.saveMediaLocally(orgID, data, mimeType, fileHeader.Filename)
	if err != nil {
		return models.CannedResponseAttachment{}, fmt.Errorf("failed to save attachment")
	}

	return models.CannedResponseAttachment{
		ID:        uuid.NewString(),
		Type:      mediaType,
		MimeType:  mimeType,
		FileName:  fileHeader.Filename,
		FilePath:  relativePath,
		FileSize:  int64(len(data)),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func normalizeCannedAttachmentType(mimeType string) (string, bool) {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return models.CannedResponseAttachmentTypeImage, true
	case strings.HasPrefix(mimeType, "video/"):
		return models.CannedResponseAttachmentTypeVideo, true
	default:
		return "", false
	}
}

func (a *App) cleanupCannedResponseAttachments(attachments models.CannedResponseAttachments) {
	for _, attachment := range attachments {
		fullPath, err := a.resolveMediaFilePath(attachment.FilePath)
		if err != nil {
			a.Log.Warn("Skipping canned attachment cleanup due to invalid path", "path", attachment.FilePath, "error", err)
			continue
		}
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			a.Log.Warn("Failed to delete canned attachment file", "path", fullPath, "error", err)
		}
	}
}

func (a *App) resolveMediaFilePath(relativePath string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(relativePath))
	if cleanPath == "" || cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid media file path")
	}

	basePath, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return "", err
	}
	fullPath, err := filepath.Abs(filepath.Join(basePath, cleanPath))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(fullPath, basePath+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid media file path")
	}

	return fullPath, nil
}

func (a *App) readCannedAttachmentData(attachment models.CannedResponseAttachment) ([]byte, error) {
	fullPath, err := a.resolveMediaFilePath(attachment.FilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment path")
	}

	// #nosec G304 -- fullPath is resolved and constrained to media storage root in resolveMediaFilePath.
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment file")
	}
	return data, nil
}
