package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
)

// UserRequest represents the request body for creating/updating a user.
// Note: is_super_admin is intentionally excluded to prevent mass assignment.
// Super admin status changes are handled via parseSuperAdminField.
type UserRequest struct {
	Email    string     `json:"email"`
	Password string     `json:"password"`
	FullName string     `json:"full_name"`
	RoleID   *uuid.UUID `json:"role_id"`
	IsActive *bool      `json:"is_active"`
}

// superAdminField is used to extract is_super_admin separately from the request body.
// Only super admins can use this field.
type superAdminField struct {
	IsSuperAdmin *bool `json:"is_super_admin"`
}

// parseSuperAdminField extracts is_super_admin from the raw request body.
func parseSuperAdminField(r *fastglue.Request) *bool {
	var f superAdminField
	if err := r.Decode(&f, "json"); err != nil {
		return nil
	}
	return f.IsSuperAdmin
}

// UserResponse represents the response for a user (without sensitive data)
type UserResponse struct {
	ID             uuid.UUID    `json:"id"`
	Email          string       `json:"email"`
	FullName       string       `json:"full_name"`
	RoleID         *uuid.UUID   `json:"role_id,omitempty"`
	Role           *RoleInfo    `json:"role,omitempty"`
	IsActive       bool         `json:"is_active"`
	IsAvailable    bool         `json:"is_available"`
	IsSuperAdmin   bool         `json:"is_super_admin"`
	IsMember       bool         `json:"is_member"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	Settings       models.JSONB `json:"settings,omitempty"`
	CreatedAt      string       `json:"created_at"`
	UpdatedAt      string       `json:"updated_at"`
}

// PermissionInfo represents permission info in role response
type PermissionInfo struct {
	ID          uuid.UUID `json:"id"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description,omitempty"`
}

// RoleInfo represents role info in user response
type RoleInfo struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	IsSystem    bool             `json:"is_system"`
	Permissions []PermissionInfo `json:"permissions"`
}

// UserSettingsRequest represents notification/settings preferences
type UserSettingsRequest struct {
	EmailNotifications *bool                           `json:"email_notifications"`
	NewMessageAlerts   *bool                           `json:"new_message_alerts"`
	CampaignUpdates    *bool                           `json:"campaign_updates"`
	NotificationSound  *string                         `json:"notification_sound"`
	ThemeMode          *string                         `json:"theme_mode"`
	ThemePreset        *string                         `json:"theme_preset"`
	ChatBackground     OptionalUserChatBackgroundField `json:"chat_background"`
}

type UserChatBackgroundRequest struct {
	Kind           string `json:"kind"`
	PresetID       string `json:"preset_id,omitempty"`
	CustomAssetID  string `json:"custom_asset_id,omitempty"`
	CustomFilename string `json:"custom_filename,omitempty"`
	CustomMimeType string `json:"custom_mime_type,omitempty"`
}

type UserChatBackground struct {
	Kind           string `json:"kind"`
	PresetID       string `json:"preset_id,omitempty"`
	CustomAssetID  string `json:"custom_asset_id,omitempty"`
	CustomFilename string `json:"custom_filename,omitempty"`
	CustomMimeType string `json:"custom_mime_type,omitempty"`
}

type OptionalUserChatBackgroundField struct {
	Set   bool
	Value *UserChatBackgroundRequest
}

func (f *OptionalUserChatBackgroundField) UnmarshalJSON(data []byte) error {
	f.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Value = nil
		return nil
	}

	var value UserChatBackgroundRequest
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.Value = &value
	return nil
}

const (
	notificationSound1          = "notification1"
	notificationSound2          = "notification2"
	notificationSound           = "notification"
	themeModeLight              = "light"
	themeModeDark               = "dark"
	themeModeSystem             = "system"
	themePresetTwitter          = "twitter"
	themePresetOceanBreeze      = "ocean-breeze"
	themePresetSoftPop          = "soft-pop"
	themePresetCaffeineLegacy   = "caffeine"
	themePresetAmberMinimal     = "amber-minimal"
	maxChatBackgroundUploadSize = 5 << 20
	chatBackgroundKindPreset    = "preset"
	chatBackgroundKindCustom    = "custom"
)

var allowedChatBackgroundPresetIDs = map[string]struct{}{
	"aurora-veil":  {},
	"sunset-dunes": {},
	"paper-garden": {},
	"linen-grid":   {},
	"dot-orbit":    {},
	"ripple-lines": {},
}

func normalizeNotificationSound(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case notificationSound2:
		return notificationSound2
	case notificationSound:
		return notificationSound
	case notificationSound1:
		fallthrough
	default:
		return notificationSound1
	}
}

func normalizeThemeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case themeModeLight:
		return themeModeLight
	case themeModeDark:
		return themeModeDark
	case themeModeSystem:
		fallthrough
	default:
		return themeModeSystem
	}
}

func normalizeThemePreset(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case themePresetCaffeineLegacy:
		return themePresetSoftPop
	case themePresetSoftPop:
		return themePresetSoftPop
	case themePresetAmberMinimal:
		return themePresetAmberMinimal
	case themePresetOceanBreeze:
		return themePresetOceanBreeze
	case themePresetTwitter:
		fallthrough
	default:
		return themePresetTwitter
	}
}

func normalizeChatBackgroundMIME(raw string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(raw, ";")[0])) {
	case "image/jpeg":
		return "image/jpeg"
	case "image/png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	default:
		return ""
	}
}

func isAllowedChatBackgroundPresetID(id string) bool {
	_, ok := allowedChatBackgroundPresetIDs[strings.TrimSpace(id)]
	return ok
}

func decodeStoredChatBackground(raw any) *UserChatBackground {
	if raw == nil {
		return nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var bg UserChatBackground
	if err := json.Unmarshal(payload, &bg); err != nil {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(bg.Kind)) {
	case chatBackgroundKindPreset:
		bg.PresetID = strings.TrimSpace(bg.PresetID)
		if !isAllowedChatBackgroundPresetID(bg.PresetID) {
			return nil
		}
		return &UserChatBackground{
			Kind:     chatBackgroundKindPreset,
			PresetID: bg.PresetID,
		}
	case chatBackgroundKindCustom:
		bg.CustomAssetID = strings.TrimSpace(bg.CustomAssetID)
		bg.CustomFilename = sanitizeFilename(bg.CustomFilename)
		bg.CustomMimeType = normalizeChatBackgroundMIME(bg.CustomMimeType)
		if bg.CustomAssetID == "" || bg.CustomMimeType == "" {
			return nil
		}
		if bg.CustomFilename == "" || bg.CustomFilename == "unnamed" {
			bg.CustomFilename = bg.CustomAssetID + getExtensionFromMimeType(bg.CustomMimeType)
		}
		return &UserChatBackground{
			Kind:           chatBackgroundKindCustom,
			CustomAssetID:  bg.CustomAssetID,
			CustomFilename: bg.CustomFilename,
			CustomMimeType: bg.CustomMimeType,
		}
	default:
		return nil
	}
}

func (bg UserChatBackground) toSettingsValue() map[string]any {
	if bg.Kind == chatBackgroundKindPreset {
		return map[string]any{
			"kind":      bg.Kind,
			"preset_id": bg.PresetID,
		}
	}

	return map[string]any{
		"kind":             bg.Kind,
		"custom_asset_id":  bg.CustomAssetID,
		"custom_filename":  bg.CustomFilename,
		"custom_mime_type": bg.CustomMimeType,
	}
}

func chatBackgroundRelativePath(userID uuid.UUID, bg *UserChatBackground) string {
	if bg == nil || bg.Kind != chatBackgroundKindCustom {
		return ""
	}
	ext := getExtensionFromMimeType(bg.CustomMimeType)
	if ext == "" {
		return ""
	}
	return filepath.Join("chat-backgrounds", userID.String(), bg.CustomAssetID+ext)
}

func (a *App) deleteStoredMediaFile(relativePath string) error {
	cleanRelativePath := filepath.Clean(strings.TrimSpace(relativePath))
	if cleanRelativePath == "" || cleanRelativePath == "." {
		return nil
	}

	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		return fmt.Errorf("resolve storage path: %w", err)
	}
	fullPath, err := filepath.Abs(filepath.Join(baseDir, cleanRelativePath))
	if err != nil {
		return fmt.Errorf("resolve file path: %w", err)
	}
	if !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path")
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to delete symlink")
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

func (a *App) resolveRequestedChatBackground(
	user *models.User,
	req *UserChatBackgroundRequest,
) (*UserChatBackground, string, error) {
	existing := decodeStoredChatBackground(user.Settings["chat_background"])
	if req == nil {
		cleanupPath := ""
		if existing != nil && existing.Kind == chatBackgroundKindCustom {
			cleanupPath = chatBackgroundRelativePath(user.ID, existing)
		}
		return nil, cleanupPath, nil
	}
	switch strings.ToLower(strings.TrimSpace(req.Kind)) {
	case chatBackgroundKindPreset:
		presetID := strings.TrimSpace(req.PresetID)
		if !isAllowedChatBackgroundPresetID(presetID) {
			return nil, "", fmt.Errorf("invalid chat background preset")
		}
		cleanupPath := ""
		if existing != nil && existing.Kind == chatBackgroundKindCustom {
			cleanupPath = chatBackgroundRelativePath(user.ID, existing)
		}
		return &UserChatBackground{
			Kind:     chatBackgroundKindPreset,
			PresetID: presetID,
		}, cleanupPath, nil
	case chatBackgroundKindCustom:
		if existing == nil || existing.Kind != chatBackgroundKindCustom {
			return nil, "", fmt.Errorf("upload a custom chat background before saving")
		}
		return existing, "", nil
	default:
		return nil, "", fmt.Errorf("invalid chat background kind")
	}
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ListUsers returns all users for the organization
func (a *App) ListUsers(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceUsers, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))

	// Query users via user_organizations to include cross-org members.
	joinClause := "JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL"

	countQuery := a.DB.Joins(joinClause, orgID).Where("users.deleted_at IS NULL")
	dataQuery := a.DB.Joins(joinClause, orgID).Where("users.deleted_at IS NULL")
	if search != "" {
		countQuery = countQuery.Where("users.full_name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
		dataQuery = dataQuery.Where("users.full_name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	countQuery.Model(&models.User{}).Count(&total)

	var users []models.User
	if err := pg.Apply(dataQuery.Order("users.created_at DESC")).
		Find(&users).Error; err != nil {
		a.Log.Error("Failed to list users", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list users", nil, "")
	}

	// Fetch org-specific roles and membership info from user_organizations
	userIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	var orgMemberships []models.UserOrganization
	if len(userIDs) > 0 {
		a.DB.Where("user_id IN ? AND organization_id = ?", userIDs, orgID).
			Preload("Role").
			Find(&orgMemberships)
	}
	orgRoleMap := make(map[uuid.UUID]*models.CustomRole, len(orgMemberships))
	for _, m := range orgMemberships {
		if m.Role != nil {
			orgRoleMap[m.UserID] = m.Role
		}
	}

	// Fetch actual home org IDs (separate query avoids JOIN column conflict)
	homeOrgMap := make(map[uuid.UUID]uuid.UUID, len(users))
	if len(userIDs) > 0 {
		type idOrg struct {
			ID             uuid.UUID
			OrganizationID uuid.UUID
		}
		var homeOrgs []idOrg
		a.DB.Model(&models.User{}).Select("id, organization_id").Where("id IN ?", userIDs).Find(&homeOrgs)
		for _, ho := range homeOrgs {
			homeOrgMap[ho.ID] = ho.OrganizationID
		}
	}

	// Convert to response format, using org-specific role
	response := make([]UserResponse, len(users))
	for i, user := range users {
		// Override user's role with org-specific role for response
		if orgRole, ok := orgRoleMap[user.ID]; ok {
			user.Role = orgRole
			user.RoleID = &orgRole.ID
		}
		resp := userToResponse(user)
		resp.IsMember = homeOrgMap[user.ID] != orgID
		response[i] = resp
	}

	return r.SendEnvelope(map[string]interface{}{
		"users": response,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

// GetUser returns a single user
func (a *App) GetUser(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceUsers, models.ActionRead); err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "user")
	if err != nil {
		return nil
	}

	// Query via user_organizations to find both native and cross-org members.
	// Select("users.*") avoids column conflict with user_organizations.organization_id.
	var user models.User
	if err := a.DB.
		Select("users.*").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", id).
		First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	// Load org-specific role from user_organizations
	var userOrg models.UserOrganization
	if err := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Preload("Role").First(&userOrg).Error; err == nil && userOrg.RoleID != nil {
		user.RoleID = userOrg.RoleID
		user.Role = userOrg.Role
	} else {
		a.DB.Preload("Role").First(&user, user.ID)
	}

	resp := userToResponse(user)
	resp.IsMember = user.OrganizationID != orgID
	return r.SendEnvelope(resp)
}

// CreateUser creates a new user (admin only)
func (a *App) CreateUser(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceUsers, models.ActionWrite); err != nil {
		return nil
	}

	var req UserRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Email, password, and full_name are required", nil, "")
	}
	if err := validatePasswordStrength(req.Password); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "password")
	}
	if !a.checkQuotaOrRespond(r, license.ResourceUsers, orgID) {
		return nil
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
		// No role specified, use default role
		var defaultRole models.CustomRole
		if err := a.DB.Where("organization_id = ? AND is_default = ?", orgID, true).First(&defaultRole).Error; err == nil {
			roleID = &defaultRole.ID
		}
	}

	// Check if email already exists (including soft-deleted users)
	var existingUser models.User
	if err := a.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Email already exists", nil, "")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		a.Log.Error("Failed to hash password", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create user", nil, "")
	}

	isSuperAdmin := false
	if saField := parseSuperAdminField(r); saField != nil && *saField {
		if !a.IsSuperAdmin(userID) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only super admins can create super admins", nil, "")
		}
		isSuperAdmin = true
	}

	// Check for soft-deleted user with same email and restore them
	var softDeleted models.User
	if err := a.DB.Unscoped().Where("email = ? AND deleted_at IS NOT NULL", req.Email).First(&softDeleted).Error; err == nil {
		// Restore the soft-deleted user with new details
		if err := a.DB.Unscoped().Model(&softDeleted).Updates(map[string]interface{}{
			"deleted_at":      nil,
			"organization_id": orgID,
			"password_hash":   string(hashedPassword),
			"full_name":       req.FullName,
			"role_id":         roleID,
			"is_active":       true,
			"is_super_admin":  isSuperAdmin,
		}).Error; err != nil {
			a.Log.Error("Failed to restore user", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create user", nil, "")
		}

		// Restore or create UserOrganization entry
		var existingOrg models.UserOrganization
		if err := a.DB.Unscoped().Where("user_id = ? AND organization_id = ?", softDeleted.ID, orgID).First(&existingOrg).Error; err == nil {
			a.DB.Unscoped().Model(&existingOrg).Updates(map[string]interface{}{
				"deleted_at": nil,
				"role_id":    roleID,
				"is_default": true,
			})
		} else {
			a.DB.Create(&models.UserOrganization{
				UserID:         softDeleted.ID,
				OrganizationID: orgID,
				RoleID:         roleID,
				IsDefault:      true,
			})
		}

		// Load role for response
		if roleID != nil {
			var role models.CustomRole
			if err := a.DB.Where("id = ?", *roleID).First(&role).Error; err == nil {
				softDeleted.Role = &role
			}
		}
		softDeleted.OrganizationID = orgID
		softDeleted.PasswordHash = string(hashedPassword)
		softDeleted.FullName = req.FullName
		softDeleted.RoleID = roleID
		softDeleted.IsActive = true
		softDeleted.IsSuperAdmin = isSuperAdmin

		return r.SendEnvelope(userToResponse(softDeleted))
	}

	user := models.User{
		OrganizationID: orgID,
		Email:          req.Email,
		PasswordHash:   string(hashedPassword),
		FullName:       req.FullName,
		RoleID:         roleID,
		IsActive:       true,
		IsSuperAdmin:   isSuperAdmin,
	}

	if err := a.DB.Create(&user).Error; err != nil {
		a.Log.Error("Failed to create user", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create user", nil, "")
	}

	// Create UserOrganization entry
	userOrg := models.UserOrganization{
		UserID:         user.ID,
		OrganizationID: orgID,
		RoleID:         roleID,
		IsDefault:      true,
	}
	if err := a.DB.Create(&userOrg).Error; err != nil {
		a.Log.Error("Failed to create user organization entry", "error", err)
		// Non-fatal: user was already created
	}

	// Load role for response
	a.DB.Preload("Role").First(&user, user.ID)

	return r.SendEnvelope(userToResponse(user))
}

// UpdateUser updates a user
func (a *App) UpdateUser(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	currentUserID, _ := r.RequestCtx.UserValue("user_id").(uuid.UUID)

	id, err := parsePathUUID(r, "id", "user")
	if err != nil {
		return nil
	}

	// Users can update themselves, others need users:write permission
	if currentUserID != id && !a.HasPermission(currentUserID, models.ResourceUsers, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	// Find user via user_organizations (supports cross-org members).
	// Select("users.*") avoids column conflict with user_organizations.organization_id.
	var user models.User
	if err := a.DB.
		Select("users.*").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", id).
		Preload("Role").
		First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	isMember := user.OrganizationID != orgID

	// Load org-specific role for members
	if isMember {
		var userOrg models.UserOrganization
		if err := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Preload("Role").First(&userOrg).Error; err == nil && userOrg.RoleID != nil {
			user.RoleID = userOrg.RoleID
			user.Role = userOrg.Role
		}
	}

	var req UserRequest
	if err := r.Decode(&req, "json"); err != nil {
		a.Log.Error("UpdateUser: Failed to decode request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Only users with users:write permission can change roles
	if req.RoleID != nil && !a.HasPermission(currentUserID, models.ResourceUsers, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions to change roles", nil, "")
	}

	// For cross-org members, only allow role updates
	if isMember {
		if req.RoleID == nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Only role can be updated for organization members", nil, "")
		}
		// Validate role exists and belongs to org
		var newRole models.CustomRole
		if err := a.DB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&newRole).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
		}
		// Update role in user_organizations only
		if err := a.DB.Model(&models.UserOrganization{}).
			Where("user_id = ? AND organization_id = ?", id, orgID).
			Update("role_id", req.RoleID).Error; err != nil {
			a.Log.Error("Failed to update member role", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update member role", nil, "")
		}
		a.InvalidateUserPermissionsCache(user.ID)

		// Return updated response
		user.RoleID = req.RoleID
		user.Role = &newRole
		resp := userToResponse(user)
		resp.IsMember = true
		return r.SendEnvelope(resp)
	}

	// Native user: full update
	if req.Email != "" {
		var existingUser models.User
		if err := a.DB.Where("email = ? AND id != ?", req.Email, id).First(&existingUser).Error; err == nil {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "Email already exists", nil, "")
		}
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Password != "" {
		if currentUserID == id {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Use /me/password to change your password", nil, "")
		}
		if err := validatePasswordStrength(req.Password); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "password")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			a.Log.Error("Failed to hash password", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update user", nil, "")
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Handle role update
	roleChanged := false
	if req.RoleID != nil {
		// Validate role exists and belongs to org
		var newRole models.CustomRole
		if err := a.DB.Where("id = ? AND organization_id = ?", req.RoleID, orgID).First(&newRole).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid role", nil, "")
		}
		// Prevent self-demotion from admin
		if currentUserID == id && user.Role != nil && user.Role.Name == "admin" && newRole.Name != "admin" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot demote yourself", nil, "")
		}
		if user.RoleID == nil || *user.RoleID != *req.RoleID {
			roleChanged = true
		}
		user.RoleID = req.RoleID
		user.Role = nil // Clear the preloaded role to prevent GORM from using the old association
	}

	if req.IsActive != nil {
		// Prevent user from deactivating themselves
		if currentUserID == id && !*req.IsActive {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot deactivate yourself", nil, "")
		}
		user.IsActive = *req.IsActive
	}

	// Handle super admin update - only superadmins can change this
	if saField := parseSuperAdminField(r); saField != nil {
		if !a.IsSuperAdmin(currentUserID) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only super admins can modify super admin status", nil, "")
		}
		// Prevent removing own super admin status
		if currentUserID == id && !*saField && user.IsSuperAdmin {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot remove your own super admin status", nil, "")
		}
		user.IsSuperAdmin = *saField
	}

	if err := a.DB.Save(&user).Error; err != nil {
		a.Log.Error("Failed to update user", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update user", nil, "")
	}

	// Invalidate permissions cache if role changed
	if roleChanged {
		// Sync role change to UserOrganization for this org
		a.DB.Model(&models.UserOrganization{}).
			Where("user_id = ? AND organization_id = ?", user.ID, orgID).
			Update("role_id", user.RoleID)
		a.InvalidateUserPermissionsCache(user.ID)
	}

	// Load role for response
	a.DB.Preload("Role").First(&user, user.ID)

	return r.SendEnvelope(userToResponse(user))
}

// DeleteUser deletes a user or removes a member from the organization
func (a *App) DeleteUser(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	currentUserID, _ := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !a.HasPermission(currentUserID, models.ResourceUsers, models.ActionDelete, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	id, err := parsePathUUID(r, "id", "user")
	if err != nil {
		return nil
	}

	// Prevent user from deleting/removing themselves
	if currentUserID == id {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete yourself", nil, "")
	}

	// Find user via user_organizations (supports cross-org members).
	// Select("users.*") avoids column conflict with user_organizations.organization_id.
	var user models.User
	if err := a.DB.
		Select("users.*").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", id).
		Preload("Role").
		First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	isMember := user.OrganizationID != orgID

	if isMember {
		// Cross-org member: only remove from this organization
		result := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Delete(&models.UserOrganization{})
		if result.Error != nil {
			a.Log.Error("Failed to remove member", "error", result.Error)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove member", nil, "")
		}
		a.InvalidateUserPermissionsCache(id)
		return r.SendEnvelope(map[string]string{"message": "Member removed from organization"})
	}

	// Native user: check last admin constraint, then delete user account

	// Load org-specific role for admin check
	var userOrg models.UserOrganization
	if err := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Preload("Role").First(&userOrg).Error; err == nil && userOrg.Role != nil && userOrg.Role.Name == "admin" {
		var adminRole models.CustomRole
		if err := a.DB.Where("organization_id = ? AND name = ? AND is_system = ?", orgID, "admin", true).First(&adminRole).Error; err == nil {
			var adminCount int64
			a.DB.Model(&models.UserOrganization{}).
				Where("organization_id = ? AND role_id = ? AND deleted_at IS NULL", orgID, adminRole.ID).
				Count(&adminCount)
			if adminCount <= 1 {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot delete the last admin", nil, "")
			}
		}
	}

	result := a.DB.Where("id = ?", id).Delete(&models.User{})
	if result.Error != nil {
		a.Log.Error("Failed to delete user", "error", result.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete user", nil, "")
	}
	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	// Delete all UserOrganization entries for this user
	a.DB.Where("user_id = ?", id).Delete(&models.UserOrganization{})

	return r.SendEnvelope(map[string]string{"message": "User deleted successfully"})
}

// GetCurrentUser returns the current authenticated user's details
func (a *App) GetCurrentUser(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).
		Preload("Role").
		First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	// Use org from JWT context (may differ from DB after org switch)
	orgID, _ := r.RequestCtx.UserValue("organization_id").(uuid.UUID)
	if orgID != uuid.Nil {
		user.OrganizationID = orgID

		// Check for org-specific role from user_organizations
		var userOrg models.UserOrganization
		if err := a.DB.Where("user_id = ? AND organization_id = ?", userID, orgID).First(&userOrg).Error; err == nil && userOrg.RoleID != nil {
			user.RoleID = userOrg.RoleID
			var role models.CustomRole
			if err := a.DB.Where("id = ?", *userOrg.RoleID).First(&role).Error; err == nil {
				user.Role = &role
			}
		}
	}

	// Load permissions from cache
	if user.Role != nil && user.RoleID != nil {
		cachedPerms, err := a.GetRolePermissionsCached(*user.RoleID)
		if err == nil {
			// Convert cached permission strings back to Permission objects
			permissions := make([]models.Permission, 0, len(cachedPerms))
			for _, p := range cachedPerms {
				parts := splitPermission(p)
				if len(parts) == 2 {
					permissions = append(permissions, models.Permission{
						Resource: parts[0],
						Action:   parts[1],
					})
				}
			}
			user.Role.Permissions = permissions
		}
	}

	return r.SendEnvelope(userToResponse(user))
}

// splitPermission splits a "resource:action" string
func splitPermission(p string) []string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == ':' {
			return []string{p[:i], p[i+1:]}
		}
	}
	return nil
}

// UpdateCurrentUserSettings updates the current user's notification/preferences settings
func (a *App) UpdateCurrentUserSettings(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	var req UserSettingsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Initialize settings if nil
	if user.Settings == nil {
		user.Settings = make(models.JSONB)
	}

	if req.EmailNotifications != nil {
		user.Settings["email_notifications"] = *req.EmailNotifications
	}
	if req.NewMessageAlerts != nil {
		user.Settings["new_message_alerts"] = *req.NewMessageAlerts
	}
	if req.CampaignUpdates != nil {
		user.Settings["campaign_updates"] = *req.CampaignUpdates
	}
	if req.NotificationSound != nil {
		user.Settings["notification_sound"] = normalizeNotificationSound(*req.NotificationSound)
	}
	if req.ThemeMode != nil {
		user.Settings["theme_mode"] = normalizeThemeMode(*req.ThemeMode)
	}
	if req.ThemePreset != nil {
		user.Settings["theme_preset"] = normalizeThemePreset(*req.ThemePreset)
	}

	if req.ChatBackground.Set {
		nextChatBackground, cleanupPath, err := a.resolveRequestedChatBackground(&user, req.ChatBackground.Value)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "chat_background")
		}
		if nextChatBackground != nil {
			user.Settings["chat_background"] = nextChatBackground.toSettingsValue()
		} else {
			delete(user.Settings, "chat_background")
		}

		if err := a.DB.Save(&user).Error; err != nil {
			a.Log.Error("Failed to update user settings", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
		}

		if cleanupPath != "" {
			if err := a.deleteStoredMediaFile(cleanupPath); err != nil {
				a.Log.Warn("Failed to delete previous chat background asset after preset switch", "path", cleanupPath, "error", err)
			}
		}
	} else {
		if err := a.DB.Save(&user).Error; err != nil {
			a.Log.Error("Failed to update user settings", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
		}
	}

	return r.SendEnvelope(map[string]interface{}{
		"message":  "Settings updated successfully",
		"settings": user.Settings,
	})
}

func (a *App) UploadCurrentUserChatBackground(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
	}

	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, "file")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to open file", nil, "file")
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxChatBackgroundUploadSize+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "file")
	}
	if len(data) > maxChatBackgroundUploadSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 5MB", nil, "file")
	}

	detectedMimeType := normalizeChatBackgroundMIME(http.DetectContentType(data))
	if detectedMimeType == "" {
		return r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			"Unsupported file type. Use JPG, PNG, or WebP",
			nil,
			"file",
		)
	}

	sanitizedFilename := sanitizeFilename(fileHeader.Filename)
	assetID := uuid.New().String()
	extension := getExtensionFromMimeType(detectedMimeType)
	if extension == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type. Use JPG, PNG, or WebP", nil, "file")
	}

	subdir := filepath.Join("chat-backgrounds", user.ID.String())
	if err := a.ensureMediaDir(subdir); err != nil {
		a.Log.Error("Failed to create chat background directory", "user_id", user.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save chat background", nil, "")
	}

	relativePath := filepath.Join(subdir, assetID+extension)
	fullPath := filepath.Join(a.getMediaStoragePath(), relativePath)
	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		a.Log.Error("Failed to write chat background file", "user_id", user.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save chat background", nil, "")
	}

	if user.Settings == nil {
		user.Settings = make(models.JSONB)
	}

	previousBackground := decodeStoredChatBackground(user.Settings["chat_background"])
	nextBackground := UserChatBackground{
		Kind:           chatBackgroundKindCustom,
		CustomAssetID:  assetID,
		CustomFilename: sanitizedFilename,
		CustomMimeType: detectedMimeType,
	}
	user.Settings["chat_background"] = nextBackground.toSettingsValue()

	if err := a.DB.Save(&user).Error; err != nil {
		_ = os.Remove(fullPath)
		a.Log.Error("Failed to persist chat background metadata", "user_id", user.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save chat background", nil, "")
	}

	previousPath := chatBackgroundRelativePath(user.ID, previousBackground)
	if previousPath != "" && previousPath != relativePath {
		if err := a.deleteStoredMediaFile(previousPath); err != nil {
			a.Log.Warn("Failed to delete previous chat background asset after replacement", "path", previousPath, "error", err)
		}
	}

	return r.SendEnvelope(map[string]any{
		"message":         "Chat background uploaded successfully",
		"chat_background": nextBackground,
	})
}

func (a *App) GetCurrentUserChatBackground(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	chatBackground := decodeStoredChatBackground(user.Settings["chat_background"])
	if chatBackground == nil || chatBackground.Kind != chatBackgroundKindCustom {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat background not found", nil, "")
	}

	relativePath := chatBackgroundRelativePath(user.ID, chatBackground)
	if relativePath == "" {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat background not found", nil, "")
	}

	if err := a.serveLocalMediaFile(r, relativePath, chatBackground.CustomMimeType); err != nil {
		return err
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "private")
	return nil
}

// ChangePassword changes the current user's password
func (a *App) ChangePassword(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	var req ChangePasswordRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Validate required fields
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Current password and new password are required", nil, "")
	}

	// Validate new password strength
	if err := validatePasswordStrength(req.NewPassword); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "new_password")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Current password is incorrect", nil, "")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		a.Log.Error("Failed to hash password", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to change password", nil, "")
	}

	user.PasswordHash = string(hashedPassword)
	if err := a.DB.Save(&user).Error; err != nil {
		a.Log.Error("Failed to update password", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to change password", nil, "")
	}

	return r.SendEnvelope(map[string]string{"message": "Password changed successfully"})
}

// Helper function to convert User to UserResponse
func userToResponse(user models.User) UserResponse {
	resp := UserResponse{
		ID:             user.ID,
		Email:          user.Email,
		FullName:       user.FullName,
		RoleID:         user.RoleID,
		IsActive:       user.IsActive,
		IsAvailable:    user.IsAvailable,
		IsSuperAdmin:   user.IsSuperAdmin,
		OrganizationID: user.OrganizationID,
		Settings:       user.Settings,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Include role info if loaded
	if user.Role != nil {
		roleInfo := &RoleInfo{
			ID:          user.Role.ID,
			Name:        user.Role.Name,
			Description: user.Role.Description,
			IsSystem:    user.Role.IsSystem,
			Permissions: []PermissionInfo{},
		}

		// Include permissions if loaded
		for _, p := range user.Role.Permissions {
			roleInfo.Permissions = append(roleInfo.Permissions, PermissionInfo{
				ID:          p.ID,
				Resource:    p.Resource,
				Action:      p.Action,
				Description: p.Description,
			})
		}

		resp.Role = roleInfo
	}

	return resp
}

// MyOrganizationResponse represents an organization in the user's org list
type MyOrganizationResponse struct {
	OrganizationID uuid.UUID  `json:"organization_id"`
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	RoleID         *uuid.UUID `json:"role_id,omitempty"`
	RoleName       string     `json:"role_name,omitempty"`
	IsDefault      bool       `json:"is_default"`
}

// ListMyOrganizations returns all organizations the current user belongs to
func (a *App) ListMyOrganizations(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var userOrgs []models.UserOrganization
	if err := a.DB.Where("user_id = ?", userID).
		Preload("Organization").
		Preload("Role").
		Find(&userOrgs).Error; err != nil {
		a.Log.Error("Failed to list user organizations", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list organizations", nil, "")
	}

	response := make([]MyOrganizationResponse, 0, len(userOrgs))
	for _, uo := range userOrgs {
		item := MyOrganizationResponse{
			OrganizationID: uo.OrganizationID,
			IsDefault:      uo.IsDefault,
			RoleID:         uo.RoleID,
		}
		if uo.Organization != nil {
			item.Name = uo.Organization.Name
			item.Slug = uo.Organization.Slug
		}
		if uo.Role != nil {
			item.RoleName = uo.Role.Name
		}
		response = append(response, item)
	}

	return r.SendEnvelope(map[string]interface{}{
		"organizations": response,
	})
}

// AvailabilityRequest represents the request body for updating availability
type AvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

// UpdateAvailability updates the current user's availability status (away/available)
func (a *App) UpdateAvailability(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	var req AvailabilityRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Only log if status is actually changing
	if user.IsAvailable != req.IsAvailable {
		now := time.Now()

		// End the previous availability log (if exists)
		a.DB.Model(&models.UserAvailabilityLog{}).
			Where("user_id = ? AND ended_at IS NULL", userID).
			Update("ended_at", now)

		// Create new availability log
		log := models.UserAvailabilityLog{
			UserID:         userID,
			OrganizationID: orgID,
			IsAvailable:    req.IsAvailable,
			StartedAt:      now,
		}
		if err := a.DB.Create(&log).Error; err != nil {
			a.Log.Error("Failed to create availability log", "error", err)
			// Continue anyway - logging failure shouldn't block availability update
		}
	}

	user.IsAvailable = req.IsAvailable

	if err := a.DB.Save(&user).Error; err != nil {
		a.Log.Error("Failed to update availability", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update availability", nil, "")
	}

	status := "available"
	transfersReturned := 0
	if !req.IsAvailable {
		status = "away"
		// Return agent's active transfers to queue when going away
		transfersReturned = a.ReturnAgentTransfersToQueue(userID, orgID)
	}

	// Get the current break start time if away
	var breakStartedAt *time.Time
	if !req.IsAvailable {
		var currentLog models.UserAvailabilityLog
		if err := a.DB.Where("user_id = ? AND is_available = false AND ended_at IS NULL", userID).
			Order("started_at DESC").First(&currentLog).Error; err == nil {
			breakStartedAt = &currentLog.StartedAt
		}
	}

	return r.SendEnvelope(map[string]interface{}{
		"message":            "Availability updated successfully",
		"is_available":       user.IsAvailable,
		"status":             status,
		"break_started_at":   breakStartedAt,
		"transfers_to_queue": transfersReturned,
	})
}
