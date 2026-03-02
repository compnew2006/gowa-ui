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
- PostgreSQL Password: 3efee7e82b30b940d0c615defebb5e8e0e93acc2e682808a
- Default Admin Email: admin+holol-wenjaz@whatomate.local
- Default Admin Password: Zqxtu8r3mRhwhJ6oLYaxqE/7
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
- PostgreSQL Password: 9975ca4213c2423bffff24f9f07c457b96b57bda32f21d14
- Default Admin Email: admin+alarkan-almthalia@whatomate.local
- Default Admin Password: lqgZPoOVWgqVqIhLWa2p0gI9
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
- PostgreSQL Password: 51eaf94195c125683c528275fbcfab4473eeaabace420917
- Default Admin Email: admin+matbaat-ruya@whatomate.local
- Default Admin Password: nRiMXl0DuJe8xfOS707NdPnU
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
