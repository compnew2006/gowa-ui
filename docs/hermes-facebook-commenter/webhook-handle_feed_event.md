# `handle_feed_event(value)` — Process Feed Event

**File:** `facebook_webhook.py`

## Description

Processes feed change events from the webhook. Handles the alternative comment format that sometimes comes through the `feed` field instead of `comments`.

## Signature

```python
def handle_feed_event(value)
```

## Parameters

| Parameter | Type    | Description                       |
|-----------|---------|-----------------------------------|
| `value`   | `dict`  | The feed change value from Meta   |

## Behavior

- Only processes events where `value.get('item') == 'comment'`
- Extracts `comment_id`, `message`, `post_id`
- Forwards to Hermes API at `HERMES_API_URL/webhook/facebook`

## Forwarded Payload

```json
{
  "type": "new_comment",
  "comment_id": "123",
  "post_id": "456",
  "message": "Nice!",
  "from": "Unknown",
  "raw": { ... }
}
```
