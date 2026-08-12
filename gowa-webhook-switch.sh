#!/usr/bin/env bash
# Temporarily re-points GOWA device webhooks to gowa-ui (test), keeping each
# device's current secret (already synced to the gowa-ui account by CreateAccount).
# Saves the exact current config per device for an EXACT revert.
# Usage: bash gowa-webhook-switch.sh "Dev1" "Dev2" ...
# Run ON the VPS.
set -uo pipefail
GUSER=gowa_main_1ccb48ee
GPASS=6l9k6CVR1xftU2h4QLgxtyMS
BASE=https://gowa.ofuqalmadenah.com
NEWURL=http://31.97.192.53:8081/api/gowa/webhook
EVENTS="message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer"
BK=/root/gowa-webhook-backup
mkdir -p "$BK"

for DEV in "$@"; do
  ENC=$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" "$DEV")
  echo "== $DEV =="
  CUR=$(curl -s --max-time 12 -u "$GUSER:$GPASS" "$BASE/devices/$ENC/webhook")
  echo "$CUR" > "$BK/$ENC.json"          # exact revert data (url+secret+events)
  SECRET=$(echo "$CUR" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('results',{}).get('webhook_secret',''))" 2>/dev/null)
  OLDURL=$(echo "$CUR" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('results',{}).get('webhook_url',''))" 2>/dev/null)
  echo "  was: $OLDURL  (secret len ${#SECRET}, saved backup)"
  RESP=$(curl -s --max-time 12 -X PATCH -u "$GUSER:$GPASS" -H 'Content-Type: application/json' \
    -d "{\"webhook_url\":\"$NEWURL\",\"webhook_secret\":\"$SECRET\",\"webhook_events\":\"$EVENTS\",\"webhook_insecure_skip_verify\":false}" \
    "$BASE/devices/$ENC/webhook")
  echo "  now: $(echo "$RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);r=d.get('results',{});print(r.get('webhook_url'))" 2>/dev/null)"
done
echo ""
echo "backups (for revert): $BK"
ls "$BK"
