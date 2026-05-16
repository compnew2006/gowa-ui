# `publish_post(message)` — Publish a Post to Facebook Page

**File:** `facebook.py`  
**Module:** Hermes Facebook Commenter Plugin  

## Description

Publishes a new text post to the configured Facebook page via the Meta Graph API (`v19.0`).

## Signature

```python
def publish_post(message: str) -> dict
```

## Parameters

| Parameter | Type   | Description                        |
|-----------|--------|------------------------------------|
| `message` | `str`  | The text content of the post       |

## Returns

`dict` — The JSON response from the Facebook Graph API. On success contains an `id` field with the new post ID.

## Dependencies

- Environment variables:
  - `FB_PAGE_ACCESS_TOKEN` — Page-scoped access token
  - `FB_PAGE_ID` — The Facebook Page ID to post to
- `requests` library

## API Call

```
POST https://graph.facebook.com/v19.0/{PAGE_ID}/feed
```

## Example

```python
from facebook import publish_post

result = publish_post("Hello from Hermes!")
print(result)
# {'id': '123456789_987654321'}
```
