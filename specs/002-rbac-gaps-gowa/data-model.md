# Data Model: RBAC Gaps in GOWA + Media Features

**Feature**: `002-rbac-gaps-gowa`
**Date**: 2026-07-12
**Constitution**: Principles 3, 4, 7, 8

---

## Permission Catalog Changes (the only schema-level change)

This feature adds **no new database tables and no new model structs**. The only schema-adjacent change is to the permission catalog (seeded data, not DDL). Per constitution Principle 7, this is a `DefaultPermissions()` entry + `SystemRolePermissions()` mapping — GORM AutoMigrate handles the `permissions` table; the seed runs on startup.

### New resource constant

**File**: `internal/models/roles.go`

```go
// Add to the PermissionResource const block (after ResourceAccounts, ~line 64):
ResourceDevices = "devices"
```

### New permissions seeded

**File**: `internal/models/roles.go` — `DefaultPermissions()` function (~line 104)

```go
// Add after the Accounts permissions block (~line 132):
// Devices (GOWA device management — pairing, provisioning, status)
{Resource: ResourceDevices, Action: ActionRead, Description: "View GOWA device status and instances"},
{Resource: ResourceDevices, Action: ActionWrite, Description: "Pair and provision GOWA devices"},
```

### System role mappings

**File**: `internal/models/roles.go` — `SystemRolePermissions()` function (~line 242)

```go
// Add to managerPermissions slice (~line 256, after "accounts:delete"):
"devices:read", "devices:write",

// Do NOT add to agentPermissions — agents must not pair/provision devices
// or see GOWA instance topology.
```

**Role mapping summary:**

| Permission | admin | manager | agent | Rationale |
|------------|-------|---------|-------|-----------|
| `devices:read` | ✅ (auto-all) | ✅ | ❌ | Agents should not see infra topology (base URLs) |
| `devices:write` | ✅ (auto-all) | ✅ | ❌ | Pairing/provisioning emits webhook_secret — never agent |

---

## Existing Entities (no structural changes — documenting the fields this feature touches)

### WhatsAppAccount (`internal/models/models.go:293`)

The GOWA fields already exist on this struct (added in the audited commit range). This feature changes how they're *populated and verified*, not their schema.

| Field | Type | GORM tag | Purpose | Change in this feature |
|-------|------|----------|---------|----------------------|
| `ProviderType` | `string` | `gorm:"size:20;default:''"` | `"meta"` (default) or `"gowa"` | **No schema change.** FR-018: validated as explicit field at create — not inferred from other fields. |
| `GowaBaseURL` | `string` | `gorm:"size:255"` | GOWA REST API base URL for this account | No change. |
| `GowaDeviceID` | `string` | `gorm:"size:255"` | Custom device ID assigned during GOWA device creation | No change. |
| `GowaWebhookSecret` | `string` | `gorm:"size:255"` (in `EncryptFields`) | HMAC secret for verifying inbound GOWA webhooks | **Behavior change (FR-017):** auto-generated via `gowa.GenerateWebhookSecret()` at create/update if not supplied; backfilled for existing secretless accounts. |
| `OrganizationID` | `uuid.UUID` | `gorm:"type:uuid;index;not null"` | Tenant scope | No change (already present per Principle 4). |

### GOWAInstance (`internal/config/config.go:166`)

This is a TOML config struct, not a DB model. One field is added to enable org-scoped instance resolution (research R7).

| Field | Type | Koanf tag | Purpose | Change |
|-------|------|-----------|---------|--------|
| `Name` | `string` | `koanf:"name"` | Instance identifier | No change. |
| `BaseURL` | `string` | `koanf:"base_url"` | GOWA REST API base URL | No change. |
| `Username` | `string` | `koanf:"username"` | Basic Auth username | No change. |
| `Password` | `string` | `koanf:"password"` | Basic Auth password | No change. |
| `WebhookURL` | `string` | `koanf:"webhook_url"` | Callback URL for GOWA → whatomate | No change. |
| `Organizations` | `[]string` | `koanf:"organizations"` | Org IDs allowed to use this instance; `["*"]` or empty = all | **NEW** (research R7). Enables org-scoped `FindGOWAInstance`. |

### Permission / CustomRole / RolePermission (`internal/models/roles.go:8,20,38`)

No structural changes. The new `devices:read` / `devices:write` permissions are seeded into the existing `permissions` table and mapped via the existing `role_permissions` junction table. The `PermissionMatrix` UI auto-discovers them (constitution Principle 3: "They appear automatically in `/settings/roles`").

---

## Validation Rules

| Rule | Where enforced | Requirement |
|------|---------------|-------------|
| GOWA account MUST have a webhook secret | `accounts.go` CreateAccount/UpdateAccount | FR-017: auto-generate if empty |
| Provider type MUST be explicit | `accounts.go` CreateAccount | FR-018: `"meta"` or `"gowa"` — not inferred |
| Webhook signature MUST be present and valid | `gowa_webhook.go` handleGowaWebhook | FR-001/002: fail-closed |
| Webhook timestamp MUST be < 5 min old | `gowa_webhook.go` handleGowaWebhook | FR-005: replay protection |
| Device handlers MUST check `devices:read`/`write` | `gowa_device.go` all handlers | FR-006/007/008 |
| `GowaCreateDevice` instance MUST be org-scoped | `gowa_device.go` GowaCreateDevice | FR-009 |
| ZIP download MUST require `contacts:export` | `media_zip.go` ServeMediaZip | FR-013 (tiered media policy) |
| Re-download MUST enforce cooldown | `media_redownload.go` RedownloadMedia | FR-014 |
| Message mutations MUST be org-scoped | `gowa_webhook.go`, `webhook.go`, `chatbot_processor.go` | FR-004: `AND organization_id = ?` |

---

## State Transitions

### Inbound GOWA webhook processing (after fix)

```
[Request arrives]
    │
    ▼
[Parse envelope + extract device_id]
    │
    ▼
[Resolve account by device_id] ──(not found)──▶ 200 "device_not_configured" (no writes)
    │
    ▼
[Account has GowaWebhookSecret?] ──(no)──▶ 403 "Account not configured for webhook verification" (log alert)
    │                                          *** BACKFILL: generate secret, persist, then reject ***
    ▼
[Signature header present?] ──(no)──▶ 403 "Missing signature"
    │
    ▼
[VerifyWebhookSignature?] ──(fail)──▶ 403 "Invalid signature"
    │
    ▼
[Timestamp < 5 min old?] ──(no)──▶ 200 "ok" (silently drop, log Warn)
    │
    ▼
[Dispatch event → all writes scoped to account.OrganizationID]
    │
    ▼
[200 "ok"]
```

### GOWA device provisioning (after fix)

```
[POST /api/gowa/create-device]
    │
    ▼
[requireAuth(r, ResourceDevices, ActionWrite)] ──(fail)──▶ 403
    │
    ▼
[Resolve orgID-scoped instance] ──(not found / wrong org)──▶ 400 "Unknown instance for your organization"
    │
    ▼
[Generate webhook_secret + device_id]
    │
    ▼
[Create device on GOWA provider]
    │
    ▼
[Return device_id + webhook_secret to caller]
    │
    ▼
[logAudit(orgID, userID, "devices", deviceID, "write", nil, device)]
```
