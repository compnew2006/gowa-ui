# Feature Specification: Close RBAC / User-Role Gaps in GOWA + Media Features

**Feature Branch**: `002-rbac-gaps-gowa`
**Created**: 2026-07-12
**Status**: Draft
**Input**: User description: "/Users/noiemany/Downloads/whatomate_GOWA/New one/whatomate/USER_ROLES_REVIEW_7509281.md" — a 27-finding role/permission gap review of the GOWA provider, media burst/zip, and group-routing changes (commits `7509281a` → `829ecf70`).

---

## Clarifications

### Session 2026-07-12

- Q: What permission policy should govern bulk media ZIP download and provider media re-download? → A: Bulk ZIP requires `contacts:export`; re-download stays at `contacts:read`, with the per-item cooldown (FR-014) as the abuse control.
- Q: Should device management use a distinct permission resource or reuse accounts? → A: Distinct `devices` resource with `devices:read` + `devices:write`, seeded by default; mapped to admin+manager (agent gets neither). Device handlers use `devices:write` for pair/QR/provision and `devices:read` for status/instances.
- Q: How should the system ensure no GOWA account is left without a webhook secret? → A: Auto-generate a secret server-side on account create/update when the caller doesn't supply one; backfill existing secretless accounts so none is left unprotected. Callers never need to supply a secret manually.
- Q: What freshness window should replay-protection use? → A: 5 minutes — reject any GOWA webhook whose timestamp is older than 5 minutes (industry standard; matches Stripe/GitHub/Slack).
- Q: How does the system know an account is GOWA-type vs Meta-type? → A: Explicit provider-type field on the account, set at creation. GOWA-type accounts trigger webhook-secret auto-generation (FR-017) and route to the GOWA provider; Meta-type accounts use the existing App Secret flow.

---

## User Scenarios & Testing *(mandatory)*

<!--
  User stories are ordered by risk: the CRITICAL unauthenticated-tenant-contamination
  vector (webhook fail-open) is P1, followed by the authenticated privilege-escalation
  gaps (device handlers), then catalog/UX/test-coverage hardening.
  Each story is independently testable and independently deployable.
-->

### User Story 1 — Inbound GOWA Webhooks Are Authenticated and Tenant-Isolated (Priority: P1)

A malicious actor outside the organization should never be able to forge an inbound WhatsApp message and inject it into an organization's conversations, contact list, or chatbot auto-reply flow. Today, the GOWA inbound webhook accepts any request that either omits the signature header or targets an account without a webhook secret configured — meaning an unauthenticated attacker can post a fake "incoming message," cause the victim organization's chatbot to auto-reply from their real WhatsApp number, create phantom contacts, and mutate (revoke/edit/react) any message whose WhatsApp message ID they can guess.

This story closes that hole by making webhook verification **fail-closed**: a request with no signature, a tampered body, or an account with no secret is rejected before any data is written, and every downstream write is scoped to the single organization that legitimately owns the targeted device.

**Why this priority**: This is the only finding in the review that permits **unauthenticated** cross-tenant contamination — every other gap requires at least a logged-in session. It must be closed first because it is independently exploitable and its blast radius spans every organization on the instance.

**Independent Test**: Send a POST to the GOWA webhook endpoint with a valid-looking payload but (a) no signature header, (b) a tampered body, and (c) a wrong secret. In all three cases the system returns an error and writes nothing. Then send a properly signed request for a known device and verify the message appears only in that device's organization.

**Acceptance Scenarios**:

1. **Given** an inbound GOWA webhook request that omits the signature header, **When** it arrives at the webhook endpoint, **Then** the system rejects it with an authentication error and performs no database writes, no contact creation, no message storage, and no chatbot reply.
2. **Given** an inbound request whose body has been altered after signing, **When** signature verification runs, **Then** the system rejects it and performs no downstream processing.
3. **Given** a GOWA account that has no webhook secret configured, **When** any inbound webhook for that account arrives (even one that would otherwise be valid), **Then** the system rejects it rather than processing it unsigned, and logs an alert that the account lacks a secret.
4. **Given** a legitimately signed webhook for device D owned by organization A, **When** it is processed, **Then** every resulting write (contact, message, status update, reaction, revocation, edit) is scoped exclusively to organization A — no write can touch a message, contact, or broadcast belonging to organization B.
5. **Given** a webhook carrying a reaction/revocation/edit/ack referencing a message ID, **When** the system applies the update, **Then** it verifies the referenced message belongs to the same organization as the resolved device before mutating it; a mismatched reference is ignored.
6. **Given** the same valid webhook delivered twice, **When** the system processes the second delivery, **Then** idempotent events are deduplicated and non-idempotent events (connection-status changes, revocations) are protected by a freshness/replay window so they cannot be replayed indefinitely.

---

### User Story 2 — Device Management Is Restricted to Authorized Roles (Priority: P1)

An administrator or manager configures a new GOWA WhatsApp number by generating a pairing QR code, entering a phone-pairing code, checking the device connection status, and provisioning a new device on a GOWA instance. These are privileged infrastructure actions that emit credentials (including the webhook secret). Today, every one of these actions is reachable by **any authenticated user regardless of role** — including a lowest-privilege agent — because the handlers do not check any permission. An agent can pair a device, and — worse — can call the create-device endpoint to provision a device on a GOWA instance belonging to **any organization**, because instance selection ignores the caller's organization.

This story ensures device pairing, provisioning, and status checks enforce the new dedicated device-management permission (`devices:write` for pair/QR/provision, `devices:read` for status/instances), and that device provisioning is scoped to the caller's own organization.

**Why this priority**: Though it requires authentication (unlike Story 1), it is a direct privilege-escalation: an agent performing admin-only infrastructure actions and obtaining credentials. It shares P1 because it is the second independently-exploitable critical vector.

**Independent Test**: Log in as an agent (lowest privilege) and attempt to call the device-pair, QR, status, instance-list, and create-device endpoints. All return "insufficient permissions." Log in as a manager (holding `devices:read` + `devices:write`) and verify all five succeed for accounts in their own organization. Attempt create-device targeting another organization's instance and verify it is refused.

**Acceptance Scenarios**:

1. **Given** an authenticated agent (who lacks `devices:write`), **When** they request a pairing QR code, submit a pair code, or provision a new device, **Then** each request is denied with an "insufficient permissions" error and no device action is taken on the GOWA provider.
2. **Given** an authenticated manager (who holds `devices:write`), **When** they request a QR code or submit a pair code for an account in their own organization, **Then** the request succeeds.
3. **Given** an authenticated manager in organization A, **When** they attempt to provision a device against a GOWA instance configured for organization B, **Then** the request is refused — instance selection is scoped to the caller's organization.
4. **Given** an authenticated agent (who lacks `devices:read`), **When** they request the list of GOWA instances, **Then** the request is denied — the instance list (which exposes internal service topology) is visible only to roles with `devices:read`.
5. **Given** an authenticated user with `devices:read`, **When** they request that account's GOWA connection status, **Then** the status is returned; a user without `devices:read` is denied.
6. **Given** a device-provisioning request that succeeds, **When** the webhook secret is returned to the caller, **Then** it is returned only to the authorized manager/admin (holding `devices:write`) who initiated provisioning, never exposed to lower-privilege roles.

---

### User Story 3 — The Permission Catalog Knows About Device Management (Priority: P2)

An administrator manages roles and permissions from the role-settings page. When the GOWA device-management feature was added, no corresponding permission entries were created in the catalog. This means even if the handlers (Story 2) are fixed to check a "device management" permission, that permission does not exist in the seeded list, so the check would fail for everyone — or, if the handlers reuse the existing account permission, the feature cannot express "can manage accounts but not provision devices" as a distinct privilege.

This story adds device-management as a first-class permission in the catalog, seeded by default and mapped to the standard system roles, so administrators can grant or revoke it independently of account CRUD — mirroring how every other feature (chat, contacts, campaigns, IVR) has its own permission group.

**Why this priority**: It is the enabling prerequisite for Story 2's permission checks to be meaningful and granular. It is P2 rather than P1 because, as a stopgap, the handlers can reuse the existing account-management permission — but the catalog gap must be closed for least-privilege to be expressible.

**Independent Test**: As an admin, open the role-settings page and verify a "Device Management" permission group appears with read and write entries. Create a custom role with device-management write but not account delete, assign a user, and verify that user can provision devices but cannot delete accounts.

**Acceptance Scenarios**:

1. **Given** the system has been updated, **When** an admin opens the role-settings permission matrix, **Then** a "Device Management" group is visible with at least "view device status" (read) and "pair and provision devices" (write) entries, alongside the existing account permissions.
2. **Given** the default "agent" system role, **When** the system is deployed, **Then** the agent role does **not** include device-management write (agents cannot pair/provision) and does not include device-management read (agents do not see instance topology).
3. **Given** the default "manager" and "admin" system roles, **When** deployed, **Then** both include device-management read and write.
4. **Given** an admin creating a custom role, **When** they grant device-management write without granting account-management delete, **Then** a user assigned that role can provision devices but cannot delete accounts — the two privileges are independently controllable.
5. **Given** a freshly initialized database, **When** the default permissions are seeded, **Then** the device-management permissions are present without requiring manual migration steps.

---

### User Story 4 — The Frontend Hides Device-Management Actions From Unauthorized Users (Priority: P2)

An agent viewing a GOWA account's detail page should not see a "Connect Device" button or a device-provisioning form, because they are not permitted to perform those actions. Today the button is shown to anyone who can open the account detail page (anyone with account-read), which means an agent sees a button that — until Story 2 is fixed — actually works, and even afterward would just produce a confusing "forbidden" error on click.

This story makes the frontend consistent with the backend permission gating: action buttons that require a permission the viewer lacks are hidden or disabled, matching the pattern the rest of the account-detail page already uses for Save and Delete. The "Connect Device" button and pair/provision controls require `devices:write`; the instance/status panel requires `devices:read`.

**Why this priority**: It is the UX counterpart to Story 2. It is P2 because the backend gate (Story 2) is the actual security boundary; the frontend gate prevents confusion and defense-in-depth, but is not the primary control.

**Independent Test**: Log in as an agent, open a GOWA account's detail page, and verify no "Connect Device" button, no pair-code form, and no provisioning UI is visible. Log in as a manager and verify all are visible and functional.

**Acceptance Scenarios**:

1. **Given** an agent viewing a GOWA account's detail page, **When** the page renders, **Then** the "Connect Device" button, the pair-code input, the device-status panel, and the instance/provisioning controls are hidden (the agent lacks both `devices:read` and `devices:write`).
2. **Given** a manager viewing the same account, **When** the page renders, **Then** all device-management controls are visible and enabled (the manager holds `devices:read` and `devices:write`).
3. **Given** the account-detail page's existing Save and Delete controls, **When** rendered for any role, **Then** they continue to follow their existing permission gating (this story does not regress them).
4. **Given** a user whose device-management permission is revoked while viewing the page, **When** they next load or interact with the page, **Then** the controls are hidden without requiring a full re-login (permission changes take effect promptly).

---

### User Story 5 — Media Export and Re-download Are Permission-Aware and Abuse-Resistant (Priority: P2)

A user viewing a chat can select multiple media items and download them as a ZIP archive, or re-trigger a provider download for a broken media item. Today these affordances are shown to every chat user and the backend gates them on the broad "view contacts" permission, which means any agent can bulk-download all media in a conversation they can see, and can repeatedly force the system to re-fetch media from the upstream WhatsApp provider — a costly, rate-limited operation — with no cooldown.

This story applies a tiered access policy: bulk ZIP download requires the existing export permission (since it is a bulk data export), while single-item re-download remains available to anyone who can view the conversation (gated on the read permission) but is protected from abuse by a per-item cooldown. The frontend hides the bulk-download controls from users lacking the export permission.

**Why this priority**: It is a defense-in-depth and abuse-mitigation story. The review confirmed there is no IDOR (media is org-scoped correctly), so it is not a data-leak critical — but bulk export and unbounded provider re-fetch are real abuse surfaces. P2 reflects "important but not unauthenticated-critical."

**Independent Test**: As an agent, open a chat and verify whether the bulk-download and re-download controls appear (per the confirmed policy). Trigger a re-download twice rapidly and verify the second is rate-limited. As a role without the export permission, verify the bulk-download control is hidden.

**Acceptance Scenarios**:

1. **Given** a user without the export permission, **When** they open a chat, **Then** the "download as ZIP" and "download separately" controls are hidden.
2. **Given** a user with the export permission, **When** they request a ZIP of selected media, **Then** only media belonging to conversations their role permits them to see is included (the existing org and assignment scoping is preserved).
3. **Given** a user who can view the conversation (read permission), **When** a media item fails to load, **Then** the "retry download" control is visible and functional — re-download is gated on the read permission, not a separate privilege.
4. **Given** a user triggering a provider re-download for a media item, **When** they trigger it again for the same item within the cooldown window, **Then** the second request is throttled and the user is informed a re-download was recently performed.
5. **Given** a ZIP download request containing a very large number of items, **When** the system processes it, **Then** it enforces a maximum item count (already present) and a maximum total size guard to prevent memory exhaustion.

---

### User Story 6 — Security-Critical Paths Have Tests (Priority: P3)

Every permission gate and webhook-verification path added by the preceding stories must be covered by automated tests, so the gaps cannot silently regress. Today, the review found zero authorization tests for the device-provisioning, instance-list, webhook-handler, and media-redownload paths — only happy-path and input-validation tests exist. A future refactor could remove a `requireAuth` call or weaken the webhook guard and nothing would fail.

This story adds the missing tests: permission-denied (403) cases, unauthenticated (401) cases, cross-organization (IDOR) rejection cases, signature-rejection cases, and replay cases for the security-critical paths identified in the review.

**Why this priority**: Tests are essential for preventing regression but do not themselves close a live vulnerability, so they follow the fixes. P3 reflects "enabling/protective, not directly exploitable."

**Independent Test**: Run the test suite and verify new tests exist and pass for: agent denied on each device endpoint, cross-org device provisioning refused, webhook rejected with missing/tampered/wrong signature, webhook rejected when account has no secret, webhook rejected when older than 5 minutes (replay), cross-org message mutation ignored, and media redownload rate-limited.

**Acceptance Scenarios**:

1. **Given** the device-management endpoints, **When** the test suite runs, **Then** tests assert that an agent-role caller receives a 403 on pair/QR/status/instances/create-device, and a manager-role caller succeeds for their own org.
2. **Given** the device-provisioning endpoint, **When** a cross-organization provisioning attempt is tested, **Then** the test asserts it is refused.
3. **Given** the GOWA webhook handler, **When** the test suite runs, **Then** tests assert rejection for: missing signature header, tampered body, wrong secret, and an account with no secret configured.
4. **Given** a webhook carrying a reaction/revocation/edit referencing another organization's message, **When** processed, **Then** the test asserts the cross-org reference is ignored (no mutation).
5. **Given** the media re-download endpoint, **When** a test triggers two re-downloads within the cooldown, **Then** the test asserts the second is throttled.
6. **Given** the media ZIP endpoint, **When** a test requests more items than the maximum, **Then** the test asserts the request is rejected or truncated to the limit.

---

### Edge Cases

- What happens when a GOWA account is created via the API without a webhook secret — the system auto-generates one server-side (FR-017); no account is ever left webhook-unprotected.
- What happens when an admin revokes a manager's device-management permission while the manager has a pairing QR flow mid-session — does the in-flight request complete, or is it re-checked?
- What happens when a webhook arrives for a device ID that matches accounts in two organizations (e.g., a reused phone number) — the system must deterministically resolve to exactly one organization and never write to both.
- What happens when a replayed webhook arrives after the 5-minute replay window expires — it is silently dropped and the drop is logged for monitoring (FR-005).
- What happens when a user with export permission requests a ZIP, but some selected items belong to conversations they can no longer access (permission revoked mid-request) — are those items silently excluded?
- What happens when the permission catalog is seeded on a database that already has custom roles — are existing roles left untouched and only the new default permissions added?

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST reject any inbound GOWA webhook request that is not accompanied by a valid cryptographic signature matching the target account's configured secret.
- **FR-002**: System MUST reject any inbound GOWA webhook for an account that has no webhook secret configured, rather than processing it without verification.
- **FR-003**: System MUST scope every write triggered by an inbound webhook (contact creation, message storage, status update, reaction, revocation, edit, chatbot reply, WebSocket broadcast) to the single organization that owns the resolved device.
- **FR-004**: System MUST verify that a reaction/revocation/edit/ack target message belongs to the same organization as the resolved device before mutating it.
- **FR-005**: System MUST protect non-idempotent webhook events (connection-status, revocation, edit) against replay by rejecting any GOWA webhook whose timestamp is older than 5 minutes.
- **FR-006**: System MUST require the `devices:write` permission for device pairing (QR code and pair-code) and device provisioning actions.
- **FR-007**: System MUST require the `devices:read` permission for device connection-status queries.
- **FR-008**: System MUST restrict the GOWA instance list (which exposes internal service topology) to roles holding `devices:read`.
- **FR-009**: System MUST scope device-provisioning instance selection to the caller's organization; a caller must not be able to provision a device against an instance configured for another organization.
- **FR-010**: System MUST expose a dedicated device-management permission in the role permission catalog, distinct from account CRUD, with at least a read and a write action.
- **FR-011**: System MUST seed the device-management permissions by default on fresh database initialization, and map them to system roles: admin and manager receive read and write; agent receives neither.
- **FR-012**: Frontend MUST hide the "Connect Device" button, pair-code form, device-status panel, and provisioning controls from users lacking `devices:read` (for status/instances) or `devices:write` (for connect/pair/provision).
- **FR-013**: Frontend MUST hide bulk media-export controls (download-as-ZIP, download-separately) from users lacking the export permission. Frontend MUST keep the media re-download control visible to any user who can view the conversation (read permission), since re-download is gated on read, not a separate privilege.
- **FR-014**: System MUST enforce a cooldown on provider media re-download requests per item to prevent abuse of the upstream WhatsApp provider.
- **FR-015**: System MUST enforce a maximum total size on ZIP media archive generation in addition to the existing maximum item count.
- **FR-016**: System MUST include automated tests covering: permission-denied (403) on every device endpoint, cross-organization provisioning refusal, webhook signature rejection (missing/tampered/wrong/empty-secret), cross-organization message-mutation rejection, and media re-download throttle.
- **FR-017**: System MUST ensure every GOWA account has a webhook secret before it can accept inbound webhooks. When a GOWA account is created or updated and the caller does not supply a secret, the system MUST auto-generate one server-side. Existing GOWA accounts without a secret MUST be backfilled so no account is ever left unprotected. Callers are never required to supply a secret manually.
- **FR-018**: System MUST classify each WhatsApp account by an explicit provider-type field (GOWA or Meta) set at creation. GOWA-type accounts trigger webhook-secret auto-generation (FR-017) and route inbound/outbound traffic to the GOWA provider; Meta-type accounts use the existing App Secret verification and Meta provider flow. The provider type MUST NOT be inferred from which fields happen to be populated.

### Key Entities *(include if feature involves data)*

- **GOWA Device**: A WhatsApp device managed via the GOWA provider, owned by exactly one organization. Pairing (QR/pair-code), connection status, and provisioning are privileged operations. Has an associated webhook secret used to verify inbound webhooks.
- **WhatsApp Account**: An organization's configured WhatsApp number, classified by an explicit provider-type field as either GOWA or Meta. The type determines provider routing, webhook verification method, and whether a webhook secret is auto-generated.
- **Organization (Tenant)**: The isolation boundary. Every webhook-originated write, every device, and every message mutation must be scoped to exactly one organization.
- **Permission**: A granular (resource, action) pair grantable to roles. Device management is a new resource with read and write actions, seeded by default and mapped to system roles.
- **Role**: A named collection of permissions assigned to users (system roles: admin, manager, agent; plus custom roles). Device-management permissions are independently grantable per role.
- **Inbound Webhook**: A machine-to-machine request from the GOWA provider delivering a message/event. Must be cryptographically authenticated and replay-protected before any processing.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of inbound GOWA webhook requests lacking a valid signature are rejected with zero database writes — verifiable by sending forged requests and confirming no contacts, messages, or broadcasts are created.
- **SC-002**: An authenticated agent-role user receives an "insufficient permissions" response on 100% of device-pairing, device-provisioning, and instance-listing attempts — verifiable by attempting all five endpoints as an agent.
- **SC-003**: A user from organization A cannot provision a device against an organization-B instance — the cross-org attempt is refused 100% of the time.
- **SC-004**: Every security-critical path identified in the review has at least one automated test covering the rejection/forbidden case, and the full test suite passes — zero untested permission or signature paths remain.
- **SC-005**: An agent viewing a GOWA account detail page sees zero device-management controls (no Connect Device button, no pair form, no provisioning UI) — verifiable by visual inspection as the agent role.
- **SC-006**: A user triggering two provider media re-downloads for the same item within the cooldown window has the second request throttled — verifiable by rapid double-request.
- **SC-007**: All 27 findings enumerated in the source review are either resolved or explicitly accepted-with-justification; zero CRITICAL findings remain open.
- **SC-008**: A newly initialized system has the device-management permission group present in the role-settings matrix without any manual migration step.
