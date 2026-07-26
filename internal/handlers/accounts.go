package handlers

import (
	"context"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/crypto"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// AccountRequest represents the request body for creating/updating an account
type AccountRequest struct {
	Name string `json:"name" validate:"required"`
	// GOWA credentials
	GowaBaseURL       string `json:"gowa_base_url"`
	GowaDeviceID      string `json:"gowa_device_id"`
	GowaWebhookSecret string `json:"gowa_webhook_secret"`
	// Shared
	IsDefaultIncoming bool `json:"is_default_incoming"`
	IsDefaultOutgoing bool `json:"is_default_outgoing"`
	AutoReadReceipt   bool `json:"auto_read_receipt"`
}

// AccountResponse represents the response for an account (without sensitive data)
type AccountResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// GOWA credentials
	GowaBaseURL          string `json:"gowa_base_url,omitempty"`
	GowaDeviceID         string `json:"gowa_device_id,omitempty"`
	HasGowaWebhookSecret bool   `json:"has_gowa_webhook_secret"`
	// GOWA live state (best-effort, empty when GOWA is unreachable)
	GowaJID       string `json:"gowa_jid,omitempty"`
	GowaConnected *bool  `json:"gowa_connected,omitempty"`
	// Shared
	IsDefaultIncoming bool       `json:"is_default_incoming"`
	IsDefaultOutgoing bool       `json:"is_default_outgoing"`
	AutoReadReceipt   bool       `json:"auto_read_receipt"`
	Status            string     `json:"status"`
	CreatedByID       *uuid.UUID `json:"created_by_id,omitempty"`
	CreatedByName     string     `json:"created_by_name,omitempty"`
	UpdatedByID       *uuid.UUID `json:"updated_by_id,omitempty"`
	UpdatedByName     string     `json:"updated_by_name,omitempty"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

// ListAccounts returns all WhatsApp accounts for the organization
func (a *App) ListAccounts(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionRead)
	if err != nil {
		return nil
	}

	var accounts []models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to list accounts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list accounts", nil, "")
	}

	// Convert to response format (hide sensitive data) and enrich accounts
	// with live connection state (best-effort — a down GOWA instance gets
	// empty fields, not an error).
	response := make([]AccountResponse, len(accounts))
	ctx := context.Background()
	for i, acc := range accounts {
		resp := accountToResponse(acc)
		if acc.GowaDeviceID != "" {
			a.enrichGowaStatus(ctx, &resp, &acc)
		}
		response[i] = resp
	}

	return r.SendEnvelope(map[string]any{
		"accounts": response,
	})
}

// CreateAccount creates a new WhatsApp account
func (a *App) CreateAccount(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite)
	if err != nil {
		return nil
	}

	var req AccountRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.Name == "" || req.GowaBaseURL == "" || req.GowaDeviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name, gowa_base_url, and gowa_device_id are required", nil, "")
	}
	// Auto-generate webhook secret if not supplied (FR-017).
	// Callers never need to provide one manually — the system ensures
	// every account has a secret before it can accept webhooks.
	if req.GowaWebhookSecret == "" {
		req.GowaWebhookSecret = gowa.GenerateWebhookSecret()
	}

	account := models.WhatsAppAccount{
		BaseModel:         models.BaseModel{},
		OrganizationID:    orgID,
		Name:              req.Name,
		GowaBaseURL:       req.GowaBaseURL,
		GowaDeviceID:      req.GowaDeviceID,
		GowaWebhookSecret: req.GowaWebhookSecret,
		IsDefaultIncoming: req.IsDefaultIncoming,
		IsDefaultOutgoing: req.IsDefaultOutgoing,
		AutoReadReceipt:   req.AutoReadReceipt,
		Status:            "active",
		CreatedByID:       &userID,
		UpdatedByID:       &userID,
	}

	if err := a.encryptAccountSecrets(&account); err != nil {
		a.Log.Error("Failed to encrypt account secrets", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	// If this is set as default, unset other defaults
	if req.IsDefaultIncoming {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}

	if err := a.DB.Create(&account).Error; err != nil {
		a.Log.Error("Failed to create account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	a.DB.Preload("CreatedBy").Preload("UpdatedBy").First(&account, "id = ?", account.ID)
	a.logAudit(orgID, userID,
		"account", account.ID, models.AuditActionCreated, nil, &account)

	return r.SendEnvelope(accountToResponse(account))
}

// GetAccount returns a single WhatsApp account
func (a *App) GetAccount(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionRead)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](
		a.DB.Preload("CreatedBy").Preload("UpdatedBy"), r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	return r.SendEnvelope(accountToResponse(*account))
}

// UpdateAccount updates a WhatsApp account
func (a *App) UpdateAccount(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := a.resolveWhatsAppAccountByID(r, id, orgID)
	if err != nil {
		return nil
	}

	oldAccount := *account // value copy for audit

	var req AccountRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Update fields if provided
	if req.Name != "" {
		account.Name = req.Name
	}

	// Update GOWA fields
	if req.GowaBaseURL != "" {
		account.GowaBaseURL = req.GowaBaseURL
	}
	if req.GowaDeviceID != "" {
		account.GowaDeviceID = req.GowaDeviceID
	}
	if req.GowaWebhookSecret != "" {
		enc, err := crypto.Encrypt(req.GowaWebhookSecret, a.Config.App.EncryptionKey)
		if err != nil {
			a.Log.Error("Failed to encrypt GOWA webhook secret", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
		}
		account.GowaWebhookSecret = enc
	} else if account.GowaWebhookSecret == "" {
		// Auto-generate webhook secret for accounts that don't have one (FR-017).
		secret := gowa.GenerateWebhookSecret()
		enc, err := crypto.Encrypt(secret, a.Config.App.EncryptionKey)
		if err != nil {
			a.Log.Error("Failed to encrypt auto-generated GOWA webhook secret", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
		}
		account.GowaWebhookSecret = enc
	}

	account.AutoReadReceipt = req.AutoReadReceipt

	// Handle default flags
	if req.IsDefaultIncoming && !account.IsDefaultIncoming {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing && !account.IsDefaultOutgoing {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}
	account.IsDefaultIncoming = req.IsDefaultIncoming
	account.IsDefaultOutgoing = req.IsDefaultOutgoing
	account.UpdatedByID = &userID

	if err := a.DB.Save(account).Error; err != nil {
		a.Log.Error("Failed to update account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
	}

	// Invalidate cache
	a.InvalidateWhatsAppAccountCache(account.GowaDeviceID)

	a.DB.Preload("CreatedBy").Preload("UpdatedBy").First(account, "id = ?", account.ID)

	a.logAudit(orgID, userID,
		"account", account.ID, models.AuditActionUpdated, &oldAccount, account)

	return r.SendEnvelope(accountToResponse(*account))
}

// DeleteAccount deletes a WhatsApp account
func (a *App) DeleteAccount(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionDelete)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	// Get account first for cache invalidation and audit
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	if err := a.DB.Delete(account).Error; err != nil {
		a.Log.Error("Failed to delete account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete account", nil, "")
	}

	// Invalidate cache
	a.InvalidateWhatsAppAccountCache(account.GowaDeviceID)

	a.logAudit(orgID, userID,
		"account", id, models.AuditActionDeleted, account, nil)

	return r.SendEnvelope(map[string]string{"message": "Account deleted successfully"})
}

// Helper functions

func accountToResponse(acc models.WhatsAppAccount) AccountResponse {
	resp := AccountResponse{
		ID:                   acc.ID,
		Name:                 acc.Name,
		GowaBaseURL:          acc.GowaBaseURL,
		GowaDeviceID:         acc.GowaDeviceID,
		HasGowaWebhookSecret: acc.GowaWebhookSecret != "",
		IsDefaultIncoming:    acc.IsDefaultIncoming,
		IsDefaultOutgoing:    acc.IsDefaultOutgoing,
		AutoReadReceipt:      acc.AutoReadReceipt,
		Status:               acc.Status,
		CreatedByID:          acc.CreatedByID,
		UpdatedByID:          acc.UpdatedByID,
		CreatedAt:            acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            acc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if acc.CreatedBy != nil {
		resp.CreatedByName = acc.CreatedBy.FullName
	}
	if acc.UpdatedBy != nil {
		resp.UpdatedByName = acc.UpdatedBy.FullName
	}
	return resp
}

// enrichGowaStatus fetches the live connection state from GOWA and populates
// GowaJID / GowaConnected on the response. Best-effort: if GOWA is unreachable,
// the fields stay empty and the response is still returned.
func (a *App) enrichGowaStatus(ctx context.Context, resp *AccountResponse, acc *models.WhatsAppAccount) {
	provider := a.resolveProvider(acc)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return
	}
	status, err := gowaClient.GetAppStatus(ctx, acc.GowaDeviceID)
	if err != nil {
		return // GOWA unreachable — leave fields empty
	}
	connected := status.IsConnected
	resp.GowaConnected = &connected
	resp.GowaJID = status.JID
}

func (a *App) encryptAccountSecrets(account *models.WhatsAppAccount) error {
	return crypto.EncryptFields(a.Config.App.EncryptionKey,
		&account.GowaWebhookSecret)
}
