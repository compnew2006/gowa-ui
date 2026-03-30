import requests

BASE_URL = "http://localhost:8080"
LOGIN_PATH = "/api/auth/login"
CONTACTS_PATH = "/api/contacts"
TIMEOUT = 30

def test_postapiauthloginwithvalidcredentials():
    login_url = BASE_URL + LOGIN_PATH
    contacts_url = BASE_URL + CONTACTS_PATH

    payload = {
        "email": "admin@admin.com",
        "password": "adminpassword12"
    }
    headers = {
        "Content-Type": "application/json"
    }

    session = requests.Session()
    # Login request
    try:
        resp = session.post(login_url, json=payload, headers=headers, timeout=TIMEOUT)
    except requests.RequestException as e:
        assert False, f"Login request failed: {e}"

    # Assert login success status
    assert resp.status_code == 200, f"Expected 200 OK, got {resp.status_code}"

    # Check httpOnly auth cookie with 'whm_' prefix exists
    auth_cookies = [c for c in session.cookies if c.name.startswith("whm_")]
    assert auth_cookies, "Expected auth cookie with prefix 'whm_' not found"

    # Extract CSRF token from 'whm_csrf' cookie
    csrf_token = session.cookies.get("whm_csrf")
    assert csrf_token is not None, "CSRF token cookie 'whm_csrf' not found"

    # Access protected resource with session cookies and CSRF token in header
    protected_headers = {
        "X-CSRF-Token": csrf_token
    }
    try:
        contacts_resp = session.get(contacts_url, headers=protected_headers, timeout=TIMEOUT)
    except requests.RequestException as e:
        assert False, f"GET /api/contacts request failed: {e}"

    # Assert access to protected resource successful
    assert contacts_resp.status_code == 200, f"Expected 200 OK from GET /api/contacts, got {contacts_resp.status_code}"

test_postapiauthloginwithvalidcredentials()