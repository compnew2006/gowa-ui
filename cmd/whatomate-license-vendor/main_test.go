package main

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/licenseissuer"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyID(t *testing.T) {
	assert.Equal(t, licenseissuer.DefaultKeyID, defaultKeyID)
	assert.NotEmpty(t, defaultKeyID)
}

func TestPrintUsageDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		printUsage()
	})
}

func TestDefaultIssueOptions(t *testing.T) {
	opts := licenseissuer.DefaultIssueOptions()

	assert.NotEmpty(t, opts.KeyID)
	assert.Greater(t, opts.Organizations, 0)
	assert.Greater(t, opts.UsersPerOrg, 0)
	assert.Greater(t, opts.Workers, 0)
}
