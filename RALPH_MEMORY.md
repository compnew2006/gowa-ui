## [2026-03-01] Issue: Chat New Message Collision

- **The Trap:** I assumed `fetchContact` only fetched and stored contact metadata in the background, unaware that it unconditionally overwrites the active UI state (`currentContact`).
- **The Reality:** When a WebSocket message arrives from an unknown contact, `contactsStore.fetchContact` is triggered in the background, immediately hijacking the active screen.
- **The Fix:** Modified `fetchContact` to selectively update `currentContact` only if the fetched ID matches the currently active ID. Added an E2E test.
- **The Law:** Background data fetches must never mutate active UI focus state without explicit comparison against the currently viewed entity.
