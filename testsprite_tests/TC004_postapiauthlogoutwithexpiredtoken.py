import requests

BASE_URL = "http://localhost:8080"
LOGIN_URL = f"{BASE_URL}/api/auth/login"
LOGOUT_URL = f"{BASE_URL}/api/auth/logout"
TIMEOUT = 30


def test_postapiauthlogoutwithexpiredtoken():
    session = requests.Session()

    # Step 1: Perform login to get valid auth cookies and csrf token
    login_payload = {
        "email": "admin@admin.com",
        "password": "adminpassword12"
    }
    try:
        login_resp = session.post(
            LOGIN_URL,
            json=login_payload,
            timeout=TIMEOUT,
        )
        # Login should succeed to obtain a valid session first
        assert login_resp.status_code == 200, "Login failed unexpectedly"

        # Extract auth cookies with 'whm_' prefix and CSRF token from 'whm_csrf' cookie
        whm_cookies = {k: v for k, v in session.cookies.items() if k.startswith("whm_")}
        assert whm_cookies, "No auth cookies with 'whm_' prefix found after login"
        csrf_token = session.cookies.get("whm_csrf")
        assert csrf_token, "CSRF token 'whm_csrf' cookie not found after login"

        # Step 2: Overwrite session cookies to simulate an expired/invalid token by clearing them
        for cookie_name in list(session.cookies.keys()):
            if cookie_name.startswith("whm_"):
                session.cookies.set(cookie_name, "invalid_or_expired_value")

        # Restore the valid CSRF cookie so it passes the CSRF middleware and hits the Logout token validator
        session.cookies.set("whm_csrf", csrf_token)

        # Step 3: Attempt logout with this expired/invalid token/session
        headers = {
            # Send original (now invalid) CSRF token to attempt logout
            "X-CSRF-TOKEN": csrf_token,
            "Content-Type": "application/json",
        }
        logout_resp = session.post(
            LOGOUT_URL,
            headers=headers,
            timeout=TIMEOUT,
        )

        # Validate response is 401 Unauthorized for expired/invalid token
        assert logout_resp.status_code == 401, (
            f"Expected 401 Unauthorized for expired/invalid token logout, got {logout_resp.status_code}"
        )
    finally:
        session.close()


test_postapiauthlogoutwithexpiredtoken()
