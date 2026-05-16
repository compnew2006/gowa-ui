# `get_all_posts(business_id)` — Get All Posts for a Business

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Retrieves all recent posts from a specific business's Facebook page, including nested comments.

## Signature

```python
def get_all_posts(self, business_id: str) -> list
```

## Parameters

| Parameter     | Type    | Description                     |
|---------------|---------|---------------------------------|
| `business_id` | `str`   | The business identifier         |

## Returns

`list` — List of post objects (each with `id`, `message`, `created_time`, `comments`), or empty list if business not found.

## API Call

```
GET https://graph.facebook.com/v19.0/{page_id}/posts?fields=id,message,created_time,comments{message,from}
```

## Example

```python
posts = manager.get_all_posts("maktabat_al_arkan")
for post in posts:
    print(post['message'])
```
