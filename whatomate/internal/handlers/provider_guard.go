package handlers

import (
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ProviderGuard returns a handler wrapper that blocks requests when the
// active provider is NOT the given requiredProvider.
// When blocked, it returns 404 with a clear "feature_unavailable" error.
func (a *App) ProviderGuard(requiredProvider string, next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		provider := a.Config.WhatsApp.Provider
		if provider == "" {
			provider = "meta"
		}
		if provider != requiredProvider {
			return r.SendErrorEnvelope(
				fasthttp.StatusNotFound,
				"This feature requires "+requiredProvider+" provider",
				map[string]string{"error": "feature_unavailable"},
				"",
			)
		}
		return next(r)
	}
}
