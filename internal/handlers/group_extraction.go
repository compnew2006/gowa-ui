package handlers

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type GroupExtractionCampaignRequest struct {
	Name       string `json:"name" validate:"required"`
	InstanceID string `json:"instance_id" validate:"required"`
}

func (a *App) ListGroupExtractionCampaigns(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	status := string(r.RequestCtx.QueryArgs().Peek("status"))
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	query := a.DB.Where("organization_id = ?", orgID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Model(&models.GroupExtractionCampaign{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count campaigns", nil, "")
	}

	var campaigns []models.GroupExtractionCampaign
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&campaigns).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}

	return r.SendEnvelope(map[string]any{"data": campaigns, "total": total, "page": pg.Page, "limit": pg.Limit})
}

func (a *App) CreateGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}

	var req GroupExtractionCampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "")
	}
	if req.InstanceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance ID is required", nil, "")
	}

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance not found", nil, "")
	}

	campaign := models.GroupExtractionCampaign{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(req.Name),
		InstanceID:     instanceID,
		InstanceName:   instance.Name,
		Status:         models.GroupExtractionStatusDraft,
		CreatedBy:      userID,
	}

	if err := a.DB.Create(&campaign).Error; err != nil {
		a.Log.Error("Failed to create group extraction campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create campaign", nil, "")
	}

	return r.SendEnvelope(campaign)
}

func (a *App) GetGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	campaign, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(campaign)
}

func (a *App) UpdateGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.GroupExtractionStatusDraft && existing.Status != models.GroupExtractionStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only update draft or paused campaigns", nil, "")
	}

	var req GroupExtractionCampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.InstanceID != "" {
		instanceID, parseErr := uuid.Parse(req.InstanceID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
		}
		var instance models.WhatsAppInstance
		if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance not found", nil, "")
		}
		updates["instance_id"] = instanceID
		updates["instance_name"] = instance.Name
	}

	if err := a.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(updates).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign updated"})
}

func (a *App) DeleteGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionDelete); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status == models.GroupExtractionStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete a running campaign", nil, "")
	}
	a.DB.Where("campaign_id = ?", id).Delete(&models.GroupExtractionResult{})
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.GroupExtractionCampaign{}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign deleted"})
}

func (a *App) StartGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.GroupExtractionStatusDraft && existing.Status != models.GroupExtractionStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign cannot be started in current state", nil, "")
	}
	if existing.InstanceID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign has no instance configured", nil, "")
	}

	now := time.Now()
	if err := a.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(map[string]any{
		"status":     models.GroupExtractionStatusProcessing,
		"started_at": now,
	}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to start campaign", nil, "")
	}

	job := &queue.GroupExtractionJob{
		CampaignID:     id,
		OrganizationID: orgID,
		InstanceID:     existing.InstanceID,
		EnqueuedAt:     time.Now(),
	}
	if err := a.Queue.EnqueueGroupExtraction(r.RequestCtx, job); err != nil {
		a.Log.Error("Failed to enqueue group extraction job", "error", err, "campaign_id", id)
		a.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ?", id).Updates(map[string]any{"status": existing.Status, "started_at": nil})
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to queue job", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Campaign started"})
}

func (a *App) PauseGroupExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.GroupExtractionStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign is not processing", nil, "")
	}
	if err := a.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Update("status", models.GroupExtractionStatusPaused).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to pause campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign paused"})
}

func (a *App) GetGroupExtractionCampaignStats(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	id, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	campaign, err := findByIDAndOrg[models.GroupExtractionCampaign](a.DB, r, id, orgID, "Group extraction campaign")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(map[string]any{
		"total_groups":    campaign.TotalGroups,
		"extracted_count": campaign.ExtractedCount,
		"failed_count":    campaign.FailedCount,
		"status":          campaign.Status,
	})
}

func (a *App) GetGroupExtractionCampaignResults(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	campaignID, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	var campaign models.GroupExtractionCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	pg := parsePagination(r)
	statusFilter := string(r.RequestCtx.QueryArgs().Peek("status"))
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	query := a.DB.Where("campaign_id = ?", campaignID)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("group_jid LIKE ? OR group_name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Model(&models.GroupExtractionResult{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count results", nil, "")
	}

	var results []models.GroupExtractionResult
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&results).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list results", nil, "")
	}

	return r.SendEnvelope(map[string]any{"data": results, "total": total, "page": pg.Page, "limit": pg.Limit})
}

func (a *App) ExportGroupExtractionCampaignResults(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionRead); err != nil {
		return nil
	}
	campaignID, err := parsePathUUID(r, "id", "group extraction campaign")
	if err != nil {
		return nil
	}
	var campaign models.GroupExtractionCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	var results []models.GroupExtractionResult
	if err := a.DB.Where("campaign_id = ?", campaignID).Order("created_at ASC").Find(&results).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load results", nil, "")
	}

	var buf strings.Builder
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"Group JID", "Group Name", "Participant Count", "Is Admin", "Description", "Status"})

	for _, res := range results {
		isAdmin := "false"
		if res.IsAdmin {
			isAdmin = "true"
		}
		_ = writer.Write([]string{res.GroupJID, res.GroupName, fmt.Sprintf("%d", res.ParticipantCount), isAdmin, res.Description, string(res.Status)})
	}
	writer.Flush()

	filename := fmt.Sprintf("group_extraction_%s.csv", time.Now().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "text/csv; charset=utf-8")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	r.RequestCtx.SetBody([]byte(buf.String()))
	return nil
}
