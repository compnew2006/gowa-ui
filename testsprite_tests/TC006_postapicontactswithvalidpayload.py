import requests

BASE_URL = "http://localhost:8080"

def test_post_api_contacts_with_valid_payload():
    session = requests.Session()
    timeout = 30
    contact_id = None
    headers = {}
    try:
        # Login to get auth cookies and CSRF token
        login_payload = {
            "email": "admin@admin.com",
            "password": "adminpassword12"
        }
        login_resp = session.post(f"{BASE_URL}/api/auth/login", json=login_payload, timeout=timeout)
        assert login_resp.status_code == 200, f"Login failed with status code {login_resp.status_code}"

        # Extract auth cookie (starting with 'whm_')
        auth_cookies = {k: v for k, v in session.cookies.items() if k.startswith("whm_")}
        assert auth_cookies, "Authentication cookie with 'whm_' prefix not found after login"

        # Extract CSRF token from cookie named 'whm_csrf'
        csrf_token = session.cookies.get("whm_csrf")
        assert csrf_token, "CSRF token cookie 'whm_csrf' not found after login"

        # Prepare valid minimal payload for creating a contact
        import time
        contact_payload = {
            "name": f"Test User {int(time.time())}",
            "phone_number": f"+1{int(time.time())}"[:12]
        }

        headers = {
            "X-CSRF-Token": csrf_token
        }

        # POST /api/contacts to create the contact
        create_resp = session.post(f"{BASE_URL}/api/contacts", json=contact_payload, headers=headers, timeout=timeout)
        assert create_resp.status_code == 200, f"Expected 200 OK, got {create_resp.status_code}"

        created_contact = create_resp.json()
        assert isinstance(created_contact, dict), "Response JSON is not a dictionary"
        
        # Unwrap the actual resource from the data attribute
        contact_data = created_contact.get("data", {})

        # Expect at least the contact ID to confirm creation
        contact_id = contact_data.get("id") or contact_data.get("_id")
        assert contact_id, "Created contact resource missing 'id' field"

    finally:
        # Cleanup: delete the created contact if contact_id exists
        if contact_id:
            try:
                session.delete(f"{BASE_URL}/api/contacts/{contact_id}", headers=headers, timeout=timeout)
            except Exception:
                pass

test_post_api_contacts_with_valid_payload()
