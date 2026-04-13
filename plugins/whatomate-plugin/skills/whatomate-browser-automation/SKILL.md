---
name: whatomate-browser-automation
description: Use when inspecting, debugging, or validating Whatomate in a real browser. Prefer Chrome DevTools for interactive debugging and Playwright for repeatable flows.
---

# Whatomate Browser Automation

Use this skill for browser work against Whatomate.

## Tool choice

- Use Chrome DevTools when you need interactive inspection:
  - check console errors
  - inspect network requests
  - verify DOM state after user actions
  - capture screenshots during manual debugging
- Use Playwright when you need a repeatable flow:
  - open the app and navigate predictable steps
  - fill forms, click buttons, and capture screenshots
  - repeat login, activation, and settings flows

## Defaults

- Frontend URL: `${WHATOMATE_FRONTEND_URL:-http://127.0.0.1:3000}`
- Backend URL: `${WHATOMATE_BACKEND_URL:-http://127.0.0.1:8080}`

## Playwright notes

- The Playwright wrapper lives at `$CODEX_HOME/skills/playwright/scripts/playwright_cli.sh`.
- `npx` must be installed before using the wrapper.
- Re-snapshot after navigation or major DOM changes.

## Suggested workflow

1. Make sure the Whatomate frontend is running.
2. Use Chrome DevTools for exploratory debugging.
3. Use `./scripts/open-local-with-playwright.sh` for a repeatable browser entry point.
4. Save browser evidence when it helps explain the result.
