package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// OrganizationSettings represents the settings structure
type OrganizationSettings struct {
	MaskPhoneNumbers                     bool              `json:"mask_phone_numbers"`
	StrictSendingRestrictions            bool              `json:"strict_sending_restrictions_enabled"`
	Timezone                             string            `json:"timezone"`
	DateFormat                           string            `json:"date_format"`
	AssignedChatResetEnabled             bool              `json:"assigned_chat_reset_enabled"`
	AssignedChatResetMode                string            `json:"assigned_chat_reset_mode"`
	AssignedChatResetHour                int               `json:"assigned_chat_reset_hour"`
	ChatCloseRatingEnabled               bool              `json:"chat_close_rating_enabled"`
	ChatCloseRatingWindowDays            int               `json:"chat_close_rating_window_days"`
	ChatCloseRatingFollowupWindowMinutes int               `json:"chat_close_rating_followup_window_minutes"`
	ChatCloseRatingTemplates             map[string]string `json:"chat_close_rating_templates"`
}

// GetOrganizationSettings returns the organization settings
func (a *App) GetOrganizationSettings(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var org models.Organization
	if err := a.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Parse settings from JSONB
	settings := OrganizationSettings{
		MaskPhoneNumbers:                     false,
		StrictSendingRestrictions:            false,
		Timezone:                             "UTC",
		DateFormat:                           "YYYY-MM-DD",
		AssignedChatResetEnabled:             true,
		AssignedChatResetMode:                string(ChatAssignmentResetModeMidnight),
		AssignedChatResetHour:                0,
		ChatCloseRatingEnabled:               true,
		ChatCloseRatingWindowDays:            defaultChatCloseRatingWindowDays,
		ChatCloseRatingFollowupWindowMinutes: defaultChatCloseRatingFollowupWindowMinutes,
		ChatCloseRatingTemplates:             cloneDefaultChatCloseRatingTemplates(),
	}

	if org.Settings != nil {
		if v, ok := org.Settings["mask_phone_numbers"].(bool); ok {
			settings.MaskPhoneNumbers = v
		}
		if v, ok := org.Settings[organizationSettingStrictSendingRestrictionsEnabled].(bool); ok {
			settings.StrictSendingRestrictions = v
		}
		if v, ok := org.Settings["timezone"].(string); ok && v != "" {
			settings.Timezone = v
		}
		if v, ok := org.Settings["date_format"].(string); ok && v != "" {
			settings.DateFormat = v
		}

		chatResetSettings := readChatAssignmentResetSettings(org.Settings)
		settings.AssignedChatResetEnabled = chatResetSettings.Enabled
		settings.AssignedChatResetMode = string(chatResetSettings.Mode)
		settings.AssignedChatResetHour = chatResetSettings.Hour

		chatCloseRatingSettings := readChatCloseRatingSettings(org.Settings)
		settings.ChatCloseRatingEnabled = chatCloseRatingSettings.Enabled
		settings.ChatCloseRatingWindowDays = chatCloseRatingSettings.WindowDays
		settings.ChatCloseRatingFollowupWindowMinutes = chatCloseRatingSettings.FollowupWindowMinutes
		settings.ChatCloseRatingTemplates = chatCloseRatingSettings.Templates
	}

	return r.SendEnvelope(map[string]interface{}{
		"settings": settings,
		"name":     org.Name,
	})
}

// UpdateOrganizationSettings updates the organization settings
func (a *App) UpdateOrganizationSettings(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req struct {
		MaskPhoneNumbers                     *bool              `json:"mask_phone_numbers"`
		StrictSendingRestrictions            *bool              `json:"strict_sending_restrictions_enabled"`
		Timezone                             *string            `json:"timezone"`
		DateFormat                           *string            `json:"date_format"`
		Name                                 *string            `json:"name"`
		AssignedChatResetEnabled             *bool              `json:"assigned_chat_reset_enabled"`
		AssignedChatResetMode                *string            `json:"assigned_chat_reset_mode"`
		AssignedChatResetHour                *int               `json:"assigned_chat_reset_hour"`
		ChatCloseRatingEnabled               *bool              `json:"chat_close_rating_enabled"`
		ChatCloseRatingWindowDays            *int               `json:"chat_close_rating_window_days"`
		ChatCloseRatingFollowupWindowMinutes *int               `json:"chat_close_rating_followup_window_minutes"`
		ChatCloseRatingTemplates             *map[string]string `json:"chat_close_rating_templates"`
	}

	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	if err := validateChatAssignmentResetInputs(req.AssignedChatResetMode, req.AssignedChatResetHour); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	if req.ChatCloseRatingWindowDays != nil {
		if *req.ChatCloseRatingWindowDays < 1 || *req.ChatCloseRatingWindowDays > maxChatCloseRatingWindowDays {
			return r.SendErrorEnvelope(
				fasthttp.StatusBadRequest,
				fmt.Sprintf("chat_close_rating_window_days must be between 1 and %d", maxChatCloseRatingWindowDays),
				nil,
				"",
			)
		}
	}
	if req.ChatCloseRatingFollowupWindowMinutes != nil {
		if *req.ChatCloseRatingFollowupWindowMinutes < 1 || *req.ChatCloseRatingFollowupWindowMinutes > maxChatCloseRatingFollowupWindowMinutes {
			return r.SendErrorEnvelope(
				fasthttp.StatusBadRequest,
				fmt.Sprintf("chat_close_rating_followup_window_minutes must be between 1 and %d", maxChatCloseRatingFollowupWindowMinutes),
				nil,
				"",
			)
		}
	}

	var org models.Organization
	if err := a.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
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
	if req.Timezone != nil {
		org.Settings["timezone"] = *req.Timezone
	}
	if req.DateFormat != nil {
		org.Settings["date_format"] = *req.DateFormat
	}
	if req.AssignedChatResetEnabled != nil {
		org.Settings[organizationSettingAssignedChatResetEnabled] = *req.AssignedChatResetEnabled
	}
	if req.ChatCloseRatingEnabled != nil {
		org.Settings[organizationSettingChatCloseRatingEnabled] = *req.ChatCloseRatingEnabled
	}
	if req.ChatCloseRatingWindowDays != nil {
		org.Settings[organizationSettingChatCloseRatingWindowDays] = *req.ChatCloseRatingWindowDays
	}
	if req.ChatCloseRatingFollowupWindowMinutes != nil {
		org.Settings[organizationSettingChatCloseRatingFollowupWindowMinutes] = *req.ChatCloseRatingFollowupWindowMinutes
	}
	if req.ChatCloseRatingTemplates != nil {
		parsedTemplates := parseChatCloseRatingTemplates(*req.ChatCloseRatingTemplates)
		templateJSON := models.JSONB{}
		for language, template := range parsedTemplates {
			templateJSON[language] = template
		}
		org.Settings[organizationSettingChatCloseRatingTemplates] = templateJSON
	}

	modeProvided := false
	var selectedResetMode ChatAssignmentResetMode
	if req.AssignedChatResetMode != nil {
		modeProvided = true
		selectedResetMode = normalizeChatAssignmentResetMode(*req.AssignedChatResetMode)
		org.Settings[organizationSettingAssignedChatResetMode] = string(selectedResetMode)
	}
	if req.AssignedChatResetHour != nil {
		org.Settings[organizationSettingAssignedChatResetHour] = *req.AssignedChatResetHour
	}
	if modeProvided && selectedResetMode == ChatAssignmentResetModeMidnight {
		org.Settings[organizationSettingAssignedChatResetHour] = 0
	}
	if req.Name != nil && *req.Name != "" {
		org.Name = *req.Name
	}

	if err := a.DB.Save(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"message": "Settings updated successfully",
	})
}

// MaskPhoneNumber masks a phone number showing only last 4 digits
func MaskPhoneNumber(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	masked := ""
	for i := 0; i < len(phone)-4; i++ {
		masked += "*"
	}
	return masked + phone[len(phone)-4:]
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
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Super admins or users with organizations:read permission
	if !a.IsSuperAdmin(userID) && !a.HasPermission(userID, models.ResourceOrganizations, models.ActionRead) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	var orgs []models.Organization
	if err := a.DB.Order("name ASC").Find(&orgs).Error; err != nil {
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
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var org models.Organization
	if err := a.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
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
}

// CreateOrganization creates a new organization
func (a *App) CreateOrganization(r *fastglue.Request) error {
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

	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Organization name is required", nil, "")
	}

	// Start transaction
	tx := a.DB.Begin()
	if tx.Error != nil {
		a.Log.Error("Failed to begin transaction", "error", tx.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create organization", nil, "")
	}

	org := models.Organization{
		Name:     req.Name,
		Slug:     generateSlug(req.Name),
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
	if err := a.DB.Where("id = ?", targetOrgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Keep at least one active organization in the system.
	var orgCount int64
	if err := a.DB.Model(&models.Organization{}).Count(&orgCount).Error; err != nil {
		a.Log.Error("Failed to count organizations before delete", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete organization", nil, "")
	}
	if orgCount <= 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete the last organization", nil, "")
	}

	var currentUser models.User
	if err := a.DB.Select("id", "organization_id").Where("id = ?", userID).First(&currentUser).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if currentUser.OrganizationID == targetOrgID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete your home organization", nil, "")
	}

	tx := a.DB.Begin()
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
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceOrganizations, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	baseQuery := a.DB.Table("user_organizations").
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
		if err := a.DB.Where("id = ?", req.UserID).First(&targetUser).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
		}
	} else if req.Email != "" {
		if err := a.DB.Where("email = ?", req.Email).First(&targetUser).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No user found with this email", nil, "")
		}
	} else {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "user_id or email is required", nil, "")
	}

	// Check if already a member
	var existingCount int64
	a.DB.Model(&models.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", targetUser.ID, orgID).
		Count(&existingCount)
	if existingCount > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "User is already a member of this organization", nil, "")
	}

	// Determine role
	var roleID *uuid.UUID
	if req.RoleID != nil {
		// Validate role exists and belongs to org
		var role models.CustomRole
		if err := a.DB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&role).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
		}
		roleID = req.RoleID
	} else {
		// Use org's default role
		var defaultRole models.CustomRole
		if err := a.DB.Where("organization_id = ? AND is_default = ?", orgID, true).First(&defaultRole).Error; err == nil {
			roleID = &defaultRole.ID
		}
	}

	userOrg := models.UserOrganization{
		UserID:         targetUser.ID,
		OrganizationID: orgID,
		RoleID:         roleID,
		IsDefault:      false,
	}

	if err := a.DB.Create(&userOrg).Error; err != nil {
		a.Log.Error("Failed to add organization member", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to add member", nil, "")
	}

	return r.SendEnvelope(map[string]string{"message": "Member added successfully"})
}

// RemoveOrganizationMember removes a user from the current organization
func (a *App) RemoveOrganizationMember(r *fastglue.Request) error {
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

	result := a.DB.Where("user_id = ? AND organization_id = ?", targetUserID, orgID).
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
	if err := a.DB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&role).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
	}

	// Update the user's role in this org
	result := a.DB.Model(&models.UserOrganization{}).
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
