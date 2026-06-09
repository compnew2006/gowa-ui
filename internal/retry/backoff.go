package retry

import (
	"context"
	"time"
)

func WithBackoff[T any](
	ctx context.Context,
	retryCount int,
	delay time.Duration,
	fn func(context.Context) (T, error),
	onRetry func(attempt int, err error),
) (T, error) {
	result, err := fn(ctx)
	if err == nil {
		return result, nil
	}

	for attempt := 1; attempt <= retryCount; attempt++ {
		if onRetry != nil {
			onRetry(attempt, err)
		}

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			var zero T
			return zero, ctx.Err()
		}

		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}
	}

	var zero T
	return zero, err
}
