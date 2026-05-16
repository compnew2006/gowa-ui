# `webhook()` (POST) — Webhook Event Handler (Production)

**File:** `facebook_webhook_gunicorn.py`  
**Endpoint:** `POST /webhook`

## Description

Production-ready webhook receiver. Same logic as the basic version but with structured logging and event logging to file.

## Signature

```python
@app.route('/webhook', methods=['POST'])
def webhook()
```

## Behavior

1. Logs raw event to file via `log_event()`
2. Validates `object == 'page'`
3. Routes changes to `handle_comment_event()` or `handle_feed_event()`
4. Logs unhandled fields via `logger.info`

## Improvements Over Basic Version

- Structured logging with `logger.info`/`logger.warning`
- Raw events saved to log file
- Handles unknown fields gracefully with log message
