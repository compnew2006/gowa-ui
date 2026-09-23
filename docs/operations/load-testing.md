# Read-only API load test

This runbook defines a repeatable k6 workload for the authenticated contact list and agent analytics APIs. Run it only against an isolated staging deployment with representative, synthetic data. Both requests use `GET`; the script does not create contacts, send messages, change assignments, or modify campaigns.

## Workload and targets

The primary script, `loadtests/contacts_read.js`, exercises:

- `GET /api/contacts?page=1&limit=50&has_messages=true` on every iteration. The list is paginated at the same 50-row default used by the chat view.
- `GET /api/analytics/agents?from=...&to=...&group_by=day` on roughly one quarter of iterations, over the previous seven days. The handler returns an agent's own statistics for a user without organization-wide analytics permission, or the organization's analytics for a permitted user.

The default ramp holds 5 virtual users for 3 minutes, ramps to 40 over 2 minutes, holds 40 for 3 minutes, then ramps down over 1 minute. `K6_PEAK_VUS` changes the peak from 1 to 200; the warm-up level is capped at 5. Each virtual user reuses one staged access token, and a comma-separated token list distributes users across distinct test accounts.

Default acceptance targets are:

| Measure | Overall | Contacts | Agent analytics |
| --- | ---: | ---: | ---: |
| p50 latency | < 300 ms | < 300 ms | < 500 ms |
| p95 latency | < 1,500 ms | < 1,200 ms | < 2,000 ms |
| p99 latency | < 3,000 ms | < 2,500 ms | < 3,500 ms |

Overall failed HTTP requests must stay below 1%, successful checks above 99%, and average throughput above 5 requests/second. The k6 terminal summary prints p50/p95/p99, `http_req_failed`, and `http_reqs` count/rate. Thresholds fail the run when a target is missed.

These default targets are **provisional** because the project has no agreed production SLA. Latency thresholds are milliseconds. A `429` response counts as a failed request and check. Record the app version, dataset size, instance count, Postgres pool settings, and k6 version with every run; change target defaults only after agreeing on the service objective.

## Prerequisites and run

1. Confirm the target is a dedicated staging environment, has synthetic or sanitized data, and has no production WhatsApp provider connected. The script requires an HTTPS hostname containing `staging`, `stage`, `stg`, `test`, or `uat` and also requires `K6_TARGET_ENV=staging`.
2. Create one or more temporary, read-only staging users with access to contacts and analytics. Use short-lived access tokens. The analytics handler permits agents to view their own statistics; an admin analytics token may cause a broader query. Do not use production credentials or paste tokens into a command line, ticket, or shell history.
3. Install k6 using the method approved for the staging operator's workstation or CI runner.
4. Supply credentials through the process environment or a secret manager. Do not put tokens in the script or save them in this repository.

Example for a shell session where the values are already supplied securely:

```sh
export BASE_URL='https://staging.example.test'
export K6_TARGET_ENV='staging'
export K6_TOKENS="$STAGING_READONLY_TOKEN_A,$STAGING_READONLY_TOKEN_B"
k6 run loadtests/contacts_read.js
```

To raise concurrency in a planned capacity run, set `K6_PEAK_VUS` (for example, `K6_PEAK_VUS=60`). Avoid increasing it during an active run. Start with the default 40 VUs and review application, database, Redis, and load-generator health before raising the peak.

Thresholds can be tuned per run without editing the script. Latency values are milliseconds; error rate is a fraction from 0 to 1; throughput is requests per second.

| Environment variable | Default | Applies to |
| --- | ---: | --- |
| `K6_P50_MS`, `K6_P95_MS`, `K6_P99_MS` | `300`, `1500`, `3000` | All requests |
| `K6_CONTACTS_P50_MS`, `K6_CONTACTS_P95_MS`, `K6_CONTACTS_P99_MS` | `300`, `1200`, `2500` | Contacts endpoint |
| `K6_ANALYTICS_P50_MS`, `K6_ANALYTICS_P95_MS`, `K6_ANALYTICS_P99_MS` | `500`, `2000`, `3500` | Agent analytics endpoint |
| `K6_MAX_ERROR_RATE` | `0.01` | Failed HTTP request ratio |
| `K6_MIN_RPS` | `5` | Average request throughput |

The core workload accepts `K6_TOKEN` for one test account or `K6_TOKENS` for comma-separated access tokens belonging to distinct staged accounts. Tokens are assigned consistently by virtual user and are never printed by the script.

The API middleware applies a user-aware global rate limit to `/api` routes. The shipped default is 200 API requests per user per 60 seconds. Since k6 sends one contact request per iteration plus occasional analytics, a single access token can hit that limit well before the server is saturated. Use a small set of distinct, temporary staging accounts via `K6_TOKENS`, or have the staging operator set an intentional test-only limit before the run. Record any staging limit change and restore it afterward. Do not interpret `429`s as application capacity results.

### Optional single login/refresh check

`loadtests/auth_smoke.js` performs exactly one login followed by one refresh on the same virtual user. It is a low-volume functional and latency check, not an auth capacity test. Login sets a refresh cookie; refresh consumes that single-use token and issues its replacement. This writes short-lived token state to Redis. Do not run it concurrently or convert it into a loop against a shared account.

Use a dedicated staging account and supply its credentials through the environment or a secret manager. The script requires the same staging target guards as the read-only workload:

```sh
export BASE_URL='https://staging.example.test'
export K6_TARGET_ENV='staging'
k6 run loadtests/auth_smoke.js
```

Set `K6_AUTH_EMAIL` and `K6_AUTH_PASSWORD` securely in the process environment before running. When auth rate limiting is enabled, the configured defaults are 10 login attempts and 30 refresh attempts per 60 seconds, keyed by client IP. The smoke script makes only one of each. Avoid repeated runs during an active window, especially with reverse proxies that cause multiple staging operators to share one client IP.

## Database connection observation

Record the configured `database.max_open_conns` per application instance (default 25), replica count, and Postgres `max_connections` before starting. App pool capacity is approximately `max_open_conns × app replicas`, plus other services and administrative sessions; this is a configuration ceiling, not the number of connections currently in use.

From a read-only Postgres session connected to the staging database, sample sessions during a baseline period and throughout the load ramp:

```sql
SELECT clock_timestamp() AS sampled_at,
       state,
       count(*) AS connections
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state
ORDER BY state;
```

To compare the observed sessions with the server ceiling:

```sql
SELECT current_setting('max_connections')::integer AS max_connections,
       count(*) AS database_sessions
FROM pg_stat_activity
WHERE datname = current_database();
```

In `psql`, append `\watch 5` after either query to sample every five seconds, or use the staging monitoring system to capture the same measure. If staging sets a Postgres `application_name`, filter on that value to isolate the app; do not assume every session in the database belongs to this service. Save the timestamped samples with the k6 summary so connection growth and pool saturation can be compared to latency and error spikes.

## Stop conditions and report

Stop the run if staging begins sending real WhatsApp traffic, error rates rise sharply, the database nears its connection ceiling, or the environment shows resource pressure that risks other staging users. After a run, capture k6's threshold result and throughput alongside the database samples, app/DB replica counts, test token count, test data size, and any rate-limit changes. Revoke temporary access tokens when the run is complete.
