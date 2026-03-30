import requests

def test_postapiauthloginwithinvalidcredentials():
    base_url = "http://localhost:8080"
    login_url = f"{base_url}/api/auth/login"
    payload = {
        "authType": "basic token",
        "credential": {
            "username": "admin@admin.com",
            "password": "admin"
        }
    }
    headers = {
        "Content-Type": "application/json"
    }
    timeout = 30
    try:
        response = requests.post(login_url, json=payload, headers=headers, timeout=timeout)
    except requests.RequestException as e:
        assert False, f"Request failed: {e}"

    # Validate status code
    assert response.status_code == 401, f"Expected 401 Unauthorized, got {response.status_code}"

    # Validate error message in response JSON
    try:
        resp_json = response.json()
    except ValueError:
        assert False, "Response is not valid JSON"

    assert "error" in resp_json or "message" in resp_json, "Error message not found in response"

    # Validate no auth cookies with 'whm_' prefix are set
    cookies_keys = [cookie.name for cookie in response.cookies]
    auth_cookies = [ck for ck in cookies_keys if ck.startswith('whm_')]
    assert len(auth_cookies) == 0, "Auth cookies with 'whm_' prefix should not be set on failed login"

test_postapiauthloginwithinvalidcredentials()