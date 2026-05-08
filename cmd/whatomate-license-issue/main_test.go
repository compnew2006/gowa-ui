package main

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/licenseissuer"
	"github.com/stretchr/testify/assert"
)

func TestDefaultIssueOptions(t *testing.T) {
	opts := licenseissuer.DefaultIssueOptions()

	assert.NotEmpty(t, opts.KeyID)
	assert.Greater(t, opts.Organizations, 0)
	assert.Greater(t, opts.UsersPerOrg, 0)
	assert.Greater(t, opts.WhatsAppEndpointsPerOrg, 0)
	assert.Greater(t, opts.Workers, 0)
	assert.NotEmpty(t, opts.Duration)
	assert.NotEmpty(t, opts.Tier)
}
