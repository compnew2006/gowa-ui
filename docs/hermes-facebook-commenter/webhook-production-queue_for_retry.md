# `queue_for_retry(payload)` — Queue Failed Events for Retry

**File:** `facebook_webhook_gunicorn.py`

## Description

Saves failed webhook events to a retry queue directory on disk for later reprocessing.

## Signature

```python
def queue_for_retry(payload)
```

## Parameters

| Parameter | Type    | Description                    |
|-----------|---------|--------------------------------|
| `payload` | `dict`  | The event that failed to forward |

## Behavior

1. Creates `~/.hermes/webhook_queue/` directory if needed
2. Saves payload as `{timestamp}_{comment_id}.json`
3. Logs the queue file path

## File Format

```
~/.hermes/webhook_queue/1715850000_123456789.json
```

## Example Queued File

```json
{
  "type": "new_comment",
  "comment_id": "123",
  "post_id": "456",
  "message": "Hello!",
  "from": "John",
  "raw": { ... }
}
```
