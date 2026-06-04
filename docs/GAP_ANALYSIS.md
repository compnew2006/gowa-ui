# Campaigns Gap Analysis

## Summary

This document identifies gaps, missing features, and issues discovered in the `/campaigns` workflow of the Whatomate project. Each gap includes severity assessment, proposed solution, impact analysis, and affected functions.

---

## GAP-01: Scheduled Campaign Execution Not Implemented

**Severity:** HIGH
**Category:** Missing Feature
**Status:** `ScheduledAt` field exists but no scheduler consumes it

### Description

The `BulkMessageCampaign` model has a `ScheduledAt` field and the `CampaignStatusScheduled` constant is defined. Users can set `scheduled_at` when creating or updating a campaign. However:

1. No background scheduler, cron, or ticker reads `ScheduledAt` to auto-start campaigns
2. The `scheduled` status is **never assigned** by any code path — it is only accepted as a valid state in `StartCampaign`
3. The field is stored and returned in API responses, creating a false impression of functionality

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/models/bulk.go:28` | `ScheduledAt` field | Dead field — stored but never consumed |
| `internal/models/constants.go:148` | `CampaignStatusScheduled` | Constant defined but never assigned |
| `internal/handlers/campaigns.go:218` | `CreateCampaign` | Accepts `scheduled_at` without functional purpose |
| `internal/handlers/campaigns.go:350` | `UpdateCampaign` | Allows updating `scheduled_at` without functional purpose |
| `cmd/whatomate/main.go:436-468` | Background workers | No scheduled campaign worker exists |

### Proposed Solution

Add a `CampaignSchedulerWorker` that runs periodically (every 30-60 seconds):

```go
// internal/worker/campaign_scheduler.go
type CampaignSchedulerWorker struct {
    db   *gorm.DB
    log  *zap.Logger
    app  *handlers.App
}

func (w *CampaignSchedulerWorker) Run(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processScheduledCampaigns(ctx)
        }
    }
}

func (w *CampaignSchedulerWorker) processScheduledCampaigns(ctx context.Context) {
    var campaigns []models.BulkMessageCampaign
    now := time.Now().UTC()
    w.db.Where("status = ? AND scheduled_at <= ? AND scheduled_at IS NOT NULL",
        models.CampaignStatusDraft, now).
        Find(&campaigns)
    
    for _, campaign := range campaigns {
        // Call StartCampaign logic directly
        // Set status to CampaignStatusScheduled first, then to processing
    }
}
```

Wire in `cmd/whatomate/main.go` alongside other background workers.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Enables time-based campaign automation; completes the partial scheduling feature |
| **Positive** | Users can schedule campaigns for optimal send times |
| **Negative** | Additional background worker consumes resources |
| **Negative** | Need to handle edge cases: server restart, missed schedules, concurrent starts |
| **Risk** | Race condition if user manually starts a scheduled campaign before its time |

### Affected Functions (by the solution)

- `cmd/whatomate/main.go` — add scheduler worker startup
- `internal/handlers/campaigns.go` — `StartCampaign` may need to handle `scheduled` → `processing` transition explicitly
- `internal/worker/` — new file `campaign_scheduler.go`

---

## GAP-02: Whatsmeow Delivery Receipts Do Not Update Campaign Stats

**Severity:** HIGH
**Category:** Functional Bug
**Status:** Delivery receipts update Message records but not campaign counters

### Description

When a whatsmeow campaign message is delivered or read, the receipt handler in `pkg/whatsmeow/events.go:handleReceipt()` updates the `Message` record status and broadcasts via WebSocket. However, it does **NOT**:

1. Check for `campaign_id` in message metadata
2. Update `BulkMessageRecipient` status or timestamps (`delivered_at`, `read_at`)
3. Increment `BulkMessageCampaign` counters (`delivered_count`, `read_count`)
4. Publish campaign stats update via Redis pub/sub

This means whatsmeow campaigns always show `delivered_count = 0` and `read_count = 0` even after messages are successfully delivered and read by recipients.

### Comparison with Meta Provider

The Meta Cloud API webhook handler (`internal/handlers/webhook.go:applyMessageStatusUpdate`) correctly handles all of the above. The whatsmeow receipt handler is missing the equivalent logic.

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `pkg/whatsmeow/events.go:267` | `handleReceipt` | Does not propagate receipt to campaign system |
| `internal/handlers/campaigns.go:1384` | `incrementCampaignStat` | Never called for whatsmeow receipts |
| `internal/handlers/campaigns.go:1433` | `recalculateCampaignStats` | Manual reconciliation exists but never auto-triggered |
| `internal/worker/worker.go:512` | `publishCampaignStats` | Never called for whatsmeow receipts |

### Proposed Solution

Add campaign stat propagation to `handleReceipt` in `pkg/whatsmeow/events.go`:

```go
func (cm *ConnectionManager) handleReceipt(ctx context.Context, info proto.MessageInfo) {
    // ... existing receipt processing ...
    
    // After updating Message status:
    if msg.Metadata != nil {
        if campaignIDStr, ok := msg.Metadata["campaign_id"].(string); ok {
            campaignID, err := uuid.Parse(campaignIDStr)
            if err == nil {
                // Update BulkMessageRecipient
                cm.db.Model(&models.BulkMessageRecipient{}).
                    Where("campaign_id = ? AND whats_app_message_id = ?", campaignID, info.ID).
                    Updates(map[string]interface{}{
                        "status":      newStatus,
                        "delivered_at": deliveredAt,
                        "read_at":      readAt,
                    })
                
                // Increment campaign counter
                if newStatus == models.MessageStatusDelivered || newStatus == models.MessageStatusRead {
                    column := "delivered_count"
                    if newStatus == models.MessageStatusRead {
                        column = "read_count"
                    }
                    cm.db.Model(&models.BulkMessageCampaign{}).
                        Where("id = ?", campaignID).
                        UpdateColumn(column, gorm.Expr(column + " + 1"))
                    
                    // Publish stats update
                    // (requires access to Redis publisher)
                }
            }
        }
    }
}
```

**Challenge:** `ConnectionManager` in `pkg/whatsmeow/` does not have direct access to the Redis publisher or campaign stats broadcast mechanism. This requires either:
- Injecting a callback/handler interface into `ConnectionManager`
- Moving the receipt-to-campaign logic to a shared package
- Using an event/channel pattern

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | whatsmeow campaign analytics become accurate |
| **Positive** | Consistent behavior between Meta and whatsmeow providers |
| **Negative** | Additional DB queries per receipt (could be batched) |
| **Negative** | Requires refactoring `ConnectionManager` to accept a campaign stats callback |
| **Risk** | High receipt volume could cause DB contention; need batching |

### Affected Functions (by the solution)

- `pkg/whatsmeow/events.go` — `handleReceipt` (core change)
- `pkg/whatsmeow/connection_manager.go` — may need new interface/callback field
- `internal/handlers/campaigns.go` — `incrementCampaignStat` may be extracted to shared package
- `cmd/whatomate/main.go` — wiring the callback during setup

---

## GAP-03: No Batch Size Limit on Recipient Import

**Severity:** MEDIUM
**Category:** Security / Resource Exhaustion
**Status:** Unbounded JSON array accepted

### Description

`ImportRecipients` (`campaigns.go:929-1029`) accepts an array of recipients with no upper bound. A user with `campaigns:write` permission could POST millions of recipients, causing:

1. Memory exhaustion building the `seenPhones` dedup map
2. Very large INSERT statements
3. Storage quota bypass (quota is checked on media, not recipient count)

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/handlers/campaigns.go:929` | `ImportRecipients` | No `len(req.Recipients)` check |
| `internal/handlers/campaigns.go:967` | Dedup loop | O(n) memory for phone map |

### Proposed Solution

```go
const maxRecipientsPerImport = 10000

func (a *App) ImportRecipients(r *fastglue.Request) error {
    // ... after parsing request ...
    if len(req.Recipients) > maxRecipientsPerImport {
        return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
            fmt.Sprintf("Too many recipients. Maximum %d per import", maxRecipientsPerImport),
            nil, "")
    }
    // ... continue with import ...
}
```

Optionally make `maxRecipientsPerImport` configurable via `config.toml`.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents resource exhaustion attacks |
| **Positive** | Encourages batched imports for large campaigns |
| **Negative** | Users with legitimate large lists need multiple imports |
| **Risk** | Very low — purely additive validation |

### Affected Functions

- `internal/handlers/campaigns.go` — `ImportRecipients`
- `internal/config/config.go` — optional config field

---

## GAP-04: No Rate Limiting on Campaign Endpoints

**Severity:** MEDIUM
**Category:** Security
**Status:** Rate limiting exists but is not applied to campaign routes

### Description

Rate limiting middleware exists (`internal/middleware/ratelimit.go`) and is applied to auth endpoints and webhooks. However, no campaign endpoints have rate limits. This means:

1. `StartCampaign` can be called rapidly (mitigated by status checks, but still generates DB writes)
2. `ImportRecipients` can be called repeatedly with large batches
3. `UploadCampaignMedia` can be used to stress disk I/O

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `cmd/whatomate/main.go:1394-1409` | Route registration | No `withRateLimit` wrapper |
| `internal/middleware/ratelimit.go` | Rate limiter | Exists but unused for campaigns |

### Proposed Solution

Apply targeted rate limits to mutation endpoints:

```go
// In cmd/whatomate/main.go route registration:
g.POST("/api/campaigns", withRateLimit(app.StartCampaign, 10, time.Minute))     // 10 creates/min
g.POST("/api/campaigns/{id}/start", withRateLimit(app.StartCampaign, 5, time.Minute))  // 5 starts/min
g.POST("/api/campaigns/{id}/recipients/import", withRateLimit(app.ImportRecipients, 10, time.Minute)) // 10 imports/min
g.POST("/api/campaigns/{id}/media", withRateLimit(app.UploadCampaignMedia, 5, time.Minute)) // 5 uploads/min
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents abuse and accidental rapid-fire actions |
| **Positive** | Protects backend resources |
| **Negative** | Could slow down legitimate bulk operations |
| **Risk** | Very low — same pattern already used for auth endpoints |

### Affected Functions

- `cmd/whatomate/main.go` — route registration
- `internal/middleware/ratelimit.go` — no changes needed

---

## GAP-05: Campaign Completion Based on Send, Not Delivery

**Severity:** LOW
**Category:** Design Limitation
**Status:** By design, but may confuse users

### Description

`checkCampaignCompletion()` marks a campaign as `completed` when all recipients have been processed (sent or failed). This happens immediately after the last message is sent, not after all delivery receipts arrive. The `completed_at` timestamp reflects send-completion, not delivery-completion.

There is no "fully delivered" state or mechanism to know when all messages have been actually delivered/read.

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/worker/worker.go:530` | `checkCampaignCompletion` | Completes on send, not delivery |
| `internal/handlers/campaigns.go:1433` | `recalculateCampaignStats` | Manual reconciliation exists but never auto-called |

### Proposed Solution (Optional)

Add an optional `delivered_at` field to `BulkMessageCampaign` and a background job that checks if all sent messages have delivery receipts:

```go
// New status: CampaignStatusDelivered (optional)
// Or: Add delivered_percentage field computed from sent_count vs delivered_count
// Frontend can show "100% sent, 78% delivered" progress bar
```

Alternatively, add a periodic reconciliation job that calls `recalculateCampaignStats()` for recently completed campaigns.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | More accurate campaign reporting |
| **Negative** | Additional complexity and background processing |
| **Risk** | Low — optional enhancement |

### Affected Functions

- `internal/models/bulk.go` — optional new fields
- `internal/worker/worker.go` — optional new completion check
- `internal/handlers/campaigns.go` — `recalculateCampaignStats` auto-trigger

---

## GAP-06: No Tests for Worker Job Processing

**Severity:** HIGH
**Category:** Testing Gap
**Status:** Core send logic has zero direct tests

### Description

`HandleRecipientJob` in `internal/worker/worker.go` (189 lines) is the most critical function in the campaign system. It handles idempotency, policy checks, template resolution, rate limiting, message sending, counter increments, and completion detection. Despite this, it has **zero direct tests**.

### Missing Test Scenarios

1. Paused/cancelled campaign skips in-flight jobs
2. Redis delay slot reservation
3. Campaign completion detection
4. Template placeholder resolution with full placeholder set
5. `renderCampaignTemplateBody` rendering
6. `classifyCampaignMediaType` classification
7. Concurrent job completion race condition
8. Idempotency lock when same recipient job delivered twice

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/worker/worker.go:189` | `HandleRecipientJob` | Zero tests |
| `internal/worker/worker.go:530` | `checkCampaignCompletion` | Zero tests |
| `internal/worker/worker.go:503` | `incrementCampaignCount` | Zero tests |
| `internal/worker/campaign_template_placeholders.go` | All functions | Zero tests |

### Proposed Solution

Create `internal/worker/worker_campaign_test.go` with integration tests that:

1. Set up a test campaign with recipients in the test DB
2. Mock or use the real Redis queue
3. Call `HandleRecipientJob` directly
4. Assert recipient status, campaign counters, message creation
5. Test edge cases: paused campaign, cancelled campaign, missing contact

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents regressions in the most critical path |
| **Positive** | Documents expected behavior |
| **Negative** | Requires test infrastructure (DB + Redis) |
| **Risk** | None — purely additive |

### Affected Functions (by the tests)

- `internal/worker/worker.go` — `HandleRecipientJob`, `checkCampaignCompletion`, `incrementCampaignCount`, `publishCampaignStats`
- `internal/worker/campaign_template_placeholders.go` — all functions
- New test file: `internal/worker/worker_campaign_test.go`

---

## GAP-07: No Tests for Auto-Pause on Instance Disconnect

**Severity:** MEDIUM
**Category:** Testing Gap
**Status:** Auto-pause logic has zero tests

### Description

`pauseActiveCampaignsForInstance()` in `pkg/whatsmeow/events_campaign_pause.go` pauses all processing/queued campaigns when a WhatsApp instance is banned or logged out. This is a critical safety mechanism, but has **zero tests**.

### Missing Test Scenarios

1. Campaigns paused when instance gets `TemporaryBan`
2. Campaigns paused when instance gets `LoggedOut`
3. Queued (not just processing) campaigns are also paused
4. Paused campaigns can be resumed after reconnection
5. No campaigns paused for other organizations' instances

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `pkg/whatsmeow/events_campaign_pause.go:10` | `pauseActiveCampaignsForInstance` | Zero tests |

### Proposed Solution

Create `pkg/whatsmeow/events_campaign_pause_test.go` with integration tests.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Validates critical safety mechanism |
| **Negative** | Requires test DB setup |
| **Risk** | None |

---

## GAP-08: Auto-Campaign Worker Has Minimal Test Coverage

**Severity:** MEDIUM
**Category:** Testing Gap
**Status:** 525 lines of worker code, only 2 unit tests

### Description

`instance_auto_campaign_worker.go` (525 lines) has only 2 tests for `resolveAutoCampaignWindow`. The remaining ~500 lines are untested.

### Missing Test Scenarios

1. Full `runOnce` flow (campaign creation + auto-start)
2. Contact loading within time window
3. Duplicate campaign name detection
4. Campaign creation with media attachment
5. Auto-start policy enforcement
6. `buildAutoCampaignName` formatting
7. `normalizeAutoCampaignMessageTemplate` placeholder normalization
8. `isAutoCampaignDue` timing logic
9. `persistLastGeneratedAt` persistence

### Proposed Solution

Add unit tests for pure functions and integration tests for DB-dependent flows.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents regressions in auto-campaign generation |
| **Negative** | Time investment for comprehensive coverage |
| **Risk** | None |

---

## GAP-09: MIME Type Validation Based on Client Header

**Severity:** LOW
**Category:** Security
**Status:** No magic byte detection

### Description

`UploadCampaignMedia` trusts the `Content-Type` header from the client without inspecting file magic bytes. An attacker could upload a non-image file with `Content-Type: image/jpeg`.

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/handlers/campaigns.go:1188` | `UploadCampaignMedia` | MIME type from client header |
| `internal/handlers/instance_auto_campaign_media.go:24` | `UploadInstanceAutoCampaignMedia` | Same issue |

### Proposed Solution

Add `http.DetectContentType()` check after reading the file bytes:

```go
detectedMIME := http.DetectContentType(data[:min(512, len(data))])
if detectedMIME != mimeType {
    return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
        "File content does not match declared type", nil, "")
}
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents file type spoofing |
| **Negative** | `http.DetectContentType` may not detect all MIME types accurately |
| **Risk** | Very low — defense-in-depth improvement |

### Affected Functions

- `internal/handlers/campaigns.go` — `UploadCampaignMedia`
- `internal/handlers/instance_auto_campaign_media.go` — `UploadInstanceAutoCampaignMedia`

---

## GAP-10: Cancelled Campaign Leaves Orphaned Queue Messages

**Severity:** LOW
**Category:** Data Integrity
**Status:** Cancelled campaigns don't clean up Redis Stream

### Description

When a campaign is cancelled, the handler updates the DB status to `cancelled` and the worker skips cancelled campaigns at step 3 of `HandleRecipientJob`. However, pending messages already enqueued in the Redis Stream remain there until they are consumed and skipped. This wastes processing resources.

### Affected Files & Functions

| File | Function | Impact |
|------|----------|--------|
| `internal/handlers/campaigns.go:795` | `CancelCampaign` | No Redis Stream cleanup |
| `internal/worker/worker.go:210` | `HandleRecipientJob` | Skips but doesn't remove |

### Proposed Solution

In `CancelCampaign`, after setting status to `cancelled`, optionally trim the campaign's pending messages from the Redis Stream:

```go
// After status update:
// XTRIM whatomate:campaigns:<orgID> MINID <currentTimestamp>
// Or: let workers naturally drain and skip (current behavior, acceptable)
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Cleaner queue state |
| **Negative** | XTRIM by ID requires knowing message IDs; may affect other campaigns in same stream |
| **Risk** | Low — current behavior is functionally correct, just wasteful |

### Affected Functions

- `internal/handlers/campaigns.go` — `CancelCampaign`

---

## GAP-11: No End-to-End Integration Test for Campaign Lifecycle

**Severity:** MEDIUM
**Category:** Testing Gap
**Status:** No single test exercises the full pipeline

### Description

There is no integration test that exercises the complete campaign lifecycle: create → import recipients → start → worker processes → completion → delivery receipts → stats verification. The E2E Playwright tests are UI-only with mocked API responses.

### Proposed Solution

Add a Go integration test in `internal/handlers/campaigns_integration_test.go` that:

1. Creates a campaign via handler
2. Imports recipients
3. Starts the campaign
4. Manually calls `HandleRecipientJob` for each recipient
5. Verifies campaign status transitions to `completed`
6. Simulates delivery receipts via webhook handler
7. Verifies final stats accuracy

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Catches integration issues between components |
| **Negative** | Slower test execution |
| **Risk** | None |

---

## GAP-12: CampaignsView.vue Is Oversized (3,535 Lines)

**Severity:** MEDIUM
**Category:** Code Quality
**Status:** Single-file component with all campaign logic

### Description

`frontend/src/views/settings/CampaignsView.vue` is a 3,535-line single-file component containing:
- All campaign CRUD logic
- All recipient management (manual, contacts, CSV)
- All dialog components (create, edit, recipients, delete, cancel, media preview)
- All local state management
- All WebSocket integration

This makes the component difficult to maintain, test, and review.

### Proposed Solution

Extract into smaller, focused components:

```
views/settings/CampaignsView.vue          Main view (layout + list)
components/campaigns/CampaignCreateDialog.vue
components/campaigns/CampaignEditDialog.vue
components/campaigns/CampaignRecipientsDialog.vue
components/campaigns/CampaignAddRecipientsDialog.vue
components/campaigns/CampaignMediaPreview.vue
components/campaigns/CampaignStatusBadge.vue
composables/useCampaigns.ts               Shared logic + state
stores/campaigns.ts                       Pinia store (if needed)
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Better maintainability, testability, and code reuse |
| **Positive** | Smaller components are easier to review and modify |
| **Negative** | Significant refactoring effort |
| **Negative** | Risk of breaking existing functionality during refactor |
| **Risk** | Medium — pure refactor, no behavior change, but large surface area |

### Affected Functions/Files

- `frontend/src/views/settings/CampaignsView.vue` — major refactor
- New files: multiple component and composable files

---

## Priority Matrix

| Priority | Gap | Effort | Impact |
|----------|-----|--------|--------|
| **P0** | GAP-02: Whatsmeow delivery receipts | Medium | Fixes broken analytics |
| **P0** | GAP-06: Worker job tests | Medium | Prevents regressions |
| **P1** | GAP-01: Scheduled execution | Medium | Completes partial feature |
| **P1** | GAP-03: Batch size limit | Low | Security hardening |
| **P1** | GAP-04: Rate limiting | Low | Security hardening |
| **P2** | GAP-07: Auto-pause tests | Low | Safety validation |
| **P2** | GAP-08: Auto-campaign tests | Medium | Regression prevention |
| **P2** | GAP-11: E2E integration test | Medium | Pipeline validation |
| **P3** | GAP-05: Completion tracking | Medium | UX improvement |
| **P3** | GAP-09: MIME detection | Low | Security hardening |
| **P3** | GAP-10: Queue cleanup | Low | Performance |
| **P3** | GAP-12: Component refactor | High | Code quality |

---

# Settings Route Gap Analysis

> Added: 2026-04-29  
> Module: `/settings` and `/settings/*` routes

## Summary

This section identifies gaps, missing features, and issues discovered in the `/settings` workflow of the Whatomate project. Each gap includes severity assessment, proposed solution, impact analysis, and affected functions.

---

## SETTINGS-GAP-01: Shared `isSubmitting` State Across All Settings Tabs

**Severity:** HIGH  
**Category:** UX Bug  
**Status:** Active — all 4 tabs share a single `isSubmitting` ref

### Description

In `SettingsView.vue`, the `isSubmitting` ref is shared across all 4 tabs (General, Appearance, Notifications, Chat). When a user saves on any tab, all tabs' save buttons disable simultaneously. This creates a confusing UX where a user saving Appearance settings sees General and Notification save buttons become disabled.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | `isSubmitting` ref | Shared by all 4 save functions |
| `frontend/src/views/settings/SettingsView.vue` | `saveGeneralSettings()` | Sets isSubmitting |
| `frontend/src/views/settings/SettingsView.vue` | `saveAppearanceSettings()` | Sets isSubmitting |
| `frontend/src/views/settings/SettingsView.vue` | `saveNotificationSettings()` | Sets isSubmitting |
| `frontend/src/views/settings/SettingsView.vue` | `saveChatSettings()` | Sets isSubmitting |

### Proposed Solution

Replace the single `isSubmitting` with per-tab loading states:

```typescript
const isGeneralSubmitting = ref(false)
const isAppearanceSubmitting = ref(false)
const isNotificationSubmitting = ref(false)
const isChatSubmitting = ref(false)
```

Bind each tab's save button to its own loading state.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Users can save settings on different tabs independently |
| **Positive** | Clearer UX feedback per section |
| **Negative** | None — purely a UX fix |
| **Risk** | Very low — no backend changes needed |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — `isSubmitting` ref → 4 separate refs, all `save*` functions, all tab templates

---

## SETTINGS-GAP-02: No Loading Indicator on Settings Page Initial Load

**Severity:** HIGH  
**Category:** UX Bug  
**Status:** `isLoading` ref exists but is never used in the template

### Description

`SettingsView.vue` sets `isLoading = true` during `onMounted` data fetching but the template never references `isLoading`. The user sees an empty form with no spinner, skeleton, or loading indicator. If the API is slow, the user sees blank inputs and may think settings are empty rather than loading.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | `isLoading` ref | Set in onMounted but never rendered |
| `frontend/src/views/settings/SettingsView.vue` | `<template>` section | No v-if/v-show for loading state |

### Proposed Solution

Add a loading skeleton or spinner that displays when `isLoading` is true:

```vue
<div v-if="isLoading" class="flex items-center justify-center p-12">
  <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
</div>
<div v-else class="space-y-6">
  <!-- existing tab content -->
</div>
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | User sees clear loading state on initial settings load |
| **Positive** | Prevents confusion about empty settings |
| **Negative** | None |
| **Risk** | None — cosmetic fix |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — template section

---

## SETTINGS-GAP-03: No Load-Error UI for Settings Page

**Severity:** MEDIUM  
**Category:** UX Bug  
**Status:** `onMounted` catch only logs to console

### Description

If the initial `organizationService.getSettings()` or `usersService.me()` call fails, the error is caught and logged to console but the user sees an empty form with no error message and no retry button.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | `onMounted()` catch block | `console.error` only |

### Proposed Solution

```typescript
const loadError = ref<string | null>(null)

// In onMounted catch:
loadError.value = getErrorMessage(error, 'Failed to load settings')

// In template:
<div v-if="loadError" class="...">
  <p>{{ loadError }}</p>
  <Button @click="loadSettings">Retry</Button>
</div>
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | User sees error message with retry option |
| **Positive** | Improves reliability perception |
| **Negative** | None |
| **Risk** | None |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — `onMounted`, template

---

## SETTINGS-GAP-04: No Dirty-Check for General, Notifications, and Chat Tabs

**Severity:** MEDIUM  
**Category:** UX Enhancement  
**Status:** Only Appearance tab has dirty-checking (`isAppearanceDirty`)

### Description

The Appearance tab correctly tracks dirty state and enables/disables the Save button based on whether changes exist. General, Notifications, and Chat tabs do not track dirty state — their Save buttons are always enabled even when no changes have been made. This can cause unnecessary API calls and confusion.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | General tab save | Always enabled |
| `frontend/src/views/settings/SettingsView.vue` | Notification tab save | Always enabled |
| `frontend/src/views/settings/SettingsView.vue` | Chat tab save | Always enabled |

### Proposed Solution

Implement `isDirty` computed properties for each tab by comparing current form values against loaded values:

```typescript
const initialGeneralSettings = ref<GeneralSettingsForm | null>(null)

const isGeneralDirty = computed(() => {
  if (!initialGeneralSettings.value) return false
  return JSON.stringify(generalSettings.value) !== JSON.stringify(initialGeneralSettings.value)
})
```

Bind `:disabled="!isGeneralDirty || isGeneralSubmitting"` to each tab's save button.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents unnecessary API calls |
| **Positive** | Clear feedback about unsaved changes |
| **Negative** | Slight increase in component complexity |
| **Risk** | Low — JSON comparison may not work for all edge cases |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — `saveGeneralSettings`, `saveNotificationSettings`, `saveChatSettings` + templates

---

## SETTINGS-GAP-05: No Unsaved Changes Navigation Guard (General/Notification/Chat)

**Severity:** MEDIUM  
**Category:** UX Enhancement  
**Status:** Only Appearance has `onBeforeRouteLeave` guard

### Description

When a user makes changes to General, Notification, or Chat settings and navigates away without saving, the changes are silently discarded. Only the Appearance tab has a route-leave guard that reverts unsaved changes.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | `onBeforeRouteLeave` | Only handles appearance |

### Proposed Solution

Extend the `onBeforeRouteLeave` guard to check dirty state for all tabs and show a confirmation dialog:

```typescript
onBeforeRouteLeave(() => {
  if (isGeneralDirty.value || isNotificationDirty.value || isChatDirty.value || isAppearanceDirty.value) {
    const answer = window.confirm('You have unsaved changes. Leave anyway?')
    if (!answer) return false
  }
})
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents accidental data loss |
| **Negative** | Adds one more dialog to dismiss |
| **Risk** | Low — standard UX pattern |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — `onBeforeRouteLeave`

---

## SETTINGS-GAP-06: No Confirmation Before Running Uploads Cleanup

**Severity:** MEDIUM  
**Category:** Safety  
**Status:** Clicking "Run Now" immediately deletes files

### Description

The "Run Now" button in the Uploads Cleanup section immediately triggers permanent file deletion without a confirmation dialog. A misclick could delete files that are still needed.

### Affected Files & Functions

| File | Function/Ref | Impact |
|------|-------------|--------|
| `frontend/src/views/settings/SettingsView.vue` | `runUploadsCleanupNow()` | No confirmation |

### Proposed Solution

```typescript
async function runUploadsCleanupNow() {
  const confirmed = await confirmDialog({
    title: t('settings.uploads_cleanup.confirm_title'),
    message: t('settings.uploads_cleanup.confirm_message'),
    confirmText: t('common.run'),
    destructive: true,
  })
  if (!confirmed) return
  // ... existing cleanup logic ...
}
```

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents accidental file deletion |
| **Negative** | One extra click for intentional cleanup |
| **Risk** | None |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — `runUploadsCleanupNow`

---

## SETTINGS-GAP-07: Inconsistent Permission Guards Across Settings Views

**Severity:** HIGH  
**Category:** Security / Architecture  
**Status:** Route-level guards exist, but component-level guards are inconsistent

### Description

Settings views have inconsistent permission enforcement patterns:

| View | Permission Guard | Pattern |
|------|-----------------|---------|
| SettingsView | `hasPermission("settings.general", "write")` | Correct |
| UsersView | `hasPermission("users", "write")` | Correct |
| RolesView | `hasPermission("roles", "write/delete")` | Correct |
| ContactsView | `hasPermission("contacts", "write/import/export")` | Correct |
| TeamsView | `isAdmin = userRole === "admin"` | **Wrong** — uses role name |
| TagsView | None | **Missing** |
| CannedResponsesView | None | **Missing** |
| WebhooksView | None | **Missing** |
| APIKeysView | None | **Missing** |
| SSOSettingsView | None | **Missing** |
| ChatbotSettingsView | None | **Missing** |
| CustomActionsView | None | **Missing** |
| ClosedChatsView | None | **Missing** |
| InstanceHealthView | None | **Missing** |

While route-level guards prevent unauthorized navigation, component-level action buttons are visible to all users who can access the route.

### Affected Files & Functions

| File | Issue |
|------|-------|
| `frontend/src/views/settings/TeamsView.vue` | Uses `userRole === "admin"` instead of `hasPermission("teams", "write")` |
| `frontend/src/views/settings/TagsView.vue` | No permission checks |
| `frontend/src/views/settings/CannedResponsesView.vue` | No permission checks |
| `frontend/src/views/settings/WebhooksView.vue` | No permission checks |
| `frontend/src/views/settings/APIKeysView.vue` | No permission checks |
| `frontend/src/views/settings/SSOSettingsView.vue` | No permission checks |
| `frontend/src/views/settings/ChatbotSettingsView.vue` | No permission checks |
| `frontend/src/views/settings/CustomActionsView.vue` | No permission checks |
| `frontend/src/views/settings/ClosedChatsView.vue` | No permission checks |
| `frontend/src/views/settings/InstanceHealthView.vue` | No permission checks |

### Proposed Solution

1. Add consistent `canWrite`, `canDelete` computed properties to each view
2. Fix `TeamsView.vue` to use `hasPermission("teams", "write")` instead of `isAdmin`
3. Bind `:disabled="!canWrite"` or `v-if="canWrite"` to action buttons

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Consistent security model across all settings views |
| **Positive** | UI accurately reflects user capabilities |
| **Negative** | Some users may see fewer action buttons (expected — already blocked by backend) |
| **Negative** | Requires updating 10+ component files |
| **Risk** | Low — backend already enforces permissions |

### Affected Functions

- 10+ settings view components — add permission computed properties and bind to action buttons
- `frontend/src/views/settings/TeamsView.vue` — fix `isAdmin` to use `hasPermission`

---

## SETTINGS-GAP-08: Backend Organization Member Routes Not Exposed in Frontend API

**Severity:** MEDIUM  
**Category:** Missing Frontend Feature  
**Status:** Backend routes exist but no frontend service methods

### Description

The backend has full organization member management routes, but the frontend `organizationsService` only exposes `list()`, `create()`, `delete()`, and `addMember()`. Missing:

| Backend Route | Purpose |
|--------------|---------|
| `GET /api/organizations/current` | Get current org details |
| `GET /api/organizations/members` | List org members |
| `PUT /api/organizations/members/{member_id}` | Update member role |
| `DELETE /api/organizations/members/{member_id}` | Remove member |

### Affected Files & Functions

| File | Missing Method |
|------|---------------|
| `frontend/src/services/api.ts` | `organizationsService.getCurrent()` |
| `frontend/src/services/api.ts` | `organizationsService.listMembers()` |
| `frontend/src/services/api.ts` | `organizationsService.updateMember()` |
| `frontend/src/services/api.ts` | `organizationsService.removeMember()` |

### Proposed Solution

Add the missing service methods to `organizationsService` in `frontend/src/services/api.ts`.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Enables full organization member management from the UI |
| **Negative** | Requires new UI components |
| **Risk** | Low — purely additive |

### Affected Functions

- `frontend/src/services/api.ts` — `organizationsService`

---

## SETTINGS-GAP-09: SSO Settings View Makes Inline API Calls

**Severity:** LOW  
**Category:** Code Quality  
**Status:** SSOSettingsView directly calls `api.get/put/delete` without a service module

### Description

All other settings views use a dedicated service object from `frontend/src/services/api.ts`. SSOSettingsView makes inline API calls, inconsistent with the established pattern.

### Proposed Solution

Create a dedicated `ssoService` in `frontend/src/services/api.ts`.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Consistent API access pattern |
| **Negative** | Minor refactor |
| **Risk** | None |

### Affected Functions

- `frontend/src/services/api.ts` — new `ssoService`
- `frontend/src/views/settings/SSOSettingsView.vue` — replace inline calls

---

## SETTINGS-GAP-10: PendingChatsView and AssignedChatsView Missing i18n

**Severity:** LOW  
**Category:** i18n  
**Status:** Hardcoded English strings

### Description

`PendingChatsView.vue` and `AssignedChatsView.vue` use hardcoded English strings instead of `vue-i18n` translation keys.

### Proposed Solution

Replace all hardcoded strings with `t('settings.pending_chats.title')` etc. and add entries to locale files.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Consistent i18n support |
| **Negative** | Translation effort for ar/es locales |
| **Risk** | None |

### Affected Functions

- `frontend/src/views/settings/PendingChatsView.vue`, `AssignedChatsView.vue`
- `frontend/src/i18n/locales/en.ts`, `es.ts`, `ar.ts`

---

## SETTINGS-GAP-11: PendingChatsView and AssignedChatsView Hardcoded 200-Item Limit

**Severity:** MEDIUM  
**Category:** Functional Limitation  
**Status:** `limit: 200` hardcoded, no pagination

### Description

Both views fetch data with `limit: 200` and perform client-side search only. If there are more than 200 pending/assigned chats, the user cannot see or search them.

### Proposed Solution

Add server-side pagination matching the `ClosedChatsView.vue` pattern.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Handles organizations with >200 active chats |
| **Negative** | Requires backend pagination support verification |
| **Risk** | Low |

### Affected Functions

- `frontend/src/views/settings/PendingChatsView.vue`, `AssignedChatsView.vue`
- `frontend/src/stores/contacts.ts`
- Backend handlers — verify pagination support

---

## SETTINGS-GAP-12: No Confirmation Dialog for Destructive Team Member Removal

**Severity:** LOW  
**Category:** UX Safety  
**Status:** One-click removal with no confirmation

### Proposed Solution

Add confirmation dialog before calling `removeTeamMember`.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Prevents accidental member removals |
| **Negative** | One extra click |
| **Risk** | None |

### Affected Functions

- `frontend/src/views/settings/TeamsView.vue` — remove member handler

---

## SETTINGS-GAP-13: SettingsView Is Over-Sized (1000+ Lines in Single SFC)

**Severity:** MEDIUM  
**Category:** Code Quality  
**Status:** Single file with org settings, user settings, appearance, chat, notifications, uploads cleanup

### Proposed Solution

Extract into composables: `useChatBackground.ts`, `useAppearanceSettings.ts`, `useUploadsCleanup.ts`, `useNotificationSettings.ts`.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Better maintainability and testability |
| **Negative** | Refactoring effort, risk of breaking functionality |
| **Risk** | Medium — large surface area |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — major refactor
- New composable files

---

## SETTINGS-GAP-14: Mixed Storage Strategy for User Preferences

**Severity:** LOW  
**Category:** Architecture  
**Status:** Some settings in backend, some in localStorage, some in configStore

### Description

User preferences use inconsistent storage: media group window in localStorage, print/download buttons in configStore, all others in backend API. Users cannot tell which preferences sync across devices.

### Proposed Solution

Migrate localStorage/configStore preferences to `PUT /api/me/settings` with backend model changes.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Consistent cross-device sync |
| **Negative** | Backend migration needed |
| **Risk** | Medium — coordinated frontend + backend change |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue`
- `frontend/src/stores/config.ts`
- `internal/handlers/users.go` — `UpdateCurrentUserSettings`
- `internal/models/` — UserSettings model

---

## SETTINGS-GAP-15: No "Reset to Defaults" for Any Settings Section

**Severity:** LOW  
**Category:** Missing Feature  
**Status:** Only Appearance has "Revert" (to last saved, not factory defaults)

### Proposed Solution

Add a "Reset to Defaults" button for each tab with known default values.

### Impact Analysis

| Aspect | Impact |
|--------|--------|
| **Positive** | Easy recovery from misconfiguration |
| **Negative** | Need to maintain defaults |
| **Risk** | None |

### Affected Functions

- `frontend/src/views/settings/SettingsView.vue` — add reset functions and buttons

---

## Settings Gap Priority Matrix

| Priority | Gap | Severity | Effort | Impact |
|----------|-----|----------|--------|--------|
| **P0** | SETTINGS-GAP-07: Inconsistent permission guards | HIGH | Medium | Security/UX consistency |
| **P0** | SETTINGS-GAP-01: Shared isSubmitting across tabs | HIGH | Low | UX bug fix |
| **P0** | SETTINGS-GAP-02: No loading indicator | HIGH | Low | UX bug fix |
| **P1** | SETTINGS-GAP-03: No load-error UI | MEDIUM | Low | UX reliability |
| **P1** | SETTINGS-GAP-04: No dirty-check for 3/4 tabs | MEDIUM | Medium | UX + performance |
| **P1** | SETTINGS-GAP-06: No cleanup confirmation | MEDIUM | Low | Safety |
| **P1** | SETTINGS-GAP-11: Hardcoded 200-item limit | MEDIUM | Medium | Functional limit |
| **P2** | SETTINGS-GAP-05: No unsaved changes guard | MEDIUM | Low | UX safety |
| **P2** | SETTINGS-GAP-08: Missing org member API methods | MEDIUM | Low | Missing feature |
| **P2** | SETTINGS-GAP-13: SettingsView too large | MEDIUM | Medium | Code quality |
| **P3** | SETTINGS-GAP-09: SSO inline API calls | LOW | Low | Code quality |
| **P3** | SETTINGS-GAP-10: Missing i18n in pending/assigned | LOW | Low | i18n |
| **P3** | SETTINGS-GAP-12: No member removal confirmation | LOW | Low | UX safety |
| **P3** | SETTINGS-GAP-14: Mixed storage strategy | LOW | Medium | Architecture |
| **P3** | SETTINGS-GAP-15: No reset to defaults | LOW | Low | UX enhancement |
