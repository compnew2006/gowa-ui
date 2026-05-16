# `handle_comment_event(value)` — Process New Comment

**File:** `facebook_webhook.py`

## Description

Processes a new comment event from the webhook. Extracts comment details and forwards the event to the Hermes API for AI-powered processing.

## Signature

```python
def handle_comment_event(value)
```

## Parameters

| Parameter | Type    | Description                          |
|-----------|---------|--------------------------------------|
| `value`   | `dict`  | The comment change value from Meta   |

## Value Fields

| Field        | Description                      |
|--------------|----------------------------------|
| `id`         | Comment ID                       |
| `post_id`    | Parent post ID                   |
| `message`    | Comment text                     |
| `from.name`  | Commenter's display name         |
| `verb`       | Action type (e.g. `"added"`)     |

## Behavior

1. Extracts `comment_id`, `post_id`, `message`, `from_name`
2. Prints log: `📩 New comment from {name}: {message}`
3. Forwards to `HERMES_API_URL/webhook/facebook` as JSON

## Forwarded Payload

```json
{
  "type": "new_comment",
  "comment_id": "123",
  "post_id": "456",
  "message": "Great post!",
  "from": "John Doe",
  "raw": { ... }
}
```
