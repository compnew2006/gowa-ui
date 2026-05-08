package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidateChatbotSettingsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "chatbot:settings:" + orgID.String() + ":test_account"
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "cache should exist before invalidation")

	app.InvalidateChatbotSettingsCache(orgID)

	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "cache should be deleted after invalidation")
}

func TestInvalidateChatbotSettingsCache_MultipleAccounts(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	keys := []string{
		"chatbot:settings:" + orgID.String() + ":account1",
		"chatbot:settings:" + orgID.String() + ":account2",
		"chatbot:settings:" + orgID.String() + ":account3",
	}
	for _, key := range keys {
		require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())
	}

	app.InvalidateChatbotSettingsCache(orgID)

	for _, key := range keys {
		val := app.Redis.Get(ctx, key)
		assert.Error(t, val.Err(), "cache key %s should be deleted", key)
	}
}

func TestInvalidateChatbotSettingsCache_DifferentOrg(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	org1 := uuid.New()
	org2 := uuid.New()

	ctx := context.Background()
	key1 := "chatbot:settings:" + org1.String() + ":test_account"
	key2 := "chatbot:settings:" + org2.String() + ":test_account"
	require.NoError(t, app.Redis.Set(ctx, key1, "test", time.Minute).Err())
	require.NoError(t, app.Redis.Set(ctx, key2, "test", time.Minute).Err())

	app.InvalidateChatbotSettingsCache(org1)

	val1 := app.Redis.Get(ctx, key1)
	assert.Error(t, val1.Err(), "org1 cache should be deleted")

	val2 := app.Redis.Get(ctx, key2)
	assert.NoError(t, val2.Err(), "org2 cache should remain")
}

func TestInvalidateChatbotFlowsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "chatbot:flows:" + orgID.String()
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateChatbotFlowsCache(orgID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "flows cache should be deleted")
}

func TestInvalidateKeywordRulesCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "chatbot:keywords:" + orgID.String() + ":test_account"
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateKeywordRulesCache(orgID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "keyword rules cache should be deleted")
}

func TestInvalidateKeywordRulesCache_MultipleAccounts(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	keys := []string{
		"chatbot:keywords:" + orgID.String() + ":acc1",
		"chatbot:keywords:" + orgID.String() + ":acc2",
	}
	for _, key := range keys {
		require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())
	}

	app.InvalidateKeywordRulesCache(orgID)

	for _, key := range keys {
		val := app.Redis.Get(ctx, key)
		assert.Error(t, val.Err(), "cache key %s should be deleted", key)
	}
}

func TestInvalidateWhatsAppAccountCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	ctx := context.Background()
	phoneID := "1234567890"
	key := "whatsapp:account:" + phoneID
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateWhatsAppAccountCache(phoneID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "WhatsApp account cache should be deleted")
}

func TestInvalidateWhatsAppAccountCache_NilRedis(t *testing.T) {
	t.Parallel()

	app := &App{
		Config: &config.Config{},
		Redis:  nil,
	}

	assert.NotPanics(t, func() {
		app.InvalidateWhatsAppAccountCache("1234567890")
	})
}

func TestInvalidateWebhooksCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "webhooks:" + orgID.String()
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateWebhooksCache(orgID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "webhooks cache should be deleted")
}

func TestInvalidateSLASettingsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	ctx := context.Background()
	key := "chatbot:sla_enabled_settings"
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateSLASettingsCache()

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "SLA settings cache should be deleted")
}

func TestInvalidateAIContextsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "chatbot:ai_contexts:" + orgID.String() + ":test_account"
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateAIContextsCache(orgID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "AI contexts cache should be deleted")
}

func TestInvalidateAIContextsCache_MultipleAccounts(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	keys := []string{
		"chatbot:ai_contexts:" + orgID.String() + ":acc1",
		"chatbot:ai_contexts:" + orgID.String() + ":acc2",
	}
	for _, key := range keys {
		require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())
	}

	app.InvalidateAIContextsCache(orgID)

	for _, key := range keys {
		val := app.Redis.Get(ctx, key)
		assert.Error(t, val.Err(), "cache key %s should be deleted", key)
	}
}

func TestInvalidateUserPermissionsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	userID := uuid.New()

	ctx := context.Background()
	baseKey := "permissions:user:" + userID.String()
	orgKey := "permissions:user:" + userID.String() + ":" + uuid.New().String()
	require.NoError(t, app.Redis.Set(ctx, baseKey, "test", time.Minute).Err())
	require.NoError(t, app.Redis.Set(ctx, orgKey, "test", time.Minute).Err())

	app.InvalidateUserPermissionsCache(userID)

	val := app.Redis.Get(ctx, baseKey)
	assert.Error(t, val.Err(), "base permissions cache should be deleted")

	val = app.Redis.Get(ctx, orgKey)
	assert.Error(t, val.Err(), "org-specific permissions cache should be deleted")
}

func TestInvalidateTagsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	ctx := context.Background()
	key := "tags:" + orgID.String()
	require.NoError(t, app.Redis.Set(ctx, key, "test", time.Minute).Err())

	app.InvalidateTagsCache(orgID)

	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "tags cache should be deleted")
}

func TestInvalidateRolePermissionsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)
	roleID := uuid.New()

	ctx := context.Background()
	roleKey := "permissions:role:" + roleID.String()
	require.NoError(t, app.Redis.Set(ctx, roleKey, "test", time.Minute).Err())

	app.InvalidateRolePermissionsCache(roleID)

	val := app.Redis.Get(ctx, roleKey)
	assert.Error(t, val.Err(), "role permissions cache should be deleted")
}

func TestScopedQuery(t *testing.T) {
	db := testutil.SetupTestDB(t)
	mr := miniredis.RunT(t)
	app := &App{
		Config: &config.Config{},
		Redis:  redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		DB:     db,
	}
	orgID := uuid.New()
	userID := uuid.New()

	query := app.ScopedQuery(userID, orgID)
	assert.NotNil(t, query, "ScopedQuery should return a non-nil query")
}

func TestScopeToOrg(t *testing.T) {
	db := testutil.SetupTestDB(t)
	mr := miniredis.RunT(t)
	app := &App{
		Config: &config.Config{},
		Redis:  redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		DB:     db,
	}
	orgID := uuid.New()
	userID := uuid.New()

	baseQuery := app.DB.Model(&struct{}{})
	query := app.ScopeToOrg(baseQuery, userID, orgID)
	assert.NotNil(t, query, "ScopeToOrg should return a non-nil query")
}

func TestHasPermission_NonexistentUser(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	result := app.HasPermission(uuid.New(), "contacts", "read")
	assert.False(t, result, "nonexistent user should not have permissions")
}

func TestHasAnyPermission_NonexistentUser(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	result := app.HasAnyPermission(uuid.New(), "contacts:read", "contacts:write")
	assert.False(t, result, "nonexistent user should not have any permissions")
}

func TestIsSuperAdmin_NonexistentUser(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := &App{
		Config: &config.Config{},
		Redis:  client,
		DB:     nil,
	}

	result := app.IsSuperAdmin(uuid.New())
	assert.False(t, result, "nonexistent user should not be super admin")
}

func TestDeleteKeysByPattern(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	ctx := context.Background()
	require.NoError(t, app.Redis.Set(ctx, "test:prefix:1", "val1", time.Minute).Err())
	require.NoError(t, app.Redis.Set(ctx, "test:prefix:2", "val2", time.Minute).Err())
	require.NoError(t, app.Redis.Set(ctx, "test:other:1", "val3", time.Minute).Err())

	app.deleteKeysByPattern(ctx, "test:prefix:*")

	assert.Error(t, app.Redis.Get(ctx, "test:prefix:1").Err())
	assert.Error(t, app.Redis.Get(ctx, "test:prefix:2").Err())
	assert.NoError(t, app.Redis.Get(ctx, "test:other:1").Err(), "non-matching key should remain")
}

func TestDeleteKeysByPattern_NoMatch(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	app := createAppWithRedis(t, mr)

	ctx := context.Background()
	require.NoError(t, app.Redis.Set(ctx, "test:keep:1", "val1", time.Minute).Err())

	app.deleteKeysByPattern(ctx, "test:nomatch:*")

	assert.NoError(t, app.Redis.Get(ctx, "test:keep:1").Err(), "non-matching key should remain")
}

func TestChatbotSettingsCacheStruct(t *testing.T) {
	t.Parallel()

	cacheData := chatbotSettingsCache{
		AIAPIKey: "test-key",
	}
	assert.Equal(t, "test-key", cacheData.AIAPIKey, "AIAPIKey should be stored in cache wrapper")
}

func TestWhatsAppAccountCacheStruct(t *testing.T) {
	t.Parallel()

	cacheData := whatsAppAccountCache{
		AccessToken: "test-token",
		AppSecret:   "test-secret",
	}
	assert.Equal(t, "test-token", cacheData.AccessToken)
	assert.Equal(t, "test-secret", cacheData.AppSecret)
}

func TestUserPermissionsStruct(t *testing.T) {
	t.Parallel()

	perms := UserPermissions{
		RoleID:       uuid.New(),
		RoleName:     "Admin",
		IsSystem:     true,
		IsSuperAdmin: false,
		Permissions:  []string{"contacts:read", "contacts:write"},
	}

	assert.Equal(t, "Admin", perms.RoleName)
	assert.True(t, perms.IsSystem)
	assert.False(t, perms.IsSuperAdmin)
	assert.Equal(t, []string{"contacts:read", "contacts:write"}, perms.Permissions)
}

func TestCacheTTLConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 6*time.Hour, settingsCacheTTL)
	assert.Equal(t, 6*time.Hour, flowsCacheTTL)
	assert.Equal(t, 6*time.Hour, keywordRulesCacheTTL)
	assert.Equal(t, 6*time.Hour, whatsappAccountCacheTTL)
	assert.Equal(t, 6*time.Hour, webhooksCacheTTL)
	assert.Equal(t, 6*time.Hour, slaSettingsCacheTTL)
	assert.Equal(t, 6*time.Hour, aiContextsCacheTTL)
	assert.Equal(t, 6*time.Hour, userPermissionsCacheTTL)
	assert.Equal(t, 6*time.Hour, rolePermissionsCacheTTL)
	assert.Equal(t, 6*time.Hour, tagsCacheTTL)
}

func TestCachePrefixConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "chatbot:settings:", settingsCachePrefix)
	assert.Equal(t, "chatbot:flows:", flowsCachePrefix)
	assert.Equal(t, "chatbot:keywords:", keywordRulesCachePrefix)
	assert.Equal(t, "whatsapp:account:", whatsappAccountCachePrefix)
	assert.Equal(t, "webhooks:", webhooksCachePrefix)
	assert.Equal(t, "chatbot:sla_enabled_settings", slaSettingsCacheKey)
	assert.Equal(t, "chatbot:ai_contexts:", aiContextsCachePrefix)
	assert.Equal(t, "permissions:user:", userPermissionsCachePrefix)
	assert.Equal(t, "permissions:role:", rolePermissionsCachePrefix)
	assert.Equal(t, "tags:", tagsCachePrefix)
}

func createAppWithRedis(t *testing.T, mr *miniredis.Miniredis) *App {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &App{
		Config: &config.Config{},
		Redis:  client,
		DB:     testutil.SetupTestDB(t),
	}
}
