package handlers

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultCampaignMinDelaySeconds = 20
	defaultCampaignMaxDelaySeconds = 45
	defaultCampaignImportLimit     = 10000
)

// CampaignRequest represents campaign create/update request
type CampaignRequest struct {
	Name            string     `json:"name" validate:"required"`
	WhatsAppAccount string     `json:"whatsapp_account" validate:"required"`
	TemplateID      string     `json:"template_id"`
	BodyContent     string     `json:"body_content"`
	HeaderMediaID   string     `json:"header_media_id"`
	MinDelaySeconds *int       `json:"min_delay_seconds"`
	MaxDelaySeconds *int       `json:"max_delay_seconds"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
}

// CampaignResponse represents campaign in API responses
type CampaignResponse struct {
	ID                  uuid.UUID             `json:"id"`
	Name                string                `json:"name"`
	WhatsAppAccount     string                `json:"whatsapp_account"`
	TemplateID          uuid.UUID             `json:"template_id"`
	TemplateName        string                `json:"template_name,omitempty"`
	HeaderMediaID       string                `json:"header_media_id,omitempty"`
	HeaderMediaFilename string                `json:"header_media_filename,omitempty"`
	HeaderMediaMimeType string                `json:"header_media_mime_type,omitempty"`
	MinDelaySeconds     int                   `json:"min_delay_seconds"`
	MaxDelaySeconds     int                   `json:"max_delay_seconds"`
	Status              models.CampaignStatus `json:"status"`
	TotalRecipients     int                   `json:"total_recipients"`
	SentCount           int                   `json:"sent_count"`
	DeliveredCount      int                   `json:"delivered_count"`
	ReadCount           int                   `json:"read_count"`
	FailedCount         int                   `json:"failed_count"`
	ScheduledAt         *time.Time            `json:"scheduled_at,omitempty"`
	StartedAt           *time.Time            `json:"started_at,omitempty"`
	CompletedAt         *time.Time            `json:"completed_at,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

// RecipientRequest represents recipient import request
type RecipientRequest struct {
	PhoneNumber    string                 `json:"phone_number" validate:"required"`
	RecipientName  string                 `json:"recipient_name"`
	TemplateParams map[string]interface{} `json:"template_params"`
}

func (a *App) requireCampaignPermission(r *fastglue.Request, userID uuid.UUID, action string) error {
	return a.requirePermission(r, userID, models.ResourceCampaigns, action)
}

// ListCampaigns implements campaign listing
func (a *App) ListCampaigns(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)

	// Get query params
	status := string(r.RequestCtx.QueryArgs().Peek("status"))
	whatsappAccount := string(r.RequestCtx.QueryArgs().Peek("whatsapp_account"))
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	baseQuery := requestDB.Where("organization_id = ?", orgID)

	if search != "" {
		baseQuery = baseQuery.Where("name ILIKE ?", "%"+search+"%")
	}

	if status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}
	if whatsappAccount != "" {
		baseQuery = baseQuery.Where("whats_app_account = ?", whatsappAccount)
	}
	if from, ok := parseDateParam(r, "from"); ok {
		baseQuery = baseQuery.Where("created_at >= ?", from)
	}
	if to, ok := parseDateParam(r, "to"); ok {
		baseQuery = baseQuery.Where("created_at <= ?", endOfDay(to))
	}

	// Get total count
	var total int64
	baseQuery.Model(&models.BulkMessageCampaign{}).Count(&total)

	var campaigns []models.BulkMessageCampaign
	if err := pg.Apply(baseQuery.
		Preload("Template").
		Order("created_at DESC")).
		Find(&campaigns).Error; err != nil {
		a.Log.Error("Failed to list campaigns", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}

	// Convert to response format
	response := make([]CampaignResponse, len(campaigns))
	for i, c := range campaigns {
		response[i] = CampaignResponse{
			ID:                  c.ID,
			Name:                c.Name,
			WhatsAppAccount:     c.WhatsAppAccount,
			TemplateID:          c.TemplateID,
			HeaderMediaID:       c.HeaderMediaID,
			HeaderMediaFilename: c.HeaderMediaFilename,
			HeaderMediaMimeType: c.HeaderMediaMimeType,
			MinDelaySeconds:     c.MinDelaySeconds,
			MaxDelaySeconds:     c.MaxDelaySeconds,
			Status:              c.Status,
			TotalRecipients:     c.TotalRecipients,
			SentCount:           c.SentCount,
			DeliveredCount:      c.DeliveredCount,
			ReadCount:           c.ReadCount,
			FailedCount:         c.FailedCount,
			ScheduledAt:         c.ScheduledAt,
			StartedAt:           c.StartedAt,
			CompletedAt:         c.CompletedAt,
			CreatedAt:           c.CreatedAt,
			UpdatedAt:           c.UpdatedAt,
		}
		if c.Template != nil {
			response[i].TemplateName = campaignTemplateDisplayName(c.Template)
		}
	}

	return r.SendEnvelope(map[string]interface{}{
		"campaigns": response,
		"total":     total,
		"page":      pg.Page,
		"limit":     pg.Limit,
	})
}

// CreateCampaign implements campaign creation
func (a *App) CreateCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req CampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign name is required", nil, "")
	}
	if strings.TrimSpace(req.WhatsAppAccount) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp account is required", nil, "")
	}

	if err := a.validateCampaignSender(orgID, req.WhatsAppAccount); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	template, err := a.resolveCampaignTemplate(orgID, req)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	templateID := template.ID
	minDelaySeconds, maxDelaySeconds, err := normalizeCampaignDelayRange(
		defaultCampaignMinDelaySeconds,
		defaultCampaignMaxDelaySeconds,
		req.MinDelaySeconds,
		req.MaxDelaySeconds,
	)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	if err := validateCampaignDelayFloor(minDelaySeconds, maxDelaySeconds, a.campaignDelayFloorSeconds(orgID)); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	campaign := models.BulkMessageCampaign{
		OrganizationID:  orgID,
		WhatsAppAccount: req.WhatsAppAccount,
		Name:            strings.TrimSpace(req.Name),
		TemplateID:      templateID,
		HeaderMediaID:   req.HeaderMediaID,
		MinDelaySeconds: minDelaySeconds,
		MaxDelaySeconds: maxDelaySeconds,
		Status:          models.CampaignStatusDraft,
		ScheduledAt:     req.ScheduledAt,
		CreatedBy:       userID,
	}

	if err := requestDB.Create(&campaign).Error; err != nil {
		a.Log.Error("Failed to create campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create campaign", nil, "")
	}

	a.Log.Info("Campaign created", "campaign_id", campaign.ID, "name", campaign.Name)

	return r.SendEnvelope(CampaignResponse{
		ID:                  campaign.ID,
		Name:                campaign.Name,
		WhatsAppAccount:     campaign.WhatsAppAccount,
		TemplateID:          campaign.TemplateID,
		TemplateName:        campaignTemplateDisplayName(template),
		HeaderMediaID:       campaign.HeaderMediaID,
		HeaderMediaFilename: campaign.HeaderMediaFilename,
		HeaderMediaMimeType: campaign.HeaderMediaMimeType,
		MinDelaySeconds:     campaign.MinDelaySeconds,
		MaxDelaySeconds:     campaign.MaxDelaySeconds,
		Status:              campaign.Status,
		TotalRecipients:     campaign.TotalRecipients,
		SentCount:           campaign.SentCount,
		DeliveredCount:      campaign.DeliveredCount,
		FailedCount:         campaign.FailedCount,
		ScheduledAt:         campaign.ScheduledAt,
		CreatedAt:           campaign.CreatedAt,
		UpdatedAt:           campaign.UpdatedAt,
	})
}

// GetCampaign implements getting a single campaign
func (a *App) GetCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	var campaign models.BulkMessageCampaign
	if err := requestDB.Where("id = ? AND organization_id = ?", id, orgID).
		Preload("Template").
		First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	response := CampaignResponse{
		ID:                  campaign.ID,
		Name:                campaign.Name,
		WhatsAppAccount:     campaign.WhatsAppAccount,
		TemplateID:          campaign.TemplateID,
		HeaderMediaID:       campaign.HeaderMediaID,
		HeaderMediaFilename: campaign.HeaderMediaFilename,
		HeaderMediaMimeType: campaign.HeaderMediaMimeType,
		MinDelaySeconds:     campaign.MinDelaySeconds,
		MaxDelaySeconds:     campaign.MaxDelaySeconds,
		Status:              campaign.Status,
		TotalRecipients:     campaign.TotalRecipients,
		SentCount:           campaign.SentCount,
		DeliveredCount:      campaign.DeliveredCount,
		FailedCount:         campaign.FailedCount,
		ScheduledAt:         campaign.ScheduledAt,
		StartedAt:           campaign.StartedAt,
		CompletedAt:         campaign.CompletedAt,
		CreatedAt:           campaign.CreatedAt,
		UpdatedAt:           campaign.UpdatedAt,
	}
	if campaign.Template != nil {
		response.TemplateName = campaignTemplateDisplayName(campaign.Template)
	}

	return r.SendEnvelope(response)
}

// UpdateCampaign implements campaign update
func (a *App) UpdateCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	// Only allow updates to draft campaigns
	if campaign.Status != models.CampaignStatusDraft {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only update draft campaigns", nil, "")
	}

	var req CampaignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	minDelaySeconds, maxDelaySeconds, err := normalizeCampaignDelayRange(
		campaign.MinDelaySeconds,
		campaign.MaxDelaySeconds,
		req.MinDelaySeconds,
		req.MaxDelaySeconds,
	)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	if req.MinDelaySeconds != nil || req.MaxDelaySeconds != nil {
		if err := validateCampaignDelayFloor(minDelaySeconds, maxDelaySeconds, a.campaignDelayFloorSeconds(orgID)); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
	}

	// Update fields
	updates := map[string]interface{}{
		"name":              req.Name,
		"scheduled_at":      req.ScheduledAt,
		"min_delay_seconds": minDelaySeconds,
		"max_delay_seconds": maxDelaySeconds,
	}

	if req.TemplateID != "" {
		templateID, err := uuid.Parse(req.TemplateID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid template ID", nil, "")
		}
		updates["template_id"] = templateID
	}

	if req.WhatsAppAccount != "" {
		if err := a.validateCampaignSender(orgID, req.WhatsAppAccount); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
		updates["whats_app_account"] = req.WhatsAppAccount
	}

	if a.isWhatsmeowProvider() && strings.TrimSpace(req.BodyContent) != "" {
		trimmedBody := strings.TrimSpace(req.BodyContent)
		if err := requestDB.Model(&models.Template{}).
			Where("id = ? AND organization_id = ?", campaign.TemplateID, orgID).
			Update("body_content", trimmedBody).Error; err != nil {
			a.Log.Error("Failed to update campaign message body", "error", err, "campaign_id", campaign.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign message body", nil, "")
		}
	}

	if err := requestDB.Model(campaign).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign", nil, "")
	}
	requestDB.

		// Reload campaign
		Where("id = ?", id).Preload("Template").First(campaign)

	response := CampaignResponse{
		ID:                  campaign.ID,
		Name:                campaign.Name,
		WhatsAppAccount:     campaign.WhatsAppAccount,
		TemplateID:          campaign.TemplateID,
		HeaderMediaID:       campaign.HeaderMediaID,
		HeaderMediaFilename: campaign.HeaderMediaFilename,
		HeaderMediaMimeType: campaign.HeaderMediaMimeType,
		MinDelaySeconds:     campaign.MinDelaySeconds,
		MaxDelaySeconds:     campaign.MaxDelaySeconds,
		Status:              campaign.Status,
		TotalRecipients:     campaign.TotalRecipients,
		SentCount:           campaign.SentCount,
		DeliveredCount:      campaign.DeliveredCount,
		FailedCount:         campaign.FailedCount,
		ScheduledAt:         campaign.ScheduledAt,
		CreatedAt:           campaign.CreatedAt,
		UpdatedAt:           campaign.UpdatedAt,
	}
	if campaign.Template != nil {
		response.TemplateName = campaignTemplateDisplayName(campaign.Template)
	}

	return r.SendEnvelope(response)
}

func (a *App) validateCampaignSender(orgID uuid.UUID, sender string) error {
	if a.isWhatsmeowProvider() {
		instanceID, err := uuid.Parse(strings.TrimSpace(sender))
		if err != nil {
			return fmt.Errorf("invalid WhatsApp instance ID")
		}

		var instance models.WhatsAppInstance
		if err := a.DB.
			Select("id", "organization_id", "status", "send_blocked_until", "send_block_reason").
			Where("id = ? AND organization_id = ?", instanceID, orgID).
			First(&instance).Error; err != nil {
			return fmt.Errorf("WhatsApp instance not found")
		}
		if instance.Status != models.InstanceStatusConnected {
			return fmt.Errorf("WhatsApp instance is not connected")
		}
		if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
			return fmt.Errorf("WhatsApp instance is blocked: %s", blockReason)
		}
		return nil
	}

	if _, err := a.resolveWhatsAppAccount(orgID, sender); err != nil {
		return fmt.Errorf("WhatsApp account not found")
	}
	return nil
}

func (a *App) resolveCampaignTemplate(orgID uuid.UUID, req CampaignRequest) (*models.Template, error) {
	templateIDRaw := strings.TrimSpace(req.TemplateID)
	if templateIDRaw != "" {
		templateID, err := uuid.Parse(templateIDRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid template ID")
		}

		var template models.Template
		if err := a.DB.Where("id = ? AND organization_id = ?", templateID, orgID).First(&template).Error; err != nil {
			return nil, fmt.Errorf("template not found")
		}
		return &template, nil
	}

	if !a.isWhatsmeowProvider() {
		return nil, fmt.Errorf("template ID is required")
	}

	body := strings.TrimSpace(req.BodyContent)
	if body == "" {
		return nil, fmt.Errorf("campaign message body is required")
	}

	displayName := strings.TrimSpace(req.Name)
	if displayName == "" {
		displayName = "Campaign Message"
	}

	template := &models.Template{
		OrganizationID:  orgID,
		WhatsAppAccount: strings.TrimSpace(req.WhatsAppAccount),
		Name:            fmt.Sprintf("campaign_%s", uuid.NewString()[:8]),
		DisplayName:     displayName,
		Language:        "en",
		Category:        "UTILITY",
		Status:          "APPROVED",
		BodyContent:     body,
	}

	if err := a.DB.Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create campaign template")
	}

	return template, nil
}

func campaignTemplateDisplayName(template *models.Template) string {
	if template == nil {
		return ""
	}
	if strings.TrimSpace(template.DisplayName) != "" {
		return template.DisplayName
	}
	return template.Name
}

func normalizeCampaignDelayRange(currentMin, currentMax int, requestedMin, requestedMax *int) (int, int, error) {
	minDelay := currentMin
	maxDelay := currentMax

	if requestedMin != nil {
		minDelay = *requestedMin
	}
	if requestedMax != nil {
		maxDelay = *requestedMax
	}

	if minDelay < 0 || maxDelay < 0 {
		return 0, 0, fmt.Errorf("campaign delay must be non-negative")
	}
	if minDelay > maxDelay {
		return 0, 0, fmt.Errorf("campaign delay min cannot be greater than max")
	}

	return minDelay, maxDelay, nil
}

func normalizeCampaignRecipientPhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *App) campaignImportLimit() int {
	if a != nil && a.Config != nil && a.Config.Campaigns.MaxImportRecipients > 0 {
		return a.Config.Campaigns.MaxImportRecipients
	}
	return defaultCampaignImportLimit
}

func resolveCampaignUploadMIME(headerMIME, filename string, data []byte) (string, bool) {
	mimeType := normalizeWhatsAppMediaMIME(resolveWhatsAppMediaMIME(headerMIME, filename, data))
	if _, ok := whatsappImageMIMEs[mimeType]; ok {
		return mimeType, true
	}
	if _, ok := whatsappVideoMIMEs[mimeType]; ok {
		return mimeType, true
	}
	if _, ok := whatsappAudioMIMEs[mimeType]; ok {
		return mimeType, true
	}
	switch mimeType {
	case "application/pdf",
		"application/msword",
		"application/vnd.ms-excel",
		"application/vnd.ms-powerpoint",
		"text/plain":
		return mimeType, true
	default:
		if _, ok := whatsappOOXMLMIMEs[mimeType]; ok {
			return mimeType, true
		}
		return mimeType, false
	}
}

func (a *App) shouldEnforceInboundOnlyForSystemSends(orgID uuid.UUID) bool {
	policy := a.loadOrganizationStrictPolicySettings(orgID)
	if !policy.StrictEnabled || !policy.ApplyToSystem {
		return false
	}
	if normalizeOutboundMode(policy.OutboundMode) != organizationOutboundModeInboundOnly {
		return false
	}
	return policy.shouldEnforceStrictPolicy(time.Now().UTC())
}

func (a *App) loadInboundHistoryPhoneSet(orgID uuid.UUID) (map[string]struct{}, error) {
	phoneSet := make(map[string]struct{})
	if a == nil || a.DB == nil || orgID == uuid.Nil {
		return phoneSet, nil
	}

	var phones []string
	if err := a.DB.Model(&models.Contact{}).
		Joins("JOIN messages ON messages.contact_id = contacts.id").
		Where("contacts.organization_id = ? AND messages.organization_id = ? AND messages.direction = ?", orgID, orgID, models.DirectionIncoming).
		Distinct().
		Pluck("contacts.phone_number", &phones).Error; err != nil {
		return nil, err
	}

	for _, phone := range phones {
		if normalized := normalizeCampaignRecipientPhone(phone); normalized != "" {
			phoneSet[normalized] = struct{}{}
		}
	}
	return phoneSet, nil
}

func (a *App) countInboundPolicyViolationsForRecipients(orgID uuid.UUID, recipients []models.BulkMessageRecipient) (int, error) {
	if len(recipients) == 0 {
		return 0, nil
	}

	inboundSet, err := a.loadInboundHistoryPhoneSet(orgID)
	if err != nil {
		return 0, err
	}
	if len(inboundSet) == 0 {
		return len(recipients), nil
	}

	violations := 0
	seen := make(map[string]struct{}, len(recipients))
	for _, recipient := range recipients {
		phone := normalizeCampaignRecipientPhone(recipient.PhoneNumber)
		if phone == "" {
			violations++
			continue
		}
		if _, exists := seen[phone]; exists {
			continue
		}
		seen[phone] = struct{}{}
		if _, ok := inboundSet[phone]; !ok {
			violations++
		}
	}
	return violations, nil
}

// DeleteCampaign implements campaign deletion
func (a *App) DeleteCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionDelete); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	// Don't allow deletion of running campaigns
	if campaign.Status == models.CampaignStatusProcessing || campaign.Status == models.CampaignStatusQueued {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete running campaign", nil, "")
	}

	// Delete recipients first
	if err := requestDB.Where("campaign_id = ?", id).Delete(&models.BulkMessageRecipient{}).Error; err != nil {
		a.Log.Error("Failed to delete campaign recipients", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete campaign", nil, "")
	}

	// Delete campaign
	if err := requestDB.Delete(campaign).Error; err != nil {
		a.Log.Error("Failed to delete campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete campaign", nil, "")
	}

	a.Log.Info("Campaign deleted", "campaign_id", id)

	return r.SendEnvelope(map[string]interface{}{
		"message": "Campaign deleted successfully",
	})
}

// StartCampaign implements starting a campaign
func (a *App) StartCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionExecute); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}
	started, err := a.StartCampaignByID(r.RequestCtx, requestDB, orgID, campaign.ID)
	if err != nil {
		var startErr *campaignStartError
		if errors.As(err, &startErr) {
			switch startErr.kind {
			case campaignStartForbidden:
				return r.SendErrorEnvelope(fasthttp.StatusForbidden, startErr.Error(), reasonCodeDetails(startErr.reasonCode), "")
			case campaignStartConflict:
				return r.SendErrorEnvelope(fasthttp.StatusConflict, startErr.Error(), nil, "")
			case campaignStartBadRequest:
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, startErr.Error(), nil, "")
			default:
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, startErr.Error(), nil, "")
			}
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to start campaign", nil, "")
	}

	a.Log.Info("Campaign started", "campaign_id", id, "recipients", started.enqueuedCount)

	return r.SendEnvelope(map[string]interface{}{
		"message": "Campaign started",
		"status":  started.status,
	})
}

// PauseCampaign implements pausing a campaign
func (a *App) PauseCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionExecute); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	if campaign.Status != models.CampaignStatusProcessing && campaign.Status != models.CampaignStatusQueued {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign is not running", nil, "")
	}

	if err := requestDB.Model(campaign).Update("status", models.CampaignStatusPaused).Error; err != nil {
		a.Log.Error("Failed to pause campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to pause campaign", nil, "")
	}

	a.Log.Info("Campaign paused", "campaign_id", id)

	return r.SendEnvelope(map[string]interface{}{
		"message": "Campaign paused",
		"status":  models.CampaignStatusPaused,
	})
}

// CancelCampaign implements cancelling a campaign
func (a *App) CancelCampaign(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionExecute); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	if campaign.Status == models.CampaignStatusCompleted || campaign.Status == models.CampaignStatusCancelled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Campaign already finished", nil, "")
	}

	if err := requestDB.Model(campaign).Update("status", models.CampaignStatusCancelled).Error; err != nil {
		a.Log.Error("Failed to cancel campaign", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to cancel campaign", nil, "")
	}

	a.Log.Info("Campaign cancelled", "campaign_id", id)

	return r.SendEnvelope(map[string]interface{}{
		"message": "Campaign cancelled",
		"status":  models.CampaignStatusCancelled,
	})
}

// RetryFailed retries sending to all failed recipients
func (a *App) RetryFailed(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionExecute); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	// Only allow retry on completed or paused campaigns
	if campaign.Status != models.CampaignStatusCompleted && campaign.Status != models.CampaignStatusPaused && campaign.Status != models.CampaignStatusFailed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only retry failed messages on completed, paused, or failed campaigns", nil, "")
	}

	// Get failed recipients
	var failedRecipients []models.BulkMessageRecipient
	if err := requestDB.Where("campaign_id = ? AND status = ?", id, models.MessageStatusFailed).Find(&failedRecipients).Error; err != nil {
		a.Log.Error("Failed to load failed recipients", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load failed recipients", nil, "")
	}

	if len(failedRecipients) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No failed messages to retry", nil, "")
	}

	// Reset failed recipients to pending
	if err := requestDB.Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND status = ?", id, models.MessageStatusFailed).
		Updates(map[string]interface{}{
			"status":        models.MessageStatusPending,
			"error_message": "",
		}).Error; err != nil {
		a.Log.Error("Failed to reset failed recipients", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reset failed recipients", nil, "")
	}

	// Reset failed messages in messages table to pending
	if err := requestDB.Model(&models.Message{}).
		Where("metadata->>'campaign_id' = ? AND status = ?", id.String(), models.MessageStatusFailed).
		Updates(map[string]interface{}{
			"status":        models.MessageStatusPending,
			"error_message": "",
		}).Error; err != nil {
		a.Log.Error("Failed to reset failed messages", "error", err)
	}

	// Recalculate campaign stats from messages table
	a.recalculateCampaignStats(id)

	// Update campaign status to processing
	if err := requestDB.Model(campaign).Update("status", models.CampaignStatusProcessing).Error; err != nil {
		a.Log.Error("Failed to update campaign status", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update campaign", nil, "")
	}

	a.Log.Info("Retrying failed messages", "campaign_id", id, "failed_count", len(failedRecipients))

	// Enqueue failed recipients as individual jobs for parallel processing
	jobs := make([]*queue.RecipientJob, len(failedRecipients))
	for i, recipient := range failedRecipients {
		jobs[i] = &queue.RecipientJob{
			CampaignID:     id,
			RecipientID:    recipient.ID,
			OrganizationID: orgID,
			PhoneNumber:    recipient.PhoneNumber,
			RecipientName:  recipient.RecipientName,
			TemplateParams: recipient.TemplateParams,
		}
	}

	if err := a.Queue.EnqueueRecipients(r.RequestCtx, jobs); err != nil {
		a.Log.Error("Failed to enqueue recipients for retry", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to queue recipients", nil, "")
	}

	a.Log.Info("Failed recipients enqueued for retry", "campaign_id", id, "count", len(jobs))

	return r.SendEnvelope(map[string]interface{}{
		"message":     "Retrying failed messages",
		"retry_count": len(failedRecipients),
		"status":      models.CampaignStatusProcessing,
	})
}

// ImportRecipients implements adding recipients to a campaign
func (a *App) ImportRecipients(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	if campaign.Status != models.CampaignStatusDraft {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only add recipients to draft campaigns", nil, "")
	}

	var req struct {
		Recipients []RecipientRequest `json:"recipients" validate:"required"`
	}
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if limit := a.campaignImportLimit(); limit > 0 && len(req.Recipients) > limit {
		return r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			fmt.Sprintf("Recipient import exceeds the maximum of %d recipients", limit),
			nil,
			"recipients",
		)
	}

	normalizedInboundSet, err := a.loadInboundHistoryPhoneSet(orgID)
	if err != nil {
		a.Log.Error("Failed to load inbound history set for recipient import", "error", err, "campaign_id", id)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to validate recipients", nil, "")
	}
	enforceInboundOnly := a.shouldEnforceInboundOnlyForSystemSends(orgID)

	recipients := make([]models.BulkMessageRecipient, 0, len(req.Recipients))
	seenPhones := make(map[string]struct{}, len(req.Recipients))
	for _, rec := range req.Recipients {
		normalizedPhone := normalizeCampaignRecipientPhone(rec.PhoneNumber)
		if normalizedPhone == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid recipient phone_number", nil, "phone_number")
		}

		if enforceInboundOnly {
			if _, ok := normalizedInboundSet[normalizedPhone]; !ok {
				return r.SendErrorEnvelope(
					fasthttp.StatusForbidden,
					"Recipient rejected by strict inbound-only policy (no inbound history)",
					reasonCodeDetails(ReasonCodePolicyNoInbound),
					"",
				)
			}
		}

		if _, exists := seenPhones[normalizedPhone]; exists {
			continue
		}
		seenPhones[normalizedPhone] = struct{}{}

		recipients = append(recipients, models.BulkMessageRecipient{
			CampaignID:      id,
			PhoneNumber:     normalizedPhone,
			PhoneNormalized: normalizedPhone,
			RecipientName:   strings.TrimSpace(rec.RecipientName),
			TemplateParams:  models.JSONB(rec.TemplateParams),
			Status:          models.MessageStatusPending,
		})
	}

	if len(recipients) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No valid recipients to import", nil, "")
	}

	result := requestDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "phone_normalized"}},
		DoNothing: true,
	}).Create(&recipients)
	if result.Error != nil {
		a.Log.Error("Failed to add recipients", "error", result.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to add recipients", nil, "")
	}
	addedCount := int(result.RowsAffected)

	// Update total recipients count
	var totalCount int64
	requestDB.
		Model(&models.BulkMessageRecipient{}).Where("campaign_id = ?", id).Count(&totalCount)
	requestDB.
		Model(campaign).Update("total_recipients", totalCount)

	a.Log.Info("Recipients added to campaign", "campaign_id", id, "count", len(req.Recipients))

	return r.SendEnvelope(map[string]interface{}{
		"message":          "Recipients added successfully",
		"added_count":      addedCount,
		"total_recipients": totalCount,
	})
}

// GetCampaignRecipients implements listing campaign recipients
func (a *App) GetCampaignRecipients(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	// Verify campaign belongs to org
	_, err = findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, id, orgID, "Campaign")
	if err != nil {
		return nil
	}

	var recipients []models.BulkMessageRecipient
	if err := requestDB.Where("campaign_id = ?", id).Order("created_at ASC").Find(&recipients).Error; err != nil {
		a.Log.Error("Failed to list recipients", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list recipients", nil, "")
	}

	if a.ShouldMaskPhoneNumbers(orgID) {
		for i := range recipients {
			recipients[i].PhoneNumber = MaskPhoneNumber(recipients[i].PhoneNumber)
			recipients[i].RecipientName = MaskIfPhoneNumber(recipients[i].RecipientName)
		}
	}

	return r.SendEnvelope(map[string]interface{}{
		"recipients": recipients,
		"total":      len(recipients),
	})
}

// DeleteCampaignRecipient deletes a single recipient from a campaign
func (a *App) DeleteCampaignRecipient(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignUUID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	recipientUUID, err := parsePathUUID(r, "recipientId", "recipient")
	if err != nil {
		return nil
	}

	// Verify campaign belongs to org and is in draft status
	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, campaignUUID, orgID, "Campaign")
	if err != nil {
		return nil
	}

	if campaign.Status != models.CampaignStatusDraft {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only delete recipients from draft campaigns", nil, "")
	}

	// Verify recipient belongs to campaign and delete
	result := requestDB.Where("id = ? AND campaign_id = ?", recipientUUID, campaignUUID).Delete(&models.BulkMessageRecipient{})
	if result.Error != nil {
		a.Log.Error("Failed to delete recipient", "error", result.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete recipient", nil, "")
	}

	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Recipient not found", nil, "")
	}
	requestDB.

		// Update campaign recipient count
		Model(campaign).Update("total_recipients", gorm.Expr("total_recipients - 1"))

	return r.SendEnvelope(map[string]interface{}{
		"message": "Recipient deleted successfully",
	})
}

// UploadCampaignMedia uploads media for a campaign's template header
func (a *App) UploadCampaignMedia(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	campaignUUID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	// Get campaign with template
	var campaign models.BulkMessageCampaign
	if err := requestDB.Where("id = ? AND organization_id = ?", campaignUUID, orgID).
		Preload("Template").
		First(&campaign).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Campaign not found", nil, "")
	}

	// Only allow media upload for draft campaigns
	if campaign.Status != models.CampaignStatusDraft {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Can only upload media for draft campaigns", nil, "")
	}

	providerIsWhatsmeow := a.isWhatsmeowProvider()

	// Meta template campaigns require a media header to accept uploaded media.
	if !providerIsWhatsmeow && (campaign.Template == nil || campaign.Template.HeaderType == "" || campaign.Template.HeaderType == "TEXT") {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Template does not have a media header", nil, "")
	}

	// Parse multipart form
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
	}

	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, "")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to open file", nil, "")
	}
	defer func() { _ = file.Close() }()

	// Read file content (limit to 16MB)
	const maxMediaSize = 16 << 20 // 16MB
	data, err := io.ReadAll(io.LimitReader(file, maxMediaSize+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}
	if len(data) > maxMediaSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 16MB", nil, "")
	}

	// Determine and validate MIME type from file bytes, falling back to safe metadata.
	mimeType, allowed := resolveCampaignUploadMIME(fileHeader.Header.Get("Content-Type"), fileHeader.Filename, data)
	if !allowed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type: "+mimeType, nil, "")
	}

	mediaID := ""
	if !providerIsWhatsmeow {
		// Meta provider requires uploading media first and storing returned media ID.
		account, err := a.resolveWhatsAppAccount(orgID, campaign.WhatsAppAccount)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp account not found", nil, "")
		}

		waAccount := a.toWhatsAppAccount(account)
		ctx := r.RequestCtx
		mediaID, err = a.WhatsApp.UploadMedia(ctx, waAccount, data, mimeType, fileHeader.Filename)
		if err != nil {
			a.Log.Error("Failed to upload media to WhatsApp", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to upload media to WhatsApp", nil, "")
		}
	}

	// Save file locally for preview
	if !a.checkQuotaWithDeltaOrRespond(r, license.ResourceStorage, orgID, int64(len(data))) {
		return nil
	}

	localPath, err := a.saveCampaignMedia(orgID, campaignUUID.String(), data, mimeType)
	if err != nil {
		a.Log.Error("Failed to save media locally", "error", err)
		// Don't fail the request, just log the error - preview won't work
	}

	// Update campaign with media ID, filename, mime type, and local path
	updates := map[string]interface{}{
		"header_media_id":         mediaID,
		"header_media_filename":   sanitizeFilename(fileHeader.Filename),
		"header_media_mime_type":  mimeType,
		"header_media_local_path": localPath,
	}
	if err := requestDB.Model(&campaign).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update campaign with media info", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media info", nil, "")
	}

	a.Log.Info("Campaign media uploaded", "campaign_id", campaignUUID, "media_id", mediaID, "filename", fileHeader.Filename, "local_path", localPath)

	return r.SendEnvelope(map[string]interface{}{
		"media_id":   mediaID,
		"filename":   fileHeader.Filename,
		"mime_type":  mimeType,
		"local_path": localPath,
		"message":    "Media uploaded successfully",
	})
}

// saveCampaignMedia saves uploaded media locally for preview
func (a *App) saveCampaignMedia(orgID uuid.UUID, campaignID string, data []byte, mimeType string) (string, error) {
	// Determine file extension
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}

	// Create campaigns media directory
	subdir := organizationMediaSubdir(orgID, "campaigns")
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Generate filename using campaign ID
	filename := campaignID + ext
	relativePath := filepath.Join(subdir, filename)
	filePath := filepath.Join(a.getMediaStoragePath(), relativePath)

	// Save file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	// Return relative path for storage
	a.Log.Info("Campaign media saved locally", "path", relativePath, "size", len(data))

	return relativePath, nil
}

// ServeCampaignMedia serves campaign media files for preview
func (a *App) ServeCampaignMedia(r *fastglue.Request) error {
	requestDB :=
		// Get auth context
		a.requestDB(r)

	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireCampaignPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	// Get campaign ID from URL
	campaignUUID, err := parsePathUUID(r, "id", "campaign")
	if err != nil {
		return nil
	}

	// Find campaign and verify access
	campaign, err := findByIDAndOrg[models.BulkMessageCampaign](requestDB, r, campaignUUID, orgID, "Campaign")
	if err != nil {
		return nil
	}

	// Check if campaign has media
	if campaign.HeaderMediaLocalPath == "" {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No media found", nil, "")
	}

	// Security: prevent directory traversal and symlink attacks
	filePath := filepath.Clean(campaign.HeaderMediaLocalPath)
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Storage configuration error", nil, "")
	}
	fullPath, err := filepath.Abs(filepath.Join(baseDir, filePath))
	if err != nil || !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	// Reject symlinks
	info, err := os.Lstat(fullPath)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "File not found", nil, "")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid file path", nil, "")
	}

	// Read file
	// #nosec G304 -- fullPath is sanitized and bounded to baseDir with symlink checks above.
	data, err := os.ReadFile(fullPath)
	if err != nil {
		a.Log.Error("Failed to read media file", "path", fullPath, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}

	// Use stored mime type or determine from extension
	contentType := campaign.HeaderMediaMimeType
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		contentType = getMimeTypeFromExtension(ext)
	}

	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, max-age=3600")
	r.RequestCtx.SetBody(data)

	return nil
}

// getMimeTypeFromExtension returns MIME type from file extension
func getMimeTypeFromExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".3gp":
		return "video/3gpp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// recalculateCampaignStats recalculates all campaign stats from messages table
func (a *App) recalculateCampaignStats(campaignID uuid.UUID) {
	var stats struct {
		Sent      int64
		Delivered int64
		Read      int64
		Failed    int64
	}

	if err := a.DB.Model(&models.Message{}).
		Where("metadata->>'campaign_id' = ?", campaignID.String()).
		Select(`
			COUNT(CASE WHEN status IN ('sent','delivered','read') THEN 1 END) as sent,
			COUNT(CASE WHEN status IN ('delivered','read') THEN 1 END) as delivered,
			COUNT(CASE WHEN status = 'read' THEN 1 END) as read,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		`).Scan(&stats).Error; err != nil {
		a.Log.Error("Failed to scan campaign message stats", "error", err, "campaign_id", campaignID)
		return
	}

	if err := a.DB.Model(&models.BulkMessageCampaign{}).Where("id = ?", campaignID).
		Updates(map[string]interface{}{
			"sent_count":      stats.Sent,
			"delivered_count": stats.Delivered,
			"read_count":      stats.Read,
			"failed_count":    stats.Failed,
		}).Error; err != nil {
		a.Log.Error("Failed to recalculate campaign stats", "error", err, "campaign_id", campaignID)
	}
}

// sanitizeFilename removes path separators, dangerous characters, and truncates length.
var safeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizeFilename(name string) string {
	// Strip any path component
	name = filepath.Base(name)
	// Replace unsafe characters
	name = safeFilenameRe.ReplaceAllString(name, "_")
	// Truncate to 255 chars
	if len(name) > 255 {
		name = name[:255]
	}
	if name == "" || name == "." || name == ".." {
		name = "unnamed"
	}
	return name
}
