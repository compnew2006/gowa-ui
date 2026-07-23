package whatsapp_test

import (
	"context"
	"testing"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

// fakeProvider is a test Provider implementation for verifying registry routing.
type fakeProvider struct {
	name string
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Capabilities() whatsapp.Capabilities {
	return whatsapp.Capabilities{}
}

// Implement all Provider methods as stubs — we only care about Name() for routing.
func (f *fakeProvider) SendTextMessage(_ context.Context, _ *whatsapp.Account, _ whatsapp.Recipient, _ string, _ ...string) (string, error) {
	return "", nil
}

func TestRegistry_ReturnsMetaForNilAccount(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(nil)
	assert.Equal(t, "meta", p.Name())
}

func TestRegistry_ReturnsMetaForMetaAccount(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(&whatsapp.Account{ProviderType: "meta"})
	assert.Equal(t, "meta", p.Name())
}

func TestRegistry_ReturnsMetaForEmptyProviderType(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(&whatsapp.Account{ProviderType: ""})
	assert.Equal(t, "meta", p.Name())
}

func TestRegistry_MemoReturnsSameInstance(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	assert.Same(t, meta, reg.Meta())
}

func TestRegistry_GetByTypeMeta(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.GetByType("meta", "", "")
	assert.Equal(t, "meta", p.Name())
}

func TestRegistry_GetByTypeGowaWithoutFactory(t *testing.T) {
	t.Parallel()
	// If no GOWA factory is registered, falls back to Meta.
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.GetByType("gowa", "http://localhost:3000", "dev1")
	// Without factory, should fall back to Meta.
	assert.Equal(t, "meta", p.Name())
}
