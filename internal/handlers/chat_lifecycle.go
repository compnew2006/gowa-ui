package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contactDateBasis string

const (
	contactDateBasisCreated     contactDateBasis = "created"
	contactDateBasisIncomingAny contactDateBasis = "incoming_any"
)

func normalizeContactStatus(contact *models.Contact) models.ChatStatus {
	if contact == nil {
		return models.ChatStatusPending
	}
	status := contact.EffectiveStatus()
	contact.Status = status
	return status
}

func parseChatStatusFilter(raw string) (*models.ChatStatus, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	switch strings.ToLower(trimmed) {
	case string(models.ChatStatusPending):
		status := models.ChatStatusPending
		return &status, nil
	case string(models.ChatStatusOpen):
		status := models.ChatStatusOpen
		return &status, nil
	case string(models.ChatStatusClosed):
		status := models.ChatStatusClosed
		return &status, nil
	default:
		return nil, fmt.Errorf("invalid status filter")
	}
}

func applyChatStatusFilter(query *gorm.DB, status models.ChatStatus) *gorm.DB {
	switch status {
	case models.ChatStatusClosed:
		return query.Where("status = ?", models.ChatStatusClosed)
	case models.ChatStatusOpen:
		return query.Where(
			"(status = ? OR ((status IS NULL OR status = '' OR status = ?) AND assigned_user_id IS NOT NULL))",
			models.ChatStatusOpen,
			models.ChatStatusPending,
		)
	default: // pending
		return query.Where(
			"(status IS NULL OR status = '' OR status = ?) AND assigned_user_id IS NULL",
			models.ChatStatusPending,
		)
	}
}

func applyDefaultActiveChatFilter(query *gorm.DB) *gorm.DB {
	return query.Where("(status IS NULL OR status = '' OR status <> ?)", models.ChatStatusClosed)
}

func parseAssignedToFilter(raw string, currentUserID uuid.UUID) (*uuid.UUID, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false, nil
	}
	if strings.EqualFold(trimmed, "me") {
		id := currentUserID
		return &id, true, nil
	}
	if strings.EqualFold(trimmed, "unassigned") {
		return nil, true, nil
	}

	assignedUserID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, false, fmt.Errorf("invalid assigned_to filter")
	}
	return &assignedUserID, true, nil
}

func parseContactDateBasis(raw string) (contactDateBasis, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return contactDateBasisCreated, nil
	}

	switch strings.ToLower(trimmed) {
	case string(contactDateBasisCreated):
		return contactDateBasisCreated, nil
	case string(contactDateBasisIncomingAny):
		return contactDateBasisIncomingAny, nil
	default:
		return "", fmt.Errorf("invalid date_basis filter")
	}
}

func applyContactDateBasisFilter(
	query *gorm.DB,
	orgID uuid.UUID,
	basis contactDateBasis,
	dateFrom *time.Time,
	dateTo *time.Time,
	instanceID *uuid.UUID,
) *gorm.DB {
	switch basis {
	case contactDateBasisIncomingAny:
		subquery := query.Session(&gorm.Session{NewDB: true}).
			Model(&models.Message{}).
			Select("1").
			Where("messages.organization_id = ?", orgID).
			Where("messages.contact_id = contacts.id").
			Where("messages.direction = ?", models.DirectionIncoming)
		if instanceID != nil {
			subquery = subquery.Where("messages.instance_id = ?", *instanceID)
		}
		if dateFrom != nil {
			subquery = subquery.Where("messages.created_at >= ?", *dateFrom)
		}
		if dateTo != nil {
			subquery = subquery.Where("messages.created_at <= ?", endOfDay(*dateTo))
		}
		return query.Where("EXISTS (?)", subquery)
	default:
		if dateFrom != nil {
			query = query.Where("contacts.created_at >= ?", *dateFrom)
		}
		if dateTo != nil {
			query = query.Where("contacts.created_at <= ?", endOfDay(*dateTo))
		}
		return query
	}
}

func (a *App) canBypassPendingChatRestriction(userID, orgID uuid.UUID) bool {
	var user models.User
	if err := a.DB.Select("is_super_admin").Where("id = ?", userID).First(&user).Error; err == nil && user.IsSuperAdmin {
		return true
	}

	perms, err := a.getUserPermissionsCached(userID, orgID)
	if err != nil {
		return false
	}
	return strings.EqualFold(perms.RoleName, "admin")
}

func (a *App) canReadAllContacts(userID, orgID uuid.UUID) bool {
	return a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) ||
		a.canBypassPendingChatRestriction(userID, orgID)
}

func (a *App) shouldRestrictChatVisibilityToAgentScope(userID, orgID uuid.UUID) bool {
	var user models.User
	if err := a.DB.Select("is_super_admin").Where("id = ?", userID).First(&user).Error; err == nil && user.IsSuperAdmin {
		return false
	}

	perms, err := a.getUserPermissionsCached(userID, orgID)
	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(perms.RoleName), "agent")
}

func applyAgentVisibleChatAccessFilter(query *gorm.DB, userID uuid.UUID) *gorm.DB {
	return query.Where(
		"(is_public = ? OR assigned_user_id = ? OR EXISTS (SELECT 1 FROM contact_collaborators cc WHERE cc.contact_id = contacts.id AND cc.user_id = ? AND cc.status IN ? AND cc.deleted_at IS NULL) OR ((status IS NULL OR status = '' OR status = ?) AND assigned_user_id IS NULL))",
		true,
		userID,
		userID,
		collaboratorAccessStatuses(),
		models.ChatStatusPending,
	)
}

func applyAgentVisibleChatListFilter(query *gorm.DB, userID uuid.UUID) *gorm.DB {
	return applyAgentVisibleChatAccessFilter(query, userID)
}

func (a *App) canAccessRestrictedChatWithoutClaim(contact models.Contact, userID, orgID uuid.UUID) bool {
	if contact.IsPublic {
		return true
	}
	if a.isContactCollaborator(orgID, contact.ID, userID) {
		return true
	}
	return a.canViewRestrictedChatWithoutClaim(userID, orgID)
}

func (a *App) canSendRestrictedChatWithoutClaimForContact(contact models.Contact, userID, orgID uuid.UUID) bool {
	if contact.IsPublic {
		return true
	}
	if a.isContactCollaborator(orgID, contact.ID, userID) {
		return true
	}
	return a.canSendRestrictedChatWithoutClaim(userID, orgID)
}

func isChatRestrictedForMessageRead(contact models.Contact) bool {
	status := contact.EffectiveStatus()
	return status == models.ChatStatusPending || contact.AssignedUserID == nil
}

func chatAssignmentUpdates(assignee *uuid.UUID) map[string]any {
	if assignee != nil {
		return map[string]any{
			"assigned_user_id":  assignee,
			"status":            models.ChatStatusOpen,
			"closed_at":         nil,
			"closed_by_user_id": nil,
		}
	}

	return map[string]any{
		"assigned_user_id":  nil,
		"status":            models.ChatStatusPending,
		"closed_at":         nil,
		"closed_by_user_id": nil,
	}
}

func closeChatUpdates(closedByUserID uuid.UUID, currentAssignee *uuid.UUID) map[string]any {
	closedAt := time.Now().UTC()
	assignee := currentAssignee
	if assignee == nil {
		assignee = &closedByUserID
	}

	return map[string]any{
		"status":            models.ChatStatusClosed,
		"assigned_user_id":  assignee,
		"closed_at":         &closedAt,
		"closed_by_user_id": &closedByUserID,
	}
}

func closeChatUpdatesForSoftDelete(closedByUserID uuid.UUID, closedAt time.Time) map[string]any {
	return map[string]any{
		"status":            models.ChatStatusClosed,
		"assigned_user_id":  nil,
		"closed_at":         &closedAt,
		"closed_by_user_id": &closedByUserID,
	}
}

func reopenChatUpdates() map[string]any {
	return chatAssignmentUpdates(nil)
}

// reopenClosedChatToPending unassigns and reopens closed chats back to pending queue.
func (a *App) reopenClosedChatToPending(contact *models.Contact) (bool, error) {
	if contact == nil {
		return false, nil
	}
	if normalizeContactStatus(contact) != models.ChatStatusClosed {
		return false, nil
	}

	if err := a.DB.Model(contact).Updates(reopenChatUpdates()).Error; err != nil {
		return false, err
	}

	contact.Status = models.ChatStatusPending
	contact.AssignedUserID = nil
	contact.ClosedAt = nil
	contact.ClosedByUserID = nil
	contact.ClosedByUser = nil
	return true, nil
}

func userFullName(user *models.User) string {
	if user == nil {
		return ""
	}
	return strings.TrimSpace(user.FullName)
}
