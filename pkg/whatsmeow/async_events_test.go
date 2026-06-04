package whatsmeow

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestAsyncEventDispatcherDispatchReturnsWhileHandlerBlocked(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})

	d := newAsyncEventDispatcher(2, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if evt == "block" {
			close(handlerStarted)
			<-releaseHandler
		}
	}, nil, nil)
	defer d.StopAll()

	require.True(t, d.Dispatch("block", instanceID, orgID))
	require.Eventually(t, func() bool {
		select {
		case <-handlerStarted:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	started := time.Now()
	require.True(t, d.Dispatch("queued", instanceID, orgID))
	assert.Less(t, time.Since(started), 50*time.Millisecond)
	close(releaseHandler)
}

func TestAsyncEventDispatcherPreservesPerInstanceFIFO(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	done := make(chan struct{})

	var mu sync.Mutex
	seen := make([]int, 0, 5)
	d := newAsyncEventDispatcher(8, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, evt.(int))
		if len(seen) == 5 {
			close(done)
		}
	}, nil, nil)
	defer d.StopAll()

	for i := 0; i < 5; i++ {
		require.True(t, d.Dispatch(i, instanceID, orgID))
	}

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{0, 1, 2, 3, 4}, seen)
}

func TestAsyncEventDispatcherProcessesInstancesIndependently(t *testing.T) {
	blockedInstanceID := uuid.New()
	freeInstanceID := uuid.New()
	orgID := uuid.New()
	blockedStarted := make(chan struct{})
	releaseBlocked := make(chan struct{})
	freeProcessed := make(chan struct{})

	d := newAsyncEventDispatcher(2, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if instanceID == blockedInstanceID {
			close(blockedStarted)
			<-releaseBlocked
			return
		}
		if instanceID == freeInstanceID {
			close(freeProcessed)
		}
	}, nil, nil)
	defer d.StopAll()

	require.True(t, d.Dispatch("blocked", blockedInstanceID, orgID))
	require.Eventually(t, func() bool {
		select {
		case <-blockedStarted:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.True(t, d.Dispatch("free", freeInstanceID, orgID))
	require.Eventually(t, func() bool {
		select {
		case <-freeProcessed:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	close(releaseBlocked)
}

func TestAsyncEventDispatcherDropsWhenBufferFull(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})

	var dropped uint64
	d := newAsyncEventDispatcher(1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if evt == "block" {
			close(handlerStarted)
			<-releaseHandler
		}
	}, nil, func(uuid.UUID) {
		dropped++
	})
	defer d.StopAll()

	require.True(t, d.Dispatch("block", instanceID, orgID))
	require.Eventually(t, func() bool {
		select {
		case <-handlerStarted:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.True(t, d.Dispatch("queued", instanceID, orgID))
	started := time.Now()
	assert.False(t, d.Dispatch("drop", instanceID, orgID))
	assert.Less(t, time.Since(started), 50*time.Millisecond)
	assert.Equal(t, uint64(1), dropped)
	close(releaseHandler)
}

func TestAsyncEventDispatcherStopInstanceDrainsQueueAndStopAllClosesDispatcher(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	processed := make(chan string, 2)

	d := newAsyncEventDispatcher(4, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		processed <- evt.(string)
	}, nil, nil)

	require.True(t, d.Dispatch("first", instanceID, orgID))
	require.True(t, d.Dispatch("second", instanceID, orgID))
	d.StopInstance(instanceID)

	require.Eventually(t, func() bool {
		return len(processed) == 2
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, "first", <-processed)
	assert.Equal(t, "second", <-processed)

	d.StopAll()
	assert.False(t, d.Dispatch("after-stop", instanceID, orgID))
}
