package whatsmeow

import (
	"context"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestQueueJobReceivesTimeoutBoundContext(t *testing.T) {
	manager := NewQueueManager(config.WhatsmeowConfig{
		RateLimitMinDelayMs: 1,
		RateLimitMaxDelayMs: 1,
		QueueTimeoutSeconds: 1,
	}, logf.New(logf.Opts{}))
	t.Cleanup(manager.Close)

	ctxDone := make(chan error, 4)
	require.NoError(t, manager.Enqueue("instance-timeout", func(ctx context.Context) error {
		<-ctx.Done()
		err := ctx.Err()
		ctxDone <- err
		return err
	}))

	select {
	case err := <-ctxDone:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for bounded job context deadline")
	}
}

func TestQueueDepthObserverTracksEnqueueAndDequeue(t *testing.T) {
	manager := NewQueueManager(config.WhatsmeowConfig{
		RateLimitMinDelayMs: 1,
		RateLimitMaxDelayMs: 1,
		QueueTimeoutSeconds: 5,
	}, logf.New(logf.Opts{}))
	t.Cleanup(manager.Close)

	depths := make(chan int64, 16)
	manager.SetDepthObserver(func(instanceID string, depth int64) {
		if instanceID == "instance-depth" {
			depths <- depth
		}
	})

	release := make(chan struct{})
	done := make(chan struct{})
	require.NoError(t, manager.Enqueue("instance-depth", func(ctx context.Context) error {
		<-release
		close(done)
		return nil
	}))

	assert.Equal(t, int64(1), waitDepthValue(t, depths, 3*time.Second))
	assert.Equal(t, int64(0), waitDepthValue(t, depths, 3*time.Second))

	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for queue job completion")
	}
}

func waitDepthValue(t *testing.T, depths <-chan int64, timeout time.Duration) int64 {
	t.Helper()
	select {
	case depth := <-depths:
		return depth
	case <-time.After(timeout):
		t.Fatal("timed out waiting for queue depth update")
		return -1
	}
}

func TestQueueDelayRespectsIdleQueue(t *testing.T) {
	// Configure a substantial delay (e.g. 500ms)
	manager := NewQueueManager(config.WhatsmeowConfig{
		RateLimitMinDelayMs: 500,
		RateLimitMaxDelayMs: 500,
		QueueTimeoutSeconds: 5,
	}, logf.New(logf.Opts{}))
	t.Cleanup(manager.Close)

	// Enqueue the first job on a fresh/idle queue
	start := time.Now()
	done1 := make(chan struct{})
	require.NoError(t, manager.Enqueue("instance-delay", func(ctx context.Context) error {
		close(done1)
		return nil
	}))

	select {
	case <-done1:
		duration := time.Since(start)
		// First job should run immediately (under 100ms)
		assert.Less(t, duration, 100*time.Millisecond, "idle queue job should execute immediately without rate-limit sleep")
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for first job execution")
	}

	// Enqueue a second job back-to-back
	start2 := time.Now()
	done2 := make(chan struct{})
	require.NoError(t, manager.Enqueue("instance-delay", func(ctx context.Context) error {
		close(done2)
		return nil
	}))

	select {
	case <-done2:
		duration2 := time.Since(start2)
		// Second job must be delayed by at least 400ms (accounting for scheduling overhead)
		assert.GreaterOrEqual(t, duration2, 400*time.Millisecond, "consecutive job should be rate-limited")
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for second job execution")
	}
}
