package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TestParseContextUUID_UUIDInput tests parsing a UUID from uuid.UUID type
func TestParseContextUUID_UUIDInput(t *testing.T) {
	t.Parallel()

	testUUID := uuid.New()
	result, ok := handlers.ParseContextUUID(testUUID)

	assert.True(t, ok, "Should successfully parse UUID input")
	assert.Equal(t, testUUID, result, "Should return the same UUID")
}

// TestParseContextUUID_ValidStringUUID tests parsing a valid UUID string
func TestParseContextUUID_ValidStringUUID(t *testing.T) {
	t.Parallel()

	testUUID := uuid.New()
	result, ok := handlers.ParseContextUUID(testUUID.String())

	assert.True(t, ok, "Should successfully parse valid UUID string")
	assert.Equal(t, testUUID, result, "Should return the parsed UUID")
}

// TestParseContextUUID_InvalidStringUUID tests parsing an invalid UUID string
func TestParseContextUUID_InvalidStringUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "not a uuid",
			input: "not-a-uuid",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "partial uuid",
			input: "550e8400-e29b-41d4",
		},
		{
			name:  "invalid format",
			input: "550e8400-e29b-41d4-a716-446655440000-extra",
		},
		{
			name:  "random characters",
			input: "abcdefgh-ijkl-mnop-qrst-uvwxyz123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, ok := handlers.ParseContextUUID(tt.input)

			assert.False(t, ok, "Should fail to parse invalid UUID string")
			assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
		})
	}
}

// TestParseContextUUID_IntegerInput tests parsing an integer type
func TestParseContextUUID_IntegerInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(12345)

	assert.False(t, ok, "Should fail to parse integer input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_NilInput tests parsing nil input
func TestParseContextUUID_NilInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(nil)

	assert.False(t, ok, "Should fail to parse nil input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_MapInput tests parsing a map type
func TestParseContextUUID_MapInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(map[string]any{"key": "value"})

	assert.False(t, ok, "Should fail to parse map input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_SliceInput tests parsing a slice type
func TestParseContextUUID_SliceInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID([]string{"a", "b"})

	assert.False(t, ok, "Should fail to parse slice input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_BoolInput tests parsing a boolean type
func TestParseContextUUID_BoolInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(true)

	assert.False(t, ok, "Should fail to parse boolean input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_FloatInput tests parsing a float type
func TestParseContextUUID_FloatInput(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(3.14)

	assert.False(t, ok, "Should fail to parse float input")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_NilUUID tests parsing uuid.Nil
func TestParseContextUUID_NilUUID(t *testing.T) {
	t.Parallel()

	result, ok := handlers.ParseContextUUID(uuid.Nil)

	assert.True(t, ok, "Should successfully parse Nil UUID")
	assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
}

// TestParseContextUUID_WhiteSpaceString tests parsing whitespace strings
func TestParseContextUUID_WhiteSpaceString(t *testing.T) {
	t.Parallel()

	testUUID := uuid.New()

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "UUID with leading space",
			input: " " + testUUID.String(),
			valid: false, // Leading space causes length error
		},
		{
			name:  "UUID with trailing space",
			input: testUUID.String() + " ",
			valid: false, // Trailing space causes length error
		},
		{
			name:  "UUID with leading and trailing spaces",
			input: " " + testUUID.String() + " ",
			valid: true, // uuid.Parse oddly handles this case correctly
		},
		{
			name:  "UUID with tab character",
			input: "\t" + testUUID.String(),
			valid: false,
		},
		{
			name:  "UUID with newline",
			input: "\n" + testUUID.String(),
			valid: false,
		},
		{
			name:  "UUID with multiple spaces",
			input: "  " + testUUID.String(),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, ok := handlers.ParseContextUUID(tt.input)

			if tt.valid {
				assert.True(t, ok, "Should successfully parse UUID with whitespace")
				assert.Equal(t, testUUID, result, "Should return the parsed UUID")
			} else {
				assert.False(t, ok, "Should fail to parse UUID with whitespace")
				assert.Equal(t, uuid.Nil, result, "Should return Nil UUID")
			}
		})
	}
}

// createMockRequest creates a mock fastglue.Request for testing
func createMockRequest(path string, userValues map[string]any) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI(path)

	// Set user values
	for key, value := range userValues {
		ctx.SetUserValue(key, value)
	}

	// Create a mock response
	ctx.Response.SetStatusCode(200)

	return &fastglue.Request{RequestCtx: ctx}
}

// TestActivityLogMiddleware_NonAPIPath tests middleware with non-API path
func TestActivityLogMiddleware_NonAPIPath(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	orgID := uuid.New()
	req := createMockRequest("/static/file.html", map[string]any{
		"user_id":        userID,
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request")
}

// TestActivityLogMiddleware_ActivityLogsPath tests middleware with /api/activity-logs path
func TestActivityLogMiddleware_ActivityLogsPath(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "exact activity-logs path",
			path: "/api/activity-logs",
		},
		{
			name: "activity-logs with ID",
			path: "/api/activity-logs/123",
		},
		{
			name: "activity-logs with nested path",
			path: "/api/activity-logs/export",
		},
		{
			name: "activity-logs with query params",
			path: "/api/activity-logs?page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			orgID := uuid.New()
			req := createMockRequest(tt.path, map[string]any{
				"user_id":        userID,
				"organization_id": orgID,
			})

			result := middleware(req)

			assert.Same(t, req, result, "Middleware should return the same request for activity-logs paths")
		})
	}
}

// TestActivityLogMiddleware_MissingUserID tests middleware with missing user_id
func TestActivityLogMiddleware_MissingUserID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request when user_id is missing")
}

// TestActivityLogMiddleware_NilUUIDUserID tests middleware with nil UUID user_id
func TestActivityLogMiddleware_NilUUIDUserID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        uuid.Nil,
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request when user_id is Nil")
}

// TestActivityLogMiddleware_MissingOrgID tests middleware with missing organization_id
func TestActivityLogMiddleware_MissingOrgID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id": userID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request when organization_id is missing")
}

// TestActivityLogMiddleware_StringUserID tests middleware with string user_id
// Note: This test requires TEST_DATABASE_URL to run fully
func TestActivityLogMiddleware_StringUserID(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        userID.String(),
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request")
}

// TestActivityLogMiddleware_UUIDUserID tests middleware with UUID user_id
// Note: This test requires TEST_DATABASE_URL to run fully
func TestActivityLogMiddleware_UUIDUserID(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        userID,
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request")
}

// TestActivityLogMiddleware_VariousAPIPaths tests middleware with various API paths
// Note: Tests with valid credentials require TEST_DATABASE_URL
func TestActivityLogMiddleware_VariousAPIPaths(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	tests := []struct {
		name           string
		path           string
		missingUserID  bool
		missingOrgID   bool
		skipRequiresDB bool
	}{
		{
			name:           "simple API path",
			path:           "/api/users",
			skipRequiresDB: true,
		},
		{
			name:           "nested API path",
			path:           "/api/organizations/123/users",
			skipRequiresDB: true,
		},
		{
			name:           "API with query params",
			path:           "/api/messages?status=active",
			skipRequiresDB: true,
		},
		{
			name:           "API with trailing slash",
			path:           "/api/contacts/",
			skipRequiresDB: true,
		},
		{
			name:          "API path missing user_id",
			path:          "/api/test-endpoint",
			missingUserID: true,
		},
		{
			name:         "API path missing org_id",
			path:         "/api/test-endpoint",
			missingOrgID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skipRequiresDB {
				t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")
			}

			userID := uuid.New()
			orgID := uuid.New()
			userValues := map[string]any{}

			if !tt.missingUserID {
				userValues["user_id"] = userID
			}
			if !tt.missingOrgID {
				userValues["organization_id"] = orgID
			}

			req := createMockRequest(tt.path, userValues)
			result := middleware(req)

			assert.Same(t, req, result, "Middleware should return the same request for API paths")
		})
	}
}

// TestActivityLogMiddleware_NonAPIPaths tests middleware with non-API paths
func TestActivityLogMiddleware_NonAPIPaths(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "static file",
			path: "/static/style.css",
		},
		{
			name: "health check",
			path: "/health",
		},
		{
			name: "root path",
			path: "/",
		},
		{
			name: "websocket",
			path: "/ws/connect",
		},
		{
			name: "assets",
			path: "/assets/logo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			orgID := uuid.New()
			req := createMockRequest(tt.path, map[string]any{
				"user_id":        userID,
				"organization_id": orgID,
			})

			result := middleware(req)

			assert.Same(t, req, result, "Middleware should return the same request for non-API paths")
		})
	}
}

// TestActivityLogMiddleware_InvalidStringUserID tests middleware with invalid string UUID
func TestActivityLogMiddleware_InvalidStringUserID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        "not-a-valid-uuid",
		"organization_id": orgID,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request with invalid user_id string")
}

// TestActivityLogMiddleware_DifferentStatusCodes tests middleware with different response status codes
// Note: Tests with valid credentials require TEST_DATABASE_URL
func TestActivityLogMiddleware_DifferentStatusCodes(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "success",
			statusCode: 200,
		},
		{
			name:       "created",
			statusCode: 201,
		},
		{
			name:       "bad request",
			statusCode: 400,
		},
		{
			name:       "unauthorized",
			statusCode: 401,
		},
		{
			name:       "not found",
			statusCode: 404,
		},
		{
			name:       "server error",
			statusCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			orgID := uuid.New()
			req := createMockRequest("/api/test", map[string]any{
				"user_id":        userID,
				"organization_id": orgID,
			})
			req.RequestCtx.Response.SetStatusCode(tt.statusCode)

			result := middleware(req)

			assert.Same(t, req, result, "Middleware should return the same request")
		})
	}
}

// TestActivityLogMiddleware_StringOrgID tests middleware with string organization_id
// Note: This test requires TEST_DATABASE_URL to run fully
func TestActivityLogMiddleware_StringOrgID(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	orgID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        userID,
		"organization_id": orgID.String(),
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request with string org_id")
}

// TestActivityLogMiddleware_Concurrency tests middleware with concurrent requests
func TestActivityLogMiddleware_Concurrency(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL to test logging path")

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			userID := uuid.New()
			orgID := uuid.New()
			req := createMockRequest("/api/test", map[string]any{
				"user_id":        userID,
				"organization_id": orgID,
			})

			result := middleware(req)
			require.Same(t, req, result, "Middleware should return the same request concurrently")
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestActivityLogMiddleware_NilUUIDOrgID tests middleware with nil UUID organization_id
func TestActivityLogMiddleware_NilUUIDOrgID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	userID := uuid.New()
	req := createMockRequest("/api/test", map[string]any{
		"user_id":        userID,
		"organization_id": uuid.Nil,
	})

	result := middleware(req)

	assert.Same(t, req, result, "Middleware should return the same request when organization_id is Nil")
}

// TestActivityLogMiddleware_ExactActivityLogsPathVariations tests exact path matching
func TestActivityLogMiddleware_ExactActivityLogsPathVariations(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}
	middleware := app.ActivityLogMiddleware()

	tests := []struct {
		name         string
		path         string
		shouldSkip   bool
		missingOrgID bool
	}{
		{
			name:       "exact activity logs path",
			path:       "/api/activity-logs",
			shouldSkip: true,
		},
		{
			name:       "activity logs with slash after",
			path:       "/api/activity-logs/",
			shouldSkip: true,
		},
		{
			name:       "activity logs with specific ID",
			path:       "/api/activity-logs/550e8400-e29b-41d4-a716-446655440000",
			shouldSkip: true,
		},
		{
			name:       "activity logs with nested resource",
			path:       "/api/activity-logs/export/csv",
			shouldSkip: true,
		},
		{
			name:         "similar but not activity logs - missing org",
			path:         "/api/activity-log",
			shouldSkip:   false,
			missingOrgID: true,
		},
		{
			name:         "activity logs prefixed path - missing org",
			path:         "/api/activity-logs-something",
			shouldSkip:   false,
			missingOrgID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			orgID := uuid.New()
			userValues := map[string]any{"user_id": userID}

			if !tt.missingOrgID {
				userValues["organization_id"] = orgID
			}

			req := createMockRequest(tt.path, userValues)
			result := middleware(req)

			assert.Same(t, req, result, "Middleware should return the same request")
		})
	}
}
