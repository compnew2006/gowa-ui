#!/usr/bin/env bash
# Connects the gowa-ui instance to the GOWA server and lists connected devices.
# Run ON the VPS. Talks to the local gowa-ui API (127.0.0.1:8081) as the admin.
set -euo pipefail
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
API=http://127.0.0.1:8081
ORIGIN=http://31.97.192.53:8081
J=/tmp/gowa.cookies
EMAIL=admin@gowa-ui.local
PASS=$(grep admin_password /opt/gowa-ui/.deploy-secrets | cut -d= -f2 | tr -d ' ')
GUSER=$GOWA_USER
GPASS=$GOWA_PASSWORD
BASE=${GOWA_BASE_URL%/}
trap 'rm -f "$J"' EXIT

echo "== 1) login as admin =="
LOGIN_BODY=$(LOGIN_EMAIL="$EMAIL" LOGIN_PASSWORD="$PASS" python3 -c 'import json,os;print(json.dumps({"email":os.environ["LOGIN_EMAIL"],"password":os.environ["LOGIN_PASSWORD"]}))')
printf '%s' "$LOGIN_BODY" | curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
  --data-binary @- -o /dev/null -w "login -> HTTP %{http_code}\n"

# helper: extract ids from a SendEnvelope
pyid(){ python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('id',''))" 2>/dev/null; }

echo "== 2) ensure GOWA instance exists (idempotent) =="
INST_ID=$(curl -s -b "$J" "$API/api/gowa/servers" \
  | EXPECTED_BASE_URL="$BASE" python3 -c 'import os,sys,json;d=json.load(sys.stdin);insts=d.get("data",d).get("instances",[]) or d.get("data",d) or [];[print(i["id"]) for i in (insts if isinstance(insts,list) else []) if i.get("base_url")==os.environ["EXPECTED_BASE_URL"]]' 2>/dev/null | head -1 || true)
if [ -z "$INST_ID" ]; then
  echo "creating instance..."
  INSTANCE_BODY=$(INSTANCE_BASE="$BASE" INSTANCE_USERNAME="$GUSER" INSTANCE_PASSWORD="$GPASS" INSTANCE_WEBHOOK_URL="$ORIGIN/api/gowa/webhook" python3 -c '
import json,os
print(json.dumps({"name":"GOWA Main","base_url":os.environ["INSTANCE_BASE"],"username":os.environ["INSTANCE_USERNAME"],"password":os.environ["INSTANCE_PASSWORD"],"webhook_url":os.environ["INSTANCE_WEBHOOK_URL"],"is_active":True}))
')
  INST_ID=$(printf '%s' "$INSTANCE_BODY" | curl -s -b "$J" -X POST "$API/api/gowa/servers" -H 'Content-Type: application/json' --data-binary @- \
    | python3 -c "import sys,json;d=json.load(sys.stdin);print((d.get('data',d).get('instance') or {}).get('id',''))" 2>/dev/null)
fi
echo "instance id: $INST_ID"

echo "== 3) connected devices on the GOWA server =="
curl -s -b "$J" "$API/api/gowa/servers/$INST_ID/devices" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
data=d.get('data',d)
devs = data.get('devices') or data.get('results') or (data if isinstance(data,list) else [])
print('total devices:', len(devs))
for x in devs:
    print(' -', repr(x.get('id')), '| state=', x.get('state'), '|', x.get('jid') or x.get('display_name') or '')
"
