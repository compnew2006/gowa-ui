# PLAN.md

## Immediate Next Steps

- **Completed Feature 1**: Implemented, tested, and verified **Phone Number Validation & WhatsApp Registration Check (wa_filter)** feature end-to-end with 100% clean frontend typescript builds and backend handler/worker test suites passing.
- **Completed Feature 2**: Implemented, verified, and unit-tested the generic **SelectableDataTable component** and **useSelectableTable composable** system. Refactored `WhatsAppFilter.vue` to leverage this new architecture, resulting in robust row/page/matching selections and debounced keyword filtering. Passed all 4 unit tests and completed a 100% green frontend typecheck.
- **Completed E2E Test Suite**: Developed, verified, and executed a comprehensive Playwright E2E test suite for the selectable results table (`e2e/tests/settings/whatsapp-filter.spec.ts`), which runs with 100% success using fully-mocked routing endpoints even when offline.
- Configure `VITE_PUBLIC_MARKETING_BASE_URL` for the real sidecar destination in each environment.
- Decide whether the sidecar will keep posting leads into `/api/public/lead-requests` or own lead capture end-to-end later.
- Add a dedicated public-route E2E harness that can run without the current auth-heavy Playwright global setup.
