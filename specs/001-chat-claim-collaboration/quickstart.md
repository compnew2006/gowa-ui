# Quickstart: Chat Status, Claim & Collaboration

**Feature**: 001-chat-claim-collaboration  
**Date**: 2026-07-12

---

## Prerequisites

- Go 1.25+ with PostgreSQL 17 and Redis 7 running
- Frontend dependencies installed (`cd frontend && npm install`)
- A connected WhatsApp account (GOWA or Meta)
- At least 2 users with different roles (agent + manager/admin)

---

## Step 1: Build & Run

```bash
# Backend — build and run with migration
make build
./whatomate server -config config.toml -migrate

# Frontend (separate terminal, for dev mode)
cd frontend && npm run dev
```

---

## Step 2: Configure Roles

1. Log in as admin at `http://localhost:18080`
2. Navigate to **Settings → Roles** (`/settings/roles`)
3. Edit the **Agent** role:
   - Under **Chat** group, verify `chat.assign:write` is enabled (✅)
   - Verify `chat.collaborate:write` is disabled (❌)
4. (Optional) Create a custom role:
   - Click **New Role**, name it "Accounting Staff"
   - Enable: `chat:read`, `chat:write`, `chat.collaborate:write`
   - Disable: `chat.assign:write`
   - Save and assign a user (e.g., Sarah) to this role

---

## Step 3: Test the Pending State

1. From a test phone, send a WhatsApp message to your business number
2. As an agent, open the chat sidebar at `/chat`
3. Verify the new conversation appears with:
   - Customer name + phone number visible
   - Red badge showing "1" (unread count)
   - `chat_status: "pending"` in the contact data

```sql
-- Verify in database:
SELECT phone_number, metadata->>'chat_status', assigned_user_id
FROM contacts
WHERE metadata->>'chat_status' = 'pending'
ORDER BY created_at DESC LIMIT 5;
```

---

## Step 4: Test the Claim Flow

1. Click the pending conversation in the sidebar
2. Verify you see the **"🔒 Claim this chat"** screen (NOT the messages)
3. Verify it shows "1 message waiting"
4. Click **"Claim Conversation"**
5. Verify:
   - Messages become visible
   - System message appears: "🔔 {Your Name} claimed this conversation"
   - Header shows "✅ Assigned to: {Your Name}"
6. In a second browser (different agent), verify the conversation no longer shows as claimable

---

## Step 5: Test Collaboration

1. As Agent A (who claimed the conversation), verify you can send messages
2. Log in as User B (with `chat.collaborate:write`, e.g., manager or accounting staff)
3. Open the same conversation from the sidebar
4. Verify the **"Join as collaborator"** screen appears
5. Click **"Join"**
6. Verify:
   - System message: "🔔 {User B Name} joined the conversation"
   - Collaborators bar shows User B's avatar
   - User B can read all messages
   - User B can send replies
7. As User B, click **"Leave"**
8. Verify:
   - System message: "🔔 {User B Name} left the conversation"
   - User B is removed from collaborators

---

## Step 6: Test Permission Guards

1. As a regular agent (no `chat.collaborate:write`):
   - Try opening a conversation assigned to another agent
   - Verify you get 403/409 (no join button visible)
2. Try claiming an already-assigned conversation:
   - Verify 409 error with the current owner's name
3. Try claiming a closed conversation (if any exist):
   - Verify 409 error "Cannot claim a closed chat"

---

## Step 7: Test Manager Kick

1. As Agent A, claim a conversation and have User B join as collaborator
2. Log in as a Manager (with `chat.collaborate:write`)
3. Open the conversation → verify you can see all participants
4. Find the "Remove" button next to User B's avatar in the collaborators bar
5. Click "Remove" next to User B
6. Verify:
   - System message: "🔔 {Manager} removed {User B} from the conversation"
   - User B disappears from collaborators bar
   - User B loses access to the conversation
7. As a regular agent, verify there is NO "Remove" button (agents can't kick)

---

## Step 8: Test Owner Leave → Auto-Close

1. As Agent A, claim a conversation (no collaborators joined)
2. Click "Leave" (the owner leave action)
3. Verify the conversation closes:
   - System message: "🔔 Conversation closed"
   - `chat_status` changes to `"closed"` in DB
   - Conversation shows as read-only in the sidebar

```sql
SELECT metadata->>'chat_status' FROM contacts WHERE id = ?;
-- Should show "closed"
```

4. Now test: claim another conversation, have User B join as collaborator
5. As Agent A (owner), click "Leave"
6. Verify conversation stays `open` (collaborators remain):
   - System message: "🔔 {Agent A} left. 1 collaborator remains."
   - `assigned_user_id` is cleared but collaborators stay

---

## Step 9: Test Auto-Revert (Inactivity Timeout)

1. As admin, go to **Settings → General** and set `chat_inactivity_timeout_hours` to a small test value (e.g., `1`)
2. As an agent, claim a conversation and send/receive a message
3. Wait for the timeout to elapse (1 hour in this test, or temporarily lower to a few minutes for testing)
4. Verify the background worker reverts the conversation:
   - System message: "🔔 Conversation released due to inactivity"
   - `chat_status` changes to `"pending"`
   - `assigned_user_id` is cleared
   - `collaborators` are cleared
   - The conversation reappears in the pending queue

```sql
SELECT metadata->>'chat_status', assigned_user_id, metadata->'collaborators'
FROM contacts WHERE id = ?;
-- chat_status = "pending", assigned_user_id = NULL, collaborators = []
```

5. Reset the timeout to 24 (production default) in Settings

---

## Step 10: Verify Backward Compatibility

```sql
-- All pre-existing contacts should have NO chat_status (default = open):
SELECT count(*) FROM contacts WHERE metadata->>'chat_status' IS NULL;
-- This count should match your total contacts before the feature was deployed

-- Verify they still work normally (no pending guard):
-- Open an old conversation → messages should load immediately (no claim screen)
```

---

## Troubleshooting

| Issue | Check |
|-------|-------|
| Claim button does nothing | Check browser console for 403 — verify user has `chat.assign:write` |
| Messages still hidden after claim | Verify `chat_status` changed to `open` in DB: `SELECT metadata->>'chat_status' FROM contacts WHERE id = ?` |
| Join button not visible | Verify user has `chat.collaborate:write` permission in `/settings/roles` |
| WebSocket events not received | Check `a.WSHub != nil` in server logs; check browser WS connection in DevTools |
| System messages don't appear | Verify message `metadata.is_system_message = true` in DB |
| Old conversations show claim screen | This should NOT happen — verify `EffectiveStatus()` returns `open` for null metadata |
