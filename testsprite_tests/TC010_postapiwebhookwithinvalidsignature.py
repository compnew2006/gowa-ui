import requests
import hashlib
import hmac
import json

BASE_URL = "http://localhost:8080"
LOGIN_URL = f"{BASE_URL}/api/auth/login"
WEBHOOK_URL = f"{BASE_URL}/api/webhook"
TIMEOUT = 10

def test_post_api_webhook_with_invalid_signature():
    with requests.Session() as session:
        # Step 1: Login
        login_payload = {
            "email": "admin@admin.com",
            "password": "adminpassword12"
        }
        login_resp = session.post(LOGIN_URL, json=login_payload, timeout=TIMEOUT)
        assert login_resp.status_code == 200, f"Login failed: {login_resp.text}"
        
        cookies = session.cookies.get_dict()
        assert "whm_csrf" in cookies, "CSRF token cookie 'whm_csrf' not found"
        csrf_token = cookies["whm_csrf"]

        # Step 2: Prepare a payload
        payload = {"object":"whatsapp_business_account","entry":[]}
        payload_bytes = json.dumps(payload, separators=(',', ':')).encode("utf-8")

        # Step 3: Use an INVALID signature
        headers = {
            "Content-Type": "application/json",
            "X-Hub-Signature-256": "sha256=invalidhashvaluehere",
            "X-CSRF-Token": csrf_token
        }

        # Step 4: POST the webhook payload
        webhook_resp = session.post(WEBHOOK_URL, data=payload_bytes, headers=headers, timeout=TIMEOUT)
        
        # Correct assertion for signature failure
        assert webhook_resp.status_code == 403, f"Expected 403 Forbidden for invalid signature, got {webhook_resp.status_code}"
        print(f"Test TC010 Passed: Invalid signature correctly rejected with 403.")

if __name__ == "__main__":
    test_post_api_webhook_with_invalid_signature()
