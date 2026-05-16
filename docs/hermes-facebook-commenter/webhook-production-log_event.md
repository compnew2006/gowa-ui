# `log_event(event_data)` — Log Events to File

**File:** `facebook_webhook_gunicorn.py`

## Description

Appends a log entry for each webhook event to a persistent file for debugging and audit trails.

## Signature

```python
def log_event(event_data)
```

## Parameters

| Parameter    | Type    | Description                   |
|--------------|---------|-------------------------------|
| `event_data` | `str`   | The event string to log       |

## Behavior

1. Ensures log directory `~/.hermes/` exists
2. Appends the event string to `~/.hermes/webhook_events.log`
3. Each line is a separate event

## Example Log File

```
EVENT: {"object": "page", "entry": [...]}
COMMENT: John Doe: Great post!
EVENT: {"object": "page", "entry": [...]}
COMMENT: Jane: Thanks!
```
