# Whatomate — Plugins, Modules & License Entitlements

> **Updated:** 2026-06-22
> **Scope:** Plugin architecture, the DB-controlled Module system, the License→Module entitlement bridge, plugin-namespaced RBAC, and operational how-to.
> **Related:** [`ARCHITECTURE.md`](./ARCHITECTURE.md), [`LICENSING_SYSTEM_PRD.md`](./LICENSING_SYSTEM_PRD.md), [`DATABASE_SCHEMA.md`](./DATABASE_SCHEMA.md), [`CODE_MAP.md`](./CODE_MAP.md)

---

## 1. The Three Layers (read this first)

Whatomate separates ** extensibility **, ** feature gating **, and ** capacity licensing ** into three independent layers. Understanding the boundary between them is essential before changing any of them.

| Layer | Package | What it decides | Granularity | Backed by |
|---|---|---|---|---|
| **Plugin** | `internal/core` + `plugin/<name>/` | *Which code is compiled in and registers routes/handlers/models* | Per-binary (compile time) | Go blank imports in `cmd/whatomate/main.go` |
| **Module** | `internal/core/module_manager.go` + `plugin/module-management/` | *Which compiled plugins are turned ON, globally or per organization* | Per-deployment (global) and per-organization | `module_catalog`, `organization_modules` tables |
| **License** | `internal/license/` | *Host-level capacity caps and the tier that constrains which modules may be enabled* | Per-host (HWID-bound JWT) | `license_records` table + Ed25519 token |

**Rule of thumb:**
- A new product feature → **Plugin**.
- "Enable facebook for tenant A but not tenant B" → **Module** (per-org toggle).
- "Cap this deployment at 10 organizations / 5 WA endpoints per org / 50 GB storage" → **License**.
- "Only `pro`/`enterprise` tiers may use instagram" → **License→Module bridge** (tier gate on module enablement).

The three layers compose at runtime like this:

```
Request → /api/facebook/accounts
   │
   ▼
[fastglue route is registered ONLY if the plugin is compiled in]
   │
   ▼
core.GateModule("facebook-accounts", handler)
   │
   ├─ 1. License tier gate   (LicenseAllowsModule(tier, key))
   │     └─ if tier active and module NOT in tier map → 404
   │
   ├─ 2. Module state gate   (ModuleManager.IsEnabled(orgID, key))
   │     └─ global_enabled AND organization_enabled AND deps satisfied → ok
   │     └─ else → 404
   │
   ▼
handler body
   │
   ├─ 3. RBAC   (app.HasPermission(userID, resource, action, orgID))
   │     └─ core catalog (e.g. accounts:write) AND/OR plugin namespace
   │        (e.g. plugin.facebook.accounts:pages_manage)
   │
   └─ 4. Quota (a.License.CheckQuotaWithDelta(...))  ← only for countable resources
```

Each layer fails **closed** (deny by default) and returns a status that reveals nothing about the next layer:
- Not licensed / module off → `404 Not Found` (indistinguishable from a non-existent route).
- RBAC fail → `403 Forbidden` with `Insufficient permissions: <resource>:<action>`.
- Quota exceeded → `402 Payment Required` with `{resource, current, limit, over_quota}`.

---

## 2. Plugin System

### 2.1 Plugin interface

Every plugin implements `core.Plugin` (`internal/core/plugin.go`):

```go
type Plugin interface {
    Name() string
    Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error
    Routes(g *fastglue.Fastglue)
    Migrate(db *gorm.DB) error
}
```

Plugins self-register via `init()` + `core.RegisterPlugin(...)`, and are activated by a **blank import** in `cmd/whatomate/main.go`.

### 2.2 Optional capabilities

A plugin can opt into additional capabilities by implementing one of the optional interfaces. Both follow the same "embed `Plugin` + add one method" pattern, so a plugin can implement none, one, or both.

#### `ManagedPlugin` — participate in the Module system

```go
type ManagedPlugin interface {
    Plugin
    Manifest() ModuleManifest
}
```

`ModuleManifest` declares the module's key, version, schema version, dependencies, and `DefaultEnabled`. Plugins that do NOT implement this keep the legacy lifecycle (always on if compiled in).

#### `PermissionProvidingPlugin` — contribute to the RBAC catalog

```go
type PermissionProvidingPlugin interface {
    Plugin
    Permissions() []PluginPermission
}

type PluginPermission struct {
    Resource    string  // e.g. "plugin.facebook.accounts"
    Action      string  // e.g. "pages_manage"
    Description string
}
```

Declared permissions are seeded into the `permissions` table at startup via `core.SyncPluginPermissions(...)` and enforced through the **existing** `app.HasPermission(...)` machinery — there is no parallel authorization system.

**Naming convention** (required):
- `Resource`: `plugin.<pluginName>.<feature>` — fits the `size:50` column and matches the existing dotted namespace (`settings.general`, `analytics.agents`).
- `Action`: a standard `models.Action*` verb (`read`, `write`, `delete`, ...) OR a plugin-specific sub-feature verb (`pages_manage`) when the action doesn't map to a core verb.
- Combined key passed to `HasPermission`: `plugin.facebook.accounts:pages_manage`.

### 2.3 Plugin lifecycle

`cmd/whatomate/main.go` runs this exact sequence on startup:

1. `core.InitPlugins(app, db, rdb, log)` → validates manifests, sorts dependencies topologically, builds the `ModuleManager`, then calls each plugin's `Init(...)`.
2. `core.SetLicenseTierGetter(...)` → wires the license-tier resolver for `GateModule` (see §4).
3. (if `--migrate`) `core.RunPluginMigrations(db)` → runs each plugin's `Migrate(...)` + `SyncManagedModules`.
4. `core.SyncManagedModules(ctx)` → reconciles `module_catalog` / `organization_modules` with compiled manifests; seeds defaults for new organizations.
5. `core.SyncPluginPermissions(ctx, db)` → idempotently seeds every `PermissionProvidingPlugin` permission into the `permissions` table.
6. `core.RegisterPluginRoutes(g)` → registers each plugin's routes on the fastglue router.

### 2.4 Creating a new plugin

Minimal layout under `plugin/<name>/`:

```
plugin/<name>/
  plugin.go    ← registers via init(), implements Plugin (+ optional interfaces)
  handler.go   ← handlers (optional, can be in plugin.go for small plugins)
  model.go     ← plugin-owned GORM models (optional)
```

Skeleton:

```go
package myplugin

import (
    "log/slog"

    "github.com/compnew2006/whatomate/internal/core"
    "github.com/compnew2006/whatomate/internal/handlers"
    "github.com/compnew2006/whatomate/internal/middleware"
    "github.com/compnew2006/whatomate/internal/models"
    "github.com/compnew2006/whatomate/internal/tenant"
    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "github.com/valyala/fasthttp"
    "github.com/zerodha/fastglue"
    "gorm.io/gorm"
)

type Plugin struct {
    app *handlers.App
}

func init() { core.RegisterPlugin(&Plugin{}) }

func (p *Plugin) Name() string { return "my-feature" }

// ManagedPlugin: opt into the Module system (DB-controlled enable/disable).
func (p *Plugin) Manifest() core.ModuleManifest {
    return core.ModuleManifest{
        Key:            p.Name(),
        DisplayName:    "My Feature",
        Version:        "1.0.0",
        SchemaVersion:  1,
        DefaultEnabled: true,
    }
}

// PermissionProvidingPlugin: declare fine-grained, plugin-namespaced RBAC.
func (p *Plugin) Permissions() []core.PluginPermission {
    return []core.PluginPermission{{
        Resource:    "plugin.my-feature.things",
        Action:      "write",
        Description: "Create and edit things",
    }}
}

func (p *Plugin) Init(app *handlers.App, _ *gorm.DB, _ *redis.Client, _ *slog.Logger) error {
    p.app = app
    return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
    // Wrap every route in core.GateModule so the Module system + License bridge
    // gate access. Handlers are only reached if the module is licensed AND enabled.
    g.GET("/api/my-feature/things", core.GateModule(p.Name(), p.listThings))
}

func (p *Plugin) Migrate(db *gorm.DB) error {
    // Plugin owns its schema. Do NOT add plugin models to internal/database/postgres.go.
    return db.AutoMigrate(&Thing{})
}

func (p *Plugin) listThings(r *fastglue.Request) error {
    orgID, ok := middleware.GetOrganizationID(r)
    if !ok {
        return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
    }
    userID, ok := middleware.GetUserID(r)
    if !ok || p.app == nil {
        return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
    }
    // Reuse the existing RBAC machinery — no parallel auth system.
    if !p.app.HasPermission(userID, "plugin.my-feature.things", "read", orgID) {
        return r.SendErrorEnvelope(fasthttp.StatusForbidden,
            "Insufficient permissions: plugin.my-feature.things:read", nil, "")
    }
    scopedDB := tenant.ScopedDB(p.app.DB, orgID)
    var things []Thing
    if err := scopedDB.Find(&things).Error; err != nil {
        return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "query failed", nil, "")
    }
    return r.SendEnvelope(things)
}

// Compile-time assertions for both optional interfaces.
var (
    _ core.ManagedPlugin             = (*Plugin)(nil)
    _ core.PermissionProvidingPlugin = (*Plugin)(nil)
)
```

Then activate the plugin with a blank import in `cmd/whatomate/main.go`:

```go
import (
    _ "github.com/compnew2006/whatomate/plugin/my-feature"
    // ... other plugins
)
```

### 2.5 Plugin conventions (mandatory)

- **Tenant scoping in plugins:** use `middleware.GetOrganizationID(r)` + `tenant.ScopedDB(p.app.DB, orgID)`. Do NOT use `app.requestDB` (that's an unexported core-handler helper).
- **Migrations:** plugin-owned via `Migrate(db)`. Never add plugin models to `internal/database/postgres.go`.
- **Response shape:** `r.SendEnvelope(data)` for success, `r.SendErrorEnvelope(statusCode, message, nil, "")` for errors. Never raw JSON.
- **RBAC:** always go through `app.HasPermission`. Never implement a parallel permission check.
- **Route gating:** wrap every route in `core.GateModule(p.Name(), handler)`.
- **Imports:** `core` must not import `database` (cycle). Plugin-permission seeding therefore lives in `core.SyncPluginPermissions`, not in `database.SeedPermissionsAndRoles`.

---

## 3. Module System (DB-controlled feature gating)

The Module system answers: *"this plugin is compiled in, but should it be active for this organization?"*

### 3.1 Tables

| Table | Owner | Purpose |
|---|---|---|
| `module_catalog` | `internal/core/module_manager.go` | One row per compiled module. Carries `global_enabled`, `default_enabled`, `compiled_version`, `schema_version`, `installed_schema_version`, `dependencies`, `technical`. |
| `organization_modules` | `internal/core/module_manager.go` | Per-org override: `(organization_id, module_key, enabled)`. |
| `module_schema_versions` | `internal/core/module_manager.go` | Tracks the latest applied schema version per module. |

Migrations run via `ModuleManager.Migrate(ctx)` (called from `SyncManagedModules`).

### 3.2 Effective state resolution

A module is **effective-enabled** for an organization when ALL of these hold:

1. `module_catalog.global_enabled = true` (super-admin global toggle).
2. `module_catalog.installed_schema_version >= manifest.schema_version` (migration has caught up).
3. `organization_modules.enabled = true` for that org (or the manifest default if no row).
4. **Every dependency** is itself effective-enabled (recursive).

This is implemented in `ModuleManager.IsEnabled` / `ModuleManager.effective` and surfaced via `ListEffective`. The recursion is cycle-safe (visiting map).

### 3.3 Dependency safety

Disabling a module that has enabled dependents (globally or per-org) fails with `ErrModuleHasEnabledDependents`. Enabling a module automatically enables its dependencies first (transactional, recursive). This prevents broken graphs.

### 3.4 Frontend

- `frontend/src/views/settings/ModulesView.vue` — admin UI to toggle modules globally and per-org.
- `frontend/src/modules/registry.ts` — maps frontend routes to module keys (hides UI when module off).
- `frontend/src/services/modules.ts` — API client for `/api/modules/*`.

---

## 4. License → Module Bridge (entitlement control)

### 4.1 Why a bridge (not a merge)

The License is a **host-bound Ed25519 JWT** carrying global caps (`MaxOrganizations`, `MaxUsersPerOrg`, `MaxWhatsAppEndpointsPerOrg`, `MaxStorageBytesPerOrg`, `MaxWorkers`) and a `Tier`. Per-organization entitlements do NOT belong in the JWT — it's the wrong granularity. The Module system is the right granularity. The bridge makes the License tier constrain *which modules may be enabled at all*, leaving the per-org on/off to the Module system.

### 4.2 The tier map

Single source of truth: `internal/core/license_tiers.go`.

```go
var tierModules = map[string]map[string]bool{
    "trial":     {"facebook-core": true, "facebook-accounts": true},
    "starter":   {"facebook-core": true, "facebook-accounts": true, "facebook-comments": true},
    "pro":       {"*": true},   // wildcard: every registered module
    "enterprise":{"*": true},
}
```

`LicenseAllowsModule(tier, key)` returns:
- `true` if the tier has `"*"` OR the explicit key.
- `false` for unknown tiers and empty tier strings (**deny by default**).

To change entitlements for a tier, edit this map. There is intentionally no per-deployment override table (yet) — see §7.

### 4.3 Where the bridge fires

Two chokepoints, no duplication:

1. **Runtime route gate** — `core.GateModule(key, handler)` in `internal/core/module_manager.go`. Before consulting the Module state, it calls `licenseTierGetter()`; if the tier is active and the module isn't licensed, it returns `404`. The getter is injected by `cmd/whatomate/main.go` via `core.SetLicenseTierGetter(...)` to avoid a `core → license` import cycle.
2. **Admin write gate** — `plugin/module-management` `updateGlobal` and `updateOrganization`. Before flipping a module on, they call `licenseAllows(key)`; refusal returns `403` with `{"error":"module_not_licensed","tier":"..."}` and writes a `license_deny` audit row.

**Backwards compatibility:** when `app.License == nil` or `state.Enabled == false`, the tier is `""` and `LicenseAllowsModule` returns `false` — but `GateModule` treats `tier == ""` as "license inactive, skip the tier check" and falls back to the pure Module state. So unlicensed deployments behave exactly as before.

### 4.4 Tier flow at request time

```
request → GateModule(key, h)
   tier = licenseTierGetter()           // "" when no license
   if tier != "" && !LicenseAllowsModule(tier, key):
       return 404
   ... continue to Module state check ...
```

---

## 5. Quotas & Limits (License caps)

Quotas are enforced by `license.Service.CheckQuotaWithDelta(ctx, resource, orgID, delta)` and surfaced via `app.checkQuotaOrRespond` / `app.checkQuotaWithDeltaOrRespond` (the DRY helpers — use these, do not call `CheckQuotaWithDelta` directly from handlers).

| Resource constant | Counted against | Enforced at |
|---|---|---|
| `ResourceOrganizations` | `MaxOrganizations` (global) | Organization create |
| `ResourceUsers` | `MaxUsersPerOrg` (per org) | User invite/create |
| `ResourceEndpoints` | `MaxWhatsAppEndpointsPerOrg` (per org) | WhatsApp endpoint create |
| `ResourceStorage` | `MaxStorageBytesPerOrg` (per org) | Before every upload (hard limit) |

Response on exceeded: `402 Payment Required` with `{resource, current, limit, over_quota}`.

When the license is in a state requiring quota cleanup (`RequiresQuotaCleanup()`), `checkQuotaOrRespond` short-circuits with `SendLicenseQuotaCleanupRequired`.

---

## 6. Audit Trail

### 6.1 License events (`license_events`)

Append-only, hardware-bound. Written by `license.Service.recordEvent(...)`. Covers activation, expiry, suspension, grace entry, quota overages. **Exempt from tenant scoping** (no `organization_id`).

### 6.2 Module events (`module_events`) — NEW

Append-only, org-scoped. Written by `plugin/module-management.recordEvent(...)` on every give/ungive attempt.

| Field | Values |
|---|---|
| `scope` | `global` \| `organization` |
| `module_key` | the module being toggled |
| `action` | `enable` \| `disable` \| `license_deny` \| `conflict` |
| `enabled` | the requested new boolean (nullable) |
| `organization_id` | NULL for global scope, set for org scope |
| `actor_user_id` | who made the change (nullable for system) |
| `actor_email` | convenience denormalized email |
| `reason` | e.g. `"module not licensed for current tier"`, `"module has enabled dependents"` |
| `details` | JSONB free-form context |

Read via:
- `GET /api/admin/modules/events` — super-admin only, global feed.
- `GET /api/organizations/{id}/modules/events` — org-scoped, requires `organizations:read` on the target org (or super-admin).

### 6.3 Agent-selection / uploads-cleanup audits (precedent)

Per-domain audit tables (`agent_selection_audit_events`, `instance_uploads_cleanup_audits`) follow the same shape and are the established pattern for new audit surfaces.

---

## 7. How to use — operator / admin runbook

### 7.1 "Enable Facebook for tenant A but not tenant B"

**Precondition:** the deployment's license tier must include the `facebook-accounts` module (see §4.2). On `pro`/`enterprise` this is automatic. On `trial`/`starter`, check the tier map.

**Steps:**
1. As super-admin, ensure the module is globally enabled:
   ```
   PUT /api/admin/modules/facebook-accounts
   {"enabled": true}
   ```
2. For tenant A's organization, enable it:
   ```
   PUT /api/organizations/<orgA-id>/modules/facebook-accounts
   {"enabled": true}
   ```
3. For tenant B's organization, disable it (or simply never enable it):
   ```
   PUT /api/organizations/<orgB-id>/modules/facebook-accounts
   {"enabled": false}
   ```
4. Verify:
   ```
   GET /api/modules/effective           # from tenant A's session → facebook-accounts effective_enabled=true
   GET /api/modules/effective           # from tenant B's session → effective_enabled=false
   ```
   Tenant A's `/api/facebook/*` routes now respond; tenant B's return `404`.

**Audit:** every call above writes a `module_events` row. Review with `GET /api/admin/modules/events`.

### 7.2 "Revoke a module from a tier" (deprecation / un-licensing)

Edit `tierModules` in `internal/core/license_tiers.go` to remove the module key from the tier (or remove the tier entry). Rebuild + redeploy. Existing org-level enable rows stay in place but become ineffective: `GateModule` returns `404` because the tier no longer licenses the module. No tenant data is deleted.

To audit who had it enabled before deprecation: `GET /api/admin/modules/events?action=enable&module_key=<key>`.

### 7.3 "Give a user permission to manage Facebook pages only"

This uses the **plugin RBAC** layer (§2.2). The `plugin.facebook.accounts:pages_manage` permission is seeded automatically at startup.

1. Ensure the user's role (or a custom role) includes `plugin.facebook.accounts:pages_manage`. Roles are managed via the standard `/api/roles` endpoints.
2. The user can now call connect/disconnect/remove page endpoints. A user without this permission (but with `accounts:write`) can still list/create accounts but **cannot** modify page connections — they get `403 Insufficient permissions: plugin.facebook.accounts:pages_manage`.

Super-admins bypass all RBAC checks (`HasPermission` short-circuits on `IsSuperAdmin`).

### 7.4 "Check why a route returns 404"

Walk the layers in order:
1. Is the plugin compiled in? → check `cmd/whatomate/main.go` blank imports.
2. Is the license tier active and does it include the module? → `GET /api/license/bootstrap` shows the tier; compare to `internal/core/license_tiers.go`.
3. Is the module globally enabled? → `GET /api/admin/modules` (super-admin).
4. Is the module enabled for the org? → `GET /api/modules/effective` from the user's session.
5. Are dependencies enabled? → `ListEffective` shows `effective_enabled` per module including deps.

### 7.5 "Add a new plugin-scoped permission"

1. Make the plugin implement `core.PermissionProvidingPlugin` (see §2.4 skeleton).
2. Restart — `core.SyncPluginPermissions` seeds it idempotently into `permissions`.
3. (Admin role) To auto-grant it to every admin, extend `models.SystemRolePermissions()` or include it explicitly per role via `/api/roles`.
4. Enforce in the handler with `app.HasPermission(userID, "plugin.<name>.<feature>", "<action>", orgID)`.

### 7.6 "Add a new tier or change what it includes"

Edit `tierModules` in `internal/core/license_tiers.go`. Common operations:

```go
// Add a new tier
"business": {
    "facebook-core": true, "facebook-accounts": true,
    "facebook-comments": true, "instagram-direct": true,
},

// Restrict an existing tier (remove a key)
"starter": {
    "facebook-core": true, "facebook-accounts": true,
    // facebook-comments removed
},
```

Rebuild + redeploy. The change is read on every request — no DB migration needed. Existing `organization_modules` rows that are no longer licensed become inert (route returns `404`).

---

## 8. API Reference (module management)

All endpoints are registered by `plugin/module-management`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/modules/effective` | authenticated | List modules effective for the caller's organization |
| `GET` | `/api/admin/modules` | super-admin | List modules with global+org state for `uuid.Nil` |
| `PUT` | `/api/admin/modules/{key}` | super-admin | Set **global** enable flag. Body: `{"enabled": bool}`. 403 `module_not_licensed` if tier forbids. |
| `GET` | `/api/admin/modules/events` | super-admin | Global module audit feed (last 200) |
| `GET` | `/api/organizations/{id}/modules` | `organizations:read` on `{id}` (or super-admin) | List modules for a specific org |
| `PUT` | `/api/organizations/{id}/modules/{key}` | `organizations:write` on `{id}` (or super-admin) | Set **per-org** enable flag. 403 `module_not_licensed` if tier forbids. |
| `GET` | `/api/organizations/{id}/modules/events` | `organizations:read` on `{id}` (or super-admin) | Org-scoped module audit feed (last 200) |

**Error envelopes:**
- `403 {"error":"module_not_licensed","key":"...","tier":"..."}` — tier does not include this module.
- `409 "Module has enabled dependents"` — disabling would orphan an enabled dependent.
- `404 "Module not found"` — unknown module key.

---

## 9. Known Limitations & Deferred Work

| Gap | Status | Workaround |
|---|---|---|
| **Scheduled activation / expiry** of modules (`valid_from` / `valid_until`) | Not implemented | Toggle manually; track externally |
| **Per-deployment plan overrides** (a `license_module_overrides` table) | Not implemented | Edit `tierModules` and rebuild |
| **Full plugin-RBAC rollout** across all facebook/instagram/whatsapp plugins | Pattern proven on `facebook-accounts` only | Adopt the `PermissionProvidingPlugin` pattern per plugin (mechanical) |
| **Admin role auto-inherits plugin permissions** | `SystemRolePermissions()` iterates `DefaultPermissions()` only | Grant plugin perms to admin role explicitly until extended |
| **License-deny E2E test** with a real `*license.Service` | Unit-covered via `LicenseAllowsModule` + `GateModule` stubs | Add a DB-backed E2E test in a follow-up |

---

## 10. File Map

| File | Role |
|---|---|
| `internal/core/plugin.go` | `Plugin`, `ManagedPlugin`, `PermissionProvidingPlugin` interfaces; `ResolvePlugins`; `SyncPluginPermissions`; `PluginPermissions()` |
| `internal/core/module_manager.go` | `ModuleManager`; `GateModule`; `SetLicenseTierGetter` |
| `internal/core/license_tiers.go` | `tierModules` map; `LicenseAllowsModule` |
| `internal/license/service.go` | `Service.CurrentState()`; `CheckQuotaWithDelta`; `IsLocked` |
| `internal/license/token.go` | `LicenseClaims` (Ed25519 JWT, host-bound) |
| `internal/models/models.go` | `LicenseRecord`, `LicenseEvent` |
| `internal/models/roles.go` | `Permission`, `DefaultPermissions`, `SystemRolePermissions` |
| `internal/handlers/license.go` | `checkQuotaOrRespond`, `checkQuotaWithDeltaOrRespond`, license-block helpers |
| `internal/handlers/cache.go` | `App.HasPermission` (RBAC entry point) |
| `plugin/module-management/plugin.go` | Module CRUD + audit writer + events endpoints + license-deny guard |
| `plugin/module-management/audit.go` | `ModuleEvent` model + `MigrateModuleEvents` |
| `cmd/whatomate/main.go` | Wires `SetLicenseTierGetter` + `SyncPluginPermissions` at startup |
| `frontend/src/views/settings/ModulesView.vue` | Admin UI for module toggles |
| `frontend/src/modules/registry.ts` | Frontend route → module-key mapping |
| `frontend/src/services/modules.ts` | Modules API client |
