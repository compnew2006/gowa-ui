# Feature Specification: Per-Instance Uploads Cleanup Retention

**Feature Branch**: `001-per-instance-uploads-cleanup`
**Created**: 2026-06-06
**Status**: Draft
**Input**: User description: "Uploads Cleanup feature in the /settings is working for full the system i want it for each whatsapp instance separately to allow the user make instance 5 days and another instance 1 month"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure per-instance retention from the Instance settings (Priority: P1)

A workspace admin with multiple WhatsApp instances wants different media retention policies per instance. They open the WhatsApp instance they care about (e.g., "Sales EU"), see that media files are currently cleaned up after the workspace-wide default, and set a custom retention of 5 days for that instance. They repeat the same for another instance ("Support LATAM") and set 30 days because they need to keep references longer for compliance.

**Why this priority**: This is the core value of the feature — different instances have different operational/compliance needs. Without it, the feature does nothing.

**Independent Test**: Can be fully tested by opening any single instance's settings, changing its retention, and verifying the value is saved, displayed on reload, and used by the cleanup worker for files belonging to that instance. Delivers immediate value (granular control) without requiring multi-instance scenarios.

**Acceptance Scenarios**:
1. **Given** an admin viewing an instance's settings page, **When** they enter a retention value (e.g., 5) and save, **Then** the value is persisted and the UI shows the new value on reload.
2. **Given** an instance with a custom retention of 5 days, **When** the scheduled cleanup runs, **Then** only files older than 5 days belonging to that instance are deleted, while files belonging to other instances with different retention values are not affected.
3. **Given** an instance with no custom retention set, **When** the cleanup runs, **Then** it falls back to the workspace-wide default retention.
4. **Given** an admin sets a retention of 0 for an instance, **When** the cleanup runs, **Then** uploads cleanup is disabled for that specific instance (instance-level override of the system setting).
5. **Given** a workspace with no system-wide retention configured, **When** an instance has no custom retention either, **Then** uploads cleanup is disabled and no files are deleted for that instance.
6. **Given** an instance with a custom retention value, **When** the admin toggles "Inherit workspace default" ON and saves, **Then** the instance's custom retention is cleared and the UI now displays the effective value as the workspace default (or "Disabled" if the workspace default is 0).

---

### User Story 2 - View and override retention from the workspace Uploads Cleanup section (Priority: P2)

A workspace admin reviewing the workspace-wide Uploads Cleanup section wants to see at a glance which instances have custom retention overrides vs. which are using the default. From the same panel they can jump to each instance to change its retention.

**Why this priority**: Improves discoverability and management efficiency for admins with many instances. Builds on Story 1 by giving a single overview, but the feature is still fully usable without it (per-instance configuration from the instance page alone).

**Independent Test**: Can be tested by opening the workspace Uploads Cleanup section, viewing the list of instances and their retention status, and clicking into an instance to change its value. Delivers value as a configuration overview even if per-instance editing is not in this surface.

**Acceptance Scenarios**:
1. **Given** a workspace with multiple instances, **When** the admin opens the Uploads Cleanup section, **Then** they see a list/table of instances with their effective retention (custom or "Default" / "Disabled") and last cleanup run timestamp per instance.
2. **Given** the per-instance list in the Uploads Cleanup section, **When** the admin clicks an instance, **Then** they are taken to that instance's settings where they can edit the retention.
3. **Given** the per-instance list, **When** the admin uses "Run cleanup now" at the workspace level, **Then** the operation runs against all instances in the workspace using each instance's effective retention.
4. **Given** an instance's settings page after at least one retention change has occurred, **When** the page loads, **Then** a "Last 5 changes" history list is visible showing the actor, timestamp, old value, and new value for each recent change.

---

### User Story 3 - "Run cleanup now" for a single instance (Priority: P3)

A workspace admin who just changed the retention on a specific instance wants to trigger cleanup for that instance immediately without running cleanup for the entire workspace (which would sweep all instances and the admin might not want that yet).

**Why this priority**: Operational convenience — saves the admin from waiting for the next scheduled run or running workspace-wide cleanup. Lower priority because the scheduled run and workspace-wide manual run still cover the underlying need.

**Independent Test**: Can be tested by opening an instance's settings, clicking "Run cleanup now" for that instance, and verifying only that instance's files (per its retention) are processed.

**Acceptance Scenarios**:
1. **Given** an instance with a custom retention value, **When** the admin clicks "Run cleanup now" for that instance, **Then** only that instance's expired files are deleted and a result (count, retention used) is shown.
2. **Given** a cleanup is currently running for the workspace, **When** the admin attempts to run cleanup for a single instance, **Then** the request is rejected with a clear "already running" message.

---

### Edge Cases

- **What happens when a file belongs to multiple instances or no instance (legacy/global uploads)?** Legacy or un-scoped uploads fall back to the workspace default retention so they are not orphaned.
- **What happens when an instance is deleted?** Any cleanup reference to it must not cause failures. Pending operations referencing the deleted instance should be skipped.
- **What happens when an instance is renamed?** Retention settings are tied to the instance identity, not its display name — renaming must not reset or lose the retention value.
- **What happens when retention is set to a value larger than the workspace allows?** Existing validation bounds (0 to max) must apply per instance just as they do for the workspace setting.
- **What happens when the cleanup worker is mid-run and a retention value is changed?** The next scheduled run uses the new value; the currently running batch completes with the value it started with to avoid partial-state surprises.
- **What happens when an instance has retention set but the instance has zero upload files?** Cleanup runs cleanly, deletes 0 files, logs the result, and the run is considered successful.
- **What happens if the same retention is set in both the workspace default and the instance override?** The instance override always wins for that instance.
- **What happens to existing global uploads when a user transitions to per-instance configuration?** Pre-existing global uploads continue to use the workspace default until they age out; new uploads are scoped to the instance that created them.
- **What happens when an instance is disconnected or banned during a cleanup run?** Cleanup must still execute for the instance regardless of connection status (connected, disconnected, banned). The worker reads the instance's persisted retention config and deletes its local files without depending on a live WhatsApp session. This prevents storage leaks for offline or banned instances.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow a workspace admin to set a numeric retention value (in days, 0 = disabled, bounded by the existing max retention) for each individual WhatsApp instance.
- **FR-002**: System MUST persist the per-instance retention value and its "last cleanup run date" so the value survives reloads and is auditable.
- **FR-003**: System MUST use the instance's own retention value when running cleanup for files belonging to that instance, overriding any workspace-wide default.
- **FR-004**: System MUST fall back to the workspace-wide retention default for an instance that has no custom retention set, and disable cleanup only when both the instance value and the workspace default are 0.
- **FR-005**: System MUST display the effective retention for each instance (custom value, "Uses default (N days)", or "Disabled") in both the per-instance settings and the workspace Uploads Cleanup overview. The per-instance UI MUST include an explicit "Inherit workspace default" toggle; when ON, the numeric input is disabled and the effective workspace default is shown as a preview; when OFF, the user may enter a custom value (0 = disable for this instance, >0 = N days).
- **FR-006**: System MUST continue to use the existing workspace-wide Uploads Cleanup schedule (hour, timezone, last-run date) for triggering cleanup runs; cleanup is run per-instance within each scheduled execution.
- **FR-007**: System MUST support a manual "Run cleanup now" action scoped to a single instance, in addition to the existing workspace-wide manual run, and must prevent concurrent runs.
- **FR-008**: System MUST report cleanup results per instance in the manual run response (and logs) so admins can see which instance was cleaned, with what retention, and how many files were deleted.
- **FR-009**: System MUST scope upload files to instances such that the cleanup worker can determine which instance a file belongs to (instance-scoped storage path or equivalent association).
- **FR-010**: System MUST skip cleanup for files that cannot be associated with any instance and fall back to the workspace default retention for those files, so legacy or un-scoped uploads are not orphaned.
- **FR-011**: System MUST respect existing authorization — only users with the existing `settings.uploads_cleanup` (read/write/execute) permissions may view, edit, or run per-instance cleanup.
- **FR-012**: System MUST validate the per-instance retention input (numeric, integer, within allowed bounds) and reject invalid values with a clear, user-facing error.
- **FR-013**: System MUST continue to schedule and run cleanup at the workspace level on the existing cadence; the per-instance configuration does not introduce additional scheduled jobs.
- **FR-014**: System MUST scale the per-instance overview and any per-instance listing to an unbounded number of instances per workspace by using server-side pagination and search; the worker MUST iterate all instances efficiently without N+1 database round-trips for the iteration itself.
- **FR-015**: System MUST require the admin to edit per-instance retention one instance at a time; no bulk-apply action is exposed in this iteration. The workspace overview links to each instance for editing and does not provide a multi-select / batch-set control.
- **FR-016**: System MUST persist an audit record for every per-instance retention change (actor, timestamp, old value, new value) and MUST surface a "Last 5 changes" history list on the instance's settings page, so admins can review who changed the retention and when.

### Key Entities *(include if feature involves data)*

- **WhatsApp Instance**: Represents a single WhatsApp connection. Gains a structured "uploads cleanup" sub-setting containing `retention_days` (int, 0 = use workspace default) and `last_run_date` (string, optional). Identity is tied to the instance record, not the display name, so renames do not affect retention.
- **Workspace (Organization) Uploads Cleanup Settings**: Existing entity. The workspace-level `uploads_cleanup_retention_days` and `uploads_cleanup_schedule_hour` remain authoritative defaults and schedule.
- **Upload File**: An uploaded media file. Gains an instance association (or its storage path is updated to include the instance identifier) so the cleanup worker can resolve the file to a specific instance and apply the correct retention.
- **Cleanup Run Result**: Existing summary structure, extended to report per-instance outcomes (instance id/name, retention used, deleted files) when a run touches multiple instances.

## Clarifications

### Session 2026-06-06

- Q: How should the per-instance UI let an admin say "use the workspace default" (i.e., clear a custom retention override)? → A: Explicit "Inherit default" toggle. The per-instance UI exposes a toggle that, when ON, disables the numeric input and shows the effective workspace default as a preview; when OFF, the user can enter a custom value or 0 to disable for that instance.
- Q: How many WhatsApp instances per workspace should the per-instance cleanup UI and worker be designed to support? → A: Unbounded — design for an unbounded number of instances per workspace. The per-instance overview uses server-side pagination and search; the worker iterates all instances efficiently without N+1 database round-trips for the iteration itself.
- Q: What should the cleanup worker do for an instance that is in a non-active state (disconnected, banned, or otherwise not currently connected to WhatsApp)? → A: Cleanup always, regardless of status. The worker reads the instance's persisted retention config and deletes its local files without depending on a live WhatsApp session. This applies to connected, disconnected, and banned instances alike.
- Q: Should the workspace Uploads Cleanup overview support bulk-apply (setting the same retention value on multiple selected instances in a single action)? → A: No — one instance at a time. The workspace overview does not expose a multi-select or batch-set control; each instance must be edited individually from its own settings page.
- Q: Should the system keep an audit trail of per-instance retention changes, and should it be visible in the UI? → A: Yes — persist an audit record on every change and surface a "Last 5 changes" history list on the instance's settings page. Each audit entry records actor, timestamp, old value, and new value.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An admin can change the retention for any given instance and have the new value visibly reflected in the instance settings and the workspace Uploads Cleanup overview within 2 seconds of saving.
- **SC-002**: When the scheduled cleanup runs in a workspace with at least two instances having different retention values, files older than instance A's retention are deleted for instance A only, and files older than instance B's retention are deleted for instance B only — verified by inspecting the file system and the run report.
- **SC-003**: An instance with no custom retention value still has its files cleaned using the workspace default, and a workspace with no retention configured leaves files untouched.
- **SC-004**: An admin can trigger a single-instance manual cleanup and see a result within 10 seconds for typical volumes (hundreds of files per instance) and the response includes the instance name, retention used, and deleted-file count.
- **SC-005**: After the feature ships, zero cleanup-related errors are introduced for legacy/global uploads — these continue to be handled (or skipped) without failure.
- **SC-006**: Admins and operators can determine the effective retention for any instance (custom, default, or disabled) from either the instance settings or the workspace overview without needing to consult logs.

## Assumptions *(mandatory)*

- **A-001**: The existing workspace-level Uploads Cleanup setting (retention days + schedule hour + timezone) remains authoritative for the schedule. Per-instance configuration only overrides the retention value, not the schedule.
- **A-002**: The existing `settings.uploads_cleanup` (read/write/execute) permission set is reused for per-instance management. No new permission is required.
- **A-003**: "1 month" in the user request is interpreted as 30 days. The retention input continues to be in days, with a value range of 0 to the existing maximum (no UI change to introduce month/week units in this iteration).
- **A-004**: Upload files can be associated with a single owning instance. Multi-instance file sharing is out of scope.
- **A-005**: The cleanup worker can be extended to iterate per-instance without changing the scheduling mechanism — i.e., a single scheduled tick becomes a per-instance sweep.
- **A-006**: The current maximum retention bound used in workspace settings is reused for per-instance validation; the bound itself is not expanded in this iteration.
- **A-007**: Per-instance retention is stored as structured data on the instance record (e.g., in its `Settings` JSON column or a dedicated column) — the spec does not dictate which, only that it persists and is queryable.
- **A-008**: The UI for per-instance configuration lives on the existing instance settings surface and is also summarised in the workspace Uploads Cleanup section. No new top-level navigation is introduced.
- **A-009**: The per-instance overview must scale to an unbounded number of instances per workspace and therefore uses server-side pagination and search. The cleanup worker iterates all instances per scheduled tick using an efficient bulk read of instance configs and resolves files to instances without N+1 DB lookups per instance.
- **A-010**: Cleanup status is decoupled from WhatsApp connection status. The worker must run cleanup for any instance that has files and a retention policy (or inherits one), regardless of whether the instance is connected, disconnected, or banned.
- **A-011**: Retention changes are auditable. The system records actor, timestamp, old value, and new value for every per-instance retention change and exposes the most recent entries (up to 5) on the instance's settings page.
