package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/contactutil"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// DirectoryImportRequest represents a request to import directory groups into a campaign.
type DirectoryImportRequest struct {
	CampaignID string   `json:"campaign_id" validate:"required"`
	GroupIDs   []string `json:"group_ids" validate:"required,min=1"`
}

func (a *App) SearchGroupDirectory(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	q := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	country := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("country")))
	category := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("category")))
	pg := parsePaginationWithDefaults(r, 20, 100)

	query := a.DB.Where("organization_id = ?", orgID)
	if q != "" {
		query = query.Where("name ILIKE ?", "%"+q+"%")
	}
	if country != "" {
		query = query.Where("country = ?", country)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var total int64
	if err := query.Model(&models.GroupDirectory{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count groups", nil, "")
	}

	var directories []models.GroupDirectory
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&directories).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search groups", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"data":  directories,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

func (a *App) GetGroupDirectoryCategories(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var categories []string
	if err := a.DB.Model(&models.GroupDirectory{}).
		Where("organization_id = ? AND category != ''", orgID).
		Distinct("category").
		Order("category").
		Pluck("category", &categories).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load categories", nil, "")
	}

	return r.SendEnvelope(categories)
}

func (a *App) GetGroupDirectoryCountries(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var countries []string
	if err := a.DB.Model(&models.GroupDirectory{}).
		Where("organization_id = ? AND country != ''", orgID).
		Distinct("country").
		Order("country").
		Pluck("country", &countries).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load countries", nil, "")
	}

	return r.SendEnvelope(countries)
}

func (a *App) CreateGroupDirectory(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req models.GroupDirectory
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	req.OrganizationID = orgID
	req.ID = uuid.Nil

	if req.GroupJID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "group_jid is required", nil, "")
	}
	if !contactutil.IsValidGroupJID(req.GroupJID) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid group JID format", nil, "")
	}
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name is required", nil, "")
	}

	if err := a.DB.Create(&req).Error; err != nil {
		a.Log.Error("Failed to create group directory entry", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create group entry", nil, "")
	}

	return r.SendEnvelope(req)
}

func (a *App) UpdateGroupDirectory(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "group directory")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupDirectory](a.DB, r, id, orgID, "Group directory")
	if err != nil {
		return nil
	}
	_ = existing

	var req models.GroupDirectory
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["country"] = req.Country
	updates["language"] = req.Language
	updates["category"] = req.Category
	updates["image_url"] = req.ImageURL
	updates["join_link"] = req.JoinLink
	updates["participant_count"] = req.ParticipantCount

	if err := a.DB.Model(&models.GroupDirectory{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(updates).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update group entry", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Group directory entry updated"})
}

func (a *App) DeleteGroupDirectory(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "group directory")
	if err != nil {
		return nil
	}

	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.GroupDirectory{}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete group entry", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Group directory entry deleted"})
}

// GroupLinkPreviewRequest represents a request to preview a WhatsApp group from an invite link.
type GroupLinkPreviewRequest struct {
	InstanceID string `json:"instance_id" validate:"required"`
	InviteLink string `json:"invite_link" validate:"required"`
}

func (a *App) PreviewGroupFromLink(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req GroupLinkPreviewRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	if req.InviteLink == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invite_link is required", nil, "")
	}

	// Verify the instance belongs to the organization.
	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", req.InstanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "WhatsApp instance not found", nil, "")
	}

	gip, ok := a.MessageProvider.(provider.GroupInfoProvider)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Group info provider not available", nil, "")
	}

	info, err := gip.GetGroupInfoFromLink(r.RequestCtx, req.InstanceID, req.InviteLink)
	if err != nil {
		a.Log.Error("Failed to get group info from link", "error", err, "link", req.InviteLink)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch group info from link", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"jid":              info.JID,
		"name":             info.Name,
		"participant_count": info.ParticipantCount,
		"invite_link":      req.InviteLink,
	})
}

func (a *App) ImportDirectoryGroupsToCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req DirectoryImportRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	campaignID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid campaign ID", nil, "")
	}

	requestDB := a.requestDB(r)
	var campaign models.BulkMessageCampaign
	if err := requestDB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	if campaign.Status != models.CampaignStatusDraft && campaign.Status != models.CampaignStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only import groups to draft or paused campaigns", nil, "")
	}

	var groups []models.GroupDirectory
	if err := a.DB.Where("id IN ? AND organization_id = ?", req.GroupIDs, orgID).Find(&groups).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load groups", nil, "")
	}

	if len(groups) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No valid groups found", nil, "")
	}

	recipients := make([]models.BulkMessageRecipient, 0, len(groups))
	for _, g := range groups {
		recipients = append(recipients, models.BulkMessageRecipient{
			CampaignID:        campaign.ID,
			PhoneNumber:      g.GroupJID,
			RecipientName:    g.Name,
			RecipientType:     models.RecipientTypeGroup,
			GroupJID:          g.GroupJID,
			GroupName:         g.Name,
			ParticipantCount:  g.ParticipantCount,
			Status:            models.MessageStatusPending,
		})
	}

	if err := requestDB.Create(&recipients).Error; err != nil {
		a.Log.Error("Failed to import group recipients", "error", err, "campaign_id", campaignID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import groups", nil, "")
	}

	if err := requestDB.Model(&campaign).
		Update("total_recipients", gorm.Expr("total_recipients + ?", len(recipients))).Error; err != nil {
		a.Log.Error("Failed to update campaign recipient count", "error", err, "campaign_id", campaignID)
	}

	return r.SendEnvelope(map[string]any{
		"message":     "Groups imported to campaign",
		"added_count": len(recipients),
	})
}
