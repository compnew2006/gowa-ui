# `RateLimiter` — Token Bucket Rate Limiter

**File:** `rate_limiter.py` (new)

## Description

Thread-safe token bucket rate limiter for Facebook Graph API calls. Prevents hitting Meta API rate limits (200 calls/hour per page, 200/hour per app). Used by `facebook.py` and `multi_business_facebook.py`.

## Class: `RateLimiter`

### Constants

| Constant            | Value | Description                    |
|---------------------|-------|--------------------------------|
| `GLOBAL_MAX_CALLS`  | 200   | Max calls per hour per app     |
| `BUSINESS_MAX_CALLS`| 200   | Max calls per hour per page    |
| `WINDOW_SECONDS`    | 3600  | Sliding window in seconds (1h) |

### `__init__(max_calls, window)`

Creates a rate limiter with configurable limits and window size.

### `acquire(key) -> bool`

Try to acquire a rate limit slot. Returns `True` if within limit, `False` if exceeded.

### `wait_and_acquire(key, timeout) -> bool`

Blocks until a slot is available or timeout expires. Sleeps 1s between retries. Default timeout: 60s.

### `get_remaining(key) -> int`

Returns remaining calls allowed in the current window.

## Module-Level

```python
rate_limiter = RateLimiter()  # Global instance

def rate_limited(key: str = "global"):
    """Decorator for rate-limiting API functions"""
```

## Example

```python
from rate_limiter import rate_limiter, rate_limited

@rate_limited(key="page:123456789")
def my_api_call():
    ...

# Or inline:
if rate_limiter.wait_and_acquire("business:bakery"):
    result = requests.post(...)
```
