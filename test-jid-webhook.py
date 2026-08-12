#!/usr/bin/env python3
# Test: does GOWA accept the JID (ASCII) in the webhook path for an Arabic-id device?
import json, subprocess, urllib.parse
GUSER="gowa_main_1ccb48ee"; GPASS="6l9k6CVR1xftU2h4QLgxtyMS"; BASE="https://gowa.ofuqalmadenah.com"
JID="966594374625@s.whatsapp.net"

def show(label, args):
    r=subprocess.run(args, capture_output=True, text=True)
    print(f"  {label} -> {r.stdout[:220]}")

print("=== GET /devices/<jid>/webhook (jid url-encoded) ===")
show("enc-jid", ["curl","-s","--max-time","10","-u",f"{GUSER}:{GPASS}", f"{BASE}/devices/{urllib.parse.quote(JID, safe='@')}/webhook"])
print("=== GET /devices/<jid>/webhook (jid raw) ===")
show("raw-jid", ["curl","-s","--max-time","10","-u",f"{GUSER}:{GPASS}", f"{BASE}/devices/{JID}/webhook"])
print("=== GET /devices/<phone>/webhook (bare digits) ===")
show("digits", ["curl","-s","--max-time","10","-u",f"{GUSER}:{GPASS}", f"{BASE}/devices/966594374625/webhook"])
