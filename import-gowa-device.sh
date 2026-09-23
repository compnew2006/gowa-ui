#!/usr/bin/env bash
# Imports GOWA device(s) as WhatsApp accounts in gowa-ui and syncs their history.
# Usage: bash import-gowa-device.sh <device_id> [<device_id> ...]
# Run ON the VPS. Talks to local gowa-ui API as admin.
set -uo pipefail
umask 077
API=http://127.0.0.1:8081
ORIGIN=http://31.97.192.53:8081
J=$(mktemp "${TMPDIR:-/tmp}/gowa-cookies.XXXXXX")
trap 'rm -f "$J"' EXIT
EMAIL=admin@gowa-ui.local
PASS=$(grep '^admin_password=' /opt/gowa-ui/.deploy-secrets | head -n 1 | cut -d= -f2-)
: "${PASS:?admin_password is missing from /opt/gowa-ui/.deploy-secrets}"
BASE=https://gowa.ofuqalmadenah.com
ENC(){ python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" "$1"; }

# login fresh
python3 -c \
  'import json,sys; print(json.dumps({"email": sys.argv[1], "password": sys.stdin.read().rstrip("\n")}))' "$EMAIL" <<< "$PASS" \
  | curl -s -c "$J" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' -H "Origin: $ORIGIN" \
      --data-binary @- -o /dev/null
unset PASS

# resolve instance id
INST_ID=$(curl -s -b "$J" "$API/api/gowa/servers" \
  | python3 -c "import sys,json;d=json.load(sys.stdin);insts=d.get('data',d).get('instances',[]) or [];print(next((i['id'] for i in insts if i.get('base_url')=='$BASE'),''))" 2>/dev/null)
echo "instance: $INST_ID"

for DEV in "$@"; do
  NAME="$DEV"
  echo ""
  echo "== device: $DEV =="
  # 1) create account (idempotent — skip if it already exists)
  RESP=$(curl -s -b "$J" -X POST "$API/api/accounts" -H 'Content-Type: application/json' \
    -d "{\"name\":\"$NAME\",\"gowa_base_url\":\"$BASE\",\"gowa_device_id\":\"$DEV\"}")
  CODE=$(echo "$RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('code') or d.get('status') or '')" 2>/dev/null || echo "?")
  echo "create account -> $CODE | $(echo "$RESP" | head -c 160)"

  # 2) sync messages (pulls history -> chats appear). deviceId is URL-encoded.
  DEVE=$(ENC "$DEV")
  SYNC=$(curl -s --max-time 60 -b "$J" -X POST "$API/api/gowa/servers/$INST_ID/devices/$DEVE/sync-messages" \
    -H "Origin: $ORIGIN")
  echo "sync-messages  -> $(echo "$SYNC" | head -c 200)"
done

echo ""
echo "== gowa_ui DB: contacts + messages now =="
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'contacts='||count(*) FROM contacts;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'messages='||count(*) FROM messages;\"" 2>/dev/null
su - postgres -c "psql -d gowa_ui -tAc \"SELECT 'accounts='||count(*) FROM whatsapp_accounts;\"" 2>/dev/null
