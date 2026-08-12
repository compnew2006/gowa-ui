#!/usr/bin/env bash
# Connects the gowa-ui instance to the GOWA server and lists connected devices.
# Run ON the VPS. Talks to the local gowa-ui API (127.0.0.1:8081) as the admin.
set -euo pipefail
API=http://127.0.0.1:8081
ORIGIN=http://31.97.192.53:8081
J=/tmp/gowa.cookies
EMAIL=admin@gowa-ui.local
PASS=$(grep admin_password /opt/gowa-ui/.deploy-secrets | cut -d= -f2 | tr -d ' ')
BASE=https://gowa.ofuqalmadenah.com
GUSER=gowa_main_1ccb48ee
GPASS=6l9k6CVR1xftU2h4QLgxtyMS

echo "== 1) login as admin =="
curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" -o /dev/null -w "login -> HTTP %{http_code}\n"

# helper: extract ids from a SendEnvelope
pyid(){ python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('id',''))" 2>/dev/null; }

echo "== 2) ensure GOWA instance exists (idempotent) =="
INST_ID=$(curl -s -b "$J" "$API/api/gowa/servers" \
  | python3 -c "import sys,json;d=json.load(sys.stdin);insts=d.get('data',d).get('instances',[]) or d.get('data',d) or [];import sys as s;[print(i['id']) for i in (insts if isinstance(insts,list) else []) if i.get('base_url')=='$BASE']" 2>/dev/null | head -1 || true)
if [ -z "$INST_ID" ]; then
  echo "creating instance..."
  INST_ID=$(curl -s -b "$J" -X POST "$API/api/gowa/servers" -H 'Content-Type: application/json' \
    -d "{\"name\":\"GOWA Main\",\"base_url\":\"$BASE\",\"username\":\"$GUSER\",\"password\":\"$GPASS\",\"webhook_url\":\"$ORIGIN/api/gowa/webhook\",\"is_active\":true}" \
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

rm -f "$J"
