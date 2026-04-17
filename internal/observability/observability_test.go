package observability

import (
	"net"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func newTestRequest(t *testing.T, path string, remoteIP string) *fastglue.Request {
	t.Helper()

	req := &fasthttp.Request{}
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(path)

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, &net.TCPAddr{IP: net.ParseIP(remoteIP), Port: 12345}, nil)
	return &fastglue.Request{RequestCtx: ctx}
}

func TestMetricsHandler_RequiresTokenWhenConfigured(t *testing.T) {
	t.Parallel()

	manager := NewManager(config.ObservabilityConfig{
		EnableMetrics: true,
		AccessToken:   "secret-token",
	}, nil, nil)

	req := newTestRequest(t, "/metrics", "127.0.0.1")

	err := manager.MetricsHandler()(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode())
	assert.Contains(t, string(req.RequestCtx.Response.Body()), "Observability token required")
}

func TestMetricsHandler_AllowsLoopbackWithoutToken(t *testing.T) {
	t.Parallel()

	manager := NewManager(config.ObservabilityConfig{
		EnableMetrics: true,
	}, nil, nil)

	manager.observeRequest(fasthttp.MethodGet, "/api/analytics", fasthttp.StatusOK, 75*time.Millisecond)

	req := newTestRequest(t, "/metrics", "127.0.0.1")

	err := manager.MetricsHandler()(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, req.RequestCtx.Response.StatusCode())
	assert.Contains(t, string(req.RequestCtx.Response.Body()), `whatomate_http_requests_total{method="GET",route_group="analytics",status_class="2xx"} 1`)
}

func TestMetricsHandler_BlocksNonLoopbackWithoutToken(t *testing.T) {
	t.Parallel()

	manager := NewManager(config.ObservabilityConfig{
		EnableMetrics: true,
	}, nil, nil)

	req := newTestRequest(t, "/metrics", "10.0.0.5")

	err := manager.MetricsHandler()(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, req.RequestCtx.Response.StatusCode())
	assert.Contains(t, string(req.RequestCtx.Response.Body()), "loopback-only")
}
