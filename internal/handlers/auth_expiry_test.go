package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextAccessTokenExpiry(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*60*60)

	tests := []struct {
		name     string
		now      time.Time
		expected time.Time
	}{
		{
			name:     "midday expires at next midnight",
			now:      time.Date(2026, time.February, 28, 12, 30, 0, 0, loc),
			expected: time.Date(2026, time.March, 1, 0, 0, 0, 0, loc),
		},
		{
			name:     "one second before midnight has one second ttl",
			now:      time.Date(2026, time.February, 28, 23, 59, 59, 0, loc),
			expected: time.Date(2026, time.March, 1, 0, 0, 0, 0, loc),
		},
		{
			name:     "exact midnight expires next day midnight",
			now:      time.Date(2026, time.February, 28, 0, 0, 0, 0, loc),
			expected: time.Date(2026, time.March, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, nextAccessTokenExpiry(tc.now))
		})
	}
}

func TestAccessTokenTTLSeconds(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, time.February, 28, 23, 59, 59, 0, loc)
	expiresAt := time.Date(2026, time.March, 1, 0, 0, 0, 0, loc)
	assert.Equal(t, 1, accessTokenTTLSeconds(now, expiresAt))

	now = time.Date(2026, time.February, 28, 0, 0, 0, 0, loc)
	expiresAt = time.Date(2026, time.March, 1, 0, 0, 0, 0, loc)
	assert.Equal(t, 24*60*60, accessTokenTTLSeconds(now, expiresAt))

	now = time.Date(2026, time.March, 1, 0, 0, 1, 0, loc)
	expiresAt = time.Date(2026, time.March, 1, 0, 0, 0, 0, loc)
	assert.Equal(t, 1, accessTokenTTLSeconds(now, expiresAt))
}
