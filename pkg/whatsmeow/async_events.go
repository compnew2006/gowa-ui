package whatsmeow

import (
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/types/events"
)

const defaultEventBufferSize = 32768

// Overflow log sampling: at most one critical_overflow log per instance per interval.
const overflowLogSampleInterval = 30 * time.Second

// Drop reason/categories reported by enqueueHigh. Both drop branches share the
// reasonCriticalOverflow category (kept stable so existing log matchers/alerts
// still fire and sampling applies uniformly), but carry a distinct queue_state
// so operators can tell a poisoned (stopped) instance — the common auto-reconnect
// bug — from a genuinely saturated shard.
const (
	reasonCriticalOverflow = "critical_overflow" // log event category for both enqueueHigh drop branches
	queueStateStopped      = "instance_stopped"  // priorityQueueFor returned nil: instance in stopped[] or dispatcher closed
	queueStateShardFull    = "shard_full"        // shard channel at capacity past the bounded retry window
	queueStateLowOverflow  = "low_overflow"      // low-priority queue full (drop-newest) or instance stopped
	queueStateCircuitOpen  = "circuit_open"      // low-priority event dropped because the circuit breaker is open
	queueStateLegacy       = "legacy_drop"       // legacy single-queue path drop (PriorityQueuesEnabled=false)
)

type asyncEventHandler func(evt interface{}, instanceID, orgID uuid.UUID)

// ─── Legacy single-queue types (used when PriorityQueuesEnabled=false) ───

type asyncEvent struct {
	evt        interface{}
	instanceID uuid.UUID
	orgID      uuid.UUID
}

type asyncEventQueue struct {
	instanceID uuid.UUID
	orgID      uuid.UUID
	events     chan asyncEvent
	mu         sync.Mutex
	closed     bool
}

// ─── Priority queue types (used when PriorityQueuesEnabled=true) ───

type queuedEvent struct {
	evt        interface{}
	instanceID uuid.UUID
	orgID      uuid.UUID
	enqueuedAt time.Time
	chatKey    string
}

// instanceQueues holds the per-instance priority lanes.
type instanceQueues struct {
	instanceID uuid.UUID
	orgID      uuid.UUID

	msgQueues []chan queuedEvent // one per shard (default 4)
	lowQueue  chan queuedEvent   // single low-priority channel (default 512)

	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup // tracks all workers for this instance

	// Lag tracking: last-dequeued event lag per queue type, in nanoseconds.
	// Updated atomically by workers on every dequeue. Read by PriorityConsumerLag.
	lastMsgLagNano atomic.Int64
	lastLowLagNano atomic.Int64
}

// ─── Dispatcher ───

type asyncEventDispatcher struct {
	// config (legacy path)
	bufferSize int
	// config (priority path)
	priorityEnabled bool
	msgQueueSize    int
	lowQueueSize    int
	msgShards       int
	lowWorkers      int
	highTimeoutMs   int
	drainTimeoutSec int

	// core
	handler     asyncEventHandler
	logger      logf.Logger
	updateDepth func(uuid.UUID, int64)
	markDropped func(uuid.UUID, string)

	// state
	mu      sync.Mutex
	queues  map[uuid.UUID]*asyncEventQueue // legacy: per-instance single queue
	pQueues map[uuid.UUID]*instanceQueues  // priority: per-instance priority lanes
	wg      sync.WaitGroup
	closed  bool
	stopped map[uuid.UUID]struct{}

	// circuit breaker state (priority path only)
	cbMu        sync.Mutex
	lowCounts   map[uuid.UUID][]int64 // rolling window low-priority counts
	cbRate      int                   // rate/minute threshold
	cbWindows   int                   // consecutive windows to trip
	cbCooldown  time.Duration
	cbOpenUntil map[uuid.UUID]time.Time
	cbDone      chan struct{} // closed when dispatcher shuts down

	// overflow log sampler
	overflowLogMu   sync.Mutex
	lastOverflowLog map[uuid.UUID]time.Time
}

// ─── Constructor ───

func newAsyncEventDispatcher(cfg priorityQueueConfig, bufferSize int, logger logf.Logger, handler asyncEventHandler,
	updateDepth func(uuid.UUID, int64), markDropped func(uuid.UUID, string)) *asyncEventDispatcher {

	if bufferSize <= 0 {
		bufferSize = defaultEventBufferSize
	}

	enabled := cfg.priorityEnabled
	msgSize := defaultIfZero(cfg.msgQueueSize, 2048)
	lowSize := defaultIfZero(cfg.lowQueueSize, 512)
	shards := defaultIfZero(cfg.msgShards, 4)
	workers := defaultIfZero(cfg.lowWorkers, 2)
	timeoutMs := defaultIfZero(cfg.highTimeoutMs, 10)
	drainSec := defaultIfZero(cfg.drainTimeoutSec, 5)
	cbRate := defaultIfZero(cfg.cbRate, 60)
	cbWindows := defaultIfZero(cfg.cbWindows, 2)
	cbCooldown := time.Duration(defaultIfZero(cfg.cbCooldownSec, 300)) * time.Second

	d := &asyncEventDispatcher{
		bufferSize:      bufferSize,
		priorityEnabled: enabled,
		msgQueueSize:    msgSize,
		lowQueueSize:    lowSize,
		msgShards:       shards,
		lowWorkers:      workers,
		highTimeoutMs:   timeoutMs,
		drainTimeoutSec: drainSec,

		handler:     handler,
		logger:      logger,
		updateDepth: updateDepth,
		markDropped: markDropped,

		queues:  make(map[uuid.UUID]*asyncEventQueue),
		pQueues: make(map[uuid.UUID]*instanceQueues),
		stopped: make(map[uuid.UUID]struct{}),

		cbRate:          cbRate,
		cbWindows:       cbWindows,
		cbCooldown:      cbCooldown,
		lowCounts:       make(map[uuid.UUID][]int64),
		cbOpenUntil:     make(map[uuid.UUID]time.Time),
		lastOverflowLog: make(map[uuid.UUID]time.Time),
	}

	// Start circuit breaker window ticker when priority queues are enabled.
	if enabled && cbRate > 0 {
		d.cbDone = make(chan struct{})
		go d.circuitBreakerTickerLoop()
	}

	return d
}

// priorityQueueConfig is the subset of config needed for the dispatcher.
// This decouples async_events.go from internal/config.
type priorityQueueConfig struct {
	priorityEnabled bool
	msgQueueSize    int
	lowQueueSize    int
	msgShards       int
	lowWorkers      int
	highTimeoutMs   int
	drainTimeoutSec int
	cbRate          int
	cbWindows       int
	cbCooldownSec   int
}

func defaultIfZero(val, dflt int) int {
	if val <= 0 {
		return dflt
	}
	return val
}

// ─── Dispatch ───

func (d *asyncEventDispatcher) Dispatch(evt interface{}, instanceID, orgID uuid.UUID) bool {
	if d == nil {
		return false
	}

	if d.priorityEnabled {
		return d.priorityDispatch(evt, instanceID, orgID)
	}
	return d.legacyDispatch(evt, instanceID, orgID)
}

// ─── Legacy dispatch path (PriorityQueuesEnabled=false) ───

func (d *asyncEventDispatcher) legacyDispatch(evt interface{}, instanceID, orgID uuid.UUID) bool {
	queue := d.queueFor(instanceID, orgID)
	if queue == nil {
		d.markEventDropped(instanceID, queueStateLegacy)
		d.logEventDropLegacy(instanceID, evt)
		return false
	}

	event := asyncEvent{evt: evt, instanceID: instanceID, orgID: orgID}
	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		d.markEventDropped(instanceID, queueStateLegacy)
		d.logEventDropLegacy(instanceID, evt)
		return false
	}

	select {
	case queue.events <- event:
		d.updateQueueDepth(instanceID, len(queue.events))
		return true
	default:
		d.markEventDropped(instanceID, queueStateLegacy)
		d.logEventDropLegacy(instanceID, evt)
		return false
	}
}

// ─── Priority dispatch path (PriorityQueuesEnabled=true) ───

func (d *asyncEventDispatcher) priorityDispatch(evt interface{}, instanceID, orgID uuid.UUID) bool {
	class := d.classifyEvent(evt)

	switch class {
	case eventClassMessage:
		return d.enqueueHigh(evt, instanceID, orgID)
	case eventClassLow:
		return d.enqueueLow(evt, instanceID, orgID)
	case eventClassLifecycle:
		// should never arrive here — lifecycle events bypass the dispatcher entirely
		return false
	default:
		// Unknown non-lifecycle events default to low priority.
		return d.enqueueLow(evt, instanceID, orgID)
	}
}

type eventPriorityClass int

const (
	eventClassMessage   eventPriorityClass = iota // high — Message, Receipt, Call events
	eventClassLow                                 // low — HistorySync, Contact, AppState, etc.
	eventClassLifecycle                           // bypass — Connected, Disconnected, etc.
)

func (d *asyncEventDispatcher) classifyEvent(evt interface{}) eventPriorityClass {
	switch v := evt.(type) {
	// ── High priority ──
	case *events.Message:
		if isStatusMessageInfo(v.Info) {
			return eventClassLow
		}
		return eventClassMessage
	case *events.Receipt,
		*events.CallOffer,
		*events.CallOfferNotice,
		*events.CallPreAccept,
		*events.CallAccept,
		*events.CallTransport,
		*events.CallTerminate,
		*events.CallReject,
		*events.UnknownCallEvent:
		return eventClassMessage

	// ── Low priority ──
	case *events.HistorySync,
		*events.Contact,
		*events.AppState,
		*events.AppStateSyncComplete,
		*events.AppStateSyncError,
		*events.Presence,
		*events.ChatPresence,
		*events.DeleteForMe,
		*events.DeleteChat,
		*events.OfflineSyncCompleted,
		*events.PushName:
		return eventClassLow

	// ── Lifecycle (should bypass dispatcher) ──
	case *events.Connected,
		*events.Disconnected,
		*events.LoggedOut,
		*events.TemporaryBan,
		*events.PairSuccess,
		*events.QR:
		return eventClassLifecycle

	// ── Unknown non-lifecycle → low (default) ──
	default:
		return eventClassLow
	}
}

// chatKeyForEvent returns a stable string key for shard routing.
func chatKeyForEvent(evt interface{}) string {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.Chat.String() != "" {
			return v.Info.Chat.String()
		}
	case *events.Receipt:
		if v.Chat.String() != "" {
			return v.Chat.String()
		}
	case *events.CallOffer:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallOfferNotice:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallPreAccept:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallAccept:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallTransport:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallTerminate:
		if v.CallID != "" {
			return v.CallID
		}
	case *events.CallReject:
		if v.CallID != "" {
			return v.CallID
		}
	}
	return fmt.Sprintf("_inst_%T", evt)
}

func shardIndex(key string, shards int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32()) % shards
}

// enqueueHigh sends a high-priority event to the chat-sharded msg queue.
// Uses immediate select, then bounded retry; never blocks indefinitely.
func (d *asyncEventDispatcher) enqueueHigh(evt interface{}, instanceID, orgID uuid.UUID) bool {
	queues := d.priorityQueueFor(instanceID, orgID)
	if queues == nil {
		d.markEventDropped(instanceID, queueStateStopped)
		d.logPriorityDrop(instanceID, evt, reasonCriticalOverflow, queueStateStopped)
		return false
	}

	key := chatKeyForEvent(evt)
	idx := shardIndex(key, len(queues.msgQueues))
	target := queues.msgQueues[idx]

	qevt := queuedEvent{
		evt:        evt,
		instanceID: instanceID,
		orgID:      orgID,
		enqueuedAt: time.Now(),
		chatKey:    key,
	}

	// First attempt — non-blocking.
	select {
	case target <- qevt:
		d.updatePriorityQueueDepth(instanceID, d.msgDepth(queues))
		return true
	default:
	}

	// Bounded retry.
	deadline := time.Now().Add(time.Duration(d.highTimeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case target <- qevt:
			d.updatePriorityQueueDepth(instanceID, d.msgDepth(queues))
			return true
		default:
			time.Sleep(100 * time.Microsecond)
		}
	}

	// All attempts failed → critical_overflow (shard saturated).
	d.markEventDropped(instanceID, queueStateShardFull)
	d.logPriorityDrop(instanceID, evt, reasonCriticalOverflow, queueStateShardFull)
	return false
}

// enqueueLow sends a low-priority event. Drop-newest when full (non-blocking).
func (d *asyncEventDispatcher) enqueueLow(evt interface{}, instanceID, orgID uuid.UUID) bool {
	queues := d.priorityQueueFor(instanceID, orgID)
	if queues == nil {
		d.markEventDropped(instanceID, queueStateStopped)
		d.logPriorityDrop(instanceID, evt, "low_overflow", queueStateStopped)
		return false
	}

	// Circuit breaker: if instance is flooding, skip HistorySync only.
	// All other low-priority events are counted as circuit_open drops.
	if d.circuitBreakerOpen(instanceID) {
		if _, isHistorySync := evt.(*events.HistorySync); isHistorySync {
			return false // silently skip HistorySync during cooldown
		}
		// Other low events: count as circuit_open drop but do not enqueue.
		d.markEventDropped(instanceID, queueStateCircuitOpen)
		d.logPriorityDrop(instanceID, evt, "circuit_open", queueStateCircuitOpen)
		return false
	}
	d.recordLowEvent(instanceID)

	qevt := queuedEvent{
		evt:        evt,
		instanceID: instanceID,
		orgID:      orgID,
		enqueuedAt: time.Now(),
		chatKey:    chatKeyForEvent(evt),
	}

	// drop-newest: if queue is full, discard the newest (this) event.
	select {
	case queues.lowQueue <- qevt:
		d.updatePriorityQueueDepth(instanceID, int64(len(queues.lowQueue)))
		return true
	default:
		d.markEventDropped(instanceID, queueStateLowOverflow)
		d.logPriorityDrop(instanceID, evt, "low_overflow", queueStateLowOverflow)
		return false
	}
}

func (d *asyncEventDispatcher) msgDepth(queues *instanceQueues) int64 {
	var total int
	for _, q := range queues.msgQueues {
		total += len(q)
	}
	return int64(total)
}

// ─── Queue creation (priority path) ───

func (d *asyncEventDispatcher) priorityQueueFor(instanceID, orgID uuid.UUID) *instanceQueues {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	if _, stopped := d.stopped[instanceID]; stopped {
		return nil
	}
	if pq := d.pQueues[instanceID]; pq != nil {
		return pq
	}

	shards := d.msgShards
	if shards < 1 {
		shards = 1
	}

	pq := &instanceQueues{
		instanceID: instanceID,
		orgID:      orgID,
		msgQueues:  make([]chan queuedEvent, shards),
		lowQueue:   make(chan queuedEvent, d.lowQueueSize),
	}

	for i := 0; i < shards; i++ {
		pq.msgQueues[i] = make(chan queuedEvent, d.msgQueueSize)
		pq.wg.Add(1)
		go d.msgWorker(pq, i)
	}

	// Low workers
	lowW := d.lowWorkers
	if lowW < 1 {
		lowW = 1
	}
	for i := 0; i < lowW; i++ {
		pq.wg.Add(1)
		go d.lowWorker(pq)
	}

	d.pQueues[instanceID] = pq
	return pq
}

// msgWorker drains a single shard of the high-priority channel.
func (d *asyncEventDispatcher) msgWorker(pq *instanceQueues, shard int) {
	defer pq.wg.Done()
	ch := pq.msgQueues[shard]
	for qevt := range ch {
		pq.lastMsgLagNano.Store(time.Since(qevt.enqueuedAt).Nanoseconds())
		d.safeHandle(queuedEvent{evt: qevt.evt, instanceID: qevt.instanceID, orgID: qevt.orgID})
		d.updatePriorityQueueDepth(qevt.instanceID, d.msgDepth(pq))
	}
}

// lowWorker drains the low-priority channel.
func (d *asyncEventDispatcher) lowWorker(pq *instanceQueues) {
	defer pq.wg.Done()
	for qevt := range pq.lowQueue {
		pq.lastLowLagNano.Store(time.Since(qevt.enqueuedAt).Nanoseconds())
		d.safeHandle(queuedEvent{evt: qevt.evt, instanceID: qevt.instanceID, orgID: qevt.orgID})
	}
}

// ─── Circuit breaker ───

func (d *asyncEventDispatcher) recordLowEvent(instanceID uuid.UUID) {
	if d == nil || d.cbRate <= 0 {
		return
	}
	d.cbMu.Lock()
	defer d.cbMu.Unlock()

	// Lazy-init window
	if _, ok := d.lowCounts[instanceID]; !ok {
		d.lowCounts[instanceID] = make([]int64, d.cbWindows)
	}
}

// circuitBreakerOpen returns true if the circuit breaker is open for this instance.
// It records the event count and checks the threshold.
//
// NOTE: the rate "per minute" tracking is best-effort (tick-less). For precise
// windowing, a future follow-up could use a dedicated ticker. This implementation
// increments a counter and expects an external periodic reset.
func (d *asyncEventDispatcher) circuitBreakerOpen(instanceID uuid.UUID) bool {
	if d == nil || d.cbRate <= 0 {
		return false
	}
	d.cbMu.Lock()
	defer d.cbMu.Unlock()

	// Check if currently in cooldown.
	if until, ok := d.cbOpenUntil[instanceID]; ok {
		if time.Now().Before(until) {
			return true
		}
		delete(d.cbOpenUntil, instanceID)
		d.lowCounts[instanceID] = nil
	}

	// Ensure window slice exists.
	windows, ok := d.lowCounts[instanceID]
	if !ok || len(windows) != d.cbWindows {
		d.lowCounts[instanceID] = make([]int64, d.cbWindows)
		return false
	}

	// Increment current window (index 0 is newest).
	windows[0]++

	// Check if all windows exceed threshold.
	for _, w := range windows {
		if w < int64(d.cbRate) {
			return false
		}
	}

	// Trip breaker.
	now := time.Now()
	d.cbOpenUntil[instanceID] = now.Add(d.cbCooldown)
	d.logger.Warn("Circuit breaker opened for instance — skipping HistorySync and low-priority events",
		"component", "whatsmeow",
		"event", "circuit_breaker_open",
		"instance_id", instanceID,
		"cooldown_sec", d.cbCooldown.Seconds(),
	)
	// Reset windows after tripping.
	d.lowCounts[instanceID] = make([]int64, d.cbWindows)
	return true
}

// ResetCircuitBreakerWindows advances the rolling window for all instances.
// Should be called periodically (every minute) from an external goroutine.
func (d *asyncEventDispatcher) ResetCircuitBreakerWindows() {
	if d == nil {
		return
	}
	d.cbMu.Lock()
	defer d.cbMu.Unlock()

	for instanceID, windows := range d.lowCounts {
		if len(windows) != d.cbWindows {
			d.lowCounts[instanceID] = make([]int64, d.cbWindows)
			continue
		}
		// Shift windows right; newest at index 0.
		copy(windows[1:], windows[:len(windows)-1])
		windows[0] = 0
	}
}

// IsCircuitBreakerOpen reports whether the breaker is open for an instance (for metrics).
func (d *asyncEventDispatcher) IsCircuitBreakerOpen(instanceID uuid.UUID) bool {
	if d == nil {
		return false
	}
	d.cbMu.Lock()
	defer d.cbMu.Unlock()

	if until, ok := d.cbOpenUntil[instanceID]; ok {
		return time.Now().Before(until)
	}
	return false
}

// circuitBreakerTickerLoop advances rolling windows every minute.
func (d *asyncEventDispatcher) circuitBreakerTickerLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.ResetCircuitBreakerWindows()
		case <-d.cbDone:
			return
		}
	}
}

// ─── Legacy queue creation ───

func (d *asyncEventDispatcher) queueFor(instanceID, orgID uuid.UUID) *asyncEventQueue {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	if _, stopped := d.stopped[instanceID]; stopped {
		return nil
	}
	if queue := d.queues[instanceID]; queue != nil {
		return queue
	}

	queue := &asyncEventQueue{
		instanceID: instanceID,
		orgID:      orgID,
		events:     make(chan asyncEvent, d.bufferSize),
	}
	d.queues[instanceID] = queue
	d.wg.Add(1)
	go d.processLoop(queue)
	return queue
}

func (d *asyncEventDispatcher) processLoop(queue *asyncEventQueue) {
	defer d.wg.Done()

	for event := range queue.events {
		d.updateQueueDepth(event.instanceID, len(queue.events))
		d.safeHandleLegacy(event)
		d.updateQueueDepth(event.instanceID, len(queue.events))
	}
}

// ─── Instance lifecycle ───

func (d *asyncEventDispatcher) StopInstance(instanceID uuid.UUID) {
	if d == nil {
		return
	}

	if d.priorityEnabled {
		d.stopInstancePriority(instanceID)
	} else {
		d.stopInstanceLegacy(instanceID)
	}
}

func (d *asyncEventDispatcher) stopInstanceLegacy(instanceID uuid.UUID) {
	d.mu.Lock()
	queue := d.queues[instanceID]
	delete(d.queues, instanceID)
	d.stopped[instanceID] = struct{}{}
	d.mu.Unlock()

	if queue == nil {
		d.updateQueueDepth(instanceID, 0)
		return
	}
	d.closeQueue(queue)
	d.updateQueueDepth(instanceID, 0)
}

func (d *asyncEventDispatcher) stopInstancePriority(instanceID uuid.UUID) {
	d.mu.Lock()
	pq := d.pQueues[instanceID]
	delete(d.pQueues, instanceID)
	d.stopped[instanceID] = struct{}{}
	d.mu.Unlock()

	if pq == nil {
		d.updatePriorityQueueDepth(instanceID, 0)
		return
	}
	d.closePriorityQueues(pq)

	// Drain workers up to configured timeout.
	drainDone := make(chan struct{})
	go func() {
		pq.wg.Wait()
		close(drainDone)
	}()
	timeout := time.Duration(d.drainTimeoutSec) * time.Second
	select {
	case <-drainDone:
	case <-time.After(timeout):
		remaining := d.priorityRemainingDepth(pq)
		d.logger.Warn("Timed out draining priority queues for instance",
			"component", "whatsmeow",
			"event", "shutdown_dropped",
			"instance_id", instanceID,
			"remaining", remaining,
		)
	}
	d.updatePriorityQueueDepth(instanceID, 0)
}

func (d *asyncEventDispatcher) AllowInstance(instanceID, orgID uuid.UUID) {
	if d == nil {
		return
	}
	d.mu.Lock()
	delete(d.stopped, instanceID)
	d.mu.Unlock()

	// Clear circuit breaker state on allowed.
	if d.priorityEnabled {
		d.cbMu.Lock()
		delete(d.cbOpenUntil, instanceID)
		delete(d.lowCounts, instanceID)
		d.cbMu.Unlock()
	}
}

func (d *asyncEventDispatcher) StopAll() {
	if d == nil {
		return
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true

	// Stop circuit breaker ticker inside the closed guard so a second call
	// does not close an already-closed channel.
	if d.cbDone != nil {
		close(d.cbDone)
	}

	if !d.priorityEnabled {
		d.stopAllLegacy()
		return
	}

	// Collect all priority queues while holding lock.
	pqs := make([]*instanceQueues, 0, len(d.pQueues))
	for instanceID, pq := range d.pQueues {
		pqs = append(pqs, pq)
		delete(d.pQueues, instanceID)
	}
	d.mu.Unlock()

	for _, pq := range pqs {
		d.closePriorityQueues(pq)
	}

	// Drain.
	done := make(chan struct{})
	go func() {
		for _, pq := range pqs {
			pq.wg.Wait()
		}
		close(done)
	}()

	timeout := time.Duration(d.drainTimeoutSec) * time.Second
	select {
	case <-done:
	case <-time.After(timeout):
		d.logger.Warn("Timed out draining priority queues during StopAll",
			"component", "whatsmeow",
			"event", "shutdown_timeout",
		)
	}
}

func (d *asyncEventDispatcher) stopAllLegacy() {
	queues := make([]*asyncEventQueue, 0, len(d.queues))
	for instanceID, queue := range d.queues {
		queues = append(queues, queue)
		delete(d.queues, instanceID)
	}
	d.mu.Unlock()

	for _, queue := range queues {
		d.closeQueue(queue)
		d.updateQueueDepth(queue.instanceID, 0)
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	const defaultStopTimeout = 30 * time.Second
	select {
	case <-done:
	case <-time.After(defaultStopTimeout):
		d.logger.Warn("Timed out waiting for async event workers to stop", "component", "whatsmeow", "timeout", defaultStopTimeout)
	}
}

// ─── Queue close helpers ───

func (d *asyncEventDispatcher) closeQueue(queue *asyncEventQueue) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return
	}
	queue.closed = true
	close(queue.events)
}

func (d *asyncEventDispatcher) closePriorityQueues(pq *instanceQueues) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.closed {
		return
	}
	pq.closed = true

	for _, ch := range pq.msgQueues {
		close(ch)
	}
	close(pq.lowQueue)
}

// ─── Safe handlers ───

func (d *asyncEventDispatcher) safeHandle(event queuedEvent) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("Panic in async event handler; event skipped",
				"component", "whatsmeow",
				"event", "async_event_panic",
				"instance_id", event.instanceID,
				"panic", r,
			)
		}
	}()
	if d.handler != nil {
		d.handler(event.evt, event.instanceID, event.orgID)
	}
}

func (d *asyncEventDispatcher) safeHandleLegacy(event asyncEvent) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("Panic in async event handler; event skipped",
				"component", "whatsmeow",
				"event", "async_event_panic",
				"instance_id", event.instanceID,
				"panic", r,
			)
		}
	}()
	if d.handler != nil {
		d.handler(event.evt, event.instanceID, event.orgID)
	}
}

// ─── Logging ───

func (d *asyncEventDispatcher) logEventDropLegacy(instanceID uuid.UUID, evt interface{}) {
	if d == nil {
		return
	}
	d.logger.Warn(
		"WhatsMeow event buffer full or stopped; dropping event",
		"component", "whatsmeow",
		"event", "async_event_drop",
		"instance_id", instanceID,
		"event_type", eventTypeName(evt),
	)
}

func (d *asyncEventDispatcher) logPriorityDrop(instanceID uuid.UUID, evt interface{}, reason, queueState string) {
	if d == nil {
		return
	}
	// NOTE: markEventDropped is called by the caller (enqueueHigh/enqueueLow) before
	// entering this function. We do NOT call it here to avoid double-counting.

	isCritical := reason == reasonCriticalOverflow
	if !isCritical {
		// Low-priority drops: log at warn level with full detail.
		d.logger.Warn(
			"WhatsMeow priority event drop",
			"component", "whatsmeow",
			"event", "priority_event_drop",
			"instance_id", instanceID,
			"reason", reason,
			"event_type", eventTypeName(evt),
		)
		return
	}

	// critical_overflow: sample the warn log, but ALWAYS count the metric.
	now := time.Now()
	d.overflowLogMu.Lock()
	last, hasLast := d.lastOverflowLog[instanceID]
	shouldLog := !hasLast || now.Sub(last) >= overflowLogSampleInterval
	if shouldLog {
		d.lastOverflowLog[instanceID] = now
	}
	d.overflowLogMu.Unlock()

	if shouldLog {
		// queue_state distinguishes a poisoned instance (instance_stopped —
		// the common auto-reconnect bug where the dispatcher is in stopped[]
		// and priorityQueueFor returns nil) from a genuinely saturated shard
		// (shard_full). Without it the two branches are indistinguishable.
		d.logger.Warn(
			"WhatsMeow critical overflow; dropping high-priority event",
			"component", "whatsmeow",
			"event", "critical_overflow",
			"instance_id", instanceID,
			"event_type", eventTypeName(evt),
			"reason", reason,
			"queue_state", queueState,
		)
	}
}

// ─── Depth / dropped helpers ───

func (d *asyncEventDispatcher) updateQueueDepth(instanceID uuid.UUID, depth int) {
	if d == nil || d.updateDepth == nil {
		return
	}
	d.updateDepth(instanceID, int64(depth))
}

func (d *asyncEventDispatcher) updatePriorityQueueDepth(instanceID uuid.UUID, depth int64) {
	if d == nil || d.updateDepth == nil {
		return
	}
	d.updateDepth(instanceID, depth)
}

func (d *asyncEventDispatcher) markEventDropped(instanceID uuid.UUID, queueState string) {
	if d == nil || d.markDropped == nil {
		return
	}
	d.markDropped(instanceID, queueState)
}

func (d *asyncEventDispatcher) priorityRemainingDepth(pq *instanceQueues) int {
	var total int
	for _, q := range pq.msgQueues {
		total += len(q)
	}
	total += len(pq.lowQueue)
	return total
}

// ─── Metrics snapshot providers ───

// PriorityQueueDepth returns the current depths for an instance's priority queues.
func (d *asyncEventDispatcher) PriorityQueueDepth(instanceID uuid.UUID) (msgDepth int64, lowDepth int64) {
	if d == nil || !d.priorityEnabled {
		return 0, 0
	}
	d.mu.Lock()
	pq := d.pQueues[instanceID]
	d.mu.Unlock()
	if pq == nil {
		return 0, 0
	}
	return d.msgDepth(pq), int64(len(pq.lowQueue))
}

// PriorityConsumerLag returns the max consumer lag in seconds for an instance.
// Lag is computed from the most recently dequeued event per queue type, updated
// atomically by workers to avoid breaking per-chat FIFO through channel peeking.
func (d *asyncEventDispatcher) PriorityConsumerLag(instanceID uuid.UUID) (msgLagSec, lowLagSec float64) {
	if d == nil || !d.priorityEnabled {
		return 0, 0
	}
	d.mu.Lock()
	pq := d.pQueues[instanceID]
	d.mu.Unlock()
	if pq == nil {
		return 0, 0
	}

	msgLagNs := pq.lastMsgLagNano.Load()
	lowLagNs := pq.lastLowLagNano.Load()
	return float64(msgLagNs) / float64(time.Second), float64(lowLagNs) / float64(time.Second)
}

// ─── Helpers ───

func eventTypeName(evt interface{}) string {
	if evt == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", evt)
}
