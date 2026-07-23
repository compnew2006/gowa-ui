package whatsapp

import (
	"fmt"
	"sync"
)

// Registry holds provider instances and resolves the correct one for a given
// account. The Meta client is a singleton (it is account-agnostic — each call
// passes the account's credentials explicitly). GOWA clients are keyed by
// base URL because each GOWA instance runs at a different endpoint.
//
// Usage:
//
//	reg := whatsapp.NewRegistry(metaClient)
//	provider := reg.Get(account) // returns metaProvider or gowa.Client
type Registry struct {
	mu   sync.RWMutex
	meta Provider
	gowa map[string]Provider // keyed by base URL
	log  logger
}

// logger is a minimal logging interface to avoid importing logf here.
type logger interface {
	Info(msg string, ctx ...any)
	Warn(msg string, ctx ...any)
	Error(msg string, ctx ...any)
	Debug(msg string, ctx ...any)
}

// NewRegistry creates a registry with the given Meta provider as the default.
func NewRegistry(meta Provider, log logger) *Registry {
	return &Registry{
		meta: meta,
		gowa: make(map[string]Provider),
		log:  log,
	}
}

// Get returns the provider for the given account credentials.
// ProviderType "gowa" → GOWA client for the account's GowaBaseURL.
// Everything else (including "") → Meta client.
func (r *Registry) Get(account *Account) Provider {
	if account == nil {
		return r.meta
	}

	if account.ProviderType == "gowa" {
		return r.getOrCreateGowa(account.GowaBaseURL, account.GowaDeviceID)
	}

	return r.meta
}

// GetByType returns the provider for a raw provider type string and optional
// GOWA base URL. This is useful for code paths that don't have a full Account
// (e.g. webhook routing).
func (r *Registry) GetByType(providerType, gowaBaseURL, gowaDeviceID string) Provider {
	if providerType == "gowa" {
		return r.getOrCreateGowa(gowaBaseURL, gowaDeviceID)
	}
	return r.meta
}

// getOrCreateGowa returns an existing GOWA client for the base URL, or
// creates one on first use. The device ID is per-account and passed through
// on each API call, so the client itself is shared across accounts on the
// same GOWA instance.
func (r *Registry) getOrCreateGowa(baseURL, deviceID string) Provider {
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	r.mu.RLock()
	if p, ok := r.gowa[baseURL]; ok {
		r.mu.RUnlock()
		return p
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if p, ok := r.gowa[baseURL]; ok {
		return p
	}

	// Create a new GOWA client via the factory function.
	// We use a factory indirection to avoid an import cycle between
	// whatsapp (this package) and gowa (which imports whatsapp).
	p := gowaFactory(baseURL)
	if p == nil {
		// Factory not registered — fall back to Meta with a warning.
		if r.log != nil {
			r.log.Warn("GOWA provider factory not registered, falling back to Meta", "base_url", baseURL)
		}
		return r.meta
	}
	r.gowa[baseURL] = p
	if r.log != nil {
		r.log.Info("Created GOWA provider", "base_url", baseURL, "device_id", deviceID)
	}
	return p
}

// --- Factory registration ---

// gowaFactory is set by RegisterGowaFactory so that the whatsapp package
// can create GOWA clients without importing the gowa package directly
// (which would create an import cycle: gowa imports whatsapp for types,
// so whatsapp cannot import gowa).
var gowaFactory = func(baseURL string) Provider {
	return nil
}

// RegisterGowaFactory sets the function used to create GOWA provider
// instances. The credentialResolver is called with each base URL to get
// the Basic Auth credentials for that instance. This must be called once
// at startup (typically from main.go) before any GOWA account is resolved.
func RegisterGowaFactory(
	credentialResolver func(baseURL string) (username, password string),
	newClient func(baseURL, username, password string) Provider,
) {
	gowaFactory = func(baseURL string) Provider {
		user, pass := credentialResolver(baseURL)
		return newClient(baseURL, user, pass)
	}
}

// Meta returns the Meta provider (useful for code paths that always need Meta).
func (r *Registry) Meta() Provider { return r.meta }

// String returns a human-readable summary for debugging.
func (r *Registry) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return fmt.Sprintf("Registry{meta=%T, gowa_instances=%d}", r.meta, len(r.gowa))
}
