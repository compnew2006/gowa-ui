package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/compnew2006/gowa-ui/internal/chatlifecycle"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ensureClaimableChatStatus normalizes the lifecycle status when a new
// customer-side message lands on the conversation — regardless of whether it
// was received on the connected number or sent from its phone (is_from_me).
// Unassigned conversations become pending (they must be claimed before an
// agent can view them); closed conversations reopen as pending via
// ChatLifecycle.CustomerReopen (system message + WS broadcast). Assigned open
// conversations are left untouched — the inactivity timer resets via
// last_message_at.
func (a *App) ensureClaimableChatStatus(orgID uuid.UUID, contact *models.Contact, reopenNote string) {
	if contact.AssignedUserID == nil {
		if contact.EffectiveStatus() != models.ChatStatusPending {
			contact.SetStatus(models.ChatStatusPending)
			a.DB.Model(contact).Update("metadata", contact.Metadata)
		}
		return
	}
	a.ChatLifecycle.CustomerReopen(context.Background(), orgID, contact, reopenNote)
}

// loadContactByPath parses the {id} contact path param and loads the contact
// through scopeAssignedContact, so lifecycle actions (claim/release/close/
// join/leave/reopen, notes) enforce the same visibility as the chat list —
// account-scoped agents cannot act on conversations outside their assigned
// accounts. On error it sends the HTTP response and returns ok=false —
// callers should `return nil`. It returns a value (not a pointer) so callers
// keep passing &contact to the lifecycle service unchanged.
func (a *App) loadContactByPath(r *fastglue.Request, orgID, userID uuid.UUID) (models.Contact, bool) {
	var contact models.Contact
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return contact, false
	}
	query := a.scopeAssignedContact(a.DB.Where("id = ? AND organization_id = ?", contactID, orgID), userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		return contact, false
	}
	return contact, true
}

// ClaimChat allows an agent to claim a pending (unassigned) conversation.
// Thin HTTP adapter over Service.Claim. Route: PUT /api/contacts/{id}/claim
// Permission: chat.assign:write
func (a *App) ClaimChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	hasCollaborate := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	outcome, agentName, otherAgentName, err := a.ChatLifecycle.Claim(r.RequestCtx, orgID, userID, &contact, hasCollaborate)
	if err != nil {
		a.Log.Error("Failed to claim chat", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to claim chat", nil, "")
	}

	switch outcome {
	case chatlifecycle.ClaimConflictOther:
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			fmt.Sprintf("This chat is already assigned to %s", otherAgentName),
			map[string]any{"current_agent": otherAgentName, "can_join": hasCollaborate},
			"already_assigned")
	case chatlifecycle.ClaimRerouteJoin:
		// Assigned to another agent + caller has collaborate → join instead.
		res, jerr := a.ChatLifecycle.Join(r.RequestCtx, orgID, userID, &contact)
		if jerr != nil {
			a.Log.Error("Failed to join chat", "error", jerr, "contact_id", contact.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to join chat", nil, "")
		}
		return r.SendEnvelope(map[string]any{
			"contact_id":   contact.ID,
			"collaborator": true,
			"user_name":    res.UserName,
		})
	case chatlifecycle.ClaimAlreadySelf:
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"assigned":   true,
			"message":    "Already assigned to you",
		})
	default: // ClaimDone
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"assigned":   true,
			"agent_name": agentName,
		})
	}
}

// ReleaseChat returns an assigned (open or closed) conversation to the pending
// pool. Thin HTTP adapter over Service.Release: parse → auth → lookup → call
// the service → map the typed result onto the HTTP envelope.
//
// All business rules (closed-chat policy, idempotency, collaborator clearing)
// and side effects (system message, audit, WS broadcast) live in the service.
// Route: PUT /api/contacts/{id}/release
// Permission: chat.assign:write (caller must be the assignee or an admin/manager).
func (a *App) ReleaseChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	// Authorization: assignee or admin/manager (ghost-release). The service
	// double-checks this as defense-in-depth, but the HTTP-friendly error
	// message is produced here so the response body stays user-facing.
	isAssignee := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isAdminOrManager := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
	if !isAssignee && !isAdminOrManager {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You are not allowed to release this chat", nil, "")
	}

	released, err := a.ChatLifecycle.Release(r.RequestCtx, orgID, userID, &contact, isAssignee, isAdminOrManager)
	if err != nil {
		switch {
		case errors.Is(err, chatlifecycle.ErrClosedReleaseByAgent):
			return r.SendErrorEnvelope(fasthttp.StatusForbidden,
				"Only admins can release a closed chat — reopen it first", nil, "")
		case errors.Is(err, chatlifecycle.ErrNotAuthorized):
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You are not allowed to release this chat", nil, "")
		default:
			a.Log.Error("Failed to release chat", "error", err, "contact_id", contact.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to release chat", nil, "")
		}
	}

	// Idempotent no-op: the chat was already pending + unassigned. Preserve
	// the historical envelope shape (released:true + message) for clients.
	if !released {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"released":   true,
			"message":    "Conversation is already pending",
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":  contact.ID,
		"released":    true,
		"chat_status": "pending",
	})
}

// JoinChat allows a user with chat.collaborate:write to join an assigned
// conversation. Thin adapter over Service.Join.
// Route: POST /api/contacts/{id}/join  Permission: chat.collaborate:write
func (a *App) JoinChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	res, err := a.ChatLifecycle.Join(r.RequestCtx, orgID, userID, &contact)
	if err != nil {
		a.Log.Error("Failed to join chat", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to join chat", nil, "")
	}

	switch res.Outcome {
	case chatlifecycle.JoinAlreadyOwner:
		return r.SendEnvelope(map[string]any{"message": "You are the primary owner of this conversation"})
	case chatlifecycle.JoinAlreadyCollaborator:
		return r.SendEnvelope(map[string]any{"message": "You are already a collaborator"})
	default:
		return r.SendEnvelope(map[string]any{
			"contact_id":   contact.ID,
			"collaborator": true,
			"user_name":    res.UserName,
		})
	}
}

// InviteCollaborator allows a manager/admin to add another agent as a
// collaborator. Thin adapter over Service.Invite.
// Route: POST /api/contacts/{id}/collaborators/{user_id}  Permission: chat.collaborate:write
func (a *App) InviteCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	targetUserIDStr, _ := r.RequestCtx.UserValue("user_id").(string)
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID", nil, "")
	}

	// Scoped load: only act on conversations the caller can see.
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}

	res, err := a.ChatLifecycle.Invite(r.RequestCtx, orgID, userID, targetUserID, contact)
	if err != nil {
		// Service returns a wrapped "target user not found" error.
		a.Log.Error("Failed to invite collaborator", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
	}

	switch res.Outcome {
	case chatlifecycle.InviteAlreadyOwner:
		return r.SendEnvelope(map[string]any{"message": "User is already the primary owner"})
	case chatlifecycle.InviteAlreadyCollaborator:
		return r.SendEnvelope(map[string]any{"message": "User is already a collaborator"})
	default:
		return r.SendEnvelope(map[string]any{
			"contact_id":   contact.ID,
			"collaborator": true,
			"user_name":    res.TargetName,
		})
	}
}

// LeaveChat removes the requesting user from the conversation. Thin adapter
// over Service.Leave. If the owner is the last participant, the conversation
// is closed; if collaborators remain, ownership is transferred.
// Route: DELETE /api/contacts/{id}/join
func (a *App) LeaveChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	isOwner := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	isAdminOrManager := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)

	if !isOwner && !isCollaborator && !isAdminOrManager {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"You are not a participant in this conversation", nil, "")
	}

	res, err := a.ChatLifecycle.Leave(r.RequestCtx, orgID, userID, &contact, isOwner, isCollaborator, isAdminOrManager)
	if err != nil {
		a.Log.Error("Failed to leave chat", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to leave chat", nil, "")
	}

	switch res.Outcome {
	case chatlifecycle.LeaveGhostExit:
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"left":       true,
			"ghost_exit": true,
		})
	case chatlifecycle.LeaveClosedChat:
		// Leaving as the last participant closed the conversation — same CSAT
		// trigger as an explicit close.
		go a.maybeSendCloseRatingPrompt(orgID, userID, contact)
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"left":       true,
			"closed":     true,
		})
	default: // LeaveOwnershipTransferred, LeaveCollaboratorRemoved
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"left":       true,
		})
	}
}

// RemoveCollaborator allows a manager/admin to remove a collaborator.
// Thin adapter over Service.RemoveCollaborator.
// Route: DELETE /api/contacts/{id}/collaborators/{user_id}  Permission: chat.collaborate:write
func (a *App) RemoveCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	targetUserIDStr, _ := r.RequestCtx.UserValue("user_id").(string)
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID", nil, "")
	}

	// Scoped load: only act on conversations the caller can see.
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}

	res, err := a.ChatLifecycle.RemoveCollaborator(r.RequestCtx, orgID, userID, targetUserID, contact)
	if err != nil {
		switch {
		case errors.Is(err, chatlifecycle.ErrCannotRemoveOwner):
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"Cannot remove the primary owner. Unassign the conversation instead.", nil, "")
		case errors.Is(err, chatlifecycle.ErrNotCollaborator):
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"User is not a collaborator on this conversation", nil, "")
		default:
			a.Log.Error("Failed to remove collaborator", "error", err, "contact_id", contact.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove collaborator", nil, "")
		}
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":   contact.ID,
		"removed":      true,
		"removed_user": res.TargetName,
	})
}

// CloseChat closes an open conversation. Thin adapter over Service.Close.
// Only the assigned agent, a collaborator, or a manager/admin may close.
// Route: PUT /api/contacts/{id}/close
func (a *App) CloseChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	isOwner := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	hasCollaboratePerm := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	hasContactsRead := a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID)

	if !isOwner && !isCollaborator && !hasCollaboratePerm && !hasContactsRead {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	if err := a.ChatLifecycle.Close(r.RequestCtx, orgID, userID, &contact); err != nil {
		if errors.Is(err, chatlifecycle.ErrAlreadyClosed) {
			return r.SendEnvelope(map[string]any{
				"contact_id": contact.ID,
				"message":    "Conversation is already closed",
			})
		}
		a.Log.Error("Failed to close chat", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
	}

	// Kick off the CSAT rating cycle off-path — messaging stays out of the
	// chatlifecycle service, so the prompt is triggered here in the handler.
	go a.maybeSendCloseRatingPrompt(orgID, userID, contact)

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"closed":     true,
	})
}

// ReopenChat reopens a closed conversation WITHOUT assigning it to the caller.
// Thin adapter over Service.Reopen. Admin/manager only (contacts:write).
// Route: PUT /api/contacts/{id}/reopen
func (a *App) ReopenChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceContacts, models.ActionWrite)
	if err != nil {
		return nil
	}

	contact, ok := a.loadContactByPath(r, orgID, userID)
	if !ok {
		return nil
	}

	reopened, err := a.ChatLifecycle.Reopen(r.RequestCtx, orgID, userID, &contact)
	if err != nil {
		a.Log.Error("Failed to reopen chat", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reopen chat", nil, "")
	}

	if !reopened {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"reopened":   false,
			"message":    "Conversation is already open",
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"reopened":   true,
	})
}

// BulkReleaseChats releases many conversations back to the pending pool in one
// request. Thin adapter over Service.BulkRelease — the body parse + cap stay
// here, the per-item loop (auth, policy, mutation, audit, WS) lives in the service.
// Route: POST /api/contacts/bulk-release  Body: { "contact_ids": ["<uuid>", ...] }
func (a *App) BulkReleaseChats(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	var req struct {
		ContactIDs []string `json:"contact_ids"`
	}
	bodyBytes := r.RequestCtx.PostBody()
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		}
	}
	if len(req.ContactIDs) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "contact_ids is required and must be non-empty", nil, "")
	}
	if len(req.ContactIDs) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot release more than 500 chats at once", nil, "")
	}

	isAdminOrManager := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)

	// Scope the batch up-front: only contacts visible through
	// scopeAssignedContact may be released. Invisible IDs are reported as
	// "not found" failures instead of being acted on.
	visible := make(map[uuid.UUID]bool, len(req.ContactIDs))
	parsedIDs := make([]uuid.UUID, 0, len(req.ContactIDs))
	for _, raw := range req.ContactIDs {
		if id, err := uuid.Parse(raw); err == nil {
			parsedIDs = append(parsedIDs, id)
		}
	}
	if len(parsedIDs) > 0 {
		var visibleIDs []uuid.UUID
		if err := a.scopeAssignedContact(
			a.DB.Model(&models.Contact{}).Select("id"), userID, orgID,
		).Where("id IN ?", parsedIDs).Pluck("id", &visibleIDs).Error; err != nil {
			a.Log.Error("Failed to scope bulk-release contacts", "error", err, "user_id", userID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to release chats", nil, "")
		}
		for _, id := range visibleIDs {
			visible[id] = true
		}
	}
	scopedIDs := make([]string, 0, len(req.ContactIDs))
	failed := make([]map[string]any, 0)
	for _, raw := range req.ContactIDs {
		if id, err := uuid.Parse(raw); err == nil && visible[id] {
			scopedIDs = append(scopedIDs, raw)
		} else if _, err := uuid.Parse(raw); err != nil {
			scopedIDs = append(scopedIDs, raw) // let the service report "invalid uuid"
		} else {
			failed = append(failed, map[string]any{"contact_id": raw, "reason": "not found"})
		}
	}
	result := a.ChatLifecycle.BulkRelease(r.RequestCtx, orgID, userID, scopedIDs, isAdminOrManager)
	result.Failed = append(result.Failed, failed...)

	return r.SendEnvelope(map[string]any{
		"released_ids": result.ReleasedIDs,
		"released":     result.ReleasedIDs,
		"failed":       result.Failed,
		"requested":    len(req.ContactIDs),
	})
}
