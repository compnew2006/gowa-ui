package handlers

import (
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Agent-initiated transfer: the current assignee hands an open conversation
// to a colleague. Unlike AssignContact (admin/manager, contacts:write), this
// is gated by chat.assign:write and limited to the caller's own conversation
// and to colleagues who share one of the caller's WhatsApp accounts. Both
// paths go through ChatLifecycle.Assign, so the target always receives the
// permanent access grant and can open a conversation that sits under an
// account they are not assigned to.

// TransferTarget is one colleague an assignee may transfer a conversation to.
type TransferTarget struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	RoleName    string    `json:"role_name"`
	IsAvailable bool      `json:"is_available"`
}

// TransferChatRequest is the body of PUT /api/contacts/{id}/transfer.
type TransferChatRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// transferTargets returns the colleagues callerID may transfer a conversation
// to: active members of the org (other than the caller) who hold chat:read and
// share at least one WhatsApp account with the caller. A user with no account
// assignments sees every account (the scopeAccountsToUser fallback), so they
// always share one, and an unrestricted caller (or super admin) sees every
// active chat user. onlyID narrows the lookup to one candidate for the
// transfer-time eligibility check. Errors are returned so callers fail closed.
func (a *App) transferTargets(callerID, orgID uuid.UUID, onlyID *uuid.UUID) ([]TransferTarget, error) {
	query := a.DB.Table("users").
		Select("users.id, users.full_name, users.is_available, COALESCE(custom_roles.name, '') AS role_name").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Joins("LEFT JOIN custom_roles ON custom_roles.id = COALESCE(user_organizations.role_id, users.role_id)").
		Where("users.deleted_at IS NULL AND users.is_active = ? AND users.id <> ?", true, callerID)
	if onlyID != nil {
		query = query.Where("users.id = ?", *onlyID)
	}

	if !a.IsSuperAdmin(callerID) {
		callerAccounts, err := a.assignedAccountIDs(callerID, orgID)
		if err != nil {
			return nil, err
		}
		if len(callerAccounts) > 0 {
			// Users with account assignments in this org (org-scoped like
			// assignedAccountIDs). Built fresh per use: a shared *gorm.DB
			// would accumulate the first subquery's conditions.
			orgAssignees := func() *gorm.DB {
				return a.DB.Table("user_whatsapp_accounts").
					Select("user_whatsapp_accounts.user_id").
					Joins("JOIN whatsapp_accounts ON whatsapp_accounts.id = user_whatsapp_accounts.whats_app_account_id AND whatsapp_accounts.deleted_at IS NULL").
					Where("whatsapp_accounts.organization_id = ?", orgID)
			}
			query = query.Where("(users.id IN (?) OR users.id NOT IN (?))",
				orgAssignees().Where("user_whatsapp_accounts.whats_app_account_id IN ?", callerAccounts),
				orgAssignees())
		}
	}

	var rows []TransferTarget
	if err := query.Order("users.full_name").Scan(&rows).Error; err != nil {
		return nil, err
	}

	// The target must be able to open /chat at all; the access grant only
	// widens which conversations they see there.
	targets := make([]TransferTarget, 0, len(rows))
	for _, t := range rows {
		if a.HasPermission(t.ID, models.ResourceChat, models.ActionRead, orgID) {
			targets = append(targets, t)
		}
	}
	return targets, nil
}

// ListTransferTargets returns the colleagues the caller may transfer a
// conversation to (see transferTargets).
// Route: GET /api/contacts/transfer-targets  Permission: chat.assign:write
func (a *App) ListTransferTargets(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	targets, err := a.transferTargets(userID, orgID, nil)
	if err != nil {
		a.Log.Error("Failed to load transfer targets", "error", err, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load colleagues", nil, "")
	}

	return r.SendEnvelope(map[string]any{"users": targets})
}

// TransferChat hands an open conversation from its current assignee (the
// caller) to an eligible colleague. Thin adapter over ChatLifecycle.Assign,
// which persists the assignment, mints the target's access grant, and writes
// the system message, audit entry, and broadcast in one place.
// Route: PUT /api/contacts/{id}/transfer  Permission: chat.assign:write
func (a *App) TransferChat(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)
	if err != nil {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req TransferChatRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if req.UserID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "user_id is required", nil, "")
	}
	if req.UserID == userID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "You already own this conversation", nil, "")
	}

	// Scoped load + historical read-only refusal (a grant-only viewer is
	// never the assignee, but the refusal message is the accurate one).
	contact, err := a.findScopedMutableContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}

	if contact.AssignedUserID == nil || *contact.AssignedUserID != userID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden,
			"Only the assigned agent can transfer this conversation", nil, "")
	}
	if contact.EffectiveStatus() != models.ChatStatusOpen {
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			"Only open conversations can be transferred", nil, "")
	}

	eligible, err := a.transferTargets(userID, orgID, &req.UserID)
	if err != nil {
		a.Log.Error("Failed to check transfer target", "error", err, "user_id", userID, "target_user_id", req.UserID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to transfer conversation", nil, "")
	}
	if len(eligible) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"This colleague cannot receive the conversation", nil, "")
	}

	if err := a.ChatLifecycle.Assign(r.RequestCtx, orgID, userID, contact, &req.UserID); err != nil {
		a.Log.Error("Failed to transfer conversation", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to transfer conversation", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message":          "Conversation transferred",
		"assigned_user_id": req.UserID,
		"assigned_to_name": eligible[0].FullName,
	})
}
