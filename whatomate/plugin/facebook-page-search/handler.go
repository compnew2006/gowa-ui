package facebookpagesearch

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	facebookcore "github.com/compnew2006/whatomate/plugin/facebook-core"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type PageSearchResponse struct {
	Name           string `json:"name"`
	PageID         string `json:"page_id"`
	FollowersCount string `json:"followers_count"`
}

func (p *Plugin) SearchFBPages(r *fastglue.Request) error {
	organizationID, organizationOK := middleware.GetOrganizationID(r)
	userID, userOK := middleware.GetUserID(r)
	if !organizationOK || !userOK || organizationID == uuid.Nil || userID == uuid.Nil || p.app == nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, models.ActionRead, organizationID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}
	requestDB := tenant.ScopedDB(p.app.DB, organizationID)

	campaignID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("campaign_id")))
	if campaignID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "campaign_id is required", nil, "campaign_id")
	}

	page, err := facebookcore.ParseIntQuery(r, "page", 1, 1, 1<<31-1)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid page", nil, "page")
	}
	perPage, err := facebookcore.ParseIntQuery(r, "per_page", 25, 1, 200)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid per_page (1-200)", nil, "per_page")
	}

	queryText := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	var total int64
	countQuery := requestDB.Model(&PageSearch{}).
		Where("organization_id = ? AND campaign_id = ?", organizationID, campaignID)
	if queryText != "" {
		like := "%" + queryText + "%"
		countQuery = countQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		p.app.Log.Error("Failed to count FB page searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search pages", nil, "")
	}

	var rows []PageSearch
	listQuery := requestDB.Model(&PageSearch{}).
		Where("organization_id = ? AND campaign_id = ?", organizationID, campaignID).
		Order("created_at DESC, id DESC").
		Limit(perPage).
		Offset((page - 1) * perPage)
	if queryText != "" {
		like := "%" + queryText + "%"
		listQuery = listQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := listQuery.Find(&rows).Error; err != nil {
		p.app.Log.Error("Failed to list FB page searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search pages", nil, "")
	}

	data := make([]PageSearchResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, PageSearchResponse{
			Name:           row.Name,
			PageID:         row.PageID,
		FollowersCount: row.FollowersCount,
		})
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return r.SendEnvelope(map[string]interface{}{
		"success":     true,
		"campaign_id": campaignID,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
		"data":        data,
	})
}

