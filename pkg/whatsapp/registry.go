package whatsapp

import (
	"sync"
)

// Registry holds provider instances and resolves the correct one for a given
// account. GOWA clients are keyed by base URL because each GOWA instance runs
// at a different endpoint.
//
// Usage:
//
//	reg := whatsapp.NewRegistry(log)
//	provider := reg.Get(account) // returns the gowa.Client for the account
type Registry struct {
	mu   sync.RWMutex
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

// NewRegistry creates an empty GOWA provider registry.
func NewRegistry(log logger) *Registry {
	return &Registry{
		gowa: make(map[string]Provider),
		log:  log,
	}
}

// Get returns the GOWA provider for the given account credentials.
func (r *Registry) Get(account *Account) Provider {
	if account == nil {
		return r.getOrCreateGowa("", "")
	}
	return r.getOrCreateGowa(account.GowaBaseURL, account.GowaDeviceID)
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
		// Factory not registered — a startup wiring bug.
		if r.log != nil {
			r.log.Error("GOWA provider factory not registered", "base_url", baseURL)
		}
		return nil
	}
	r.gowa[baseURL] = p
	if r.log != nil {
		r.log.Info("Created GOWA provider", "base_url", baseURL, "device_id", deviceID)
	}
	return p
}

// InvalidateGowa drops the cached GOWA client for the given base URL (or all
// of them when baseURL is empty) so the next Get() re-resolves credentials via
// the factory. Call this whenever a GOWA instance's credentials change in the
// DB, otherwise the stale (cached) client keeps using the old username/password.
func (r *Registry) InvalidateGowa(baseURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if baseURL == "" {
		r.gowa = make(map[string]Provider)
		return
	}
	delete(r.gowa, baseURL)
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

// NewRegistryWithFactory is a convenience over the two-call
// RegisterGowaFactory + NewRegistry idiom that main.go and several test
// wrappers open-code identically. It registers the supplied per-call-site
// factory closures (so each caller controls its own credResolver and
// newClient — RegisterGowaFactory is process-global, callers must NOT
// share a hard-coded default) and returns a fresh Registry built with log.
//
// Each caller passes its OWN closures and its OWN logger so the
// process-global factory mutation is owned by the caller, not by this
// helper.
func NewRegistryWithFactory(
	log logger,
	credentialResolver func(baseURL string) (username, password string),
	newClient func(baseURL, username, password string) Provider,
) *Registry {
	RegisterGowaFactory(credentialResolver, newClient)
	return NewRegistry(log)
}
