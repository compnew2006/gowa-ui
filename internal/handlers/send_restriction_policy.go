package handlers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	organizationSettingStrictSendingRestrictionsEnabled = "strict_sending_restrictions_enabled"
	userSettingSendRestrictions                         = "send_restrictions"
)

// restrictedSendViolationError is returned when a user is blocked by strict send restrictions.
type restrictedSendViolationError struct {
	message string
}

func (e *restrictedSendViolationError) Error() string {
	if e == nil {
		return "message blocked by strict sending restrictions"
	}
	if strings.TrimSpace(e.message) == "" {
		return "message blocked by strict sending restrictions"
	}
	return e.message
}

func asRestrictedSendViolation(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	var violation *restrictedSendViolationError
	if !errors.As(err, &violation) {
		return "", false
	}
	return violation.Error(), true
}

type sendRestrictionsSettings struct {
	Enabled            bool
	IncludeAllContacts bool
	AuthorizedNumbers  []string
	AllowedInstanceID  *uuid.UUID
	PrefixAgentName    bool
}

func readSendRestrictionsSettings(settings models.JSONB) sendRestrictionsSettings {
	cfg := sendRestrictionsSettings{
		Enabled:            false,
		IncludeAllContacts: false,
		AuthorizedNumbers:  []string{},
		AllowedInstanceID:  nil,
		PrefixAgentName:    true,
	}
	if settings == nil {
		return cfg
	}

	raw, ok := settings[userSettingSendRestrictions]
	if !ok || raw == nil {
		return cfg
	}

	var payload map[string]interface{}
	switch typed := raw.(type) {
	case map[string]interface{}:
		payload = typed
	case models.JSONB:
		payload = map[string]interface{}(typed)
	default:
		return cfg
	}

	if enabled, ok := payload["enabled"].(bool); ok {
		cfg.Enabled = enabled
	}
	if includeAllContacts, ok := payload["include_all_contacts"].(bool); ok {
		cfg.IncludeAllContacts = includeAllContacts
	}
	if prefixAgentName, ok := payload["prefix_agent_name"].(bool); ok {
		cfg.PrefixAgentName = prefixAgentName
	}

	if rawNumbers, ok := payload["authorized_numbers"]; ok {
		cfg.AuthorizedNumbers = normalizeRestrictedNumbers(asStringSlice(rawNumbers))
	}
	cfg.AllowedInstanceID = parseOptionalUUID(payload["allowed_instance_id"])

	return cfg
}

func writeSendRestrictionsSettings(settings models.JSONB, cfg sendRestrictionsSettings) models.JSONB {
	if settings == nil {
		settings = models.JSONB{}
	}
	restrictions := models.JSONB{
		"enabled":              cfg.Enabled,
		"include_all_contacts": cfg.IncludeAllContacts,
		"authorized_numbers":   normalizeRestrictedNumbers(cfg.AuthorizedNumbers),
		"prefix_agent_name":    cfg.PrefixAgentName,
	}
	if cfg.AllowedInstanceID != nil {
		restrictions["allowed_instance_id"] = cfg.AllowedInstanceID.String()
	} else {
		restrictions["allowed_instance_id"] = nil
	}
	settings[userSettingSendRestrictions] = restrictions
	return settings
}

func (a *App) shouldPrefixAgentNameForUser(orgID, userID uuid.UUID) bool {
	if a == nil || a.DB == nil || userID == uuid.Nil {
		return false
	}

	if orgID == uuid.Nil {
		return true
	}

	user, err := a.loadUserForSendRestrictions(orgID, userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			a.Log.Error("Failed to resolve user settings for message prefix", "error", err, "org_id", orgID, "user_id", userID)
		}
		return true
	}

	cfg := readSendRestrictionsSettings(user.Settings)
	return cfg.PrefixAgentName
}

func parseOptionalUUID(raw interface{}) *uuid.UUID {
	switch typed := raw.(type) {
	case uuid.UUID:
		value := typed
		return &value
	case *uuid.UUID:
		if typed == nil {
			return nil
		}
		value := *typed
		return &value
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		parsed, err := uuid.Parse(trimmed)
		if err != nil {
			return nil
		}
		return &parsed
	case []byte:
		trimmed := strings.TrimSpace(string(typed))
		if trimmed == "" {
			return nil
		}
		parsed, err := uuid.Parse(trimmed)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func stringifyOptionalUUID(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	formatted := value.String()
	return &formatted
}

func asStringSlice(raw interface{}) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string{}, typed...)
	case models.StringArray:
		return append([]string{}, typed...)
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			value, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, value)
		}
		return out
	default:
		return nil
	}
}

func normalizeRestrictedNumbers(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := normalizeRestrictedPhoneNumber(value)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}

	sort.Strings(normalized)
	return normalized
}

func normalizeRestrictedPhoneNumber(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if direct := directUserFromConversationID(trimmed); direct != "" {
		trimmed = direct
	}

	trimmed = strings.ReplaceAll(trimmed, " ", "")
	trimmed = strings.TrimPrefix(trimmed, "+")

	var digits strings.Builder
	digits.Grow(len(trimmed))
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}

	return digits.String()
}

func containsRestrictedNumber(numbers []string, target string) bool {
	if len(numbers) == 0 || target == "" {
		return false
	}
	index := sort.SearchStrings(numbers, target)
	return index < len(numbers) && numbers[index] == target
}

func mergeRestrictedNumbers(existing []string, additions []string) ([]string, bool) {
	merged := normalizeRestrictedNumbers(append(append([]string{}, existing...), additions...))
	if len(merged) != len(normalizeRestrictedNumbers(existing)) {
		return merged, true
	}

	current := normalizeRestrictedNumbers(existing)
	for i := range merged {
		if merged[i] != current[i] {
			return merged, true
		}
	}

	return merged, false
}

func (a *App) isStrictSendingRestrictionsEnabled(orgID uuid.UUID) bool {
	if a == nil || a.DB == nil || orgID == uuid.Nil {
		return false
	}

	var org models.Organization
	if err := a.DB.Select("settings").Where("id = ?", orgID).First(&org).Error; err != nil {
		return false
	}

	if org.Settings == nil {
		return false
	}
	enabled, ok := org.Settings[organizationSettingStrictSendingRestrictionsEnabled].(bool)
	return ok && enabled
}

func (a *App) loadUserForSendRestrictions(orgID, userID uuid.UUID) (*models.User, error) {
	if a == nil || a.DB == nil {
		return nil, gorm.ErrInvalidDB
	}

	var user models.User
	err := a.DB.
		Select("users.id", "users.settings").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (a *App) saveUserSendRestrictions(userID uuid.UUID, existingSettings models.JSONB, cfg sendRestrictionsSettings) error {
	if a == nil || a.DB == nil {
		return gorm.ErrInvalidDB
	}
	updatedSettings := writeSendRestrictionsSettings(existingSettings, cfg)
	return a.DB.Model(&models.User{}).Where("id = ?", userID).Update("settings", updatedSettings).Error
}

func (a *App) contactHasIncomingHistory(orgID, contactID uuid.UUID) (bool, error) {
	var count int64
	err := a.DB.Model(&models.Message{}).
		Where("organization_id = ? AND contact_id = ? AND direction = ?", orgID, contactID, models.DirectionIncoming).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *App) collectAutoAuthorizedNumbersForUser(orgID, userID uuid.UUID) ([]string, error) {
	type phoneRow struct {
		PhoneNumber string `gorm:"column:phone_number"`
	}

	rows := make([]phoneRow, 0)
	err := a.DB.Model(&models.Contact{}).
		Select("DISTINCT contacts.phone_number").
		Joins("JOIN messages ON messages.contact_id = contacts.id AND messages.direction = ?", models.DirectionIncoming).
		Where("contacts.organization_id = ? AND contacts.assigned_user_id = ? AND contacts.deleted_at IS NULL", orgID, userID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	numbers := make([]string, 0, len(rows))
	for _, row := range rows {
		if normalized := normalizeRestrictedPhoneNumber(row.PhoneNumber); normalized != "" {
			numbers = append(numbers, normalized)
		}
	}

	return normalizeRestrictedNumbers(numbers), nil
}

func (a *App) collectAllContactNumbersForOrg(orgID uuid.UUID) ([]string, error) {
	type contactRow struct {
		PhoneNumber string       `gorm:"column:phone_number"`
		Metadata    models.JSONB `gorm:"column:metadata"`
	}

	rows := make([]contactRow, 0)
	err := a.DB.Model(&models.Contact{}).
		Select("phone_number", "metadata").
		Where("organization_id = ? AND contacts.deleted_at IS NULL", orgID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	numbers := make([]string, 0, len(rows))
	for _, row := range rows {
		if isGroupConversationID(row.PhoneNumber) || isChannelConversationID(row.PhoneNumber) {
			continue
		}
		if row.Metadata != nil {
			if isGroup, ok := row.Metadata["is_group_chat"].(bool); ok && isGroup {
				continue
			}
			if isChannel, ok := row.Metadata["is_channel_chat"].(bool); ok && isChannel {
				continue
			}
		}
		if normalized := normalizeRestrictedPhoneNumber(row.PhoneNumber); normalized != "" {
			numbers = append(numbers, normalized)
		}
	}

	return normalizeRestrictedNumbers(numbers), nil
}

func (a *App) syncUserRestrictionsWithSources(orgID uuid.UUID, user *models.User, cfg sendRestrictionsSettings) (sendRestrictionsSettings, error) {
	if user == nil {
		return cfg, nil
	}

	autoNumbers, err := a.collectAutoAuthorizedNumbersForUser(orgID, user.ID)
	if err != nil {
		return cfg, err
	}
	merged, changed := mergeRestrictedNumbers(cfg.AuthorizedNumbers, autoNumbers)
	if cfg.IncludeAllContacts {
		contactNumbers, collectErr := a.collectAllContactNumbersForOrg(orgID)
		if collectErr != nil {
			return cfg, collectErr
		}
		var allContactsChanged bool
		merged, allContactsChanged = mergeRestrictedNumbers(merged, contactNumbers)
		changed = changed || allContactsChanged
	}
	if !changed {
		cfg.AuthorizedNumbers = merged
		return cfg, nil
	}

	cfg.AuthorizedNumbers = merged
	if err := a.saveUserSendRestrictions(user.ID, user.Settings, cfg); err != nil {
		return cfg, err
	}
	user.Settings = writeSendRestrictionsSettings(user.Settings, cfg)
	return cfg, nil
}

func (a *App) getRestrictedInstanceForUser(orgID, userID uuid.UUID) (*uuid.UUID, error) {
	if a == nil || a.DB == nil || orgID == uuid.Nil || userID == uuid.Nil || !a.isWhatsmeowProvider() {
		return nil, nil
	}

	user, err := a.loadUserForSendRestrictions(orgID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	cfg := readSendRestrictionsSettings(user.Settings)
	if cfg.AllowedInstanceID == nil {
		return nil, nil
	}

	return cfg.AllowedInstanceID, nil
}

func resolveOutgoingInstanceID(req OutgoingMessageRequest) *uuid.UUID {
	if req.InstanceID != nil {
		return req.InstanceID
	}
	if req.Contact != nil {
		return req.Contact.InstanceID
	}
	return nil
}

func (a *App) enforceStrictSendRestrictions(ctx context.Context, req OutgoingMessageRequest, opts MessageSendOptions) error {
	if a == nil || a.DB == nil || opts.SentByUserID == nil || req.Contact == nil {
		return nil
	}

	if isGroupContact(req.Contact) || isChannelContact(req.Contact) {
		return nil
	}

	orgID := req.Contact.OrganizationID
	if orgID == uuid.Nil && req.Account != nil {
		orgID = req.Account.OrganizationID
	}
	if orgID == uuid.Nil || !a.isStrictSendingRestrictionsEnabled(orgID) {
		return nil
	}

	user, err := a.loadUserForSendRestrictions(orgID, *opts.SentByUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to load user send restrictions: %w", err)
	}

	cfg := readSendRestrictionsSettings(user.Settings)
	if !cfg.Enabled {
		return nil
	}

	cfg, err = a.syncUserRestrictionsWithSources(orgID, user, cfg)
	if err != nil {
		return fmt.Errorf("failed to sync authorized numbers: %w", err)
	}

	targetPhone := strings.TrimSpace(req.Contact.PhoneNumber)
	if canonical := a.resolveDirectRecipientFromConversation(ctx, req.Contact); canonical != "" {
		targetPhone = canonical
	}
	if a.isWhatsmeowProvider() {
		if cfg.AllowedInstanceID == nil {
			reason := "restricted user does not have an allowed instance configured"
			a.logRestrictedSendBlocked(orgID, *opts.SentByUserID, req.Contact, req.Type, targetPhone, reason)
			return &restrictedSendViolationError{message: "Message blocked by strict sending restrictions. Your user must be assigned to a WhatsApp instance."}
		}

		outgoingInstanceID := resolveOutgoingInstanceID(req)
		if outgoingInstanceID == nil || *outgoingInstanceID != *cfg.AllowedInstanceID {
			requestedInstance := ""
			if outgoingInstanceID != nil {
				requestedInstance = outgoingInstanceID.String()
			}
			reason := fmt.Sprintf("instance mismatch (allowed=%s, requested=%s)", cfg.AllowedInstanceID.String(), requestedInstance)
			a.logRestrictedSendBlocked(orgID, *opts.SentByUserID, req.Contact, req.Type, targetPhone, reason)
			return &restrictedSendViolationError{message: "Message blocked by strict sending restrictions. You can only send and receive chats on your assigned WhatsApp instance."}
		}
	}

	targetNumber := normalizeRestrictedPhoneNumber(targetPhone)
	if targetNumber == "" {
		reason := "contact phone number could not be normalized"
		a.logRestrictedSendBlocked(orgID, *opts.SentByUserID, req.Contact, req.Type, targetPhone, reason)
		return &restrictedSendViolationError{message: "Message blocked by strict sending restrictions. This chat does not map to a valid phone number."}
	}

	if containsRestrictedNumber(cfg.AuthorizedNumbers, targetNumber) {
		return nil
	}

	hasIncomingHistory, err := a.contactHasIncomingHistory(orgID, req.Contact.ID)
	if err != nil {
		return fmt.Errorf("failed to validate incoming history: %w", err)
	}
	if hasIncomingHistory {
		cfg.AuthorizedNumbers, _ = mergeRestrictedNumbers(cfg.AuthorizedNumbers, []string{targetNumber})
		if err := a.saveUserSendRestrictions(user.ID, user.Settings, cfg); err != nil {
			return fmt.Errorf("failed to persist authorized number: %w", err)
		}
		return nil
	}

	reason := "number is not present in user's authorized list and has no prior incoming history"
	a.logRestrictedSendBlocked(orgID, *opts.SentByUserID, req.Contact, req.Type, targetNumber, reason)
	return &restrictedSendViolationError{message: "Message blocked by strict sending restrictions. You can only send to phone numbers that have previously sent incoming messages in your assigned chats."}
}

func (a *App) logRestrictedSendBlocked(
	orgID, userID uuid.UUID,
	contact *models.Contact,
	messageType models.MessageType,
	targetPhone,
	reason string,
) {
	if a == nil || a.DB == nil {
		return
	}

	metadata := models.JSONB{
		"message_type": messageType,
		"target_phone": normalizeActivityText(targetPhone, 80),
		"reason":       normalizeActivityText(reason, 220),
	}

	entry := &models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "security",
		EventType:      "security.restricted_send_blocked",
		Action:         "send_message",
		Status:         "blocked",
		Source:         "security",
		Metadata:       metadata,
	}

	if contact != nil {
		contactID := contact.ID
		entry.ContactID = &contactID
		if name := normalizeActivityText(contact.ProfileName, 120); name != "" {
			entry.Metadata["contact_name"] = name
		}
		if phone := normalizeActivityText(contact.PhoneNumber, 80); phone != "" {
			entry.Metadata["contact_phone"] = phone
		}
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log restricted send block", "error", err, "org_id", orgID, "user_id", userID)
	}
}
