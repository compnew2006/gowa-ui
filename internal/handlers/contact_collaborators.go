package handlers

import (
	"errors"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

var errCollaboratorForbidden = errors.New("contact collaborator forbidden")

type contactCollaboratorRow struct {
	ID             uuid.UUID                `gorm:"column:id"`
	ContactID      uuid.UUID                `gorm:"column:contact_id"`
	UserID         uuid.UUID                `gorm:"column:user_id"`
	UserName       *string                  `gorm:"column:user_name"`
	Role           models.CollaboratorRole  `gorm:"column:role"`
	Status         models.CollaboratorStatus `gorm:"column:status"`
	InvitedByUserID uuid.UUID               `gorm:"column:invited_by_user_id"`
	InvitedByName  *string                  `gorm:"column:invited_by_name"`
	CreatedAt      time.Time                `gorm:"column:created_at"`
	AcceptedAt     *time.Time               `gorm:"column:accepted_at"`
}

type ContactCollaboratorResponse struct {
	ID              string  `json:"id"`
	ContactID       string  `json:"contact_id"`
	UserID          string  `json:"user_id"`
	UserName        string  `json:"user_name,omitempty"`
	Role            string  `json:"role"`
	Status          string  `json:"status"`
	InvitedByUserID string  `json:"invited_by_user_id"`
	InvitedByName   string  `json:"invited_by_name,omitempty"`
	InvitedAt       string  `json:"invited_at"`
	AcceptedAt      *string `json:"accepted_at,omitempty"`
}

type InviteContactCollaboratorRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role,omitempty"`
}

type updateCollaboratorStatusRequest struct {
	Status string `json:"status"`
}

func (a *App) loadContactForCollaboration(r *fastglue.Request, orgID, userID, contactID uuid.UUID) (*models.Contact, error) {
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}

	restrictedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	query = applyRestrictedInstanceVisibilityFilter(query, restrictedInstanceIDs)

	if err := query.First(&contact).Error; err != nil {
		return nil, err
	}
	normalizeContactStatus(&contact)
	if isChatRestrictedForMessageRead(contact) && !a.canAccessRestrictedChatWithoutClaim(contact, userID, orgID) {
		return nil, errCollaboratorForbidden
	}

	return &contact, nil
}

// ListContactCollaborators lists collaborators for a contact.
func (a *App) ListContactCollaborators(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	if _, err := a.loadContactForCollaboration(r, orgID, userID, contactID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		}
		if errors.Is(err, errCollaboratorForbidden) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this chat", nil, "")
		}
		a.Log.Error("Failed to load contact for collaborators", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load collaborators", nil, "")
	}

	var rows []contactCollaboratorRow
	query := a.DB.Table("contact_collaborators").
		Select("contact_collaborators.id, contact_collaborators.contact_id, contact_collaborators.user_id, contact_collaborators.role, contact_collaborators.status, contact_collaborators.invited_by_user_id, contact_collaborators.created_at, contact_collaborators.accepted_at, users.full_name as user_name, invited_by.full_name as invited_by_name").
		Joins("LEFT JOIN users ON users.id = contact_collaborators.user_id").
		Joins("LEFT JOIN users AS invited_by ON invited_by.id = contact_collaborators.invited_by_user_id").
		Where("contact_collaborators.organization_id = ? AND contact_collaborators.contact_id = ? AND contact_collaborators.deleted_at IS NULL", orgID, contactID).
		Order("contact_collaborators.created_at ASC")

	if err := query.Scan(&rows).Error; err != nil {
		a.Log.Error("Failed to list contact collaborators", "error", err, "contact_id", contactID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load collaborators", nil, "")
	}

	response := make([]ContactCollaboratorResponse, 0, len(rows))
	for _, row := range rows {
		var acceptedAt *string
		if row.AcceptedAt != nil {
			val := row.AcceptedAt.UTC().Format(time.RFC3339)
			acceptedAt = &val
		}
		response = append(response, ContactCollaboratorResponse{
			ID:              row.ID.String(),
			ContactID:       row.ContactID.String(),
			UserID:          row.UserID.String(),
			UserName:        stringPtrValue(row.UserName),
			Role:            string(row.Role),
			Status:          string(row.Status),
			InvitedByUserID: row.InvitedByUserID.String(),
			InvitedByName:   stringPtrValue(row.InvitedByName),
			InvitedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
			AcceptedAt:      acceptedAt,
		})
	}

	return r.SendEnvelope(map[string]any{"collaborators": response})
}

// InviteContactCollaborator invites a user to collaborate on a contact.
func (a *App) InviteContactCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.HasPermission(userID, models.ResourceChatCollaborators, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to invite collaborators", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req InviteContactCollaboratorRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	inviteUserID, parseErr := uuid.Parse(req.UserID)
	if parseErr != nil || inviteUserID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user_id", nil, "user_id")
	}
	if inviteUserID == userID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot invite yourself", nil, "user_id")
	}

	contact, err := a.loadContactForCollaboration(r, orgID, userID, contactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		}
		if errors.Is(err, errCollaboratorForbidden) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this chat", nil, "")
		}
		a.Log.Error("Failed to load contact for invite", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to invite collaborator", nil, "")
	}

	allowed, err := a.canUserSeeContactInstance(orgID, inviteUserID, contact)
	if err != nil {
		a.Log.Error("Failed to validate invitee instance access", "error", err, "user_id", inviteUserID, "contact_id", contactID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to validate collaborator access", nil, "")
	}
	if !allowed {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invitee does not have access to this WhatsApp account", nil, "")
	}

	var invitee models.User
	if err := a.DB.Where("id = ? AND organization_id = ?", inviteUserID, orgID).First(&invitee).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
	}

	role := models.CollaboratorRoleAssistant
	if req.Role != "" {
		switch models.CollaboratorRole(req.Role) {
		case models.CollaboratorRoleAssistant, models.CollaboratorRoleViewer:
			role = models.CollaboratorRole(req.Role)
		default:
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid collaborator role", nil, "role")
		}
	}

	var collaborator models.ContactCollaborator
	err = a.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ? AND deleted_at IS NULL", orgID, contactID, inviteUserID).
		First(&collaborator).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.Log.Error("Failed to lookup collaborator", "error", err, "contact_id", contactID, "user_id", inviteUserID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to invite collaborator", nil, "")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		collaborator = models.ContactCollaborator{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  orgID,
			ContactID:       contactID,
			UserID:          inviteUserID,
			Role:            role,
			Status:          models.CollaboratorStatusInvited,
			InvitedByUserID: userID,
		}
		if err := a.DB.Create(&collaborator).Error; err != nil {
			a.Log.Error("Failed to create collaborator invite", "error", err, "contact_id", contactID, "user_id", inviteUserID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to invite collaborator", nil, "")
		}
	} else {
		updates := map[string]any{
			"role":              role,
			"status":            models.CollaboratorStatusInvited,
			"invited_by_user_id": userID,
			"accepted_at":       nil,
			"declined_at":       nil,
		}
		if err := a.DB.Model(&collaborator).Updates(updates).Error; err != nil {
			a.Log.Error("Failed to update collaborator invite", "error", err, "contact_id", contactID, "user_id", inviteUserID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to invite collaborator", nil, "")
		}
	}

	a.appendSystemChatMessage(contact, "System: A collaborator has been invited to this chat.", models.JSONB{
		"event_type":         "chat_collaborator_invited",
		"invited_by_user_id": userID.String(),
		"invited_user_id":    inviteUserID.String(),
	})

	if a.WSHub != nil {
		a.WSHub.BroadcastToUser(orgID, inviteUserID, websocket.WSMessage{
			Type: websocket.TypeChatCollaboratorInvite,
			Payload: map[string]any{
				"contact_id":      contactID.String(),
				"invited_by":      userID.String(),
				"invited_by_name": a.ResolveActivityActorName(userID),
			},
		})
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type: websocket.TypeChatCollaboratorUpdate,
			Payload: map[string]any{
				"contact_id": contactID.String(),
			},
		})
	}

	return r.SendEnvelope(map[string]any{"message": "Collaborator invited"})
}

// AcceptContactCollaborator accepts a collaboration invite for the current user.
func (a *App) AcceptContactCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	collabUserID, err := parsePathUUID(r, "user_id", "user")
	if err != nil {
		return nil
	}
	if collabUserID != userID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You can only accept your own invite", nil, "")
	}

	contact, err := a.loadContactForCollaboration(r, orgID, userID, contactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		}
		if errors.Is(err, errCollaboratorForbidden) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this chat", nil, "")
		}
		a.Log.Error("Failed to load contact for collaborator accept", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to accept invite", nil, "")
	}

	var collaborator models.ContactCollaborator
	if err := a.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ? AND deleted_at IS NULL", orgID, contactID, userID).
		First(&collaborator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Invite not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to accept invite", nil, "")
	}
	if collaborator.Status == models.CollaboratorStatusAccepted {
		return r.SendEnvelope(map[string]any{"message": "Invite already accepted"})
	}
	if collaborator.Status == models.CollaboratorStatusDeclined {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Invite already declined", nil, "")
	}

	now := time.Now().UTC()
	if err := a.DB.Model(&collaborator).Updates(map[string]any{
		"status":      models.CollaboratorStatusAccepted,
		"accepted_at": &now,
		"declined_at": nil,
	}).Error; err != nil {
		a.Log.Error("Failed to accept collaborator invite", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to accept invite", nil, "")
	}

	a.appendSystemChatMessage(contact, "System: A collaborator joined this chat.", models.JSONB{
		"event_type":      "chat_collaborator_accepted",
		"collaborator_id": userID.String(),
	})

	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type: websocket.TypeChatCollaboratorUpdate,
			Payload: map[string]any{
				"contact_id": contactID.String(),
			},
		})
	}

	return r.SendEnvelope(map[string]any{"message": "Invite accepted"})
}

// DeclineContactCollaborator declines a collaboration invite for the current user.
func (a *App) DeclineContactCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	collabUserID, err := parsePathUUID(r, "user_id", "user")
	if err != nil {
		return nil
	}
	if collabUserID != userID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You can only decline your own invite", nil, "")
	}

	var collaborator models.ContactCollaborator
	if err := a.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ? AND deleted_at IS NULL", orgID, contactID, userID).
		First(&collaborator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Invite not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to decline invite", nil, "")
	}
	if collaborator.Status == models.CollaboratorStatusDeclined {
		return r.SendEnvelope(map[string]any{"message": "Invite already declined"})
	}
	if collaborator.Status == models.CollaboratorStatusAccepted {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Invite already accepted", nil, "")
	}

	now := time.Now().UTC()
	if err := a.DB.Model(&collaborator).Updates(map[string]any{
		"status":      models.CollaboratorStatusDeclined,
		"declined_at": &now,
	}).Error; err != nil {
		a.Log.Error("Failed to decline collaborator invite", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to decline invite", nil, "")
	}

	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type: websocket.TypeChatCollaboratorUpdate,
			Payload: map[string]any{
				"contact_id": contactID.String(),
			},
		})
	}

	return r.SendEnvelope(map[string]any{"message": "Invite declined"})
}

// RemoveContactCollaborator removes a collaborator from a contact.
func (a *App) RemoveContactCollaborator(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	collabUserID, err := parsePathUUID(r, "user_id", "user")
	if err != nil {
		return nil
	}

	if _, err := a.loadContactForCollaboration(r, orgID, userID, contactID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		}
		if errors.Is(err, errCollaboratorForbidden) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this chat", nil, "")
		}
		a.Log.Error("Failed to load contact for collaborator removal", "error", err, "contact_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove collaborator", nil, "")
	}

	if collabUserID != userID && !a.HasPermission(userID, models.ResourceChatCollaborators, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to remove collaborators", nil, "")
	}

	var collaborator models.ContactCollaborator
	if err := a.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ? AND deleted_at IS NULL", orgID, contactID, collabUserID).
		First(&collaborator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Collaborator not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove collaborator", nil, "")
	}

	if err := a.DB.Delete(&collaborator).Error; err != nil {
		a.Log.Error("Failed to remove collaborator", "error", err, "contact_id", contactID, "user_id", collabUserID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to remove collaborator", nil, "")
	}

	if a.WSHub != nil {
		a.WSHub.BroadcastToContact(orgID, contactID, websocket.WSMessage{
			Type: websocket.TypeChatCollaboratorUpdate,
			Payload: map[string]any{
				"contact_id": contactID.String(),
			},
		})
	}

	return r.SendEnvelope(map[string]any{"message": "Collaborator removed"})
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
