# `get_comments(post_id)` — Fetch Comments on a Post

**File:** `facebook.py`  
**Module:** Hermes Facebook Commenter Plugin  

## Description

Retrieves all comments on a specific Facebook post using the Meta Graph API.

## Signature

```python
def get_comments(post_id: str) -> list
```

## Parameters

| Parameter | Type    | Description                |
|-----------|---------|----------------------------|
| `post_id` | `str`   | The Facebook post ID       |

## Returns

`list` — A list of comment objects. Each comment contains:
- `id` — Comment ID
- `message` — Comment text
- `from` — Author info `{id, name}`
- `created_time` — Timestamp

## API Call

```
GET https://graph.facebook.com/v19.0/{post_id}/comments?fields=id,message,from,created_time
```

## Example

```python
from facebook import get_comments

comments = get_comments("123456789_987654321")
for c in comments:
    print(f"{c['from']['name']}: {c['message']}")
```
