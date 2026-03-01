package whatsmeow

import (
	"context"
	"errors"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/types"
)

type mockChatPresenceSender struct {
	calls []types.ChatPresence
	err   error
}

func (m *mockChatPresenceSender) SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error {
	_ = ctx
	_ = jid
	_ = media
	m.calls = append(m.calls, state)
	if m.err != nil {
		return m.err
	}
	return nil
}

func TestTypingIndicatorComputeDelayWithinBounds(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       120,
		TypingMaxDelayMs:       600,
		TypingCharDelayMs:      40,
		TypingCooldownMs:       800,
	})
	require.NotNil(t, planner)
	planner.random = mrand.New(mrand.NewSource(1)) //nolint:gosec

	delay := planner.computeDelay("hello this is a long enough preview")

	assert.GreaterOrEqual(t, delay, 120*time.Millisecond)
	assert.LessOrEqual(t, delay, 600*time.Millisecond)
}

func TestTypingIndicatorCooldownPerChat(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       100,
		TypingMaxDelayMs:       300,
		TypingCharDelayMs:      20,
		TypingCooldownMs:       1200,
	})

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	planner.now = func() time.Time { return now }

	direct := types.NewJID("201234567890", "s.whatsapp.net")
	otherDirect := types.NewJID("201111111111", "s.whatsapp.net")

	assert.True(t, planner.shouldSimulate(context.Background(), direct, "hello"))
	assert.False(t, planner.shouldSimulate(context.Background(), direct, "hello again"))
	assert.True(t, planner.shouldSimulate(context.Background(), otherDirect, "hello other"))

	now = now.Add(1200 * time.Millisecond)
	assert.True(t, planner.shouldSimulate(context.Background(), direct, "allowed after cooldown"))
}

func TestTypingIndicatorSkipsGroupsChannelsAndEmptyPreview(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       100,
		TypingMaxDelayMs:       300,
		TypingCharDelayMs:      20,
		TypingCooldownMs:       1200,
	})

	group := types.NewJID("12345", "g.us")
	channel := types.NewJID("abcd", "newsletter")
	direct := types.NewJID("201234567890", "s.whatsapp.net")

	assert.False(t, planner.shouldSimulate(context.Background(), group, "hello group"))
	assert.False(t, planner.shouldSimulate(context.Background(), channel, "hello channel"))
	assert.False(t, planner.shouldSimulate(context.Background(), direct, "   "))
}

func TestTypingIndicatorSkipsWhenContextDisablesIt(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       100,
		TypingMaxDelayMs:       300,
		TypingCharDelayMs:      20,
		TypingCooldownMs:       1200,
	})
	direct := types.NewJID("201234567890", "s.whatsapp.net")

	ctx := provider.WithSkipTypingIndicator(context.Background())
	assert.False(t, planner.shouldSimulate(ctx, direct, "hello"))
}

func TestTypingIndicatorPresenceFailureDoesNotPanicOrSleep(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       100,
		TypingMaxDelayMs:       300,
		TypingCharDelayMs:      20,
		TypingCooldownMs:       1200,
	})
	sender := &mockChatPresenceSender{err: errors.New("presence failed")}
	slept := false
	planner.sleep = func(ctx context.Context, d time.Duration) error {
		_ = ctx
		_ = d
		slept = true
		return nil
	}

	planner.simulate(context.Background(), sender, types.NewJID("201234567890", "s.whatsapp.net"), "hello")

	assert.Equal(t, []types.ChatPresence{types.ChatPresenceComposing}, sender.calls)
	assert.False(t, slept)
}

func TestTypingIndicatorSimulateSendsComposingThenPaused(t *testing.T) {
	planner := newTypingIndicatorPlanner(&config.WhatsmeowConfig{
		TypingIndicatorEnabled: true,
		TypingMinDelayMs:       100,
		TypingMaxDelayMs:       300,
		TypingCharDelayMs:      20,
		TypingCooldownMs:       1200,
	})
	sender := &mockChatPresenceSender{}
	planner.sleep = func(ctx context.Context, d time.Duration) error {
		_ = ctx
		assert.GreaterOrEqual(t, d, 100*time.Millisecond)
		assert.LessOrEqual(t, d, 300*time.Millisecond)
		return nil
	}

	planner.simulate(context.Background(), sender, types.NewJID("201234567890", "s.whatsapp.net"), "hello")

	assert.Equal(t, []types.ChatPresence{types.ChatPresenceComposing, types.ChatPresencePaused}, sender.calls)
}
