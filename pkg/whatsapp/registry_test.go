package whatsapp_test

import (
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
)

func registerTestGowaFactory() {
	whatsapp.RegisterGowaFactory(
		func(_ uuid.UUID, baseURL string) (string, string) { return "user", "pass" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
}

func TestRegistry_ReturnsGowaForNilAccount(t *testing.T) {
	registerTestGowaFactory()
	reg := whatsapp.NewRegistry(testutil.NopLogger())
	p := reg.Get(nil)
	assert.NotNil(t, p)
}

func TestRegistry_CachesClientPerBaseURL(t *testing.T) {
	registerTestGowaFactory()
	reg := whatsapp.NewRegistry(testutil.NopLogger())
	a := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.test:3000"})
	b := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.test:3000", GowaDeviceID: "other-device"})
	assert.Same(t, a, b)

	c := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.other:3000"})
	assert.NotSame(t, a, c)
}

func TestRegistry_InvalidateGowaDropsCachedClient(t *testing.T) {
	registerTestGowaFactory()
	reg := whatsapp.NewRegistry(testutil.NopLogger())
	a := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.test:3000"})
	reg.InvalidateGowa(uuid.Nil, "http://gowa.test:3000")
	b := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.test:3000"})
	assert.NotSame(t, a, b)
}

// TestRegistry_OrgScopingIsolatesSameBaseURL locks the tenant-isolation fix:
// two organizations that register the SAME GOWA base URL must get distinct
// cached clients (keyed by org+base URL), so org B can never reuse — or be
// served — org A's Basic Auth credentials. A per-call counter proves the
// factory was invoked once per org, not shared.
func TestRegistry_OrgScopingIsolatesSameBaseURL(t *testing.T) {
	const sameURL = "http://gowa.shared:3000"
	var calls int64
	whatsapp.RegisterGowaFactory(
		func(orgID uuid.UUID, baseURL string) (string, string) {
			atomic.AddInt64(&calls, 1)
			// Return org-distinct credentials so a shared client would be detectable.
			return orgID.String() + "-user", "pass"
		},
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
	reg := whatsapp.NewRegistry(testutil.NopLogger())

	orgA := uuid.New()
	orgB := uuid.New()
	a := reg.Get(&whatsapp.Account{OrganizationID: orgA, GowaBaseURL: sameURL})
	b := reg.Get(&whatsapp.Account{OrganizationID: orgB, GowaBaseURL: sameURL})

	// Same base URL, different org → distinct cached providers.
	assert.NotSame(t, a, b, "two orgs with the same GOWA base URL must NOT share a client")

	// Re-fetching the same org reuses the cached client (no new factory call).
	a2 := reg.Get(&whatsapp.Account{OrganizationID: orgA, GowaBaseURL: sameURL})
	assert.Same(t, a, a2, "same org+base URL must reuse the cached client")
	assert.EqualValues(t, 2, atomic.LoadInt64(&calls), "factory called once per org, not per Get")

	// Invalidate only org A's client; org B's stays cached.
	reg.InvalidateGowa(orgA, sameURL)
	a3 := reg.Get(&whatsapp.Account{OrganizationID: orgA, GowaBaseURL: sameURL})
	assert.NotSame(t, a, a3, "org A client rebuilt after invalidate")
	b2 := reg.Get(&whatsapp.Account{OrganizationID: orgB, GowaBaseURL: sameURL})
	assert.Same(t, b, b2, "org B client untouched by org A invalidation")
}
