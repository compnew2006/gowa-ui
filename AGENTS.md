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
   └─ imports a service from frontend/src/services/api.ts
        └─ axios call → Go handler (internal/handlers/*.go)
             └─ service/model (internal/models/, internal/...)
```

To trace a feature from page to backend with graphify:
```bash
graphify explain "contactsService"      # the service a page imports
graphify path "ContactsView" "contactsService"
```

## graphify gotchas specific to this repo

- **Soft string references are invisible to the graph.** `Contact.WhatsAppAccount`
  is a `string` (not a GORM FK) pointing at `WhatsAppAccount.Name`, repeated
  across ~15 tables (`Message`, `CannedResponse`, `ChatbotConfig`, `BulkCampaign`,
  `Call`, `Catalog`, `IVRFlow`, …). AST extraction cannot see these as edges.
  If your question is about cross-table relationships by `whatsapp_account`,
  grep the schema in `internal/models/` directly.
- **Renaming `WhatsAppAccount.Name` does not cascade** — every referencing row
  keeps the stale string. There is no DB-level integrity here.

## Conventions

- **Go:** idiomatic, table-driven tests (`*_test.go`), `gofmt` + `go vet`.
- **Vue:** `<script setup lang="ts">`, Composition API, shadcn-vue components.
- **Tests:** a backend change must update or add `*_test.go`; a frontend change
  should keep `npm run typecheck` green. Run the relevant suite before claiming done.
- **Secrets:** never commit `config.toml` credentials; use `config.example.toml`.
