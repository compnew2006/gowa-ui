# `webhook()` (POST) — Receive Webhook Events

**File:** `facebook_webhook.py`  
**Endpoint:** `POST /webhook`

## Description

Receives incoming webhook events from Meta. Processes page updates and routes comment and feed events to the appropriate handlers.

## Signature

```python
@app.route('/webhook', methods=['POST'])
def webhook()
```

## Request Body

```json
{
  "object": "page",
  "entry": [
    {
      "changes": [
        {
          "field": "comments",
          "value": { ... }
        }
      ]
    }
  ]
}
```

## Behavior

- Validates `data.get('object') == 'page'`
- Iterates over entries and their changes
- Routes to `handle_comment_event()` for field `"comments"`
- Routes to `handle_feed_event()` for field `"feed"`
- Always returns `"OK", 200`

## Returns

- `200` with `"OK"` string (always, even for unhandled events)
