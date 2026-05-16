# `forward_to_hermes(payload)` — Forward Event to Hermes API

**File:** `facebook_webhook_gunicorn.py`

## Description

Forwards a webhook event to the Hermes API endpoint for AI processing. Handles failures by queueing events for retry.

## Signature

```python
def forward_to_hermes(payload)
```

## Parameters

| Parameter | Type    | Description                    |
|-----------|---------|--------------------------------|
| `payload` | `dict`  | The event data to forward      |

## Behavior

- POSTs to `HERMES_API_URL/webhook/facebook` with 5-second timeout
- Logs success or failure via `logger`
- On failure (connection error), calls `queue_for_retry(payload)`

## Environment Variables

- `HERMES_API_URL` — Hermes API base URL (default: `http://localhost:8080`)
