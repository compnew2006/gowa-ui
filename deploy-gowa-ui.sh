#!/usr/bin/env bash
# Sets up gowa-ui as a SEPARATE service on the VPS.
# Does NOT touch whatomate (/opt/whatomate*, :18123, whatomate_prod DB, redis db 0).
# All secrets are generated ON the VPS and stored to /opt/gowa-ui/.deploy-secrets (root-only).
set -euo pipefail

APP_DIR=/opt/gowa-ui
BIN=$APP_DIR/gowa-ui
CFG=$APP_DIR/config.toml
DB_NAME=gowa_ui
DB_USER=gowa_ui
REDIS_DB=2         # whatomate uses db 0; gowa-ui uses db 2 (distinct)
PORT=8081          # whatomate uses 18123; gowa-ui uses 8081 (free)

echo "==[ gowa-ui separate deploy ]=="

# 1) dirs
mkdir -p "$APP_DIR"/uploads

# 2) generate secrets locally on the VPS (never printed)
ENC_KEY=$(openssl rand -hex 32)          # 64 hex chars >= 32
JWT_SECRET=$(openssl rand -hex 32)
DB_PASS=$(openssl rand -hex 16)          # hex => safe in SQL quotes
ADMIN_PASS=$(openssl rand -base64 18 | tr -d '/+=' | cut -c1-20)

# 3) create a dedicated Postgres role + database (separate from whatomate_prod)
if su - postgres -c "psql -tAc \"SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'\"" | grep -q 1; then
  echo "role ${DB_USER} exists; leaving as-is"
else
  su - postgres -c "psql -c \"CREATE ROLE ${DB_USER} LOGIN PASSWORD '${DB_PASS}';\"" >/dev/null
  echo "created role ${DB_USER}"
fi
if su - postgres -c "psql -tAc \"SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'\"" | grep -q 1; then
  echo "database ${DB_NAME} exists; reusing"
else
  su - postgres -c "psql -c \"CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};\"" >/dev/null
  echo "created database ${DB_NAME}"
fi

# 4) write config (VPS values; reuses existing Redis password + a distinct db index)
REDIS_PASS=$(sed -n "/^\[redis\]/,/^\[/p" /opt/whatomate/config.toml | awk -F'"' '/password/{print $2; exit}')
cat > "$CFG" <<EOF
# gowa-ui config — separate service. Generated $(date -u '+%Y-%m-%d %H:%M:%S UTC')
[app]
name = "Gowa-UI"
environment = "production"
debug = false
encryption_key = "${ENC_KEY}"

[server]
host = "0.0.0.0"
port = ${PORT}
read_timeout = 30
write_timeout = 30
base_path = ""
allowed_origins = ""

[database]
host = "127.0.0.1"
port = 5432
user = "${DB_USER}"
password = "${DB_PASS}"
name = "${DB_NAME}"
ssl_mode = "disable"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300

[redis]
host = "127.0.0.1"
port = 6379
username = ""
password = "${REDIS_PASS}"
db = ${REDIS_DB}
tls = false

[jwt]
secret = "${JWT_SECRET}"
access_expiry_mins = 15
refresh_expiry_days = 1

[storage]
type = "local"
local_path = "${APP_DIR}/uploads"

[cookie]
domain = ""
secure = false

[rate_limit]
enabled = true
login_max_attempts = 10
register_max_attempts = 10
refresh_max_attempts = 30
sso_max_attempts = 10
window_seconds = 60
trust_proxy = false
api_max_requests = 200
api_window_seconds = 60

[default_admin]
email = "admin@gowa-ui.local"
password = "${ADMIN_PASS}"
full_name = "Gowa-UI Admin"
EOF
chmod 600 "$CFG"

# 5) systemd unit
cat > /etc/systemd/system/gowa-ui.service <<'UNIT'
[Unit]
Description=Gowa-UI (separate test service — does NOT replace whatomate)
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
WorkingDirectory=/opt/gowa-ui
ExecStart=/opt/gowa-ui/gowa-ui server -config /opt/gowa-ui/config.toml -migrate
Restart=always
RestartSec=5
KillSignal=SIGINT
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
UNIT

# 6) save secrets for the operator (root-only)
cat > "$APP_DIR"/.deploy-secrets <<EOF
# gowa-ui separate deploy secrets — generated $(date -u '+%Y-%m-%d %H:%M:%S UTC')
admin_email = admin@gowa-ui.local
admin_password = ${ADMIN_PASS}
db_user = ${DB_USER}
db_name = ${DB_NAME}
db_password = ${DB_PASS}
redis_db = ${REDIS_DB}
port = ${PORT}
config = ${CFG}
EOF
chmod 600 "$APP_DIR"/.deploy-secrets

# 7) enable + start (creates schema in the EMPTY gowa_ui DB; does not touch whatomate_prod)
systemctl daemon-reload
systemctl enable gowa-ui >/dev/null 2>&1 || true
systemctl restart gowa-ui
echo "started gowa-ui.service — waiting for it to come up..."
sleep 6
systemctl --no-pager --lines=0 status gowa-ui 2>&1 | head -8 || true

echo "==[ deploy script done ]=="
echo "Secrets saved to: ${APP_DIR}/.deploy-secrets (chmod 600)"
