# Whatomate Dedicated Instances (3)

Generated: 2026-02-25 23:25:52 UTC
Server IP: 31.97.192.53
Base Domain Pattern (suggested): <tenant>.ofuqalmadenah.com

## Summary

- Dedicated instance = separate systemd service, PostgreSQL DB/user, uploads path, config, internal port
- Shared binary: /opt/whatomate/bin/whatomate
- SSL is enabled for all 3 subdomains (Let's Encrypt via Certbot)
- Nginx HTTPS vhosts are active and Certbot auto-renew is configured

## حلول وانجاز

- Tenant slug: holol-wenjaz
- Suggested domain: holol-wenjaz.ofuqalmadenah.com
- Systemd service: whatomate@holol-wenjaz
- Internal port: 127.0.0.1:18124
- Redis DB index: 1
- Instance dir: /opt/whatomate/instances/holol-wenjaz
- Uploads dir: /opt/whatomate/instances/holol-wenjaz/uploads
- Config: /opt/whatomate/instances/holol-wenjaz/config.toml
- PostgreSQL DB: whatomate_holol_wenjaz
- PostgreSQL User: whatomate_holol_wenjaz
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+holol-wenjaz@whatomate.local
- Default Admin Password: [REDACTED]
- Nginx vhost: /etc/nginx/sites-available/whatomate-holol-wenjaz.conf
- Certbot command used: certbot --nginx -d holol-wenjaz.ofuqalmadenah.com

## الاركان المثالية

- Tenant slug: alarkan-almthalia
- Suggested domain: alarkan-almthalia.ofuqalmadenah.com
- Systemd service: whatomate@alarkan-almthalia
- Internal port: 127.0.0.1:18125
- Redis DB index: 2
- Instance dir: /opt/whatomate/instances/alarkan-almthalia
- Uploads dir: /opt/whatomate/instances/alarkan-almthalia/uploads
- Config: /opt/whatomate/instances/alarkan-almthalia/config.toml
- PostgreSQL DB: whatomate_alarkan_almthalia
- PostgreSQL User: whatomate_alarkan_almthalia
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+alarkan-almthalia@whatomate.local
- Default Admin Password: [REDACTED]
- Nginx vhost: /etc/nginx/sites-available/whatomate-alarkan-almthalia.conf
- Certbot command used: certbot --nginx -d alarkan-almthalia.ofuqalmadenah.com

## مطبعة رؤية

- Tenant slug: matbaat-ruya
- Suggested domain: matbaat-ruya.ofuqalmadenah.com
- Systemd service: whatomate@matbaat-ruya
- Internal port: 127.0.0.1:18126
- Redis DB index: 3
- Instance dir: /opt/whatomate/instances/matbaat-ruya
- Uploads dir: /opt/whatomate/instances/matbaat-ruya/uploads
- Config: /opt/whatomate/instances/matbaat-ruya/config.toml
- PostgreSQL DB: whatomate_matbaat_ruya
- PostgreSQL User: whatomate_matbaat_ruya
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+matbaat-ruya@whatomate.local
- Default Admin Password: [REDACTED]
- Nginx vhost: /etc/nginx/sites-available/whatomate-matbaat-ruya.conf
- Certbot command used: certbot --nginx -d matbaat-ruya.ofuqalmadenah.com

## Verification

### whatomate@holol-wenjaz

- enabled: enabled
- active: active

### whatomate@alarkan-almthalia

- enabled: enabled
- active: active

### whatomate@matbaat-ruya

- enabled: enabled
- active: active

### Local Port Listeners

State Recv-Q Send-Q Local Address:Port Peer Address:PortProcess  
LISTEN 0 4096 127.0.0.1:18124 0.0.0.0:_ users:(("whatomate",pid=1604127,fd=10))  
LISTEN 0 4096 127.0.0.1:18125 0.0.0.0:_ users:(("whatomate",pid=1604191,fd=8))  
LISTEN 0 4096 127.0.0.1:18126 0.0.0.0:\* users:(("whatomate",pid=1604256,fd=8))

### Local HTTP Smoke (Host header)

- holol-wenjaz.ofuqalmadenah.com -> OK (Whatomate frontend served on :18124)
- alarkan-almthalia.ofuqalmadenah.com -> OK (Whatomate frontend served on :18125)
- matbaat-ruya.ofuqalmadenah.com -> OK (Whatomate frontend served on :18126)

## SSL Enablement Update

Updated: 2026-02-25 23:33:58 UTC

- All 3 subdomains now have valid HTTPS certificates issued by Let's Encrypt
- Certbot deployed the certificates directly into the Nginx tenant vhosts
- Auto-renew is enabled by Certbot's scheduled task

### حلول وانجاز

- Live URL: https://holol-wenjaz.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18124
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/holol-wenjaz.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/holol-wenjaz.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:31:43 2026 GMT

### الاركان المثالية

- Live URL: https://alarkan-almthalia.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18125
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/alarkan-almthalia.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/alarkan-almthalia.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:31:57 2026 GMT

### مطبعة رؤية

- Live URL: https://matbaat-ruya.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18126
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/matbaat-ruya.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/matbaat-ruya.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:32:13 2026 GMT

### Renewal Check

- Test renewal: certbot renew --dry-run

### Local HTTPS Smoke (SNI + local resolve)

- holol-wenjaz.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed
- alarkan-almthalia.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed
- matbaat-ruya.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed

### How to apply code changes to all instances

Update source in /opt/whatomate-src (git pull / rsync)
Build new binary on VPS
Replace /opt/whatomate/bin/whatomate
Restart all instance services
Commands (all instances)

cd /opt/whatomate-src
git pull # or rsync your updated code here

# Build (frontend + backend)

export PATH=/usr/local/go/bin:$PATH
make build-prod

# Install shared binary

install -m 755 /opt/whatomate-src/whatomate /opt/whatomate/bin/whatomate

# Restart all tenants

systemctl restart whatomate@holol-wenjaz
systemctl restart whatomate@alarkan-almthalia
systemctl restart whatomate@matbaat-ruya
Verify

systemctl status whatomate@holol-wenjaz --no-pager -l
systemctl status whatomate@alarkan-almthalia --no-pager -l
systemctl status whatomate@matbaat-ruya --no-pager -l
journalctl -u whatomate@holol-wenjaz -n 50 --no-pager
Safer pattern (recommended)

Restart one instance first (test)
If OK, restart the remaining two
systemctl restart whatomate@holol-wenjaz

# test website/login

systemctl restart whatomate@alarkan-almthalia whatomate@matbaat-ruya
Important note about DB migrations

Your service runs with -migrate, so on restart each instance may run migrations on its own database.
This is good for multi-tenant isolated DBs, but if a migration is risky, test on one tenant first.
Best next improvement

Create one script like deploy_whatomate_all.sh to automate:
sync/build/install
restart all instances
health checks
rollback if one fails

## Deployment Update

Updated: 2026-02-26 23:36:13 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via rsync; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `e0a23f5` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260226_233612.bak`
- Installed binary SHA256: `1fda6a038c26fbac983c9d9b904d22df6da7ac309e7a89b7aea7447d458873ad`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`

## Deployment Update

Updated: 2026-03-09 06:49:10 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_064910.bak`
- Installed binary SHA256: `57d6c12141abaed291898bf01e47ca69c17c5e2684097c5535deec120ca4c56a`
- Note: the local cross-compiled Linux binary was not used because it crashed on this VPS with `SIGSEGV`; the final deployment was built natively on the server and verified healthy.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- During restart, temporary `502` responses appeared while migrations/startup were in progress; final state is healthy.

## Deployment Update

Updated: 2026-02-27 20:43:46 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via rsync; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `e0a23f5` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260227_204257.bak`
- Installed binary SHA256: `bd4fd646c39552f35183136433dd6d74e7a3f4bf5683d41148acd5fc9b927370`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediately after restart, temporary `502` responses were observed while services were booting/migrating; final status is healthy.

## Deployment Update

Updated: 2026-02-27 21:50:56 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `cc8cbc8` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260227_214922.bak`
- Installed binary SHA256: `be5e506a104297b545692dff50ee25c9653d87745c7ad4d5ebebed295315fc1a`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were running migrations/boot; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:00:54 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_130054.bak`
- Installed binary SHA256: `a8fdcf89bc36ea137703b357fc23ff5cd7b6c77e0124768383254bc9f145d26f`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Initial checks during restart briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-02-28 19:42:52 UTC

- Deployed from local workspace: /Users/noiemany/Downloads/whatomate_GOWA/whatomate
- Source sync target: /opt/whatomate-src (via rsync; excluded caches/uploads/build artifacts)
- Source revision on deploy: 93f8b57 (local working tree had uncommitted changes)
- Build command: make build-prod
- Installed binary: /opt/whatomate/bin/whatomate
- Backup binary created: /opt/whatomate/bin/whatomate.20260228_194252.bak
- Installed binary SHA256: b9e356d8530538edade5202adb3425b2e67bd20482a8e57e4ad09cb8ee0d60db

### Services Restarted

- whatomate
- whatomate@holol-wenjaz
- whatomate@alarkan-almthalia
- whatomate@matbaat-ruya

### Post-Deploy Verification

- Listener ports active: 127.0.0.1:18123, 127.0.0.1:18124, 127.0.0.1:18125, 127.0.0.1:18126
- HTTPS smoke:
  - https://ofuqalmadenah.com -> 200
  - https://holol-wenjaz.ofuqalmadenah.com -> 200
  - https://alarkan-almthalia.ofuqalmadenah.com -> 200
  - https://matbaat-ruya.ofuqalmadenah.com -> 200

### Note

- Immediate post-restart checks briefly returned 502 while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-02-28 21:55:40 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260228_215404.bak`
- Installed binary SHA256: `b5223e64545021a63a33bbbf86379612892a24760536fb8bbf9b68402c00590b`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate checks after restart briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:21:23 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_132214.bak`
- Installed binary SHA256: `86934df1bd263ce19344829c75b6e18769e46852f8a824da3701614373e9eb98`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:57:52 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_135752.bak`
- Installed binary SHA256: `66b724514cb31eb4e4c49e570edabb01478d2cd1efa34afb36aa37db14f86fcc`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 19:09:57 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `6ed1c10` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_190933.bak`
- Installed binary SHA256: `8f61e12afae2b4b9067e66f211dfc1ac33d1f4f4e6c4a1a2080185d3fd213e69`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-02 01:04:04 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `6ed1c10` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260302_010403.bak`
- Installed binary SHA256: `28709e58397e704e5c87e81c06af90982a7e389b2fa50697f6de2188f6a35ba1`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- The core phone matching algorithm was fundamentally migrated from manual digit grouping loops to Google's official `libphonenumber` regex structural matching library, perfectly isolating Arabic and English multi-byte inputs without polluting explicit standard IDs and accounts.

## Deployment Update

Updated: 2026-03-03 14:48:57 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `fdfa791` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260303_144743.bak`
- Installed binary SHA256: `deed3269e20ab1b550304c00157f2a24d0e710de251de197ef490ac536991f12`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- The deployment was completed successfully using the standard workflow. All services were verified as operational post-restart.

## Deployment Update

Updated: 2026-03-03 15:14:24 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `fdfa791` (local working tree had uncommitted changes at `17:10`)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260303_151354.bak`
- Installed binary SHA256: `016c70f911f21051fdf82d504c33fde6a0792fa62a49e4ac6cc89c8b606bace7`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- This deployment includes new changes made to `frontend/src/services/api.ts` and `internal/config/config.go` observed at 17:10 local time.

## Deployment Update

Updated: 2026-03-03 15:25:01 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Direct binary upload artifact: `/tmp/whatomate-linux-20260303_172132`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.backup.20260303_152408`
- Installed binary SHA1: `4a6643270cb44ecfa72ae58b4aadaa677d575e52`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all 4 services `active`
- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- API check: `https://ofuqalmadenah.com/api/statuses` now returns `401 Missing authorization` (route is present; previous browser-side `404` issue was tied to old backend state before restart)

### Note

- A short startup window produced temporary `502` while services were restarting/migrating; final state is healthy.

## Storage Optimization Update

Updated: 2026-03-03 15:35:00 UTC

### What was implemented on VPS

- Added automated housekeeping script: `/usr/local/bin/whatomate-housekeeping.sh`
- Added settings file: `/etc/default/whatomate-housekeeping`
- Added systemd service: `/etc/systemd/system/whatomate-housekeeping.service`
- Added systemd timer: `/etc/systemd/system/whatomate-housekeeping.timer`
- Timer schedule: daily at `03:30 UTC` (`RandomizedDelaySec=20m`, `Persistent=true`)

### Housekeeping tasks

- Deduplicate identical media files using hardlinks in:
  - `/opt/whatomate/uploads`
  - `/opt/whatomate/instances/holol-wenjaz/uploads`
- Remove expired WhatsApp statuses from each tenant DB and delete associated local media files
- Keep only latest 5 binary backups in `/opt/whatomate/bin`
- Clear source-only artifacts in `/opt/whatomate-src/uploads` and test reports
- Vacuum systemd journal to a max size of `200M`

### Immediate one-time reclaim completed

- Dry-run estimated reclaim:
  - `/opt/whatomate/uploads`: `13.57 GiB`
  - `/opt/whatomate/instances/holol-wenjaz/uploads`: `4.6 GiB`
- Real dedupe reclaim completed: `18.17 GiB` total
- Old binary backups pruned: `/opt/whatomate/bin` reduced from `932M` to `262M`
- Source artifacts cleanup: `/opt/whatomate-src` reduced from `702M` to `514M`

### Current disk snapshot

- `/opt/whatomate/uploads`: `28G` (was `41G`)
- `/opt/whatomate/instances/holol-wenjaz/uploads`: `3.9G` (was `8.5G`)
- `/var/log/journal`: `77M`
- Root filesystem `/`: `64%` used (`62G` used / `35G` free)

### Service health

- `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`: all `active`

### How to control policy

- Edit `/etc/default/whatomate-housekeeping` and tune:
  - `STATUS_GRACE_HOURS`
  - `KEEP_BACKUPS`
  - `JOURNAL_MAX_SIZE`
  - `ENABLE_HARDLINK_DEDUP`
  - `CLEAN_SOURCE_UPLOADS`
  - `DRY_RUN`
- Apply changes:
  - `systemctl daemon-reload`
  - `systemctl restart whatomate-housekeeping.timer`


## Storage Policy Update (Message Media Safety)

Updated: 2026-03-03 15:51:30 UTC

- Housekeeping policy updated to preserve chat media files.
- New setting in `/etc/default/whatomate-housekeeping`:
  - `DELETE_STATUS_MEDIA_FILES=0` (default)
- Behavior now:
  - Expired `whatsapp_statuses` DB rows are deleted.
  - Media files are **not** deleted by default.
- Safety guard added in script:
  - If file deletion is enabled in future, script checks `messages.media_url` references first and keeps referenced files.

## Deployment Update

Updated: 2026-03-04 23:53:40 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `506a787` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260304_235338.bak`
- Installed binary SHA256: `26edbaa0e95ac568ed3ae330d669571adb962cae0adccdc04286e5746dab3513`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: `whatomate@holol-wenjaz` active
- Listener ports expected active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`

### Note

- This deployment includes the latest WebSocket `fastHTTPUpgrader` fixes that explicitly echo the `whm.v1` Subprotocol to resolve real-time message connection drops.

## Deployment Update

Updated: 2026-03-09 04:34:54 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded `.git`, node modules, local build/test artifacts, local env files, `config.toml`, and `uploads/`)
- Source revision on deploy: `506a787` (local working tree had uncommitted changes)
- Build host: local macOS workspace
- Build reason: VPS Go version is `1.22.2` while the repo currently requires `go 1.25.7`, so the production Linux binary was built locally and uploaded
- Build command: `GOOS=linux GOARCH=amd64 make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_043047.bak`
- Installed binary SHA256: `d63df8c5318a95a484fe2c151e1ded0a834c4a6df6c32547207f820ee3e531d2`
- Installed binary version output: `Whatomate 506a787-dirty (built 2026-03-09_04:29:44)`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Immediately after restart, short-lived `502` responses appeared for the base service and the first tenant during startup; both recovered to `200` once the processes finished binding and initialization.

## Deployment Update

Updated: 2026-03-09 07:04:18 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_070335.bak`
- Installed binary SHA256: `ab4484d4f2e53f4c2c6a846af59e277afaeb5226984f96c27335dc01d6c5b95d`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:03:18)`
- Deployment purpose: fix assigned chats for agents where the `Assigned` counter increased after reassignment but the chat stayed hidden in the sidebar because of the implicit frontend instance filter.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Initial status right after restart: all services `active`, URLs returned temporary `502` during startup warmup
- Final listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- Final HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

## Deployment Update

Updated: 2026-03-09 07:12:04 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_071137.bak`
- Installed binary SHA256: `06dc0dc299068f23b50a7150e487f2c213f18011832b37c4b0ddfef0b0e505fa`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:11:25)`
- Deployment purpose: replace `Unknown Instance` in the chat sidebar for self-assigned chats on restricted instances by using a safe fallback label from the chat payload when the instance is not available in `instancesStore`.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all four services `active`
- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

## Deployment Update

Updated: 2026-03-09 07:24:58 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `node_modules/`, generated `dist/`, and local security/report artifacts)
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_072333.bak`
- Installed binary SHA256: `6de468c6859100477bee7b5f04af37a8ffc4418e8b4380df0a85a35bba8d2566`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:24:14)`
- Deployment purpose: deploy the current local project state to production, including the latest workspace changes.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Frontend production build completed successfully on the VPS. Vite emitted the existing warning about `<script src=\"./theme-init.js\">` in `index.html`, but the final build and all runtime checks completed successfully.

## Deployment Update

Updated: 2026-03-11 12:47:45 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via tar archive; missing cmd/whatomate directory补)
- Source revision on deploy: Current working tree (uncommitted changes from test-strategy improvements)
- Native build command on VPS: `cd /opt/whatomate-src && CGO_ENABLED=0 go build -ldflags '-s -w' -o whatomate-new ./cmd/whatomate`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260311_124745.bak`
- Installed binary SHA256: `a7055dadac86cbe762805552c5b703e10348adbe78d1a56ab507ce83de09c2dc`
- Binary size: 46MB (with embedded frontend)
- Deployment notes:
  - Initial deployment attempt failed because tar archive was missing cmd/whatomate directory
  - Transferred cmd directory separately and rebuilt binary natively on VPS
  - Frontend properly embedded into binary (6.5MB of assets)
  - Previous binary (1.8MB) was missing embedded frontend and exited immediately

### Services Restarted

- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all three services `active`
- Listener ports active:
  - `127.0.0.1:18124` (holol-wenjaz)
  - `127.0.0.1:18125` (alarkan-almthalia)
  - `127.0.0.1:18126` (matbaat-ruya)
- Process IDs:
  - holol-wenjaz: PID 3038998
  - alarkan-almthalia: PID 3038999
  - matbaat-ruya: PID 3039000
- Log verification: All services show "Server listening" messages

### Note

- Deployment completed after fixing missing cmd/whatomate directory issue
- Frontend embedding now working correctly with proper 46MB binary size
- All tenant services operational with embedded frontend assets

## Deployment Update

Updated: 2026-03-12 11:56:35 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (incremental tar sync of the current changed workspace files into the existing source tree)
- Source revision on deploy: `b70cecd` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src/frontend && npm install && cd /opt/whatomate-src && VERSION=b70cecd-dirty GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260312_115332.bak`
- Installed binary SHA256: `d077226cb4bf9f5a4bc19ff1acdc934b4e40fba78eec095a612d8b074b8588d5`
- Installed binary version output: `Whatomate b70cecd-dirty (built 2026-03-12_11:51:57)`
- Deployment purpose: deploy the current local project state and remove the nginx-side upload block that returned `413 Request Entity Too Large` for a 2.1 MB media upload on `https://ofuqalmadenah.com/api/messages/media`.
- Edge configuration change:
  - Added `client_max_body_size 110M;` to:
    - `/etc/nginx/sites-available/ofuqalmadenah`
    - `/etc/nginx/sites-available/whatomate-holol-wenjaz.conf`
    - `/etc/nginx/sites-available/whatomate-alarkan-almthalia.conf`
    - `/etc/nginx/sites-available/whatomate-matbaat-ruya.conf`
  - Validated with `nginx -t` and reloaded nginx successfully.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- Upload ingress verification:
  - A 2.2 MB multipart POST to `https://ofuqalmadenah.com/api/messages/media` now returns `401 Missing authorization` JSON instead of nginx `413`, confirming the request reaches Whatomate.

### Note

- The old full-tree rsync path from macOS stalled; the successful deployment used an incremental tar sync of the changed workspace files into `/opt/whatomate-src`.
- The frontend build completed successfully on the VPS. Vite emitted the existing warning about `<script src="./theme-init.js">` in `index.html`, but the build, binary install, nginx reload, and service restarts all completed successfully.

## Deployment Update

Updated: 2026-03-12 13:55:17 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: source-only tar stream from `git ls-files --cached --modified --others --exclude-standard`, mirrored on the VPS into `/opt/whatomate-src`
- Source revision on deploy: `b70cecd` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Version-stamp rebuild on VPS: `cd /opt/whatomate-src && VERSION=b70cecd-dirty CGO_ENABLED=0 go build -ldflags "...main.Version=b70cecd-dirty..." -o whatomate ./cmd/whatomate`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260312_135323.bak`
- Installed binary SHA256: `ca9c67d86a0f2c188a400a2d6eedfea6f88f132333c0f30344ee6f6d851bf64f`
- Installed binary version output: `Whatomate b70cecd-dirty (built 2026-03-12_13:53:43)`
- Deployment purpose: publish the current local project state, including the new multi-file attachment send flow in chat and the related frontend/docs updates.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- The first tenant HTTP checks returned temporary `502` responses while the tenant services were still finishing startup and migrations; once their listeners bound to `127.0.0.1:18124-18126`, all tenant login pages returned `200`.
- The clean source-only sync avoided re-uploading the local `uploads/` directory and other large local artifacts that are not required for production builds.

## Deployment Update

Updated: 2026-03-17 03:28:38 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `1870edb` (working tree clean)
- Native build command on VPS: `cd /opt/whatomate-src && VERSION=1870edb GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260317_032750.bak`
- Installed binary SHA256: `fe13b8b49fc5f5918b6d03584afbe2e39fb12e535ba30cc8085bee82bbce3bda`
- Installed binary version output: `Whatomate 1870edb (built 2026-03-17_03:27:26)`
- Deployment purpose: deploy the current local project state to production.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- Process IDs:
  - whatomate: PID 3152955
  - holol-wenjaz: PID 3152967
  - alarkan-almthalia: PID 3152960
  - matbaat-ruya: PID 3152948
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Vite emitted the existing warning about `<script src="./theme-init.js">` lacking `type="module"`; the build and embed steps still completed successfully.
