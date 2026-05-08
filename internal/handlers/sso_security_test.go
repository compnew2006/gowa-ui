package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSOSecurityIsPrivateOrLocalIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"224.0.0.1", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		assert.NotNil(t, ip, "invalid IP %s", tt.ip)
		assert.Equal(t, tt.private, isPrivateOrLocalIP(ip), "IP=%s", tt.ip)
	}
}

func TestSSOSecurityPKCEChallenge(t *testing.T) {
	verifier := "test-verifier-code-1234567890"
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	assert.Equal(t, expected, pkceChallenge(verifier))
}

func TestSSOSecurityGeneratePKCEVerifier(t *testing.T) {
	v, err := generatePKCEVerifier()
	assert.NoError(t, err)
	assert.Len(t, v, 64)
}

func TestSSOSecurityPKCEChallengeDeterministic(t *testing.T) {
	assert.Equal(t, pkceChallenge("abc"), pkceChallenge("abc"))
	assert.NotEqual(t, pkceChallenge("abc"), pkceChallenge("xyz"))
}

func TestSSOSecurityAllowLocalEndpointsNil(t *testing.T) {
	var a *App
	assert.False(t, a.allowLocalSSOEndpoints())
}
