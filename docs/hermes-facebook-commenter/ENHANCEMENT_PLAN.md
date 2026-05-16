# Enhancement Plan — Hermes Facebook Commenter

## Overview

4 enhancements across 6 Python files. Each enhancement has its own agent assignment.

---

## Enhancement 1: Error Handling — `facebook.py`

**Scope:** `facebook.py` — 4 API functions

**Current:** Raw `requests.get/post` calls with no try/except. Any network error or non-JSON response crashes the caller.

**Target:**
- Wrap each API call in try/except
- Return `{"error": str(e)}` on network errors
- Return `res.json()` normally but handle non-200 status codes
- Log failures via `logging` module (add `import logging`)

**Files affected:** `docs/hermes-facebook-commenter/facebook.py`

---

## Enhancement 2: Logging — `facebook_webhook.py`

**Scope:** `facebook_webhook.py` — 92 lines

**Current:** Uses `print()` for all output. No timestamps, log levels, or file logging.

**Target:**
- Replace all `print()` with `logging` calls
- Add `logging.basicConfig()` with format including timestamps
- Use appropriate levels: `info`, `warning`, `error`
- Add log file handler to `~/.hermes/webhook_basic.log`
- Use `logger.info` in `verify_webhook()`, `webhook()`, `handle_*` functions

**Files affected:** `docs/hermes-facebook-commenter/facebook_webhook.py`

---

## Enhancement 3: Learning Engine — `multi_business_facebook.learn_from_interaction()`

**Scope:** `multi_business_facebook.py` — `learn_from_interaction()` method + supporting helpers

**Current:** Stub. Iterates services/prices but does nothing (`pass`). Only saves raw interaction to memory.

**Target:**
- Extract keyword frequency from customer messages
- Detect new service mentions not in current config → flag as "suggested services"
- Detect new price mentions → flag as "suggested prices"
- Track question types asked (price, hours, location, service, other)
- Save structured analytics: `{business_id}_analytics.jsonl`
- Add method `get_business_analytics(business_id)` to retrieve insights
- Add `_extract_keywords(text)` helper

**Files affected:** `docs/hermes-facebook-commenter/multi_business_facebook.py`

---

## Enhancement 4: Rate Limiting — All API Files

**Scope:**
- `facebook.py` — direct calls
- `multi_business_facebook.py` — `publish_post`, `get_comments`, `reply_to_comment`, `get_all_posts`

**Current:** No rate limiting. Can hit Meta API rate limits (200 calls/hour per page for most tiers).

**Target:**
- Add `RateLimiter` class with token bucket algorithm
- Track per-business + global rate limits
- Add exponential backoff on HTTP 429 responses
- Configurable limits via constants
- Apply to all Graph API calls in both files

**New file:** `rate_limiter.py` (shared utility)
**Modified files:** `facebook.py`, `multi_business_facebook.py`

---

## Deployment

After each enhancement:
1. Apply changes to local copy in `docs/hermes-facebook-commenter/`
2. SCP updated files to VPS: `root@31.97.192.53:/root/.hermes/plugins/`
3. Restart affected services (`systemctl restart hermes-facebook-webhook`)
4. Quick test: `curl` health endpoint + post test comment

---

## Deployment

After each enhancement:
1. Apply changes to local copy in `docs/hermes-facebook-commenter/`
2. SCP updated files to VPS: `root@31.97.192.53:/root/.hermes/plugins/`
3. Restart affected services (`systemctl restart hermes-facebook-webhook`)
4. Quick test: `curl` health endpoint + Python import test

### Enhancement 3 Note (Learning Engine)

The initial Ruflo agent (`fb-learning-engine`) described the implementation but did NOT actually write code. The `learn_from_interaction()` stub remained unchanged. Manual fix applied:
- Added `self.analytics = {}` in `__init__`
- Added `_extract_keywords(text)` — extracts 3+ char non-stop-words with `Counter`
- Rewrote `learn_from_interaction()` — detects question type, new services, price mentions, updates analytics
- Added `get_business_analytics(business_id)` — returns structured analytics from memory or filesystem
- Added convenience function `get_business_analytics()` at module level

---

## HTML Guide Generation

After all enhancements, generate a single-page HTML guide from all markdown files in `docs/hermes-facebook-commenter/`:
- Parse each `.md` file
- Build navigation sidebar by category
- Render as a clean HTML page (inline CSS, no external deps)
- Include function signatures, descriptions, parameters, examples
- Output to `docs/hermes-facebook-commenter/guide.html`

### Result
Generated `guide.html` (72 KB) — 47 function sections, dark theme (OKLCH), sidebar navigation, code copy buttons, cross-reference links, responsive layout. See the `fb-html-guide` agent output for full description.
