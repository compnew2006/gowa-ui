# Phase 0 Research: RBAC Gaps in GOWA + Media Features

**Feature**: `002-rbac-gaps-gowa`
**Date**: 2026-07-12
**Status**: All decisions resolved

---

## R1. Webhook HMAC fail-close strategy

**Decision**: Replace the conditional guard `if sigHeader != "" && account.GowaWebhookSecret != ""` with a fail-closed check: reject if `secret == ""` OR `header == ""` OR `!verify(...)`.

**Rationale**: The current guard at `gowa_webhook.go:72` skips verification on two independent conditions, making the `main.go:543` comment "HMAC verified in handler" false. Fail-closed is the only correct posture for a multi-tenant inbound webhook — any request that cannot be authenticated MUST be rejected before any write. The underlying `gowa.VerifyWebhookSignature` (`pkg/gowa/verify.go:45`) is already correct (constant-time HMAC-SHA256); only the handler guard is broken.

**Alternatives considered**:
- *Require header only (skip if secret empty)*: Rejected — accounts without secrets would accept unsigned webhooks indefinitely (the current C4 vulnerability).
- *IP allowlist*: Rejected as sole control — GOWA instances may have dynamic IPs or sit behind a load balancer; HMAC is cryptographic proof of origin, IP is not.
- *Mutual TLS*: Rejected — overkill for this integration; GOWA is an HTTP webhook sender, not a long-lived mTLS peer.

**Implementation note**: The rejection must happen BEFORE `getGowaAccountByDeviceID` is called for the fallback path (which iterates all tenants' accounts and makes outbound calls — finding M5). Reorder: resolve account → if no secret, reject → if no header, reject → verify → process.

---

## R2. Replay protection mechanism

**Decision**: Reject webhooks whose `envelope.Timestamp` (or `msg.Timestamp`) is older than 5 minutes from the server's current time. Log rejected replays at `Warn` level for monitoring.

**Rationale**: 5 minutes is the industry standard (Stripe, GitHub, Slack). The GOWA webhook payload includes a Unix timestamp field (`pkg/gowa/webhook.go:18,29`). Clock drift between the GOWA server and whatomate is absorbed by the 5-minute window. Non-idempotent events (connection-status, revoked, edited) are the primary replay risk — message events are already deduplicated by `msg.ID` (`webhook.go:357-363`).

**Alternatives considered**:
- *Nonce store (Redis SET with TTL)*: Rejected — adds infrastructure complexity for marginal benefit over a timestamp window. A timestamp check is stateless and sufficient for the threat model (replay of a captured webhook, not real-time MITM).
- *15-minute window*: Rejected — unnecessarily long; extends the replay attack window.
- *Idempotency-only (no timestamp)*: Rejected — only covers `message` events; ack/connection/revoked/edited remain replayable (finding H3).

**Implementation note**: Add a `checkReplay(timestamp int64, maxAge time.Duration) bool` helper in `pkg/gowa/webhook.go` (pure function, unit-testable). Call it after HMAC verification, before event dispatch.

---

## R3. `devices` permission resource design

**Decision**: Add a new `ResourceDevices = "devices"` constant with two permissions: `devices:read` (view device status, list instances) and `devices:write` (pair QR, pair-code, provision device). Seed in `DefaultPermissions()`. Map to `admin` (auto-all) and `manager` (explicit); `agent` gets neither.

**Rationale**: Device provisioning emits the `webhook_secret` (a credential) and manipulates external infrastructure — this is admin/manager territory, never agent. A distinct resource (vs. reusing `accounts`) lets admins grant "manage accounts but not provision devices" independently, matching how every other feature (chat, campaigns, IVR) has its own resource group. The constitution (Principle 3) mandates: "New permissions are added as new `Resource` constants + `DefaultPermissions` entries."

**Alternatives considered**:
- *Reuse `accounts:write`*: Rejected — agents already have `accounts:read`, and conflating account CRUD with device provisioning prevents least-privilege expression. Also, `GowaInstances` exposes infra topology that `accounts:read` agents should not see.
- *Hybrid (devices:write + accounts:read)*: Rejected — agents with `accounts:read` would still see GOWA instance base URLs via the instance-list endpoint, which is a topology leak.

**Implementation note**: Add to `internal/models/roles.go`:
```go
ResourceDevices = "devices"  // in the const block (after ResourceAccounts)
// In DefaultPermissions():
{Resource: ResourceDevices, Action: ActionRead, Description: "View GOWA device status and instances"},
{Resource: ResourceDevices, Action: ActionWrite, Description: "Pair and provision GOWA devices"},
// In SystemRolePermissions() managerPermissions:
"devices:read", "devices:write",
// agentPermissions: (neither — do not add)
```

---

## R4. Org-scoping the device-id webhook lookup

**Decision**: The `getGowaAccountByDeviceID` query (`gowa_webhook.go:113-116`) currently matches across all orgs globally. After HMAC verification confirms the request is from GOWA, the lookup itself can remain global (since a validly-signed webhook proves the device is real), BUT all downstream writes MUST use `account.OrganizationID` exclusively (already the case for most writes). The remaining gaps are the reaction/revoked/edited/ack updates that match on `whats_app_message_id` without an org clause — these MUST add `AND organization_id = ?`.

**Rationale**: The webhook is already authenticated by HMAC (after R1). The device-id → account resolution is a lookup, not an authorization decision — the authorization is "is this webhook genuinely from GOWA for this device?" which HMAC answers. The real cross-tenant risk is in the *mutation* queries (C7), not the lookup. Scoping the lookup query itself is unnecessary (and would break the legitimate fallback path at `gowa_webhook.go:122-142` which checks connected JIDs across instances). However, the fallback path's outbound `GetAppStatus` calls (M5) should be rate-limited or removed if they become an abuse vector.

**Alternatives considered**:
- *Add `organization_id` to the lookup query*: Rejected — the webhook doesn't carry an org ID; the org is determined BY the device-id lookup. You can't filter by orgID before you know which org the device belongs to.
- *Remove the fallback path entirely*: Considered — it makes outbound calls to every tenant's GOWA instance on an unauthenticated request. Should be removed or gated behind HMAC verification (move it after the HMAC check, not before).

**Implementation note**: The fix for C7 is adding `AND organization_id = ?` to:
- `gowa_webhook.go:654` (processGowaRevoked): `Where("whats_app_message_id = ? AND organization_id = ?", revoked.RevokedMessageID, account.OrganizationID)`
- `gowa_webhook.go:702` (processGowaEdited): same pattern
- `webhook.go:407` (updateMessageStatus): add `AND organization_id = ?` — but this is called from both Meta and GOWA paths, so the orgID must be passed in (signature change)
- `chatbot_processor.go:1297` (handleIncomingReaction): same pattern

---

## R5. Webhook secret auto-generation

**Decision**: When a GOWA-type account is created or updated and the caller does not supply a `GowaWebhookSecret`, the system auto-generates one via the existing `gowa.GenerateWebhookSecret()` and stores it (encrypted, via the existing `EncryptFields` mechanism at `accounts.go:1049`). Existing GOWA accounts without a secret are backfilled — either via a startup migration check or lazily on first webhook arrival (generate + persist before rejecting).

**Rationale**: The caller should never need to manually supply a webhook secret — it's an internal credential, not a user-facing config. The existing `gowa.GenerateWebhookSecret()` (`gowa_device.go:195`) already produces a suitable secret. Auto-generation at creation ensures no GOWA account is ever left unprotected (FR-017).

**Alternatives considered**:
- *Require caller-supplied secret (400 if missing)*: Rejected — adds friction and a migration burden for existing accounts; the secret is infrastructure, not user input.
- *Lazy generation on first webhook*: Rejected as sole mechanism — the first webhook would be rejected (no secret yet), and the account would appear broken. Better to generate at creation.
- *Startup backfill migration*: Preferred for existing accounts — a simple `UPDATE whatsapp_accounts SET gowa_webhook_secret = ? WHERE provider_type = 'gowa' AND gowa_webhook_secret = ''` loop at startup, generating a secret per account.

**Implementation note**: In `accounts.go` `CreateAccount` (~line 130) and `UpdateAccount` (~line 290), after determining the account is GOWA-type, add:
```go
if req.GowaWebhookSecret == "" && account.IsGowa() {
    account.GowaWebhookSecret = gowa.GenerateWebhookSecret()
}
```
For backfill, add a check in the startup sequence (`cmd/whatomate/main.go` after DB init) that scans for secretless GOWA accounts and generates secrets.

---

## R6. Media export permission tiering

**Decision**: `ServeMediaZip` (`media_zip.go:81`) changes its permission gate from `contacts:read` to `contacts:export`. `RedownloadMedia` (`media_redownload.go:48`) stays on `contacts:read` but adds a per-item cooldown (Redis key `media:redownload:{message_id}` with a TTL, e.g. 60 seconds). The frontend hides ZIP/separate-download controls from users lacking `contacts:export`; the retry button stays visible to all `contacts:read` users.

**Rationale**: Bulk ZIP download is a data export — mapping it to `contacts:export` (which already exists, is seeded, and is mapped to manager) is consistent with how the project treats contact export. Re-download is a single-item recovery operation gated on read + cooldown; restricting it to `contacts:write` would block agents who can chat but lack write, which is unnecessary friction for a single-item fix.

**Alternatives considered**:
- *New `media:download` / `media:redownload` resources*: Rejected — adds catalog/migration work beyond what's needed; `contacts:export` already exists and fits the use case.
- *Gate re-download on `contacts:write`*: Rejected — re-download is a recovery action, not a mutation the user intended; `contacts:read` + cooldown is sufficient abuse control.

**Implementation note**: In `media_zip.go:81`, change `HasPermission(userID, ResourceContacts, ActionRead, orgID)` to `HasPermission(userID, ResourceContacts, ActionExport, orgID)`. In `media_redownload.go`, before the provider call (~line 81), add a Redis `SET media:redownload:{msgID} 1 EX 60 NX` check; if the key exists, return a 429. In the frontend `ChatView.vue:1979` and `MediaBurstDialog.vue`, wrap the collect/zip controls in `v-if="authStore.hasPermission('contacts', 'export')"`.

---

## R7. Org-scoped instance resolution for GowaCreateDevice

**Decision**: `GowaCreateDevice` (`gowa_device.go:158`) currently lets the caller pick any `base_url` from the global `a.Config.GOWAInstances` list. The fix scopes instance selection to the caller's organization. Since GOWA instances are configured globally in TOML (not per-org in the DB), the scoping is done by either: (a) a config-level `AllowedOrganizations []uuid.UUID` field on `GOWAInstance`, or (b) a simpler approach — since most deployments have one org per instance, validate that the selected instance's webhook URL domain matches the caller's org's configured accounts.

**Rationale**: Without org-scoping, a user from org A can provision devices on org B's GOWA instance and obtain its credentials (finding C2). The TOML-based config doesn't currently map instances to orgs, so a lightweight mapping is needed.

**Alternatives considered**:
- *Per-org instance config in the DB*: Rejected — over-engineering; GOWA instances are infrastructure configured by the deployer, not per-org records.
- *Single global instance (ignore base_url from caller)*: Rejected — multi-instance support was a deliberate feature (commit `668db22`); the fix should preserve it but scope it.

**Implementation note**: Add an `Organizations []string` (org IDs or `*` for all) field to `config.GOWAInstance` (`config.go:166`). In `FindGOWAInstance`, filter by the caller's orgID. If no org mapping is configured (backward compat), allow all (but log a warning). This keeps existing deployments working while enabling org-scoped multi-instance.

---

## R8. `updateMessageStatus` org-scope signature change

**Decision**: `updateMessageStatus` (`webhook.go:399`) is called from both the Meta webhook path and the GOWA webhook path. It currently takes only `(whatsappMsgID, statusValue, errors)`. To add the `AND organization_id = ?` clause (R4), the function signature must gain an `orgID uuid.UUID` parameter, and all callers must pass it.

**Rationale**: The function matches messages by `whats_app_message_id` globally (finding C7). Adding an orgID filter prevents a forged webhook from mutating another org's message status. The signature change is small but touches all callers — the Meta webhook path (`webhook.go:328`) has the account/org context available, and the GOWA path (`gowa_webhook.go` ack handler) has `account.OrganizationID`.

**Alternatives considered**:
- *Create a separate GOWA-specific status update function*: Rejected — duplicates logic; the fix is the same for both providers.
- *Filter by account ID instead of org*: Rejected — orgID is the tenant boundary (Principle 4); accountID would still allow cross-account-within-org which is less critical but inconsistent.

**Implementation note**: Change `updateMessageStatus(whatsappMsgID, statusValue string, errors []WebhookStatusError)` to `updateMessageStatus(orgID uuid.UUID, whatsappMsgID, statusValue string, errors []WebhookStatusError)`. Update the query to `Where("whats_app_message_id = ? AND organization_id = ?", whatsappMsgID, orgID)`. Update all 2-3 callers to pass the orgID from their resolved account.
