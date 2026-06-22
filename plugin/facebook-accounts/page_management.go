package facebookaccounts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/plugin/facebook-accounts/accountdata"
	facebookgraph "github.com/compnew2006/whatomate/plugin/facebook-core/graph"
)

type facebookPagesResponse struct {
	Data []map[string]any `json:"data"`
}

func (p *Plugin) RefreshFacebookAccountPages(r *fastglue.Request) error {
	requestDB, account, _, ok := p.pageManagementContext(r)
	if !ok {
		return nil
	}
	userToken, err := crypto.Decrypt(account.AccessToken, p.app.Config.App.EncryptionKey)
	if err != nil {
		p.app.Log.Error("Failed to decrypt Facebook OAuth user token", "error", err, "account_id", account.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook user token", nil, "")
	}
	graphConfig, err := p.graphRuntimeConfig()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	pages, refreshedTokens, err := p.fetchFacebookPages(graphConfig, userToken)
	if err != nil {
		p.app.Log.Error("Failed to refresh Facebook pages", "error", err, "account_id", account.ID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to refresh Facebook pages", nil, "")
	}
	pageTokens, err := p.decryptPageTokens(*account)
	if err != nil {
		p.app.Log.Error("Failed to decrypt Facebook page tokens", "error", err, "account_id", account.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook page tokens", nil, "")
	}
	for pageID := range pageTokens {
		if token := strings.TrimSpace(refreshedTokens[pageID]); token != "" {
			pageTokens[pageID] = token
		}
	}
	pages = markPagesConnected(pages, pageTokens)
	updated, err := p.updateAccountPages(requestDB, *account, pages, pageTokens)
	if err != nil {
		p.app.Log.Error("Failed to save refreshed Facebook pages", "error", err, "account_id", account.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save Facebook pages", nil, "")
	}
	return r.SendEnvelope(accountdata.ToResponse(*updated))
}

func (p *Plugin) ConnectFacebookAccountPage(r *fastglue.Request) error {
	return p.withPageManagement(r, func(account *models.FacebookAccount, pageTokens map[string]string, pageID string) ([]map[string]any, error) {
		userToken, err := crypto.Decrypt(account.AccessToken, p.app.Config.App.EncryptionKey)
		if err != nil {
			p.app.Log.Error("Failed to decrypt Facebook OAuth user token", "error", err, "account_id", account.ID)
			return nil, r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook user token", nil, "")
		}
		graphConfig, err := p.graphRuntimeConfig()
		if err != nil {
			return nil, r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
		fetchedPages, fetchedTokens, err := p.fetchFacebookPages(graphConfig, userToken)
		if err != nil {
			p.app.Log.Error("Failed to fetch Facebook pages for connect", "error", err, "account_id", account.ID, "page_id", pageID)
			return nil, r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to fetch Facebook pages", nil, "")
		}
		pageToken := strings.TrimSpace(fetchedTokens[pageID])
		if pageToken == "" {
			return nil, r.SendErrorEnvelope(fasthttp.StatusNotFound, "Page token not found for this account", nil, "page_id")
		}
		pageTokens[pageID] = pageToken
		return mergePages(account.Data, fetchedPages), nil
	})
}

func (p *Plugin) DisconnectFacebookAccountPage(r *fastglue.Request) error {
	return p.withPageManagement(r, func(account *models.FacebookAccount, pageTokens map[string]string, pageID string) ([]map[string]any, error) {
		delete(pageTokens, pageID)
		return pagesFromData(account.Data), nil
	})
}

func (p *Plugin) RemoveFacebookAccountPage(r *fastglue.Request) error {
	return p.withPageManagement(r, func(account *models.FacebookAccount, pageTokens map[string]string, pageID string) ([]map[string]any, error) {
		delete(pageTokens, pageID)
		pages, removed := removePage(pagesFromData(account.Data), pageID)
		if !removed {
			return nil, r.SendErrorEnvelope(fasthttp.StatusNotFound, "Page not found for this account", nil, "page_id")
		}
		return pages, nil
	})
}

func (p *Plugin) withPageManagement(
	r *fastglue.Request,
	pageAction func(account *models.FacebookAccount, pageTokens map[string]string, pageID string) ([]map[string]any, error),
) error {
	requestDB, account, pageID, ok := p.pageManagementContext(r)
	if !ok {
		return nil
	}
	pageTokens, err := p.decryptPageTokens(*account)
	if err != nil {
		p.app.Log.Error("Failed to decrypt Facebook page tokens", "error", err, "account_id", account.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook page tokens", nil, "")
	}
	pages, err := pageAction(account, pageTokens, pageID)
	if err != nil {
		return err
	}
	if !pagesContain(pages, pageID) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Page not found for this account", nil, "page_id")
	}
	pages = markPagesConnected(pages, pageTokens)
	updated, err := p.updateAccountPages(requestDB, *account, pages, pageTokens)
	if err != nil {
		p.app.Log.Error("Failed to save Facebook page update", "error", err, "account_id", account.ID, "page_id", pageID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save Facebook page update", nil, "")
	}
	return r.SendEnvelope(accountdata.ToResponse(*updated))
}

func (p *Plugin) pageManagementContext(r *fastglue.Request) (*gorm.DB, *models.FacebookAccount, string, bool) {
	requestDB, organizationID, ok := p.accountContext(r, models.ActionWrite)
	if !ok {
		return nil, nil, "", false
	}
	// Additional plugin-namespaced permission: destructive page lifecycle
	// (connect/disconnect/remove) requires plugin.facebook.accounts:pages_manage
	// on top of the base accounts:write granted by accountContext. Super-admins
	// bypass via HasPermission's IsSuperAdmin short-circuit.
	userID, _ := middleware.GetUserID(r)
	if !p.app.HasPermission(userID, resourceFBAccountsPagesManage, actionPagesManage, organizationID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions: plugin.facebook.accounts:pages_manage", nil, "")
		return nil, nil, "", false
	}
	account, ok := findAccount(r, requestDB, organizationID)
	if !ok {
		return nil, nil, "", false
	}
	if account.Method != models.FBAccountMethodOAuth || strings.TrimSpace(account.AccessToken) == "" {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Facebook OAuth account is required", nil, "")
		return nil, nil, "", false
	}
	pageID := strings.TrimSpace(fmt.Sprint(r.RequestCtx.UserValue("page_id")))
	return requestDB, account, pageID, true
}

func (p *Plugin) fetchFacebookPages(config graphRuntimeConfig, accessToken string) ([]map[string]any, map[string]string, error) {
	query := url.Values{
		"fields":       {"id,name,access_token,category,picture.width(200)"},
		"access_token": {accessToken},
	}
	endpoint := fmt.Sprintf("%s/%s/me/accounts?%s", config.BaseURL, config.APIVersion, query.Encode())
	var payload facebookPagesResponse
	if err := facebookgraph.New(p.app.OAuthHTTPClient()).JSONRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return nil, nil, err
	}

	pageTokens := make(map[string]string, len(payload.Data))
	safePages := make([]map[string]any, 0, len(payload.Data))
	for _, page := range payload.Data {
		pageID, _ := page["id"].(string)
		if pageID == "" {
			continue
		}
		if token, _ := page["access_token"].(string); token != "" {
			pageTokens[pageID] = token
			page["connected"] = true
		} else {
			page["connected"] = false
		}
		delete(page, "access_token")
		safePages = append(safePages, page)
	}
	return safePages, pageTokens, nil
}

func (p *Plugin) updateAccountPages(db *gorm.DB, account models.FacebookAccount, pages []map[string]any, pageTokens map[string]string) (*models.FacebookAccount, error) {
	var updated models.FacebookAccount
	err := db.Transaction(func(tx *gorm.DB) error {
		pageTokensJSON, err := json.Marshal(pageTokens)
		if err != nil {
			return err
		}
		encryptedPageTokens, err := crypto.Encrypt(string(pageTokensJSON), p.app.Config.App.EncryptionKey)
		if err != nil {
			return err
		}
		data := models.JSONB{}
		for key, value := range account.Data {
			data[key] = value
		}
		data["pages"] = pages
		data["page_count"] = len(pages)
		updates := map[string]any{
			"page_tokens": encryptedPageTokens,
			"data":        data,
		}
		if err := tx.Model(&models.FacebookAccount{}).
			Where("id = ? AND organization_id = ?", account.ID, account.OrganizationID).
			Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&updated, "id = ? AND organization_id = ?", account.ID, account.OrganizationID).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func mergePages(currentData models.JSONB, fetchedPages []map[string]any) []map[string]any {
	merged := pagesFromData(currentData)
	indexByID := map[string]int{}
	for index, page := range merged {
		if pageID := strings.TrimSpace(fmt.Sprint(page["id"])); pageID != "" {
			indexByID[pageID] = index
		}
	}
	for _, fetched := range fetchedPages {
		pageID := strings.TrimSpace(fmt.Sprint(fetched["id"]))
		if pageID == "" {
			continue
		}
		clean := clonePage(fetched)
		if index, ok := indexByID[pageID]; ok {
			for key, value := range clean {
				if key != "connected" {
					merged[index][key] = value
				}
			}
			continue
		}
		clean["connected"] = false
		merged = append(merged, clean)
		indexByID[pageID] = len(merged) - 1
	}
	return merged
}

func pagesFromData(data models.JSONB) []map[string]any {
	if data == nil {
		return []map[string]any{}
	}
	rawPages, ok := data["pages"].([]any)
	if !ok {
		if typedPages, ok := data["pages"].([]map[string]any); ok {
			pages := make([]map[string]any, 0, len(typedPages))
			for _, page := range typedPages {
				pages = append(pages, clonePage(page))
			}
			return pages
		}
		return []map[string]any{}
	}
	pages := make([]map[string]any, 0, len(rawPages))
	for _, rawPage := range rawPages {
		if page, ok := rawPage.(map[string]any); ok {
			pages = append(pages, clonePage(page))
		}
	}
	return pages
}

func clonePage(page map[string]any) map[string]any {
	clean := make(map[string]any, len(page))
	for key, value := range page {
		if key == "access_token" {
			continue
		}
		clean[key] = value
	}
	return clean
}

func markPagesConnected(pages []map[string]any, pageTokens map[string]string) []map[string]any {
	marked := make([]map[string]any, 0, len(pages))
	for _, page := range pages {
		clean := clonePage(page)
		pageID := strings.TrimSpace(fmt.Sprint(clean["id"]))
		clean["connected"] = pageID != "" && strings.TrimSpace(pageTokens[pageID]) != ""
		marked = append(marked, clean)
	}
	return marked
}

func pagesContain(pages []map[string]any, pageID string) bool {
	for _, page := range pages {
		if strings.TrimSpace(fmt.Sprint(page["id"])) == pageID {
			return true
		}
	}
	return false
}

func removePage(pages []map[string]any, pageID string) ([]map[string]any, bool) {
	filtered := make([]map[string]any, 0, len(pages))
	removed := false
	for _, page := range pages {
		if strings.TrimSpace(fmt.Sprint(page["id"])) == pageID {
			removed = true
			continue
		}
		filtered = append(filtered, clonePage(page))
	}
	return filtered, removed
}
