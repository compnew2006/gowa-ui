#!/usr/bin/env python3
# Test: does GOWA accept the JID (ASCII) in the webhook path for an Arabic-id device?
import base64, os, urllib.error, urllib.parse, urllib.request, sys

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
JID="966594374625@s.whatsapp.net"
AUTH = "Basic " + base64.b64encode(f"{GUSER}:{GPASS}".encode()).decode()

def show(label, path):
    request=urllib.request.Request(f"{BASE}{path}", headers={"Authorization":AUTH})
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            status=response.status
    except urllib.error.HTTPError as error:
        status=error.code
    except (urllib.error.URLError, TimeoutError):
        status="unavailable"
    print(f"  {label} -> HTTP {status}")

print("=== GET /devices/<jid>/webhook (jid url-encoded) ===")
show("enc-jid", f"/devices/{urllib.parse.quote(JID, safe='@')}/webhook")
print("=== GET /devices/<jid>/webhook (jid raw) ===")
show("raw-jid", f"/devices/{JID}/webhook")
print("=== GET /devices/<phone>/webhook (bare digits) ===")
show("digits", "/devices/966594374625/webhook")
