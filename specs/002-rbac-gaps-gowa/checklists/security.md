# Security Requirements Checklist: Close RBAC / User-Role Gaps in GOWA + Media Features

**Purpose**: Validate the quality, completeness, clarity, and consistency of the *security requirements* in the spec, plan, contracts, and research docs — NOT whether the implementation works. These are "unit tests for English": does the documentation that defines the security fix have any gaps, ambiguities, or contradictions that could leave a vulnerability open or cause a misimplementation?
**Created**: 2026-07-12
**Feature**: [spec.md](../spec.md) | [plan.md](../plan.md) | [contracts/](../contracts/)

**Focus**: Security requirements (HMAC fail-close, replay protection, org-scoping, permission gating, IDOR, secret handling, tenant isolation)
**Depth**: Standard (pre-implementation review gate)
**Audience**: PR reviewer / security reviewer

---

## Requirement Completeness

- [ ] CHK001 Are requirements specified for ALL five GOWA device-management endpoints (QR, pair-code, status, instances, create-device) having an explicit permission gate? [Completeness, Spec §FR-006/007/008, contracts/gowa-device-api.md]
- [ ] CHK002 Are requirements specified for the GOWA webhook handler rejecting requests when the signature header is absent — not only when it's present-and-wrong? [Completeness, Spec §FR-001, contracts/gowa-webhook-api.md]
- [ ] CHK003 Are requirements specified for the GOWA webhook handler rejecting requests when the account has no webhook secret configured (the fail-open-on-empty-secret path)? [Completeness, Spec §FR-002/017, contracts/gowa-webhook-api.md]
- [ ] CHK004 Are requirements specified for org-scoping on EVERY downstream write triggered by a verified webhook (contact creation, message storage, status update, reaction, revocation, edit, chatbot reply, WebSocket broadcast)? [Completeness, Spec §FR-003/004, contracts/gowa-webhook-api.md]
- [ ] CHK005 Are requirements specified for org-scoping on the reaction/revoked/edited/ack DB mutation queries specifically (the `whats_app_message_id`-only matches)? [Completeness, Spec §FR-004, research R4]
- [ ] CHK006 Are requirements specified for replay protection covering ALL non-idempotent webhook events (connection-status, revoked, edited, ack), not just message events? [Completeness, Spec §FR-005, research R2]
- [ ] CHK007 Are requirements specified for the `devices` permission being seeded in the default permission catalog AND mapped to system roles (admin/manager/agent)? [Completeness, Spec §FR-010/011, data-model.md]
- [ ] CHK008 Are requirements specified for the GOWA instance list endpoint NOT exposing credentials (username/password) in its response? [Completeness, contracts/gowa-device-api.md]
- [ ] CHK009 Are requirements specified for auditing the device-provisioning action (the one that emits the webhook secret)? [Completeness, Spec §FR-006, constitution Principle 17]
- [ ] CHK010 Are requirements specified for the media ZIP download being gated on the export permission (not just read)? [Completeness, Spec §FR-013, contracts/media-api.md]
- [ ] CHK011 Are requirements specified for the media re-download cooldown having a defined duration and a defined rejection response? [Completeness, Spec §FR-014, contracts/media-api.md]
- [ ] CHK012 Are requirements specified for the media ZIP total-size guard having a defined threshold and rejection response? [Completeness, Spec §FR-015, contracts/media-api.md]
- [ ] CHK013 Are requirements specified for auto-generating the webhook secret at account creation AND backfilling existing secretless accounts? [Completeness, Spec §FR-017, research R5]
- [ ] CHK014 Are requirements specified for the provider-type field being explicit (not inferred from populated fields)? [Completeness, Spec §FR-018]

## Requirement Clarity

- [ ] CHK015 Is the replay-protection freshness window quantified with a specific value (5 minutes) rather than a vague "defined window"? [Clarity, Spec §FR-005, Clarifications Q4]
- [ ] CHK016 Is the re-download cooldown duration quantified with a specific value (60 seconds) rather than "a cooldown"? [Clarity, Spec §FR-014, contracts/media-api.md]
- [ ] CHK017 Is the ZIP total-size guard threshold quantified with a specific value (250 MB) rather than "a maximum size"? [Clarity, Spec §FR-015, contracts/media-api.md]
- [ ] CHK018 Is the maximum ZIP item count specified (50) and is it clear whether the count check runs before or after UUID parsing? [Clarity, contracts/media-api.md, research R6]
- [ ] CHK019 Is the distinction between `devices:read` (status, instances) and `devices:write` (pair, QR, provision) explicitly mapped to each endpoint? [Clarity, Spec §FR-006/007/008, contracts/gowa-device-api.md]
- [ ] CHK020 Is the distinction between `contacts:export` (ZIP) and `contacts:read` (re-download) explicitly stated as a tiered policy? [Clarity, Spec §FR-013, Clarifications Q1, contracts/media-api.md]
- [ ] CHK021 Is the fail-close verification ordering specified (resolve account → check secret exists → check header exists → verify HMAC → check replay → dispatch)? [Clarity, contracts/gowa-webhook-api.md, data-model.md state-transition diagram]
- [ ] CHK022 Is the org-scoped instance resolution mechanism specified clearly enough that an implementer knows how `Organizations []string` on `GOWAInstance` is interpreted (`["*"]` = all, specific UUIDs = restricted)? [Clarity, research R7, data-model.md]
- [ ] CHK023 Is the `updateMessageStatus` signature change (adding `orgID` parameter) specified with enough detail that all callers (both Meta and GOWA webhook paths) are identified? [Clarity, research R8, contracts/gowa-webhook-api.md]
- [ ] CHK024 Is the behavior for the unauthenticated fallback path (`getGowaAccountByDeviceID` iterating all tenants + making outbound `GetAppStatus` calls) specified — should it be removed, moved after HMAC, or rate-limited? [Clarity, research R4, Gap]

## Requirement Consistency

- [ ] CHK025 Do the spec, contracts, and research docs agree on which permission gates each device endpoint uses (e.g., does `GowaDeviceStatus` consistently use `devices:read` everywhere)? [Consistency, Spec §FR-007, contracts/gowa-device-api.md, research R3]
- [ ] CHK026 Does the media policy (ZIP → export, re-download → read) remain consistent across the spec, contracts, and tasks (no document says read while another says export for the same endpoint)? [Consistency, Spec §FR-013, contracts/media-api.md, Clarifications Q1]
- [ ] CHK027 Is the agent role mapping consistent — does the spec say "agent gets neither devices:read nor devices:write" and does the data-model confirm the same in `SystemRolePermissions()`? [Consistency, Spec §FR-011, data-model.md]
- [ ] CHK028 Is the replay-window value (5 min) consistent across the spec (FR-005), the Clarifications log (Q4), the contracts (gowa-webhook-api.md), and the research (R2)? [Consistency]
- [ ] CHK029 Is the webhook-secret auto-generation requirement (FR-017) consistent with the fail-close requirement (FR-002) — i.e., if accounts always have secrets, the empty-secret rejection path is a safety net, not the primary flow? [Consistency, Spec §FR-002/017, research R1/R5]
- [ ] CHK030 Is the org-scoping requirement (FR-004) consistent with the constitution Principle 4 mandate ("every query scoped by organization_id")? [Consistency, constitution §P4]

## Acceptance Criteria Quality

- [ ] CHK031 Are the success criteria measurable and security-verifiable (e.g., "100% of forged webhooks rejected with zero DB writes" rather than "webhooks are secure")? [Measurability, Spec §SC-001/002/003]
- [ ] CHK032 Can the cross-org provisioning refusal (SC-003) be objectively verified without ambiguity about what constitutes "org B's instance"? [Measurability, Spec §SC-003, research R7]
- [ ] CHK033 Can the "zero device-management controls visible to agents" criterion (SC-005) be objectively verified — is "zero controls" enumerating specific UI elements? [Measurability, Spec §SC-005, contracts/media-api.md frontend-gating table]
- [ ] CHK034 Is the "all 27 findings resolved or accepted-with-justification" criterion (SC-007) traceable — is there a mapping from each finding to the requirement that addresses it? [Measurability, Spec §SC-007, Gap]

## Scenario Coverage

- [ ] CHK035 Are requirements defined for the scenario where a validly-signed webhook references a message ID belonging to a DIFFERENT organization than the resolved device (cross-org mutation attempt)? [Coverage, Spec §FR-004, contracts/gowa-webhook-api.md]
- [ ] CHK036 Are requirements defined for the scenario where two organizations share a phone number or device ID (ambiguous device-id resolution)? [Coverage, Spec §Edge Cases, Gap]
- [ ] CHK037 Are requirements defined for the scenario where a webhook arrives during the backfill window (account has no secret yet, backfill hasn't run)? [Coverage, Spec §FR-017, research R5, Gap]
- [ ] CHK038 Are requirements defined for the scenario where a manager's device-management permission is revoked mid-pairing-session (in-flight request)? [Coverage, Spec §Edge Cases]
- [ ] CHK039 Are requirements defined for the scenario where a user requests a ZIP containing items from conversations they can no longer access (permission revoked mid-request)? [Coverage, Spec §Edge Cases]
- [ ] CHK040 Are requirements defined for the scenario where the permission catalog is seeded on a database that already has custom roles (existing roles untouched, only new defaults added)? [Coverage, Spec §Edge Cases]
- [ ] CHK041 Are requirements defined for the unauthenticated outbound-call amplification vector (the fallback path making `GetAppStatus` calls to every tenant's GOWA instance on an unauthenticated POST)? [Coverage, finding M5, research R4, Gap]
- [ ] CHK042 Are requirements defined for the information-leak vector (distinct responses for "unknown device" vs "bad signature" enabling device enumeration)? [Coverage, finding M4, contracts/gowa-webhook-api.md, Gap]

## Edge Case Coverage

- [ ] CHK043 Are edge-case requirements defined for what happens when the GOWA webhook payload has a zero or missing timestamp (does replay protection reject or skip)? [Edge Case, Spec §FR-005, research R2]
- [ ] CHK044 Are edge-case requirements defined for clock drift between the GOWA server and whatomate (does the 5-min window account for future-dated timestamps)? [Edge Case, research R2, contracts/gowa-webhook-api.md]
- [ ] CHK045 Are edge-case requirements defined for the scenario where the `pathDeviceID` (per-device webhook route) conflicts with the payload's `device_id` (which takes precedence)? [Edge Case, contracts/gowa-webhook-api.md, Gap]
- [ ] CHK046 Are edge-case requirements defined for what HTTP status the system returns when silently dropping a stale/replayed webhook (200 to prevent GOWA retries, vs 403)? [Edge Case, Spec §FR-005, contracts/gowa-webhook-api.md]
- [ ] CHK047 Are edge-case requirements defined for the scenario where a re-download cooldown key exists in Redis but the media file was deleted (does the 429 still fire, or is the cooldown cleared)? [Edge Case, Spec §FR-014, Gap]
- [ ] CHK048 Are edge-case requirements defined for the scenario where a GOWA account is switched from Meta-type to GOWA-type via update (does the secret auto-generate on type change)? [Edge Case, Spec §FR-017/018, Gap]

## Non-Functional Requirements (Security-Specific)

- [ ] CHK049 Are observability/logging requirements specified for security-rejection events (missing signature, invalid signature, empty secret, stale timestamp, cross-org mutation blocked)? [Non-Functional, Spec §FR-016, constitution Principle 16]
- [ ] CHK050 Are requirements specified for what log level and fields each rejection event uses (e.g., `Warn` with `device_id` + `timestamp`, no secret leakage in logs)? [Non-Functional, constitution Principle 16, Gap]
- [ ] CHK051 Are requirements specified for the webhook secret being encrypted at rest (not stored in plaintext)? [Non-Functional, constitution Principle 18, data-model.md]
- [ ] CHK052 Are requirements specified for the webhook secret NOT being logged or leaked in error responses? [Non-Functional, Gap]
- [ ] CHK053 Are requirements specified for rate-limiting the public GOWA webhook endpoint to prevent brute-force signature attempts? [Non-Functional, Gap]
- [ ] CHK054 Are requirements specified for the timing-safety of the HMAC comparison (constant-time, no early-return on length mismatch)? [Non-Functional, contracts/gowa-webhook-api.md, Gap — note: the existing `VerifyWebhookSignature` is already constant-time, but is this requirement documented?]

## Dependencies & Assumptions

- [ ] CHK055 Is the assumption documented that GOWA always sends a timestamp in the webhook payload (if not, replay protection would need a different anchor)? [Assumption, Spec §FR-005, research R2]
- [ ] CHK056 Is the assumption documented that the `X-Hub-Signature-256` header format matches Meta's `sha256={hex}` format (GOWA may use a different header name or format)? [Assumption, contracts/gowa-webhook-api.md, Gap]
- [ ] CHK057 Is the dependency on Redis documented for the re-download cooldown (what happens if Redis is unavailable — does re-download fail-open or fail-closed)? [Dependency, Spec §FR-014, Gap]
- [ ] CHK058 Is the dependency on the existing `contacts:export` permission being already seeded and mapped to manager documented (so the media ZIP gate works without additional catalog changes)? [Dependency, data-model.md, research R6]
- [ ] CHK059 Is the assumption documented that the `Organizations []string` field on `GOWAInstance` defaults to "all orgs" when empty (backward compatibility for existing deployments)? [Assumption, research R7, data-model.md]

## Ambiguities & Conflicts

- [ ] CHK060 Is there a potential conflict between FR-002 (reject if account has no secret) and FR-017 (auto-generate if no secret) — specifically, does auto-generation happen BEFORE the first webhook (so FR-002 never fires for new accounts), or can a webhook arrive in the window between account creation and secret backfill? [Conflict, Spec §FR-002/017]
- [ ] CHK061 Is there ambiguity about whether the `getGowaAccountByDeviceID` fallback path (outbound calls to all tenants) should be removed entirely, moved after HMAC, or rate-limited — the research (R4) discusses it but the spec doesn't state a clear requirement? [Ambiguity, research R4, Spec §FR-003]
- [ ] CHK062 Is there ambiguity about whether the `devices:read` permission for `GowaInstances` should also restrict the response to only instances the caller's org can use (vs. returning all instances but denying access at the permission level)? [Ambiguity, Spec §FR-008/009, contracts/gowa-device-api.md]
- [ ] CHK063 Is there ambiguity about whether the `updateMessageStatus` org-scoping (T023) applies to the Meta webhook path as well as GOWA — and if so, is the Meta path's account resolution guaranteed to provide an orgID? [Ambiguity, research R8, Gap]
- [ ] CHK064 Is there ambiguity about whether the media re-download cooldown is per-user or per-message (the spec says "per item" but doesn't clarify if user A's re-download blocks user B's re-download of the same message)? [Ambiguity, Spec §FR-014, contracts/media-api.md]

## Notes

- Check items off as completed: `[x]`
- Add comments or findings inline
- Items reference spec sections `[Spec §FR-X]`, contracts, research (R#), or mark `[Gap]` where a requirement is missing
- This checklist tests the *requirements documentation*, NOT the implementation
- Items marked `[Gap]` indicate a requirement that should be added to the spec/contracts before implementation
- Items marked `[Ambiguity]` or `[Conflict]` indicate requirements that need clarification
- The existing `requirements.md` checklist (from `/speckit.specify`) validates general spec quality; this `security.md` checklist focuses specifically on security-requirement completeness
