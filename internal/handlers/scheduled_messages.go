package handlers

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ============================================================================
// Scheduled Messages
//
// CRUD for messages queued to be sent at a future time. Rows are fired by
// the ScheduledMessageProcessor (scheduled_message_processor.go) through the
// unified SendOutgoingMessage path. Access follows the same rules as
// immediate sends: agents without contacts:read can only schedule for their
// assigned contacts (scopeAssignedContact).
// ============================================================================

// ScheduledMessageRequest represents a create/update scheduled message request.
// The content shape mirrors SendMessageRequest so the frontend composer can
// reuse its payload building.
type ScheduledMessageRequest struct {
	Type    models.MessageType `json:"type"`
	Content struct {
		Body string `json:"body"`
		// Media fields — media_data is base64 and is persisted to local
		// storage at schedule time; only the local path is stored.
		MediaData     string `json:"media_data,omitempty"`
		MediaMimeType string `json:"media_mime_type,omitempty"`
		MediaFilename string `json:"media_filename,omitempty"`
		MediaURL      string `json:"media_url,omitempty"`
	} `json:"content"`
	WhatsAppAccount string         `json:"whatsapp_account,omitempty"`
	ScheduledAt     time.Time      `json:"scheduled_at"`
	TemplateID      string         `json:"template_id,omitempty"`
	TemplateParams  map[string]any `json:"template_params,omitempty"`
}

// CreateScheduledMessage schedules a message for a contact.
//
// POST /api/contacts/{id}/scheduled-messages
func (a *App) CreateScheduledMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req ScheduledMessageRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Scheduled time must be in the future (small grace for clock skew).
	if req.ScheduledAt.IsZero() || req.ScheduledAt.Before(time.Now().Add(30*time.Second)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "scheduled_at must be in the future", nil, "")
	}

	if req.Type == "" {
		req.Type = models.MessageTypeText
	}

	// Same visibility rule as SendMessage: agents can only schedule for
	// contacts they can access.
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Resolve the WhatsApp account now so a bad account fails at schedule
	// time, not at fire time. Prefer request override over contact default.
	accountName := contact.WhatsAppAccount
	if req.WhatsAppAccount != "" {
		accountName = req.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, accountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to resolve WhatsApp account", nil, "")
	}

	sm := models.ScheduledMessage{
		OrganizationID:  orgID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		MessageType:     req.Type,
		Content:         req.Content.Body,
		ScheduledAt:     req.ScheduledAt.UTC(),
		Status:          models.ScheduledMessageStatusPending,
		CreatedBy:       userID,
	}

	switch req.Type {
	case models.MessageTypeText:
		if req.Content.Body == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message body is required", nil, "")
		}

	case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
		// Persist media locally now; the processor re-reads the file from
		// MediaURL at fire time (same retry path as immediate media sends).
		sm.MediaMimeType = req.Content.MediaMimeType
		sm.MediaFilename = req.Content.MediaFilename
		sm.MediaURL = req.Content.MediaURL
		if req.Content.MediaData != "" {
			data, decErr := base64.StdEncoding.DecodeString(req.Content.MediaData)
			if decErr != nil || len(data) == 0 {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid media data", nil, "")
			}
			mediaURL, saveErr := a.saveMediaLocally(data, req.Content.MediaMimeType, req.Content.MediaFilename)
			if saveErr != nil {
				a.Log.Error("Failed to save scheduled message media", "error", saveErr)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media", nil, "")
			}
			sm.MediaURL = mediaURL
		}
		if sm.MediaURL == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Media data is required", nil, "")
		}

	case models.MessageTypeTemplate:
		templateID, parseErr := uuid.Parse(req.TemplateID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid template ID", nil, "")
		}
		if _, err := findByIDAndOrg[models.Template](a.DB, r, templateID, orgID, "Template"); err != nil {
			return nil
		}
		sm.TemplateID = &templateID
		if req.TemplateParams != nil {
			sm.TemplateParams = models.JSONB(req.TemplateParams)
		}

	default:
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported message type", nil, "")
	}

	if err := a.DB.Create(&sm).Error; err != nil {
		a.Log.Error("Failed to create scheduled message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to schedule message", nil, "")
	}

	a.logAudit(orgID, userID, "scheduled_message", sm.ID, models.AuditActionCreated, nil, &sm)
	a.broadcastScheduledMessageEvent(websocket.TypeScheduledMessageCreated, &sm)

	a.Log.Info("Message scheduled", "scheduled_message_id", sm.ID,
		"contact_id", contact.ID, "scheduled_at", sm.ScheduledAt.Format(time.RFC3339))

	return r.SendEnvelope(sm)
}

// ListContactScheduledMessages lists scheduled messages for one contact.
//
// GET /api/contacts/{id}/scheduled-messages
func (a *App) ListContactScheduledMessages(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Contact must be visible to the caller.
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	pg := parsePagination(r)
	baseQuery := a.DB.Model(&models.ScheduledMessage{}).
		Where("organization_id = ? AND contact_id = ?", orgID, contactID)
	if status := string(r.RequestCtx.QueryArgs().Peek("status")); status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}

	var total int64
	baseQuery.Count(&total)

	var messages []models.ScheduledMessage
	if err := pg.Apply(baseQuery.Order("scheduled_at ASC")).Find(&messages).Error; err != nil {
		a.Log.Error("Failed to list scheduled messages", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list scheduled messages", nil, "")
	}

	return r.SendEnvelope(listEnvelope("scheduled_messages", messages, total, pg))
}

// ListScheduledMessages lists scheduled messages across the organization.
// Agents without contacts:read only see rows for contacts they can access.
//
// GET /api/scheduled-messages
func (a *App) ListScheduledMessages(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	pg := parsePagination(r)
	baseQuery := a.DB.Model(&models.ScheduledMessage{}).Where("organization_id = ?", orgID)

	if status := string(r.RequestCtx.QueryArgs().Peek("status")); status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}
	if accountName := string(r.RequestCtx.QueryArgs().Peek("whatsapp_account")); accountName != "" {
		baseQuery = baseQuery.Where("whats_app_account = ?", accountName)
	}

	// Restrict to visible contacts for callers without contacts:read.
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		scopedContacts := a.scopeAssignedContact(
			a.DB.Model(&models.Contact{}).Select("id").Where("organization_id = ?", orgID),
			userID, orgID)
		baseQuery = baseQuery.Where("contact_id IN (?)", scopedContacts)
	}

	var total int64
	baseQuery.Count(&total)

	var messages []models.ScheduledMessage
	if err := pg.Apply(baseQuery.Preload("Contact").Order("scheduled_at ASC")).
		Find(&messages).Error; err != nil {
		a.Log.Error("Failed to list scheduled messages", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list scheduled messages", nil, "")
	}

	return r.SendEnvelope(listEnvelope("scheduled_messages", messages, total, pg))
}

// UpdateScheduledMessage edits a pending scheduled message's body and/or
// scheduled time. Non-pending rows (already firing, sent, failed, cancelled)
// are immutable.
//
// PUT /api/scheduled-messages/{id}
func (a *App) UpdateScheduledMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "scheduled message")
	if err != nil {
		return nil
	}

	var req ScheduledMessageRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	sm, err := a.loadScopedScheduledMessage(r, id, orgID, userID)
	if err != nil {
		return nil
	}
	if sm.Status != models.ScheduledMessageStatusPending {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Only pending scheduled messages can be edited", nil, "")
	}

	oldCopy := *sm

	updates := map[string]any{}
	if req.Content.Body != "" {
		updates["content"] = req.Content.Body
	}
	if !req.ScheduledAt.IsZero() {
		if req.ScheduledAt.Before(time.Now().Add(30 * time.Second)) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "scheduled_at must be in the future", nil, "")
		}
		updates["scheduled_at"] = req.ScheduledAt.UTC()
	}
	if len(updates) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Nothing to update", nil, "")
	}

	// Guard the status again inside the UPDATE so an edit racing the
	// processor's pending→processing claim loses cleanly.
	res := a.DB.Model(&models.ScheduledMessage{}).
		Where("id = ? AND status = ?", sm.ID, models.ScheduledMessageStatusPending).
		Updates(updates)
	if res.Error != nil {
		a.Log.Error("Failed to update scheduled message", "error", res.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update scheduled message", nil, "")
	}
	if res.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Scheduled message is already being sent", nil, "")
	}

	a.DB.First(sm, "id = ?", sm.ID)
	a.logAudit(orgID, userID, "scheduled_message", sm.ID, models.AuditActionUpdated, &oldCopy, sm)
	a.broadcastScheduledMessageEvent(websocket.TypeScheduledMessageUpdated, sm)

	return r.SendEnvelope(sm)
}

// CancelScheduledMessage cancels a pending scheduled message. The row is kept
// (status=cancelled) for history rather than deleted.
//
// DELETE /api/scheduled-messages/{id}
func (a *App) CancelScheduledMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "scheduled message")
	if err != nil {
		return nil
	}

	sm, err := a.loadScopedScheduledMessage(r, id, orgID, userID)
	if err != nil {
		return nil
	}
	if sm.Status != models.ScheduledMessageStatusPending {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Only pending scheduled messages can be cancelled", nil, "")
	}

	oldCopy := *sm

	// Status guard in the UPDATE closes the race with the processor claim.
	res := a.DB.Model(&models.ScheduledMessage{}).
		Where("id = ? AND status = ?", sm.ID, models.ScheduledMessageStatusPending).
		Update("status", models.ScheduledMessageStatusCancelled)
	if res.Error != nil {
		a.Log.Error("Failed to cancel scheduled message", "error", res.Error)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to cancel scheduled message", nil, "")
	}
	if res.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Scheduled message is already being sent", nil, "")
	}

	sm.Status = models.ScheduledMessageStatusCancelled
	a.logAudit(orgID, userID, "scheduled_message", sm.ID, models.AuditActionDeleted, &oldCopy, sm)
	a.broadcastScheduledMessageEvent(websocket.TypeScheduledMessageUpdated, sm)

	return r.SendEnvelope(sm)
}

// loadScopedScheduledMessage loads a scheduled message by ID within the org,
// enforcing contact visibility for callers without contacts:read. On failure
// it has already written the error envelope (callers return nil).
func (a *App) loadScopedScheduledMessage(r *fastglue.Request, id, orgID, userID uuid.UUID) (*models.ScheduledMessage, error) {
	var sm models.ScheduledMessage
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&sm).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Scheduled message not found", nil, "")
		return nil, errEnvelopeSent
	}

	// The caller must be able to access the underlying contact.
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", sm.ContactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Scheduled message not found", nil, "")
		return nil, errEnvelopeSent
	}

	return &sm, nil
}

// broadcastScheduledMessageEvent notifies clients viewing the contact that a
// scheduled message was created or changed state.
func (a *App) broadcastScheduledMessageEvent(eventType string, sm *models.ScheduledMessage) {
	if a.WSHub == nil {
		return
	}
	a.WSHub.BroadcastToContact(sm.OrganizationID, sm.ContactID, websocket.WSMessage{
		Type:    eventType,
		Payload: sm,
	})
}
