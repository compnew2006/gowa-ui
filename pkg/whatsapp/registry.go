package whatsapp

import (
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Registry holds provider instances and resolves the correct one for a given
// account. GOWA clients are keyed by (organization_id, base URL): each GOWA
// instance runs at a different endpoint, AND two organizations that register
// the same GOWA base URL with different Basic Auth credentials must get
// isolated cached clients — otherwise the shared client would use whichever
// org's credentials were resolved first, leaking credentials across tenants.
//
// Usage:
//
//	reg := whatsapp.NewRegistry(log)
//	provider := reg.Get(account) // returns the gowa.Client for the account's org+base URL
type Registry struct {
	mu   sync.RWMutex
	gowa map[string]Provider // keyed by gowaClientKey(orgID, baseURL)
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
		return r.getOrCreateGowa(uuid.Nil, "", "")
	}
	return r.getOrCreateGowa(account.OrganizationID, account.GowaBaseURL, account.GowaDeviceID)
}

// gowaClientKey is the registry cache key for a GOWA client. It is org-scoped
// so two organizations that register the same GOWA base URL with different
// Basic Auth credentials get isolated cached clients and never share — or leak
// — each other's credentials. A zero orgID (uuid.Nil: legacy / not-yet-resolved
// account, or a synthetic test account) falls back to the bare base URL so the
// pre-fix behavior is preserved during rollout and for callers that don't know
// the org.
func gowaClientKey(orgID uuid.UUID, baseURL string) string {
	if orgID == uuid.Nil {
		return baseURL
	}
	return orgID.String() + "|" + baseURL
}

// getOrCreateGowa returns an existing GOWA client for the (org, base URL) pair,
// or creates one on first use. The device ID is per-account and passed through
// on each API call, so the client itself is shared across accounts on the same
// org+GOWA instance. Org-scoping the key is what prevents cross-tenant
// credential reuse (see ResolveGowaCreds).
func (r *Registry) getOrCreateGowa(orgID uuid.UUID, baseURL, deviceID string) Provider {
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	key := gowaClientKey(orgID, baseURL)

	r.mu.RLock()
	if p, ok := r.gowa[key]; ok {
		r.mu.RUnlock()
		return p
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if p, ok := r.gowa[key]; ok {
		return p
	}

	// Create a new GOWA client via the factory function.
	// We use a factory indirection to avoid an import cycle between
	// whatsapp (this package) and gowa (which imports whatsapp).
	p := gowaFactory(orgID, baseURL)
	if p == nil {
		// Factory not registered — a startup wiring bug.
		if r.log != nil {
			r.log.Error("GOWA provider factory not registered", "base_url", baseURL, "org_id", orgID)
		}
		return nil
	}
	r.gowa[key] = p
	if r.log != nil {
		r.log.Info("Created GOWA provider", "base_url", baseURL, "device_id", deviceID, "org_id", orgID)
	}
	return p
}

// InvalidateGowa drops the cached GOWA client for the given (org, base URL) so
// the next Get() re-resolves credentials via the factory. Call this whenever a
// GOWA instance's credentials change in the DB, otherwise the stale (cached)
// client keeps using the old username/password.
//
// baseURL == "" drops every cached client (regardless of org). A zero orgID
// with a non-empty baseURL drops every client whose key maps to that baseURL
// across all orgs — used by callers that don't know the owning org.
func (r *Registry) InvalidateGowa(orgID uuid.UUID, baseURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if baseURL == "" {
		r.gowa = make(map[string]Provider)
		return
	}
	if orgID == uuid.Nil {
		suffix := "|" + baseURL
		for k := range r.gowa {
			if k == baseURL || strings.HasSuffix(k, suffix) {
				delete(r.gowa, k)
			}
		}
		return
	}
	delete(r.gowa, gowaClientKey(orgID, baseURL))
}

// --- Factory registration ---

// gowaFactory is set by RegisterGowaFactory so that the whatsapp package
// can create GOWA clients without importing the gowa package directly
// (which would create an import cycle: gowa imports whatsapp for types,
// so whatsapp cannot import gowa).
var gowaFactory = func(orgID uuid.UUID, baseURL string) Provider {
	return nil
}

// RegisterGowaFactory sets the function used to create GOWA provider
// instances. The credentialResolver is called with each (org, base URL) to get
// the Basic Auth credentials for that org's instance — org-scoped resolution is
// what keeps credentials isolated across tenants. This must be called once at
// startup (typically from main.go) before any GOWA account is resolved.
func RegisterGowaFactory(
	credentialResolver func(orgID uuid.UUID, baseURL string) (username, password string),
	newClient func(baseURL, username, password string) Provider,
) {
	gowaFactory = func(orgID uuid.UUID, baseURL string) Provider {
		user, pass := credentialResolver(orgID, baseURL)
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
	credentialResolver func(orgID uuid.UUID, baseURL string) (username, password string),
	newClient func(baseURL, username, password string) Provider,
) *Registry {
	RegisterGowaFactory(credentialResolver, newClient)
	return NewRegistry(log)
}
