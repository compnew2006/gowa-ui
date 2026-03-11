package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// TestGetAppConfig_MetaProvider tests config with Meta provider
func TestGetAppConfig_MetaProvider(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	// Set provider to meta
	app.Config.WhatsApp.Provider = "meta"

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetAppConfig(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.AppConfigResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Equal(t, "meta", resp.WhatsAppProvider)
	assert.True(t, resp.Features.Templates, "Templates should be enabled for Meta")
	assert.True(t, resp.Features.Flows, "Flows should be enabled for Meta")
	assert.True(t, resp.Features.Catalog, "Catalog should be enabled for Meta")
	assert.True(t, resp.Features.BusinessProfile, "Business profile should be enabled for Meta")
	assert.True(t, resp.Features.Campaigns, "Campaigns should be enabled")
	assert.True(t, resp.Features.MetaInsights, "Meta insights should be enabled for Meta")
}

// TestGetAppConfig_WhatsmeowProvider tests config with Whatsmeow provider
func TestGetAppConfig_WhatsmeowProvider(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	// Set provider to whatsmeow
	app.Config.WhatsApp.Provider = "whatsmeow"

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetAppConfig(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.AppConfigResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Equal(t, "whatsmeow", resp.WhatsAppProvider)
	assert.False(t, resp.Features.Templates, "Templates should be disabled for Whatsmeow")
	assert.False(t, resp.Features.Flows, "Flows should be disabled for Whatsmeow")
	assert.False(t, resp.Features.Catalog, "Catalog should be disabled for Whatsmeow")
	assert.False(t, resp.Features.BusinessProfile, "Business profile should be disabled for Whatsmeow")
	assert.True(t, resp.Features.Campaigns, "Campaigns should be enabled for Whatsmeow")
	assert.False(t, resp.Features.MetaInsights, "Meta insights should be disabled for Whatsmeow")
}

// TestGetAppConfig_EmptyProvider_defaultsToMeta tests config with empty provider defaults to meta
func TestGetAppConfig_EmptyProvider_defaultsToMeta(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	// Set provider to empty string
	app.Config.WhatsApp.Provider = ""

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetAppConfig(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.AppConfigResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Equal(t, "meta", resp.WhatsAppProvider, "Should default to meta when provider is empty")
	assert.True(t, resp.Features.Templates, "Templates should be enabled when defaulted to Meta")
}

// TestGetAppConfig_ResponseStructure tests the response structure
func TestGetAppConfig_ResponseStructure(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "meta"

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetAppConfig(req)
	require.NoError(t, err)

	var resp handlers.AppConfigResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)

	// Verify all required fields are present
	assert.NotEmpty(t, resp.WhatsAppProvider, "WhatsAppProvider should not be empty")
	assert.Contains(t, resp, "Features", "Response should contain Features field")

	// Verify Features structure
	assert.Contains(t, resp.Features, "Templates")
	assert.Contains(t, resp.Features, "Flows")
	assert.Contains(t, resp.Features, "Catalog")
	assert.Contains(t, resp.Features, "BusinessProfile")
	assert.Contains(t, resp.Features, "Campaigns")
	assert.Contains(t, resp.Features, "MetaInsights")
}

// TestGetAppConfig_JSONSerialization tests that response can be serialized to JSON
func TestGetAppConfig_JSONSerialization(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "meta"

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetAppConfig(req)
	require.NoError(t, err)

	// Get response body
	body := testutil.GetResponseBody(req)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(body, &parsed)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify envelope structure
	assert.Contains(t, parsed, "status")
	assert.Contains(t, parsed, "data")

	data := parsed["data"].(map[string]interface{})
	assert.Contains(t, data, "whatsapp_provider")
	assert.Contains(t, data, "features")
}

// TestGetAppConfig_FeatureFlagsConsistency tests that Meta-only features are consistently enabled/disabled
func TestGetAppConfig_FeatureFlagsConsistency(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	testCases := []struct {
		provider           string
		templates          bool
		flows              bool
		catalog            bool
		businessProfile    bool
		campaigns          bool
		metaInsights       bool
	}{
		{
			provider:         "meta",
			templates:        true,
			flows:            true,
			catalog:          true,
			businessProfile:  true,
			campaigns:        true,
			metaInsights:     true,
		},
		{
			provider:         "whatsmeow",
			templates:        false,
			flows:            false,
			catalog:          false,
			businessProfile:  false,
			campaigns:        true, // Campaigns work for both
			metaInsights:     false,
		},
		{
			provider:         "", // Empty should default to meta
			templates:        true,
			flows:            true,
			catalog:          true,
			businessProfile:  true,
			campaigns:        true,
			metaInsights:     true,
		},
	}

	for _, tc := range testCases {
		app.Config.WhatsApp.Provider = tc.provider

		req := testutil.NewJSONRequest(t, nil)
		err := app.GetAppConfig(req)
		require.NoError(t, err)

		var resp handlers.AppConfigResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		assert.Equal(t, tc.templates, resp.Features.Templates, "Templates flag mismatch for provider %s", tc.provider)
		assert.Equal(t, tc.flows, resp.Features.Flows, "Flows flag mismatch for provider %s", tc.provider)
		assert.Equal(t, tc.catalog, resp.Features.Catalog, "Catalog flag mismatch for provider %s", tc.provider)
		assert.Equal(t, tc.businessProfile, resp.Features.BusinessProfile, "Business profile flag mismatch for provider %s", tc.provider)
		assert.Equal(t, tc.campaigns, resp.Features.Campaigns, "Campaigns flag mismatch for provider %s", tc.provider)
		assert.Equal(t, tc.metaInsights, resp.Features.MetaInsights, "Meta insights flag mismatch for provider %s", tc.provider)
	}
}

// TestGetAppConfig_CampaignsAlwaysEnabled tests that campaigns are always enabled
func TestGetAppConfig_CampaignsAlwaysEnabled(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	providers := []string{"meta", "whatsmeow", ""}

	for _, provider := range providers {
		app.Config.WhatsApp.Provider = provider

		req := testutil.NewJSONRequest(t, nil)
		err := app.GetAppConfig(req)
		require.NoError(t, err)

		var resp handlers.AppConfigResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		assert.True(t, resp.Features.Campaigns, "Campaigns should always be enabled regardless of provider")
	}
}

// TestGetAppConfig_HTTPMethod tests that only GET requests work
func TestGetAppConfig_HTTPMethod(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "meta"

	// Test with GET (implicit in NewJSONRequest)
	req := testutil.NewJSONRequest(t, nil)
	err := app.GetAppConfig(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}
