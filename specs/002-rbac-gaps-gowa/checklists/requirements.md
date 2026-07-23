# Specification Quality Checklist: Close RBAC / User-Role Gaps in GOWA + Media Features

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-07-12
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

### Validation run 1 (2026-07-12) — PASS

**Content Quality**: PASS. The spec describes WHAT must be enforced (permissions, signature verification, tenant isolation) and WHY (the gaps), without prescribing HOW (no Go/fastglue/gorm/HMAC-SHA256 references). The single technical-sounding term — "cryptographic signature" (FR-001) — is used at a stakeholder level of abstraction (analogous to "password" or "2FA code" in a business spec) and does not dictate the algorithm. Permission names like "account-management write" are framed as user-facing capabilities, matching the existing project spec convention (`001-chat-claim-collaboration` references `chat.assign:write`).

**Requirement Completeness**: PASS. All 17 functional requirements are testable and map to acceptance scenarios. Success criteria are measurable (100% rejection, zero writes, zero controls visible) and technology-agnostic. Six edge cases identified. Scope is bounded to the 27 review findings; out-of-scope items (the separate MetaAI repo, non-RBAC concerns) are excluded.

**Feature Readiness**: PASS. Every FR has a corresponding acceptance scenario. Six user stories cover the primary flows (webhook auth, device RBAC, catalog, frontend, media, tests) and are independently testable per the template's MVP guidance.

### Assumptions documented in the spec
- Device-management permission is a distinct resource (not overloaded onto accounts) so least-privilege is expressible (Story 3).
- The instance-list endpoint exposes topology and should be admin/manager-only, not agent-visible (Story 2, acceptance 4).
- The media export/re-download policy confirmation (Story 5) is framed as a decision to confirm; a reasonable default (gate export behind an export permission, throttle re-download) is provided and documented in the Edge Cases.
- Replay protection uses a freshness window; the exact window is an implementation detail (Edge Cases covers the expiry behavior).

### Open decisions surfaced (no NEEDS CLARIFICATION markers — informed defaults provided)
The spec uses informed defaults rather than blocking clarifications for: (1) the media-export access policy — default "export permission required for bulk ZIP, read for single"; (2) whether a secretless GOWA account is blocked at creation vs. auto-generated — default "blocked/auto-generated so none is left unprotected" (FR-017); (3) device-management as a distinct permission vs. reusing accounts — default "distinct" (Story 3). These are documented as assumptions and edge cases, consistent with the skill's "make informed guesses" guidance.

### Clarification session 2026-07-12 — 5 questions resolved
All three previously-defaulted decisions were confirmed by the user, plus two additional decisions resolved:
1. **Media access policy** → Bulk ZIP requires `contacts:export`; re-download stays at `contacts:read` + cooldown. (Updated Story 5, FR-013.)
2. **Device permission resource** → Distinct `devices` resource (`devices:read` + `devices:write`), seeded, mapped to admin+manager. (Confirmed Story 3; updated Story 2/4 acceptance scenarios, FR-006/007/008/012 to name `devices:*` explicitly.)
3. **Webhook secret handling** → Auto-generate on create/update; backfill existing secretless accounts. (Refined FR-017; resolved edge case.)
4. **Replay window** → 5 minutes. (Refined FR-005; updated Story 6 test list; resolved edge case.)
5. **Account type classification** → Explicit provider-type field (GOWA/Meta) set at creation. (Added FR-018; updated Key Entities.)

Checklist re-validated after clarification: all items still PASS. No contradictions introduced.
