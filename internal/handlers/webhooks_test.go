package handlers

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"valid with path and query", "https://api.example.com/hook?token=abc", false},
		{"invalid URL", "://not-a-url", true},
		{"empty string", "", true},
		{"ftp scheme", "ftp://example.com/hook", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"no scheme", "example.com/hook", true},
		{"localhost", "http://localhost/hook", true},
		{"localhost uppercase", "http://LOCALHOST/hook", true},
		{"0.0.0.0", "http://0.0.0.0/hook", true},
		{".local suffix", "http://myhost.local/hook", true},
		{".internal suffix", "http://myhost.internal/hook", true},
		{"loopback IPv4", "http://127.0.0.1/hook", true},
		{"loopback IPv6", "http://[::1]/hook", true},
		{"private 10.x", "http://10.0.0.1/hook", true},
		{"private 192.168.x", "http://192.168.1.1/hook", true},
		{"private 172.16.x", "http://172.16.0.1/hook", true},
		{"unspecified 0.0.0.0", "http://0.0.0.0/hook", true},
		{"link-local IPv4", "http://169.254.1.1/hook", true},
		{"empty hostname", "https:///hook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err, "expected error for URL: %s", tt.url)
			} else {
				assert.NoError(t, err, "unexpected error for URL: %s", tt.url)
			}
		})
	}
}

func TestSSRFSafeDialer_BlocksLoopback(t *testing.T) {
	t.Parallel()

	dialer := SSRFSafeDialer()

	_, err := dialer(t.Context(), "tcp", "127.0.0.1:80")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

func TestSSRFSafeDialer_BlocksPrivateIP(t *testing.T) {
	t.Parallel()

	dialer := SSRFSafeDialer()

	_, err := dialer(t.Context(), "tcp", "10.0.0.1:443")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

func TestSSRFSafeDialer_BlocksLinkLocal(t *testing.T) {
	t.Parallel()

	dialer := SSRFSafeDialer()

	_, err := dialer(t.Context(), "tcp", "169.254.1.1:80")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

func TestSSRFSafeDialer_InvalidAddr(t *testing.T) {
	t.Parallel()

	dialer := SSRFSafeDialer()

	_, err := dialer(t.Context(), "tcp", "not-valid-addr")
	assert.Error(t, err)
}

func TestSSRFSafeDialer_RecognizesPrivateIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		priv bool
	}{
		{"loopback", "127.0.0.1", true},
		{"private 10", "10.0.0.1", true},
		{"private 192.168", "192.168.0.1", true},
		{"private 172.16", "172.16.0.1", true},
		{"link-local", "169.254.1.1", true},
		{"link-local multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
		{"public", "8.8.8.8", false},
		{"public 2", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "failed to parse IP: %s", tt.ip)

			isBlocked := ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
				ip.IsLinkLocalMulticast() || ip.IsUnspecified()
			assert.Equal(t, tt.priv, isBlocked, "IP %s privacy check mismatch", tt.ip)
		})
	}
}
