package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"gorm.io/gorm/clause"
)

// ScheduledMessageProcessor fires due scheduled messages. It ticks every
// minute, atomically claims pending rows whose scheduled_at has passed
// (pending→processing UPDATE ... RETURNING, so concurrent replicas can never
// double-send), and dispatches each through the unified SendOutgoingMessage
// path — the exact same code an immediate send uses.
//
// Lifecycle mirrors ChatResetProcessor (Start/Stop with a context + stopCh).
// Unlike the chat reset, idempotency here is enforced at the database level:
// a double-fired scheduled message is user-visible, so an in-memory guard is
// not enough.
type ScheduledMessageProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

// staleProcessingAge is how long a row may sit in "processing" before the
// startup catch-up assumes the previous process crashed mid-send and returns
// it to pending for a retry.
const staleProcessingAge = 10 * time.Minute

// NewScheduledMessageProcessor creates a new scheduled-message dispatcher.
// The interval controls the due-row poll frequency; the default is one minute.
func NewScheduledMessageProcessor(app *App, interval time.Duration) *ScheduledMessageProcessor {
	return &ScheduledMessageProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the dispatch loop. Blocks until the context is cancelled or
// Stop is called.
func (p *ScheduledMessageProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Scheduled message processor started", "interval", p.interval)

	// Catch up shortly after startup: recover rows stranded in "processing"
	// by a crash, then fire anything that came due while the server was down.
	time.AfterFunc(10*time.Second, func() {
		p.RecoverStale(time.Now())
		p.ProcessDue(time.Now())
	})

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Scheduled message processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("Scheduled message processor stopped")
			return
		case now := <-ticker.C:
			p.ProcessDue(now)
		}
	}
}

// Stop signals the processor to exit.
func (p *ScheduledMessageProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// RecoverStale returns rows stuck in "processing" longer than
// staleProcessingAge to pending so they are retried. Only expected after a
// crash between the claim and the final status update.
func (p *ScheduledMessageProcessor) RecoverStale(now time.Time) {
	res := p.app.DB.Model(&models.ScheduledMessage{}).
		Where("status = ? AND updated_at < ?", models.ScheduledMessageStatusProcessing, now.Add(-staleProcessingAge)).
		Update("status", models.ScheduledMessageStatusPending)
	if res.Error != nil {
		p.app.Log.Error("Failed to recover stale scheduled messages", "error", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		p.app.Log.Warn("Recovered stale scheduled messages", "count", res.RowsAffected)
	}
}

// ProcessDue atomically claims all due pending rows and sends them. The
// claim (UPDATE ... RETURNING) is the idempotency barrier: a row can only
// ever transition pending→processing once.
func (p *ScheduledMessageProcessor) ProcessDue(now time.Time) {
	var due []models.ScheduledMessage
	res := p.app.DB.Model(&due).Clauses(clause.Returning{}).
		Where("status = ? AND scheduled_at <= ? AND deleted_at IS NULL",
			models.ScheduledMessageStatusPending, now).
		Update("status", models.ScheduledMessageStatusProcessing)
	if res.Error != nil {
		p.app.Log.Error("Failed to claim due scheduled messages", "error", res.Error)
		return
	}
	if len(due) == 0 {
		return
	}

	p.app.Log.Info("Firing due scheduled messages", "count", len(due))
	for i := range due {
		p.processOne(&due[i])
	}
}

// processOne sends a single claimed scheduled message and records the outcome.
func (p *ScheduledMessageProcessor) processOne(sm *models.ScheduledMessage) {
	msgReq, err := p.buildOutgoingRequest(sm)
	if err != nil {
		p.finish(sm, nil, err)
		return
	}

	// Send synchronously so the outcome is known before the row is finalized.
	// Attribute the message to the scheduling user, same as an agent send.
	opts := DefaultSendOptions()
	opts.Async = false
	opts.SentByUserID = &sm.CreatedBy

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	msg, err := p.app.SendOutgoingMessage(ctx, *msgReq, opts)
	if err != nil {
		p.finish(sm, nil, err)
		return
	}

	// SendOutgoingMessage swallows provider errors into the Message row's
	// status (finalizeMessageSend), so re-read it to learn the real outcome.
	var final models.Message
	if err := p.app.DB.Select("status", "error_message").
		First(&final, "id = ?", msg.ID).Error; err == nil &&
		final.Status == models.MessageStatusFailed {
		p.finish(sm, msg, fmt.Errorf("%s", final.ErrorMessage))
		return
	}

	p.finish(sm, msg, nil)
}

// buildOutgoingRequest translates a scheduled row into the unified sender's
// request, resolving the account, contact, and template it references.
func (p *ScheduledMessageProcessor) buildOutgoingRequest(sm *models.ScheduledMessage) (*OutgoingMessageRequest, error) {
	account, err := p.app.resolveWhatsAppAccount(sm.OrganizationID, sm.WhatsAppAccount)
	if err != nil {
		return nil, fmt.Errorf("whatsapp account %q not found", sm.WhatsAppAccount)
	}

	var contact models.Contact
	if err := p.app.DB.Where("id = ? AND organization_id = ?", sm.ContactID, sm.OrganizationID).
		First(&contact).Error; err != nil {
		return nil, fmt.Errorf("contact not found")
	}

	req := &OutgoingMessageRequest{
		Account: account,
		Contact: &contact,
		Type:    sm.MessageType,
	}

	switch sm.MessageType {
	case models.MessageTypeText:
		req.Content = sm.Content

	case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
		// MediaData stays empty — SendOutgoingMessage's retry path re-reads
		// the file from local storage via MediaURL at fire time.
		req.Caption = sm.Content
		req.MediaURL = sm.MediaURL
		req.MediaMimeType = sm.MediaMimeType
		req.MediaFilename = sm.MediaFilename

	case models.MessageTypeTemplate:
		if sm.TemplateID == nil {
			return nil, fmt.Errorf("template ID missing")
		}
		var template models.Template
		if err := p.app.DB.Where("id = ? AND organization_id = ?", *sm.TemplateID, sm.OrganizationID).
			First(&template).Error; err != nil {
			return nil, fmt.Errorf("template not found")
		}
		req.Template = &template
		req.BodyParams = jsonbToStringMap(sm.TemplateParams)

	default:
		return nil, fmt.Errorf("unsupported message type: %s", sm.MessageType)
	}

	return req, nil
}

// finish records the terminal state of a claimed row and broadcasts it.
func (p *ScheduledMessageProcessor) finish(sm *models.ScheduledMessage, msg *models.Message, sendErr error) {
	updates := map[string]any{}
	if sendErr != nil {
		sm.Status = models.ScheduledMessageStatusFailed
		sm.ErrorMessage = sendErr.Error()
		updates["status"] = sm.Status
		updates["error_message"] = sm.ErrorMessage
		p.app.Log.Error("Scheduled message send failed",
			"error", sendErr, "scheduled_message_id", sm.ID, "contact_id", sm.ContactID)
	} else {
		sm.Status = models.ScheduledMessageStatusSent
		updates["status"] = sm.Status
		p.app.Log.Info("Scheduled message sent",
			"scheduled_message_id", sm.ID, "contact_id", sm.ContactID)
	}
	if msg != nil {
		msgID := msg.ID
		sm.SentMessageID = &msgID
		updates["sent_message_id"] = msgID
	}

	if err := p.app.DB.Model(&models.ScheduledMessage{}).
		Where("id = ?", sm.ID).Updates(updates).Error; err != nil {
		p.app.Log.Error("Failed to finalize scheduled message", "error", err, "scheduled_message_id", sm.ID)
	}

	p.app.broadcastScheduledMessageEvent(websocket.TypeScheduledMessageUpdated, sm)
}

// jsonbToStringMap flattens a JSONB parameter map into the string→string map
// the template renderer expects.
func jsonbToStringMap(params models.JSONB) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for k, v := range params {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
