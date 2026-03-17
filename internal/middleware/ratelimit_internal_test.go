package middleware

import (
	"net"
	"testing"

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
