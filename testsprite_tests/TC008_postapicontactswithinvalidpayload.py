import requests

BASE_URL = "http://localhost:8080"
LOGIN_ENDPOINT = "/api/auth/login"
CONTACTS_ENDPOINT = "/api/contacts"
TIMEOUT = 30

def test_postapicontactswithinvalidpayload():
    session = requests.Session()
    login_payload = {
        "email": "admin@admin.com",
        "password": "adminpassword12"
    }
    # Login to get auth cookies and CSRF token
    login_resp = session.post(
        BASE_URL + LOGIN_ENDPOINT,
        json=login_payload,
        timeout=TIMEOUT
    )
    assert login_resp.status_code == 200, f"Login failed, status code {login_resp.status_code}"
    
    # Check auth cookies with 'whm_' prefix exist
    whm_cookies = {k: v for k, v in session.cookies.items() if k.startswith('whm_')}
    assert whm_cookies, "Auth cookies with 'whm_' prefix not found in login response"
    
    # Extract CSRF token from 'whm_csrf' cookie
    csrf_token = session.cookies.get('whm_csrf')
    assert csrf_token, "CSRF token 'whm_csrf' cookie not found"

    # Prepare invalid payload missing 'phone' field (required)
    invalid_contact_payload = {
        "name": "Invalid Contact Test",
        "email": "invalidcontact@example.com"
        # 'phone' field is intentionally missing
    }
    
    headers = {
        "X-CSRF-Token": csrf_token
    }
    
    # POST /api/contacts with invalid payload
    resp = session.post(
        BASE_URL + CONTACTS_ENDPOINT,
        json=invalid_contact_payload,
        headers=headers,
        timeout=TIMEOUT
    )

    # Assert 400 Bad Request and response contains validation errors
    assert resp.status_code == 400, f"Expected 400 Bad Request, got {resp.status_code}"
    try:
        resp_json = resp.json()
    except Exception:
        resp_json = {}
    # Validate presence of validation error message related to missing phone
    error_messages = []
    if isinstance(resp_json, dict):
        # Check common patterns for validation errors
        if "errors" in resp_json and isinstance(resp_json["errors"], list):
            error_messages = resp_json["errors"]
        elif "message" in resp_json:
            error_messages = [resp_json["message"]]
        else:
            error_messages = [str(resp_json)]
    assert any("phone" in str(msg).lower() for msg in error_messages), "Validation error about missing phone field not found in response"


test_postapicontactswithinvalidpayload()