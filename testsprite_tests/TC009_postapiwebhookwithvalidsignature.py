import requests
import hashlib
import hmac
import json
import time

BASE_URL = "http://localhost:8080"
LOGIN_URL = f"{BASE_URL}/api/auth/login"
WEBHOOK_URL = f"{BASE_URL}/api/webhook"
CONTACTS_URL = f"{BASE_URL}/api/contacts"
TIMEOUT = 10

def test_post_api_webhook_with_valid_signature():
    with requests.Session() as session:
        # Step 1: Login to get a valid session and CSRF token
        login_payload = {
            "email": "admin@admin.com",
            "password": "adminpassword12"
        }
        login_resp = session.post(LOGIN_URL, json=login_payload, timeout=TIMEOUT)
        assert login_resp.status_code == 200, f"Login failed: {login_resp.text}"
        
        cookies = session.cookies.get_dict()
        assert "whm_csrf" in cookies, "CSRF token cookie 'whm_csrf' not found"
        csrf_token = cookies["whm_csrf"]

        # Step 2: Prepare a Meta-compliant WhatsApp webhook payload
        phone_id = "TEST_PHONE"
        waba_id = "TEST_WABA"
        wa_id = "12345550123"
        
        # Use dynamic message ID and timestamp
        timestamp = int(time.time())
        msg_id = f"wamid.ts{timestamp}"
        
        webhook_payload = {
            "object": "whatsapp_business_account",
            "entry": [{
                "id": waba_id,
                "changes": [{
                    "field": "messages",
                    "value": {
                        "messaging_product": "whatsapp",
                        "metadata": {
                            "display_phone_number": "1234567890",
                            "phone_number_id": phone_id
                        },
                        "contacts": [{
                            "profile": {"name": "Test Sprite User"},
                            "wa_id": wa_id
                        }],
                        "messages": [{
                            "from": wa_id,
                            "id": msg_id,
                            "timestamp": str(timestamp),
                            "text": {"body": "Hello from TestSprite Verification"},
                            "type": "text"
                        }]
                    }
                }]
            }]
        }
        
        # Use compact JSON for signature consistency
        payload_bytes = json.dumps(webhook_payload, separators=(',', ':')).encode("utf-8")

        # Step 3: Compute valid signature using the secret we put in the DB
        webhook_secret = b"tester_secret"
        signature = "sha256=" + hmac.new(webhook_secret, payload_bytes, hashlib.sha256).hexdigest()

        headers = {
            "Content-Type": "application/json",
            "X-Hub-Signature-256": signature,
            "X-CSRF-Token": csrf_token
        }

        # Step 4: POST the webhook payload
        webhook_resp = session.post(WEBHOOK_URL, data=payload_bytes, headers=headers, timeout=TIMEOUT)
        assert webhook_resp.status_code == 200, f"Webhook POST failed: {webhook_resp.status_code} {webhook_resp.text}"

        # Step 5: Wait for background processing (contact creation)
        time.sleep(3)
        
        contacts_resp = session.get(CONTACTS_URL)
        assert contacts_resp.status_code == 200, f"Contacts fetch failed: {contacts_resp.text}"
        contacts_data = contacts_resp.json().get("data", {})
        
        # Search for the phone number
        items = contacts_data.get("contacts") or contacts_data.get("items") or []
        matched_contacts = [c for c in items if wa_id in str(c.get("phone_number", "")) or wa_id in str(c.get("wa_id", ""))]
        
        if not matched_contacts:
            # Try to list all if first check fails
            print(f"DEBUG: Found contacts in DB: {[c.get('phone_number') for c in items]}")
            
        assert matched_contacts, f"Expected contact {wa_id} from webhook not found in contacts list"
        print(f"Test TC009 Passed: Webhook message verified and contact {wa_id} found.")

if __name__ == "__main__":
    test_post_api_webhook_with_valid_signature()
