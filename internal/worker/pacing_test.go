package worker

import (
	"context"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountPacePerMinute(t *testing.T) {
	tests := []struct {
		name  string
		cfg   int
		block map[string]any
		want  int
	}{
		{name: "no block, no config → unlimited", cfg: 0, block: nil, want: 0},
		{name: "config default only", cfg: 45, block: nil, want: 45},
		{name: "block wins over config", cfg: 45, block: map[string]any{"messages_per_minute": float64(20)}, want: 20},
		{name: "zero/invalid block falls back to config", cfg: 45, block: map[string]any{"messages_per_minute": float64(0)}, want: 45},
		{name: "non-numeric block falls back to config", cfg: 30, block: map[string]any{"messages_per_minute": "fast"}, want: 30},
		{name: "block without config", cfg: 0, block: map[string]any{"messages_per_minute": float64(60)}, want: 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{Config: &config.Config{}}
			w.Config.Campaigns.DefaultPacingPerMinute = tt.cfg
			// The function receives the account's FULL settings map and
			// looks up its send_pacing block inside.
			settings := map[string]any{}
			if tt.block != nil {
				settings["send_pacing"] = tt.block
			}
			assert.Equal(t, tt.want, w.accountPacePerMinute(settings))
		})
	}
}

func TestPaceCampaignSend_NoRedisOrZeroRateIsNoop(t *testing.T) {
	// No Redis + no rate: must return immediately (no panic, no block).
	w := &Worker{Log: testutil.NopLogger()}
	done := make(chan struct{})
	go func() {
		w.paceCampaignSend(context.Background(), uuid.New(), "acct", 0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("paceCampaignSend blocked on a no-op configuration")
	}
}

func TestReservePaceSlot(t *testing.T) {
	rdb := testutil.SetupTestRedis(t)
	if rdb == nil {
		t.Skip("TEST_REDIS_URL not set")
	}
	w := &Worker{Redis: rdb, Log: testutil.NopLogger()}
	ctx := context.Background()
	key := "campaign_pace:test:" + uuid.NewString()

	// First two sends within a budget of 2 are granted...
	for i := 0; i < 2; i++ {
		ok, wait, err := w.reservePaceSlot(ctx, key, 2)
		require.NoError(t, err)
		assert.True(t, ok, "slot %d should be granted", i+1)
		assert.Zero(t, wait)
	}
	// ...the third is blocked with a wait bounded by the window.
	ok, wait, err := w.reservePaceSlot(ctx, key, 2)
	require.NoError(t, err)
	assert.False(t, ok, "third send must be paced")
	assert.Greater(t, wait, time.Duration(0))
	assert.LessOrEqual(t, wait, paceWindow)
}
