package handlers

import (
	"encoding/csv"
	"io"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// GroupJoinCampaignRequest represents a group join campaign create/update request.
type GroupJoinCampaignRequest struct {
	Name     string   `json:"name" validate:"required"`
	Accounts []string `json:"accounts"` // WhatsApp account names
	Speed    string   `json:"speed"`    // "slow" or "fast"
}

// GroupJoinRecipientRequest represents adding recipients to a group join campaign.
type GroupJoinRecipientRequest struct {
	InviteLinks []string `json:"invite_links"` // Raw invite links or CSV text
}

func (a *App) requireGroupJoinPermission(r *fastglue.Request, userID uuid.UUID, action string) error {
	return a.requirePermission(r, userID, models.ResourceCampaigns, action)
}

// ListGroupJoinCampaigns lists all group join campaigns for the organization.
func (a *App) ListGroupJoinCampaigns(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionRead); err != nil {
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
	if err := query.Model(&models.GroupJoinCampaign{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count campaigns", nil, "")
	}

	var campaigns []models.GroupJoinCampaign
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&campaigns).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"data":  campaigns,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

// CreateGroupJoinCampaign creates a new group join campaign.
func (a *App) CreateGroupJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req GroupJoinCampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "")
	}

	speed := models.GroupJoinSpeed(req.Speed)
	if speed == "" {
		speed = models.GroupJoinSpeedSlow
	}
	if speed != models.GroupJoinSpeedSlow && speed != models.GroupJoinSpeedFast {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Speed must be 'slow' or 'fast'", nil, "")
	}

	campaign := models.GroupJoinCampaign{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(req.Name),
		Accounts:       convertStringSliceToJSONBArray(req.Accounts),
		Speed:          speed,
		Status:         models.GroupJoinStatusDraft,
		CreatedBy:      userID,
	}

	if err := a.DB.Create(&campaign).Error; err != nil {
		a.Log.Error("Failed to create group join campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create campaign", nil, "")
	}

	return r.SendEnvelope(campaign)
}

// GetGroupJoinCampaign returns a single group join campaign by ID.
func (a *App) GetGroupJoinCampaign(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	return r.SendEnvelope(campaign)
}

// UpdateGroupJoinCampaign updates a group join campaign (only draft or paused).
func (a *App) UpdateGroupJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	if existing.Status != models.GroupJoinStatusDraft && existing.Status != models.GroupJoinStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only update draft or paused campaigns", nil, "")
	}

	var req GroupJoinCampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Accounts != nil {
		updates["accounts"] = convertStringSliceToJSONBArray(req.Accounts)
	}
	if req.Speed != "" {
		speed := models.GroupJoinSpeed(req.Speed)
		if speed != models.GroupJoinSpeedSlow && speed != models.GroupJoinSpeedFast {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Speed must be 'slow' or 'fast'", nil, "")
		}
		updates["speed"] = speed
	}

	if err := a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(updates).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Campaign updated"})
}

// DeleteGroupJoinCampaign deletes a group join campaign and its recipients.
func (a *App) DeleteGroupJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionDelete); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	if existing.Status == models.GroupJoinStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete a running campaign", nil, "")
	}

	// Delete recipients first.
	if err := a.DB.Where("campaign_id = ?", id).Delete(&models.GroupJoinRecipient{}).Error; err != nil {
		a.Log.Error("Failed to delete group join recipients", "error", err, "campaign_id", id)
	}

	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.GroupJoinCampaign{}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete campaign", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Campaign deleted"})
}

// StartGroupJoinCampaign starts a group join campaign.
func (a *App) StartGroupJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	if existing.Status != models.GroupJoinStatusDraft && existing.Status != models.GroupJoinStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign cannot be started in current state", nil, "")
	}

	// Load pending recipients.
	var recipients []models.GroupJoinRecipient
	if err := a.DB.Where("campaign_id = ? AND status = ?", id, models.GroupJoinRecipientPending).Find(&recipients).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load recipients", nil, "")
	}
	if len(recipients) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign has no pending recipients", nil, "")
	}

	// Validate accounts.
	if len(existing.Accounts) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign has no WhatsApp accounts configured", nil, "")
	}

	// Extract account names from JSONBArray.
	accountNames := extractStringSliceFromJSONBArray(existing.Accounts)

	// Update campaign status.
	now := time.Now()
	if err := a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Updates(map[string]any{
		"status":     models.GroupJoinStatusProcessing,
		"started_at": now,
	}).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to start campaign", nil, "")
	}

	// Enqueue jobs for each recipient, round-robining across accounts.
	jobs := make([]*queue.GroupJoinJob, len(recipients))
	for i, recipient := range recipients {
		accountIdx := i % len(accountNames)
		jobs[i] = &queue.GroupJoinJob{
			CampaignID:     id,
			RecipientID:    recipient.ID,
			OrganizationID: orgID,
			InstanceID:      accountNames[accountIdx],
			InviteLink:      recipient.InviteLink,
		}
	}

	if err := a.Queue.EnqueueGroupJoins(r.RequestCtx, jobs); err != nil {
		a.Log.Error("Failed to enqueue group join jobs", "error", err, "campaign_id", id)
		// Rollback status.
		_ = a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ?", id).Updates(map[string]any{
			"status":     existing.Status,
			"started_at": nil,
		})
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to queue jobs", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message":        "Campaign started",
		"enqueued_count": len(jobs),
	})
}

// PauseGroupJoinCampaign pauses a running group join campaign.
func (a *App) PauseGroupJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	if existing.Status != models.GroupJoinStatusProcessing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign is not processing", nil, "")
	}

	if err := a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ? AND organization_id = ?", id, orgID).Update("status", models.GroupJoinStatusPaused).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to pause campaign", nil, "")
	}

	return r.SendEnvelope(map[string]any{"message": "Campaign paused"})
}

// GroupJoinCampaignStats returns statistics for a group join campaign.
func (a *App) GroupJoinCampaignStats(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, id, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	return r.SendEnvelope(map[string]any{
		"total_recipients": campaign.TotalRecipients,
		"joined_count":     campaign.JoinedCount,
		"failed_count":     campaign.FailedCount,
		"skipped_count":    campaign.SkippedCount,
		"status":           campaign.Status,
	})
}

// ListGroupJoinRecipients lists recipients for a group join campaign.
func (a *App) ListGroupJoinRecipients(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	campaignID, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	// Verify campaign exists and belongs to org.
	var campaign models.GroupJoinCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	pg := parsePagination(r)
	statusFilter := string(r.RequestCtx.QueryArgs().Peek("status"))

	query := a.DB.Where("campaign_id = ?", campaignID)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var total int64
	if err := query.Model(&models.GroupJoinRecipient{}).Count(&total).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count recipients", nil, "")
	}

	var recipients []models.GroupJoinRecipient
	if err := query.Order("created_at DESC").Offset(pg.Offset).Limit(pg.Limit).Find(&recipients).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list recipients", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"data":  recipients,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

// UploadGroupJoinRecipients uploads invite links to a group join campaign.
func (a *App) UploadGroupJoinRecipients(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignID, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, campaignID, orgID, "Group join campaign")
	if err != nil {
		return nil
	}

	if existing.Status != models.GroupJoinStatusDraft && existing.Status != models.GroupJoinStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only add recipients to draft or paused campaigns", nil, "")
	}

	// Check content type to handle both JSON and CSV.
	contentType := string(r.RequestCtx.Request.Header.ContentType())

	var inviteLinks []string

	if strings.Contains(contentType, "text/csv") || strings.Contains(contentType, "multipart/form-data") {
		// Handle CSV file upload.
		formFile, err := r.RequestCtx.FormFile("file")
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file uploaded", nil, "")
		}

		file, err := formFile.Open()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to open uploaded file", nil, "")
		}
		defer file.Close()

		reader := csv.NewReader(file)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			for _, field := range record {
				field = strings.TrimSpace(field)
				if field != "" {
					inviteLinks = append(inviteLinks, field)
				}
			}
		}
	} else {
		// Handle JSON body.
		var req GroupJoinRecipientRequest
		if err := a.decodeRequest(r, &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		}
		inviteLinks = req.InviteLinks
	}

	if len(inviteLinks) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No invite links provided", nil, "")
	}

	// Clean and deduplicate links.
	seen := make(map[string]bool)
	uniqueLinks := make([]string, 0, len(inviteLinks))
	for _, link := range inviteLinks {
		link = strings.TrimSpace(link)
		if link == "" || seen[link] {
			continue
		}
		seen[link] = true
		uniqueLinks = append(uniqueLinks, link)
	}

	recipients := make([]models.GroupJoinRecipient, 0, len(uniqueLinks))
	for _, link := range uniqueLinks {
		recipients = append(recipients, models.GroupJoinRecipient{
			CampaignID: campaignID,
			InviteLink: link,
			Status:     models.GroupJoinRecipientPending,
		})
	}

	if err := a.DB.Create(&recipients).Error; err != nil {
		a.Log.Error("Failed to create group join recipients", "error", err, "campaign_id", campaignID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save recipients", nil, "")
	}

	// Update total count.
	if err := a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ?", campaignID).
		Update("total_recipients", gorm.Expr("total_recipients + ?", len(recipients))).Error; err != nil {
		a.Log.Error("Failed to update campaign recipient count", "error", err, "campaign_id", campaignID)
	}

	return r.SendEnvelope(map[string]any{
		"message":     "Recipients uploaded",
		"added_count": len(recipients),
	})
}

// ImportDirectoryGroupsToJoinCampaign imports groups from the directory into a join campaign.
func (a *App) ImportDirectoryGroupsToJoinCampaign(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
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

	existing, err := findByIDAndOrg[models.GroupJoinCampaign](a.DB, r, campaignID, orgID, "Group join campaign")
	if err != nil {
		return nil
	}
	_ = existing

	if existing.Status != models.GroupJoinStatusDraft && existing.Status != models.GroupJoinStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only import to draft or paused campaigns", nil, "")
	}

	var groups []models.GroupDirectory
	if err := a.DB.Where("id IN ? AND organization_id = ?", req.GroupIDs, orgID).Find(&groups).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load groups", nil, "")
	}

	if len(groups) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No valid groups found", nil, "")
	}

	recipients := make([]models.GroupJoinRecipient, 0, len(groups))
	for _, g := range groups {
		link := g.JoinLink
		if link == "" {
			continue
		}
		recipients = append(recipients, models.GroupJoinRecipient{
			CampaignID:      campaignID,
			InviteLink:      link,
			GroupName:       g.Name,
			ParticipantCount: g.ParticipantCount,
			Status:          models.GroupJoinRecipientPending,
		})
	}

	if len(recipients) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No groups with invite links found", nil, "")
	}

	if err := a.DB.Create(&recipients).Error; err != nil {
		a.Log.Error("Failed to import group join recipients", "error", err, "campaign_id", campaignID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import groups", nil, "")
	}

	if err := a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ?", campaignID).
		Update("total_recipients", gorm.Expr("total_recipients + ?", len(recipients))).Error; err != nil {
		a.Log.Error("Failed to update campaign recipient count", "error", err, "campaign_id", campaignID)
	}

	return r.SendEnvelope(map[string]any{
		"message":     "Groups imported to campaign",
		"added_count": len(recipients),
	})
}

// DeleteGroupJoinRecipient removes a recipient from a group join campaign.
func (a *App) DeleteGroupJoinRecipient(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireGroupJoinPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignID, err := parsePathUUID(r, "id", "group join campaign")
	if err != nil {
		return nil
	}

	recipientID, err := parsePathUUID(r, "recipientId", "recipient")
	if err != nil {
		return nil
	}

	// Verify campaign exists and belongs to org.
	var campaign models.GroupJoinCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}
	_ = campaign

	// Delete recipient.
	result := a.DB.Where("id = ? AND campaign_id = ?", recipientID, campaignID).Delete(&models.GroupJoinRecipient{})
	if result.Error != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete recipient", nil, "")
	}
	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Recipient not found", nil, "")
	}

	// Decrement total count.
	_ = a.DB.Model(&models.GroupJoinCampaign{}).Where("id = ?", campaignID).
		Update("total_recipients", gorm.Expr("GREATEST(total_recipients - 1, 0)")).Error

	return r.SendEnvelope(map[string]any{"message": "Recipient deleted"})
}

// convertStringSliceToJSONBArray converts a string slice to models.JSONBArray.
func convertStringSliceToJSONBArray(items []string) models.JSONBArray {
	if items == nil {
		return models.JSONBArray{}
	}
	result := make(models.JSONBArray, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

// extractStringSliceFromJSONBArray extracts string values from a models.JSONBArray.
func extractStringSliceFromJSONBArray(arr models.JSONBArray) []string {
	if arr == nil {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				result = append(result, s)
			}
		}
	}
	return result
}
