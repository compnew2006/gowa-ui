# Whatomate — Dead Code & DRY Analysis Report

**Date:** 2026-07-29
**Scope:** Full repository, with focus on `internal/` (Go) and `frontend/src/` (Vue).
**Graph:** graphify AST graph — 7,752 nodes / 20,187 edges / 308 communities.

## Methodology (cross-validated, low false-positive)

Findings are only reported when **multiple independent tools agree**:

| Tool | Purpose | How it was used |
|------|---------|-----------------|
| `graphify` AST graph | Function relationships & fan-in/fan-out | Parsed `graph.json`; a function is a dead-code *candidate* when it has **zero incoming `calls`/`references`/`indirect_call`/`re_exports`** edges |
| `deadcode ./...` | SSA reachability from `main` (production) | Authoritative "unreachable from entrypoint" |
| `deadcode -test ./...` | Reachability **including test binaries** | Distinguishes *truly dead* from *test-only-alive* |
| `staticcheck -checks U1000` | "never referenced anywhere" | Confirms symbols unused even by tests |
| `knip` | Frontend unused exports / deps / types | Ran in `frontend/` |
| `jscpd` (`--min-tokens 70`) | Copy-paste duplication | 122 clones, 1,774 duplicated lines |
| `grep` cross-check | Kill false positives (string/reflection refs) | Every candidate manually verified |

> **Important nuance:** graphify's AST extraction cannot see Go method dispatch through
> interfaces, `echo`/`fastglue` route registration by function value, or reflection.
> Vue `<template>` bindings are also invisible as `calls` edges. Therefore **all 501 raw
> Vue "candidates" and most handler methods were rejected** — they are wired up via route
> tables and template bindings. Only findings confirmed by SSA tools + grep are listed below.

---

## 1. Dead Code

### 1.1 Go — Truly dead (unreachable from `main` **and** unused by tests) — SAFE TO DELETE

Confirmed by `deadcode ./...` ∩ `deadcode -test ./...` (and grep shows only comment/def lines).

| Symbol | File | Notes |
|--------|------|-------|
| `Service.ResetAssignedChatsTx` | `internal/chatlifecycle/reset.go:160` | Tx variant never wired; non-Tx version is used |
| `FindContact` | `internal/contactutil/contactutil.go:76` | Superseded by `GetOrCreateContact` (12 callers) |
| `AutoMigrate` | `internal/database/postgres.go:118` | Migrations run elsewhere; wrapper orphaned |
| `CreateIndexes` | `internal/database/postgres.go:321` | Never invoked at startup |
| `App.requirePermission` | `internal/handlers/app.go:215` | Lowercase dup of permission checks actually used |
| `App.HasAnyPermission` | `internal/handlers/cache.go:502` | No callers |
| `App.ScopedQuery` | `internal/handlers/cache.go:539` | No callers |
| `App.InvalidateOrgPermissionsCache` | `internal/handlers/cache.go:639` | No callers |
| `RolePermission.TableName` | `internal/models/roles.go:43` | GORM never resolves this type via TableName |
| `middleware.OrganizationContext` | `internal/middleware/middleware.go:247` | Replaced by App-level org resolution |
| `middleware.GetUser` | `internal/middleware/middleware.go:343` | Unused accessor |
| `middleware.GetOrganization` | `internal/middleware/middleware.go:349` | Unused accessor |
| `middleware.IsSuperAdmin` | `internal/middleware/middleware.go:355` | Package-level; `App.IsSuperAdmin` is the live one |

### 1.2 Go — Dead in production, **only kept alive by tests** — REVIEW

These are unreachable from `main` (`deadcode ./...`) but referenced by `*_test.go`.
Either the feature was removed and tests weren't, or these are intentional test seams.

| Symbol | File | Recommendation |
|--------|------|----------------|
| `Assigner.GetAvailableAgents` | `internal/assignment/assigner.go:56` | Verify assignment path still needs it; else delete with test |
| `ResolvePerAgentTimeout` | `internal/assignment/assigner.go:157` | Same |
| `App.WaitForBackgroundTasks` | `internal/handlers/app.go:49` | Likely a graceful-shutdown seam — keep if wired via `main` shutdown, else delete |
| `APISendOptions` | `internal/handlers/messages.go:115` | Constructor with no production caller |
| `middleware.CORS` / `Auth` / `RequirePermission` / `RequireAnyPermission` / `GetUserID` / `GetOrganizationID` | `internal/middleware/middleware.go:73,125,292,310,331,337` | Entire legacy auth-middleware layer appears superseded — the router uses `SecurityHeaders`, `RequestLogger`, `Recovery`, `CSRFProtection`, `RateLimitOpts`, `ParseAllowedOrigins` only. **Confirm before deleting the group.** |
| `Subscriber.Close` | `internal/queue/pubsub.go:116` | Resource-cleanup method never called in prod |
| `isPositionalParam` / `ValidateHeaderParamCount` / `ValidateNoMixedParams` | `internal/templateutil/templateutil.go:13,26,36` | Validation helpers only exercised in tests |
| `NewClient` / `Client.SendChan` | `internal/websocket/client.go:58,277` | `SendChan` comment says "for use in tests" |
| `Hub.BroadcastToUsers` / `GetClientCount` / `IsUserOnline` / `FilterOnlineUsers` / `Unregister` | `internal/websocket/hub.go:207,225,232,265,289` | Hub API surface larger than what's used; trim to actual callers |

### 1.3 Go — Test helper bloat (`test/testutil`, `test/fixtures`)

`deadcode -test` flags **~120 unused builder/helper functions** in `test/testutil/*.go`
and `test/fixtures/models/factories.go` (e.g. the entire `*Builder` fluent APIs,
`MockWhatsAppClient.*`, `MockQueue.*`, many `testutil.*Ptr` helpers).
These are a **test scaffolding library** — much is deliberately provided for future tests.
**Do not bulk-delete.** Treat as low priority; prune only helpers with zero references if
the team wants a lean fixtures package. Full list in `deadcode -test` output.

### 1.4 Frontend — Unused exports/deps (from `knip`)

**Unused runtime dependencies** (`frontend/package.json`):
- `vaul-vue`, `vee-validate` — verify no dynamic use, then remove from `package.json`.
- devDependency `@vue/tsconfig`.

**Unused exports in shipped `src/` (not just e2e):**

| Export | File |
|--------|------|
| `DetailPageLayout` | `src/components/shared/index.ts` |
| `getLocale` | `src/i18n/index.ts` |
| `getErrorMessage` | `src/lib/api-error.ts` (note: a *different* `getErrorMessage` in `lib/api-utils.ts` has 90 callers — do not confuse) |
| `DEFAULT_PAGE_SIZE` | `src/lib/constants.ts` |
| `formatTime`, `truncate`, `debounce`, `generateId`, `formatPrice` | `src/lib/utils.ts` |

**Unused exported types:** `PanelField`/`PanelSection` (PanelConfigEditor.vue), `NavItem`
(navigation.ts), `SupportedLocale` (i18n), `CannedResponseButton`/`AuditLogChange`/`IVRNodePosition`
(services/api.ts), `UserSettings`/`Permission`/`UserRole`/`AuthState` (stores/auth.ts),
`Collaborator`/`ReplyPreview` (stores/contacts.ts), `TransferConfig`/`SimulationStatus`/
`MessageType`/`SimulationSnapshot` (types/flow-preview.ts).

> The bulk of knip's remaining "unused exports" are in `frontend/e2e/` (test framework
> helpers deliberately exported for reuse) — **excluded** from priority cleanup.

---

## 2. DRY Violations (Duplicate Code)

`jscpd` found **122 clones / 1,774 duplicated lines**. Beyond literal clones, graph fan-in
analysis surfaced a structural (non-clone) repetition that is the single biggest DRY issue.

### 2.1 🔴 Handler request-preamble (STRUCTURAL — highest impact)

Not a jscpd clone (variable names differ) but the dominant repeated shape. The graph shows:

- `parsePathUUID()` — **102 callers**
- `findByIDAndOrg()` — **52 callers**
- `a.getOrgID(r)` — **37 call sites**, almost always immediately followed by the **identical**
  `return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")`
- that exact `StatusUnauthorized, "Unauthorized"` envelope line appears **120 times** in `internal/handlers/`

Representative shape repeated across `campaigns.go`, `chat_lifecycle.go`, `contacts.go`,
`roles.go`, `tags.go`, `canned_responses.go`, `users.go`, `webhooks.go`, …:

```go
orgID, err := a.getOrgID(r)
if err != nil {
    return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
}
id, err := parsePathUUID(r, "id", "campaign")
if err != nil { return nil }
entity, err := findByIDAndOrg[models.X](a.DB, r, id, orgID, "X")
if err != nil { return nil }
```

**Fix:** introduce a generic resolver helper, e.g.
`resolveOrgEntity[T](a, r, "id", "X") (orgID, *T, error)` (or a small middleware/decorator)
that collapses the 3-step preamble into one call. Eliminates ~120 error-envelope repetitions
and the 37 org-preamble blocks. Highest ROI change in the codebase.

### 2.2 🔴 Auth token-issuance block (`internal/handlers/auth.go`)

Lines `89-107` duplicated at `247-264` **and** `463-481` (Login / Refresh / SSO callback).

```go
accessToken, err := a.generateAccessToken(&user)  // + error envelope
refreshToken, err := a.generateRefreshToken(&user) // + error envelope
a.setAuthCookies(r, accessToken, refreshToken)
return r.SendEnvelope(CookieAuthResponse{...})
```

**Fix:** extract `a.issueAuthTokens(r, &user) error`.

### 2.3 🟠 Intra-file Go clones (extract private helpers)

| Clone | Location | Fix |
|-------|----------|-----|
| 30L | `handlers/gowa_instances.go:618-647` ↔ `823-852` | Shared instance-response builder |
| 28L+18L×2 | `handlers/chatbot_processor.go:988-1102`, `1015-1264` | Repeated node-processing branches → dispatch table |
| 23L+18L | `handlers/chat_lifecycle.go:200-327`, `247-369` | Shared close/reopen routine |
| 20L | `handlers/conversation_notes.go:143-223` | Shared note CRUD helper |
| 19L | `handlers/roles.go:112-226` | Shared role create/update mapping |
| 18L | `handlers/campaigns.go` (271/677, 405/523, 405/489, 405/557) | Campaign lifecycle state-transition helper |
| 17L | `handlers/canned_responses.go:171-241` | Shared response mapping |
| 16L | `handlers/contacts.go:1090-1105` ↔ `handlers/media.go:159-174` | **Cross-file** media/upload helper → move to shared util |
| 16L | `handlers/tags.go:123-247`, `users.go:394-584`, `custom_actions.go`, `webhooks.go` | Per-file extract |

`campaigns.go` (333 dup-lines), `gowa_instances.go` (265), `chat_lifecycle.go` (241),
`contacts.go` (238), `chatbot_processor.go` (211) are the most clone-heavy files.

### 2.4 🟠 Vue template duplication

| Clone | Location | Fix |
|-------|----------|-----|
| **145L** | `views/chatbot/KeywordDetailView.vue:217-361` ↔ `views/settings/TeamDetailView.vue:245-339` | Largest clone in repo — extract a shared detail-form/section component |
| 29L | `views/chat/ChatView.vue:3056-3114` (self) | Extract repeated message-row sub-template |
| 30L | `components/chatbot/ChatNodeProperties.vue:421-630` (self) | Extract button/header list editor |
| 21L+19L | `components/layout/AppLayout.vue:204-289` (self) | Extract nav-section component |
| 17L | `components/shared/CreateContactDialog.vue` ↔ `views/settings/ContactDetailView.vue` | Shared contact-field fieldset |
| 14L | `components/chat/ContactInfoPanel.vue` ↔ `CreateContactDialog.vue` | Same fieldset |
| 11L | `AccountCallRejectPanel.vue` ↔ `AccountChatResetPanel.vue` | Shared per-account settings panel scaffold |

### 2.5 🟡 Vue script/TS duplication

- `AuditLogPanel.vue:37-50` ↔ `AuditLogDetailView.vue:57-68` — shared audit-formatting composable.
- `ConversationNotes.vue:146-154` ↔ `DashboardView.vue:303-311` — shared time-format util
  (note: `lib/utils.ts` already exports an unused `formatTime` — **consolidate onto it**).
- `AIContextDetailView.vue` ↔ `TeamDetailView.vue`, `AIContextsView.vue` ↔ `ChatbotFlowsView.vue`,
  `ChatbotView.vue` (self) — small `<script setup>` boilerplate; extract composables.

### 2.6 🟡 CSS

Repeated `@font-face`/utility blocks in `assets/fonts.css` and `assets/index.css` — minor.

---

## 3. Function Relationships & Dependency Map

### 3.1 Architecture (confirmed by graph communities)

```
Vue view (frontend/src/views/**/*.vue)
  └─ services/api.ts (axios)         ── 90 callers of getErrorMessage (lib/api-utils.ts)
       └─ Go handler (internal/handlers/*.go)   ── App methods on *fastglue.Request
            ├─ helpers.go   parsePathUUID(102), findByIDAndOrg(52), parsePagination(20), listEnvelope(13)
            ├─ template_engine.go   processTemplate(22), getNestedValue(14), processConditionals(14), evaluateCondition(13)
            ├─ contactutil   GetOrCreateContact(12)
            ├─ templateutil  ExtParamNames(24), ResolveParamsFromMap(12)
            ├─ audit         GetUserName(15)
            ├─ queue/redis   NewRedisQueue(14)
            ├─ websocket     NewUnauthenticatedClient(12)
            └─ pkg/gowa      New(47), PhoneFromJID(12) ; pkg/whatsapp NewRegistry(13)
```

### 3.2 Central hubs (highest fan-in — the DRY leverage points)

| Function | Callers | File |
|----------|--------:|------|
| `parsePathUUID` | 102 | `internal/handlers/helpers.go` |
| `getErrorMessage` | 90 | `frontend/src/lib/api-utils.ts` |
| `findByIDAndOrg` | 52 | `internal/handlers/helpers.go` |
| `gowa.New` | 47 | `pkg/gowa/client.go` |
| `chatlifecycle .Join` | 28 | `internal/chatlifecycle/service.go` |
| `ExtParamNames` | 24 | `internal/templateutil/templateutil.go` |
| `processTemplate` | 22 | `internal/handlers/template_engine.go` |

The already-heavily-used `helpers.go` utilities prove the team's shared-helper pattern works;
§2.1 recommends extending it to the org-preamble that currently escapes it.

### 3.3 Largest code communities (by node count)

`Chat View Components` (161) · `Organization Builder` (111) · `WebSocket Middleware` (102) ·
`WebSocket Client` (92) · `Gowa Server Settings` (88) · `App Layout & Nav` (87) ·
`Canned Response Picker` (84) · `Message Parsers` (80) · `Audit Log & Detail Panels` (73) ·
`Contact Detail View` (71) · `Campaign & Header Utils` (71).

---

## 4. Refactoring Recommendations (DRY)

1. **Extract a handler org/entity resolver** (§2.1). One helper replaces ~37 preambles and
   removes ~120 duplicated `Unauthorized` envelope lines. Do this first — it touches the most
   files and lowers the bar for every future handler.
2. **`a.issueAuthTokens(r, &user)`** (§2.2) — collapses 3 identical token blocks in `auth.go`.
3. **Per-file private helpers** for the intra-file clones in `campaigns.go`,
   `chatbot_processor.go`, `chat_lifecycle.go`, `gowa_instances.go`, `conversation_notes.go`,
   `roles.go`, `canned_responses.go` (§2.3).
4. **Move the `contacts.go` ↔ `media.go` upload block** into a shared util (only confirmed
   cross-file Go clone).
5. **Shared Vue components/composables**: a detail-section component for the 145L
   KeywordDetailView/TeamDetailView clone; a contact-fieldset component; a per-account
   settings-panel scaffold; consolidate time formatting onto the existing (currently unused)
   `lib/utils.ts` `formatTime`.
6. **Delete confirmed-dead symbols** in §1.1 and unused frontend exports/deps in §1.4.

## 5. Priority Ranking

| Priority | Item | Why first |
|----------|------|-----------|
| **P0** | §2.1 handler preamble resolver | Largest blast radius; 120+ repeated lines, 37 blocks; improves every handler |
| **P0** | §1.1 delete truly-dead Go funcs (13) | Zero-risk (SSA + tests + grep agree); shrinks maintenance surface immediately |
| **P1** | §2.2 `issueAuthTokens` + §2.4 145L Vue detail-section clone | High-value, well-bounded extractions |
| **P1** | §1.2 legacy `middleware.*` auth layer | Confirm it's fully superseded, then remove the whole dead layer (10 funcs) |
| **P2** | §2.3 intra-file Go helper extraction (campaigns/chatbot_processor/chat_lifecycle/gowa) | Clone-heavy files; steady readability gains |
| **P2** | §1.4 frontend unused deps/exports | Cleanup + smaller bundle (`vaul-vue`, `vee-validate`) |
| **P3** | §2.4/§2.5 remaining Vue/TS clones + shared composables | Incremental component hygiene |
| **P3** | §1.3 test scaffolding pruning + §2.6 CSS | Lowest risk-adjusted value; much is intentional |

## 6. How to Reproduce / Verify

```bash
graphify update . --force                       # rebuild AST graph (7752 nodes)
deadcode ./...                                  # production-unreachable
deadcode -test ./internal/... ./pkg/... ./cmd/...  # truly-dead vs test-only
staticcheck -checks U1000 ./internal/... ./pkg/... ./cmd/...
cd frontend && npx knip --no-progress           # frontend unused
npx jscpd internal/ frontend/src/ --min-tokens 70 --ignore "**/*_test.go,**/node_modules/**,**/components/ui/**,**/dist/**"
```

> After any deletion, re-run `graphify update . --force` and `go build ./... && go test ./...`
> before claiming done. Verify each §1.2 middleware removal against the router in
> `cmd/whatomate/main.go` — those functions are the only remaining risk of a false positive.
