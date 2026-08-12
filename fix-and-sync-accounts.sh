#!/usr/bin/env bash
# Fixes the truncated GOWA device_ids on the existing accounts and syncs history.
# Run ON the VPS.
set -uo pipefail
API=http://127.0.0.1:8081
ORIGIN=http://31.97.192.53:8081
J=/tmp/gowa.cookies
EMAIL=admin@gowa-ui.local
PASS=$(grep admin_password /opt/gowa-ui/.deploy-secrets | cut -d= -f2 | tr -d ' ')
BASE=https://gowa.ofuqalmadenah.com
ENC(){ python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" "$1"; }

echo "== 1) fix truncated device_ids -> correct full GOWA device ids (+jid) =="
su - postgres -c "psql -d gowa_ui" <<'SQL'
UPDATE whatsapp_accounts SET gowa_device_id='امين -4210',         gowa_jid='966531521631@s.whatsapp.net' WHERE gowa_device_id='4210';
UPDATE whatsapp_accounts SET gowa_device_id='Print-Labn-4614',   gowa_jid='966553444614@s.whatsapp.net' WHERE gowa_device_id='Labn-4614';
UPDATE whatsapp_accounts SET gowa_device_id='Adv-1926',          gowa_jid='966535551926@s.whatsapp.net' WHERE gowa_device_id='1926';
UPDATE whatsapp_accounts SET gowa_device_id='تصميم عسير -4625',  gowa_jid='966594374625@s.whatsapp.net' WHERE gowa_device_id='4625';
UPDATE whatsapp_accounts SET gowa_device_id='4395 - عماد',       gowa_jid='966506524395@s.whatsapp.net' WHERE gowa_device_id='4395';
UPDATE whatsapp_accounts SET gowa_device_id='عائشة -8930',        gowa_jid='966556698930@s.whatsapp.net' WHERE gowa_device_id='8930';
UPDATE whatsapp_accounts SET gowa_device_id='محمد ابراهيم -6178', gowa_jid='966531526178@s.whatsapp.net' WHERE gowa_device_id='6178';
SELECT name, gowa_device_id, gowa_jid FROM whatsapp_accounts ORDER BY gowa_device_id;
SQL

echo ""
echo "== 2) login (capture CSRF) =="
curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" -o /dev/null -w "login -> HTTP %{http_code}\n"
CSRF=$(awk '$6=="whm_csrf"{print $7}' "$J")
INST_ID=$(curl -s -b "$J" "$API/api/gowa/servers" \
  | python3 -c "import sys,json;d=json.load(sys.stdin);insts=d.get('data',d).get('instances',[]) or [];print(next((i['id'] for i in insts if i.get('base_url')=='$BASE'),''))" 2>/dev/null)
echo "instance: $INST_ID ; csrf captured: $([ -n "$CSRF" ] && echo yes || echo NO)"

echo ""
echo "== 3) sync each device's history =="
su - postgres -c "psql -d gowa_ui -tAc \"SELECT gowa_device_id FROM whatsapp_accounts ORDER BY gowa_device_id;\"" 2>/dev/null | while IFS= read -r DEV; do
  [ -z "$DEV" ] && continue
  DEVE=$(ENC "$DEV")
  CODE=$(curl -s --max-time 90 -o /dev/null -w "%{http_code}" -b "$J" -X POST \
    "$API/api/gowa/servers/$INST_ID/devices/$DEVE/sync-messages" \
    -H "Origin: $ORIGIN" -H "X-CSRF-Token: $CSRF")
  echo "sync $DEV -> HTTP $CODE"
done

echo ""
echo "== 4) result: contacts + messages in gowa_ui =="
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'contacts='||count(*) FROM contacts;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'messages='||count(*) FROM messages;\"" 2>/dev/null
rm -f "$J"
