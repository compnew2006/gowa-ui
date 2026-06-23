package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicenseAllowsModule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		tier      string
		moduleKey string
		want      bool
	}{
		// trial tier: limited set
		{"trial allows facebook-core", "trial", "facebook-core", true},
		{"trial allows facebook-accounts", "trial", "facebook-accounts", true},
		{"trial denies facebook-comments", "trial", "facebook-comments", false},
		{"trial denies unknown module", "trial", "instagram-direct", false},

		// starter tier: adds comments
		{"starter allows facebook-comments", "starter", "facebook-comments", true},
		{"starter denies unlisted module", "starter", "instagram-direct", false},

		// pro/enterprise: wildcard grants everything
		{"pro wildcard allows any module", "pro", "instagram-direct", true},
		{"pro wildcard allows facebook-core", "pro", "facebook-core", true},
		{"enterprise wildcard allows any module", "enterprise", "whatever-future", true},

		// production: paid host-bound deployments emit this tier string from
		// the license issuer; it must be unrestricted like pro/enterprise.
		{"production wildcard allows any module", "production", "instagram-direct", true},
		{"production wildcard allows facebook-core", "production", "facebook-core", true},
		{"production wildcard allows facebook-people-search", "production", "facebook-people-search", true},

		// deny-by-default for unknown / empty tiers
		{"unknown tier denies", "bogus-tier", "facebook-core", false},
		{"empty tier denies", "", "facebook-core", false},

		// empty module key never allowed
		{"pro empty module denied", "pro", "", false},
		{"empty empty denied", "", "", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, LicenseAllowsModule(tc.tier, tc.moduleKey))
		})
	}
}

func TestSetLicenseTierGetter(t *testing.T) {
	t.Parallel()

	// Snapshot and restore the package-level getter so this test cannot leak
	// state into other tests in the package.
	previous := licenseTierGetter
	t.Cleanup(func() { licenseTierGetter = previous })

	t.Run("nil getter resets to safe default", func(t *testing.T) {
		t.Parallel()
		SetLicenseTierGetter(nil)
		assert.Equal(t, "", licenseTierGetter())
	})

	t.Run("non-nil getter is installed", func(t *testing.T) {
		t.Parallel()
		SetLicenseTierGetter(func() string { return "trial" })
		assert.Equal(t, "trial", licenseTierGetter())
	})
}
