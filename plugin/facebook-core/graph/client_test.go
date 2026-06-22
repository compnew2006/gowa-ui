package graph

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientJSONRequest(t *testing.T) {
	t.Parallel()

	client := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", req.Method)
		}
		return response(http.StatusOK, `{"data":"ok"}`), nil
	})})

	var payload map[string]string
	if err := client.JSONRequest(http.MethodGet, "https://graph.facebook.test/me", nil, &payload); err != nil {
		t.Fatalf("JSONRequest() error = %v", err)
	}
	if payload["data"] != "ok" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestClientGetAndGenericStatusError(t *testing.T) {
	t.Parallel()

	calls := 0
	client := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(http.StatusOK, `{"id":"page-1"}`), nil
		}
		return response(http.StatusForbidden, `{"error":"forbidden"}`), nil
	})})

	payload, err := client.Get("https://graph.facebook.test/page-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if payload["id"] != "page-1" {
		t.Fatalf("payload = %#v", payload)
	}

	var denied map[string]any
	err = client.JSONRequest(http.MethodGet, "https://graph.facebook.test/denied", nil, &denied)
	if err == nil || err.Error() != "Facebook Graph API returned status 403" {
		t.Fatalf("JSONRequest() error = %v", err)
	}
}

func TestResponseErrorWithoutGraphPayload(t *testing.T) {
	t.Parallel()

	err := ResponseError(http.StatusBadGateway, map[string]any{"message": "gateway"})
	if err == nil || err.Error() != "Facebook Graph API returned status 502" {
		t.Fatalf("ResponseError() = %v", err)
	}
}

func TestClientFormPostPreservesGraphErrorDetails(t *testing.T) {
	t.Parallel()

	client := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("content type = %q", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "message=hello" {
			t.Fatalf("body = %q", body)
		}
		return response(http.StatusBadRequest, `{"error":{"message":"bad token","code":190,"error_subcode":460,"type":"OAuthException"}}`), nil
	})})

	payload, err := client.FormPost("https://graph.facebook.test/page/feed", url.Values{"message": []string{"hello"}})
	if payload == nil {
		t.Fatal("expected error payload")
	}
	want := "Facebook Graph API returned status 400: bad token: code=190: subcode=460: type=OAuthException"
	if err == nil || err.Error() != want {
		t.Fatalf("FormPost() error = %v, want %q", err, want)
	}
}

func TestClientJSONPost(t *testing.T) {
	t.Parallel()

	client := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q", got)
		}
		return response(http.StatusOK, `{"id":"message-1"}`), nil
	})})

	payload, err := client.JSONPost("https://graph.facebook.test/me/messages", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("JSONPost() error = %v", err)
	}
	if payload["id"] != "message-1" {
		t.Fatalf("payload = %#v", payload)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
