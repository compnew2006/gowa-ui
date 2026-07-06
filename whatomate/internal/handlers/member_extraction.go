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

type MemberExtractionCampaignRequest struct {
	Name       string `json:"name" validate:"required"`
	InstanceID string `json:"instance_id" validate:"required"`
	GroupJID   string `json:"group_jid" validate:"required"`
}

func (a *App) ListMemberExtractionCampaigns(r *fastglue.Request) error {
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
	if err := query.Model(&models.MemberExtractionCampaign{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count campaigns", nil, "")
	}

	var campaigns []models.MemberExtractionCampaign
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&campaigns).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}

	return r.SendEnvelope(paginatedEnvelope("data", campaigns, total, pg))
}

func (a *App) CreateMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}

	var req MemberExtractionCampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "")
	}
	if req.InstanceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance ID is required", nil, "")
	}
	if strings.TrimSpace(req.GroupJID) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group JID is required", nil, "")
	}

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance not found", nil, "")
	}

	campaign := models.MemberExtractionCampaign{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(req.Name),
		InstanceID:     instanceID,
		InstanceName:   instance.Name,
		GroupJID:       strings.TrimSpace(req.GroupJID),
		GroupName:      "",
		Status:         models.MemberExtractionStatusDraft,
		CreatedBy:      userID,
	}

	if err := a.DB.Create(&campaign).Error; err != nil {
		a.Log.Error("Failed to create member extraction campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create campaign", nil, "")
	}

	return r.SendEnvelope(campaign)
}

func (a *App) GetMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	campaign, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(campaign)
}

func (a *App) UpdateMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.MemberExtractionStatusDraft && existing.Status != models.MemberExtractionStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only update draft or paused campaigns", nil, "")
	}

	var req MemberExtractionCampaignRequest
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
	if req.GroupJID != "" {
		updates["group_jid"] = strings.TrimSpace(req.GroupJID)
	}

	if err := a.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(updates).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign updated"})
}

func (a *App) DeleteMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionDelete); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status == models.MemberExtractionStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete a running campaign", nil, "")
	}
	a.DB.Where("campaign_id = ?", id).Delete(&models.MemberExtractionResult{})
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.MemberExtractionCampaign{}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign deleted"})
}

func (a *App) StartMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.MemberExtractionStatusDraft && existing.Status != models.MemberExtractionStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign cannot be started in current state", nil, "")
	}
	if existing.InstanceID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign has no instance configured", nil, "")
	}
	if existing.GroupJID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign has no group configured", nil, "")
	}

	now := time.Now()
	if err := a.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(map[string]any{
		"status":     models.MemberExtractionStatusProcessing,
		"started_at": now,
	}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to start campaign", nil, "")
	}

	job := &queue.MemberExtractionJob{
		CampaignID:     id,
		OrganizationID: orgID,
		InstanceID:     existing.InstanceID,
		GroupJID:       existing.GroupJID,
		EnqueuedAt:     time.Now(),
	}
	if err := a.Queue.EnqueueMemberExtraction(r.RequestCtx, job); err != nil {
		a.Log.Error("Failed to enqueue member extraction job", "error", err, "campaign_id", id)
		a.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ?", id).Updates(map[string]any{"status": existing.Status, "started_at": nil})
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to queue job", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Campaign started"})
}

func (a *App) PauseMemberExtractionCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	existing, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	if existing.Status != models.MemberExtractionStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign is not processing", nil, "")
	}
	if err := a.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Update("status", models.MemberExtractionStatusPaused).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to pause campaign", nil, "")
	}
	return r.SendEnvelope(map[string]any{"message": "Campaign paused"})
}

func (a *App) GetMemberExtractionCampaignStats(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	id, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	campaign, err := findByIDAndOrg[models.MemberExtractionCampaign](a.DB, r, id, orgID, "Member extraction campaign")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(map[string]any{
		"total_members":   campaign.TotalMembers,
		"extracted_count": campaign.ExtractedCount,
		"failed_count":    campaign.FailedCount,
		"status":          campaign.Status,
	})
}

func (a *App) GetMemberExtractionCampaignResults(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	campaignID, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	var campaign models.MemberExtractionCampaign
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
		query = query.Where("participant_jid ILIKE ? OR phone_number LIKE ? OR push_name ILIKE ?", pattern, pattern, pattern)
	}

	var total int64
	if err := query.Model(&models.MemberExtractionResult{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count results", nil, "")
	}

	var results []models.MemberExtractionResult
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&results).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list results", nil, "")
	}

	return r.SendEnvelope(paginatedEnvelope("data", results, total, pg))
}

func (a *App) ExportMemberExtractionCampaignResults(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceCampaigns, models.ActionRead); err != nil {
		return nil
	}
	campaignID, err := parsePathUUID(r, "id", "member extraction campaign")
	if err != nil {
		return nil
	}
	var campaign models.MemberExtractionCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	var results []models.MemberExtractionResult
	if err := a.DB.Where("campaign_id = ?", campaignID).Order("created_at ASC").Find(&results).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load results", nil, "")
	}

	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	var buf strings.Builder
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"Participant JID", "Phone Number", "Push Name", "Is Admin", "Is Super Admin", "Status"})

	for _, res := range results {
		phone := res.PhoneNumber
		if shouldMask {
			phone = MaskPhoneNumber(phone)
		}
		isAdmin := "false"
		if res.IsAdmin {
			isAdmin = "true"
		}
		isSuperAdmin := "false"
		if res.IsSuperAdmin {
			isSuperAdmin = "true"
		}
		_ = writer.Write([]string{res.ParticipantJID, phone, res.PushName, isAdmin, isSuperAdmin, string(res.Status)})
	}
	writer.Flush()

	filename := fmt.Sprintf("member_extraction_%s.csv", time.Now().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "text/csv; charset=utf-8")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	r.RequestCtx.SetBody([]byte(buf.String()))
	return nil
}
