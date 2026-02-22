# Memory

## 2026-02-22 01:07 Feature: Canned Responses with WhatsApp Rich Text + Media Attachments

- **The Trap:** Canned responses were text-only and provided no authoring support for WhatsApp-native styling syntax, so agents had to manually format and manually attach recurring media.
- **The Reality:** Support teams need reusable quick replies that can bundle rich text (`*bold*`, `_italic_`, `~strike~`, `\`\`\`mono\`\`\``) and recurring photo/video assets in one repeatable flow.
- **The Fix:** Added typed canned-response attachment storage (`attachments` JSONB), multipart create/update support for image/video files, secure attachment lifecycle cleanup on update/delete, and a new `POST /api/canned-responses/{id}/send` endpoint that dispatches canned text + stored attachments via existing outbound messaging pipelines. Upgraded frontend canned-response editor with a WhatsApp rich-text toolbar + placeholder inserts, media attachment management UI, and chat-side send integration that dispatches bundled attachments when selected canned responses include media.
- **The Law:** If a canned reply is operational content, treat it as a composable message bundle (format + media) and send it through the same provider-safe pipeline as normal outbound messages.

## 2026-02-21 21:11 Feature: Scheduled Assigned-Chat Reset to Pending

- **The Trap:** Relying on manual queue cleanup leaves assigned chats stale across shifts and blocks fair queue re-entry.
- **The Reality:** Assigned chats need an organization-level daily reset schedule that is timezone-aware and admin-configurable, while avoiding first-run destructive resets for default midnight behavior.
- **The Fix:** Added org settings for `assigned_chat_reset_mode` (`midnight` or `custom_hour`) and `assigned_chat_reset_hour` (0-23), added a background `ChatAssignmentResetWorker` that runs every minute, resets assigned active chats to pending once per local day, persists a per-org last-run date, and broadcasts `contact_update` events for refreshed queue state.
- **The Law:** If queue ownership must expire on schedule, enforce it server-side with idempotent per-day execution keyed to organization timezone and persisted last-run state.

## 2026-02-21 02:14 Issue: Mobile-Sent WhatsApp Messages Were Missing from Live Chat Thread

- **The Trap:** Treating all `IsFromMe` realtime events as duplicates of dashboard/API sends.
- **The Reality:** Messages sent from the linked phone arrive as `IsFromMe` events too; dropping them prevented live synchronization into the chat thread and hid outgoing sent state until history refresh.
- **The Fix:** Updated `handleMessage` to accept `IsFromMe` events only when `DeviceSentMeta` is present (device-sent sync), then persist/broadcast as outgoing with `sent` status.
- **The Law:** For multi-device WhatsApp flows, distinguish self-origin runtime echoes from cross-device sync events before filtering `IsFromMe`.

## 2026-02-19 05:46 Issue: CORS/WS Empty-Origin Policy Was Over-Permissive

- **The Trap:** Treating an empty `allowed_origins` configuration as a development convenience that allows any origin.
- **The Reality:** With cookie auth and websocket token flows, permissive origin fallback expands cross-origin attack surface and weakens boundary assumptions.
- **The Fix:** Replaced allow-all fallback with a unified safe policy: explicit allowlist when configured, otherwise same-origin + localhost loopback only; applied the same evaluator to both CORS and websocket `CheckOrigin`.
- **The Law:** If credentials are in play, origin policy must fail closed by default and be identical across HTTP CORS and websocket upgrades.

## 2026-02-19 05:40 Issue: Fail-Fast JWT Validation Broke Startup UX for Default Configs

- **The Trap:** Enforcing strict JWT secret validation without immediately updating default config/docs creates a startup dead-end for users who follow copy-paste setup steps.
- **The Reality:** The previous placeholder secret became intentionally blocked, so fresh/local setups failed at boot until users manually discovered a valid replacement.
- **The Fix:** Replaced placeholder JWT defaults in config templates with explicit required-secret instructions, improved validation errors with remediation text, and documented the exact generation command (`openssl rand -hex 32`).
- **The Law:** Any security hardening that turns warnings into fatal startup checks must ship with synchronized setup docs and actionable operator guidance in the same change.

## [2026-02-19] Issue: JWT Secret Misconfiguration Could Fail Open or Fail Silently

- **The Trap:** Allowing startup with an empty/default JWT secret and suppressing token-generation errors in some auth paths.
- **The Reality:** Empty or predictable JWT secrets are a security-critical misconfiguration; ignored signing errors can produce partial auth responses and hard-to-diagnose authentication failures.
- **The Fix:** Added centralized JWT validation (`ValidateJWTSecret`) with fail-fast startup checks, added middleware fail-safe for blank secret, centralized handler key access, and replaced ignored token-generation errors with explicit 500 handling.
- **The Law:** JWT signing key configuration must fail fast at process startup and all token issuance paths must propagate signing errors explicitly.

## [2026-02-19] Issue: WhatsApp Connect Stuck on "Please Wait" Until Manual Refresh

- **The Trap:** Assuming websocket QR/connected events are always delivered in-order and that a connect API response is enough to drive UI state.
- **The Reality:** Already-linked sessions could be marked connected without broadcasting `instance_connected`, and async connect failures only existed in server logs. The UI could remain in loading state forever without a reconciliation pass.
- **The Fix:** Emitted connected events in the direct `Connect` path and on `events.Connected`, broadcasted `instance_reconnect_failed` on async connect/reconnect errors, and added frontend state reconciliation/watchdog + QR modal error feedback.
- **The Law:** Event-driven connection UX must include deterministic reconciliation and explicit failure signaling; never rely on best-effort websocket delivery alone.

## [2026-02-18] Feature: Media File Grouping & Batch Download

- **The Trap:** Rendering every media message independently makes batch downloads tedious when multiple files arrive together.
- **The Reality:** WhatsApp sends multi-photo/document shares as separate messages within seconds. Users need a one-click download for all related files.
- **The Fix:** Created `useMediaGroups.ts` composable (timestamp-proximity grouping of consecutive incoming media within 60s), `MediaGroupBar.vue` (visual label + JSZip batch download), and wired both into `ChatView.vue` with a green left-border accent for grouped messages. Zero existing functions modified.
- **The Law:** Front-end grouping logic must be computed and side-effect-free; zip creation belongs in the download handler, not in the rendering pipeline.

## [2026-02-18] Feature: Configurable Media Grouping Window

- **The Trap:** Hardcoding the 60-second grouping window leaves no way for users to adjust sensitivity.
- **The Reality:** Different users send files at different cadences; a 60s window may be too tight or too loose depending on workflow.
- **The Fix:** Refactored `useMediaGroups.ts` to read the window from `localStorage` (`chat.mediaGroupWindowSeconds`), clamped 5–300s. Added a new "Chat" tab in `SettingsView.vue` with a dropdown offering 6 presets (15s, 30s, 60s, 2min, 3min, 5min). No backend changes needed.
- **The Law:** User-facing timing thresholds should always be configurable; use localStorage for per-user frontend preferences that don't need server persistence.

## [2026-02-18] Issue: Restart Left Instances Stuck in Connecting

- **The Trap:** Treating `connecting` as a durable status across process restarts, even for instances with no linked WhatsApp session.
- **The Reality:** After an unclean shutdown, unpaired instances (`jid` empty) could remain `connecting` in DB, so the UI rendered a permanent `Connecting...` state instead of allowing a fresh QR scan.
- **The Fix:** Added startup status reconciliation in the whatsmeow manager to reset stale `connecting` + empty `jid` rows to `disconnected`, and updated reconnect failure handling to explicitly set `disconnected`.
- **The Law:** Transient runtime states (like `connecting`) must be reconciled at startup; never persist them indefinitely without runtime ownership checks.

## [2026-02-18] Issue: QR Regeneration Endpoint Was a Connect Alias

- **The Trap:** Assuming the reconnect endpoint could regenerate QR codes while internally reusing the same connect logic.
- **The Reality:** `ReconnectInstance` called `ConnectInstance`, and the connection manager intentionally no-oped when a client was already in memory, so "refresh QR" could silently do nothing.
- **The Fix:** Changed reconnect to force `Disconnect` then async `Connect`, and wired a "Regenerate QR" button in the modal to call this endpoint with loading feedback.
- **The Law:** A refresh endpoint must enforce state transition semantics explicitly; aliases to idempotent connect paths are not enough for regeneration workflows.

## [2026-02-18] Issue: Duplicate Instance Names + QR Modal Reopen Stuck on Waiting

- **The Trap:** Assuming instance names were implicitly unique and resetting QR state on every connect click.
- **The Reality:** Instance creation/renaming had no uniqueness validation, and reopening the QR modal cleared the last valid QR code before any new websocket payload arrived.
- **The Fix:** Added normalized (trimmed, case-insensitive) per-org duplicate checks for `CreateInstance` and `UpdateInstance`; cached QR payloads per instance with timeout-aware reuse so modal reopen shows active QR immediately.
- **The Law:** Treat uniqueness as an explicit backend invariant, and never discard event-driven UI state on reopen unless replacement data is already available.

## [2026-02-18] Issue: Mentions and Edited Messages Falling Back to Unsupported

- **The Trap:** Treating group mention tokens in message text as final display values and assuming edit events always arrive pre-unwrapped in a text-ready shape.
- **The Reality:** WhatsApp mention text can contain LID-style numeric placeholders while the true identities are in `contextInfo.mentionedJID`; edited messages can arrive via `ProtocolMessage.MESSAGE_EDIT` wrappers that must be explicitly unwrapped.
- **The Fix:** Added mention-token normalization using `mentionedJID` resolution (including LID->PN lookup) and explicit MESSAGE_EDIT unwrapping in inbound extraction, plus regression tests for mention replacement and protocol edit parsing.
- **The Law:** For WhatsApp payloads, never trust visible text tokens alone; always cross-check context metadata (`mentionedJID`, protocol wrapper type) before rendering or persisting message content.

## [2026-02-18] Feature: Admin Chat Deletion + Instance/Type Chat Filtering

- **The Trap:** Treating chat deletion as a generic `contacts:delete` permission and assuming tag-only filtering would be enough for multi-instance operations.
- **The Reality:** Managers may still hold delete permissions in custom roles, but product requirements now require deletion to be admin-exclusive; operations also need filters by instance and conversation class (private/group/channel), not just tags.
- **The Fix:** Added role-aware admin deletion guard (`canDeleteAnyChat`) with super-admin override, added backend `ListContacts` filters for `instance_id` and `chat_types`, and wired frontend chat filter controls + contact creation instance selector to pass/store `instance_id`.
- **The Law:** When product semantics override generic RBAC permissions, enforce explicit role-gated behavior at the handler boundary and mirror the same constraints in the UI affordances.

## [2026-02-17] Feature: Configurable Multi-Instance Chat Tags (Frontend)

- **The Trap:** Assuming basic color-by-index labels are enough once `instance_id` is present on messages.
- **The Reality:** Agents need operational flexibility: per-instance custom labels/colors, display-mode switching (name/phone/custom), and overflow handling when many instances exist.
- **The Fix:** Added shared `instance-tag` utility module, moved instance badges from message bubbles to conversation rows in the chat sidebar, added a sidebar display-mode selector (name/phone/custom), and added `InstanceTagSettings.vue` in instance cards for per-instance custom label/color persisted in `instance.settings`.
- **The Law:** For multi-instance UX, a tag is not complete until it supports both identity controls (what text) and visual controls (which color), plus a scalable legend for >4 instances.

## [2026-02-17] Feature: Instance Tags on Messages

- **The Trap:** Backend `Message` model already has `InstanceID` but the API responses and WebSocket payloads never exposed it to the frontend.
- **The Reality:** `MessageResponse`, `buildMessagesResponse`, and `broadcastNewMessage` all needed explicit `instance_id` population. The GORM model's JSON tags don't auto-flow into hand-crafted response structs.
- **The Fix:** Added `InstanceID *string` to `MessageResponse`, populated it in 3 response builders + 1 WS payload. Created `InstanceTag.vue` pill badge with 8-color palette, auto-hides for single-instance setups.
- **The Law:** When adding frontend-visible fields from existing models, always check: (a) the response struct, (b) every response builder, (c) the WebSocket broadcast payload — all three must be updated independently.

## [2026-02-17] Issue: RefreshToken Loses Switched Organization

- **The Trap:** Assuming `RefreshToken` correctly carries forward the user's current org context.
- **The Reality:** `RefreshToken` loads the user from DB, which resets `OrganizationID` to the default, discarding the switched org stored in JWT claims.
- **The Fix:** After loading the user in `RefreshToken`, set `user.OrganizationID = claims.OrganizationID` before generating new tokens.
- **The Law:** Any token refresh or re-issue flow must preserve context from the existing JWT claims, not just reload from DB defaults.


## [2026-02-17] Issue: Build Failure in MigrationView (useToast)

- **The Trap:** Using a non-existent composable `@/composables/useToast` and assuming a `showToast` method name.
- **The Reality:** The project uses the standard `shadcn-vue` toast implementation via `@/components/ui/toast`, where the hook returns `toast` instead of `showToast`.
- **The Fix:** Updated `MigrationView.vue` to import `useToast` from `@/components/ui/toast` and call `toast({ title, description, variant })`.
- **The Law:** Always verify the project's standard component implementations (especially UI elements like Toasts) before assuming method names or import paths.

## [2026-02-17] Feature: Whatsmeow Integration (Foundation)

- **The Trap:** Assuming `config.example.toml` matched `config.toml` structure.
- **The Reality:** `config.example.toml` was missing `[whatsapp]` section entirely.
- **The Fix:** Used `replace_file_content` to add multiple sections at once targeting `[storage]`.
- **The Law:** Always verify file content before multi-line replaces, especially on example configs.

## [2026-02-17] Feature: Whatsmeow Integration (US1 Backend)

- **The Trap:** Guessing `whatsmeow` sqlstore method signatures and shadowing variables in Go.
- **The Reality:** `GetDevice` required a context argument (based on compiler error), and `:=` shadowed outer `err` variable inside `if` block.
- **The Fix:** Used `var` declaration to avoid shadowing and passed `ctx` to `GetDevice`.
- **The Law:** When handling errors in conditional blocks in Go, be extremely careful with `:=` vs `=`. Use strict typed variable declarations when in doubt.

## [2026-02-17] Feature: Whatsmeow Integration (US1 Frontend)

- **The Trap:** Duplicating imports when appending to a list using `replace_file_content` with imprecise context.
- **The Reality:** The tool replaced a partial match, leaving the original import line intact if context lines were not carefully chosen.
- **The Fix:** Explicitly verify imports before/after or use wider context to ensure complete replacement of the block.
- **The Law:** When modifying imports, check for duplicates after the operation, or replace the entire import block to be safe.

## [2026-02-17] Feature: Whatsmeow Integration (US2 Message Handling)

- **The Trap:** Trying to add reply-to-message support by modifying the existing `MessageProvider.SendText` signature.
- **The Reality:** Changing an interface method signature would break all implementations. The reply-to context is optional and not supported by all providers.
- **The Fix:** Created a separate `ReplyProvider` interface that callers can type-assert to check for capability. The whatsmeow adapter implements both `MessageProvider` and `ReplyProvider`.
- **The Law:** Never modify an existing interface signature to add optional behavior. Use Go's interface composition and type assertions instead.

## [2026-02-17] Feature: Whatsmeow Integration (T054 Group Messages)

- **The Trap:** Using `Contact.PhoneNumber` as the chat JID for reactions and read receipts on group messages.
- **The Reality:** In group messages, the contact is the individual sender, but the chat JID must be the group JID. Reactions and read receipts need to target the group chat, not the individual's 1:1 chat.
- **The Fix:** Used `message.ConversationID` (which stores the group JID like `120363...@g.us`) as the chat JID, and `Contact.PhoneNumber` as the sender JID. Added `isGroupJID()` helper to detect group JIDs.
- **The Law:** In WhatsApp group messaging, always distinguish between chat JID (the group) and sender JID (the individual). Store the group JID in `ConversationID` and use it for all chat-level operations (reactions, read receipts, sending messages).

## [2026-02-17] Feature: Whatsmeow Integration (US3 Multi-Instance)

- **The Trap:** Treating all send flows as if they used a single default account without validating instance state.
- **The Reality:** Multi-instance mode requires explicit instance ownership + connected-state validation before outbound sends, otherwise messages can silently route to disconnected or wrong instances.
- **The Fix:** Added `resolveOutboundInstance` routing helper, runtime health counters in the connection manager, and persisted ban/logout notifications with WebSocket fanout.
- **The Law:** In multi-instance systems, resolve target instance first, validate tenancy and connection state second, and only then execute message send logic.

## [2026-02-17] Feature: Whatsmeow Integration (Linter Fixes)

- **The Trap:** Ignoring error return values from `resp.Body.Close()` and `cm.Disconnect()` in Go.
- **The Reality:** The `errcheck` linter correctly identified these as potential resource leaks or silent failures, even though they rarely fail in practice.
- **The Fix:** Used anonymous functions in `defer` to explicitly ignore the `Close()` error and added error logging for the `Disconnect()` call.
- **The Law:** Always handle or explicitly ignore errors from `io.Closer` or cleanup functions to ensure static analysis passes and potential resource leaks are documented.

## [2026-02-17] Issue: Orphaned Goroutines during Instance Connection

- **The Trap:** Launching asynchronous `Connect` calls in HTTP handlers using raw `go func()` without WaitGroup tracking.
- **The Reality:** During server shutdown, raw goroutines may be cut off without completing their state updates (like updating DB status to 'connected'), leading to inconsistent database state on next boot.
- **The Fix:** Wrapped the `Connect` call in `internal/handlers/instances.go` with `a.wg.Add(1)` and `defer a.wg.Done()`.
- **The Law:** Every long-running or critical asynchronous operation initiated by a request must be tracked by the application's central WaitGroup to ensure graceful shutdown completion.

## [2026-02-17] Issue: Hardcoded Logic and Missing Timeouts

- **The Trap:** Using hardcoded retry counts and delays for network operations, and using `context.Background()` without timeouts in background goroutines.
- **The Reality:** While "good enough" for MVP, hardcoded values make production tuning difficult, and missing timeouts can lead to goroutine leaks if library calls hang indefinitely.
- **The Fix:** Moved retry logic to `config.WhatsmeowConfig` and added `context.WithTimeout` to all background handler goroutines.
- **The Law:** Never use infinite `context.Background()` for network calls; always enforce a sensible timeout and make retry parameters configurable.

## [2026-02-17] Issue: Received Messages Empty Bubble (WS Content Format)

- **The Trap:** Broadcasting the raw `models.Message` struct via WebSocket, assuming the frontend would handle it the same as HTTP responses.
- **The Reality:** The HTTP path wraps `content` as `{"body": "text"}`, but the WS broadcast sent it as a raw string. The frontend's `getMessageContent()` expects `message.content.body` (an object), so `"text".body` yields `undefined` → empty bubble.
- **The Fix:** Replaced raw struct broadcast in `events.go` with a structured `map[string]any` payload matching the HTTP response format, including `assigned_user_id` and `profile_name` for notification support.
- **The Law:** WebSocket payloads must match the HTTP response format for the same resource. When adding a new broadcast, always check the existing HTTP serialization and replicate it.

## 2026-02-19 06:12 - Security Hardening Batch (Tasks 5-9)

- **Trap:** Multiple trust boundaries were fail-open at runtime (`webhook` signature paths, queue retry lifecycle, JS runtime execution limits, and secret encryption behavior when key config was absent).
- **Reality:** Security-critical paths were spread across handlers/worker/queue with inconsistent enforcement and no centralized runtime guardrails for retries, idempotency, and distributed redirect behavior.
- **Fix:** Added fail-closed secret encryption/decryption behavior, strict webhook request validation module, queue DLQ + periodic pending reclaim, recipient-level worker idempotency lock, configured client routing helper, and JS timeout + Redis redirect-token runtime module.
- **Law:** When a feature crosses process boundaries (API, worker, Redis, external providers), enforce invariants at every boundary; do not trust configuration defaults or process-local state for security-critical operations.

## 2026-02-20 00:14 Issue: Chat Lifecycle Needed Explicit Status/Assignment Boundaries

- **The Trap:** Treating contacts as always-active chats left assignment state implicit, making pending/assigned/closed behavior inconsistent between backend filtering, permission gates, and frontend views.
- **The Reality:** Queue-style operations require explicit lifecycle state (`pending`, `open`, `closed`) and server-side enforcement; UI-only hiding is insufficient because API reads can bypass frontend restrictions.
- **The Fix:** Added lifecycle fields and migration indexes to `contacts`, implemented chat lifecycle helpers + `/api/chats` endpoints (`list/claim/close/messages`), enforced pending/unassigned message-read restrictions at handler level, added read-only closed-chat behavior, and migrated frontend chat state into pending/assigned/closed buckets with claim/close transitions and a new settings closed-chats view.
- **The Law:** If chat visibility depends on assignment, enforce it in backend message handlers first and let frontend state mirror backend lifecycle transitions.

## 2026-02-20 00:52 Issue: Closed Chats Needed Deterministic Reopen Semantics

- **The Trap:** Closing a chat while leaving assignment attached caused stale ownership semantics when the customer replied later, and UI had no direct reopen action from closed state.
- **The Reality:** Reopen must always return chats to a neutral queue state (`pending` + unassigned), whether triggered manually or by inbound customer activity.
- **The Fix:** Added `PUT /api/chats/{id}/reopen`, wired auto-reopen on inbound message persistence (Meta + Whatsmeow), normalized WebSocket payloads with `contact_status`, and added reopen controls in both `ChatView` and `ClosedChatsView` plus new settings queue pages for pending/assigned/closed visibility.
- **The Law:** Reopening a closed chat must be ownership-neutral; never carry old assignee state into reopened queue items.
