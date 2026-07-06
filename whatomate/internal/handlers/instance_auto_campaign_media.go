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
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const maxAutoCampaignMediaUploadSize = 16 << 20 // 16MB

// UploadInstanceAutoCampaignMedia uploads a media file used by per-instance auto campaigns.
func (a *App) UploadInstanceAutoCampaignMedia(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Auto campaign media is only available for Whatsmeow provider", nil, "")
	}

	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	instanceID, err := parsePathUUID(r, "id", "instance")
	if err != nil {
		return nil
	}

	instance, err := findByIDAndOrg[models.WhatsAppInstance](requestDB, r, instanceID, orgID, "Instance")
	if err != nil {
		return nil
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

	data, err := io.ReadAll(io.LimitReader(file, maxAutoCampaignMediaUploadSize+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}
	if len(data) > maxAutoCampaignMediaUploadSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 16MB", nil, "")
	}

	mimeType, allowed := resolveCampaignUploadMIME(fileHeader.Header.Get("Content-Type"), fileHeader.Filename, data)
	if !allowed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type: "+mimeType, nil, "")
	}
	if !a.checkQuotaWithDeltaOrRespond(r, license.ResourceStorage, orgID, int64(len(data))) {
		return nil
	}

	localPath, err := a.saveInstanceAutoCampaignMedia(
		orgID,
		instance.ID.String(),
		sanitizeFilename(fileHeader.Filename),
		data,
		mimeType,
	)
	if err != nil {
		a.Log.Error("Failed to save auto campaign media", "instance_id", instance.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media", nil, "")
	}

	autoCampaign := waManager.AutoCampaignSettingsFromSettings(instance.Settings)
	autoCampaign.MediaLocalPath = localPath
	autoCampaign.MediaMimeType = mimeType
	autoCampaign.MediaFilename = sanitizeFilename(fileHeader.Filename)

	if err := waManager.ValidateAutoCampaignSettings(autoCampaign.ToJSONB()); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "settings.auto_campaign")
	}
	if err := a.persistInstanceAutoCampaignSettings(instance.ID, autoCampaign); err != nil {
		a.Log.Error("Failed to persist auto campaign media settings", "instance_id", instance.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media settings", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message":       "Media uploaded successfully",
		"auto_campaign": autoCampaign.ToJSONB(),
		"filename":      autoCampaign.MediaFilename,
		"mime_type":     autoCampaign.MediaMimeType,
		"local_path":    autoCampaign.MediaLocalPath,
	})
}

func (a *App) saveInstanceAutoCampaignMedia(orgID uuid.UUID, instanceID, originalFilename string, data []byte, mimeType string) (string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(originalFilename)))
	if ext == "" || ext == "." {
		ext = getExtensionFromMimeType(mimeType)
	}
	if ext == "" || ext == "." {
		ext = ".bin"
	}

	subdir := organizationMediaSubdir(orgID, "instance-auto-campaigns")
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	filename := instanceID + ext
	relativePath := filepath.Join(subdir, filename)
	filePath := filepath.Join(a.getMediaStoragePath(), relativePath)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	return relativePath, nil
}

func (a *App) persistInstanceAutoCampaignSettings(instanceID uuid.UUID, settings waManager.AutoCampaignSettings) error {
	payload, err := json.Marshal(settings.ToJSONB())
	if err != nil {
		return fmt.Errorf("failed to encode auto campaign settings: %w", err)
	}

	return a.DB.Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Update(
			"settings",
			gorm.Expr(
				fmt.Sprintf("jsonb_set(COALESCE(settings, '{}'::jsonb), '{%s}', ?::jsonb, true)", waManager.InstanceSettingAutoCampaign),
				string(payload),
			),
		).Error
}
