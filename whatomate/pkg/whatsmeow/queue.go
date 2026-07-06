package whatsmeow

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/zerodha/logf"
)

// Job represents a unit of work (sending a message)
type Job func(ctx context.Context) error

// QueuedSend is a queueable outbound send. Run executes the send; the optional
// OnRetryReset hook is invoked once before the first retry when Run fails with
// a session-desync class error (WhatsApp "server returned error 400"). It is
// used to clear the recipient's Signal sessions so the retry rebuilds them
// from a fresh prekey exchange instead of reusing the stale session that
// caused the 400.
type QueuedSend struct {
	Run          Job
	OnRetryReset func(ctx context.Context)
}

// InstanceQueue manages the message queue for a single WhatsApp instance
type InstanceQueue struct {
	instanceID    string
	jobs          chan QueuedSend
	stop          chan struct{}
	config        config.WhatsmeowConfig
	logger        logf.Logger
	lastActive    time.Time
	lastProcessed time.Time
	mu            sync.Mutex // protects lastActive and checks
	onDepth       func(instanceID string, depth int64)
}

// QueueManager manages queues for multiple instances
type QueueManager struct {
	queues      sync.Map // map[string]*InstanceQueue
	config      config.WhatsmeowConfig
	logger      logf.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	depthReport func(instanceID string, depth int64)
}

// NewQueueManager creates a new QueueManager
func NewQueueManager(cfg config.WhatsmeowConfig, logger logf.Logger) *QueueManager {
	ctx, cancel := context.WithCancel(context.Background())
	qm := &QueueManager{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup routine for idle queues
	go qm.cleanupLoop()

	return qm
}

// SetDepthObserver registers a callback invoked whenever queue depth changes for an instance.
func (m *QueueManager) SetDepthObserver(observer func(instanceID string, depth int64)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.depthReport = observer
}

// Enqueue adds a job to the instance's queue. It is kept for backward
// compatibility (Job has no reset hook); new callers should prefer EnqueueSend
// so they can supply a session-reset hook for 400 recovery.
func (m *QueueManager) Enqueue(instanceID string, job Job) error {
	return m.EnqueueSend(instanceID, QueuedSend{Run: job})
}

// EnqueueSend adds a QueuedSend to the instance's queue. The send's optional
// OnRetryReset hook is invoked by the worker before the first retry when the
// send fails with a session-desync class error.
func (m *QueueManager) EnqueueSend(instanceID string, send QueuedSend) error {
	if send.Run == nil {
		return errors.New("queued send has no Run function")
	}
	q, _ := m.queues.LoadOrStore(instanceID, m.newInstanceQueue(instanceID))
	queue := q.(*InstanceQueue)

	// Non-blocking write if possible, or maybe sufficient buffer?
	// For now using blocking write to ensure backpressure if needed.

	select {
	case queue.jobs <- send:
		queue.updateActivity()
		queue.reportDepth()
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	default:
		// Fallback if full? Configurable buffer size?
		// For now assume channel buffer of 100 is enough given rate limits.
		// If full, we block or error.
		// Let's block with context check.
		select {
		case queue.jobs <- send:
			queue.updateActivity()
			queue.reportDepth()
			return nil
		case <-time.After(5 * time.Second):
			return errors.New("queue full, timed out")
		}
	}
}

// Close shuts down the manager and all queues
func (m *QueueManager) Close() {
	m.cancel()
	m.queues.Range(func(key, value interface{}) bool {
		q := value.(*InstanceQueue)
		close(q.stop)
		q.reportDepthWithValue(0)
		return true
	})
}

func (m *QueueManager) newInstanceQueue(instanceID string) *InstanceQueue {
	q := &InstanceQueue{
		instanceID: instanceID,
		jobs:       make(chan QueuedSend, 100), // Buffer size 100
		stop:       make(chan struct{}),
		config:     m.config,
		logger:     m.logger, // Passed directly, fields added in log calls
		lastActive: time.Now(),
		onDepth:    m.reportDepth,
	}
	go q.worker()
	return q
}

func (m *QueueManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			limit := time.Duration(m.config.QueueTimeoutSeconds) * time.Second
			if limit == 0 {
				limit = 5 * time.Minute // Default safe limit
			}

			m.queues.Range(func(key, value interface{}) bool {
				q := value.(*InstanceQueue)
				q.mu.Lock()
				idle := time.Since(q.lastActive)
				q.mu.Unlock()

				if idle > limit {
					m.logger.Info("removing idle queue", "instance_id", key, "idle_duration", idle)
					close(q.stop)
					q.reportDepthWithValue(0)
					m.queues.Delete(key)
				}
				return true
			})
		}
	}
}

func (q *InstanceQueue) updateActivity() {
	q.mu.Lock()
	q.lastActive = time.Now()
	q.mu.Unlock()
}

func (q *InstanceQueue) worker() {
	q.logger.Info("starting queue worker", "instance_id", q.instanceID)
	defer q.logger.Info("queue worker stopped", "instance_id", q.instanceID)

	for {
		select {
		case <-q.stop:
			return
		case send := <-q.jobs:
			q.updateActivity()
			q.reportDepth()
			q.process(send)
			q.reportDepth()
		}
	}
}

func (q *InstanceQueue) process(send QueuedSend) {
	// 1. Rate Limiting Delay
	minDelay := q.config.RateLimitMinDelayMs
	maxDelay := q.config.RateLimitMaxDelayMs
	if minDelay == 0 {
		minDelay = 1000
	}
	if maxDelay == 0 {
		maxDelay = 3000
	}
	if minDelay < 0 {
		minDelay = 0
	}
	if maxDelay < minDelay {
		maxDelay = minDelay
	}

	// Random delay
	delayMs := minDelay
	if maxDelay > minDelay {
		delayMs += secureRandomIntn(maxDelay - minDelay + 1)
	}
	requiredDelay := time.Duration(delayMs) * time.Millisecond

	// Only sleep if not enough time has elapsed since last processed message.
	q.mu.Lock()
	elapsed := time.Since(q.lastProcessed)
	q.mu.Unlock()

	if remaining := requiredDelay - elapsed; remaining > 0 {
		time.Sleep(remaining)
	}

	q.mu.Lock()
	q.lastProcessed = time.Now()
	q.mu.Unlock()

	// 2. Execution with Exponential Backoff
	maxRetries := 3
	baseBackoff := 1 * time.Second
	jobTimeout := time.Duration(q.config.QueueTimeoutSeconds) * time.Second
	if jobTimeout <= 0 {
		jobTimeout = 30 * time.Second
	}

	// sessionResetTriggered guards the reset hook so it runs at most once per
	// job. It fires on the transition from attempt 0 to attempt 1 — i.e. after
	// the first failure but before the first retry — and only when the failure
	// is the session-desync class that a recipient reset can fix.
	sessionResetTriggered := false

	for i := 0; i <= maxRetries; i++ {
		jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		err := send.Run(jobCtx)
		cancel()
		if err == nil {
			return // Success
		}
		if !shouldRetrySendError(err) {
			q.logger.Warn("job failed with permanent error; skipping retries", "instance_id", q.instanceID, "error", err)
			return
		}

		if i == maxRetries {
			q.logger.Error("job failed after retries", "instance_id", q.instanceID, "error", err)
			return
		}

		// Before the first retry, clear the recipient's Signal sessions when
		// the failure is a WhatsApp "server returned error 400" (session
		// desync). This lets the retry rebuild a clean session via the prekey
		// flow instead of reusing the stale one that caused the 400.
		if !sessionResetTriggered && isSessionDesyncError(err) && send.OnRetryReset != nil {
			resetCtx, resetCancel := context.WithTimeout(context.Background(), jobTimeout)
			q.logger.Warn(
				"resetting recipient session before retry",
				"instance_id", q.instanceID,
				"attempt", i+1,
				"error", err,
			)
			send.OnRetryReset(resetCtx)
			resetCancel()
			sessionResetTriggered = true
		}

		q.logger.Warn("job failed, retrying", "instance_id", q.instanceID, "attempt", i+1, "error", err)

		// Backoff
		backoff := baseBackoff * (1 << i) // 1s, 2s, 4s...
		select {
		case <-time.After(backoff):
			// continue retry
		case <-q.stop:
			return
		}
	}
}

func (q *InstanceQueue) reportDepth() {
	if q == nil {
		return
	}
	q.reportDepthWithValue(int64(len(q.jobs)))
}

func (q *InstanceQueue) reportDepthWithValue(depth int64) {
	if q == nil || q.onDepth == nil {
		return
	}
	if depth < 0 {
		depth = 0
	}
	q.onDepth(q.instanceID, depth)
}

func (m *QueueManager) reportDepth(instanceID string, depth int64) {
	if m == nil {
		return
	}
	m.mu.RLock()
	reporter := m.depthReport
	m.mu.RUnlock()
	if reporter == nil {
		return
	}
	reporter(instanceID, depth)
}
