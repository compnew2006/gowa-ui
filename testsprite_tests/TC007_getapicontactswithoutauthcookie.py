import requests

def test_get_api_contacts_without_auth_cookie():
    base_url = "http://localhost:8080"
    url = f"{base_url}/api/contacts"
    try:
        response = requests.get(url, timeout=30)
    except requests.RequestException as e:
        assert False, f"Request failed: {e}"

    assert response.status_code == 401, f"Expected 401 Unauthorized, got {response.status_code}"

test_get_api_contacts_without_auth_cookie()