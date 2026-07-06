# Feature: Customer Agent Selection for WhatsMeow

## Overview

Customer Agent Selection lets a WhatsApp customer choose the employee, team, queue, or configured custom action they want after a configurable delay, while preserving the existing pending/open assignment lifecycle. The feature is implemented as an additive routing layer for WhatsMeow inbound conversations, using the existing contact assignment and agent transfer behavior instead of replacing it.

The primary user value is faster and clearer routing: customers can choose the right destination, admins can control which available agents appear, supervisors get an audit trail, and existing Whatomate daily operations continue without breaking current chat, claim, assignment, transfer, chatbot, or WhatsMeow logic.

## Goals

- Allow customers messaging via WhatsMeow to choose a specific visible agent, team, general queue, or custom final option.
- Send the selection menu only after a configurable X-minute delay when the chat is still pending and unassigned.
- Keep chats pending when the customer does not reply or chooses a configured non-assignment action.
- Let admins add, remove, rename, order, and hide agents from the customer-facing list without changing user accounts.
- Show only agents that are enabled for the menu and currently eligible, including availability and instance access checks.
- Add a reliable audit trail for menu delivery, customer choices, routing decisions, failures, timeouts, and assignment results.
- Preserve existing assignment, transfer, chat status, tenant, role, permission, and WhatsMeow behavior.

## Non-Goals

- Do not replace `AssignContact`, `ClaimChat`, `CreateAgentTransfer`, `AssignAgentTransfer`, or `chatAssignmentUpdates`.
- Do not change the meaning of `Contact.Status`, `Contact.AssignedUserID`, `User.IsAvailable`, teams, or transfers.
- Do not expose every employee automatically to customers.
- Do not require WhatsApp interactive list support for the first release.
- Do not auto-assign a pending chat unless the customer selects an assignment-capable option or a configured fallback explicitly does so.

## Current System Context

- Contacts are assigned through `Contact.AssignedUserID`.
- Pending chats are represented by unassigned contacts with pending status.
- Existing assignment logic uses `chatAssignmentUpdates` to keep contact assignment and status transitions consistent.
- Existing transfers use `AgentTransfer`, `Team`, and `TeamMember`.
- Users already include `IsActive` and `IsAvailable`.
- WhatsMeow inbound processing saves incoming messages before chatbot processing.
- Existing chat system messages are visible in the thread, but a separate audit model is required for operational traceability.

## Architectural Impact Assessment

- Areas affected: WhatsMeow inbound routing, chatbot processing entry point, contact assignment, transfer creation, admin settings, audit reporting, frontend settings.
- New extension points: customer routing service, delayed prompt scheduler, selection session processor, audit writer, settings API, audit API.
- Risks: duplicate assignment from repeated replies, showing unavailable agents, breaking pending behavior, coupling the new feature to chatbot internals, noisy system messages, and inconsistent instance access checks.
- Proposed mitigations: add new tables and services, keep one small inbound hook, reuse existing assignment helpers, enforce transactions for session state changes, gate everything behind settings, and log every state transition to audit.
- Refactor needed? No for first release. Only minimal wiring is needed in existing entry points; the feature logic should live in new files.

## Blast Radius Analysis

- Target: new Customer Agent Selection feature module.
- Directly affected:
  - new models under `internal/models`
  - new handlers under `internal/handlers`
  - new routing service under `internal/handlers` or a future service package matching project conventions
  - new frontend settings/audit views
- Existing wire-in points:
  - `cmd/whatomate/main.go` for routes
  - database migration/AutoMigrate setup for new models
  - WhatsMeow inbound/chatbot processor hook after inbound message save and before normal chatbot routing
- Indirectly affected:
  - contact list pending/open views
  - agent transfer queue
  - websocket lifecycle updates
  - chatbot flow execution
- Risk level: MEDIUM.
- Safe to proceed: YES, if the first implementation is feature-flagged, additive, and reuses existing assignment/transfer functions.

## Functional Requirements

### FR-001: Feature Enablement

Where Customer Agent Selection is disabled for an organization or WhatsMeow instance, the system shall not create selection sessions, schedule delayed prompts, send customer selection menus, or intercept customer replies.

### FR-002: WhatsMeow Scope

Where Customer Agent Selection is enabled, when an inbound message is received from a WhatsMeow account, the system shall evaluate the routing feature only for that WhatsMeow account and shall leave Meta provider behavior unchanged.

Where an admin selects one or more allowed WhatsMeow instances in Customer Agent Selection settings, the system shall evaluate the routing feature only for messages whose contact belongs to one of those instances. An empty allowed-instance list shall preserve the previous global behavior and apply to all WhatsMeow instances.

### FR-003: Delayed Prompt Scheduling

While a WhatsMeow inbound message belongs to an unassigned pending chat, when the feature is enabled and the trigger rules match, the system shall create or reuse a selection session with status `waiting_delay` and `prompt_due_at` equal to the configured delay.

### FR-004: Preserve Pending During Delay

While a selection session is in `waiting_delay`, the system shall keep the contact unassigned and pending unless an existing user action or existing system process assigns, claims, closes, or reopens the chat.

### FR-005: Cancel Prompt When Chat Is Assigned

While a selection session is in `waiting_delay`, when the contact becomes assigned before `prompt_due_at`, the system shall cancel the session and shall not send the customer selection menu.

### FR-006: Send Prompt After X Minutes

While a selection session is in `waiting_delay`, when `prompt_due_at` is reached and the contact is still pending and unassigned, the system shall send the configured customer selection menu through WhatsMeow.

### FR-007: Do Not Prompt Active Transfers

While a contact has an active agent transfer, when a delayed prompt becomes due, the system shall skip menu sending and record an audit event explaining that an active transfer already exists.

### FR-008: Dynamic Agent Visibility

When building the menu, the system shall include only agents that are enabled in the customer-facing participant list, active, available when `show_only_when_available` is true, within configured capacity, members of the organization, and allowed to access the contact's WhatsMeow instance.

### FR-009: Admin Managed Agent List

While an admin has routing settings permission, when the admin adds, removes, disables, renames, or reorders an agent in the customer-facing participant list, the system shall update future menus without changing the user's core account identity.

### FR-010: Team and Queue Options

When an admin configures team or queue options, the system shall include those options in the menu according to their enabled status, sort order, and instance constraints.

### FR-011: Custom Final Option

When an admin configures a custom final option such as `سأذهب للفرع للطباعة`, the system shall render that option at the end of the menu after all enabled agent, team, and queue options.

### FR-012: Custom Final Option Text

While an admin has routing settings permission, when the admin changes the custom final option text or response text, the system shall use the new text in future menus and responses without requiring deployment.

### FR-013: Menu Numbering

When the system renders a WhatsMeow text menu, the system shall number visible options sequentially based on the rendered menu, not based on database IDs or hidden options.

### FR-014: Customer Selection Parsing

While a selection session is in `menu_sent`, when the customer replies with a valid option number or supported alias, the system shall map the reply to the rendered option snapshot for that session.

### FR-015: Agent Selection Assignment

While a customer selects a valid agent option, when the selected agent is still eligible, the system shall assign the contact to that agent using the existing assignment lifecycle behavior and shall mark the session `selected`.

### FR-016: Agent Becomes Unavailable

While a customer selects an agent option, when the selected agent is no longer eligible, the system shall not assign the contact and shall respond using the configured unavailable-agent fallback.

### FR-017: Team Selection Transfer

While a customer selects a team option, when the team is active and eligible, the system shall create or update an active agent transfer using source `customer_selection` and shall assign an agent only if the existing team assignment strategy chooses an eligible agent.

### FR-018: Queue Selection Transfer

While a customer selects a general queue option, when queue routing is enabled, the system shall create an unassigned active transfer with source `customer_selection`.

### FR-019: Custom Final Option Action

While a customer selects the custom final option, the system shall execute the configured action: `send_only`, `keep_pending`, `close_chat`, or `assign_to_team`.

### FR-020: Keep Pending Custom Action

While the custom final option action is `send_only` or `keep_pending`, when the customer selects it, the system shall leave the contact unassigned and pending.

### FR-021: No Reply Timeout

While a selection session is in `menu_sent`, when the configured selection timeout expires without a valid customer reply, the system shall mark the session `timeout`, record an audit event, and leave the contact pending and unassigned.

### FR-022: Invalid Reply Handling

While a selection session is in `menu_sent`, when the customer sends an invalid reply, the system shall increment the invalid attempt count, record an audit event, and send the configured invalid-reply message until the maximum attempt limit is reached.

### FR-023: Max Invalid Attempts

While invalid replies reach the configured maximum attempt limit, the system shall expire or cancel the session according to settings and shall leave the contact pending unless a configured fallback explicitly routes it.

### FR-024: Session Idempotency

When the same inbound WhatsApp message is processed more than once, the system shall not create duplicate assignments, transfers, responses, or audit events.

### FR-025: Concurrent Reply Safety

When two valid replies for the same active session are processed concurrently, the system shall accept only the first committed reply and shall ignore or audit the later reply as already handled.

### FR-026: Existing Manual Assignment Wins

While a selection session is active, when a staff member manually claims or assigns the contact before the customer selection is committed, the system shall stop automated routing for that session and shall not overwrite the staff assignment.

### FR-027: Audit Events

When any meaningful Customer Agent Selection state transition occurs, the system shall write an audit event containing organization, contact, account or instance, session, event type, selected option when applicable, previous assignee, new assignee, transfer ID when applicable, message references when available, reason, metadata, and timestamp.

### FR-028: System Chat Messages

When the feature assigns or routes a contact, the system shall add an existing-style system chat message for agent visibility without using that message as the sole audit source.

### FR-029: Audit Search API

While an authenticated user has audit access permission, when they query Customer Agent Selection audit events, the system shall support filtering by organization scope, date range, contact, agent, team, event type, session status, and WhatsMeow account or instance.

### FR-030: Tenant Isolation

When any Customer Agent Selection API, scheduler, session processor, or audit query runs, the system shall enforce organization scope and shall not expose or mutate another organization's settings, sessions, participants, contacts, agents, teams, transfers, or audit records.

### FR-031: Permission Enforcement

When a user manages routing settings or views audit records, the system shall require explicit permissions or an existing admin/super-admin permission path accepted by the project authorization model.

### FR-032: Websocket Updates

When Customer Agent Selection changes a contact assignment, transfer state, or chat lifecycle state, the system shall broadcast using existing websocket lifecycle or transfer update patterns.

### FR-033: Feature Flag Rollback

When an admin disables the feature, the system shall stop new sessions and prompts immediately, while existing sessions shall expire without assigning contacts unless already committed.

### FR-034: Settings Defaults

When a new organization or instance has no Customer Agent Selection settings, the system shall default to disabled, no visible agents, text menu mode, pending-preserving timeout behavior, and no automatic fallback assignment.

## Data Model Specification

### `agent_selection_settings`

Purpose: Stores organization and optional instance-level behavior.

Required fields:

- `id`
- `organization_id`
- `instance_id` nullable
- `allowed_instance_ids` JSONB array, empty means all WhatsMeow instances for the organization
- `enabled`
- `trigger_mode`: `first_pending_message`, `keyword`, `after_office_hours`, `chatbot_step`, `manual_test`
- `trigger_keywords` JSONB
- `prompt_delay_minutes`
- `selection_timeout_minutes`
- `max_invalid_attempts`
- `menu_header_text`
- `menu_footer_text`
- `invalid_reply_text`
- `timeout_response_text`
- `unavailable_agent_text`
- `custom_final_option_enabled`
- `custom_final_option_text`
- `custom_final_option_response`
- `custom_final_option_action`: `send_only`, `keep_pending`, `close_chat`, `assign_to_team`
- `custom_final_option_team_id` nullable
- `hide_unavailable_agents`
- `created_at`
- `updated_at`

### `agent_selection_participants`

Purpose: Controls which users are customer-visible without changing core user identity.

Required fields:

- `id`
- `organization_id`
- `settings_id`
- `user_id`
- `display_name`
- `description` nullable
- `is_enabled`
- `sort_order`
- `show_only_when_available`
- `max_open_chats` nullable
- `metadata` JSONB
- `created_at`
- `updated_at`

### `agent_selection_options`

Purpose: Stores non-agent options, including teams and queues.

Required fields:

- `id`
- `organization_id`
- `settings_id`
- `option_type`: `agent`, `team`, `queue`, `custom`
- `user_id` nullable
- `team_id` nullable
- `label`
- `description` nullable
- `is_enabled`
- `sort_order`
- `action`
- `metadata` JSONB
- `created_at`
- `updated_at`

### `agent_selection_sessions`

Purpose: Tracks delayed prompt and reply handling.

Required fields:

- `id`
- `organization_id`
- `contact_id`
- `instance_id` nullable
- `whatsapp_account`
- `status`: `waiting_delay`, `menu_sent`, `selected`, `timeout`, `cancelled`, `expired`, `error`
- `trigger_message_id` nullable
- `prompt_message_id` nullable
- `prompt_due_at`
- `menu_sent_at` nullable
- `expires_at` nullable
- `selected_option_id` nullable
- `selected_user_id` nullable
- `selected_team_id` nullable
- `transfer_id` nullable
- `invalid_attempts`
- `rendered_options_snapshot` JSONB
- `metadata` JSONB
- `created_at`
- `updated_at`

### `agent_selection_audit_events`

Purpose: Immutable operational audit trail.

Required fields:

- `id`
- `organization_id`
- `contact_id` nullable
- `session_id` nullable
- `instance_id` nullable
- `whatsapp_account` nullable
- `event_type`
- `actor_type`: `customer`, `system`, `admin`, `agent`
- `actor_user_id` nullable
- `selected_option_id` nullable
- `selected_agent_id` nullable
- `selected_team_id` nullable
- `previous_assigned_user_id` nullable
- `new_assigned_user_id` nullable
- `transfer_id` nullable
- `inbound_message_id` nullable
- `outbound_message_id` nullable
- `reason` nullable
- `metadata` JSONB
- `created_at`

## API Specification

### Admin Settings

- `GET /api/agent-selection/settings`
- `PUT /api/agent-selection/settings`
- `GET /api/agent-selection/participants`
- `POST /api/agent-selection/participants`
- `PUT /api/agent-selection/participants/{id}`
- `DELETE /api/agent-selection/participants/{id}`
- `GET /api/agent-selection/options`
- `POST /api/agent-selection/options`
- `PUT /api/agent-selection/options/{id}`
- `DELETE /api/agent-selection/options/{id}`
- `POST /api/agent-selection/preview`

### Audit and Operations

- `GET /api/agent-selection/audit`
- `GET /api/agent-selection/sessions`
- `POST /api/agent-selection/sessions/{id}/cancel`
- `POST /api/agent-selection/test-send`

All endpoints shall use existing fasthttp/fastglue handler patterns, existing auth context, existing tenant scope behavior, and project permission conventions.

## WhatsMeow Message Format

The first release shall use text menus.

Example:

```text
من فضلك اختر من تريد التواصل معه:

1. أحمد - الدعم الفني
2. سارة - المبيعات
3. فريق الحسابات

4. سأذهب للفرع للطباعة
```

Rules:

- The menu must be generated from the session's eligible option snapshot.
- Hidden or unavailable agents must not create number gaps.
- Customer replies are matched against the snapshot, not live reordered data.
- The custom final option, when enabled, must be rendered last.
- Arabic and English text must be configurable through settings.

## Scheduler and Processing Behavior

- The implementation may use an existing worker, queue, ticker, or job mechanism already present in the project.
- The scheduler shall process due `waiting_delay` sessions.
- The scheduler shall process expired `menu_sent` sessions.
- Processing must be idempotent.
- Session state updates must be transactional where assignment or transfer creation is involved.
- The scheduler must re-check contact state at execution time, not rely only on state captured at session creation.

## Audit Event Types

- `session_created`
- `delay_started`
- `prompt_skipped_assigned`
- `prompt_skipped_active_transfer`
- `menu_sent`
- `menu_send_failed`
- `valid_reply_received`
- `invalid_reply_received`
- `max_invalid_attempts_reached`
- `agent_selected`
- `agent_unavailable`
- `agent_assigned`
- `team_selected`
- `team_transfer_created`
- `queue_selected`
- `queue_transfer_created`
- `custom_option_selected`
- `custom_action_completed`
- `selection_timeout`
- `session_cancelled`
- `session_expired`
- `processing_error`

## Non-Functional Requirements

### Performance

- Menu preview and settings APIs should respond within 300ms p95 for normal tenant sizes.
- Due-session processing should handle at least 1,000 due sessions per minute per worker without duplicate assignment.
- Audit queries should be indexed by organization, created date, contact, session, event type, selected agent, and selected team.

### Reliability

- All assignment-producing operations must be idempotent.
- Session processing must tolerate worker restarts.
- A failed menu send must not assign the chat.
- A failed audit write must be treated as an operational error for the feature path and must be visible in logs.

### Security

- Settings and audit endpoints must require authenticated users.
- Admin operations must require routing settings permissions or equivalent admin capability.
- Audit reads must enforce organization scope and role permissions.
- Customer-facing menus must not leak disabled, inactive, unavailable, unauthorized, or cross-tenant users.
- Free-text admin settings must be sanitized in the frontend and safely encoded in API responses.

### Privacy

- Customer-visible agent names must come from `display_name`, not necessarily `User.FullName`.
- Agents must be hidden by default until explicitly enabled.
- Audit metadata must not store unnecessary secrets, tokens, or message credentials.

### Compatibility

- Backend code must use fastglue/fasthttp conventions.
- Handler tests must not use `net/http`, `http.Handler`, or `httptest`.
- Production build must continue embedding the Vue SPA normally.
- Meta provider behavior must remain unaffected unless separately enabled in a future release.

### Observability

- Every skipped prompt and failed routing decision must create an audit event with a reason.
- Logs should include session ID, organization ID, contact ID, and event type.
- Metrics are recommended for due sessions, prompts sent, selections, timeouts, invalid replies, unavailable selections, assignments, and failures.

## Acceptance Criteria

### AC-001: Delayed Menu Sent for Pending Chat

Given Customer Agent Selection is enabled with a 3-minute prompt delay
And a WhatsMeow customer sends a message to an unassigned pending chat
When 3 minutes pass and the chat is still pending and unassigned
Then the system sends the customer selection menu
And the chat remains pending
And an audit event `menu_sent` is recorded.

### AC-002: Prompt Is Not Sent After Manual Claim

Given a delayed prompt is scheduled for a pending chat
When an agent claims the chat before the prompt due time
Then the system cancels or skips the prompt
And no menu is sent
And the manual assignment remains unchanged
And an audit event `prompt_skipped_assigned` is recorded.

### AC-003: No Reply Keeps Pending

Given a customer selection menu was sent
When the customer does not reply before the configured timeout
Then the session is marked `timeout`
And the contact remains pending and unassigned
And an audit event `selection_timeout` is recorded.

### AC-004: Available Agent Appears in Menu

Given an agent is enabled as a participant
And the agent is active, available, within capacity, and can access the contact instance
When the menu is rendered
Then the agent appears in the menu with the configured display name.

### AC-005: Unavailable Agent Is Hidden

Given an agent is enabled as a participant
And `show_only_when_available` is true
And the agent is not available
When the menu is rendered
Then the agent does not appear in the menu
And the menu numbering has no gaps.

### AC-006: Admin Removes Agent from Menu

Given an agent is enabled in the customer-facing participant list
When an admin disables or removes that participant
Then future menus do not show that agent
And the user's core account remains active and unchanged.

### AC-007: Customer Selects Agent

Given a menu was sent with an eligible agent as option 1
When the customer replies `1`
Then the contact is assigned to that agent using existing assignment lifecycle logic
And the contact status becomes open according to existing behavior
And a system chat message is added
And audit events record the selection and assignment.

### AC-008: Selected Agent Becomes Unavailable

Given a menu was sent with an agent option
And the agent becomes unavailable after the menu was sent
When the customer selects that agent
Then the system does not assign the contact
And sends the configured unavailable-agent response
And records `agent_unavailable`
And leaves the contact pending unless fallback settings say otherwise.

### AC-009: Customer Selects Team

Given a menu was sent with a team option
When the customer selects that team
Then the system creates an active transfer with source `customer_selection`
And applies existing team assignment strategy if configured
And records transfer and audit events.

### AC-010: Customer Selects Custom Final Option

Given the custom final option text is `سأذهب للفرع للطباعة`
When the customer selects that option
Then the system sends the configured custom response
And executes the configured custom action
And records `custom_option_selected`
And keeps the chat pending when the action is `send_only` or `keep_pending`.

### AC-011: Invalid Reply

Given a menu was sent with options 1 through 4
When the customer replies with `9`
Then the system sends the configured invalid-reply response
And increments invalid attempts
And records `invalid_reply_received`
And does not assign the contact.

### AC-012: Duplicate Reply Is Idempotent

Given the customer reply was already processed successfully
When the same WhatsApp inbound message is processed again
Then the system does not create a duplicate assignment, transfer, response, or audit event.

### AC-013: Concurrent Reply Safety

Given a session is active
When two valid replies are processed at the same time
Then only one reply commits a routing decision
And the final contact assignment or transfer is consistent
And the later reply is ignored or audited as already handled.

### AC-014: Cross-Tenant Protection

Given an admin belongs to organization A
When they request settings, participants, sessions, or audit records for organization B
Then the system returns forbidden or empty scoped results according to existing tenant rules.

### AC-015: Feature Disabled Rollback

Given Customer Agent Selection is disabled
When new WhatsMeow messages arrive
Then the system does not create new sessions or send menus
And existing chat behavior remains unchanged.

## Error Handling

| Error Condition | HTTP Code | User/Admin Message | System Behavior |
| --- | --- | --- | --- |
| Invalid settings payload | 400 | `Invalid routing settings` | Do not save settings; return field errors. |
| Unauthorized settings access | 401 | `Authentication required` | Use existing auth behavior. |
| Forbidden settings mutation | 403 | `You do not have permission to manage routing settings` | Do not mutate settings. |
| Participant user not found | 404 | `Agent not found` | Do not create participant. |
| Participant user outside organization | 403 | `Agent is not available for this organization` | Do not create participant. |
| Duplicate participant | 409 | `Agent is already in this routing list` | Keep existing participant unchanged. |
| Team not found or inactive | 404/409 | `Team is not available` | Do not create or enable option. |
| Menu send failure | N/A | Optional customer retry later | Keep session error or retryable status; do not assign. |
| Invalid customer reply | N/A | Configured invalid reply text | Increment invalid attempts; keep pending. |
| Selected agent unavailable | N/A | Configured unavailable agent text | Keep pending unless fallback configured. |
| Active transfer already exists | N/A | Optional already-routing response | Skip new transfer; audit reason. |
| Audit query too broad | 400 | `Please narrow the date range` | Protect performance. |

## Implementation TODO

### Backend Models and Database

- [ ] Add `AgentSelectionSettings` model.
- [ ] Add `AgentSelectionParticipant` model.
- [ ] Add `AgentSelectionOption` model.
- [ ] Add `AgentSelectionSession` model.
- [ ] Add `AgentSelectionAuditEvent` model.
- [ ] Add source constant `customer_selection` for transfer source usage.
- [ ] Add database indexes for organization/date/session/contact/event/agent/team filters.
- [ ] Add additive migrations or AutoMigrate registration for the new models.

### Backend Services

- [ ] Implement settings resolver with organization and instance fallback.
- [ ] Implement eligible option builder.
- [ ] Implement menu renderer with Arabic/English configurable text.
- [ ] Implement delayed prompt scheduler.
- [ ] Implement selection session state machine.
- [ ] Implement inbound reply parser using the rendered options snapshot.
- [ ] Implement idempotency guard for inbound WhatsApp message IDs.
- [ ] Implement audit writer.
- [ ] Implement agent eligibility checks using active, available, participant settings, capacity, organization membership, and instance access.
- [ ] Implement team and queue transfer routing with source `customer_selection`.
- [ ] Implement custom final option actions.

### Backend Handlers and Routes

- [ ] Add settings CRUD endpoints.
- [ ] Add participant CRUD endpoints.
- [ ] Add team/queue/custom option CRUD endpoints.
- [ ] Add menu preview endpoint.
- [ ] Add audit search endpoint.
- [ ] Add active sessions endpoint.
- [ ] Add session cancel endpoint.
- [ ] Add test-send endpoint for admins.
- [ ] Register routes in `cmd/whatomate/main.go`.

### WhatsMeow Integration

- [ ] Add a small inbound hook after incoming message persistence and before normal chatbot handling.
- [ ] Only process active selection sessions or create delayed sessions when trigger rules match.
- [ ] Keep Meta provider unaffected.
- [ ] Use WhatsMeow text send path for first release.
- [ ] Re-check contact assignment and active transfer state before sending or committing routing.

### Frontend

- [ ] Add `Customer Routing` settings page.
- [ ] Add enable/disable toggle.
- [ ] Add prompt delay and timeout controls.
- [ ] Add menu header/footer/invalid/unavailable/timeout text fields.
- [ ] Add participant management UI for add/remove/enable/disable/reorder/display name.
- [ ] Add team/queue/custom final option configuration.
- [ ] Add live menu preview.
- [ ] Add audit view with filters.
- [ ] Add active sessions operational view or include it in audit/operations.

### Testing

- [ ] Unit test option rendering and numbering.
- [ ] Unit test agent eligibility filtering.
- [ ] Unit test reply parsing against snapshot.
- [ ] Unit test custom final option behavior.
- [ ] Integration test delayed prompt creation.
- [ ] Integration test due prompt sends only while pending.
- [ ] Integration test no-reply timeout leaves pending.
- [ ] Integration test selecting an available agent assigns through existing lifecycle.
- [ ] Integration test selected unavailable agent does not assign.
- [ ] Integration test team selection creates customer-selection transfer.
- [ ] Integration test duplicate inbound message idempotency.
- [ ] Integration test concurrent replies.
- [ ] Permission tests for settings and audit APIs.
- [ ] Frontend unit tests for settings forms.
- [ ] E2E test for admin configuration and menu preview.
- [ ] WhatsMeow smoke test in staging or local provider harness.

### Documentation and Rollout

- [ ] Document feature settings for admins.
- [ ] Document daily operations for agents and supervisors.
- [ ] Document audit event meanings.
- [ ] Add deployment note explaining feature flag rollback.
- [ ] Roll out disabled by default.
- [ ] Enable first for one WhatsMeow instance.
- [ ] Monitor audit events, failed sends, timeouts, and assignment outcomes.

## Operational Workflow

### Admin Daily Workflow

1. Open Customer Routing settings.
2. Enable the feature for a WhatsMeow instance.
3. Configure prompt delay, timeout, and messages.
4. Add customer-visible agents with display names.
5. Add teams, queue, and optional final custom action.
6. Preview the menu.
7. Save settings.
8. Review audit events after activation.

### Agent Daily Workflow

1. Set availability using the existing availability mechanism.
2. Receive assigned chats normally when customers choose them.
3. See system messages in the chat when customer selection routes a chat.
4. Continue using existing claim, assign, close, and reopen workflows.

### Supervisor Daily Workflow

1. Monitor pending chats.
2. Review selection timeouts.
3. Review unavailable-agent selections.
4. Adjust participant list and capacity rules.
5. Audit routing outcomes by agent, team, and date.

## Open Questions

- Should the first trigger be only after the first pending inbound message, or after every new customer message while pending once the previous session expires?
- Should the custom final option default action be `keep_pending` or `send_only`?
- Should capacity be based on open assigned chats only, or open chats plus active transfers?
- Should an admin be able to configure different menus per WhatsMeow instance, team, or business hours schedule in the first release?
- Should timeout send a message to the customer, or silently keep the chat pending?
- Should audit events be retained forever, or follow a configurable retention policy?
