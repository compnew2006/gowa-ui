package handlers

import (
	"errors"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type FBCreateAccountRequest struct {
	Name        string                       `json:"name"`
	AccountUID  string                       `json:"account_uid"`
	Method      models.FacebookAccountMethod `json:"method"`
	CookiesText string                       `json:"cookies_text"`
	Data        models.JSONB                 `json:"data"`
}

type FBUpdateAccountRequest struct {
	Name        *string                       `json:"name"`
	AccountUID  *string                       `json:"account_uid"`
	Status      *models.FacebookAccountStatus `json:"status"`
	Method      *models.FacebookAccountMethod `json:"method"`
	CookiesText *string                       `json:"cookies_text"`
	Data        *models.JSONB                 `json:"data"`
}

type FBAccountResponse struct {
	ID             uuid.UUID                    `json:"id"`
	Name           string                       `json:"name"`
	AccountUID     string                       `json:"account_uid"`
	Platform       string                       `json:"platform"`
	Email          string                       `json:"email,omitempty"`
	AvatarURL      string                       `json:"avatar_url,omitempty"`
	Status         models.FacebookAccountStatus `json:"status"`
	Method         models.FacebookAccountMethod `json:"method"`
	Data           models.JSONB                 `json:"data"`
	HasCookies     bool                         `json:"has_cookies"`
	OAuthConnected bool                         `json:"oauth_connected"`
	TokenExpiresAt string                       `json:"token_expires_at,omitempty"`
	ConnectedAt    string                       `json:"connected_at,omitempty"`
	LastRenewedAt  string                       `json:"last_renewed_at,omitempty"`
	PageCount      int                          `json:"page_count"`
	CreatedAt      string                       `json:"created_at"`
	UpdatedAt      string                       `json:"updated_at"`
}

func fbAccountToResponse(account models.FacebookAccount) FBAccountResponse {
	tokenExpiresAt := ""
	if account.TokenExpiresAt != nil {
		tokenExpiresAt = account.TokenExpiresAt.Format("2006-01-02T15:04:05Z")
	}
	connectedAt := ""
	if account.ConnectedAt != nil {
		connectedAt = account.ConnectedAt.Format("2006-01-02T15:04:05Z")
	}
	lastRenewedAt := ""
	if account.LastRenewedAt != nil {
		lastRenewedAt = account.LastRenewedAt.Format("2006-01-02T15:04:05Z")
	}
	pageCount := 0
	if account.Data != nil {
		if rawPageCount, ok := account.Data["page_count"].(int); ok {
			pageCount = rawPageCount
		} else if rawPageCount, ok := account.Data["page_count"].(float64); ok {
			pageCount = int(rawPageCount)
		} else if pages, ok := account.Data["pages"].([]interface{}); ok {
			pageCount = len(pages)
		}
	}

	return FBAccountResponse{
		ID:             account.ID,
		Name:           account.Name,
		AccountUID:     account.AccountUID,
		Platform:       account.Platform,
		Email:          account.Email,
		AvatarURL:      account.AvatarURL,
		Status:         account.Status,
		Method:         account.Method,
		Data:           account.Data,
		HasCookies:     account.CookiesText != "",
		OAuthConnected: account.AccessToken != "",
		TokenExpiresAt: tokenExpiresAt,
		ConnectedAt:    connectedAt,
		LastRenewedAt:  lastRenewedAt,
		PageCount:      pageCount,
		CreatedAt:      account.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      account.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (a *App) ListFBAccounts(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	var accounts []models.FacebookAccount
	if err := requestDB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to list Facebook accounts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list Facebook accounts", nil, "")
	}

	response := make([]FBAccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = fbAccountToResponse(acc)
	}

	return r.SendEnvelope(map[string]interface{}{
		"accounts": response,
	})
}

func (a *App) CreateFBAccount(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	var req FBCreateAccountRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "name")
	}

	if req.Method == "" {
		req.Method = models.FBAccountMethodCookies
	}

	if req.Data == nil {
		req.Data = models.JSONB{}
	}

	encKey := a.Config.App.EncryptionKey
	encCookiesText := ""
	if req.CookiesText != "" {
		encCookiesText, err = crypto.Encrypt(req.CookiesText, encKey)
		if err != nil {
			a.Log.Error("Failed to encrypt cookies", "error", err)
			if errors.Is(err, crypto.ErrMissingEncryptionKey) {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "app.encryption_key must be configured before creating accounts", nil, "")
			}
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
		}
	}

	account := models.FacebookAccount{
		OrganizationID: orgID,
		Name:           req.Name,
		AccountUID:     req.AccountUID,
		Status:         models.FBAccountStatusActive,
		Method:         req.Method,
		CookiesText:    encCookiesText,
		Data:           req.Data,
	}

	if err := requestDB.Create(&account).Error; err != nil {
		a.Log.Error("Failed to create Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	return r.SendEnvelope(fbAccountToResponse(account))
}

func (a *App) GetFBAccount(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "Facebook account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.FacebookAccount](requestDB, r, id, orgID, "Facebook account")
	if err != nil {
		return nil
	}

	return r.SendEnvelope(fbAccountToResponse(*account))
}

func (a *App) UpdateFBAccount(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "Facebook account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.FacebookAccount](requestDB, r, id, orgID, "Facebook account")
	if err != nil {
		return nil
	}

	var req FBUpdateAccountRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	updates := map[string]interface{}{}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.AccountUID != nil {
		updates["account_uid"] = *req.AccountUID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Method != nil {
		updates["method"] = *req.Method
	}
	if req.CookiesText != nil {
		encKey := a.Config.App.EncryptionKey
		if *req.CookiesText != "" {
			encCookies, encErr := crypto.Encrypt(*req.CookiesText, encKey)
			if encErr != nil {
				a.Log.Error("Failed to encrypt cookies", "error", encErr)
				if errors.Is(encErr, crypto.ErrMissingEncryptionKey) {
					return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "app.encryption_key must be configured", nil, "")
				}
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
			}
			updates["cookies_text"] = encCookies
		} else {
			updates["cookies_text"] = ""
		}
	}
	if req.Data != nil {
		merged := make(models.JSONB, len(account.Data)+len(*req.Data))
		for k, v := range account.Data {
			merged[k] = v
		}
		for k, v := range *req.Data {
			merged[k] = v
		}
		updates["data"] = merged
	}

	if err := requestDB.Model(account).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
	}

	if err := requestDB.First(account, "id = ?", id).Error; err != nil {
		a.Log.Error("Failed to reload Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reload account", nil, "")
	}

	return r.SendEnvelope(fbAccountToResponse(*account))
}

func (a *App) DeleteFBAccount(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionDelete); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "Facebook account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.FacebookAccount](requestDB, r, id, orgID, "Facebook account")
	if err != nil {
		return nil
	}

	if err := requestDB.Delete(account).Error; err != nil {
		a.Log.Error("Failed to delete Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete account", nil, "")
	}

	return r.SendEnvelope(map[string]string{"status": "deleted"})
}
