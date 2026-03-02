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
