# Feature: Remove Public Pricing Pages And Prepare Sidecar Handoff

## Objective
Remove the public pricing surface from the main frontend in a way that lets a dedicated sidecar take ownership of marketing content, public lead capture UX, and future offer experiments without breaking existing lead operations.

## Selected Skills For This Planning Task
- `architecture-guardian`: used to constrain blast radius, define the migration seam, and avoid coupling the future sidecar directly into core app modules.
- `spec-miner`: used to reverse-engineer the current `/pricing`, `/plans`, and `/offer` behavior from actual router, frontend, and backend code.

## Required Competencies For Implementation
- Vue 3 + Vue Router route migration and public-page removal.
- Vite deploy/runtime behavior for SPA fallbacks and external redirects.
- Go `fastglue` handler changes for public lead ingestion and source validation.
- GORM/model awareness for preserving `lead_requests` data continuity.
- Reverse proxy or edge routing knowledge for handing `/pricing`, `/plans`, and `/offer` to a sidecar.
- Playwright and browser-devtools smoke testing for redirect and public-page verification.

## Current-State Findings
- Public route ownership is entirely inside the main frontend router:
  - `/pricing`
  - alias `/plans`
  - alias `/offer`
- All three routes render the same view: `frontend/src/views/public/PricingLandingView.vue`.
- The page is not purely static. It submits demo requests through `leadRequestsService.createPublic(...)` to `POST /api/public/lead-requests`.
- Lead requests are visible and managed in the authenticated admin UI at `/settings/lead-requests`.
- Backend validation currently hardcodes the marketing surface:
  - `source_page` must equal `pricing`
  - `source_route` must be one of `/pricing`, `/plans`, `/offer`
- There is no dedicated automated E2E coverage for the public pricing route or alias behavior today.

## Blast Radius Analysis
- Target: public pricing route group in `frontend/src/router/index.ts` and `frontend/src/views/public/PricingLandingView.vue`
- Directly affected:
  - `frontend/src/router/index.ts`
  - `frontend/src/views/public/PricingLandingView.vue`
  - `frontend/src/services/api.ts`
  - `internal/handlers/lead_requests.go`
  - `internal/models/lead_request.go`
- Indirectly affected (1 level):
  - `frontend/src/views/settings/LeadRequestsView.vue`
  - `frontend/src/components/layout/navigation.ts`
  - `cmd/whatomate/main.go`
  - `internal/handlers/lead_requests_test.go`
- Risk level: MEDIUM
- Safe to proceed: YES, if removal is done as a strangler migration with a redirect/proxy seam and lead ingestion contract preserved initially.

## Architectural Impact Assessment
- Areas affected: public frontend routing, public lead ingestion API contract, deployment/edge routing, internal lead-ops dashboard.
- New extension points:
  - configurable external marketing origin for public route handoff
  - config-driven allowed lead sources instead of hardcoded pricing-only values
  - optional redirect shell or edge redirect for legacy URLs
- Risks:
  - deleting the page without a redirect loses existing indexed/public entry URLs
  - deleting the page before preserving the lead API breaks lead capture
  - coupling the sidecar directly into the authenticated SPA keeps the monolith responsible for marketing concerns
- Proposed mitigations:
  - keep route ownership stable while changing implementation to redirect/proxy
  - keep `lead_requests` storage/admin inbox in the monolith for phase 1
  - generalize source validation before the sidecar posts real traffic
- Refactor needed? YES, but small and boundary-focused. This should be a route-handoff refactor, not a broad frontend rewrite.

## Skeptical Review
- The Plan: delete `PricingLandingView.vue`, delete the router entry, then build the sidecar later.
- The Critic: "This is fragile because `/pricing`, `/plans`, and `/offer` are live entrypoints, and the current public form writes into an existing lead workflow."
- The Defense/Fix: "Introduce a sidecar handoff seam first, keep lead ingestion compatible, and only then remove the page implementation from the monolith."

## Migration Requirements
- While the sidecar is not live, when a user opens `/pricing`, `/plans`, or `/offer`, the system shall keep the current public page behavior unchanged.
- While the sidecar is live, when a user opens `/pricing`, `/plans`, or `/offer`, the system shall route or redirect the request to the sidecar-owned marketing experience.
- While the sidecar still writes into the existing lead inbox, when a public lead is submitted, the system shall continue storing records in `lead_requests` without requiring admin UI changes.
- While legacy links remain in circulation, when `/plans` or `/offer` is requested, the system shall preserve equivalent behavior instead of returning an accidental SPA 404.
- While the migration is incomplete, when tests run, the system shall verify at least one public-route smoke path and one lead-ingestion path.

## Recommended Target Architecture
- Sidecar owns:
  - pricing content
  - offer copy and experiments
  - public lead form UX
  - public analytics/SEO concerns
- Main app keeps initially:
  - `POST /api/public/lead-requests`
  - `lead_requests` persistence and admin workflow
  - authenticated lead management UI under `/settings/lead-requests`
- Handoff seam:
  - public route redirect/proxy from the monolith or edge layer to the sidecar
  - config value such as `PUBLIC_MARKETING_BASE_URL`
  - source metadata widened from pricing-only to marketing-source aware

## Implementation Plan
1. Add a route-handoff strategy before deleting anything.
   - Choose one of:
     - edge/proxy ownership for `/pricing`, `/plans`, `/offer`
     - monolith router redirect view that forwards to the sidecar origin
   - Keep legacy URLs stable.
2. Generalize lead-source validation.
   - Replace hardcoded `source_page == "pricing"` assumptions with a small allowlist or config-backed source catalog.
   - Keep accepting `/pricing`, `/plans`, and `/offer` during transition.
   - Allow the sidecar to identify itself cleanly without forking the admin workflow.
3. Add coverage before removal.
   - Add one Playwright smoke spec for `/pricing`, `/plans`, and `/offer`.
   - Add backend tests covering accepted sidecar source metadata once the contract is widened.
   - Keep a manual Chrome DevTools smoke checklist for final redirect validation.
4. Remove the page implementation from the monolith.
   - Delete `PricingLandingView.vue` only after the handoff seam is in place.
   - Replace the router entry with redirect behavior or remove it entirely if the edge layer owns those paths.
5. Clean up monolith-only marketing artifacts.
   - Remove dead imports, copy, and route-specific types that are no longer needed.
   - Reassess whether `requested_plan` enums still belong in the monolith or should become generic lead-source metadata.
6. Decide the phase-2 ownership model.
   - Option A, recommended: sidecar keeps using the monolith lead API.
   - Option B: move lead ingestion fully into the sidecar/backend-for-marketing, then retire the monolith lead request API and admin page only after replacement exists.

## Recommended Execution Order
1. Introduce config for sidecar destination and document routing ownership.
2. Add smoke coverage for current public route behavior.
3. Generalize lead-source validation and tests.
4. Swap public routes to redirect/proxy behavior.
5. Remove `PricingLandingView.vue` and any dead route-local copy/types.
6. Re-run browser smoke on `/pricing`, `/plans`, `/offer`, and `/settings/lead-requests`.

## Verification Plan
- Build:
  - `npm run build` in `frontend/`
- Automated:
  - Playwright smoke for `/pricing`
  - Playwright smoke for `/plans`
  - Playwright smoke for `/offer`
  - backend tests for `internal/handlers/lead_requests.go`
- Manual:
  - Chrome DevTools: verify redirect status, final URL, console cleanliness, and visible page shell
  - authenticated smoke: verify `/settings/lead-requests` still loads and existing lead records remain usable

## Open Questions
- Should the sidecar post to the monolith API on the same origin, or through a dedicated public API hostname?
- Do you want legacy URLs redirected permanently (`301/308`) or temporarily (`302/307`) during rollout?
- Should `requested_plan` remain a first-class enum, or should the future sidecar send more generic campaign/source metadata?
- Will the sidecar live on the same domain path space or on a dedicated marketing subdomain?
