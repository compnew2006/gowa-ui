package handlers

import (
	"errors"

	"github.com/compnew2006/gowa-ui/internal/accountaccess"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Send-account authorization: which WhatsApp account a user may send FROM.
//
// Seeing a conversation (scopeAssignedContact) and being allowed to write in
// it (rejectHistoricalAssignmentAccess / rejectNonParticipant) do not decide
// the sending number. Without this gate any chat:write user could name any
// org account in whatsapp_account and send from it. Every write path that
// talks to WhatsApp on an account's behalf (text, media, template, reaction,
// typing, revoke, scheduled messages, campaigns, status) resolves its account
// through resolveSendAccount or authorizeSendAccount.
//
// A user may send from an account when they are a super admin, have no
// account assignment in the org (documented fallback), or are assigned the
// account. With the org setting allow_cross_account_send, a current
// participant of the conversation may also reply from an unassigned account
// that already carries messages in that conversation.

// Sentinel denials. Their text is the stable error code sent to clients.
var (
	errAccountNotFound   = errors.New("account_not_found")
	errAccountNotAllowed = errors.New("account_not_allowed")
	errAccountRequired   = errors.New("account_required")
)

// sendAccountAuditResource is the audit_logs.resource_type of send denials.
const sendAccountAuditResource = "send_account"

// sendAccountPolicy returns the organization's send-account policy.
func (a *App) sendAccountPolicy(orgID uuid.UUID) accountaccess.SendPolicy {
	return accountaccess.LoadSendPolicy(a.DB, orgID)
}

// crossAccountSendEnabled reports whether the org allows participants to
// reply from unassigned accounts already present in the conversation.
func (a *App) crossAccountSendEnabled(orgID uuid.UUID) bool {
	return a.sendAccountPolicy(orgID).AllowCrossAccount
}

// accountAssignmentAllows is canAccessAccount for send decisions, except a
// failed assignment lookup fails CLOSED instead of falling back to full access.
func (a *App) accountAssignmentAllows(userID, orgID, accountID uuid.UUID) (bool, error) {
	if a.IsSuperAdmin(userID) {
		return true, nil
	}
	ids, err := a.assignedAccountIDs(userID, orgID)
	if err != nil {
		return false, err
	}
	return accountaccess.AssignmentAllows(ids, accountID), nil
}

// contactHasAccountMessages reports whether the conversation has at least one
// message sent or received through the named account. Errors fail closed.
func (a *App) contactHasAccountMessages(orgID, contactID uuid.UUID, accountName string) bool {
	var found []uuid.UUID
	if err := a.DB.Model(&models.Message{}).
		Where("contact_id = ? AND organization_id = ? AND whats_app_account = ?", contactID, orgID, accountName).
		Limit(1).Pluck("id", &found).Error; err != nil {
		a.Log.Error("Failed to check conversation account messages", "error", err, "contact_id", contactID)
		return false
	}
	return len(found) > 0
}

// authorizeSendAccount returns nil when userID may send from account in this
// conversation, or errAccountNotFound / errAccountNotAllowed. contact may be
// nil for sends that are not part of a conversation (status, campaigns), in
// which case only the assignment rule applies.
func (a *App) authorizeSendAccount(userID, orgID uuid.UUID, contact *models.Contact, account *models.WhatsAppAccount) error {
	if account == nil || account.OrganizationID != orgID || account.DeletedAt.Valid {
		return errAccountNotFound
	}
	allowed, err := a.accountAssignmentAllows(userID, orgID, account.ID)
	if err != nil {
		a.Log.Error("Failed to load account assignments for send", "error", err, "user_id", userID)
		return errAccountNotAllowed
	}
	if allowed {
		return nil
	}
	if contact == nil || !a.crossAccountSendEnabled(orgID) {
		return errAccountNotAllowed
	}
	// Cross-account replies need a CURRENT participant. Historical grant
	// holders and account-only viewers never qualify; closed_by is audit-only.
	switch a.conversationAccessMode(contact, userID, orgID) {
	case AccessModeStandard, AccessModeCurrentAssignee, AccessModeCollaborator:
	default:
		return errAccountNotAllowed
	}
	if contact.WhatsAppAccount == account.Name || a.contactHasAccountMessages(orgID, contact.ID, account.Name) {
		return nil
	}
	return errAccountNotAllowed
}

// findSendAccount loads a non-deleted org account by name with secrets
// decrypted, ready for the provider.
func (a *App) findSendAccount(orgID uuid.UUID, name string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount
	if err := a.DB.Where("name = ? AND organization_id = ?", name, orgID).First(&account).Error; err != nil {
		return nil, errAccountNotFound
	}
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// findDefaultOutgoingAccount loads the org's default outgoing account. Unlike
// resolveWhatsAppAccount it never falls back to an arbitrary org account.
func (a *App) findDefaultOutgoingAccount(orgID uuid.UUID) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).First(&account).Error; err != nil {
		return nil, errAccountNotFound
	}
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// conversationAccountNames lists the accounts of a conversation: the
// contact's own account first, then every account that has messages in it,
// most recently active first.
func (a *App) conversationAccountNames(orgID uuid.UUID, contact *models.Contact) []string {
	var names []string
	if contact.WhatsAppAccount != "" {
		names = append(names, contact.WhatsAppAccount)
	}
	var messageAccounts []string
	if err := a.DB.Model(&models.Message{}).
		Select("whats_app_account").
		Where("contact_id = ? AND organization_id = ? AND whats_app_account <> ''", contact.ID, orgID).
		Group("whats_app_account").
		Order("MAX(created_at) DESC").
		Pluck("whats_app_account", &messageAccounts).Error; err != nil {
		a.Log.Error("Failed to list conversation accounts", "error", err, "contact_id", contact.ID)
	}
	return appendUniqueNames(names, messageAccounts...)
}

// appendUniqueNames appends the non-empty names not already in list.
func appendUniqueNames(list []string, names ...string) []string {
	for _, n := range names {
		if n == "" {
			continue
		}
		dup := false
		for _, existing := range list {
			if existing == n {
				dup = true
				break
			}
		}
		if !dup {
			list = append(list, n)
		}
	}
	return list
}

// auditSendDenial records a refused (or, in audit-only mode, would-be refused)
// send. It deliberately carries identifiers only — never message text or media.
func (a *App) auditSendDenial(orgID, userID uuid.UUID, contact *models.Contact, account *models.WhatsAppAccount, endpoint string, reason error, enforced bool) {
	action := models.AuditActionSendDenied
	if !enforced {
		action = models.AuditActionSendWouldBeDenied
	}
	resourceID := uuid.Nil
	details := map[string]any{
		"endpoint": endpoint,
		"reason":   reason.Error(),
	}
	if account != nil {
		resourceID = account.ID
		details["account_id"] = account.ID.String()
		details["account_name"] = account.Name
	}
	if contact != nil {
		details["contact_id"] = contact.ID.String()
	}
	a.logAudit(orgID, userID, sendAccountAuditResource, resourceID, action, nil, details)
}

// sendSendAccountError writes the client-facing envelope for a send-account
// denial. The error code is stable so the UI can react to it.
func sendSendAccountError(r *fastglue.Request, reason error) {
	switch {
	case errors.Is(reason, errAccountNotFound):
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "WhatsApp account not found", nil, fastglue.ErrorType(errAccountNotFound.Error()))
	case errors.Is(reason, errAccountRequired):
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"No WhatsApp account you may send from is available for this conversation", nil, fastglue.ErrorType(errAccountRequired.Error()))
	default:
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden,
			"You do not have access to send from this WhatsApp account", nil, fastglue.ErrorType(errAccountNotAllowed.Error()))
	}
}

// rejectSendAccount handles an authorizeSendAccount denial for a request.
// Under enforcement it audits send_denied, sends the error envelope and
// returns true. In audit-only mode it audits send_would_be_denied and returns
// false so the caller proceeds. A missing account is never let through.
func (a *App) rejectSendAccount(r *fastglue.Request, userID, orgID uuid.UUID, contact *models.Contact, account *models.WhatsAppAccount, endpoint string, reason error) bool {
	if errors.Is(reason, errAccountNotFound) {
		sendSendAccountError(r, reason)
		return true
	}
	enforced := a.sendAccountPolicy(orgID).Enforce
	a.auditSendDenial(orgID, userID, contact, account, endpoint, reason, enforced)
	if !enforced {
		return false
	}
	sendSendAccountError(r, reason)
	return true
}

// sendAccountQuery describes how a write path picks its sending account.
type sendAccountQuery struct {
	// Requested is the account the client named, or the one fixed by the
	// target message (reactions, revokes). Empty means "pick one".
	Requested string
	// Preferred are implicit candidates tried before the contact's own
	// account (e.g. the template's account).
	Preferred []string
	// Endpoint labels the write path in audit records.
	Endpoint string
}

// resolveSendAccount picks and authorizes the account a write path sends from.
// It returns ok=false after sending the error envelope.
//
// A requested account must exist in the org (404 account_not_found) and pass
// authorizeSendAccount. Without one, the first candidate the user may send
// from wins: preferred accounts, the contact's account, the org default
// outgoing account, then accounts with messages in the conversation. There is
// no fallback to an arbitrary org account; nothing usable is 400
// account_required. In audit-only mode the legacy choice is kept and only
// recorded when it would have been refused.
func (a *App) resolveSendAccount(r *fastglue.Request, userID, orgID uuid.UUID, contact *models.Contact, q sendAccountQuery) (*models.WhatsAppAccount, bool) {
	if q.Requested != "" {
		account, err := a.findSendAccount(orgID, q.Requested)
		if err != nil {
			sendSendAccountError(r, errAccountNotFound)
			return nil, false
		}
		if err := a.authorizeSendAccount(userID, orgID, contact, account); err != nil {
			if a.rejectSendAccount(r, userID, orgID, contact, account, q.Endpoint, err) {
				return nil, false
			}
		}
		return account, true
	}

	candidates := appendUniqueNames(nil, q.Preferred...)
	if contact != nil {
		candidates = appendUniqueNames(candidates, contact.WhatsAppAccount)
	}

	// Fast path: the legacy choice (first named candidate, else the default
	// outgoing account) is usually allowed — no policy lookup needed.
	var legacy *models.WhatsAppAccount
	if len(candidates) > 0 {
		legacy, _ = a.findSendAccount(orgID, candidates[0])
	} else {
		legacy, _ = a.findDefaultOutgoingAccount(orgID)
	}
	if legacy != nil && a.authorizeSendAccount(userID, orgID, contact, legacy) == nil {
		return legacy, true
	}

	if !a.sendAccountPolicy(orgID).Enforce {
		// Audit-only: keep the exact legacy behavior (including its
		// any-account fallback) and record that it would be refused.
		legacyName := ""
		if len(candidates) > 0 {
			legacyName = candidates[0]
		}
		account, err := a.resolveWhatsAppAccount(orgID, legacyName)
		if err != nil {
			_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to resolve WhatsApp account", nil, "")
			return nil, false
		}
		if reason := a.authorizeSendAccount(userID, orgID, contact, account); reason != nil {
			a.auditSendDenial(orgID, userID, contact, account, q.Endpoint, reason, false)
		}
		return account, true
	}

	if def, err := a.findDefaultOutgoingAccount(orgID); err == nil {
		candidates = appendUniqueNames(candidates, def.Name)
	}
	if contact != nil {
		candidates = appendUniqueNames(candidates, a.conversationAccountNames(orgID, contact)...)
	}
	for _, name := range candidates {
		account, err := a.findSendAccount(orgID, name)
		if err != nil {
			continue
		}
		if a.authorizeSendAccount(userID, orgID, contact, account) == nil {
			return account, true
		}
	}

	a.auditSendDenial(orgID, userID, contact, legacy, q.Endpoint, errAccountRequired, true)
	sendSendAccountError(r, errAccountRequired)
	return nil, false
}

// SendAccountOption is one account of a conversation and whether the caller
// may send from it.
type SendAccountOption struct {
	Name    string  `json:"name"`
	CanSend bool    `json:"can_send"`
	Reason  *string `json:"reason"`
}

// ListContactSendAccounts returns the conversation's accounts with the
// server's send decision for each, so the UI never re-implements the policy.
// It applies the same conversation gates as SendMessage and exposes names
// only — never credentials.
//
// GET /api/contacts/{id}/send-accounts
func (a *App) ListContactSendAccounts(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	query := a.scopeAssignedContact(a.DB.Where("id = ? AND organization_id = ?", contactID, orgID), userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	if a.rejectHistoricalAssignmentAccess(r, &contact, userID, orgID) {
		return nil
	}
	if a.rejectNonParticipant(r, &contact, userID, orgID) {
		return nil
	}

	names := a.conversationAccountNames(orgID, &contact)
	options := make([]SendAccountOption, 0, len(names))
	for _, name := range names {
		opt := SendAccountOption{Name: name}
		var reason error
		account, findErr := a.findSendAccount(orgID, name)
		if findErr != nil {
			reason = errAccountNotFound
		} else {
			reason = a.authorizeSendAccount(userID, orgID, &contact, account)
		}
		if reason == nil {
			opt.CanSend = true
		} else {
			code := reason.Error()
			opt.Reason = &code
		}
		options = append(options, opt)
	}

	return r.SendEnvelope(map[string]any{"accounts": options})
}

// authorizeCampaignActor checks that the caller acting on a campaign (start,
// retry) may send from the campaign's account. On denial it sends the error
// envelope and returns false.
func (a *App) authorizeCampaignActor(r *fastglue.Request, orgID uuid.UUID, campaign *models.BulkMessageCampaign, endpoint string) bool {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return false
	}
	_, ok := a.resolveSendAccount(r, userID, orgID, nil, sendAccountQuery{
		Requested: campaign.WhatsAppAccount,
		Endpoint:  endpoint,
	})
	return ok
}

// authorizeBackgroundSend re-checks, at fire time, that the user who created a
// deferred send (scheduled message, campaign) may still send from account —
// their assignment can be revoked after scheduling. It returns nil when the
// send may proceed (allowed, or the org runs the policy in audit-only mode)
// and the denial reason otherwise; denials are audited against the creator.
func (a *App) authorizeBackgroundSend(creatorID, orgID uuid.UUID, contact *models.Contact, account *models.WhatsAppAccount, endpoint string) error {
	reason := a.authorizeSendAccount(creatorID, orgID, contact, account)
	if reason == nil {
		return nil
	}
	if errors.Is(reason, errAccountNotFound) {
		return reason
	}
	enforced := a.sendAccountPolicy(orgID).Enforce
	a.auditSendDenial(orgID, creatorID, contact, account, endpoint, reason, enforced)
	if !enforced {
		return nil
	}
	return reason
}
