# `handle_feed_event(value)` — Process Feed Event (Multi-Business)

**File:** `multi_business_webhook.py`

## Description

Handles feed events for multi-business setup. Looks up the business by `page_id` and generates auto-replies when enabled.

## Signature

```python
def handle_feed_event(value)
```

## Parameters

| Parameter | Type    | Description                     |
|-----------|---------|---------------------------------|
| `value`   | `dict`  | Feed change value from Meta     |

## Behavior

1. Only processes `value.get('item') == 'comment'`
2. Extracts `page_id`, finds business via `manager.get_business_by_page_id()`
3. If business not found, logs error
4. If `auto_reply` enabled:
   - Generates contextual reply
   - Sends via `manager.reply_to_comment()`
   - Logs success/failure
