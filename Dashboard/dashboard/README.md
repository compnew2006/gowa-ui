# AI Automation Dashboard

Next.js dashboard for AI-assisted social comment automation, review queues, campaigns, customers, compliance, and agency branding.

## Environment

- `API_URL`: backend origin used by the server-side `/api/:path*` proxy. Required in production; use HTTPS unless it targets loopback, e.g. `http://127.0.0.1:8000`. Defaults to `http://localhost:8000` only outside production.
- `NEXT_PUBLIC_SENTRY_DSN`: optional browser Sentry DSN.
- `NEXT_PUBLIC_SENTRY_REPLAY`: set to `true` only when session replay is explicitly approved. Defaults off.
- `ENABLE_REGISTRATION` / `NEXT_PUBLIC_ENABLE_REGISTRATION`: set to `true` only for controlled invite/bootstrap environments. Defaults off. The server route rejects registration when disabled.
- `SENTRY_DSN`: optional server Sentry DSN.
- `PORT`: optional local Next port. Defaults to `3000` in npm scripts.

The backend API is not implemented in this folder. When an endpoint is unavailable, the frontend should show a clear unavailable/error state rather than simulating success.

## Scripts

- `npm run dev`: start the local Next dev server.
- `npm run build`: production build.
- `npm run lint`: non-interactive ESLint.
- `npm run typecheck`: TypeScript verification.
- `npm run test`: Vitest unit/component tests.
- `npm run test:e2e:chromium`: Playwright smoke tests in Chromium.
- `npm run test:e2e`: full Playwright suite.

## Quality Gates

Before review, run:

```bash
npm run lint
npm run typecheck
npm run test
npm run test:e2e:chromium
npm run build
```

## Notes

- The browser command-console route was removed because command execution must not be exposed through the dashboard.
- Public registration is disabled by default, and the frontend never sends a requested admin role during registration.
- Browser sessions no longer persist bearer tokens in Web Storage. Login stores the backend bearer token in an HttpOnly `__Host-` cookie, and the server-side API proxy attaches it to upstream requests.
- Production responses include nonce-based CSP and standard hardening headers.
- Frontend upload and display URL validation is defense-in-depth only. The backend must still enforce authentication, RBAC, MIME/content sniffing, file size, malware scanning, storage isolation, and rate limits.
- The app shell is RTL-first and responsive: desktop uses a right-side sidebar, mobile uses a drawer.
- Frontend contracts exist for post content generation, campaign recipient preview, and branding logo upload; backend support may return 404/501 until implemented.
