package middleware_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testJWTSecret = "unit-test-signing-value-1234567890"

// newTestRequest creates a fastglue request for testing.
func newTestRequest() *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	return &fastglue.Request{RequestCtx: ctx}
}

// generateTestToken creates a valid JWT token for testing.
func generateTestToken(t *testing.T, userID, orgID uuid.UUID, email string, roleID *uuid.UUID, expiry time.Duration) string {
	t.Helper()

	claims := middleware.JWTClaims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		RoleID:         roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return tokenString
}

type tenantScopedRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `gorm:"type:uuid"`
}

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return db, mock
}

func TestCORS(t *testing.T) {
	t.Parallel()

	allowlist := middleware.ParseAllowedOrigins("https://example.com/")

	tests := []struct {
		name           string
		host           string
		origin         string
		allowedOrigins map[string]bool
		wantOrigin     string
		wantCreds      string
	}{
		{
			name:           "explicitly allowed origin",
			host:           "api.example.com",
			origin:         "https://example.com",
			allowedOrigins: allowlist,
			wantOrigin:     "https://example.com",
			wantCreds:      "true",
		},
		{
			name:           "explicit allowlist blocks unlisted origin",
			host:           "api.example.com",
			origin:         "https://evil.example",
			allowedOrigins: allowlist,
			wantOrigin:     "",
			wantCreds:      "",
		},
		{
			name:           "empty allowlist allows same origin",
			host:           "app.example.com",
			origin:         "http://app.example.com",
			allowedOrigins: nil,
			wantOrigin:     "http://app.example.com",
			wantCreds:      "true",
		},
		{
			name:           "empty allowlist allows localhost development origin",
			host:           "api.example.com",
			origin:         "http://localhost:3000",
			allowedOrigins: nil,
			wantOrigin:     "http://localhost:3000",
			wantCreds:      "true",
		},
		{
			name:           "empty allowlist blocks non-loopback cross origin",
			host:           "api.example.com",
			origin:         "https://attacker.example",
			allowedOrigins: nil,
			wantOrigin:     "",
			wantCreds:      "",
		},
		{
			name:           "without origin header",
			host:           "api.example.com",
			origin:         "",
			allowedOrigins: nil,
			wantOrigin:     "",
			wantCreds:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := newTestRequest()
			req.RequestCtx.Request.SetHost(tt.host)
			if tt.origin != "" {
				req.RequestCtx.Request.Header.Set("Origin", tt.origin)
			}

			corsMiddleware := middleware.CORS(tt.allowedOrigins)
			result := corsMiddleware(req)

			require.NotNil(t, result, "CORS middleware should return request")

			// Check CORS headers
			assert.Equal(t, tt.wantOrigin, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Origin")))
			assert.Equal(t, tt.wantCreds, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Credentials")))
			if tt.wantOrigin != "" {
				assert.Contains(t, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Methods")), "GET")
				assert.Contains(t, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Methods")), "POST")
				assert.Contains(t, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Headers")), "Authorization")
				assert.Contains(t, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Headers")), "X-API-Key")
				assert.Contains(t, string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Headers")), "X-Organization-ID")
				assert.Equal(t, "Origin", string(result.RequestCtx.Response.Header.Peek("Vary")))
			} else {
				assert.Equal(t, "", string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Methods")))
				assert.Equal(t, "", string(result.RequestCtx.Response.Header.Peek("Access-Control-Allow-Headers")))
			}
		})
	}
}

func TestParseAllowedOrigins_NormalizesAndSkipsInvalid(t *testing.T) {
	t.Parallel()

	origins := middleware.ParseAllowedOrigins(" https://EXAMPLE.com/ , https://example.com:443, http://localhost:3000, not-an-origin, javascript:alert(1) ")

	assert.Len(t, origins, 2)
	assert.True(t, origins["https://example.com"])
	assert.True(t, origins["http://localhost:3000"])
}

func TestIsOriginAllowedForRequest(t *testing.T) {
	t.Parallel()

	allowlist := middleware.ParseAllowedOrigins("https://app.example.com")

	tests := []struct {
		name           string
		origin         string
		allowedOrigins map[string]bool
		requestHost    string
		requestTLS     bool
		wantAllowed    bool
	}{
		{
			name:           "empty origin allowed for non-browser clients",
			origin:         "",
			allowedOrigins: nil,
			requestHost:    "api.example.com",
			requestTLS:     false,
			wantAllowed:    true,
		},
		{
			name:           "empty origin denied when allowlist is configured",
			origin:         "",
			allowedOrigins: allowlist,
			requestHost:    "api.example.com",
			requestTLS:     false,
			wantAllowed:    false,
		},
		{
			name:           "allowlist match allowed",
			origin:         "https://app.example.com",
			allowedOrigins: allowlist,
			requestHost:    "api.example.com",
			requestTLS:     true,
			wantAllowed:    true,
		},
		{
			name:           "allowlist mismatch denied",
			origin:         "https://evil.example",
			allowedOrigins: allowlist,
			requestHost:    "api.example.com",
			requestTLS:     true,
			wantAllowed:    false,
		},
		{
			name:           "fallback same origin allowed",
			origin:         "https://api.example.com",
			allowedOrigins: nil,
			requestHost:    "api.example.com",
			requestTLS:     true,
			wantAllowed:    true,
		},
		{
			name:           "fallback localhost allowed",
			origin:         "http://localhost:5173",
			allowedOrigins: nil,
			requestHost:    "api.example.com",
			requestTLS:     false,
			wantAllowed:    true,
		},
		{
			name:           "fallback cross origin denied",
			origin:         "https://evil.example",
			allowedOrigins: nil,
			requestHost:    "api.example.com",
			requestTLS:     true,
			wantAllowed:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := middleware.IsOriginAllowedForRequest(tc.origin, tc.allowedOrigins, tc.requestHost, tc.requestTLS)
			assert.Equal(t, tc.wantAllowed, got)
		})
	}
}

func TestRecovery(t *testing.T) {
	t.Parallel()

	log := testutil.NopLogger()
	recoveryMiddleware := middleware.Recovery(log)

	t.Run("normal request passes through", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		result := recoveryMiddleware(req)

		require.NotNil(t, result, "should return request")
	})

	// Note: Testing panic recovery is tricky because the panic happens
	// after the middleware returns. The Recovery middleware is designed
	// to wrap handlers, not to be tested in isolation.
}

func TestTenantScope(t *testing.T) {
	t.Parallel()

	t.Run("stores scoped db for default organization", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockGormDB(t)
		req := newTestRequest()
		orgID := uuid.New()
		recordID := uuid.New()
		req.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, orgID)

		result := middleware.TenantScope(db)(req)
		require.NotNil(t, result)

		storedOrgID, ok := result.RequestCtx.UserValue(middleware.ContextKeyOrganizationID).(uuid.UUID)
		require.True(t, ok)
		assert.Equal(t, orgID, storedOrgID)

		scopedDB, ok := tenant.GetScopedDB(result)
		require.True(t, ok)
		require.NotNil(t, scopedDB)

		mock.ExpectQuery(`SELECT .*FROM "tenant_scoped_records".*organization_id`).
			WithArgs(recordID, orgID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id"}))

		var records []tenantScopedRecord
		err := scopedDB.Model(&tenantScopedRecord{}).
			Where("id = ?", recordID).
			Find(&records).Error
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fresh scoped db sessions do not leak joins between sequential queries", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockGormDB(t)
		orgID := uuid.New()
		recordID := uuid.New()

		scopedDB := tenant.ScopedDB(db, orgID)

		mock.ExpectQuery(`SELECT .* FROM "tenant_scoped_records" LEFT JOIN organizations ON organizations.id = tenant_scoped_records.organization_id .*organization_id`).
			WithArgs(recordID, orgID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id"}))

		var records []tenantScopedRecord
		err := scopedDB.Session(&gorm.Session{}).
			Table("tenant_scoped_records").
			Joins("LEFT JOIN organizations ON organizations.id = tenant_scoped_records.organization_id").
			Where("id = ?", recordID).
			Find(&records).Error
		require.NoError(t, err)

		mock.ExpectQuery(`SELECT .* FROM "tenant_scoped_records" WHERE id = .*organization_id`).
			WithArgs(recordID, orgID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id"}))

		err = scopedDB.Session(&gorm.Session{}).
			Model(&tenantScopedRecord{}).
			Where("id = ?", recordID).
			Find(&records).Error
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("honors membership-based organization override", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockGormDB(t)
		req := newTestRequest()
		defaultOrgID := uuid.New()
		overrideOrgID := uuid.New()
		userID := uuid.New()

		req.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, defaultOrgID)
		req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, userID)
		req.RequestCtx.Request.Header.Set("X-Organization-ID", overrideOrgID.String())

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_organizations"`).
			WithArgs(userID, overrideOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		result := middleware.TenantScope(db)(req)
		require.NotNil(t, result)

		storedOrgID, ok := result.RequestCtx.UserValue(middleware.ContextKeyOrganizationID).(uuid.UUID)
		require.True(t, ok)
		assert.Equal(t, overrideOrgID, storedOrgID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("honors super-admin organization override", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockGormDB(t)
		req := newTestRequest()
		defaultOrgID := uuid.New()
		overrideOrgID := uuid.New()

		req.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, defaultOrgID)
		req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, uuid.New())
		req.RequestCtx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
		req.RequestCtx.Request.Header.Set("X-Organization-ID", overrideOrgID.String())

		mock.ExpectQuery(`SELECT count\(\*\) FROM "organizations"`).
			WithArgs(overrideOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		result := middleware.TenantScope(db)(req)
		require.NotNil(t, result)

		storedOrgID, ok := result.RequestCtx.UserValue(middleware.ContextKeyOrganizationID).(uuid.UUID)
		require.True(t, ok)
		assert.Equal(t, overrideOrgID, storedOrgID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	t.Run("non-API path gets CSP", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		req.RequestCtx.Request.SetRequestURI("/dashboard")
		securityHeadersMiddleware := middleware.SecurityHeaders(false)
		result := securityHeadersMiddleware(req)

		require.NotNil(t, result, "SecurityHeaders middleware should return request")

		headers := &result.RequestCtx.Response.Header
		assert.Equal(t, "nosniff", string(headers.Peek("X-Content-Type-Options")))
		assert.Equal(t, "DENY", string(headers.Peek("X-Frame-Options")))
		assert.Equal(t, "strict-origin-when-cross-origin", string(headers.Peek("Referrer-Policy")))
		assert.Equal(t, "camera=(), microphone=(), geolocation=()", string(headers.Peek("Permissions-Policy")))
		assert.Equal(t, "0", string(headers.Peek("X-XSS-Protection")))

		csp := string(headers.Peek("Content-Security-Policy"))
		require.NotEmpty(t, csp, "CSP header must be set for non-API paths")
		assert.Contains(t, csp, "default-src 'self'")
		assert.Contains(t, csp, "object-src 'none'")
		assert.Contains(t, csp, "frame-ancestors 'none'")
		assert.Contains(t, csp, "frame-src 'self' data: blob: https:")
		assert.Contains(t, csp, "script-src 'self'")
		assert.Contains(t, csp, "img-src 'self' data: blob: https:")
		assert.Contains(t, csp, "media-src 'self' data: blob: https:")
		assert.Contains(t, csp, "connect-src 'self' ws: wss: blob:")
	})

	t.Run("API path skips CSP", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		req.RequestCtx.Request.SetRequestURI("/api/health")
		securityHeadersMiddleware := middleware.SecurityHeaders(false)
		result := securityHeadersMiddleware(req)

		require.NotNil(t, result)

		headers := &result.RequestCtx.Response.Header
		assert.Equal(t, "", string(headers.Peek("Content-Security-Policy")))
		assert.Equal(t, "nosniff", string(headers.Peek("X-Content-Type-Options")))
	})

	t.Run("production adds HSTS", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		req.RequestCtx.Request.SetRequestURI("/dashboard")
		securityHeadersMiddleware := middleware.SecurityHeaders(true)
		result := securityHeadersMiddleware(req)

		require.NotNil(t, result)

		headers := &result.RequestCtx.Response.Header
		assert.Equal(t, "max-age=31536000; includeSubDomains", string(headers.Peek("Strict-Transport-Security")))
	})

	t.Run("non-production no HSTS", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		req.RequestCtx.Request.SetRequestURI("/dashboard")
		securityHeadersMiddleware := middleware.SecurityHeaders(false)
		result := securityHeadersMiddleware(req)

		require.NotNil(t, result)

		headers := &result.RequestCtx.Response.Header
		assert.Equal(t, "", string(headers.Peek("Strict-Transport-Security")))
	})
}

func TestAuth_ValidJWT(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	email := "test@example.com"
	roleID := uuid.New()

	token := generateTestToken(t, userID, orgID, email, &roleID, time.Hour)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)

	authMiddleware := middleware.Auth(testJWTSecret)
	result := authMiddleware(req)

	require.NotNil(t, result, "should return request for valid token")

	// Verify context values were set
	gotUserID, ok := result.RequestCtx.UserValue(middleware.ContextKeyUserID).(uuid.UUID)
	require.True(t, ok, "user_id should be uuid.UUID")
	assert.Equal(t, userID, gotUserID)

	gotOrgID, ok := result.RequestCtx.UserValue(middleware.ContextKeyOrganizationID).(uuid.UUID)
	require.True(t, ok, "organization_id should be uuid.UUID")
	assert.Equal(t, orgID, gotOrgID)

	gotEmail, ok := result.RequestCtx.UserValue(middleware.ContextKeyEmail).(string)
	require.True(t, ok, "email should be string")
	assert.Equal(t, email, gotEmail)

	gotRoleID, ok := result.RequestCtx.UserValue(middleware.ContextKeyRoleID).(uuid.UUID)
	require.True(t, ok, "role_id should be uuid.UUID")
	assert.Equal(t, roleID, gotRoleID)
}

func TestAuth_ExpiredJWT(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	roleID := uuid.New()

	// Create an expired token
	token := generateTestToken(t, userID, orgID, "test@example.com", &roleID, -time.Hour)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)

	authMiddleware := middleware.Auth(testJWTSecret)
	result := authMiddleware(req)

	assert.Nil(t, result, "should return nil for expired token")
	assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
}

func TestAuth_InvalidJWT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing authorization header",
			header: "",
		},
		{
			name:   "invalid format - no Bearer prefix",
			header: "invalid-token",
		},
		{
			name:   "invalid format - wrong prefix",
			header: "Basic some-token",
		},
		{
			name:   "malformed token",
			header: "Bearer not.a.valid.jwt",
		},
		{
			name:   "wrong secret",
			header: "Bearer " + generateTokenWithSecret(t, "wrong-secret-key-that-is-long-enough"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := newTestRequest()
			if tt.header != "" {
				req.RequestCtx.Request.Header.Set("Authorization", tt.header)
			}

			authMiddleware := middleware.Auth(testJWTSecret)
			result := authMiddleware(req)

			assert.Nil(t, result, "should return nil for invalid token")
			assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
		})
	}
}

func TestAuth_RejectsNonAccessTokenSubjects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
	}{
		{name: "refresh subject", subject: "refresh"},
		{name: "ws subject", subject: "ws"},
		{name: "empty subject", subject: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			roleID := uuid.New()
			claims := middleware.JWTClaims{
				UserID:         uuid.New(),
				OrganizationID: uuid.New(),
				Email:          "test@example.com",
				RoleID:         &roleID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					Subject:   tt.subject,
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString([]byte(testJWTSecret))
			require.NoError(t, err)

			req := newTestRequest()
			req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+tokenString)

			authMiddleware := middleware.Auth(testJWTSecret)
			result := authMiddleware(req)

			assert.Nil(t, result, "should reject non-access JWT subject")
			assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
		})
	}
}

func TestAuth_RejectsUnexpectedJWTSigningMethod(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	claims := middleware.JWTClaims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		RoleID:         &roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenString, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+tokenString)

	authMiddleware := middleware.Auth(testJWTSecret)
	result := authMiddleware(req)

	assert.Nil(t, result, "should reject JWTs signed with unexpected algorithms")
	assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
}

func TestAuth_RejectsJWTWithNoneSigningMethod(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	claims := middleware.JWTClaims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		RoleID:         &roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+tokenString)

	authMiddleware := middleware.Auth(testJWTSecret)
	result := authMiddleware(req)

	assert.Nil(t, result, "should reject JWTs signed with the none algorithm")
	assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
}

func TestAuth_MisconfiguredSecret(t *testing.T) {
	t.Parallel()

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer some-token")

	authMiddleware := middleware.Auth("   ")
	result := authMiddleware(req)

	assert.Nil(t, result, "should return nil when JWT secret is misconfigured")
	assert.Equal(t, fasthttp.StatusInternalServerError, req.RequestCtx.Response.StatusCode())
}

func TestAuth_DifferentRoleIDs(t *testing.T) {
	t.Parallel()

	roleIDs := []*uuid.UUID{
		func() *uuid.UUID { id := uuid.New(); return &id }(),
		func() *uuid.UUID { id := uuid.New(); return &id }(),
		func() *uuid.UUID { id := uuid.New(); return &id }(),
	}

	for i, roleID := range roleIDs {
		t.Run("role_"+roleID.String()[:8], func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			orgID := uuid.New()
			token := generateTestToken(t, userID, orgID, "test@example.com", roleIDs[i], time.Hour)

			req := newTestRequest()
			req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)

			authMiddleware := middleware.Auth(testJWTSecret)
			result := authMiddleware(req)

			require.NotNil(t, result)

			gotRoleID := result.RequestCtx.UserValue(middleware.ContextKeyRoleID).(uuid.UUID)
			assert.Equal(t, *roleIDs[i], gotRoleID)
		})
	}
}

func TestAuth_NilRoleID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	token := generateTestToken(t, userID, orgID, "test@example.com", nil, time.Hour)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)

	authMiddleware := middleware.Auth(testJWTSecret)
	result := authMiddleware(req)

	require.NotNil(t, result, "should return request for valid token with nil roleID")

	// Verify roleID is not set in context when nil
	gotRoleID := result.RequestCtx.UserValue(middleware.ContextKeyRoleID)
	assert.Nil(t, gotRoleID, "role_id should not be set when nil in claims")
}

func TestRequirePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		hasPermission bool
		wantAllowed   bool
	}{
		{
			name:          "user with permission allowed",
			hasPermission: true,
			wantAllowed:   true,
		},
		{
			name:          "user without permission denied",
			hasPermission: false,
			wantAllowed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			req := newTestRequest()
			req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, userID)

			// Create a mock permission checker
			checker := func(uid uuid.UUID, resource, action string) bool {
				return tt.hasPermission
			}

			permMiddleware := middleware.RequirePermission(checker, "contacts", "read")
			result := permMiddleware(req)

			if tt.wantAllowed {
				assert.NotNil(t, result, "should allow access")
			} else {
				assert.Nil(t, result, "should deny access")
				assert.Equal(t, fasthttp.StatusForbidden, req.RequestCtx.Response.StatusCode())
			}
		})
	}
}

func TestRequirePermission_NoUserInContext(t *testing.T) {
	t.Parallel()

	req := newTestRequest()
	// Don't set any user in context

	checker := func(uid uuid.UUID, resource, action string) bool {
		return true
	}

	permMiddleware := middleware.RequirePermission(checker, "contacts", "read")
	result := permMiddleware(req)

	assert.Nil(t, result, "should deny when user not in context")
	assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
}

func TestRequireAnyPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		allowedPerms map[string]bool
		checkPerms   []string
		wantAllowed  bool
	}{
		{
			name:         "user with first permission allowed",
			allowedPerms: map[string]bool{"contacts:read": true},
			checkPerms:   []string{"contacts:read", "contacts:write"},
			wantAllowed:  true,
		},
		{
			name:         "user with second permission allowed",
			allowedPerms: map[string]bool{"contacts:write": true},
			checkPerms:   []string{"contacts:read", "contacts:write"},
			wantAllowed:  true,
		},
		{
			name:         "user without any permission denied",
			allowedPerms: map[string]bool{},
			checkPerms:   []string{"contacts:read", "contacts:write"},
			wantAllowed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			req := newTestRequest()
			req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, userID)

			// Create a mock permission checker
			checker := func(uid uuid.UUID, resource, action string) bool {
				perm := resource + ":" + action
				return tt.allowedPerms[perm]
			}

			permMiddleware := middleware.RequireAnyPermission(checker, tt.checkPerms...)
			result := permMiddleware(req)

			if tt.wantAllowed {
				assert.NotNil(t, result, "should allow access")
			} else {
				assert.Nil(t, result, "should deny access")
				assert.Equal(t, fasthttp.StatusForbidden, req.RequestCtx.Response.StatusCode())
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	t.Parallel()

	t.Run("user ID exists", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		req := newTestRequest()
		req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, expectedID)

		userID, ok := middleware.GetUserID(req)

		assert.True(t, ok)
		assert.Equal(t, expectedID, userID)
	})

	t.Run("user ID not set", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()

		_, ok := middleware.GetUserID(req)

		assert.False(t, ok)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()
		req.RequestCtx.SetUserValue(middleware.ContextKeyUserID, "not-a-uuid")

		_, ok := middleware.GetUserID(req)

		assert.False(t, ok)
	})
}

func TestGetOrganizationID(t *testing.T) {
	t.Parallel()

	t.Run("organization ID exists", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		req := newTestRequest()
		req.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, expectedID)

		orgID, ok := middleware.GetOrganizationID(req)

		assert.True(t, ok)
		assert.Equal(t, expectedID, orgID)
	})

	t.Run("organization ID not set", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest()

		_, ok := middleware.GetOrganizationID(req)

		assert.False(t, ok)
	})
}

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	log := testutil.NopLogger()
	loggerMiddleware := middleware.RequestLogger(log)

	req := newTestRequest()
	result := loggerMiddleware(req)

	require.NotNil(t, result)

	// Check that request_start was set
	startTime, ok := result.RequestCtx.UserValue("request_start").(time.Time)
	assert.True(t, ok, "request_start should be set")
	assert.WithinDuration(t, time.Now(), startTime, time.Second)
}

func TestJWTClaims(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	orgID := uuid.New()
	roleID := uuid.New()

	claims := middleware.JWTClaims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          "test@example.com",
		RoleID:         &roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	// Parse token back
	parsedToken, err := jwt.ParseWithClaims(tokenString, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, parsedToken.Valid)

	parsedClaims, ok := parsedToken.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	assert.Equal(t, userID, parsedClaims.UserID)
	assert.Equal(t, orgID, parsedClaims.OrganizationID)
	assert.Equal(t, "test@example.com", parsedClaims.Email)
	require.NotNil(t, parsedClaims.RoleID)
	assert.Equal(t, roleID, *parsedClaims.RoleID)
}

func TestAuth_MultipleMiddlewareChain(t *testing.T) {
	t.Parallel()

	// Test that Auth works correctly when chained with other middleware
	userID := uuid.New()
	orgID := uuid.New()
	roleID := uuid.New()
	token := generateTestToken(t, userID, orgID, "test@example.com", &roleID, time.Hour)

	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)
	req.RequestCtx.Request.SetHost("example.com")
	req.RequestCtx.Request.Header.Set("Origin", "http://example.com")

	// Apply CORS first
	corsMiddleware := middleware.CORS(nil)
	req = corsMiddleware(req)
	require.NotNil(t, req)

	// Then Auth
	authMiddleware := middleware.Auth(testJWTSecret)
	req = authMiddleware(req)
	require.NotNil(t, req)

	// Then RequirePermission (replaces RequireRole)
	checker := func(uid uuid.UUID, resource, action string) bool {
		return uid == userID // Allow the authenticated user
	}
	permMiddleware := middleware.RequirePermission(checker, "contacts", "read")
	req = permMiddleware(req)
	require.NotNil(t, req)

	// Verify all context values are still present
	assert.Equal(t, userID, req.RequestCtx.UserValue(middleware.ContextKeyUserID))
	assert.Equal(t, orgID, req.RequestCtx.UserValue(middleware.ContextKeyOrganizationID))
	assert.Equal(t, roleID, req.RequestCtx.UserValue(middleware.ContextKeyRoleID))

	// Verify CORS headers are still present
	assert.Equal(t, "http://example.com", string(req.RequestCtx.Response.Header.Peek("Access-Control-Allow-Origin")))
}

// generateTokenWithSecret creates a token signed with a specific secret.
func generateTokenWithSecret(t *testing.T, secret string) string {
	t.Helper()

	roleID := uuid.New()
	claims := middleware.JWTClaims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		RoleID:         &roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenString
}
