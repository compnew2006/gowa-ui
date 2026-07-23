# Tasks: Chat Status, Claim & Collaboration System

**Input**: Design documents from `/specs/001-chat-claim-collaboration/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md  
**Tests**: Not requested — manual testing per quickstart.md  
**Organization**: Tasks grouped by user story (4 stories from spec.md)

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3, US4)
- All paths relative to repository root `whatomate/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No project initialization needed — existing Go + Vue project. This phase verifies the feature branch and reviews integration points.

- [ ] T001 Verify feature branch `001-chat-claim-collaboration` is checked out and clean
- [ ] T002 Review `internal/handlers/contacts.go:258-276` (GetMessages handler) to confirm exact insertion point for privacy guard
- [ ] T003 Review `internal/handlers/chatbot_processor.go:137-194` (processIncomingMessageFull) to confirm GetOrCreateContact call location for pending logic

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core model types and permissions that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create `internal/models/chat_status.go` with ChatStatus type (pending/open/closed constants), `EffectiveStatus()` method reading from `Contact.Metadata["chat_status"]` (defaults to "open" if absent), `SetStatus()` method writing to Metadata, `Collaborator` struct (UserID, Name, Role, JoinedAt), and collaborator helper methods on Contact: `GetCollaborators()`, `IsCollaborator(userID)`, `AddCollaborator(user)`, `RemoveCollaborator(userID)`
- [X] T005 Add `ResourceChatCollaborate = "chat.collaborate"` constant in `internal/models/roles.go` after line 72 (`ResourceChatAssign`)
- [X] T006 Add `{Resource: ResourceChatCollaborate, Action: ActionWrite, Description: "Join assigned chats as a collaborator"}` to `DefaultPermissions()` in `internal/models/roles.go` after line 169
- [X] T007 Add `"chat.assign:write"` to `agentPermissions` slice in `SystemRolePermissions()` at `internal/models/roles.go:296`
- [X] T008 Add `"chat.collaborate:write"` to `managerPermissions` slice in `SystemRolePermissions()` at `internal/models/roles.go:268`
- [X] T009 Add WebSocket message type constants `TypeChatClaimed = "chat_claimed"`, `TypeCollaboratorJoined = "collaborator_joined"`, `TypeCollaboratorLeft = "collaborator_left"` to `internal/websocket/messages.go` after line 30
- [ ] T010 Add `chat_inactivity_timeout_hours` to `Organization.Settings` defaults in `internal/database/postgres.go` seed/init logic (default: 24)

**Checkpoint**: Foundation ready — models, permissions, and WS constants exist. User story implementation can begin.

---

## Phase 3: User Story 1 — Incoming Message Becomes Pending Chat (Priority: P1) 🎯 MVP

**Goal**: New incoming messages from unassigned contacts automatically get `chat_status = "pending"`, and agents see the contact in their sidebar but cannot read message content until they claim it.

**Independent Test**: Send a WhatsApp message from a test phone → verify contact appears in sidebar with red badge, but opening it shows a "Claim to view" screen instead of messages. Verify `metadata->>'chat_status' = 'pending'` in database.

### Implementation for User Story 1

- [X] T011 [US1] Modify `processIncomingMessageFull` in `internal/handlers/chatbot_processor.go` — after `GetOrCreateContact` returns, if `contact.AssignedUserID == nil` and `contact.Metadata["chat_status"]` is nil, set it to `"pending"` and save via `a.DB.Model(contact).Update("metadata", contact.Metadata)`
- [X] T012 [US1] Add privacy guard in `GetMessages` handler at `internal/handlers/contacts.go:276` — compute `canViewContent` (true if: `contacts:read` permission OR `assigned_user_id == userID` OR `IsCollaborator(userID)` OR `chat.collaborate:write` permission). If `!canViewContent && contact.EffectiveStatus() == ChatStatusPending`, return 403 with `code: "chat_not_claimed"` and `pending_message_count` in data
- [X] T013 [US1] Add `ChatStatus string \`json:"chat_status,omitempty"\`` and `Collaborators []models.Collaborator \`json:"collaborators,omitempty"\`` fields to `ContactResponse` struct in `internal/handlers/contacts.go:48`
- [X] T014 [US1] Populate `ChatStatus: string(contact.EffectiveStatus())` and `Collaborators: contact.GetCollaborators()` in `buildContactResponse` return at `internal/handlers/contacts.go:1654`
- [X] T015 [US1] Add `chat_status?: 'pending' | 'open' | 'closed'` and `collaborators?: Collaborator[]` to `Contact` interface in `frontend/src/stores/contacts.ts:36`
- [X] T016 [US1] Add `export interface Collaborator { user_id: string; name: string; role: string; joined_at: string }` to `frontend/src/stores/contacts.ts`
- [X] T017 [US1] Add `isPendingClaim` computed property in `frontend/src/stores/contacts.ts` — returns true if `currentContact.chat_status === 'pending' && !currentContact.assigned_user_id`
- [X] T018 [US1] Add `pendingMessageCount` ref and modify `fetchMessages` in `frontend/src/stores/contacts.ts:188` — catch 403 with `code === 'chat_not_claimed'`, extract `pending_message_count` from response, leave `messages.value = []` so UI shows claim screen
- [X] T019 [US1] Add claim screen UI in `frontend/src/views/chat/ChatView.vue` — `v-if="contactsStore.isPendingClaim"` block with Lock icon (lucide-vue-next), pending message count text, and disabled-state placeholder (claim button added in US2)
- [X] T020 [P] [US1] Add i18n keys to `frontend/src/i18n/locales/en.json`: `chat.chatNotClaimed`, `chat.claimToViewMessages`, `chat.messagesWaiting`
- [X] T021 [P] [US1] Add i18n keys to `frontend/src/i18n/locales/ar.json` (Arabic translations for the same keys)

**Checkpoint**: Incoming messages create pending contacts. Agents see them in sidebar but content is hidden behind a claim screen. No claim button yet (that's US2).

---

## Phase 4: User Story 2 — Agent Claims a Pending Conversation (Priority: P1)

**Goal**: Agent presses "Claim" on a pending conversation → conversation is assigned to them, status changes to open, messages become visible, system message posted, WebSocket broadcast.

**Independent Test**: Click "Claim" on a pending conversation → verify `assigned_user_id` changes, `chat_status` becomes "open", system message appears, messages load. Second agent sees 409 if they try to claim the same one.

### Implementation for User Story 2

- [X] T022 [US2] Create `internal/handlers/chat_lifecycle.go` with `createSystemMessage(orgID, contactID uuid.UUID, content string, metadata models.JSONB)` helper — creates a Message record with `Direction: outgoing`, `MessageType: text`, `Status: sent`, and the given metadata
- [X] T023 [US2] Implement `ClaimChat` handler in `internal/handlers/chat_lifecycle.go` — `requireAuth(r, models.ResourceChatAssign, models.ActionWrite)`, load contact via org-scoped query, guard 1 (closed → 409 `chat_closed`), guard 2 (assigned to other without `chat.collaborate:write` → 409 `already_assigned` with owner name), guard 3 (assigned to self → idempotent 200), set `AssignedUserID = &userID` + `SetStatus(ChatStatusOpen)` + `a.DB.Save(contact)`, create system message, broadcast `TypeChatClaimed` via `a.WSHub.BroadcastToOrg`, return envelope
- [X] T024 [US2] Register route `g.PUT("/api/contacts/{id}/claim", app.ClaimChat)` in `cmd/whatomate/main.go` after line 687
- [X] T025 [US2] Add `claimChat(contactId: string)` action in `frontend/src/stores/contacts.ts` — `api.put('/contacts/${contactId}/claim')`, update contact locally (`assigned_user_id`, `chat_status = 'open'`), re-fetch messages, toast success
- [X] T026 [US2] Update claim screen in `frontend/src/views/chat/ChatView.vue` — add `<Button @click="handleClaim" :disabled="isClaiming">` with Hand icon (lucide-vue-next), add `isClaiming` ref + `handleClaim()` function calling `contactsStore.claimChat()`
- [X] T027 [US2] Add WebSocket handler case `'chat_claimed'` in `frontend/src/services/websocket.ts` handleMessage switch — find contact in list, update `assigned_user_id` and `chat_status`, if current contact re-fetch messages
- [X] T028 [P] [US2] Modify `AssignContact` handler at `internal/handlers/contacts.go:1204` — after `Update("assigned_user_id", ...)`, also set `contact.SetStatus(models.ChatStatusOpen)` and save metadata so manual assignment is consistent with claim
- [X] T029 [P] [US2] Add i18n keys to `frontend/src/i18n/locales/en.json` and `ar.json`: `chat.claimChat`, `chat.claimedSuccessfully`

**Checkpoint**: Agent can claim pending conversations. Messages unlock. Other agents blocked. Manual assignment also sets open status.

---

## Phase 5: User Story 3 — Collaborator Joins an Assigned Conversation (Priority: P2)

**Goal**: User with `chat.collaborate:write` can join an assigned conversation as a collaborator. Multiple collaborators allowed. Manager/admin can remove collaborators. Agents self-leave only. Owner leaving (last participant) closes conversation.

**Independent Test**: As Agent A, claim a conversation. As User B (manager role), open same conversation → see "Join" button → join → verify collaborator bar appears, messages visible, can reply. Manager removes collaborator. Owner self-leaves → conversation closes.

### Implementation for User Story 3

- [X] T030 [US3] Implement `JoinChat` handler in `internal/handlers/chat_lifecycle.go` — `requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)`, load contact, reject if already owner or already collaborator (idempotent 200), call shared `joinAsCollaborator` logic (load user FullName + Role name, `contact.AddCollaborator(...)`, save metadata, create system message "🔔 {Name} joined", broadcast `TypeCollaboratorJoined`)
- [X] T031 [US3] Implement `LeaveChat` handler in `internal/handlers/chat_lifecycle.go` — extract user via `getOrgAndUserID`, load contact, reject if primary owner (they must unassign → close instead), reject if not a collaborator (400), `contact.RemoveCollaborator(userID)`, check if any participants remain (owner + collaborators); if none remain → `contact.SetStatus(ChatStatusClosed)` instead of leaving open, save, create system message "🔔 {Name} left", broadcast `TypeCollaboratorLeft`
- [X] T032 [US3] Implement `RemoveCollaborator` handler in `internal/handlers/chat_lifecycle.go` — `requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)` (managers/admins only), path param `{user_id}`, load contact, reject if target is the primary owner, `contact.RemoveCollaborator(targetUserID)`, save, system message "🔔 {Name} was removed by {ManagerName}", broadcast `TypeCollaboratorLeft`
- [X] T033 [US3] Register routes in `cmd/whatomate/main.go` after line 687: `g.POST("/api/contacts/{id}/join", app.JoinChat)`, `g.DELETE("/api/contacts/{id}/join", app.LeaveChat)`, `g.DELETE("/api/contacts/{id}/collaborators/{user_id}", app.RemoveCollaborator)`
- [X] T034 [US3] Add computed properties in `frontend/src/stores/contacts.ts`: `isAssignedToMe` (assigned_user_id === authStore.user?.id), `isAssignedToOther` (assigned to someone else), `isCollaborator` (in collaborators array), `canViewMessages` (any access condition true), `canCollaborate` (authStore.hasPermission('chat.collaborate', 'write'))
- [X] T035 [US3] Add `joinChat(contactId)` and `leaveChat(contactId)` actions in `frontend/src/stores/contacts.ts` — call respective API endpoints, re-fetch contact + messages, toast feedback
- [X] T036 [US3] Add join screen UI in `frontend/src/views/chat/ChatView.vue` — `v-if="contactsStore.isAssignedToOther && !contactsStore.isCollaborator && contactsStore.canCollaborate"` block with Users icon, "Join as collaborator" button calling `handleJoin()`
- [X] T037 [US3] Add collaborators bar in `frontend/src/views/chat/ChatView.vue` conversation header — `v-if="contactsStore.currentContact?.collaborators?.length"` showing avatar circles with initials + tooltip with name/role, plus "Leave" button for collaborators and "Remove" dropdown for managers
- [X] T038 [US3] Add WebSocket handler cases in `frontend/src/services/websocket.ts`: `'collaborator_joined'` (push to collaborators array, re-fetch messages if current contact) and `'collaborator_left'` (filter from collaborators array, toast if transferred)
- [X] T039 [P] [US3] Add i18n keys to `frontend/src/i18n/locales/en.json` and `ar.json`: `chat.joinAsCollaborator`, `chat.leaveConversation`, `chat.collaborators`, `chat.removeCollaborator`, `chat.conversationClosed`

**Checkpoint**: Full collaboration works. Managers can join/remove. Agents self-leave. Owner leaving closes conversation.

---

## Phase 6: User Story 4 — Admin Configures Collaboration Permissions (Priority: P2)

**Goal**: New permissions appear automatically in `/settings/roles` PermissionMatrix. Admin can create custom roles with collaboration permissions. Changes take effect immediately.

**Independent Test**: Go to `/settings/roles` → verify `chat.assign:write` and `chat.collaborate:write` appear under Chat group. Create "Accounting Staff" role with collaborate but not assign. Assign a user → verify they can join but not claim.

### Implementation for User Story 4

- [ ] T040 [US4] Verify permissions appear automatically — no code change needed (PermissionMatrix reads from `/api/permissions` which returns `DefaultPermissions()` entries). Confirm `chat.assign:write` and `chat.collaborate:write` are visible under Chat group by running `make build && ./whatomate server -migrate` and checking the API
- [ ] T041 [US4] Verify the `agent` system role now includes `chat.assign:write` (from T007) by checking `SystemRolePermissions()` output — agent can claim but not collaborate
- [ ] T042 [US4] Verify the `manager` system role now includes `chat.collaborate:write` (from T008) — manager can join any conversation
- [ ] T043 [P] [US4] Add `chat_inactivity_timeout_hours` setting field to Settings UI in `frontend/src/views/settings/SettingsView.vue` — number input bound to `generalSettings.chat_inactivity_timeout_hours`, labeled "Chat inactivity timeout (hours)", default 24, with help text "Conversations auto-release to pending after this many hours of inactivity"

**Checkpoint**: Permissions are live in the roles UI. Settings page has the timeout field. All 4 user stories are functionally complete.

---

## Phase 7: Auto-Revert Worker (Cross-Cutting)

**Purpose**: Background worker that reverts open conversations to pending after inactivity timeout (FR-016, FR-017)

- [ ] T044 Create `internal/handlers/chat_inactivity_worker.go` with a `ChatInactivityWorker` struct holding `*App` + check interval — `Run()` method with `time.NewTicker(5 * time.Minute)`, on each tick: query contacts where `chat_status = 'open'` AND `last_message_at < NOW() - interval 'X hours'` (X from org settings), for each: clear `assigned_user_id`, clear `collaborators`, set `chat_status = 'pending'`, create system message "🔔 Conversation released due to inactivity", broadcast `TypeChatClaimed` with revert payload
- [ ] T045 Start the worker in `cmd/whatomate/main.go` — add `go handlers.NewChatInactivityWorker(app, 5*time.Minute).Run()` in the server startup section (near other background workers)

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Remaining i18n locales, testing, and validation

- [ ] T046 [P] Add i18n keys to `frontend/src/i18n/locales/es.json` (Spanish translations for all new keys)
- [ ] T047 [P] Add i18n keys to `frontend/src/i18n/locales/hi.json` (Hindi translations)
- [ ] T048 [P] Add i18n keys to `frontend/src/i18n/locales/ta.json` (Tamil translations)
- [ ] T049 Run `make build` to verify Go compilation with all changes
- [ ] T050 Run `cd frontend && npm run build` to verify frontend compilation
- [ ] T051 Run manual testing per `specs/001-chat-claim-collaboration/quickstart.md` — complete all 7 steps and verify each checkpoint
- [ ] T052 Verify backward compatibility — open a pre-existing contact (no `chat_status` in metadata) and confirm messages load without claim screen (EffectiveStatus defaults to "open")
- [ ] T053 Verify concurrent claim safety — open two browser sessions as different agents, claim the same pending conversation simultaneously, confirm only one succeeds

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — verification only
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational — MVP, must complete first
- **US2 (Phase 4)**: Depends on US1 (needs pending state + claim screen to exist)
- **US3 (Phase 5)**: Depends on US2 (needs claim/open conversations to exist before collaboration)
- **US4 (Phase 6)**: Depends on Foundational (permissions from T005-T008) — can run in parallel with US2/US3
- **Auto-Revert (Phase 7)**: Depends on US1+US2 (needs pending/open lifecycle to exist)
- **Polish (Phase 8)**: Depends on all prior phases

### User Story Dependencies

```
Foundational (T004-T010)
       │
       ├──► US1 (T011-T021) ──► US2 (T022-T029) ──► US3 (T030-T039)
       │                                    │
       └──► US4 (T040-T043) ────────────────┘
                       │
                       ▼
              Auto-Revert (T044-T045)
                       │
                       ▼
                 Polish (T046-T053)
```

### Within Each User Story

1. Backend model/handler changes first
2. Route registration
3. Frontend store logic
4. Frontend UI components
5. WebSocket handlers
6. i18n keys (parallelizable)

### Parallel Opportunities

- T020 + T021 (en + ar i18n for US1) — different files
- T028 + T029 (AssignContact modification + i18n for US2) — different files
- T039 (i18n for US3) — independent of backend tasks
- T043 (Settings UI) — independent of US2/US3 backend work
- T046 + T047 + T048 (es + hi + ta i18n) — three different files, fully parallel

---

## Parallel Example: User Story 1

```bash
# After T011-T014 (backend) are done, these frontend tasks can run in parallel:
Task: T015 "Add chat_status/collaborators to Contact interface in frontend/src/stores/contacts.ts"
Task: T016 "Add Collaborator interface in frontend/src/stores/contacts.ts"  # same file — sequential after T015
Task: T020 "Add en.json i18n keys"  # different file — parallel
Task: T021 "Add ar.json i18n keys"  # different file — parallel
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verify integration points)
2. Complete Phase 2: Foundational (models + permissions + WS constants)
3. Complete Phase 3: User Story 1 (pending state + privacy guard)
4. **STOP and VALIDATE**: Send a test message → verify pending status + hidden content
5. Deploy/demo if ready — at this point, messages are protected

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → Messages auto-pending, content hidden → **Test independently** (MVP!)
3. Add US2 → Agents can claim → **Test independently**
4. Add US3 → Collaboration works → **Test independently**
5. Add US4 → Permissions configurable → **Test independently**
6. Add Auto-Revert → Open conversations auto-release → **Test independently**
7. Polish → All locales + full validation

---

## Notes

- Tasks T004-T009 (Foundational) are the hard prerequisite — nothing works without the ChatStatus type and permissions
- No database migrations needed — all data in existing JSONB Metadata
- The permission system auto-propagates: adding to `DefaultPermissions()` + `SystemRolePermissions()` is sufficient — PermissionMatrix UI reads from the API
- The WebSocket handler in `websocket.ts` picks fields explicitly (Constitution Principle 9) — new system message events get their own switch cases, not extensions of `handleNewMessage`
- The auto-revert worker (Phase 7) can be deferred to a follow-up if MVP is needed urgently
- All `a.WSHub.BroadcastToOrg` calls must null-check the hub (Constitution Principle 9)
