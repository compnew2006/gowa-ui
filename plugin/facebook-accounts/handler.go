package facebookaccounts

import (
	"errors"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/plugin/facebook-accounts/accountdata"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type createAccountRequest struct {
	Name        string                       `json:"name"`
	AccountUID  string                       `json:"account_uid"`
	Method      models.FacebookAccountMethod `json:"method"`
	CookiesText string                       `json:"cookies_text"`
	Data        models.JSONB                 `json:"data"`
}

type updateAccountRequest struct {
	Name        *string                       `json:"name"`
	AccountUID  *string                       `json:"account_uid"`
	Status      *models.FacebookAccountStatus `json:"status"`
	Method      *models.FacebookAccountMethod `json:"method"`
	CookiesText *string                       `json:"cookies_text"`
	Data        *models.JSONB                 `json:"data"`
}

func (p *Plugin) accountContext(r *fastglue.Request, action string) (*gorm.DB, uuid.UUID, bool) {
	organizationID, organizationOK := middleware.GetOrganizationID(r)
	userID, userOK := middleware.GetUserID(r)
	if !organizationOK || !userOK || organizationID == uuid.Nil || userID == uuid.Nil || p.app == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return nil, uuid.Nil, false
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, action, organizationID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return nil, uuid.Nil, false
	}
	return tenant.ScopedDB(p.app.DB, organizationID), organizationID, true
}

func (p *Plugin) ListFBAccounts(r *fastglue.Request) error {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionRead)
	if !ok {
		return nil
	}
	var accounts []models.FacebookAccount
	if err := requestDB.Where("organization_id = ?", organizationID).
		Order("created_at DESC").
		Find(&accounts).Error; err != nil {
		p.app.Log.Error("Failed to list Facebook accounts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list Facebook accounts", nil, "")
	}
	response := make([]accountdata.Response, len(accounts))
	for index, account := range accounts {
		response[index] = accountdata.ToResponse(account)
	}
	return r.SendEnvelope(map[string]interface{}{"accounts": response})
}

func (p *Plugin) CreateFBAccount(r *fastglue.Request) error {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionWrite)
	if !ok {
		return nil
	}
	var request createAccountRequest
	if err := r.Decode(&request, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if request.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "name")
	}
	if request.Method == "" {
		request.Method = models.FBAccountMethodCookies
	}
	if request.Data == nil {
		request.Data = models.JSONB{}
	}

	encryptedCookies := ""
	if request.CookiesText != "" {
		var err error
		encryptedCookies, err = crypto.Encrypt(request.CookiesText, p.app.Config.App.EncryptionKey)
		if err != nil {
			p.app.Log.Error("Failed to encrypt cookies", "error", err)
			if errors.Is(err, crypto.ErrMissingEncryptionKey) {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "app.encryption_key must be configured before creating accounts", nil, "")
			}
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
		}
	}

	account := models.FacebookAccount{
		OrganizationID: organizationID,
		Name:           request.Name,
		AccountUID:     request.AccountUID,
		Status:         models.FBAccountStatusActive,
		Method:         request.Method,
		CookiesText:    encryptedCookies,
		Data:           request.Data,
	}
	if err := requestDB.Create(&account).Error; err != nil {
		p.app.Log.Error("Failed to create Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}
	return r.SendEnvelope(accountdata.ToResponse(account))
}

func (p *Plugin) GetFBAccount(r *fastglue.Request) error {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionRead)
	if !ok {
		return nil
	}
	account, ok := findAccount(r, requestDB, organizationID)
	if !ok {
		return nil
	}
	return r.SendEnvelope(accountdata.ToResponse(*account))
}

func (p *Plugin) UpdateFBAccount(r *fastglue.Request) error {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionWrite)
	if !ok {
		return nil
	}
	account, ok := findAccount(r, requestDB, organizationID)
	if !ok {
		return nil
	}
	var request updateAccountRequest
	if err := r.Decode(&request, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	updates := map[string]interface{}{}
	if request.Name != nil {
		updates["name"] = *request.Name
	}
	if request.AccountUID != nil {
		updates["account_uid"] = *request.AccountUID
	}
	if request.Status != nil {
		updates["status"] = *request.Status
	}
	if request.Method != nil {
		updates["method"] = *request.Method
	}
	if request.CookiesText != nil {
		if *request.CookiesText == "" {
			updates["cookies_text"] = ""
		} else {
			encryptedCookies, err := crypto.Encrypt(*request.CookiesText, p.app.Config.App.EncryptionKey)
			if err != nil {
				p.app.Log.Error("Failed to encrypt cookies", "error", err)
				if errors.Is(err, crypto.ErrMissingEncryptionKey) {
					return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "app.encryption_key must be configured", nil, "")
				}
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
			}
			updates["cookies_text"] = encryptedCookies
		}
	}
	if request.Data != nil {
		merged := make(models.JSONB, len(account.Data)+len(*request.Data))
		for key, value := range account.Data {
			merged[key] = value
		}
		for key, value := range *request.Data {
			merged[key] = value
		}
		updates["data"] = merged
	}
	if err := requestDB.Model(account).Updates(updates).Error; err != nil {
		p.app.Log.Error("Failed to update Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
	}
	if err := requestDB.First(account, "id = ?", account.ID).Error; err != nil {
		p.app.Log.Error("Failed to reload Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reload account", nil, "")
	}
	return r.SendEnvelope(accountdata.ToResponse(*account))
}

func (p *Plugin) DeleteFBAccount(r *fastglue.Request) error {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionDelete)
	if !ok {
		return nil
	}
	account, ok := findAccount(r, requestDB, organizationID)
	if !ok {
		return nil
	}
	if err := requestDB.Delete(account).Error; err != nil {
		p.app.Log.Error("Failed to delete Facebook account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete account", nil, "")
	}
	return r.SendEnvelope(map[string]string{"status": "deleted"})
}

func findAccount(r *fastglue.Request, db *gorm.DB, organizationID uuid.UUID) (*models.FacebookAccount, bool) {
	rawID, ok := r.RequestCtx.UserValue("id").(string)
	if !ok {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook account ID", nil, "id")
		return nil, false
	}
	accountID, err := uuid.Parse(rawID)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook account ID", nil, "id")
		return nil, false
	}
	var account models.FacebookAccount
	err = db.Where("id = ? AND organization_id = ?", accountID, organizationID).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Facebook account not found", nil, "")
		return nil, false
	}
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load Facebook account", nil, "")
		return nil, false
	}
	return &account, true
}
