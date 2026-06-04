package whatsmeow

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zerodha/logf"
)

const defaultEventBufferSize = 4096

type asyncEventHandler func(evt interface{}, instanceID, orgID uuid.UUID)

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

// asyncEventDispatcher decouples whatsmeow's reader callback from event I/O.
// Each instance has one FIFO worker to preserve event ordering for that
// connection while still returning immediately from AddEventHandler.
type asyncEventDispatcher struct {
	bufferSize  int
	handler     asyncEventHandler
	logger      logf.Logger
	updateDepth func(uuid.UUID, int64)
	markDropped func(uuid.UUID)

	mu     sync.Mutex
	queues map[uuid.UUID]*asyncEventQueue
	wg     sync.WaitGroup
	closed bool
}

func newAsyncEventDispatcher(bufferSize int, logger logf.Logger, handler asyncEventHandler, updateDepth func(uuid.UUID, int64), markDropped func(uuid.UUID)) *asyncEventDispatcher {
	if bufferSize <= 0 {
		bufferSize = defaultEventBufferSize
	}
	return &asyncEventDispatcher{
		bufferSize:  bufferSize,
		handler:     handler,
		logger:      logger,
		updateDepth: updateDepth,
		markDropped: markDropped,
		queues:      make(map[uuid.UUID]*asyncEventQueue),
	}
}

func (d *asyncEventDispatcher) Dispatch(evt interface{}, instanceID, orgID uuid.UUID) bool {
	if d == nil {
		return false
	}

	queue := d.queueFor(instanceID, orgID)
	if queue == nil {
		return false
	}

	event := asyncEvent{evt: evt, instanceID: instanceID, orgID: orgID}
	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		d.logEventDrop(instanceID, evt)
		return false
	}

	select {
	case queue.events <- event:
		d.updateQueueDepth(instanceID, len(queue.events))
		return true
	default:
		d.markEventDropped(instanceID)
		d.logEventDrop(instanceID, evt)
		return false
	}
}

func (d *asyncEventDispatcher) StopInstance(instanceID uuid.UUID) {
	if d == nil {
		return
	}

	d.mu.Lock()
	queue := d.queues[instanceID]
	delete(d.queues, instanceID)
	d.mu.Unlock()

	if queue == nil {
		d.updateQueueDepth(instanceID, 0)
		return
	}
	d.closeQueue(queue)
	d.updateQueueDepth(instanceID, 0)
}

const defaultStopTimeout = 30 * time.Second

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

	select {
	case <-done:
	case <-time.After(defaultStopTimeout):
		d.logger.Warn("Timed out waiting for async event workers to stop", "component", "whatsmeow", "timeout", defaultStopTimeout)
	}
}

func (d *asyncEventDispatcher) queueFor(instanceID, orgID uuid.UUID) *asyncEventQueue {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
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
		if d.handler != nil {
			d.handler(event.evt, event.instanceID, event.orgID)
		}
		d.updateQueueDepth(event.instanceID, len(queue.events))
	}
}

func (d *asyncEventDispatcher) closeQueue(queue *asyncEventQueue) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return
	}
	queue.closed = true
	close(queue.events)
}

func (d *asyncEventDispatcher) logEventDrop(instanceID uuid.UUID, evt interface{}) {
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

func (d *asyncEventDispatcher) updateQueueDepth(instanceID uuid.UUID, depth int) {
	if d == nil || d.updateDepth == nil {
		return
	}
	d.updateDepth(instanceID, int64(depth))
}

func (d *asyncEventDispatcher) markEventDropped(instanceID uuid.UUID) {
	if d == nil || d.markDropped == nil {
		return
	}
	d.markDropped(instanceID)
}

func eventTypeName(evt interface{}) string {
	if evt == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", evt)
}
