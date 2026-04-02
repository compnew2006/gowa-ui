# Upstream Differences Report

**Repository:** `compnew2006/whatomate` (fork) vs `shridarpatil/whatomate` (upstream)
**Generated:** 2026-04-02
**Total upstream commits not in fork:** 592 (531 non-merge)

---

## Executive Summary

The upstream repository has advanced significantly beyond your fork with **592 commits** spanning major feature additions, critical security fixes, performance improvements, and architectural refactoring. Key areas of divergence include:

1. **WhatsApp Calling System** — A complete voice calling subsystem with IVR, call transfers, WebRTC, TURN support, TTS, and hold music
2. **Critical Security Fixes** — 8+ Phase 1 security patches including httpOnly cookies, SSRF prevention, XSS fixes, and ReDoS patches
3. **Audit Logging System** — Comprehensive audit trails across templates, accounts, teams, IVR flows, and chatbot flows
4. **Detail Pages** — Full detail views for campaigns, accounts, teams, templates, keyword rules, and AI contexts
5. **Vue Flow Node Editor** — Visual drag-and-drop flow builder replacing the previous step-based editor
6. **Internationalization (i18n)** — Full multi-language support with Crowdin integration (Spanish, Hindi, Tamil, Italian)
7. **Multi-Organization Management** — Organization switching, member management, and org-scoped resources
8. **Dashboard Widgets** — Customizable, permission-based dashboard widgets with "Group By" charts
9. **Performance Optimizations** — N+1 query elimination, Brotli compression, code splitting, JOIN query optimizations
10. **UI/UX Overhaul** — Page transitions, skeleton loaders, dark-first design, DataTable consistency

---

## Critical Security Commits (Merge Immediately)

| SHA       | Date       | Author         | Summary                                                                   | Affected Files                                                                                                                           | Recommendation                                                                                                               |
| --------- | ---------- | -------------- | ------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `502d123` | 2026-04-02 | shridhar       | fix: guard against nil tracks in AudioBridge to prevent panic             | `internal/calling/bridge.go`                                                                                                             | **MERGE** — Prevents backend panic/crash                                                                                     |
| `1f06fcc` | 2026-02-09 | Shridhar Patil | security: implement Phase 1 critical security fixes (8 issues) (#157)     | 29 files across `internal/handlers/`, `internal/middleware/`, `internal/crypto/`, `frontend/src/stores/auth.ts`, `cmd/whatomate/main.go` | **MERGE URGENTLY** — 8 critical security vulnerabilities including auth token handling, input validation, and access control |
| `aeb5977` | 2026-02-10 | Shridhar Patil | security: migrate auth tokens from localStorage to httpOnly cookies       | `frontend/src/stores/auth.ts`, `internal/handlers/auth.go`, `internal/middleware/middleware.go`                                          | **MERGE URGENTLY** — Prevents XSS-based token theft                                                                          |
| `e60ffa4` | 2026-03-05 | shridhar       | security(ivr): validate HTTP callback URLs to prevent SSRF                | IVR flow handler files                                                                                                                   | **MERGE URGENTLY** — Prevents Server-Side Request Forgery attacks                                                            |
| `7837e75` | 2026-03-05 | shridhar       | security(ivr): pin HTTP callback to validated public IP to satisfy CodeQL | IVR flow handler files                                                                                                                   | **MERGE** — Additional SSRF defense layer                                                                                    |
| `ba71c41` | 2026-03-02 | shridhar       | fix(deps): patch minimatch ReDoS vulnerability (CVE < 3.1.3)              | `package.json`, `package-lock.json`                                                                                                      | **MERGE URGENTLY** — Known CVE vulnerability                                                                                 |
| `b155082` | 2026-01-09 | shridhar       | Fix XSS vulnerability in template preview                                 | Template preview components                                                                                                              | **MERGE URGENTLY** — Cross-site scripting vulnerability                                                                      |
| `9a75859` | 2026-03-19 | shridhar       | fix: prevent random logouts caused by token refresh race condition        | Auth/token handling files                                                                                                                | **MERGE** — Fixes user experience issue with random logouts                                                                  |
| `735cc64` | 2026-03-19 | shridhar       | fix: call transfer accept race condition corrupting transfer state        | Call transfer handling files                                                                                                             | **MERGE** — Prevents data corruption in call transfers                                                                       |
| `bffda67` | 2026-02-28 | shridhar       | fix: resolve additional crash, security, and deadlock bugs                | Multiple backend files                                                                                                                   | **MERGE** — Fixes crashes and deadlocks                                                                                      |
| `26cb090` | 2026-02-28 | shridhar       | fix: resolve 3 P0 backend panics and silent auth failures                 | Backend handler files                                                                                                                    | **MERGE URGENTLY** — P0 severity crash fixes                                                                                 |
| `968eb73` | 2026-02-10 | Shridhar Patil | security: implement deferred security audit issues                        | Multiple security-related files                                                                                                          | **MERGE** — Additional security hardening                                                                                    |
| `72869b0` | 2026-02-10 | Shridhar Patil | security: enforce phone masking across all output channels                | Phone masking across UI and API                                                                                                          | **MERGE** — Privacy compliance                                                                                               |
| `4a23b38` | 2026-01-24 | Shridhar Patil | fix: Address code scanning security alerts (#79)                          | Multiple files flagged by code scanning                                                                                                  | **MERGE** — GitHub security alerts                                                                                           |

---

## Dependency Updates (Security-Relevant)

| SHA       | Date       | Author     | Summary                                                                      | Recommendation                       |
| --------- | ---------- | ---------- | ---------------------------------------------------------------------------- | ------------------------------------ |
| `9916cc3` | 2026-03-27 | dependabot | chore(deps): bump the npm_and_yarn group across 2 directories with 2 updates | **MERGE**                            |
| `7fe4f39` | 2026-03-26 | dependabot | chore(deps): bump the npm_and_yarn group across 2 directories with 2 updates | **MERGE**                            |
| `8b3a5de` | 2026-03-21 | dependabot | chore(deps): bump the npm_and_yarn group across 2 directories with 2 updates | **MERGE**                            |
| `c693b7c` | 2026-03-18 | dependabot | chore(deps): bump h3                                                         | **MERGE**                            |
| `befa3bd` | 2026-03-18 | dependabot | chore(deps): bump devalue                                                    | **MERGE**                            |
| `7c01f18` | 2026-01-22 | dependabot | chore(deps): bump dompurify                                                  | **MERGE** — XSS prevention library   |
| `4d78b23` | 2026-03-06 | dependabot | chore(deps): bump svgo                                                       | **MERGE**                            |
| `6e06469` | 2026-03-03 | dependabot | chore(deps): bump the npm_and_yarn group across 2 directories with 1 update  | **MERGE**                            |
| `bdb8aef` | 2025-12-10 | dependabot | Bump h3 from 1.15.4 to 1.15.5 in /docs (#76)                                 | **MERGE**                            |
| `17a6aa5` | 2025-12-10 | dependabot | Bump devalue from 5.6.1 to 5.6.2 in /docs (#75)                              | **MERGE**                            |
| `bb1d890` | 2025-12-10 | dependabot | Bump diff from 5.2.0 to 5.2.2 in /docs (#73)                                 | **MERGE**                            |
| `8ccda22` | 2025-12-09 | dependabot | Bump github.com/go-viper/mapstructure/v2 from 2.0.0-alpha.1 to 2.4.0 (#71)   | **MERGE**                            |
| `92b9a1a` | 2025-12-09 | dependabot | Bump golang.org/x/crypto from 0.31.0 to 0.45.0 (#69)                         | **MERGE** — Crypto library update    |
| `fb6ca63` | 2025-12-09 | dependabot | Bump lodash from 4.17.21 to 4.17.23 in /frontend (#72)                       | **MERGE**                            |
| `a6b0024` | 2025-12-09 | dependabot | Bump github.com/jackc/pgx/v5 from 5.4.3 to 5.5.4 (#68)                       | **MERGE** — PostgreSQL driver update |

---

## Performance Commits

| SHA       | Date       | Author   | Summary                                                                  | Affected Files                  | Recommendation                                  |
| --------- | ---------- | -------- | ------------------------------------------------------------------------ | ------------------------------- | ----------------------------------------------- |
| `fd4b791` | 2026-02-06 | shridhar | perf: optimize campaign stats update to eliminate N+1 queries (#160)     | Campaign stats queries          | **MERGE** — Significant performance improvement |
| `3ff56d8` | 2026-02-08 | shridhar | perf: replace permissions Preload(IN) with JOIN query (#156)             | Permission queries              | **MERGE** — Query optimization                  |
| `d6d5986` | 2025-11-19 | shridhar | perf: Add Brotli and Gzip pre-compression for static assets              | Server static file handling     | **MERGE** — Reduced bandwidth                   |
| `0cefa23` | 2025-11-19 | shridhar | perf: Optimize frontend bundle size with code splitting and lazy loading | Vite config, lazy-loaded routes | **MERGE** — Faster page loads                   |
| `56352fe` | 2025-12-03 | shridhar | refactor: Use shared HTTPClient with connection pooling (#57)            | HTTP client configuration       | **MERGE** — Connection reuse                    |

---

## Refactoring & Code Quality

| SHA       | Date       | Author   | Summary                                                                                   | Recommendation                     |
| --------- | ---------- | -------- | ----------------------------------------------------------------------------------------- | ---------------------------------- |
| `cba4f46` | 2026-01-14 | shridhar | refactor: merge SendTemplateMessage and SendTemplateMessageWithComponents into one method | **MERGE** — Reduces duplication    |
| `9a25d19` | 2026-01-14 | shridhar | refactor: deduplicate backend methods across handlers, worker, and WhatsApp client        | **MERGE** — Code quality           |
| `8943a35` | 2025-11-18 | shridhar | refactor: Deduplicate backend code across handlers, worker, and database (#90)            | **MERGE** — Code quality           |
| `fbcd8ee` | 2025-11-17 | shridhar | refactor: Extract reusable components and composables to reduce duplication (#99)         | **MERGE** — Component reuse        |
| `9227481` | 2026-03-28 | shridhar | refactor: unify toast usage with shared useAppToast composable                            | **MERGE** — Consistent toast usage |
| `6b77ae7` | 2026-03-30 | shridhar | refactor: convert accounts list to DataTable for consistent UI                            | **MERGE** — UI consistency         |
| `a064b36` | 2026-03-24 | shridhar | refactor: extract phone masking to internal/utils shared package                          | **MERGE** — Code organization      |
| `3c6e45b` | 2026-02-28 | shridhar | refactor: deduplicate shared patterns across packages                                     | **MERGE** — Code quality           |
| `72bf1dd` | 2026-02-28 | shridhar | refactor: extract shared helpers and fix payload inconsistencies                          | **MERGE** — Code quality           |

---

## Major Feature Commits

### Multi-Organization & Permissions

| SHA       | Date       | Author   | Summary                                                                                | Affected Files                                                                     | Recommendation                           |
| --------- | ---------- | -------- | -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------- |
| `14b6092` | 2026-01-05 | shridhar | Feat/multi org management (#145)                                                       | 40+ files: org models, handlers, middleware, frontend stores, OrganizationSwitcher | **MERGE** — Multi-tenant support         |
| `8a9b1e3` | 2025-12-05 | shridhar | feat: Customizable dashboard widgets with permission-based access control (#66)        | Dashboard widgets, roles, permissions                                              | **MERGE** — Customizable dashboards      |
| `9e7d2e4` | 2026-03-31 | shridhar | fix: add missing chatbot.ai:delete permission to admin role and permission definitions | Role definitions, permission models                                                | **MERGE** — Permission fix               |
| `cb8ead1` | 2026-03-23 | shridhar | feat: reusable team assignment with per-agent rotation and Redis caching               | Team assignment, rotation, caching                                                 | **MERGE** — Load balancing for transfers |

### Audit Logging System

| SHA       | Date       | Author   | Summary                                                                | Affected Files                            | Recommendation                               |
| --------- | ---------- | -------- | ---------------------------------------------------------------------- | ----------------------------------------- | -------------------------------------------- |
| `efc128a` | 2026-03-30 | shridhar | feat: add audit logging and metadata to WhatsApp accounts              | Account models, handlers, frontend        | **MERGE** — Account change tracking          |
| `a9a56e2` | 2026-03-30 | shridhar | feat: add account detail page with audit log and metadata              | Account detail view, router, API          | **MERGE** — Account detail page              |
| `ebe2db4` | 2026-03-30 | shridhar | feat: add team detail page with audit log system                       | Team detail view and audit components     | **MERGE** — Team detail page                 |
| `e996073` | 2026-03-31 | shridhar | feat: add audit logging to chatbot flows with step change tracking     | Flow audit logging, models, handlers      | **MERGE** — Flow change tracking             |
| `57fab86` | 2026-03-31 | shridhar | feat: add IVR flow audit logging with node-level change tracking       | IVR audit logging files                   | **MERGE** — IVR change tracking              |
| `ca00252` | 2026-03-31 | shridhar | feat: IVR flow audit with nested config diffs and right sidebar panel  | IVR audit diff and UI panel               | **MERGE** — Visual config diff               |
| `1ba5931` | 2026-03-30 | shridhar | feat: log masked audit entries when access token or app secret changes | Audit masking logic                       | **MERGE** — Security-sensitive audit masking |
| `9caf2f4` | 2026-04-01 | shridhar | feat: added template details page and audit logging for templates      | Template detail view, audit panel, router | **MERGE** — Template detail page with audit  |

### WhatsApp Calling System

| SHA        | Date       | Author   | Summary                                                                                               | Affected Files                                                                                                                 | Recommendation                                     |
| ---------- | ---------- | -------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------- |
| `913b6de`  | 2026-01-13 | shridhar | feat: add WhatsApp calling system with IVR, call transfers, and outgoing calls                        | 35+ files: `internal/calling/*`, `frontend/src/components/calling/*`, `frontend/src/stores/calling.ts`, `pkg/whatsapp/call.go` | **MERGE** — Entire new calling subsystem           |
| `a6686f5`  | 2026-01-14 | shridhar | feat: add TTS, call permissions, outgoing call fixes, and WebRTC improvements                         | Calling and WebRTC files                                                                                                       | **MERGE** — Text-to-speech and WebRTC improvements |
| `01a76a3`  | 2026-01-15 | shridhar | feat: make ICE servers configurable with TURN support                                                 | `internal/config/config.go`, WebRTC files                                                                                      | **MERGE** — TURN support for NAT traversal         |
| `fadbeef`  | 2026-01-16 | shridhar | feat: fix incoming calls via TURN relay, OGG audio parsing, and inline DTMF detection                 | Audio and DTMF handling files                                                                                                  | **MERGE** — Fixes incoming calls through firewalls |
| `c846dd7`  | 2026-01-20 | shridhar | feat: agent-initiated call transfer with hold music, IVR path improvements, and call log enhancements | Call transfer and IVR files                                                                                                    | **MERGE** — Agent call transfer functionality      |
| `67fadfe`  | 2026-01-21 | shridhar | feat: post-transfer IVR continuation with completed/no_answer branching                               | IVR flow continuation files                                                                                                    | **MERGE** — Post-transfer IVR logic                |
| `a486c81`  | 2026-02-11 | shridhar | feat: add post-call IVR for outgoing calls                                                            | Outgoing call IVR files                                                                                                        | **MERGE** — Post-call survey capability            |
| `b927834`  | 2026-01-25 | shridhar | feat: add Piper TTS to Docker image for IVR text-to-speech                                            | `Dockerfile`, TTS configuration                                                                                                | **MERGE** — TTS support in Docker                  |
| `abf01655` | 2026-01-26 | shridhar | feat: per-org hold music and ringback tone upload                                                     | Audio upload and org settings files                                                                                            | **MERGE** — Customizable hold music                |
| `430933b`  | 2026-01-28 | shridhar | feat: add call recording, IVR call start/enabled split, and call log improvements                     | Call recording and log files                                                                                                   | **MERGE** — Call recording feature                 |
| `12c9b8c`  | 2026-03-24 | shridhar | feat: add HTTP callback hooks to transfer IVR node for CRM integration                                | IVR transfer node files                                                                                                        | **MERGE** — CRM integration via webhooks           |
| `8272dde`  | 2026-03-18 | shridhar | feat: add Redis TLS and username support (fixes #255)                                                 | `internal/database/redis.go`, `internal/config/config.go`, `config.example.toml`                                               | **MERGE** — Redis TLS for secure connections       |

### Campaign Enhancements

| SHA       | Date       | Author   | Summary                                                            | Affected Files                            | Recommendation                                  |
| --------- | ---------- | -------- | ------------------------------------------------------------------ | ----------------------------------------- | ----------------------------------------------- |
| `3a583d6` | 2026-03-31 | shridhar | feat: add campaign detail page with audit logging and progress bar | Campaign detail view, progress bar, audit | **MERGE** — Campaign detail page                |
| `8d68fc1` | 2026-04-01 | shridhar | feat: add recipients, media upload, and actions to campaign detail | Campaign recipients, media upload         | **MERGE** — Campaign management improvements    |
| `ca83134` | 2026-04-01 | shridhar | feat: add recipient validation and audit logging                   | Recipient validation, audit logging       | **MERGE** — Prevents sending to empty campaigns |

### Flow Builder & UI

| SHA       | Date       | Author   | Summary                                                                   | Affected Files                                                                 | Recommendation                                         |
| --------- | ---------- | -------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------ |
| `11bdf26` | 2026-03-29 | shridhar | feat: add Vue Flow node editor for chatbot flow builder                   | `frontend/src/components/chatbot/flow-builder/*`, `useChatbotFlowConverter.ts` | **MERGE** — Visual flow builder (major UX improvement) |
| `b7cad2a` | 2026-03-29 | shridhar | feat: polish flow builder UI and extract shared FlowCanvas                | FlowCanvas component, UI polish                                                | **MERGE** — Flow builder improvements                  |
| `50536bf` | 2026-03-28 | shridhar | feat: add page transitions, skeleton loaders, and micro-interactions      | Transition components, skeleton loaders                                        | **MERGE** — UX polish                                  |
| `886f0ad` | 2026-03-28 | shridhar | feat: enhance dashboard cards, empty states, dialogs, inputs, and sidebar | Dashboard, empty states, dialog components                                     | **MERGE** — UI improvements                            |
| `21ecbbf` | 2026-03-30 | shridhar | feat: reorganize sidebar into sectioned navigation                        | Sidebar navigation, layout components                                          | **MERGE** — Navigation reorganization                  |

### Analytics & Dashboard

| SHA       | Date       | Author   | Summary                                                    | Affected Files                            | Recommendation                         |
| --------- | ---------- | -------- | ---------------------------------------------------------- | ----------------------------------------- | -------------------------------------- |
| `59d7b4f` | 2025-11-15 | shridhar | feat: Add Meta Insights analytics dashboard (#105)         | Meta Insights views, API handlers, models | **MERGE** — Meta analytics integration |
| `e12bc37` | 2025-11-20 | shridhar | feat: Add "Group By" field for chart widgets (#89)         | Chart widget configuration                | **MERGE** — Chart grouping             |
| `f1cd0e0` | 2026-02-08 | shridhar | feat: expand dashboard shortcuts with all nav pages (#154) | Dashboard shortcuts                       | **MERGE** — Navigation shortcuts       |

### Internationalization

| SHA       | Date       | Author   | Summary                                                            | Affected Files                                                | Recommendation                            |
| --------- | ---------- | -------- | ------------------------------------------------------------------ | ------------------------------------------------------------- | ----------------------------------------- |
| `09272f2` | 2025-12-28 | shridhar | feat: Add internationalization (i18n) support (#124)               | `frontend/src/i18n/*`, `LanguageSwitcher.vue`, `package.json` | **MERGE** — Foundation for multi-language |
| `70ed2c8` | 2026-02-27 | shridhar | feat: add all Meta-supported languages with searchable combobox    | Language combobox, all locale files                           | **MERGE** — Full language support         |
| `bf18298` | 2026-03-02 | shridhar | Add Tamil language support to i18n                                 | `ta.json` locale file                                         | **MERGE** if Tamil needed                 |
| `c702aba` | 2026-02-12 | shridhar | fix: complete Spanish translations and fix Hindi/Tamil regressions | `es.json`, `hi.json`, `ta.json`                               | **MERGE** — Translation completeness      |

### Other Notable Features

| SHA       | Date       | Author              | Summary                                                                    | Affected Files                                 | Recommendation                         |
| --------- | ---------- | ------------------- | -------------------------------------------------------------------------- | ---------------------------------------------- | -------------------------------------- |
| `db29e6f` | 2026-03-12 | gianpieropapappicco | feat: support dynamic URL button parameters when sending template messages | Template message sending, URL button handling  | **MERGE** — Dynamic template buttons   |
| `b79442e` | 2026-03-16 | gianpieropapappicco | Added support for COPY_CODE button                                         | Template button types                          | **MERGE** — WhatsApp COPY_CODE support |
| `826c8d0` | 2026-03-27 | shridhar            | feat: add WhatsApp 24-hour service window awareness (#182)                 | Service window tracking, message handlers      | **MERGE** — WhatsApp compliance        |
| `58a5d46` | 2025-12-04 | shridhar            | feat: Add webhook signature verification using Meta App Secret (#62)       | `internal/handlers/webhook.go`, account models | **MERGE** — Webhook security           |
| `94e1bf4` | 2025-11-28 | shridhar            | feat: Add support for wa-cloud-proxy (#52)                                 | Proxy configuration, webhook handling          | **MERGE** if using cloud proxy         |
| `6c12772` | 2025-12-01 | shridhar            | Feat/contacts crud import export (#143)                                    | Contact import/export handlers, UI             | **MERGE** — Contact management         |
| `2534079` | 2025-11-18 | shridhar            | Feat/subscribe app webhooks (#108)                                         | Webhook subscription handlers                  | **MERGE** — Webhook management         |
| `c265726` | 2026-01-30 | shridhar            | feat: add WhatsApp 24-hour service window awareness (#182)                 | Service window logic                           | **MERGE** — WhatsApp window compliance |

---

## Recommended Merge Strategy

### Phase 1: Critical Security (Merge First)

1. `1f06fcc` — Phase 1 critical security fixes (8 issues)
2. `aeb5977` — httpOnly cookie migration
3. `e60ffa4` + `7837e75` — SSRF prevention
4. `ba71c41` — minimatch ReDoS CVE
5. `b155082` — XSS vulnerability fix
6. `26cb090` — P0 backend panics
7. `bffda67` — crash, security, and deadlock bugs
8. `9a75859` — token refresh race condition

### Phase 2: Foundation & Dependencies

1. All dependabot dependency updates
2. `09272f2` — i18n foundation
3. `14b6092` — multi-org management
4. `8a9b1e3` — dashboard widgets
5. `58a5d46` — webhook signature verification

### Phase 3: Major Features

1. `913b6de` + calling ecosystem — WhatsApp calling system
2. `11bdf26` — Vue Flow node editor
3. `3a583d6` + campaign ecosystem — Campaign detail pages
4. Audit logging ecosystem — `efc128a`, `e996073`, `57fab86`, `9caf2f4`

### Phase 4: Polish & Performance

1. Performance optimizations (`fd4b791`, `d6d5986`, `0cefa23`)
2. UI/UX improvements (`50536bf`, `886f0ad`, `9227481`)
3. Remaining feature commits and refactoring

---

## Risk Assessment

- **High Risk if Not Merged:** Security vulnerabilities (XSS, SSRF, ReDoS, token exposure), backend panics, random user logouts
- **Medium Risk if Not Merged:** Missing WhatsApp compliance (24-hour window), performance degradation, missing calling features
- **Low Risk if Not Merged:** UI polish, additional languages, optional features like TTS

---

## Notes

- This report covers **non-merge commits only**. Merge commits from pull requests are excluded but their individual commits are included.
- Some commits are interdependent and should be merged in sequence (e.g., the calling system builds on earlier WebRTC commits).
- The Vue Flow editor (`11bdf26`) replaces the previous step-based flow builder — merging this will require migrating any existing flow data.
- The i18n system (`09272f2`) is foundational; all subsequent language migration commits depend on it.
- Consider creating a dedicated branch (`upstream-sync`) to test the merge before applying to your main branch.
