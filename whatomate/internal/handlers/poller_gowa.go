package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/gowa"
	"github.com/google/uuid"
)

// GOWA polling reconciler (Stage 7).
//
// This is the safety net behind the Stage 6 webhook receiver. The webhook
// channel is primary (real-time push), but it can drop events when:
//
//   - the network between GOWA and whatomate blips during delivery
//   - GOWA restarts and loses in-flight webhook goroutines
//   - whatomate is briefly down or behind, and GOWA's 5-attempt retry is
//     exhausted before whatomate recovers
//   - the operator rotates WebhookSecret and forgets to update GOWA, so
//     signatures fail until corrected
//
// The reconciler polls each active instance at cfg.Gowa.PollingIntervalSeconds
// and uses ListChats + GetChatMessages to discover messages that the webhook
// never delivered. Because handleGowaMessage is idempotent (FirstOrCreate on
// whats_app_message_id), the poller can safely re-feed events that the
// webhook also delivered — duplicates collapse silently.
//
// State: per-instance watermark stored in watermarks, mapping instance UUID
// → last successfully observed message timestamp. The poller requests
// GetChatMessages with start_time = watermark + 1µs on each sweep. On first
// run the watermark is zero, so the poller backfills (bounded by GOWA's own
// limit=50 default per chat — only the most recent messages come in). This
// is intentional: a fresh whatomate deployment should not try to import
// years of history on first boot.

// gowaPoller is the reconciler state. It is safe for concurrent use — the
// watermark map is guarded by mu. Only one goroutine runs the poll loop
// (started by Start, stopped by Stop).
type gowaPoller struct {
	app *App

	mu         sync.Mutex
	watermarks map[uuid.UUID]time.Time // instance ID → last observed msg time

	cancel context.CancelFunc
	wg     sync.WaitGroup
	run    bool
	muRun  sync.Mutex
}

// newGowaPoller builds the reconciler. Does not start it.
func newGowaPoller(app *App) *gowaPoller {
	return &gowaPoller{
		app:        app,
		watermarks: make(map[uuid.UUID]time.Time),
	}
}

// Start launches the background poller. It is idempotent: a second call
// returns nil without starting a second goroutine.
func (p *gowaPoller) Start(ctx context.Context) error {
	p.muRun.Lock()
	defer p.muRun.Unlock()
	if p.run {
		return nil
	}
	if p.app.GowaClient == nil {
		return nil
	}
	if !p.app.Config.Gowa.PollingEnabled {
		p.app.Log.Info("GOWA polling reconciler disabled by config")
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.run = true

	p.wg.Add(1)
	go p.loop(ctx)
	p.app.Log.Info("GOWA polling reconciler started",
		"interval_s", p.app.Config.Gowa.PollingIntervalSeconds)
	return nil
}

// Stop signals the loop to exit and waits for it to drain.
func (p *gowaPoller) Stop() {
	p.muRun.Lock()
	if !p.run {
		p.muRun.Unlock()
		return
	}
	p.run = false
	if p.cancel != nil {
		p.cancel()
	}
	p.muRun.Unlock()
	p.wg.Wait()
	p.app.Log.Info("GOWA polling reconciler stopped")
}

// loop is the main ticker loop. It polls every
// cfg.Gowa.PollingIntervalSeconds until ctx is cancelled.
func (p *gowaPoller) loop(ctx context.Context) {
	defer p.wg.Done()
	interval := time.Duration(p.app.Config.Gowa.PollingIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.reconcileOnce(ctx)
		}
	}
}

// reconcileOnce runs a single reconciliation pass across all active GOWA
// instances. Failures per instance are logged and swallowed — one bad
// instance must not abort the sweep.
func (p *gowaPoller) reconcileOnce(ctx context.Context) {
	db := p.app.DB.WithContext(ctx)
	var instances []models.WhatsAppInstance
	// Only reconcile instances that are not deleted. We do NOT filter by
	// status == connected, because a disconnected instance may still have
	// been receiving messages before it dropped — we want to catch those on
	// reconnect. The GOWA side will return empty for truly offline devices.
	if err := db.Find(&instances).Error; err != nil {
		p.app.Log.Warn("GOWA poller: failed to list instances", "error", err)
		return
	}
	if len(instances) == 0 {
		return
	}

	// Bound concurrency to avoid hammering GOWA. 5 parallel instances is a
	// reasonable cap; the rest wait in the semaphore.
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for i := range instances {
		inst := instances[i]
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p.reconcileInstance(ctx, &inst)
		}()
	}
	wg.Wait()
}

// reconcileInstance pulls the freshest chats for one instance and feeds any
// missing messages through the same handleGowaMessage path the webhook uses.
// Idempotency on whats_app_message_id makes re-feeding safe.
//
// It also syncs the instance status from GOWA's live status, so an instance
// that's logged in on GOWA but still marked "disconnected" locally (e.g.
// after a whatomate restart) gets promoted to "connected" automatically.
func (p *gowaPoller) reconcileInstance(ctx context.Context, instance *models.WhatsAppInstance) {
	deviceID := gowaDeviceID(instance)
	if deviceID == "" {
		return
	}

	// Sync live status from GOWA. This is the only path that promotes an
	// instance to "connected" in gowa mode (the whatsmeow Connect path does
	// not run). Without this, the outbound send path's instance selector
	// rejects every instance as "not connected".
	statusCtx, statusCancel := context.WithTimeout(ctx, 10*time.Second)
	gowaStatus, statusErr := p.app.gowaFetchStatus(statusCtx, instance)
	statusCancel()
	if statusErr != nil {
		// GOWA unreachable or device unknown — leave status as-is.
		p.app.Log.Debug("GOWA poller: status probe failed",
			"error", statusErr, "instance_id", instance.ID, "device_id", deviceID)
	} else {
		// Build a synthetic DeviceStatus to reuse syncInstanceStatusFromGowa
		syntheticStatus := &gowa.DeviceStatus{
			DeviceID:    deviceID,
			IsLoggedIn:  gowaStatus == models.InstanceStatusConnected,
			IsConnected: gowaStatus == models.InstanceStatusConnected || gowaStatus == models.InstanceStatusConnecting,
		}
		p.syncInstanceStatusFromGowa(ctx, instance, syntheticStatus)
	}

	// Read the watermark for pagination cursor.
	p.mu.Lock()
	since := p.watermarks[instance.ID]
	p.mu.Unlock()

	// Pull recent chats. We use a small limit because we only want the
	// currently-active conversations; long-tail dormant chats get picked up
	// by the webhook when they wake up.
	listCtx, listCancel := context.WithTimeout(ctx, 15*time.Second)
	chatsResp, err := p.app.GowaClient.ListChats(listCtx, deviceID, gowa.ListChatsFilter{
		Limit:  25,
		Offset: 0,
	})
	listCancel()
	if err != nil {
		// Likely GOWA unreachable or the device isn't paired yet. Quietly
		// skip — this is the safety net, not the primary channel.
		return
	}

	latest := since
	for _, chat := range chatsResp.Data {
		// Skip chats whose last activity is older than our watermark; they
		// have nothing new since the previous sweep.
		if chat.LastMessageTime != nil && !chat.LastMessageTime.After(since) {
			continue
		}

		// Pull messages newer than the watermark for this chat.
		msgCtx, msgCancel := context.WithTimeout(ctx, 15*time.Second)
		msgs, msgErr := p.app.GowaClient.GetChatMessages(msgCtx, deviceID, chat.JID,
			gowa.GetMessagesFilter{
				Limit:     50,
				Offset:    0,
				StartTime: since.Add(time.Microsecond).Format(time.RFC3339Nano),
			})
		msgCancel()
		if msgErr != nil {
			continue
		}
		for _, m := range msgs.Data {
			// Re-shape to the webhook payload and route through the same
			// handler. This guarantees the poller and webhook produce
			// identical on-disk state.
			payload := gowaMessagePayload{
				ID:        m.ID,
				Timestamp: m.Timestamp.Format(time.RFC3339),
				IsFromMe:  m.IsFromMe,
				From:      m.SenderJID,
				Text:      m.Content,
				MediaType: m.MediaType,
				URL:       m.URL,
				Filename:  m.Filename,
			}
			// For incoming messages the sender is the remote party; we fall
			// back to the chat JID if GOWA didn't surface one. For outgoing
			// echoes (IsFromMe) the sender is our own JID, so we set ChatID to
			// the chat JID — the conversation's counterparty — and let the
			// webhook handler resolve the contact from ChatID.
			payload.ChatID = chat.JID
			if !payload.IsFromMe && payload.From == "" {
				payload.From = chat.JID
			}
			raw, _ := json.Marshal(payload)
			p.app.handleGowaMessage(ctx, instance, raw)
			if m.Timestamp.After(latest) {
				latest = m.Timestamp
			}
		}
	}

	// Advance the watermark so the next sweep only fetches newer messages.
	if latest.After(since) {
		p.mu.Lock()
		p.watermarks[instance.ID] = latest
		p.mu.Unlock()
	}
}

// syncInstanceStatusFromGowa updates the local instance row to reflect the
// live GOWA device status. Promotes to "connected" when GOWA reports
// is_logged_in, and demotes to "disconnected" when both flags are false.
// Without this, the outbound send path's instance selector rejects every
// instance as "not connected" because the whatsmeow Connect path never ran.
//
// On any status TRANSITION (e.g. disconnected → connected after a QR scan),
// we also broadcast a WebSocket event so the UI updates in real time — the
// frontend closes the QR modal and refreshes the instance row without
// requiring a manual page reload. This matters in GOWA mode because GOWA's
// webhook subscription does not include connection/pairing events, so the
// poller is the only path that detects a successful QR scan.
func (p *gowaPoller) syncInstanceStatusFromGowa(ctx context.Context, instance *models.WhatsAppInstance, status *gowa.DeviceStatus) {
	if status == nil {
		return
	}
	desired := models.InstanceStatusDisconnected
	switch {
	case status.IsLoggedIn:
		desired = models.InstanceStatusConnected
	case status.IsConnected:
		desired = models.InstanceStatusConnecting
	}
	previousStatus := instance.Status
	if desired == previousStatus && instance.JID != "" {
		return
	}

	updates := map[string]any{"status": desired}
	// On a fresh pairing, GOWA's /devices/:id carries the JID of the newly
	// linked account. Patch it onto the instance row so subsequent lookups
	// (and the broadcast payload) carry the real phone number.
	if desired == models.InstanceStatusConnected {
		if device, err := p.app.GowaClient.GetDevice(ctx, status.DeviceID); err == nil && device != nil && device.JID != "" {
			if instance.JID == "" {
				updates["jid"] = device.JID
			}
			if instance.PhoneNumber == "" {
				updates["phone_number"] = jidToPhone(device.JID)
			}
		}
		if instance.LastConnectedAt == nil {
			now := time.Now()
			updates["last_connected_at"] = &now
		}
	}

	if err := p.app.DB.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instance.ID).
		Updates(updates).Error; err != nil {
		p.app.Log.Warn("GOWA poller: failed to sync instance status",
			"error", err, "instance_id", instance.ID, "desired", desired)
		return
	}
	if desired != previousStatus {
		p.app.Log.Info("GOWA poller: synced instance status from GOWA",
			"instance_id", instance.ID, "device_id", status.DeviceID,
			"old", previousStatus, "new", desired)
		instance.Status = desired
		// Reflect patched JID/phone back onto the in-memory row so the
		// broadcast payload carries them.
		if jid, ok := updates["jid"].(string); ok {
			instance.JID = jid
		}
		if phone, ok := updates["phone_number"].(string); ok {
			instance.PhoneNumber = phone
		}
		p.broadcastInstanceStatusChange(instance, desired, previousStatus)
	}
}

// broadcastInstanceStatusChange pushes a WebSocket notification so the UI
// reacts to GOWA device transitions in real time. Mirrors the events the
// whatsmeow ConnectionManager emits (instance_connected /
// instance_disconnected) so the frontend uses the exact same handlers.
func (p *gowaPoller) broadcastInstanceStatusChange(instance *models.WhatsAppInstance, desired, previous models.InstanceStatus) {
	if p.app.WSHub == nil {
		return
	}
	phone := strings.TrimSpace(instance.PhoneNumber)
	switch desired {
	case models.InstanceStatusConnected:
		p.app.Log.Info("GOWA poller: broadcasting instance_connected",
			"instance_id", instance.ID, "phone", phone)
		p.app.WSHub.BroadcastToOrg(instance.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeInstanceConnected,
			Payload: websocket.InstancePayload{
				InstanceID:  instance.ID.String(),
				PhoneNumber: phone,
				Status:      string(models.InstanceStatusConnected),
			},
		})
	case models.InstanceStatusDisconnected:
		// Only broadcast a disconnect when we actually dropped (not on the
		// initial disconnected → disconnected noop). Avoids noise on cold
		// starts where every unpaired instance would otherwise emit.
		if previous == models.InstanceStatusConnected || previous == models.InstanceStatusConnecting {
			p.app.Log.Info("GOWA poller: broadcasting instance_disconnected",
				"instance_id", instance.ID, "previous", previous)
			p.app.WSHub.BroadcastToOrg(instance.OrganizationID, websocket.WSMessage{
				Type: websocket.TypeInstanceDisconnected,
				Payload: websocket.InstancePayload{
					InstanceID: instance.ID.String(),
					PhoneNumber: phone,
					Status:      string(models.InstanceStatusDisconnected),
				},
			})
		}
	}
}
