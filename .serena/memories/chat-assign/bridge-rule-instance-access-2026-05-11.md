## Assignment Bridge Rule Implementation (2026-05-11)

### Problem
Admins/Managers without explicit instance access couldn't assign chats from restricted instances. Assignees of chats on restricted instances couldn't see instance metadata.

### Changes Made

1. **`chat_access_policy.go` — `applyRestrictedInstanceVisibilityFilter`**
   - Added `userID uuid.UUID` parameter
   - Changed filter from `instance_id IN ?` to `(instance_id IN ? OR assigned_user_id = ?)`
   - "Bridge rule": assigned chats are always visible to the assignee, regardless of instance restrictions

2. **`contacts_management.go` — `AssignContact`**
   - Added `!a.canBypassPendingChatRestriction(userID, orgID)` guard to the `canUserSeeContactInstance` check
   - Admins/super-admins can now assign chats from any instance to any user

3. **`contacts_management.go` — `buildLifecycleContactQuery`**
   - Updated to pass `userID` to the filter (affects ClaimChat, CloseChat)

4. **`contacts.go`** — Updated 3 call sites (ListContacts, GetContact, GetMessages)

5. **`contact_collaborators.go`** — Updated `loadContactForCollaboration`

6. **`websocket.go`** — Updated `canSubscribeToContactUpdates`

7. **`chat_access_policy_test.go`** — Updated 2 existing test calls with `uuid.Nil` parameter

### Security Constraint
Bridge rule only exposes chats where `assigned_user_id = currentUserID`. Other chats on the same restricted instance remain hidden. When assignment is removed, instance info becomes hidden again (unless user has global access).

### Bug Fixes (2026-05-11) — LIVE TESTED

#### Runtime Fix: "table name contacts specified more than once"
- **Root cause**: `requestDB` is tenant-scoped via `ScopedDB` which adds `WHERE contacts.organization_id = ?`. When GORM's `Model(contact).Updates(map)` is called, it combines the scope clause with its own table reference, producing a duplicate table reference in the SQL.
- **Fix**: Changed `requestDB.Model(contact).Updates(...)` and `requestDB.Where(...).First(contact)` to `a.DB.Model(contact).Updates(...)` and `a.DB.Where(...)` for the update/reload operations in `AssignContact`. Safe because the contact was already verified via `findByIDAndOrg` (which uses `requestDB` with tenant scope).

### Bug Fixes (2026-05-11)

#### Bug 1: "User not found" on assign
- **Root cause**: `AssignContact` validated assignee with `WHERE id = ? AND organization_id = ?` (home-org only), but `ListUsers` uses `JOIN user_organizations` (multi-org). Users visible in the assign dialog could fail the check.
- **Fix**: Created `userBelongsToOrg` helper that checks both `users.organization_id` AND `user_organizations` membership via EXISTS subquery.
- **Error message**: Changed from "User not found" to "User is not a member of this organization" for clarity.

#### Bug 2: "Failed to assign contact" on unassign
- **Root cause**: No closed-chat guard. Unassigning a closed chat would attempt to set status to "pending" which could conflict with business logic.
- **Fix**: Added `normalizeContactStatus` check before the update — closed chats now return 409 Conflict: "Cannot assign or unassign a closed chat. Reopen it first."
- **Also**: The old `req.UserID != nil` check didn't guard against `uuid.Nil`, which could trigger the user lookup for null UUIDs. Fixed to `req.UserID != nil && *req.UserID != uuid.Nil`.

### New Tests Added
- `TestApp_AssignContact_AllowsMultiOrgUser` — user with different home org but member via user_organizations
- `TestApp_AssignContact_RejectsUserNotInOrg` — user with no org membership at all
- `TestApp_AssignContact_UnassignOpenChat` — unassign open chat → pending
- `TestApp_AssignContact_RejectsAssignClosedChat` — 409 on assign closed chat
- `TestApp_AssignContact_RejectsUnassignClosedChat` — 409 on unassign closed chat

### Not Changed (follow-up candidates)