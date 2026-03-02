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
