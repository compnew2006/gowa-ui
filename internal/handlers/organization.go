package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// OrganizationSettings represents the settings structure
type OrganizationSettings struct {
	MaskPhoneNumbers            bool       `json:"mask_phone_numbers"`
	StrictSendingRestrictions   bool       `json:"strict_sending_restrictions_enabled"`
	UploadsCleanupRetentionDays int        `json:"uploads_cleanup_retention_days"`
	UploadsCleanupScheduleHour  int        `json:"uploads_cleanup_schedule_hour"`
	OutboundMode                string     `json:"outbound_mode"`
	StrictSendingApplyToSystem  bool       `json:"strict_sending_apply_to_system"`
	CampaignDraftOnly           bool       `json:"campaign_draft_only"`
	StrictRolloutMode           string     `json:"strict_rollout_mode"`
	StrictRolloutEnforceAt      *time.Time `json:"strict_rollout_enforce_at,omitempty"`
	Timezone                    string     `json:"timezone"`
	DateFormat                  string     `json:"date_format"`
}

type organizationSettingsResponse struct {
	Settings OrganizationSettings `json:"settings"`
	Name     string               `json:"name"`
	Slug     string               `json:"slug"`
}

// GetOrganizationSettings returns the organization settings
func (a *App) GetOrganizationSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	canReadGeneral := a.HasPermission(userID, models.ResourceSettingsGeneral, models.ActionRead, orgID)
	canReadUploadsCleanup := a.canAccessUploadsCleanupSettings(userID, orgID)
	if !canReadGeneral && !canReadUploadsCleanup {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	loadDB := requestDB.Session(&gorm.Session{NewDB: true})
	var org models.Organization
	if err := loadDB.Where("id = ?", orgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Parse settings from JSONB
	settings := OrganizationSettings{
		MaskPhoneNumbers:            false,
		StrictSendingRestrictions:   false,
		UploadsCleanupRetentionDays: defaultUploadsCleanupRetentionDays,
		UploadsCleanupScheduleHour:  defaultUploadsCleanupScheduleHour,
		OutboundMode:                organizationOutboundModeMixed,
		StrictSendingApplyToSystem:  true,
		CampaignDraftOnly:           false,
		StrictRolloutMode:           organizationStrictRolloutModeEnforce,
		Timezone:                    "UTC",
		DateFormat:                  "YYYY-MM-DD",
	}

	if org.Settings != nil {
		if canReadGeneral {
			if v, ok := org.Settings["mask_phone_numbers"].(bool); ok {
				settings.MaskPhoneNumbers = v
			}
			if v, ok := org.Settings[organizationSettingStrictSendingRestrictionsEnabled].(bool); ok {
				settings.StrictSendingRestrictions = v
			}
			settings.OutboundMode = normalizeOutboundMode(parseOrganizationStringSetting(org.Settings, organizationSettingOutboundMode, settings.OutboundMode))
			settings.StrictSendingApplyToSystem = parseOrganizationBoolSetting(org.Settings, organizationSettingStrictSendingApplyToSystem, settings.StrictSendingApplyToSystem)
			settings.CampaignDraftOnly = parseOrganizationBoolSetting(org.Settings, organizationSettingCampaignDraftOnly, settings.CampaignDraftOnly)
			settings.StrictRolloutMode = normalizeRolloutMode(parseOrganizationStringSetting(org.Settings, organizationSettingStrictRolloutMode, settings.StrictRolloutMode))
			settings.StrictRolloutEnforceAt = parseOrganizationTimeSetting(org.Settings, organizationSettingStrictRolloutEnforceAt)
			if v, ok := org.Settings["timezone"].(string); ok && v != "" {
				settings.Timezone = v
			}
			if v, ok := org.Settings["date_format"].(string); ok && v != "" {
				settings.DateFormat = v
			}
		}

		if canReadUploadsCleanup {
			settings.UploadsCleanupRetentionDays = parseUploadsCleanupRetentionDays(org.Settings)
			settings.UploadsCleanupScheduleHour = parseUploadsCleanupScheduleHour(org.Settings)
			settings.Timezone = parseOrganizationTimezone(org.Settings)
		}
	}

	return r.SendEnvelope(organizationSettingsResponse{
		Settings: settings,
		Name:     org.Name,
		Slug:     org.Slug,
	})
}

// UpdateOrganizationSettings updates the organization settings
func (a *App) UpdateOrganizationSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req struct {
		MaskPhoneNumbers            *bool   `json:"mask_phone_numbers"`
		StrictSendingRestrictions   *bool   `json:"strict_sending_restrictions_enabled"`
		UploadsCleanupRetentionDays *int    `json:"uploads_cleanup_retention_days"`
		UploadsCleanupScheduleHour  *int    `json:"uploads_cleanup_schedule_hour"`
		OutboundMode                *string `json:"outbound_mode"`
		StrictSendingApplyToSystem  *bool   `json:"strict_sending_apply_to_system"`
		CampaignDraftOnly           *bool   `json:"campaign_draft_only"`
		StrictRolloutMode           *string `json:"strict_rollout_mode"`
		StrictRolloutEnforceAt      *string `json:"strict_rollout_enforce_at"`
		Timezone                    *string `json:"timezone"`
		DateFormat                  *string `json:"date_format"`
		Name                        *string `json:"name"`
		Slug                        *string `json:"slug"`
	}

	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	wantsGeneralSettingsUpdate := req.MaskPhoneNumbers != nil ||
		req.StrictSendingRestrictions != nil ||
		req.OutboundMode != nil ||
		req.StrictSendingApplyToSystem != nil ||
		req.CampaignDraftOnly != nil ||
		req.StrictRolloutMode != nil ||
		req.StrictRolloutEnforceAt != nil ||
		req.Timezone != nil ||
		req.DateFormat != nil ||
		req.Name != nil ||
		req.Slug != nil
	wantsUploadsCleanupUpdate := req.UploadsCleanupRetentionDays != nil || req.UploadsCleanupScheduleHour != nil

	if wantsGeneralSettingsUpdate {
		if err := a.requirePermission(r, userID, models.ResourceSettingsGeneral, models.ActionWrite); err != nil {
			return nil
		}
	}
	if wantsUploadsCleanupUpdate && !a.canWriteUploadsCleanupSettings(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	if req.OutboundMode != nil {
		rawMode := strings.ToLower(strings.TrimSpace(*req.OutboundMode))
		if rawMode != organizationOutboundModeInboundOnly && rawMode != organizationOutboundModeMixed {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "outbound_mode must be inbound_only or mixed", nil, "outbound_mode")
		}
	}
	if req.StrictRolloutMode != nil {
		rawMode := strings.ToLower(strings.TrimSpace(*req.StrictRolloutMode))
		if rawMode != organizationStrictRolloutModeAudit && rawMode != organizationStrictRolloutModeEnforce {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "strict_rollout_mode must be audit or enforce", nil, "strict_rollout_mode")
		}
	}
	if req.StrictRolloutEnforceAt != nil {
		text := strings.TrimSpace(*req.StrictRolloutEnforceAt)
		if text != "" {
			parsed := parseOrganizationTimeSetting(models.JSONB{organizationSettingStrictRolloutEnforceAt: text}, organizationSettingStrictRolloutEnforceAt)
			if parsed == nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "strict_rollout_enforce_at must be a valid timestamp", nil, "strict_rollout_enforce_at")
			}
		}
	}
	if req.UploadsCleanupRetentionDays != nil {
		if *req.UploadsCleanupRetentionDays < 0 || *req.UploadsCleanupRetentionDays > maxUploadsCleanupRetentionDays {
			return r.SendErrorEnvelope(
				fasthttp.StatusBadRequest,
				fmt.Sprintf("uploads_cleanup_retention_days must be between 0 and %d", maxUploadsCleanupRetentionDays),
				nil,
				"uploads_cleanup_retention_days",
			)
		}
	}
	if req.UploadsCleanupScheduleHour != nil {
		if *req.UploadsCleanupScheduleHour < 0 || *req.UploadsCleanupScheduleHour > 23 {
			return r.SendErrorEnvelope(
				fasthttp.StatusBadRequest,
				"uploads_cleanup_schedule_hour must be between 0 and 23",
				nil,
				"uploads_cleanup_schedule_hour",
			)
		}
	}
	if req.Slug != nil && normalizeOrganizationSlug(*req.Slug) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "slug must contain at least one letter or number", nil, "slug")
	}

	var org models.Organization
	if err := requestDB.Where("id = ?", orgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Update settings
	if org.Settings == nil {
		org.Settings = models.JSONB{}
	}

	if req.MaskPhoneNumbers != nil {
		org.Settings["mask_phone_numbers"] = *req.MaskPhoneNumbers
	}
	if req.StrictSendingRestrictions != nil {
		org.Settings[organizationSettingStrictSendingRestrictionsEnabled] = *req.StrictSendingRestrictions
	}
	if req.UploadsCleanupRetentionDays != nil {
		org.Settings[organizationSettingUploadsCleanupRetentionDays] = normalizeUploadsCleanupRetentionDays(*req.UploadsCleanupRetentionDays)
	}
	if req.UploadsCleanupScheduleHour != nil {
		org.Settings[organizationSettingUploadsCleanupScheduleHour] = normalizeUploadsCleanupScheduleHour(*req.UploadsCleanupScheduleHour)
	}
	if req.OutboundMode != nil {
		org.Settings[organizationSettingOutboundMode] = normalizeOutboundMode(*req.OutboundMode)
	}
	if req.StrictSendingApplyToSystem != nil {
		org.Settings[organizationSettingStrictSendingApplyToSystem] = *req.StrictSendingApplyToSystem
	}
	if req.CampaignDraftOnly != nil {
		org.Settings[organizationSettingCampaignDraftOnly] = *req.CampaignDraftOnly
	}
	if req.StrictRolloutMode != nil {
		org.Settings[organizationSettingStrictRolloutMode] = normalizeRolloutMode(*req.StrictRolloutMode)
	}
	if req.StrictRolloutEnforceAt != nil {
		text := strings.TrimSpace(*req.StrictRolloutEnforceAt)
		if text == "" {
			org.Settings[organizationSettingStrictRolloutEnforceAt] = nil
		} else if parsed := parseOrganizationTimeSetting(models.JSONB{organizationSettingStrictRolloutEnforceAt: text}, organizationSettingStrictRolloutEnforceAt); parsed != nil {
			org.Settings[organizationSettingStrictRolloutEnforceAt] = parsed.UTC().Format(time.RFC3339)
		}
	}
	if req.Timezone != nil {
		org.Settings["timezone"] = *req.Timezone
	}
	if req.DateFormat != nil {
		org.Settings["date_format"] = *req.DateFormat
	}
	delete(org.Settings, organizationSettingAssignedChatResetEnabled)
	delete(org.Settings, organizationSettingAssignedChatResetMode)
	delete(org.Settings, organizationSettingAssignedChatResetHour)
	delete(org.Settings, organizationSettingAssignedChatResetLastDate)
	delete(org.Settings, organizationSettingChatCloseRatingEnabled)
	delete(org.Settings, organizationSettingChatCloseRatingWindowDays)
	delete(org.Settings, organizationSettingChatCloseRatingFollowupWindowMinutes)
	delete(org.Settings, organizationSettingChatCloseRatingTemplates)
	if req.Name != nil {
		if trimmedName := strings.TrimSpace(*req.Name); trimmedName != "" {
			org.Name = trimmedName
		}
	}
	if req.Slug != nil {
		nextSlug := normalizeOrganizationSlug(*req.Slug)
		if nextSlug != org.Slug {
			if err := ensureOrganizationSlugAvailable(requestDB.Session(&gorm.Session{NewDB: true}), nextSlug, org.ID); err != nil {
				if errors.Is(err, errOrganizationSlugTaken) {
					return r.SendErrorEnvelope(fasthttp.StatusConflict, "Organization slug is already in use", nil, "slug")
				}
				a.Log.Error("Failed to validate organization slug", "error", err, "organization_id", org.ID)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
			}
			org.Slug = nextSlug
		}
	}

	saveDB := requestDB.Session(&gorm.Session{NewDB: true})
	if err := saveDB.Save(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"message": "Settings updated successfully",
	})
}

// MaskPhoneNumber masks a phone number showing only last 4 digits
func MaskPhoneNumber(phone string) string {
	runes := []rune(phone)
	if len(runes) <= 4 {
		return phone
	}
	masked := ""
	for i := 0; i < len(runes)-4; i++ {
		masked += "*"
	}
	return masked + string(runes[len(runes)-4:])
}

// A generalized chunker that plucks 9-16 digit sequences natively starting with generic indicators.
// This is intentionally broad since structural validation is outsourced to Google libphonenumber.
var intlPhoneRegex = regexp.MustCompile(`(?:^|[^\p{Nd}])((?:\+|00|0|٠٠|٠)[\s\-\.]?[\p{Nd}][\p{Nd}\s\-\.]{5,18}[\p{Nd}]|[\p{Nd}][\p{Nd}\s\-\.]{6,18}[\p{Nd}])(?:[^\p{Nd}]|$)`)

// arabicToASCII maps Arabic-Indic digits to ASCII digits for consistent internal prefix validation
var arabicToASCII = strings.NewReplacer(
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
	"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	"۴", "4", "۵", "5", "۶", "6", // Persian extended
)

// defaultPhoneParsingRegions defines regions to cross-check when a local number is provided.
var defaultPhoneParsingRegions = []string{"SA", "EG", "AE", "US", "GB"}

func MaskPhoneNumbersInText(text string) string {
	matches := intlPhoneRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var result strings.Builder
	lastIndex := 0

	for _, matchIdxs := range matches {
		if len(matchIdxs) < 4 {
			continue // Safeguard against weird regex parses (should always have group 1)
		}

		// matchIdxs[0:2] is the FULL match (including non-digit boundaries)
		// matchIdxs[2:4] is the FIRST capture group (just the phone sequence)
		groupStart := matchIdxs[2]
		groupEnd := matchIdxs[3]

		// Append everything from the end of the last processed chunk up to the start of the CAPTURE GROUP
		// This preserves the leading boundary character, if any
		result.WriteString(text[lastIndex:groupStart])
		rawNumber := text[groupStart:groupEnd]

		// Ensure any Arabic-Indic digits are mapped to standard digits so HasPrefix("00") works seamlessly
		normalizedNumber := arabicToASCII.Replace(rawNumber)

		// Cleanup raw strings slightly for faster parsing attempts
		stripped := strings.ReplaceAll(normalizedNumber, " ", "")
		stripped = strings.ReplaceAll(stripped, "-", "")
		stripped = strings.ReplaceAll(stripped, ".", "")

		isValid := false

		asIntl := stripped
		if strings.HasPrefix(asIntl, "00") {
			asIntl = "+" + asIntl[2:]
		} else if !strings.HasPrefix(asIntl, "+") && !strings.HasPrefix(asIntl, "0") {
			asIntl = "+" + asIntl
		}

		numIntl, errIntl := phonenumbers.Parse(asIntl, "ZZ")
		if errIntl == nil && phonenumbers.IsValidNumber(numIntl) {
			isValid = true
		}

		if !isValid {
			for _, region := range defaultPhoneParsingRegions {
				numLocal, errLocal := phonenumbers.Parse(stripped, region)
				if errLocal == nil && phonenumbers.IsValidNumber(numLocal) {
					isValid = true
					break
				}
			}
		}

		if isValid {
			result.WriteString(MaskPhoneNumber(rawNumber))
		} else {
			result.WriteString(rawNumber)
		}

		lastIndex = groupEnd
	}

	// Append any remaining trailing characters after the last processed capture group
	result.WriteString(text[lastIndex:])
	return result.String()
}

// LooksLikePhoneNumber checks if a string looks like a phone number
// (mostly digits, optionally with common phone formatting characters)
func LooksLikePhoneNumber(s string) bool {
	if len(s) < 7 {
		return false
	}
	digitCount := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digitCount++
		}
	}
	// If at least 7 digits and more than 70% of the string is digits
	return digitCount >= 7 && float64(digitCount)/float64(len(s)) > 0.7
}

// MaskIfPhoneNumber masks a string if it looks like a phone number
func MaskIfPhoneNumber(s string) string {
	if LooksLikePhoneNumber(s) {
		return MaskPhoneNumber(s)
	}
	return s
}

func maskContactPhoneAndName(phone, name string, shouldMask bool) (string, string) {
	if !shouldMask {
		return phone, name
	}
	return MaskPhoneNumber(phone), MaskIfPhoneNumber(name)
}

// ShouldMaskPhoneNumbers checks if phone masking is enabled for the organization
func (a *App) ShouldMaskPhoneNumbers(orgID interface{}) bool {
	var org models.Organization
	if err := a.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		return false
	}

	if org.Settings != nil {
		if v, ok := org.Settings["mask_phone_numbers"].(bool); ok {
			return v
		}
	}
	return false
}

// OrganizationResponse represents an organization in API responses
type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug,omitempty"`
	CreatedAt string    `json:"created_at"`
}

// ListOrganizations returns all organizations (super admin or users with organizations:read)
func (a *App) ListOrganizations(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Super admins or users with organizations:read permission
	if !a.IsSuperAdmin(userID) && !a.HasPermission(userID, models.ResourceOrganizations, models.ActionRead, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	var orgs []models.Organization
	if err := requestDB.Order("name ASC").Find(&orgs).Error; err != nil {
		a.Log.Error("Failed to list organizations", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list organizations", nil, "")
	}

	response := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		response[i] = OrganizationResponse{
			ID:        org.ID,
			Name:      org.Name,
			Slug:      org.Slug,
			CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return r.SendEnvelope(map[string]any{
		"organizations": response,
	})
}

// GetCurrentOrganization returns the current user's organization details
func (a *App) GetCurrentOrganization(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var org models.Organization
	if err := requestDB.Where("id = ?", orgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	return r.SendEnvelope(OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// CreateOrganizationRequest represents the request body for creating an organization
type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// CreateOrganization creates a new organization
func (a *App) CreateOrganization(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionWrite); err != nil {
		return nil
	}

	var req CreateOrganizationRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Organization name is required", nil, "")
	}
	if !a.checkQuotaOrRespond(r, license.ResourceOrganizations, uuid.Nil) {
		return nil
	}

	// Start transaction
	tx := requestDB.Begin()
	if tx.Error != nil {
		a.Log.Error("Failed to begin transaction", "error", tx.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	slug, err := resolveOrganizationSlug(tx, req.Slug, req.Name, uuid.Nil)
	if err != nil {
		tx.Rollback()
		switch {
		case errors.Is(err, errInvalidOrganizationSlug):
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "slug must contain at least one letter or number", nil, "slug")
		case errors.Is(err, errOrganizationSlugTaken):
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "Organization slug is already in use", nil, "slug")
		default:
			a.Log.Error("Failed to resolve organization slug", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
		}
	}

	org := models.Organization{
		Name:     req.Name,
		Slug:     slug,
		Settings: models.JSONB{},
	}

	if err := tx.Create(&org).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to create organization", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	// Seed system roles for the new organization
	if err := database.SeedSystemRolesForOrg(tx, org.ID); err != nil {
		tx.Rollback()
		a.Log.Error("Failed to seed system roles", "error", err, "org_id", org.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	// Create default chatbot settings
	chatbotSettings := models.ChatbotSettings{
		OrganizationID:     org.ID,
		IsEnabled:          false,
		SessionTimeoutMins: 30,
	}
	if err := tx.Create(&chatbotSettings).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to create chatbot settings", "error", err, "org_id", org.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	// Get admin role for this org and add the creator as admin
	var adminRole models.CustomRole
	if err := tx.Where("organization_id = ? AND name = ? AND is_system = ?", org.ID, "admin", true).First(&adminRole).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to find admin role", "error", err, "org_id", org.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	userOrg := models.UserOrganization{
		UserID:         userID,
		OrganizationID: org.ID,
		RoleID:         &adminRole.ID,
		IsDefault:      false,
	}
	if err := tx.Create(&userOrg).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to add creator to organization", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	if err := tx.Commit().Error; err != nil {
		a.Log.Error("Failed to commit transaction", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	a.Log.Info("Created organization", "org_id", org.ID, "org_name", org.Name, "created_by", userID)

	return r.SendEnvelope(OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// DeleteOrganization removes an organization and disables access for its users.
func (a *App) DeleteOrganization(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionDelete); err != nil {
		return nil
	}

	targetOrgID, err := parsePathUUID(r, "id", "organization")
	if err != nil {
		return nil
	}

	var org models.Organization
	if err := requestDB.Session(&gorm.Session{}).
		Where("id = ?", targetOrgID).
		First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Keep at least one active organization in the system.
	var orgCount int64
	if err := requestDB.Session(&gorm.Session{}).
		Model(&models.Organization{}).
		Count(&orgCount).Error; err != nil {
		a.Log.Error("Failed to count organizations before delete", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}
	if orgCount <= 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete the last organization", nil, "")
	}

	var currentUser models.User
	if err := requestDB.Session(&gorm.Session{}).
		Select("id", "organization_id").
		Where("id = ?", userID).
		First(&currentUser).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if currentUser.OrganizationID == targetOrgID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete your home organization", nil, "")
	}

	tx := requestDB.Session(&gorm.Session{}).Begin()
	if tx.Error != nil {
		a.Log.Error("Failed to begin transaction for organization delete", "error", tx.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	var memberUserIDs []uuid.UUID
	if err := tx.Model(&models.UserOrganization{}).
		Where("organization_id = ?", targetOrgID).
		Distinct().
		Pluck("user_id", &memberUserIDs).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to list organization members for delete", "error", err, "organization_id", targetOrgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	var nativeUserIDs []uuid.UUID
	if err := tx.Model(&models.User{}).
		Where("organization_id = ?", targetOrgID).
		Pluck("id", &nativeUserIDs).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to list native users for organization delete", "error", err, "organization_id", targetOrgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	if err := tx.Where("organization_id = ?", targetOrgID).Delete(&models.UserOrganization{}).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to delete organization memberships", "error", err, "organization_id", targetOrgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	// Users whose home org is being deleted are soft-deleted to avoid orphaned auth state.
	if len(nativeUserIDs) > 0 {
		if err := tx.Where("user_id IN ?", nativeUserIDs).Delete(&models.UserOrganization{}).Error; err != nil {
			tx.Rollback()
			a.Log.Error("Failed to delete user memberships for native users", "error", err, "organization_id", targetOrgID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
		}
		if err := tx.Where("id IN ?", nativeUserIDs).Delete(&models.User{}).Error; err != nil {
			tx.Rollback()
			a.Log.Error("Failed to delete native users for organization", "error", err, "organization_id", targetOrgID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
		}
	}

	if err := tx.Delete(&org).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to delete organization record", "error", err, "organization_id", targetOrgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	if err := tx.Commit().Error; err != nil {
		a.Log.Error("Failed to commit organization delete transaction", "error", err, "organization_id", targetOrgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}

	affectedUsers := make(map[uuid.UUID]struct{}, len(memberUserIDs)+len(nativeUserIDs))
	for _, id := range memberUserIDs {
		affectedUsers[id] = struct{}{}
	}
	for _, id := range nativeUserIDs {
		affectedUsers[id] = struct{}{}
	}
	for id := range affectedUsers {
		a.InvalidateUserPermissionsCache(id)
	}

	return r.SendEnvelope(map[string]string{"message": "Organization deleted successfully"})
}

// MemberResponse represents an organization member in API responses
type MemberResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	RoleID         *uuid.UUID `json:"role_id,omitempty"`
	RoleName       string     `json:"role_name,omitempty"`
	IsDefault      bool       `json:"is_default"`
	Email          string     `json:"email"`
	FullName       string     `json:"full_name"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ListOrganizationMembers returns all members of the current organization
func (a *App) ListOrganizationMembers(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	baseQuery := requestDB.Table("user_organizations").
		Joins("LEFT JOIN users ON users.id = user_organizations.user_id AND users.deleted_at IS NULL").
		Joins("LEFT JOIN custom_roles ON custom_roles.id = user_organizations.role_id AND custom_roles.deleted_at IS NULL").
		Where("user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID)

	if search != "" {
		baseQuery = baseQuery.Where("users.full_name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	baseQuery.Count(&total)

	var response []MemberResponse
	if err := pg.Apply(baseQuery.
		Select(`user_organizations.id, user_organizations.user_id, user_organizations.organization_id,
			user_organizations.role_id, user_organizations.is_default, user_organizations.created_at,
			users.email, users.full_name, users.is_active,
			custom_roles.name AS role_name`).
		Order("user_organizations.created_at DESC")).
		Scan(&response).Error; err != nil {
		a.Log.Error("Failed to list organization members", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list members", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"members": response,
		"total":   total,
		"page":    pg.Page,
		"limit":   pg.Limit,
	})
}

// AddMemberRequest represents the request body for adding a member to an organization
type AddMemberRequest struct {
	UserID uuid.UUID  `json:"user_id"`
	Email  string     `json:"email"`
	RoleID *uuid.UUID `json:"role_id"`
}

// AddOrganizationMember adds an existing user to the current organization
func (a *App) AddOrganizationMember(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionAssign); err != nil {
		return nil
	}

	var req AddMemberRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Resolve target user by user_id or email
	var targetUser models.User
	if req.UserID != uuid.Nil {
		if err := requestDB.Where("id = ?", req.UserID).First(&targetUser).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
		}
	} else if req.Email != "" {
		if err := requestDB.Where("email = ?", req.Email).First(&targetUser).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No user found with this email", nil, "")
		}
	} else {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "user_id or email is required", nil, "")
	}

	// Check if already a member
	var existingCount int64
	requestDB.
		Model(&models.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", targetUser.ID, orgID).
		Count(&existingCount)
	if existingCount > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "User is already a member of this organization", nil, "")
	}
	if !a.checkQuotaOrRespond(r, license.ResourceUsers, orgID) {
		return nil
	}

	// Determine role
	var roleID *uuid.UUID
	if req.RoleID != nil {
		// Validate role exists and belongs to org
		var role models.CustomRole
		if err := requestDB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&role).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
		}
		roleID = req.RoleID
	} else {
		// Use org's default role
		var defaultRole models.CustomRole
		if err := requestDB.Where("organization_id = ? AND is_default = ?", orgID, true).First(&defaultRole).Error; err == nil {
			roleID = &defaultRole.ID
		}
	}

	userOrg := models.UserOrganization{
		UserID:         targetUser.ID,
		OrganizationID: orgID,
		RoleID:         roleID,
		IsDefault:      false,
	}

	if err := requestDB.Create(&userOrg).Error; err != nil {
		a.Log.Error("Failed to add organization member", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to add member", nil, "")
	}

	return r.SendEnvelope(map[string]string{"message": "Member added successfully"})
}

// RemoveOrganizationMember removes a user from the current organization
func (a *App) RemoveOrganizationMember(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionAssign); err != nil {
		return nil
	}

	targetUserID, err := parsePathUUID(r, "member_id", "member")
	if err != nil {
		return nil
	}

	// Cannot remove self
	if targetUserID == userID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot remove yourself from the organization", nil, "")
	}

	result := requestDB.Where("user_id = ? AND organization_id = ?", targetUserID, orgID).
		Delete(&models.UserOrganization{})
	if result.Error != nil {
		a.Log.Error("Failed to remove organization member", "error", result.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove member", nil, "")
	}
	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Member not found in this organization", nil, "")
	}

	// Invalidate removed user's permission cache
	a.InvalidateUserPermissionsCache(targetUserID)

	return r.SendEnvelope(map[string]string{"message": "Member removed successfully"})
}

// UpdateMemberRoleRequest represents the request body for updating a member's role
type UpdateMemberRoleRequest struct {
	RoleID uuid.UUID `json:"role_id"`
}

// UpdateOrganizationMemberRole updates a member's role in the current organization
func (a *App) UpdateOrganizationMemberRole(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionAssign); err != nil {
		return nil
	}

	targetUserID, err := parsePathUUID(r, "member_id", "member")
	if err != nil {
		return nil
	}

	var req UpdateMemberRoleRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.RoleID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "role_id is required", nil, "")
	}

	// Validate role exists and belongs to org
	var role models.CustomRole
	if err := requestDB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&role).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
	}

	// Update the user's role in this org
	result := requestDB.Model(&models.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", targetUserID, orgID).
		Update("role_id", req.RoleID)
	if result.Error != nil {
		a.Log.Error("Failed to update member role", "error", result.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update member role", nil, "")
	}
	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Member not found in this organization", nil, "")
	}

	// Invalidate permission cache
	a.InvalidateUserPermissionsCache(targetUserID)

	return r.SendEnvelope(map[string]string{"message": "Member role updated successfully"})
}
