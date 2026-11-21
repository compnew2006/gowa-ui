# Agent Least-Privilege Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prune the live agent role back to its 10 defined permissions and lock system roles so no one can widen them again.

**Architecture:** Exact-match reconciliation in `SyncSystemRolePermissions` (runs on startup via `SeedSystemRolesForAllOrgs`) removes excess grants from system roles; `UpdateRole` rejects permission changes on system roles; the frontend disables permission editing for system roles. No permission definitions change.

**Tech Stack:** Go 1.25 (GORM + PostgreSQL), Vue 3 + TypeScript frontend.

**Spec:** Live CRM audit 2026-11-21 (Super Admin): agent role held 21 permissions including full WhatsApp accounts (read/write/delete/assign), `contacts:delete`, `analytics:write/delete`. Code at `db8c58b` defines agent narrowly in `internal/models/roles.go` `SystemRolePermissions()`: exactly 10 permissions — `accounts:read`, `chat:read`, `chat:write`, `chat.revoke:write`, `chat.assign:write`, `contacts:read`, `tags:read`, `analytics.agents:read`, `canned_responses:read`, `custom_actions:execute`.

## Global Constraints

- Work only in worktree `~/workspace/gowa-ui-agent-fix`, branch `fix/agent-least-privilege`, base `db8c58b`. Do not push, deploy, merge, or touch the live CRM, the main clone, or `~/workspace/superpowers`.
- Do not commit to main. Local commit only, no push.
- Keep `chat.assign:write` and `chat.revoke:write` for agent — transfer (`internal/handlers/chat_transfer.go`) depends on `chat.assign:write`; revoke is for own sent messages. Do not remove them.
- Do not change `SystemRolePermissions()` values or project dependencies.
- Custom (non-system) roles must be untouched by reconciliation.
- A backend change must update or add `*_test.go`; a frontend change must keep `npm run typecheck` green. Follow `gofmt` + `go vet`.
- Never weaken a test to make it pass.

## Review Focus

- A system role named something not in `SystemRolePermissions()` (unknown name) must be left alone by sync — expected: skipped, not emptied.
- A `SyncSystemRolePermissions` run must not delete the `Permission` rows themselves, only the role→permission links.
- An `UpdateRole` request that changes only the description of a system role must still succeed (200) for a Super Admin.
- A non-super-admin `UpdateRole` request with permissions on a system role keeps existing behaviour (permissions ignored, 200) — only the Super Admin bypass is closed, because that was the only path that actually changed permissions. `TestApp_UpdateRole_SystemRoleOnlyDescription` must keep passing unchanged.
- The frontend must still allow editing a system role's description while permissions are disabled.

---

### Task A: Backend reconcile — exact-match sync for system roles

**Files:**
- Modify: `internal/database/postgres.go:780-850` (`SyncSystemRolePermissions`)
- Test: `internal/database/database_test.go` (new `--- SyncSystemRolePermissions ---` section)

**Interfaces:**
- Consumes: `models.SystemRolePermissions() map[string][]string`, `models.Permission`, `role_permissions` join table.
- Produces: `SyncSystemRolePermissions(db *gorm.DB) error` with exact-match semantics for system roles only.

- [ ] **Step 1: Write the failing test** `TestSyncSystemRolePermissions_RemovesExcessAgentPermissions` in `internal/database/database_test.go`: seed permissions + system roles for an org, grant the agent role the 10 defined permissions plus `accounts:write`, `accounts:delete`, `accounts:assign`, `contacts:delete`, `analytics:write`, `analytics:delete`, then call `database.SyncSystemRolePermissions(db)` and assert the agent role ends with exactly the 10 defined permissions (use `models.SystemRolePermissions()["agent"]` as expected set). Also assert a custom role's extra permissions are untouched.
- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/database/ -run TestSyncSystemRolePermissions_RemovesExcessAgentPermissions -v`
Expected: FAIL — excess permissions still present.

- [ ] **Step 3: Implement exact-match reconciliation in `SyncSystemRolePermissions`** in `internal/database/postgres.go`: for each system role whose name is in `SystemRolePermissions()`, build the desired permission-ID set from the map and `Replace` the role's `Permissions` association with exactly that set (add missing, remove excess). Skip unknown role names. Update the function's doc comment (it currently says "It never removes permissions").
- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/database/ -run TestSyncSystemRolePermissions -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/database/postgres.go internal/database/database_test.go
git commit -m "fix(db): reconcile system role permissions to exact match"
```

### Task B: Backend lock — reject system-role permission changes in UpdateRole

**Files:**
- Modify: `internal/handlers/roles.go:226-295` (`UpdateRole`, system-role branch)
- Test: `internal/handlers/roles_test.go`

**Interfaces:**
- Consumes: `RoleRequest.Permissions []string`, `r.RequestCtx.UserValue("is_super_admin")`.
- Produces: `UpdateRole` returns HTTP 403 with an error envelope when a permission change is requested on a system role; description-only updates still return 200.

- [ ] **Step 1: Write the failing test** `TestApp_UpdateRole_SystemRolePermissionsRejectedForSuperAdmin` in `internal/handlers/roles_test.go`: create a system role named `agent` with the 10 defined permissions, a Super Admin user (`is_super_admin` user value = true), send `UpdateRole` with `Permissions: ["accounts:read", "accounts:delete"]`, assert status 403 and that the role's permissions in the DB are unchanged (still exactly the 10).
- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/ -run TestApp_UpdateRole_SystemRolePermissionsRejectedForSuperAdmin -v`
Expected: FAIL — currently returns 200 and replaces permissions.

- [ ] **Step 3: Implement in `UpdateRole`** in `internal/handlers/roles.go`: in the `role.IsSystem` branch, if `isSuperAdmin && len(req.Permissions) > 0`, return `r.SendErrorEnvelope(fasthttp.StatusForbidden, "System role permissions cannot be changed", nil, "")` before any save, replacing the current super-admin permission-replacement path. Non-super-admin requests keep existing behaviour (permissions silently ignored, description-only update proceeds). Keep the description-only path (save + audit + cache invalidation) unchanged.
- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handlers/ -run TestApp_UpdateRole -v`
Expected: PASS, including existing `TestApp_UpdateRole_SystemRoleOnlyDescription`.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/roles.go internal/handlers/roles_test.go
git commit -m "fix(handlers): reject permission changes on system roles"
```

### Task C: Models guard — pin the exact agent permission set

**Files:**
- Modify: `internal/models/models_test.go` (extend near `TestAccountsAssignPermission`)

**Interfaces:**
- Consumes: `models.SystemRolePermissions()`.
- Produces: test asserting the exact agent set.

- [ ] **Step 1: Write the failing test** `TestSystemRolePermissions_AgentIsLeastPrivilege`: assert `SystemRolePermissions()["agent"]` has length 10 and `ElementsMatch` the exact list from the Spec, and `NotContains` each of `accounts:write`, `accounts:delete`, `accounts:assign`, `contacts:write`, `contacts:delete`, `analytics:write`, `analytics:delete`, `templates:read`, `campaigns:read`, `users:read`, `roles:read`, `api_keys:read`, `webhooks:read`.
- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/models/ -run TestSystemRolePermissions_AgentIsLeastPrivilege -v`
Expected: FAIL with "function not defined" (test does not exist yet) — then passes immediately once written, since the model already defines the 10; that pass is the guard, not the fix. Record this in the report.
- [ ] **Step 3: Implement** — no production change; the test is the deliverable.
- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/models/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/models/models_test.go
git commit -m "test(models): pin exact agent permission set"
```

### Task D: Frontend — disable permission editing for system roles

**Files:**
- Modify: `frontend/src/views/settings/RoleDetailView.vue:56` (`canEditPermissions`)

**Interfaces:**
- Consumes: `isSystem` computed, `isSuperAdmin` computed.
- Produces: `canEditPermissions` false whenever `isSystem` is true.

- [ ] **Step 1: Change `canEditPermissions`** to `computed(() => !isSystem.value)`. Do not change `canEditForm` (description editing stays available). Note: no frontend unit test covers this computed; verification is `typecheck` plus the backend rejection in Task B.
- [ ] **Step 2: Run typecheck**

Run: `cd frontend && npm run typecheck`
Expected: clean, no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/settings/RoleDetailView.vue
git commit -m "fix(frontend): disable permission editing for system roles"
```

## Verification (whole branch)

- [ ] `gofmt -l internal/database internal/handlers internal/models` — no output.
- [ ] `go vet ./internal/database/ ./internal/handlers/ ./internal/models/`
- [ ] `go test ./internal/models/ ./internal/database/ ./internal/handlers/ -run 'TestSyncSystemRolePermissions|TestApp_UpdateRole|TestSystemRolePermissions|TestAccountsAssignPermission|TestSeedSystemRolesForOrg' -v` — DB/handler tests require `TEST_DATABASE_URL` (Postgres); if unavailable they skip — report exactly what ran, what skipped, and why.
- [ ] `cd frontend && npm run typecheck`
- [ ] `git diff --stat` and `git diff` review against this plan; fix regressions, do not weaken tests.
