package handlers

import (
	"net/url"
	"testing"
)

// TestValidateWebhookURL_SSRFGuard covers the structural SSRF guard (Fix 1, H1)
// reused by SendTemplateMessage's header_media_url fetch and by the webhook /
// custom-action fetches. It must reject schemes other than http/https and
// block hostnames/IPs that resolve to private/loopback/link-local/unspecified
// ranges, while accepting ordinary public URLs.
//
// This is a pure-function unit test (no HTTP / DB infra) — the runtime
// DNS-rebinding defense is provided by SSRFSafeDialer on a.HTTPClient and is
// exercised indirectly via the network layer, not here.
func TestValidateWebhookURL_SSRFGuard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Scheme allowlist.
		{"ftp scheme rejected", "ftp://example.com/x.png", true},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"javascript scheme rejected", "javascript:alert(1)", true},

		// Loopback / link-local / private IP literals (cloud metadata + localhost).
		{"loopback v4 rejected", "http://127.0.0.1/x.png", true},
		{"loopback v6 rejected", "http://[::1]/x.png", true},
		{"aws metadata rejected", "http://169.254.169.254/latest/meta-data/", true},
		{"private rfc1918 rejected", "http://10.0.0.1/x.png", true},
		{"private rfc1918 rejected 2", "http://192.168.1.1/x.png", true},
		{"private rfc1918 rejected 3", "http://172.16.0.1/x.png", true},

		// Internal hostnames (the function's internal-hostname blocklist).
		{"localhost rejected", "http://localhost/x.png", true},

		// Valid public URLs (regression — must still pass).
		{"public https accepted", "https://cdn.example.com/img.png", false},
		{"public http accepted", "http://example.com/img.png", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateWebhookURL(tc.url)
			if tc.wantErr && err == nil {
				t.Errorf("validateWebhookURL(%q) expected error, got nil", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateWebhookURL(%q) expected no error, got %v", tc.url, err)
			}
			// Sanity: ensure the parsed URL is well-formed for non-empty inputs.
			if _, perr := url.Parse(tc.url); perr != nil && !tc.wantErr {
				t.Errorf("test case URL %q does not parse: %v", tc.url, perr)
			}
		})
	}
}
