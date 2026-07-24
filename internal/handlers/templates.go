package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TemplateRequest represents the request body for creating/updating a template
type TemplateRequest struct {
	WhatsAppAccount string `json:"whatsapp_account" validate:"required"` // WhatsApp account name
	Name            string `json:"name" validate:"required"`
	DisplayName     string `json:"display_name"`
	Language        string `json:"language" validate:"required"`
	Category        string `json:"category" validate:"required"` // MARKETING, UTILITY, AUTHENTICATION
	HeaderType      string `json:"header_type"`                  // TEXT, IMAGE, DOCUMENT, VIDEO, NONE
	HeaderContent   string `json:"header_content"`
	BodyContent     string `json:"body_content"`
	FooterContent   string `json:"footer_content"`
	Buttons         []any  `json:"buttons"`
	SampleValues    []any  `json:"sample_values"`

	// Authentication template fields
	AddSecurityRecommendation bool `json:"add_security_recommendation"` // Add "For your security, do not share this code."
	CodeExpirationMinutes     int  `json:"code_expiration_minutes"`     // 1-90, 0 means no expiration footer
}

// TemplateResponse represents the response for a template
type TemplateResponse struct {
	ID                        uuid.UUID `json:"id"`
	WhatsAppAccount           string    `json:"whatsapp_account"` // WhatsApp account name
	MetaTemplateID            string    `json:"meta_template_id"`
	Name                      string    `json:"name"`
	DisplayName               string    `json:"display_name"`
	Language                  string    `json:"language"`
	Category                  string    `json:"category"`
	Status                    string    `json:"status"`
	HeaderType                string    `json:"header_type"`
	HeaderContent             string    `json:"header_content"`
	BodyContent               string    `json:"body_content"`
	FooterContent             string    `json:"footer_content"`
	Buttons                   []any     `json:"buttons"`
	SampleValues              []any     `json:"sample_values"`
	AddSecurityRecommendation bool      `json:"add_security_recommendation"`
	CodeExpirationMinutes     int       `json:"code_expiration_minutes"`
	QualityRating             string    `json:"quality_rating"`
	CreatedByName             string    `json:"created_by_name,omitempty"`
	UpdatedByName             string    `json:"updated_by_name,omitempty"`
	CreatedAt                 string    `json:"created_at"`
	UpdatedAt                 string    `json:"updated_at"`
}

// ListTemplates returns all templates for the organization
func (a *App) ListTemplates(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	pg := parsePagination(r)

	// Optional filters
	accountName := string(r.RequestCtx.QueryArgs().Peek("account")) // Filter by account name
	status := string(r.RequestCtx.QueryArgs().Peek("status"))
	category := string(r.RequestCtx.QueryArgs().Peek("category"))
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	query := a.DB.Where("organization_id = ?", orgID)

	if accountName != "" {
		query = query.Where("whats_app_account = ?", accountName)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR display_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.Template{}).Count(&total)

	var templates []models.Template
	if err := pg.Apply(query.Order("created_at DESC")).
		Find(&templates).Error; err != nil {
		a.Log.Error("Failed to list templates", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list templates", nil, "")
	}

	response := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		response[i] = templateToResponse(t)
	}

	return r.SendEnvelope(listEnvelope("templates", response, total, pg))
}

// CreateTemplate creates a new message template

// GetTemplate returns a single template
func (a *App) GetTemplate(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "template")
	if err != nil {
		return nil
	}

	var template models.Template
	if err := a.DB.Preload("CreatedBy").Preload("UpdatedBy").
		Where("id = ? AND organization_id = ?", id, orgID).First(&template).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Template not found", nil, "")
	}

	resp := templateToResponse(template)
	if template.CreatedBy != nil {
		resp.CreatedByName = template.CreatedBy.FullName
	}
	if template.UpdatedBy != nil {
		resp.UpdatedByName = template.UpdatedBy.FullName
	}

	return r.SendEnvelope(resp)
}

// UpdateTemplate updates a message template

// DeleteTemplate deletes a message template

// SubmitTemplate submits a template to Meta for approval

// submitTemplateToMeta submits a template to Meta's API (creates new or updates existing)
func (a *App) submitTemplateToMeta(account *models.WhatsAppAccount, template *models.Template) (string, error) {
	waAccount := a.toWhatsAppAccount(account)

	submission := &whatsapp.TemplateSubmission{
		MetaTemplateID:            template.MetaTemplateID, // If set, will update instead of create
		Name:                      template.Name,
		Language:                  template.Language,
		Category:                  template.Category,
		HeaderType:                template.HeaderType,
		HeaderContent:             template.HeaderContent,
		BodyContent:               template.BodyContent,
		FooterContent:             template.FooterContent,
		Buttons:                   template.Buttons,
		SampleValues:              template.SampleValues,
		AddSecurityRecommendation: template.AddSecurityRecommendation,
		CodeExpirationMinutes:     template.CodeExpirationMinutes,
	}

	ctx := context.Background()
	return a.WhatsApp.SubmitTemplate(ctx, waAccount, submission)
}

// SyncTemplates syncs templates from Meta API

func (a *App) fetchTemplatesFromMeta(account *models.WhatsAppAccount) ([]whatsapp.MetaTemplate, error) {
	waAccount := a.toWhatsAppAccount(account)

	ctx := context.Background()
	return a.WhatsApp.FetchTemplates(ctx, waAccount)
}

func (a *App) deleteTemplateFromMeta(account *models.WhatsAppAccount, templateName string) {
	waAccount := a.toWhatsAppAccount(account)

	ctx := context.Background()
	if err := a.WhatsApp.DeleteTemplate(ctx, waAccount, templateName); err != nil {
		a.Log.Error("Failed to delete template from Meta", "error", err, "template", templateName)
	}
}

// Helper functions

func templateToResponse(t models.Template) TemplateResponse {
	return TemplateResponse{
		ID:                        t.ID,
		WhatsAppAccount:           t.WhatsAppAccount,
		MetaTemplateID:            t.MetaTemplateID,
		Name:                      t.Name,
		DisplayName:               t.DisplayName,
		Language:                  t.Language,
		Category:                  t.Category,
		Status:                    t.Status,
		QualityRating:             t.QualityRating,
		HeaderType:                t.HeaderType,
		HeaderContent:             t.HeaderContent,
		BodyContent:               t.BodyContent,
		FooterContent:             t.FooterContent,
		Buttons:                   convertFromJSONBArray(t.Buttons),
		SampleValues:              convertFromJSONBArray(t.SampleValues),
		AddSecurityRecommendation: t.AddSecurityRecommendation,
		CodeExpirationMinutes:     t.CodeExpirationMinutes,
		CreatedAt:                 t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:                 t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func normalizeTemplateName(name string) string {
	// Convert to lowercase and replace spaces with underscores
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

func convertToJSONBArray(arr []any) models.JSONBArray {
	if arr == nil {
		return models.JSONBArray{}
	}
	return models.JSONBArray(arr)
}

func convertFromJSONBArray(arr models.JSONBArray) []any {
	if arr == nil {
		return []any{}
	}
	return []any(arr)
}

// UploadTemplateMedia uploads a media file for use as template header sample
// Returns a file handle that can be used in template creation

// diffButtons compares old and new button arrays and returns per-button field-level changes.
func diffButtons(oldButtons, newButtons models.JSONBArray) []map[string]any {
	var changes []map[string]any

	toButtonMap := func(btn any) map[string]string {
		m, ok := btn.(map[string]any)
		if !ok {
			return nil
		}
		result := make(map[string]string)
		for k, v := range m {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result
	}

	maxLen := len(oldButtons)
	if len(newButtons) > maxLen {
		maxLen = len(newButtons)
	}

	for i := 0; i < maxLen; i++ {
		label := fmt.Sprintf("Button %d", i+1)
		if i >= len(oldButtons) {
			// New button added
			newBtn := toButtonMap(newButtons[i])
			if newBtn != nil {
				if t := newBtn["text"]; t != "" {
					label = fmt.Sprintf("Button %d (%s)", i+1, t)
				}
			}
			changes = append(changes, map[string]any{
				"field": label, "old_value": nil, "new_value": "added",
			})
			continue
		}
		if i >= len(newButtons) {
			// Button removed
			oldBtn := toButtonMap(oldButtons[i])
			if oldBtn != nil {
				if t := oldBtn["text"]; t != "" {
					label = fmt.Sprintf("Button %d (%s)", i+1, t)
				}
			}
			changes = append(changes, map[string]any{
				"field": label, "old_value": "removed", "new_value": nil,
			})
			continue
		}

		oldBtn := toButtonMap(oldButtons[i])
		newBtn := toButtonMap(newButtons[i])
		if oldBtn == nil || newBtn == nil {
			continue
		}

		// Determine button label from new text (or old if new is empty)
		if t := newBtn["text"]; t != "" {
			label = fmt.Sprintf("Button %d (%s)", i+1, t)
		} else if t := oldBtn["text"]; t != "" {
			label = fmt.Sprintf("Button %d (%s)", i+1, t)
		}

		// Compare individual fields
		fields := []string{"type", "text", "url", "phone_number", "example"}
		for _, f := range fields {
			oldVal, newVal := oldBtn[f], newBtn[f]
			if oldVal != newVal {
				changes = append(changes, map[string]any{
					"field":     label + " → " + f,
					"old_value": oldVal,
					"new_value": newVal,
				})
			}
		}
	}

	return changes
}
