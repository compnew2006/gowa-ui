package whatsmeow

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/pkg/provider"
	"go.mau.fi/whatsmeow/types"
)

type chatPresenceSender interface {
	SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error
}

type typingIndicatorPlanner struct {
	enabled   bool
	minDelay  time.Duration
	maxDelay  time.Duration
	charDelay time.Duration
	cooldown  time.Duration
	minChars  int

	mu         sync.Mutex
	lastByChat map[string]time.Time
	random     *rand.Rand
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error
	warn       func(string, ...any)
}

func newTypingIndicatorPlanner(cfg *config.WhatsmeowConfig) *typingIndicatorPlanner {
	planner := &typingIndicatorPlanner{
		enabled:    cfg != nil && cfg.TypingIndicatorEnabled,
		minDelay:   durationFromMs(cfg, 700, func(c *config.WhatsmeowConfig) int { return c.TypingMinDelayMs }),
		maxDelay:   durationFromMs(cfg, 3000, func(c *config.WhatsmeowConfig) int { return c.TypingMaxDelayMs }),
		charDelay:  durationFromMs(cfg, 35, func(c *config.WhatsmeowConfig) int { return c.TypingCharDelayMs }),
		cooldown:   durationFromMs(cfg, 4000, func(c *config.WhatsmeowConfig) int { return c.TypingCooldownMs }),
		minChars:   3,
		lastByChat: make(map[string]time.Time),
		random:     rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec
		now:        func() time.Time { return time.Now().UTC() },
		sleep:      sleepWithTypingContext,
		warn:       func(string, ...any) {},
	}

	if planner.maxDelay < planner.minDelay {
		planner.maxDelay = planner.minDelay
	}
	if planner.charDelay < 0 {
		planner.charDelay = 0
	}
	if planner.cooldown < 0 {
		planner.cooldown = 0
	}

	return planner
}

func durationFromMs(cfg *config.WhatsmeowConfig, fallback int, getValue func(*config.WhatsmeowConfig) int) time.Duration {
	if cfg == nil {
		return time.Duration(fallback) * time.Millisecond
	}
	value := getValue(cfg)
	if value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Millisecond
}

func (p *typingIndicatorPlanner) simulate(ctx context.Context, client chatPresenceSender, chatJID types.JID, previewText string) {
	if p == nil || client == nil {
		return
	}
	if !p.shouldSimulate(ctx, chatJID, previewText) {
		return
	}

	if err := client.SendChatPresence(ctx, chatJID, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil {
		p.warn("Typing indicator composing presence failed", "chat_jid", chatJID.String(), "error", err)
		return
	}

	delay := p.computeDelay(previewText)
	if delay > 0 {
		if err := p.sleep(ctx, delay); err != nil {
			return
		}
	}

	_ = client.SendChatPresence(ctx, chatJID, types.ChatPresencePaused, "")
}

func (p *typingIndicatorPlanner) shouldSimulate(ctx context.Context, chatJID types.JID, previewText string) bool {
	if !p.enabled || provider.ShouldSkipTypingIndicator(ctx) {
		return false
	}
	if !isDirectChatJID(chatJID) {
		return false
	}
	if strings.TrimSpace(previewText) == "" {
		return false
	}
	if p.minChars > 0 && utf8.RuneCountInString(strings.TrimSpace(previewText)) < p.minChars {
		return false
	}

	chatKey := chatJID.String()
	now := p.now()

	p.mu.Lock()
	defer p.mu.Unlock()

	lastAt, ok := p.lastByChat[chatKey]
	if ok && p.cooldown > 0 && now.Sub(lastAt) < p.cooldown {
		return false
	}
	p.lastByChat[chatKey] = now
	return true
}

func (p *typingIndicatorPlanner) computeDelay(previewText string) time.Duration {
	trimmed := strings.TrimSpace(previewText)
	runesCount := utf8.RuneCountInString(trimmed)
	if runesCount < 0 {
		runesCount = 0
	}

	delay := p.minDelay + (time.Duration(runesCount) * p.charDelay)
	if delay < p.minDelay {
		delay = p.minDelay
	}
	if delay > p.maxDelay {
		delay = p.maxDelay
	}
	if p.maxDelay > delay {
		jitterCap := p.charDelay
		if jitterCap <= 0 {
			jitterCap = 120 * time.Millisecond
		}
		remaining := p.maxDelay - delay
		if jitterCap > remaining {
			jitterCap = remaining
		}
		if jitterCap > 0 {
			delay += time.Duration(p.random.Int63n(int64(jitterCap + time.Millisecond)))
		}
	}
	if delay > p.maxDelay {
		delay = p.maxDelay
	}
	return delay
}

func isDirectChatJID(jid types.JID) bool {
	server := strings.ToLower(strings.TrimSpace(jid.Server))
	switch server {
	case "s.whatsapp.net", "lid":
		return true
	default:
		return false
	}
}

func sleepWithTypingContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
