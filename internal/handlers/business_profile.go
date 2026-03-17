package handlers

import (
	"io"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// GetBusinessProfile returns the business profile for a WhatsApp account
func (a *App) GetBusinessProfile(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := a.resolveWhatsAppAccountByID(r, id, orgID)
	if err != nil {
		if err.Error() == "failed to decrypt account secrets" {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to decrypt account secrets", nil, "")
		}
		return nil
	}

	// Create a context for the request
	ctx := r.RequestCtx

	// Call the WhatsApp client
	profile, err := a.WhatsApp.GetBusinessProfile(ctx, a.toWhatsAppAccount(account))
	if err != nil {
		a.Log.Error("Failed to get business profile", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to get business profile", nil, "")
	}

	return r.SendEnvelope(profile)
}

// UpdateBusinessProfile updates the business profile for a WhatsApp account
func (a *App) UpdateBusinessProfile(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := a.resolveWhatsAppAccountByID(r, id, orgID)
	if err != nil {
		if err.Error() == "failed to decrypt account secrets" {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to decrypt account secrets", nil, "")
		}
		return nil
	}

	var input whatsapp.BusinessProfileInput
	if err := a.decodeRequest(r, &input); err != nil {
		return nil
	}

	ctx := r.RequestCtx
	waAccount := a.toWhatsAppAccount(account)

	if err := a.WhatsApp.UpdateBusinessProfile(ctx, waAccount, input); err != nil {
		a.Log.Error("Failed to update business profile", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update business profile", nil, "")
	}

	// Re-fetch to ensure we have the latest state
	profile, err := a.WhatsApp.GetBusinessProfile(ctx, waAccount)
	if err != nil {
		// If re-fetch fails, just return success message
		return r.SendEnvelope(map[string]string{"message": "Profile updated successfully"})
	}

	return r.SendEnvelope(profile)
}

// UpdateProfilePicture handles the profile picture upload
func (a *App) UpdateProfilePicture(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := a.resolveWhatsAppAccountByID(r, id, orgID)
	if err != nil {
		if err.Error() == "failed to decrypt account secrets" {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to decrypt account secrets", nil, "")
		}
		return nil
	}

	// 1. Get the file from request
	fileHeader, err := r.RequestCtx.FormFile("file")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing file", nil, "")
	}

	// 2. Open and read file
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to open file", nil, "")
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			a.Log.Error("Failed to close file", "error", cerr)
		}
	}()

	if fileHeader.Size > whatsappImageMaxBytes {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 5MB", nil, "")
	}

	fileContent, err := io.ReadAll(io.LimitReader(file, whatsappImageMaxBytes+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}
	if len(fileContent) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File is empty", nil, "")
	}
	if int64(len(fileContent)) > whatsappImageMaxBytes {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 5MB", nil, "")
	}

	mimeType := normalizeWhatsAppMediaMIME(resolveWhatsAppMediaMIME(
		fileHeader.Header.Get("Content-Type"),
		fileHeader.Filename,
		fileContent,
	))
	if _, ok := whatsappImageMIMEs[mimeType]; !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type", nil, "")
	}

	ctx := r.RequestCtx
	waAccount := a.toWhatsAppAccount(account)

	// Upload to Meta to get handle
	handle, err := a.WhatsApp.UploadProfilePicture(ctx, waAccount, fileContent, mimeType)
	if err != nil {
		a.Log.Error("Failed to upload profile picture", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to upload profile picture", nil, "")
	}

	// Update Business Profile with the handle
	input := whatsapp.BusinessProfileInput{
		MessagingProduct:     "whatsapp",
		ProfilePictureHandle: handle,
	}

	err = a.WhatsApp.UpdateBusinessProfile(ctx, waAccount, input)

	if err != nil {
		a.Log.Error("Failed to update profile request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Uploaded but failed to set profile picture", nil, "")
	}

	return r.SendEnvelope(map[string]string{
		"message": "Profile picture updated successfully",
		"handle":  handle,
	})
}
