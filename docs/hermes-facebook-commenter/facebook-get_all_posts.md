# `get_all_posts()` — Get Recent Page Posts

**File:** `facebook.py`  
**Module:** Hermes Facebook Commenter Plugin  

## Description

Retrieves recent posts from the configured Facebook page, including their comments.

## Signature

```python
def get_all_posts() -> list
```

## Parameters

None. Uses environment variables `FB_PAGE_ACCESS_TOKEN` and `FB_PAGE_ID`.

## Returns

`list` — A list of post objects. Each post contains:
- `id` — Post ID
- `message` — Post text
- `created_time` — Timestamp
- `comments` — Nested comment data `{data: [{message, from}]}`

## API Call

```
GET https://graph.facebook.com/v19.0/{PAGE_ID}/posts?fields=id,message,created_time,comments{message,from}
```

## Example

```python
from facebook import get_all_posts

posts = get_all_posts()
for post in posts:
    print(f"[{post['id']}] {post['message'][:50]}")
```
