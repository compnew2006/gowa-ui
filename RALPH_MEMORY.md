## [2026-04-05] Issue: MkDocs Config Value 'docs_dir' Error

- **The Trap:** Setting `docs_dir: .` in `mkdocs.yml` to keep all files in a single flat directory for simplicity.
- **The Reality:** MkDocs 1.6+ and the `mkdocs-static-i18n` plugin (with `docs_structure: folder`) often fail to resolve this because they expect a dedicated sub-directory for source files, or they collide with the configuration file itself when scanning the root.
- **The Fix:** Created a dedicated `docs/` subdirectory inside `docs/wiki/`, moved all markdown source folders (`en/`, `ar/`) into it, and updated `mkdocs.yml` to `docs_dir: docs`.
- **The Law:** Always separate your `mkdocs.yml` from your Markdown source files by placing the latter in a `docs/` subdirectory; never use `docs_dir: .` for production-grade, multi-plugin documentation projects.

## 2026-03-17 00:00 Issue: Assignment permissions ignored chat.assign


- The Trap: Assuming contact assignment was controlled by `contacts:write` and role names, so `chat.assign:write` changes in `/settings/roles` would work.
- The Reality: The backend and frontend checked different permission keys, so role changes did not grant assignment; instance restrictions were also not enforced on assignments.
- The Fix: Centralized assignment authorization on `chat.assign:write` (or `contacts:write` fallback) and enforced assignee instance access for contacts and transfers, with UI filtering on allowed instance IDs.
- The Law: Align UI and server authorization checks to the same permission keys, and enforce access restrictions server-side even when the UI filters lists.

## [2026-03-01] Issue: Mask phone numbers in chat messages

- **The Trap:** Focusing only on the REST API response (`buildMessagesResponse`) to apply data modifiers like phone number masking.
- **The Reality:** Real-time applications concurrently stream state updates via WebSockets (`broadcastNewMessage`), completely bypassing the HTTP response formatters.
- **The Fix:** Duplicated the `MaskPhoneNumbersInText` masking logic directly inside the WebSocket payload factory `messages.go:broadcastNewMessage`.
- **The Law:** Always verify if a data mutation must be applied to both the HTTP REST rendering pipeline AND the WebSocket real-time event pipeline.

## [2026-03-01] Issue: Chat New Message Collision

- **The Trap:** I assumed `fetchContact` only fetched and stored contact metadata in the background, unaware that it unconditionally overwrites the active UI state (`currentContact`).
- **The Reality:** When a WebSocket message arrives from an unknown contact, `contactsStore.fetchContact` is triggered in the background, immediately hijacking the active screen.
- **The Fix:** Modified `fetchContact` to selectively update `currentContact` only if the fetched ID matches the currently active ID. Added an E2E test.
- **The Law:** Background data fetches must never mutate active UI focus state without explicit comparison against the currently viewed entity.

## [2026-03-01] Issue: Chat Image Load Scroll Jump

- **The Trap:** When opening a chat, `scrollToBottom` correctly positions the user, but subsequently loading images append height, pushing the scroll window back up relatively.
- **The Reality:** The browser maintains `scrollTop` without an `overflow-anchor: auto` effect available, meaning any async block-level expansion shifts the viewport.
- **The Fix:** Bound native `@load` listeners onto all chat `<img>` renders that re-trigger an instant `scrollToBottom` _only if_ the user's viewport is still near the bottom when the event fires.
- **The Law:** Async-rendered media inside a reverse-chronological view must strictly preserve scroll anchor intent (bottom-pinning) via explicit resize/load handlers.

## [2026-03-02] Issue: Build failure after handler refactoring

- **The Trap:** Splitting large files (auth.go, sso.go) based on surface-level usage, assuming all types and methods were moved correctly.
- **The Reality:** Significant handlers (CreateRegisterInvite) and specific request types (SwitchOrgRequest, LogoutRequest) were missed, leading to build errors in cmd/main.go and missing symbols in handlers.
- **The Fix:** Restored missing handlers from the original file (via git), defined missing request types in auth_types.go, and verified with a full project build.
- **The Law:** Never delete the original monolithic or critical file until a full project build (go build ./cmd/...) confirms no undefined symbols or broken dependencies.

## [2026-03-02] Issue: Low test coverage due to skipped database tests

- **The Trap:** Assuming `go test ./...` provides a comprehensive view of quality.
- **The Reality:** Many critical database and repository tests are silently skipped if `TEST_DATABASE_URL` is not provided, masking potential regressions in persistence logic. 
- **The Fix:** Conducted a comprehensive audit of tests, identifying 0% coverage in `internal/database` and `internal/contactutil` due to environment constraints.
- **The Law:** Always verify test skip conditions in CI/CD and local environments; "Green" tests do not guarantee safety if critical paths are skipped.

## [2026-03-02] Issue: XSS in v-html and SQLi in dynamic column filters

- **The Trap:** Using Vue's `v-html` with naive string replacements for HTML escaping and dynamically forming SQL `WHERE` clauses from JSON keys without explicitly strict regular expression whitelists.
- **The Reality:** GORM query builders and basic HTML replacement chains are routinely vulnerable to injection (XSS/SQLi) if user-provided keys or data are interpolated or unhandled efficiently. Even when utilizing `DOMPurify.sanitize()`, `v-html` still presents security surface area and code smell.
- **The Fix:** Removed `v-html` and `DOMPurify` entirely from Vue components (`CampaignsView.vue`, `TemplatesView.vue`). Replaced them with custom regex parsers (`parseFormatPreview`, `parseTemplateParams`) that output an array of distinct tokens, which are safely rendered via `<template v-for>` using Vue's native HTML interpolation defenses.
- **The Law:** Never trust user input with `v-html` or dynamic SQL interpolation; use native structured parsing or standard escaping mechanisms.

## [2026-03-02] Issue: DSN Injection and IPv6 Formatting Bugs in Database Connection Config

- **The Trap:** Constructing database connection strings using `fmt.Sprintf` with raw configuration metrics, assuming the user's password and hostname will never contain spaces, special characters, or be in an IPv6 format.
- **The Reality:** Standard string formatting fails completely when passwords contain special characters (like `@` or `?`) or if an IPv6 address lacks `[]` encapsulation, leading to connection failures or DSN injection attacks.
- **The Fix:** Replaced `fmt.Sprintf` with standard library URL builders (`url.URL` for Postgres) and host/port combinations (`net.JoinHostPort` for Redis and Postgres) to automatically format connection strings appropriately.
- **The Law:** Always use native standard library builders (such as `url.URL` or `net.JoinHostPort`) for construction of URLs or network addresses instead of raw string interpolation.
- **The Law:** Never output user data inside `v-html` directives; always construct arrays of atomic data chunks and render them using Vue's safe interpolation syntax `{{ }}` sequentially to maintain styling contexts securely.

## [2026-03-05] Issue: WebSocket Connection Dropped on Auth Subprotocol

- **The Trap:** Changing the frontend to send the WebSocket authentication token via the `Sec-WebSocket-Protocol` header (e.g., `Sec-WebSocket-Protocol: whm.v1, auth.token`) without configuring the backend `fastHTTPUpgrader` to echo back the agreed subprotocol.
- **The Reality:** According to the WebSocket RFC, if the client sends a list of subprotocols, the server MUST explicitly select and return one in the HTTP 101 Switching Protocols response (e.g., `Sec-WebSocket-Protocol: whm.v1`). If it doesn't, strict clients (browsers, Playwright) instantly terminate the connection.
- **The Fix:** Explicitly set `up.Subprotocols = []string{"whm.v1"}` in the `fastHTTPUpgrader` configuration right before calling `Upgrade()`.
- **The Law:** Always echo the negotiated `Sec-WebSocket-Protocol` during the WS handshake if the client requests one, otherwise the client will inevitably sever the connection.

## [2026-03-11] Issue: Nil pointer dereference in database migrations

- **The Trap:** Assuming the `*gorm.DB` connection passed to migration and seeding functions is always initialized and non-nil.
- **The Reality:** Unit tests often pass nil ORM instances or simulate edge cases where the database connection fails early, leading to immediate panics during `AutoMigrate` or `RunMigrationWithProgress`.
- **The Fix:** Added explicit `if db == nil { return fmt.Errorf("database connection is nil") }` checks to all major database lifecycle functions in `postgres.go`.
- **The Law:** Never assume a shared dependency like a database handle is non-nil in lifecycle or migration logic; always guard against initialization failures.

## [2026-03-11] Issue: Flaky Redis tests due to hardcoded port conflicts

- **The Trap:** Hardcoding `Addr: "localhost:6379"` in Redis client tests, assuming the port is always available and points to the `miniredis` mock.
- **The Reality:** Parallel test execution or existing local Redis instances cause connection collisions, leading to "address already in use" or tests accidentally connecting to a real local database instead of the mock.
- **The Fix:** Created a `getMockRedisConfig` helper that dynamically retrieves the random port assigned by `miniredis.Run()` and injects it into the client configuration.
- **The Law:** Always use dynamic port allocation for mock services in unit tests to ensure isolation and prevent environmental cross-contamination.

## [2026-03-30] Issue: Frontend E2E tests target wrong port or blank screens

- **The Trap:** Relying on the development server (localhost:8080) for automated frontend testing, assuming it's identical to the production build.
- **The Reality:** The dev server often has different routing, HMR overhead, or proxy behaviors that can cause hydration/blank screen issues (404 on valid routes or blank pages) under automated load.
- **The Fix:** Switched to the production preview server (`npm run preview` on port 3000). This provides a stable, minified environment that more accurately reflects the user experience and reduces Vite-side hydration timing issues.
- **The Law:** Always use `npm run build && npm run preview` (production mode) when running comprehensive automated E2E or AI-driven tests to ensure environment stability and realistic performance.
## [2026-03-30] Issue: Frontend Blank Screens from Permission Mismatches

- **The Trap:** Frontend permission checks (`hasPermission`) only matched `Permission` objects (resource/action). However, some backend responses or creation workflows use raw string keys like `"resource:action"`.
- **The Reality:** When the frontend encountered a string instead of an object, the check failed, triggering an unauthorized redirect to a potentially unauthorized "blank" route.
- **The Fix:** Refactor `authStore.hasPermission` in `auth.ts` to handle both object and string formats using a centralized `targetKey` mapping.
- **The Law:** Always design permission guards to handle both granular objects and flat string keys to ensure resilience across different API synchronization stages.

## [2026-04-01] Issue: Integrating PostHog into a Vue 3/Vite project

- **The Trap:** Initializing PostHog directly in `main.ts` without checking for environment variables, which can lead to console errors or app crashes in environments where PostHog is not yet configured.
- **The Reality:** Vite's `import.meta.env` requires explicit typing in `env.d.ts` for safety, and PostHog's automatic pageview tracking can sometimes conflict with single-page app (SPA) routers if not handled via a navigation guard.
- **The Fix:** Created a fault-tolerant initialization utility in `src/lib/posthog.ts` that includes a `DEV` mode warning and integrated a `router.afterEach` guard in `main.ts` for consistent page-level capture.
- **The Law:** Always use a dedicated initialization utility with environment guards for third-party analytics to ensure the core application remains resilient even if the service is missing or fails.

## 2026-04-02 19:08 Issue: Removing public pricing routes without a handoff seam

- The Trap: Deleting `PricingLandingView.vue` and the `/pricing` aliases directly, assuming the marketing sidecar would be wired everywhere in the same rollout.
- The Reality: `/pricing`, `/plans`, and `/offer` are stable public entry URLs and part of the lead-capture boundary, so hard deletion would create broken links and a brittle migration.
- The Fix: Replaced the routes with a configurable sidecar-handoff view, removed the old page content, and generalized lead-source validation while keeping lead storage/admin handling in the monolith.
- The Law: When moving public pages to a sidecar, preserve the public URLs first and migrate ownership through a redirect or proxy seam instead of a hard delete.
