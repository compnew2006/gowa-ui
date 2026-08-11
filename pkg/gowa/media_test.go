package gowa_test

import (
	"testing"

	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
)

// TestURLMatchesBase locks the SSRF gate used before fetching a webhook-supplied
// media URL (gap #7): only URLs on the GOWA instance's own host:port are
// allowed, so an attacker-controlled URL in a signed webhook can neither leak
// Basic Auth nor be fetched as an SSRF vector.
func TestURLMatchesBase(t *testing.T) {
	const base = "http://gowa.corp:3000"

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"same host:port", "http://gowa.corp:3000/statics/media/x.jpg", true},
		// Host match is origin-scoped (host:port); a path-only difference is fine.
		{"same host different path", "http://gowa.corp:3000/other/y.png", true},
		// Different port → different origin (would hit a different service).
		{"different port", "http://gowa.corp:8080/x.jpg", false},
		// External host → blocked (the core SSRF case).
		{"external host", "http://attacker.example/x.jpg", false},
		// Loopback / private IPs are not the configured base host.
		{"localhost", "http://127.0.0.1/x.jpg", false},
		{"metadata service", "http://169.254.169.254/latest/meta-data", false},
		// Relative / non-absolute URLs are not fetchable as-is → false (callers
		// must resolve them against baseURL themselves).
		{"relative path", "/statics/media/x.jpg", false},
		{"bare path", "statics/media/x.jpg", false},
		// Subdomain of the base host is a DIFFERENT host → blocked.
		{"subdomain", "http://evil.gowa.corp:3000/x.jpg", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, gowa.URLMatchesBase(tc.url, base),
				"URLMatchesBase(%q, %q)", tc.url, base)
		})
	}
}
