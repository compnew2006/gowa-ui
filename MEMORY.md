# MEMORY.md

## 2026-02-25 19:25 Context
*   **Project state**: Agent-name message prefixing is now user-configurable from Send Restrictions.
*   **Current Session Goal**: Move prefix control from role permissions to per-user Send Restrictions settings.
*   **Architectural Decisions**:
    1.  Added `prefix_agent_name` to user `send_restrictions` payloads with default `true` for backward-compatible prefix behavior.
    2.  Replaced role-permission checks in `SendOutgoingMessage` prefix logic with org-scoped user settings lookup.
    3.  Extended Users settings dialog/API + localization to manage prefix toggle alongside other send restriction controls.

## 2026-02-25 19:17 Context
*   **Project state**: Role/permission behavior around outbound chat text formatting is now configurable per role.
*   **Current Session Goal**: Allow organizations to control whether outgoing agent messages include the agent full-name prefix.
*   **Architectural Decisions**:
    1.  Added a dedicated `chat:prefix` permission and integrated it into default role permissions for `admin`, `manager`, and `agent`.
    2.  Gated `SendOutgoingMessage` prefix logic behind permission checks so roles without `chat:prefix` send plain text unchanged.
    3.  Added idempotent DB backfill for existing system roles plus regression tests and role-matrix label updates for explicit UI control.

## 2026-02-25 18:28 Context
*   **Project state**: Hardening chat lifecycle audit traces and close-rating reliability for Whatsmeow-based inbox flows.
*   **Current Session Goal**: Ensure claim chat always emits a visible `chat_claimed` system message and update project docs/session records.
*   **Architectural Decisions**:
    1.  Centralized claim system-message emission in `ClaimChat` so successful idempotent claims (already assigned to same user) still log an explicit claim event.
    2.  Added focused regression coverage for claim-message creation on both pending-claim and already-assigned claim paths.
    3.  Documented claim-audit behavior in `README.md` and recorded the change in `CHANGELOG.md`, `MEMORY.md`, and `session_summary.md`.

## 2026-02-24 15:19 Context
*   **Project state**: Added a production-oriented TypeScript MCP sidecar package under `mcp-server/`.
*   **Current Session Goal**: Expose Whatomate operations via MCP Tools, Resources, and Prompts with stdio + HTTP transports and OpenAI summarization.
*   **Architectural Decisions**:
    1.  Sidecar isolated as a Node package with strict config parsing and outbound host allowlisting.
    2.  MCP registrations split into module registries (`tools`, `resources`, `prompts`) with dedicated typed clients.
    3.  Streamable HTTP on `/mcp` is primary remote transport; legacy `/sse` + `/messages` is optional via feature flag.
    4.  Tool outputs normalize to deterministic `structuredContent` and text payloads for MCP clients.
    5.  CI now includes `mcp-server` lint/typecheck/test/e2e workflow.

## 2026-02-22 22:54 Context
*   **Project state**: Active development of Whatsmeow integration and GOWA (whatomate).
*   **Current Session Goal**: Group sidebar contacts by phone number across multiple accounts, controlled by a setting.
*   **Architectural Decisions**: 
    1.  Will use a toggle setting, stored likely in localStorage for quick access.
    2.  `contactsStore.ts` will respect this setting when computing `filteredContacts`.
    3.  `ChatView.vue` will display an account toggle header if a grouped contact has messages from multiple accounts.

## Past Learnings
*   (Refer to RALPH_MEMORY for previous session learnings).
