// Package accountaccess holds the WhatsApp-account assignment rules shared by
// the HTTP handlers and the campaign worker. A user assigned a subset of the
// org's accounts (user_whatsapp_accounts) may operate only on that subset;
// a user with no assignment in the org keeps full org access (fallback).
//
// It also owns the per-organization send-account policy switches stored in
// organizations.settings, so the request path and background senders read
// them the same way.
package accountaccess

import (
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Organization settings keys (organizations.settings JSONB) for the
// send-account policy.
const (
	// SettingAllowCrossAccountSend lets a current conversation participant
	// reply from an account they are not assigned to, as long as that account
	// already carries messages in the conversation. Default false.
	SettingAllowCrossAccountSend = "allow_cross_account_send"
	// SettingEnforceSendAccountPolicy blocks denied sends when true (default).
	// When false the policy runs in audit-only mode: denials are recorded as
	// send_would_be_denied and the send proceeds with the legacy account choice.
	SettingEnforceSendAccountPolicy = "enforce_send_account_policy"
)

// SendPolicy is the effective send-account policy of an organization.
type SendPolicy struct {
	Enforce           bool
	AllowCrossAccount bool
}

// DefaultSendPolicy is the safe default: enforce, no cross-account sending.
func DefaultSendPolicy() SendPolicy {
	return SendPolicy{Enforce: true}
}

// ParseSendPolicy applies the organization settings over the safe defaults.
// Non-boolean values are ignored.
func ParseSendPolicy(settings models.JSONB) SendPolicy {
	p := DefaultSendPolicy()
	if v, ok := settings[SettingEnforceSendAccountPolicy].(bool); ok {
		p.Enforce = v
	}
	if v, ok := settings[SettingAllowCrossAccountSend].(bool); ok {
		p.AllowCrossAccount = v
	}
	return p
}

// LoadSendPolicy reads the organization's policy. Lookup errors return the
// safe defaults so a DB failure never relaxes enforcement.
func LoadSendPolicy(db *gorm.DB, orgID uuid.UUID) SendPolicy {
	var org models.Organization
	if err := db.Select("id", "settings").Where("id = ?", orgID).First(&org).Error; err != nil {
		return DefaultSendPolicy()
	}
	return ParseSendPolicy(org.Settings)
}

// AssignedAccountIDs returns the IDs of non-deleted accounts within the org
// explicitly assigned to the user. An empty result means the user has no
// assignments in this org (full org access fallback).
func AssignedAccountIDs(db *gorm.DB, userID, orgID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := db.Model(&models.UserWhatsAppAccount{}).
		Joins("JOIN whatsapp_accounts ON whatsapp_accounts.id = user_whatsapp_accounts.whats_app_account_id AND whatsapp_accounts.deleted_at IS NULL").
		Where("user_whatsapp_accounts.user_id = ? AND whatsapp_accounts.organization_id = ?", userID, orgID).
		Pluck("user_whatsapp_accounts.whats_app_account_id", &ids).Error
	return ids, err
}

// AssignmentAllows reports whether an assignment set covers accountID. An
// empty set is the no-assignment fallback and covers every org account.
func AssignmentAllows(assigned []uuid.UUID, accountID uuid.UUID) bool {
	if len(assigned) == 0 {
		return true
	}
	for _, id := range assigned {
		if id == accountID {
			return true
		}
	}
	return false
}

// UserMayUseAccount is the assignment check for callers without the handler
// permission cache (background workers): super admins always pass, otherwise
// the user's assignment set must cover the account. Errors are returned so
// callers can fail closed.
func UserMayUseAccount(db *gorm.DB, userID, orgID, accountID uuid.UUID) (bool, error) {
	var user models.User
	if err := db.Select("id", "is_super_admin").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}
	if user.IsSuperAdmin {
		return true, nil
	}
	ids, err := AssignedAccountIDs(db, userID, orgID)
	if err != nil {
		return false, err
	}
	return AssignmentAllows(ids, accountID), nil
}
