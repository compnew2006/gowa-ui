package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/internal/websocket"
	"gorm.io/gorm"
)

const (
	// Cache TTLs - 6 hours since these rarely change (invalidated on update anyway)
	whatsappAccountCacheTTL = 6 * time.Hour
	webhooksCacheTTL        = 6 * time.Hour
	userPermissionsCacheTTL = 6 * time.Hour
	rolePermissionsCacheTTL = 6 * time.Hour
	tagsCacheTTL            = 6 * time.Hour

	// Cache key prefixes
	whatsappAccountCachePrefix = "whatsapp:account:"
	webhooksCachePrefix        = "webhooks:"
	userPermissionsCachePrefix = "permissions:user:"
	rolePermissionsCachePrefix = "permissions:role:"
	tagsCachePrefix            = "tags:"
)

// deleteKeysByPattern deletes all keys matching a pattern
func (a *App) deleteKeysByPattern(ctx context.Context, pattern string) {
	iter := a.Redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		a.Redis.Del(ctx, iter.Val())
	}
}

// getWhatsAppAccountCached retrieves WhatsApp account by device ID from cache or database
func (a *App) getWhatsAppAccountCached(phoneID string) (*models.WhatsAppAccount, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", whatsappAccountCachePrefix, phoneID)

	// Try cache first
	cached, err := a.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var account models.WhatsAppAccount
		if err := json.Unmarshal([]byte(cached), &account); err == nil {
			a.decryptAccountSecrets(&account)
			return &account, nil
		}
	}

	// Cache miss - fetch from database.
	// Match by gowa_device_id / gowa_jid (where phoneID is the device JID) or
	// the phone number portion (where phoneID might be just the digits).
	var account models.WhatsAppAccount
	phoneDigits := phoneID
	if idx := strings.Index(phoneID, "@"); idx > 0 {
		phoneDigits = phoneID[:idx]
	}
	if err := a.DB.Where(
		"gowa_device_id = ? OR gowa_device_id = ? OR gowa_jid = ? OR gowa_jid = ?",
		phoneID, phoneDigits, phoneID, phoneDigits,
	).First(&account).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(account); err == nil {
		a.Redis.Set(ctx, cacheKey, data, whatsappAccountCacheTTL)
	}

	// Decrypt secrets before returning
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// decryptAccountSecrets decrypts the encrypted secrets on a WhatsApp account.
// Handles both encrypted ("enc:" prefixed) and legacy unencrypted values transparently.
func (a *App) decryptAccountSecrets(account *models.WhatsAppAccount) {
	account.DecryptSecrets(a.Config.App.EncryptionKey)
}

// InvalidateWhatsAppAccountCache invalidates the WhatsApp account cache
func (a *App) InvalidateWhatsAppAccountCache(phoneID string) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", whatsappAccountCachePrefix, phoneID)
	a.Redis.Del(ctx, cacheKey)
}

// getWebhooksCached retrieves active webhooks for an organization from cache or database
func (a *App) getWebhooksCached(orgID uuid.UUID) ([]models.Webhook, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", webhooksCachePrefix, orgID.String())

	// Try cache first
	cached, err := a.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var webhooks []models.Webhook
		if err := json.Unmarshal([]byte(cached), &webhooks); err == nil {
			return webhooks, nil
		}
	}

	// Cache miss - fetch from database
	var webhooks []models.Webhook
	if err := a.DB.Where("organization_id = ? AND is_active = ?", orgID, true).Find(&webhooks).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(webhooks); err == nil {
		a.Redis.Set(ctx, cacheKey, data, webhooksCacheTTL)
	}

	return webhooks, nil
}

// InvalidateWebhooksCache invalidates the webhooks cache for an organization
func (a *App) InvalidateWebhooksCache(orgID uuid.UUID) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", webhooksCachePrefix, orgID.String())
	a.Redis.Del(ctx, cacheKey)
}

// UserPermissions represents cached user permissions
type UserPermissions struct {
	RoleID       uuid.UUID `json:"role_id"`
	RoleName     string    `json:"role_name"`
	IsSystem     bool      `json:"is_system"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	Permissions  []string  `json:"permissions"` // Format: "resource:action"
}

// getUserPermissionsCached retrieves user permissions from cache or database.
// When orgID is provided, it looks up the user's role from user_organizations for that org.
// When orgID is not provided, it falls back to the user's default RoleID.
func (a *App) getUserPermissionsCached(userID uuid.UUID, orgIDs ...uuid.UUID) (*UserPermissions, error) {
	ctx := context.Background()

	// Determine cache key based on whether orgID is provided
	var cacheKey string
	var orgID uuid.UUID
	if len(orgIDs) > 0 && orgIDs[0] != uuid.Nil {
		orgID = orgIDs[0]
		cacheKey = fmt.Sprintf("%s%s:%s", userPermissionsCachePrefix, userID.String(), orgID.String())
	} else {
		cacheKey = fmt.Sprintf("%s%s", userPermissionsCachePrefix, userID.String())
	}

	// Try cache first (if Redis is available)
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var perms UserPermissions
			if err := json.Unmarshal([]byte(cached), &perms); err == nil {
				return &perms, nil
			}
		}
	}

	// Cache miss - fetch from database
	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Determine which role to use
	var roleID *uuid.UUID
	if orgID != uuid.Nil {
		// Look up role from user_organizations for this specific org
		var userOrg models.UserOrganization
		if err := a.DB.Where("user_id = ? AND organization_id = ?", userID, orgID).First(&userOrg).Error; err == nil && userOrg.RoleID != nil {
			roleID = userOrg.RoleID
		} else {
			// Fall back to user's default role
			roleID = user.RoleID
		}
	} else {
		roleID = user.RoleID
	}

	if roleID == nil {
		return nil, gorm.ErrRecordNotFound
	}

	// Fetch role and load permissions via JOIN
	var role models.CustomRole
	if err := a.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
		return nil, err
	}
	if err := a.loadRolePermissions(&role); err != nil {
		return nil, err
	}

	// Build permissions list
	perms := UserPermissions{
		RoleID:       role.ID,
		RoleName:     role.Name,
		IsSystem:     role.IsSystem,
		IsSuperAdmin: user.IsSuperAdmin,
		Permissions:  make([]string, 0, len(role.Permissions)),
	}

	for _, p := range role.Permissions {
		perms.Permissions = append(perms.Permissions, p.Resource+":"+p.Action)
	}

	// Cache the result (if Redis is available)
	if a.Redis != nil {
		if data, err := json.Marshal(perms); err == nil {
			a.Redis.Set(ctx, cacheKey, data, userPermissionsCacheTTL)
		}
	}

	return &perms, nil
}

// HasPermission checks if a user has a specific permission.
// Super admins have all permissions automatically.
// Optional orgIDs parameter allows checking permissions for a specific org.
func (a *App) HasPermission(userID uuid.UUID, resource, action string, orgIDs ...uuid.UUID) bool {
	perms, err := a.getUserPermissionsCached(userID, orgIDs...)
	if err != nil {
		a.Log.Error("Failed to get user permissions", "error", err, "user_id", userID)
		return false
	}

	// Super admins have all permissions
	if perms.IsSuperAdmin {
		return true
	}

	permKey := resource + ":" + action
	for _, p := range perms.Permissions {
		if p == permKey {
			return true
		}
	}

	return false
}

// IsSuperAdmin checks if a user is a super admin
func (a *App) IsSuperAdmin(userID uuid.UUID) bool {
	perms, err := a.getUserPermissionsCached(userID)
	if err != nil {
		return false
	}
	return perms.IsSuperAdmin
}

// ScopeToOrg adds organization scoping to an existing query
// Always filters by organization - uuid.Nil is not allowed
func (a *App) ScopeToOrg(query *gorm.DB, userID, orgID uuid.UUID) *gorm.DB {
	return query.Where("organization_id = ?", orgID)
}

// GetRolePermissionsCached retrieves role permissions from cache or database
func (a *App) GetRolePermissionsCached(roleID uuid.UUID) ([]string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", rolePermissionsCachePrefix, roleID.String())

	// Try cache first
	cached, err := a.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var perms []string
		if err := json.Unmarshal([]byte(cached), &perms); err == nil {
			return perms, nil
		}
	}

	// Cache miss - fetch from database via JOIN
	var role models.CustomRole
	if err := a.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
		return nil, err
	}
	if err := a.loadRolePermissions(&role); err != nil {
		return nil, err
	}

	// Build permissions list
	perms := make([]string, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		perms = append(perms, p.Resource+":"+p.Action)
	}

	// Cache the result
	if data, err := json.Marshal(perms); err == nil {
		a.Redis.Set(ctx, cacheKey, data, rolePermissionsCacheTTL)
	}

	return perms, nil
}

// InvalidateUserPermissionsCache invalidates the permissions cache for a user
func (a *App) InvalidateUserPermissionsCache(userID uuid.UUID) {
	ctx := context.Background()
	// Delete the base key (no org suffix)
	cacheKey := fmt.Sprintf("%s%s", userPermissionsCachePrefix, userID.String())
	a.Redis.Del(ctx, cacheKey)
	// Delete all org-specific keys
	pattern := fmt.Sprintf("%s%s:*", userPermissionsCachePrefix, userID.String())
	a.deleteKeysByPattern(ctx, pattern)

	// Notify user via WebSocket to refresh their permissions
	a.notifyUserPermissionsChanged(userID)
}

// InvalidateRolePermissionsCache invalidates the permissions cache for a role and all users with that role
func (a *App) InvalidateRolePermissionsCache(roleID uuid.UUID) {
	ctx := context.Background()

	// Delete role cache
	roleCacheKey := fmt.Sprintf("%s%s", rolePermissionsCachePrefix, roleID.String())
	a.Redis.Del(ctx, roleCacheKey)

	// Collect all user IDs that have this role (deduplicated)
	userIDs := make(map[uuid.UUID]bool)

	// Users with this role as their default role
	var users []models.User
	if err := a.DB.Select("id").Where("role_id = ?", roleID).Find(&users).Error; err != nil {
		a.Log.Error("Failed to find users for role permission cache invalidation", "error", err, "role_id", roleID)
	}
	for _, u := range users {
		userIDs[u.ID] = true
	}

	// Users with this role via org-specific assignment (user_organizations)
	var orgUserIDs []uuid.UUID
	if err := a.DB.Table("user_organizations").
		Select("user_id").
		Where("role_id = ? AND deleted_at IS NULL", roleID).
		Pluck("user_id", &orgUserIDs).Error; err != nil {
		a.Log.Error("Failed to find org users for role permission cache invalidation", "error", err, "role_id", roleID)
	}
	for _, uid := range orgUserIDs {
		userIDs[uid] = true
	}

	// Invalidate cache for all affected users (both base and org-specific keys)
	for uid := range userIDs {
		a.InvalidateUserPermissionsCache(uid)
	}
}

// notifyUserPermissionsChanged sends a WebSocket message to a user to refresh their permissions
func (a *App) notifyUserPermissionsChanged(userID uuid.UUID) {
	if a.WSHub == nil {
		return
	}

	// Get user's organization ID for the broadcast
	var user models.User
	if err := a.DB.Select("organization_id").Where("id = ?", userID).First(&user).Error; err != nil {
		a.Log.Error("Failed to find user for permissions notification", "error", err, "user_id", userID)
		return
	}

	a.WSHub.BroadcastToUser(user.OrganizationID, userID, websocket.WSMessage{
		Type:    websocket.TypePermissionsUpdated,
		Payload: map[string]string{"message": "Your permissions have been updated"},
	})
}

// getTagsCached retrieves tags for an organization from cache or database
func (a *App) getTagsCached(orgID uuid.UUID) ([]models.Tag, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", tagsCachePrefix, orgID.String())

	// Try cache first
	cached, err := a.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var tags []models.Tag
		if err := json.Unmarshal([]byte(cached), &tags); err == nil {
			return tags, nil
		}
	}

	// Cache miss - fetch from database
	var tags []models.Tag
	if err := a.DB.Where("organization_id = ?", orgID).Order("name ASC").Find(&tags).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(tags); err == nil {
		a.Redis.Set(ctx, cacheKey, data, tagsCacheTTL)
	}

	return tags, nil
}

// InvalidateTagsCache invalidates the tags cache for an organization
func (a *App) InvalidateTagsCache(orgID uuid.UUID) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", tagsCachePrefix, orgID.String())
	a.Redis.Del(ctx, cacheKey)
}
