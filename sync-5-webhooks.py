#!/usr/bin/env python3
# Sets the webhook on the 5 empty GOWA devices to point to gowa-ui, using each
# account's decrypted gowa-ui secret (guarantees HMAC match). Run ON the VPS.
import json, os, subprocess, urllib.error, urllib.parse, urllib.request, re, base64, sys
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

required = ("GOWA_USER", "GOWA_PASSWORD", "GOWA_BASE_URL")
missing = [name for name in required if not os.environ.get(name)]
if missing:
    sys.exit("Missing required environment variable(s): " + ", ".join(missing))

cfg = open("/opt/gowa-ui/config.toml").read()
KEY = re.search(r'^encryption_key\s*=\s*"([^"]+)"', cfg, re.M).group(1)[:32].encode()
def dec(v):
    if not v or not v.startswith("enc:"): return v
    d = base64.b64decode(v[4:]); return AESGCM(KEY).decrypt(d[:12], d[12:], None).decode()

GUSER = os.environ["GOWA_USER"]
GPASS = os.environ["GOWA_PASSWORD"]
BASE = os.environ["GOWA_BASE_URL"].rstrip("/")
parsed_base = urllib.parse.urlsplit(BASE)
if parsed_base.scheme not in ("http", "https") or not parsed_base.hostname or parsed_base.username or parsed_base.password or parsed_base.query or parsed_base.fragment:
    sys.exit("GOWA_BASE_URL must be an HTTP(S) URL without embedded credentials or query data")
NEWURL="http://31.97.192.53:8081/api/gowa/webhook"
EVENTS="message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer"
NEED=["4395 - عماد","تصميم عسير -4625","عائشة -8930","امين -4210","محمد ابراهيم -6178"]
AUTH = "Basic " + base64.b64encode(f"{GUSER}:{GPASS}".encode()).decode()

# load account secrets from DB
sql='select gowa_device_id,gowa_webhook_secret from whatsapp_accounts;'
r=subprocess.run(["bash","-lc",f'su - postgres -c "psql -d gowa_ui -tAF \'\\t\' -c \\"{sql}\\""'],capture_output=True,text=True)
secrets={}
for line in r.stdout.strip().splitlines():
    parts=line.split("\t")
    if len(parts)==2: secrets[parts[0]]=parts[1]

print("=== set webhook on the 5 empty devices (decrypt gowa-ui secret -> GOWA) ===")
for dev in NEED:
    enc_sec = secrets.get(dev, "")
    plain = dec(enc_sec)
    body=json.dumps({"webhook_url":NEWURL,"webhook_secret":plain,"webhook_events":EVENTS,"webhook_insecure_skip_verify":False})
    e=urllib.parse.quote(dev,safe="")
    try:
        request=urllib.request.Request(
            f"{BASE}/devices/{e}/webhook",
            data=body.encode(),
            headers={"Authorization":AUTH,"Content-Type":"application/json"},
            method="PATCH",
        )
        with urllib.request.urlopen(request, timeout=12) as response:
            res=json.load(response).get("results",{})
        print(f"  {dev:22} -> webhook updated  secret_len={len(res.get('webhook_secret','') or '')}")
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        print(f"  {dev:22} -> ERROR (GOWA returned an invalid response)")
print("\ndone.")
