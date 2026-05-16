"""
Token bucket rate limiter for Facebook Graph API calls.
Prevents hitting Meta API rate limits.
"""
import time
import logging
import threading
from typing import Dict, Optional

logger = logging.getLogger(__name__)

class RateLimiter:
    """Token bucket rate limiter. Per-business and global limits."""
    
    # Meta API typical limits
    GLOBAL_MAX_CALLS = 200       # per hour per app
    BUSINESS_MAX_CALLS = 200     # per hour per page
    WINDOW_SECONDS = 3600        # 1 hour window
    
    def __init__(self, max_calls: int = BUSINESS_MAX_CALLS, window: int = WINDOW_SECONDS):
        self.max_calls = max_calls
        self.window = window
        self.buckets: Dict[str, list] = {}  # key -> [timestamp1, timestamp2, ...]
        self._lock = threading.Lock()
    
    def _get_bucket(self, key: str) -> list:
        if key not in self.buckets:
            self.buckets[key] = []
        return self.buckets[key]
    
    def _prune(self, key: str):
        """Remove timestamps outside the window."""
        now = time.time()
        bucket = self._get_bucket(key)
        cutoff = now - self.window
        self.buckets[key] = [t for t in bucket if t > cutoff]
    
    def acquire(self, key: str = "global") -> bool:
        """Try to acquire a rate limit slot. Returns True if allowed."""
        with self._lock:
            self._prune(key)
            bucket = self._get_bucket(key)
            if len(bucket) < self.max_calls:
                bucket.append(time.time())
                return True
            return False
    
    def wait_and_acquire(self, key: str = "global", timeout: float = 60.0) -> bool:
        """Block until a slot is available or timeout expires."""
        start = time.time()
        while time.time() - start < timeout:
            if self.acquire(key):
                return True
            sleep_time = min(1.0, timeout - (time.time() - start))
            if sleep_time > 0:
                time.sleep(sleep_time)
        logger.warning(f"Rate limit timeout for {key}")
        return False
    
    def get_remaining(self, key: str = "global") -> int:
        """Get remaining calls allowed in current window."""
        with self._lock:
            self._prune(key)
            return self.max_calls - len(self._get_bucket(key))


# Global instance
rate_limiter = RateLimiter()


def rate_limited(key: str = "global"):
    """Decorator for rate-limited API calls."""
    def decorator(func):
        def wrapper(*args, **kwargs):
            if not rate_limiter.wait_and_acquire(key):
                logger.error(f"Rate limit exceeded for {key}")
                return {"error": "Rate limit exceeded. Try again later."}
            return func(*args, **kwargs)
        return wrapper
    return decorator
