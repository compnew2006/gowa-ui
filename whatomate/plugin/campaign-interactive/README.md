# Campaign Interactive Plugin

Provides poll vote tracking and analytics for campaign messages sent as native WhatsApp polls.

## Overview

When a campaign is configured with a poll question and options, messages are sent as native WhatsApp polls via the whatsmeow provider. This plugin provides a REST API to retrieve aggregated vote counts and percentages for those polls.

## Registration

Activated via blank import in `cmd/whatomate/main.go`:

```go
import _ "github.com/compnew2006/whatomate/plugin/campaign-interactive"
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/campaigns/{id}/poll/votes` | Get poll vote analytics for a campaign |

### GET /api/campaigns/{id}/poll/votes

Returns aggregated vote counts for a campaign's poll.

**Response (200):**

```json
{
  "question": "Did you enjoy this?",
  "options": ["Yes", "No", "Maybe"],
  "total": 3,
  "results": [
    { "option": "Yes", "count": 2, "percentage": "66.7%" },
    { "option": "No", "count": 1, "percentage": "33.3%" },
    { "option": "Maybe", "count": 0, "percentage": "0.0%" }
  ]
}
```

**Error responses:**
- `401` — Missing or invalid authentication
- `404` — Campaign not found or has no poll

## Files

| File | Purpose |
|------|---------|
| `plugin.go` | Plugin registration, `init()`, `Routes()`, `Migrate()` |
| `handler.go` | HTTP handler with tenant scoping |
| `service.go` | Vote counting business logic |

## How Vote Counting Works

1. Loads the campaign and verifies it has a `PollQuestion`
2. Collects all recipient `WhatsAppMessageID` values for the campaign
3. Queries the `messages` table for poll vote replies (`interactive_data->>'type' = 'poll_vote'`) linked to the original poll messages
4. Parses each vote's `interactive_data.selected_options` JSON array
5. Counts matches against the campaign's `PollOptions`
6. Returns per-option counts and percentages

## Tenant Scoping

All queries are scoped to the campaign's `OrganizationID`. The handler resolves the org ID via `middleware.GetOrganizationID(rc)` and validates it matches the campaign.

## Dependencies

- Campaign poll fields on `BulkMessageCampaign` (`poll_question`, `poll_options`, `poll_max_selections`) — defined in `internal/models/bulk.go`
- `MessageTypePoll` constant — defined in `internal/models/constants.go`
- Poll vote messages with `interactive_data.selected_options` — created by `pkg/whatsmeow/poll_vote.go`
