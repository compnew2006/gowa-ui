# PLAN.md

## Immediate Next Steps

- Configure `VITE_PUBLIC_MARKETING_BASE_URL` for the real sidecar destination in each environment.
- Decide whether the sidecar will keep posting leads into `/api/public/lead-requests` or own lead capture end-to-end later.
- Add a dedicated public-route E2E harness that can run without the current auth-heavy Playwright global setup.
