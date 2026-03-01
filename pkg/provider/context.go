package provider

import "context"

type skipTypingIndicatorContextKey struct{}

// WithSkipTypingIndicator marks provider sends to bypass typing indicators.
func WithSkipTypingIndicator(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, skipTypingIndicatorContextKey{}, true)
}

// ShouldSkipTypingIndicator returns true when typing indicator should be skipped.
func ShouldSkipTypingIndicator(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, ok := ctx.Value(skipTypingIndicatorContextKey{}).(bool)
	return ok && value
}
