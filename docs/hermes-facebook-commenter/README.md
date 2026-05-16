# Hermes Facebook Commenter Agent — Documentation

> Auto-generated documentation of the Hermes Facebook Commenter plugin system.
> All source files pulled from VPS `31.97.192.53` — `/root/.hermes/plugins/`

## Source Files

| File | Lines | Description |
|------|-------|-------------|
| `facebook.py` | 87 | Simple single-page Facebook API wrapper (+ error handling, rate limiting) |
| `multi_business_facebook.py` | 505 | Multi-business manager with memory, auto-reply, learning engine, rate limiting |
| `facebook_webhook.py` | 111 | Basic Flask webhook for Meta events (+ proper logging) |
| `facebook_webhook_gunicorn.py` | 173 | Production webhook with gunicorn, logging, retry queue |
| `multi_business_webhook.py` | 263 | Multi-business webhook with auto-reply routing (fixed missing import) |
| `add_business.py` | 150 | Interactive CLI tool for adding businesses |
| `rate_limiter.py` | 80 | Token bucket rate limiter (shared utility, new) |

## Documentation Index

### `facebook.py` — Simple Facebook API
- [`facebook-publish_post`](facebook-publish_post.md) — Publish post to page
- [`facebook-get_comments`](facebook-get_comments.md) — Get post comments
- [`facebook-reply_to_comment`](facebook-reply_to_comment.md) — Reply to a comment
- [`facebook-get_all_posts`](facebook-get_all_posts.md) — Get all posts

### `multi_business_facebook.py` — Multi-Business Manager

**Class: BusinessConfig**
- [`multibiz-BusinessConfig`](multibiz-BusinessConfig.md) — Business configuration class

**Class: MultiBusinessFacebookManager**
- [`multibiz-manager_init`](multibiz-manager_init.md) — Initialize manager & directories
- [`multibiz-load_all_businesses`](multibiz-load_all_businesses.md) — Load configs from disk
- [`multibiz-get_business`](multibiz-get_business.md) — Get business by ID
- [`multibiz-get_business_by_page_id`](multibiz-get_business_by_page_id.md) — Find business by Page ID
- [`multibiz-add_business`](multibiz-add_business.md) — Add new business
- [`multibiz-publish_post`](multibiz-publish_post.md) — Publish post for a business
- [`multibiz-get_comments`](multibiz-get_comments.md) — Get comments for a business
- [`multibiz-reply_to_comment`](multibiz-reply_to_comment.md) — Reply as a business
- [`multibiz-get_all_posts`](multibiz-get_all_posts.md) — Get all posts for a business
- [`multibiz-save_business_memory`](multibiz-save_business_memory.md) — Save to business memory
- [`multibiz-get_business_memory`](multibiz-get_business_memory.md) — Retrieve business memory
- [`multibiz-learn_from_interaction`](multibiz-learn_from_interaction.md) — Learn from interactions
- [`multibiz-generate_reply`](multibiz-generate_reply.md) — Generate contextual reply
- [`multibiz-detect_language`](multibiz-detect_language.md) — Detect Arabic/English
- [`multibiz-generate_price_reply`](multibiz-generate_price_reply.md) — Price query reply
- [`multibiz-generate_hours_reply`](multibiz-generate_hours_reply.md) — Hours query reply
- [`multibiz-generate_location_reply`](multibiz-generate_location_reply.md) — Location query reply
- [`multibiz-generate_services_reply`](multibiz-generate_services_reply.md) — Services query reply
- [`multibiz-list_businesses`](multibiz-list_businesses.md) — List all businesses
- [`multibiz-convenience_functions`](multibiz-convenience_functions.md) — Module-level convenience APIs

### `facebook_webhook.py` — Basic Webhook
- [`webhook-verify_webhook`](webhook-verify_webhook.md) — Meta verification (GET)
- [`webhook-webhook_handler`](webhook-webhook_handler.md) — Receive events (POST)
- [`webhook-handle_comment_event`](webhook-handle_comment_event.md) — Process comment event
- [`webhook-handle_feed_event`](webhook-handle_feed_event.md) — Process feed event

### `facebook_webhook_gunicorn.py` — Production Webhook
- [`webhook-production-health_check`](webhook-production-health_check.md) — Health check endpoint
- [`webhook-production-verify_webhook`](webhook-production-verify_webhook.md) — Meta verification (prod)
- [`webhook-production-webhook_handler`](webhook-production-webhook_handler.md) — Event receiver (prod)
- [`webhook-production-handle_comment_event`](webhook-production-handle_comment_event.md) — Comment handler (prod)
- [`webhook-production-handle_feed_event`](webhook-production-handle_feed_event.md) — Feed handler (prod)
- [`webhook-production-forward_to_hermes`](webhook-production-forward_to_hermes.md) — Forward to Hermes API
- [`webhook-production-queue_for_retry`](webhook-production-queue_for_retry.md) — Retry queue for failures
- [`webhook-production-log_event`](webhook-production-log_event.md) — Event logging to file

### `multi_business_webhook.py` — Multi-Business Webhook
- [`multiwebhook-health_check`](multiwebhook-health_check.md) — Health check with business count
- [`multiwebhook-list_businesses`](multiwebhook-list_businesses.md) — List businesses endpoint
- [`multiwebhook-get_business_details`](multiwebhook-get_business_details.md) — Business details endpoint
- [`multiwebhook-verify_webhook`](multiwebhook-verify_webhook.md) — Meta verification (multi)
- [`multiwebhook-webhook_handler`](multiwebhook-webhook_handler.md) — Event receiver (multi)
- [`multiwebhook-handle_comment_event`](multiwebhook-handle_comment_event.md) — Auto-reply comment handler
- [`multiwebhook-handle_feed_event`](multiwebhook-handle_feed_event.md) — Feed handler (multi)
- [`multiwebhook-forward_to_hermes`](multiwebhook-forward_to_hermes.md) — Forward to Hermes (multi)

### `rate_limiter.py` — Rate Limiter (NEW)
- [`rate_limiter-overview`](rate_limiter-overview.md) — Token bucket rate limiter class & integration

### `add_business.py` — Business Setup Tool
- [`add_business-interactive`](add_business-interactive.md) — Interactive business wizard
- [`add_business-main`](add_business-main.md) — CLI entry point

## External Guides (from VPS)
- [`WEBHOOK_GUIDE`](WEBHOOK_GUIDE.md) — Webhook setup guide
- [`PRODUCTION_SETUP`](PRODUCTION_SETUP.md) — Production deployment guide
- [`MULTI_BUSINESS_GUIDE`](MULTI_BUSINESS_GUIDE.md) — Multi-business usage guide

## Architecture Overview

```
                    ┌──────────────────────┐
                    │   Meta Graph API     │
                    │  (Facebook servers)  │
                    └──────┬───────────────┘
                           │ Webhook Events
                           ▼
              ┌──────────────────────────┐
              │ multi_business_webhook.py│ (Flask, port 5000)
              │  POST /webhook           │
              │  GET  /health            │
              │  GET  /webhook           │
              └──────┬───────────────────┘
                     │ auto-reply + route by page_id
                     ▼
         ┌──────────────────────────┐
         │ multi_business_facebook  │
         │ .py (Business Manager)   │
         └──┬───────┬───────┬───────┘
            │       │       │
            ▼       ▼       ▼
      Business  Business  Business
      Config 1  Config 2  Config N
   (~/.hermes/businesses/*.json)

         ┌──────────────────────────┐
         │  Hermes API              │
         │  (AI processing)         │
         └──────────────────────────┘
```
