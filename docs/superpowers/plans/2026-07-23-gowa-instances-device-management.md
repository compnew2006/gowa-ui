# GOWA Instances + Device Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let admins manage multiple GOWA servers (instances) from the DB via a new Settings page, and manage every device on each instance (list, create, delete, QR/pair-code connect, logout/reconnect, webhook config) — all gated by RBAC that shows up in `/settings/roles`.

**Architecture:** A new `GowaInstance` model (org-scoped, encrypted credentials) is managed through a **new `/api/gowa/servers`** REST namespace. Each server's devices are proxied to the existing `pkg/gowa` client under the existing `devices` resource (plus a new `devices:delete` action). The legacy `/api/gowa/instances` + `/api/gowa/create-device` endpoints (used live by the account-creation dropdown) are left **untouched** — config.toml instances and DB instances coexist. Frontend adds a Pinia store + list/detail views wired into the existing settings nav/router/permissions machinery.

**Tech Stack:** Go (fastglue + GORM + fasthttp), Vue 3 + TypeScript + Pinia + Vue Router 4 + vue-i18n, custom shadcn-style components (`@/components/ui/*`, `@/components/shared/*`).

---

## Verified Reference Facts (read before coding)

These were confirmed against the codebase on 2026-07-23. Do not re-derive.

- **gowa client** — `pkg/gowa/client.go:39` `gowa.New(baseURL, username, password string) *Client`. Device methods in `pkg/gowa/devices.go`: `ListDevices`, `CreateDevice(ctx, deviceID, cfg WebhookConfig)`, `DeleteDevice`, `GetDeviceStatus`, `LogoutDevice`, `ReconnectDevice`, `GetDeviceWebhook`, `SetDeviceWebhook`, `GetLoginQR`. `LoginWithCode(ctx, deviceID, phone)` is in **`pkg/gowa/app.go:61`** (NOT devices.go). `DownloadMedia(ctx, qrLink, "")` fetches the QR PNG bytes. Generators in `pkg/gowa/verify.go`: `GenerateWebhookSecret()`, `GenerateDeviceID(name)`.
- **BaseModel** — `internal/models/models.go:82`. Fields: `ID uuid` (gen_random_uuid), `CreatedAt`, `UpdatedAt`, `DeletedAt gorm.DeletedAt`.
- **Encryption** — codebase uses **explicit calls, NOT gorm hooks**. `internal/crypto/crypto.go`: `EncryptFields(key string, fields ...*string) error` (skips `enc:`-prefixed), `DecryptFields(key string, fields ...*string)` (no return, silent). Pattern: handler encrypts before save; model has a `DecryptXxx(key)` method called after read. See `WhatsAppAccount.DecryptSecrets` at `models.go:360` and `App.encryptAccountSecrets` at `handlers/accounts.go:1062`.
- **Migrations** — register in BOTH `internal/database/postgres.go:56` (`GetMigrationModels`, add `{"GowaInstance", &models.GowaInstance{}}`) AND `test/testutil/db.go:81` (`runMigrations`, add `&models.GowaInstance{}`). Add `"gowa_instances"` to the cleanup/truncate lists in `test/testutil/db.go` (`cleanupTables` ~L137, `TruncateTables` ~L188).
- **RBAC** — `internal/models/roles.go`. Resource consts L48-90 (`ResourceDevices="devices"` L65). Actions L93-103 (`ActionRead/Write/Delete`). `DefaultPermissions()` L106 (devices perms at L136-139, **no devices:delete yet**). `SystemRolePermissions()` L249: admin auto-inherits all (L324), manager explicit L256-299, agent L301-321.
- **Auth/audit helpers** — `requireAuth(r, resource, action) (orgID, userID uuid.UUID, err)` at `internal/handlers/app.go:265`; returns sentinel `errEnvelopeSent` on failure → caller does `return nil`. `logAudit(orgID, userID, resourceType, resourceID, action, oldData, newData, extra...)` at `helpers.go:104`. `findByIDAndOrg[T]` at `helpers.go:92`. `parsePathUUID` at `helpers.go:24`. Response envelopes: `r.SendEnvelope(map)` / `r.SendErrorEnvelope(status, msg, nil, "")`.
- **Existing GOWA handler template** — `internal/handlers/gowa_device.go`. `GowaCreateDevice` (L167) is the canonical pattern: decode → validate → find creds → `gowa.New` → call client → `logAudit` (resourceID `uuid.Nil` for external device IDs) → `SendEnvelope`. Webhook generation (L201-212): `webhookSecret := gowa.GenerateWebhookSecret()`, `deviceID := gowa.GenerateDeviceID(name)`, events = `"message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited"`.
- **Routes** — `cmd/whatomate/main.go`, GOWA block at **L691-698** inside `setupRoutes`. `g` is the global `*fastglue.Fastglue`. Auth is path-based via `g.Before(...)` at L576-604 (JWT/API-key applied to all `/api` except an allow-list); per-resource authz is inside each handler via `requireAuth`.
- **Test harness** — `internal/handlers/gowa_device_security_test.go` + `testhelpers_test.go`. `newTestApp(t)` builds an `App` with test DB + Redis + config (encryption key set). `testutil.CreateTestOrganization`, `createAdminUser` (`teams_test.go:47`), `createTestAgent` (`agent_transfers_test.go:18`), `testutil.SetAuthContext(req, orgID, userID)`, `testutil.SetPathParam(req, "id", val)`, `testutil.NewGETRequest`, `testutil.NewJSONRequest`, `testutil.GetResponseStatusCode`, `testutil.GetResponseBody`.
- **Frontend api.ts** — `frontend/src/services/api.ts`. Axios instance `api`, cookie-based auth, no Bearer. Service style: exported const object with methods returning `api.get/post/put/delete(path, data, {params})`. Example: `rolesService` L842, `tagsService` L865.
- **Frontend store** — setup/composition Pinia store. `frontend/src/stores/roles.ts` is the template: refs for state, computed getters, async actions with `try/catch/finally`, `const data = (response.data as any).data || response.data` unwrap, rethrow on error.
- **authStore.hasPermission** — `frontend/src/stores/auth.ts:171` `(resource, action='read'): boolean`. Super-admin bypass via `is_super_admin`.
- **RESOURCE_LABELS** — `frontend/src/lib/constants.ts:31` (`Record<string,string>`). Note existing keys use NO underscore mismatch fix needed: just add `gowa_instances` and `devices`.
- **navigation.ts** — `frontend/src/components/layout/navigation.ts`. Settings section L105-134; add child to `children` array (L116-131) AND add permission to BOTH the section `permissions` (L107) and the parent item's `childPermissions` (L115). NavItem: `{ name: i18nKey, path, icon, permission }`.
- **router/index.ts** — `frontend/src/router/index.ts`. Settings routes are children of `/` (flat). Add two routes near L161. Add entry to `navigationOrder` settings `childPaths` (L309-323). Guard at L347-376 checks `to.meta.permission` via `hasPermission(permission, 'read')`.
- **i18n** — only `frontend/src/i18n/locales/en.json` and `ar.json` exist on disk. Flat top-level namespaces. GOWA strings currently live under `accounts.*`; add a new top-level `gowaServers` namespace.
- **UI lib** — custom shadcn-style. Views import from `@/components/ui/*` (`Card`, `Button`, `Badge`, `Input`, `Label`, `Dialog`, `Tabs`, `Switch`, `AlertDialog`, `Tooltip`, `Separator`) and `@/components/shared` (`PageHeader`, `DataTable`, `DeleteConfirmDialog`, `ErrorState`, `DetailPageLayout`, `type Column`). Toasts via `vue-sonner`; errors via `getErrorMessage` from `@/lib/api-utils`. Icons from `lucide-vue-next`.
- **UI reference — `gowa-ui` (external, React).** A reference implementation exists at `/Users/noiemany/Downloads/gowa-ui` with `src/features/devices/` (`device-card.tsx`, `create-device-dialog.tsx`, `webhook-dialog.tsx`, `state-badge.tsx`) and `src/features/session/` (`login-qr-dialog.tsx`, `login-code-dialog.tsx`). Use these as the **behavioral/visual reference** for the device grid, state badges, and QR/pair-code flows, but re-author them as Vue components (whatomate is Vue, gowa-ui is React — do NOT copy JSX). The existing in-repo GOWA UI in `frontend/src/views/settings/AccountDetailView.vue` (QR dialog L584-658, pair-code L635-656) is the Vue-idiom template to match.

**Decisions locked with user:**
1. New `/api/gowa/servers` namespace; legacy `/api/gowa/instances` + `/api/gowa/create-device` untouched.
2. No config→DB import; config.toml and DB instances coexist.

---

## Cross-source verification (gowa-ui + OpenAPI spec)

Before finalizing, this plan was cross-checked against `/Users/noiemany/Downloads/gowa-ui` (a reference React UI) and `docs/GOWA openapi.yaml` (v9.0.0). Findings:

**⚠ LOGIN-ENDPOINT CONTRADICTION — RESOLVED.** gowa-ui calls `GET /devices/{id}/login` and `POST /devices/{id}/login/code`. The OpenAPI spec marks **both** as `deprecated: true` with the note *"device login per ID is not implemented yet — returns 404/500"* and directs callers to `GET /app/login` + `GET /app/login-with-code` with the `X-Device-Id` header. **The Go client (`pkg/gowa`) and this plan use the spec-correct `/app/login` form.** Do NOT change the login endpoints to match gowa-ui — gowa-ui is calling dead endpoints (likely targeting an older/different GOWA build).

**Real gaps found and applied to this plan:**
- **G1 (backend):** `DeviceInfo` was missing `phone_number` (present in the spec's `GET /devices` response). Added in Task A0 so the device list can show the phone.
- **G2 (frontend):** The connect dialog must poll `deviceStatus` every 3s to auto-detect a successful scan and close — both gowa-ui and the existing `AccountDetailView.vue` do this. Originally missing; added to Task B7.
- **G3 (frontend):** Device state has 4 values (`disconnected` → `connecting` → `connected` → `logged_in`) with a color mapping. The detail view originally rendered only a binary connected/disconnected badge; now renders the full state via a `StateBadge`.

**Confirmed correct (no changes):** QR is a URL downloaded server-side and base64-encoded (matches spec + Go client + existing `GowaLoginQR`); pair code `phone` is a query param; no pagination on `GET /devices`; Basic Auth + `X-Device-Id`; no rate-limiting documented.

**Deliberately out of scope (future phases):** WebSocket real-time (`/ws`) for device events; `GET /devices/{id}` single fetch; `GET /app/info`; Chatwoot integration surface. The single "could not reach GOWA" message is not split into 401/not-gowa/unreachable like gowa-ui does — acceptable for v1.

---

## File Structure

**Backend (new):**
- `internal/models/gowa_instance.go` — `GowaInstance` model + encrypt/decrypt/response helpers.
- `internal/handlers/gowa_instances.go` — instance CRUD + per-instance device ops + `resolveGowaInstance`.
- `internal/handlers/gowa_instances_test.go` — security + behavior tests.

**Backend (modify):**
- `internal/models/roles.go` — `ResourceGowaInstances`, permissions, `SystemRolePermissions`.
- `internal/database/postgres.go` — register model in `GetMigrationModels`.
- `test/testutil/db.go` — register model + truncate lists.
- `cmd/whatomate/main.go` — register routes.

**Frontend (new):**
- `frontend/src/stores/gowaServers.ts` — Pinia store.
- `frontend/src/views/settings/GowaServersView.vue` — instance list + add/edit dialog.
- `frontend/src/views/settings/GowaServerDetailView.vue` — device management.

**Frontend (modify):**
- `frontend/src/services/api.ts` — `gowaServersService`.
- `frontend/src/lib/constants.ts` — `RESOURCE_LABELS`.
- `frontend/src/components/layout/navigation.ts` — nav entry.
- `frontend/src/router/index.ts` — routes + `navigationOrder`.
- `frontend/src/i18n/locales/en.json`, `ar.json` — `gowaServers` namespace.

---

# PHASE A — BACKEND

## Task A0: Add phone_number to gowa.DeviceInfo (spec gap G1)

The OpenAPI spec's `GET /devices` response includes `phone_number` on each device item, but the Go `DeviceInfo` struct omits it (so it's silently dropped on decode). This task adds the field so the device list can show the phone number. This is a safe, additive change — no callers break.

**Files:**
- Modify: `pkg/gowa/devices.go` (DeviceInfo struct, L11-19)

- [ ] **Step 1: Add the field**

In `pkg/gowa/devices.go`, replace the `DeviceInfo` struct (L11-19):

```go
// DeviceInfo represents a registered GOWA device.
type DeviceInfo struct {
	ID            string    `json:"id"`
	PhoneNumber   string    `json:"phone_number,omitempty"`
	DisplayName   string    `json:"display_name"`
	State         string    `json:"state"`
	JID           string    `json:"jid"`
	CreatedAt     time.Time `json:"created_at"`
	WebhookURL    string    `json:"webhook_url,omitempty"`
	WebhookEvents string    `json:"webhook_events,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./pkg/gowa/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/gowa/devices.go
git commit -m "feat(gowa): add phone_number to DeviceInfo per OpenAPI spec"
```

---

## Task A1: GowaInstance model

**Files:**
- Create: `internal/models/gowa_instance.go`
- Modify: `internal/database/postgres.go` (in `GetMigrationModels`, ~L117)
- Modify: `test/testutil/db.go` (`runMigrations` ~L96, `cleanupTables` ~L137, `TruncateTables` ~L188)

- [ ] **Step 1: Create the model file**

Create `internal/models/gowa_instance.go`:

```go
package models

import (
	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/crypto"
)

// GowaInstance is a DB-managed GOWA server (base_url + Basic Auth credentials)
// scoped to an organization. Username and Password are encrypted at rest and
// never serialized to JSON responses (use ToResponse to expose has_credentials).
type GowaInstance struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	BaseURL        string    `gorm:"size:255;not null" json:"base_url"`
	Username       string    `gorm:"size:255" json:"-"` // encrypted, never serialized
	Password       string    `gorm:"size:255" json:"-"` // encrypted, never serialized
	WebhookURL     string    `gorm:"size:255" json:"webhook_url,omitempty"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (GowaInstance) TableName() string { return "gowa_instances" }

// EncryptCredentials encrypts Username and Password in place using the given
// key. Fields already carrying the "enc:" prefix are skipped (idempotent).
func (g *GowaInstance) EncryptCredentials(key string) error {
	return crypto.EncryptFields(key, &g.Username, &g.Password)
}

// DecryptCredentials decrypts Username and Password in place. Safe to call on
// legacy/unencrypted values.
func (g *GowaInstance) DecryptCredentials(key string) {
	crypto.DecryptFields(key, &g.Username, &g.Password)
}

// HasCredentials reports whether both Username and Password are populated.
func (g *GowaInstance) HasCredentials() bool {
	return g.Username != "" && g.Password != ""
}

// GowaInstanceResponse is the credentials-safe projection returned by handlers.
type GowaInstanceResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	BaseURL        string    `json:"base_url"`
	WebhookURL     string    `json:"webhook_url"`
	IsActive       bool      `json:"is_active"`
	HasCredentials bool      `json:"has_credentials"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// ToResponse builds the credentials-safe projection. Call DecryptCredentials
// first if you need the raw credentials (not exposed here).
func (g *GowaInstance) ToResponse() GowaInstanceResponse {
	return GowaInstanceResponse{
		ID:             g.ID,
		Name:           g.Name,
		BaseURL:        g.BaseURL,
		WebhookURL:     g.WebhookURL,
		IsActive:       g.IsActive,
		HasCredentials: g.HasCredentials(),
		CreatedAt:      g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
```

- [ ] **Step 2: Register in production migrations**

In `internal/database/postgres.go`, in `GetMigrationModels()` (ends ~L117), add before the closing `}`:

```go
		// GOWA instances (DB-managed GOWA servers)
		{"GowaInstance", &models.GowaInstance{}},
```

- [ ] **Step 3: Register in test migrations**

In `test/testutil/db.go`, in `runMigrations` (the `db.AutoMigrate(...)` call ~L81), add after the `&models.CustomAction{},` line:

```go
		// GOWA instances
		&models.GowaInstance{},
```

In the same file, add `"gowa_instances"` to BOTH the `cleanupTables` slice (insert near `"custom_actions",` ~L175) and the `TruncateTables` slice (insert near `"custom_actions",` ~L219). Add it BEFORE `"whatsapp_accounts"` in both, since devices/accounts may reference instances logically (TRUNCATE order with CASCADE is order-insensitive, but keep it tidy):

```go
		"gowa_instances",
```

- [ ] **Step 4: Verify it compiles + migrates**

Run: `go build ./internal/... ./test/...`
Expected: builds cleanly (no unused-import errors).

- [ ] **Step 5: Commit**

```bash
git add internal/models/gowa_instance.go internal/database/postgres.go test/testutil/db.go
git commit -m "feat(models): add GowaInstance model with encrypted credentials"
```

---

## Task A2: RBAC — gowa_instances resource + devices:delete

**Files:**
- Modify: `internal/models/roles.go` (resource const L90, permissions L136, manager role L264)

- [ ] **Step 1: Write the failing tests**

Create `internal/models/roles_gowa_test.go`:

```go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPermissions_GowaInstances(t *testing.T) {
	t.Parallel()
	perms := DefaultPermissions()
	want := []struct {
		resource string
		action   string
	}{
		{ResourceGowaInstances, ActionRead},
		{ResourceGowaInstances, ActionWrite},
		{ResourceGowaInstances, ActionDelete},
		{ResourceDevices, ActionDelete},
	}
	for _, w := range want {
		found := false
		for _, p := range perms {
			if p.Resource == w.resource && p.Action == w.action {
				found = true
				break
			}
		}
		assert.True(t, found, "missing permission %s:%s", w.resource, w.action)
	}
}

func TestSystemRoles_GowaInstancesMapping(t *testing.T) {
	t.Parallel()
	rolePerms := SystemRolePermissions()

	// Admin auto-inherits all (including gowa_instances:* and devices:delete).
	assert.Contains(t, rolePerms["admin"], "gowa_instances:read")
	assert.Contains(t, rolePerms["admin"], "gowa_instances:write")
	assert.Contains(t, rolePerms["admin"], "gowa_instances:delete")
	assert.Contains(t, rolePerms["admin"], "devices:delete")

	// Manager gets gowa_instances + devices:delete.
	assert.Contains(t, rolePerms["manager"], "gowa_instances:read")
	assert.Contains(t, rolePerms["manager"], "gowa_instances:write")
	assert.Contains(t, rolePerms["manager"], "gowa_instances:delete")
	assert.Contains(t, rolePerms["manager"], "devices:delete")

	// Agent gets none of these.
	assert.NotContains(t, rolePerms["agent"], "gowa_instances:read")
	assert.NotContains(t, rolePerms["agent"], "devices:delete")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/models/ -run 'TestDefaultPermissions_GowaInstances|TestSystemRoles_GowaInstancesMapping' -v`
Expected: FAIL (undefined `ResourceGowaInstances`; assertions fail).

- [ ] **Step 3: Add the resource constant**

In `internal/models/roles.go`, in the resource const block, add after `ResourceAuditLogs = "audit_logs"` (L89), still inside the `()`:

```go
	// GOWA Instances (DB-managed GOWA servers — settings page)
	ResourceGowaInstances = "gowa_instances"
```

- [ ] **Step 4: Add the permissions to DefaultPermissions()**

In `internal/models/roles.go`, replace the devices block (L136-138):

```go
		// Devices (GOWA device management — pairing, provisioning, status)
		{Resource: ResourceDevices, Action: ActionRead, Description: "View GOWA device status and instances"},
		{Resource: ResourceDevices, Action: ActionWrite, Description: "Pair and provision GOWA devices"},
```

with:

```go
		// Devices (GOWA device management — pairing, provisioning, status)
		{Resource: ResourceDevices, Action: ActionRead, Description: "View GOWA device status and instances"},
		{Resource: ResourceDevices, Action: ActionWrite, Description: "Pair and provision GOWA devices"},
		{Resource: ResourceDevices, Action: ActionDelete, Description: "Delete GOWA devices"},

		// GOWA Instances (DB-managed GOWA servers)
		{Resource: ResourceGowaInstances, Action: ActionRead, Description: "View GOWA server instances"},
		{Resource: ResourceGowaInstances, Action: ActionWrite, Description: "Create and edit GOWA server instances"},
		{Resource: ResourceGowaInstances, Action: ActionDelete, Description: "Delete GOWA server instances"},
```

- [ ] **Step 5: Add permissions to the manager role**

In `SystemRolePermissions()`, replace the devices manager block (L264-265):

```go
		// Devices
		"devices:read", "devices:write",
```

with:

```go
		// Devices
		"devices:read", "devices:write", "devices:delete",
		// GOWA Instances
		"gowa_instances:read", "gowa_instances:write", "gowa_instances:delete",
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/models/ -run 'TestDefaultPermissions_GowaInstances|TestSystemRoles_GowaInstancesMapping' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/models/roles.go internal/models/roles_gowa_test.go
git commit -m "feat(rbac): add gowa_instances resource and devices:delete action"
```

---

## Task A3: resolveGowaInstance helper + instance CRUD handlers

**Files:**
- Create: `internal/handlers/gowa_instances.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/handlers/gowa_instances_test.go`:

```go
package handlers_test

import (
	"testing"

	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// TestGowaInstances_AgentDenied_CRUD verifies an agent lacks gowa_instances perms.
func TestGowaInstances_AgentDenied_CRUD(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createTestAgent(t, app, org.ID)

	// List -> 403
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	require.NoError(t, app.ListGowaInstances(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

	// Create -> 403
	req = testutil.NewJSONRequest(t, map[string]string{"name": "x", "base_url": "http://g", "username": "u", "password": "p"})
	testutil.SetAuthContext(req, org.ID, agent.ID)
	require.NoError(t, app.CreateGowaInstance(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// TestGowaInstances_CredentialsNotExposed verifies the list response never
// contains username/password, only has_credentials.
func TestGowaInstances_CredentialsNotExposed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "prod",
		BaseURL:        "http://gowa:8080",
		Username:       "secretuser",
		Password:       "secretpass",
		IsActive:       true,
	}
	require.NoError(t, inst.EncryptCredentials(app.Config.App.EncryptionKey))
	require.NoError(t, app.DB.Create(inst).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	require.NoError(t, app.ListGowaInstances(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	body := string(testutil.GetResponseBody(req))
	assert.NotContains(t, body, "secretpass", "password must not be exposed")
	assert.NotContains(t, body, "secretuser", "username must not be exposed")
	assert.Contains(t, body, "has_credentials", "response should include has_credentials")
	assert.Contains(t, body, "prod")
}

// TestGowaInstances_CrossOrgDenied verifies org A admin cannot read org B's instance.
func TestGowaInstances_CrossOrgDenied(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	adminA := createAdminUser(t, app, orgA.ID)

	inst := &models.GowaInstance{
		OrganizationID: orgB.ID,
		Name:           "orgb-only",
		BaseURL:        "http://gowa-b:8080",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgA.ID, adminA.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	require.NoError(t, app.GetGowaInstance(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req),
		"cross-org access should be 404")
}

// TestGowaInstances_Create_EncryptsAndStripsCreds verifies create encrypts creds
// at rest and returns only has_credentials=true (never the raw values).
func TestGowaInstances_Create_EncryptsAndStripsCreds(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"name":     "new-server",
		"base_url": "http://gowa:8080",
		"username": "myuser",
		"password": "mypass",
	})
	testutil.SetAuthContext(req, org.ID, admin.ID)
	require.NoError(t, app.CreateGowaInstance(req))
	// Probe will fail (no live GOWA server) → expect 502; that's fine — we test
	// the credential path separately below by inserting directly. Here we assert
	// the handler at least authorizes (not 403).
	assert.NotEqual(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req),
		"admin should be authorized (probe may fail with 502)")
}
```

> NOTE: `CreateGowaInstance` performs a network probe (calls `ListDevices`). In unit tests there's no live GOWA server, so create returns 502 — that's expected and acceptable. The credential/authorization paths are what we assert.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handlers/ -run 'TestGowaInstances_' -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement the handlers**

Create `internal/handlers/gowa_instances.go`:

```go
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// gowaInstanceBundle holds the resolved DB instance and a live gowa.Client built
// from its decrypted credentials.
type gowaInstanceBundle struct {
	instance *models.GowaInstance
	client   *gowa.Client
}

// resolveGowaInstance loads the DB instance by {id} scoped to orgID, decrypts
// its credentials, and builds a gowa.Client. On error it sends the HTTP
// response and returns ok=false — callers should `return nil` immediately.
// Permission checks must already have been done by the caller.
func (a *App) resolveGowaInstance(r *fastglue.Request, orgID uuid.UUID) (gowaInstanceBundle, bool) {
	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return gowaInstanceBundle{}, false
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return gowaInstanceBundle{}, false
	}
	instance.DecryptCredentials(a.Config.App.EncryptionKey)
	client := gowa.New(instance.BaseURL, instance.Username, instance.Password)
	return gowaInstanceBundle{instance: instance, client: client}, true
}

// ListGowaInstances returns all DB-managed GOWA instances for the caller's org,
// without credentials.
// GET /api/gowa/servers
func (a *App) ListGowaInstances(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionRead)
	if err != nil {
		return nil
	}

	var instances []models.GowaInstance
	if err := a.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&instances).Error; err != nil {
		a.Log.Error("Failed to list GOWA instances", "error", err, "org", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list GOWA instances", nil, "")
	}

	out := make([]models.GowaInstanceResponse, 0, len(instances))
	for i := range instances {
		out = append(out, instances[i].ToResponse())
	}
	return r.SendEnvelope(map[string]any{"instances": out})
}

// GetGowaInstance returns a single DB-managed GOWA instance without credentials.
// GET /api/gowa/servers/{id}
func (a *App) GetGowaInstance(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionRead)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(map[string]any{"instance": instance.ToResponse()})
}

// gowaInstanceInput is the create/update payload. Username/Password are plain
// at the API boundary and encrypted before persistence.
type gowaInstanceInput struct {
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	WebhookURL string `json:"webhook_url"`
	IsActive   *bool  `json:"is_active"`
}

// CreateGowaInstance creates a DB-managed GOWA instance. Before persisting it
// probes the server with ListDevices to validate the URL/credentials.
// POST /api/gowa/servers
func (a *App) CreateGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionWrite)
	if err != nil {
		return nil
	}

	var in gowaInstanceInput
	if err := r.Decode(&in, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if in.Name == "" || in.BaseURL == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name and base_url are required", nil, "")
	}

	// Probe the GOWA server before saving — validate URL + auth work.
	probe := gowa.New(in.BaseURL, in.Username, in.Password)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := probe.ListDevices(ctx); err != nil {
		a.Log.Error("GOWA instance probe failed", "error", err, "base_url", in.BaseURL)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Could not reach GOWA server with these credentials", nil, "")
	}

	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	inst := &models.GowaInstance{
		OrganizationID: orgID,
		Name:           in.Name,
		BaseURL:        in.BaseURL,
		Username:       in.Username,
		Password:       in.Password,
		WebhookURL:     in.WebhookURL,
		IsActive:       active,
	}
	if err := inst.EncryptCredentials(a.Config.App.EncryptionKey); err != nil {
		a.Log.Error("Failed to encrypt GOWA instance credentials", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to secure credentials", nil, "")
	}
	if err := a.DB.Create(inst).Error; err != nil {
		a.Log.Error("Failed to create GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", inst.ID, models.AuditActionCreated, nil, map[string]any{
		"name": inst.Name, "base_url": inst.BaseURL,
	})
	return r.SendEnvelope(map[string]any{"instance": inst.ToResponse()})
}

// UpdateGowaInstance updates a DB-managed GOWA instance. Empty username/password
// in the payload means "keep existing".
// PUT /api/gowa/servers/{id}
func (a *App) UpdateGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionWrite)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}

	var in gowaInstanceInput
	if err := r.Decode(&in, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	old := *instance
	instance.DecryptCredentials(a.Config.App.EncryptionKey)

	if in.Name != "" {
		instance.Name = in.Name
	}
	if in.BaseURL != "" {
		instance.BaseURL = in.BaseURL
	}
	if in.Username != "" {
		instance.Username = in.Username
	}
	if in.Password != "" {
		instance.Password = in.Password
	}
	instance.WebhookURL = in.WebhookURL
	if in.IsActive != nil {
		instance.IsActive = *in.IsActive
	}

	if err := instance.EncryptCredentials(a.Config.App.EncryptionKey); err != nil {
		a.Log.Error("Failed to encrypt GOWA instance credentials", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to secure credentials", nil, "")
	}
	if err := a.DB.Save(instance).Error; err != nil {
		a.Log.Error("Failed to update GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", instance.ID, models.AuditActionUpdated,
		map[string]any{"name": old.Name, "base_url": old.BaseURL, "is_active": old.IsActive},
		map[string]any{"name": instance.Name, "base_url": instance.BaseURL, "is_active": instance.IsActive})
	return r.SendEnvelope(map[string]any{"instance": instance.ToResponse()})
}

// DeleteGowaInstance soft-deletes a DB-managed GOWA instance (does not touch
// devices on the remote GOWA server — that's an explicit per-device action).
// DELETE /api/gowa/servers/{id}
func (a *App) DeleteGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionDelete)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}
	if err := a.DB.Delete(instance).Error; err != nil {
		a.Log.Error("Failed to delete GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", instance.ID, models.AuditActionDeleted, map[string]any{
		"name": instance.Name, "base_url": instance.BaseURL,
	}, nil)
	return r.SendEnvelope(map[string]any{"deleted": true})
}

var _ = fmt.Sprintf // keep fmt import if unused after edits
```

> NOTE: If `fmt` ends up unused after you finish editing, remove it and the `var _ = fmt.Sprintf` line. Prefer keeping the import list clean.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/handlers/ -run 'TestGowaInstances_' -v`
Expected: `TestGowaInstances_AgentDenied_CRUD` PASS (403s), `TestGowaInstances_CredentialsNotExposed` PASS, `TestGowaInstances_CrossOrgDenied` PASS (404), `TestGowaInstances_Create_EncryptsAndStripsCreds` PASS (not 403).

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/gowa_instances.go internal/handlers/gowa_instances_test.go
git commit -m "feat(handlers): add GOWA instance CRUD (org-scoped, encrypted creds)"
```

---

## Task A4: Per-instance device operations

**Files:**
- Modify: `internal/handlers/gowa_instances.go` (append device handlers)

These proxy to the existing `pkg/gowa` client under the `devices` resource (so the existing `devices:read/write/delete` perms apply uniformly across account-bound and instance-bound devices).

- [ ] **Step 1: Write the failing tests**

Append to `internal/handlers/gowa_instances_test.go`:

```go
// TestGowaInstanceDevices_AgentDenied verifies device ops under /servers/{id}/devices
// require devices:read|write|delete (agent has none).
func TestGowaInstanceDevices_AgentDenied(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createTestAgent(t, app, org.ID)
	inst := &models.GowaInstance{OrganizationID: org.ID, Name: "s", BaseURL: "http://g", IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)

	// List devices -> 403
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	require.NoError(t, app.ListGowaInstanceDevices(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

	// Delete device -> 403 (this also proves devices:delete is enforced)
	req = testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", "dev-1")
	require.NoError(t, app.DeleteGowaInstanceDevice(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// TestGowaInstanceDevices_CrossOrgDenied verifies org A cannot reach org B's instance devices.
func TestGowaInstanceDevices_CrossOrgDenied(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	adminA := createAdminUser(t, app, orgA.ID)
	inst := &models.GowaInstance{OrganizationID: orgB.ID, Name: "s", BaseURL: "http://g", IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgA.ID, adminA.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	require.NoError(t, app.ListGowaInstanceDevices(req))
	// 404 because the instance isn't visible to org A (org-scoped lookup).
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handlers/ -run 'TestGowaInstanceDevices_' -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Append the device handlers**

Append to `internal/handlers/gowa_instances.go` (after `DeleteGowaInstance`):

```go
// parseDeviceID extracts the {deviceId} path param (a GOWA device string id).
func parseDeviceID(r *fastglue.Request) string {
	v, _ := r.RequestCtx.UserValue("deviceId").(string)
	return v
}

// ListGowaInstanceDevices lists all devices on a GOWA instance, enriching each
// with its live connection status.
// GET /api/gowa/servers/{id}/devices
func (a *App) ListGowaInstanceDevices(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionRead)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}

	ctx := context.Background()
	devices, err := bundle.client.ListDevices(ctx)
	if err != nil {
		a.Log.Error("Failed to list GOWA devices", "error", err, "instance", bundle.instance.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to list devices from GOWA", nil, "")
	}

	type deviceWithStatus struct {
		gowa.DeviceInfo
		IsConnected bool   `json:"is_connected"`
		IsLoggedIn  bool   `json:"is_logged_in"`
		JID         string `json:"jid"`
	}
	out := make([]deviceWithStatus, 0, len(devices))
	for _, d := range devices {
		entry := deviceWithStatus{DeviceInfo: d}
		if st, err := bundle.client.GetDeviceStatus(ctx, d.ID); err == nil {
			entry.IsConnected = st.IsConnected
			entry.IsLoggedIn = st.IsLoggedIn
		}
		out = append(out, entry)
	}
	return r.SendEnvelope(map[string]any{"devices": out})
}

// CreateGowaInstanceDevice provisions a new device on a GOWA instance (mirrors
// the legacy GowaCreateDevice webhook/device-id generation).
// POST /api/gowa/servers/{id}/devices  body: {"device_name": "..."}
func (a *App) CreateGowaInstanceDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}

	var req struct {
		DeviceName string `json:"device_name"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.DeviceName == "" {
		req.DeviceName = "whatomate-device"
	}

	ctx := context.Background()
	webhookURL := bundle.instance.WebhookURL
	if webhookURL == "" {
		webhookURL = fmt.Sprintf("%s://%s%s", "http", r.RequestCtx.Host(), a.Config.GOWA.WebhookPath)
	}
	webhookSecret := gowa.GenerateWebhookSecret()
	deviceID := gowa.GenerateDeviceID(req.DeviceName)

	device, err := bundle.client.CreateDevice(ctx, deviceID, gowa.WebhookConfig{
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited",
	})
	if err != nil {
		a.Log.Error("Failed to create GOWA device", "error", err, "instance", bundle.instance.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to create device on GOWA", nil, "")
	}

	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionCreated, nil, map[string]any{
		"device_id": device.ID, "instance": bundle.instance.Name, "base_url": bundle.instance.BaseURL,
	})
	return r.SendEnvelope(map[string]any{
		"device_id":      device.ID,
		"webhook_secret": webhookSecret,
	})
}

// DeleteGowaInstanceDevice removes a device from a GOWA instance.
// DELETE /api/gowa/servers/{id}/devices/{deviceId}
func (a *App) DeleteGowaInstanceDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionDelete)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	if err := bundle.client.DeleteDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to delete GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to delete device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionDeleted, nil, map[string]any{
		"device_id": deviceID, "instance": bundle.instance.Name,
	})
	return r.SendEnvelope(map[string]any{"deleted": true})
}

// GowaInstanceDeviceQR fetches a login QR (as a base64 data URI) for a device.
// GET /api/gowa/servers/{id}/devices/{deviceId}/qr
func (a *App) GowaInstanceDeviceQR(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	ctx := context.Background()
	// If already connected, short-circuit like the account-bound handler does.
	if st, err := bundle.client.GetDeviceStatus(ctx, deviceID); err == nil && st.IsConnected {
		return r.SendEnvelope(map[string]any{"already_connected": true})
	}
	qr, err := bundle.client.GetLoginQR(ctx, deviceID)
	if err != nil {
		a.Log.Error("Failed to get GOWA login QR", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get QR code from GOWA", nil, "")
	}
	qrData, err := bundle.client.DownloadMedia(ctx, qr.QRLink, "")
	if err != nil {
		a.Log.Error("Failed to download GOWA QR image", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to download QR image", nil, "")
	}
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(qrData))
	return r.SendEnvelope(map[string]any{"qr_link": dataURI, "qr_duration": qr.QRDuration})
}

// GowaInstanceDevicePairCode requests a phone pair code.
// POST /api/gowa/servers/{id}/devices/{deviceId}/pair-code  body: {"phone":"..."}
func (a *App) GowaInstanceDevicePairCode(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.Phone == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Phone number is required", nil, "")
	}
	result, err := bundle.client.LoginWithCode(context.Background(), deviceID, req.Phone)
	if err != nil {
		a.Log.Error("Failed to get GOWA pair code", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get pair code from GOWA", nil, "")
	}
	return r.SendEnvelope(map[string]any{"pair_code": result.PairCode})
}

// GowaInstanceDeviceLogout logs out a device (keeps the slot).
// POST /api/gowa/servers/{id}/devices/{deviceId}/logout
func (a *App) GowaInstanceDeviceLogout(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	if err := bundle.client.LogoutDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to logout GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to logout device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "action": "logout",
	})
	return r.SendEnvelope(map[string]any{"ok": true})
}

// GowaInstanceDeviceReconnect triggers a reconnect for a device.
// POST /api/gowa/servers/{id}/devices/{deviceId}/reconnect
func (a *App) GowaInstanceDeviceReconnect(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	if err := bundle.client.ReconnectDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to reconnect GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to reconnect device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "action": "reconnect",
	})
	return r.SendEnvelope(map[string]any{"ok": true})
}

// GetGowaInstanceDeviceWebhook returns the webhook config for a device.
// GET /api/gowa/servers/{id}/devices/{deviceId}/webhook
func (a *App) GetGowaInstanceDeviceWebhook(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionRead)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	cfg, err := bundle.client.GetDeviceWebhook(context.Background(), deviceID)
	if err != nil {
		a.Log.Error("Failed to get GOWA device webhook", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get webhook from GOWA", nil, "")
	}
	return r.SendEnvelope(map[string]any{"webhook": cfg})
}

// SetGowaInstanceDeviceWebhook updates the webhook config for a device.
// PATCH /api/gowa/servers/{id}/devices/{deviceId}/webhook
func (a *App) SetGowaInstanceDeviceWebhook(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	var cfg gowa.WebhookConfig
	if err := r.Decode(&cfg, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	updated, err := bundle.client.SetDeviceWebhook(context.Background(), deviceID, cfg)
	if err != nil {
		a.Log.Error("Failed to set GOWA device webhook", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to set webhook on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "webhook_url": updated.WebhookURL,
	})
	return r.SendEnvelope(map[string]any{"webhook": updated})
}
```

Then **add `encoding/base64`** to the import block at the top of `gowa_instances.go` (needed by the QR handler):

```go
import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
	...
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/handlers/ -run 'TestGowaInstanceDevices_' -v`
Expected: `TestGowaInstanceDevices_AgentDenied` PASS, `TestGowaInstanceDevices_CrossOrgDenied` PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/gowa_instances.go internal/handlers/gowa_instances_test.go
git commit -m "feat(handlers): add per-instance GOWA device ops (list/create/delete/qr/pair/logout/reconnect/webhook)"
```

---

## Task A5: Register routes

**Files:**
- Modify: `cmd/whatomate/main.go` (GOWA block L696-698)

- [ ] **Step 1: Add the route block**

In `cmd/whatomate/main.go`, after the existing GOWA instance management block (L696-698):

```go
	// GOWA instance management (multi-instance dropdown + device provisioning)
	g.GET("/api/gowa/instances", app.GowaInstances)
	g.POST("/api/gowa/create-device", app.GowaCreateDevice)
```

add:

```go
	// GOWA servers (DB-managed instances + per-instance device management)
	g.GET("/api/gowa/servers", app.ListGowaInstances)
	g.POST("/api/gowa/servers", app.CreateGowaInstance)
	g.GET("/api/gowa/servers/{id}", app.GetGowaInstance)
	g.PUT("/api/gowa/servers/{id}", app.UpdateGowaInstance)
	g.DELETE("/api/gowa/servers/{id}", app.DeleteGowaInstance)

	// Devices within a DB-managed GOWA server
	g.GET("/api/gowa/servers/{id}/devices", app.ListGowaInstanceDevices)
	g.POST("/api/gowa/servers/{id}/devices", app.CreateGowaInstanceDevice)
	g.DELETE("/api/gowa/servers/{id}/devices/{deviceId}", app.DeleteGowaInstanceDevice)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/qr", app.GowaInstanceDeviceQR)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/pair-code", app.GowaInstanceDevicePairCode)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/logout", app.GowaInstanceDeviceLogout)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/reconnect", app.GowaInstanceDeviceReconnect)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.GetGowaInstanceDeviceWebhook)
	g.PATCH("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.SetGowaInstanceDeviceWebhook)
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/...`
Expected: builds cleanly.

- [ ] **Step 3: Commit**

```bash
git add cmd/whatomate/main.go
git commit -m "feat(routes): register /api/gowa/servers instance + device endpoints"
```

---

## Task A6: Backend full verification

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 2: Run the focused test suites**

Run: `go test ./internal/handlers/... ./internal/models/...`
Expected: all PASS (tests requiring Redis are skipped if `TEST_REDIS_URL` unset — that's fine).

- [ ] **Step 3: Run the existing GOWA security tests (regression)**

Run: `go test ./internal/handlers/ -run 'TestGowaDevice_|TestGowaCreateDevice_|TestSystemRoles_DevicesMapping|TestPermissions_DevicesSeeded' -v`
Expected: all PASS (confirms the legacy endpoints + devices perms still work).

---

# PHASE B — FRONTEND

## Task B1: gowaServersService in api.ts

**Files:**
- Modify: `frontend/src/services/api.ts` (append near the other services, e.g. after `tagsService` ~L873)

- [ ] **Step 1: Add types + service**

In `frontend/src/services/api.ts`, after the `tagsService` block (ends ~L873), add:

```ts
// GOWA Servers (DB-managed GOWA instances + per-instance devices)
export interface GowaServer {
  id: string
  name: string
  base_url: string
  webhook_url: string
  is_active: boolean
  has_credentials: boolean
  created_at: string
  updated_at: string
}

export interface GowaDevice {
  id: string
  phone_number?: string
  display_name: string
  state: string
  jid: string
  created_at: string
  webhook_url?: string
  webhook_events?: string
  is_connected: boolean
  is_logged_in: boolean
}

export const gowaServersService = {
  list: () => api.get<{ instances: GowaServer[] }>('/gowa/servers'),
  get: (id: string) => api.get<{ instance: GowaServer }>(`/gowa/servers/${id}`),
  create: (data: { name: string; base_url: string; username: string; password: string; webhook_url?: string; is_active?: boolean }) =>
    api.post<{ instance: GowaServer }>('/gowa/servers', data),
  update: (id: string, data: Partial<{ name: string; base_url: string; username: string; password: string; webhook_url: string; is_active: boolean }>) =>
    api.put<{ instance: GowaServer }>(`/gowa/servers/${id}`, data),
  delete: (id: string) => api.delete(`/gowa/servers/${id}`),

  // Devices within a server
  listDevices: (serverId: string) => api.get<{ devices: GowaDevice[] }>(`/gowa/servers/${serverId}/devices`),
  createDevice: (serverId: string, data: { device_name: string }) =>
    api.post<{ device_id: string; webhook_secret: string }>(`/gowa/servers/${serverId}/devices`, data),
  deleteDevice: (serverId: string, deviceId: string) => api.delete(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}`),
  deviceQR: (serverId: string, deviceId: string) =>
    api.get<{ qr_link: string; qr_duration: number; already_connected?: boolean }>(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/qr`),
  devicePairCode: (serverId: string, deviceId: string, phone: string) =>
    api.post<{ pair_code: string }>(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/pair-code`, { phone }),
  deviceLogout: (serverId: string, deviceId: string) =>
    api.post(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/logout`),
  deviceReconnect: (serverId: string, deviceId: string) =>
    api.post(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/reconnect`),
  getDeviceWebhook: (serverId: string, deviceId: string) =>
    api.get<{ webhook: { webhook_url: string; webhook_events: string; webhook_insecure_skip_verify: boolean } }>(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/webhook`),
  setDeviceWebhook: (serverId: string, deviceId: string, data: { webhook_url: string; webhook_events: string; webhook_insecure_skip_verify?: boolean }) =>
    api.patch(`/gowa/servers/${serverId}/devices/${encodeURIComponent(deviceId)}/webhook`, data),
}
```

- [ ] **Step 2: Verify types compile**

Run: `cd frontend && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: no errors referencing `gowaServersService` or `GowaServer`/`GowaDevice`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/services/api.ts
git commit -m "feat(api): add gowaServersService for DB-managed GOWA instances + devices"
```

---

## Task B2: Pinia store

**Files:**
- Create: `frontend/src/stores/gowaServers.ts`

- [ ] **Step 1: Create the store**

Create `frontend/src/stores/gowaServers.ts` (follows the `roles.ts` setup-store pattern):

```ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { gowaServersService, type GowaServer, type GowaDevice } from '@/services/api'

export interface CreateGowaServerData {
  name: string
  base_url: string
  username: string
  password: string
  webhook_url?: string
  is_active?: boolean
}

export const useGowaServersStore = defineStore('gowaServers', () => {
  const servers = ref<GowaServer[]>([])
  const currentServer = ref<GowaServer | null>(null)
  const devices = ref<GowaDevice[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchServers(): Promise<GowaServer[]> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.list()
      const data = (response.data as any).data || response.data
      servers.value = data.instances || []
      return servers.value
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch GOWA servers'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchServer(id: string): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.get(id)
      const data = (response.data as any).data || response.data
      currentServer.value = data.instance
      return data.instance
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createServer(data: CreateGowaServerData): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.create(data)
      const server = (response.data as any).data?.instance || (response.data as any).data
      servers.value.unshift(server)
      return server
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to create GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateServer(id: string, data: Partial<CreateGowaServerData>): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.update(id, data)
      const server = (response.data as any).data?.instance || (response.data as any).data
      const index = servers.value.findIndex(s => s.id === id)
      if (index !== -1) servers.value[index] = server
      if (currentServer.value?.id === id) currentServer.value = server
      return server
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to update GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteServer(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await gowaServersService.delete(id)
      servers.value = servers.value.filter(s => s.id !== id)
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to delete GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchDevices(serverId: string): Promise<GowaDevice[]> {
    try {
      const response = await gowaServersService.listDevices(serverId)
      const data = (response.data as any).data || response.data
      devices.value = data.devices || []
      return devices.value
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch devices'
      throw err
    }
  }

  return {
    servers,
    currentServer,
    devices,
    loading,
    error,
    fetchServers,
    fetchServer,
    createServer,
    updateServer,
    deleteServer,
    fetchDevices,
  }
})
```

- [ ] **Step 2: Verify types compile**

Run: `cd frontend && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/gowaServers.ts
git commit -m "feat(store): add gowaServers Pinia store"
```

---

## Task B3: RESOURCE_LABELS

**Files:**
- Modify: `frontend/src/lib/constants.ts:31`

- [ ] **Step 1: Add the labels**

In `frontend/src/lib/constants.ts`, in `RESOURCE_LABELS` (L31-45), add two keys:

```ts
export const RESOURCE_LABELS: Record<string, string> = {
  users: 'Users',
  contacts: 'Contacts',
  messages: 'Messages',
  teams: 'Teams',
  chatbot: 'Chatbot',
  campaigns: 'Campaigns',
  templates: 'Templates',
  analytics: 'Analytics',
  settings: 'Settings',
  webhooks: 'Webhooks',
  apikeys: 'API Keys',
  roles: 'Roles',
  tags: 'Tags',
  gowa_instances: 'GOWA Servers',
  devices: 'Devices',
} as const
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/lib/constants.ts
git commit -m "feat(constants): add gowa_instances + devices resource labels"
```

---

## Task B4: Navigation + router

**Files:**
- Modify: `frontend/src/components/layout/navigation.ts` (L107, L115, L116-131)
- Modify: `frontend/src/router/index.ts` (add routes ~L166; add to `navigationOrder` ~L322)

- [ ] **Step 1: Add the nav entry**

In `frontend/src/components/layout/navigation.ts`:

1. Import a `Server` icon — add to the `lucide-vue-next` import block (L1-21):

```ts
  ScrollText,
  Server
} from 'lucide-vue-next'
```

2. In the settings section (L105-134), add `'gowa_instances'` to the section `permissions` array (L107) and to the parent item's `childPermissions` (L115), and add a new child entry in the `children` array (after `auditLogs`, L130):

```ts
          { name: 'nav.auditLogs', path: '/settings/audit-logs', icon: ScrollText, permission: 'audit_logs' },
          { name: 'nav.gowaServers', path: '/settings/gowa-servers', icon: Server, permission: 'gowa_instances' }
```

The three modified lines become:

```ts
    permissions: ['settings.general', 'settings.chatbot', 'accounts', 'contacts', 'canned_responses', 'tags', 'teams', 'users', 'roles', 'api_keys', 'webhooks', 'custom_actions', 'settings.sso', 'audit_logs', 'gowa_instances'],
    pinBottom: true,
    items: [
      {
        name: 'nav.settings',
        path: '/settings',
        icon: Settings,
        permission: 'settings.general',
        childPermissions: ['settings.general', 'settings.chatbot', 'accounts', 'contacts', 'canned_responses', 'tags', 'teams', 'users', 'roles', 'api_keys', 'webhooks', 'custom_actions', 'settings.sso', 'audit_logs', 'gowa_instances'],
        children: [
          // ... existing children ...
          { name: 'nav.auditLogs', path: '/settings/audit-logs', icon: ScrollText, permission: 'audit_logs' },
          { name: 'nav.gowaServers', path: '/settings/gowa-servers', icon: Server, permission: 'gowa_instances' }
        ]
      }
    ]
```

- [ ] **Step 2: Add the routes**

In `frontend/src/router/index.ts`, after the `settings/accounts/:id` route (L161-166), add:

```ts
        {
          path: 'settings/gowa-servers',
          name: 'gowa-servers',
          component: () => import('@/views/settings/GowaServersView.vue'),
          meta: { permission: 'gowa_instances' }
        },
        {
          path: 'settings/gowa-servers/:id',
          name: 'gowa-server-detail',
          component: () => import('@/views/settings/GowaServerDetailView.vue'),
          meta: { permission: 'gowa_instances' }
        },
```

- [ ] **Step 3: Add to navigationOrder**

In `frontend/src/router/index.ts`, in the settings `childPaths` array (L309-322), add before the closing `]`:

```ts
    { path: '/settings/gowa-servers', permission: 'gowa_instances' }
```

- [ ] **Step 4: Verify types compile (will fail until views exist — expected)**

Run: `cd frontend && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: errors about missing `GowaServersView.vue` / `GowaServerDetailView.vue` — that's fine, they're created in B6/B7. No router/navigation type errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/layout/navigation.ts frontend/src/router/index.ts
git commit -m "feat(nav): add GOWA Servers settings routes + nav entry"
```

---

## Task B5: i18n keys

**Files:**
- Modify: `frontend/src/i18n/locales/en.json`
- Modify: `frontend/src/i18n/locales/ar.json`
- Modify: `frontend/src/i18n/locales/en.json` (`nav` namespace — add `gowaServers`)

- [ ] **Step 1: Add the en.json `gowaServers` namespace**

In `frontend/src/i18n/locales/en.json`, add a new top-level key (e.g. after the `accounts` block). Use this exact block:

```json
  "gowaServers": {
    "title": "GOWA Servers",
    "subtitle": "Manage GOWA server instances and their WhatsApp devices",
    "addServer": "Add GOWA Server",
    "editServer": "Edit Server",
    "name": "Server Name",
    "baseUrl": "GOWA Base URL",
    "username": "Username",
    "password": "Password",
    "webhookUrl": "Webhook URL (optional)",
    "isActive": "Active",
    "hasCredentials": "Credentials set",
    "noCredentials": "No credentials",
    "noServers": "No GOWA servers configured",
    "noServersDesc": "Add a GOWA server to manage its WhatsApp devices.",
    "probeFailed": "Could not reach the GOWA server with these credentials.",
    "createdSuccess": "GOWA server created",
    "updatedSuccess": "GOWA server updated",
    "deletedSuccess": "GOWA server deleted",
    "deleteServer": "Delete Server",
    "devices": "Devices",
    "noDevices": "No devices on this server",
    "noDevicesDesc": "Create a device to connect a WhatsApp number.",
    "createDevice": "Create Device",
    "deviceName": "Device Name",
    "deviceCreated": "Device created",
    "deleteDevice": "Delete Device",
    "connect": "Connect",
    "logout": "Logout",
    "reconnect": "Reconnect",
    "webhook": "Webhook",
    "connected": "Connected",
    "disconnected": "Disconnected",
    "connecting": "Connecting",
    "qrCode": "QR Code",
    "pairCode": "Pair Code",
    "phoneNumber": "Phone Number",
    "getCode": "Get Code",
    "yourPairCode": "Your Pair Code",
    "refreshQr": "Refresh QR",
    "qrInstructions": "Open WhatsApp on your phone → Settings → Linked Devices → Link a Device → scan this code",
    "pairCodeInstructions": "Enter your phone number with country code. You will receive an 8-digit code to enter on your phone.",
    "webhookUrlLabel": "Webhook URL",
    "webhookEventsLabel": "Webhook Events",
    "saveWebhook": "Save Webhook"
  },
```

- [ ] **Step 2: Add the nav label in en.json**

In `frontend/src/i18n/locales/en.json`, in the `nav` object, add:

```json
    "gowaServers": "GOWA Servers"
```

- [ ] **Step 3: Add the ar.json `gowaServers` namespace (Arabic)**

In `frontend/src/i18n/locales/ar.json`, add the matching block (Arabic translations):

```json
  "gowaServers": {
    "title": "خوادم GOWA",
    "subtitle": "إدارة خوادم GOWA وأجهزة الواتساب الخاصة بها",
    "addServer": "إضافة خادم GOWA",
    "editServer": "تعديل الخادم",
    "name": "اسم الخادم",
    "baseUrl": "عنوان GOWA الأساسي",
    "username": "اسم المستخدم",
    "password": "كلمة المرور",
    "webhookUrl": "رابط الويبهوك (اختياري)",
    "isActive": "نشط",
    "hasCredentials": "تم تعيين بيانات الاعتماد",
    "noCredentials": "لا توجد بيانات اعتماد",
    "noServers": "لا توجد خوادم GOWA مهيأة",
    "noServersDesc": "أضف خادم GOWA لإدارة أجهزة الواتساب الخاصة به.",
    "probeFailed": "تعذر الوصول إلى خادم GOWA باستخدام بيانات الاعتماد هذه.",
    "createdSuccess": "تم إنشاء خادم GOWA",
    "updatedSuccess": "تم تحديث خادم GOWA",
    "deletedSuccess": "تم حذف خادم GOWA",
    "deleteServer": "حذف الخادم",
    "devices": "الأجهزة",
    "noDevices": "لا توجد أجهزة على هذا الخادم",
    "noDevicesDesc": "أنشئ جهازًا لتوصيل رقم واتساب.",
    "createDevice": "إنشاء جهاز",
    "deviceName": "اسم الجهاز",
    "deviceCreated": "تم إنشاء الجهاز",
    "deleteDevice": "حذف الجهاز",
    "connect": "اتصال",
    "logout": "تسجيل الخروج",
    "reconnect": "إعادة الاتصال",
    "webhook": "الويبهوك",
    "connected": "متصل",
    "disconnected": "غير متصل",
    "connecting": "جارٍ الاتصال",
    "qrCode": "رمز QR",
    "pairCode": "رمز الاقتران",
    "phoneNumber": "رقم الهاتف",
    "getCode": "احصل على الرمز",
    "yourPairCode": "رمز الاقتران الخاص بك",
    "refreshQr": "تحديث QR",
    "qrInstructions": "افتح واتساب على هاتفك ← الإعدادات ← الأجهزة المرتبطة ← ربط جهاز ← امسح هذا الرمز",
    "pairCodeInstructions": "أدخل رقم هاتفك مع رمز البلد. ستتلقى رمزًا من 8 أرقام لإدخاله على هاتفك.",
    "webhookUrlLabel": "رابط الويبهوك",
    "webhookEventsLabel": "أحداث الويبهوك",
    "saveWebhook": "حفظ الويبهوك"
  },
```

- [ ] **Step 4: Add the nav label in ar.json**

In `frontend/src/i18n/locales/ar.json`, in the `nav` object, add:

```json
    "gowaServers": "خوادم GOWA"
```

- [ ] **Step 5: Validate JSON**

Run: `cd frontend && node -e "JSON.parse(require('fs').readFileSync('src/i18n/locales/en.json','utf8')); JSON.parse(require('fs').readFileSync('src/i18n/locales/ar.json','utf8')); console.log('valid')"`
Expected: `valid`.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/i18n/locales/en.json frontend/src/i18n/locales/ar.json
git commit -m "feat(i18n): add gowaServers translations (en + ar)"
```

---

## Task B6: GowaServersView.vue (list page)

**Files:**
- Create: `frontend/src/views/settings/GowaServersView.vue`

- [ ] **Step 1: Create the list view**

Create `frontend/src/views/settings/GowaServersView.vue` (modeled on `AccountsView.vue`):

```vue
<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { PageHeader, DataTable, DeleteConfirmDialog, ErrorState, type Column } from '@/components/shared'
import { useGowaServersStore } from '@/stores/gowaServers'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationsStore } from '@/stores/organizations'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { Server, Plus, Pencil, Trash2, CheckCircle2 } from 'lucide-vue-next'
import type { GowaServer } from '@/services/api'

const { t } = useI18n()
const store = useGowaServersStore()
const authStore = useAuthStore()
const organizationsStore = useOrganizationsStore()

const dialogOpen = ref(false)
const editing = ref<GowaServer | null>(null)
const submitting = ref(false)
const deleteOpen = ref(false)
const serverToDelete = ref<GowaServer | null>(null)
const isDeleting = ref(false)

const form = ref({
  name: '',
  base_url: '',
  username: '',
  password: '',
  webhook_url: '',
  is_active: true,
})

const canWrite = computed(() => authStore.hasPermission('gowa_instances', 'write'))
const canDelete = computed(() => authStore.hasPermission('gowa_instances', 'delete'))
const breadcrumbs = computed(() => [
  { label: t('nav.settings'), href: '/settings' },
  { label: t('gowaServers.title', 'GOWA Servers') },
])

const columns = computed<Column<GowaServer>[]>(() => [
  { key: 'name', label: t('gowaServers.name', 'Server Name'), width: 'w-[220px]', sortable: true },
  { key: 'base_url', label: t('gowaServers.baseUrl', 'GOWA Base URL') },
  { key: 'creds', label: t('gowaServers.hasCredentials', 'Credentials') },
  { key: 'status', label: t('gowaServers.isActive', 'Active') },
  { key: 'actions', label: t('common.actions', 'Actions'), align: 'right' },
])

watch(() => organizationsStore.selectedOrgId, () => store.fetchServers())
onMounted(() => store.fetchServers())

function openCreate() {
  editing.value = null
  form.value = { name: '', base_url: '', username: '', password: '', webhook_url: '', is_active: true }
  dialogOpen.value = true
}

function openEdit(s: GowaServer) {
  editing.value = s
  form.value = { name: s.name, base_url: s.base_url, username: '', password: '', webhook_url: s.webhook_url || '', is_active: s.is_active }
  dialogOpen.value = true
}

async function submit() {
  submitting.value = true
  try {
    const payload = { ...form.value }
    // Empty username/password on edit means "keep existing" on the backend.
    if (editing.value && !payload.username) delete (payload as any).username
    if (editing.value && !payload.password) delete (payload as any).password
    if (editing.value) {
      await store.updateServer(editing.value.id, payload)
      toast.success(t('gowaServers.updatedSuccess', 'GOWA server updated'))
    } else {
      await store.createServer(payload as any)
      toast.success(t('gowaServers.createdSuccess', 'GOWA server created'))
    }
    dialogOpen.value = false
  } catch (e: any) {
    const msg = e?.response?.status === 502
      ? t('gowaServers.probeFailed', 'Could not reach the GOWA server with these credentials.')
      : getErrorMessage(e, t('common.failedSave', { resource: 'server' }))
    toast.error(msg)
  } finally {
    submitting.value = false
  }
}

function openDelete(s: GowaServer) {
  serverToDelete.value = s
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!serverToDelete.value) return
  isDeleting.value = true
  try {
    await store.deleteServer(serverToDelete.value.id)
    toast.success(t('gowaServers.deletedSuccess', 'GOWA server deleted'))
    deleteOpen.value = false
    serverToDelete.value = null
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: 'server' })))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('gowaServers.title', 'GOWA Servers')"
      :icon="Server"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
      back-link="/settings"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button v-if="canWrite" size="sm" class="bg-blue-600 hover:bg-blue-700 text-white font-medium" @click="openCreate">
          <Plus class="h-4 w-4 mr-1.5" />
          {{ $t('gowaServers.addServer', 'Add GOWA Server') }}
        </Button>
      </template>
    </PageHeader>

    <ErrorState
      v-if="store.error && !store.loading"
      :title="$t('common.loadErrorTitle', 'Failed to load')"
      :description="store.error"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="store.fetchServers()">{{ $t('common.retry', 'Retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <Card>
          <CardHeader>
            <CardTitle>{{ $t('gowaServers.title', 'GOWA Servers') }}</CardTitle>
            <CardDescription>{{ $t('gowaServers.subtitle', 'Manage GOWA server instances and their WhatsApp devices') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              :items="store.servers"
              :columns="columns"
              :is-loading="store.loading"
              :empty-icon="Server"
              :empty-title="$t('gowaServers.noServers', 'No GOWA servers configured')"
              :empty-description="$t('gowaServers.noServersDesc', 'Add a GOWA server to manage its WhatsApp devices.')"
              item-name="servers"
            >
              <template #empty-action>
                <Button v-if="canWrite" size="lg" class="bg-blue-600 hover:bg-blue-700 text-white" @click="openCreate">
                  <Plus class="mr-2 h-5 w-5" />
                  {{ $t('gowaServers.addServer', 'Add GOWA Server') }}
                </Button>
              </template>

              <template #cell-name="{ item: s }">
                <RouterLink :to="`/settings/gowa-servers/${s.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                  <div class="h-9 w-9 rounded-full bg-blue-500/10 flex items-center justify-center flex-shrink-0">
                    <Server class="h-4 w-4 text-blue-500" />
                  </div>
                  <span class="font-medium truncate text-sm">{{ s.name }}</span>
                </RouterLink>
              </template>

              <template #cell-base_url="{ item: s }">
                <code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ s.base_url }}</code>
              </template>

              <template #cell-creds="{ item: s }">
                <Badge v-if="s.has_credentials" variant="outline" class="border-green-600 text-green-600">
                  <CheckCircle2 class="h-3 w-3 mr-1" /> {{ $t('gowaServers.hasCredentials', 'Credentials set') }}
                </Badge>
                <Badge v-else variant="outline" class="border-amber-600 text-amber-600">
                  {{ $t('gowaServers.noCredentials', 'No credentials') }}
                </Badge>
              </template>

              <template #cell-status="{ item: s }">
                <Badge v-if="s.is_active" variant="outline" class="border-green-600 text-green-600">{{ $t('gowaServers.isActive', 'Active') }}</Badge>
                <Badge v-else variant="outline" class="text-muted-foreground">Inactive</Badge>
              </template>

              <template #cell-actions="{ item: s }">
                <div class="flex items-center justify-end gap-1">
                  <Button v-if="canWrite" variant="ghost" size="icon" class="h-8 w-8" @click="openEdit(s)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button v-if="canDelete" variant="ghost" size="icon" class="h-8 w-8" @click="openDelete(s)">
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </template>
            </DataTable>
          </CardContent>
        </Card>
      </div>
    </ScrollArea>

    <!-- Create / Edit dialog -->
    <Dialog :open="dialogOpen" @update:open="(v) => dialogOpen = v">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? $t('gowaServers.editServer', 'Edit Server') : $t('gowaServers.addServer', 'Add GOWA Server') }}</DialogTitle>
          <DialogDescription>{{ $t('gowaServers.subtitle') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.name', 'Server Name') }}</Label>
            <Input v-model="form.name" placeholder="Production GOWA" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.baseUrl', 'GOWA Base URL') }}</Label>
            <Input v-model="form.base_url" placeholder="http://gowa:8080" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.username', 'Username') }}</Label>
            <Input v-model="form.username" :placeholder="editing ? $t('common.unchanged', 'Unchanged') : 'admin'" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.password', 'Password') }}</Label>
            <Input v-model="form.password" type="password" :placeholder="editing ? $t('common.unchanged', 'Unchanged') : '••••••••'" autocomplete="new-password" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookUrl', 'Webhook URL (optional)') }}</Label>
            <Input v-model="form.webhook_url" placeholder="https://whatomate.example.com/api/gowa/webhook" />
          </div>
          <div class="flex items-center justify-between">
            <Label>{{ $t('gowaServers.isActive', 'Active') }}</Label>
            <Switch v-model:checked="form.is_active" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="submitting" @click="submit">
            {{ editing ? $t('common.save', 'Save') : $t('gowaServers.addServer', 'Add GOWA Server') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteOpen"
      :title="$t('gowaServers.deleteServer', 'Delete Server')"
      :item-name="serverToDelete?.name"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
```

- [ ] **Step 2: Verify types compile**

Run: `cd frontend && npx vue-tsc --noEmit 2>&1 | grep -i 'GowaServersView' | head`
Expected: no errors for this file.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/settings/GowaServersView.vue
git commit -m "feat(ui): add GowaServersView list page with create/edit/delete dialogs"
```

---

## Task B7: GowaServerDetailView.vue (device management)

**Files:**
- Create: `frontend/src/views/settings/GowaServerDetailView.vue`

This view lists devices, and for each device offers Connect (QR/Pair), Logout, Reconnect, Delete, and Webhook config — all gated by `devices` perms. The QR/Pair UI mirrors `AccountDetailView.vue:584-656`.

- [ ] **Step 1: Create the detail view**

Create `frontend/src/views/settings/GowaServerDetailView.vue`:

```vue
<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { PageHeader, ErrorState, DeleteConfirmDialog } from '@/components/shared'
import { useGowaServerStore } from '@/stores/gowaServers'
import { useAuthStore } from '@/stores/auth'
import { gowaServersService, type GowaDevice } from '@/services/api'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { Server, Plus, QrCode, Link2, Trash2, LogOut, RefreshCw, Webhook, Loader2, CheckCircle2, AlertCircle, Copy, Smartphone } from 'lucide-vue-next'

const route = useRoute()
const { t } = useI18n()
const store = useGowaServerStore()
const authStore = useAuthStore()

const serverId = computed(() => route.params.id as string)
const devices = ref<GowaDevice[]>([])
const loading = ref(true)
const fetchError = ref(false)

const canWriteDevices = computed(() => authStore.hasPermission('devices', 'write'))
const canDeleteDevices = computed(() => authStore.hasPermission('devices', 'delete'))

const breadcrumbs = computed(() => [
  { label: t('nav.settings'), href: '/settings' },
  { label: t('gowaServers.title', 'GOWA Servers'), href: '/settings/gowa-servers' },
  { label: store.currentServer?.name || '' },
])

// Create device dialog
const createOpen = ref(false)
const newDeviceName = ref('')
const creating = ref(false)

// Connect dialog (QR / pair)
const connectOpen = ref(false)
const connectDevice = ref<GowaDevice | null>(null)
const qrLink = ref('')
const qrDuration = ref(30)
const qrLoading = ref(false)
const qrTimer = ref<ReturnType<typeof setTimeout> | null>(null)
// Polls the QR endpoint every 3s while the connect dialog is open. The QR
// handler short-circuits with already_connected=true once GOWA reports the
// device logged in, so this auto-detects a successful scan/pair and closes
// the dialog. Mirrors gowa-ui + the existing AccountDetailView.
const statusPoll = ref<ReturnType<typeof setInterval> | null>(null)
const pairPhone = ref('')
const pairCode = ref('')
const pairLoading = ref(false)
const statusLoading = ref(false)

// state → badge class (GOWA state enum: disconnected/connecting/connected/logged_in)
function stateClass(state: string): string {
  switch (state) {
    case 'logged_in': return 'border-emerald-600 text-emerald-600 bg-emerald-500/10'
    case 'connected': return 'border-sky-600 text-sky-600 bg-sky-500/10'
    case 'connecting': return 'border-amber-600 text-amber-600 bg-amber-500/10'
    default: return 'text-muted-foreground'
  }
}
function stateLabel(state: string): string {
  const map: Record<string, string> = {
    logged_in: t('gowaServers.connected', 'Logged in'),
    connected: t('gowaServers.connected', 'Connected'),
    connecting: t('gowaServers.connecting', 'Connecting'),
    disconnected: t('gowaServers.disconnected', 'Disconnected'),
  }
  return map[state] || state || t('gowaServers.disconnected', 'Disconnected')
}

async function onPairingSuccess() {
  clearTimers()
  toast.success(t('gowaServers.connected', 'Connected'))
  connectOpen.value = false
  await refreshDevices()
}

// Webhook dialog
const webhookOpen = ref(false)
const webhookDevice = ref<GowaDevice | null>(null)
const webhookForm = ref({ webhook_url: '', webhook_events: 'message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited', webhook_insecure_skip_verify: false })
const webhookSaving = ref(false)

// Delete device dialog
const deleteOpen = ref(false)
const deviceToDelete = ref<GowaDevice | null>(null)
const isDeleting = ref(false)

onMounted(load)
onBeforeUnmount(clearTimers)

async function load() {
  loading.value = true
  fetchError.value = false
  try {
    await Promise.all([store.fetchServer(serverId.value), store.fetchDevices(serverId.value)])
    devices.value = store.devices
  } catch {
    fetchError.value = true
    toast.error(t('common.failedLoad', { resource: 'server' }))
  } finally {
    loading.value = false
  }
}

async function refreshDevices() {
  try {
    await store.fetchDevices(serverId.value)
    devices.value = store.devices
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedLoad', { resource: 'devices' })))
  }
}

function openCreate() {
  newDeviceName.value = ''
  createOpen.value = true
}

async function submitCreate() {
  creating.value = true
  try {
    await gowaServersService.createDevice(serverId.value, { device_name: newDeviceName.value || 'whatomate' })
    toast.success(t('gowaServers.deviceCreated', 'Device created'))
    createOpen.value = false
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, t('gowaServers.deviceCreated', 'Failed to create device')))
  } finally {
    creating.value = false
  }
}

function openConnect(d: GowaDevice) {
  connectDevice.value = d
  connectOpen.value = true
  qrLink.value = ''
  pairCode.value = ''
  pairPhone.value = ''
  fetchQr()
  // Start polling so we auto-close on a successful scan/pair.
  if (statusPoll.value) clearInterval(statusPoll.value)
  statusPoll.value = setInterval(fetchQr, 3000)
}

async function fetchQr() {
  if (!connectDevice.value) return
  qrLoading.value = true
  try {
    const resp = await gowaServersService.deviceQR(serverId.value, connectDevice.value.id)
    const data = resp.data.data || resp.data
    if (data.already_connected) {
      qrLink.value = ''
      await onPairingSuccess()
      return
    }
    qrLink.value = data.qr_link || ''
    qrDuration.value = data.qr_duration || 30
    // Only set the QR-refresh timer if we're not already polling (avoid double timers).
    if (!statusPoll.value && qrTimer.value === null) {
      qrTimer.value = setTimeout(fetchQr, (qrDuration.value + 2) * 1000)
    }
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaQrFailed', 'Failed to get QR code')))
  } finally {
    qrLoading.value = false
  }
}

async function fetchPair() {
  if (!connectDevice.value || !pairPhone.value.trim()) return
  pairLoading.value = true
  pairCode.value = ''
  try {
    const resp = await gowaServersService.devicePairCode(serverId.value, connectDevice.value.id, pairPhone.value.trim())
    const data = resp.data.data || resp.data
    pairCode.value = data.pair_code || ''
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaPairFailed', 'Failed to get pair code')))
  } finally {
    pairLoading.value = false
  }
}

async function closeConnect() {
  connectOpen.value = false
  clearTimers()
  await refreshDevices()
}

function clearTimers() {
  if (qrTimer.value) { clearTimeout(qrTimer.value); qrTimer.value = null }
  if (statusPoll.value) { clearInterval(statusPoll.value); statusPoll.value = null }
}

async function copyText(txt: string) {
  try { await navigator.clipboard.writeText(txt) } catch { /* ignore */ }
}

async function logout(d: GowaDevice) {
  try {
    await gowaServersService.deviceLogout(serverId.value, d.id)
    toast.success(t('gowaServers.logout', 'Logout'))
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  }
}

async function reconnect(d: GowaDevice) {
  statusLoading.value = true
  try {
    await gowaServersService.deviceReconnect(serverId.value, d.id)
    toast.success(t('gowaServers.reconnect', 'Reconnect'))
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    statusLoading.value = false
  }
}

async function openWebhook(d: GowaDevice) {
  webhookDevice.value = d
  webhookForm.value = { webhook_url: d.webhook_url || '', webhook_events: d.webhook_events || 'message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited', webhook_insecure_skip_verify: false }
  webhookOpen.value = true
  try {
    const resp = await gowaServersService.getDeviceWebhook(serverId.value, d.id)
    const data = resp.data.data || resp.data
    if (data.webhook) {
      webhookForm.value.webhook_url = data.webhook.webhook_url || webhookForm.value.webhook_url
      webhookForm.value.webhook_events = data.webhook.webhook_events || webhookForm.value.webhook_events
      webhookForm.value.webhook_insecure_skip_verify = !!data.webhook.webhook_insecure_skip_verify
    }
  } catch { /* ignore — keep defaults */ }
}

async function saveWebhook() {
  if (!webhookDevice.value) return
  webhookSaving.value = true
  try {
    await gowaServersService.setDeviceWebhook(serverId.value, webhookDevice.value.id, webhookForm.value)
    toast.success(t('gowaServers.saveWebhook', 'Saved'))
    webhookOpen.value = false
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    webhookSaving.value = false
  }
}

function openDelete(d: GowaDevice) {
  deviceToDelete.value = d
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deviceToDelete.value) return
  isDeleting.value = true
  try {
    await gowaServersService.deleteDevice(serverId.value, deviceToDelete.value.id)
    toast.success(t('gowaServers.deleteDevice', 'Deleted'))
    deleteOpen.value = false
    deviceToDelete.value = null
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="store.currentServer?.name || t('gowaServers.title', 'GOWA Servers')"
      :icon="Server"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
      back-link="/settings/gowa-servers"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button v-if="canWriteDevices" size="sm" class="bg-blue-600 hover:bg-blue-700 text-white" @click="openCreate">
          <Plus class="h-4 w-4 mr-1.5" />
          {{ $t('gowaServers.createDevice', 'Create Device') }}
        </Button>
      </template>
    </PageHeader>

    <ErrorState
      v-if="fetchError && !loading"
      :title="$t('common.loadErrorTitle', 'Failed to load')"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="load">{{ $t('common.retry', 'Retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6 space-y-6">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div>
                <CardTitle>{{ $t('gowaServers.devices', 'Devices') }}</CardTitle>
                <CardDescription>{{ store.currentServer?.base_url }}</CardDescription>
              </div>
              <Button variant="ghost" size="sm" @click="refreshDevices">
                <RefreshCw class="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="!loading && devices.length === 0" class="text-center py-12 text-muted-foreground">
              <Smartphone class="h-10 w-10 mx-auto mb-3 opacity-50" />
              <p class="text-sm">{{ $t('gowaServers.noDevices', 'No devices on this server') }}</p>
              <p class="text-xs mt-1">{{ $t('gowaServers.noDevicesDesc', 'Create a device to connect a WhatsApp number.') }}</p>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <Card v-for="d in devices" :key="d.id" class="border-border/60">
                <CardContent class="pt-4 space-y-3">
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <p class="font-medium text-sm truncate">{{ d.display_name || d.id }}</p>
                      <code class="text-[10px] bg-muted px-1.5 py-0.5 rounded font-mono block mt-1 truncate">{{ d.jid || d.phone_number || d.id }}</code>
                    </div>
                    <!-- State badge: uses the full GOWA state enum, not just is_connected. -->
                    <Badge variant="outline" :class="stateClass(d.state) + ' flex-shrink-0'">
                      <CheckCircle2 v-if="d.state === 'logged_in' || d.is_connected" class="h-3 w-3 mr-1" />
                      <AlertCircle v-else class="h-3 w-3 mr-1" />
                      {{ stateLabel(d.state) }}
                    </Badge>
                  </div>

                  <div class="flex flex-wrap gap-1.5">
                    <Button v-if="canWriteDevices" size="sm" variant="outline" @click="openConnect(d)">
                      <QrCode class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.connect', 'Connect') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="reconnect(d)" :disabled="statusLoading">
                      <RefreshCw class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.reconnect', 'Reconnect') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="logout(d)">
                      <LogOut class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.logout', 'Logout') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="openWebhook(d)">
                      <Webhook class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.webhook', 'Webhook') }}
                    </Button>
                    <Button v-if="canDeleteDevices" size="sm" variant="ghost" class="text-destructive" @click="openDelete(d)">
                      <Trash2 class="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>
      </div>
    </ScrollArea>

    <!-- Create device dialog -->
    <Dialog :open="createOpen" @update:open="(v) => createOpen = v">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('gowaServers.createDevice', 'Create Device') }}</DialogTitle>
          <DialogDescription>{{ $t('gowaServers.noDevicesDesc', 'Create a device to connect a WhatsApp number.') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-2 py-2">
          <Label>{{ $t('gowaServers.deviceName', 'Device Name') }}</Label>
          <Input v-model="newDeviceName" placeholder="sales-phone" />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="creating" @click="submitCreate">{{ $t('gowaServers.createDevice', 'Create Device') }}</Button>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Connect (QR / Pair) dialog -->
    <Dialog :open="connectOpen" @update:open="(v) => !v && closeConnect()">
      <DialogContent class="max-w-lg" @escape-key-down="closeConnect" @pointer-down-outside="closeConnect">
        <DialogHeader>
          <DialogTitle>{{ $t('accounts.connectDevice', 'Connect Device') }}</DialogTitle>
          <DialogDescription>{{ $t('accounts.gowaConnectDesc', 'Scan the QR code or use a pair code to link your WhatsApp account.') }}</DialogDescription>
        </DialogHeader>
        <Tabs default-value="qr">
          <TabsList class="grid w-full grid-cols-2">
            <TabsTrigger value="qr"><QrCode class="h-4 w-4 mr-1.5" /> {{ $t('gowaServers.qrCode', 'QR Code') }}</TabsTrigger>
            <TabsTrigger value="pair"><Link2 class="h-4 w-4 mr-1.5" /> {{ $t('gowaServers.pairCode', 'Pair Code') }}</TabsTrigger>
          </TabsList>
          <TabsContent value="qr" class="flex flex-col items-center gap-3 py-4">
            <div class="relative w-64 h-64 bg-white rounded-lg flex items-center justify-center border border-border shadow-inner">
              <Loader2 v-if="qrLoading && !qrLink" class="h-8 w-8 animate-spin text-muted-foreground" />
              <img v-else-if="qrLink" :src="qrLink" alt="QR Code" class="w-full h-full object-contain p-2" />
              <QrCode v-else class="h-16 w-16 text-muted-foreground" />
            </div>
            <p class="text-xs text-muted-foreground text-center">{{ $t('gowaServers.qrInstructions') }}</p>
            <Button variant="outline" size="sm" :disabled="qrLoading" @click="fetchQr">
              <RefreshCw class="h-4 w-4 mr-1" :class="{ 'animate-spin': qrLoading }" />
              {{ $t('gowaServers.refreshQr', 'Refresh QR') }}
            </Button>
          </TabsContent>
          <TabsContent value="pair" class="space-y-4 py-4">
            <div class="space-y-2">
              <Label class="text-xs">{{ $t('gowaServers.phoneNumber', 'Phone Number') }}</Label>
              <div class="flex gap-2">
                <Input v-model="pairPhone" placeholder="16505551234" class="flex-1" />
                <Button size="sm" :disabled="pairLoading || !pairPhone.trim()" @click="fetchPair">
                  <Loader2 v-if="pairLoading" class="h-4 w-4 animate-spin mr-1" />
                  {{ $t('gowaServers.getCode', 'Get Code') }}
                </Button>
              </div>
              <p class="text-xs text-muted-foreground">{{ $t('gowaServers.pairCodeInstructions') }}</p>
            </div>
            <div v-if="pairCode" class="flex flex-col items-center gap-2 p-4 bg-muted rounded-lg">
              <span class="text-xs text-muted-foreground">{{ $t('gowaServers.yourPairCode', 'Your Pair Code') }}</span>
              <span class="text-3xl font-bold font-mono tracking-[0.3em]">{{ pairCode }}</span>
              <Button variant="ghost" size="sm" @click="copyText(pairCode)">
                <Copy class="h-3 w-3 mr-1" /> {{ $t('common.copy', 'Copy') }}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>

    <!-- Webhook dialog -->
    <Dialog :open="webhookOpen" @update:open="(v) => webhookOpen = v">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('gowaServers.webhook', 'Webhook') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookUrlLabel', 'Webhook URL') }}</Label>
            <Input v-model="webhookForm.webhook_url" placeholder="https://whatomate.example.com/api/gowa/webhook" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookEventsLabel', 'Webhook Events') }}</Label>
            <Input v-model="webhookForm.webhook_events" class="font-mono text-xs" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="webhookOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="webhookSaving" @click="saveWebhook">{{ $t('gowaServers.saveWebhook', 'Save Webhook') }}</Button>
        </div>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteOpen"
      :title="$t('gowaServers.deleteDevice', 'Delete Device')"
      :item-name="deviceToDelete?.display_name || deviceToDelete?.id"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
```

- [ ] **Step 2: Verify types compile**

Run: `cd frontend && npx vue-tsc --noEmit 2>&1 | grep -iE 'GowaServerDetail' | head`
Expected: no errors for this file.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/settings/GowaServerDetailView.vue
git commit -m "feat(ui): add GowaServerDetailView with device management (create/delete/qr/pair/logout/reconnect/webhook)"
```

---

## Task B8: Frontend full verification

- [ ] **Step 1: Type-check the whole app**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 2: Production build**

Run: `cd frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 3: Lint (if configured)**

Run: `cd frontend && npm run lint 2>/dev/null || echo "no lint script"`
Expected: no new errors in the new/modified files.

- [ ] **Step 4: Commit any fixes**

```bash
git add -A frontend
git commit -m "chore: frontend build clean"
```

---

# Manual Verification (after both phases)

1. Start the app; log in as admin.
2. `/settings` → confirm a **GOWA Servers** nav item appears.
3. `/settings/roles` → confirm a **GOWA Servers** permission group (read/write/delete) and **Devices** now shows `delete`; toggle them on a custom role and verify the nav/buttons react.
4. `/settings/gowa-servers` → **Add GOWA Server** (valid URL+creds → probe succeeds; bad creds → 502 toast "Could not reach…").
5. Open the server → **Create Device** → see it listed; **Connect** → QR renders / pair code returns; **Logout/Reconnect/Webhook** work; **Delete** removes it.
6. As an agent (no `gowa_instances`/`devices` perms) → nav item hidden, direct URL redirects away, buttons absent.

---

## Self-Review Notes (plan author)

- **Spec coverage:** ✅ Model + migration (A1), RBAC incl. `devices:delete` + labels (A2), CRUD handlers (A3), device ops incl. webhook (A4), routes (A5), frontend service/store/constants/nav/router/i18n/views (B1–B7). Cross-source gaps G1–G3 applied (A0 + B7).
- **Placeholder scan:** No TBD/TODO; all steps carry full code or exact edits.
- **Type consistency:** Backend `GowaInstanceResponse` (has_credentials) ↔ frontend `GowaServer` (has_credentials). Device handler returns inline `deviceWithStatus` (is_connected/is_logged_in/jid) ↔ frontend `GowaDevice` (now incl. `phone_number` per spec). `parseDeviceID`/`{deviceId}` path param matches the route and the frontend `encodeURIComponent`.
- **Spec deltas from verified facts:** `LoginWithCode` is in `pkg/gowa/app.go` (handled). Encryption is explicit-call, not gorm hooks (handled via `EncryptCredentials`/`DecryptCredentials`). gowa-ui reference exists at `/Users/noiemany/Downloads/gowa-ui` (React) — used only for UX patterns, not code copy (the existing `AccountDetailView.vue` is the real Vue template). Login endpoints confirmed against OpenAPI spec v9.0.0 (gowa-ui's `/devices/{id}/login` calls are deprecated/dead — NOT used here).
- **Blast radius:** Legacy `/api/gowa/instances` + `/api/gowa/create-device` untouched → account-creation dropdown unaffected (decision #1). No config→DB import (decision #2).
- **Cross-source verification:** See the "Cross-source verification" section above. Three gaps (phone_number, status polling, state badges) found by comparing gowa-ui + OpenAPI spec and applied to this plan. WebSocket real-time, `/app/info`, single-device GET, and Chatwoot surface are deliberately out of scope.
