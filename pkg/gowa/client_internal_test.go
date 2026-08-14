package gowa

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Internal-package test: httpClient is unexported. Clients built by New must
// share one pooled transport so connections stay warm across per-instance
// client rebuilds (instance endpoints, probes) instead of churning.
func TestNewUsesSharedPooledTransport(t *testing.T) {
	t.Parallel()
	a := New("http://localhost:3000", "", "")
	b := New("https://gowa.example.com", "user", "pass")

	assert.Same(t, a.httpClient.Transport, b.httpClient.Transport,
		"clients built by New must share one transport")
	tr, ok := a.httpClient.Transport.(*http.Transport)
	if assert.True(t, ok, "transport should be *http.Transport") {
		assert.Equal(t, 32, tr.MaxIdleConnsPerHost)
	}
}
