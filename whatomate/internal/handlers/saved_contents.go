package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type SavedContentRequest struct {
	Name     string `json:"name"`
	Body     string `json:"body"`
	Category string `json:"category"`
}

type SavedContentResponse struct {
	ID            uuid.UUID          `json:"id"`
	Name          string             `json:"name"`
	Body          string             `json:"body"`
	Variables     models.StringArray `json:"variables"`
	Category      string             `json:"category"`
	Preview       string             `json:"preview"`
	MediaID       string             `json:"media_id,omitempty"`
	MediaFilename string             `json:"media_filename,omitempty"`
	MediaMimeType string             `json:"media_mime_type,omitempty"`
	CreatedBy     string             `json:"created_by,omitempty"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
}

type SavedContentImportItem struct {
	Name     string `json:"name"`
	Body     string `json:"body"`
	Category string `json:"category"`
}

func savedContentToResponse(sc models.SavedContent) SavedContentResponse {
	var createdBy string
	if sc.CreatedBy != nil {
		createdBy = sc.CreatedBy.FullName
	}
	return SavedContentResponse{
		ID:            sc.ID,
		Name:          sc.Name,
		Body:          sc.Body,
		Variables:     sc.Variables,
		Category:      sc.Category,
		Preview:       models.RenderPreview(sc.Body),
		MediaID:       sc.MediaID,
		MediaFilename: sc.MediaFilename,
		MediaMimeType: sc.MediaMimeType,
		CreatedBy:     createdBy,
		CreatedAt:     sc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     sc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (a *App) ListSavedContents(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	category := string(r.RequestCtx.QueryArgs().Peek("category"))
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	query := requestDB.Where("organization_id = ?", orgID)

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("name ILIKE ? OR body ILIKE ?", searchPattern, searchPattern)
	}

	var total int64
	query.Model(&models.SavedContent{}).Count(&total)

	var contents []models.SavedContent
	if err := pg.Apply(query.Order("name ASC")).
		Preload("CreatedBy").
		Find(&contents).Error; err != nil {
		a.Log.Error("Failed to list saved contents", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to list saved contents", nil, "")
	}

	result := make([]SavedContentResponse, len(contents))
	for i, sc := range contents {
		result[i] = savedContentToResponse(sc)
	}

	return r.SendEnvelope(map[string]any{
		"saved_contents": result,
		"total":          total,
		"page":           pg.Page,
		"limit":          pg.Limit,
	})
}

func (a *App) CreateSavedContent(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionWrite); err != nil {
		return nil
	}

	var req SavedContentRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	name := strings.TrimSpace(req.Name)
	body := strings.TrimSpace(req.Body)
	if name == "" || body == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"name and body are required", nil, "")
	}
	if len(body) > 51200 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"body must not exceed 50KB", nil, "")
	}

	var existing models.SavedContent
	if err := a.DB.Where("organization_id = ? AND name = ?", orgID, name).
		First(&existing).Error; err == nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			"Saved content with this name already exists", nil, "")
	}

	content := models.SavedContent{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           name,
		Body:           body,
		Variables:      models.ExtractVariables(body),
		Category:       strings.TrimSpace(req.Category),
		CreatedByID:    userID,
	}

	if err := a.DB.Create(&content).Error; err != nil {
		if isDuplicateKeyError(err) {
			return r.SendErrorEnvelope(fasthttp.StatusConflict,
				"Saved content with this name already exists", nil, "")
		}
		a.Log.Error("Failed to create saved content", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to create saved content", nil, "")
	}

	var result models.SavedContent
	if err := a.DB.Where("id = ? AND organization_id = ?", content.ID, orgID).Preload("CreatedBy").First(&result).Error; err != nil {
		return r.SendEnvelope(savedContentToResponse(content))
	}
	return r.SendEnvelope(savedContentToResponse(result))
}

func (a *App) GetSavedContent(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := requestDB.Preload("CreatedBy").
		Where("id = ? AND organization_id = ?", id, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound,
			"Saved content not found", nil, "")
	}

	return r.SendEnvelope(savedContentToResponse(content))
}

func (a *App) UpdateSavedContent(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound,
			"Saved content not found", nil, "")
	}

	var req SavedContentRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	updatedName := strings.TrimSpace(req.Name)
	if updatedName != "" && updatedName != content.Name {
		var duplicate models.SavedContent
		if err := a.DB.Where("organization_id = ? AND name = ? AND id <> ?", orgID, updatedName, id).
			First(&duplicate).Error; err == nil {
			return r.SendErrorEnvelope(fasthttp.StatusConflict,
				"Saved content with this name already exists", nil, "")
		}
		content.Name = updatedName
	}

	if req.Body != "" {
		body := strings.TrimSpace(req.Body)
		if len(body) > 51200 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"body must not exceed 50KB", nil, "")
		}
		content.Body = body
		content.Variables = models.ExtractVariables(body)
	}

	if req.Category != "" || strings.TrimSpace(req.Category) == "" {
		content.Category = strings.TrimSpace(req.Category)
	}

	if err := a.DB.Model(&models.SavedContent{}).Where("id = ? AND organization_id = ?", content.ID, orgID).Updates(map[string]any{
		"name":      content.Name,
		"body":      content.Body,
		"variables": content.Variables,
		"category":  content.Category,
	}).Error; err != nil {
		a.Log.Error("Failed to update saved content", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to update saved content", nil, "")
	}

	a.DB.Where("id = ? AND organization_id = ?", content.ID, orgID).Preload("CreatedBy").First(&content)
	return r.SendEnvelope(savedContentToResponse(content))
}

func (a *App) DeleteSavedContent(r *fastglue.Request) error {
	var (
		orgID  uuid.UUID
		userID uuid.UUID
		err    error
	)
	orgID, userID, err = a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionDelete); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound,
			"Saved content not found", nil, "")
	}

	if err := a.DB.Where("id = ? AND organization_id = ?", content.ID, orgID).
		Delete(&models.SavedContent{}).Error; err != nil {
		a.Log.Error("Failed to delete saved content", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to delete saved content", nil, "")
	}

	// Clean up associated media file
	if content.MediaLocalPath != "" {
		fullPath := filepath.Join(a.getMediaStoragePath(), content.MediaLocalPath)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			a.Log.Error("Failed to delete saved content media file", "path", fullPath, "error", err)
		}
	}

	return r.SendEnvelope(map[string]string{"message": "Saved content deleted"})
}

func (a *App) ListSavedContentCategories(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionRead); err != nil {
		return nil
	}

	var categories []string
	if err := requestDB.Model(&models.SavedContent{}).
		Where("organization_id = ? AND category != '' AND category IS NOT NULL", orgID).
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		a.Log.Error("Failed to list saved content categories", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to list categories", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"categories": categories,
	})
}

func (a *App) PreviewSavedContent(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := requestDB.Where("id = ? AND organization_id = ?", id, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound,
			"Saved content not found", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"preview":   models.RenderPreview(content.Body),
		"variables": content.Variables,
	})
}

func (a *App) ImportSavedContents(r *fastglue.Request) error {
	var (
		orgID  uuid.UUID
		userID uuid.UUID
		err    error
	)
	orgID, userID, err = a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionImport); err != nil {
		return nil
	}

	var items []SavedContentImportItem
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &items); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body, expected JSON array", nil, "")
	}

	if len(items) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No items to import", nil, "")
	}

	created := make([]SavedContentResponse, 0, len(items))
	skipped := 0

	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		body := strings.TrimSpace(item.Body)
		if name == "" || body == "" {
			skipped++
			continue
		}
		if len(body) > 51200 {
			skipped++
			continue
		}

		var existing models.SavedContent
		if err := a.DB.Where("organization_id = ? AND name = ?", orgID, name).
			First(&existing).Error; err == nil {
			skipped++
			continue
		}

		content := models.SavedContent{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: orgID,
			Name:           name,
			Body:           body,
			Variables:      models.ExtractVariables(body),
			Category:       strings.TrimSpace(item.Category),
			CreatedByID:    userID,
		}

		if err := a.DB.Create(&content).Error; err != nil {
			a.Log.Error("Failed to import saved content", "name", name, "error", err)
			skipped++
			continue
		}

		created = append(created, savedContentToResponse(content))
	}

	return r.SendEnvelope(map[string]any{
		"imported": len(created),
		"skipped":  skipped,
		"total":    len(items),
		"items":    created,
	})
}

func (a *App) saveSavedContentMedia(orgID uuid.UUID, contentID string, data []byte, mimeType string) (string, error) {
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}

	subdir := organizationMediaSubdir(orgID, "saved-contents")
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	filename := contentID + ext
	relativePath := filepath.Join(subdir, filename)
	filePath, err := a.resolveMediaFilePath(relativePath)
	if err != nil {
		return "", fmt.Errorf("invalid media file path: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	a.Log.Info("Saved content media saved locally", "path", relativePath, "size", len(data))
	return relativePath, nil
}

func (a *App) UploadSavedContentMedia(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionWrite); err != nil {
		return nil
	}

	contentUUID, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := a.DB.Where("id = ? AND organization_id = ?", contentUUID, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Saved content not found", nil, "")
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
	}

	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, "")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to open file", nil, "")
	}
	defer func() { _ = file.Close() }()

	const maxMediaSize = 16 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxMediaSize+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}
	if len(data) > maxMediaSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 16MB", nil, "")
	}

	mimeType, allowed := resolveCampaignUploadMIME(fileHeader.Header.Get("Content-Type"), fileHeader.Filename, data)
	if !allowed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type: "+mimeType, nil, "")
	}

	// Remove old media file if updating
	if content.MediaLocalPath != "" {
		oldPath := filepath.Join(a.getMediaStoragePath(), content.MediaLocalPath)
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			a.Log.Error("Failed to remove old saved content media", "path", oldPath, "error", err)
		}
	}

	if !a.checkQuotaWithDeltaOrRespond(r, license.ResourceStorage, orgID, int64(len(data))) {
		return nil
	}

	localPath, err := a.saveSavedContentMedia(orgID, contentUUID.String(), data, mimeType)
	if err != nil {
		a.Log.Error("Failed to save media locally", "error", err)
	}

	updates := map[string]any{
		"media_id":         "",
		"media_filename":   sanitizeFilename(fileHeader.Filename),
		"media_mime_type":  mimeType,
		"media_local_path": localPath,
	}
	if err := a.DB.Model(&models.SavedContent{}).Where("id = ? AND organization_id = ?", content.ID, orgID).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update saved content with media info", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media info", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"filename":  fileHeader.Filename,
		"mime_type": mimeType,
		"message":   "Media uploaded successfully",
	})
}

func (a *App) ServeSavedContentMedia(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceSavedContents, models.ActionRead); err != nil {
		return nil
	}

	contentUUID, err := parsePathUUID(r, "id", "saved content")
	if err != nil {
		return nil
	}

	var content models.SavedContent
	if err := requestDB.Where("id = ? AND organization_id = ?", contentUUID, orgID).
		First(&content).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Saved content not found", nil, "")
	}

	if content.MediaLocalPath == "" {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No media found", nil, "")
	}

	filePath := filepath.Clean(content.MediaLocalPath)
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Storage configuration error", nil, "")
	}
	fullPath, err := filepath.Abs(filepath.Join(baseDir, filePath))
	if err != nil || !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "File not found", nil, "")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		a.Log.Error("Failed to read saved content media file", "path", fullPath, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}

	contentType := content.MediaMimeType
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		contentType = getMimeTypeFromExtension(ext)
	}

	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, max-age=3600")
	r.RequestCtx.SetBody(data)
	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
