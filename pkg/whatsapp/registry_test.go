package whatsapp_test

import (
	"testing"

	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

func registerTestGowaFactory() {
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "user", "pass" },
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
	reg.InvalidateGowa("http://gowa.test:3000")
	b := reg.Get(&whatsapp.Account{GowaBaseURL: "http://gowa.test:3000"})
	assert.NotSame(t, a, b)
}
