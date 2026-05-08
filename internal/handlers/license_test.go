package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicenseHandlerLicenseAllowsPublicRequest(t *testing.T) {
	a := &App{}

	publicPaths := []string{"/health", "/ready", "/api/license/bootstrap", "/api/license/activate", "/api/webhook", "", "/"}
	for _, p := range publicPaths {
		t.Run(p, func(t *testing.T) {
			assert.True(t, a.licenseAllowsPublicRequest(p))
		})
	}
}

func TestLicenseHandlerLicenseAllowsPublicRequestNonAPI(t *testing.T) {
	a := &App{}
	assert.True(t, a.licenseAllowsPublicRequest("/static/app.js"))
	assert.True(t, a.licenseAllowsPublicRequest("/favicon.ico"))
}

func TestLicenseHandlerLicenseAllowsPublicRequestAPIBlocked(t *testing.T) {
	a := &App{}
	assert.False(t, a.licenseAllowsPublicRequest("/api/contacts"))
	assert.False(t, a.licenseAllowsPublicRequest("/api/messages"))
}

func TestLicenseHandlerLicenseBlocksRequestNilLicense(t *testing.T) {
	a := &App{License: nil}
	assert.False(t, a.LicenseBlocksRequest("GET", "/api/contacts"))
}

func TestLicenseHandlerLicenseBlocksValueDeliveryNil(t *testing.T) {
	a := &App{License: nil}
	assert.False(t, a.licenseBlocksValueDelivery())
}

func TestLicenseHandlerActivateLicenseRequest(t *testing.T) {
	req := activateLicenseRequest{Token: "my-token"}
	assert.Equal(t, "my-token", req.Token)
}
