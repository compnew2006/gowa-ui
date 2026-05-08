package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationSlugNormalizeBasic(t *testing.T) {
	assert.Equal(t, "my-org", normalizeOrganizationSlug("My Org"))
}

func TestOrganizationSlugNormalizeLowercases(t *testing.T) {
	assert.Equal(t, "acme-corp", normalizeOrganizationSlug("ACME CORP"))
}

func TestOrganizationSlugNormalizeNumbers(t *testing.T) {
	assert.Equal(t, "org-123", normalizeOrganizationSlug("org 123"))
}

func TestOrganizationSlugNormalizeStripsSpecial(t *testing.T) {
	assert.Equal(t, "abc", normalizeOrganizationSlug("a@b#c!"))
}

func TestOrganizationSlugNormalizeEmpty(t *testing.T) {
	assert.Equal(t, "", normalizeOrganizationSlug(""))
	assert.Equal(t, "", normalizeOrganizationSlug("   "))
	assert.Equal(t, "", normalizeOrganizationSlug("@#$%"))
}

func TestOrganizationSlugNormalizeMultipleSeparators(t *testing.T) {
	assert.Equal(t, "a-b", normalizeOrganizationSlug("a  --  b"))
	assert.Equal(t, "a-b", normalizeOrganizationSlug("a_-_b"))
	assert.Equal(t, "a-b", normalizeOrganizationSlug("a . b"))
}

func TestOrganizationSlugNormalizeTrimsLeadingTrailing(t *testing.T) {
	assert.Equal(t, "abc", normalizeOrganizationSlug("--abc--"))
	assert.Equal(t, "abc", normalizeOrganizationSlug("  abc  "))
}
