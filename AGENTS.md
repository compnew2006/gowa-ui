# Gowa-UI — Agent Instructions

Work as my coding partner. Prioritize correctness, focused changes, existing architecture, and solutions I can maintain. These rules travel with this repository and apply to Codex and Claude Code; honor more specific project instructions and explicit task constraints. Preserve existing work and use the coding client's configured model unless I request a change.

## Start and scope

- Identify the repository root, stack, current Git changes, relevant instructions, and existing validation commands. Discover and read applicable repository `AGENTS.md` and project instruction files even when the active client does not load them automatically; verify what actually loaded. Convert the request into observable acceptance criteria. Keep simple plans brief; plan dependencies, affected callers, and risks for complex changes.
- Make routine, reversible decisions autonomously and carry authorized work through implementation and verification. Ask only for essential missing information or an action requiring authorization; continue independent work while awaiting an answer. Authorization already given in the conversation remains valid.
- Preserve unrelated edits and public behavior. Do not reset, overwrite, or clean up my work to simplify the task. Reuse established helpers and abstractions; avoid unrelated refactoring and speculative features.
- Never edit project `.env` or secret files; report required changes instead. Keep credentials out of source, logs, tool output, and memory. Report exposed secrets without repeating them; do not rotate credentials without authorization.
- Commits, pushes, publishing, deployment, real-data migrations, external messages, paid jobs, unrelated global changes, and tool/dependency upgrades need authorization unless already granted. A request for implementation does not automatically authorize those actions. A loop does not schedule future runs.

## Tool readiness and portable discovery

- Check the active OS/client and actual session capabilities when starting in a new environment; reuse verified setup and recheck when it changes. Select tools needed for the task. Distinguish missing installation, missing registration, disconnection, and a session needing reload. Documentation-only work does not require installing unrelated coding tools.
- I authorize installing missing workflow tools, necessary missing prerequisites, and minimum local MCP/skill registration before work that depends on them. Restore project dependencies from existing manifests and lockfiles when required. Preserve working runtimes, version constraints, credentials, indexes, browser profiles, and unrelated client settings; this does not authorize upgrades or knowledge resets.
- Discover commands through the current environment (`command -v` on macOS/Linux or `Get-Command` in PowerShell), supported client configuration, package-manager metadata, and skill catalogs. Resolve actual files and symlinks; account for a desktop client's launch environment. Keep discovered installation paths in machine-local configuration only, never in reusable prompts or shared instructions. Do not assume usernames, drives, clone locations, or package-manager prefixes.
- Before installing or repairing a tool, read `references/agent-tooling.md`, resolved relative to this instruction file, then open the relevant current official installation guide. This reference is loaded on demand; do not read it for every task. If it is missing, identify the tool's verified official source before proceeding. Back up affected client configuration and verify startup, MCP initialization/capabilities, or skill discovery afterward.
- Use `browser-controller` as the Browser Controller MCP registration name. Verify its actual provider, executable, connection, and exposed tools; reuse the installation and avoid duplicate registrations. If setup requires a client reload or manual browser pairing, report the exact remaining step and continue independent work. Configuration edits do not change tools already loaded into a session.

## Code discovery and knowledge

<!-- codebase-memory-mcp:start -->
- Prefer **codebase-memory-mcp** for semantic code discovery, architecture, and call relationships. Use `search_graph`, `trace_path`, `get_code_snippet`, `query_graph`, and `get_architecture` according to the current tool schemas. Confirm the indexed root, checkout/worktree, freshness, and coverage before relying on results.
- Native reads and literal searches are appropriate for configuration, non-code files, strings, dynamic/string-based relationships, or unavailable/insufficient graph results. Source code and behavior remain authoritative; verify graph matches against them.
<!-- codebase-memory-mcp:end -->

- Use **Serena** for appropriate symbol navigation and structural edits; read its current usage instructions and confirm the active project. Treat memories and caches as potentially stale.
- Use **Graphify** when dependency or impact analysis adds evidence. Use the current repository's graph and exact, verified symbols; build or refresh only when needed. Do not duplicate analysis already answered reliably or make every small change depend on a full graph build.
- Use **Context7 or official documentation** for uncertain APIs, matching the project's installed versions. Follow primary sources rather than guessing APIs or tool names.
- Discover optional **SkillPilot** routing through its actual metadata; if absent, use client catalogs directly. Installed tools are not necessarily exposed in the current session.
- Use relevant specialist skills, including installed ECC skills, and agents when they add clear value and are permitted. Give parallel workers distinct ownership, preserve other workers' edits, and verify their results. Keep simple tasks simple.

## Engineering loop

Repeat **inspect → implement → verify → review → fix** for the next unmet acceptance criterion. Each iteration must produce a coherent improvement, new diagnostic evidence, or a concrete blocker.

1. **Inspect:** Read exact source and relevant tests. Check existing behavior, reusable patterns, affected callers, authorization boundaries, and data relationships. Choose the smallest coherent change.
2. **Implement:** Follow the project's architecture and language conventions. Validate inputs at boundaries, handle errors, and preserve established API/data formats. Prefer immutable updates where appropriate to the language and existing design; do not impose a new architecture or arbitrary file-size rule.
3. **Verify:** Run relevant project checks and inspect actual results. For bugs, add a meaningful regression test and observe its intended failure before the fix when practical. Follow any explicit project TDD/coverage requirements; do not impose a universal coverage percentage or require unrelated test layers. Start targeted, then broaden for shared code, critical flows, or evidence of wider impact. Use isolated test data. Inspect scripts before running commands that may modify files.
4. **Review and fix:** Review the actual diff against acceptance criteria, regressions, edge cases, duplication, security, and unrelated changes. Fix findings and rerun affected checks. Never weaken a valid test just to obtain a pass. Repeat passed checks only when new changes or unresolved evidence justify it.

- Apply relevant build, lint, type, unit/integration/E2E, and security checks. Use configured security tools where they address the change. Do not claim repository-wide coverage or security from a narrow check.
- Diagnose failures before retrying. After two attempts with the same failure, change the hypothesis or method and gather new evidence. Use a permitted, reliable alternative when an auxiliary tool fails; stop dependent work only when proceeding would be unsafe or unverifiable. Continue independent useful work.

## Browser and UI verification

- **Local project pages:** Use **Playwright MCP** for `localhost`, `127.0.0.1`, or `::1`. Read run instructions, start or reuse the right server, verify readiness and the actual URL, then exercise affected success/validation/error flows. Inspect console errors and failed requests when diagnosing failures. Run automated project E2E tests when required or appropriate; interactive MCP checks complement them.
- **Internet pages:** Use **browser-controller MCP** through my already opened profiles/tabs and existing login sessions. Discover the intended profile/tab through supported tools or connection metadata; verify the profile and URL before acting. Preserve unrelated tabs and profile settings. Do not substitute a fresh profile silently. Existing authorization boundaries still apply to actions on pages.
- **Impeccable is required for user-facing UI/UX changes**, including layout, styling, interaction, responsiveness, accessibility, and copy. Discover its installed skill through the active client and read `SKILL.md` plus relevant references. Follow the installed version's setup/context and command guidance; obtain essential missing design decisions before dependent work and preserve the established design system unless a redesign is requested.
- Verify affected interactions, loading/empty/error/disabled states, keyboard/focus access, responsiveness, contrast, and reduced motion when relevant. Check Arabic/RTL, English/LTR, and supported themes when affected. Inspect the rendered result with the appropriate browser MCP. Batch visual findings, fix them together, and use one confirmation pass for cosmetic polish; keep fixing reproducible functional/accessibility/acceptance failures.
- Record the tested URL without secrets, browser MCP, relevant profile identity, scenarios, and actual results. If browser access is unavailable, complete independent checks and identify the remaining browser/visual verification. Backend-only work does not require Impeccable.

## Completion and reporting

- Continue while meaningful authorized progress remains. Finish when acceptance criteria are satisfied, required checks pass, and the final diff is reviewed. Honor explicit iteration/time/cost limits; do not invent a task-wide retry limit or present blocked/unverified work as complete.
- For long work, keep a compact checkpoint of the goal, completed criteria, changed files, check results, and next step in the conversation or existing tracker. Resume from it after context loss. Use existing project docs for lasting knowledge and avoid unsolicited top-level tracking files or duplicate notes.
- Refresh relevant code knowledge through the installed tool's supported mechanism when source relationships change and confirm its status. Impact analysis alone is not proof of an index refresh. Store only useful, verified knowledge and never credentials.
- Report concisely in simple Egyptian Arabic unless I request another language: outcome, key files, checks actually performed, and material remaining limitations or knowledge-refresh status. Keep technical names and commands in English. Claim success only with evidence.

## Task requests

Use `task-template.md` for individual task details when helpful. Accept clear natural-language requests and infer nonessential missing fields from context.

---


## Stack

WhatsApp Business messaging platform: **Go backend** (`cmd/`, `internal/`,
`pkg/`) + **Vue 3 frontend** (`frontend/`). Data layer is GORM + PostgreSQL,
with Redis for caching/queues.

## Architecture map (full-stack chain)

```
Vue page (frontend/src/views/.../*.vue)
   └─ view is a thin shell that wires composables (frontend/src/composables/)
        └─ a composable imports a service from frontend/src/services/api.ts
             └─ axios call → Go handler (internal/handlers/*.go)
                  └─ service/model (internal/models/, internal/...)
```

To trace a feature from page to backend with graphify:
```bash
graphify explain "contactsService"      # the service a page imports
graphify path "ContactsView" "contactsService"
```

## File-organization conventions

Large files are split by **concern**, keeping each file under ~1k lines and one
responsibility. Do not re-merge these.

- **Backend handlers (`internal/handlers/`)** — a `*App` method's home file is
  its concern, not its route prefix. Moving a method between files in this
  package is transparent (routing in `cmd/gowa-ui/main.go` references
  `app.MethodName`, not the source file).
  - `contacts.go` — contact CRUD + assignment/tags + response builders only.
  - `internal_contacts.go` — org-account/private-contact detection, manual
    Private-tab changes, and merged account-to-account conversation rows.
  - `messages.go` — message list/send/revoke/react/typing + read-state +
    the WhatsApp account/provider resolvers + `gowaChatJID`.
  - `contacts_avatars.go` — contact profile-picture fetch/cache/serve.
  - `business_hours.go` — per-account business-hours block + outside-hours
    away reply (hooked at the end of `processGowaMessage`; cooldown-guarded).
  - `device_alerts.go` — device disconnect/recovery alerting (audit + WS +
    optional Telegram), hooked from `processGowaConnection` and
    `updateGowaDeviceAccountStatus`.
  - `campaign_pacing_settings.go` — per-account send-pacing block (the
    worker side lives in `internal/worker/pacing.go`).
- **Frontend views (`frontend/src/views/`)** — a view stays an orchestration
  shell (route/contacts/messages watchers + lifecycle + simple view state).
  Domain logic lives in a composable under `frontend/src/composables/`.
  - `ChatView.vue` is the reference split: its `<template>` is untouched
    (e2e tests depend on the DOM), and the script delegates to `useChat*`
    composables (`useChatMessaging`, `useChatContactsList`, `useMessageFormat`,
    `useChatScroll`, `useChatMedia`, `useChatTyping`, `useChatCannedTemplates`,
    `useChatLifecycle`).
  - When a composable needs a DOM `ref="x"` template binding, **declare the ref
    in the view and pass it in** — vue-tsc does not reliably track
    template-ref usage on refs destructured from a composable.
  - Shared state a view reads/writes across composables (e.g. `selectedAccount`)
    is owned by the view and passed to each composable to avoid temporal-dead-
    zone ordering between composables.

## graphify gotchas specific to this repo

- **Soft string references are invisible to the graph.** `Contact.WhatsAppAccount`
  is a `string` (not a GORM FK) pointing at `WhatsAppAccount.Name`, repeated
  across 7 tables (`Contact`, `Message`, `Template`, `BulkMessageCampaign`,
  `NotificationRule`, `ScheduledMessage`, `ChatClosureRating`). AST extraction
  cannot see these as edges.
  If your question is about cross-table relationships by `whatsapp_account`,
  grep the schema in `internal/models/` directly.
- **Renaming `WhatsAppAccount.Name` does not cascade** — every referencing row
  keeps the stale string. There is no DB-level integrity here.

## Contact visibility scoping

- **`scopeAssignedContact` (`internal/handlers/contacts.go`) is the single
  gate for contact/conversation visibility.** It OR-combines two grants and is
  applied at every contact endpoint (ListContacts, GetMessages, media serving,
  scheduled messages — ~17 call sites). New contact endpoints must route their
  query through it, never scope by hand.
- **Grant 1 — account scoping:** a user assigned a subset of WhatsApp accounts
  (`user_whatsapp_accounts`) sees conversations under those accounts; super
  admins and users with **no** assignment fall back to full org visibility.
  Because contacts key off `whats_app_account` (the account **Name** string),
  the assigned account IDs → names are resolved before filtering. (Contacts
  mirror of `scopeAccountsToUser` in accounts.go, used by
  `/settings/accounts`.)
- **Grant 2 — involvement:** being the assignee, a collaborator, or the
  holder of an ACTIVE assignment access grant
  (`contact_assignment_access_grants`, minted ONLY by direct admin assignment
  — `chatlifecycle.upsertAssignmentGrant`) makes that ONE conversation
  visible even under an account the user is NOT assigned to. A grant holder
  whose assignment has moved on is READ-ONLY: every write/lifecycle path
  refuses via `rejectHistoricalAssignmentAccess` (central in
  `loadContactByPath` + `findScopedMutableContact` + inline in message/
  template/scheduled/media write paths). `metadata.closed_by` is an AUDIT
  stamp only — never a visibility path; revoking the grant ends access even
  for the agent who closed the conversation. Release grant + release
  assignment commit in ONE transaction (`ChatLifecycle.ReleaseWithDB`).
- **Private/internal conversations** are the union of chats with one of the
  org's connected WhatsApp numbers and contacts manually moved to Private.
  The manual marker is the boolean `Contact.Metadata[models.MetaInternalChat]`
  (`internal_chat`); `ContactResponse.is_internal` is the combined result,
  while `internal_marked` reports only the manual marker. UI toggles must use
  `internal_marked`, since an org-number conversation stays internal even if
  its manual marker is cleared. `PUT /api/contacts/{id}/internal` requires
  `chat:write`, scopes mutations through `findScopedMutableContact`, patches
  only this JSONB key (preserving concurrent metadata changes), audits a real
  state change, and sends a contact-scoped `contact_update` event.
- `show_internal_tab` is a per-user display preference, defaulting to true.
  `PUT /me/settings` is a partial update: omitted notification or chat-setting
  fields must remain unchanged. Hiding Private only hides its sidebar tab for
  that user; it does not revoke chat visibility, and cross-tab search can still
  return those conversations.
- `handlers.isInternalContact` is the central guard for customer-facing
  automations. Check it before sending customer replies or consuming pending
  customer-automation state. This includes all branches of close-rating flows:
  guarding prompt creation alone does not protect an already-pending rating
  cycle and its thank-you reply. Queue/batch SQL that excludes internal chats
  should append `models.ExcludeInternalContactsSQL` only when querying the
  `contacts` table; it excludes both manual markers and org account numbers.
- `ListInternalConversations` merges only the two sides of chats between org
  account numbers and still scopes each side with `scopeAssignedContact`.
  Manually marked ordinary contacts are returned in the regular contact list
  with `is_internal=true`; the frontend adds them to Private as unpaired rows.
- **`contacts:read` (chat visibility) ≠ `contacts.manage:read` (settings page).**
  `contacts:read` drives chat-list scoping inside `scopeAssignedContact`
  (users with it see their accounts' conversations plus any they are involved
  in; without it, involvement only). The
  `/settings/contacts` management page (and its Import/Export) is gated
  separately on the `contacts.manage` resource (`router/index.ts` route meta +
  `navigation.ts`, checked with the `read` action). This lets a role see
  conversations in `/chat` while being blocked from the contacts directory.
  Default seeding: `admin` + `manager` get `contacts.manage:read`; `agent` does
  not. Import/Export stay additionally enforced by the `contacts:import`/
  `contacts:export` actions.
- **WebSocket broadcasts are contact-scoped (`internal/handlers/ws_scoping.go`).**
  Conversation-content events (new_message, reaction_update, message_edited,
  status_update, chat presence) go through `wsContactRecipients` →
  `BroadcastToUsers` — the same population `scopeAssignedContact` lets read the
  conversation (involvement on any account; otherwise contacts:read + account
  coverage). Errors fail CLOSED (broadcast dropped, never org-wide). Lifecycle
  events (chat_claimed/released/closed/access-revoked) and ops events (campaign/
  device/app status) stay org-wide BY DESIGN: they carry no conversation content
  and the user who LOST access must still receive the one event that tells their
  UI to drop the chat. Assignment changes must go through `ChatLifecycle.Assign`
  (AssignContact AND UpdateContact's `assigned_user_id`/`clear_assigned_agent`
  fields) — a raw `assigned_user_id` write would create an assignment with no
  access grant. Access modes: `standard` | `current_assignee` | `collaborator`
  (ACTIVE collaborator = full access) | `historical_assignment_read_only`.

## Conventions

- **Go:** idiomatic, table-driven tests (`*_test.go`), `gofmt` + `go vet`.
- **Vue:** `<script setup lang="ts">`, Composition API, shadcn-vue components.
- **Tests:** a backend change must update or add `*_test.go`; a frontend change
  should keep `npm run typecheck` green. Run the relevant suite before claiming done.
- **Secrets:** never commit `config.toml` credentials; use `config.example.toml`.
