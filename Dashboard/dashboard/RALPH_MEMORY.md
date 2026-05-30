# Ralph Memory (Persistent Learning)

## [2026-05-23] Issue: Empty Dashboard and "Failed to load posts" error

- **The Trap:** Assuming `Base.metadata.create_all` would automatically resolve any schema discrepancies on pre-existing database tables, and assuming only starting the frontend dev server was enough.
- **The Reality:** Pre-existing PostgreSQL database tables bypassed SQLAlchemy's automatic creation, leaving newly added fields (e.g. `customers.whatsapp_id`, `settings.whatsapp_notification_phone`) missing. This led to SQL `ProgrammingError` (`UndefinedColumnError`) whenever the backend attempted queries, crashing the API routes and leaving the dashboard completely empty.
- **The Fix:** 
  1. Updated the backend startup script `start.sh` to work locally and default to port `8000` (which is proxied by Next.js).
  2. Surgically altered PostgreSQL database tables to add missing columns (`whatsapp_id` on `customers`, `whatsapp_notification_phone`, `whatsapp_notification_api_key`, and `enable_private_replies` on `settings`).
  3. Created and executed a comprehensive database seeding script (`seed_db.py`) to fully populate all tables with realistic Arabic mock data for a rich, beautiful visual experience.
  4. Started the backend FastAPI server to run concurrently with the Next.js dashboard.
- **The Law:** Always verify database schema completeness when relying on dynamic schema creation over pre-existing tables, and ensure both backend and frontend servers are properly running and seeded with data.

## [2026-05-23] Issue: React child object crash on `/audit-logs` page

- **The Trap:** Direct rendering of a database JSON field (the `details` object) as a React JSX child `{log.details}` assuming it would either be implicitly handled or always string-like.
- **The Reality:** Passing an object directly into React JSX as a child crashes the virtual DOM with: `Objects are not valid as a React child (found: object with keys {ip})`.
- **The Fix:** Used a robust, premium conditional formatter to check if `log.details` is an object, mapping its entries cleanly to a human-readable key-value string (e.g. `ip: 192.168.1.50`), and fallback gracefully to `log.reason` or `-`.
- **The Law:** Never render JSON or raw dictionary object values directly as JSX children; always sanitize and convert objects/dictionaries to string representations or components before rendering.

## [2026-05-23] Issue: Secure Production Authentication and Messaging Fallbacks

- **The Trap:** Designing an auth check on the frontend that causes flash-of-unstyled-content (FOUC), or using masked API settings variables for outbound background integrations which triggers auth failures.
- **The Reality:** 
  1. Frontend navigation checks in `useEffect` can briefly flash authenticated views if not guarded by isolated rendering of public routes and strict null-state loaders in `AppShell`.
  2. Outbound official Meta Cloud API services must query unmasked database rows directly (not through cached serializing functions) to prevent sending raw string stars `"****"` as bearer authorization tokens to Meta's endpoints.
- **The Fix:**
  1. Added a client-side route check and premium dark glassmorphic loading guard in `AppShell` (`app-shell.tsx`) to isolate public routes and strictly block unauthorized renders.
  2. Built `whatsapp_cloud_api.py` to fetch unmasked database row configurations directly from SQLAlchemy settings.
  3. Integrated graceful fallbacks in scheduled posts and message replies so that if official Meta credentials are missing or fail, it smoothly routes through the local bridge ports.
- **The Law:** Always isolate public/private routes at the highest shell container level with loading states to prevent DOM flashes, and bypass serialization layers when retrieving credentials for outbound service calls to prevent passing masked values.

