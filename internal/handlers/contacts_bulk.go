package handlers

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const bulkMaxContacts = 100

type BulkContactIDsRequest struct {
	ContactIDs []uuid.UUID `json:"contact_ids"`
}

type BulkAssignRequest struct {
	ContactIDs []uuid.UUID  `json:"contact_ids"`
	UserID     *uuid.UUID   `json:"user_id"`
}

type BulkContactResult struct {
	ContactID uuid.UUID `json:"contact_id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
}

type BulkResponse struct {
	Total   int                 `json:"total"`
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
	Results []BulkContactResult `json:"results"`
}

func (r BulkContactIDsRequest) Validate() error {
	if len(r.ContactIDs) == 0 {
		return fmt.Errorf("contact_ids is required")
	}
	if len(r.ContactIDs) > bulkMaxContacts {
		return fmt.Errorf("maximum %d contacts allowed per bulk operation", bulkMaxContacts)
	}
	seen := make(map[uuid.UUID]struct{}, len(r.ContactIDs))
	for _, id := range r.ContactIDs {
		if id == uuid.Nil {
			return fmt.Errorf("contact_ids contains a nil UUID")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("contact_ids contains duplicate: %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (r BulkAssignRequest) Validate() error {
	if len(r.ContactIDs) == 0 {
		return fmt.Errorf("contact_ids is required")
	}
	if len(r.ContactIDs) > bulkMaxContacts {
		return fmt.Errorf("maximum %d contacts allowed per bulk operation", bulkMaxContacts)
	}
	if r.UserID != nil && *r.UserID == uuid.Nil {
		return fmt.Errorf("user_id must be a valid UUID or null to unassign")
	}
	seen := make(map[uuid.UUID]struct{}, len(r.ContactIDs))
	for _, id := range r.ContactIDs {
		if id == uuid.Nil {
			return fmt.Errorf("contact_ids contains a nil UUID")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("contact_ids contains duplicate: %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (a *App) BulkCloseChats(r *fastglue.Request) error {
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

	var req BulkContactIDsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := req.Validate(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	isAdmin := a.canBypassPendingChatRestriction(userID, orgID)

	var contacts []models.Contact
	if err := requestDB.Session(&gorm.Session{}).
		Where("id IN ? AND organization_id = ?", req.ContactIDs, orgID).
		Find(&contacts).Error; err != nil {
		a.Log.Error("Bulk close: failed to fetch contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch contacts", nil, "")
	}

	contactMap := make(map[uuid.UUID]models.Contact, len(contacts))
	for _, c := range contacts {
		contactMap[c.ID] = c
	}

	restrictedInstanceIDs, riErr := a.getRestrictedInstancesForUser(orgID, userID)
	if riErr != nil {
		a.Log.Error("Bulk close: failed to resolve restricted instances", "error", riErr)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to resolve instance access", nil, "")
	}
	restrictInstances := len(restrictedInstanceIDs) > 0

	results := make([]BulkContactResult, 0, len(req.ContactIDs))
	successCount := 0
	failedCount := 0

	for _, contactID := range req.ContactIDs {
		contact, found := contactMap[contactID]
		if !found {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Contact not found"})
			failedCount++
			continue
		}

		if restrictInstances && contact.InstanceID != nil && *contact.InstanceID != uuid.Nil {
			if !containsRestrictedUUID(restrictedInstanceIDs, *contact.InstanceID) {
				results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "No access to this contact's instance"})
				failedCount++
				continue
			}
		}

		status := normalizeContactStatus(&contact)
		if status == models.ChatStatusClosed {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "success"})
			successCount++
			continue
		}

		if !isAdmin && contact.AssignedUserID != nil && *contact.AssignedUserID != userID {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Only the assigned user can close this chat"})
			failedCount++
			continue
		}

		if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
			Where("id = ?", contact.ID).
			Updates(closeChatUpdates(userID, contact.AssignedUserID)).Error; err != nil {
			a.Log.Error("Bulk close: failed to close chat", "error", err, "contact_id", contact.ID)
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Failed to close chat"})
			failedCount++
			continue
		}

		var updated models.Contact
		if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contact.ID).First(&updated).Error; err == nil {
			a.appendClosedChatSystemMessage(&updated, userID)
			a.handleManualChatCloseRatingPrompt(orgID, userID, &updated)
			a.broadcastContactLifecycleUpdate(orgID, &updated, false)
		}

		results = append(results, BulkContactResult{ContactID: contactID, Status: "success"})
		successCount++
	}

	return r.SendEnvelope(BulkResponse{
		Total:   len(req.ContactIDs),
		Success: successCount,
		Failed:  failedCount,
		Results: results,
	})
}

func (a *App) BulkAssignChats(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.canAssignContacts(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to assign contacts", nil, "")
	}

	var req BulkAssignRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := req.Validate(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	if req.UserID != nil && *req.UserID != uuid.Nil {
		if _, err := findAssignableOrgUser(a.DB, *req.UserID, orgID); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Assignee user not found", nil, "")
		}
	}

	var contacts []models.Contact
	if err := requestDB.Session(&gorm.Session{}).
		Where("id IN ? AND organization_id = ?", req.ContactIDs, orgID).
		Find(&contacts).Error; err != nil {
		a.Log.Error("Bulk assign: failed to fetch contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch contacts", nil, "")
	}

	contactMap := make(map[uuid.UUID]models.Contact, len(contacts))
	for _, c := range contacts {
		contactMap[c.ID] = c
	}

	restrictedInstanceIDs, riErr := a.getRestrictedInstancesForUser(orgID, userID)
	if riErr != nil {
		a.Log.Error("Bulk assign: failed to resolve restricted instances", "error", riErr)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to resolve instance access", nil, "")
	}
	restrictInstances := len(restrictedInstanceIDs) > 0

	results := make([]BulkContactResult, 0, len(req.ContactIDs))
	successCount := 0
	failedCount := 0

	for _, contactID := range req.ContactIDs {
		contact, found := contactMap[contactID]
		if !found {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Contact not found"})
			failedCount++
			continue
		}

		if restrictInstances && contact.InstanceID != nil && *contact.InstanceID != uuid.Nil {
			if !containsRestrictedUUID(restrictedInstanceIDs, *contact.InstanceID) {
				results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "No access to this contact's instance"})
				failedCount++
				continue
			}
		}

		if req.UserID != nil && *req.UserID != uuid.Nil {
			allowed, err := a.canUserSeeContactInstance(orgID, *req.UserID, &contact)
			if err != nil {
				results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Failed to validate assignee access"})
				failedCount++
				continue
			}
			if !allowed {
				results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Assignee does not have access to this WhatsApp account"})
				failedCount++
				continue
			}
		}

		var previousAssignedUserID *uuid.UUID
		if contact.AssignedUserID != nil {
			prev := *contact.AssignedUserID
			previousAssignedUserID = &prev
		}

		if err := requestDB.Model(&contact).Updates(chatAssignmentUpdates(req.UserID)).Error; err != nil {
			a.Log.Error("Bulk assign: failed to assign contact", "error", err, "contact_id", contact.ID)
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Failed to assign contact"})
			failedCount++
			continue
		}

		if err := requestDB.Where("id = ?", contact.ID).First(&contact).Error; err != nil {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "success"})
			successCount++
			continue
		}

		notifyAssignee := false
		if contact.AssignedUserID != nil {
			notifyAssignee = previousAssignedUserID == nil || *previousAssignedUserID != *contact.AssignedUserID
		}
		if notifyAssignee {
			a.appendAssignedChatSystemMessage(&contact, userID, contact.AssignedUserID)
		}
		a.broadcastContactLifecycleUpdate(orgID, &contact, notifyAssignee)

		results = append(results, BulkContactResult{ContactID: contactID, Status: "success"})
		successCount++
	}

	return r.SendEnvelope(BulkResponse{
		Total:   len(req.ContactIDs),
		Success: successCount,
		Failed:  failedCount,
		Results: results,
	})
}

func (a *App) BulkReopenChats(r *fastglue.Request) error {
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

	var req BulkContactIDsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := req.Validate(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	var contacts []models.Contact
	if err := requestDB.Session(&gorm.Session{}).
		Where("id IN ? AND organization_id = ?", req.ContactIDs, orgID).
		Find(&contacts).Error; err != nil {
		a.Log.Error("Bulk reopen: failed to fetch contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch contacts", nil, "")
	}

	contactMap := make(map[uuid.UUID]models.Contact, len(contacts))
	for _, c := range contacts {
		contactMap[c.ID] = c
	}

	restrictedInstanceIDs, riErr := a.getRestrictedInstancesForUser(orgID, userID)
	if riErr != nil {
		a.Log.Error("Bulk reopen: failed to resolve restricted instances", "error", riErr)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to resolve instance access", nil, "")
	}
	restrictInstances := len(restrictedInstanceIDs) > 0

	results := make([]BulkContactResult, 0, len(req.ContactIDs))
	successCount := 0
	failedCount := 0

	for _, contactID := range req.ContactIDs {
		contact, found := contactMap[contactID]
		if !found {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Contact not found"})
			failedCount++
			continue
		}

		if restrictInstances && contact.InstanceID != nil && *contact.InstanceID != uuid.Nil {
			if !containsRestrictedUUID(restrictedInstanceIDs, *contact.InstanceID) {
				results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "No access to this contact's instance"})
				failedCount++
				continue
			}
		}

		if normalizeContactStatus(&contact) != models.ChatStatusClosed {
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Only closed chats can be reopened"})
			failedCount++
			continue
		}

		if err := requestDB.Session(&gorm.Session{}).Model(&models.Contact{}).
			Where("id = ?", contact.ID).
			Updates(reopenChatUpdates()).Error; err != nil {
			a.Log.Error("Bulk reopen: failed to reopen chat", "error", err, "contact_id", contact.ID)
			results = append(results, BulkContactResult{ContactID: contactID, Status: "failed", Error: "Failed to reopen chat"})
			failedCount++
			continue
		}

		var updated models.Contact
		if err := requestDB.Session(&gorm.Session{}).Preload("ClosedByUser").Where("id = ?", contact.ID).First(&updated).Error; err == nil {
			a.broadcastContactLifecycleUpdate(orgID, &updated, false)
		}

		results = append(results, BulkContactResult{ContactID: contactID, Status: "success"})
		successCount++
	}

	return r.SendEnvelope(BulkResponse{
		Total:   len(req.ContactIDs),
		Success: successCount,
		Failed:  failedCount,
		Results: results,
	})
}
