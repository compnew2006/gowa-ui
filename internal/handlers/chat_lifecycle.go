package handlers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// createSystemMessage creates a system message record in the conversation timeline.
func (a *App) createSystemMessage(orgID, contactID uuid.UUID, content string, metadata models.JSONB) {
	if metadata == nil {
		metadata = models.JSONB{}
	}
	metadata["is_system_message"] = true

	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		ContactID:      contactID,
		Direction:      models.DirectionOutgoing,
		MessageType:    models.MessageTypeText,
		Content:        content,
		Status:         models.MessageStatusSent,
		Metadata:       metadata,
	}
	if err := a.DB.Create(msg).Error; err != nil {
		a.Log.Error("Failed to create system message", "error", err, "contact_id", contactID)
	}
}

// ClaimChat allows an agent to claim a pending (unassigned) conversation.
// Route: PUT /api/contacts/{id}/claim
// Permission: chat.assign:write
func (a *App) ClaimChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Closed conversations CAN be claimed — this reopens them.
	// The claim will set status to open and assign the agent.

	// Guard 2: already assigned to another agent
	if contact.AssignedUserID != nil && *contact.AssignedUserID != userID {
		// Check if user has collaborate permission (they should join instead)
		hasCollaborate := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
		if hasCollaborate {
			return a.joinAsCollaborator(r, &contact, userID, orgID)
		}

		var currentAgent models.User
		agentName := "another agent"
		if a.DB.First(&currentAgent, "id = ?", *contact.AssignedUserID).Error == nil {
			agentName = currentAgent.FullName
		}
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			fmt.Sprintf("This chat is already assigned to %s", agentName),
			map[string]any{"current_agent": agentName, "can_join": hasCollaborate},
			"already_assigned")
	}

	// Guard 3: already assigned to self (idempotent)
	if contact.AssignedUserID != nil && *contact.AssignedUserID == userID {
		if contact.EffectiveStatus() != models.ChatStatusOpen {
			contact.SetStatus(models.ChatStatusOpen)
			a.DB.Model(&contact).Update("metadata", contact.Metadata)
		}
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"assigned":   true,
			"message":    "Already assigned to you",
		})
	}

	// Track if this is a reopen (was closed)
	wasClosed := contact.EffectiveStatus() == models.ChatStatusClosed

	// Capture pre-mutation values for the audit log BEFORE mutation. We cannot
	// rely on a struct snapshot (`oldContact := contact`) for status, because
	// status lives in the JSONB `Metadata` map, which the snapshot aliases — by
	// the time we build the audit diff, the shared map has already been mutated.
	oldStatus := string(contact.EffectiveStatus())
	oldAssigned := contact.AssignedUserID

	// Action: assign + set status to open
	contact.AssignedUserID = &userID
	contact.SetStatus(models.ChatStatusOpen)
	if err := a.DB.Save(&contact).Error; err != nil {
		a.Log.Error("Failed to claim chat", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to claim chat", nil, "")
	}

	// Load agent name for system message
	var agent models.User
	agentName := "Unknown"
	if a.DB.First(&agent, "id = ?", userID).Error == nil {
		agentName = agent.FullName
	}

	// Different message for reopen vs claim
	if wasClosed {
		a.createSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s reopened this conversation", agentName),
			models.JSONB{"system_type": "chat_reopened", "agent_id": userID.String(), "agent_name": agentName})
	} else {
		a.createSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s claimed this conversation", agentName),
			models.JSONB{"system_type": "chat_claimed", "agent_id": userID.String(), "agent_name": agentName})
	}

	// Audit: extraChanges safeguard defeats the audit.LogAudit no-op-on-empty-diff.
	// oldStatus/oldAssigned were captured BEFORE mutation so the recorded diff
	// reflects the true pre-mutation state (status lives in JSONB metadata, so a
	// struct snapshot would alias the already-mutated map).
	a.logAudit(orgID, userID, "contact", contact.ID, models.AuditActionUpdated, nil, &contact,
		map[string]any{
			"chat_status":      map[string]any{"old": oldStatus, "new": string(models.ChatStatusOpen)},
			"assigned_user_id": map[string]any{"old": oldAssigned, "new": &userID},
		})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeChatClaimed,
			Payload: map[string]any{
				"contact_id":       contact.ID.String(),
				"assigned_to":      userID.String(),
				"assigned_user_id": userID.String(),
				"assigned_to_name": agentName,
				"chat_status":      string(models.ChatStatusOpen),
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"assigned":   true,
		"agent_name": agentName,
	})
}

// ReleaseChat returns an assigned (open) conversation to the pending pool.
// It unassigns the current owner, sets status to pending, and clears collaborators.
// Route: PUT /api/contacts/{id}/release
// Permission: chat.assign:write (caller must be the assignee or an admin/manager).
func (a *App) ReleaseChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Authorization: assignee or admin/manager (ghost-release).
	isAssignee := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isAdminOrManager := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
	if !isAssignee && !isAdminOrManager {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You are not allowed to release this chat", nil, "")
	}

	// Idempotent: already pending and unassigned — safe no-op success.
	if contact.EffectiveStatus() == models.ChatStatusPending && contact.AssignedUserID == nil {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"released":   true,
			"message":    "Conversation is already pending",
		})
	}

	// Capture pre-mutation values for the audit log BEFORE mutation. We cannot
	// rely on a struct snapshot (`oldContact := contact`) for status, because
	// status lives in the JSONB `Metadata` map, which the snapshot aliases — by
	// the time we build the audit diff, the shared map has already been mutated.
	oldStatus := string(contact.EffectiveStatus())
	oldAssigned := contact.AssignedUserID

	// Mutation: unassign + set pending + clear collaborators.
	contact.AssignedUserID = nil
	contact.SetStatus(models.ChatStatusPending)
	contact.ClearCollaborators()
	if err := a.DB.Model(&contact).Updates(map[string]any{
		"assigned_user_id": nil,
		"metadata":         contact.Metadata,
	}).Error; err != nil {
		a.Log.Error("Failed to release chat", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to release chat", nil, "")
	}

	// Load agent name for system message.
	var agent models.User
	agentName := "Unknown"
	if a.DB.First(&agent, "id = ?", userID).Error == nil {
		agentName = agent.FullName
	}

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s released this conversation", agentName),
		models.JSONB{"system_type": "chat_released", "agent_id": userID.String(), "agent_name": agentName})

	// Audit: extraChanges safeguard defeats the audit.LogAudit no-op-on-empty-diff.
	// oldStatus/oldAssigned were captured BEFORE mutation so the recorded diff
	// reflects the true pre-mutation state (status lives in JSONB metadata, so a
	// struct snapshot would alias the already-mutated map).
	a.logAudit(orgID, userID, "contact", contact.ID, models.AuditActionUpdated, nil, &contact,
		map[string]any{
			"chat_status":      map[string]any{"old": oldStatus, "new": string(models.ChatStatusPending)},
			"assigned_user_id": map[string]any{"old": oldAssigned, "new": nil},
		})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeChatReleased,
			Payload: map[string]any{
				"contact_id":  contact.ID.String(),
				"released_by": userID.String(),
				"chat_status": string(models.ChatStatusPending),
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":  contact.ID,
		"released":    true,
		"chat_status": "pending",
	})
}

// JoinChat allows a user with chat.collaborate:write to join an assigned conversation.
// Route: POST /api/contacts/{id}/join
// Permission: chat.collaborate:write
func (a *App) JoinChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatCollaborate, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Already the primary owner?
	if contact.AssignedUserID != nil && *contact.AssignedUserID == userID {
		return r.SendEnvelope(map[string]any{
			"message": "You are the primary owner of this conversation",
		})
	}

	// Already a collaborator?
	if contact.IsCollaborator(userID.String()) {
		return r.SendEnvelope(map[string]any{
			"message": "You are already a collaborator",
		})
	}

	return a.joinAsCollaborator(r, &contact, userID, orgID)
}

// ─────────────────────────────────────────────────────────────────────────────
// InviteCollaborator allows a manager/admin to add another agent as a
// collaborator to a conversation.
// Route: POST /api/contacts/{id}/collaborators/{user_id}
// Permission: chat.collaborate:write
// ─────────────────────────────────────────────────────────────────────────────
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

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Verify target user exists in same org
	var targetUser models.User
	if err := a.DB.Where("id = ? AND organization_id = ?", targetUserID, orgID).First(&targetUser).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
	}

	// Can't invite the owner (they're already the owner)
	if contact.AssignedUserID != nil && *contact.AssignedUserID == targetUserID {
		return r.SendEnvelope(map[string]any{
			"message": "User is already the primary owner",
		})
	}

	// Already a collaborator?
	if contact.IsCollaborator(targetUserIDStr) {
		return r.SendEnvelope(map[string]any{
			"message": "User is already a collaborator",
		})
	}

	// Load inviter name
	var inviter models.User
	inviterName := "Unknown"
	if a.DB.First(&inviter, "id = ?", userID).Error == nil {
		inviterName = inviter.FullName
	}

	targetRole := ""
	if targetUser.RoleID != nil {
		var role models.CustomRole
		if a.DB.First(&role, "id = ?", *targetUser.RoleID).Error == nil {
			targetRole = role.Name
		}
	}

	contact.AddCollaborator(models.Collaborator{
		UserID:   targetUserIDStr,
		Name:     targetUser.FullName,
		Role:     targetRole,
		JoinedAt: time.Now(),
	})
	if err := a.DB.Model(&contact).Update("metadata", contact.Metadata).Error; err != nil {
		a.Log.Error("Failed to invite collaborator", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to invite collaborator", nil, "")
	}

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s was added to the conversation by %s", targetUser.FullName, inviterName),
		models.JSONB{
			"system_type": "collaborator_joined",
			"agent_id":    targetUserIDStr,
			"invited_by":  userID.String(),
		})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeCollaboratorJoined,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    targetUserIDStr,
				"user_name":  targetUser.FullName,
				"user_role":  targetRole,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":   contact.ID,
		"collaborator": true,
		"user_name":    targetUser.FullName,
	})
}

// joinAsCollaborator is the shared logic for joining a conversation as collaborator.
func (a *App) joinAsCollaborator(r *fastglue.Request, contact *models.Contact, userID, orgID uuid.UUID) error {
	var user models.User
	userName := "Unknown"
	userRole := ""
	if a.DB.First(&user, "id = ?", userID).Error == nil {
		userName = user.FullName
		if user.RoleID != nil {
			var role models.CustomRole
			if a.DB.First(&role, "id = ?", *user.RoleID).Error == nil {
				userRole = role.Name
			}
		}
	}

	contact.AddCollaborator(models.Collaborator{
		UserID:   userID.String(),
		Name:     userName,
		Role:     userRole,
		JoinedAt: time.Now(),
	})
	if err := a.DB.Model(contact).Update("metadata", contact.Metadata).Error; err != nil {
		a.Log.Error("Failed to join chat", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to join chat", nil, "")
	}

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s joined the conversation", userName),
		models.JSONB{"system_type": "collaborator_joined", "agent_id": userID.String()})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeCollaboratorJoined,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    userID.String(),
				"user_name":  userName,
				"user_role":  userRole,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":   contact.ID,
		"collaborator": true,
		"user_name":    userName,
	})
}

// LeaveChat removes the requesting user from the collaborators list.
// If the user is the primary owner and the last participant, the conversation is closed.
// Route: DELETE /api/contacts/{id}/join
func (a *App) LeaveChat(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	isOwner := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	// Admins/managers ghost-view chats: they are not persisted as collaborators,
	// but they must be able to Leave (ghost-exit) and Close without joining first.
	isAdminOrManager := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)

	if !isOwner && !isCollaborator && !isAdminOrManager {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"You are not a participant in this conversation", nil, "")
	}

	// Admin/manager ghost-exit: not a real participant, so just return success
	// without modifying the conversation's collaborators or status.
	if !isOwner && !isCollaborator && isAdminOrManager {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"left":       true,
			"ghost_exit": true,
		})
	}

	var user models.User
	userName := "Unknown"
	if a.DB.First(&user, "id = ?", userID).Error == nil {
		userName = user.FullName
	}

	// If owner is leaving: check if there are other collaborators
	if isOwner {
		collaborators := contact.GetCollaborators()
		if len(collaborators) == 0 {
			// Last participant leaving → close the conversation
			contact.AssignedUserID = nil
			contact.ClearCollaborators()
			contact.SetStatus(models.ChatStatusClosed)
			a.DB.Model(&contact).Updates(map[string]any{
				"assigned_user_id": nil,
				"metadata":         contact.Metadata,
			})

			a.createSystemMessage(orgID, contact.ID,
				fmt.Sprintf("🔔 %s closed this conversation", userName),
				models.JSONB{"system_type": "chat_closed", "agent_id": userID.String()})

			if a.WSHub != nil {
				a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
					Type: websocket.TypeChatClosed,
					Payload: map[string]any{
						"contact_id":  contact.ID.String(),
						"chat_status": string(models.ChatStatusClosed),
						"closed":      true,
					},
				})
			}

			return r.SendEnvelope(map[string]any{
				"contact_id": contact.ID,
				"left":       true,
				"closed":     true,
			})
		}
		// Owner leaves but collaborators remain — transfer ownership to first collaborator
		newOwnerID, _ := uuid.Parse(collaborators[0].UserID)
		contact.AssignedUserID = &newOwnerID
		contact.RemoveCollaborator(collaborators[0].UserID)
		a.DB.Model(&contact).Updates(map[string]any{
			"assigned_user_id": newOwnerID,
			"metadata":         contact.Metadata,
		})

		a.createSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s left the conversation. Ownership transferred to %s", userName, collaborators[0].Name),
			models.JSONB{"system_type": "collaborator_left", "agent_id": userID.String()})

		if a.WSHub != nil {
			a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeCollaboratorLeft,
				Payload: map[string]any{
					"contact_id": contact.ID.String(),
					"user_id":    userID.String(),
					"user_name":  userName,
				},
			})
		}

		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"left":       true,
		})
	}

	// Regular collaborator leaving
	contact.RemoveCollaborator(userID.String())
	a.DB.Model(&contact).Update("metadata", contact.Metadata)

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s left the conversation", userName),
		models.JSONB{"system_type": "collaborator_left", "agent_id": userID.String()})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeCollaboratorLeft,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    userID.String(),
				"user_name":  userName,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"left":       true,
	})
}

// RemoveCollaborator allows a manager/admin to remove a collaborator from a conversation.
// Route: DELETE /api/contacts/{id}/collaborators/{user_id}
// Permission: chat.collaborate:write
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

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Cannot remove the primary owner
	if contact.AssignedUserID != nil && *contact.AssignedUserID == targetUserID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Cannot remove the primary owner. Unassign the conversation instead.", nil, "")
	}

	if !contact.IsCollaborator(targetUserIDStr) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"User is not a collaborator on this conversation", nil, "")
	}

	// Load names for system message
	var targetUser models.User
	targetName := "Unknown"
	if a.DB.First(&targetUser, "id = ?", targetUserID).Error == nil {
		targetName = targetUser.FullName
	}

	var manager models.User
	managerName := "Unknown"
	if a.DB.First(&manager, "id = ?", userID).Error == nil {
		managerName = manager.FullName
	}

	contact.RemoveCollaborator(targetUserIDStr)
	a.DB.Model(&contact).Update("metadata", contact.Metadata)

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s was removed from the conversation by %s", targetName, managerName),
		models.JSONB{
			"system_type": "collaborator_removed",
			"agent_id":    targetUserIDStr,
			"removed_by":  userID.String(),
		})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeCollaboratorLeft,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    targetUserIDStr,
				"user_name":  targetName,
				"removed":    true,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":   contact.ID,
		"removed":      true,
		"removed_user": targetName,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// CloseChat closes an open conversation. Only the assigned agent or a
// manager/admin (chat.collaborate:write) can close it.
// Route: PUT /api/contacts/{id}/close
// ─────────────────────────────────────────────────────────────────────────────
func (a *App) CloseChat(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Only the assigned agent or a manager/admin can close
	isOwner := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	hasCollaboratePerm := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	hasContactsRead := a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID)

	if !isOwner && !isCollaborator && !hasCollaboratePerm && !hasContactsRead {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	// Guard: already closed
	if contact.EffectiveStatus() == models.ChatStatusClosed {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"message":    "Conversation is already closed",
		})
	}

	// Load agent name for system message
	var agent models.User
	agentName := "Unknown"
	if a.DB.First(&agent, "id = ?", userID).Error == nil {
		agentName = agent.FullName
	}

	// Close the conversation
	contact.SetStatus(models.ChatStatusClosed)
	if err := a.DB.Model(&contact).Update("metadata", contact.Metadata).Error; err != nil {
		a.Log.Error("Failed to close chat", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
	}

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s closed this conversation", agentName),
		models.JSONB{"system_type": "chat_closed", "agent_id": userID.String()})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeChatClosed,
			Payload: map[string]any{
				"contact_id":       contact.ID.String(),
				"chat_status":      string(models.ChatStatusClosed),
				"closed":           true,
				"assigned_user_id": "",
				"assigned_to":      "",
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"closed":     true,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// ReopenChat reopens a closed conversation WITHOUT assigning it to the caller.
// Only managers/admins (contacts:write) can reopen — agents must Claim to take
// ownership. Per spec §3: "To view again, user must click [Reopen Conversation].
// This sets status='open'." The last_closed_at timestamp is preserved so the
// per-agent unread-count logic (created_at > last_closed_at) stays correct.
// Route: PUT /api/contacts/{id}/reopen
// ─────────────────────────────────────────────────────────────────────────────
func (a *App) ReopenChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceContacts, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Idempotent: already open
	if contact.EffectiveStatus() == models.ChatStatusOpen {
		return r.SendEnvelope(map[string]any{
			"contact_id": contact.ID,
			"reopened":   false,
			"message":    "Conversation is already open",
		})
	}

	contact.SetStatus(models.ChatStatusOpen)
	if err := a.DB.Model(&contact).Update("metadata", contact.Metadata).Error; err != nil {
		a.Log.Error("Failed to reopen chat", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reopen chat", nil, "")
	}

	var admin models.User
	adminName := "Unknown"
	if a.DB.First(&admin, "id = ?", userID).Error == nil {
		adminName = admin.FullName
	}

	a.createSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s reopened this conversation", adminName),
		models.JSONB{"system_type": "chat_reopened", "agent_id": userID.String()})

	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeChatReopened,
			Payload: map[string]any{
				"contact_id":  contact.ID.String(),
				"chat_status": string(models.ChatStatusOpen),
				"reopened":    true,
				"by":          userID.String(),
				"by_name":     adminName,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id": contact.ID,
		"reopened":   true,
	})
}
