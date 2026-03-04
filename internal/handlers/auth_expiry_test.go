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
		ttlMins  int
		expected time.Time
	}{
		{
			name:     "uses configured ttl minutes",
			now:      time.Date(2026, time.February, 28, 12, 30, 0, 0, loc),
			ttlMins:  15,
			expected: time.Date(2026, time.February, 28, 12, 45, 0, 0, loc),
		},
		{
			name:     "falls back to default for invalid ttl",
			now:      time.Date(2026, time.February, 28, 23, 59, 59, 0, loc),
			ttlMins:  0,
			expected: time.Date(2026, time.March, 1, 0, 14, 59, 0, loc),
		},
		{
			name:     "supports short custom ttl",
			now:      time.Date(2026, time.February, 28, 0, 0, 0, 0, loc),
			ttlMins:  2,
			expected: time.Date(2026, time.February, 28, 0, 2, 0, 0, loc),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, nextAccessTokenExpiry(tc.now, tc.ttlMins))
		})
	}
}

func TestAccessTokenTTLSeconds(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, time.February, 28, 12, 0, 0, 0, loc)
	expiresAt := now.Add(15 * time.Minute)
	assert.Equal(t, 900, accessTokenTTLSeconds(now, expiresAt))

	now = time.Date(2026, time.February, 28, 12, 0, 0, 0, loc)
	expiresAt = now.Add(500 * time.Millisecond)
	assert.Equal(t, 1, accessTokenTTLSeconds(now, expiresAt))

	now = time.Date(2026, time.March, 1, 0, 0, 1, 0, loc)
	expiresAt = time.Date(2026, time.March, 1, 0, 0, 0, 0, loc)
	assert.Equal(t, 1, accessTokenTTLSeconds(now, expiresAt))
}
