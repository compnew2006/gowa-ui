import requests

BASE_URL = "http://localhost:8080"
LOGIN_URL = f"{BASE_URL}/api/auth/login"
CONTACTS_URL = f"{BASE_URL}/api/contacts"
TIMEOUT = 30

def test_getapicontactswithvalidauthcookie():
    session = requests.Session()
    login_payload = {
        "email": "admin@admin.com",
        "password": "adminpassword12"
    }
    try:
        # Login to get auth cookie and csrf token
        login_resp = session.post(LOGIN_URL, json=login_payload, timeout=TIMEOUT)
        assert login_resp.status_code == 200, f"Login failed with status {login_resp.status_code}"
        # Verify presence of 'whm_' prefixed auth cookie
        whm_auth_cookies = {k: v for k, v in session.cookies.items() if k.startswith("whm_")}
        assert whm_auth_cookies, "No 'whm_' prefixed auth cookies found after login"
        # Extract CSRF token from 'whm_csrf' cookie
        csrf_token = session.cookies.get("whm_csrf")
        assert csrf_token, "CSRF token 'whm_csrf' cookie not found after login"

        headers = {
            "X-CSRF-Token": csrf_token
        }
        # GET /api/contacts with valid auth cookie and CSRF token
        contacts_resp = session.get(CONTACTS_URL, headers=headers, timeout=TIMEOUT)
        assert contacts_resp.status_code == 200, f"GET /api/contacts failed with status {contacts_resp.status_code}"

        # Validate response JSON structure: paginated list including aggregated unread counts
        data = contacts_resp.json()
        assert "data" in data, "'data' wrapper missing in response"
    finally:
        # Logout to clear session cookies if login succeeded
        if any(k.startswith("whm_") for k in session.cookies.keys()):
            logout_resp = session.post(f"{BASE_URL}/api/auth/logout", headers={"X-CSRF-Token": session.cookies.get("whm_csrf")}, timeout=TIMEOUT)
            # logout_resp may be 200 or 401 if session already invalidated, no assertion needed here

test_getapicontactswithvalidauthcookie()