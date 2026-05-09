package middleware

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func newRateLimitTestRequest(remoteAddr string, headers map[string]string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	for key, value := range headers {
		ctx.Request.Header.Set(key, value)
	}
	if remoteAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", remoteAddr)
		if err == nil {
			ctx.SetRemoteAddr(addr)
		}
	}
	return &fastglue.Request{RequestCtx: ctx}
}

func TestExtractClientIP_TrustProxyRequiresTrustedPeer(t *testing.T) {
	t.Parallel()

	req := newRateLimitTestRequest("198.51.100.10:4321", map[string]string{
		"X-Forwarded-For": "203.0.113.15",
	})

	assert.Equal(t, "198.51.100.10", extractClientIP(req, true))
}

func TestExtractClientIP_TrustProxyUsesForwardedHeadersFromPrivatePeer(t *testing.T) {
	t.Parallel()

	req := newRateLimitTestRequest("10.0.0.10:4321", map[string]string{
		"X-Forwarded-For": "203.0.113.15, 10.0.0.10",
	})

	assert.Equal(t, "203.0.113.15", extractClientIP(req, true))
}

func TestExtractClientIP_TrustProxyFallsBackOnInvalidForwardedHeader(t *testing.T) {
	t.Parallel()

	req := newRateLimitTestRequest("10.0.0.10:4321", map[string]string{
		"X-Forwarded-For": "not-an-ip",
		"X-Real-IP":       "198.51.100.44",
	})

	assert.Equal(t, "198.51.100.44", extractClientIP(req, true))
}

func TestExtractClientIP_TrustProxyUsesRemoteAddrWhenHeadersMissing(t *testing.T) {
	t.Parallel()

	req := newRateLimitTestRequest("10.0.0.10:4321", nil)

	assert.Equal(t, "10.0.0.10", extractClientIP(req, true))
}

func TestAPIKeyRateLimiting_BlocksAfterMaxFailures(t *testing.T) {
	apiKeyFailureLimiter = sync.Map{}
	defer func() { apiKeyFailureLimiter = sync.Map{} }()

	ip := "192.168.1.100"

	assert.False(t, isAPIKeyRateLimited(ip), "should not be limited initially")

	for i := 0; i < apiKeyAuthMaxFailures; i++ {
		recordAPIKeyFailure(ip)
	}

	assert.True(t, isAPIKeyRateLimited(ip), "should be limited after max failures")
}

func TestAPIKeyRateLimiting_DifferentIPsIndependent(t *testing.T) {
	apiKeyFailureLimiter = sync.Map{}
	defer func() { apiKeyFailureLimiter = sync.Map{} }()

	ip1 := "192.168.1.100"
	ip2 := "10.0.0.1"

	for i := 0; i < apiKeyAuthMaxFailures; i++ {
		recordAPIKeyFailure(ip1)
	}

	assert.True(t, isAPIKeyRateLimited(ip1), "ip1 should be limited")
	assert.False(t, isAPIKeyRateLimited(ip2), "ip2 should not be limited")
}

func TestAPIKeyRateLimiting_WindowExpiry(t *testing.T) {
	apiKeyFailureLimiter = sync.Map{}
	defer func() { apiKeyFailureLimiter = sync.Map{} }()

	ip := "192.168.1.100"

	entry := &apiKeyFailureEntry{count: apiKeyAuthMaxFailures, expiresAt: time.Now().Add(-1 * time.Second)}
	apiKeyFailureLimiter.Store(ip, entry)

	assert.False(t, isAPIKeyRateLimited(ip), "should not be limited after window expires")

	recordAPIKeyFailure(ip)
	assert.False(t, isAPIKeyRateLimited(ip), "should reset count after expired window")
}
