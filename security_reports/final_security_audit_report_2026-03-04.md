# Whatomate Comprehensive Security Vulnerability Report

Date: 2026-03-04
Scope: Full repository (`backend`, `frontend`, `docs`, configs)
Method: Manual review + static analysis + dependency auditing + secret scanning

## 1) Sub-Agent Team Configuration (20 domain agents)

This assessment was executed as 20 specialized sub-agents, each focused on a distinct security domain:

1. Input Validation Agent
2. Authentication Agent
3. Authorization/RBAC Agent
4. Session/JWT Agent
5. API Security Agent
6. Injection Attacks Agent
7. SSRF/LFI Agent
8. Data Encryption at Rest Agent
9. Data Encryption in Transit/TLS Agent
10. Secret Management Agent
11. Configuration Hardening Agent
12. Logging & Sensitive Data Exposure Agent
13. WebSocket Security Agent
14. File Handling & Storage Agent
15. Dependency Security (Go) Agent
16. Dependency Security (Node/npm) Agent
17. Client-Side Security Agent
18. Third-Party Integrations Agent
19. Rate Limiting & Abuse Controls Agent
20. Supply-Chain/CI Artifact Security Agent

All sub-agent outputs were consolidated and severity-ranked using: Critical, High, Medium, Low, Informational.

## 2) Tooling and Evidence Artifacts

- `gosec`: `security_reports/gosec.json`
- `govulncheck`: `security_reports/govulncheck_latest.txt`, `security_reports/govulncheck_latest_verbose.txt`
- `npm audit`:
  - `security_reports/frontend_npm_audit.json`
  - `security_reports/frontend_npm_audit_prod.json`
  - `security_reports/docs_npm_audit.json`
  - `security_reports/docs_npm_audit_prod.json`
- `gitleaks`: `security_reports/gitleaks.json`, `security_reports/gitleaks.log`

Scan snapshots:
- `gosec`: 20 findings (`HIGH=7`, `MEDIUM=12`, `LOW=1`)
- `govulncheck`: 3 reachable stdlib vulns in current code paths
- npm audits:
  - frontend all deps: 6 high; frontend prod: 0
  - docs all/prod: 1 high (`rollup` path traversal advisory)
- gitleaks: 46 matches, with one high-confidence repo secret in `config.toml`

## 3) Severity-Ranked Findings

## Critical

### C-01: SSRF + Arbitrary Local File Read via `media_url`
- Severity: **Critical**
- CWE: CWE-918 (SSRF), CWE-22/CWE-73 class local file exposure
- Evidence:
  - `internal/handlers/statuses.go:184-194` accepts user `media_url` and forwards to provider send methods.
  - `pkg/whatsmeow/adapter_media_helpers.go:26-38` allows `http`, `https`, `file://`, and local paths.
  - `pkg/whatsmeow/adapter_media_helpers.go:42` uses `http.Get(u)` with no outbound host restrictions.
  - `pkg/whatsmeow/adapter_media_helpers.go:77-79` allows absolute local paths.
  - `pkg/whatsmeow/adapter_media_helpers.go:88` reads file directly from disk.
- Affected components:
  - Status send API: `/api/instances/{id}/status/send`
  - Whatsmeow media helper pipeline
- Exploitation vector:
  - Authenticated user with `chat:write` submits `media_url=file:///etc/passwd` (or app secrets path), causing server-side file read and upload.
  - `media_url=http://169.254.169.254/...` can be used for metadata/internal network probing.
- Impact:
  - Server-side data exfiltration, internal network access, cloud credential disclosure.
- Recommended remediation:
  - Enforce strict allowlist for media sources (prefer only managed uploads/media IDs).
  - Block `file://` and absolute/relative local path inputs for API-supplied URLs.
  - Replace `http.Get` with hardened client: deny private/link-local/loopback CIDRs, require DNS/IP validation, set low timeouts, max body size.
  - Add audit logging and alerting for blocked SSRF attempts.

## High

### H-01: Broken Authorization on Chatbot Admin Endpoints
- Severity: **High**
- CWE: CWE-285 (Improper Authorization)
- Evidence:
  - Endpoints use only org extraction (`getOrgID`) without permission checks:
    - `internal/handlers/chatbot.go:100`, `197`, `412`, `464`, `525`, `558`, `624`, `1039`, `1088`, `1139`, `1159`, `1220`, `1248`, `1275`
  - Contrast: flow endpoints in same file enforce RBAC (`HasPermission`) at `:658`, `:731`, `:838`, `:866`, `:999`.
  - Routes directly exposed in `cmd/whatomate/main.go:858-910`.
- Affected components:
  - Chatbot settings, keyword rules, AI contexts, chatbot sessions APIs.
- Exploitation vector:
  - Any authenticated org member can query or mutate chatbot controls if they lack intended admin privileges.
- Impact:
  - Unauthorized configuration changes, workflow sabotage, sensitive chatbot/session data exposure.
- Recommended remediation:
  - Add explicit permission checks per endpoint (`flows_chatbot` or dedicated chatbot resources/actions).
  - Enforce least privilege matrix (read/write/delete separated).
  - Add regression tests for forbidden access by low-privilege roles.

### H-02: Refresh Endpoint Accepts Non-Refresh JWTs (Token Confusion)
- Severity: **High**
- CWE: CWE-287/CWE-613 class auth/session weakness
- Evidence:
  - `internal/handlers/auth_handlers.go:283-294` parses generic `JWTClaims` for refresh.
  - `internal/handlers/auth_handlers.go:296-303` only checks Redis rotation/revocation when `claims.ID != ""`.
  - Access tokens are generated without JTI: `internal/handlers/auth_utils.go:22-33`.
  - Refresh tokens include JTI: `internal/handlers/auth_utils.go:52-64`.
  - No token-type/subject check for refresh endpoint.
- Affected components:
  - `/api/auth/refresh`
- Exploitation vector:
  - Present valid access token as refresh token in request body; flow can mint new tokens while access token remains valid.
- Impact:
  - Session extension beyond intended token class boundaries.
- Recommended remediation:
  - Add mandatory `token_type=refresh` (or strict `sub`) claim and enforce it.
  - Require JTI presence and active Redis record for all refresh operations.
  - Reject any token without refresh-specific claims.

### H-03: SQL Injection Risk via Unvalidated Widget Aggregation `field`
- Severity: **High**
- CWE: CWE-89
- Evidence:
  - Raw concatenation into SQL select:
    - `internal/handlers/widgets.go:870` `SUM(` + field + `)`
    - `internal/handlers/widgets.go:872` `AVG(` + field + `)`
  - `field` accepted from user without whitelist validation in create/update:
    - `internal/handlers/widgets.go:336`, `413-415`
  - Note: filter fields have regex hardening at `:1253-1280`, but `widget.Field` does not.
- Affected components:
  - Widgets analytics APIs and message metric query path.
- Exploitation vector:
  - User with analytics write capability creates/updates widget field to crafted SQL expression.
- Impact:
  - Query manipulation, potential data exposure or DB denial-of-service.
- Recommended remediation:
  - Replace dynamic field injection with strict enum allowlist per datasource.
  - Prefer GORM-safe column mapping instead of string concatenation.
  - Add unit tests for malicious `field` payloads.

### H-04: AI API Keys Not Encrypted at Rest (Stored/Cached in Plaintext)
- Severity: **High**
- CWE: CWE-312/CWE-922
- Evidence:
  - Model comment claims encrypted: `internal/models/chatbot.go:49`.
  - Update path stores key directly: `internal/handlers/chatbot.go:320-321`.
  - Cache stores key explicitly in Redis payload:
    - `internal/handlers/cache.go:45`, `59`, `78`.
  - No chatbot AI-key `Encrypt`/`Decrypt` path found.
- Affected components:
  - Chatbot settings table (`ai_api_key`) and Redis chatbot settings cache.
- Exploitation vector:
  - DB/Redis read access yields third-party AI credentials.
- Impact:
  - External API account compromise and cost/data exposure.
- Recommended remediation:
  - Encrypt AI API key before persistence using existing app crypto utilities.
  - Decrypt only at execution boundary and avoid caching raw secrets.
  - Rotate existing keys after encryption migration.

### H-05: Insecure Bootstrap Secrets/Credentials in Repository + Default Admin Seeding
- Severity: **High**
- CWE: CWE-798/CWE-321
- Evidence:
  - Repository config includes JWT secret literal: `config.toml:39`.
  - Repository config includes default admin creds: `config.toml:88-89`.
  - Default admin fallback values in code:
    - `internal/config/config.go:321-325`.
  - Auto-create default admin if no users:
    - `internal/database/postgres.go:205-207`, `475-551`.
- Affected components:
  - Initial deployment/bootstrap, token signing, admin auth.
- Exploitation vector:
  - Deploying with defaults or leaked config enables credential stuffing and token forgery scenarios.
- Impact:
  - Full admin takeover, auth bypass possibilities.
- Recommended remediation:
  - Remove committed secrets from tracked config; use env-managed secrets.
  - Force startup failure when default admin/JWT secret values are unchanged.
  - Require one-time setup flow for first admin creation.

### H-06: Reachable Go Stdlib TLS Vulnerabilities (Current toolchain)
- Severity: **High**
- Evidence (`security_reports/govulncheck_latest.txt`):
  - GO-2026-4340 (`crypto/tls`) fixed in Go 1.25.6
  - GO-2026-4337 (`crypto/tls`) fixed in Go 1.25.7
  - Current found version: go1.25.5
- Affected components:
  - Outbound HTTPS/TLS operations (WhatsApp client, uploads, Redis TLS traces)
- Impact:
  - TLS security degradation risks depending on traffic conditions.
- Recommended remediation:
  - Upgrade Go runtime/toolchain to **>=1.25.7** and rebuild all artifacts.

## Medium

### M-01: Missing Permission Check on `GET /api/users/{id}`
- Severity: **Medium**
- CWE: CWE-285
- Evidence:
  - `ListUsers` enforces permission: `internal/handlers/users.go:114-116`.
  - `GetUser` lacks equivalent permission check: `internal/handlers/users.go:195-229`.
  - Route exposed: `cmd/whatomate/main.go:708`.
- Impact:
  - Unauthorized user metadata access within organization scope.
- Remediation:
  - Enforce `users:read` (or owner/self exception logic) in `GetUser`.

### M-02: Sensitive Data Logged (Tokens and Request Body)
- Severity: **Medium**
- CWE: CWE-532
- Evidence:
  - Webhook verify token logged on failure: `internal/handlers/webhook.go:47`.
  - User update decode failure logs raw request body: `internal/handlers/users.go:418`.
  - Redirect token logged on consume error: `internal/handlers/custom_actions.go:363`.
- Impact:
  - Token/credential leakage through log systems.
- Remediation:
  - Mask/redact secrets and avoid logging raw request bodies for auth/user endpoints.

### M-03: Weak Password Policy + No Central Validation Enforcement
- Severity: **Medium**
- CWE: CWE-521
- Evidence:
  - Password change minimum set to 6 chars only: `internal/handlers/users.go:736-737`.
  - Request decoding path lacks validator enforcement: `internal/handlers/app.go:220-227`.
- Impact:
  - Increased brute-force/spray success probability.
- Remediation:
  - Enforce strong policy (length, entropy/complexity, breached-password checks) across all auth flows.

### M-04: Rate Limiting Disabled in Default Config
- Severity: **Medium** (deployment-dependent)
- Evidence:
  - `config.toml:76` sets `rate_limit.enabled = false`.
  - Auth routes can run without limiter when disabled: `cmd/whatomate/main.go:621-623`.
- Impact:
  - Higher brute-force/credential-stuffing risk if deployed as configured.
- Remediation:
  - Enable rate limiting by default in production profile; add startup guardrails.

### M-05: WebSocket Token Transport via Query Parameter
- Severity: **Medium**
- Evidence:
  - Backend accepts token from query first: `internal/handlers/websocket.go:77-80`.
  - Frontend sends token in URL query: `frontend/src/services/websocket.ts:281-283`.
  - WS token is short-lived (`30s`): `internal/handlers/auth_handlers.go:471`.
- Impact:
  - URL token leakage via browser history/proxy/log systems.
- Remediation:
  - Prefer header/cookie-based handshake auth; avoid query token transport.

### M-06: Reachable stdlib `net/url` DoS Vulnerability
- Severity: **Medium**
- Evidence:
  - GO-2026-4341 (`net/url`) in go1.25.5 fixed in go1.25.6 (`security_reports/govulncheck_latest.txt`).
- Impact:
  - Potential memory exhaustion in query parsing paths.
- Remediation:
  - Upgrade Go toolchain to >=1.25.7.

### M-07: Documentation Pipeline Dependency Vulnerability (`rollup`)
- Severity: **Medium** (docs/build pipeline surface)
- Evidence:
  - `security_reports/docs_npm_audit.json` and `_prod.json`: high advisory on `rollup` path traversal (`GHSA-mw96-cpmx-2vgc`).
- Impact:
  - Build pipeline risk if docs build consumes untrusted paths/artifacts.
- Remediation:
  - Upgrade `rollup` to patched version immediately.

## Low

### L-01: Broad File and Directory Permissions for Media Artifacts
- Severity: **Low**
- CWE: CWE-732
- Evidence:
  - `os.MkdirAll(..., 0755)`: `internal/handlers/media.go:29`
  - `os.WriteFile(..., 0644)`: `internal/handlers/media.go:193`
  - Similar permission findings flagged by gosec in additional media handlers.
- Impact:
  - Increased local disclosure risk on multi-user hosts.
- Remediation:
  - Restrict directory/file modes to least privilege (e.g., `0750` and `0600`).

### L-02: Additional Static Findings from `gosec` (Code Quality/Safety)
- Severity: **Low** (triage required)
- Evidence:
  - Integer conversion risks (`G115`), weak RNG (`G404` in non-crypto contexts), request-context usage (`G118`), unchecked error (`G104`).
- Impact:
  - Stability/safety concerns; limited direct exploitability from current evidence.
- Remediation:
  - Triage and patch per finding in `security_reports/gosec.json`.

## Informational

### I-01: npm Audit Findings Mostly in Dev/Test Dependency Chains
- Evidence:
  - frontend prod: 0 vulnerabilities
  - all-deps scans include dev tooling vulns (`minimatch`, `vite/vitest` chains)
- Recommendation:
  - Still patch for CI/build hygiene and future supply-chain safety.

### I-02: Secret Scan Noise with One Confirmed Repo Secret
- Evidence:
  - gitleaks found 46 matches; many from docs/assets and non-runtime artifacts.
  - Confirmed sensitive item in repository config: `config.toml:39`.
- Recommendation:
  - Move all secrets to environment/secret manager and re-scan with tuned allowlists.

## 4) Third-Party Library Security Summary

Go ecosystem:
- Immediate action: upgrade Go toolchain to >=1.25.7.
- Additional non-reachable findings in verbose report:
  - `filippo.io/edwards25519` advisory (imported package result)
  - stdlib module advisory (`archive/zip`) not shown reachable by symbols

Node ecosystem:
- Application runtime packages are mostly clean in prod scans.
- Build/docs chains still carry high/moderate advisories and should be updated before publication.

## 5) Best-Practice Compliance Snapshot

Strengths observed:
- Cookie auth has `HttpOnly` for access/refresh and `SameSite=Lax` handling (`internal/handlers/cookies.go`).
- CSRF double-submit protection exists for cookie-auth mutating routes (`internal/middleware/csrf.go`).
- Many endpoints properly enforce RBAC via `requirePermission`/`HasPermission`.

Gaps requiring correction before publication:
- SSRF/LFI protections for media fetch path
- Missing RBAC checks on chatbot and specific user endpoint
- Token-type enforcement on refresh
- Secret-at-rest handling (AI API keys)
- Insecure default credential/secret bootstrap practices

## 6) Prioritized Remediation Plan (Publication Blockers First)

P0 (must fix before release):
1. Patch SSRF/LFI in media URL handling.
2. Add RBAC checks to all chatbot settings/keywords/AI/sessions endpoints.
3. Enforce refresh-token type + JTI requirement in `/api/auth/refresh`.
4. Fix SQL injection vector in widget `field` aggregation.
5. Remove committed secrets/default credentials and hard-fail on insecure defaults.
6. Upgrade Go toolchain to >=1.25.7.

P1:
1. Encrypt chatbot AI API keys at rest and stop caching plaintext keys.
2. Add permission check for `GET /api/users/{id}`.
3. Remove/ redact sensitive logs.
4. Enable auth rate limiting in production defaults.

P2:
1. Move WS token auth away from query parameters.
2. Tighten file permissions for media storage.
3. Update docs/dev dependency chains (`rollup`, `minimatch`, `vitest/vite` paths).

## 7) Overall Risk Assessment

Current security posture before publication: **High Risk**

Primary drivers:
- One critical exploit path (SSRF + local file read)
- Multiple high-impact auth/RBAC and token/session weaknesses
- Hardcoded/default secret hygiene issues
- Reachable TLS/runtime vulnerabilities tied to current Go version

Recommendation: **Do not publish until all P0 items are remediated and re-verified with repeat scans.**
