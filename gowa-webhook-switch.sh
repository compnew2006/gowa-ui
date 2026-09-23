#!/usr/bin/env bash
# Temporarily re-points GOWA device webhooks to gowa-ui (test), keeping each
# device's current secret (already synced to the gowa-ui account by CreateAccount).
# Saves the exact current config per device for an EXACT revert.
# Usage: bash gowa-webhook-switch.sh "Dev1" "Dev2" ...
# Run ON the VPS.
set -uo pipefail
umask 077
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
NEWURL=http://31.97.192.53:8081/api/gowa/webhook
EVENTS="message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer"
BK=/root/gowa-webhook-backup
mkdir -p "$BK"

# Keep authentication out of curl's process arguments. The temporary config
# contains only a base64 Authorization header and is removed after each call.
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

for DEV in "$@"; do
  ENC=$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" "$DEV")
  echo "== $DEV =="
  CUR=$(gowa_curl -s --max-time 12 "$BASE/devices/$ENC/webhook")
  echo "$CUR" > "$BK/$ENC.json"          # exact revert data (url+secret+events)
  SECRET=$(echo "$CUR" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('results',{}).get('webhook_secret',''))" 2>/dev/null)
  OLDURL=$(echo "$CUR" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('results',{}).get('webhook_url',''))" 2>/dev/null)
  echo "  previous webhook captured (secret len ${#SECRET}, saved backup)"
  RESP=$(WEBHOOK_URL="$NEWURL" WEBHOOK_SECRET="$SECRET" WEBHOOK_EVENTS="$EVENTS" python3 -c '
import json, os
print(json.dumps({"webhook_url":os.environ["WEBHOOK_URL"],"webhook_secret":os.environ["WEBHOOK_SECRET"],"webhook_events":os.environ["WEBHOOK_EVENTS"],"webhook_insecure_skip_verify":False}))
' | gowa_curl -s --max-time 12 -X PATCH -H 'Content-Type: application/json' --data-binary @- \
    "$BASE/devices/$ENC/webhook")
  echo "  webhook updated"
done
echo ""
echo "backups (for revert): $BK"
ls "$BK"
