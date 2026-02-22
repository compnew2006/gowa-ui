package handlers

import (
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

// AppConfigResponse is the public config payload returned by GET /api/config.
type AppConfigResponse struct {
	WhatsAppProvider string       `json:"whatsapp_provider"`
	Features         FeatureFlags `json:"features"`
}

// GetAppConfig returns the active WhatsApp provider and feature flags.
// Meta-only features (templates, flows, catalog, business profile, campaigns,
// meta insights) are available only when the provider is "meta".
func (a *App) GetAppConfig(r *fastglue.Request) error {
	provider := a.Config.WhatsApp.Provider
	if provider == "" {
		provider = "meta"
	}

	isMeta := provider == "meta"

	resp := AppConfigResponse{
		WhatsAppProvider: provider,
		Features: FeatureFlags{
			Templates:       isMeta,
			Flows:           isMeta,
			Catalog:         isMeta,
			BusinessProfile: isMeta,
			Campaigns:       isMeta,
			MetaInsights:    isMeta,
		},
	}

	return r.SendEnvelope(resp)
}
