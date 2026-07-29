package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ConversationNoteRequest represents the request body for creating/updating a note.
type ConversationNoteRequest struct {
	Content string `json:"content"`
}

// ConversationNoteResponse represents the API response for a conversation note.
type ConversationNoteResponse struct {
	ID            uuid.UUID `json:"id"`
	ContactID     uuid.UUID `json:"contact_id"`
	CreatedByID   uuid.UUID `json:"created_by_id"`
	CreatedByName string    `json:"created_by_name"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ListConversationNotes returns paginated notes for a contact (latest at bottom).
func (a *App) ListConversationNotes(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceChat, models.ActionRead)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	pg := parsePaginationWithDefaults(r, 30, 100)
	limit := pg.Limit

	query := a.DB.Where("organization_id = ? AND contact_id = ?", orgID, contactID)

	// Get total count
	var total int64
	query.Model(&models.ConversationNote{}).Count(&total)

	// Cursor-based pagination: load notes before a specific ID
	beforeIDStr := string(r.RequestCtx.QueryArgs().Peek("before"))
	if beforeIDStr != "" {
		beforeID, err := uuid.Parse(beforeIDStr)
		if err == nil {
			var beforeNote models.ConversationNote
			if err := a.DB.Where("id = ?", beforeID).First(&beforeNote).Error; err == nil {
				query = query.Where("created_at < ?", beforeNote.CreatedAt)
			}
		}
	}

	// Fetch DESC then reverse to get chronological order (oldest first, latest last)
	var notes []models.ConversationNote
	if err := query.Preload("CreatedBy").
		Order("created_at DESC").
		Limit(limit).
		Find(&notes).Error; err != nil {
		a.Log.Error("Failed to list conversation notes", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to list notes", nil, "")
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(notes)-1; i < j; i, j = i+1, j-1 {
		notes[i], notes[j] = notes[j], notes[i]
	}

	result := make([]ConversationNoteResponse, len(notes))
	for i, n := range notes {
		result[i] = noteToResponse(n)
	}

	return r.SendEnvelope(map[string]any{
		"notes":    result,
		"total":    total,
		"has_more": len(notes) == limit,
	})
}

// CreateConversationNote creates a new note on a contact.
func (a *App) CreateConversationNote(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req ConversationNoteRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "content is required", nil, "")
	}

	note := models.ConversationNote{
		OrganizationID: orgID,
		ContactID:      contactID,
		CreatedByID:    userID,
		Content:        req.Content,
	}

	if err := a.DB.Create(&note).Error; err != nil {
		a.Log.Error("Failed to create conversation note", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to create note", nil, "")
	}

	// Load the creator relation for the response
	var user models.User
	a.DB.First(&user, "id = ?", userID)
	note.CreatedBy = &user

	resp := noteToResponse(note)

	// Broadcast via WebSocket
	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type:    websocket.TypeConversationNoteCreated,
			Payload: resp,
		})
	}

	return r.SendEnvelope(resp)
}

// resolveOwnedNote validates the {id} contact path param, loads the {note_id}
// note scoped to orgID, and verifies the caller (userID) is its creator. On any
// failure it sends the HTTP response and returns ok=false — callers should
// `return nil`. verb is used in the forbidden message (e.g. "edit"/"delete").
func (a *App) resolveOwnedNote(r *fastglue.Request, orgID, userID uuid.UUID, verb string) (*models.ConversationNote, bool) {
	if _, err := parsePathUUID(r, "id", "contact"); err != nil {
		return nil, false
	}

	noteID, err := parsePathUUID(r, "note_id", "note")
	if err != nil {
		return nil, false
	}

	note, err := findByIDAndOrg[models.ConversationNote](a.DB, r, noteID, orgID, "Note")
	if err != nil {
		return nil, false
	}

	// Only the creator can modify their own notes
	if note.CreatedByID != userID {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "You can only "+verb+" your own notes", nil, "")
		return nil, false
	}

	return note, true
}

// UpdateConversationNote updates an existing note (creator only).
func (a *App) UpdateConversationNote(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}

	note, ok := a.resolveOwnedNote(r, orgID, userID, "edit")
	if !ok {
		return nil
	}

	var req ConversationNoteRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "content is required", nil, "")
	}

	note.Content = req.Content
	if err := a.DB.Save(note).Error; err != nil {
		a.Log.Error("Failed to update conversation note", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to update note", nil, "")
	}

	// Load the creator relation for the response
	var user models.User
	a.DB.First(&user, "id = ?", note.CreatedByID)
	note.CreatedBy = &user

	resp := noteToResponse(*note)

	// Broadcast via WebSocket
	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, note.ContactID, websocket.WSMessage{
			Type:    websocket.TypeConversationNoteUpdated,
			Payload: resp,
		})
	}

	return r.SendEnvelope(resp)
}

// DeleteConversationNote deletes a note (creator only).
func (a *App) DeleteConversationNote(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}

	note, ok := a.resolveOwnedNote(r, orgID, userID, "delete")
	if !ok {
		return nil
	}

	contactID := note.ContactID
	noteID := note.ID

	if err := a.DB.Delete(note).Error; err != nil {
		a.Log.Error("Failed to delete conversation note", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to delete note", nil, "")
	}

	// Broadcast via WebSocket
	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type: websocket.TypeConversationNoteDeleted,
			Payload: map[string]any{
				"id":         noteID,
				"contact_id": contactID,
			},
		})
	}

	return r.SendEnvelope(map[string]string{"message": "Note deleted"})
}

func noteToResponse(n models.ConversationNote) ConversationNoteResponse {
	createdByName := ""
	if n.CreatedBy != nil {
		createdByName = n.CreatedBy.FullName
	}
	return ConversationNoteResponse{
		ID:            n.ID,
		ContactID:     n.ContactID,
		CreatedByID:   n.CreatedByID,
		CreatedByName: createdByName,
		Content:       n.Content,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}
