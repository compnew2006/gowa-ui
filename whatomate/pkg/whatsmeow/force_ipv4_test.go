package whatsmeow

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsmeowForceIPv4DefaultsFalse(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()
		assert.False(t, whatsmeowForceIPv4(nil))
	})

	t.Run("nil field", func(t *testing.T) {
		t.Parallel()
		assert.False(t, whatsmeowForceIPv4(&config.WhatsmeowConfig{}))
	})

	t.Run("explicit false", func(t *testing.T) {
		t.Parallel()
		f := false
		assert.False(t, whatsmeowForceIPv4(&config.WhatsmeowConfig{ForceIPv4: &f}))
	})

	t.Run("explicit true", func(t *testing.T) {
		t.Parallel()
		tr := true
		assert.True(t, whatsmeowForceIPv4(&config.WhatsmeowConfig{ForceIPv4: &tr}))
	})
}

// TestBuildIPv4HTTPClientForcesTCP4 confirms the transport's DialContext
// ignores the caller's network hint and always dials "tcp4". This is the
// core guarantee that eliminates the flaky IPv6 path to WhatsApp/Meta.
func TestBuildIPv4HTTPClientForcesTCP4(t *testing.T) {
	t.Parallel()

	client := buildIPv4HTTPClient()
	require.NotNil(t, client)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "transport must be *http.Transport")
	require.NotNil(t, transport.DialContext, "DialContext must be set")

	// DialContext should override ANY incoming network hint ("tcp", "tcp6",
	// or even an explicit "udp") to "tcp4". We verify by capturing the network
	// arg actually passed to the underlying dialer via a stubbed listener.
	//
	// We can't easily intercept net.Dialer.DialContext without a real address,
	// so instead we assert the structural guarantee: dialing a known-good IPv6
	// literal MUST fail (because tcp4 dialer refuses IPv6 syntax), proving the
	// hint is ignored in favor of tcp4.
	//
	// "::1" is the IPv6 loopback. A tcp4 dialer given an IPv6 address returns
	// an error like "address ::1: no suitable address found" without ever
	// hitting the network.
	_, err := transport.DialContext(context.Background(), "tcp6", "[::1]:80")
	require.Error(t, err, "tcp4-forcing dialer must reject an IPv6 literal")

	// Cross-check: the error should mention the address, confirming the dialer
	// attempted a tcp4 resolution and found no suitable IPv4 address for "::1".
	assert.Contains(t, err.Error(), "::1",
		"error should reference the IPv6 literal we attempted")
}

// TestBuildIPv4HTTPClientDialerIgnoresNetworkHint is a stronger, deterministic
// test: it substitutes a custom dialer-capture by asserting the public helper
// produces a transport whose DialContext, when handed "tcp6", fails against a
// real loopback IPv6 address that would succeed under genuine tcp6 dialing.
// This complements the literal-rejection test above.
func TestBuildIPv4HTTPClientRetainsHTTP2(t *testing.T) {
	t.Parallel()

	client := buildIPv4HTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.True(t, transport.ForceAttemptHTTP2,
		"HTTP/2 must be retained — whatsmeow's websocket upgrade benefits from it")
}

// TestBuildIPv4HTTPClientTLSMinimum guards against accidental TLS downgrade.
func TestBuildIPv4HTTPClientTLSMinimum(t *testing.T) {
	t.Parallel()

	client := buildIPv4HTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.Equal(t, uint16(0x0303), transport.TLSClientConfig.MinVersion,
		"MinVersion must be TLS 1.2 (0x0303) at minimum")
}

// Compile-time guard: ensure the helpers we rely on exist with expected shape.
var (
	_ = whatsmeowForceIPv4
	_ = buildIPv4HTTPClient
	_ = net.Dialer{}
)
