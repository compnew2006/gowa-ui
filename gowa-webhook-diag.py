#!/usr/bin/env python3
# Diagnose GOWA device webhook state vs gowa-ui, on the VPS.
import base64, json, os, subprocess, urllib.error, urllib.parse, urllib.request, sys

required = ("GOWA_USER", "GOWA_PASSWORD", "GOWA_BASE_URL")
missing = [name for name in required if not os.environ.get(name)]
if missing:
    sys.exit("Missing required environment variable(s): " + ", ".join(missing))
GUSER = os.environ["GOWA_USER"]
GPASS = os.environ["GOWA_PASSWORD"]
BASE = os.environ["GOWA_BASE_URL"].rstrip("/")
parsed_base = urllib.parse.urlsplit(BASE)
if parsed_base.scheme not in ("http", "https") or not parsed_base.hostname or parsed_base.username or parsed_base.password or parsed_base.query or parsed_base.fragment:
    sys.exit("GOWA_BASE_URL must be an HTTP(S) URL without embedded credentials or query data")
DEVS=["Print-Labn-4614","Adv-1926","4395 - عماد","تصميم عسير -4625","عائشة -8930","امين -4210","محمد ابراهيم -6178"]
AUTH = "Basic " + base64.b64encode(f"{GUSER}:{GPASS}".encode()).decode()

def gowa_get(path):
    request = urllib.request.Request(BASE + path, headers={"Authorization": AUTH})
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            return json.load(response)
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return {}

print("=== GOWA device webhook configs ===")
for d in DEVS:
    enc=urllib.parse.quote(d,safe="")
    cfg=gowa_get(f"/devices/{enc}/webhook")
    res=cfg.get("results",{})
    sec=res.get("webhook_secret","") or ""
    print(f"  {d!r:40} webhook_configured={bool(res.get('webhook_url'))}  secret_len={len(sec)}  events={res.get('webhook_events','')!r}")

print("\n=== gowa-ui inbox (events received + HMAC passed) ===")
r=subprocess.run(["bash","-lc","su - postgres -c \"psql -d gowa_ui -tAc \\\"select status,event,count(*) from gowa_webhook_events group by status,event order by status;\\\"\""],capture_output=True,text=True)
print(r.stdout or "(empty)")
