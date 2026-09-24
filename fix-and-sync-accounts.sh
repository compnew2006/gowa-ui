#!/usr/bin/env bash
# Fixes the truncated GOWA device_ids on the existing accounts and syncs history.
# Run ON the VPS.
set -uo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
umask 077
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

echo "== 1) login (capture CSRF) =="
python3 -c \
  'import json,sys; print(json.dumps({"email": sys.argv[1], "password": sys.stdin.read().rstrip("\n")}))' "$EMAIL" <<< "$PASS" \
  | curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
      --data-binary @- -o /dev/null -w "login -> HTTP %{http_code}\n"
unset PASS
CSRF=$(awk '$6=="whm_csrf"{print $7}' "$J")
: "${CSRF:?login did not return a CSRF token}"
printf 'X-CSRF-Token: %s\n' "$CSRF" > "$CSRF_HEADER_FILE"
INST_ID="$(resolve_gowa_instance_id)"
INSTANCE_FOUND=$([ -n "$INST_ID" ] && echo yes || echo no)
: "${INST_ID:?No GOWA instance matches the configured GOWA_BASE_URL}"
echo "GOWA instance resolved: ${INSTANCE_FOUND} ; csrf captured: $([ -n "$CSRF" ] && echo yes || echo NO)"

echo ""
echo "== 3) apply private device-id mapping =="
DEVICE_MAP_FILE="${GOWA_DEVICE_MAP_FILE:-/opt/gowa-ui/.gowa-device-map.tsv}"
if [[ ! -r "$DEVICE_MAP_FILE" ]]; then
  echo "Device map is missing or unreadable: $DEVICE_MAP_FILE" >&2
  echo "Create a root-owned mode-0600 TSV with old_device_id, full_device_id, jid, and organization_id columns." >&2
  exit 1
fi
echo ""
echo "== 4) sync each device's history =="
DEVICE_NUMBER=0
if ! python3 "$SCRIPT_DIR/scripts/gowa_device_map.py" "$DEVICE_MAP_FILE" "$BASE" \
    | su - postgres -c 'psql -X -q -tA -v ON_ERROR_STOP=1 -d gowa_ui' \
    | while IFS= read -r RECORD; do
  if [[ "$RECORD" == "MAPPING_COMMITTED" ]]; then
    echo "Device-id mapping committed atomically."
    continue
  fi
  [[ "$RECORD" == DEVICE:* ]] || continue
  DEV="${RECORD#DEVICE:}"
  [ -z "$DEV" ] && continue
  DEVICE_NUMBER=$((DEVICE_NUMBER + 1))
  DEVE=$(ENC "$DEV")
  CODE=$(curl -sS --max-time 90 -o /dev/null -w "%{http_code}" -b "$J" -X POST \
    "$API/api/gowa/servers/$INST_ID/devices/$DEVE/sync-messages" \
    -H "Origin: $ORIGIN" -H "@$CSRF_HEADER_FILE")
  if [[ ! "$CODE" =~ ^2[0-9][0-9]$ ]]; then
    echo "sync device $DEVICE_NUMBER failed -> HTTP ${CODE:-unknown}" >&2
    exit 1
  fi
  echo "sync device $DEVICE_NUMBER -> HTTP $CODE"
done; then
  echo "The mapping transaction or one or more history sync requests failed; any database remap was atomic." >&2
  exit 1
fi

echo ""
echo "== 4) result: contacts + messages in gowa_ui =="
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'contacts='||count(*) FROM contacts;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'messages='||count(*) FROM messages;\"" 2>/dev/null
rm -f "$J"
