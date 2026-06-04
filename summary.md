# Port: `GET /api/facebook/page-search` (kingmaster → whatomate)

## What was ported

Mirrored kingmaster's `Dashboard/kingmaster/root/api/serchpag.php` (the API called by
`Dashboard/kingmaster/root/serch_fb_page.php`) as a native whatomate handler.

### Source vs. target

| kingmaster (MySQL / PHP)                        | whatomate (PostgreSQL 17 / Go)                              |
| ----------------------------------------------- | ----------------------------------------------------------- |
| `Dashboard/kingmaster/root/serch_fb_page.php`   | n/a (UI shell only — delegates to the API below)            |
| `Dashboard/kingmaster/root/api/serchpag.php`    | `internal/handlers/fb_page_search.go` → `SearchFBPages`     |
| `GET api/serchpag.php?campaign_id=…&q=…&page=…&per_page=…` | `GET /api/facebook/page-search?campaign_id=…&q=…&page=…&per_page=…` |
| Table `fb_serch` (MySQL)                        | Table `fb_page_searches` (Postgres, GORM-migrated)          |
| Global table (no `org_id` column)               | Tenant-scoped via `organization_id uuid NOT NULL`           |

## Behavior parity

- Required query param: `campaign_id`.
- Optional `q` filter: case-insensitive substring match against `name` OR `page_id`
  (Postgres `ILIKE`, equivalent to MySQL `LIKE` under `utf8mb4_unicode_ci`).
- Pagination: `per_page` (default 25, clamped 1–200), `page` (default 1, clamped ≥1).
- Order: `created_at DESC, id DESC` — newest first. (PHP used `id DESC` on a
  bigint auto-increment; the Go port uses `created_at DESC` first because
  whatomate's `BaseModel.ID` is a `uuid.UUID` and UUID lexical sort does not
  correspond to insertion order.)
- Response shape (matches PHP `api/serchpag.php` exactly):
  ```json
  {
    "status": "success",
    "data": {
      "success": true,
      "campaign_id": "...",
      "page": 1,
      "per_page": 25,
      "total": 0,
      "total_pages": 0,
      "data": [
        { "name": "...", "page_id": "...", "followers_count": "..." }
      ]
    }
  }
  ```

## Schema fix (carried over from kingmaster bug)

The original PHP `serchpag.php` queries `name LIKE :q1 OR phone LIKE :q2`, but the
kingmaster `fb_serch` schema has **no `phone` column** — only `page_id`. That is a
typo in the PHP that silently makes the search never match on the phone field.

The Go port fixes this by searching on `page_id` (the real column). If a future
port needs to recover the original `phone` semantics, that field needs to be
added to the table first.

## Files added

| File                                            | Purpose                                             |
| ----------------------------------------------- | --------------------------------------------------- |
| `internal/models/fb_page_search.go`             | `FBPageSearch` GORM model (`TableName() = "fb_page_searches"`) |
| `internal/handlers/fb_page_search.go`           | `SearchFBPages` handler + `parseIntQuery` helper    |
| `internal/handlers/fb_page_search_test.go`      | 9 test cases (see below)                            |

## Files modified

| File                                  | Change                                                                  |
| ------------------------------------- | ----------------------------------------------------------------------- |
| `internal/database/postgres.go`       | Added `{"FBPageSearch", &models.FBPageSearch{}}` to the AutoMigrate list |
| `test/testutil/db.go`                 | Added `&models.FBPageSearch{}` to the test migration list               |
| `internal/config/config.go`           | Added `Facebook FacebookConfig` field + `FacebookConfig{AccessToken, APIVersion, BaseURL}` struct |
| `config.example.toml`                 | Added `[facebook]` block after `[facebook_oauth]`                       |
| `cmd/whatomate/main.go`               | Registered `g.GET("/api/facebook/page-search", app.SearchFBPages)` after the `comments` route group |

## Auth & RBAC

- Route is mounted inside the existing JWT-gated group in `cmd/whatomate/main.go`
  (where the other `fb_*` routes live), so it requires a valid JWT.
- Per-request: `getOrgAndUserID(r)` + `requirePermission(r, models.ResourceAccounts, models.ActionRead)`
  — same RBAC pattern as the other Facebook handlers.

## Configuration (new `[facebook]` block)

```toml
[facebook]
# Per-organization Facebook Graph API credentials (used for future write endpoints).
# Currently unused by /api/facebook/page-search but reserved.
access_token = ""
api_version = "v20.0"
base_url = "https://graph.facebook.com"
```

## Verification

- `go build ./internal/... ./pkg/... ./cmd/whatomate/...` → clean.
- `go vet ./internal/handlers/... ./internal/models/... ./internal/config/... ./internal/database/... ./pkg/...` → clean (one pre-existing `unreachable code` in `test/testutil/db.go:144`, unrelated to this change).
- `go test -v -run "SearchFBPages" -count=1 ./internal/handlers/` with `TEST_DATABASE_URL`/`TEST_REDIS_URL` set → **9/9 pass**:
  - `RequiresCampaignID` — missing `campaign_id` returns 400.
  - `RequiresAuth` — no JWT returns 401.
  - `RejectsForbiddenUser` — user without `accounts:read` returns 403.
  - `EmptyResult` — unknown campaign returns 200 with empty `data[]`.
  - `ReturnsAllRowsForCampaign` — 3 rows for the same campaign returned newest-first.
  - `QueryFilterMatchesName` — `q` substring matches `name`.
  - `QueryFilterMatchesPageID` — `q` substring matches `page_id`.
  - `Pagination` — `per_page=3, page=2` returns correct slice + `total_pages=3`.
  - `TenantIsolation` — org A cannot see org B's rows for the same `campaign_id`.

## Known limitations / out of scope

- No frontend route added — this is API-only.
- `facebook.access_token` is accepted in config but unused by this endpoint
  (reserved for future Graph API call sites).
- Followers count is stored as `longtext` (string) to mirror the MySQL schema
  exactly. If large numbers are expected, a `bigint` migration is a follow-up.
- golangci-lint in this environment is built against Go 1.24 and rejects the
  Go 1.25.9 module; lint was not run end-to-end. Manual review + `go vet` were
  used instead.

## 2026-06-04 - Facebook Comments Sidebar Scroll Fix

### Task

Fix bug: at `/facebook/comments`, the Inbox (left) column of the 3-column page had no scrollbar, so users couldn't reach comments past the visible viewport. Same root cause would affect the Thread (center) and Overview (right) columns once their content exceeded viewport height.

### Root cause

The grid wrapper at `FacebookCommentsView.vue:413-415` was:
```
class="grid min-h-0 flex-1 gap-4 p-4 lg:grid-cols-[360px_minmax(0,1fr)_320px]"
```

CSS Grid's default `grid-auto-rows: auto` sizes the single implicit row to its content. Even with `min-h-0 flex-1` on the grid (which constrains the grid container via the parent flex chain), the row itself was not constrained — it grew past the grid container's available height. As a result:
- the row's height = max content height across the 3 cells
- each Card is the row's full height (default `align-self: stretch`)
- the ScrollArea inside the Inbox Card has `flex-1 min-h-0` of the CardContent (also `flex h-full min-h-0 flex-col`) — correct chain — but the Card itself was as tall as the content (because the row was unbounded), so there was no overflow for ScrollArea to scroll

### Fix

One line CSS class added to the grid:
```
lg:grid-rows-[minmax(0,1fr)]
```

`minmax(0, 1fr)` clamps the row between 0 and `1fr` of the grid container. The `min(0)` is critical — without it, the row's intrinsic min-size would still be the content height and the constraint would have no effect. The `lg:` prefix scopes the rule to the desktop 3-column layout; on smaller screens (default 1-column) auto rows remain correct.

The internal flex chain (Card → CardContent flex-col → ScrollArea flex-1) was already correct — it just had nothing to scroll because the parent was unbounded.

### Code changes

- `frontend/src/views/facebook/FacebookCommentsView.vue` — added `lg:grid-rows-[minmax(0,1fr)]` to the grid wrapper at L414. Single class addition, no other changes.

### Verification

- `cd frontend && npm run typecheck` → clean
- `cd frontend && npm run build` → success, `index-BM73qrpE.css` contains `lg\:grid-rows-\[minmax\(0\,1fr\)\]{grid-template-rows:minmax(0,1fr)}`
- Built Go binary with embedded dist:
  - `cp -r frontend/dist/* internal/frontend/dist/` (the Makefile's `build-prod` target does this; doing `go build` directly skips it and ships the OLD frontend)
  - `env -u GOOS -u GOARCH GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=comments-scroll-fix-20260604_013200-3f31242c -X main.BuildTime=… -X …/license.EmbeddedPublicKeyRingBase64=…"`
  - Result: ELF 64-bit, statically linked, 58966178 bytes
  - SHA256: `6def64dfb72ec38879a862fe1f206732cc9684b3ba961173d4dad475ac4e7d6f`
- Deployed: scp → /opt/whatomate/bin/whatomate.sandbox.comments-scroll-fix-20260604_013200-3f31242c → repoint .active + .green → `systemctl restart whatomate-sandbox.service`
- Post-restart: PID 2268910, active since 2026-06-04 01:36:35 UTC, `curl https://sandbox.ofuqalmadenah.com/` → 200, license bootstrap → status=active tier=production
- Deployed CSS bundle served at `/assets/index-BM73qrpE.css` contains the new rule

### Lessons

- **CSS Grid auto rows are NOT constrained by `min-h-0` on the container.** You must explicitly set `grid-template-rows: minmax(0, 1fr)` to allow grid cells to shrink below their content. This is the same gotcha as flexbox's `min-height: auto` default on items.
- **Build pipeline ordering matters.** The Makefile has `build-prod` → `frontend-build` → `embed-frontend` → `go build`. Skipping the `embed-frontend` step ships a binary that still has the previous frontend in `internal/frontend/dist/`. Symptom is subtle: the Go binary reports a new version, the API responds correctly, but the SPA's CSS/JS bundles are old hashes.
- **When deploying only frontend changes**, you still need the full Go build cycle (and the license ldflag) — `//go:embed` requires recompilation.

### Rollback

`whatomate-sandbox-switch blue` → restores `whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c` (the previous green, which has the FB admin filter from the prior deploy).
