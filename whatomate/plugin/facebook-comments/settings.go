package facebookcomments

import (
	"strings"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/plugin/facebook-comments/commentdata"
)

func (p *Plugin) GetFacebookCommentSettings(r *fastglue.Request) error {
	requestDB, orgID, ok := p.commentContext(r, models.ActionRead)
	if !ok {
		return nil
	}

	settings, err := commentdata.GetOrCreateSettings(requestDB, orgID)
	if err != nil {
		p.app.Log.Error("Failed to load Facebook comment settings", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load Facebook comment settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (p *Plugin) UpdateFacebookCommentSettings(r *fastglue.Request) error {
	requestDB, orgID, ok := p.commentContext(r, models.ActionWrite)
	if !ok {
		return nil
	}

	var req commentdata.SettingsRequest
	if !decodeRequest(r, &req) {
		return nil
	}
	settings, err := commentdata.GetOrCreateSettings(requestDB, orgID)
	if err != nil {
		p.app.Log.Error("Failed to load Facebook comment settings for update", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update Facebook comment settings", nil, "")
	}

	commentdata.ApplySettingsRequest(settings, req)
	if err := requestDB.Save(settings).Error; err != nil {
		p.app.Log.Error("Failed to save Facebook comment settings", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update Facebook comment settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (p *Plugin) commentContext(r *fastglue.Request, action string) (*gorm.DB, uuid.UUID, bool) {
	orgID, orgOK := middleware.GetOrganizationID(r)
	userID, userOK := middleware.GetUserID(r)
	if !orgOK || !userOK || orgID == uuid.Nil || userID == uuid.Nil || p.app == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return nil, uuid.Nil, false
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, action, orgID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return nil, uuid.Nil, false
	}
	return tenant.ScopedDB(p.app.DB, orgID), orgID, true
}

func decodeRequest(r *fastglue.Request, v any) bool {
	if err := r.Decode(v, "json"); err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		return false
	}
	return true
}

// GetPageCommentSettings handles GET /api/facebook/comments/pages/{page_id}/settings.
func (p *Plugin) GetPageCommentSettings(r *fastglue.Request) error {
	requestDB, orgID, ok := p.commentContext(r, models.ActionRead)
	if !ok {
		return nil
	}
	pageID := pageCommentID(r)
	if pageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "page_id is required", nil, "")
	}
	settings, err := commentdata.GetOrCreatePageSettings(requestDB, orgID, pageID, p.resolvePageAccount)
	if err != nil {
		p.app.Log.Error("Failed to load Facebook page comment settings", "error", err, "organization_id", orgID, "page_id", pageID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load page comment settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

// UpdatePageCommentSettings handles PUT /api/facebook/comments/pages/{page_id}/settings.
func (p *Plugin) UpdatePageCommentSettings(r *fastglue.Request) error {
	requestDB, orgID, ok := p.commentContext(r, models.ActionWrite)
	if !ok {
		return nil
	}
	pageID := pageCommentID(r)
	if pageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "page_id is required", nil, "")
	}
	var req commentdata.PageSettingsRequest
	if !decodeRequest(r, &req) {
		return nil
	}
	settings, err := commentdata.GetOrCreatePageSettings(requestDB, orgID, pageID, p.resolvePageAccount)
	if err != nil {
		p.app.Log.Error("Failed to load page settings for update", "error", err, "org_id", orgID, "page_id", pageID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update page settings", nil, "")
	}
	commentdata.ApplyPageSettingsRequest(settings, req)
	if err := requestDB.Save(settings).Error; err != nil {
		p.app.Log.Error("Failed to save page comment settings", "error", err, "org_id", orgID, "page_id", pageID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update page settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

// pageCommentID extracts the {page_id} route parameter.
func pageCommentID(r *fastglue.Request) string {
	pageID, _ := r.RequestCtx.UserValue("page_id").(string)
	return strings.TrimSpace(pageID)
}

// resolvePageAccount preserves the historical account-resolution behavior:
// findFacebookAccountByPageID first, then the first active OAuth account in
// the org as a fallback. Returns uuid.Nil when no account is available.
func (p *Plugin) resolvePageAccount(_ *gorm.DB, orgID uuid.UUID, pageID string) uuid.UUID {
	if p.app == nil {
		return uuid.Nil
	}
	if acct, _, err := p.app.FindFacebookAccountByPageID(pageID); err == nil && acct != nil {
		return acct.ID
	}
	var firstAcct models.FacebookAccount
	if err := p.app.DB.Where("organization_id = ? AND method = ? AND status = ?",
		orgID, models.FBAccountMethodOAuth, models.FBAccountStatusActive).
		Order("created_at ASC").First(&firstAcct).Error; err == nil {
		return firstAcct.ID
	}
	return uuid.Nil
}
