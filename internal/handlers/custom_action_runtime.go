package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/redis/go-redis/v9"
)

const (
	customActionRedirectTokenPrefix = "whatomate:custom_action:redirect:"
	customActionRedirectTokenTTL    = 30 * time.Second
	defaultJavaScriptTimeout        = 2 * time.Second
	maxJavaScriptTimeout            = 10 * time.Second
)

var (
	errJavaScriptTimeout = errors.New("javascript execution timed out")

	// Backward-compatible in-memory fallback when Redis is unavailable in tests/single-node dev.
	redirectTokens     = make(map[string]redirectToken)
	redirectTokenMutex sync.RWMutex
)

type redirectToken struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (a *App) saveRedirectToken(ctx context.Context, token string, data redirectToken) error {
	if a.Redis != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal redirect token: %w", err)
		}
		return a.Redis.Set(ctx, customActionRedirectTokenPrefix+token, payload, customActionRedirectTokenTTL).Err()
	}

	redirectTokenMutex.Lock()
	redirectTokens[token] = data
	redirectTokenMutex.Unlock()
	return nil
}

func (a *App) consumeRedirectToken(ctx context.Context, token string) (redirectToken, bool, error) {
	if a.Redis != nil {
		raw, err := a.Redis.GetDel(ctx, customActionRedirectTokenPrefix+token).Result()
		if errors.Is(err, redis.Nil) {
			return redirectToken{}, false, nil
		}
		if err != nil {
			return redirectToken{}, false, fmt.Errorf("failed to load redirect token: %w", err)
		}

		var rt redirectToken
		if err := json.Unmarshal([]byte(raw), &rt); err != nil {
			return redirectToken{}, false, fmt.Errorf("failed to decode redirect token: %w", err)
		}
		if time.Now().After(rt.ExpiresAt) {
			return redirectToken{}, false, nil
		}
		return rt, true, nil
	}

	redirectTokenMutex.Lock()
	rt, exists := redirectTokens[token]
	if exists {
		delete(redirectTokens, token)
	}
	redirectTokenMutex.Unlock()
	if !exists || time.Now().After(rt.ExpiresAt) {
		return redirectToken{}, false, nil
	}
	return rt, true, nil
}

func resolveJavaScriptTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return defaultJavaScriptTimeout
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeout > maxJavaScriptTimeout {
		return maxJavaScriptTimeout
	}
	return timeout
}

func runJavaScriptWithTimeout(vm *goja.Runtime, script string, timeout time.Duration) (goja.Value, error) {
	type result struct {
		val goja.Value
		err error
	}

	done := make(chan result, 1)
	go func() {
		val, err := vm.RunString(script)
		done <- result{val: val, err: err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-done:
		return res.val, res.err
	case <-timer.C:
		vm.Interrupt(errJavaScriptTimeout)
		res := <-done
		if res.err != nil {
			if errors.Is(res.err, errJavaScriptTimeout) || strings.Contains(strings.ToLower(res.err.Error()), "interrupted") {
				return nil, fmt.Errorf("javascript execution timed out after %s", timeout)
			}
			return nil, res.err
		}
		return nil, fmt.Errorf("javascript execution timed out after %s", timeout)
	}
}
