# Whatomate Security and Code Health Audit

Date: 2026-05-08
Branch: agent/security-audit-dead-code-scan
Scope: report-only security and dead-code audit of the current workspace. No source-code fixes were applied in this pass.

## Method

- Used Serena MCP for project file/code reads, symbol lookup, reference tracing, pattern search, and writing this report.
- Used codebase-memory-mcp graph search/snippets for function discovery, caller/callee evidence, and dead-code candidates.
- Applied the security-reviewer and desloppify review lenses. The project-local `.agents/skills-guide.md` was missing and `.agents/skills/*` is ignored by Serena, so the skill bodies could not be loaded from the project. I used the installed skill inventory plus the relevant review workflow instead.
- `ruflo` was requested, but no callable ruflo namespace/tool was exposed by tool discovery. The review used Serena, codebase-memory-mcp, and the available semantic/code graph tooling.
- Serena `initial_instructions`, `project_overview`, `style_and_conventions`, `suggested_commands`, `restart_language_server`, and `done_checklist` are visible in config only as available-but-inactive, so they could not be called directly.
- Assumption: because no answer was received to the clarification question, this pass is an audit report only, not a patch pass.

## Threat Ranking

### CRITICAL

No confirmed critical unauthenticated remote exploit was found in this pass. There are, however, multiple HIGH issues that should block publication until fixed and reverified.

### HIGH

1. Plaintext password persistence in browser localStorage

Evidence:
- `frontend/src/views/auth/LoginView.vue:31` defines `auth:remembered_login_credentials`.
- `frontend/src/views/auth/LoginView.vue:33-36` includes `password` in the remembered credential shape.
- `frontend/src/views/auth/LoginView.vue:68-84` loads the password from `localStorage` into the login form.
- `frontend/src/views/auth/LoginView.vue:109-120` writes the raw password to `localStorage` after login.

Impact: any XSS, malicious browser extension, shared workstation user, browser sync leak, or local malware gets the user's real account password, not just an app-scoped session token. This is worse than token theft because the password may be reused and can outlive the app session.

Recommendation: remove password persistence entirely. If needed, remember only the email address and rely on browser password managers/WebAuthn/passkeys for password fill.

2. Production config validators exist but are not called on normal server/worker startup

Evidence:
- `cmd/whatomate/main.go:129-147` calls `ValidateJWTSecret`, `ValidateEncryptionKey`, `ValidateDefaultAdmin`, and `ValidateLicenseConfig` only.
- `internal/config/security_validation.go:44-62` implements `ValidateDatabaseCredentials` for insecure production DB users/passwords.
- `internal/config/security_validation.go:65-84` implements `ValidateWebhookVerifyToken` for empty/placeholder production webhook tokens.
- `config.example.toml` still documents `database.user = "change-me"`, `database.password = "change-me"`, and `whatsapp.webhook_verify_token = "change-me"`.

Impact: a production deployment can start with placeholder database credentials or webhook verify token even though validators exist. For a publish/release gate, this is a hard blocker.

Recommendation: call both validators from `loadAndValidateConfig`, add tests that production startup fails on placeholders, and fix `config.example.toml` to make insecure examples impossible to copy accidentally.

3. Custom webhook actions validate the stored URL, but not the final substituted URL

Evidence:
- `internal/handlers/custom_actions.go:640-671` validates `config.url` on create/update.
- `internal/handlers/custom_actions.go:385-460` runs `replaceVariables(config.URL, ctxData)` and passes the result directly to `http.NewRequest`.
- `internal/handlers/custom_actions.go:463-504` shows URL actions do the safer thing: they validate `finalURL` after variable replacement.
- `internal/handlers/webhooks.go:21-52` contains the SSRF URL validator that is skipped at webhook execution time.

Impact: an allowed custom-action template can become a different URL at execution after contact/user/org variable substitution. Runtime `SSRFSafeDialer` helps with private IPs, but it does not replace policy validation of the final URL string and hostname suffix. It also still permits exfiltration to attacker-controlled public URLs.

Recommendation: call `validateWebhookURL(url)` immediately after variable replacement in `executeWebhookAction`, before `http.NewRequest`. Consider restricting which variables may appear in host/scheme portions.

4. Server-side JavaScript custom actions can return arbitrary redirect URLs directly to the frontend

Evidence:
- `internal/handlers/custom_actions.go:508-578` extracts `jsResult["url"]` and assigns it directly to `result.RedirectURL`.
- `frontend/src/views/chat/ChatView.vue:1371-1384` opens any resulting http/https URL with `window.open`.
- Normal URL actions use a one-time redirect token and final URL validation in `internal/handlers/custom_actions.go:463-504`; JavaScript actions bypass that path.
- `internal/handlers/custom_actions.go:285-364` allows custom action execution for users with either `custom_actions:write` or `chat:write`.

Impact: a configured JavaScript action can turn the app into a trusted launcher for arbitrary external URLs. Depending on who can configure or trigger these actions, this can be used for phishing or policy bypass.

Recommendation: route JavaScript-returned URLs through the same `validateWebhookURL` plus one-time redirect-token flow as URL actions, or require returned URLs to be relative app paths.

5. WebSocket token is short-lived but exposed to JavaScript and sent in the WebSocket subprotocol

Evidence:
- `internal/handlers/auth_handlers.go:495-526` returns a 30-second JWT in a JSON response with `Cache-Control: no-store`.
- `frontend/src/services/websocket.ts:312-316` sends the token as `auth.<token>` in `Sec-WebSocket-Protocol` and again in the auth message payload.

Impact: the current design is much improved by short TTL and no-store, but the token is still visible to XSS and may be logged by reverse proxies or WebSocket infrastructure that records subprotocol headers.

Recommendation: make the token one-time-use server-side, avoid logging `Sec-WebSocket-Protocol`, consider binding it to the authenticated cookie/session, and prefer a handshake that does not place bearer material in commonly logged headers.

6. Uploaded/chat media filenames can be replayed into Content-Disposition without header-safe encoding

Evidence:
- `internal/handlers/contacts_messaging.go:510-512` stores `fileHeader.Filename` as `MediaFilename` for outgoing media.
- `internal/handlers/media.go:239-240` and `internal/handlers/media.go:268-269` interpolate `message.MediaFilename` into `Content-Disposition` as `inline; filename="%s"`.

Impact: path storage is safe because media files are saved with UUID names, but response headers still use the original filename. Quotes, control characters, or unusual bytes can produce broken headers or browser-specific behavior. fasthttp may mitigate CRLF injection, but the code should not rely on that.

Recommendation: reuse `sanitizeFilename` or encode with a proper `filename*=` strategy before writing `Content-Disposition`.

### MEDIUM

1. CSP still allows inline styles

Evidence:
- `internal/middleware/middleware.go:33` sets `style-src 'self' 'unsafe-inline'`.
- `internal/middleware/middleware.go:254` applies that policy to non-skipped routes.

Impact: this does not create XSS by itself, but it weakens browser containment once any HTML/style injection exists.

Recommendation: remove inline style dependence, use nonces/hashes where needed, and keep CSP route coverage.

2. API key lookup remains global before optional organization match

Evidence:
- `internal/middleware/middleware.go:376-424` looks up active keys by prefix globally, then checks `X-Organization-ID` only if that header is present.

Impact: header mismatch is correctly rejected and absent org header sets the request org from the key, so I did not confirm direct tenant escape. The lookup is still less strict than a tenant-scoped design and depends on prefix filtering plus bcrypt verification across candidate keys.

Recommendation: require `X-Organization-ID` for API-key auth or add an explicit tenant-scoped lookup path when the caller supplies an org. Keep the constant-time comparison.

3. Observability can be tokenless on loopback

Evidence:
- `internal/observability/observability.go:137-169` allows `/metrics` and `/debug/pprof/*` with no token when `ctx.RemoteIP()` is loopback.
- `config.example.toml` has observability disabled by default, which is good.

Impact: if a reverse proxy, sidecar, tunnel, or local process exposes these endpoints, pprof and metrics can leak sensitive runtime data.

Recommendation: require `observability.access_token` whenever metrics or pprof are enabled in production.

4. Generated secret-scan reports preserve raw match/secret fields

Evidence:
- `security_reports/gitleaks_latest.json`, `security_reports/gitleaks_worktree_latest.json`, `security_reports/gitleaks_worktree_fresh_2026-03-04.json`, and `security_reports/gitleaks.json` contain many `Secret`/`Match` fields.
- `semgrep_summary.json`, `semgrep_results.json`, `semgrep_latest.json`, and `semgrep_secrets.json` are present in the project root.

Impact: even if many entries are examples or false positives, publishing raw scanner JSON can leak actual secrets and creates recurring scanner noise.

Recommendation: do not publish raw secret-scan reports. Redact `Secret`/`Match`, store reports outside the repo, or keep only summarized sanitized findings.

5. CSV formula injection protection is intentionally incomplete

Evidence:
- `internal/handlers/import_export.go:323-331` escapes only leading `=` and `@`.
- The comment explicitly skips `+` and `-` because they can be legitimate phone numbers or negative values.

Impact: spreadsheet software can treat leading `+`, `-`, tab, or carriage return as formula triggers. A malicious exported field may execute as a formula when opened by staff in Excel/LibreOffice/Sheets.

Recommendation: escape all known formula prefixes globally, or special-case phone number columns with safer display formatting while still escaping other columns.

### LOW / HYGIENE

1. `config.example.toml` has invalid TOML for license enablement

Evidence:
- `config.example.toml:15` uses `enabled = enable` rather than a boolean or quoted string.

Impact: copy/paste deployment from the example can fail before startup. This is not directly exploitable but is a release-quality issue.

Recommendation: use a valid boolean, for example `enabled = true` or `enabled = false`, matching the intended default.

2. `go test ./...` is broken by ignored local repro files in `tmp/`

Evidence:
- `go test ./...` fails in package `github.com/compnew2006/whatomate/tmp` with multiple `main redeclared` errors involving `tmp/gorm_reuse_repro.go`, `tmp/gorm_reuse_dryrun.go`, and `tmp/inspect_org_save.go`.
- `.gitignore` ignores `tmp/`, but the Go tool still includes a present local `tmp` directory in `./...`.

Impact: local and CI verification can fail depending on workspace contents. It also undermines confidence in final publish checks.

Recommendation: delete local repro files, move them outside the module, or add `//go:build ignore` to standalone scratch programs.

3. Unused middleware helper candidates

Evidence:
- codebase-memory graph showed zero inbound references for several non-test helpers.
- Serena `find_referencing_symbols` returned no references for:
  - `internal/middleware/middleware.go:426-465` `OrganizationContext`
  - `internal/middleware/middleware.go:537-540` `GetUser`
  - `internal/middleware/middleware.go:543-546` `GetOrganization`
  - `internal/middleware/middleware.go:549-552` `IsSuperAdmin`

Impact: low runtime risk, but these helpers can confuse future auth/middleware changes because they imply context-loading paths that are not used.

Recommendation: confirm they are not public API, then remove with `safe_delete_symbol` in a cleanup branch.

4. Frontend lint has one unused variable warning

Evidence:
- `frontend/e2e/tests/chat/soft-delete.spec.ts:11` has unused `loginAsAdmin`.

Impact: minor hygiene issue.

Recommendation: remove the unused import/variable or use it.

## Confirmed Prior Fixes / Strong Controls

- Object storage local path traversal appears fixed. `internal/storage/object_storage.go:70-95` resolves, cleans, and checks local object paths under the storage root.
- CSRF token comparison uses constant-time comparison in `internal/middleware/csrf.go` and skips bearer/API-key auth as intended.
- SSO callback no longer reflects raw provider error descriptions; it uses a fixed message and URL escaping.
- CSP is now applied to SPA routes; skip logic is limited to API, WebSocket, and static asset paths.
- CORS now only returns allow headers for allowed origins.
- Login uses a real bcrypt dummy hash for nonexistent users and marks auth responses `Cache-Control: no-store`.
- WebSocket upgrade validates `Origin` and requires the short-lived WS token before upgrade.
- Import/export SQL uses table/column whitelists and bind arguments.
- Widget raw SQL paths use server-side whitelists for table, column, grouping, and filter fields.
- Webhook URL validation plus `SSRFSafeDialer` provide good baseline SSRF protection for normal webhook paths.

## Verification Results

Commands were run via Serena `execute_shell_command` except for one process cleanup after Playwright exceeded the tool timeout and left child processes running; no source code was read with shell.

- `git status --short --branch`: workspace already had pre-existing dirty/untracked changes before this report, including `.desloppify/*` and several `cmd/whatomate/*.go` files. I did not revert or stage those changes.
- `git checkout -b agent/security-audit-dead-code-scan`: passed.
- `go test ./...`: failed because the local ignored `tmp/` package contains multiple standalone `main` files with redeclared `main`.
- `go test ./cmd/... ./internal/... ./pkg/... ./test/...`: passed.
- `cd frontend && npm run typecheck`: failed with existing TypeScript errors, mainly readonly test fixture arrays, missing component custom properties in tests, deep type instantiation in `use-toast.ts`, `content.body` typed as `{}`, and non-exported store types used by views.
- `cd frontend && npx eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts --ignore-path .gitignore`: passed with one warning (`loginAsAdmin` unused in `frontend/e2e/tests/chat/soft-delete.spec.ts`). I intentionally did not run the repo script `npm run lint` because it includes `--fix` and would rewrite files outside Serena.
- `cd frontend && npm run test:unit`: passed, 31 test files and 149 tests.
- `cd frontend && npx playwright test --project=chromium`: attempted, but exceeded Serena's 120s tool timeout and left Playwright child processes. Those were stopped before continuing.
- `cd frontend && CI=1 BASE_URL=http://localhost:8080 npx playwright test --project=chromium --reporter=list --global-timeout=30000`: completed but produced more output than Serena returned.
- `cd frontend && CI=1 BASE_URL=http://localhost:8080 npx playwright test e2e/tests/auth/login.spec.ts --project=chromium --reporter=line --global-timeout=30000`: failed all 9 tests because no backend was listening on `localhost:8080`; global setup and page navigation hit `ECONNREFUSED`.

## Files Modified / Created / Deleted

Modified:
- `summary.md` only.

Created:
- none.

Deleted:
- none.

No dependencies, migrations, or environment variables were changed. No tests were added because this was a report-only audit pass.

## Publish Recommendation

Do not publish this app yet. Patch the HIGH items first, remove or isolate the `tmp/` repro files so `go test ./...` is trustworthy, fix frontend typecheck, and rerun Playwright with the backend running on `localhost:8080` or the correct `BASE_URL`.

Suggested first patch order:
1. Remove password storage from `LoginView.vue`.
2. Call production DB/webhook validators from `loadAndValidateConfig` and fix `config.example.toml`.
3. Revalidate final webhook custom-action URLs after variable substitution.
4. Route JavaScript custom-action URLs through the validated redirect-token flow.
5. Sanitize/encode `Content-Disposition` filenames.
6. Clean local `tmp/` repro files or add ignore build tags.
