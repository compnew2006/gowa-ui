# API Contract: Media Export & Re-download

**Feature**: `002-rbac-gaps-gowa`
**Constitution**: Principles 3, 4, 5, 14

---

## Permission Policy (confirmed in clarification Q1)

| Endpoint | Permission | Rationale |
|----------|------------|-----------|
| `GET /api/media/zip` | `contacts:export` | Bulk ZIP download is a data export (research R6) |
| `POST /api/media/{message_id}/redownload` | `contacts:read` + cooldown | Single-item recovery, not a bulk export |
| `GET /api/media/{message_id}` | `contacts:read` | Unchanged (existing, correct) |

---

## GET /api/media/zip

Downloads a ZIP archive of media files for the given message IDs.

**Permission**: `contacts:export` (changed from `contacts:read`)

**Query params**:
| Param | Type | Required | Default | Constraint |
|-------|------|----------|---------|------------|
| `ids` | string (comma-separated UUIDs) | ✅ Yes | — | Max 50 IDs (existing `maxZipMessageIDs`) |
| `contact_id` | UUID | No | — | If provided, scopes to a specific conversation |

**Success 200**: Binary `application/zip` stream (not a JSON envelope — this is a file download).

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `contacts:export` (was `contacts:read`) |
| `No message IDs provided` | 400 | Empty `ids` param |
| `Too many message IDs` | 400 | > 50 IDs |
| `No media found` | 404 | No matching media for the given IDs in the caller's org |

**Org-scoping** (unchanged, already correct — IDOR-safe):
- Query: `WHERE id IN ? AND organization_id = ? AND media_url <> ''` (`media_zip.go:69`)
- Per-contact ownership: `canAccessContactMedia` checks `scopeAssignedContact` + team-transfer membership (`media_zip.go:163-179`)

**Size guard** (new — FR-015):
- Max total uncompressed size: 250 MB. If the sum of media file sizes exceeds this, return `413 "ZIP archive too large"`.

**Change from current**: `HasPermission(userID, ResourceContacts, ActionRead, orgID)` → `HasPermission(userID, ResourceContacts, ActionExport, orgID)` at `media_zip.go:81`.

---

## POST /api/media/{message_id}/redownload

Triggers a provider re-download for a message whose media failed to fetch.

**Permission**: `contacts:read` (unchanged)

**Cooldown** (new — FR-014, research R6):
- Redis key: `media:redownload:{message_id}`
- TTL: 60 seconds
- If key exists on request: return `429 "Re-download recently performed for this message. Please wait and try again."`

**Path params**: `message_id` (UUID)

**Success 200** (envelope):
```json
{
  "status": "success",
  "data": {
    "message_id": "uuid-here",
    "media_url": "/uploads/media/abc123.jpg",
    "media_mime_type": "image/jpeg",
    "redownloaded": true
  }
}
```

**Errors**:
| Code | HTTP | Condition |
|------|------|-----------|
| `Unauthorized` | 401 | Not authenticated |
| `Insufficient permissions` | 403 | Lacks `contacts:read` or not assigned to the contact |
| `Message not found` | 404 | Message ID not in caller's org (IDOR-safe) |
| `Re-download recently performed` | 429 | Cooldown active (60s) |
| `Media download failed` | 502 | Provider re-fetch failed |

**Org-scoping** (unchanged, already correct):
- `WHERE id = ? AND organization_id = ?` (`media_redownload.go:43`)
- Provider call happens AFTER all authz + cooldown checks pass

**Change from current**: Add Redis cooldown check before the provider call at `media_redownload.go:81`.

---

## Frontend gating (FR-013)

| Component | Gate | Behavior |
|-----------|------|----------|
| ChatView.vue "Collect files" button (`:1979`) | `authStore.hasPermission('contacts', 'export')` | Hidden if lacks export |
| ChatView.vue floating chip (`:2081`) | `authStore.hasPermission('contacts', 'export')` | Hidden if lacks export |
| MediaBurstDialog.vue ZIP button (`:111`) | `authStore.hasPermission('contacts', 'export')` | Hidden if lacks export |
| MediaBurstDialog.vue separate button (`:107`) | `authStore.hasPermission('contacts', 'export')` | Hidden if lacks export |
| MediaRetryButton.vue retry (`:25`) | `authStore.hasPermission('contacts', 'read')` | Visible (stays at read — re-download is read + cooldown) |
| useMediaExport.ts `downloadAsZip` (`:44`) | Check `hasPermission('contacts','export')` before fetch | Return early if lacks permission |
| AccountDetailView.vue "Connect Device" (`:490`) | `authStore.hasPermission('devices', 'write')` | Hidden if lacks devices:write |

**i18n keys to add** (all 5 locales: en, ar, es, hi, ta):
- `errors.media.tooLarge` — "ZIP archive too large"
- `errors.media.redownloadCooldown` — "Re-download recently performed. Please wait and try again."
- `errors.gowa.missingSignature` — "Missing webhook signature"
- `errors.gowa.invalidSignature` — "Invalid webhook signature"
- `errors.gowa.noSecret` — "Account not configured for webhook verification"
