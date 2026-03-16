package handlers_test

import (
	"net"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// createFastGlueRequest creates a mock fastglue.Request for testing
func createFastGlueRequest(method, path, userAgent string, headers map[string]string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI(path)
	ctx.Request.Header.SetMethod(method)

	if userAgent != "" {
		ctx.Request.Header.Set("User-Agent", userAgent)
	}

	for key, value := range headers {
		ctx.Request.Header.Set(key, value)
	}

	return &fastglue.Request{RequestCtx: ctx}
}

// TestRequestPath tests the requestPath helper function
func TestRequestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "api path",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "nested path",
			path:     "/api/organizations/123/users",
			expected: "/api/organizations/123/users",
		},
		{
			name:     "path with query params",
			path:     "/api/messages?status=active",
			expected: "/api/messages", // fasthttp.Path() doesn't include query params
		},
		{
			name:     "empty path",
			path:     "",
			expected: "/", // fasthttp normalizes empty path to "/"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("GET", tt.path, "", nil)
			result := handlers.RequestPath(req)

			assert.Equal(t, tt.expected, result, "RequestPath should return the correct path")
		})
	}
}

// TestRequestPath_NilRequest tests requestPath with nil request
func TestRequestPath_NilRequest(t *testing.T) {
	t.Parallel()

	result := handlers.RequestPath(nil)
	assert.Equal(t, "", result, "RequestPath should return empty string for nil request")
}

// TestRequestMethod tests the requestMethod helper function
func TestRequestMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
	}{
		{name: "GET", method: "GET"},
		{name: "POST", method: "POST"},
		{name: "PUT", method: "PUT"},
		{name: "DELETE", method: "DELETE"},
		{name: "PATCH", method: "PATCH"},
		{name: "HEAD", method: "HEAD"},
		{name: "OPTIONS", method: "OPTIONS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest(tt.method, "/test", "", nil)
			result := handlers.RequestMethod(req)

			assert.Equal(t, tt.method, result, "RequestMethod should return the correct method")
		})
	}
}

// TestRequestMethod_NilRequest tests requestMethod with nil request
func TestRequestMethod_NilRequest(t *testing.T) {
	t.Parallel()

	result := handlers.RequestMethod(nil)
	assert.Equal(t, "", result, "RequestMethod should return empty string for nil request")
}

// TestRequestUserAgent tests the requestUserAgent helper function
func TestRequestUserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "chrome browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		},
		{
			name:      "firefox browser",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
			expected:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  "",
		},
		{
			name:      "mobile user agent",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			expected:  "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("GET", "/test", tt.userAgent, nil)
			result := handlers.RequestUserAgent(req)

			assert.Equal(t, tt.expected, result, "RequestUserAgent should return the correct user agent")
		})
	}
}

// TestRequestUserAgent_NilRequest tests requestUserAgent with nil request
func TestRequestUserAgent_NilRequest(t *testing.T) {
	t.Parallel()

	result := handlers.RequestUserAgent(nil)
	assert.Equal(t, "", result, "RequestUserAgent should return empty string for nil request")
}

// TestRequestClientIP_NoTrustProxy tests requestClientIP without trust proxy
func TestRequestClientIP_NoTrustProxy(t *testing.T) {
	t.Parallel()

	req := createFastGlueRequest("GET", "/test", "", nil)
	addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.100:12345")
	req.RequestCtx.SetRemoteAddr(addr)

	result := handlers.RequestClientIP(req, false)

	assert.Equal(t, "192.168.1.100", result, "RequestClientIP should return the remote IP without trust proxy")
}

// TestRequestClientIP_TrustProxy_XForwardedFor tests requestClientIP with X-Forwarded-For
func TestRequestClientIP_TrustProxy_XForwardedFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		xForwardedFor string
		expectedIP    string
		trustProxy    bool
	}{
		{
			name:          "single IP",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "203.0.113.1",
			trustProxy:    true,
		},
		{
			name:          "multiple IPs - should use first",
			xForwardedFor: "203.0.113.1, 198.51.100.1",
			expectedIP:    "203.0.113.1",
			trustProxy:    true,
		},
		{
			name:          "IP with whitespace",
			xForwardedFor: " 203.0.113.1 ",
			expectedIP:    "203.0.113.1",
			trustProxy:    true,
		},
		{
			name:          "empty X-Forwarded-For",
			xForwardedFor: "",
			expectedIP:    "192.168.1.100",
			trustProxy:    true,
		},
		{
			name:          "X-Forwarded-For ignored when trust proxy is false",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "192.168.1.100",
			trustProxy:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("GET", "/test", "", map[string]string{"X-Forwarded-For": tt.xForwardedFor})
			addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.100:12345")
			req.RequestCtx.SetRemoteAddr(addr)

			result := handlers.RequestClientIP(req, tt.trustProxy)

			assert.Equal(t, tt.expectedIP, result, "RequestClientIP should return the correct IP")
		})
	}
}

// TestRequestClientIP_TrustProxy_XRealIP tests requestClientIP with X-Real-IP
func TestRequestClientIP_TrustProxy_XRealIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		xRealIP    string
		expectedIP string
		trustProxy bool
	}{
		{
			name:       "X-Real-IP present",
			xRealIP:    "203.0.113.50",
			expectedIP: "203.0.113.50",
			trustProxy: true,
		},
		{
			name:       "X-Real-IP with whitespace",
			xRealIP:    " 203.0.113.50 ",
			expectedIP: "203.0.113.50",
			trustProxy: true,
		},
		{
			name:       "empty X-Real-IP",
			xRealIP:    "",
			expectedIP: "192.168.1.100",
			trustProxy: true,
		},
		{
			name:       "X-Real-IP ignored when trust proxy is false",
			xRealIP:    "203.0.113.50",
			expectedIP: "192.168.1.100",
			trustProxy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("GET", "/test", "", map[string]string{"X-Real-IP": tt.xRealIP})
			addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.100:12345")
			req.RequestCtx.SetRemoteAddr(addr)

			result := handlers.RequestClientIP(req, tt.trustProxy)

			assert.Equal(t, tt.expectedIP, result, "RequestClientIP should return the correct IP")
		})
	}
}

// TestRequestClientIP_BothHeaders tests requestClientIP with both X-Forwarded-For and X-Real-IP
func TestRequestClientIP_BothHeaders(t *testing.T) {
	t.Parallel()

	req := createFastGlueRequest("GET", "/test", "", map[string]string{
		"X-Forwarded-For": "203.0.113.1",
		"X-Real-IP":       "203.0.113.50",
	})
	addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.100:12345")
	req.RequestCtx.SetRemoteAddr(addr)

	result := handlers.RequestClientIP(req, true)

	// X-Forwarded-For takes precedence
	assert.Equal(t, "203.0.113.1", result, "RequestClientIP should prefer X-Forwarded-For over X-Real-IP")
}

// TestRequestClientIP_NilRequest tests requestClientIP with nil request
func TestRequestClientIP_NilRequest(t *testing.T) {
	t.Parallel()

	result := handlers.RequestClientIP(nil, false)
	assert.Equal(t, "", result, "RequestClientIP should return empty string for nil request")
}

// TestNormalizeActivityText tests the normalizeActivityText helper function
func TestNormalizeActivityText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			limit:    0,
			expected: "Hello World",
		},
		{
			name:     "extra whitespace",
			input:    "Hello    World   Test",
			limit:    0,
			expected: "Hello World Test",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "   Hello World   ",
			limit:    0,
			expected: "Hello World",
		},
		{
			name:     "newlines and tabs",
			input:    "Hello\n\tWorld\t\nTest",
			limit:    0,
			expected: "Hello World Test",
		},
		{
			name:     "empty string",
			input:    "",
			limit:    0,
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			limit:    0,
			expected: "",
		},
		{
			name:     "with limit - not exceeded",
			input:    "Hello World",
			limit:    20,
			expected: "Hello World",
		},
		{
			name:     "with limit - exceeded",
			input:    "Hello World Test",
			limit:    10,
			expected: "Hello Worl...",
		},
		{
			name:     "with limit - exact length",
			input:    "Hello",
			limit:    5,
			expected: "Hello",
		},
		{
			name:     "negative limit",
			input:    "Hello World",
			limit:    -1,
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.NormalizeActivityText(tt.input, tt.limit)

			assert.Equal(t, tt.expected, result, "NormalizeActivityText should return the correct normalized text")
		})
	}
}

// TestTrustProxyEnabled tests the trustProxyEnabled method
func TestTrustProxyEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *config.Config
		appNil   bool
		expected bool
	}{
		{
			name:     "nil app",
			appNil:   true,
			expected: false,
		},
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name: "trust proxy enabled",
			config: &config.Config{
				RateLimit: config.RateLimitConfig{
					TrustProxy: true,
				},
			},
			expected: true,
		},
		{
			name: "trust proxy disabled",
			config: &config.Config{
				RateLimit: config.RateLimitConfig{
					TrustProxy: false,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var app *handlers.App
			if !tt.appNil {
				app = &handlers.App{Config: tt.config}
			}

			result := app.TrustProxyEnabled()

			assert.Equal(t, tt.expected, result, "TrustProxyEnabled should return the correct value")
		})
	}
}

// TestInsertActivity_NilEntry tests insertActivity with nil entry
func TestInsertActivity_NilEntry(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL")

	app := &handlers.App{
		Config: &config.Config{},
	}

	err := app.InsertActivity(nil)

	assert.Error(t, err, "InsertActivity should return error for nil entry")
	assert.Contains(t, err.Error(), "nil", "Error should mention nil entry")
}

// TestResolveActivityActorName_NilUUID tests resolveActivityActorName with nil UUID
func TestResolveActivityActorName_NilUUID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	result := app.ResolveActivityActorName(uuid.Nil)

	assert.Equal(t, "", result, "ResolveActivityActorName should return empty string for nil UUID")
}

// TestLogAuthSuccess_Success tests LogAuthSuccess with valid user
func TestLogAuthSuccess_Success(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Email:          "test@example.com",
	}

	req := createFastGlueRequest("POST", "/api/auth/login", "Mozilla/5.0", nil)

	// Create a mock app with minimal setup
	app := &handlers.App{
		Config: &config.Config{},
		Log:    createTestLogger(),
	}

	// This should not panic
	app.LogAuthSuccess(req, user)
}

// TestLogAuthSuccess_NilUser tests LogAuthSuccess with nil user
func TestLogAuthSuccess_NilUser(t *testing.T) {
	t.Parallel()

	req := createFastGlueRequest("POST", "/api/auth/login", "Mozilla/5.0", nil)
	app := &handlers.App{
		Config: &config.Config{},
		Log:    createTestLogger(),
	}

	// This should not panic with nil user
	app.LogAuthSuccess(req, nil)
}

// TestLogAuthFailure_BasicTests tests LogAuthFailure with various scenarios
func TestLogAuthFailure_BasicTests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()

	tests := []struct {
		name   string
		email  string
		userID *uuid.UUID
		orgID  *uuid.UUID
		reason string
	}{
		{
			name:   "with all parameters",
			email:  "test@example.com",
			userID: &userID,
			orgID:  &orgID,
			reason: "invalid_password",
		},
		{
			name:   "with nil userID",
			email:  "unknown@example.com",
			userID: nil,
			orgID:  &orgID,
			reason: "user_not_found",
		},
		{
			name:   "with nil orgID",
			email:  "test@example.com",
			userID: &userID,
			orgID:  nil,
			reason: "org_not_found",
		},
		{
			name:   "minimal parameters",
			email:  "",
			userID: nil,
			orgID:  nil,
			reason: "missing_credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("POST", "/api/auth/login", "Mozilla/5.0", nil)
			app := &handlers.App{
				Config: &config.Config{},
				Log:    createTestLogger(),
			}

			// This should not panic
			app.LogAuthFailure(req, tt.email, tt.userID, tt.orgID, tt.reason)
		})
	}
}

// TestLogLogout_BasicTests tests LogLogout with various scenarios
func TestLogLogout_BasicTests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()

	tests := []struct {
		name   string
		userID *uuid.UUID
		orgID  *uuid.UUID
	}{
		{
			name:   "with both IDs",
			userID: &userID,
			orgID:  &orgID,
		},
		{
			name:   "with nil userID",
			userID: nil,
			orgID:  &orgID,
		},
		{
			name:   "with nil orgID",
			userID: &userID,
			orgID:  nil,
		},
		{
			name:   "with both nil",
			userID: nil,
			orgID:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("POST", "/api/auth/logout", "Mozilla/5.0", nil)
			app := &handlers.App{
				Config: &config.Config{},
				Log:    createTestLogger(),
			}

			// This should not panic
			app.LogLogout(req, tt.userID, tt.orgID)
		})
	}
}

// TestLogConversationResponse_BasicTests tests LogConversationResponse
func TestLogConversationResponse_BasicTests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	contactID := uuid.New()
	messageID := uuid.New()

	tests := []struct {
		name           string
		userID         uuid.UUID
		orgID          uuid.UUID
		contactID      uuid.UUID
		messageID      uuid.UUID
		messageType    models.MessageType
		messageContent string
		chatName       string
		chatPhone      string
	}{
		{
			name:           "with all parameters",
			userID:         userID,
			orgID:          orgID,
			contactID:      contactID,
			messageID:      messageID,
			messageType:    models.MessageTypeText,
			messageContent: "Hello, world!",
			chatName:       "John Doe",
			chatPhone:      "+1234567890",
		},
		{
			name:           "with minimal parameters",
			userID:         userID,
			orgID:          orgID,
			contactID:      contactID,
			messageID:      messageID,
			messageType:    models.MessageTypeAudio,
			messageContent: "",
			chatName:       "",
			chatPhone:      "",
		},
		{
			name:           "with long content",
			userID:         userID,
			orgID:          orgID,
			contactID:      contactID,
			messageID:      messageID,
			messageType:    models.MessageTypeText,
			messageContent: string(make([]byte, 300)), // Long content
			chatName:       "Test User",
			chatPhone:      "+9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &handlers.App{
				Config: &config.Config{},
				Log:    createTestLogger(),
			}

			// This should not panic
			app.LogConversationResponse(
				tt.userID, tt.orgID, tt.contactID, tt.messageID,
				tt.messageType, tt.messageContent, tt.chatName, tt.chatPhone,
			)
		})
	}
}

// TestLogSystemInteraction_BasicTests tests LogSystemInteraction
func TestLogSystemInteraction_BasicTests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "success status", statusCode: 200},
		{name: "redirect status", statusCode: 301},
		{name: "client error", statusCode: 404},
		{name: "server error", statusCode: 500},
		{name: "success with 201", statusCode: 201},
		{name: "success with 204", statusCode: 204},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("GET", "/api/data", "Mozilla/5.0", nil)
			app := &handlers.App{
				Config: &config.Config{},
				Log:    createTestLogger(),
			}

			// This should not panic
			app.LogSystemInteraction(req, userID, orgID, tt.statusCode)
		})
	}
}

// TestLogCustomEvent_BasicTests tests LogCustomEvent
func TestLogCustomEvent_BasicTests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	contactID := uuid.New()
	messageID := uuid.New()

	tests := []struct {
		name      string
		category  string
		eventType string
		action    string
		contactID *uuid.UUID
		messageID *uuid.UUID
		metadata  models.JSONB
	}{
		{
			name:      "with all parameters",
			category:  "custom",
			eventType: "custom.event",
			action:    "custom_action",
			contactID: &contactID,
			messageID: &messageID,
			metadata:  models.JSONB{"key": "value"},
		},
		{
			name:      "with minimal parameters",
			category:  "test",
			eventType: "test.event",
			action:    "test_action",
			contactID: nil,
			messageID: nil,
			metadata:  models.JSONB{},
		},
		{
			name:      "with complex metadata",
			category:  "analytics",
			eventType: "analytics.track",
			action:    "page_view",
			contactID: &contactID,
			messageID: nil,
			metadata: models.JSONB{
				"page":     "/dashboard",
				"duration": 1234,
				"features": []string{"feature1", "feature2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := createFastGlueRequest("POST", "/api/custom", "Mozilla/5.0", nil)
			app := &handlers.App{
				Config: &config.Config{},
				Log:    createTestLogger(),
			}

			// This should not panic
			_, _ = app.LogCustomEvent(
				req, userID, orgID,
				tt.category, tt.eventType, tt.action,
				tt.contactID, tt.messageID,
				tt.metadata,
			)
		})
	}
}

// createTestLogger creates a minimal logger for testing
func createTestLogger() logf.Logger {
	return logf.New(logf.Opts{
		Level:        logf.DebugLevel,
		EnableCaller: false,
	})
}

// TestListOwnEvents_BasicTests tests ListOwnEvents with database
func TestListOwnEvents_BasicTests(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL")

	db := testutil.SetupTestDB(t)
	userID := uuid.New()
	orgID := uuid.New()

	app := &handlers.App{
		Config: &config.Config{},
		DB:     db,
		Log:    createTestLogger(),
	}

	// Create some test activity logs
	now := time.Now()
	testLogs := []models.ActivityLog{
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-2 * time.Hour),
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "auth",
			EventType:      "auth.login",
			Action:         "login",
			Status:         "success",
			Source:         "auth",
		},
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-1 * time.Hour),
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "system",
			EventType:      "system.api_interaction",
			Action:         "api_request",
			Status:         "success",
			Source:         "system",
		},
		{
			BaseModel: models.BaseModel{
				CreatedAt: now,
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "engagement",
			EventType:      "engagement.conversation_response",
			Action:         "send_message",
			Status:         "success",
			Source:         "engagement",
		},
	}

	for _, log := range testLogs {
		err := app.InsertActivity(&log)
		assert.NoError(t, err, "Should be able to insert test logs")
	}

	tests := []struct {
		name     string
		userID   uuid.UUID
		orgID    uuid.UUID
		filter   handlers.ActivityListFilter
		expected int
	}{
		{
			name:     "all events",
			userID:   userID,
			orgID:    orgID,
			filter:   handlers.ActivityListFilter{Pagination: handlers.Pagination{Limit: 10}},
			expected: 3,
		},
		{
			name:   "filter by category",
			userID: userID,
			orgID:  orgID,
			filter: handlers.ActivityListFilter{
				Pagination: handlers.Pagination{Limit: 10},
				Category:   "auth",
			},
			expected: 1,
		},
		{
			name:   "filter by event type",
			userID: userID,
			orgID:  orgID,
			filter: handlers.ActivityListFilter{
				Pagination: handlers.Pagination{Limit: 10},
				EventType:  "auth.login",
			},
			expected: 1,
		},
		{
			name:   "filter by source",
			userID: userID,
			orgID:  orgID,
			filter: handlers.ActivityListFilter{
				Pagination: handlers.Pagination{Limit: 10},
				Source:     "system",
			},
			expected: 1,
		},
		{
			name:   "filter by status",
			userID: userID,
			orgID:  orgID,
			filter: handlers.ActivityListFilter{
				Pagination: handlers.Pagination{Limit: 10},
				Status:     "success",
			},
			expected: 3,
		},
		{
			name:   "filter by date range",
			userID: userID,
			orgID:  orgID,
			filter: handlers.ActivityListFilter{
				Pagination: handlers.Pagination{Limit: 10},
				StartDate:  &[]time.Time{now.Add(-3 * time.Hour)}[0],
				EndDate:    &[]time.Time{now.Add(-30 * time.Minute)}[0],
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, total, err := app.ListOwnEvents(tt.userID, tt.orgID, tt.filter)

			assert.NoError(t, err, "ListOwnEvents should not return error")
			assert.Equal(t, int64(tt.expected), total, "Should return correct total count")
			assert.Equal(t, tt.expected, len(logs), "Should return correct number of logs")
		})
	}
}

// TestPurgeOlderThan_BasicTests tests PurgeOlderThan with database
func TestPurgeOlderThan_BasicTests(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL")

	db := testutil.SetupTestDB(t)
	userID := uuid.New()
	orgID := uuid.New()

	app := &handlers.App{
		Config: &config.Config{},
		DB:     db,
		Log:    createTestLogger(),
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	// Create some test activity logs
	testLogs := []models.ActivityLog{
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-48 * time.Hour), // Old, should be purged
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "auth",
			EventType:      "auth.login",
			Action:         "login",
			Status:         "success",
		},
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-12 * time.Hour), // New, should remain
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "system",
			EventType:      "system.api_interaction",
			Action:         "api_request",
			Status:         "success",
		},
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-36 * time.Hour), // Old, should be purged
			},
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       "engagement",
			EventType:      "engagement.conversation_response",
			Action:         "send_message",
			Status:         "success",
		},
	}

	for _, log := range testLogs {
		err := app.InsertActivity(&log)
		assert.NoError(t, err, "Should be able to insert test logs")
	}

	// Count before purge
	var countBefore int64
	db.Model(&models.ActivityLog{}).Count(&countBefore)
	assert.Equal(t, int64(3), countBefore, "Should have 3 logs before purge")

	// Purge logs older than cutoff
	rowsAffected, err := app.PurgeOlderThan(cutoff)
	assert.NoError(t, err, "PurgeOlderThan should not return error")
	assert.Equal(t, int64(2), rowsAffected, "Should purge 2 logs")

	// Count after purge
	var countAfter int64
	db.Model(&models.ActivityLog{}).Count(&countAfter)
	assert.Equal(t, int64(1), countAfter, "Should have 1 log after purge")
}

// TestPurgeOlderThan_NoLogs tests PurgeOlderThan when no logs exist
func TestPurgeOlderThan_NoLogs(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL")

	db := testutil.SetupTestDB(t)
	app := &handlers.App{
		Config: &config.Config{},
		DB:     db,
		Log:    createTestLogger(),
	}

	now := time.Now()
	rowsAffected, err := app.PurgeOlderThan(now)

	assert.NoError(t, err, "PurgeOlderThan should not return error")
	assert.Equal(t, int64(0), rowsAffected, "Should purge 0 logs when none exist")
}
