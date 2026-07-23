# Requirements Quality Checklist: Chat Status, Claim & Collaboration

**Purpose**: Validate that the spec's requirements are complete, clear, consistent, and measurable — NOT testing the implementation, testing the documentation quality  
**Created**: 2026-07-12  
**Feature**: [spec.md](../spec.md) | [plan.md](../plan.md) | [tasks.md](../tasks.md)  
**Scope**: All requirement dimensions (functional, security, UX, edge cases, non-functional)

---

## Requirement Completeness

- [ ] CHK001 Are all three chat lifecycle states (pending, open, closed) explicitly defined with entry/exit conditions in the spec? [Completeness, Spec §FR-001]
- [ ] CHK002 Is the complete list of actions that reset the inactivity timer documented (incoming msg, outgoing msg, claim, collaboration action)? [Completeness, Spec §FR-017]
- [ ] CHK003 Are requirements specified for what happens to collaborators when a conversation auto-reverts to pending? [Completeness, Spec §FR-017, Clarifications]
- [ ] CHK004 Is the behavior defined for a customer who sends a new message to a conversation that was auto-reverted to pending? [Gap, Spec §FR-002]
- [ ] CHK005 Are requirements defined for what the customer experiences during all lifecycle transitions (do they receive any notification)? [Gap]
- [ ] CHK006 Is the maximum number of collaborators per conversation specified? [Gap, Spec §FR-007]
- [ ] CHK007 Are the exact system message text templates defined for all lifecycle events (claim, join, leave, remove, auto-revert, close)? [Completeness, Spec §FR-013]
- [ ] CHK008 Is it specified whether a closed conversation can receive incoming messages from the customer (and what happens)? [Gap, Spec §FR-001]

## Requirement Clarity

- [ ] CHK009 Is "inactivity" precisely defined — does it mean no messages, or no UI interactions, or no API calls? [Clarity, Spec §FR-017]
- [ ] CHK010 Is the term "collaborator" vs "owner" vs "agent" consistently used and defined in a glossary? [Clarity]
- [ ] CHK011 Is the exact error message text specified for each rejection scenario (409 already_assigned, 409 chat_closed, 403 forbidden)? [Clarity, Spec §FR-005]
- [ ] CHK012 Is the meaning of "last participant" unambiguous — does it mean only the owner, or owner + all collaborators? [Clarity, Spec §FR-008, Clarifications]
- [ ] CHK013 Are the WebSocket event payload field names documented for all 3 new event types? [Clarity, Contracts §api.md]
- [ ] CHK014 Is "silently reverts" precisely scoped — does the system message appear before, during, or after the revert? [Clarity, Clarifications Session 2026-07-12]

## Requirement Consistency

- [ ] CHK015 Do the permission rules in FR-009/FR-010 align with the acceptance scenarios in US4? [Consistency, Spec §FR-009 vs §US4]
- [ ] CHK016 Is the claim idempotency rule (FR-006) consistent with the acceptance scenario 4 in US2? [Consistency, Spec §FR-006 vs §US2]
- [ ] CHK017 Does the privacy guard access logic (FR-003) align with the GetMessages acceptance scenario in US1? [Consistency, Spec §FR-003 vs §US1]
- [ ] CHK018 Is the "owner leaving closes conversation" rule (Clarifications Q1) consistent with the LeaveChat acceptance scenario (US3 scenario 4)? [Consistency, Clarifications vs §US3]
- [ ] CHK019 Do the role permission defaults (FR-010) match the clarified answer that managers can remove collaborators? [Consistency, Spec §FR-010 vs Clarifications Q1]
- [ ] CHK020 Is the auto-revert behavior (FR-017) consistent with the "pending never auto-closes" rule (FR-016)? [Consistency, Spec §FR-016 vs §FR-017]

## Acceptance Criteria Quality

- [ ] CHK021 Can SC-003 ("zero unauthorized message reads") be objectively verified — what counts as a "read"? [Measurability, Spec §SC-003]
- [ ] CHK022 Is SC-002 ("under 3 clicks") measurable — are the 3 clicks explicitly enumerated? [Measurability, Spec §SC-002]
- [ ] CHK023 Can SC-004 ("real-time under 2 seconds") be tested without implementation — is the measurement method defined? [Measurability, Spec §SC-004]
- [ ] CHK024 Is SC-007 ("concurrent claim — exactly one succeeds") testable with a specific test procedure? [Measurability, Spec §SC-007]
- [ ] CHK025 Are success criteria defined for the auto-revert worker (e.g., "95% of expired conversations revert within 5 minutes of timeout")? [Gap, Spec §SC]

## Scenario Coverage

- [ ] CHK026 Are recovery flow requirements defined for when the auto-revert worker fails or crashes mid-revert? [Gap, Recovery Flow]
- [ ] CHK027 Are requirements defined for a collaborator who is in the middle of typing a reply when auto-revert fires? [Gap, Exception Flow]
- [ ] CHK028 Is the scenario defined where a manager removes the last collaborator AND the owner in the same operation? [Gap, Edge Case]
- [ ] CHK029 Are requirements specified for when a user's role is changed WHILE they are an active collaborator (not just permission revoked)? [Coverage, Spec §US3 scenario 6]
- [ ] CHK030 Is the alternate flow defined for claiming via the existing AssignContact endpoint vs the new ClaimChat endpoint — do both set status to open? [Coverage, Spec §FR-012]
- [ ] CHK031 Are requirements defined for contacts that are BOTH group chats AND pending (do group messages trigger pending)? [Coverage, Gap]

## Edge Case Coverage

- [ ] CHK032 Is the behavior defined when a contact has `chat_status = "pending"` but `assigned_user_id` is somehow non-null (data inconsistency)? [Edge Case, Gap]
- [ ] CHK033 Is the behavior defined when a collaborator's user account is deleted/deactivated while they are still in the collaborators list? [Edge Case, Gap]
- [ ] CHK034 Are requirements defined for organizations with `chat_inactivity_timeout_hours` set to 0 (disabled)? [Edge Case, Spec §FR-017]
- [ ] CHK035 Is the behavior defined for extremely large collaborator lists (e.g., 50+ collaborators) and is there a cap? [Edge Case, Gap]
- [ ] CHK036 Is the behavior defined when two managers simultaneously remove different collaborators from the same conversation? [Edge Case, Concurrency]
- [ ] CHK037 Are requirements defined for WebSocket delivery failure during a claim (does the claim still succeed server-side)? [Edge Case, Spec §Edge Cases]

## Non-Functional Requirements

- [ ] CHK038 Are performance requirements specified for the auto-revert worker query at scale (e.g., 10,000 open conversations)? [Gap, Performance]
- [ ] CHK039 Are observability requirements defined — should claims, joins, leaves, and auto-reverts be logged for audit? [Gap, Spec §FR-013 vs Constitution Principle 17]
- [ ] CHK040 Is the WebSocket broadcast scope specified — does `BroadcastToOrg` send to ALL users or just involved parties (potential information leak of customer names)? [Security, Spec §FR-004]
- [ ] CHK041 Are rate limiting requirements defined for the claim/join/leave endpoints to prevent abuse? [Gap, Security]
- [ ] CHK042 Are accessibility requirements (ARIA labels, keyboard navigation) specified for the claim screen, join screen, and collaborators bar? [Gap, Accessibility]
- [ ] CHK043 Are RTL/layout requirements addressed for the collaborators bar in Arabic locale? [Gap, i18n/L10n]

## Dependencies & Assumptions

- [ ] CHK044 Is the assumption that "metadata defaults to {}" validated for all pre-existing contacts in production databases? [Assumption, Spec §Assumptions]
- [ ] CHK045 Is the dependency on `processIncomingMessageFull` as the single convergence point documented and validated for both Meta and GOWA providers? [Dependency, Spec §Assumptions]
- [ ] CHK046 Is the assumption that PermissionMatrix auto-renders new permissions validated — does the frontend cache the permission list? [Assumption, Spec §Assumptions]
- [ ] CHK047 Is the dependency on `a.WSHub != nil` null-checking documented as a requirement (Constitution Principle 9)? [Dependency]

## Ambiguities & Conflicts

- [ ] CHK048 Does the spec resolve whether a user with `contacts:read` permission sees pending conversations in their sidebar (they bypass the privacy guard, but do they see the claim button)? [Ambiguity, Spec §FR-003]
- [ ] CHK049 Is there a conflict between "agents cannot remove each other" (Clarification Q1) and the existing `AssignContact` endpoint which agents with `contacts:write` can call to reassign? [Conflict, Spec §FR-008 vs §FR-012]
- [ ] CHK050 Is it ambiguous whether the auto-revert system message is visible to the customer on WhatsApp or only in the whatomate UI? [Ambiguity, Spec §FR-017]

---

## Notes

- Items marked `[Gap]` indicate requirements that appear to be missing entirely from the spec
- Items marked `[Ambiguity]` indicate requirements that exist but could be interpreted multiple ways
- Items marked `[Conflict]` indicate requirements that may contradict each other
- Items marked `[Edge Case]` indicate boundary scenarios not addressed in the spec
- Address high-priority items (CHK004, CHK006, CHK008, CHK028, CHK040, CHK049) before implementation to reduce rework risk
- Low-priority items (CHK035, CHK036, CHK043) can be deferred to post-MVP
