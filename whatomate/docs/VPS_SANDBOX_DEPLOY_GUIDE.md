# VPS Source Sandbox Deployment

This guide runs the current Whatomate source tree side by side with production on the same VPS, behind `sandbox.ofuqalmadenah.com`, on a separate loopback port.

## Important

- Using the same PostgreSQL database means sandbox writes are real production writes.
- `app.sandbox_mode = true` is intended for this exact case. It disables startup schema upgrades, Whatsmeow reconnect automation, recurring background jobs, and embedded workers.
- Do not run the sandbox with `-migrate`.
- Reuse the production `storage.local_path` only if you need existing media and uploads to resolve exactly like production.
- Use a separate Redis DB index for the sandbox so auth/rate-limit keys do not mix with production.

## Recommended layout

- Source checkout: `/opt/whatomate-sandbox/src`
- Production config reused from: `/opt/whatomate/config.toml`
- Sandbox service: `whatomate-sandbox.service`
- Sandbox loopback port: `127.0.0.1:18127`
- Public URL: `https://sandbox.ofuqalmadenah.com`

## Files added in this repo

- Systemd unit template: [whatomate-sandbox-source.service.example](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/deploy/systemd/whatomate-sandbox-source.service.example)
- Nginx vhost template: [sandbox.ofuqalmadenah.com.conf.example](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/deploy/nginx/sandbox.ofuqalmadenah.com.conf.example)
- Source-run wrapper: [run_source_server.sh](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/scripts/run_source_server.sh)

## Rollout

1. Sync the current repo to the VPS.

```bash
sudo mkdir -p /opt/whatomate-sandbox
sudo rsync -a --delete /opt/whatomate-src/ /opt/whatomate-sandbox/src/
```

2. Install the systemd unit and review the `User=` / `Group=` lines to match the current production service if needed.

```bash
sudo install -m 0644 /opt/whatomate-sandbox/src/deploy/systemd/whatomate-sandbox-source.service.example /etc/systemd/system/whatomate-sandbox.service
sudoedit /etc/systemd/system/whatomate-sandbox.service
sudo systemctl daemon-reload
```

3. Install the Nginx vhost.

```bash
sudo install -m 0644 /opt/whatomate-sandbox/src/deploy/nginx/sandbox.ofuqalmadenah.com.conf.example /etc/nginx/sites-available/sandbox.ofuqalmadenah.com.conf
sudo ln -sfn /etc/nginx/sites-available/sandbox.ofuqalmadenah.com.conf /etc/nginx/sites-enabled/sandbox.ofuqalmadenah.com.conf
sudo nginx -t
sudo systemctl reload nginx
```

4. Point DNS `A` record `sandbox.ofuqalmadenah.com` to the VPS IP.

5. Start the sandbox service.

```bash
sudo systemctl enable --now whatomate-sandbox.service
sudo systemctl status whatomate-sandbox.service --no-pager -l
```

6. Issue TLS after the HTTP vhost is live.

```bash
sudo certbot --nginx -d sandbox.ofuqalmadenah.com
```

## Verification

```bash
curl -I http://127.0.0.1:18127/login
curl -I https://sandbox.ofuqalmadenah.com/login
sudo journalctl -u whatomate-sandbox.service -n 100 --no-pager
ss -ltnp | grep 18127
```

Expected:

- the source-run service binds to `127.0.0.1:18127`
- `https://sandbox.ofuqalmadenah.com/login` returns `200`
- the logs include sandbox-mode warnings about skipped background automation

## Rollback

```bash
sudo systemctl disable --now whatomate-sandbox.service
sudo rm -f /etc/nginx/sites-enabled/sandbox.ofuqalmadenah.com.conf
sudo nginx -t
sudo systemctl reload nginx
```

## Notes on safety

- This setup is good for UI and request-path validation against live data.
- It is not safe for destructive functional testing, outbound message testing, or schema testing.
- If you need realistic but isolated testing, clone PostgreSQL and uploads instead of sharing them.
