# `handle_comment_event(value)` — Process Comment (Production)

**File:** `facebook_webhook_gunicorn.py`

## Description

Production version of comment event handling. Includes logging, event file logging, and forwards to Hermes via `forward_to_hermes()`.

## Signature

```python
def handle_comment_event(value)
```

## Parameters

| Parameter | Type    | Description                     |
|-----------|---------|---------------------------------|
| `value`   | `dict`  | Comment change value from Meta  |

## Behavior

1. Extracts `comment_id`, `post_id`, `message`, `from_name`, `verb`
2. Logs via `logger.info` and `log_event()`
3. Builds payload with type, comment_id, post_id, message, from, verb, raw
4. Calls `forward_to_hermes(payload)`

## Improvements Over Basic Version

- Captures `verb` field (added, edited, etc.)
- Uses `log_event()` for persistent event logging
- Calls `forward_to_hermes()` instead of inline requests
