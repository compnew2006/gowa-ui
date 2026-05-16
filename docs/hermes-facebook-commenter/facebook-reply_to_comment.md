# `reply_to_comment(comment_id, message)` — Reply to a Comment

**File:** `facebook.py`  
**Module:** Hermes Facebook Commenter Plugin  

## Description

Replies directly to a specific Facebook comment as the page.

## Signature

```python
def reply_to_comment(comment_id: str, message: str) -> dict
```

## Parameters

| Parameter    | Type    | Description           |
|--------------|---------|-----------------------|
| `comment_id` | `str`   | The comment ID        |
| `message`    | `str`   | The reply text        |

## Returns

`dict` — The JSON response from the Graph API. Contains `id` on success.

## API Call

```
POST https://graph.facebook.com/v19.0/{comment_id}/replies
```

## Example

```python
from facebook import reply_to_comment

result = reply_to_comment("987654321", "Thank you for your feedback!")
print(result)
```
