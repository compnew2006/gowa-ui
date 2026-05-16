# `handle_feed_event(value)` — Process Feed Event (Production)

**File:** `facebook_webhook_gunicorn.py`

## Description

Production version of feed event handling. Uses structured logging and the centralized `forward_to_hermes()` function.

## Signature

```python
def handle_feed_event(value)
```

## Parameters

| Parameter | Type    | Description                     |
|-----------|---------|---------------------------------|
| `value`   | `dict`  | Feed change value from Meta     |

## Behavior

- Only processes `value.get('item') == 'comment'`
- Extracts `comment_id`, `message`, `post_id`, `sender_name`
- Builds and forwards payload via `forward_to_hermes()`
