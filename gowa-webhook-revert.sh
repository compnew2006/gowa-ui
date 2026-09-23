#!/usr/bin/env bash
# REVERT: restores each GOWA device's webhook to its pre-switch config saved in
# /root/gowa-webhook-backup/. Run ON the VPS after testing.
set -uo pipefail
: "${GOWA_USER:?GOWA_USER is required; export the GOWA API username}"
: "${GOWA_PASSWORD:?GOWA_PASSWORD is required; export the GOWA API password}"
: "${GOWA_BASE_URL:?GOWA_BASE_URL is required; export the GOWA API base URL}"
python3 - <<'PY' || { echo "GOWA_BASE_URL must be an HTTP(S) URL without embedded credentials or query data" >&2; exit 1; }
import os, sys
from urllib.parse import urlsplit
u = urlsplit(os.environ["GOWA_BASE_URL"])
sys.exit(0 if u.scheme in ("http", "https") and u.hostname and not u.username and not u.password and not u.query and not u.fragment else 1)
PY
GUSER=$GOWA_USER
GPASS=$GOWA_PASSWORD
BASE=${GOWA_BASE_URL%/}
BK=/root/gowa-webhook-backup
echo "Reverting GOWA device webhooks to pre-switch config..."

CURL_CONFIG=""
cleanup_curl_config() {
  if [ -n "$CURL_CONFIG" ]; then
    rm -f "$CURL_CONFIG"
    CURL_CONFIG=""
  fi
}
trap cleanup_curl_config EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

gowa_curl() {
  local status
  CURL_CONFIG=$(mktemp)
  chmod 600 "$CURL_CONFIG"
  python3 - "$CURL_CONFIG" <<'PY'
import base64, os, sys
token = base64.b64encode((os.environ["GOWA_USER"] + ":" + os.environ["GOWA_PASSWORD"]).encode()).decode()
with open(sys.argv[1], "w", encoding="utf-8") as f:
    f.write('header = "Authorization: Basic ' + token + '"\n')
PY
  curl --config "$CURL_CONFIG" "$@"
  status=$?
  cleanup_curl_config
  return "$status"
}

for f in "$BK"/*.json; do
  ENC=$(basename "$f" .json)
  body=$(python3 -c "
import json
d=json.load(open('$f')).get('results',{})
print(json.dumps({'webhook_url':d.get('webhook_url','') or '','webhook_secret':d.get('webhook_secret','') or '','webhook_events':d.get('webhook_events','') or '','webhook_insecure_skip_verify':False}))
" 2>/dev/null)
  [ -z "$body" ] && { echo "  $ENC: no valid backup (skip)"; continue; }
  CODE=$(printf '%s' "$body" | gowa_curl -s --max-time 12 -o /dev/null -w "%{http_code}" -X PATCH \
    -H 'Content-Type: application/json' --data-binary @- "$BASE/devices/$ENC/webhook")
  echo "  $ENC -> reverted (HTTP $CODE)"
done
echo "Done. Devices restored to their pre-switch webhook config."
