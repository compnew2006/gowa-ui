# Chat Realtime Message and Notification Fix

Date: 2026-05-10
Branch: `agent/chat-realtime-notifications`

## Task

Investigate `http://localhost:8080/chat/e3945298-ea4d-4f2f-890f-e03f30ab7db6`, where incoming messages were reported as not appearing and incoming-message notifications were not visible.

## Findings

- Browser reproduction on the running `localhost:8080` app showed the route loads, `/api/chats/e3945298-ea4d-4f2f-890f-e03f30ab7db6/messages` returns two messages, and the DOM contains the rendered outgoing message text.
- The notification gap was confirmed in frontend logic: active-chat incoming messages played sound but intentionally suppressed toast notifications, so a user already viewing the conversation saw no visible notification.
- The notification bell counted unread conversations from `contactsStore.sortedContacts`, which is scoped to the active sidebar tab. Direct-linked pending/unassigned chats can be visible in the main panel while omitted from the assigned-tab sidebar and therefore omitted from the notification bell.
- Realtime `new_message` events did not consistently carry `whatsapp_account` from backend payload through frontend store insertion, while initial message fetches are account-aware via `/messages?account=...`.

## Changes

- `frontend/src/services/websocket.ts`
  - Always shows a toast for eligible incoming messages, including when the user is already viewing that chat.
  - Preserves `whatsapp_account` on incoming realtime messages.
  - Added missing imports for `chatsService`, `unwrapResponse`, and `MessagesListPayload` used by missed-message replay.
- `frontend/src/components/NotificationBell.vue`
  - Builds unread chat notifications from all loaded chat buckets (`contacts`, `pendingChats`, `assignedChats`, `closedChats`) instead of only the active sidebar tab.
  - Deduplicates unread contacts and sorts them by latest message time.
- `internal/handlers/messages.go`
  - Includes `whatsapp_account` in outbound realtime `new_message` WebSocket payloads.
- `internal/handlers/chatbot_message.go`
  - Includes `whatsapp_account` in inbound realtime `new_message` WebSocket payloads.
- `frontend/src/services/websocket.test.ts`
  - Added coverage proving active-chat incoming messages append to the active thread and still show a toast.
  - Verifies `whatsapp_account` is preserved in the realtime message passed to the store.
- `frontend/src/components/NotificationBell.test.ts`
  - Added coverage proving unread pending chats appear in the notification bell even when the sidebar is on Assigned.

## Verification

Passed:

- `npx vitest run src/services/websocket.test.ts src/components/NotificationBell.test.ts` — 8 tests passed.
- `npm run lint` — passed with 4 pre-existing warnings.
- `npm run build` — passed; Vite emitted existing chunk-size/empty-chunk warnings.
- `npx playwright test e2e/tests/chat/chat.spec.ts --project=chromium --workers=1` — 20 tests passed.
- `go test ./internal/handlers -run 'TestApp_(SendMessage|ProcessIncoming|SaveIncoming)|Test.*Message'` — passed.
- `go test ./cmd/... ./internal/... ./pkg/... ./test/...` — passed.

Known pre-existing verification failures:

- `npm run typecheck` still fails on existing TypeScript issues outside this fix, including unused imports in `ChatInputBar.vue`, readonly fixture typing in `ContactInfoPanel.test.ts`, and type/export issues in chatbot/dashboard/settings files. The previous `websocket.ts` missing-import type errors are fixed in this change.
- `go test ./...` fails only because the repo's `tmp` package contains multiple scratch `main` files (`tmp/gorm_reuse_repro.go`, `tmp/gorm_reuse_dryrun.go`, `tmp/inspect_org_save.go`). Application package tests pass when `tmp` is excluded.
- A parallel Playwright chat run failed once because the Vite web server became unreachable after worker timeouts; the same chat spec passed serially with one worker.
- The full Playwright suite exceeded the 120s tool limit before code changes, so final browser verification focused on the chat spec relevant to this task.

## Dependencies and Environment

No dependency or configuration changes were required.

## Limitations

The running `localhost:8080` backend served its already-built frontend assets during initial browser reproduction. The source changes were verified through unit tests, Vite production build, and Playwright against the frontend dev server.
