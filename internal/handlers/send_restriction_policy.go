package handlers

import (
	"context"

	"github.com/compnew2006/whatomate/pkg/chat_close_ratings"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	organizationSettingStrictSendingRestrictionsEnabled = "strict_sending_restrictions_enabled"
	organizationSettingOutboundMode                     = "outbound_mode"
	organizationSettingStrictSendingApplyToSystem       = "strict_sending_apply_to_system"
	organizationSettingCampaignDraftOnly                = "campaign_draft_only"
	organizationSettingStrictRolloutMode                = "strict_rollout_mode"
	organizationSettingStrictRolloutEnforceAt           = "strict_rollout_enforce_at"
	organizationOutboundModeInboundOnly                 = "inbound_only"
	organizationOutboundModeMixed                       = "mixed"
	organizationStrictRolloutModeAudit                  = "audit"
	organizationStrictRolloutModeEnforce                = "enforce"
	userSettingSendRestrictions                         = "send_restrictions"
)

// restrictedSendViolationError is returned when a user is blocked by strict send restrictions.
type restrictedSendViolationError struct {
	message    string
	reasonCode string
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

func asRestrictedSendViolationWithReason(err error) (string, string, bool) {
	if err == nil {
		return "", "", false
	}
	var violation *restrictedSendViolationError
	if !errors.As(err, &violation) {
		return "", "", false
	}
	return violation.Error(), strings.TrimSpace(violation.reasonCode), true
}

type organizationStrictPolicySettings struct {
	StrictEnabled      bool
	OutboundMode       string
	ApplyToSystem      bool
	CampaignDraftOnly  bool
	StrictRolloutMode  string
	StrictRolloutAfter *time.Time
}

type sendRestrictionsSettings struct {
	Enabled                bool
	IncludeAllContacts     bool
	AuthorizedNumbers      []string
	AllowedInstanceID      *uuid.UUID
	AllowedInstanceIDs     []uuid.UUID
	PrefixAgentName        bool
	AllowUnclaimedChatView bool
	AllowUnclaimedChatSend bool
}

func readSendRestrictionsSettings(settings models.JSONB) sendRestrictionsSettings {
	cfg := sendRestrictionsSettings{
		Enabled:                false,
		IncludeAllContacts:     false,
		AuthorizedNumbers:      []string{},
		AllowedInstanceID:      nil,
		AllowedInstanceIDs:     []uuid.UUID{},
		PrefixAgentName:        true,
		AllowUnclaimedChatView: false,
		AllowUnclaimedChatSend: false,
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
	if allowUnclaimedChatView, ok := payload["allow_unclaimed_chat_view"].(bool); ok {
		cfg.AllowUnclaimedChatView = allowUnclaimedChatView
	}
	if allowUnclaimedChatSend, ok := payload["allow_unclaimed_chat_send"].(bool); ok {
		cfg.AllowUnclaimedChatSend = allowUnclaimedChatSend
	}

	if rawNumbers, ok := payload["authorized_numbers"]; ok {
		cfg.AuthorizedNumbers = normalizeRestrictedNumbers(chat_close_ratings.AsStringSlice(rawNumbers))
	}
	cfg.AllowedInstanceIDs = normalizeRestrictedUUIDs(parseUUIDSlice(payload["allowed_instance_ids"]))
	if len(cfg.AllowedInstanceIDs) == 0 {
		if legacy := parseOptionalUUID(payload["allowed_instance_id"]); legacy != nil {
			cfg.AllowedInstanceIDs = []uuid.UUID{*legacy}
		}
	}
	cfg.AllowedInstanceID = firstRestrictedUUID(cfg.AllowedInstanceIDs)
	if cfg.AllowUnclaimedChatSend && !cfg.AllowUnclaimedChatView {
		cfg.AllowUnclaimedChatView = true
	}

	return cfg
}

func writeSendRestrictionsSettings(settings models.JSONB, cfg sendRestrictionsSettings) models.JSONB {
	if settings == nil {
		settings = models.JSONB{}
	}
	allowedInstanceIDs := normalizeRestrictedUUIDs(cfg.AllowedInstanceIDs)
	if len(allowedInstanceIDs) == 0 && cfg.AllowedInstanceID != nil {
		allowedInstanceIDs = []uuid.UUID{*cfg.AllowedInstanceID}
	}
	restrictions := models.JSONB{
		"enabled":                   cfg.Enabled,
		"include_all_contacts":      cfg.IncludeAllContacts,
		"authorized_numbers":        normalizeRestrictedNumbers(cfg.AuthorizedNumbers),
		"allowed_instance_ids":      stringifyUUIDs(allowedInstanceIDs),
		"prefix_agent_name":         cfg.PrefixAgentName,
		"allow_unclaimed_chat_view": cfg.AllowUnclaimedChatView,
		"allow_unclaimed_chat_send": cfg.AllowUnclaimedChatSend,
	}
	if len(allowedInstanceIDs) > 0 {
		restrictions["allowed_instance_id"] = allowedInstanceIDs[0].String()
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

func stringifyUUIDs(values []uuid.UUID) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		out = append(out, value.String())
	}
	return out
}

func parseUUIDSlice(raw interface{}) []uuid.UUID {
	switch typed := raw.(type) {
	case []uuid.UUID:
		return append([]uuid.UUID{}, typed...)
	case []string:
		out := make([]uuid.UUID, 0, len(typed))
		for _, item := range typed {
			if parsed := parseOptionalUUID(item); parsed != nil {
				out = append(out, *parsed)
			}
		}
		return out
	case models.StringArray:
		out := make([]uuid.UUID, 0, len(typed))
		for _, item := range typed {
			if parsed := parseOptionalUUID(item); parsed != nil {
				out = append(out, *parsed)
			}
		}
		return out
	case []interface{}:
		out := make([]uuid.UUID, 0, len(typed))
		for _, item := range typed {
			if parsed := parseOptionalUUID(item); parsed != nil {
				out = append(out, *parsed)
			}
		}
		return out
	default:
		return nil
	}
}

func normalizeRestrictedUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return []uuid.UUID{}
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	normalized := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func firstRestrictedUUID(values []uuid.UUID) *uuid.UUID {
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		v := value
		return &v
	}
	return nil
}

func containsRestrictedUUID(values []uuid.UUID, needle uuid.UUID) bool {
	if needle == uuid.Nil {
		return false
	}
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func allowedInstanceIDsForRestrictions(cfg sendRestrictionsSettings) []uuid.UUID {
	ids := normalizeRestrictedUUIDs(cfg.AllowedInstanceIDs)
	if len(ids) > 0 {
		return ids
	}
	if cfg.AllowedInstanceID != nil && *cfg.AllowedInstanceID != uuid.Nil {
		return []uuid.UUID{*cfg.AllowedInstanceID}
	}
	return []uuid.UUID{}
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

func parseOrganizationTimeSetting(settings models.JSONB, key string) *time.Time {
	if settings == nil {
		return nil
	}
	raw := settings[key]
	switch typed := raw.(type) {
	case time.Time:
		ts := typed.UTC()
		return &ts
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, text); err == nil {
				ts := parsed.UTC()
				return &ts
			}
		}
	case []byte:
		return parseOrganizationTimeSetting(models.JSONB{key: string(typed)}, key)
	}
	return nil
}

func normalizeOutboundMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case organizationOutboundModeMixed:
		return organizationOutboundModeMixed
	default:
		return organizationOutboundModeInboundOnly
	}
}

func normalizeRolloutMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case organizationStrictRolloutModeAudit:
		return organizationStrictRolloutModeAudit
	default:
		return organizationStrictRolloutModeEnforce
	}
}

func (a *App) loadOrganizationStrictPolicySettings(orgID uuid.UUID) organizationStrictPolicySettings {
	defaults := organizationStrictPolicySettings{
		StrictEnabled:      false,
		OutboundMode:       organizationOutboundModeMixed,
		ApplyToSystem:      true,
		CampaignDraftOnly:  false,
		StrictRolloutMode:  organizationStrictRolloutModeEnforce,
		StrictRolloutAfter: nil,
	}

	if a == nil || a.DB == nil || orgID == uuid.Nil {
		return defaults
	}

	var org models.Organization
	if err := a.DB.Select("settings").Where("id = ?", orgID).First(&org).Error; err != nil {
		return defaults
	}

	if org.Settings == nil {
		return defaults
	}

	settings := defaults
	settings.StrictEnabled = org.Settings.Bool(organizationSettingStrictSendingRestrictionsEnabled, false)
	settings.OutboundMode = normalizeOutboundMode(org.Settings.String(organizationSettingOutboundMode, defaults.OutboundMode))
	settings.ApplyToSystem = org.Settings.Bool(organizationSettingStrictSendingApplyToSystem, true)
	settings.CampaignDraftOnly = org.Settings.Bool(organizationSettingCampaignDraftOnly, false)
	settings.StrictRolloutMode = normalizeRolloutMode(org.Settings.String(organizationSettingStrictRolloutMode, defaults.StrictRolloutMode))
	settings.StrictRolloutAfter = parseOrganizationTimeSetting(org.Settings, organizationSettingStrictRolloutEnforceAt)

	return settings
}

func (s organizationStrictPolicySettings) shouldEnforceStrictPolicy(now time.Time) bool {
	if normalizeRolloutMode(s.StrictRolloutMode) == organizationStrictRolloutModeEnforce {
		return true
	}
	if s.StrictRolloutAfter == nil {
		return false
	}
	return !now.UTC().Before(s.StrictRolloutAfter.UTC())
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

func (a *App) getRestrictedInstancesForUser(orgID, userID uuid.UUID) ([]uuid.UUID, error) {
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
	allowedInstanceIDs := allowedInstanceIDsForRestrictions(cfg)
	if len(allowedInstanceIDs) == 0 {
		return nil, nil
	}

	return allowedInstanceIDs, nil
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
	if a == nil || a.DB == nil || req.Contact == nil {
		return nil
	}

	if isGroupContact(req.Contact) || isChannelContact(req.Contact) {
		return nil
	}

	orgID := req.Contact.OrganizationID
	if orgID == uuid.Nil && req.Account != nil {
		orgID = req.Account.OrganizationID
	}
	if orgID == uuid.Nil {
		return nil
	}

	policy := a.loadOrganizationStrictPolicySettings(orgID)
	if !policy.StrictEnabled {
		return nil
	}
	if normalizeOutboundMode(policy.OutboundMode) != organizationOutboundModeInboundOnly {
		return nil
	}

	actorType := opts.resolvedActorType()
	isUserSend := actorType == MessageActorUser && opts.SentByUserID != nil
	if !isUserSend && !policy.ApplyToSystem {
		return nil
	}

	shouldEnforce := policy.shouldEnforceStrictPolicy(time.Now().UTC())

	var (
		user *models.User
		cfg  sendRestrictionsSettings
		err  error
	)
	if isUserSend {
		user, err = a.loadUserForSendRestrictions(orgID, *opts.SentByUserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("failed to load user send restrictions: %w", err)
		}

		cfg = readSendRestrictionsSettings(user.Settings)
		if !cfg.Enabled {
			return nil
		}

		cfg, err = a.syncUserRestrictionsWithSources(orgID, user, cfg)
		if err != nil {
			return fmt.Errorf("failed to sync authorized numbers: %w", err)
		}
	}

	targetPhone := strings.TrimSpace(req.Contact.PhoneNumber)
	if canonical := a.resolveDirectRecipientFromConversation(ctx, req.Contact); canonical != "" {
		targetPhone = canonical
	}
	if a.isWhatsmeowProvider() && isUserSend {
		allowedInstanceIDs := allowedInstanceIDsForRestrictions(cfg)
		allowAssignedInstanceBypass := isContactAssignedToUser(req.Contact, *opts.SentByUserID)
		if len(allowedInstanceIDs) == 0 {
			if !allowAssignedInstanceBypass {
				if !shouldEnforce {
					return nil
				}
				return &restrictedSendViolationError{
					message:    "Message blocked by strict sending restrictions. Your user must be assigned to a WhatsApp instance.",
					reasonCode: ReasonCodePolicyNoInstance,
				}
			}
		} else {
			outgoingInstanceID := resolveOutgoingInstanceID(req)
			if outgoingInstanceID == nil || !containsRestrictedUUID(allowedInstanceIDs, *outgoingInstanceID) {
				if !allowAssignedInstanceBypass {
					if !shouldEnforce {
						return nil
					}
					return &restrictedSendViolationError{
						message:    "Message blocked by strict sending restrictions. You can only send and receive chats on your assigned WhatsApp instances.",
						reasonCode: ReasonCodePolicyNoInstance,
					}
				}
			}
		}
	}
	targetNumber := normalizeRestrictedPhoneNumber(targetPhone)
	if targetNumber == "" {
		if !shouldEnforce {
			return nil
		}
		return &restrictedSendViolationError{
			message:    "Message blocked by strict sending restrictions. This chat does not map to a valid phone number.",
			reasonCode: ReasonCodePolicyNoInbound,
		}
	}

	if isUserSend && containsRestrictedNumber(cfg.AuthorizedNumbers, targetNumber) {
		return nil
	}

	hasIncomingHistory, err := req.Contact.HasIncomingHistory(a.DB)
	if err != nil {
		return fmt.Errorf("failed to validate incoming history: %w", err)
	}
	if hasIncomingHistory {
		if user != nil {
			cfg.AuthorizedNumbers, _ = mergeRestrictedNumbers(cfg.AuthorizedNumbers, []string{targetNumber})
			if err := a.saveUserSendRestrictions(user.ID, user.Settings, cfg); err != nil {
				return fmt.Errorf("failed to persist authorized number: %w", err)
			}
		}
		return nil
	}

	if !shouldEnforce {
		return nil
	}
	return &restrictedSendViolationError{
		message:    "Message blocked by strict sending restrictions. You can only send to phone numbers that have previously sent incoming messages in your assigned chats.",
		reasonCode: ReasonCodePolicyNoInbound,
	}
}
