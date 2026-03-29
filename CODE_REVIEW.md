# Code Review: Whatomate

**Date**: 2026-03-29  
**Reviewer**: Automated Code Review  
**Scope**: Full-stack (Go backend + Vue 3 frontend)  

## Summary

Full-stack WhatsApp customer service platform (Go backend + Vue 3 frontend). The codebase shows mature architecture overall — cookie-based auth with CSRF, WebSocket reconnection with backoff, Pinia stores, and lazy-loaded routes. However, there are significant issues: **data races in the WebSocket layer**, **JWT algorithm confusion allowing potential auth bypass**, and a **3500-line god-component** in the frontend.

**Verdict**: [x] Request Changes

---

## Critical Issues (Must Fix)

### 1. [hub.go:156 / client.go:326] Data race on `Client.currentContact`

Hub goroutine reads `client.currentContact` under only map `RLock`, while `handleSetContact` writes it with **no synchronization**. This is undefined behavior.

**Fix**: Add a `sync.Mutex` to `Client` protecting `currentContact`, or use `atomic.Pointer[uuid.UUID]`.

### 2. [messages.go:259-430] Data race on `msg.InstanceID` in async goroutine

`SendOutgoingMessage` spawns a goroutine that mutates `msg.InstanceID` while the caller still reads it.

**Fix**: Copy needed fields into locals before spawning the goroutine. Do not mutate the shared struct.

### 3. [auth_utils.go:141, auth_handlers.go:519] JWT algorithm confusion — no `WithValidMethods`

`validateRegisterInviteToken` and `Logout` use `jwt.ParseWithClaims` **without** restricting the signing algorithm. An attacker could craft a token with `alg: none` to bypass verification.

```go
// Current (vulnerable)
token, err := jwt.ParseWithClaims(tokenString, &RegisterInviteClaims{}, func(t *jwt.Token) (interface{}, error) {
    return a.jwtSecretBytes()
})

// Fix: add WithValidMethods
token, err := jwt.ParseWithClaims(tokenString, &RegisterInviteClaims{}, func(t *jwt.Token) (interface{}, error) {
    return a.jwtSecretBytes()
}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
```

### 4. [contacts.go:661-678] N+1 query — 50+ DB round-trips per contacts page load

Each contact triggers a separate `COUNT` query for unread messages.

**Fix**: Batch with a single `GROUP BY contact_id` query.

### 5. [media.go:217] Unguarded type assertion panics

```go
messageIDStr := r.RequestCtx.UserValue("message_id").(string) // panics if nil
```

**Fix**: Use comma-ok assertion with error response.

### 6. [auth.ts:66-98] User/permissions trusted from localStorage without server verification

`restoreSession()` reads user data including roles/permissions from localStorage and uses them immediately. `refreshUserData()` runs in background but stale/tampered data is active before it completes.

**Fix**: Await `refreshUserData()` before returning `true`, or only store a minimal identifier.

---

## Major Issues (Should Fix)

### Backend

| # | File | Issue |
|---|------|-------|
| M1 | `contacts.go:1172` | `markMessagesAsRead` loads ALL unread messages into memory — OOM risk |
| M2 | `hub.go:234,239` | `Register`/`Unregister` block on unbuffered channel — potential deadlock |
| M3 | `contacts.go:329,647` | DB queries use `context.Background()` or no context — not cancelled on disconnect |
| M4 | `main.go:502-504` | Worker `messageProvider` stays nil for Meta provider — nil pointer panic |
| M5 | `worker.go:411-452` | `checkCampaignCompletion` race — two workers can both mark complete |
| M6 | `worker.go:382,389` | `updateRecipientStatus` and `incrementCampaignCount` silently swallow DB errors |
| M7 | `contacts.go:664` | Write operation (repair + DELETE) inside a GET list endpoint |
| M8 | `webhook.go:132-152` | JSON body parsed twice — wasteful, potential TOCTOU |

### Frontend

| # | File | Issue |
|---|------|-------|
| M9 | `ChatView.vue` | 3500+ line god-component — should be decomposed into composables + child components |
| M10 | `websocket.ts` | `disconnect()` never called on logout — WebSocket leaks across sessions |
| M11 | `websocket.ts:11-15` | Module-level audio/notification state never cleaned up |
| M12 | `instances.ts:109-122` | Health polling interval never auto-stopped |
| M13 | `websocket.ts:354-413` | `handleMessage` catch block swallows ALL errors, not just parse errors |
| M14 | `contacts.ts:762-811` | Race condition in `fetchChats` — concurrent calls overwrite each other |
| M15 | `transfers.ts:95-100` | `myTransfers` reads `user_id` from localStorage (never set) — always empty |
| M16 | `api.ts` | Pervasive `any` types in service layer — no compile-time API contract checking |
| M17 | `ChatView.vue:1881-1914` | Two deep watchers on large arrays cause expensive reactive traversal |
| M18 | `websocket.ts:880-884` | No message queue during WebSocket disconnection — messages silently dropped |
| M19 | `api.ts:559-573` | `uploadMedia` uses raw `axios` instead of configured `api` instance — bypasses interceptors |

---

## Minor Issues

| # | File | Issue |
|---|------|-------|
| m1 | Multiple auth_handlers.go | Duplicate permission hydration logic (3 implementations) — extract helper |
| m2 | `postgres.go:278` | Manual `repeatChar` loop instead of `strings.Repeat` |
| m3 | Throughout | Magic numbers: `256` (channel buffer), `4096` (max msg), `5s`/`30s` timeouts |
| m4 | `hub.go:142-145` | Broadcast silently drops messages when client buffer full |
| m5 | `main.go:681` | `len(path) >= 13` prefix check — use `strings.HasPrefix` |
| m6 | `auth.ts` + `api.ts` | Duplicate `Permission` and `User` interface definitions across files |
| m7 | All stores | Inconsistent `response.data.data \|\| response.data` unwrapping — use `unwrapResponse` utility |
| m8 | `websocket.ts` reconnect | No jitter on exponential backoff — thundering herd on server restart |

---

## Positive Feedback

- Cookie-based auth with httpOnly cookies + CSRF token is well-implemented
- WebSocket reconnection with exponential backoff is solid
- Race condition handling via sequence counters in message fetching is a good pattern
- Clean separation between stores, services, composables, and lib utilities
- Proper test factories and E2E test infrastructure with Playwright
- Lazy-loaded routes for code splitting
- Comprehensive i18n support with RTL (Arabic)

---

## Test Coverage Assessment

- [x] Unit tests present (stores, composables, lib utilities)
- [x] E2E tests comprehensive (30+ spec files)
- [ ] Backend handler tests coverage unclear
- [ ] WebSocket integration tests needed (data race conditions untested)
- [ ] Missing tests for the critical worker campaign completion logic

---

## Top 3 Immediate Actions

1. **Fix JWT `WithValidMethods`** (C3) — one-line fix, direct auth bypass vector
2. **Add synchronization to WebSocket `currentContact`** (C1) — undefined behavior in production
3. **Batch N+1 unread count query** (C4) — immediate user-facing performance improvement

---

## Detailed Frontend Findings

### SECURITY

#### CRITICAL-SEC-1: User object stored unencrypted in localStorage

**File:** `frontend/src/stores/auth.ts` (lines 66, 79-96, 114, 159-160)

The entire user object (including `organization_id`, `role_id`, permissions, settings) is serialized to localStorage as plaintext JSON. While auth tokens are properly handled via httpOnly cookies, the user profile data includes role/permission information that a malicious script or XSS could tamper with to escalate privileges client-side.

Line 98 does call `refreshUserData()` in the background, but the stale localStorage data is used immediately for permission checks before the refresh completes. If the refresh fails silently (line 118), the stale/tampered data persists.

**Fix:** Wait for `refreshUserData()` to complete before returning `true` from `restoreSession()`. Consider only storing a minimal user identifier in localStorage and fetching full user data from the server on restore.

#### MAJOR-SEC-2: No CSRF token validation on initial login/register

**File:** `frontend/src/services/api.ts` (lines 71-82)

The CSRF token is only read for POST/PUT/DELETE/PATCH requests. Login and register POST endpoints do send the CSRF token if the cookie exists, but the CSRF cookie may not be set yet on first visit.

**Fix:** Verify that the backend sets the CSRF cookie on initial page load or on a dedicated pre-auth endpoint.

### STATE MANAGEMENT

#### MAJOR-STATE-1: WebSocket singleton never properly cleaned up on logout

**File:** `frontend/src/stores/auth.ts` (lines 146-154)

When the user logs out, `authStore.clearAuth()` is called, but `wsService.disconnect()` is never called from the auth store.

**Fix:** Add `wsService.disconnect()` to the `clearAuth()` function.

#### MAJOR-STATE-2: Module-level notification audio element leaks

**File:** `frontend/src/services/websocket.ts` (lines 11-15, 76-95)

Module-level variables `notificationSound`, `notificationSoundPending`, `interactionListenerAttached`, `notificationSourceIndex`, and `activeNotificationSound` are never cleaned up.

**Fix:** Add a `cleanup()` method to the WebSocket service class that clears the audio element, removes event listeners, and resets all module-level state.

#### MAJOR-STATE-3: Health polling interval never auto-stopped

**File:** `frontend/src/stores/instances.ts` (lines 109-122)

`startHealthPolling()` starts a `setInterval` but `stopHealthPolling()` is never called automatically.

**Fix:** Return a cleanup function from `startHealthPolling()`, or use `onScopeDispose`.

#### MINOR-STATE-4: myTransfers computed reads from localStorage instead of auth store

**File:** `frontend/src/stores/transfers.ts` (lines 95-100)

```typescript
const myTransfers = computed(() => {
  const userId = localStorage.getItem('user_id')  // Never set anywhere
  return transfers.value.filter(t => t.status === 'active' && t.agent_id === userId)
})
```

This computed property will always return an empty array.

**Fix:** Use `useAuthStore().user?.id` instead.

### PERFORMANCE

#### MAJOR-PERF-1: ChatView.vue is a 3500+ line god-component

This single file contains 50+ `ref()` declarations, 15+ `computed()` properties, 10+ `watch()` calls, and 40+ functions.

**Fix:** Extract into multiple composable functions and child components:
- `useChatMessages()` — message display, media loading, scrolling
- `useChatInput()` — message input, typing indicators, slash commands
- `useChatSidebar()` — sidebar contacts, filtering, search, resize
- `useChatActions()` — claim, close, reopen, transfer, assign
- `useMediaUpload()` — file selection, upload, preview
- `useBatchPrint()` — print selection and execution

#### MAJOR-PERF-2: Deep watchers on large reactive arrays

**File:** `frontend/src/views/chat/ChatView.vue` (lines 1881-1914)

Two deep watchers observe large arrays causing expensive reactive traversal on every property change.

**Fix:** Watch only specific properties needed (e.g., `messages.length` instead of deep-watching the entire array).

#### MAJOR-PERF-3: Double watcher on messages triggers redundant work

**File:** `frontend/src/views/chat/ChatView.vue` (lines 1861-1901)

Two separate watchers both call `loadMediaForMessages()` and `resolveMentionsForCurrentMessages()`. When a new message is added, both fire, causing duplicate work.

**Fix:** Merge into a single watcher or debounce the second one.

### ERROR HANDLING

#### MAJOR-ERR-1: Unhandled promise rejection in store actions

**File:** `frontend/src/stores/contacts.ts` (lines 997-1005)

`claimChat`, `closeChat`, `reopenChat`, and `setChatPublic` do NOT have try/catch blocks unlike other store actions.

**Fix:** Add try/catch to all store actions that make API calls.

#### MAJOR-ERR-2: WebSocket message handler silently swallows all errors

**File:** `frontend/src/services/websocket.ts` (lines 354-413)

The catch block swallows errors from not just JSON parsing, but from ALL handler methods.

**Fix:** Separate JSON parsing from message handling; only swallow parse errors.

### TYPESCRIPT

#### MAJOR-TS-1: Pervasive `any` type in API service layer

**File:** `frontend/src/services/api.ts` (multiple lines)

Multiple service methods use `any` for request/response types, eliminating TypeScript's ability to catch type mismatches.

**Fix:** Define proper interfaces for all request/response types.

#### MINOR-TS-2: Duplicate Permission interface definitions

**File:** `frontend/src/stores/auth.ts` (lines 22-27) and `frontend/src/services/api.ts` (lines 1329-1335)

Two different `Permission` interfaces exist with overlapping but different fields.

**Fix:** Create a shared `types/permission.ts` file.

#### MINOR-TS-3: Duplicate User interface definitions

**File:** `frontend/src/stores/auth.ts` (lines 37-48) and `frontend/src/stores/users.ts` (lines 13-26)

Both define a `User` interface with overlapping but different fields.

**Fix:** Create a shared `types/user.ts` file.

### WEBSOCKET

#### MAJOR-WS-1: No message queue during disconnection

**File:** `frontend/src/services/websocket.ts` (lines 880-884)

Messages are silently dropped when the WebSocket is disconnected. The `setCurrentContact()` call sends a `set_contact` message that is lost on disconnect.

**Fix:** Queue messages sent while disconnected and replay them on reconnect.

#### MINOR-WS-2: WebSocket reconnect delay lacks jitter

**File:** `frontend/src/services/websocket.ts` (lines 860-871)

All clients will hit the server at the same intervals during mass reconnection events.

**Fix:** Add random jitter: `delay + delay * 0.3 * Math.random()`

### API INTEGRATION

#### MAJOR-API-1: Inconsistent API response unwrapping pattern

Every store repeats `const data = response.data.data || response.data` or casts to `any` first.

**Fix:** Use the `unwrapResponse` and `unwrapItemResponse` utilities from `api-utils.ts` consistently.

#### MAJOR-API-2: Race condition in concurrent contact/chat fetches

**File:** `frontend/src/stores/contacts.ts` (lines 762-811)

`fetchChats()` fires three concurrent API requests but if called twice rapidly, both calls share the same `isLoading` ref and the second call's results overwrite the first's.

**Fix:** Use a sequence counter to ignore stale results.

#### MINOR-API-3: uploadMedia uses raw axios instead of api instance

**File:** `frontend/src/services/api.ts` (lines 559-573)

Bypasses the response interceptor (which handles 401 token refresh), the request interceptor, and the configured timeout.

**Fix:** Use the `api` instance directly with CSRF header.

---

## Detailed Backend Findings

### CRITICAL

#### C1: Race condition on `Client.currentContact`

**File:** `internal/websocket/hub.go:156`, `internal/websocket/client.go:326`

`Hub.broadcastMessage()` reads `client.currentContact` while holding only `RLock` on the clients map, but `Client.handleSetContact()` writes `c.currentContact = &contactID` with no synchronization. The Hub's `Run` goroutine and the client's `ReadPump` goroutine access this field concurrently.

**Fix:** Add a mutex to `Client` protecting `currentContact`, or use `atomic.Pointer[uuid.UUID]`.

#### C2: Race condition: `msg.InstanceID` mutated in async goroutine

**File:** `internal/handlers/messages.go:259-306`, `internal/handlers/messages.go:425-430`

`SendOutgoingMessage` spawns an async goroutine that calls `sendViaProvider`, which mutates `msg.InstanceID`. Meanwhile, the caller reads `msg.InstanceID` and passes `msg` to `broadcastNewMessage`.

**Fix:** Copy `msg.ID` and `msg.InstanceID` into local variables before spawning the goroutine.

#### C3: JWT algorithm confusion — missing `WithValidMethods`

**File:** `internal/handlers/auth_utils.go:141-143`, `internal/handlers/auth_handlers.go:519-521`

The auth middleware correctly enforces HS256, but `validateRegisterInviteToken` and `Logout` use `jwt.ParseWithClaims` without `jwt.WithValidMethods`. An attacker could craft a token with `alg: none` or `alg: RS256` to bypass signature verification.

**Fix:** Add `jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})` to both calls.

#### C4: N+1 query in `ListContacts`

**File:** `internal/handlers/contacts.go:661-678`

For every contact in the paginated list, a separate COUNT query is executed. With 50 contacts per page, this produces 50+ DB round-trips.

**Fix:** Batch the unread count using a single query with `GROUP BY contact_id` and `WHERE contact_id IN (?)`.

#### C5: Potential panic in `ServeMedia`

**File:** `internal/handlers/media.go:217`

```go
messageIDStr := r.RequestCtx.UserValue("message_id").(string)
```

If `message_id` is nil or not a string, this panics.

**Fix:** Use comma-ok assertion with proper error response.

### MAJOR

#### M1: Unbounded allocation in `markMessagesAsRead`

**File:** `internal/handlers/contacts.go:1172-1174`

Loads ALL unread messages into memory. A contact with thousands of unread messages causes massive allocation.

**Fix:** Only query the `whats_app_message_id` column and add a LIMIT, or process in batches.

#### M2: Hub Register/Unregister block on unbuffered channel

**File:** `internal/websocket/hub.go:234,239`

If the Hub's `Run` loop is processing a large broadcast, the unbuffered send blocks the caller's goroutine.

**Fix:** Use buffered channels or add select-with-default.

#### M3: Missing request context propagation in DB queries

**File:** `internal/handlers/contacts.go:329,647` and throughout handlers

DB queries use `context.Background()` or no context at all. When the client disconnects, queries continue running.

**Fix:** Extract context from fasthttp request and propagate to all DB calls.

#### M4: Worker missing MessageProvider for Meta provider

**File:** `cmd/whatomate/main.go:502-504`

When `cfg.WhatsApp.Provider != "whatsmeow"`, the `messageProvider` variable stays nil. Any job that tries to use it will panic.

**Fix:** Initialize a Meta adapter for the worker path.

#### M5: Race condition in `checkCampaignCompletion`

**File:** `internal/worker/worker.go:411-452`

Two workers could simultaneously see `pendingCount == 0` and both mark the campaign complete.

**Fix:** Use a CAS pattern with `WHERE id = ? AND status = 'processing'` and check `RowsAffected`.

#### M6: Silently swallowed DB errors in worker

**File:** `internal/worker/worker.go:382,389`

`updateRecipientStatus` and `incrementCampaignCount` ignore return values.

**Fix:** Log errors from these operations.

#### M7: Write operation inside GET list endpoint

**File:** `internal/handlers/contacts.go:664`

`repairDirectContactPhoneFromConversation` executes a database transaction (including DELETE) inside a GET endpoint.

**Fix:** Move repair logic to a background job or separate endpoint.

#### M8: Webhook body parsed twice

**File:** `internal/handlers/webhook.go:132-152`

The body is parsed into `webhookSignaturePayload` first for signature validation, then parsed again into `WebhookPayload`.

**Fix:** Parse once and reuse the result.

---

## Findings Summary

| Severity | Category | Backend | Frontend | Total |
|----------|----------|---------|----------|-------|
| CRITICAL | Security | 2 | 1 | 3 |
| CRITICAL | Data Race | 2 | 0 | 2 |
| CRITICAL | Performance | 1 | 0 | 1 |
| MAJOR | Security | 0 | 1 | 1 |
| MAJOR | Performance | 0 | 3 | 3 |
| MAJOR | State Management | 0 | 3 | 3 |
| MAJOR | Error Handling | 1 | 2 | 3 |
| MAJOR | Concurrency | 3 | 1 | 4 |
| MAJOR | Code Quality | 4 | 1 | 5 |
| MAJOR | TypeScript | 0 | 1 | 1 |
| MINOR | Various | 8 | 10 | 18 |
| **Total** | | **21** | **27** | **48** |
