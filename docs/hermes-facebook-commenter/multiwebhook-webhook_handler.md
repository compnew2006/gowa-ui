# `webhook()` (POST) — Webhook Handler (Multi-Business)

**File:** `multi_business_webhook.py`  
**Endpoint:** `POST /webhook`

## Description

Receives Meta webhook events and routes them to the correct business based on `page_id`. Logs all raw events to disk.

## Signature

```python
@app.route('/webhook', methods=['POST'])
def webhook()
```

## Behavior

1. Logs raw event via `log_event("raw", data)`
2. Validates `object == 'page'`
3. Routes changes to `handle_comment_event()` or `handle_feed_event()`
4. Always returns `"OK", 200`
