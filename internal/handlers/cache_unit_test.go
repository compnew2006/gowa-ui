package handlers_test

import (
	"context"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
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

// TestRequestPath_GETRequest tests requestPath with GET request
func TestRequestPath_GETRequest(t *testing.T) {
	t.Parallel()

	req := &fastglue.Request{RequestCtx: newFastHTTPCtx("GET", "/api/test", nil)}
	result := handlers.RequestPath(req)

	assert.Equal(t, "/api/test", result, "requestPath should return the request path")
}

// TestRequestPath_POSTRequest tests requestPath with POST request
func TestRequestPath_POSTRequest(t *testing.T) {
	t.Parallel()

	req := &fastglue.Request{RequestCtx: newFastHTTPCtx("POST", "/api/users", nil)}
	result := handlers.RequestPath(req)

	assert.Equal(t, "/api/users", result, "requestPath should return the request path")
}

// TestRequestPath_WithQueryParams tests requestPath with query parameters
func TestRequestPath_WithQueryParams(t *testing.T) {
	t.Parallel()

	ctx := newFastHTTPCtx("GET", "/api/test?page=1&limit=10", nil)
	req := &fastglue.Request{RequestCtx: ctx}
	result := handlers.RequestPath(req)

	assert.Equal(t, "/api/test", result, "requestPath should return path without query params")
}

// TestRequestMethod_GET tests RequestMethod with GET
func TestRequestMethod_GET(t *testing.T) {
	t.Parallel()

	req := &fastglue.Request{RequestCtx: newFastHTTPCtx("GET", "/api/test", nil)}
	result := handlers.RequestMethod(req)

	assert.Equal(t, "GET", result, "RequestMethod should return GET")
}

// TestRequestMethod_POST tests RequestMethod with POST
func TestRequestMethod_POST(t *testing.T) {
	t.Parallel()

	req := &fastglue.Request{RequestCtx: newFastHTTPCtx("POST", "/api/users", nil)}
	result := handlers.RequestMethod(req)

	assert.Equal(t, "POST", result, "RequestMethod should return POST")
}

// TestRequestUserAgent_Present tests RequestUserAgent when present
func TestRequestUserAgent_Present(t *testing.T) {
	t.Parallel()

	ctx := newFastHTTPCtx("GET", "/api/test", nil)
	ctx.Request.Header.Set("User-Agent", "TestAgent/1.0")
	req := &fastglue.Request{RequestCtx: ctx}

	result := handlers.RequestUserAgent(req)

	assert.Equal(t, "TestAgent/1.0", result, "RequestUserAgent should return the user agent")
}

// TestRequestUserAgent_Missing tests RequestUserAgent when missing
func TestRequestUserAgent_Missing(t *testing.T) {
	t.Parallel()

	req := &fastglue.Request{RequestCtx: newFastHTTPCtx("GET", "/api/test", nil)}
	result := handlers.RequestUserAgent(req)

	assert.Empty(t, result, "RequestUserAgent should return empty string when missing")
}


// TestRequestClientIP_ProxyTrusted_XForwardedFor tests RequestClientIP with proxy and X-Forwarded-For
func TestRequestClientIP_ProxyTrusted_XForwardedFor(t *testing.T) {
	t.Parallel()

	ctx := newFastHTTPCtx("GET", "/api/test", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
	ctx.Request.Header.Set("RemoteAddr", "192.168.1.1:12345")
	req := &fastglue.Request{RequestCtx: ctx}

	result := handlers.RequestClientIP(req, true)

	assert.Equal(t, "203.0.113.1", result, "RequestClientIP should return first IP from X-Forwarded-For")
}

// TestRequestClientIP_ProxyTrusted_XRealIP tests RequestClientIP with proxy and X-Real-IP
func TestRequestClientIP_ProxyTrusted_XRealIP(t *testing.T) {
	t.Parallel()

	ctx := newFastHTTPCtx("GET", "/api/test", nil)
	ctx.Request.Header.Set("X-Real-IP", "203.0.113.1")
	ctx.Request.Header.Set("RemoteAddr", "192.168.1.1:12345")
	req := &fastglue.Request{RequestCtx: ctx}

	result := handlers.RequestClientIP(req, true)

	assert.Equal(t, "203.0.113.1", result, "RequestClientIP should return X-Real-IP")
}

// TestNormalizeActivityText_ShortText tests normalizeActivityText with short text
func TestNormalizeActivityText_ShortText(t *testing.T) {
	t.Parallel()

	text := "Short text"
	result := handlers.NormalizeActivityText(text, 100)

	assert.Equal(t, text, result, "normalizeActivityText should not modify short text")
}

// TestNormalizeActivityText_LongText tests normalizeActivityText with text longer than limit
func TestNormalizeActivityText_LongText(t *testing.T) {
	t.Parallel()

	text := "This is a very long text that exceeds the limit and should be truncated"
	limit := 20
	result := handlers.NormalizeActivityText(text, limit)

	assert.LessOrEqual(t, len(result), limit+3, "normalizeActivityText should truncate to limit + '...'")
	assert.Contains(t, result, "...", "normalizeActivityText should add ellipsis")
}

// TestNormalizeActivityText_EmptyText tests normalizeActivityText with empty text
func TestNormalizeActivityText_EmptyText(t *testing.T) {
	t.Parallel()

	result := handlers.NormalizeActivityText("", 100)

	assert.Empty(t, result, "normalizeActivityText should handle empty text")
}

// TestTrustProxyEnabled_Default tests TrustProxyEnabled with default config
func TestTrustProxyEnabled_Default(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	result := app.TrustProxyEnabled()

	assert.False(t, result, "TrustProxyEnabled should return false by default")
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
func newFastHTTPCtx(method, path string, body []byte) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	if body != nil {
		ctx.Request.SetBody(body)
	}
	return ctx
}

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
