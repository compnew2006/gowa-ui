# Per-User WhatsApp Account Visibility

## User Story

> As an **organization admin**, I want to assign specific WhatsApp accounts to
> each user so that agents only see (and act on) the accounts relevant to their
> work, instead of every account in the organization.

> As an **agent**, when I open the app I only see the WhatsApp accounts my
> admin assigned to me — in the accounts settings page, the chat account tabs,
> contact dialogs, and anywhere else accounts are listed.

## Behavior Summary

| User state | What they see |
|---|---|
| Has 1+ account assignments in the org | Only the assigned accounts |
| Has **no** assignments in the org | **All** org accounts (fallback) |
| Is a super admin | All org accounts, always (assignments ignored) |

Assignments are **org-scoped**: a user who belongs to multiple organizations
has an independent assignment set per organization.

## Data Model

New join table `user_whatsapp_accounts` (`models.UserWhatsAppAccount`):

| Column | Notes |
|---|---|
| `id` | UUID PK |
| `user_id` | FK → `users.id`, part of composite unique index |
| `whats_app_account_id` | FK → `whatsapp_accounts.id`, part of composite unique index |
| `created_at` | timestamp |

Design points:

- **No soft delete.** Rows are hard-deleted on unassignment so the composite
  unique index `(user_id, whats_app_account_id)` never collides with stale
  soft-deleted pairs.
- Registered in both migration lists: `internal/database/postgres.go`
  (`GetMigrationModels`) and `test/testutil/db.go`.

## API Changes

### `GET /accounts` (read filtering)

`ListAccounts` scopes the query through `scopeAccountsToUser`:

1. Super admin → unrestricted org query.
2. Load the user's assignment IDs **joined against live (non-deleted) accounts
   of the current org** — assignments in other orgs or pointing at deleted
   accounts are invisible.
3. Zero assignments → unrestricted org query (fallback).
4. Otherwise → `WHERE whatsapp_accounts.id IN (assigned IDs)`.

Because `accountsService` in `frontend/src/services/api.ts` is the single
frontend entry point for `/accounts`, the filtering automatically propagates to
every consumer (AccountsView, ChatView account tabs, ContactsView, contact
dialogs) with **no frontend changes required for read paths**.

### `GET/PUT/DELETE /accounts/:id` (write filtering)

`GetAccount`, `UpdateAccount` and `DeleteAccount` call `canAccessAccount`
before touching the record. A restricted user operating on an unassigned
account receives **404 Not Found** (not 403) so the account's existence is not
leaked. `DeleteAccount` also removes any dangling assignment rows for the
deleted account.

`CreateAccount` is intentionally unrestricted beyond the existing
`accounts:write` permission — a newly created account is visible to its
creator via the fallback rule or until assignments say otherwise.

### `POST /users`, `PUT /users/:id` (assignment management)

`UserRequest` gains `whatsapp_account_ids`:

- **omitted / `null`** → assignments left untouched
- **`[]` (empty array)** → all assignments in this org cleared (user returns
  to full org visibility)
- **`[id, ...]`** → replace-all semantics *within the current org*; every ID
  must belong to an org account or the request fails with
  **400 "One or more account IDs are invalid"** and nothing is changed
  (validation happens before the transactional replace).

Responses from `GET /users/:id`, `POST /users` and `PUT /users/:id` include
`whatsapp_account_ids` (the user's assignments in the current org).
`GET /users` (list) intentionally omits it to avoid N+1 queries.

`DELETE /users/:id` cleans up assignments: org-scoped removal for cross-org
members, full removal for native users.

## Permission Interaction

| Action | Requirement |
|---|---|
| Seeing the (filtered) account list | `accounts:read` — unchanged |
| Editing/deleting an account | `accounts:write` / `accounts:delete` **and** the account must be visible to the user |
| Changing anyone's assignments | `users:write` |
| Changing your **own** assignments | also `users:write` — otherwise 403 "Insufficient permissions to change account assignments" |

The self-update gate is the key safety property: without it, an agent could
lift their own visibility restriction via `PUT /users/{their-own-id}`.

Cross-org members can have their **role and account assignments** updated by
the host org (previously role only).

## Frontend Workflow

1. **Create user** (`UsersView.vue` dialog): an optional "WhatsApp Account
   Access" checklist appears when the org has accounts. Unchecked = no
   restriction. Checked IDs are sent as `whatsapp_account_ids`.
2. **Edit user** (`UserDetailView.vue`): the same checklist is populated from
   the `whatsapp_account_ids` in the user response. Saving always sends the
   current selection (empty selection clears assignments); the field is only
   sent when the editor has `users:write`.
3. **Agent experience**: nothing new to learn — all account pickers and lists
   simply show fewer accounts.

Note: the checklist itself is populated from `/accounts`, so an admin who is
themselves restricted can only assign accounts they can see.

## Edge Cases

- **No assignments (fallback)** — user sees every org account. This keeps
  existing installs working with zero migration effort and matches the "if no
  specific assignments exist, show all" requirement.
- **All assignments cleared** — identical to "no assignments": full visibility
  is restored, which admins should be aware of.
- **Assigned account gets deleted** — the delete handler removes the
  assignment rows; the visibility join additionally ignores soft-deleted
  accounts, so a user whose *only* assignment was deleted falls back to full
  org visibility rather than seeing nothing.
- **Duplicate IDs in the request** — deduplicated server-side before the
  unique-index insert.
- **Account from another org in the payload** — rejected with 400; the
  transaction never runs.
- **Multi-org member** — replacing assignments in org A leaves the member's
  assignments in org B untouched (the delete inside the sync transaction is
  scoped by an org-account subquery).
- **Super admin with assignments** — assignments are stored but ignored; super
  admins are never restricted.
- **Assignment lookup failure** (DB error) — the scoping helper logs and fails
  open to org-wide visibility rather than blanking the account list.
- **Background/system jobs** (history sync, campaign workers, webhook
  processing) query accounts as the system, not as a user, and are
  intentionally not filtered.

## Test Coverage

- `internal/handlers/accounts_test.go`: assigned-only listing, no-assignment
  fallback, cross-org assignment ignored, super admin unrestricted,
  Get/Update/Delete returning 404 for unassigned accounts, assignment cleanup
  on account delete.
- `internal/handlers/users_test.go`: create with assignments, set/replace/
  clear via update, omitted field untouched, foreign-org ID rejected with 400,
  self-update without `users:write` rejected with 403, `GET /users/:id`
  including assignment IDs.
- Fixture: `testutil.AssignAccountToUser`.
