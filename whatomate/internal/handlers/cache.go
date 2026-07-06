package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// Cache TTLs - 6 hours since these rarely change (invalidated on update anyway)
	settingsCacheTTL        = 6 * time.Hour
	flowsCacheTTL           = 6 * time.Hour
	keywordRulesCacheTTL    = 6 * time.Hour
	whatsappAccountCacheTTL = 6 * time.Hour
	webhooksCacheTTL        = 6 * time.Hour
	slaSettingsCacheTTL     = 6 * time.Hour
	aiContextsCacheTTL      = 6 * time.Hour
	userPermissionsCacheTTL = 6 * time.Hour
	rolePermissionsCacheTTL = 6 * time.Hour
	tagsCacheTTL            = 6 * time.Hour

	// Cache key prefixes
	settingsCachePrefix        = "chatbot:settings:"
	flowsCachePrefix           = "chatbot:flows:"
	keywordRulesCachePrefix    = "chatbot:keywords:"
	whatsappAccountCachePrefix = "whatsapp:account:"
	webhooksCachePrefix        = "webhooks:"
	slaSettingsCacheKey        = "chatbot:sla_enabled_settings"
	aiContextsCachePrefix      = "chatbot:ai_contexts:"
	userPermissionsCachePrefix = "permissions:user:"
	rolePermissionsCachePrefix = "permissions:role:"
	tagsCachePrefix            = "tags:"
)

// chatbotSettingsCache is used for caching since AI.APIKey has json:"-" tag
type chatbotSettingsCache struct {
	models.ChatbotSettings
	AIAPIKey string `json:"ai_api_key_cache"`
}

// getChatbotSettingsCached retrieves chatbot settings from cache or database
func (a *App) getChatbotSettingsCached(orgID uuid.UUID, whatsAppAccount string) (*models.ChatbotSettings, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s:%s", settingsCachePrefix, orgID.String(), whatsAppAccount)

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var cacheData chatbotSettingsCache
			if err := json.Unmarshal([]byte(cached), &cacheData); err == nil {
				// Restore the API key from the cache wrapper
				cacheData.AI.APIKey = cacheData.AIAPIKey
				if err := a.decryptChatbotAIKey(&cacheData.ChatbotSettings); err != nil {
					return nil, err
				}
				return &cacheData.ChatbotSettings, nil
			}
		}
	}

	// Cache miss - fetch from database
	var settings models.ChatbotSettings
	result := a.DB.Where("organization_id = ? AND (whats_app_account = ? OR whats_app_account = '')",
		orgID, whatsAppAccount).
		Order("CASE WHEN whats_app_account = '' THEN 1 ELSE 0 END"). // Prefer account-specific settings
		First(&settings)

	if result.Error != nil {
		return nil, result.Error
	}

	// Cache the result (include AI APIKey explicitly since it has json:"-" tag)
	if a.Redis != nil {
		cacheData := chatbotSettingsCache{
			ChatbotSettings: settings,
			AIAPIKey:        settings.AI.APIKey,
		}
		if data, err := json.Marshal(cacheData); err == nil {
			a.Redis.Set(ctx, cacheKey, data, settingsCacheTTL)
		}
	}

	if err := a.decryptChatbotAIKey(&settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (a *App) decryptChatbotAIKey(settings *models.ChatbotSettings) error {
	if settings == nil || settings.AI.APIKey == "" {
		return nil
	}
	encKey := ""
	allowLegacy := true
	if a.Config != nil {
		encKey = a.Config.App.EncryptionKey
		if a.Config.App.AllowLegacyEncryption != nil {
			allowLegacy = *a.Config.App.AllowLegacyEncryption
		}
	}
	if !crypto.IsEncrypted(settings.AI.APIKey) {
		return fmt.Errorf("failed to decrypt chatbot AI API key: %w", crypto.ErrNotEncrypted)
	}
	dec, err := crypto.DecryptWithPolicy(settings.AI.APIKey, encKey, allowLegacy)
	if err != nil {
		return fmt.Errorf("failed to decrypt chatbot AI API key: %w", err)
	}
	settings.AI.APIKey = dec
	return nil
}

// getChatbotFlowsCached retrieves all enabled flows with steps from cache or database
func (a *App) getChatbotFlowsCached(orgID uuid.UUID) ([]models.ChatbotFlow, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", flowsCachePrefix, orgID.String())

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var flows []models.ChatbotFlow
			if err := json.Unmarshal([]byte(cached), &flows); err == nil {
				return flows, nil
			}
		}
	}

	// Cache miss - fetch from database
	var flows []models.ChatbotFlow
	if err := a.DB.Where("organization_id = ? AND is_enabled = true", orgID).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_order ASC")
		}).
		Find(&flows).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(flows); err == nil {
			a.Redis.Set(ctx, cacheKey, data, flowsCacheTTL)
		}
	}

	return flows, nil
}

// getChatbotFlowByIDCached retrieves a specific flow by ID from the cached flows list
func (a *App) getChatbotFlowByIDCached(orgID uuid.UUID, flowID uuid.UUID) (*models.ChatbotFlow, error) {
	flows, err := a.getChatbotFlowsCached(orgID)
	if err != nil {
		return nil, err
	}

	for i := range flows {
		if flows[i].ID == flowID {
			return &flows[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// getKeywordRulesCached retrieves keyword rules from cache or database
func (a *App) getKeywordRulesCached(orgID uuid.UUID, whatsAppAccount string) ([]models.KeywordRule, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s:%s", keywordRulesCachePrefix, orgID.String(), whatsAppAccount)

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var rules []models.KeywordRule
			if err := json.Unmarshal([]byte(cached), &rules); err == nil {
				return rules, nil
			}
		}
	}

	// Cache miss - fetch from database (account-specific + global)
	var rules []models.KeywordRule

	// Get account-specific rules
	var accountRules []models.KeywordRule
	if err := a.DB.Where("organization_id = ? AND whats_app_account = ? AND is_enabled = true",
		orgID, whatsAppAccount).
		Order("priority DESC").
		Find(&accountRules).Error; err != nil {
		a.Log.Error("Failed to fetch account keyword rules", "error", err, "org_id", orgID)
	}

	// Get global rules (whats_app_account = '')
	var globalRules []models.KeywordRule
	if err := a.DB.Where("organization_id = ? AND whats_app_account = '' AND is_enabled = true",
		orgID).
		Order("priority DESC").
		Find(&globalRules).Error; err != nil {
		a.Log.Error("Failed to fetch global keyword rules", "error", err, "org_id", orgID)
	}

	// Merge: account-specific first, then global
	rules = append(accountRules, globalRules...)

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(rules); err == nil {
			a.Redis.Set(ctx, cacheKey, data, keywordRulesCacheTTL)
		}
	}

	return rules, nil
}

// InvalidateChatbotSettingsCache invalidates the settings cache for an organization
func (a *App) InvalidateChatbotSettingsCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", settingsCachePrefix, orgID.String())
	a.deleteKeysByPattern(ctx, pattern)
}

// InvalidateChatbotFlowsCache invalidates the flows cache for an organization
func (a *App) InvalidateChatbotFlowsCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", flowsCachePrefix, orgID.String())
	a.Redis.Del(ctx, cacheKey)
}

// InvalidateKeywordRulesCache invalidates the keyword rules cache for an organization
func (a *App) InvalidateKeywordRulesCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", keywordRulesCachePrefix, orgID.String())
	a.deleteKeysByPattern(ctx, pattern)
}

// deleteKeysByPattern deletes all keys matching a pattern
func (a *App) deleteKeysByPattern(ctx context.Context, pattern string) {
	if a.Redis == nil {
		return
	}
	iter := a.Redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		a.Redis.Del(ctx, iter.Val())
	}
}

// whatsAppAccountCache is used for caching since AccessToken and AppSecret have json:"-" tag
type whatsAppAccountCache struct {
	models.WhatsAppAccount
	AccessToken string `json:"access_token"`
	AppSecret   string `json:"app_secret"`
}

// getWhatsAppAccountCached retrieves WhatsApp account by phone_id from cache or database
func (a *App) getWhatsAppAccountCached(phoneID string) (*models.WhatsAppAccount, error) {
	if a.Redis == nil {
		var account models.WhatsAppAccount
		if err := a.DB.Where("phone_id = ?", phoneID).First(&account).Error; err != nil {
			return nil, err
		}
		if err := a.decryptAccountSecrets(&account); err != nil {
			return nil, err
		}
		return &account, nil
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", whatsappAccountCachePrefix, phoneID)

	// Try cache first
	cached, err := a.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var cacheData whatsAppAccountCache
		if err := json.Unmarshal([]byte(cached), &cacheData); err == nil {
			cacheData.WhatsAppAccount.AccessToken = cacheData.AccessToken
			cacheData.WhatsAppAccount.AppSecret = cacheData.AppSecret
			if err := a.decryptAccountSecrets(&cacheData.WhatsAppAccount); err != nil {
				return nil, err
			}
			return &cacheData.WhatsAppAccount, nil
		}
	}

	// Cache miss - fetch from database
	var account models.WhatsAppAccount
	if err := a.DB.Where("phone_id = ?", phoneID).First(&account).Error; err != nil {
		return nil, err
	}

	// Cache the result (include AccessToken and AppSecret explicitly since they have json:"-")
	cacheData := whatsAppAccountCache{
		WhatsAppAccount: account,
		AccessToken:     account.AccessToken,
		AppSecret:       account.AppSecret,
	}
	// #nosec G117 -- cached secrets are stored encrypted-at-rest and decrypted only at access boundary.
	if data, err := json.Marshal(cacheData); err == nil {
		a.Redis.Set(ctx, cacheKey, data, whatsappAccountCacheTTL)
	}

	// Decrypt secrets before returning
	if err := a.decryptAccountSecrets(&account); err != nil {
		return nil, err
	}
	return &account, nil
}

// decryptAccountSecrets decrypts the encrypted secrets on a WhatsApp account.
// Handles both encrypted ("enc:" prefixed) and legacy unencrypted values transparently.
func (a *App) decryptAccountSecrets(account *models.WhatsAppAccount) error {
	encKey := ""
	allowLegacy := true
	if a.Config != nil {
		encKey = a.Config.App.EncryptionKey
		if a.Config.App.AllowLegacyEncryption != nil {
			allowLegacy = *a.Config.App.AllowLegacyEncryption
		}
	}
	// Let DecryptWithPolicy handle unencrypted (legacy) values transparently.
	if dec, err := crypto.DecryptWithPolicy(account.AccessToken, encKey, allowLegacy); err == nil {
		account.AccessToken = dec
	} else if account.AccessToken != "" {
		return fmt.Errorf("failed to decrypt access token for account %s: %w", account.Name, err)
	}

	if dec, err := crypto.DecryptWithPolicy(account.AppSecret, encKey, allowLegacy); err == nil {
		account.AppSecret = dec
	} else if account.AppSecret != "" {
		return fmt.Errorf("failed to decrypt app secret for account %s: %w", account.Name, err)
	}
	return nil
}

// InvalidateWhatsAppAccountCache invalidates the WhatsApp account cache
func (a *App) InvalidateWhatsAppAccountCache(phoneID string) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", whatsappAccountCachePrefix, phoneID)
	a.Redis.Del(ctx, cacheKey)
}

// getWebhooksCached retrieves active webhooks for an organization from cache or database
func (a *App) getWebhooksCached(orgID uuid.UUID) ([]models.Webhook, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", webhooksCachePrefix, orgID.String())

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var webhooks []models.Webhook
			if err := json.Unmarshal([]byte(cached), &webhooks); err == nil {
				return webhooks, nil
			}
		}
	}

	// Cache miss - fetch from database
	var webhooks []models.Webhook
	if err := a.DB.Where("organization_id = ? AND is_active = ?", orgID, true).Find(&webhooks).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(webhooks); err == nil {
			a.Redis.Set(ctx, cacheKey, data, webhooksCacheTTL)
		}
	}

	return webhooks, nil
}

// InvalidateWebhooksCache invalidates the webhooks cache for an organization
func (a *App) InvalidateWebhooksCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", webhooksCachePrefix, orgID.String())
	a.Redis.Del(ctx, cacheKey)
}

// getSLAEnabledSettingsCached retrieves all SLA-enabled chatbot settings from cache or database
func (a *App) getSLAEnabledSettingsCached() ([]models.ChatbotSettings, error) {
	ctx := context.Background()

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, slaSettingsCacheKey).Result()
		if err == nil && cached != "" {
			var settings []models.ChatbotSettings
			if err := json.Unmarshal([]byte(cached), &settings); err == nil {
				return settings, nil
			}
		}
	}

	// Cache miss - fetch from database
	var settings []models.ChatbotSettings
	if err := a.DB.Where("sla_enabled = ?", true).Find(&settings).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(settings); err == nil {
			a.Redis.Set(ctx, slaSettingsCacheKey, data, slaSettingsCacheTTL)
		}
	}

	return settings, nil
}

// InvalidateSLASettingsCache invalidates the SLA settings cache
func (a *App) InvalidateSLASettingsCache() {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	a.Redis.Del(ctx, slaSettingsCacheKey)
}

// getAIContextsCached retrieves AI contexts from cache or database
func (a *App) getAIContextsCached(orgID uuid.UUID, whatsAppAccount string) ([]models.AIContext, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s:%s", aiContextsCachePrefix, orgID.String(), whatsAppAccount)

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var contexts []models.AIContext
			if err := json.Unmarshal([]byte(cached), &contexts); err == nil {
				return contexts, nil
			}
		}
	}

	// Cache miss - fetch from database (account-specific + global)
	var contexts []models.AIContext

	// Get account-specific contexts
	var accountContexts []models.AIContext
	if err := a.DB.Where("organization_id = ? AND whats_app_account = ? AND is_enabled = true",
		orgID, whatsAppAccount).
		Order("priority DESC").
		Find(&accountContexts).Error; err != nil {
		a.Log.Error("Failed to fetch account AI contexts", "error", err, "org_id", orgID)
	}

	// Get global contexts (whats_app_account = '')
	var globalContexts []models.AIContext
	if err := a.DB.Where("organization_id = ? AND whats_app_account = '' AND is_enabled = true",
		orgID).
		Order("priority DESC").
		Find(&globalContexts).Error; err != nil {
		a.Log.Error("Failed to fetch global AI contexts", "error", err, "org_id", orgID)
	}

	// Merge: account-specific first, then global
	contexts = append(accountContexts, globalContexts...)

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(contexts); err == nil {
			a.Redis.Set(ctx, cacheKey, data, aiContextsCacheTTL)
		}
	}

	return contexts, nil
}

// InvalidateAIContextsCache invalidates the AI contexts cache for an organization
func (a *App) InvalidateAIContextsCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", aiContextsCachePrefix, orgID.String())
	a.deleteKeysByPattern(ctx, pattern)
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
// HasPermission checks if a user has a specific permission.
//
// Super admins automatically have all permissions.
//
// Returns true if the user has the permission, false otherwise.
// Returns false on error (failed to fetch permissions from cache/database).
//
// Parameters:
//   - userID: The user ID to check permissions for
//   - resource: The resource type (e.g., models.ResourceUsers, models.ResourceContacts)
//   - action: The action type (e.g., models.ActionRead, models.ActionWrite)
//   - orgIDs: Optional organization IDs to check org-specific permissions for.
//     If not provided, checks the user's default role permissions.
//     If multiple orgs are provided, checks if user has permission in ANY of them.
//
// Returns:
//   - true: User has the permission (or is super admin)
//   - false: User lacks the permission, or error occurred fetching permissions
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

// HasAnyPermission checks if a user has any of the specified permissions.
// Super admins have all permissions automatically.
// To check in a specific org, use HasAnyPermissionInOrg instead.
func (a *App) HasAnyPermission(userID uuid.UUID, permissions ...string) bool {
	perms, err := a.getUserPermissionsCached(userID)
	if err != nil {
		a.Log.Error("Failed to get user permissions", "error", err, "user_id", userID)
		return false
	}

	// Super admins have all permissions
	if perms.IsSuperAdmin {
		return true
	}

	permSet := make(map[string]bool)
	for _, p := range perms.Permissions {
		permSet[p] = true
	}

	for _, p := range permissions {
		if permSet[p] {
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

// ScopedQuery returns a gorm query scoped to the organization
// Always filters by organization - uuid.Nil is not allowed
func (a *App) ScopedQuery(userID, orgID uuid.UUID) *gorm.DB {
	return tenant.ScopedDB(a.DB, orgID)
}

// ScopeToOrg adds organization scoping to an existing query
// Always filters by organization - uuid.Nil is not allowed
func (a *App) ScopeToOrg(query *gorm.DB, userID, orgID uuid.UUID) *gorm.DB {
	return tenant.ScopedDB(query, orgID)
}

// GetRolePermissionsCached retrieves role permissions from cache or database.
//
// Returns a slice of permission strings in "resource:action" format (e.g., "users:read", "contacts:write").
// The permissions are cached in Redis with rolePermissionsCacheTTL.
//
// Returns:
//   - []string: List of permissions in "resource:action" format
//   - error: Returns error if database query fails (e.g., role not found)
//
// Cache behavior:
//   - First checks Redis cache for cached permissions
//   - On cache miss, queries database via JOIN to get permissions
//   - Populates cache with result for future calls
func (a *App) GetRolePermissionsCached(roleID uuid.UUID) ([]string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", rolePermissionsCachePrefix, roleID.String())

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var perms []string
			if err := json.Unmarshal([]byte(cached), &perms); err == nil {
				return perms, nil
			}
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
	if a.Redis != nil {
		if data, err := json.Marshal(perms); err == nil {
			a.Redis.Set(ctx, cacheKey, data, rolePermissionsCacheTTL)
		}
	}

	return perms, nil
}

// InvalidateUserPermissionsCache invalidates the permissions cache for a user.
//
// This function deletes both the base user permissions cache and all org-specific
// cache entries for the user. Additionally, it sends a WebSocket message to the user
// to notify them that their permissions have changed (triggering a client-side refresh).
//
// Side effects:
//   - Deletes Redis cache entries for user permissions
//   - Sends WebSocket notification to user (triggers client-side permissions refresh)
//
// Parameters:
//   - userID: The user ID whose permissions cache should be invalidated
func (a *App) InvalidateUserPermissionsCache(userID uuid.UUID) {
	if a.Redis == nil {
		return
	}
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

// InvalidateRolePermissionsCache invalidates the permissions cache for a role and all users with that role.
//
// This function cascades the invalidation: it deletes the role's permissions cache,
// finds all users assigned this role, invalidates their individual permissions caches,
// and sends WebSocket notifications to all affected users.
//
// Side effects:
//   - Deletes Redis cache entry for the role permissions
//   - Deletes Redis cache entries for all users with this role
//   - Sends WebSocket notifications to all affected users (triggers client-side permissions refresh)
//
// If the database query to find users fails, this function logs the error and returns
// without attempting to notify users (since we don't know who they are).
//
// Parameters:
//   - roleID: The role ID whose cache should be invalidated (affects all users with this role)
func (a *App) InvalidateRolePermissionsCache(roleID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()

	// Delete role cache
	roleCacheKey := fmt.Sprintf("%s%s", rolePermissionsCachePrefix, roleID.String())
	a.Redis.Del(ctx, roleCacheKey)

	// Find all users with this role and invalidate their cache
	var users []models.User
	if err := a.DB.Select("id, organization_id").Where("role_id = ?", roleID).Find(&users).Error; err != nil {
		a.Log.Error("Failed to find users for role permission cache invalidation", "error", err, "role_id", roleID)
		return
	}

	for _, user := range users {
		userCacheKey := fmt.Sprintf("%s%s", userPermissionsCachePrefix, user.ID.String())
		a.Redis.Del(ctx, userCacheKey)

		// Notify user via WebSocket to refresh their permissions
		a.notifyUserPermissionsChanged(user.ID)
	}
}

// InvalidateOrgPermissionsCache invalidates all permission caches for an organization.
//
// This function finds all roles belonging to the organization and invalidates
// each role's permissions cache (which in turn invalidates all user permissions caches
// for users with those roles and sends WebSocket notifications).
//
// Side effects:
//   - Queries database for all roles in the organization
//   - Calls InvalidateRolePermissionsCache for each role (see that function for full side effects)
//
// If the database query to find roles fails, this function logs the error and returns
// without attempting to invalidate any caches.
//
// Parameters:
//   - orgID: The organization ID whose role permissions should all be invalidated
func (a *App) InvalidateOrgPermissionsCache(orgID uuid.UUID) {
	// Find all roles in this org
	var roles []models.CustomRole
	if err := a.DB.Select("id").Where("organization_id = ?", orgID).Find(&roles).Error; err != nil {
		a.Log.Error("Failed to find roles for org permission cache invalidation", "error", err, "org_id", orgID)
		return
	}

	for _, role := range roles {
		a.InvalidateRolePermissionsCache(role.ID)
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
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var tags []models.Tag
			if err := json.Unmarshal([]byte(cached), &tags); err == nil {
				return tags, nil
			}
		}
	}

	// Cache miss - fetch from database
	var tags []models.Tag
	if err := a.DB.Where("organization_id = ?", orgID).Order("name ASC").Find(&tags).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(tags); err == nil {
			a.Redis.Set(ctx, cacheKey, data, tagsCacheTTL)
		}
	}

	return tags, nil
}

// InvalidateTagsCache invalidates the tags cache for an organization
func (a *App) InvalidateTagsCache(orgID uuid.UUID) {
	if a.Redis == nil {
		return
	}
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", tagsCachePrefix, orgID.String())
	a.Redis.Del(ctx, cacheKey)
}
