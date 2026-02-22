package whatsmeow

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/zerodha/logf"
)

// Job represents a unit of work (sending a message)
type Job func(ctx context.Context) error

// InstanceQueue manages the message queue for a single WhatsApp instance
type InstanceQueue struct {
	instanceID string
	jobs       chan Job
	stop       chan struct{}
	config     config.WhatsmeowConfig
	logger     logf.Logger
	lastActive time.Time
	mu         sync.Mutex // protects lastActive and checks
}

// QueueManager manages queues for multiple instances
type QueueManager struct {
	queues sync.Map // map[string]*InstanceQueue
	config config.WhatsmeowConfig
	logger logf.Logger
	ctx    context.Context
	cancel context.CancelFunc
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

// Enqueue adds a job to the instance's queue
func (m *QueueManager) Enqueue(instanceID string, job Job) error {
	q, _ := m.queues.LoadOrStore(instanceID, m.newInstanceQueue(instanceID))
	queue := q.(*InstanceQueue)

	// Non-blocking write if possible, or maybe sufficient buffer?
	// For now using blocking write to ensure backpressure if needed.

	select {
	case queue.jobs <- job:
		queue.updateActivity()
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	default:
		// Fallback if full? Configurable buffer size?
		// For now assume channel buffer of 100 is enough given rate limits.
		// If full, we block or error.
		// Let's block with context check.
		select {
		case queue.jobs <- job:
			queue.updateActivity()
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
		return true
	})
}

func (m *QueueManager) newInstanceQueue(instanceID string) *InstanceQueue {
	q := &InstanceQueue{
		instanceID: instanceID,
		jobs:       make(chan Job, 100), // Buffer size 100
		stop:       make(chan struct{}),
		config:     m.config,
		logger:     m.logger, // Passed directly, fields added in log calls
		lastActive: time.Now(),
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
		case job := <-q.jobs:
			q.updateActivity()
			q.process(job)
		}
	}
}

func (q *InstanceQueue) process(job Job) {
	// 1. Rate Limiting Delay
	minDelay := q.config.RateLimitMinDelayMs
	maxDelay := q.config.RateLimitMaxDelayMs
	if minDelay == 0 {
		minDelay = 1000
	}
	if maxDelay == 0 {
		maxDelay = 3000
	}

	// Random delay
	delay := time.Duration(minDelay+rand.Intn(maxDelay-minDelay+1)) * time.Millisecond
	time.Sleep(delay)

	// 2. Execution with Exponential Backoff
	maxRetries := 3
	baseBackoff := 1 * time.Second

	for i := 0; i <= maxRetries; i++ {
		err := job(context.Background()) // Should we pass a timeout ctx?
		if err == nil {
			return // Success
		}

		if i == maxRetries {
			q.logger.Error("job failed after retries", "instance_id", q.instanceID, "error", err)
			return
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
