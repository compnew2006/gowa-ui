package handlers

import (
	"time"

	"github.com/compnew2006/gowa-ui/internal/audit"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Assignment access-grant management: listing who holds permanent
// (post-assignment) access to a conversation, and Release — the ONLY action
// that ends a grant. Assignment itself (Service.Assign) creates grants;
// unassign/close merely move the holder to read-only.
//
// NOTE: the route param is deliberately named {target_user_id} — the auth
// middleware overwrites the "user_id" UserValue with the CALLER's identity
// after the router sets path params, so any handler reading UserValue
// ("user_id") as a path target would silently operate on the caller instead.

// requireGrantManagementPerm checks chat.assign:write OR contacts:write —
// the same population that may assign conversations may manage their grants.
// Sends the error envelope and returns false when neither is held.
func (a *App) requireGrantManagementPerm(r *fastglue.Request, orgID, userID uuid.UUID) bool {
	if a.HasPermission(userID, models.ResourceChatAssign, models.ActionWrite, orgID) ||
		a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return true
	}
	_ = r.SendErrorEnvelope(fasthttp.StatusForbidden,
		"You do not have permission to manage access grants", nil, "")
	return false
}

// AccessGrantResponse is one active grant, resolved to display names.
type AccessGrantResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	UserName    string    `json:"user_name"`
	GrantedBy   uuid.UUID `json:"granted_by"`
	GrantedName string    `json:"granted_by_name"`
	GrantedAt   time.Time `json:"granted_at"`
	IsAssignee  bool      `json:"is_current_assignee"`
}

// ListAccessGrants returns the ACTIVE assignment access grants for a
// conversation (the "previous access" section in the assign dialog).
// Route: GET /api/contacts/{id}/access-grants
func (a *App) ListAccessGrants(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	if !a.requireGrantManagementPerm(r, orgID, userID) {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}
	// A historical read-only holder may neither see other users' grants nor
	// act on them — grant management belongs to account/assign-permission
	// holders with normal access.
	if a.rejectHistoricalAssignmentAccess(r, contact, userID, orgID) {
		return nil
	}

	var grants []models.ContactAssignmentAccessGrant
	if err := a.DB.
		Where("contact_id = ? AND organization_id = ? AND revoked_at IS NULL", contact.ID, orgID).
		Order("granted_at DESC").
		Find(&grants).Error; err != nil {
		a.Log.Error("Failed to list access grants", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list access grants", nil, "")
	}

	response := make([]AccessGrantResponse, 0, len(grants))
	for _, g := range grants {
		response = append(response, AccessGrantResponse{
			UserID:      g.UserID,
			UserName:    audit.GetUserName(a.DB, g.UserID),
			GrantedBy:   g.GrantedBy,
			GrantedName: audit.GetUserName(a.DB, g.GrantedBy),
			GrantedAt:   g.GrantedAt,
			IsAssignee:  contact.AssignedUserID != nil && *contact.AssignedUserID == g.UserID,
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":    contact.ID,
		"access_grants": response,
	})
}

// RevokeAccessGrant is the Release for a permanent assignment access grant.
// Idempotent: revoking an already-revoked (or never-granted) access still 200s.
//
//	If the target user is the CURRENT assignee:
//	  - open conversation  → released to pending + unassigned (same semantics
//	    as Service.Release, including system message + broadcast);
//	  - closed conversation → stays closed (assignment was already cleared by
//	    close; closed_at/closed_by metadata preserved untouched).
//	If the conversation is assigned to someone else, only the target's grant
//	is revoked — the current assignee is untouched.
//
// Route: DELETE /api/contacts/{id}/access-grants/{target_user_id}
func (a *App) RevokeAccessGrant(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	if !a.requireGrantManagementPerm(r, orgID, userID) {
		return nil
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}
	// Same rule as ListAccessGrants: a historical read-only holder cannot
	// manage grants (including releasing other users' access).
	if a.rejectHistoricalAssignmentAccess(r, contact, userID, orgID) {
		return nil
	}

	targetID, err := parsePathUUID(r, "target_user_id", "user")
	if err != nil {
		return nil
	}

	// ATOMIC Release: the grant revocation and (for the assignee case) the
	// assignment release commit together or not at all — a partial failure
	// must never leave a user released but still holding a permanent grant
	// (unsafe direction), nor stripped of the grant while still responsible
	// for the conversation (safe but confusing direction).
	revoked := false
	assigneeReleased := false
	txErr := a.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Model(&models.ContactAssignmentAccessGrant{}).
			Where("contact_id = ? AND user_id = ? AND organization_id = ? AND revoked_at IS NULL",
				contact.ID, targetID, orgID).
			Updates(map[string]any{"revoked_at": now, "revoked_by": userID})
		if res.Error != nil {
			return res.Error
		}
		revoked = res.RowsAffected > 0

		// If the released user is the current assignee, release the assignment
		// too — the "Release" the plan defines for the assignee case.
		if contact.AssignedUserID != nil && *contact.AssignedUserID == targetID {
			if contact.EffectiveStatus() == models.ChatStatusClosed {
				// Closed stays closed; close already cleared the assignee, but
				// a legacy closed-but-assigned row is normalized here.
				if err := tx.Model(&models.Contact{}).Where("id = ?", contact.ID).
					Update("assigned_user_id", nil).Error; err != nil {
					return err
				}
				assigneeReleased = true
				return nil
			}
			released, rerr := a.ChatLifecycle.ReleaseWithDB(r.RequestCtx, tx, orgID, userID, contact, false, true)
			if rerr != nil {
				return rerr
			}
			assigneeReleased = released
		}
		return nil
	})
	if txErr != nil {
		a.Log.Error("Failed to release access", "error", txErr, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to release access", nil, "")
	}

	if revoked || assigneeReleased {
		actorName := audit.GetUserName(a.DB, userID)
		audit.LogAudit(a.DB, orgID, userID, actorName,
			"contact", contact.ID, models.AuditActionUpdated, nil, contact,
			map[string]any{
				"access_grant_released": map[string]any{
					"user_id":          targetID.String(),
					"was_assignee":     assigneeReleased,
					"grant_existed":    revoked,
					"conversation_now": string(contact.EffectiveStatus()),
				},
			})
	}

	// Broadcast so the released user's clients drop the conversation
	// immediately (the server scope is the authority; this is UX only).
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeChatAccessRevoked,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    targetID.String(),
				"released":   true,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"contact_id":         contact.ID,
		"released_user_id":   targetID,
		"grant_revoked":      revoked,
		"assignment_release": assigneeReleased,
		"message":            "Access released",
	})
}
