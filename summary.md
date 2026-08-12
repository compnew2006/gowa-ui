# gowa-ui → VPS deployment summary (separate service)

**Date:** 2026-08-12
**Target VPS:** 31.97.192.53 (root)
**Result:** ✅ gowa-ui deployed as a **separate** service. **whatomate (production) was NOT touched.**

> ⚠️ **Important correction to the original task framing:**
> The local working folder `~/Downloads/whatomate` is actually the **gowa-ui** project
> (`compnew2006/gowa-ui`, GOWA-provider based). The VPS production app `ofuqalmadenah.com`
> runs a **different** project, **whatomate** (`compnew2006/whatomate`, whatsmeow/Cloud-API based).
> Per your explicit choice ("خدمة منفصلة — آمن"), gowa-ui was deployed as a NEW, separate
> service. It did **not** replace whatomate (which would have broken the live site).

---

## What was deployed

| Item | Value |
|---|---|
| Project | gowa-ui (`compnew2006/gowa-ui`), commit `d6cb0922` |
| On VPS | `/opt/gowa-ui/` (binary `gowa-ui`, 32 MB, statically linked linux/amd64) |
| Service | `gowa-ui.service` (systemd), enabled + active |
| Port | **8081** (whatomate uses 18123 — untouched) |
| Database | **`gowa_ui`** (separate role + DB; whatomate uses `whatomate_prod` — untouched) |
| Redis | db index **2** (whatomate uses db 0 — untouched) |
| Config | `/opt/gowa-ui/config.toml` (chmod 600) |
| Frontend | embedded in the binary |
| Environment | `staging` (so auth cookies work over plain HTTP — see Notes) |

## What was NOT touched (production safety)

- `/opt/whatomate*`, `whatomate.service`, `whatomate@holol-wenjaz.service` — **untouched, still running** (verified HTTP 200 on `:18123`).
- `whatomate_prod` / `whatomate_holol_wenjaz` databases — **untouched**.
- GOWA servers `/opt/gowa-main`, `/opt/gowa-holol` (`gowa-main.service`, `gowa-holol.service`) — **untouched**.
- No destructive operations were performed → **no backup was required/created** (nothing existing was modified or deleted).

## Bugs found & fixed during deploy (in gowa-ui code)

1. **`provider_type` column reference** — `BackfillGowaWebhookSecrets` queried a `provider_type`
   column that no longer exists on the `WhatsAppAccount` model (worked locally only because of a
   stale column). Crashed against a fresh DB. Fixed: filter by `gowa_device_id <> ''` instead.
   → `internal/handlers/gowa_backfill.go`
2. **Secure cookies over HTTP** — `environment = "production"` auto-enables `secure` cookies,
   which browsers reject on plain HTTP → login didn't persist. Set `environment = "staging"` for
   this HTTP test instance (non-secure cookies → login works).

## Access

- **URL:** http://31.97.192.53:8081
- **Admin login:** `admin@gowa-ui.local`
- **Admin password:** stored on the VPS at `/opt/gowa-ui/.deploy-secrets` (chmod 600). Retrieve with:
  ```bash
  ssh root@31.97.192.53 'cat /opt/gowa-ui/.deploy-secrets'
  ```

## Manage the service

```bash
# on the VPS:
systemctl status gowa-ui        # health
systemctl restart gowa-ui       # restart (re-runs migrations due to -migrate flag)
journalctl -u gowa-ui -f        # live logs
systemctl stop gowa-ui          # stop
systemctl disable gowa-ui       # stop + disable auto-start
```

## Verification results

| Check | Result |
|---|---|
| `curl :8081/` | HTTP 200 |
| `curl :8081/login` | HTTP 200 (UI renders) |
| External reachability (Mac → http://31.97.192.53:8081) | HTTP 200 |
| `gowa_ui` DB tables | 30 tables, all owned by `gowa_ui`, incl. `gowa_webhook_events` (durable inbox), `gowa_instances` |
| `POST /api/auth/login` | HTTP 200 (credentials valid, DB-backed auth works) |
| `GET /api/me` (with cookie) | HTTP 200 (session persists) |
| Browser login → dashboard | ✅ renders (sidebar: Dashboard / Chat / Campaigns / Agent Analytics / Settings) |
| Background processors | all running, incl. **GOWA webhook inbox processor** (5s poll + wake) |
| whatomate production (`:18123`) | **HTTP 200, active 6+ days — untouched** |

## Next steps (not done — ask if you want them)

1. **HTTPS / real domain**: put gowa-ui behind nginx with TLS (then switch `environment = "production"`
   so secure cookies work). Until then it's HTTP on a bare IP.
2. **Configure the GOWA instance** inside gowa-ui (Settings → GOWA Servers): add the GOWA server
   (`gowa-main`, reachable at `gowa.ofuqalmadenah.com` or the VPS internal address) with its Basic
   Auth creds. gowa-ui is GOWA-provider based and needs this to send/receive.
3. **Webhook URL**: for real-time chat in gowa-ui, the GOWA server must be able to reach this
   instance's webhook (`http://31.97.192.53:8081/api/gowa/webhook/...` or the HTTPS domain).

## How it was built (reproduce)

```bash
cd ~/Downloads/whatomate
(cd frontend && npm run build)
rm -rf internal/frontend/dist/* && cp -r frontend/dist/* internal/frontend/dist/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags "-s -w -X main.Version=$(git describe --tags --always) -X main.BuildTime=$(date -u +%Y-%m-%d_%H:%M:%S)" \
  -o gowa-ui-linux ./cmd/gowa-ui
```
