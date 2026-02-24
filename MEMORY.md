# MEMORY.md

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
