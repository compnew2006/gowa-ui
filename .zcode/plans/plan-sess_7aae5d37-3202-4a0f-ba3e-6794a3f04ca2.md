# P0 + P1 Architecture Refactor — Plan

Two contained, independently-shippable refactors. Each is gated by tests and fully reversible per file. The graph's `affected` command guards every step.

**Scope decisions (per my judgment, you deferred):**
- **P0 migrates ONLY `internal/handlers/chat_lifecycle.go`** — strangler pattern. The duplicated logic in `chatbot_processor.go` (lines 200-243, literally copy-pasted twice) and the third assign path in `contacts.go:1585-1610` become an *obvious* follow-up once the service exists, but expanding blast radius into unexplored files now is the big-bang anti-pattern. The new service API is designed so that follow-up is mechanical.
- **P1 is a mechanical 3-way split** of `ChatView.vue` preserving behavior exactly. The two cross-region reach-throughs (`replyToMessage`→composer focus, `sendMessage`→timeline scroll) are preserved via parent-owned orchestration, not rewritten.

---

## Part A — Blast Radius Analysis (per architecture-guardian skill)

```
Target: 9 handler methods in internal/handlers/chat_lifecycle.go + createSystemMessage + joinAsCollaborator
Directly affected:
  - internal/handlers/chat_lifecycle.go (rewritten as thin HTTP adapters)
  - cmd/whatomate/main.go:749-757 (handlers stay on *App; route registration unchanged)
  - internal/chatlifecycle/ (NEW package)
Indirectly affected (verified by grep — ZERO external callers):
  - none. The 9 handlers are referenced only in main.go routes. createSystemMessage has 2 external
    callers in chatbot_processor.go:218,240 — kept working via a delegating *App method (step B.4.5).
Risk level: MEDIUM (logic move across a package boundary, but no signature changes, no new callers)
Safe to proceed: YES — reversible per-file, tests gate each step, graph's `affected` guards dependents.
```

```
Target: frontend/src/views/chat/ChatView.vue (3,685 lines)
Directly affected:
  - ChatView.vue (becomes ~1,200-line layout shell + state owner)
  - frontend/src/components/chat/ChatSidebar.vue (NEW)
  - frontend/src/components/chat/ChatTimeline.vue (NEW)
  - frontend/src/components/chat/ChatComposer.vue (NEW)
Indirectly affected: none (no other component imports ChatView; it's a route target)
Risk level: MEDIUM (mechanical extraction, behavior-preserving; reach-throughs preserved via parent)
Safe to proceed: YES — npm run build + npm run typecheck gate every step.
```

---

## Part B — P0: Extract `internal/chatlifecycle/` service

### B.1 New package shape (mirrors `internal/assignment/` exactly)

`internal/chatlifecycle/service.go`:
```go
package chatlifecycle

type Service struct {
    db    *gorm.DB
    wsHub *websocket.Hub   // nil-safe (precedent: calling.NewManager takes *websocket.Hub)
    log   logf.Logger
}

func New(db *gorm.DB, wsHub *websocket.Hub, log logf.Logger) *Service {
    return &Service{db: db, wsHub: wsHub, log: log}
}
```

Wired in `cmd/whatomate/main.go` next to `app.Assigner = assignment.New(...)`:
```go
app.ChatLifecycle = chatlifecycle.New(db, app.WSHub, lo)
```
Field added to `*App` (`internal/handlers/app.go`, next to `Assigner`):
```go
ChatLifecycle *chatlifecycle.Service
```

### B.2 The API — pure decision + side-effect methods, no `*fastglue.Request`

Each method takes already-resolved inputs (org/user IDs, the contact, target user) and returns a typed result. HTTP parsing, `requireAuth`, `parsePathUUID`, envelope shaping **stay in the handler**. The service owns: business rule, mutation, persistence, system message, audit, WS broadcast.

```go
// ClaimResult tells the handler which HTTP path to take.
type ClaimResult int
const (
    ClaimDone         ClaimResult = iota  // assigned; emit success envelope
    ClaimAlreadySelf                      // idempotent self-assign
    ClaimRerouteJoin                      // assigned to other + caller has collaborate → join instead
    ClaimConflictOther                    // assigned to other, no collaborate → 409
)

func (s *Service) Claim(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact,
    hasCollaboratePerm bool) (ClaimResult, *models.User, error)

func (s *Service) Release(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact,
    isAssignee, isAdminOrManager bool) (released bool, err error)
//   returns ErrClosedReleaseByAgent, ErrNotAuthorized, or nil (idempotent → released=false, err=nil)

func (s *Service) Close(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) error
func (s *Service) Reopen(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) (bool, error)
func (s *Service) Join(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) (JoinResult, error)
func (s *Service) Invite(ctx context.Context, orgID, inviterID, targetID uuid.UUID, contact *models.Contact) (InviteResult, error)
func (s *Service) Leave(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) (LeaveResult, error)
func (s *Service) RemoveCollaborator(ctx context.Context, orgID, actorID, targetID uuid.UUID, contact *models.Contact) error

// BulkRelease processes a batch; returns per-item results. Loop body delegates to a private
// releaseOne() used by both Release and BulkRelease so the logic is single-sourced.
type BulkResult struct { ReleasedIDs []string; Failed []map[string]any }
func (s *Service) BulkRelease(ctx context.Context, orgID, userID uuid.UUID, ids []string,
    isAdminOrManager bool) BulkResult

// CustomerReopen reopens a closed chat on inbound customer message. NEW method that subsumes the
// duplicated block in chatbot_processor.go:200-243 — API ready now, wired as a follow-up.
func (s *Service) CustomerReopen(ctx context.Context, orgID uuid.UUID, contact *models.Contact) (reopened bool)

// CreateSystemMessage is exported so chatbot_processor.go's 2 call sites can migrate off the
// *App method later. For now the handler keeps a delegating *App.createSystemMessage.
func (s *Service) CreateSystemMessage(orgID, contactID uuid.UUID, content string, metadata models.JSONB)
```

**Typed sentinel errors** (handler maps to HTTP, exactly mirroring today's status codes):
```go
var (
    ErrNotFound             = errors.New("chat: contact not found")
    ErrNotAuthorized        = errors.New("chat: not authorized")
    ErrClosedReleaseByAgent = errors.New("chat: closed chat requires admin to release")
    ErrAlreadyCollaborator  = errors.New("chat: already a collaborator")
    ErrCannotRemoveOwner    = errors.New("chat: cannot remove owner")
    ErrNotCollaborator      = errors.New("chat: not a collaborator")
)
```

### B.3 What moves where

| Concern | Stays in handler (HTTP adapter) | Moves to service |
|---|---|---|
| `requireAuth` / `getOrgAndUserID` / `HasPermission` | ✅ (needs `*fastglue.Request`) | — |
| `parsePathUUID`, body unmarshal, `r.RequestCtx.UserValue` | ✅ | — |
| DB lookup `a.DB.Where(...).First(&contact)` | ✅ (handler owns the 404) | — |
| Business rule (can actor X do Y? new state?) | — | ✅ |
| `SetStatus`/`ClearCollaborators`/`AssignedUserID` mutation | — | ✅ |
| `oldStatus`/`oldAssigned` pre-mutation capture | — | ✅ (the JSONB-aliasing hazard lives here) |
| GORM persist (`Updates`/`Save`) | — | ✅ |
| `createSystemMessage` | — | ✅ (`s.CreateSystemMessage`) |
| `audit.LogAudit` + `extraChanges` safeguard | — | ✅ (calls `audit.LogAudit` + `audit.GetUserName` directly; loses nothing vs. the `*App.logAudit` wrapper) |
| `wsHub.BroadcastToOrg` | — | ✅ (nil-guarded; precedent: `calling.Manager`) |
| `r.SendEnvelope` / `r.SendErrorEnvelope` | ✅ | — |

### B.4 Migration sequence (file-by-file, test-gated)

1. **Create `internal/chatlifecycle/service.go`** with struct, `New`, typed errors, and `Release` + `CustomerReopen` + `CreateSystemMessage` methods only. Add **co-located `internal/chatlifecycle/service_test.go`** with pure unit tests for `Release` (the same 5 cases as the handler test, but testing the service directly without `*App`). Uses a real test DB via `testutil.SetupTestDB` but **no Redis** — proving the Postgres-only testability the refactor promised.
2. **Add `App.ChatLifecycle` field + wire in `main.go`.** Build green. No handler change yet.
3. **Rewrite `ReleaseChat` handler** as a ~25-line HTTP adapter: parse → auth → lookup → `s.Release(ctx, ...)` → map result/error to envelope. Run `go test ./internal/handlers/ -run TestApp_ReleaseChat` — all 5 must still pass (proves behavior preservation). Run `graphify affected "ReleaseChat"` to confirm zero unexpected dependents.
4. **Migrate the remaining 8 handlers** one at a time, same shape. After each: `go build ./... && go vet ./...`. After all 8: full handler test suite.
5. **Move `createSystemMessage` and `joinAsCollaborator` into the service.** Keep a thin `*App.createSystemMessage` delegator so `chatbot_processor.go:218,240` keeps compiling — those 2 call sites are explicitly out of scope (strangler) but must not break. Delegator carries a `// TODO: migrate chatbot_processor to s.CreateSystemMessage` comment.
6. **Add unit tests** for the previously-untested handlers (`Claim`, `Close`, `Reopen`, `Leave`, `Join`, `Invite`, `RemoveCollaborator`) — the exploration flagged this coverage gap. This is where the refactor pays off: impossible to test without `*App`+Redis before.

### B.5 Risk mitigations (P0)

- **JSONB-aliasing hazard** (status lives in `Metadata` map; struct snapshots alias the mutated map): the service captures `oldStatus`/`oldAssigned` into locals **before** mutation — same pattern as today. The audit `extraChanges` map is built from these locals.
- **`extraChanges` safeguard** (audit silently no-ops on empty diff): preserved exactly. Unit test `TestService_Release_PersistsAuditEntry` asserts the row exists.
- **WS nil-guard**: `if s.wsHub != nil { ... }` — matches today's `if a.WSHub != nil`. Tests pass `nil`.
- **No behavior change**: the 7 existing tests (5 Release + 2 BulkRelease) are the regression contract. They must pass unchanged — same assertions, same DB state. If any flips, the migration is wrong.

---

## Part C — P1: Split `ChatView.vue` into 3 children

### C.1 Target structure (mirrors `ContactInfoPanel.vue` conventions)

```
frontend/src/views/chat/ChatView.vue          ~1,200 lines (layout shell + state owner)
  ├─ frontend/src/components/chat/ChatSidebar.vue    ~650 lines
  ├─ frontend/src/components/chat/ChatTimeline.vue    ~900 lines
  └─ frontend/src/components/chat/ChatComposer.vue    ~750 lines
```

All three under `frontend/src/components/chat/` (where `ContactInfoPanel`, `ConversationNotes` live). Explicit imports (no auto-import — verified, `vite.config.ts` has no `unplugin-vue-components`).

### C.2 Props/emits contracts (tuple-form `defineEmits`, per `ContactInfoPanel.vue:73-76`)

**ChatSidebar.vue** — reads `contactsStore` directly (per ContactInfoPanel precedent: children use shared stores for reads + local mutations, emit only what the parent must sync):
```ts
defineProps<{ modelValue?: string }>()
defineEmits<{
  select: [contactId: string]            // parent routes (replaces today's router.push in sidebar)
  bulkRelease: [contactIds: string[]]
}>()
```
Owns: `tabStripRef`, `onTabKeydown`, `tabLabel`, `contactsScroll` (useInfiniteScroll), sidebar resize, tag-filter popover, add-contact dialog, bulk-select state (reads `contactsStore.bulkSelectMode`/`selectedContactIds`; calls `contactsStore.bulkReleaseChats` directly per the "child may mutate via store" convention).

**ChatTimeline.vue**:
```ts
defineProps<{ account: string | null }>()  // parent-owned multi-account (drives revoke visibility)
defineEmits<{
  reply: [message: Message]
  retry: [message: Message]
  reaction: [messageId: string, emoji: string]
}>()
defineExpose({ scrollToBottom, resetScroll })
```
Owns: `messagesEndRef`, `messagesScroll`, `isAtBottom`, `stickyDate`/`showStickyDate`, `brokenMediaIds`, all render helpers (`getMessageContent`, `getSystemMessageText`, `formatMessageTime`, etc.), `useMediaBurst`. Reads `contactsStore.messages`/`currentContact` directly.

**ChatComposer.vue**:
```ts
defineProps<{ account: string | null }>()
defineEmits<{ sent: [] }>()
defineExpose({ focus })  // parent calls after reply (precedent: CannedResponsePicker.vue:68, FlowBuilder.vue:345)
```
Owns: `messageInput`, `messageInputRef`, `fileInputRef`, `emojiPickerOpen`, all canned/template/media dialog state, typing indicator, `useHeaderMedia`. Reads `contactsStore.currentContact`/`replyingTo` directly; calls `contactsStore.sendMessage`/`sendTemplate`/`addMessage`.

### C.3 The two cross-region reach-throughs (preserved, lowest risk)

1. **`replyToMessage` (timeline) → focus composer**: timeline emits `reply(message)`; parent sets `contactsStore.setReplyingTo(message)` then calls `chatComposerRef.value?.focus()`. `defineExpose({ focus })` on composer.
2. **`sendMessage` (composer) → scroll timeline**: composer emits `sent`; parent calls `chatTimelineRef.value?.scrollToBottom()`. Timeline `defineExpose({ scrollToBottom })`.

Both reach-throughs already work today through parent-owned refs. The split keeps the *parent* as orchestrator — same control flow, componentized. No new coupling.

### C.4 Parent (`ChatView.vue`) keeps ownership of

- `currentContact` lifecycle (`selectContact`, `watch(contactId)`, `onUserActive`, route clearing)
- `selectedAccount` (multi-account) — passed as prop to timeline + composer; account-tabs strip + `switchAccount` stay in parent header
- Header actions (Release/Leave/Reopen/Close/Invite/Assign/Resume/custom-actions) — header NOT extracted (4th region; out of scope to keep blast radius flat)
- Right-side panels (`ContactInfoPanel`, `ConversationNotes`) — already separate, unchanged
- All dialogs (template/canned/assign/invite/media) — stay in parent (triggered from header + composer; follow-up)
- WS wiring: `wsService.setCurrentContact` stays in parent (3 sites). WS handlers in `services/websocket.ts` mutate the store — **zero change** to WS behavior.
- `watch(() => contactsStore.messages.length)` for unread-pill + auto-scroll — stays in parent, calls `chatTimelineRef.scrollToBottom()` via defineExpose.

### C.5 Migration sequence (mechanical, behavior-preserving)

1. **Create `ChatSidebar.vue`** with sidebar template (lines 2037-2402) + sidebar-owned script symbols. Wire in ChatView via `<ChatSidebar @select="handleContactClick" />`. `npm run build && npm run typecheck` green. Sidebar routes via parent emit → `router.push` (unchanged).
2. **Create `ChatTimeline.vue`** with messages region (lines 2691-3205) + timeline-owned symbols + `useMediaBurst`. Wire via `<ChatTimeline :account="selectedAccount" @reply="onTimelineReply" />`. Parent's `onTimelineReply` sets replying-to + calls `chatComposerRef.focus()`.
3. **Create `ChatComposer.vue`** with composer form (lines 3241-3318) + composer-owned symbols + `useHeaderMedia`. Wire via `<ChatComposer :account="selectedAccount" @sent="chatTimelineRef?.scrollToBottom()" />`.
4. After each step: `npm run build && npm run typecheck` green. After all three: full `npm run build`.
5. **No new tests for P1** — no Vue component test framework exists (no Vitest config). Contract = build+typecheck green + manual smoke (contact select, send, reply-to-focus, scroll-on-send, tab switch, bulk-release). Flagging this honestly, not blocking.

### C.6 Risk mitigations (P1)

- **Reactivity regressions**: every child reads `contactsStore.*` directly (same as `ContactInfoPanel.vue` today) — reactivity preserved because the store is the same singleton, not copied via props. "Stale data after split" risk is near-zero.
- **Scroll/composable lifecycle**: `useInfiniteScroll` instances move with their owning child. Parent's `selectContact` currently calls `messagesScroll.cleanup()`/`.setup()` across contact switch — becomes `chatTimelineRef.value?.resetScroll()` via defineExpose. Same control flow.
- **Bundle size**: net-neutral or slightly smaller (better tree-shaking). No new dependencies.
- **CSS**: each child's `<style scoped>` carries only its own rules. The `.sticky-date-*` and `.chat-sidebar` rules move to their owning child.

---

## Part D — Verification gates (after each phase)

```bash
# Backend (P0)
go build ./... && go vet ./...
TEST_DATABASE_URL="postgres://postgres@localhost:5432/whatomate_test?sslmode=disable" \
TEST_REDIS_URL="redis://localhost:6379/0" \
go test ./internal/chatlifecycle/ -count=1 -v       # NEW pure unit tests
go test ./internal/handlers/ -run 'TestApp_ReleaseChat|TestApp_BulkReleaseChats' -count=1  # regression
graphify affected "ReleaseChat"                     # confirm zero unexpected dependents

# Frontend (P1)
cd frontend && npm run typecheck && npm run build

# Phase 6 — graph refresh (mandatory: >5 files + new symbols)
graphify update .
```

**Regression contract (must not change):**
- The 7 existing `TestApp_ReleaseChat_*` / `TestApp_BulkReleaseChats_*` tests pass with identical assertions.
- `go build`, `go vet`, `npm run typecheck`, `npm run build` all exit 0.
- i18n parity unchanged.
- Manual: claim/release/close/reopen/join/leave/invite/remove behave identically in the UI.

---

## Part E — Explicitly OUT of scope (strangler boundaries)

- **NOT migrating** `chatbot_processor.go:200-243` (duplicated customer-reopen) — `Service.CustomerReopen` API is ready, wiring is a follow-up.
- **NOT migrating** `contacts.go:1585-1610` (third assign path).
- **NOT extracting** the ChatView header (4th region) or dialogs.
- **NOT adding** Vitest / component tests — no framework exists; that's a prerequisite task.
- **NOT touching** the toast import cycle or `whatsapp_account` FK integrity (P3/P4 from the review; deferred).

Each is a clean, small follow-up once the service + 3 children exist. The strangler pattern makes the *next* refactor cheaper, not this one all-encompassing.

---

## Part F — Skeptical Review (per architecture-guardian skill)

- **The Plan:** Extract a chatlifecycle service (P0) and split ChatView into 3 children (P1).
- **The Critic:** *"Two medium-risk refactors back-to-back. If the service extraction subtly breaks the audit `extraChanges` safeguard or the JSONB-status capture, the bug is silent (audit entries just stop persisting). And a 3,685-line Vue split is the kind of mechanical task where a misplaced `v-if` or forgotten emit silently breaks reply-to-focus or scroll-on-send — and there are no component tests to catch it."*
- **The Defense:** *(1) The 7 existing handler tests ARE the P0 regression contract — they assert the exact DB state including the audit row, so a broken safeguard fails the test immediately. (2) The new pure unit tests add coverage for the 7 previously-untested handlers, so P0 leaves test coverage strictly better. (3) For P1, the contract is build+typecheck green + a documented manual smoke checklist; I am NOT claiming automated UI coverage that doesn't exist. (4) Both refactors are reversible per-file — if P1's timeline split misbehaves, revert that one file; P0 is unaffected. (5) `graphify affected` runs after P0 to catch any dependent the plan missed."*

If at any point a verification gate fails and 5 fix-iterations don't resolve it, I stop and report (per the architecture-guardian skill's loop-limit rule) rather than pushing through.