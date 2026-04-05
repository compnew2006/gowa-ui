# Session Summary

## 2026-04-05 10:21

### Completed

- Removed the legacy organization-level `Close Chat Rating` controls from the Chat tab in `frontend/src/views/settings/SettingsView.vue`.
- Removed the old organization-level API and backend settings handling for close ratings in:
  - `frontend/src/services/api.ts`
  - `internal/handlers/organization.go`
- Kept the feature instance-specific and cleaned the instance save path in:
  - `frontend/src/components/whatsmeow/InstanceChatCloseRatingPanel.vue`
  - `frontend/src/views/settings/InstancesView.vue`
  - `frontend/src/lib/instance-chat-close-rating.ts`
- Updated translations used by the instance summary card so the reply-window summary renders correctly instead of showing a raw i18n key.
- Added and updated focused tests covering:
  - legacy org-level settings removal
  - instance-specific close-rating save behavior
  - backend instance settings loading and validation

### Findings

- The stale `Close Chat Rating` section still appearing in `/settings` after the source change was caused by the running server still serving an older embedded frontend build.
- The instance dialog had a real frontend state bug where the numeric follow-up window field could display the typed value while still saving the previous default value.
- Restarting the app after rebuilding the embedded frontend was required to validate the actual live behavior on `http://localhost:8080`.

### Verification

- `go test ./internal/handlers -run 'TestApp_(GetOrganizationSettings|UpdateOrganizationSettings)|TestHandleManualChatCloseRatingPrompt|TestReadInstanceChatCloseRatingSettings'`
  - result: pass
- `go test ./pkg/whatsmeow -run 'Test(ConnectionManagerLoadChatCloseRatingSettings_UsesInstanceSettings|EnsureInstanceSettingsDefaults_InjectsChatCloseRatingDefaults|ValidateInstanceSettings_ChatCloseRating|PersistParsedMessage_|ParseInboundRatingValue_)'`
  - result: pass
- `make frontend-build embed-frontend`
  - result: pass
- `cd frontend && BASE_URL=http://localhost:8080 npx playwright test e2e/tests/settings/general-settings.spec.ts --grep 'should not show close chat rating controls in chat settings'`
  - result: pass
- `cd frontend && BASE_URL=http://localhost:8080 npx playwright test e2e/tests/settings/instances.spec.ts --grep 'should save instance specific chat close rating settings'`
  - result: pass

### Notes

- `summery.md` was updated in this session at the user's request.
- `npm --prefix frontend run typecheck` still has unrelated pre-existing repo-wide TypeScript errors outside this close-rating work.

## 2026-04-05 08:32

### Completed

- Investigated the duplicated "Close Chat Rating" controls in the organization Chat settings at `http://localhost:8080/settings` and the instance settings at `http://localhost:8080/settings/instances`.
- Traced the full code path for manual close prompt creation and inbound rating capture across:
  - `internal/handlers/chat_close_ratings.go`
  - `pkg/whatsmeow/chat_close_ratings.go`
  - `frontend/src/views/settings/SettingsView.vue`
  - `frontend/src/views/settings/InstancesView.vue`
  - `frontend/src/components/whatsmeow/InstanceCard.vue`
  - `frontend/src/components/whatsmeow/InstanceChatCloseRatingPanel.vue`
  - `frontend/src/lib/instance-chat-close-rating.ts`
- Confirmed the app is running with the WhatsMeow provider from `config.toml`.
- Verified against the live database that recent `chat_closure_ratings.close_message` rows are using the instance-level custom 1-5 template for the current instance, which proves the manual close prompt path is reading instance template overrides.

### Findings

- The organization settings page is the live source of truth for the inbound reply-capture flow in the current WhatsMeow runtime.
- The manual close prompt sender reads organization settings first and then applies instance overrides, so instance-level custom templates and instance-level disable can affect prompt creation.
- The active WhatsMeow inbound capture path only loads organization settings and ignores instance overrides:
  - it does not read instance-level `chat_close_rating_enabled`
  - it does not read instance-level `chat_close_rating_followup_window_minutes`
  - it hardcodes the reply lookup window to 2 days
- Practical result in the current app:
  - organization-level enable and follow-up window are actually enforced for reply capture
  - instance-level custom templates are actually used for the outgoing close-rating prompt text
  - instance-level enable as an override is not reliable when used to enable the feature while organization-level setting is disabled
  - instance-level follow-up window is currently not effective in the active WhatsMeow reply-capture flow

### Skills Applied

- `spec-miner` for tracing the existing feature across UI, handlers, provider runtime, and database evidence
- `debugging-wizard` for verifying the real runtime path, isolating the split behavior, and confirming the live provider-specific execution path

### Verification

- Chrome DevTools MCP against `http://localhost:8080/settings`
  - confirmed the org-level Chat tab exposes `Close Chat Rating`, `Follow-up Window (minutes)`, and `Rating Message Templates`
- Chrome DevTools MCP against `http://localhost:8080/settings/instances`
  - confirmed the instance card exposes `Chat Close Rating Settings`, an enabled switch, and the per-instance configure dialog
- Chrome DevTools MCP `fetch()` checks from the live authenticated app
  - `/api/org/settings` returned org-level close-rating settings with the default 1-10 template
  - `/api/instances` returned instance-level close-rating settings with custom 1-5 templates for instance `n0n`
- Database verification via local Postgres
  - recent `chat_closure_ratings.close_message` rows include the instance 1-5 template text, confirming instance template overrides are used during prompt creation
- Focused Go tests
  - `go test ./internal/handlers -run 'TestReadChatCloseRatingSettings|TestHandleManualChatCloseRatingPrompt' -count=1`
  - `go test ./pkg/whatsmeow -run 'TestPersistParsedMessage_DoesNotReopenClosedChatForPendingRatingReply|TestPersistParsedMessage_DoesNotReopenClosedChatForFollowupComment' -count=1`

### Recommendations

- Keep the organization settings page as the canonical source of truth for:
  - enable/disable
  - follow-up window
  - default templates
- Reduce the instance page to one explicit override surface only:
  - either keep only "Override organization templates" for per-instance message text
  - or remove the instance close-rating section entirely if per-instance customization is not required
- If instance-level overrides must remain, fix the WhatsMeow runtime first so `pkg/whatsmeow/chat_close_ratings.go` loads merged org + instance settings instead of org-only settings. Without that fix, the duplicate UI remains misleading.
- After that, tighten the frontend contract:
  - do not expose instance-level enable unless the runtime fully supports instance enable overrides end-to-end
  - do not persist instance-level follow-up window unless the runtime actually consumes it
  - align the frontend default copy so org and instance templates do not imply different rating scales unless that is intentional

### Notes

- No production code was changed in this session.
- `summery.md` was updated with this investigation record.

## 2026-04-04 19:55

### Completed

- Centered the collapsed desktop sidebar navigation icons in `frontend/src/components/layout/AppLayout.vue` by removing the hidden-label gap from the collapsed item layout.
- Removed the desktop sidebar expand/pin button from `frontend/src/components/layout/AppLayout.vue` so the sidebar now expands only from hover and focus-within behavior.
- Added a repo-level Go hot reload workflow using `air`:
  - added `.air.toml`
  - added `Makefile` targets:
    - `air-install`
    - `backend-watch`
    - `dev-watch`
- Updated `README.md` with the recommended fast development loop for:
  - frontend-only changes
  - backend-only changes
  - frontend + backend + model/schema changes

### Skills Applied

- `vue-expert` for the sidebar layout adjustment and removing the pin/expand control without breaking the existing hover/focus behavior
- `devops-engineer` for the hot-reload developer workflow, `air` configuration, and Makefile/README integration

### Verification

- Chrome DevTools MCP against a temporary local harness for the collapsed sidebar item layout:
  - confirmed the collapsed icon center matched the item center exactly
  - confirmed no expand button was rendered in the verified collapsed state
- `make air-install` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- `make -n dev-watch` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
  - confirmed the combined watcher target expands to `backend-watch` + `frontend-dev`
- `make backend-watch` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
  - verified `air` started successfully
  - verified the backend built successfully
  - verified the server booted with `-migrate`
  - verified graceful shutdown after interrupt

### Notes

- The first `air` config revision used the deprecated `build.bin` style and failed to execute the binary with arguments correctly. It was corrected to the current `entrypoint` + `args_bin` format and re-verified.

## 2026-04-03 02:28

### Completed

- Fixed the washed-out light-mode settings controls by strengthening the shared light-theme surface tokens in `frontend/src/assets/index.css` so the page background, cards, borders, and input surfaces no longer collapse into the same near-white tone.
- Updated the shared form primitives to render with clearer control affordances in light mode:
  - `frontend/src/components/ui/input/Input.vue`
  - `frontend/src/components/ui/select/SelectTrigger.vue`
  - `frontend/src/components/ui/textarea/Textarea.vue`
  - `frontend/src/components/ui/switch/Switch.vue`
- Promoted the general settings save action in `frontend/src/views/settings/SettingsView.vue` from an outline treatment to the primary button treatment so it remains clearly visible in light mode.

### Skills Applied

- `vue-expert` for the Vue 3 and design-token level fix across the shared UI primitives and settings view

### Verification

- `npx eslint src/components/ui/input/Input.vue src/components/ui/select/SelectTrigger.vue src/components/ui/switch/Switch.vue src/components/ui/textarea/Textarea.vue src/views/settings/SettingsView.vue` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP against the patched Vite app on `http://127.0.0.1:3000/settings`
  - authenticated with `admin@test.com`
  - forced `localStorage['color-mode'] = 'light'` and reloaded to verify the actual light-mode path
  - captured screenshots at:
    - `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/test-results/settings-light-after-fix-lightmode.png`
    - `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/test-results/settings-light-after-fix-lightmode-v2.png`
  - confirmed rendered control values after the fix:
    - page background `rgb(246, 249, 252)`
    - card surface `rgba(255, 255, 255, 0.95)`
    - input/select surface `rgb(226, 237, 245)`
    - unchecked switch surface `rgb(231, 239, 245)`

### Notes

- The embedded app already running on `:8080` was kept as the API/backend target; final UI verification used the local Vite frontend on `:3000` so the browser reflected the new frontend changes immediately.

## 2026-04-03 02:06

### Completed

- Fixed the sidebar organization switcher interaction in `frontend/src/components/layout/OrganizationSwitcher.vue` by replacing the unstable sidebar `Select` with a controlled popover-based organization menu.
- Kept the desktop sidebar expanded while sidebar overlays are open by wiring overlay-open state through:
  - `frontend/src/components/layout/AppLayout.vue`
  - `frontend/src/components/layout/OrganizationSwitcher.vue`
  - `frontend/src/components/layout/UserMenu.vue`
- Added stable test hooks for the organization menu trigger, content, and items so the hover-to-open sidebar path can be regression-tested.
- Added a focused regression test in `frontend/e2e/tests/settings/organization-switch.spec.ts` that verifies:
  - the sidebar expands on hover when not pinned
  - the organization menu opens from the sidebar
  - a different organization can be clicked from that menu
  - the selected organization id is persisted after switching

### Skills Applied

- `vue-expert` for the Vue 3 sidebar state propagation, organization-switcher refactor, and interaction fix
- `playwright-expert` for the targeted hover/sidebar/org-switch regression coverage

### Verification

- `BASE_URL=http://localhost:8080 npx playwright test e2e/tests/settings/organization-switch.spec.ts --project=chromium` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
  - result: `10 passed, 2 skipped`
- `npx eslint src/components/layout/AppLayout.vue src/components/layout/OrganizationSwitcher.vue src/components/layout/UserMenu.vue e2e/tests/settings/organization-switch.spec.ts` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP against `http://localhost:8080` during investigation:
  - confirmed the organization switcher was present for the authenticated sidebar flow
  - confirmed `/api/me/organizations` returned multiple organizations for the current admin user

### Notes

- `npm run typecheck` still fails in this repo because of unrelated pre-existing frontend typing issues outside the files changed in this session.
- A later follow-up browser MCP smoke using the Ruflo browser adapter was unavailable in this environment (`agent-browser` missing), so the final interaction verification relied on the passing Playwright regression against the live app on `localhost:8080`.

## 2026-04-02 22:38

### Completed

- Reworked the authenticated shell in `frontend/src/components/layout/AppLayout.vue` so the desktop sidebar now behaves like the KLiK PoS rail:
  - collapsed by default
  - expands on hover and focus-within
  - can be pinned open with persisted state in `localStorage` under `layout.sidebarPinnedOpen`
  - overlays content while the main pane keeps the collapsed reserved offset
- Updated `frontend/src/router/index.ts` and `frontend/src/components/layout/navigation.ts` so authenticated `/` now resolves through the permission fallback order, `/chat` is the first default destination, and the dashboard remains directly accessible at `/dashboard`.
- Updated `frontend/src/components/layout/OrganizationSwitcher.vue` and `frontend/src/components/layout/UserMenu.vue` to follow the new expanded/collapsed rail behavior without hard remounts.
- Fixed the dashboard shortcuts behavior in `frontend/src/views/dashboard/DashboardView.vue` so quick actions only render routes the current user can actually access, and shortcut widget editing now saves only accessible entries.
- Added and updated Playwright coverage for the new shell and routing behavior across:
  - `frontend/e2e/tests/chat/sidebar-hover.spec.ts`
  - `frontend/e2e/tests/auth/login.spec.ts`
  - `frontend/e2e/tests/dashboard/dashboard.spec.ts`
  - `frontend/e2e/tests/dashboard/dashboard-permissions.spec.ts`
  - `frontend/e2e/tests/settings/permissions.spec.ts`
  - `frontend/e2e/tests/settings/language-switch.spec.ts`
  - `frontend/e2e/tests/settings/organization-switch.spec.ts`

### Skills Applied

- `vue-expert` for the Vue 3 shell state, sidebar interaction model, and router redirect changes
- `playwright-expert` for the E2E updates, selector hardening, and browser regression verification

### Verification

- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- `npx playwright test e2e/tests/chat/sidebar-hover.spec.ts e2e/tests/dashboard/dashboard.spec.ts e2e/tests/settings/permissions.spec.ts` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- `npx playwright test e2e/tests/auth/login.spec.ts e2e/tests/dashboard/dashboard.spec.ts e2e/tests/dashboard/dashboard-permissions.spec.ts e2e/tests/settings/permissions.spec.ts e2e/tests/settings/language-switch.spec.ts e2e/tests/settings/organization-switch.spec.ts e2e/tests/chat/sidebar-hover.spec.ts` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
  - result: `66 passed, 2 skipped`
- Chrome DevTools MCP against `http://127.0.0.1:8080`:
  - confirmed authenticated `/` resolves to `/chat`
  - confirmed the desktop sidebar changes from `56px` collapsed to `224px` expanded on hover
  - confirmed main content padding stays at `56px` while the expanded sidebar overlays instead of reflowing the page
  - confirmed pinning sets `data-sidebar-pinned=\"true\"` and persists `layout.sidebarPinnedOpen=true`
  - confirmed unpinning plus focus removal returns the sidebar to the collapsed state

### Notes

- Chrome DevTools reported one browser issue warning: a form field without an `id` or `name` attribute. No new JavaScript runtime errors appeared during the verification flow.

## 2026-04-02 22:10

### Completed

- Moved the shared notification bell into the chat statuses toolbar so it now sits directly beside the statuses refresh button in `frontend/src/components/chat/status/StatusStoriesBar.vue`.
- Removed the old notification bell placements from `frontend/src/components/layout/AppLayout.vue` so the bell is no longer duplicated in the global shell.
- Added a compact trigger mode to `frontend/src/components/NotificationBell.vue` so the bell matches the status toolbar control sizing.
- Hardened `frontend/src/components/NotificationBell.vue` to accept both notifications API payload shapes:
  - bare array responses
  - object responses like `{ notifications, total }`
- Extended `frontend/e2e/tests/chat/statuses.spec.ts` to verify:
  - the bell renders inside the statuses bar
  - the bell sits between the refresh button and drawer toggle
  - opening the bell does not open the statuses drawer

### Skills Applied

- `vue-expert` for the Vue 3 component move and compact trigger integration
- `playwright-expert` for the browser verification path and E2E coverage update

### Verification

- `npx playwright test e2e/tests/chat/statuses.spec.ts` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Playwright MCP against `http://127.0.0.1:3000/chat` with mocked API routes:
  - confirmed toolbar order by bounding boxes: refresh `x=466`, bell `x=490.5`, drawer toggle `x=515`
  - confirmed the notifications popover opens successfully
  - confirmed the statuses drawer remains closed when the bell is clicked

### Notes

- `npm run typecheck` still fails because of pre-existing frontend typing issues in unrelated files and stores; this task did not resolve that broader baseline.

## 2026-04-02 21:02

### Completed

- Rethemed the header notifications surface in `frontend/src/components/NotificationBell.vue` to remove the remaining old black/green treatment.
- Updated the popover shell, action row, notification rows, and count badges to use the Twitter token palette:
  - popover surface now uses `card/popover` tokens with the shared border/shadow language
  - unread/message count pills now use the `primary` token instead of green
  - the bell counter badge now matches the new primary blue styling
  - action buttons now use token-based primary/muted surfaces instead of legacy dark-outline styles

### Verification

- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP on `http://127.0.0.1:4173/chat`
  - verified the notifications popover in `light` mode
  - verified the notifications popover in `dark` mode
  - confirmed the old green notification accents are no longer present in the updated component styling

### Notes

- The final browser state used the existing local session and did not require re-authentication.

## 2026-04-02 20:43

### Completed

- Removed the remaining green chat accents that were still visible on grouped media/file messages in `frontend/src/views/chat/ChatView.vue`, `frontend/src/components/chat/MediaGroupBar.vue`, `frontend/src/components/chat/status/StatusStoriesBar.vue`, and `frontend/src/assets/index.css`.
- Swapped the last green connector, badge, download pill, status ring, and emoji-picker focus accents onto the Twitter theme `primary` token so the chat page no longer mixes the old WhatsApp-style green with the new blue palette.
- Rethemed chat archive/delete controls to be explicit Twitter-style actions instead of the old orange/red treatment:
  - sidebar chat list action buttons in `frontend/src/views/chat/ChatView.vue`
  - contact info panel archive/delete buttons in `frontend/src/components/chat/ContactInfoPanel.vue`
  - supporting destructive action buttons in the chat thread/info panel for visual consistency

### Verification

- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP visual QA on `http://127.0.0.1:4173/chat/29c4b6d3-c54c-45ac-8ecf-cc25f4780c88` in `light` mode:
  - confirmed grouped file/media bubbles no longer show green rails or green download controls
  - confirmed the sidebar chat-row archive button is blue-tinted and the delete button uses the destructive tint
  - confirmed the composer and surrounding chat shell still render correctly after the action-button retheme

### Notes

- The route still shows the pre-existing Vue warnings about `data-testid` attributes on teleported dialog components and the form-field accessibility issue reported by Chrome DevTools; those are unrelated to this theme pass.

## 2026-04-02 20:21

### Completed

- Rechecked the remaining authenticated surfaces after the initial Twitter-theme rollout and closed the last obvious old-style gaps on the real routed pages.
- Rethemed the chat experience more completely in `frontend/src/views/chat/ChatView.vue` plus the supporting chat surfaces in `frontend/src/components/chat/ConversationNotes.vue` and `frontend/src/components/chat/status/StatusStoriesBar.vue`, including the thread header, message area, composer, sidebar states, and notes panel.
- Added a legacy utility compatibility layer in `frontend/src/assets/index.css` so older dark-first utility combinations now resolve through the Twitter token palette while broader source cleanup continues.
- Updated the shared page chrome in `frontend/src/components/shared/PageHeader.vue` to use token-based card/border/text styles and to support both `description` and legacy `subtitle` props, which fixed missing page subtitles across multiple routes.
- Reduced direct hardcoded dark/light utility usage in the highest-visibility remaining route views:
  - `frontend/src/views/settings/SettingsView.vue`
  - `frontend/src/views/settings/AccountsView.vue`
  - `frontend/src/views/chatbot/ChatbotView.vue`
  - `frontend/src/views/analytics/MetaInsightsView.vue`
- Brought those views onto token-driven cards, tabs, form controls, muted text, status badges, and helper surfaces so they visually align with the Twitter theme instead of the previous dark-first styling.

### Verification

- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP visual QA in forced `light` mode:
  - `http://127.0.0.1:4173/chat/b74832b5-bc6e-4495-8446-83c29dd93f0a`
  - `http://127.0.0.1:4173/settings`
  - `http://127.0.0.1:4173/chatbot`
  - `http://127.0.0.1:4173/analytics/meta-insights`
- Confirmed the chat thread now renders with the new sidebar, header, bubble, and composer styling instead of the old screenshot appearance.
- Confirmed the settings header now shows its subtitle and the tabs/cards/forms inherit the Twitter token styling.
- Confirmed the chatbot overview and meta-insights shell now use the updated token-based header/card/tabs treatment in light mode.

### Notes

- `prettier` still reports a parser error for `frontend/src/views/analytics/MetaInsightsView.vue`, but `vite build` succeeds and the route loads correctly in the browser; this appears to be a formatting/parser issue rather than a runtime break.
- The Meta analytics API endpoints returned `404` during the Chrome DevTools check:
  - `/api/analytics/meta/accounts`
  - `/api/analytics/meta?...`
    Those are backend/data availability issues in this local environment, not theme regressions.

## 2026-04-02 19:51

### Completed

- Applied the Twitter theme token set to the real Vue frontend in `frontend/src/assets/index.css`, including light/dark palettes, Twitter-style radii, flatter shadows, sidebar/chart tokens, and the new shared `auth-shell`, `auth-card`, `widget-surface`, and widget typography helpers.
- Updated `frontend/tailwind.config.cjs` to consume the new RGB CSS variables directly, expose sidebar/chart colors, and map font, radius, shadow, and letter-spacing tokens for shadcn-vue components.
- Switched font delivery to local package-based `Open Sans` via `@fontsource/open-sans` in `frontend/src/assets/fonts.css` and `frontend/package.json`, while keeping `Georgia` and `Menlo` as stack fonts through CSS variables.
- Refactored theme bootstrap in `frontend/index.html` and `frontend/src/composables/useColorMode.ts` so the app defaults to `system`, always applies either `.light` or `.dark` before mount, and keeps `color-scheme` in sync with the active mode.
- Restyled the phased-core surfaces to use tokens instead of ad hoc dark-first classes:
  - shared primitives in `frontend/src/components/ui/*`
  - app shell chrome in `frontend/src/components/layout/*`
  - notification/user popovers
  - auth screens in `frontend/src/views/auth/*`
  - dashboard widgets and dialogs in `frontend/src/views/dashboard/DashboardView.vue`

### Verification

- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend`
- Chrome DevTools MCP on `http://127.0.0.1:4173/login`:
  - verified explicit `light` mode sets `document.documentElement.className` to `light`
  - verified explicit `dark` mode sets `document.documentElement.className` to `dark`
  - verified `system` mode follows emulated `prefers-color-scheme` light and dark correctly
  - observed no console warnings or errors during the above checks
  - confirmed `/api/auth/sso/providers` returned `200` in this environment
- Chrome DevTools MCP authenticated smoke with `admin@test.com` / `Password123!`:
  - login request returned `200`
  - dashboard shell, widgets, notifications, and shortcuts loaded successfully
  - widget/data endpoints returned `200`
  - observed no console warnings or errors on the authenticated dashboard
- Chrome DevTools MCP narrow-viewport smoke:
  - verified the authenticated shell and dashboard still render at a small viewport without a fatal layout break

### Notes

- This pass intentionally stopped at the high-visibility shared surfaces and dashboard/auth flows; the repo still contains broader legacy hardcoded color utilities outside the scoped Twitter-theme rollout.

## 2026-04-02 19:08

### Completed

- Removed the embedded pricing/plans/offers page implementation from the main frontend by deleting `frontend/src/views/public/PricingLandingView.vue`.
- Replaced `/pricing`, `/plans`, and `/offer` with a marketing-sidecar handoff view backed by `VITE_PUBLIC_MARKETING_BASE_URL`.
- Added `frontend/src/lib/marketing-redirect.ts` and `frontend/src/lib/marketing-redirect.test.ts`.
- Generalized backend lead source validation in `internal/models/lead_request.go` and `internal/handlers/lead_requests.go` so future sidecar submissions are not locked to pricing-only metadata.
- Updated architecture and state docs for the new handoff boundary.

### Verification

- `npx vitest run src/lib/marketing-redirect.test.ts` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/frontend`
- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/frontend`
- `go build ./cmd/whatomate` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan`
- Chrome DevTools MCP:
  - loaded `http://127.0.0.1:4173/pricing`
  - loaded `http://127.0.0.1:4173/plans`
  - loaded `http://127.0.0.1:4173/offer`
  - confirmed all three routes render the new sidecar-handoff page
  - confirmed no console warnings or errors

### Blockers / Notes

- `npm run typecheck` still fails because of pre-existing frontend typing issues in contacts/auth/chatbot modules unrelated to this task.
- `go test ./internal/handlers -run 'TestApp_(CreatePublicLeadRequest|ListLeadRequests|UpdateLeadRequestStatus)$' -count=1` still fails to compile because `internal/handlers/campaigns_test.go` depends on a stale `testutil.MockQueue` missing `EnqueueContactRepair`.

## 2026-04-02 18:58

### Completed

- Created a dedicated worktree at `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan` on branch `codex/sidecar-removal-plan`.
- Reverse-engineered the current public pricing surface and documented the safe sidecar migration plan in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/specs/pricing-sidecar-removal_design.md`.
- Updated `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/PLAN.md` to point the next implementation pass at the new design artifact.
- Applied only the relevant planning skills for this task:
  - `architecture-guardian`
  - `spec-miner`

### Key Findings

- `/pricing`, `/plans`, and `/offer` are one public route group implemented by `frontend/src/router/index.ts` and `frontend/src/views/public/PricingLandingView.vue`.
- The marketing page is coupled to the existing lead workflow through `POST /api/public/lead-requests`.
- Backend validation in `internal/handlers/lead_requests.go` currently hardcodes `source_page=pricing` and allows only `/pricing`, `/plans`, and `/offer` as `source_route`.
- The authenticated admin lead inbox at `/settings/lead-requests` is a separate concern and should stay alive during the first sidecar migration phase.
- There is no dedicated automated E2E coverage today for the public pricing aliases.

### Verification

- `npm ci` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/frontend`
- `npm run build` in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-sidecar-removal-plan/frontend`
- Chrome DevTools MCP smoke:
  - loaded `http://127.0.0.1:4173/pricing`
  - loaded `http://127.0.0.1:4173/plans`
  - loaded `http://127.0.0.1:4173/offer`
  - no console warnings or errors observed across the checked routes

### Notes

- The first build attempt in the new worktree failed because the worktree did not yet have `frontend/node_modules`; installing dependencies in the worktree resolved that.
- The recommended migration seam is a redirect/proxy handoff for the public routes while preserving the monolith lead-ingestion contract initially.

## 2026-03-30 12:22

### Completed

- Built and deployed the CSP nonce update for inline theme initialization; created a fresh backup of the previously installed binary.
- Restarted `whatomate` plus tenant systemd services.
- Updated deployment docs and synced them to the VPS.

### Verification

- Local HTTP smoke (VPS): `ofuqalmadenah.com` -> `200`, `holol-wenjaz` -> `200`, `alarkan-almthalia` -> `200`, `matbaat-ruya` -> `200`.
- CSP header includes `script-src 'self' 'nonce-...'`, and the inline theme script includes a matching `nonce`.
- Playwright MCP loaded `https://ofuqalmadenah.com/settings` and `https://ofuqalmadenah.com/chat` with no CSP inline-script errors (only expected `401` responses due to unauthenticated session).

### Notes

- Chrome DevTools MCP was unavailable due to a profile lock; Playwright MCP was used for UI verification.
- `whatomate-housekeeping.service` is in `failed` state (pre-existing).

## 2026-03-30 11:58

### Completed

- Backed up `/opt/whatomate/bin/whatomate` before deployment and synced the updated frontend build to the VPS.
- Built with `make build-prod`, installed the new binary, and restarted `whatomate` plus all tenant services.
- Updated `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/whatomate_multi_instances_info.md` and synced it to `/root/whatomate_multi_instances_info.md` and `/root/whatomate_production_info.md`.

### Verification

- Local HTTP smoke (VPS): `ofuqalmadenah.com` -> `200`, `holol-wenjaz` -> `200`, `alarkan-almthalia` -> `200`, `matbaat-ruya` -> `200`.
- Playwright MCP loaded `https://ofuqalmadenah.com/chat` with no console errors reported.

## 2026-03-30 12:05

### Completed

- Moved the inline theme-init script to `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/public/theme-init.js` and referenced it from `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/index.html` to satisfy CSP `script-src 'self'`.
- Removed the `grid-layout` manual chunk split in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/vite.config.ts` to avoid the circular chunk and runtime `ReferenceError` in `grid-layout`.
- Rebuilt the frontend and verified the login screen renders in Vite preview without console errors.

### Verification

- `npm run build` (frontend) succeeded.
- Playwright MCP loaded `http://127.0.0.1:4173/login` with no console errors and the login form present.

## 2026-03-30 11:45

### Completed

- Backed up the existing production binary on the VPS before deploy.
- Synced the local workspace to `/opt/whatomate-src`, built with `make build-prod`, and installed the new binary to `/opt/whatomate/bin/whatomate`.
- Restarted `whatomate` and tenant services, verified local HTTP 200s.
- Updated deployment logs in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/whatomate_multi_instances_info.md` and synced to `/root/whatomate_multi_instances_info.md` + `/root/whatomate_production_info.md`.

### Verification

- Local HTTP smoke: `ofuqalmadenah.com` (127.0.0.1:18123) -> `200`
- Local HTTP smoke: `holol-wenjaz.ofuqalmadenah.com` (127.0.0.1:18124) -> `200`
- Local HTTP smoke: `alarkan-almthalia.ofuqalmadenah.com` (127.0.0.1:18125) -> `200`
- Local HTTP smoke: `matbaat-ruya.ofuqalmadenah.com` (127.0.0.1:18126) -> `200`
- MCP UI check (Playwright fallback): loaded `https://holol-wenjaz.ofuqalmadenah.com/login` (title `Whatomate`). Console reported CSP inline-script blocked and a `ReferenceError` in the `grid-layout` bundle.
- Chrome DevTools MCP could not start because a browser profile was already running.

## 2026-03-29 19:19

### Completed

- Hardened WebSocket contact subscription state with locking and updated tests.
- Resolved async send race by pre-resolving provider instance IDs before goroutines.
- Enforced JWT algorithm validation for invite tokens and logout refresh parsing.
- Batched unread contact counts with aggregate queries and fallback logic.
- Guarded media handler against invalid `message_id` assertions.
- Made auth `restoreSession` async with server-verified `/me` refresh and updated call sites.
- Marked `ResourceAPIKeys` security finding as a desloppify false positive.

### Tests

- `go test ./internal/websocket -count=1` (pass)
- `go test ./internal/handlers -count=1` (fail: `internal/handlers/campaigns_test.go` uses `testutil.MockQueue` missing `EnqueueContactRepair`)
- `npm run test:unit` (pass)
- `npm run test` (fail: Playwright suite reports widespread UI test failures; see output for details)

### Manual QA (MCP)

- Closed the existing Chrome DevTools MCP session; tool transport failed to restart after shutdown.
- Used Playwright MCP as fallback: login succeeded, chat page loaded with sidebar + message list, refresh returned to chat view, and logout returned to login screen.

## 2026-03-29 18:20

### Completed

- Ran SAST and secrets scans with Semgrep and performed a focused manual security review (auth, CSRF, SSRF, file upload paths, Dockerfiles).
- Ran dependency audits for root and frontend Node workspaces.
- Performed a basic Chrome DevTools load check of the local ACP guide page.

### Tests / Scans

- `semgrep --config=auto --exclude=node_modules --exclude=vendor --json --output semgrep_latest.json`
- `semgrep --config=p/secrets --exclude=node_modules --exclude=vendor --json --output semgrep_secrets.json`
- `npm audit --json > npm_audit_root.json` (root)
- `npm audit --json > npm_audit_frontend.json` (frontend, exit code 1 due to findings)

### Manual QA (Chrome DevTools)

- Opened `file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/acp_guide.html` and verified no console errors.

## 2026-03-29 18:25

### Completed

- Resolved high-severity frontend dependency vulnerabilities by upgrading `happy-dom` and enforcing safe transitive versions via npm overrides.
- Refreshed `npm_audit_frontend.json` with a clean audit result (0 vulnerabilities).
- Performed a post-change Chrome DevTools sanity check on the ACP guide page.

### Tests / Scans

- `npm install --package-lock-only`
- `npm audit --json > npm_audit_frontend.json`

### Manual QA (Chrome DevTools)

- Opened `file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/acp_guide.html` and verified no console errors.

## 2026-03-29 18:05

### Completed

- Added per-user chat soft-delete backend: `ContactUserDeletion` model/table, `/api/contacts/{id}/soft-delete` handler, deletion-aware contact list/message/unread filtering, and admin-only `chat_deleted_by_user` notifications with `contact_id` + metadata.
- Added frontend soft-delete actions (sidebar + contact panel), new API call, clickable notifications that open chats, and extended types for notification payloads.
- Added `en/ar/es` translations for soft delete UI and notification messaging.
- Authored design doc at `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/specs/chat-soft-delete_design.md`.

### Tests

- `go test ./internal/handlers -run Test -count=1` (fails: `internal/handlers/campaigns_test.go` uses `testutil.MockQueue` missing `EnqueueContactRepair`).

### Manual QA (Chrome DevTools)

- Opened `http://localhost:8080/chat` and verified pending chat list loads.
- “Hide chat” controls were not visible in the running instance (likely because the existing admin role lacked `contacts:soft_delete` until migrations/backfill are applied), so the end-to-end soft-delete flow could not be validated.

### Remaining

- Apply migrations/backfill in the running environment so admin/agent roles get `contacts:soft_delete`, then re-run UI checks for hide chat, admin notifications, and post-delete message visibility.

## 2026-03-29 15:22

### Completed

- Added `repairDirectContactPhoneFromConversation` wrapper to apply canonical direct-contact phone updates and enqueue background repair.
- Updated `resolveContactConversationContext` call sites to pass a context in system chat messages and contact responses.
- Re-ran `make run-migrate`; migrations progressed and server started without the previous 8080 bind error.

### Remaining

- Run `make run-migrate` without a timeout to let the server keep running if desired.
- Provide a base URL if you want Chrome DevTools-based UI verification.

### Verification

- `make run-migrate` (terminated after 15s to avoid leaving the server running)

## 2026-04-03 02:05

### Completed

- Analyzed the `/chat` page behavior for `Send Template`, `Assign to agent`, and `Transfer to Agent` from the Vue UI, frontend stores/services, and backend handlers.
- Verified the live `/chat` page with Chrome DevTools MCP on `http://localhost:8080` after logging in as `admin@test.com`.
- Confirmed `Assign Contact` dialog opens from the header action, `Transfer to Agent` is present as a header action, and the composer `Send Template` control opens the template picker.

### Key Findings

- `Send Template`
  - UI entry points:
    - Composer template icon in `frontend/src/views/chat/ChatView.vue`
    - Service-window banner CTA when the 24-hour window is expired
  - Behavior:
    - Opens `TemplatePicker`, loads only `APPROVED` templates, optionally filtered by selected account.
    - Selecting a template opens a preview / parameter dialog.
    - Sending calls `contactsStore.sendTemplate(...)`, which posts to `/api/messages/template`.
    - Backend validates approved status, resolves account, validates required params, and sends/stores a template message.
  - Primary use case:
    - Re-engagement or compliant outbound messaging, especially when freeform WhatsApp replies are blocked by the service window.

- `Assign to agent`
  - UI entry point:
    - Header action with `UserPlus` icon in `frontend/src/views/chat/ChatView.vue`
  - Behavior:
    - Opens `Assign Contact` dialog with searchable assignable users.
    - Can assign or unassign via `contactsService.assign(...)` to `/api/contacts/{id}/assign`.
    - Backend updates `assigned_user_id` and chat lifecycle status using `chatAssignmentUpdates(...)`.
    - Assignment emits a system chat message when the assignee changes.
  - Primary use case:
    - Explicit ownership routing of a chat to a specific human, without creating a chatbot transfer record.

- `Transfer to Agent`
  - UI entry point:
    - Header action with `UserX` icon in `frontend/src/views/chat/ChatView.vue`
  - Behavior:
    - Calls `chatbotService.createTransfer(...)` to `/api/chatbot/transfers`.
    - Backend creates an active `AgentTransfer`, may assign directly or send to queue, cancels active chatbot session, and refreshes transfer state.
    - After a transfer exists, the UI swaps this action for `Resume Chatbot`.
  - Primary use case:
    - Escalation / handoff from chatbot automation to human handling, with queueing, SLA, and resume semantics.

### Manual QA (Chrome DevTools)

- Logged into `http://localhost:8080/login` using `admin@test.com`.
- Opened `/chat`, selected a pending conversation, and confirmed:
  - `Assign to agent` tooltip and dialog are present.
  - `Transfer to Agent` tooltip is present on the header action.
  - `Send Template` tooltip is present on the composer control.
  - Clicking `Send Template` opens the template picker; in the current dataset it showed `No approved templates`.

## 2026-04-03 02:20

### Completed

- Removed the `/chat` template-send UI path:
  - deleted the composer `Send Template` control
  - removed the template parameter dialog and related handlers from `frontend/src/views/chat/ChatView.vue`
  - removed the frontend store/API helpers for sending templates
  - deleted the unused `frontend/src/components/chat/TemplatePicker.vue`
  - removed the dedicated template-sending Playwright spec and cleaned the chat page object
- Kept the service-window warning banner in `/chat`, but removed the legacy CTA button from it.
- Added `/settings/contacts` instance awareness:
  - new instance filter wired to `contactsService.list({ instance_id })`
  - new `WhatsApp Instance` column showing the resolved instance name and falling back to the raw instance id or `None`
  - export dialog filters now include the selected instance
  - empty-state messaging now treats instance filtering like a filtered result set instead of “no contacts yet”

### Validation

- `frontend`: `npx vitest run src/stores/contacts.test.ts` ✅
- `frontend`: file-scoped ESLint on the touched files ✅
- `frontend`: `npm run typecheck` ❌
  - still failing due pre-existing unrelated issues in:
    - `src/components/ui/toast/use-toast.ts`
    - `src/stores/roles.ts`
    - `src/views/chat/ChatView.vue` (`body` typing)
    - `src/views/chatbot/AgentTransfersView.vue`
    - `src/views/chatbot/ChatbotFlowBuilderView.vue`
    - `src/views/dashboard/DashboardView.vue`
    - `src/views/settings/TeamsView.vue`

### Manual QA (Chrome DevTools)

- `/chat`
  - logged in on `http://localhost:8080`
  - opened a pending conversation (`noiemany`)
  - confirmed there is no `Send Template` button or template icon/button in the active chat UI
- `/settings/contacts`
  - confirmed the new `WhatsApp Instance` column is visible
  - confirmed the new instance filter renders with `All instances` and `n0n`
  - selected `n0n` and verified:
    - the filter value changed to `n0n`
    - results count changed from `Showing 1 to 20 of 181 contacts` to `Showing 1 to 20 of 128 contacts`
    - rows with `None` as the instance were excluded from the filtered result

## 2026-04-04 16:31

### Skills Used

- `fullstack-guardian`
  - used because this change crossed the Vue chat UI, API payload, backend contact creation flow, and live browser verification
- `golang-pro`
  - used for the WhatsMeow-backed Go handler changes, resolver design, and table-driven backend tests

### Completed

- Changed the `/chat` create-contact flow into a WhatsMeow-driven direct chat flow when the provider is WhatsMeow:
  - the dialog now opens as `Start New Chat`
  - it requires an international phone number and a sending WhatsApp instance
  - it hides the old WhatsApp account selector in this mode
  - it sends `instance_id` plus `start_chat: true` instead of requiring a stored contact/account first
- Added backend support for direct-chat contact creation:
  - validates and normalizes international numbers
  - resolves the selected outbound instance
  - checks the destination with WhatsMeow before creating/restoring the contact
  - hydrates the contact profile name from the WhatsApp verified business/public profile when available
  - restores or creates the contact in an open/assigned chat-ready state
- Added targeted regression coverage for both the Go handler flow and the Vue dialog payload/behavior.

### Validation

- `go test -race ./internal/handlers -run 'TestNormalizeWhatsmeowDirectChatPhone|TestCreateContact_StartChat'` ✅
- `pnpm --dir frontend exec vitest run src/components/shared/CreateContactDialog.test.ts` ✅
- `pnpm --dir frontend typecheck` was checked during this session and still fails due unrelated pre-existing issues elsewhere in the repo, including:
  - `frontend/src/components/ui/toast/use-toast.ts`
  - `frontend/src/stores/contacts.ts`
  - `frontend/src/stores/roles.ts`
  - `frontend/src/views/chat/ChatView.vue`
  - `frontend/src/views/chatbot/AgentTransfersView.vue`
  - `frontend/src/views/chatbot/ChatbotFlowBuilderView.vue`
  - `frontend/src/views/dashboard/DashboardView.vue`
  - `frontend/src/views/settings/TeamsView.vue`

### Manual QA (Chrome DevTools)

- Opened `http://localhost:8080/chat` in Chrome DevTools and verified the add-contact action now opens:
  - title: `Start New Chat`
  - description: `Choose the sending WhatsApp instance and enter an international phone number to open a direct chat.`
  - fields shown: phone number, profile name, WhatsApp instance
  - old WhatsApp account selector is not shown in chat mode
- Entered an invalid number (`123`) and confirmed the browser toast:
  - `Enter a valid international phone number with country code.`
- Submitted a valid-format number and inspected the live request:
  - `POST /api/contacts`
  - request body:
    - `{"phone_number":"+12025550100","instance_id":"5cdb3701-8f23-4673-ab42-5492b226ab41","start_chat":true}`
  - response:
    - `400`
    - `phone number is not registered on WhatsApp`
- This confirms the UI is hitting the new backend flow and that server-side WhatsMeow validation is active.

## 2026-04-05 11:08

### Skills Used

- `fullstack-guardian`
  - used because this change moved a feature across Vue settings screens, instance APIs, Go handlers, worker logic, and data migration/backfill
- `playwright-expert`
  - used for targeted E2E coverage and live browser verification of the moved settings surface

### Completed

- Moved `Assigned Chat Reset` off org-level `/settings` chat preferences and into each instance card on `/settings/instances`:
  - removed the old org-level UI and save path
  - added a per-instance panel with switch, summary, dialog, mode/hour controls, and timezone hint
  - added instance-scoped locale strings and frontend helpers for normalization/sanitization
- Changed backend ownership of the feature from organization settings to `whatsapp_instances.settings`:
  - org settings no longer expose or persist `assigned_chat_reset_*`
  - instance list/get/update now normalize and preserve assigned-reset settings alongside other instance settings
  - added per-instance defaults/validation in the shared WhatsMeow instance settings pipeline
- Refactored the assigned reset worker and rollout path:
  - worker now iterates instances, uses org timezone for schedule evaluation, and resets only contacts whose `instance_id` matches the instance being processed
  - `assigned_chat_reset_last_date` is now stored on the instance
  - added idempotent legacy backfill from organization settings to instances and invoked it in both migration flow and server startup
- Added regression coverage for the new contract:
  - Go tests for instance assigned-reset defaults/validation, org settings contract removal, instance response defaults, worker scoping, and backfill behavior
  - Playwright coverage for the new instance-level UI plus absence of the old org-level controls
  - updated Playwright global setup so the seeded superadmin password can be provided from env for local verification

### Findings

- `frontend`: `npm run typecheck` still fails due unrelated pre-existing issues outside this feature slice, including:
  - `src/components/shared/CreateContactDialog.test.ts`
  - `src/components/ui/toast/use-toast.ts`
  - `src/stores/contacts.ts`
  - `src/stores/roles.ts`
  - `src/views/chat/ChatView.vue`
  - `src/views/chatbot/AgentTransfersView.vue`
  - `src/views/chatbot/ChatbotFlowBuilderView.vue`
  - `src/views/dashboard/DashboardView.vue`
  - `src/views/settings/TeamsView.vue`
- `backend/frontend serving`: `go run ./cmd/whatomate server -config config.toml -migrate -workers 0` still rendered the embedded fallback page (`Frontend not embedded: index.html not found...`) on `:8080`, so live UI verification was completed against the Vite app on `:3000` with the real backend/API on `:8080`

### Verification

- `go test ./pkg/whatsmeow -run 'AssignedChatReset|EnsureInstanceSettingsDefaults_InjectsAssignedChatResetDefaults|ValidateInstanceSettings_AssignedChatReset'` ✅
- `go test ./internal/database -run 'BackfillInstanceAssignedChatResetSettings'` ✅
- `go test ./internal/handlers -run 'AssignedChatReset|OrganizationSettings|InjectsAssignedChatResetDefaults'` ✅
- `frontend`: `npm run build` ✅
- `frontend`: `E2E_SUPERADMIN_PASSWORD=adminpassword12 npx playwright test e2e/tests/settings/instances.spec.ts e2e/tests/settings/general-settings.spec.ts` ✅ (`41 passed`)

### Manual QA (Chrome DevTools)

- Opened `http://localhost:3000/settings` and switched to the `Chat` tab:
  - confirmed the org-level `Assigned Chat Reset` controls are no longer present
- Opened `http://localhost:3000/settings/instances`:
  - confirmed the instance card shows `Assigned Chat Reset`
  - confirmed the live summary is rendered on the instance card (`Daily at 00:00 (UTC)`)
  - opened `Configure assigned chat reset` and verified the dialog shows the instance-level schedule controls and timezone hint
  - reloaded `/settings/instances` and confirmed the assigned-reset card/summary remained visible after reload

### Notes

- Live backend/API verification used the local seeded credentials from `config.toml`:
  - superadmin: `admin@admin.com`
  - Playwright setup override: `E2E_SUPERADMIN_PASSWORD=adminpassword12`

## 2026-04-05 11:18 - Instance settings cards side-by-side

### Goal

- Make the instance setting cards on `/settings/instances` render side by side on desktop, similar to the compact stats cards layout, while preserving stacked mobile behavior.

### What changed

- updated the instance card settings section to use a responsive two-column grid for:
  - `Auto-sync history`
  - `Auto-download incoming media`
  - `Call auto-reject`
  - `Auto campaign`
  - `Chat Close Rating Settings`
  - `Assigned Chat Reset`
- relaxed the summary text truncation inside those cards to allow a two-line clamp so the narrower desktop cards still read cleanly

### Verification

- `frontend`: `npm run build` ✅
- Chrome DevTools on `http://localhost:3000/settings/instances` at desktop width ✅
  - `Auto-sync history` and `Auto-download incoming media` share the first row
  - `Call auto-reject` and `Auto campaign` share the second row
  - `Chat Close Rating Settings` and `Assigned Chat Reset` share the third row

## 2026-04-05 11:24 - Instance card fit fix

### Goal

- Fix the `/settings/instances` card layout so setting blocks and configure buttons fit inside the card cleanly across narrower desktop widths, matching the issue shown in the screenshot.

### What changed

- widened instance cards at normal desktop widths by changing the page grid to use three columns only at `2xl`
- changed the inner settings grid to switch to two columns only at `xl`, so narrow cards no longer force cramped two-column content
- updated the configure buttons for:
  - `Call auto-reject`
  - `Auto campaign`
  - `Chat Close Rating Settings`
  - `Assigned Chat Reset`
- those buttons now allow wrapped text and taller button height instead of clipping text inside narrow cards

### Verification

- `frontend`: `npm run build` ✅
- Chrome DevTools on `http://localhost:3000/settings/instances` ✅
  - at `1280px` viewport the instance card rendered wide enough for a clean two-column settings layout
  - at `1024px` viewport the settings stacked into one column and the configure buttons expanded without clipping

## 2026-04-05 11:29 - Chat Source Tag side-by-side layout

### Goal

- Make the `Chat Source Tag` form use horizontal space better on `/settings/instances` by placing the controls side by side on wider screens instead of leaving them fully stacked.

### What changed

- moved `Custom Label` and `Show as` into a responsive two-column desktop grid
- moved the `Tag Color` swatches and `Save Tag Settings` action into a shared desktop row that still wraps safely if space gets tight

### Verification

- `frontend`: `npm run build` ✅
- Chrome DevTools on `http://localhost:3000/settings/instances` at `1280px` ✅
  - `Custom Label` and `Show as` rendered on the same row
  - `Tag Color` and `Save Tag Settings` rendered side by side on the following row
