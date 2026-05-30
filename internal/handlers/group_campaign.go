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

// GroupInfoResponse represents a WhatsApp group in API responses.
type GroupInfoResponse struct {
	JID              string `json:"jid"`
	Name             string `json:"name"`
	ParticipantCount int    `json:"participant_count"`
}

// GroupRecipientRequest represents a request to add group targets to a campaign.
type GroupRecipientRequest struct {
	GroupJIDs []struct {
		JID              string `json:"jid" validate:"required"`
		Name             string `json:"name"`
		ParticipantCount int    `json:"participant_count"`
	} `json:"groups" validate:"required,dive"`
}

// ListInstanceGroups returns groups available on a whatsmeow instance.
func (a *App) ListInstanceGroups(r *fastglue.Request) error {
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group listing is only available for whatsmeow instances", nil, "")
	}

	instanceID, _ := r.RequestCtx.UserValue("instanceId").(string)
	if instanceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "instanceId is required", nil, "")
	}
	if _, err := uuid.Parse(instanceID); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID format", nil, "")
	}

	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Verify the instance belongs to the organization.
	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "WhatsApp instance not found", nil, "")
	}

	gp, ok := a.MessageProvider.(provider.GroupProvider)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Group provider not available", nil, "")
	}

	groups, err := gp.GetGroups(r.RequestCtx, instanceID)
	if err != nil {
		a.Log.Error("Failed to list groups", "error", err, "instance_id", instanceID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list groups", nil, "")
	}

	// Filter by name if query provided
	if q := strings.ToLower(strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))); q != "" {
		filtered := groups[:0]
		for _, g := range groups {
			if strings.Contains(strings.ToLower(g.Name), q) {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}

	response := make([]GroupInfoResponse, len(groups))
	for i, g := range groups {
		response[i] = GroupInfoResponse{
			JID:              g.JID,
			Name:             g.Name,
			ParticipantCount: g.ParticipantCount,
		}
	}
	return r.SendEnvelope(response)
}

// ValidateGroupJIDs batch-validates a list of group JIDs.
func (a *App) ValidateGroupJIDs(r *fastglue.Request) error {
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group operations are only available for whatsmeow instances", nil, "")
	}

	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req struct {
		GroupJIDs []string `json:"group_jids"`
		CampaignID string   `json:"campaign_id"`
		InstanceID string   `json:"instance_id"`
	}
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	if len(req.GroupJIDs) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No group JIDs provided", nil, "")
	}

	// Verify campaign ownership.
	campaignID, parseErr := uuid.Parse(req.CampaignID)
	if parseErr != nil || campaignID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid campaign ID", nil, "")
	}
	var campaign models.BulkMessageCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	type validationItem struct {
		JID       string `json:"jid"`
		Valid     bool   `json:"valid"`
		Name      string `json:"name,omitempty"`
		ParticipantCount int `json:"participant_count,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	results := make([]validationItem, len(req.GroupJIDs))

	gp, ok := a.MessageProvider.(provider.GroupProvider)
	if !ok {
		for i, jid := range req.GroupJIDs {
			results[i] = validationItem{JID: jid, Valid: false, Error: "Group provider not available"}
		}
		return r.SendEnvelope(results)
	}

	instanceID := req.InstanceID
	if instanceID == "" {
		instanceID = campaign.WhatsAppAccount
	}

	for i, jid := range req.GroupJIDs {
		if !contactutil.IsValidGroupJID(jid) {
			results[i] = validationItem{JID: jid, Valid: false, Error: "Invalid group JID format"}
			continue
		}
		info, err := gp.VerifyGroupMembership(r.RequestCtx, instanceID, jid)
		if err != nil {
			results[i] = validationItem{JID: jid, Valid: false, Error: "Group not found or inaccessible"}
			continue
		}
		results[i] = validationItem{JID: jid, Valid: true, Name: info.Name, ParticipantCount: info.ParticipantCount}
	}
	return r.SendEnvelope(results)
}

// AddCampaignGroups adds group targets to a campaign.
func (a *App) AddCampaignGroups(r *fastglue.Request) error {
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group targeting is only available for whatsmeow instances", nil, "")
	}

	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	requestDB := a.requestDB(r)
	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, campaignID, orgID, "Campaign")
	if err != nil {
		return nil
	}

	if campaign.Status != models.CampaignStatusDraft && campaign.Status != models.CampaignStatusPaused {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only add groups to draft or paused campaigns", nil, "")
	}

	var req GroupRecipientRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	recipients := make([]models.BulkMessageRecipient, 0, len(req.GroupJIDs))
	for _, g := range req.GroupJIDs {
		if !contactutil.IsValidGroupJID(g.JID) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid group JID: "+g.JID, nil, "")
		}

		recipients = append(recipients, models.BulkMessageRecipient{
			CampaignID:        campaign.ID,
			PhoneNumber:      g.JID,
			RecipientName:    g.Name,
			RecipientType:     models.RecipientTypeGroup,
			GroupJID:          g.JID,
			GroupName:         g.Name,
			ParticipantCount:  g.ParticipantCount,
			Status:            models.MessageStatusPending,
		})
	}

	if err := requestDB.Create(&recipients).Error; err != nil {
		a.Log.Error("Failed to add group recipients", "error", err, "campaign_id", campaignID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to add group recipients", nil, "")
	}

	if err := requestDB.Model(campaign).
		Update("total_recipients", gorm.Expr("total_recipients + ?", len(recipients))).Error; err != nil {
		a.Log.Error("Failed to update campaign recipient count", "error", err, "campaign_id", campaignID)
	}

	return r.SendEnvelope(map[string]interface{}{
		"message":    "Group recipients added",
		"added_count": len(recipients),
	})
}

// ListCampaignGroups returns group targets for a campaign.
func (a *App) ListCampaignGroups(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	campaignID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	// Verify campaign belongs to organization.
	var campaign models.BulkMessageCampaign
	if err := a.DB.Where("id = ? AND organization_id = ?", campaignID, orgID).First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	var recipients []models.BulkMessageRecipient
	if err := a.DB.
		Where("campaign_id = ? AND recipient_type = ?", campaignID, models.RecipientTypeGroup).
		Find(&recipients).Error; err != nil {
		a.Log.Error("Failed to list group recipients", "error", err, "campaign_id", campaignID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list group recipients", nil, "")
	}

	type groupRecipientResponse struct {
		ID               string `json:"id"`
		PhoneNumber      string `json:"phone_number"`
		RecipientName    string `json:"recipient_name"`
		Status           string `json:"status"`
		SentAt           *int64 `json:"sent_at,omitempty"`
		DeliveredAt      *int64 `json:"delivered_at,omitempty"`
		ErrorMessage     string `json:"error_message,omitempty"`
		RecipientType    string `json:"recipient_type"`
		GroupJID         string `json:"group_jid"`
		GroupName        string `json:"group_name"`
		ParticipantCount int    `json:"participant_count"`
	}

	response := make([]groupRecipientResponse, len(recipients))
	for i, rec := range recipients {
		var sentAt *int64
		if rec.SentAt != nil {
			t := rec.SentAt.Unix()
			sentAt = &t
		}
		var deliveredAt *int64
		if rec.DeliveredAt != nil {
			t := rec.DeliveredAt.Unix()
			deliveredAt = &t
		}
		response[i] = groupRecipientResponse{
			ID:               rec.ID.String(),
			PhoneNumber:      rec.PhoneNumber,
			RecipientName:    rec.RecipientName,
			Status:           string(rec.Status),
			SentAt:           sentAt,
			DeliveredAt:      deliveredAt,
			ErrorMessage:     rec.ErrorMessage,
			RecipientType:    rec.RecipientType,
			GroupJID:         rec.GroupJID,
			GroupName:        rec.GroupName,
			ParticipantCount: rec.ParticipantCount,
		}
	}
	return r.SendEnvelope(response)
}

// DeleteCampaignGroup removes a group target from a campaign.
func (a *App) DeleteCampaignGroup(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	recipientID, err := parsePathUUID(r, "recipientId", "group recipient")
	if err != nil {
		return nil
	}

	requestDB := a.requestDB(r)
	var recipient models.BulkMessageRecipient
	if err := requestDB.Where("id = ? AND campaign_id = ? AND organization_id = ? AND recipient_type = ?",
		recipientID, campaignID, orgID, models.RecipientTypeGroup).First(&recipient).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Group recipient not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to find group recipient", nil, "")
	}

	if recipient.Status == models.MessageStatusSent {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot remove a group that has already been sent to", nil, "")
	}

	if err := requestDB.Delete(&recipient).Error; err != nil {
		a.Log.Error("Failed to delete group recipient", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete group recipient", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"message": "Group recipient removed",
	})
}
