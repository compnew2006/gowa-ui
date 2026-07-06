package handlers_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestParseContextUUID_UUIDType tests parseContextUUID with uuid.UUID type
func TestParseContextUUID_UUIDType(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	result, ok := handlers.ParseContextUUID(orgID)

	assert.True(t, ok, "parseContextUUID should succeed with uuid.UUID")
	assert.Equal(t, orgID, result, "parseContextUUID should return the same UUID")
}

// TestParseContextUUID_StringType tests parseContextUUID with string type
func TestParseContextUUID_StringType(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	result, ok := handlers.ParseContextUUID(orgID.String())

	assert.True(t, ok, "parseContextUUID should succeed with valid UUID string")
	assert.Equal(t, orgID, result, "parseContextUUID should parse the UUID correctly")
}

// TestParseContextUUID_InvalidString tests parseContextUUID with invalid string
func TestParseContextUUID_InvalidString(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID("not-a-uuid")

	assert.False(t, ok, "parseContextUUID should fail with invalid string")
	assert.Equal(t, uuid.Nil, result, "parseContextUUID should return Nil UUID on failure")
}

// TestParseContextUUID_NilValue tests parseContextUUID with nil value
func TestParseContextUUID_NilValue(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(nil)

	assert.False(t, ok, "parseContextUUID should fail with nil value")
	assert.Equal(t, uuid.Nil, result, "parseContextUUID should return Nil UUID on failure")
}

// TestParseContextUUID_WrongType tests parseContextUUID with wrong type
func TestParseContextUUID_WrongType(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(12345)

	assert.False(t, ok, "parseContextUUID should fail with wrong type")
	assert.Equal(t, uuid.Nil, result, "parseContextUUID should return Nil UUID on failure")
}

// TestInvalidateChatbotSettingsCache tests InvalidateChatbotSettingsCache
func TestInvalidateChatbotSettingsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// First, set a value in cache with correct key format
	ctx := context.Background()
	key := "chatbot:settings:" + orgID.String() + ":test_account"
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Verify it exists
	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "Cache value should exist")

	// Invalidate
	app.InvalidateChatbotSettingsCache(orgID)

	// Verify it's gone
	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateChatbotFlowsCache tests InvalidateChatbotFlowsCache
func TestInvalidateChatbotFlowsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// Set a value in cache
	ctx := context.Background()
	key := "chatbot:flows:" + orgID.String()
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Verify it exists
	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "Cache value should exist before invalidation")

	// Invalidate
	app.InvalidateChatbotFlowsCache(orgID)

	// Verify it's gone
	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateKeywordRulesCache tests InvalidateKeywordRulesCache
func TestInvalidateKeywordRulesCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// Set a value in cache with correct key format
	ctx := context.Background()
	key := "chatbot:keywords:" + orgID.String() + ":test_account"
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Verify it exists
	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "Cache value should exist before invalidation")

	// Invalidate
	app.InvalidateKeywordRulesCache(orgID)

	// Verify it's gone
	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateWebhooksCache tests InvalidateWebhooksCache
func TestInvalidateWebhooksCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// Set a value in cache
	ctx := context.Background()
	key := "webhooks:" + orgID.String()
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Invalidate
	app.InvalidateWebhooksCache(orgID)

	// Verify it's gone
	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateTagsCache tests InvalidateTagsCache
func TestInvalidateTagsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// Set a value in cache
	ctx := context.Background()
	key := "tags:" + orgID.String()
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Invalidate
	app.InvalidateTagsCache(orgID)

	// Verify it's gone
	val := app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateAIContextsCache tests InvalidateAIContextsCache
func TestInvalidateAIContextsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	orgID := uuid.New()

	// Set a value in cache with correct key format
	ctx := context.Background()
	key := "chatbot:ai_contexts:" + orgID.String() + ":test_account"
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Verify it exists
	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "Cache value should exist before invalidation")

	// Invalidate
	app.InvalidateAIContextsCache(orgID)

	// Verify it's gone
	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateUserPermissionsCache tests InvalidateUserPermissionsCache
func TestInvalidateUserPermissionsCache(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)

	app := createAppWithRedis(t, mr)
	userID := uuid.New()

	// Set a value in cache
	ctx := context.Background()
	key := "permissions:user:" + userID.String()
	err := app.Redis.Set(ctx, key, "test", time.Minute).Err()
	require.NoError(t, err, "Failed to set cache")

	// Verify it exists
	val := app.Redis.Get(ctx, key)
	assert.NoError(t, val.Err(), "Cache value should exist before invalidation")

	// Invalidate
	app.InvalidateUserPermissionsCache(userID)

	// Verify it's gone
	val = app.Redis.Get(ctx, key)
	assert.Error(t, val.Err(), "Cache value should be deleted")
}

// TestInvalidateRolePermissionsCache tests InvalidateRolePermissionsCache
func TestInvalidateRolePermissionsCache(t *testing.T) {
	t.Skip("Skipping: Requires database access to find users with role")
}

// Helper functions

// newFastHTTPCtx creates a fasthttp.RequestCtx for testing

// createAppWithRedis creates an App with miniredis for testing
func createAppWithRedis(t *testing.T, mr *miniredis.Miniredis) *handlers.App {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &handlers.App{
		Config: &config.Config{},
		Redis:  client,
		DB:     &gorm.DB{}, // Mock DB
	}
}
