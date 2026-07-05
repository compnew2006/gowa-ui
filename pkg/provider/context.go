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

type recipientPhoneContextKey struct{}

// WithRecipientPhone stores the recipient phone number in the context.
func WithRecipientPhone(ctx context.Context, phone string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, recipientPhoneContextKey{}, phone)
}

// GetRecipientPhone retrieves the recipient phone number from the context.
func GetRecipientPhone(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(recipientPhoneContextKey{}).(string)
	if ok {
		return value
	}
	return ""
}
