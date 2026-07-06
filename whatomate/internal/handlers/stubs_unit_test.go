package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TestGetMessageAnalytics tests the GetMessageAnalytics stub handler
func TestGetMessageAnalytics(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	err := app.GetMessageAnalytics(req)

	assert.NoError(t, err, "GetMessageAnalytics should succeed")

	// Verify response contains "Not implemented yet"
	responseBody := req.RequestCtx.Response.Body()
	assert.Contains(t, string(responseBody), "Not implemented yet", "Response should indicate not implemented")
	assert.Equal(t, fasthttp.StatusNotImplemented, req.RequestCtx.Response.StatusCode(), "Should return 501 status")
}

// TestGetChatbotAnalytics tests the GetChatbotAnalytics stub handler
func TestGetChatbotAnalytics(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	err := app.GetChatbotAnalytics(req)

	assert.NoError(t, err, "GetChatbotAnalytics should succeed")

	// Verify response contains "Not implemented yet"
	responseBody := req.RequestCtx.Response.Body()
	assert.Contains(t, string(responseBody), "Not implemented yet", "Response should indicate not implemented")
	assert.Equal(t, fasthttp.StatusNotImplemented, req.RequestCtx.Response.StatusCode(), "Should return 501 status")
}

// TestAnalyticsStubsReturnNotImplemented tests that both analytics stubs return 501
func TestAnalyticsStubsReturnNotImplemented(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	tests := []struct {
		name    string
		handler func(*fastglue.Request) error
	}{
		{
			name:    "GetMessageAnalytics",
			handler: app.GetMessageAnalytics,
		},
		{
			name:    "GetChatbotAnalytics",
			handler: app.GetChatbotAnalytics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			err := tt.handler(req)

			assert.NoError(t, err, "Handler should succeed without error")
			assert.Equal(t, fasthttp.StatusNotImplemented, req.RequestCtx.Response.StatusCode(),
				"Handler should return 501 Not Implemented status")
		})
	}
}
