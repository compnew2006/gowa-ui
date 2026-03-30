import requests

BASE_URL = "http://localhost:8080"
LOGIN_URL = f"{BASE_URL}/api/auth/login"
LOGOUT_URL = f"{BASE_URL}/api/auth/logout"
CONTACTS_URL = f"{BASE_URL}/api/contacts"
TIMEOUT = 30

def test_postapiauthlogoutwithvalidauthcookie():
    session = requests.Session()
    # Login payload with specified email and password
    login_payload = {
        "email": "admin@admin.com",
        "password": "adminpassword12"
    }
    try:
        # Step 1: Authenticate to receive auth cookie and CSRF cookie
        login_response = session.post(
            LOGIN_URL,
            json=login_payload,
            timeout=TIMEOUT
        )
        assert login_response.status_code == 200, f"Login failed with status {login_response.status_code}"

        # Extract auth cookie with prefix 'whm_'
        whm_cookies = {k: v for k, v in session.cookies.items() if k.startswith("whm_")}
        assert len(whm_cookies) > 0, "Auth cookie with 'whm_' prefix not found after login"

        # Extract CSRF token from whm_csrf cookie
        csrf_token = session.cookies.get("whm_csrf")
        assert csrf_token, "CSRF token 'whm_csrf' cookie not found"
        
        # Extract Refresh token from whm_refresh cookie
        refresh_token = session.cookies.get("whm_refresh")
        assert refresh_token, "Refresh token 'whm_refresh' cookie not found"

        # Step 2: Logout with valid auth cookie and CSRF token as header, plus refresh token in body
        logout_response = session.post(
            LOGOUT_URL,
            headers={"X-CSRF-Token": csrf_token},
            json={"refresh_token": refresh_token},
            timeout=TIMEOUT
        )
        assert logout_response.status_code == 200, f"Logout failed with status {logout_response.status_code}: {logout_response.text}"

        # Step 3: Attempt access to protected resource GET /api/contacts with cleared auth
        contacts_response = session.get(CONTACTS_URL, timeout=TIMEOUT)
        assert contacts_response.status_code == 401, f"Access to protected resource should be denied after logout, got {contacts_response.status_code}"

    finally:
        session.close()

test_postapiauthlogoutwithvalidauthcookie()
