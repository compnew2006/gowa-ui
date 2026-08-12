#!/usr/bin/env python3
# Sets the webhook on the 5 empty GOWA devices to point to gowa-ui, using each
# account's decrypted gowa-ui secret (guarantees HMAC match). Run ON the VPS.
import json, subprocess, urllib.parse, re, base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

cfg = open("/opt/gowa-ui/config.toml").read()
KEY = re.search(r'^encryption_key\s*=\s*"([^"]+)"', cfg, re.M).group(1)[:32].encode()
def dec(v):
    if not v or not v.startswith("enc:"): return v
    d = base64.b64decode(v[4:]); return AESGCM(KEY).decrypt(d[:12], d[12:], None).decode()

GUSER="gowa_main_1ccb48ee"; GPASS="6l9k6CVR1xftU2h4QLgxtyMS"; BASE="https://gowa.ofuqalmadenah.com"
NEWURL="http://31.97.192.53:8081/api/gowa/webhook"
EVENTS="message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer"
NEED=["4395 - عماد","تصميم عسير -4625","عائشة -8930","امين -4210","محمد ابراهيم -6178"]

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
    rr=subprocess.run(["curl","-s","--max-time","12","-X","PATCH","-u",f"{GUSER}:{GPASS}","-H","Content-Type: application/json","-d",body,f"{BASE}/devices/{e}/webhook"],capture_output=True,text=True)
    try:
        res=json.loads(rr.stdout).get("results",{})
        print(f"  {dev:22} -> url={res.get('webhook_url')}  secret_len={len(res.get('webhook_secret','') or '')}")
    except Exception:
        print(f"  {dev:22} -> ERROR raw={rr.stdout[:100]}")
print("\ndone.")
