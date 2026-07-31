package handlers

import (
	"errors"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
	// WhatsAppAccountIDs replaces the user's account assignments when present.
	// nil = leave assignments untouched; empty slice = clear all assignments
	// (restoring full org visibility).
	WhatsAppAccountIDs *[]uuid.UUID `json:"whatsapp_account_ids"`
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
	// WhatsAppAccountIDs lists the accounts assigned to the user in the org
	// (empty = no restriction, user sees all org accounts).
	WhatsAppAccountIDs []uuid.UUID `json:"whatsapp_account_ids"`
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
}

// errInvalidAccountAssignment is returned when an assignment payload contains
// account IDs that don't exist in the caller's organization.
var errInvalidAccountAssignment = errors.New("invalid account assignment")

// syncUserAccountAssignments replaces the user's account assignments within
// the org with the given set. Every ID must reference an account of the org.
// An empty set clears the org's assignments, restoring full org visibility.
// Assignments in other organizations are left untouched.
func (a *App) syncUserAccountAssignments(userID, orgID uuid.UUID, accountIDs []uuid.UUID) error {
	ids := dedupeUUIDs(accountIDs)
	if len(ids) > 0 {
		var count int64
		if err := a.DB.Model(&models.WhatsAppAccount{}).
			Where("id IN ? AND organization_id = ?", ids, orgID).
			Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return errInvalidAccountAssignment
		}
	}
	return a.DB.Transaction(func(tx *gorm.DB) error {
		// Replace only this org's assignments; the subquery keeps assignments
		// pointing at other orgs' accounts intact.
		if err := tx.
			Where("user_id = ? AND whats_app_account_id IN (?)",
				userID,
				tx.Model(&models.WhatsAppAccount{}).Select("id").Where("organization_id = ?", orgID),
			).
			Delete(&models.UserWhatsAppAccount{}).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := tx.Create(&models.UserWhatsAppAccount{UserID: userID, WhatsAppAccountID: id}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// sendAssignmentError maps assignment sync failures to the right envelope.
func (a *App) sendAssignmentError(r *fastglue.Request, err error) error {
	if errors.Is(err, errInvalidAccountAssignment) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "One or more account IDs are invalid", nil, "")
	}
	a.Log.Error("Failed to sync account assignments", "error", err)
	return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account assignments", nil, "")
}

// loadUserAccountIDs fetches the user's assigned account IDs for responses.
// Best-effort: lookup errors resolve to an empty list.
func (a *App) loadUserAccountIDs(userID, orgID uuid.UUID) []uuid.UUID {
	ids, err := a.assignedAccountIDs(userID, orgID)
	if err != nil || ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
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
	EmailNotifications bool `json:"email_notifications"`
	NewMessageAlerts   bool `json:"new_message_alerts"`
	CampaignUpdates    bool `json:"campaign_updates"`
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ListUsers returns all users for the organization
func (a *App) ListUsers(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceUsers, models.ActionRead)
	if err != nil {
		return nil
	}

	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))
	roleIDParam := string(r.RequestCtx.QueryArgs().Peek("role_id"))
	onlineOnly := string(r.RequestCtx.QueryArgs().Peek("online_only")) == "true"

	// Online user IDs for this org — fetched once and reused for both the
	// online_only filter and the online_count badge in the response.
	var onlineIDs []uuid.UUID
	if a.WSHub != nil {
		onlineIDs = a.WSHub.OnlineUserIDs(orgID)
	}

	// Query users via user_organizations to include cross-org members.
	joinClause := "JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL"

	countQuery := a.DB.Joins(joinClause, orgID).Where("users.deleted_at IS NULL")
	dataQuery := a.DB.Joins(joinClause, orgID).Where("users.deleted_at IS NULL")
	if search != "" {
		countQuery = countQuery.Where("users.full_name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
		dataQuery = dataQuery.Where("users.full_name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if roleIDParam != "" {
		if roleUUID, err := uuid.Parse(roleIDParam); err == nil {
			countQuery = countQuery.Where("user_organizations.role_id = ?", roleUUID)
			dataQuery = dataQuery.Where("user_organizations.role_id = ?", roleUUID)
		}
	}
	if onlineOnly {
		if len(onlineIDs) == 0 {
			return r.SendEnvelope(map[string]any{
				"users":        []UserResponse{},
				"total":        0,
				"page":         pg.Page,
				"limit":        pg.Limit,
				"online_count": 0,
			})
		}
		countQuery = countQuery.Where("users.id IN ?", onlineIDs)
		dataQuery = dataQuery.Where("users.id IN ?", onlineIDs)
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

	// Build home org map from already-fetched users (no extra query needed)
	homeOrgMap := make(map[uuid.UUID]uuid.UUID, len(users))
	for _, u := range users {
		homeOrgMap[u.ID] = u.OrganizationID
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

	return r.SendEnvelope(map[string]any{
		"users":        response,
		"total":        total,
		"page":         pg.Page,
		"limit":        pg.Limit,
		"online_count": len(onlineIDs),
	})
}

// findOrgUserWithRole loads a user scoped to the organization via the
// user_organizations join (supporting cross-org members), preloading the Role.
// On failure it sends a 404 envelope and returns errEnvelopeSent. The returned
// bool reports whether the user is a cross-org member (their home org differs).
func (a *App) findOrgUserWithRole(r *fastglue.Request, orgID, id uuid.UUID) (models.User, bool, error) {
	var user models.User
	if err := a.DB.
		Select("users.*").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", id).
		Preload("Role").
		First(&user).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
		return models.User{}, false, errEnvelopeSent
	}
	return user, user.OrganizationID != orgID, nil
}

// GetUser returns a single user
func (a *App) GetUser(r *fastglue.Request) error {
	orgID, err := a.requireOrgID(r)
	if err != nil {
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
	resp.WhatsAppAccountIDs = a.loadUserAccountIDs(user.ID, orgID)
	return r.SendEnvelope(resp)
}

// CreateUser creates a new user (admin only)
func (a *App) CreateUser(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceUsers, models.ActionWrite)
	if err != nil {
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

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid email format", nil, "")
	}

	// Assigning account access requires the dedicated accounts:assign permission.
	if req.WhatsAppAccountIDs != nil && !a.HasPermission(userID, models.ResourceAccounts, models.ActionAssign, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions to change account assignments", nil, "")
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
		if err := a.DB.Unscoped().Model(&softDeleted).Updates(map[string]any{
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
			a.DB.Unscoped().Model(&existingOrg).Updates(map[string]any{
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

		// Apply account assignments (replace-all semantics within this org)
		if req.WhatsAppAccountIDs != nil {
			if err := a.syncUserAccountAssignments(softDeleted.ID, orgID, *req.WhatsAppAccountIDs); err != nil {
				return a.sendAssignmentError(r, err)
			}
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

		a.logAudit(orgID, userID,
			"user", softDeleted.ID, models.AuditActionCreated, nil, userAuditSnapshot(&softDeleted))

		resp := userToResponse(softDeleted)
		resp.WhatsAppAccountIDs = a.loadUserAccountIDs(softDeleted.ID, orgID)
		return r.SendEnvelope(resp)
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

	// Apply account assignments (replace-all semantics within this org)
	if req.WhatsAppAccountIDs != nil {
		if err := a.syncUserAccountAssignments(user.ID, orgID, *req.WhatsAppAccountIDs); err != nil {
			return a.sendAssignmentError(r, err)
		}
	}

	// Load role for response
	a.DB.Preload("Role").First(&user, user.ID)

	a.logAudit(orgID, userID,
		"user", user.ID, models.AuditActionCreated, nil, userAuditSnapshot(&user))

	resp := userToResponse(user)
	resp.WhatsAppAccountIDs = a.loadUserAccountIDs(user.ID, orgID)
	return r.SendEnvelope(resp)
}

// UpdateUser updates a user
func (a *App) UpdateUser(r *fastglue.Request) error {
	orgID, err := a.requireOrgID(r)
	if err != nil {
		return nil
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

	user, isMember, err := a.findOrgUserWithRole(r, orgID, id)
	if err != nil {
		return nil
	}

	// Load org-specific role for members
	if isMember {
		var userOrg models.UserOrganization
		if err := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Preload("Role").First(&userOrg).Error; err == nil && userOrg.RoleID != nil {
			user.RoleID = userOrg.RoleID
			user.Role = userOrg.Role
		}
	}

	// Snapshot before changes for audit log diff.
	oldSnap := userAuditSnapshot(&user)

	var req UserRequest
	if err := r.Decode(&req, "json"); err != nil {
		a.Log.Error("UpdateUser: Failed to decode request", "error", err, "body", string(r.RequestCtx.PostBody()))
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Only users with users:write permission can change roles
	if req.RoleID != nil && !a.HasPermission(currentUserID, models.ResourceUsers, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions to change roles", nil, "")
	}

	// Account assignments are governed by the dedicated accounts:assign
	// permission — without this gate a user could lift their own visibility
	// restrictions via self-update.
	if req.WhatsAppAccountIDs != nil && !a.HasPermission(currentUserID, models.ResourceAccounts, models.ActionAssign, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions to change account assignments", nil, "")
	}

	// For cross-org members, only role and account assignments can be updated
	if isMember {
		if req.RoleID == nil && req.WhatsAppAccountIDs == nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Only role and account assignments can be updated for organization members", nil, "")
		}
		if req.WhatsAppAccountIDs != nil {
			if err := a.syncUserAccountAssignments(user.ID, orgID, *req.WhatsAppAccountIDs); err != nil {
				return a.sendAssignmentError(r, err)
			}
		}
		if req.RoleID != nil {
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
		}

		a.logAudit(orgID, currentUserID,
			"user", user.ID, models.AuditActionUpdated, oldSnap, userAuditSnapshot(&user))

		resp := userToResponse(user)
		resp.IsMember = true
		resp.WhatsAppAccountIDs = a.loadUserAccountIDs(user.ID, orgID)
		return r.SendEnvelope(resp)
	}

	// Native user: full update
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid email format", nil, "")
		}
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

	// Replace account assignments if provided (nil leaves them untouched)
	if req.WhatsAppAccountIDs != nil {
		if err := a.syncUserAccountAssignments(user.ID, orgID, *req.WhatsAppAccountIDs); err != nil {
			return a.sendAssignmentError(r, err)
		}
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

	a.logAudit(orgID, currentUserID,
		"user", user.ID, models.AuditActionUpdated, oldSnap, userAuditSnapshot(&user))

	resp := userToResponse(user)
	resp.WhatsAppAccountIDs = a.loadUserAccountIDs(user.ID, orgID)
	return r.SendEnvelope(resp)
}

// DeleteUser deletes a user or removes a member from the organization
func (a *App) DeleteUser(r *fastglue.Request) error {
	orgID, err := a.requireOrgID(r)
	if err != nil {
		return nil
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

	user, isMember, err := a.findOrgUserWithRole(r, orgID, id)
	if err != nil {
		return nil
	}

	if isMember {
		// Cross-org member: only remove from this organization
		result := a.DB.Where("user_id = ? AND organization_id = ?", id, orgID).Delete(&models.UserOrganization{})
		if result.Error != nil {
			a.Log.Error("Failed to remove member", "error", result.Error)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove member", nil, "")
		}
		a.InvalidateUserPermissionsCache(id)

		// Drop this org's account assignments so a future re-add starts clean
		a.DB.Where("user_id = ? AND whats_app_account_id IN (?)",
			id,
			a.DB.Model(&models.WhatsAppAccount{}).Select("id").Where("organization_id = ?", orgID),
		).Delete(&models.UserWhatsAppAccount{})

		a.logAudit(orgID, currentUserID,
			"user", id, models.AuditActionDeleted, userAuditSnapshot(&user), nil)

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

	// Delete all account assignments for this user
	a.DB.Where("user_id = ?", id).Delete(&models.UserWhatsAppAccount{})

	a.logAudit(orgID, currentUserID,
		"user", id, models.AuditActionDeleted, userAuditSnapshot(&user), nil)

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
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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

	oldNotif := notificationSettingsSnapshot(user.Settings)

	// Update notification settings
	user.Settings["email_notifications"] = req.EmailNotifications
	user.Settings["new_message_alerts"] = req.NewMessageAlerts
	user.Settings["campaign_updates"] = req.CampaignUpdates

	if err := a.DB.Save(&user).Error; err != nil {
		a.Log.Error("Failed to update user settings", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	newNotif := notificationSettingsSnapshot(user.Settings)
	a.logAudit(orgID, userID,
		models.ResourceSettingsNotification, userID, models.AuditActionUpdated, oldNotif, newNotif)

	return r.SendEnvelope(map[string]any{
		"message":  "Settings updated successfully",
		"settings": user.Settings,
	})
}

// notificationSettingsSnapshot extracts notification-preference fields from
// a user's JSONB settings into a map suitable for audit diffing.
func notificationSettingsSnapshot(settings models.JSONB) map[string]any {
	return map[string]any{
		"email_notifications": settings["email_notifications"],
		"new_message_alerts":  settings["new_message_alerts"],
		"campaign_updates":    settings["campaign_updates"],
	}
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

	// Validate new password length
	if len(req.NewPassword) < 6 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "New password must be at least 6 characters", nil, "")
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
// userAuditSnapshot returns a minimal, diff-friendly representation of a user
// for audit logging. Sensitive fields (password hash) and noisy fields (settings,
// availability, timestamps) are intentionally excluded.
func userAuditSnapshot(user *models.User) map[string]any {
	if user == nil {
		return nil
	}
	snap := map[string]any{
		"full_name":      user.FullName,
		"email":          user.Email,
		"is_active":      user.IsActive,
		"is_super_admin": user.IsSuperAdmin,
	}
	if user.Role != nil {
		snap["role"] = user.Role.Name
	} else {
		snap["role"] = ""
	}
	return snap
}

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

	return r.SendEnvelope(map[string]any{
		"organizations": response,
	})
}

// AvailabilityRequest represents the request body for updating availability
type AvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

// UpdateAvailability updates the current user's availability status (away/available)
func (a *App) UpdateAvailability(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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
	if !req.IsAvailable {
		status = "away"
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

	return r.SendEnvelope(map[string]any{
		"message":          "Availability updated successfully",
		"is_available":     user.IsAvailable,
		"status":           status,
		"break_started_at": breakStartedAt,
	})
}
