# Whatomate Security Audit Report

**Date**: 2026-05-08  
**Scope**: Full codebase — Go backend, Vue 3 frontend, configuration, dependencies  
**Pre-publication audit** — ranked by threat level (CRITICAL > HIGH > MEDIUM > LOW > INFO)

---

## CRITICAL (Severity 9-10)

### C1. Path Traversal in `fileSystemObjectStorage.PutObject/GetObject/DeleteObject`
**File**: `internal/storage/object_storage.go:67-108`  
**Threat**: An attacker who controls the `key` parameter can traverse directories outside the storage root using `../` sequences. Unlike `serveLocalMediaFile` in `media.go` which validates prefix containment, the `ObjectStorage` interface methods blindly join the key with the root path and operate on the resulting path.  
**Impact**: Arbitrary file read/write/delete on the server filesystem.  
**Status**: `PutObject` and `GetObject` in `fileSystemObjectStorage` perform zero path sanitization or boundary checks.  
**Fix**: Add `filepath.Clean`, `filepath.EvalSymlinks`, and prefix containment checks matching the pattern in `media.go:280-300`.

### C2. Path Traversal in `fileSystemObjectStorage.PutObject` — Directory Creation
**File**: `internal/storage/object_storage.go:68`  
**Threat**: `os.MkdirAll(filepath.Dir(path), 0755)` creates arbitrary directories on the filesystem when a malicious key like `../../../tmp/evil` is provided.  
**Impact**: Arbitrary directory creation outside storage root.  
**Fix**: Validate key against root before any filesystem operations.

### C3. Open Redirect in SSO Error Redirect
**File**: `internal/handlers/sso_handlers.go:169-170`  
**Threat**: `a.redirectWithError(r, "SSO failed: "+errorDesc)` — the `errorDesc` value comes directly from the OAuth provider's `error_description` query parameter (`r.RequestCtx.QueryArgs().Peek("error_description")`). If `redirectWithError` reflects this into a URL or HTML response, it enables reflected XSS or open redirect.  
**Impact**: Potential XSS or open redirect via crafted OAuth error callback.  
**Fix**: Sanitize or HTML-escape `errorDesc` before reflecting it in any response. Never include raw provider error strings in redirects.

---

## HIGH (Severity 7-8)

### H1. Insecure Default Configuration Values
**File**: `config.example.toml`, `internal/config/config.go`  
**Threat**: Default values for production-sensitive fields are dangerous:
- `database.user = "change-me"` / `database.password = "change-me"`
- `jwt.secret = ""` (empty — must be explicitly set)
- `encryption_key = ""` (empty — must be explicitly set)
- `whatsapp.webhook_verify_token = "change-me"`
- `database.ssl_mode = "disable"`
- `cookie.secure = false`
- `observability.enable_pprof = false` (good, but can be enabled without auth)
- `license.allow_unsafe_public_key_override = true`

**Impact**: Operators who deploy without changing defaults run with no JWT signing, no encryption, weak database credentials, and no TLS cookies.  
**Mitigation**: The `security_validation.go`, `jwt_validation.go`, and `encryption_validation.go` files validate these in production — **but only if the startup code actually calls these validators**. Verify all validators are invoked at boot.  
**Fix**: Add a startup gate that refuses to start in production unless all validators pass.

### H2. Timing-Safe Comparison Missing on CSRF Token
**File**: `internal/middleware/csrf.go:43`  
**Threat**: `csrfCookie != csrfHeader` uses Go's standard string comparison, which is vulnerable to timing side-channel attacks.  
**Impact**: An attacker could theoretically infer the CSRF token byte-by-byte via timing measurements.  
**Fix**: Use `crypto/subtle.ConstantTimeCompare` for the CSRF token comparison.

### H3. WebSocket Token Returned in API Response Body
**File**: `internal/handlers/auth_handlers.go` (GetWSToken)  
**Threat**: The WebSocket token is returned as JSON in the response body: `r.SendEnvelope(map[string]string{"token": signed})`. If the frontend stores this token where JavaScript can access it (e.g., in memory or localStorage), any XSS on the page can steal it.  
**Impact**: Stolen WS token grants real-time chat access.  
**Fix**: Consider returning the WS token in an HttpOnly cookie instead of the response body, similar to access/refresh tokens.

### H4. CSP `style-src 'unsafe-inline'` Weakens XSS Protection
**File**: `internal/middleware/middleware.go:28`  
**Threat**: The Content Security Policy includes `style-src 'self' 'unsafe-inline'`, which allows inline styles everywhere. An attacker who can inject HTML can use inline styles to exfiltrate data via CSS injection (e.g., attribute selectors + background-image exfil).  
**Impact**: CSS-based data exfiltration if HTML injection exists elsewhere.  
**Fix**: Use nonce-based or hash-based style-src. Migrate inline styles to CSS classes.

### H5. Login Endpoint Missing Constant-Time Password Comparison
**File**: `internal/handlers/auth_handlers.go:43`  
**Threat**: `bcrypt.CompareHashAndPassword` is used (which is timing-safe), but the dummy comparison on line 43 (`bcrypt.CompareHashAndPassword([]byte("$2a$10$xxxx..."), []byte(req.Password))`) leaks timing information because the dummy hash is a different length than real hashes.  
**Impact**: Minor timing leak that could theoretically aid username enumeration.  
**Fix**: Use a proper constant-time dummy comparison with a real bcrypt hash.

### H6. CSP Skipped on Root Path `/` and All Non-Extension Paths
**File**: `internal/middleware/middleware.go:34-45`  
**Threat**: `shouldSkipCSP` returns `true` for the root path `/` and any path without a file extension (e.g., `/dashboard`, `/settings`). This means the CSP header is not set for the majority of the SPA routes.  
**Impact**: No CSP protection on most application routes, making XSS exploitation easier.  
**Fix**: Apply CSP to all responses except static asset file extensions (`.js`, `.css`, `.png`, etc.), not based on URL path.

### H7. `style-src 'unsafe-inline'` + Missing CSP on SPA Routes = XSS Amplification
**Combined**: H4 + H6 together mean that the CSP is essentially non-functional for the entire Vue SPA. Any XSS vulnerability in the frontend code or dependencies is fully exploitable.

---

## MEDIUM (Severity 5-6)

### M1. ObjectStorage Interface Lacks Path Validation (Design Gap)
**File**: `internal/storage/object_storage.go`  
**Threat**: The `ObjectStorage` interface itself has no contract for path safety. Any caller that passes unsanitized keys can trigger C1/C2.  
**Impact**: Systemic risk — any code path that accepts user input and passes it to `PutObject`/`GetObject`/`DeleteObject` is exploitable.  
**Fix**: Add path validation in the interface implementation and document the contract.

### M2. `Allow-Origin` Headers Leaked on Non-CORS Requests
**File**: `internal/middleware/middleware.go:173-178`  
**Threat**: `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`, and `Access-Control-Max-Age` headers are set on ALL responses, even when the origin is not allowed. This reveals the API's capabilities to any caller.  
**Impact**: Information disclosure of allowed methods and headers.  
**Fix**: Only set these headers on OPTIONS preflight requests, and only when the origin is allowed.

### M3. Pprof Endpoints Without Strong Authentication Guard
**File**: `config.example.toml` (observability section)  
**Threat**: When `observability.enable_pprof = true` and `access_token` is empty, pprof endpoints are restricted to loopback only. However, if `TrustProxy = true` is configured, an attacker behind a trusted proxy can reach pprof via `X-Forwarded-For`.  
**Impact**: Memory/CPU profiling data exposure reveals runtime internals.  
**Fix**: Always require the access_token for pprof, regardless of client IP.

### M4. `style-src 'unsafe-inline'` in CSP
**File**: `internal/middleware/middleware.go:28`  
**(Covered by H4/H7 — listed separately for tracking)**

### M5. Custom Action JavaScript Execution (goja)
**File**: `internal/handlers/custom_action_runtime.go`  
**Threat**: Custom actions execute user-defined JavaScript using the goja runtime with a 2-10 second timeout. While the VM is sandboxed from the OS, JavaScript execution within the application context could access any data passed to it.  
**Impact**: If custom action scripts have access to sensitive request data, a compromised script could exfiltrate it.  
**Mitigation**: Timeout is enforced; VM has no filesystem/network access by default.  
**Fix**: Document the security boundary clearly. Consider running JS in a separate process with resource limits.

### M6. License Public Key Override Enabled by Default
**File**: `config.example.toml` — `license.allow_unsafe_public_key_override = true`  
**Threat**: Allows operators to substitute the license verification public key, effectively disabling license enforcement.  
**Impact**: License bypass — the entire licensing system can be defeated.  
**Fix**: Default to `false` and require explicit opt-in.

### M7. SuperAdmin Can Access Any Organization via X-Organization-ID
**File**: `internal/handlers/app.go:70-73`, `internal/tenant/scope.go:100+`  
**Threat**: The `getOrgID` function allows super admins to override the organization context via the `X-Organization-ID` header. While this is by design, there's no audit logging of cross-organization access.  
**Impact**: Super admin can silently access any tenant's data without trace.  
**Fix**: Add audit logging for super admin cross-org access.

### M8. API Key Lookup Not Scoped to Organization
**File**: `internal/middleware/middleware.go:220-225`  
**Threat**: `validateAPIKey` queries API keys globally (`db.Preload("User").Where("(key_prefix = ? OR key_prefix = ?) AND is_active = ?", ...)`), not scoped to the request's organization. An API key created in org A can authenticate requests intended for org B if the API key's user also has access to org B.  
**Impact**: Cross-tenant authentication if API keys aren't org-scoped.  
**Fix**: Add `AND organization_id = ?` to the API key query and validate it matches the request context.

---

## LOW (Severity 3-4)

### L1. `X-XSS-Protection: 0` Header
**File**: `internal/middleware/middleware.go:251`  
**Note**: This is actually correct per OWASP recommendation (use CSP instead of buggy XSS filter). Listed for awareness.

### L2. No `Strict-Transport-Security` (HSTS) Header
**File**: `internal/middleware/middleware.go:240-252`  
**Threat**: HSTS header is not set. Without it, browsers may fall back to HTTP on the first connection.  
**Impact**: Protocol downgrade attacks possible.  
**Fix**: Add `Strict-Transport-Security: max-age=31536000; includeSubDomains` in `SecurityHeaders()` when running in production.

### L3. No `Cache-Control` on Sensitive API Responses
**Threat**: API responses containing user data, settings, or tokens may be cached by intermediary proxies or browsers.  
**Impact**: Sensitive data in cache.  
**Fix**: Add `Cache-Control: no-store` to all `/api/auth/*` responses.

### L4. Webhook Verify Token Compared with `==` (Not Constant-Time)
**File**: `internal/handlers/webhook.go:33`  
**Threat**: `token == a.Config.WhatsApp.WebhookVerifyToken` uses standard comparison.  
**Impact**: Minimal — the verify token is sent in a URL query parameter, making timing attacks impractical.  
**Fix**: Use `subtle.ConstantTimeCompare` for defense-in-depth.

### L5. Refresh Token Rotation Does Not Check IP Binding
**File**: `internal/handlers/auth_handlers.go:139-165`  
**Threat**: Refresh tokens are not bound to the client IP or user-agent. If a refresh token is stolen, it can be used from any IP.  
**Impact**: Token theft enables persistent access.  
**Fix**: Consider binding refresh tokens to client fingerprint (IP + User-Agent hash) and invalidating on change.

### L6. Default Admin Bootstrap Without Password Complexity
**File**: `internal/config/default_admin_validation.go`  
**Threat**: While there's validation for minimum 12 chars in production, the default admin seeding doesn't enforce the same password policy (`validatePasswordStrength`) that registration uses.  
**Impact**: Weak bootstrap admin password possible.  
**Fix**: Apply `validatePasswordStrength` to the default admin password.

### L7. `Database.SSLMode` Defaults to `disable`
**File**: `internal/config/config.go:149`  
**Threat**: Database connections default to unencrypted.  
**Impact**: Database credentials and data can be intercepted on the network.  
**Fix**: Default to `require` in production or warn loudly.

---

## INFO (Severity 1-2)

### I1. Password Policy Does Not Require Special Characters
**File**: `internal/handlers/password_policy.go`  
**Note**: Policy requires upper + lower + digit + 12 chars minimum but no special characters. This is a design choice, not a vulnerability.

### I2. JWT Uses HS256 (Symmetric)
**Note**: HMAC-SHA256 is secure when the secret is sufficiently long (32+ chars enforced). RS256 would provide better key rotation but adds complexity. Acceptable for this application.

### I3. Rate Limiter Fails Closed (Correct Behavior)
**File**: `internal/middleware/ratelimit.go:82-87`  
**Note**: Rate limiter denies requests when Redis is unavailable. This is the correct security posture.

### I4. Good: SSRF Protection
**Files**: `internal/handlers/webhooks.go:56-80`, `internal/handlers/sso_security.go:109`  
**Note**: SSRFSafeDialer correctly blocks connections to private/loopback IPs after DNS resolution. SSO custom endpoint validation blocks private hosts in production. Webhook URL validation blocks internal hostnames.

### I5. Good: JWT Algorithm Confusion Prevention
**File**: `internal/middleware/middleware.go:192-197`  
**Note**: JWT parsing enforces `jwt.SigningMethodHS256` and rejects non-HMAC methods, preventing algorithm confusion attacks.

### I6. Good: CSRF Double-Submit Pattern
**File**: `internal/middleware/csrf.go`  
**Note**: CSRF protection correctly skips header-based auth and validates double-submit cookie pattern. Timing-safe comparison should be added (H2).

### I7. Good: SSO Security
**Note**: SSO implements PKCE, state nonce with Redis, browser-bound cookie, state deletion after use, and email domain validation. Custom provider URLs are validated for SSRF.

### I8. Good: Media File Serving Has Path Traversal Protection
**File**: `internal/handlers/media.go:280-300`  
**Note**: `serveLocalMediaFile` properly validates path containment with symlink rejection. The vulnerability is in the `ObjectStorage` abstraction layer (C1/C2).

### I9. Good: Encryption at Rest
**File**: `internal/crypto/crypto.go`  
**Note**: Uses AES-256-GCM with Argon2id key derivation (enc3). Legacy formats (enc/enc2) supported with opt-in. Production validation blocks weak keys.

### I10. Good: Security Headers Present
**File**: `internal/middleware/middleware.go:240-252`  
**Note**: X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy are all set. Missing HSTS (L2).

---

## Summary Table

| ID | Severity | Category | Title | Status |
|---|---|---|---|---|
| C1 | CRITICAL | Path Traversal | `ObjectStorage` lacks path validation | **FIXED** |
| C2 | CRITICAL | Path Traversal | `PutObject` creates arbitrary directories | **FIXED** |
| C3 | CRITICAL | XSS/Open Redirect | SSO error reflects unsanitized provider string | **FIXED** |
| H1 | HIGH | Configuration | Insecure defaults for production deployment | Partially mitigated |
| H2 | HIGH | CSRF | Non-constant-time CSRF comparison | **FIXED** |
| H3 | HIGH | Token Exposure | WS token in response body, stealable via XSS | Open |
| H4 | HIGH | CSP | `style-src 'unsafe-inline'` weakens XSS protection | Open |
| H5 | HIGH | Timing | Login dummy hash comparison leaks timing | **FIXED** |
| H6 | HIGH | CSP | CSP skipped on all SPA routes | **FIXED** |
| H7 | HIGH | CSP | Combined: no effective CSP on any SPA route | **FIXED** |
| M1 | MEDIUM | Design | ObjectStorage interface lacks safety contract | **FIXED** (via C1/C2) |
| M2 | MEDIUM | Info Leak | CORS headers leaked on non-allowed origins | **FIXED** |
| M3 | MEDIUM | Auth | Pprof accessible without strong auth when TrustProxy=true | Open |
| M4 | MEDIUM | CSP | `style-src 'unsafe-inline'` (duplicate of H4) | Open |
| M5 | MEDIUM | Sandboxing | Custom action JS execution boundaries unclear | Open |
| M6 | MEDIUM | License | Public key override enabled by default | Open |
| M7 | MEDIUM | Audit | No audit logging for super admin cross-org access | Open |
| M8 | MEDIUM | Tenant | API key lookup not org-scoped | **FIXED** |
| L1 | LOW | Headers | `X-XSS-Protection: 0` (correct per OWASP) | Info |
| L2 | LOW | Headers | Missing HSTS header | **FIXED** |
| L3 | LOW | Caching | No `Cache-Control: no-store` on auth endpoints | **FIXED** |
| L4 | LOW | Timing | Webhook verify token non-constant-time compare | **FIXED** |
| L5 | LOW | Auth | Refresh tokens not bound to client fingerprint | Open |
| L6 | LOW | Auth | Default admin password policy gap | **FIXED** |
| L7 | LOW | Transport | DB SSL mode defaults to disabled | Open |

---

## Priority Remediation Order

1. **Immediate (before launch)**: C1, C2, C3 — path traversal in ObjectStorage and SSO reflection
2. **Before launch**: H1 — verify all config validators are called at startup; H2, H6, H7 — fix CSP and CSRF
3. **Before launch**: M8 — scope API key lookup to organization
4. **Post-launch**: H3, H5, M2, M3, L2, L3 — defense-in-depth improvements
5. **Ongoing**: M5, M6, M7, L5, L6, L7 — hardening and monitoring

---

## Files Analyzed

- `internal/middleware/middleware.go`, `csrf.go`, `ratelimit.go`
- `internal/crypto/crypto.go`
- `internal/handlers/auth_handlers.go`, `auth_utils.go`, `auth_crypto.go`, `auth_types.go`, `auth_expiry.go`, `jwt_secret.go`, `cookies.go`, `password_policy.go`
- `internal/handlers/sso_handlers.go`, `sso_security.go`, `sso_utils.go`
- `internal/handlers/webhook.go`, `webhook_security.go`, `webhooks.go`
- `internal/handlers/config_handler.go`, `custom_action_runtime.go`, `provider_guard.go`
- `internal/handlers/media.go`, `users.go`, `app.go`
- `internal/handlers/websocket.go`
- `internal/storage/object_storage.go`
- `internal/tenant/scope.go`
- `internal/config/config.go`, `security_validation.go`, `jwt_validation.go`, `encryption_validation.go`, `default_admin_validation.go`
- `config.example.toml`
