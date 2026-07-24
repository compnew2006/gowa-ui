# Pending / Me Tabs — Task Checklist

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/pending-me-tabs/`
**Companion documents:** `spec.md` (the what), `plan.md` (the how + reuse map + risks).

Each task is atomic, names exact files and symbols, states checkable acceptance criteria, and lists its dependencies. The Builder works top-to-bottom; the Auditor checks each box against the running build. Verification commands are at the bottom and must be re-run after every phase.

All file paths are absolute under the repo root `/Users/noiemany/Downloads/whatomate/`.

---

## Phase A — Backend primitives

### T1 — Add `ReleaseChat` handler
- [ ] **File:** `internal/handlers/chat_lifecycle.go`
- [ ] **New symbol:** `func (a *App) ReleaseChat(r *fastglue.Request) error`
- [ ] **Clone the skeleton of** `ClaimChat` at `internal/handlers/chat_lifecycle.go:39-130` (same auth, same org-scoped lookup, same agent-name resolution).
- [ ] **Auth:** `a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)` (mirror `chat_lifecycle.go:40`).
- [ ] **Contact lookup:** `a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact)`, 404 on miss (mirror `chat_lifecycle.go:51-53`).
- [ ] **Authorization guard:** reject with `403 "You are not allowed to release this chat"` when caller is neither `*contact.AssignedUserID == userID` nor admin/manager (`a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)`). Mirror the ghost-check pattern at `chat_lifecycle.go:344-353`.
- [ ] **Snapshot** `oldContact := contact` **before** mutation (needed for the audit call).
- [ ] **Mutation:** `contact.AssignedUserID = nil`; `contact.SetStatus(models.ChatStatusPending)`; `contact.ClearCollaborators()` (all three from `internal/models/chat_status.go`).
- [ ] **Persist:** `a.DB.Model(&contact).Updates(map[string]any{"assigned_user_id": nil, "metadata": contact.Metadata})` (mirror the close-path persist at `chat_lifecycle.go:379-382`).
- [ ] **System message:** `a.createSystemMessage(orgID, contact.ID, fmt.Sprintf("🔔 %s released this conversation", agentName), models.JSONB{"system_type": "chat_released", "agent_id": userID.String(), "agent_name": agentName})` — note the `agent_name` field (D3).
- [ ] **Audit:** `a.logAudit(orgID, userID, "contact", contact.ID, models.AuditActionUpdated, &oldContact, &contact, map[string]any{"chat_status": {"old": string(oldContact.EffectiveStatus()), "new": "pending"}, "assigned_user_id": {"old": oldContact.AssignedUserID, "new": nil}})` — the `extraChanges` map is mandatory (see `helpers.go:104` and the no-op risk in `plan.md`).
- [ ] **WS broadcast:** `a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{Type: websocket.TypeChatReleased, Payload: map[string]any{"contact_id": contact.ID.String(), "released_by": userID.String(), "chat_status": string(models.ChatStatusPending)}})`.
- [ ] **Return:** `r.SendEnvelope(map[string]any{"contact_id": contact.ID, "released": true, "chat_status": "pending"})`.
- [ ] **Idempotency:** if the contact is already pending and unassigned, return success without duplicating the system message (mirror `ClaimChat`'s idempotent branch at `chat_lifecycle.go:78-88`).
- [ ] **Acceptance:** handler compiles; calling `PUT /api/contacts/{id}/release` on an open assigned chat moves it to pending, writes a system message with `system_type=chat_released`, writes an audit entry at `/settings/audit-logs`, and broadcasts `chat_released`.
- [ ] **Depends on:** T4 (WS constant).

### T2 — Augment `ClaimChat` with audit + `agent_name`
- [ ] **File:** `internal/handlers/chat_lifecycle.go`
- [ ] **Symbol:** existing `func (a *App) ClaimChat` at line 39.
- [ ] **Add `agent_name`** to both `createSystemMessage` calls (the reopen branch at line 110-112 and the claim branch at 114-116): include `"agent_name": agentName` in the `models.JSONB` metadata (D3).
- [ ] **Snapshot** `oldContact := contact` **before** the mutation at line 94.
- [ ] **After** the successful `a.DB.Save(&contact)` at line 96 and the system-message write, add: `a.logAudit(orgID, userID, "contact", contact.ID, models.AuditActionUpdated, &oldContact, &contact, map[string]any{"chat_status": {"old": string(oldContact.EffectiveStatus()), "new": string(models.ChatStatusOpen)}, "assigned_user_id": {"old": oldContact.AssignedUserID, "new": &userID}})`.
- [ ] **Do not** change any other behavior of `ClaimChat` (the idempotent branch at 78-88, the conflict response at 71-74, the reopen detection at 91, the WS broadcast at 119-130 all stay).
- [ ] **Acceptance:** claiming a pending chat writes an audit entry at `/settings/audit-logs` (resource `contact`, action `updated`, actor = claimer) and the system message row carries `metadata.agent_name`.
- [ ] **Depends on:** nothing.

### T4 — Add `TypeChatReleased` WebSocket constant
- [ ] **File:** `internal/websocket/messages.go`
- [ ] **Add** immediately after line 71: `TypeChatReleased = "chat_released"`.
- [ ] **Acceptance:** constant compiles; referenced by T1.
- [ ] **Depends on:** nothing.

---

## Phase B — Backend wiring

### T3 — Register the release route
- [ ] **File:** `cmd/whatomate/main.go`
- [ ] **Add** immediately after line 759 (`g.DELETE("/api/contacts/{id}/join", app.LeaveChat)`): `g.PUT("/api/contacts/{id}/release", app.ReleaseChat)`.
- [ ] **Auth** is enforced inside the handler via `requireAuth`, matching the sibling routes at 755-759 — no extra middleware line needed.
- [ ] **Acceptance:** `go build ./...` succeeds; `PUT /api/contacts/{id}/release` returns 200 on a valid request and 401/403/404 per the spec.
- [ ] **Depends on:** T1.

---

## Phase C — Frontend store

### T6 — Add `releaseChat` action + tab computeds
- [ ] **File:** `frontend/src/stores/contacts.ts`
- [ ] **New state:** `const activeListTab = ref<'pending' | 'me'>('pending')`.
- [ ] **New computeds** (place near the existing chat-lifecycle computeds at lines 148-211):
  - `pendingContacts` — `computed(() => sortedContacts.value.filter(c => c.chat_status === 'pending' && !c.assigned_user_id))` (reuse the existing `sortedContacts` computed).
  - `myContacts` — `computed(() => sortedContacts.value.filter(c => c.assigned_user_id === authStore.user?.id))`.
  - `pendingCount` / `myCount` — `computed(() => pendingContacts.value.length)` / `myContacts.value.length`.
  - `displayedContacts` — `computed(() => activeListTab.value === 'pending' ? pendingContacts.value : myContacts.value)`.
- [ ] **New action** `releaseChat(contactId: string)` — clone the shape of `claimChat` at lines 513-531:
  - `const response = await api.put(`/contacts/${contactId}/release`)`.
  - Optimistic local update on the matching entry in `contacts.value`: `chat_status = 'pending'`, `assigned_user_id = undefined`, `assigned_user_name = undefined`.
  - If `currentContact.value?.id === contactId`: apply the same update to `currentContact.value`, then `await fetchMessages(contactId)`.
  - Return `response.data.data || response.data`.
- [ ] **Export** all new symbols from the store's `return` / setup-state surface so `ChatView.vue` can read them.
- [ ] **Acceptance:** `npm run typecheck` passes; calling `releaseChat(id)` in the console moves the contact out of `myContacts` and into `pendingContacts`.
- [ ] **Depends on:** nothing (store-only).

---

## Phase D — Frontend view

### T5 — Tab strip in the sidebar
- [ ] **File:** `frontend/src/views/chat/ChatView.vue`
- [ ] **Insert** a two-button tab strip between line 2024 (end of the visibility-toggles block) and line 2026 (`<!-- Contacts -->`).
- [ ] **Styling:** clone the account-tab class pattern (active `bg-emerald-600 text-white`, inactive `bg-white/[0.08] text-white/60 light:bg-gray-100 light:text-gray-600`) referenced at lines 2319-2339.
- [ ] **Buttons:** Pending (`@click="contactsStore.activeListTab = 'pending'"`, badge `{{ contactsStore.pendingCount }}`) and Me (`@click="contactsStore.activeListTab = 'me'"`, badge `{{ contactsStore.myCount }}`).
- [ ] **Labels:** `$t('chat.tabPending')` and `$t('chat.tabMe')` (keys added in T9).
- [ ] **Retarget** the `v-for` at line 2030 from `contactsStore.sortedContacts` to `contactsStore.displayedContacts`.
- [ ] **Retarget** the empty-state check at line 2081 (`contactsStore.sortedContacts.length === 0`) to `contactsStore.displayedContacts.length === 0`.
- [ ] **Do not** touch the row markup at lines 2031-2074.
- [ ] **Acceptance:** clicking a tab filters the rendered list with no network call; the active tab is visually highlighted; counts update live as contacts change status.
- [ ] **Depends on:** T6, T9 (for the labels).

### T7 — Release button in the chat header
- [ ] **File:** `frontend/src/views/chat/ChatView.vue`
- [ ] **Insert** a new `<Button>` immediately after the existing Leave button block at lines 2169-2173.
- [ ] **Guard:** `v-if="contactsStore.isAssignedToMe || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)"`.
- [ ] **Props:** `variant="ghost" size="sm" class="text-xs"`; icon `LogOut` (or a more fitting icon if one is already imported); label `{{ $t('chat.releasedConversation') }}`.
- [ ] **Handler:** `@click="handleRelease"`.
- [ ] **Add** `async function handleRelease() { if (!contactsStore.currentContact) return; await contactsStore.releaseChat(contactsStore.currentContact.id); }` to the `<script setup>` block (place near the existing `handleLeave` / `handleClose`).
- [ ] **Do not** modify the existing Leave button or its guard.
- [ ] **Acceptance:** button appears only for the assignee (or admin/manager on an open chat); clicking it calls `releaseChat`; the chat moves to pending and the system message renders.
- [ ] **Depends on:** T6, T9.

### T8 — System-message i18n
- [ ] **File:** `frontend/src/views/chat/ChatView.vue`
- [ ] **Add** a function `getSystemMessageText(message: any): string` in `<script setup>`:
  - If `message.metadata?.system_type` is set AND is one of `chat_claimed | chat_released | chat_closed | chat_reopened | collaborator_joined | collaborator_left` (the six single-actor override types): `const agent = message.metadata?.agent_name || extractAgentFromLegacy(message.content) || ''; return t('chat.system.' + message.metadata.system_type, { agent })`.
  - If `system_type` is `collaborator_removed` (dual-actor — `agent_id` = removed, `removed_by` = manager, see `chat_lifecycle.go:512-518`): return `getMessageContent(message)` (fallback preserves both names).
  - Else (no `system_type`): return `getMessageContent(message)`.
  - Else: return `getMessageContent(message)` (the existing fallback).
  - `extractAgentFromLegacy(content)` is a private helper that regex-extracts the name from legacy "🔔 <name> ..." strings and returns `''` on no match.
- [ ] **Retarget** line 2460 from `{{ getMessageContent(message) }}` to `{{ getSystemMessageText(message) }}`.
- [ ] **Do not** modify the wrapper markup at lines 2454-2462.
- [ ] **Acceptance:** every system message renders in the active locale, including historical rows without `agent_name`; switching locale re-renders the message.
- [ ] **Depends on:** T9.

---

## Phase E — i18n

### T9 — Add i18n keys (en.json first, then ar.json mirror)
- [ ] **Files:** `frontend/src/i18n/locales/en.json` (schema source), then `frontend/src/i18n/locales/ar.json` (key-for-key mirror).
- [ ] **Namespace:** top-level `chat` (starts at `en.json:296`).
- [ ] **New scalar keys** (siblings of the existing `chat.leaveConversation` at line 404 — do NOT re-add `leaveConversation`, it already exists):
  - `chat.tabPending` — en: `"Pending"`, ar: `"قيد الانتظار"`.
  - `chat.tabMe` — en: `"Me"`, ar: `"محادثاتي"`.
  - `chat.releasedConversation` — en: `"Release"`, ar: `"تحرير"`.
- [ ] **New nested object** `chat.system` with one key per real `system_type` value (verified by direct read of `internal/handlers/chat_lifecycle.go` and `chatbot_processor.go`). The seven keys are:
  - `chat.system.chat_claimed` — en: `"{agent} claimed this conversation"`, ar: `"{agent} استلم هذه المحادثة"`.
  - `chat.system.chat_released` — en: `"{agent} released this conversation"`, ar: `"{agent} حرر هذه المحادثة"`.
  - `chat.system.chat_closed` — en: `"{agent} closed this conversation"`, ar: `"{agent} أغلق هذه المحادثة"`.
  - `chat.system.chat_reopened` — en: `"{agent} reopened this conversation"`, ar: `"{agent} أعاد فتح هذه المحادثة"`.
  - `chat.system.collaborator_joined` — en: `"{agent} joined the conversation"`, ar: `"{agent} انضم إلى المحادثة"`.
  - `chat.system.collaborator_left` — en: `"{agent} left the conversation"`, ar: `"{agent} غادر المحادثة"`.
  - (No `collaborator_removed` key — that dual-actor type renders via the `getMessageContent` fallback to preserve both the removed user and the manager names. See T8.)
- [ ] **Parity check:** before marking done, diff the `chat` namespace across `en.json` and `ar.json` and confirm every key above exists in both with the same shape (run the parity snippet in `Verification` below).
- [ ] **Acceptance:** `npm run build` succeeds; `npm run typecheck` succeeds; parity check passes; switching the UI to Arabic shows the translated strings.
- [ ] **Depends on:** nothing (keys are inert until T5/T7/T8 reference them).

### T9b — Handle the `chat_released` WS event on the frontend
- [ ] **File:** `frontend/src/services/websocket.ts`
- [ ] **Add** `const WS_TYPE_CHAT_RELEASED = 'chat_released'` immediately after line 76.
- [ ] **Add** a `case WS_TYPE_CHAT_RELEASED:` branch in `handleMessage` (the switch starts at line 163) that updates the local contact: set `chat_status = 'pending'`, clear `assigned_user_id` / `assigned_user_name`. Mirror the existing `WS_TYPE_CHAT_CLAIMED` handler's idempotency (only mutate if different).
- [ ] **Acceptance:** a release on another client updates this client's sidebar within the WS round-trip.
- [ ] **Depends on:** T4 (backend constant), T6 (store fields).

---

## Phase F — Optional polish

### T10 — Add `contact` to the audit resource-type filter dropdown
- [ ] **File:** `frontend/src/views/settings/AuditLogsView.vue`
- [ ] **Add** `<SelectItem value="contact">{{ t('chat.chat') }}</SelectItem>` (or a literal `"Contact"`) into the resource-type `<SelectContent>` at lines 164-173, in alphabetical position.
- [ ] **Note:** the detail-side `resourceRouteMap` at `frontend/src/views/settings/AuditLogDetailView.vue:36` **already** contains `contact: (id) => /chat?contact=${id}` — do NOT re-add it.
- [ ] **Acceptance:** selecting `contact` in the filter narrows the audit list to contact-resource entries (including the new release/claim entries).
- [ ] **Depends on:** nothing; can be deferred.

---

## Verification (run after every phase, capture tails in `implementation_report.md`)

```bash
# Backend
cd /Users/noiemany/Downloads/whatomate && go build ./... 2>&1 | tail -30
cd /Users/noiemany/Downloads/whatomate && go vet ./... 2>&1 | tail -30

# Frontend — BOTH commands (build does not type-check, typecheck does not bundle)
cd /Users/noiemany/Downloads/whatomate/frontend && npm run build 2>&1 | tail -40
cd /Users/noiemany/Downloads/whatomate/frontend && npm run typecheck 2>&1 | tail -40

# i18n parity check (chat namespace must match key-for-key across en/ar)
python3 -c "
import json
en = json.load(open('/Users/noiemany/Downloads/whatomate/frontend/src/i18n/locales/en.json'))['chat']
ar = json.load(open('/Users/noiemany/Downloads/whatomate/frontend/src/i18n/locales/ar.json'))['chat']
def flat(d, p=''):
    out = {}
    for k, v in d.items():
        key = f'{p}.{k}' if p else k
        if isinstance(v, dict): out.update(flat(v, key))
        else: out[key] = v
    return out
fe, fa = flat(en), flat(ar)
miss_en = set(fe) - set(fa)
miss_ar = set(fa) - set(fe)
required = {'tabPending','tabMe','releasedConversation',
            'system.chat_claimed','system.chat_released','system.chat_closed',
            'system.chat_reopened','system.collaborator_joined',
            'system.collaborator_left'}
missing_req = required - set(fe)
print('en-only:', miss_en or 'none')
print('ar-only:', miss_ar or 'none')
print('missing required keys in en:', missing_req or 'none')
assert not miss_en and not miss_ar and not missing_req, 'i18n parity failed'
print('i18n parity OK')
"
```

## Done definition (Auditor checks each)

- [ ] All Phase A-F tasks above are checked off.
- [ ] All four verification commands exit 0.
- [ ] i18n parity script prints `i18n parity OK`.
- [ ] Manual: claim a pending chat → it appears under Me, system message shows in active locale, audit entry appears at `/settings/audit-logs`.
- [ ] Manual: release a claimed chat → it returns to Pending, system message shows, audit entry appears.
- [ ] Manual: existing Close / Reopen / Join / Leave flows behave as before (regression).
- [ ] Manual: a second browser session sees the `chat_released` event reconcile the sidebar.
