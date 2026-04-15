## Task

Fix the reported SSO security findings in the Go backend:

- Cross-tenant / unbound custom-SSO account takeover
- Custom-provider SSRF through token and userinfo endpoints
- SSO login CSRF / browser session swapping

## Approach And Key Decisions

- Added browser-bound SSO state cookies and PKCE so a callback only completes in the browser that initiated the flow.
- Bound callback login to organization ownership and stored provider identity instead of trusting global email alone.
- Treated custom providers as higher-risk than built-in providers: existing custom-provider users must already be linked to a provider identity before login succeeds.
- Validated custom provider URLs before saving and again before use, while still allowing local callback fixtures in `test` / `development`.
- Forced OAuth token exchange through the app HTTP client context so runtime SSRF protections are applied to both token and userinfo requests.

## Files Modified

- `internal/handlers/sso_handlers.go`
- `internal/handlers/sso_handlers_test.go`
- `internal/handlers/sso_types.go`
- `internal/handlers/sso_utils.go`
- `internal/handlers/testhelpers_test.go`
- `internal/handlers/sso_security.go` (new)

## Dependencies Or Env Changes

- No new dependencies.
- Test helper now sets `App.Environment = "test"` so custom-provider URL validation can allow local mock OAuth servers in tests while production remains locked down.

## Tests Added / Updated

- Updated init tests to assert browser-bound state cookie + PKCE challenge generation.
- Updated callback tests to store browser token / PKCE verifier in Redis state and present the required cookie.
- Added regressions for:
  - mismatched browser state cookie rejection
  - configured HTTP client usage during token exchange
  - rejecting unlinked existing custom-provider users
  - rejecting cross-tenant existing users
  - rejecting private custom-provider URLs in production
- Updated existing custom-provider success coverage to require an already linked identity.

## Verification Results

- `gofmt -w internal/handlers/sso_security.go internal/handlers/sso_types.go internal/handlers/sso_utils.go internal/handlers/sso_handlers.go internal/handlers/testhelpers_test.go internal/handlers/sso_handlers_test.go`
- `go test ./internal/handlers -run 'Test.*SSO'` ✅
- `go test ./internal/handlers -run 'Test.*(SSO|Webhook|Security|Media|Auth|Middleware)'` ✅
- `go test ./internal/handlers` ✅
- `go test ./...` ❌ unchanged pre-existing root build failure:
  - `./tmp_encrypt.go:8:6: main redeclared in this block`
  - `./tmp_arabic.go:8:6: other declaration of main`

## Known Limitations

- Repo-wide `go test ./...` is still blocked by the pre-existing duplicate root helper binaries in `tmp_*.go`.
- Existing custom-provider accounts now need a stored provider binding to log in; this is intentional to prevent arbitrary identity takeover from admin-controlled custom IdPs.
