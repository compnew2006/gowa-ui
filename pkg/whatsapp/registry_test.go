package whatsapp_test

import (
	"testing"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_ReturnsMetaForNilAccount(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(nil)
	assert.Same(t, meta, p)
}

func TestRegistry_ReturnsMetaForMetaAccount(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(&whatsapp.Account{ProviderType: "meta"})
	assert.Same(t, meta, p)
}

func TestRegistry_ReturnsMetaForEmptyProviderType(t *testing.T) {
	t.Parallel()
	meta := whatsapp.New(testutil.NopLogger())
	reg := whatsapp.NewRegistry(meta, testutil.NopLogger())
	p := reg.Get(&whatsapp.Account{ProviderType: ""})
	assert.Same(t, meta, p)
}
