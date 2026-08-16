package handlers

import (
	"context"
	"testing"

	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimDeviceAlertSlot(t *testing.T) {
	// claimDeviceAlertSlot only needs the logger and Redis — no DB.
	app := &App{Log: testutil.NopLogger()}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	// First claim wins, the second within the cooldown is suppressed. With
	// no Redis configured the slot always fails open (alert anyway).
	id := uuid.New()
	first := app.claimDeviceAlertSlot(context.Background(), id)
	second := app.claimDeviceAlertSlot(context.Background(), id)
	if app.Redis == nil {
		assert.True(t, first, "no Redis: fail open")
		assert.True(t, second, "no Redis: fail open")
		return
	}
	require.True(t, first, "first alert must be claimable")
	assert.False(t, second, "cooldown must suppress the immediate re-alert")

	// A different account is unaffected — the cooldown is per account.
	assert.True(t, app.claimDeviceAlertSlot(context.Background(), uuid.New()), "per-account cooldown")
}
