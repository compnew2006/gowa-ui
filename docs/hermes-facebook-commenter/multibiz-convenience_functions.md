# Module-Level Convenience Functions

**File:** `multi_business_facebook.py`

## Description

The module exposes a global `manager = MultiBusinessFacebookManager()` instance and several convenience functions that delegate to it, allowing simple `from multi_business_facebook import publish_post` usage.

## Functions

### `publish_post(business_id, message)`
```python
def publish_post(business_id: str, message: str) -> dict
```
Delegates to `manager.publish_post()`.

### `reply_to_comment(business_id, comment_id, message)`
```python
def reply_to_comment(business_id: str, comment_id: str, message: str) -> dict
```
Delegates to `manager.reply_to_comment()`.

### `get_comments(business_id, post_id)`
```python
def get_comments(business_id: str, post_id: str) -> list
```
Delegates to `manager.get_comments()`.

### `get_all_posts(business_id)`
```python
def get_all_posts(business_id: str) -> list
```
Delegates to `manager.get_all_posts()`.

### `add_business(business_id, config)`
```python
def add_business(business_id: str, config: dict) -> bool
```
Delegates to `manager.add_business()`.

### `generate_reply(business_id, comment, customer_name)`
```python
def generate_reply(business_id: str, comment: str, customer_name: str = None) -> str
```
Delegates to `manager.generate_reply()`.

### `list_businesses()`
```python
def list_businesses() -> list
```
Delegates to `manager.list_businesses()`.

### `get_business_info(business_id)`
```python
def get_business_info(business_id: str) -> dict
```
Returns a dict with `name`, `page_name`, `services`, `prices`, `location`, `hours`, `tone` for the given business, or empty dict if not found.
