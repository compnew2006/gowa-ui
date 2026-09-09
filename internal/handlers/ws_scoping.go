package handlers

// WebSocket recipient scoping.
//
// Review finding (2026-09-09): every conversation-content broadcast used
// BroadcastToOrg, so any org member with a raw WebSocket connection received
// new_message payloads — body text, media URLs, reply previews — for
// conversations their account scope excludes (e.g. an agent assigned a
// subset of WhatsApp accounts). The frontend filtered such events out of the
// UI, but the server is the authority: filtering must happen before send.
//
// Rule: a user may receive real-time events about a contact iff that user
// could LIST/READ the conversation — exactly the scopeAssignedContact
// population:
//
//   - involvement on ANY account: current assignee, current collaborator, or
//     ACTIVE contact_assignment_access_grants holder; OR
//   - contacts:read AND account scope covers the contact's account
//     (super admin, or no account assignments = org-wide fallback, or the
//     contact's account is inside the user's assigned subset).
//
// Errors fail CLOSED: an empty recipient list drops the broadcast (UI falls
// back to refetch/refresh) rather than leaking to the whole org.
//
// Deliberately still org-wide (no conversation content in their payloads):
// lifecycle events (chat_claimed/released/closed/reopened, access-revoked —
// the user who LOST access must still receive the one event telling them to
// drop the chat) and ops events (campaign progress, device/app status).

import (
	"github.com/google/uuid"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
)

// wsContactRecipients returns the IDs of org users allowed to receive
// real-time WebSocket events about this contact. See the file comment for
// the rule; errors fail closed with an empty result.
func (a *App) wsContactRecipients(contact *models.Contact, orgID uuid.UUID) []uuid.UUID {
	if contact == nil {
		return nil
	}

	var userIDs []uuid.UUID
	if err := a.DB.Model(&models.User{}).
		Where("organization_id = ?", orgID).
		Pluck("id", &userIDs).Error; err != nil || len(userIDs) == 0 {
		a.Log.Error("ws scoping: failed to load org users, dropping broadcast",
			"error", err, "org_id", orgID, "contact_id", contact.ID)
		return nil
	}

	// The org accounts whose name matches this contact's account.
	var matchingAccountIDs []uuid.UUID
	if err := a.DB.Model(&models.WhatsAppAccount{}).
		Where("organization_id = ? AND name = ?", orgID, contact.WhatsAppAccount).
		Pluck("id", &matchingAccountIDs).Error; err != nil {
		a.Log.Error("ws scoping: failed to resolve contact account, dropping broadcast",
			"error", err, "contact_id", contact.ID)
		return nil
	}
	matching := make(map[uuid.UUID]bool, len(matchingAccountIDs))
	for _, id := range matchingAccountIDs {
		matching[id] = true
	}

	// All account assignments for these users, in one query.
	var rows []struct {
		UserID    uuid.UUID `gorm:"column:user_id"`
		AccountID uuid.UUID `gorm:"column:whats_app_account_id"`
	}
	if err := a.DB.Table("user_whatsapp_accounts").
		Select("user_id, whats_app_account_id").
		Where("user_id IN ?", userIDs).
		Scan(&rows).Error; err != nil {
		a.Log.Error("ws scoping: failed to load account assignments, dropping broadcast",
			"error", err, "contact_id", contact.ID)
		return nil
	}
	assignedAny := make(map[uuid.UUID]bool, len(rows))
	accountOK := make(map[uuid.UUID]bool, len(rows))
	for _, r := range rows {
		assignedAny[r.UserID] = true
		if matching[r.AccountID] {
			accountOK[r.UserID] = true
		}
	}

	// Active grant holders on this contact.
	var grantUserIDs []uuid.UUID
	if err := a.DB.Model(&models.ContactAssignmentAccessGrant{}).
		Where("contact_id = ? AND organization_id = ? AND revoked_at IS NULL", contact.ID, orgID).
		Pluck("user_id", &grantUserIDs).Error; err != nil {
		a.Log.Error("ws scoping: failed to load access grants, dropping broadcast",
			"error", err, "contact_id", contact.ID)
		return nil
	}
	grants := make(map[uuid.UUID]bool, len(grantUserIDs))
	for _, id := range grantUserIDs {
		grants[id] = true
	}

	collaborators := make(map[string]bool)
	for _, c := range contact.GetCollaborators() {
		collaborators[c.UserID] = true
	}

	recipients := make([]uuid.UUID, 0, len(userIDs))
	for _, uid := range userIDs {
		involved := (contact.AssignedUserID != nil && *contact.AssignedUserID == uid) ||
			collaborators[uid.String()] || grants[uid]
		if involved {
			recipients = append(recipients, uid)
			continue
		}
		// Account-scope users need contacts:read — without it, involvement is
		// the sole source of access (same as scopeAssignedContact).
		if !a.HasPermission(uid, models.ResourceContacts, models.ActionRead, orgID) {
			continue
		}
		if a.IsSuperAdmin(uid) || !assignedAny[uid] || accountOK[uid] {
			recipients = append(recipients, uid)
		}
	}
	return recipients
}

// wsContactRecipientsByID is wsContactRecipients for call sites that hold
// only the contact ID (status/reaction/edit patches). A missing contact
// (deleted) drops the broadcast.
func (a *App) wsContactRecipientsByID(orgID, contactID uuid.UUID) []uuid.UUID {
	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).
		First(&contact).Error; err != nil {
		return nil
	}
	return a.wsContactRecipients(&contact, orgID)
}

// wsContactRecipientsForChatJID resolves the contact behind a WhatsApp chat
// JID (1:1 @s.whatsapp.net or group @g.us — PhoneFromJID strips the domain,
// and contact phone_number stores the bare ID for both) under a specific
// account, then returns its recipient set. Presence events use this: without
// a matching contact there is no open chat to render the indicator for, so
// the event is dropped.
func (a *App) wsContactRecipientsForChatJID(orgID uuid.UUID, accountName, chatJID string) []uuid.UUID {
	phone := gowa.PhoneFromJID(chatJID)
	if phone == "" {
		return nil
	}
	var contact models.Contact
	if err := a.DB.Where("organization_id = ? AND whats_app_account = ? AND phone_number = ?",
		orgID, accountName, phone).First(&contact).Error; err != nil {
		return nil
	}
	return a.wsContactRecipients(&contact, orgID)
}
