# Security audit snapshot - 2026-05-08

Report-only audit saved to `summary.md` on branch `agent/security-audit-dead-code-scan`.

Top publish blockers found:
- `frontend/src/views/auth/LoginView.vue` stores raw remembered login passwords in `localStorage` under `auth:remembered_login_credentials`.
- `cmd/whatomate/main.go` `loadAndValidateConfig` does not call `config.ValidateDatabaseCredentials` or `config.ValidateWebhookVerifyToken`, even though both exist in `internal/config/security_validation.go`.
- `internal/handlers/custom_actions.go` `executeWebhookAction` validates stored webhook URLs only at create/update; it does not revalidate the post-`replaceVariables` URL before `http.NewRequest`.
- `executeJavaScriptAction` accepts `jsResult["url"]` into `ActionResult.RedirectURL`; `frontend/src/views/chat/ChatView.vue` opens arbitrary http/https URLs, bypassing the validated one-time redirect token used by URL actions.
- `ServeMedia` writes `message.MediaFilename` directly into `Content-Disposition`; outbound upload stores raw `fileHeader.Filename` as `MediaFilename`.

Medium/quality issues:
- CSP still has `style-src 'self' 'unsafe-inline'`.
- API key validation still globally queries by key prefix before optional org-header check.
- Observability allows tokenless loopback access when enabled and no access token is configured.
- Raw `security_reports/gitleaks*.json` and `semgrep*.json` files are present; gitleaks reports contain many `Secret`/`Match` fields.
- CSV export escapes only leading `=` and `@`, not `+`, `-`, tab, or CR.
- `config.example.toml` has invalid `license.enabled = enable`.
- `go test ./...` fails if local ignored `tmp/` exists, due multiple scratch `main` functions.
- Dead-code candidates with no Serena references: `OrganizationContext`, `GetUser`, `GetOrganization`, `IsSuperAdmin` in `internal/middleware/middleware.go`.

Verification snapshot:
- `go test ./cmd/... ./internal/... ./pkg/... ./test/...` passed.
- `go test ./...` failed only because local `tmp/` package had multiple `main` functions.
- Frontend unit tests passed: 31 files, 149 tests.
- Frontend typecheck failed with existing TS errors.
- ESLint non-fixing run passed with one unused warning.
- Playwright auth smoke failed because backend was not running on localhost:8080.

Confirmed fixed from prior report:
- Object storage local path traversal hardening is present.
- CSRF comparison uses constant-time compare.
- SSO callback does not reflect raw error_description.
- CSP is no longer skipped for SPA routes.
- CORS only allows configured/allowed origins.
- Login dummy bcrypt hash is present.
