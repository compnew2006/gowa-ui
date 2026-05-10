package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// AssignContactRequest represents the request to assign a contact to a user
type AssignContactRequest struct {
	UserID *uuid.UUID `json:"user_id"` // nil to unassign
}

func (a *App) appendClaimedChatSystemMessage(contact *models.Contact, userID uuid.UUID) {
	if a == nil || contact == nil {
		return
	}

	claimerName := strings.TrimSpace(a.ResolveUserDisplayName(userID))
	if claimerName == "" {
		claimerName = "An agent"
	}

	a.appendSystemChatMessage(contact, fmt.Sprintf("System: %s claimed this chat.", claimerName), models.JSONB{
		"event_type":           "chat_claimed",
		"claimed_by_user_id":   userID.String(),
		"claimed_by_user_name": claimerName,
	})
}

func (a *App) appendClosedChatSystemMessage(contact *models.Contact, userID uuid.UUID) {
	if a == nil || contact == nil {
		return
	}

	closerName := strings.TrimSpace(a.ResolveUserDisplayName(userID))
	if closerName == "" {
		closerName = "An agent"
	}

	a.appendSystemChatMessage(contact, fmt.Sprintf("System: %s closed this chat.", closerName), models.JSONB{
		"event_type":          "chat_closed",
		"closed_by_user_id":   userID.String(),
		"closed_by_user_name": closerName,
	})
}

func (a *App) appendPublicChatSystemMessage(contact *models.Contact, userID uuid.UUID, isPublic bool) {
	if a == nil || contact == nil {
		return
	}

	actorName := strings.TrimSpace(a.ResolveUserDisplayName(userID))
	if actorName == "" {
		actorName = "An agent"
	}

	if isPublic {
		a.appendSystemChatMessage(contact, fmt.Sprintf("System: %s made this chat public for all agents.", actorName), models.JSONB{
			"event_type":          "chat_public_enabled",
			"public_by_user_id":   userID.String(),
			"public_by_user_name": actorName,
		})
		return
	}

	a.appendSystemChatMessage(contact, fmt.Sprintf("System: %s removed public visibility from this chat.", actorName), models.JSONB{
		"event_type":          "chat_public_disabled",
		"public_by_user_id":   userID.String(),
		"public_by_user_name": actorName,
	})
}

func (a *App) canAssignContacts(userID, orgID uuid.UUID) bool {
	if a == nil {
		return false
	}
	return a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) ||
		a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
}

func (a *App) canUserSeeContactInstance(orgID, userID uuid.UUID, contact *models.Contact) (bool, error) {
	if a == nil || contact == nil || contact.InstanceID == nil || *contact.InstanceID == uuid.Nil {
		return true, nil
	}

	allowedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		return false, err
	}
	if allowedInstanceIDs == nil {
		return true, nil
	}

	return containsRestrictedUUID(allowedInstanceIDs, *contact.InstanceID), nil
}

func findAssignableOrgUser(db *gorm.DB, userID, orgID uuid.UUID) (*models.User, error) {
	var user models.User
	err := db.Session(&gorm.Session{}).
		Select("users.*").
		Joins("LEFT JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", userID).
		Where("(user_organizations.user_id IS NOT NULL OR users.organization_id = ?)", orgID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (a *App) buildLifecycleContactQuery(
	requestDB *gorm.DB,
	orgID, userID, contactID uuid.UUID,
) (*gorm.DB, error) {
	query := requestDB.Session(&gorm.Session{}).
		Model(&models.Contact{}).
		Where("contacts.id = ? AND contacts.organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}

	restrictedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		return nil, err
	}

	return applyRestrictedInstanceVisibilityFilterWithAssignedBypass(query, restrictedInstanceIDs, userID), nil
}

func (a *App) canEmitChatAssignmentSystemMessage(userID, orgID uuid.UUID) bool {
	if a == nil {
		return false
	}
	if a.canBypassPendingChatRestriction(userID, orgID) {
		return true
	}
	return a.canAssignContacts(userID, orgID)
}

func (a *App) appendAssignedChatSystemMessage(contact *models.Contact, actorUserID uuid.UUID, assigneeUserID *uuid.UUID) {
	if a == nil || contact == nil || assigneeUserID == nil || *assigneeUserID == uuid.Nil {
		return
	}
	if !a.canEmitChatAssignmentSystemMessage(actorUserID, contact.OrganizationID) {
		return
	}

	actorName := strings.TrimSpace(a.ResolveUserDisplayName(actorUserID))
	if actorName == "" {
		actorName = "An employee"
	}
	assigneeName := strings.TrimSpace(a.ResolveUserDisplayName(*assigneeUserID))
	if assigneeName == "" {
		assigneeName = "an agent"
	}

	a.appendSystemChatMessage(
		contact,
		fmt.Sprintf("System :%s has assigned this chat to %s", actorName, assigneeName),
		models.JSONB{
			"event_type":            "chat_assigned",
			"assigned_by_user_id":   actorUserID.String(),
			"assigned_by_user_name": actorName,
			"assigned_to_user_id":   assigneeUserID.String(),
			"assigned_to_user_name": assigneeName,
		},
	)
}

// AssignContact assigns a contact to a user (agent)
// Only users with assignment permission can assign contacts
func (a *App) AssignContact(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Only users with assignment permission can assign contacts
	if !a.canAssignContacts(userID, orgID) {
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
	contact, err := findByIDAndOrg[models.Contact](requestDB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// If assigning to a user, verify they are available in the same org.
	if req.UserID != nil {
		if _, err := findAssignableOrgUser(a.DB, *req.UserID, orgID); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
		}
	}

	var previousAssignedUserID *uuid.UUID
	if contact.AssignedUserID != nil {
		prev := *contact.AssignedUserID
		previousAssignedUserID = &prev
	}

	// Update contact assignment + lifecycle status
	if err := a.DB.Model(&models.Contact{}).Where("id = ? AND organization_id = ?", contact.ID, orgID).Updates(chatAssignmentUpdates(req.UserID)).Error; err != nil {
		a.Log.Error("Failed to assign contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	if err := a.DB.Where("id = ? AND organization_id = ?", contact.ID, orgID).First(contact).Error; err != nil {
		a.Log.Error("Failed to reload contact after assignment", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	notifyAssignee := false
	if contact.AssignedUserID != nil {
		notifyAssignee = previousAssignedUserID == nil || *previousAssignedUserID != *contact.AssignedUserID
	}
	if notifyAssignee {
		a.appendAssignedChatSystemMessage(contact, userID, contact.AssignedUserID)
	}
	a.broadcastContactLifecycleUpdate(orgID, contact, notifyAssignee)

	return r.SendEnvelope(map[string]any{
		"message":          "Contact assigned successfully",
		"assigned_user_id": req.UserID,
	})
}

// ClaimChat claims a pending chat for the current user.
func (a *App) ClaimChat(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
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
	query, scopeErr := a.buildLifecycleContactQuery(requestDB, orgID, userID, contactID)
	if scopeErr != nil {
		a.Log.Error("Failed to resolve restricted instance for claim", "error", scopeErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to claim chat", nil, "")
	}
	if err := query.First(&contact).Error; err != nil {
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
		_ = requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error
		a.appendClaimedChatSystemMessage(&contact, userID)
		return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
	}

	if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Updates(chatAssignmentUpdates(&userID)).Error; err != nil {
		a.Log.Error("Failed to claim chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to claim chat", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}

	a.appendClaimedChatSystemMessage(&contact, userID)
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
}

// CloseChat marks a chat as closed.
func (a *App) CloseChat(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
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
	query, scopeErr := a.buildLifecycleContactQuery(requestDB, orgID, userID, contactID)
	if scopeErr != nil {
		a.Log.Error("Failed to resolve restricted instance for close", "error", scopeErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		_ = requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error
		return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
	}

	if contact.AssignedUserID != nil && *contact.AssignedUserID != userID && !a.canBypassPendingChatRestriction(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only the assigned user can close this chat", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Updates(closeChatUpdates(userID, contact.AssignedUserID)).Error; err != nil {
		a.Log.Error("Failed to close chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}

	a.appendClosedChatSystemMessage(&contact, userID)
	a.handleManualChatCloseRatingPrompt(orgID, userID, &contact)
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
}

// ReopenChat reopens a closed chat and moves it back to pending unassigned queue.
func (a *App) ReopenChat(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
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
	query, scopeErr := a.buildLifecycleContactQuery(requestDB, orgID, userID, contactID)
	if scopeErr != nil {
		a.Log.Error("Failed to resolve restricted instance for reopen", "error", scopeErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reopen chat", nil, "")
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	if normalizeContactStatus(&contact) != models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Only closed chats can be reopened", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Updates(reopenChatUpdates()).Error; err != nil {
		a.Log.Error("Failed to reopen chat", "error", err, "chat_id", contactID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reopen chat", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
}

type SetChatPublicRequest struct {
	IsPublic bool `json:"is_public"`
}

// SetChatPublic toggles public visibility for a chat so all agents can access it.
func (a *App) SetChatPublic(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) &&
		!a.HasPermission(userID, models.ResourceChat, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to change chat visibility", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "chat")
	if err != nil {
		return nil
	}

	var req SetChatPublicRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	var contact models.Contact
	query, scopeErr := a.buildLifecycleContactQuery(requestDB, orgID, userID, contactID)
	if scopeErr != nil {
		a.Log.Error("Failed to resolve restricted instance for visibility update", "error", scopeErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update chat visibility", nil, "")
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Chat not found", nil, "")
	}

	if contact.IsPublic == req.IsPublic {
		_ = requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error
		return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
	}

	if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Update("is_public", req.IsPublic).Error; err != nil {
		a.Log.Error("Failed to update chat public visibility", "error", err, "chat_id", contactID, "user_id", userID, "is_public", req.IsPublic)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update chat visibility", nil, "")
	}

	if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load updated chat", nil, "")
	}

	a.appendPublicChatSystemMessage(&contact, userID, req.IsPublic)
	a.broadcastContactLifecycleUpdate(orgID, &contact, false)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
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
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	var contact models.Contact
	query := requestDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
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
	err = requestDB.Where("contact_id = ? AND organization_id = ?", contactID, orgID).
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
	requestDB := a.requestDB(r)
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
	contact, err := findByIDAndOrg[models.Contact](requestDB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// Convert tags to JSONBArray
	tagsArray := make(models.JSONBArray, len(req.Tags))
	for i, tag := range req.Tags {
		tagsArray[i] = tag
	}

	// Update contact tags
	if err := requestDB.Model(contact).Update("tags", tagsArray).Error; err != nil {
		a.Log.Error("Failed to update contact tags", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact tags", nil, "")
	}

	// Reload contact to get updated tags
	if err := requestDB.First(contact, contactID).Error; err != nil {
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
	StartChat       bool           `json:"start_chat,omitempty"`
	Tags            []string       `json:"tags"`
	Metadata        map[string]any `json:"metadata"`
}

// CreateContact creates a new contact or restores a soft-deleted one
func (a *App) CreateContact(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
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

	startChat := req.StartChat && a.isWhatsmeowProvider()

	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.ProfileName = strings.TrimSpace(req.ProfileName)
	req.WhatsAppAccount = strings.TrimSpace(req.WhatsAppAccount)

	if req.PhoneNumber == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_number is required", nil, "")
	}

	resolvedInstanceID, err := a.resolveContactInstanceID(orgID, req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "instance_id")
	}

	if startChat {
		instance, resolveErr := a.resolveOutboundInstance(orgID, req.InstanceID, resolvedInstanceID)
		if resolveErr != nil {
			if _, reasonCode, ok := asInstanceSelectionError(resolveErr); ok {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), reasonCodeDetails(reasonCode), "instance_id")
			}
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}

		resolvedInstanceID = &instance.ID

		lookupCtx, cancel := context.WithTimeout(r.RequestCtx, 8*time.Second)
		defer cancel()

		resolvedContact, lookupErr := a.resolveWhatsmeowContactResolver().ResolveDirectContact(lookupCtx, instance, req.PhoneNumber)
		switch {
		case lookupErr == nil:
			req.PhoneNumber = strings.TrimSpace(resolvedContact.CanonicalPhone)
			if req.ProfileName == "" {
				req.ProfileName = strings.TrimSpace(resolvedContact.ProfileName)
			}
			if req.WhatsAppAccount == "" {
				req.WhatsAppAccount = strings.TrimSpace(instance.PhoneNumber)
			}
		case errors.Is(lookupErr, errWhatsmeowDirectChatInvalidPhone), errors.Is(lookupErr, errWhatsmeowDirectChatNotFound):
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, lookupErr.Error(), nil, "phone_number")
		case errors.Is(lookupErr, errWhatsmeowDirectChatUnavailable):
			return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "The selected WhatsApp instance is not available for starting chats", nil, "instance_id")
		default:
			a.Log.Error("Failed to resolve WhatsMeow direct chat recipient", "error", lookupErr, "org_id", orgID, "instance_id", instance.ID, "phone_number", req.PhoneNumber)
			return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to verify phone number on WhatsApp", nil, "phone_number")
		}
	}

	// Normalize phone number
	normalizedPhone := req.PhoneNumber
	if len(normalizedPhone) > 0 && normalizedPhone[0] == '+' {
		normalizedPhone = normalizedPhone[1:]
	}

	// Check if contact exists (including soft-deleted)
	var existingContact models.Contact
	existingQuery := requestDB.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone)
	if resolvedInstanceID != nil {
		existingQuery = existingQuery.Where("instance_id = ?", *resolvedInstanceID)
	} else {
		existingQuery = existingQuery.Where("instance_id IS NULL")
	}
	if err := existingQuery.First(&existingContact).Error; err == nil {
		// Contact exists
		if existingContact.DeletedAt.Valid {
			requestDB.
				// Restore soft-deleted contact
				Unscoped().Model(&existingContact).Update("deleted_at", nil)
			existingContact.DeletedAt.Valid = false
			// Update fields
			updates := map[string]any{}
			if req.ProfileName != "" {
				updates["profile_name"] = req.ProfileName
			}
			if req.WhatsAppAccount != "" {
				updates["whats_app_account"] = req.WhatsAppAccount
			}
			if resolvedInstanceID != nil {
				updates["instance_id"] = resolvedInstanceID
			}
			if startChat {
				for key, value := range chatAssignmentUpdates(&userID) {
					updates[key] = value
				}
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
				requestDB.
					Model(&existingContact).Updates(updates)
			}
			requestDB.
				// Reload contact
				First(&existingContact, existingContact.ID)
			if startChat {
				a.broadcastContactLifecycleUpdate(orgID, &existingContact, false)
			}
			return r.SendEnvelope(a.buildContactResponse(&existingContact, orgID, userID))
		}
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Contact with this phone number already exists", nil, "")
	}

	status := models.ChatStatusPending
	var assignedUserID *uuid.UUID
	if startChat {
		status = models.ChatStatusOpen
		assignedUserID = &userID
	}

	// Create new contact
	contact := models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		InstanceID:      resolvedInstanceID,
		PhoneNumber:     normalizedPhone,
		ProfileName:     req.ProfileName,
		WhatsAppAccount: req.WhatsAppAccount,
		Status:          status,
		AssignedUserID:  assignedUserID,
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

	if err := requestDB.Create(&contact).Error; err != nil {
		a.Log.Error("Failed to create contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create contact", nil, "")
	}

	if startChat {
		a.broadcastContactLifecycleUpdate(orgID, &contact, false)
	}

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
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
	requestDB := a.requestDB(r)
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
	contact, err := findByIDAndOrg[models.Contact](requestDB, r, contactID, orgID, "Contact")
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
		if err := requestDB.Where("id = ? AND organization_id = ?", req.AssignedUserID, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Assigned user not found", nil, "")
		}
		for key, value := range chatAssignmentUpdates(req.AssignedUserID) {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No fields to update", nil, "")
	}

	if err := requestDB.Model(contact).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact", nil, "")
	}
	requestDB.

		// Reload contact
		First(contact, contactID)

	if req.AssignedUserID != nil {
		notifyAssignee := false
		if contact.AssignedUserID != nil {
			notifyAssignee = previousAssignedUserID == nil || *previousAssignedUserID != *contact.AssignedUserID
		}
		a.broadcastContactLifecycleUpdate(orgID, contact, notifyAssignee)
	}

	return r.SendEnvelope(a.buildContactResponse(contact, orgID, userID))
}

// DeleteContact soft-deletes a contact while preserving conversation history.
func (a *App) DeleteContact(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
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
	contact, err := findByIDAndOrg[models.Contact](requestDB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	if err := requestDB.Delete(contact).Error; err != nil {
		a.Log.Error("Failed to delete contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete contact", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message": "Contact deleted successfully",
	})
}

// SoftDeleteContactForUser hides a chat for the current user without deleting data.
func (a *App) SoftDeleteContactForUser(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceContacts, models.ActionSoftDelete, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to hide chats", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	contact, err := findByIDAndOrg[models.Contact](requestDB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	if normalizeContactStatus(contact) != models.ChatStatusClosed {
		closedAt := time.Now().UTC()
		if err := requestDB.Model(contact).Updates(closeChatUpdatesForSoftDelete(userID, closedAt)).Error; err != nil {
			a.Log.Error("Failed to close chat on soft delete", "error", err, "contact_id", contact.ID, "user_id", userID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to close chat", nil, "")
		}
		contact.Status = models.ChatStatusClosed
		contact.AssignedUserID = nil
		contact.ClosedAt = &closedAt
		contact.ClosedByUserID = &userID
		a.appendClosedChatSystemMessage(contact, userID)
		a.handleManualChatCloseRatingPrompt(orgID, userID, contact)
		a.broadcastContactLifecycleUpdate(orgID, contact, false)
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 5*time.Second)
	defer cancel()

	deletedAt := time.Now().UTC()
	if err := a.upsertContactUserDeletion(ctx, orgID, contact.ID, userID, deletedAt); err != nil {
		a.Log.Error("Failed to soft delete chat", "error", err, "contact_id", contact.ID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to hide chat", nil, "")
	}

	a.notifyChatDeletedByUser(ctx, orgID, userID, contact)

	return r.SendEnvelope(map[string]any{
		"message":    "Chat hidden successfully",
		"deleted_at": deletedAt,
	})
}

func (a *App) notifyChatDeletedByUser(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) {
	if contact == nil {
		return
	}

	instanceID := contact.InstanceID
	if instanceID == nil {
		var instance models.WhatsAppInstance
		if err := a.DB.WithContext(ctx).
			Where("organization_id = ?", orgID).
			Order("is_default DESC, created_at ASC").
			First(&instance).Error; err == nil {
			instanceID = &instance.ID
		}
	}
	if instanceID == nil {
		a.Log.Warn("Skipping chat deleted notification; no instance available", "contact_id", contact.ID, "org_id", orgID)
		return
	}

	actorName := strings.TrimSpace(a.ResolveUserDisplayName(userID))
	if actorName == "" {
		actorName = "A user"
	}

	conversationContext := a.resolveContactConversationContext(ctx, orgID, *contact)
	contactName := strings.TrimSpace(contact.ProfileName)
	contactPhone := strings.TrimSpace(contact.PhoneNumber)
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		contactPhone = conversationContext.ConversationID
	}
	if conversationContext.DisplayName != "" {
		contactName = conversationContext.DisplayName
	}
	contactLabel := contactName
	if contactLabel == "" {
		contactLabel = contactPhone
	}
	if contactName != "" && contactPhone != "" && contactName != contactPhone {
		contactLabel = fmt.Sprintf("%s (%s)", contactName, contactPhone)
	}
	if contactLabel == "" {
		contactLabel = "a chat"
	}

	metadata := models.JSONB{
		"actor_id":      userID.String(),
		"actor_name":    actorName,
		"contact_id":    contact.ID.String(),
		"contact_name":  contactName,
		"contact_phone": contactPhone,
	}

	notification := &models.InstanceNotification{
		OrganizationID: orgID,
		InstanceID:     *instanceID,
		EventType:      "chat_deleted_by_user",
		Message:        fmt.Sprintf("%s deleted chat %s", actorName, contactLabel),
		IsDismissed:    false,
		ContactID:      &contact.ID,
		Metadata:       metadata,
	}

	if err := a.DB.WithContext(ctx).Create(notification).Error; err != nil {
		a.Log.Error("Failed to create chat deleted notification", "error", err, "contact_id", contact.ID)
		return
	}

	if a.WSHub == nil {
		return
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeInstanceNotification,
		Payload: websocket.InstanceNotificationPayload{
			ID:         notification.ID.String(),
			InstanceID: notification.InstanceID.String(),
			EventType:  notification.EventType,
			Message:    notification.Message,
			CreatedAt:  notification.CreatedAt.Format(time.RFC3339),
			ContactID:  contact.ID.String(),
			Metadata:   map[string]any(notification.Metadata),
		},
	})
}

// buildContactResponse creates a ContactResponse from a Contact model
func (a *App) buildContactResponse(contact *models.Contact, orgID, userID uuid.UUID) ContactResponse {
	status := normalizeContactStatus(contact)
	conversationContext := a.resolveContactConversationContext(context.Background(), orgID, *contact)
	a.repairDirectContactPhoneFromConversation(contact, conversationContext.ConversationID)
	a.scheduleContactAvatarRefresh(contact)

	// Count unread messages
	var unreadCount int64
	deletedAt, _ := a.getContactUserDeletionTimestamp(context.Background(), orgID, contact.ID, userID)
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		msgQuery := buildConversationScopeQuery(a.DB, orgID, conversationContext.ConversationID, contact.InstanceID).
			Model(&models.Message{}).
			Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead)
		if deletedAt != nil {
			msgQuery = msgQuery.Where("created_at > ?", *deletedAt)
		}
		msgQuery.Count(&unreadCount)
	} else {
		msgQuery := a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?", contact.ID, models.DirectionIncoming, models.MessageStatusRead)
		if deletedAt != nil {
			msgQuery = msgQuery.Where("created_at > ?", *deletedAt)
		}
		msgQuery.Count(&unreadCount)
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
	assignedUserName := a.resolveAssignedUserName(contact, orgID)
	isCollaborator := a.isContactCollaborator(orgID, contact.ID, userID)

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
		AssignedUserName:   assignedUserName,
		IsPublic:           contact.IsPublic,
		IsCollaborator:     isCollaborator,
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

func (a *App) resolveAssignedUserName(contact *models.Contact, orgID uuid.UUID) string {
	if contact == nil || contact.AssignedUserID == nil {
		return ""
	}

	if name := strings.TrimSpace(userFullName(contact.AssignedUser)); name != "" {
		return name
	}

	var assignedUser models.User
	if err := a.DB.Select("full_name").
		Where("id = ? AND organization_id = ?", *contact.AssignedUserID, orgID).
		First(&assignedUser).Error; err != nil {
		return ""
	}

	return strings.TrimSpace(assignedUser.FullName)
}

func (a *App) broadcastContactLifecycleUpdate(orgID uuid.UUID, contact *models.Contact, notifyAssignee bool) {
	if a.WSHub == nil || contact == nil {
		return
	}

	status := normalizeContactStatus(contact)
	assignedUserName := a.resolveAssignedUserName(contact, orgID)
	assignedUserID := ""
	if contact.AssignedUserID != nil {
		assignedUserID = contact.AssignedUserID.String()
	}

	profileName := contact.ProfileName
	if a.ShouldMaskPhoneNumbers(orgID) {
		profileName = MaskIfPhoneNumber(profileName)
	}

	payload := map[string]any{
		"id":                 contact.ID.String(),
		"assigned_user_id":   assignedUserID,
		"assigned_user_name": assignedUserName,
		"is_public":          contact.IsPublic,
		"status":             status.String(),
		"profile_name":       profileName,
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeContactUpdate,
		Payload: payload,
	})

	if notifyAssignee && contact.AssignedUserID != nil {
		notifyPayload := map[string]any{
			"id":                 contact.ID.String(),
			"assigned_user_id":   assignedUserID,
			"assigned_user_name": assignedUserName,
			"is_public":          contact.IsPublic,
			"status":             status.String(),
			"profile_name":       profileName,
			"notify_assignment":  true,
		}
		a.WSHub.BroadcastToUser(orgID, *contact.AssignedUserID, websocket.WSMessage{
			Type:    websocket.TypeContactUpdate,
			Payload: notifyPayload,
		})
	}
}
