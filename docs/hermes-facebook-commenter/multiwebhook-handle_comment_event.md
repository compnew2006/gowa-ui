# `handle_comment_event(value)` — Process Comment (Multi-Business)

**File:** `multi_business_webhook.py`

## Description

The most sophisticated comment handler. Looks up the business by `page_id`, generates an auto-reply using the business's context, sends the reply, learns from the interaction, and forwards to Hermes.

## Signature

```python
def handle_comment_event(value)
```

## Parameters

| Parameter | Type    | Description                     |
|-----------|---------|---------------------------------|
| `value`   | `dict`  | Comment change value from Meta  |

## Behavior Flow

1. Extracts `page_id` from the event value
2. Looks up business via `manager.get_business_by_page_id(page_id)`
3. If business not found, logs error and returns
4. Extracts comment details (id, post_id, message, from_name, verb)
5. If `business.auto_reply` is enabled:
   a. Generates contextual reply via `manager.generate_reply()`
   b. Sends reply via `manager.reply_to_comment()`
   c. Logs success/failure
   d. Learns from interaction via `manager.learn_from_interaction()`
6. Always forwards event to Hermes API

## Key Feature

Auto-reply uses per-business knowledge (services, prices, location, hours) and auto-detects language.
