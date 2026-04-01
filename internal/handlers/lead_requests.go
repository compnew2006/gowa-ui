package handlers

import (
	"net/mail"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

var allowedLeadRequestRoutes = map[string]struct{}{
	"/pricing": {},
	"/plans":   {},
	"/offer":   {},
}

type CreateLeadRequestRequest struct {
	FullName      string `json:"full_name"`
	CompanyName   string `json:"company_name"`
	WorkEmail     string `json:"work_email"`
	PhoneWhatsApp string `json:"phone_whatsapp"`
	Country       string `json:"country,omitempty"`
	Message       string `json:"message,omitempty"`
	RequestedPlan string `json:"requested_plan,omitempty"`
	SourcePage    string `json:"source_page"`
	SourceRoute   string `json:"source_route"`
}

type UpdateLeadRequestStatusRequest struct {
	Status models.LeadRequestStatus `json:"status"`
}

type LeadRequestResponse struct {
	ID            uuid.UUID                `json:"id"`
	FullName      string                   `json:"full_name"`
	CompanyName   string                   `json:"company_name"`
	WorkEmail     string                   `json:"work_email"`
	PhoneWhatsApp string                   `json:"phone_whatsapp"`
	Country       string                   `json:"country,omitempty"`
	Message       string                   `json:"message,omitempty"`
	RequestedPlan string                   `json:"requested_plan,omitempty"`
	SourcePage    string                   `json:"source_page"`
	SourceRoute   string                   `json:"source_route"`
	Status        models.LeadRequestStatus `json:"status"`
	CreatedAt     string                   `json:"created_at"`
	UpdatedAt     string                   `json:"updated_at"`
}

func leadRequestToResponse(lead models.LeadRequest) LeadRequestResponse {
	return LeadRequestResponse{
		ID:            lead.ID,
		FullName:      lead.FullName,
		CompanyName:   lead.CompanyName,
		WorkEmail:     lead.WorkEmail,
		PhoneWhatsApp: lead.PhoneWhatsApp,
		Country:       lead.Country,
		Message:       lead.Message,
		RequestedPlan: lead.RequestedPlan,
		SourcePage:    lead.SourcePage,
		SourceRoute:   lead.SourceRoute,
		Status:        lead.Status,
		CreatedAt:     lead.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     lead.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (a *App) CreatePublicLeadRequest(r *fastglue.Request) error {
	var req CreateLeadRequestRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.WorkEmail = strings.ToLower(strings.TrimSpace(req.WorkEmail))
	req.PhoneWhatsApp = strings.TrimSpace(req.PhoneWhatsApp)
	req.Country = strings.TrimSpace(req.Country)
	req.Message = strings.TrimSpace(req.Message)
	req.RequestedPlan = strings.ToLower(strings.TrimSpace(req.RequestedPlan))
	req.SourcePage = strings.ToLower(strings.TrimSpace(req.SourcePage))
	req.SourceRoute = strings.TrimSpace(req.SourceRoute)

	if req.FullName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "full_name is required", nil, "")
	}
	if req.CompanyName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "company_name is required", nil, "")
	}
	if req.WorkEmail == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "work_email is required", nil, "")
	}
	if req.PhoneWhatsApp == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_whatsapp is required", nil, "")
	}
	if len(req.FullName) > 255 || len(req.CompanyName) > 255 || len(req.WorkEmail) > 255 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "full_name, company_name, and work_email must be at most 255 characters", nil, "")
	}
	if len(req.PhoneWhatsApp) > 100 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_whatsapp must be at most 100 characters", nil, "")
	}
	if len(req.Country) > 120 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "country must be at most 120 characters", nil, "")
	}
	if len(req.Message) > 4000 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message must be at most 4000 characters", nil, "")
	}
	if _, err := mail.ParseAddress(req.WorkEmail); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "work_email must be a valid email address", nil, "")
	}
	if req.RequestedPlan != "" && !models.IsValidLeadRequestPlan(req.RequestedPlan) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "requested_plan must be starter, growth, dedicated, or enterprise", nil, "")
	}
	if req.SourcePage != "pricing" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "source_page must be pricing", nil, "")
	}
	if _, ok := allowedLeadRequestRoutes[req.SourceRoute]; !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "source_route must be /pricing, /plans, or /offer", nil, "")
	}

	lead := models.LeadRequest{
		FullName:      req.FullName,
		CompanyName:   req.CompanyName,
		WorkEmail:     req.WorkEmail,
		PhoneWhatsApp: req.PhoneWhatsApp,
		Country:       req.Country,
		Message:       req.Message,
		RequestedPlan: req.RequestedPlan,
		SourcePage:    req.SourcePage,
		SourceRoute:   req.SourceRoute,
		Status:        models.LeadRequestStatusNew,
	}

	if err := a.DB.Create(&lead).Error; err != nil {
		a.Log.Error("Failed to create lead request", "error", err, "source_route", lead.SourceRoute)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to submit lead request", nil, "")
	}

	a.Log.Info("Public lead request submitted", "lead_request_id", lead.ID, "requested_plan", lead.RequestedPlan, "source_route", lead.SourceRoute)

	return r.SendEnvelope(map[string]any{
		"id":      lead.ID,
		"status":  lead.Status,
		"message": "Lead request submitted successfully",
	})
}

func (a *App) ListLeadRequests(r *fastglue.Request) error {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSettingsGeneral, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	search := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("search")))
	status := strings.ToLower(strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("status"))))

	query := a.DB.Model(&models.LeadRequest{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"full_name ILIKE ? OR company_name ILIKE ? OR work_email ILIKE ? OR phone_whatsapp ILIKE ? OR country ILIKE ? OR requested_plan ILIKE ? OR message ILIKE ?",
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	if status != "" && status != "all" {
		if !models.IsValidLeadRequestStatus(models.LeadRequestStatus(status)) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "status must be new, contacted, qualified, or closed", nil, "")
		}
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		a.Log.Error("Failed to count lead requests", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list lead requests", nil, "")
	}

	var leads []models.LeadRequest
	if err := pg.Apply(query.Order("created_at DESC")).Find(&leads).Error; err != nil {
		a.Log.Error("Failed to list lead requests", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list lead requests", nil, "")
	}

	response := make([]LeadRequestResponse, len(leads))
	for i, lead := range leads {
		response[i] = leadRequestToResponse(lead)
	}

	return r.SendEnvelope(map[string]any{
		"lead_requests": response,
		"total":         total,
		"page":          pg.Page,
		"limit":         pg.Limit,
	})
}

func (a *App) UpdateLeadRequestStatus(r *fastglue.Request) error {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceSettingsGeneral, models.ActionWrite); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "lead request")
	if err != nil {
		return nil
	}

	var req UpdateLeadRequestStatusRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	req.Status = models.LeadRequestStatus(strings.ToLower(strings.TrimSpace(string(req.Status))))
	if !models.IsValidLeadRequestStatus(req.Status) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "status must be new, contacted, qualified, or closed", nil, "")
	}

	var lead models.LeadRequest
	if err := a.DB.Where("id = ?", id).First(&lead).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Lead request not found", nil, "")
	}

	if err := a.DB.Model(&lead).Update("status", req.Status).Error; err != nil {
		a.Log.Error("Failed to update lead request status", "error", err, "lead_request_id", id, "status", req.Status)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update lead request", nil, "")
	}

	lead.Status = req.Status

	return r.SendEnvelope(leadRequestToResponse(lead))
}
