package handlers

import (
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// AssignContactRequest represents the request to assign a contact to a user
type AssignContactRequest struct {
	UserID *uuid.UUID `json:"user_id"` // nil to unassign
}

// AssignContact assigns a contact to a user (agent)
// Only users with write permission can assign contacts
func (a *App) AssignContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Only users with write permission can assign contacts
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to assign contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req AssignContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// If assigning to a user, verify they exist in the same org
	if req.UserID != nil {
		var user models.User
		if err := a.DB.Where("id = ? AND organization_id = ?", req.UserID, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
		}
	}

	var previousAssignedUserID *uuid.UUID
	if contact.AssignedUserID != nil {
		prev := *contact.AssignedUserID
		previousAssignedUserID = &prev
	}

	// Update contact assignment + lifecycle status
	if err := a.DB.Model(contact).Updates(chatAssignmentUpdates(req.UserID)).Error; err != nil {
		a.Log.Error("Failed to assign contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	if err := a.DB.Where("id = ?", contact.ID).First(contact).Error; err != nil {
		a.Log.Error("Failed to reload contact after assignment", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	notifyAssignee := false
	if contact.AssignedUserID != nil {
		notifyAssignee = previousAssignedUserID == nil || *previousAssignedUserID != *contact.AssignedUserID
	}
	a.broadcastContactLifecycleUpdate(orgID, contact, notifyAssignee)

	return r.SendEnvelope(map[string]any{
		"message":          "Contact assigned successfully",
		"assigned_user_id": req.UserID,
	})
}

// ClaimChat claims a pending chat for the current user.
func (a *App) ClaimChat(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceChat, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to claim chats", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "chat")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Closed chats cannot be claimed", nil, "")
	}
	if contact.AssignedUserID != nil && *contact.AssignedUserID != userID {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Chat is already assigned to another user", nil, "")
	}
	if status != models.ChatStatusPending && contact.AssignedUserID != nil && *contact.AssignedUserID == userID {
		_ = a.DB.Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error
		return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
	}

	if err := a.DB.Model(&contact).Updates(chatAssignmentUpdates(&userID)).Error; err != nil {
		a.Log.Error("Failed to claim chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to claim chat", nil, "")
	}

	if err := a.DB.Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
}

// CloseChat marks a chat as closed.
func (a *App) CloseChat(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceChat, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to close chats", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "chat")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		_ = a.DB.Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error
		return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
	}

	if contact.AssignedUserID != nil && *contact.AssignedUserID != userID && !a.canBypassPendingChatRestriction(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only the assigned user can close this chat", nil, "")
	}

	if err := a.DB.Model(&contact).Updates(closeChatUpdates(userID, contact.AssignedUserID)).Error; err != nil {
		a.Log.Error("Failed to close chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
	}

	if err := a.DB.Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
}

// ReopenChat reopens a closed chat and moves it back to pending unassigned queue.
func (a *App) ReopenChat(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceChat, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to reopen chats", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "chat")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	if normalizeContactStatus(&contact) != models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Only closed chats can be reopened", nil, "")
	}

	if err := a.DB.Model(&contact).Updates(reopenChatUpdates()).Error; err != nil {
		a.Log.Error("Failed to reopen chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reopen chat", nil, "")
	}

	if err := a.DB.Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
}

// ContactSessionDataResponse represents the session data for a contact's info panel
type ContactSessionDataResponse struct {
	SessionID   *uuid.UUID     `json:"session_id,omitempty"`
	FlowID      *uuid.UUID     `json:"flow_id,omitempty"`
	FlowName    string         `json:"flow_name,omitempty"`
	SessionData map[string]any `json:"session_data"`
	PanelConfig map[string]any `json:"panel_config"`
}

// GetContactSessionData returns session data and panel configuration for a contact
// Used by the contact info panel in the chat view
func (a *App) GetContactSessionData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Verify contact belongs to org (users without full read permission can only access assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		query = query.Where("assigned_user_id = ?", userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	response := ContactSessionDataResponse{
		SessionData: make(map[string]any),
		PanelConfig: map[string]any{"sections": []any{}},
	}

	// Get the most recent completed or active session for this contact
	var session models.ChatbotSession
	err = a.DB.Where("contact_id = ? AND organization_id = ?", contactID, orgID).
		Where("status IN ?", []models.SessionStatus{models.SessionStatusActive, models.SessionStatusCompleted}).
		Order("created_at DESC").
		First(&session).Error

	if err == nil {
		response.SessionID = &session.ID
		response.FlowID = session.CurrentFlowID

		// Get the flow to retrieve panel config
		// First try current_flow_id, then fall back to _flow_id in session_data
		var flowID *uuid.UUID
		if session.CurrentFlowID != nil {
			flowID = session.CurrentFlowID
		} else if flowIDStr, ok := session.SessionData["_flow_id"].(string); ok {
			if parsedID, err := uuid.Parse(flowIDStr); err == nil {
				flowID = &parsedID
			}
		}

		if flowID != nil {
			// Use cached flow to avoid DB query
			flow, err := a.getChatbotFlowByIDCached(orgID, *flowID)
			if err == nil && flow != nil {
				response.FlowName = flow.Name
				response.FlowID = flowID

				// Use panel config directly from flow (it's already JSONB/map)
				if len(flow.PanelConfig) > 0 {
					response.PanelConfig = flow.PanelConfig

					// Only include session data for configured fields (reduce payload)
					if session.SessionData != nil {
						configuredKeys := make(map[string]bool)
						if sections, ok := flow.PanelConfig["sections"].([]any); ok {
							for _, sec := range sections {
								if section, ok := sec.(map[string]any); ok {
									if fields, ok := section["fields"].([]any); ok {
										for _, f := range fields {
											if field, ok := f.(map[string]any); ok {
												if key, ok := field["key"].(string); ok {
													configuredKeys[key] = true
												}
											}
										}
									}
								}
							}
						}
						// Copy only configured fields to response
						for key := range configuredKeys {
							if val, exists := session.SessionData[key]; exists {
								response.SessionData[key] = val
							}
						}
					}
				}
			}
		}
	}

	return r.SendEnvelope(response)
}

// UpdateContactTagsRequest represents the request body for updating contact tags
type UpdateContactTagsRequest struct {
	Tags []string `json:"tags"`
}

// UpdateContactTags updates the tags on a contact
func (a *App) UpdateContactTags(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission - need contacts:write to update tags on contacts
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to update contact tags", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req UpdateContactTagsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// Convert tags to JSONBArray
	tagsArray := make(models.JSONBArray, len(req.Tags))
	for i, tag := range req.Tags {
		tagsArray[i] = tag
	}

	// Update contact tags
	if err := a.DB.Model(contact).Update("tags", tagsArray).Error; err != nil {
		a.Log.Error("Failed to update contact tags", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact tags", nil, "")
	}

	// Reload contact to get updated tags
	if err := a.DB.First(contact, contactID).Error; err != nil {
		a.Log.Error("Failed to reload contact", "error", err)
	}

	// Build response with tag details
	tags := []string{}
	if contact.Tags != nil {
		for _, t := range contact.Tags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	return r.SendEnvelope(map[string]any{
		"message": "Contact tags updated",
		"tags":    tags,
	})
}

// CreateContactRequest represents the request body for creating a contact
type CreateContactRequest struct {
	PhoneNumber     string         `json:"phone_number"`
	ProfileName     string         `json:"profile_name"`
	WhatsAppAccount string         `json:"whatsapp_account"`
	InstanceID      string         `json:"instance_id,omitempty"`
	Tags            []string       `json:"tags"`
	Metadata        map[string]any `json:"metadata"`
}

// CreateContact creates a new contact or restores a soft-deleted one
func (a *App) CreateContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to create contacts", nil, "")
	}

	var req CreateContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.PhoneNumber == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_number is required", nil, "")
	}

	resolvedInstanceID, err := a.resolveContactInstanceID(orgID, req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "instance_id")
	}

	// Normalize phone number
	normalizedPhone := req.PhoneNumber
	if len(normalizedPhone) > 0 && normalizedPhone[0] == '+' {
		normalizedPhone = normalizedPhone[1:]
	}

	// Check if contact exists (including soft-deleted)
	var existingContact models.Contact
	existingQuery := a.DB.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone)
	if resolvedInstanceID != nil {
		existingQuery = existingQuery.Where("instance_id = ?", *resolvedInstanceID)
	} else {
		existingQuery = existingQuery.Where("instance_id IS NULL")
	}
	if err := existingQuery.First(&existingContact).Error; err == nil {
		// Contact exists
		if existingContact.DeletedAt.Valid {
			// Restore soft-deleted contact
			a.DB.Unscoped().Model(&existingContact).Update("deleted_at", nil)
			existingContact.DeletedAt.Valid = false
			// Update fields
			updates := map[string]any{}
			if req.ProfileName != "" {
				updates["profile_name"] = req.ProfileName
			}
			if req.WhatsAppAccount != "" {
				updates["whats_app_account"] = req.WhatsAppAccount
			}
			if req.InstanceID != "" {
				updates["instance_id"] = resolvedInstanceID
			}
			if req.Tags != nil {
				tagsArray := make(models.JSONBArray, len(req.Tags))
				for i, tag := range req.Tags {
					tagsArray[i] = tag
				}
				updates["tags"] = tagsArray
			}
			if req.Metadata != nil {
				updates["metadata"] = models.JSONB(req.Metadata)
			}
			if len(updates) > 0 {
				a.DB.Model(&existingContact).Updates(updates)
			}
			// Reload contact
			a.DB.First(&existingContact, existingContact.ID)
			return r.SendEnvelope(a.buildContactResponse(&existingContact, orgID))
		}
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Contact with this phone number already exists", nil, "")
	}

	// Create new contact
	contact := models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		InstanceID:      resolvedInstanceID,
		PhoneNumber:     normalizedPhone,
		ProfileName:     req.ProfileName,
		WhatsAppAccount: req.WhatsAppAccount,
		Status:          models.ChatStatusPending,
	}

	if req.Tags != nil {
		tagsArray := make(models.JSONBArray, len(req.Tags))
		for i, tag := range req.Tags {
			tagsArray[i] = tag
		}
		contact.Tags = tagsArray
	}

	if req.Metadata != nil {
		contact.Metadata = models.JSONB(req.Metadata)
	}

	if err := a.DB.Create(&contact).Error; err != nil {
		a.Log.Error("Failed to create contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create contact", nil, "")
	}

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID))
}

// UpdateContactRequest represents the request body for updating a contact
type UpdateContactRequest struct {
	ProfileName     *string         `json:"profile_name"`
	WhatsAppAccount *string         `json:"whatsapp_account"`
	InstanceID      *string         `json:"instance_id,omitempty"`
	Tags            []string        `json:"tags"`
	Metadata        *map[string]any `json:"metadata"`
	AssignedUserID  *uuid.UUID      `json:"assigned_user_id"`
}

// UpdateContact updates an existing contact
func (a *App) UpdateContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to update contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req UpdateContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	var previousAssignedUserID *uuid.UUID
	if contact.AssignedUserID != nil {
		prev := *contact.AssignedUserID
		previousAssignedUserID = &prev
	}

	// Build updates map
	updates := map[string]any{}

	if req.ProfileName != nil {
		updates["profile_name"] = *req.ProfileName
	}
	if req.WhatsAppAccount != nil {
		updates["whats_app_account"] = *req.WhatsAppAccount
	}
	if req.InstanceID != nil {
		instanceID, resolveErr := a.resolveContactInstanceID(orgID, *req.InstanceID)
		if resolveErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		updates["instance_id"] = instanceID
	}
	if req.Tags != nil {
		tagsArray := make(models.JSONBArray, len(req.Tags))
		for i, tag := range req.Tags {
			tagsArray[i] = tag
		}
		updates["tags"] = tagsArray
	}
	if req.Metadata != nil {
		updates["metadata"] = models.JSONB(*req.Metadata)
	}
	if req.AssignedUserID != nil {
		// Verify user exists in same org
		var user models.User
		if err := a.DB.Where("id = ? AND organization_id = ?", req.AssignedUserID, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Assigned user not found", nil, "")
		}
		for key, value := range chatAssignmentUpdates(req.AssignedUserID) {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No fields to update", nil, "")
	}

	if err := a.DB.Model(contact).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact", nil, "")
	}

	// Reload contact
	a.DB.First(contact, contactID)

	if req.AssignedUserID != nil {
		notifyAssignee := false
		if contact.AssignedUserID != nil {
			notifyAssignee = previousAssignedUserID == nil || *previousAssignedUserID != *contact.AssignedUserID
		}
		a.broadcastContactLifecycleUpdate(orgID, contact, notifyAssignee)
	}

	return r.SendEnvelope(a.buildContactResponse(contact, orgID))
}

// DeleteContact soft-deletes a contact
func (a *App) DeleteContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Deleting chats is permission-gated by contacts:delete.
	if !a.canDeleteAnyChat(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to delete chats", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// Soft delete the contact
	if err := a.DB.Delete(contact).Error; err != nil {
		a.Log.Error("Failed to delete contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete contact", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message": "Contact deleted successfully",
	})
}

// buildContactResponse creates a ContactResponse from a Contact model
func (a *App) buildContactResponse(contact *models.Contact, orgID uuid.UUID) ContactResponse {
	status := normalizeContactStatus(contact)
	conversationContext := a.resolveContactConversationContext(orgID, *contact)
	a.repairDirectContactPhoneFromConversation(contact, conversationContext.ConversationID)
	a.scheduleContactAvatarRefresh(contact)

	// Count unread messages
	var unreadCount int64
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		buildConversationScopeQuery(a.DB, orgID, conversationContext.ConversationID, contact.InstanceID).
			Model(&models.Message{}).
			Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)
	} else {
		a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?", contact.ID, models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)
	}

	tags := []string{}
	if contact.Tags != nil {
		for _, t := range contact.Tags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	phoneNumber := contact.PhoneNumber
	profileName := contact.ProfileName
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		phoneNumber = conversationContext.ConversationID
	}
	if conversationContext.DisplayName != "" {
		profileName = conversationContext.DisplayName
	}
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	if shouldMask {
		phoneNumber = MaskPhoneNumber(phoneNumber)
		profileName = MaskIfPhoneNumber(profileName)
	}

	closedAt := contact.ClosedAt
	closedByUserID := contact.ClosedByUserID
	if status == models.ChatStatusClosed {
		if closedAt == nil {
			closedAt = &contact.UpdatedAt
		}
		if closedByUserID == nil && contact.AssignedUserID != nil {
			closedByUserID = contact.AssignedUserID
		}
	}
	serviceWindowOpen := contact.LastInboundAt != nil && time.Since(*contact.LastInboundAt) < 24*time.Hour

	return ContactResponse{
		ID:                 contact.ID,
		InstanceID:         stringifyInstanceID(contact.InstanceID),
		ConversationID:     conversationContext.ConversationID,
		IsGroupChat:        conversationContext.IsGroupChat,
		PhoneNumber:        phoneNumber,
		Name:               profileName,
		ProfileName:        profileName,
		AvatarURL:          contactAvatarURL(contact.Metadata),
		Status:             status.String(),
		Tags:               tags,
		Metadata:           contact.Metadata,
		LastMessageAt:      contact.LastMessageAt,
		LastMessagePreview: contact.LastMessagePreview,
		UnreadCount:        int(unreadCount),
		AssignedUserID:     contact.AssignedUserID,
		ClosedAt:           closedAt,
		ClosedByUserID:     closedByUserID,
		ClosedByName:       strings.TrimSpace(userFullName(contact.ClosedByUser)),
		WhatsAppAccount:    contact.WhatsAppAccount,
		LastInboundAt:      contact.LastInboundAt,
		ServiceWindowOpen:  serviceWindowOpen,
		CreatedAt:          contact.CreatedAt,
		UpdatedAt:          contact.UpdatedAt,
	}
}

func (a *App) broadcastContactLifecycleUpdate(orgID uuid.UUID, contact *models.Contact, notifyAssignee bool) {
	if a.WSHub == nil || contact == nil {
		return
	}

	status := normalizeContactStatus(contact)
	assignedUserID := ""
	if contact.AssignedUserID != nil {
		assignedUserID = contact.AssignedUserID.String()
	}

	profileName := contact.ProfileName
	if a.ShouldMaskPhoneNumbers(orgID) {
		profileName = MaskIfPhoneNumber(profileName)
	}

	payload := map[string]any{
		"id":               contact.ID.String(),
		"assigned_user_id": assignedUserID,
		"status":           status.String(),
		"profile_name":     profileName,
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeContactUpdate,
		Payload: payload,
	})

	if notifyAssignee && contact.AssignedUserID != nil {
		notifyPayload := map[string]any{
			"id":                contact.ID.String(),
			"assigned_user_id":  assignedUserID,
			"status":            status.String(),
			"profile_name":      profileName,
			"notify_assignment": true,
		}
		a.WSHub.BroadcastToUser(orgID, *contact.AssignedUserID, websocket.WSMessage{
			Type:    websocket.TypeContactUpdate,
			Payload: notifyPayload,
		})
	}
}
