package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveAutoCampaignWindowUsesExactLastGeneratedAt(t *testing.T) {
	location := time.FixedZone("UTC+2", 2*60*60)
	localNow := time.Date(2026, time.April, 8, 12, 1, 0, 0, location)
	lastGeneratedAt := time.Date(2026, time.April, 1, 10, 0, 30, 0, time.UTC)

	windowStart, windowEnd := resolveAutoCampaignWindow(localNow, &lastGeneratedAt, 7)

	assert.Equal(t, lastGeneratedAt.In(location), windowStart)
	assert.Equal(t, localNow, windowEnd)
}

func TestResolveAutoCampaignWindowFallsBackToIntervalOnFirstRun(t *testing.T) {
	location := time.FixedZone("UTC+2", 2*60*60)
	localNow := time.Date(2026, time.April, 8, 12, 1, 0, 0, location)

	windowStart, windowEnd := resolveAutoCampaignWindow(localNow, nil, 7)

	assert.Equal(t, localNow.AddDate(0, 0, -7), windowStart)
	assert.Equal(t, localNow, windowEnd)
}
