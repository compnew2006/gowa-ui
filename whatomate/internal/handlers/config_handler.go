package handlers

import (
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/zerodha/fastglue"
)

// FeatureFlags describes which provider-dependent features are available.
type FeatureFlags struct {
	Templates       bool `json:"templates"`
	Flows           bool `json:"flows"`
	Catalog         bool `json:"catalog"`
	BusinessProfile bool `json:"business_profile"`
	Campaigns       bool `json:"campaigns"`
	MetaInsights    bool `json:"meta_insights"`
}

type TenantConfig struct {
	SubdomainLocked  bool   `json:"subdomain_locked"`
	OrganizationSlug string `json:"organization_slug,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
}

// AppConfigResponse is the public config payload returned by GET /api/config.
type AppConfigResponse struct {
	WhatsAppProvider string       `json:"whatsapp_provider"`
	Features         FeatureFlags `json:"features"`
	Tenant           TenantConfig `json:"tenant"`
}

// GetAppConfig returns the active WhatsApp provider and feature flags.
// Meta-only features (templates, flows, catalog, business profile, meta
// insights) are available only when the provider is "meta".
// Campaigns are supported for both Meta and whatsmeow.
func (a *App) GetAppConfig(r *fastglue.Request) error {
	provider := a.Config.WhatsApp.Provider
	if provider == "" {
		provider = "meta"
	}

	isMeta := provider == "meta"
	tenantConfig := TenantConfig{}
	if hostOrg, err := tenant.ResolveHostOrganization(r, a.DB); err == nil && hostOrg != nil {
		tenantConfig = TenantConfig{
			SubdomainLocked:  true,
			OrganizationSlug: hostOrg.Slug,
			OrganizationName: hostOrg.Name,
		}
	} else if err != nil {
		a.Log.Warn("Failed to resolve tenant host configuration", "error", err)
	}

	resp := AppConfigResponse{
		WhatsAppProvider: provider,
		Features: FeatureFlags{
			Templates:       isMeta,
			Flows:           isMeta,
			Catalog:         isMeta,
			BusinessProfile: isMeta,
			Campaigns:       true,
			MetaInsights:    isMeta,
		},
		Tenant: tenantConfig,
	}

	return r.SendEnvelope(resp)
}
