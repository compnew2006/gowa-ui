package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Context keys
const (
	ContextKeyUserID         = "user_id"
	ContextKeyOrganizationID = "organization_id"
	ContextKeyEmail          = "email"
	ContextKeyRoleID         = "role_id"
	ContextKeyIsSuperAdmin   = "is_super_admin"
	ContextKeyUser           = "user"
	ContextKeyOrganization   = "organization"
)

const defaultContentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; frame-src 'self' data: blob: https:; object-src 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; media-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss: blob:"
const accessTokenSubject = "access"

const (
	apiKeyAuthMaxFailures = 10
	apiKeyAuthWindow      = 15 * time.Minute
)

type apiKeyFailureEntry struct {
	count    int
	expiresAt time.Time
}

var apiKeyFailureLimiter sync.Map

// ContentSecurityPolicyWithNonce returns the default CSP with a script nonce injected.
func ContentSecurityPolicyWithNonce(nonce string) string {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return defaultContentSecurityPolicy
	}
	return strings.Replace(defaultContentSecurityPolicy, "script-src 'self';", "script-src 'self' 'nonce-"+nonce+"';", 1)
}

func shouldSkipCSP(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/api") {
		return true
	}
	if strings.HasPrefix(path, "/ws") {
		return true
	}
	for _, suffix := range []string{".js", ".css", ".map", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".webp", ".json"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Email          string     `json:"email"`
	RoleID         *uuid.UUID `json:"role_id,omitempty"`
	IsSuperAdmin   bool       `json:"is_super_admin"`
	jwt.RegisteredClaims
}

// RequestLogger logs incoming requests
func RequestLogger(log logf.Logger) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		start := time.Now()

		// Store start time for later use
		r.RequestCtx.SetUserValue("request_start", start)

		return r
	}
}

// ParseAllowedOrigins parses a comma-separated list of allowed origins into a set.
func ParseAllowedOrigins(allowedOrigins string) map[string]bool {
	origins := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		normalized, ok := normalizeOrigin(o)
		if ok {
			origins[normalized] = true
		}
	}
	return origins
}

// IsOriginAllowed checks if origin is in the explicit allowed set.
// If no origins are configured, fallback rules apply:
// 1. same-origin requests are allowed
// 2. localhost/loopback origins are allowed for local development
// 3. all other origins are rejected
func IsOriginAllowed(origin string, allowedOrigins map[string]bool) bool {
	return IsOriginAllowedForRequest(origin, allowedOrigins, "", false)
}

// IsOriginAllowedForRequest validates an origin against explicit allow-list and
// safe defaults based on the incoming request host.
func IsOriginAllowedForRequest(origin string, allowedOrigins map[string]bool, requestHost string, requestTLS bool) bool {
	trimmedOrigin := strings.TrimSpace(origin)
	if trimmedOrigin == "" {
		// Non-browser clients may omit Origin.
		return len(allowedOrigins) == 0
	}

	normalizedOrigin, ok := normalizeOrigin(trimmedOrigin)
	if !ok {
		return false
	}

	if len(allowedOrigins) > 0 {
		return allowedOrigins[normalizedOrigin]
	}

	originURL, err := url.Parse(normalizedOrigin)
	if err != nil {
		return false
	}
	originHost := strings.ToLower(originURL.Hostname())
	originPort := effectivePort(originURL.Scheme, originURL.Port())

	reqHost, reqPort := splitHostPort(requestHost, requestTLS)
	if reqHost != "" && reqPort != "" && originHost == reqHost && originPort == reqPort {
		return true
	}

	return isLoopbackHost(originHost)
}

func normalizeOrigin(origin string) (string, bool) {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		return "", false
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", false
	}
	port := effectivePort(scheme, u.Port())
	if port == "" {
		return "", false
	}

	return scheme + "://" + formatHostPort(host, scheme, port), true
}

func effectivePort(scheme, port string) string {
	if port != "" {
		return port
	}
	if strings.EqualFold(scheme, "https") {
		return "443"
	}
	if strings.EqualFold(scheme, "http") {
		return "80"
	}
	return ""
}

func formatHostPort(host, scheme, port string) string {
	hostPart := host
	if strings.Contains(hostPart, ":") && !strings.HasPrefix(hostPart, "[") {
		hostPart = "[" + hostPart + "]"
	}

	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		return hostPart
	}
	return hostPart + ":" + port
}

func splitHostPort(requestHost string, requestTLS bool) (string, string) {
	trimmed := strings.TrimSpace(requestHost)
	if trimmed == "" {
		return "", ""
	}

	host := ""
	port := ""

	if parsedHost, parsedPort, err := net.SplitHostPort(trimmed); err == nil {
		host = strings.ToLower(strings.Trim(parsedHost, "[]"))
		port = parsedPort
	} else {
		host = strings.ToLower(strings.Trim(trimmed, "[]"))
	}

	if port == "" {
		if requestTLS {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host, port
}

func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// CORS handles Cross-Origin Resource Sharing with origin validation.
func CORS(allowedOrigins map[string]bool) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		origin := string(r.RequestCtx.Request.Header.Peek("Origin"))

		if origin != "" && IsOriginAllowedForRequest(origin, allowedOrigins, string(r.RequestCtx.Host()), r.RequestCtx.IsTLS()) {
			r.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			r.RequestCtx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
			r.RequestCtx.Response.Header.Set("Vary", "Origin")
			r.RequestCtx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			r.RequestCtx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Organization-ID, X-CSRF-Token")
			r.RequestCtx.Response.Header.Set("Access-Control-Max-Age", "86400")
		}

		return r
	}
}

// SecurityHeaders adds standard security headers to every response.
func SecurityHeaders(isProduction bool) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		h := &r.RequestCtx.Response.Header
		if !shouldSkipCSP(string(r.RequestCtx.Path())) {
			h.Set("Content-Security-Policy", defaultContentSecurityPolicy)
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("X-XSS-Protection", "0")
		if isProduction {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return r
	}
}

// Recovery recovers from panics
func Recovery(log logf.Logger) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		defer func() {
			if err := recover(); err != nil {
				errorRef := uuid.NewString()
				log.Error("Panic recovered", "error_reference", errorRef, "error", err, "path", string(r.RequestCtx.Path()))
				r.RequestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
				r.RequestCtx.Response.Header.SetContentType("application/json")
				r.RequestCtx.SetBodyString(fmt.Sprintf(`{"status":"error","message":"Internal server error","error_reference":"%s"}`, errorRef))
			}
		}()
		return r
	}
}

// Auth validates JWT tokens (legacy - use AuthWithDB for API key support)
func Auth(secret string) fastglue.FastMiddleware {
	return AuthWithDB(secret, nil)
}

// AuthWithDB validates both JWT tokens and API keys
func AuthWithDB(secret string, db *gorm.DB) fastglue.FastMiddleware {
	jwtSecret := strings.TrimSpace(secret)

	return func(r *fastglue.Request) *fastglue.Request {
		if jwtSecret == "" {
			_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Authentication is misconfigured", nil, "")
			return nil
		}

		authHeader := string(r.RequestCtx.Request.Header.Peek("Authorization"))
		apiKey := string(r.RequestCtx.Request.Header.Peek("X-API-Key"))

		if apiKey != "" && db != nil {
			clientIP := extractClientIP(r, false)
			if isAPIKeyRateLimited(clientIP) {
				_ = r.SendErrorEnvelope(fasthttp.StatusTooManyRequests,
					"Too many failed API key attempts. Please try again later.", nil, "")
				return nil
			}

			if validateAPIKey(r, apiKey, db) {
				return r
			}

			recordAPIKeyFailure(clientIP)
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid API key", nil, "")
			return nil
		}

		// Fall back to JWT authentication (Bearer header or cookie)
		var tokenString string

		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid authorization header format", nil, "")
				return nil
			}
			tokenString = parts[1]
		} else {
			// Fall back to whm_access cookie
			tokenString = string(r.RequestCtx.Request.Header.Cookie("whm_access"))
		}

		if tokenString == "" {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Missing authorization", nil, "")
			return nil
		}

		// Parse and validate token.
		// Enforce expected algorithm to prevent JWT algorithm confusion attacks.
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			signingMethod, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok || signingMethod.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected JWT signing method: %s", token.Method.Alg())
			}
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil || !token.Valid {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid or expired token", nil, "")
			return nil
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid token claims", nil, "")
			return nil
		}
		if claims.Subject != accessTokenSubject {
			// Enforce access-token type so refresh/ws tokens cannot authenticate API requests.
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid or expired token", nil, "")
			return nil
		}

		// Store claims in context
		r.RequestCtx.SetUserValue(ContextKeyUserID, claims.UserID)
		r.RequestCtx.SetUserValue(ContextKeyOrganizationID, claims.OrganizationID)
		r.RequestCtx.SetUserValue(ContextKeyEmail, claims.Email)
		if claims.RoleID != nil {
			r.RequestCtx.SetUserValue(ContextKeyRoleID, *claims.RoleID)
		}
		r.RequestCtx.SetUserValue(ContextKeyIsSuperAdmin, claims.IsSuperAdmin)

		return r
	}
}

// validateAPIKey validates an API key and sets context values.
// If the request includes an X-Organization-ID header, the key must belong
// to that organization (prevents cross-org key reuse).
func validateAPIKey(r *fastglue.Request, key string, db *gorm.DB) bool {
	if len(key) != 36 || key[:4] != "whm_" {
		return false
	}

	newPrefix := key[4:20]
	oldPrefix := key[4:12]

	var apiKeys []models.APIKey
	if err := db.Preload("User").Where("(key_prefix = ? OR key_prefix = ?) AND is_active = ?", newPrefix, oldPrefix, true).Find(&apiKeys).Error; err != nil {
		return false
	}

	headerOrgID := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek("X-Organization-ID")))

	for _, apiKey := range apiKeys {
		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(key)); err == nil {
			if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
				return false
			}

			if headerOrgID != "" {
				if subtle.ConstantTimeCompare([]byte(apiKey.OrganizationID.String()), []byte(headerOrgID)) != 1 {
					continue
				}
			}

			go func(id uuid.UUID) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				now := time.Now()
				db.WithContext(ctx).Model(&models.APIKey{}).Where("id = ?", id).Update("last_used_at", now)
			}(apiKey.ID)

			if apiKey.User != nil {
				r.RequestCtx.SetUserValue(ContextKeyUserID, apiKey.UserID)
				r.RequestCtx.SetUserValue(ContextKeyOrganizationID, apiKey.OrganizationID)
				r.RequestCtx.SetUserValue(ContextKeyEmail, apiKey.User.Email)
				if apiKey.User.RoleID != nil {
					r.RequestCtx.SetUserValue(ContextKeyRoleID, *apiKey.User.RoleID)
				}
				r.RequestCtx.SetUserValue(ContextKeyIsSuperAdmin, apiKey.User.IsSuperAdmin)
				return true
			}
		}
	}

	return false
}

func isAPIKeyRateLimited(clientIP string) bool {
	now := time.Now()
	if v, ok := apiKeyFailureLimiter.Load(clientIP); ok {
		entry := v.(*apiKeyFailureEntry)
		if now.Before(entry.expiresAt) && entry.count >= apiKeyAuthMaxFailures {
			return true
		}
	}
	return false
}

func recordAPIKeyFailure(clientIP string) {
	now := time.Now()
	newEntry := &apiKeyFailureEntry{count: 1, expiresAt: now.Add(apiKeyAuthWindow)}

	actual, _ := apiKeyFailureLimiter.LoadOrStore(clientIP, newEntry)
	entry := actual.(*apiKeyFailureEntry)

	if actual != newEntry {
		if now.After(entry.expiresAt) {
			entry.count = 1
			entry.expiresAt = now.Add(apiKeyAuthWindow)
		} else {
			entry.count++
		}
	}
}

// OrganizationContext loads organization and user from database
func OrganizationContext(db *gorm.DB) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		userID, ok := r.RequestCtx.UserValue(ContextKeyUserID).(uuid.UUID)
		if !ok {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "User ID not found in context", nil, "")
			return nil
		}

		orgID, ok := r.RequestCtx.UserValue(ContextKeyOrganizationID).(uuid.UUID)
		if !ok {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Organization ID not found in context", nil, "")
			return nil
		}

		// Load user
		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "User not found", nil, "")
			return nil
		}

		if !user.IsActive {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Account is disabled", nil, "")
			return nil
		}

		// Load organization
		var org models.Organization
		if err := db.Where("id = ?", orgID).First(&org).Error; err != nil {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Organization not found", nil, "")
			return nil
		}

		// Store in context
		r.RequestCtx.SetUserValue(ContextKeyUser, &user)
		r.RequestCtx.SetUserValue(ContextKeyOrganization, &org)

		return r
	}
}

// TenantScope resolves the effective organization and stores a scoped DB clone in the request context.
func TenantScope(db *gorm.DB) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		orgID, err := tenant.ResolveOrganizationID(r, db)
		if err != nil {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
			return nil
		}

		r.RequestCtx.SetUserValue(ContextKeyOrganizationID, orgID)
		tenant.SetScopedDB(r, tenant.ScopedDB(db, orgID))
		return r
	}
}

// PermissionChecker is a function that checks if a user has a permission
type PermissionChecker func(userID uuid.UUID, resource, action string) bool

// RequirePermission checks if user has the required permission using the provided checker
func RequirePermission(checker PermissionChecker, resource, action string) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		userID, ok := r.RequestCtx.UserValue(ContextKeyUserID).(uuid.UUID)
		if !ok {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "User not authenticated", nil, "")
			return nil
		}

		if !checker(userID, resource, action) {
			_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
			return nil
		}

		return r
	}
}

// RequireAnyPermission checks if user has any of the required permissions
func RequireAnyPermission(checker PermissionChecker, permissions ...string) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		userID, ok := r.RequestCtx.UserValue(ContextKeyUserID).(uuid.UUID)
		if !ok {
			_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "User not authenticated", nil, "")
			return nil
		}

		for _, perm := range permissions {
			parts := strings.Split(perm, ":")
			if len(parts) == 2 && checker(userID, parts[0], parts[1]) {
				return r
			}
		}

		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return nil
	}
}

// GetUserID extracts user ID from request context
func GetUserID(r *fastglue.Request) (uuid.UUID, bool) {
	userID, ok := r.RequestCtx.UserValue(ContextKeyUserID).(uuid.UUID)
	return userID, ok
}

// GetOrganizationID extracts organization ID from request context
func GetOrganizationID(r *fastglue.Request) (uuid.UUID, bool) {
	orgID, ok := r.RequestCtx.UserValue(ContextKeyOrganizationID).(uuid.UUID)
	return orgID, ok
}

// GetUser extracts user from request context
func GetUser(r *fastglue.Request) (*models.User, bool) {
	user, ok := r.RequestCtx.UserValue(ContextKeyUser).(*models.User)
	return user, ok
}

// GetOrganization extracts organization from request context
func GetOrganization(r *fastglue.Request) (*models.Organization, bool) {
	org, ok := r.RequestCtx.UserValue(ContextKeyOrganization).(*models.Organization)
	return org, ok
}

// IsSuperAdmin checks if the current user is a super admin
func IsSuperAdmin(r *fastglue.Request) bool {
	isSuperAdmin, ok := r.RequestCtx.UserValue(ContextKeyIsSuperAdmin).(bool)
	return ok && isSuperAdmin
}
