# API Contract: GOWA Inbound Webhook

**Feature**: `002-rbac-gaps-gowa`
**Constitution**: Principles 2, 4, 5, 16

These endpoints are **public** (no JWT/session auth) — they are machine-to-machine, authenticated via HMAC signature. Registered outside the authed route group (`cmd/whatomate/main.go:545-548`).

---

## POST /api/gowa/webhook

Single-endpoint webhook: GOWA sends all events here. The `device_id` in the payload resolves the target account.

**Authentication**: HMAC-SHA256 signature via `X-Hub-Signature-256` header, verified against the account's `GowaWebhookSecret`.

**Request headers**:
| Header | Required | Purpose |
|--------|----------|---------|
| `X-Hub-Signature-256` | ✅ Yes | `sha256={hex_hmac}` — HMAC of the raw body |
| `Content-Type` | ✅ Yes | `application/json` |

**Request body** (GOWA webhook envelope):
```json
{
  "event": "message",
  "device_id": "628123456789@s.whatsapp.net",
  "timestamp": 1720795200,
  "message": { "id": "wamid.XYz", "from": "628987654321", "type": "text", "text": "hello" }
}
```

**Verification flow** (fail-closed — research R1):
1. Parse envelope, extract `device_id`
2. Resolve account by `device_id` → if not found: `200 {"status":"device_not_configured"}` (no writes)
3. **If `account.GowaWebhookSecret == ""`**: backfill-generate secret, persist, then `403 "Account not configured for webhook verification"` + log alert (FR-017)
4. **If `X-Hub-Signature-256` header missing**: `403 "Missing signature"` (FR-001)
5. **If `!VerifyWebhookSignature(rawBody, sigHeader, secret)`**: `403 "Invalid signature"` (FR-001)
6. **If `timestamp` older than 5 minutes**: `200 {"status":"ok"}` (silently drop, log Warn) (FR-005)
7. Dispatch event → all writes scoped to `account.OrganizationID`

**Success 200** (envelope):
```json
{ "status": "success", "data": { "status": "ok" } }
```

**Error responses** (all use `SendErrorEnvelope`):
| Code | HTTP | Condition |
|------|------|-----------|
| `Missing signature` | 403 | No `X-Hub-Signature-256` header |
| `Invalid signature` | 403 | HMAC mismatch / tampered body |
| `Account not configured for webhook verification` | 403 | Account has no `GowaWebhookSecret` |
| `Stale webhook` | 200 | Timestamp > 5 min old (logged, silently dropped) |

**Supported events**: `message`, `message.ack`, `chat_presence`, `connection`, `message.reaction`, `message.revoked`, `message.edited`

---

## POST /api/gowa/webhook/{device_id}

Per-device webhook: the `{device_id}` path segment overrides the payload's `device_id` field.

**Authentication**: Same HMAC flow as above.

**Path params**: `device_id` (string) — GOWA session identifier (e.g. `628123456789@s.whatsapp.net`)

**Verification flow**: Identical to the single-endpoint flow. The `pathDeviceID` overrides `envelope.DeviceID` at resolution time (`gowa_webhook.go:52-54`).

---

## Downstream write contracts (org-scoped — research R4)

Every write triggered by a verified webhook MUST be scoped to `account.OrganizationID`. The following queries are fixed to add `AND organization_id = ?`:

| Operation | File:Line (current) | Fix |
|-----------|---------------------|-----|
| `processGowaRevoked` | `gowa_webhook.go:654` | `Where("whats_app_message_id = ? AND organization_id = ?", revoked.RevokedMessageID, account.OrganizationID)` |
| `processGowaEdited` | `gowa_webhook.go:702` | `Where("whats_app_message_id = ? AND organization_id = ?", edited.OriginalMessageID, account.OrganizationID)` |
| `updateMessageStatus` | `webhook.go:407` | Signature gains `orgID uuid.UUID`; query: `Where("whats_app_message_id = ? AND organization_id = ?", whatsappMsgID, orgID)` |
| `handleIncomingReaction` | `chatbot_processor.go:1297` | `Where("whats_app_message_id LIKE ? AND organization_id = ?", "%"+messageWAMID+"%", account.OrganizationID)` |

**Constitution Principle 4**: "Every database query touching tenant-scoped data MUST filter by `organization_id`."

---

## Replay protection (research R2)

After HMAC verification, before event dispatch:

```go
if !gowa.CheckReplay(envelope.Timestamp, 5*time.Minute) {
    a.Log.Warn("Stale GOWA webhook rejected (replay)", "device_id", envelope.DeviceID, "timestamp", envelope.Timestamp)
    return r.SendEnvelope(map[string]string{"status": "ok"}) // 200 to prevent GOWA retries
}
```

`CheckReplay` is a pure function in `pkg/gowa/webhook.go`:
```go
func CheckReplay(timestamp int64, maxAge time.Duration) bool {
    if timestamp == 0 { return false } // missing timestamp = reject
    age := time.Since(time.Unix(timestamp, 0))
    return age <= maxAge && age >= -maxAge // allow 5 min clock drift in either direction
}
```
