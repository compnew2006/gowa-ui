#!/usr/bin/env python3
# Diagnose GOWA device webhook state vs gowa-ui, on the VPS.
import json, subprocess, urllib.parse, sys

GUSER="gowa_main_1ccb48ee"; GPASS="6l9k6CVR1xftU2h4QLgxtyMS"
BASE="https://gowa.ofuqalmadenah.com"
DEVS=["Print-Labn-4614","Adv-1926","4395 - عماد","تصميم عسير -4625","عائشة -8930","امين -4210","محمد ابراهيم -6178"]

def gowa_get(path):
    r = subprocess.run(["curl","-s","--max-time","10","-u",f"{GUSER}:{GPASS}",BASE+path],
                       capture_output=True,text=True)
    try: return json.loads(r.stdout)
    except: return {"raw": r.stdout[:120]}

print("=== GOWA device webhook configs ===")
for d in DEVS:
    enc=urllib.parse.quote(d,safe="")
    cfg=gowa_get(f"/devices/{enc}/webhook")
    res=cfg.get("results",{})
    sec=res.get("webhook_secret","") or ""
    print(f"  {d!r:40} url={(res.get('webhook_url') or '(none)')[:55]}  secret_len={len(sec)}  events={res.get('webhook_events','')!r}")

print("\n=== gowa-ui inbox (events received + HMAC passed) ===")
r=subprocess.run(["bash","-lc","su - postgres -c \"psql -d gowa_ui -tAc \\\"select status,event,count(*) from gowa_webhook_events group by status,event order by status;\\\"\""],capture_output=True,text=True)
print(r.stdout or "(empty)")
