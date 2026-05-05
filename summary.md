# Session Summary: Whatomate VPS Deployment Update

Date: 2026-05-05
VPS: `31.97.192.53`
Local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`

## Objective

Deploy the current local Whatomate version to the VPS, back up the installed version first, remove VPS source-code/runtime code trees, leave the production binary/runtime data, and fix the license system so License Overview is no longer disabled.

## Skills And Tools Used

- Skill selected: `devops-engineer`
- Code discovery: `codebase-memory-mcp` project/index inspection, then targeted local reads where the graph toolset did not expose `search_graph`
- Deployment tooling: SSH, rsync, systemd, Nginx, curl, jq
- Browser verification: Chrome DevTools MCP

## Deployment Result

- Built natively on the VPS with Go 1.25.9 and Node 20.19.6.
- Embedded the deploy public license keyring into the binary.
- Installed binary: `/opt/whatomate/bin/whatomate`
- Version: `Whatomate 7eafdfb-deploy-20260505 (built 2026-05-05_09:45:02)`
- SHA256: `60463f9c5e3a734692c0597ead69e9c076282ea4dd6bc63ed80e723d3f2a9715`

## Backups

- Focused pre-deploy backup:
  `/root/whatomate_backups/whatomate-installed-focused-20260505_093801.tar.gz`
- Pre-install binary backup:
  `/opt/whatomate/bin/whatomate.20260505_094527.pre-20260505-deploy.bak`
- Nginx root-vhost backup:
  `/etc/nginx/sites-available/ofuqalmadenah.20260505_094726.pre-production-upstream.bak`

## License Fix

- Added `[license] enabled = true` to the main and three tenant config files.
- Activated a host-bound lifetime production license signed by key ID `deploy-20260416`.
- All four local ports now report:
  `enabled=true`, `status=active`, `locked=false`
- License limits:
  5 organizations, 50 users per org, 50 WhatsApp endpoints per org, 25 workers.

## Services

Restarted and verified active:

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

Disabled and stopped:

- `whatomate-sandbox.service`

## VPS Cleanup

Removed source/runtime code trees:

- `/opt/whatomate-src`
- `/opt/whatomate-sandbox`
- `/tmp/whatomate-deploy-src`

Retained production runtime assets and data under `/opt/whatomate`, including the shared binary, configs, instances, uploads, databases, backups, Nginx, and systemd unit files.

## Nginx Fix

The root domain was returning 502 because `/etc/nginx/sites-available/ofuqalmadenah` still proxied to the removed sandbox source runtime on `127.0.0.1:18127`.

Updated that vhost to proxy to the production main instance at `127.0.0.1:18123`, ran `nginx -t`, and reloaded Nginx.

## Verification

HTTPS smoke checks returned 200:

- `https://ofuqalmadenah.com`
- `https://www.ofuqalmadenah.com`
- `https://holol-wenjaz.ofuqalmadenah.com`
- `https://alarkan-almthalia.ofuqalmadenah.com`
- `https://matbaat-ruya.ofuqalmadenah.com`

License bootstrap checks:

- `18123`: active
- `18124`: active
- `18125`: active
- `18126`: active

Chrome DevTools MCP verification:

- Opened `https://holol-wenjaz.ofuqalmadenah.com/settings/license`.
- App redirected normally to `/login`.
- Browser-side fetch to `/api/license/bootstrap` returned active production lifetime licensing.
- Screenshot saved at `tmp/deploy-verify-holol-license.png`.

## Docs Updated

- Local: `docs/whatomate_multi_instances_info.md`
- VPS: `/root/whatomate_multi_instances_info.md` and `/root/whatomate_production_info.md` after sync step.
