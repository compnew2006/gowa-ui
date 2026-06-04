package handlers

import (
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type FBPageSearchResponse struct {
	Name           string `json:"name"`
	PageID         string `json:"page_id"`
	FollowersCount string `json:"followers_count"`
}

func (a *App) SearchFBPages(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	campaignID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("campaign_id")))
	if campaignID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "campaign_id is required", nil, "campaign_id")
	}

	page, err := a.parseIntQuery(r, "page", 1, 1, 1<<31-1)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid page", nil, "page")
	}

	perPage, err := a.parseIntQuery(r, "per_page", 25, 1, 200)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid per_page (1-200)", nil, "per_page")
	}

	q := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))

	var total int64
	countQuery := requestDB.Model(&models.FBPageSearch{}).
		Where("organization_id = ? AND campaign_id = ?", orgID, campaignID)
	if q != "" {
		like := "%" + q + "%"
		countQuery = countQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		a.Log.Error("Failed to count FB page searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search pages", nil, "")
	}

	var rows []models.FBPageSearch
	listQuery := requestDB.Model(&models.FBPageSearch{}).
		Where("organization_id = ? AND campaign_id = ?", orgID, campaignID).
		Order("created_at DESC, id DESC").
		Limit(perPage).
		Offset((page - 1) * perPage)
	if q != "" {
		like := "%" + q + "%"
		listQuery = listQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := listQuery.Find(&rows).Error; err != nil {
		a.Log.Error("Failed to list FB page searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search pages", nil, "")
	}

	data := make([]FBPageSearchResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, FBPageSearchResponse{
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

func (a *App) parseIntQuery(r *fastglue.Request, name string, def, min, max int) (int, error) {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek(name)))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if v < min {
		v = min
	}
	if v > max {
		v = max
	}
	return v, nil
}
