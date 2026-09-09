# Gowa-UI — Project Notes

> The global feature workflow (Analyze → Explore → Plan → Verify → Execute)
> lives in `~/.zcode/AGENTS.md` and applies here. This file adds
> **project-specific** context on top.

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
- **Grant 2 — involvement:** being the assignee, a collaborator, or the agent
  who closed the conversation (`metadata.closed_by`, stamped by
  `chatlifecycle.Service.Close`/`Leave`) makes that ONE conversation visible
  even under an account the user is NOT assigned to. This is how a manager
  hands a single cross-account conversation to an agent, and how a closed
  conversation stays searchable for the agent who handled it after close
  releases the assignment.
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

## Conventions

- **Go:** idiomatic, table-driven tests (`*_test.go`), `gofmt` + `go vet`.
- **Vue:** `<script setup lang="ts">`, Composition API, shadcn-vue components.
- **Tests:** a backend change must update or add `*_test.go`; a frontend change
  should keep `npm run typecheck` green. Run the relevant suite before claiming done.
- **Secrets:** never commit `config.toml` credentials; use `config.example.toml`.
