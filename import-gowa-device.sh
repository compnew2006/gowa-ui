#!/usr/bin/env bash
# Imports GOWA device(s) as WhatsApp accounts in gowa-ui and syncs their history.
# Usage: bash import-gowa-device.sh <device_id> [<device_id> ...]
# Run ON the VPS. Talks to local gowa-ui API as admin.
set -uo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
umask 077
if [[ "$#" -eq 0 ]]; then
  echo "Usage: bash import-gowa-device.sh <device_id> [<device_id> ...]" >&2
  exit 2
fi
API=http://127.0.0.1:8081
ORIGIN=http://31.97.192.53:8081
J=$(mktemp "${TMPDIR:-/tmp}/gowa-cookies.XXXXXX")
CSRF_HEADER_FILE=$(mktemp "${TMPDIR:-/tmp}/gowa-csrf-header.XXXXXX")
trap 'rm -f "$J" "$CSRF_HEADER_FILE"' EXIT
DEPLOY_SECRETS_FILE="${GOWA_UI_DEPLOY_SECRETS_FILE:-/opt/gowa-ui/.deploy-secrets}"
read_deploy_value() {
  [[ -e "$DEPLOY_SECRETS_FILE" || -L "$DEPLOY_SECRETS_FILE" ]] || return 0
  python3 "$SCRIPT_DIR/scripts/deploy_secrets.py" "$DEPLOY_SECRETS_FILE" "$1"
}
EMAIL="${GOWA_UI_ADMIN_EMAIL:-$(read_deploy_value admin_email)}"
PASS="$(read_deploy_value admin_password)"
BASE="${GOWA_BASE_URL:-$(read_deploy_value gowa_base_url)}"
: "${EMAIL:?Set GOWA_UI_ADMIN_EMAIL or admin_email in the deployment secrets file}"
: "${PASS:?admin_password is missing from $DEPLOY_SECRETS_FILE}"
: "${BASE:?Set GOWA_BASE_URL or gowa_base_url in the deployment secrets file}"
GOWA_BASE_URL="$BASE" python3 - <<'PY_VALIDATE' || exit 1
import os
import sys
from urllib.parse import urlsplit

base = os.environ["GOWA_BASE_URL"]
parsed = urlsplit(base)
if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
    sys.exit("GOWA_BASE_URL must be an HTTP(S) URL without embedded credentials, query data, or fragment")
PY_VALIDATE
BASE="${BASE%/}"
ENC(){ python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" "$1"; }
resolve_gowa_instance_id() {
  curl -s -b "$J" "$API/api/gowa/servers" \
    | GOWA_BASE_URL="$BASE" python3 -c '
import json
import os
import sys

payload = json.load(sys.stdin)
data = payload.get("data") or payload
instances = data.get("instances", []) or []
base = os.environ["GOWA_BASE_URL"]
print(next((str(instance.get("id", "")) for instance in instances
            if str(instance.get("base_url", "")).rstrip("/") == base), ""))
' 2>/dev/null
}

# login fresh
python3 -c \
  'import json,sys; print(json.dumps({"email": sys.argv[1], "password": sys.stdin.read().rstrip("\n")}))' "$EMAIL" <<< "$PASS" \
  | curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
      --data-binary @- -o /dev/null
unset PASS
CSRF=$(awk '$6=="whm_csrf"{print $7}' "$J")
: "${CSRF:?login did not return a CSRF token}"
printf 'X-CSRF-Token: %s\n' "$CSRF" > "$CSRF_HEADER_FILE"

# resolve instance id
INST_ID="$(resolve_gowa_instance_id)"
INSTANCE_FOUND=$([ -n "$INST_ID" ] && echo yes || echo no)
: "${INST_ID:?No GOWA instance matches the configured GOWA_BASE_URL}"
echo "GOWA instance resolved: ${INSTANCE_FOUND}"

DEVICE_NUMBER=0
for DEV in "$@"; do
  DEVICE_NUMBER=$((DEVICE_NUMBER + 1))
  NAME="$DEV"
  echo ""
  echo "== device #$DEVICE_NUMBER =="
  # 1) create account; stop before syncing if the API rejects it.
  ACCOUNT_BODY=$(ACCOUNT_NAME="$NAME" ACCOUNT_BASE="$BASE" ACCOUNT_DEVICE="$DEV" python3 -c 'import json,os; print(json.dumps({"name":os.environ["ACCOUNT_NAME"],"gowa_base_url":os.environ["ACCOUNT_BASE"],"gowa_device_id":os.environ["ACCOUNT_DEVICE"]}))')
  CREATE_CODE=$(printf '%s' "$ACCOUNT_BODY" | curl -sS -b "$J" -X POST "$API/api/accounts" \
    -H "Origin: $ORIGIN" -H "@$CSRF_HEADER_FILE" -H 'Content-Type: application/json' \
    --data-binary @- -o /dev/null -w "%{http_code}")
  unset ACCOUNT_BODY
  if [[ ! "$CREATE_CODE" =~ ^2[0-9][0-9]$ ]]; then
    echo "create account for device $DEVICE_NUMBER failed -> HTTP ${CREATE_CODE:-unknown}" >&2
    exit 1
  fi
  echo "create account -> HTTP $CREATE_CODE"

  # 2) sync messages (pulls history -> chats appear). deviceId is URL-encoded.
  DEVE=$(ENC "$DEV")
  SYNC_CODE=$(curl -sS --max-time 60 -o /dev/null -w "%{http_code}" -b "$J" -X POST "$API/api/gowa/servers/$INST_ID/devices/$DEVE/sync-messages" \
    -H "Origin: $ORIGIN" -H "@$CSRF_HEADER_FILE")
  if [[ ! "$SYNC_CODE" =~ ^2[0-9][0-9]$ ]]; then
    echo "sync device $DEVICE_NUMBER failed -> HTTP ${SYNC_CODE:-unknown}" >&2
    exit 1
  fi
  echo "sync-messages -> HTTP $SYNC_CODE"
done

echo ""
echo "== gowa_ui DB: contacts + messages now =="
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'contacts='||count(*) FROM contacts;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'messages='||count(*) FROM messages;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'accounts='||count(*) FROM whatsapp_accounts;\"" 2>/dev/null
