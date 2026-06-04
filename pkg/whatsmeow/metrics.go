package whatsmeow

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// InstanceHealthMetrics is the runtime health view for one instance.
type InstanceHealthMetrics struct {
	UptimeSeconds         int64   `json:"uptime_seconds"`
	MessagesSentToday     uint64  `json:"messages_sent_today"`
	MessagesReceivedToday uint64  `json:"messages_received_today"`
	MessagesFailedToday   uint64  `json:"messages_failed_today"`
	EventsDroppedToday    uint64  `json:"events_dropped_today"`
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	QueueDepth            int64   `json:"queue_depth"`
}

type instanceMetrics struct {
	dayKeyUTC          atomic.Int64
	connectedSinceUnix atomic.Int64
	messagesSent       atomic.Uint64
	messagesReceived   atomic.Uint64
	messagesFailed     atomic.Uint64
	eventsDropped      atomic.Uint64
	errors             atomic.Uint64
	queueDepth         atomic.Int64
}

func currentDayKeyUTC() int64 {
	now := time.Now().UTC()
	return int64(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

func newInstanceMetrics() *instanceMetrics {
	m := &instanceMetrics{}
	m.dayKeyUTC.Store(currentDayKeyUTC())
	return m
}

func (m *instanceMetrics) resetIfDayChanged() {
	current := currentDayKeyUTC()
	previous := m.dayKeyUTC.Load()
	if previous == current {
		return
	}
	if m.dayKeyUTC.CompareAndSwap(previous, current) {
		m.messagesSent.Store(0)
		m.messagesReceived.Store(0)
		m.messagesFailed.Store(0)
		m.eventsDropped.Store(0)
		m.errors.Store(0)
	}
}

func (cm *ConnectionManager) getOrCreateMetrics(instanceID uuid.UUID) *instanceMetrics {
	if metrics, ok := cm.metrics.Load(instanceID); ok {
		im := metrics.(*instanceMetrics)
		im.resetIfDayChanged()
		return im
	}
	im := newInstanceMetrics()
	actual, _ := cm.metrics.LoadOrStore(instanceID, im)
	metrics := actual.(*instanceMetrics)
	metrics.resetIfDayChanged()
	return metrics
}

// MarkConnected marks an instance as connected and starts uptime tracking.
func (cm *ConnectionManager) MarkConnected(instanceID uuid.UUID) {
	cm.getOrCreateMetrics(instanceID).connectedSinceUnix.Store(time.Now().Unix())
}

// MarkDisconnected marks an instance as disconnected and resets uptime.
func (cm *ConnectionManager) MarkDisconnected(instanceID uuid.UUID) {
	cm.getOrCreateMetrics(instanceID).connectedSinceUnix.Store(0)
}

// MarkMessageSent increments outbound sent message counter.
func (cm *ConnectionManager) MarkMessageSent(instanceID uuid.UUID) {
	metrics := cm.getOrCreateMetrics(instanceID)
	metrics.messagesSent.Add(1)
}

// MarkMessageReceived increments inbound received message counter.
func (cm *ConnectionManager) MarkMessageReceived(instanceID uuid.UUID) {
	metrics := cm.getOrCreateMetrics(instanceID)
	metrics.messagesReceived.Add(1)
}

// MarkMessageFailed increments failed message counter and error count.
func (cm *ConnectionManager) MarkMessageFailed(instanceID uuid.UUID) {
	metrics := cm.getOrCreateMetrics(instanceID)
	metrics.messagesFailed.Add(1)
	metrics.errors.Add(1)
}

// MarkError increments generic error counter for the instance.
func (cm *ConnectionManager) MarkError(instanceID uuid.UUID) {
	metrics := cm.getOrCreateMetrics(instanceID)
	metrics.errors.Add(1)
}

// MarkEventDropped increments the async event ingestion drop counter.
func (cm *ConnectionManager) MarkEventDropped(instanceID uuid.UUID) {
	metrics := cm.getOrCreateMetrics(instanceID)
	metrics.eventsDropped.Add(1)
	metrics.errors.Add(1)
}

// SetQueueDepth tracks queue depth for the instance.
func (cm *ConnectionManager) SetQueueDepth(instanceID uuid.UUID, depth int64) {
	if depth < 0 {
		depth = 0
	}
	cm.getOrCreateMetrics(instanceID).queueDepth.Store(depth)
}

// GetInstanceHealth returns runtime health counters for an instance.
func (cm *ConnectionManager) GetInstanceHealth(instanceID uuid.UUID) InstanceHealthMetrics {
	metrics := cm.getOrCreateMetrics(instanceID)

	sent := metrics.messagesSent.Load()
	failed := metrics.messagesFailed.Load()
	var errorRate float64
	if sent+failed > 0 {
		errorRate = (float64(failed) / float64(sent+failed)) * 100
	}

	var uptime int64
	connectedSince := metrics.connectedSinceUnix.Load()
	if connectedSince > 0 {
		uptime = time.Now().Unix() - connectedSince
		if uptime < 0 {
			uptime = 0
		}
	}

	return InstanceHealthMetrics{
		UptimeSeconds:         uptime,
		MessagesSentToday:     sent,
		MessagesReceivedToday: metrics.messagesReceived.Load(),
		MessagesFailedToday:   failed,
		EventsDroppedToday:    metrics.eventsDropped.Load(),
		ErrorRatePercent:      errorRate,
		QueueDepth:            metrics.queueDepth.Load(),
	}
}
