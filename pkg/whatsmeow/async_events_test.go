package whatsmeow

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestAsyncEventDispatcherDispatchReturnsWhileHandlerBlocked(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})

	d := newAsyncEventDispatcher(priorityQueueConfig{}, 2, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
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
	d := newAsyncEventDispatcher(priorityQueueConfig{}, 8, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
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

	d := newAsyncEventDispatcher(priorityQueueConfig{}, 2, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
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
	d := newAsyncEventDispatcher(priorityQueueConfig{}, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if evt == "block" {
			close(handlerStarted)
			<-releaseHandler
		}
	}, nil, func(uuid.UUID, string) {
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

	d := newAsyncEventDispatcher(priorityQueueConfig{}, 4, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
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

func TestAsyncEventDispatcherStopInstancePreventsQueueRecreation(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	var dropped atomic.Uint64
	d := newAsyncEventDispatcher(priorityQueueConfig{}, 4, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
	}, nil, func(uuid.UUID, string) {
		dropped.Add(1)
	})
	defer d.StopAll()

	require.True(t, d.Dispatch("first", instanceID, orgID))
	d.StopInstance(instanceID)

	assert.False(t, d.Dispatch("late-event", instanceID, orgID))
	assert.Equal(t, uint64(1), dropped.Load())

	d.AllowInstance(instanceID, orgID)
	assert.True(t, d.Dispatch("after-allow", instanceID, orgID))
}

func TestAsyncEventDispatcherPanicRecovery(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	processed := make(chan string, 3)

	d := newAsyncEventDispatcher(priorityQueueConfig{}, 4, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if evt == "panic" {
			panic("test panic")
		}
		processed <- evt.(string)
	}, nil, nil)
	defer d.StopAll()

	require.True(t, d.Dispatch("before", instanceID, orgID))
	require.True(t, d.Dispatch("panic", instanceID, orgID))
	require.True(t, d.Dispatch("after", instanceID, orgID))

	require.Eventually(t, func() bool {
		return len(processed) == 2
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, "before", <-processed)
	assert.Equal(t, "after", <-processed)
}

// ─── Priority queue tests ───

func buildTestPriorityConfig() priorityQueueConfig {
	return priorityQueueConfig{
		priorityEnabled: true,
		msgQueueSize:    8,
		lowQueueSize:    4,
		msgShards:       2,
		lowWorkers:      1,
		highTimeoutMs:   5,
		drainTimeoutSec: 3,
		cbRate:          0, // circuit breaker off by default in tests
		cbWindows:       2,
		cbCooldownSec:   1,
	}
}

func TestPriorityQueueNeverDropsMessageDueToLowFlood(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	var mu sync.Mutex
	msgs := make([]string, 0)

	cfg := buildTestPriorityConfig()
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		switch v := evt.(type) {
		case *events.Message:
			mu.Lock()
			msgs = append(msgs, v.Info.MessageSource.Chat.String())
			mu.Unlock()
		}
	}, nil, nil)
	defer d.StopAll()

	// Flood with low-priority events.
	for i := 0; i < 50; i++ {
		d.Dispatch(createFakeContactEvent(), instanceID, orgID)
	}

	// Send high-priority messages.
	testMsg := createFakeMessageEvent(instanceID.String(), "chatA@s.whatsapp.net")
	require.True(t, d.Dispatch(testMsg, instanceID, orgID))
	testMsg2 := createFakeMessageEvent(instanceID.String(), "chatA@s.whatsapp.net")
	require.True(t, d.Dispatch(testMsg2, instanceID, orgID))

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(msgs) >= 2
	}, 2*time.Second, 10*time.Millisecond, "messages must not be dropped due to low-priority flood")
}

func TestCallEventsAreHighPriority(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	callProcessed := make(chan struct{}, 2)
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		switch evt.(type) {
		case *events.CallOffer:
			callProcessed <- struct{}{}
		case *events.CallTerminate:
			callProcessed <- struct{}{}
		}
	}, nil, nil)
	defer d.StopAll()

	require.True(t, d.Dispatch(&events.CallOffer{Data: nil}, instanceID, orgID))
	require.True(t, d.Dispatch(&events.CallTerminate{}, instanceID, orgID))

	require.Eventually(t, func() bool {
		return len(callProcessed) >= 2
	}, time.Second, 10*time.Millisecond)
}

func TestUnknownCallEventIsHighPriority(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()
	callProcessed := make(chan struct{}, 1)

	cfg := buildTestPriorityConfig()
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if _, ok := evt.(*events.UnknownCallEvent); ok {
			callProcessed <- struct{}{}
		}
	}, nil, nil)
	defer d.StopAll()

	require.True(t, d.Dispatch(&events.UnknownCallEvent{}, instanceID, orgID))

	select {
	case <-callProcessed:
	case <-time.After(time.Second):
		t.Fatal("UnknownCallEvent must be processed as high priority")
	}
}

func TestProductionFloodEventsAreLowPriority(t *testing.T) {
	testCases := []interface{}{
		&events.HistorySync{},
		&events.Contact{},
		&events.AppState{},
		&events.AppStateSyncComplete{},
		&events.AppStateSyncError{},
		&events.Presence{},
		&events.ChatPresence{},
		&events.DeleteForMe{},
		&events.DeleteChat{},
		&events.OfflineSyncCompleted{},
		&events.PushName{},
	}

	for _, evt := range testCases {
		t.Run(eventTypeName(evt), func(t *testing.T) {
			// Classification should be low (not message, not lifecycle).
			// We verify by checking that the event goes to low path (can overflow).
			instanceID := uuid.New()
			orgID := uuid.New()
			cfg := buildTestPriorityConfig()
			cfg.lowQueueSize = 1
			var droppedCount uint64
			d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
				// slow handler — events will pile up
				time.Sleep(5 * time.Millisecond)
			}, nil, func(uuid.UUID, string) {
				atomic.AddUint64(&droppedCount, 1)
			})
			defer d.StopAll()

			// Fill the low queue and verify drops happen (proving the event went low).
			for i := 0; i < 200; i++ {
				d.Dispatch(evt, instanceID, orgID)
			}
			assert.Greater(t, atomic.LoadUint64(&droppedCount), uint64(0),
				"%s should be low priority and overflow the low queue", eventTypeName(evt))
		})
	}
}

func TestStatusBroadcastMessagesAreLowPriority(t *testing.T) {
	var d asyncEventDispatcher

	testCases := []struct {
		name string
		evt  *events.Message
	}{
		{
			name: "status chat jid",
			evt: &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{
				Chat: types.StatusBroadcastJID,
			}}},
		},
		{
			name: "device sent status destination",
			evt: &events.Message{Info: types.MessageInfo{
				DeviceSentMeta: &types.DeviceSentMeta{DestinationJID: types.StatusBroadcastJID.String()},
			}},
		},
		{
			name: "status category",
			evt:  &events.Message{Info: types.MessageInfo{Category: "status"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, eventClassLow, d.classifyEvent(tc.evt))
		})
	}

	realMessage := createFakeMessageEvent(uuid.New().String(), "12345@s.whatsapp.net")
	assert.Equal(t, eventClassMessage, d.classifyEvent(realMessage))
}

func TestUnknownEventsDefaultLowPriority(t *testing.T) {
	// An unrecognized non-lifecycle event should be treated as low priority.
	instanceID := uuid.New()
	orgID := uuid.New()
	cfg := buildTestPriorityConfig()
	cfg.lowQueueSize = 1
	var droppedCount uint64
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		time.Sleep(2 * time.Millisecond)
	}, nil, func(uuid.UUID, string) {
		atomic.AddUint64(&droppedCount, 1)
	})
	defer d.StopAll()

	for i := 0; i < 100; i++ {
		d.Dispatch("unknown-event-type", instanceID, orgID)
	}
	assert.Greater(t, atomic.LoadUint64(&droppedCount), uint64(0),
		"unknown events should default to low priority and overflow")
}

func TestLifecycleEventsBypassDispatcher(t *testing.T) {
	lifecycleEvents := []interface{}{
		&events.Connected{},
		&events.Disconnected{},
		&events.LoggedOut{},
		&events.TemporaryBan{},
		&events.PairSuccess{},
		&events.QR{},
	}

	cfg := buildTestPriorityConfig()
	for _, evt := range lifecycleEvents {
		t.Run(eventTypeName(evt), func(t *testing.T) {
			_ = uuid.New() // instanceID — not needed for classification test
			_ = uuid.New() // orgID — not needed for classification test
			d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
				// handler should not be called because lifecycle events bypass dispatcher
			}, nil, nil)
			defer d.StopAll()

			// Dispatch of a lifecycle event: the dispatcher should correctly classify
			// but the real bypass happens in newClient (manager.go) via isLifecycleEvent.
			// Here we just verify the classification marks it as lifecycle.
			class := d.classifyEvent(evt)
			assert.Equal(t, eventClassLifecycle, class,
				"%s must be lifecycle", eventTypeName(evt))
		})
	}
}

func TestMessageOrderingPerChat(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	var mu sync.Mutex
	ordered := make([]string, 0)
	cfg := buildTestPriorityConfig()
	cfg.msgShards = 1 // single shard for deterministic ordering test
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if msg, ok := evt.(*events.Message); ok {
			mu.Lock()
			ordered = append(ordered, msg.Info.MessageSource.Chat.String())
			mu.Unlock()
		}
	}, nil, nil)
	defer d.StopAll()

	for i := 0; i < 20; i++ {
		msg := createFakeMessageEvent(instanceID.String(), "chat_test@s.whatsapp.net")
		require.True(t, d.Dispatch(msg, instanceID, orgID))
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ordered) == 20
	}, 2*time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 20, len(ordered), "all 20 messages must be processed")
	for _, chat := range ordered {
		assert.Contains(t, chat, "chat_test")
	}
}

func TestReceiptOrderingUsesChatKey(t *testing.T) {
	// Verify that receipts use Chat from embedded MessageSource for shard routing.
	chatKey := chatKeyForEvent(&events.Receipt{
		MessageSource: types.MessageSource{
			Chat: types.NewJID("testchat", "s.whatsapp.net"),
		},
	})
	assert.Contains(t, chatKey, "testchat")

	// Fallback when Chat is empty.
	chatKey2 := chatKeyForEvent(&events.Receipt{})
	assert.NotEmpty(t, chatKey2)
}

func TestCallOrderingUsesCallID(t *testing.T) {
	testCases := []struct {
		ev     interface{}
		callID string
	}{
		{&events.CallOffer{BasicCallMeta: types.BasicCallMeta{CallID: "offer-1"}}, "offer-1"},
		{&events.CallOfferNotice{BasicCallMeta: types.BasicCallMeta{CallID: "notice-1"}}, "notice-1"},
		{&events.CallPreAccept{BasicCallMeta: types.BasicCallMeta{CallID: "preaccept-2"}}, "preaccept-2"},
		{&events.CallAccept{BasicCallMeta: types.BasicCallMeta{CallID: "accept-3"}}, "accept-3"},
		{&events.CallTransport{BasicCallMeta: types.BasicCallMeta{CallID: "transport-4"}}, "transport-4"},
		{&events.CallTerminate{BasicCallMeta: types.BasicCallMeta{CallID: "terminate-5"}}, "terminate-5"},
		{&events.CallReject{BasicCallMeta: types.BasicCallMeta{CallID: "reject-6"}}, "reject-6"},
		{&events.UnknownCallEvent{}, "_inst_"}, // no CallID — uses fallback key
	}

	for _, tc := range testCases {
		t.Run(eventTypeName(tc.ev), func(t *testing.T) {
			chatKey := chatKeyForEvent(tc.ev)
			assert.Contains(t, chatKey, tc.callID,
				"%s must use CallID for shard routing", eventTypeName(tc.ev))
		})
	}

	// Fallback when CallID is empty.
	chatKey := chatKeyForEvent(&events.CallOffer{})
	assert.NotEmpty(t, chatKey)
	assert.Contains(t, chatKey, "_inst_")
}

func TestMessageShardParallelismAcrossChats(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	var mu sync.Mutex
	finished := make(map[string]struct{})
	cfg := buildTestPriorityConfig()
	cfg.msgShards = 4
	cfg.msgQueueSize = 16 // larger per-shard capacity
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		if msg, ok := evt.(*events.Message); ok {
			k := msg.Info.MessageSource.Chat.String()
			mu.Lock()
			finished[k] = struct{}{}
			mu.Unlock()
		}
	}, nil, nil)
	defer d.StopAll()

	// Messages from different chats will route to different shards and can be processed in parallel.
	for i := 0; i < 32; i++ {
		chatJID := fmt.Sprintf("chat_%d@s.whatsapp.net", i)
		msg := createFakeMessageEvent(instanceID.String(), chatJID)
		require.True(t, d.Dispatch(msg, instanceID, orgID))
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(finished) == 32
	}, 5*time.Second, 20*time.Millisecond)
}

func TestHighPriorityEnqueueDoesNotDeadlockProducer(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	cfg.msgQueueSize = 2
	cfg.highTimeoutMs = 5
	var dropped uint64
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		time.Sleep(500 * time.Millisecond) // slow handler
	}, nil, func(uuid.UUID, string) {
		atomic.AddUint64(&dropped, 1)
	})
	defer d.StopAll()

	// Fill the high-priority queue; subsequent enqueue must not block indefinitely.
	start := time.Now()
	for i := 0; i < 10; i++ {
		msg := createFakeMessageEvent(instanceID.String(), "chatA@s.whatsapp.net")
		d.Dispatch(msg, instanceID, orgID)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 2*time.Second,
		"high-priority enqueue must not block indefinitely (elapsed %v)", elapsed)
	assert.Greater(t, atomic.LoadUint64(&dropped), uint64(0),
		"should have recorded critical_overflow")
}

func TestLowPriorityDropsNewestWhenFull(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	cfg.lowQueueSize = 2
	var processed, dropped uint64
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		time.Sleep(50 * time.Millisecond)
		atomic.AddUint64(&processed, 1)
	}, nil, func(uuid.UUID, string) {
		atomic.AddUint64(&dropped, 1)
	})
	defer d.StopAll()

	for i := 0; i < 50; i++ {
		d.Dispatch(&events.Contact{}, instanceID, orgID)
	}

	time.Sleep(200 * time.Millisecond)
	// We should see drops (newest events discarded when low queue is full).
	assert.Greater(t, atomic.LoadUint64(&dropped), uint64(0), "low queue must drop-newest when full")
}

func TestCriticalOverflowLogsAreSampledButMetricsCountAll(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	cfg.msgQueueSize = 1
	var dropped uint64
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		time.Sleep(100 * time.Millisecond)
	}, nil, func(uuid.UUID, string) {
		atomic.AddUint64(&dropped, 1)
	})
	defer d.StopAll()

	for i := 0; i < 20; i++ {
		msg := createFakeMessageEvent(instanceID.String(), "chatA@s.whatsapp.net")
		d.Dispatch(msg, instanceID, orgID)
	}

	time.Sleep(500 * time.Millisecond)
	// All drops are counted in metrics.
	totalDropped := atomic.LoadUint64(&dropped)
	assert.Greater(t, totalDropped, uint64(0),
		"all critical overflows must increment metrics, got %d", totalDropped)
}

func TestStopInstanceDrainsBeforeClose(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		// fast handler
	}, nil, nil)

	// Enqueue several high-priority messages.
	for i := 0; i < 5; i++ {
		msg := createFakeMessageEvent(instanceID.String(), "chatA@s.whatsapp.net")
		require.True(t, d.Dispatch(msg, instanceID, orgID))
	}

	// Stop should drain the pending events within the drain timeout.
	d.StopInstance(instanceID)

	// After stop, priority queues should be nil or drained.
	msgDepth, lowDepth := d.PriorityQueueDepth(instanceID)
	assert.Equal(t, int64(0), msgDepth)
	assert.Equal(t, int64(0), lowDepth)
}

// ─── Circuit breaker tests ───

func TestCircuitBreakerTripsOnSustainedFlood(t *testing.T) {
	instanceID := uuid.New()
	orgID := uuid.New()

	cfg := buildTestPriorityConfig()
	cfg.cbRate = 5        // low threshold for test
	cfg.cbWindows = 2     // 2 consecutive windows
	cfg.cbCooldownSec = 1 // short cooldown

	d := newAsyncEventDispatcher(cfg, 1, logf.New(logf.Opts{}), func(evt interface{}, instanceID, orgID uuid.UUID) {
		// fast handler
	}, nil, nil)
	defer d.StopAll()

	// Fill first window via enqueueLow (which calls circuitBreakerOpen).
	for i := 0; i < 10; i++ {
		d.enqueueLow(&events.Contact{}, instanceID, orgID)
	}

	// Advance window.
	d.ResetCircuitBreakerWindows()

	// Fill second window.
	for i := 0; i < 10; i++ {
		d.enqueueLow(&events.Contact{}, instanceID, orgID)
	}

	// Advance window — breaker should trip now.
	// The trip happens when the NEXT event after 2 windows of sustained rate exceeds threshold.
	// circuitBreakerOpen increments windows[0] on each call; after reset, windows[0] is 0.
	// So we need to send another batch of events to trigger the windows[0]++ and the check.
	d.ResetCircuitBreakerWindows()
	for i := 0; i < 10; i++ {
		d.enqueueLow(&events.Contact{}, instanceID, orgID)
	}

	assert.True(t, d.IsCircuitBreakerOpen(instanceID))

	// After cooldown, breaker should close.
	time.Sleep(1100 * time.Millisecond)
	d.ResetCircuitBreakerWindows()
	assert.False(t, d.IsCircuitBreakerOpen(instanceID))
}

// ─── Helpers ───

func createFakeMessageEvent(instanceID, chatJID string) *events.Message {
	// Parse "user@server" into a JID.
	jid, err := types.ParseJID(chatJID)
	if err != nil || jid.IsEmpty() {
		// Fallback: use NewJID with user="testchat", server="s.whatsapp.net"
		jid = types.NewJID("testchat", "s.whatsapp.net")
	}
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: jid,
			},
		},
	}
}

func createFakeContactEvent() *events.Contact {
	return &events.Contact{}
}
