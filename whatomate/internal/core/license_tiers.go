package core

// tierModules maps a license tier to the managed-module keys it is entitled to
// enable. A tier with the wildcard "*" grants every registered module.
//
// This is the single source of truth for license→module entitlement. Both
// GateModule (runtime route gating) and the module-management plugin (admin
// give/ungive) consult LicenseAllowsModule so license-check logic is not
// duplicated.
//
// Unknown tiers ("" included) intentionally grant nothing by default, satisfying
// the "existing tenants without new fields = deny by default" requirement.
//
// Note: the License system is host-bound and optional. When no license is
// active, callers pass tier == "" and modules gate purely on the existing
// ModuleManager state (see GateModule).
var tierModules = map[string]map[string]bool{
	"trial": {
		"facebook-core":     true,
		"facebook-accounts": true,
	},
	"starter": {
		"facebook-core":     true,
		"facebook-accounts": true,
		"facebook-comments": true,
	},
	"pro": {
		"*": true,
	},
	"enterprise": {
		"*": true,
	},
	// "production" is the tier string emitted by the license issuer for paid
	// host-bound deployments (see internal/licenseissuer / license_records.tier).
	// Treat it as fully unrestricted, identical to pro/enterprise: a paid
	// production license is never meant to be module-gated. Without this alias,
	// LicenseAllowsModule denies every managed module on a paid deployment
	// because "production" is otherwise an unknown tier (deny-by-default).
	"production": {
		"*": true,
	},
}

// LicenseAllowsModule reports whether the given license tier is entitled to
// enable moduleKey. A wildcard entry ("*") under a tier permits any module.
// Unknown tiers and empty tier strings return false (deny by default).
func LicenseAllowsModule(tier, moduleKey string) bool {
	if tier == "" || moduleKey == "" {
		return false
	}
	allowed, ok := tierModules[tier]
	if !ok {
		return false
	}
	return allowed["*"] || allowed[moduleKey]
}
