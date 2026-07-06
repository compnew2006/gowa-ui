# Campaigns Workflow Documentation

## Overview

The campaigns system enables organizations to send bulk WhatsApp messages to multiple recipients using pre-defined templates. It supports two WhatsApp providers:

- **Meta Cloud API** — template-based sending via WhatsApp Business API
- **Whatsmeow** — direct WhatsApp Web protocol sending with custom message bodies

---

## Architecture

### Backend Components

```
cmd/whatomate/main.go              Route registration (lines 1394-1409)
internal/handlers/campaigns.go     HTTP handlers (15 endpoints)
internal/handlers/campaign_policy.go  Start policy enforcement
internal/handlers/instance_auto_campaign_worker.go  Auto-campaign scheduler
internal/handlers/instance_auto_campaign_media.go   Auto-campaign media upload
internal/handlers/app.go           Campaign stats WebSocket subscriber
internal/worker/worker.go          Campaign job processor (HandleRecipientJob)
internal/worker/campaign_delay.go  Rate-limiting delay (Redis Lua script)
internal/worker/campaign_template_placeholders.go  Template placeholder resolution
internal/worker/send_policy.go     Organization send policy enforcement
internal/worker/idempotency.go     Recipient processing locks
internal/worker/scaler.go          Per-organization worker autoscaling
internal/queue/redis.go            Redis Streams queue (tenant-scoped)
internal/queue/pubsub.go           Redis pub/sub for real-time stats
internal/models/bulk.go            GORM models (BulkMessageCampaign, BulkMessageRecipient)
internal/models/constants.go       CampaignStatus, MessageStatus enums
internal/models/roles.go           RBAC: campaigns:read/write/delete/execute, catalogs:read/write/delete/sync, group_directory:read/write/delete/import, group_participants:read/write
pkg/whatsapp/message.go            Meta Cloud API send
pkg/whatsmeow/adapter_send.go      Whatsmeow protocol send
pkg/whatsmeow/events.go            Receipt handling
pkg/whatsmeow/events_campaign_pause.go  Auto-pause on instance ban/logout
pkg/whatsmeow/instance_auto_campaign_settings.go  Auto-campaign settings
pkg/provider/interface.go          MessageProvider interface
```

### Frontend Components

```
frontend/src/views/settings/CampaignsView.vue          Main campaign view (3,535 lines)
frontend/src/components/whatsmeow/AutoCampaignSettingsPanel.vue  Auto-campaign config
frontend/src/services/api.ts         campaignsService (14 methods)
frontend/src/services/websocket.ts   campaign_stats_update handling
frontend/src/stores/config.ts        campaigns feature flag
frontend/src/router/index.ts         /settings/campaigns route
frontend/src/lib/instance-auto-campaign.ts  Auto-campaign types & utilities
frontend/src/i18n/locales/{en,es,ar}.json  ~180 i18n keys
```

---

## Data Models

### BulkMessageCampaign (`bulk_message_campaigns`)

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid.UUID | Primary key |
| OrganizationID | uuid.UUID | Tenant scope |
| WhatsAppAccount | string(100) | Sender account/instance identifier |
| Name | string(255) | Campaign name |
| TemplateID | uuid.UUID | Associated message template |
| HeaderMediaID | text | Meta header media ID |
| HeaderMediaFilename | text | Original filename (display only) |
| HeaderMediaMimeType | text | Media MIME type |
| HeaderMediaLocalPath | text | Local filesystem path (whatsmeow) |
| MinDelaySeconds | int | Min delay between sends (default: 20) |
| MaxDelaySeconds | int | Max delay between sends (default: 45) |
| Status | CampaignStatus | Current lifecycle status |
| TotalRecipients | int | Total recipient count |
| SentCount | int | Successfully sent count |
| DeliveredCount | int | Delivered receipt count (Meta only) |
| ReadCount | int | Read receipt count (Meta only) |
| FailedCount | int | Failed send count |
| ScheduledAt | *time.Time | Scheduled start time (currently unused) |
| StartedAt | *time.Time | Actual start timestamp |
| CompletedAt | *time.Time | Completion timestamp |
| CreatedBy | uuid.UUID | Creator user ID |
| PollQuestion | text | Poll question (empty = no poll) |
| PollOptions | JSONBArray | Poll option strings (default []) |
| PollMaxSelections | int | Max selectable options (default 0 = unlimited) |

### BulkMessageRecipient (`bulk_message_recipients`)

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid.UUID | Primary key |
| CampaignID | uuid.UUID | Parent campaign |
| PhoneNumber | string(50) | Original phone input |
| PhoneNormalized | string(32) | Digits-only normalized phone |
| RecipientName | string(255) | Optional display name |
| TemplateParams | JSONB | Per-recipient template parameters |
| Status | MessageStatus | pending/sent/delivered/read/failed |
| WhatsAppMessageID | string(100) | WhatsApp message ID after send |
| MessageID | *uuid.UUID | Link to Message record |
| ErrorMessage | text | Failure reason |
| SentAt | *time.Time | Send timestamp |
| DeliveredAt | *time.Time | Delivery receipt timestamp |
| ReadAt | *time.Time | Read receipt timestamp |

**Unique index**: `(campaign_id, phone_normalized)` prevents duplicate recipients per campaign.

---

## Campaign Status Lifecycle

```
draft ──────────► scheduled* ──► processing ──► completed
  │                    │              │
  │                    │              ├────► paused ──► processing (resume)
  │                    │              │
  │                    │              └────► failed
  │                    │
  │                    └──────────────┘ (manual start before scheduled time)
  │
  └────► cancelled (from any non-terminal state)
```

* `scheduled` status is defined but never automatically assigned by the system. See Gap Analysis.

### Status Transitions

| Current Status | Action | New Status | Handler |
|----------------|--------|------------|---------|
| draft | Start | processing | `StartCampaign` |
| scheduled | Start | processing | `StartCampaign` |
| paused | Start | processing | `StartCampaign` (resume) |
| processing | Pause | paused | `PauseCampaign` |
| processing/queued | Cancel | cancelled | `CancelCampaign` |
| draft/processing/queued | Cancel | cancelled | `CancelCampaign` |
| completed/paused/failed | Retry Failed | processing | `RetryFailed` |
| processing | (auto) | completed | `checkCampaignCompletion` |
| processing/queued | (auto) | paused | `pauseActiveCampaignsForInstance` |

---

## API Endpoints

### Campaign CRUD

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| GET | `/api/campaigns` | `ListCampaigns` | campaigns:read | List with filters (status, account, search, date range) and pagination |
| POST | `/api/campaigns` | `CreateCampaign` | campaigns:write | Create campaign with name, sender, template, delays |
| GET | `/api/campaigns/{id}` | `GetCampaign` | campaigns:read | Get campaign details with template preload |
| PUT | `/api/campaigns/{id}` | `UpdateCampaign` | campaigns:write | Update draft campaign fields |
| DELETE | `/api/campaigns/{id}` | `DeleteCampaign` | campaigns:delete | Delete campaign (not allowed if processing/queued) |

### Campaign Actions

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| POST | `/api/campaigns/{id}/start` | `StartCampaign` | campaigns:execute | Start or resume campaign |
| POST | `/api/campaigns/{id}/pause` | `PauseCampaign` | campaigns:execute | Pause running campaign |
| POST | `/api/campaigns/{id}/cancel` | `CancelCampaign` | campaigns:execute | Cancel campaign |
| POST | `/api/campaigns/{id}/retry-failed` | `RetryFailed` | campaigns:execute | Retry all failed recipients |
| GET | `/api/campaigns/{id}/progress` | `GetCampaign` | campaigns:read | Get campaign progress (same as GET /{id}) |

### Recipients

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| POST | `/api/campaigns/{id}/recipients/import` | `ImportRecipients` | campaigns:write | Bulk import recipients with dedup |
| GET | `/api/campaigns/{id}/recipients` | `GetCampaignRecipients` | campaigns:read | List all recipients |
| DELETE | `/api/campaigns/{id}/recipients/{recipientId}` | `DeleteCampaignRecipient` | campaigns:write | Remove recipient from draft |

### Media

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| POST | `/api/campaigns/{id}/media` | `UploadCampaignMedia` | campaigns:write | Upload header media (16MB max) |
| GET | `/api/campaigns/{id}/media` | `ServeCampaignMedia` | campaigns:read | Serve media for preview |

### Poll Analytics (Plugin: campaign-interactive)

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| GET | `/api/campaigns/{id}/poll/votes` | Plugin handler | campaigns:read | Get aggregated poll vote counts and percentages |

### Auto-Campaign (Whatsmeow Only)

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| POST | `/api/instances/{id}/auto-campaign/media` | `UploadInstanceAutoCampaignMedia` | instances:write | Upload auto-campaign media (16MB max) |

---

## Middleware Chain

All campaign routes pass through:

1. **CORS** — Cross-origin headers
2. **Observability** — Request tracing/metrics
3. **Security Headers** — X-Content-Type-Options, etc.
4. **Request Logger** — Structured logging
5. **Recovery** — Panic recovery
6. **License Check** — Rejects if license invalid/expired
7. **CSRF Protection** — Double-submit cookie (skipped for Bearer/API-key auth)
8. **Auth (JWT + API Key)** — User authentication
9. **Tenant Scope** — Multi-tenant DB isolation via X-Organization-ID
10. **Handler-level RBAC** — Per-action permission check

---

## Campaign Workflow (End-to-End)

### 1. Campaign Creation

**Manual (API/Frontend):**
1. User creates campaign via `POST /api/campaigns` with name, sender (WhatsApp account or instance), template ID, and delay settings
2. Backend validates: sender exists, is connected, not send-blocked, template belongs to org
3. For whatsmeow: auto-creates a template from `body_content` if no template ID provided
4. Campaign created with status `draft`

**Automatic (Auto-Campaign Worker):**
1. `InstanceAutoCampaignWorker` runs every minute (configurable interval)
2. For each whatsmeow instance with auto-campaign enabled:
   - Checks if enough time has passed since last generation (`interval_days`)
   - Loads contacts with inbound messages in the time window
   - Creates campaign + recipients in DB
   - If `target_status = "run"`: validates policy and starts immediately
   - If `target_status = "draft"`: campaign stays as draft for manual review

### 2. Recipient Management

Recipients can be added via three methods:

**Manual Entry:**
- User enters phone numbers and optional names in a text area
- One per line, comma-separated, or tab-separated

**From Contacts:**
- User selects from existing contacts filtered by instance and date basis
- Contacts are deduplicated by normalized phone number

**CSV Upload:**
- User uploads a CSV file with phone number column (required) and parameter columns
- Full validation: column mapping, phone format, duplicate detection, parameter type consistency
- Supports `{{param}}` placeholder columns

All methods enforce:
- Phone normalization (digits-only)
- Deduplication by `(campaign_id, phone_normalized)` unique index
- Draft-only enforcement (cannot add recipients to running campaigns)
- Strict inbound-only policy (when enabled, all recipients must have inbound history)

### 3. Campaign Start

`POST /api/campaigns/{id}/start`:

1. Validates campaign is in `draft`, `scheduled`, or `paused` status
2. Enforces start policy:
   - `CampaignDraftOnly` org setting check
   - For whatsmeow: validates instance connected and not send-blocked
3. Validates delay floor (10-second minimum in strict mode)
4. Checks pending recipients exist
5. Validates strict inbound-only policy (all recipients must have inbound history)
6. Sets status to `processing`, records `started_at`
7. Enqueues all pending recipients as `RecipientJob` items to org-scoped Redis Stream
8. On enqueue failure: reverts status to `draft`

### 4. Queue Processing

**Queue Architecture:**
- Redis Streams with consumer groups (tenant-scoped per organization)
- Stream name: `whatomate:campaigns:<orgID>`
- Consumer group: `campaign-workers:<orgID>`
- Dead-letter queue: `whatomate:campaigns:<orgID>:dlq`
- Max delivery attempts: 5 before DLQ
- Stale pending claim interval: 5 minutes

**Worker Autoscaling:**
- `WorkerScaler` runs reconcile loop every 15 seconds
- Scales per-organization worker count based on queue depth (`XLEN`)
- Respects global worker budget and license caps (`max_workers_per_org`)
- Freezes organizations on: 3 consecutive start failures, no healthy instances (if `PauseOnDisconnect` enabled)
- Scale-up/down cooldowns prevent thrashing

**Job Processing (`HandleRecipientJob`):**

1. Acquire Redis idempotency lock on recipient ID (2-minute TTL)
2. Load recipient, skip if already processed (sent/delivered/read)
3. Load campaign with template, skip if paused/cancelled
4. Load organization send policy, check `CampaignDraftOnly` and strict inbound-only
5. Get or create Contact record for the phone number
6. For whatsmeow: validate instance connected, not blocked, has inbound history
7. Resolve template placeholders (`{{customer_name}}`, `{{contact_name}}`, `{{phone_number}}`, `{{chat_id}}`, `{{agent_name}}`, `{{organization_name}}`)
8. Apply inter-send delay via Redis Lua script (random delay in `[minDelay, maxDelay]` range, 10s floor)
9. Send message:
   - **Meta**: `sendTemplateMessage` — builds template components, calls WhatsApp Cloud API
   - **Whatsmeow**: `sendTemplateMessageViaProvider` — renders body, attaches media, sends via protocol adapter
   - **Whatsmeow Poll**: if campaign has `PollQuestion` set, sends native WhatsApp poll via `PollProvider.SendPoll()` instead of plain text
10. Create `Message` record in DB
11. Update recipient status to `sent` or `failed`
12. Atomically increment campaign counter (`sent_count` or `failed_count`)
13. Check campaign completion (all recipients processed)
14. Release idempotency lock

### 5. Progress Tracking

**Real-time Updates:**
- Worker publishes `CampaignStatsUpdate` to Redis pub/sub channel `whatomate:campaign_stats`
- `StartCampaignStatsSubscriber` in `app.go` subscribes and re-broadcasts via WebSocket
- Frontend subscribes to `wsService.onCampaignStatsUpdate()` for live counter updates
- WebSocket message type: `campaign_stats_update`

**Meta Delivery Receipts:**
- WhatsApp sends delivery receipts to `POST /api/webhook`
- `applyMessageStatusUpdate` checks for `campaign_id` in message metadata
- Updates `BulkMessageRecipient` status and timestamps
- Atomically increments `delivered_count` / `read_count` on campaign
- Broadcasts via WebSocket

**Campaign Completion:**
- After each job, `checkCampaignCompletion` counts pending recipients
- If zero pending: atomically marks campaign as `completed` with `completed_at` timestamp
- Uses CAS: `UPDATE ... SET status = 'completed' WHERE id = ? AND status = 'processing'`

### 6. Campaign Actions

**Pause:**
- Sets status from `processing`/`queued` to `paused`
- Workers skip jobs for paused campaigns (checked at step 3 of job processing)
- Auto-pause triggers when WhatsApp instance is banned or logged out

**Cancel:**
- Sets status to `cancelled` from any non-terminal state
- Workers skip jobs for cancelled campaigns
- No cleanup of queued-but-unprocessed Redis Stream messages

**Retry Failed:**
- Resets all `failed` recipients to `pending`
- Resets associated message records
- Recalculates campaign stats from messages table
- Sets campaign to `processing`
- Enqueues only the failed recipients as new jobs

**Delete:**
- Only allowed for `draft` campaigns
- Cascading delete of all recipients
- No cleanup of associated message records

---

## Auto-Campaign System (Whatsmeow Only)

### Configuration

Per-instance settings stored in instance JSONB:

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| enabled | bool | false | Enable/disable auto-campaign |
| name_prefix | string | "Auto" | Prefix for auto-generated campaign names |
| message | string | "" | Message template body |
| interval_days | int | 7 | Days between auto-generations |
| min_delay_minutes | int | 1 | Min delay between messages |
| max_delay_minutes | int | 3 | Max delay between messages |
| target_status | string | "draft" | "draft" (manual review) or "run" (auto-start) |
| media_local_path | string | "" | Path to attached media file |
| media_mime_type | string | "" | Media MIME type |
| media_filename | string | "" | Original media filename |
| last_generated_at | string | "" | ISO timestamp of last generation |

### Worker Flow

1. `InstanceAutoCampaignWorker` runs every minute
2. For each whatsmeow instance with auto-campaign enabled:
   - Check `isAutoCampaignDue(now, lastGeneratedAt, intervalDays)`
   - Determine time window since last generation
   - Load contacts with inbound messages in the window (excluding groups/channels)
   - Deduplicate by normalized phone
   - Check for duplicate campaign name
   - Create campaign + recipients in DB
   - Auto-resolve template or create from message body
   - If `target_status = "run"`: enforce policy and start campaign
   - Persist `last_generated_at`

---

## Rate Limiting & Throttling

### Inter-Message Delay

- Configured per campaign: `min_delay_seconds` and `max_delay_seconds`
- Defaults: 20s min, 45s max
- Strict floor: 10 seconds (enforced even if configured lower)
- Implemented via Redis Lua script for atomic slot reservation
- Scope: per-instance (shared across campaigns) or per-campaign (fallback)

### Worker Scaling

- Autoscaled based on queue depth per organization
- Default: 1 worker per 25 pending jobs
- Min: 0, Max: 4 per org (configurable via `worker_scaler` settings)
- Global worker budget allocated proportionally by backlog ratio
- License cap applied to `max_workers_per_org`

---

## RBAC Permissions

| Permission | Actions Allowed |
|------------|----------------|
| campaigns:read | List, view, get recipients, view media |
| campaigns:write | Create, update, import recipients, delete recipient, upload media |
| campaigns:delete | Delete campaign |
| campaigns:execute | Start, pause, cancel, retry failed |

**Role defaults:**
- Admin: all 4 permissions
- Manager: all 4 permissions
- Agent: no campaign permissions

---

## Organization Policies

| Policy | Setting Key | Effect |
|--------|-------------|--------|
| Draft Only | `campaign_draft_only` | Campaigns can only be created, never started |
| Strict Inbound Only | `strict_sending_restrictions_enabled` + `outbound_mode: inbound_only` + `strict_sending_apply_to_system: true` | Only recipients with prior inbound history can receive campaign messages |
| Send Block | Instance-level `send_blocked` | All campaigns for the instance are paused |

---

## WebSocket Protocol

**Subscribe to campaign stats:**

```typescript
wsService.onCampaignStatsUpdate((payload) => {
  // payload.campaign_id, payload.status
  // payload.sent_count, payload.delivered_count, payload.read_count, payload.failed_count
});
```

**Message type:** `campaign_stats_update`

---

## Testing

### Backend Tests

| File | Tests | Coverage |
|------|-------|----------|
| `internal/handlers/campaigns_test.go` | 35 tests | CRUD, lifecycle, policy enforcement, cross-org isolation |
| `internal/handlers/campaigns_helpers_test.go` | 63 sub-tests | Phone normalize, MIME types, filename sanitize, delay range |
| `internal/handlers/campaign_policy_test.go` | 13 sub-tests | Delay floor, policy violation errors |
| `internal/handlers/instance_auto_campaign_worker_test.go` | 2 tests | Window resolution |
| `internal/worker/campaign_delay_test.go` | 6 tests | Delay scoping, floor enforcement |
| `internal/queue/queue_test.go` | 7 tests | Queue isolation, pub/sub, job processing |
| `internal/handlers/webhook_test.go` | 5 tests | Delivery receipt → campaign stats |
| `internal/handlers/security_regression_test.go` | 3 tests | RBAC enforcement |
| `pkg/whatsmeow/instance_auto_campaign_settings_test.go` | 4 tests | Settings normalization, validation |

### Frontend Tests

| File | Tests |
|------|-------|
| `frontend/e2e/tests/campaigns/campaigns.spec.ts` | 25 E2E tests |
| `frontend/e2e/tests/campaigns/campaign-templates.spec.ts` | 7 E2E tests |
| `frontend/e2e/tests/campaigns/campaigns-contacts-source.spec.ts` | 1 E2E test |
| `frontend/src/lib/instance-auto-campaign.test.ts` | 7 unit tests |
