#!/usr/bin/env bash
# REVERT: restores each GOWA device's webhook to its pre-switch config saved in
# /root/gowa-webhook-backup/. Run ON the VPS after testing.
set -uo pipefail
GUSER=gowa_main_1ccb48ee; GPASS=6l9k6CVR1xftU2h4QLgxtyMS
BASE=https://gowa.ofuqalmadenah.com
BK=/root/gowa-webhook-backup
echo "Reverting GOWA device webhooks to pre-switch config..."
for f in "$BK"/*.json; do
  ENC=$(basename "$f" .json)
  read -r URL SEC EV < <(python3 -c "
import json,sys
d=json.load(open('$f')).get('results',{})
print(d.get('webhook_url','') or '', (d.get('webhook_secret','') or '')[:0], d.get('webhook_events','') or '')
" 2>/dev/null)
  # re-extract properly (space-split fails on secrets); use python for the PATCH body
  body=$(python3 -c "
import json
d=json.load(open('$f')).get('results',{})
print(json.dumps({'webhook_url':d.get('webhook_url','') or '','webhook_secret':d.get('webhook_secret','') or '','webhook_events':d.get('webhook_events','') or '','webhook_insecure_skip_verify':False}))
" 2>/dev/null)
  [ -z "$body" ] && { echo "  $ENC: no valid backup (skip)"; continue; }
  CODE=$(curl -s --max-time 12 -o /dev/null -w "%{http_code}" -X PATCH -u "$GUSER:$GPASS" \
    -H 'Content-Type: application/json' -d "$body" "$BASE/devices/$ENC/webhook")
  echo "  $ENC -> reverted (HTTP $CODE)"
done
echo "Done. Devices restored to their pre-switch webhook config."
