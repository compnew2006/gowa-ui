---
title: Background Workers
---

# Background Workers

Whatomate uses Redis-backed background workers for asynchronous processing. This page documents the queue system, job types, worker processes, and background goroutines.

## Redis Queue System

**Source:** `internal/queue/queue.go`, `internal/queue/consumer.go`, `internal/queue/publisher.go`

### Queue Types

| Queue | Purpose | Key Pattern |
|-------|---------|-------------|
| Campaign Queue | Campaign message jobs | `queue:campaign` |
| Inbound Media Queue | Media download jobs | `queue:inbound_media` |
| Pub/Sub: Campaign Stats | Campaign progress broadcasts | `channel:campaign_stats` |
| Pub/Sub: Notifications | System notifications | `channel:notifications` |

### Job Types

#### RecipientJob

```go
type RecipientJob struct {
    CampaignID    int64             `json:"campaign_id"`
    RecipientID   int64             `json:"recipient_id"`
    PhoneNumber   string            `json:"phone_number"`
    TemplateID    int64             `json:"template_id"`
    TemplateName  string            `json:"template_name"`
    BodyContent   string            `json:"body_content"`
    HeaderMediaID string            `json:"header_media_id,omitempty"`
    Params        map[string]string `json:"params,omitempty"`
    AccountID     int64             `json:"account_id"`
    MinDelay      int               `json:"min_delay"`
    MaxDelay      int               `json:"max_delay"`
}
```

#### InboundMediaJob

```go
type InboundMediaJob struct {
    MessageID   int64  `json:"message_id"`
    MediaURL    string `json:"media_url"`
    AccountID   int64  `json:"account_id"`
    PhoneNumber string `json:"phone_number"`
}
```

#### CampaignStatsUpdate

```go
type CampaignStatsUpdate struct {
    CampaignID int64 `json:"campaign_id"`
    Sent       int   `json:"sent"`
    Delivered  int   `json:"delivered"`
    Read       int   `json:"read"`
    Failed     int   `json:"failed"`
    Pending    int   `json:"pending"`
    Status     string `json:"status"`
}
```

### Publisher Operations

```go
// Publish a job to a queue
func (p *Publisher) Publish(queue string, job interface{}) error {
    data, _ := json.Marshal(job)
    return p.client.LPush(ctx, queue, data).Err()
}

// Publish campaign stats via pub/sub
func (p *Publisher) PublishCampaignStats(update *CampaignStatsUpdate) error {
    data, _ := json.Marshal(update)
    return p.client.Publish(ctx, "channel:campaign_stats", data).Err()
}
```

### Consumer Operations

```go
// Start consuming jobs
func (c *Consumer) Consume(queue string, handler JobHandler) {
    for {
        result, err := c.client.BRPop(ctx, 0, queue).Result()
        if err != nil {
            continue
        }
        
        go func(jobData string) {
            if err := handler(jobData); err != nil {
                // Retry with backoff
                c.Retry(queue, jobData)
            }
        }(result[1])
    }
}

// Acknowledge successful processing
func (c *Consumer) Ack(jobID string) error {
    return c.client.Del(ctx, "job:processing:"+jobID).Err()
}

// Negative acknowledge (requeue or dead-letter)
func (c *Consumer) Nack(queue string, jobData string, requeue bool) error {
    if requeue {
        return c.client.RPush(ctx, queue, jobData).Err()
    }
    return c.client.LPush(ctx, "queue:dead_letter", jobData).Err()
}
```

## Campaign Worker Processing

**Source:** `internal/worker/worker.go`

### Worker Job Processing

```
Redis Queue → Worker.HandleRecipientJob()
  1. Acquire distributed lock on recipient_id
  2. Load recipient record, verify status = pending
  3. Load campaign, verify status = running
  4. Apply campaign delay (random between min/max)
  5. Resolve template placeholders
  6. Build message payload
  7. Send via MessageProvider
  8. Update recipient status (sent/delivered/failed)
  9. Update campaign stats
  10. Publish stats to Redis pub/sub
  11. Release recipient lock
```

### Idempotency

```go
func (w *Worker) checkIdempotency(jobID string) (bool, error) {
    // Check if already processed
    exists, err := w.redis.Exists(ctx, "job:processed:"+jobID).Result()
    if err != nil {
        return false, err
    }
    return exists > 0, nil
}

func (w *Worker) markProcessed(jobID string) error {
    // Mark as processed with 24-hour expiry
    return w.redis.Set(ctx, "job:processed:"+jobID, "1", 24*time.Hour).Err()
}
```

### Campaign Delay

```go
func (w *Worker) applyDelay(minDelay, maxDelay int) {
    delay := time.Duration(rand.Intn(maxDelay-minDelay+1)+minDelay) * time.Second
    time.Sleep(delay)
}
```

## Send Policy Enforcement

**Source:** `internal/worker/send_policy.go`

Before sending a campaign message, the worker enforces policies:

```go
func (w *Worker) enforceSendPolicy(ctx context.Context, job *RecipientJob) error {
    // 1. Check business hours
    if !isWithinBusinessHours(job.OrgID) {
        return ErrOutsideBusinessHours
    }
    
    // 2. Check user send restrictions
    if err := checkUserRestrictions(job.UserID, job.ContactID); err != nil {
        return err
    }
    
    // 3. Check rate limits
    if err := checkRateLimit(job.OrgID); err != nil {
        return err
    }
    
    return nil
}
```

## Background Goroutines

The following goroutines are started in `main.go`:

### SLA Processor

**Source:** `internal/handlers/sla_processor.go`

**Interval:** Every 1 minute

```go
func (p *SLAProcessor) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        p.checkSLABreaches()
        p.autoCloseExpiredChats()
    }
}
```

**Processing:**
1. Load SLA settings for each organization
2. For each open chat:
   - Check response SLA (time since last inbound message)
   - Check resolution SLA (time since chat opened)
   - Check escalation SLA (time since first response)
3. If SLA breached:
   - Send warning message to contact (if configured)
   - Notify escalation users via WebSocket
   - Escalate to manager if `sla_escalation_minutes` exceeded
4. Auto-close chats exceeding `sla_auto_close_hours`

**SLA Settings:**
| Setting | Description |
|---------|-------------|
| `sla_response_minutes` | Max time to first response |
| `sla_resolution_minutes` | Max time to resolve chat |
| `sla_escalation_minutes` | Time before manager escalation |
| `sla_auto_close_hours` | Hours before auto-close |
| `sla_escalation_notify_ids` | User IDs to notify on escalation |

### Activity Retention Worker

**Source:** `internal/handlers/activity_retention.go`

**Interval:** Every 1 hour

```go
func (w *ActivityRetentionWorker) Start() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cutoff := time.Now().Add(-90 * 24 * time.Hour) // 90 days default
        w.DB.Where("created_at < ?", cutoff).Delete(&models.ActivityLog{})
    }
}
```

### Chat Assignment Reset Worker

**Source:** `internal/handlers/chat_assignment_reset_worker.go`

**Interval:** Every 1 minute

```go
func (w *ChatAssignmentResetWorker) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        w.checkResetRules()
    }
}
```

**Processing:**
1. Check schedule for reset rules
2. Find chats matching reset conditions
3. Reset assignments (clear `assigned_user_id`)
4. Notify affected users via WebSocket

### Instance Auto-Campaign Worker

**Source:** `internal/handlers/instance_auto_campaign_worker.go`

**Interval:** Every 1 minute

```go
func (w *InstanceAutoCampaignWorker) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        w.processAutoCampaigns()
    }
}
```

**Processing:**
1. Check instances with auto-campaign enabled
2. Find contacts matching criteria
3. Send automated messages
4. Track results

### Campaign Stats Subscriber

**Source:** `internal/queue/subscriber.go`

**Trigger:** Continuous (Redis pub/sub)

```go
func (s *CampaignStatsSubscriber) Start() {
    pubsub := s.redis.Subscribe(ctx, "channel:campaign_stats")
    ch := pubsub.Channel()
    
    for msg := range ch {
        var update CampaignStatsUpdate
        json.Unmarshal([]byte(msg.Payload), &update)
        s.hub.BroadcastToOrg(update.OrgID, &WSMessage{
            Type:    "campaign_stats_update",
            Payload: update,
        })
    }
}
```

### WhatsMeow Reconnect

**Trigger:** Server startup

```go
// Reconnect all active instances
manager.ReconnectAll()

// Auto-connect linked sessions on first run
manager.AutoConnectLinkedInstancesOnFirstRun()
```

### Status Reconciliation

**Trigger:** Server startup (30-second timeout)

Cleans up stale instance statuses from previous server sessions.

## Background Processes Summary

| Process | Interval | Source File | Purpose |
|---------|----------|-------------|---------|
| SLA Processor | 1 minute | `sla_processor.go` | Check SLA breaches, auto-close |
| Activity Retention | 1 hour | `activity_retention.go` | Delete old activity logs |
| Chat Assignment Reset | 1 minute | `chat_assignment_reset_worker.go` | Reset stale assignments |
| Instance Auto-Campaign | 1 minute | `instance_auto_campaign_worker.go` | Send automated messages |
| Campaign Worker | Continuous | `worker/worker.go` | Process campaign queue |
| Inbound Media Worker | Continuous | `worker/worker.go` | Download inbound media |
| Campaign Stats Subscriber | Continuous | `app.go` | Broadcast campaign stats via WS |
| WhatsMeow Reconnect | Startup | `main.go` | Reconnect all instances |
| Status Reconciliation | Startup (30s) | `main.go` | Clean stale instance statuses |

## See Also

- [Architecture](./architecture)
- [Caching](./caching) — Redis cache system
- [WebSocket Events](./websocket-events) — Campaign stats broadcasts
