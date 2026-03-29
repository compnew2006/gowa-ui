# Security Review Report -- Whatomate

## Executive Summary

| Field | Value |
|-------|-------|
| **Application** | Whatomate (WhatsApp Business Platform) |
| **Review Date** | 2026-03-29 |
| **Scope** | Full codebase: Go backend, Vue.js frontend, MCP sidecar |
| **Tech Stack** | Go 1.25 + FastHTTP + GORM + PostgreSQL + Redis + Vue 3 |
| **Overall Risk Level** | **HIGH** |

### Key Findings
- **0 Critical** vulnerabilities (previous C-01 SSRF has been fixed)
- **5 High** severity issues requiring immediate attention
- **17 Medium** severity issues to address before production
- **15 Low/Informational** issues for backlog

---

## Findings Summary

| Severity | Count | Action |
|----------|-------|--------|
| Critical | 0 | -- |
| High | 5 | Fix immediately |
| Medium | 17 | Fix before next release |
| Low/Info | 15 | Backlog |

---

## HIGH Severity Findings

### SEC-001: Insecure Key Derivation -- Single SHA-256, No KDF

| Field | Value |
|-------|-------|
| **CWE** | CWE-328 (Weak Hash), CWE-916 (Weak Password Recovery) |
| **Location** | `internal/crypto/crypto.go:124-141` |
| **CVSS** | 7.5 |

**Description**

The `deriveKey()` function uses a bare `sha256.Sum256` when the input is a passphrase (not raw hex/base64). No salt, no iterations, no memory hardness. Billions of guesses/second on modern GPUs.

```go
sum := sha256.Sum256([]byte(trimmed))
return sum[:]
```

**Remediation:** Use `golang.org/x/crypto/argon2id` with per-ciphertext salt.

---

### SEC-002: Insecure Legacy Key Derivation Still Active

| Field | Value |
|-------|-------|
| **CWE** | CWE-328 |
| **Location** | `internal/crypto/crypto.go:146-149` |
| **CVSS** | 7.5 |

**Description**

`deriveLegacyKey()` copies raw passphrase bytes with zero-padding. Short keys have drastically reduced key space. Legacy `enc:` prefix decryption path remains active with no migration tooling.

```go
func deriveLegacyKey(key string) []byte {
    k := make([]byte, 32)
    copy(k, []byte(key))
    return k
}
```

**Remediation:** Build migration utility to re-encrypt all `enc:` values to `enc2:`, then deprecate legacy path.

---

### SEC-003: Missing RBAC on Organization Settings Update

| Field | Value |
|-------|-------|
| **CWE** | CWE-862 (Missing Authorization) |
| **Location** | `internal/handlers/organization.go:107-108` |
| **Endpoint** | `PUT /api/org/settings` |
| **CVSS** | 7.0 |

**Description**

Any authenticated user (including lowest-privilege agents) can modify organization-wide settings -- outbound mode, strict sending restrictions, timezone, campaign settings -- with zero permission checks.

**Remediation:** Add `a.requirePermission(r, userID, models.ResourceSettings, models.ActionWrite, orgID)` at the top of `UpdateOrganizationSettings`.

---

### SEC-004: Missing RBAC on All Canned Response CRUD

| Field | Value |
|-------|-------|
| **CWE** | CWE-862 |
| **Location** | `internal/handlers/canned_responses.go:38,92,154,176,252` |
| **Endpoints** | All `/api/canned-responses/*` |
| **CVSS** | 7.0 |

**Description**

None of the 5 canned response endpoints (list, get, create, update, delete) check permissions. Any agent can inject malicious content into shared response templates used by all team members.

**Remediation:** Add `requirePermission` calls with `canned_responses:read/write/delete` to each handler.

---

### SEC-005: Self-Password-Reset Without Current Password Verification

| Field | Value |
|-------|-------|
| **CWE** | CWE-620 (Unverified Password Change) |
| **Location** | `internal/handlers/users.go:471-481` |
| **Endpoint** | `PUT /api/users/{id}` |
| **CVSS** | 7.3 |

**Description**

Users can change their own password via the user update endpoint without verifying the current password. If an attacker hijacks a session, they can lock the victim out permanently.

**Remediation:** Require current password verification for self-password changes, or route all password changes through the dedicated `ChangePassword` endpoint which already does this correctly.

---

## MEDIUM Severity Findings

| ID | Finding | Location | CWE |
|----|---------|----------|-----|
| SEC-006 | Registration endpoint leaks email existence | `auth_handlers.go:169-170` | CWE-204 |
| SEC-007 | Cookie Secure flag can be disabled in production | `cookies.go:18` | CWE-614 |
| SEC-008 | Rate limiting fails open when Redis is down | `ratelimit.go:49-52` | CWE-770 |
| SEC-009 | JWT secret length not validated at runtime | `jwt_secret.go:8-19` | CWE-326 |
| SEC-010 | Legacy crypto passthrough tolerates unencrypted secrets | `crypto.go:59-61` | CWE-312 |
| SEC-011 | Silent decryption failure preserves ciphertext in API calls | `crypto.go:108-114` | CWE-755 |
| SEC-012 | Hardcoded DB credentials in example config | `config.example.toml:26-27` | CWE-798 |
| SEC-013 | Weak default webhook verify token | `config.example.toml:45` | CWE-798 |
| SEC-014 | Empty encryption key accepted outside production | `encryption_validation.go:27-33` | CWE-321 |
| SEC-015 | Encryption key length only enforced in production | `encryption_validation.go:39-41` | CWE-321 |
| SEC-016 | Google API key exposed in URL query parameter | `chatbot_processor.go:2111-2112` | CWE-598 |
| SEC-017 | Dashboard stats accessible without permission | `analytics.go:34-38` | CWE-862 |
| SEC-018 | URL action type skips URL validation | `custom_actions.go:640-643` | CWE-918 |
| SEC-019 | Open redirect via template variable injection | `custom_actions.go:469,371` | CWE-601 |
| SEC-020 | Chatbot API URLs unvalidated at creation | `chatbot.go:791,818` | CWE-918 |
| SEC-021 | ListOrganizations missing orgID in permission check | `organization.go:410` | CWE-863 |
| SEC-022 | No max password length (CPU DoS via bcrypt) | `password_policy.go` | CWE-400 |

### SEC-006: Registration Endpoint Leaks Email Existence

**Location:** `internal/handlers/auth_handlers.go:169-170`

The registration endpoint returns a 409 error when an email already exists with a wrong password, revealing whether an email is registered. Combined with the invitation token flow, an attacker who knows a user's email and password can silently add them to a new organization.

**Remediation:** Return a generic success/error response regardless of whether the email exists. Do not validate the password against an existing account during registration.

### SEC-007: Cookie Secure Flag Can Be Disabled in Production

**Location:** `internal/handlers/cookies.go:18`

The `Secure` flag on auth cookies is configurable via `a.Config.Cookie.Secure`. If misconfigured in production, all cookie-based auth can be intercepted over HTTP.

**Remediation:** Force `Secure: true` when `app.environment == "production"` regardless of config.

### SEC-008: Rate Limiting Fails Open When Redis Is Down

**Location:** `internal/middleware/ratelimit.go:49-52`

If Redis is unavailable, all rate-limited endpoints (login, register, refresh, SSO) become unthrottled. An attacker who can disrupt Redis can bypass all rate limits.

**Remediation:** Implement fail-closed rate limiting or add a circuit breaker that denies requests when Redis is unreachable.

### SEC-009: JWT Secret Length Not Validated at Runtime

**Location:** `internal/handlers/jwt_secret.go:8-19`

The `jwtSecretBytes()` function only checks for empty strings. No minimum length is enforced at runtime. Short JWT secrets are vulnerable to offline brute-force.

**Remediation:** Enforce minimum 32-byte secret length in `jwtSecretBytes()` or at startup validation.

### SEC-010: Legacy Crypto Passthrough Tolerates Unencrypted Secrets

**Location:** `internal/crypto/crypto.go:59-61`

The `Decrypt` function returns any value without an `enc2:` or `enc:` prefix as plaintext. If an attacker strips the prefix from stored ciphertext, the system treats it as plaintext.

**Remediation:** Add an option to reject unencrypted values for known-secret fields (tokens, API keys, etc.).

### SEC-011: Silent Decryption Failure Preserves Ciphertext in API Calls

**Location:** `internal/crypto/crypto.go:108-114`

`DecryptFields` silently preserves the original value on decryption failure. If the encryption key changes, ciphertext blobs may be sent as "tokens" in outbound API requests.

**Remediation:** Propagate decryption errors or log warnings when decryption fails for sensitive fields.

### SEC-012: Hardcoded DB Credentials in Example Config

**Location:** `config.example.toml:26-27`

Example config ships with `user = "whatomate"` / `password = "whatomate"`. No validation rejects these defaults in production.

**Remediation:** Add production validation to reject known default credentials.

### SEC-013: Weak Default Webhook Verify Token

**Location:** `config.example.toml:45`

Default `webhook_verify_token = "secret"` is a well-known placeholder. No validation rejects this in production.

**Remediation:** Add production validation to reject `"secret"` and other common placeholders.

### SEC-014: Empty Encryption Key Accepted Outside Production

**Location:** `internal/config/encryption_validation.go:27-33`

In non-production environments, an empty encryption key is silently accepted. Combined with `deriveKey` returning all-zeros for empty keys, all "encrypted" secrets are trivially decryptable in dev/staging.

**Remediation:** Generate a random key when empty in non-production, or refuse to start.

### SEC-015: Encryption Key Length Only Enforced in Production

**Location:** `internal/config/encryption_validation.go:39-41`

The 32-character minimum is only enforced when `environment == "production"`. In staging/development, single-character keys are accepted. The placeholder blocklist is small (6 entries) and misses common weak values like `"test"`, `"dev"`.

**Remediation:** Enforce minimum key length across all environments, or at minimum expand the blocklist.

### SEC-016: Google API Key Exposed in URL Query Parameter

**Location:** `internal/handlers/chatbot_processor.go:2111-2112`

```go
url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
    settings.AI.Model, settings.AI.APIKey)
```

The API key appears in HTTP request logs, proxy logs, and referrer headers.

**Remediation:** Use `x-goog-api-key` header instead of query parameter.

### SEC-017: Dashboard Stats Accessible Without Permission

**Location:** `internal/handlers/analytics.go:34-38`

The `GetDashboardStats` handler exposes total message counts, contact counts, and recent message content with no permission gate. The `analytics:read` permission exists but is not enforced.

**Remediation:** Add `a.requirePermission(r, userID, models.ResourceAnalytics, models.ActionRead, orgID)`.

### SEC-018: URL Action Type Skips URL Validation

**Location:** `internal/handlers/custom_actions.go:640-643`

Webhook actions are validated with `validateWebhookURL`, but URL actions are not. Any URL scheme (`javascript:`, `data:`, `file:`) passes validation.

**Remediation:** Apply `validateWebhookURL` to URL action types as well.

### SEC-019: Open Redirect via Template Variable Injection

**Location:** `internal/handlers/custom_actions.go:469,371`

The `replaceVariables` function substitutes `{{variable}}` placeholders with context data. If the URL template includes attacker-controlled chat message data, the resulting redirect URL can point to an arbitrary destination.

**Remediation:** Validate the final URL after template substitution before storing the redirect token. Enforce `http`/`https` scheme on the resolved URL.

### SEC-020: Chatbot API URLs Unvalidated at Creation

**Location:** `internal/handlers/chatbot.go:791,818`

Chatbot API and completion config URLs are stored in the database without validation. Template variable substitution at runtime can alter the URL. The shared `SSRFSafeDialer` provides runtime SSRF protection but no application-level URL validation.

**Remediation:** Validate URLs against an allowlist of schemes and hostnames at creation time.

### SEC-021: ListOrganizations Missing orgID in Permission Check

**Location:** `internal/handlers/organization.go:410`

`HasPermission` is called without the `orgID` parameter, falling through to check the user's home org role instead of the currently active org role.

**Remediation:** Pass `orgID` to the `HasPermission` call.

### SEC-022: No Max Password Length (CPU DoS via bcrypt)

**Location:** `internal/handlers/password_policy.go`

No maximum password length is enforced. An extremely long password (e.g., 10MB) causes excessive CPU usage in `bcrypt.GenerateFromPassword`.

**Remediation:** Add a maximum password length check (e.g., 128 characters).

---

## LOW Severity / Informational Findings

| ID | Finding | Location |
|----|---------|----------|
| SEC-023 | Missing `jwt.WithValidMethods` on invite token validation | `auth_utils.go:141-143` |
| SEC-024 | Logout endpoint not rate-limited | `main.go:633` |
| SEC-025 | Logout silently ignores JWT errors (no logging) | `auth_handlers.go:519-521` |
| SEC-026 | Imperfect timing mitigation on user enumeration | `auth_handlers.go:27-31` |
| SEC-027 | bcrypt DefaultCost (10) is borderline | `auth_handlers.go:224` |
| SEC-028 | OrganizationContext skips membership check | `middleware.go:394-433` |
| SEC-029 | CSRF token not cryptographically bound to session | `csrf.go:33-37` |
| SEC-030 | SSO token exchange uses `context.Background()` | `sso_handlers.go:189` |
| SEC-031 | Fixed-window rate limiting (boundary burst) | `ratelimit.go:48-60` |
| SEC-032 | JWT secret length only enforced in production | `jwt_validation.go:33-35` |
| SEC-033 | Single encryption key for all secrets (no key separation) | `config.go:47` |
| SEC-034 | No key rotation support | all crypto files |
| SEC-035 | API keys use 128-bit entropy (adequate but minimal) | `apikeys.go:43-48` |
| SEC-036 | RBAC middleware is a no-op (no safety net) | `main.go:696-707` |
| SEC-037 | SSO auto-created users have empty password hash | `sso_handlers.go:249-258` |

---

## Automated Scan Results

### GoSec (Existing Report)

| Severity | Count |
|----------|-------|
| HIGH | 7 |
| MEDIUM | 12 |
| LOW | 1 |

**GoSec HIGH highlights:**
- Integer overflow conversions in `chat_assignment_reset_settings.go`, `flows.go`, `statuses.go`
- Weak random number generator (`math/rand`) in `typing_indicator.go:45`, `queue.go:197`
- Goroutine uses `context.Background` while request context available in `messages.go:280`

### npm Audit

| Package | Critical | High | Moderate | Low |
|---------|----------|------|----------|-----|
| Frontend | 0 | 6 | 0 | 0 |
| MCP Server | 0 | 1 | 5 | 0 |

---

## Positive Security Observations

1. **Token type segregation** via JWT `Subject` claims (`access`, `refresh`, `ws`) prevents token confusion attacks
2. **Double algorithm enforcement** (`jwt.WithValidMethods` + type assertion) blocks `alg=none`
3. **Refresh token rotation** with atomic Redis delete provides strong replay protection
4. **SSRF-safe dialer** blocks private/loopback IPs on outbound HTTP requests
5. **Whatsmeow adapter** has thorough SSRF + path traversal defense with DNS-rebinding TOCTOU protection
6. **CSRF double-submit** with `SameSite=Lax` cookies is well implemented
7. **AES-256-GCM** (authenticated encryption) for current `enc2:` encryption path
8. **Widget SQL injection** properly mitigated with 3-layer defense (regex + allowlist + re-validation)
9. **OAuth2 state** uses crypto/rand, Redis TTL, immediate deletion, provider validation
10. **Cookie security**: HttpOnly, narrow refresh path scoping, SameSite=Lax

---

## Prioritized Recommendations

### Immediate (This Sprint)

1. Add `requirePermission` to org settings and canned response endpoints (**SEC-003**, **SEC-004**)
2. Require current password for self-password changes (**SEC-005**)
3. Replace bare SHA-256 with argon2id key derivation (**SEC-001**)
4. Build legacy `enc:` to `enc2:` migration utility (**SEC-002**)

### Short-term (Next Sprint)

1. Enforce minimum JWT secret length at runtime (**SEC-009**)
2. Make rate limiting fail-closed or add circuit breaker (**SEC-008**)
3. Add max password length to prevent CPU DoS (**SEC-022**)
4. Add `jwt.WithValidMethods` to invite token validation (**SEC-023**)
5. Validate URL action types with `validateWebhookURL` (**SEC-018**)
6. Use header-based auth for Google API key (**SEC-016**)
7. Validate chatbot API URLs at creation time (**SEC-020**)

### Long-term

1. Implement centralized RBAC middleware (restore safety net, **SEC-036**)
2. Add encryption key rotation support (**SEC-034**)
3. Enforce security config validation across all environments, not just production
4. Install `govulncheck` and `gitleaks` in CI/CD pipeline
5. Schedule quarterly security reviews

---

## Tools Used

- Manual code review (Go backend, Vue.js frontend, MCP sidecar)
- GoSec SAST scanner (existing report: `security_reports/gosec.json`)
- npm audit (frontend + MCP server)
- Existing govulncheck report (`security_reports/govulncheck_latest.txt`)
- Existing Semgrep results (`semgrep_results.json`)

## References

- OWASP Top 10 2021
- CWE Database (MITRE)
- CVSS v3.1 Scoring

## Comparison with Previous Audit (2026-03-04)

| Previous Finding | Status |
|-----------------|--------|
| C-01: SSRF + local file read in whatsmeow adapter | **FIXED** -- comprehensive SSRF + path traversal defenses added |
| H-01: Missing RBAC on chatbot endpoints | Partially addressed -- some endpoints still missing checks |
| H-02: Token confusion on refresh endpoint | **FIXED** -- Subject claim validation added |
| H-03: SQL injection in widget aggregation | **FIXED** -- 3-layer defense (regex + allowlist + re-validation) |
| H-04: AI API keys stored plaintext | Still partially present -- encrypted in DB but weak key derivation |
| H-05: Hardcoded default secrets | Still present -- example config has weak defaults |
| H-06: Go toolchain TLS vulnerabilities | Needs re-assessment with current Go version |
