# API Contract: GOWA Device Management

**Feature**: `002-rbac-gaps-gowa`
**Constitution**: Principles 2, 3, 5, 6, 17

All endpoints use fastglue handlers (`func (a *App) Name(r *fastglue.Request) error`), response envelopes (`SendEnvelope`/`SendErrorEnvelope`), and handler-level `requireAuth` permission checks.

---

## Permission Requirements

| Endpoint | Permission | Constitution ref |
|----------|------------|-----------------|
| All device endpoints | `devices:read` or `devices:write` | Principle 3 (handler-level permissions) |

Every handler MUST open with:
```go
orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.Action<Read|Write>)
if err != nil { return nil }
```

---

## POST /api/gowa/create-device

Provisions a new GOWA device on an org-scoped instance. Returns the device ID and webhook secret.

**Permission**: `devices:write`

**Request body**:
```json
{
  "base_url": "http://gowa:8080",
  "device_name": "whatomate-device"
}
```

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": {
    "device_id": "test-account-d9768a03",
    "webhook_secret": "<generated-secret>",
    "base_url": "http://gowa:8080"
  }
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `devices:write` |
| `Invalid request body` | 400 | Malformed JSON |
| `base_url is required` | 400 | Missing `base_url` |
| `Unknown GOWA instance for your organization` | 400 | `base_url` not in org's allowed instances (FR-009) |
| `Failed to create device on GOWA` | 502 | GOWA provider error |

**Audit**: `logAudit(orgID, userID, "devices", deviceID, "write", nil, device)` (Principle 17)

**Change from current**: `getOrgID` → `requireAuth(ResourceDevices, ActionWrite)`; `FindGOWAInstance` gains org-scoping (research R7).

---

## GET /api/gowa/instances

Lists GOWA instances available to the caller's organization.

**Permission**: `devices:read`

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": [
    {"name": "primary", "base_url": "http://gowa:8080"}
  ]
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `devices:read` |

**Change from current**: `getOrgID` → `requireAuth(ResourceDevices, ActionRead)`; response filtered to org-allowed instances (does NOT include `username`/`password`/`webhook_url` — only `name` + `base_url`).

---

## GET /api/accounts/{id}/gowa/qr

Retrieves a pairing QR code for a GOWA account.

**Permission**: `devices:write`

**Path params**: `id` (UUID) — the WhatsApp account ID

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": {
    "qr_code": "data:image/png;base64,..."
  }
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `devices:write` |
| `Account not found` | 404 | Account ID not in caller's org (IDOR-safe via `findByIDAndOrg`) |

**Change from current**: `getOrgID` → `requireAuth(ResourceDevices, ActionWrite)`. Org-scope already present via `resolveGowaAccount` → `findByIDAndOrg` (no IDOR).

---

## POST /api/accounts/{id}/gowa/pair-code

Submits a phone-number pairing code for a GOWA account.

**Permission**: `devices:write`

**Path params**: `id` (UUID)

**Request body**:
```json
{
  "phone": "628123456789"
}
```

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": {
    "paired": true
  }
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `devices:write` |
| `phone is required` | 400 | Empty phone |
| `Account not found` | 404 | Not in caller's org |

**Change from current**: `getOrgID` → `requireAuth(ResourceDevices, ActionWrite)`.

---

## GET /api/accounts/{id}/gowa/status

Retrieves the GOWA device connection status.

**Permission**: `devices:read`

**Path params**: `id` (UUID)

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": {
    "connected": true,
    "device_id": "test-account-d9768a03",
    "phone": "628123456789"
  }
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `devices:read` |
| `Account not found` | 404 | Not in caller's org |
| `Failed to get device status` | 502 | GOWA provider error |

**Change from current**: `getOrgID` → `requireAuth(ResourceDevices, ActionRead)`.
