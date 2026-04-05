---
title: Campaigns
---

# Campaigns

Create, manage, and monitor bulk WhatsApp messaging campaigns. Send templated messages to large groups of contacts with customizable delays, scheduling, and retry capabilities.

## Campaign Overview

A campaign sends a WhatsApp template message to a list of recipients. Each campaign tracks delivery status, success rates, and failures in real time.

### Campaign States

| State | Description |
|-------|-------------|
| **Draft** | Campaign created but not yet started |
| **Running** | Messages are being sent |
| **Paused** | Sending temporarily stopped |
| **Completed** | All recipients processed |
| **Cancelled** | Campaign stopped before completion |

## Creating a Campaign

**Endpoint:** `POST /api/campaigns`

```json
{
  "name": "Holiday Promotion",
  "whatsapp_account_id": "account-uuid",
  "template_id": "template-uuid",
  "body_content": "Hi {{1}}, check out our holiday deals!",
  "header_media_id": "media-uuid",
  "min_delay_seconds": 20,
  "max_delay_seconds": 45,
  "scheduled_at": "2024-12-01T09:00:00Z"
}
```

**Requirements:**

- The template must exist and be in **approved** status
- The WhatsApp account must be active
- Delay settings must be within acceptable range (minimum 20 seconds, maximum 45 seconds by default)
- At least one recipient must be added before starting

## Listing Campaigns

**Endpoint:** `GET /api/campaigns`

Filter campaigns by:

| Filter | Description |
|--------|-------------|
| **Status** | Filter by campaign state |
| **Account** | Filter by WhatsApp account |
| **Search** | Search by campaign name |
| **Date range** | Filter by creation date |

Each campaign entry includes real-time progress stats:

```json
{
  "id": "campaign-uuid",
  "name": "Holiday Promotion",
  "status": "running",
  "total_recipients": 500,
  "sent_count": 234,
  "delivered_count": 210,
  "failed_count": 5,
  "progress_percent": 46.8,
  "template": { "name": "holiday_promo" },
  "account": { "name": "Main Account" }
}
```

## Updating a Campaign

**Endpoint:** `PUT /api/campaigns/{id}`

Update campaign details. **Note:** Active or running campaigns cannot be modified.

## Deleting a Campaign

**Endpoint:** `DELETE /api/campaigns/{id}`

Remove a campaign. **Note:** Active or running campaigns cannot be deleted.

## Managing Recipients

### Import Recipients

**Endpoint:** `POST /api/campaigns/{id}/recipients/import`

Upload a CSV or JSON file containing recipient data:

```csv
phone_number,name,param1,param2
+1234567890,Alice,order_123,10
+0987654321,Bob,order_456,20
```

**What happens:**

1. The file is parsed and validated
2. Phone numbers are checked for correct format
3. Recipients are bulk-inserted into the campaign
4. The campaign's `total_recipients` count is updated

### View Recipients

**Endpoint:** `GET /api/campaigns/{id}/recipients`

List all recipients with their individual delivery status:

| Status | Description |
|--------|-------------|
| **Pending** | Not yet processed |
| **Sent** | Message sent successfully |
| **Delivered** | Message delivered to recipient |
| **Failed** | Send failed (with error reason) |
| **Cancelled** | Skipped due to campaign cancellation |

### Remove a Recipient

**Endpoint:** `DELETE /api/campaigns/{id}/recipients/{recipientId}`

Remove a specific recipient from the campaign.

## Running a Campaign

### Start

**Endpoint:** `POST /api/campaigns/{id}/start`

Before starting, the system validates:

1. The campaign has at least one recipient
2. The template is approved
3. The WhatsApp account is active
4. Delay settings are valid

**What happens:**

1. Campaign status changes to `running`
2. Recipients are published to the Redis processing queue
3. Workers pick up jobs and send messages with randomized delays
4. Progress is broadcast via WebSocket in real time

### Pause

**Endpoint:** `POST /api/campaigns/{id}/pause`

Temporarily halt sending. Workers will skip recipients from paused campaigns. The campaign can be resumed later.

### Cancel

**Endpoint:** `POST /api/campaigns/{id}/cancel`

Permanently stop the campaign:

1. Status changes to `cancelled`
2. All pending recipients are marked as cancelled
3. The `completed_at` timestamp is recorded

### Retry Failed Recipients

**Endpoint:** `POST /api/campaigns/{id}/retry-failed`

Re-attempt sending to all recipients that previously failed:

1. Failed recipients are reset to `pending` status
2. They are re-published to the processing queue
3. Messages are resent with the same template and delays

## Campaign Media

### Upload Campaign Media

**Endpoint:** `POST /api/campaigns/{id}/media`

Upload media files (images, documents) to use in campaign template headers.

### Serve Campaign Media

**Endpoint:** `GET /api/campaigns/{id}/media`

Retrieve uploaded campaign media files.

## Campaign Policy Enforcement

Before a campaign can be created or started, the following policies are checked:

| Policy | Check |
|--------|-------|
| **Template approval** | Template must be in APPROVED status |
| **Account status** | WhatsApp account must be active |
| **Delay validation** | Min/max delay within acceptable range |
| **Recipient validation** | At least one recipient required |
| **Schedule validation** | Scheduled time must be in the future |
| **Rate limiting** | Organization campaign rate limits enforced |

If your organization has `campaign_draft_only` enabled in send restrictions, campaigns can only be created in draft mode and cannot be started directly.

## Template Placeholders

Campaign messages support dynamic placeholders that are resolved per recipient:

| Placeholder | Resolved From |
|-------------|---------------|
| `{{1}}`, `{{2}}`, etc. | Recipient-specific parameters from import |
| `{{contact.name}}` | Contact name |
| `{{contact.phone}}` | Contact phone number |
| `{{user.name}}` | Sending agent name |
| `{{organization.name}}` | Organization name |

**Example:** If your template body is `Hello {{1}}, your order {{2}} is ready!`, and the recipient import includes `param1=Alice` and `param2=ORD-123`, the message becomes `Hello Alice, your order ORD-123 is ready!`.

## Delay Settings

Each campaign has configurable delay between messages to avoid rate limiting:

| Setting | Default | Description |
|---------|---------|-------------|
| **Min delay** | 20 seconds | Minimum wait between messages |
| **Max delay** | 45 seconds | Maximum wait between messages |

The actual delay for each message is randomly selected between min and max to create natural-looking sending patterns.

## Background Processing

Campaign messages are processed by background workers:

1. Workers pick up recipient jobs from the Redis queue
2. A distributed lock prevents duplicate sends to the same recipient
3. A random delay is applied before each send
4. Template placeholders are resolved with recipient data
5. The message is sent via the WhatsApp provider
6. Recipient status and campaign stats are updated
7. Progress is broadcast via WebSocket

## See Also

- [Templates & Flows](templates-flows.md) — Creating message templates
- [Tags & Organization](tags-organization.md) — Import/export contact data
- [Analytics](analytics.md) — Viewing campaign performance metrics
- [Chat & Messaging](chat-messaging.md) — Sending individual messages
