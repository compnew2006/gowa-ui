package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contactChatType string

const (
	contactChatTypePrivate contactChatType = "private"
	contactChatTypeGroup   contactChatType = "group"
	contactChatTypeChannel contactChatType = "channel"
)

func parseContactChatTypes(raw string) ([]contactChatType, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	chatTypes := make([]contactChatType, 0, len(parts))
	seen := make(map[contactChatType]bool, len(parts))

	for _, part := range parts {
		value := contactChatType(strings.ToLower(strings.TrimSpace(part)))
		if value == "" {
			continue
		}

		switch value {
		case contactChatTypePrivate, contactChatTypeGroup, contactChatTypeChannel:
		default:
			return nil, fmt.Errorf("invalid chat type: %s", part)
		}

		if !seen[value] {
			chatTypes = append(chatTypes, value)
			seen[value] = true
		}
	}

	if len(chatTypes) == 0 {
		return nil, nil
	}

	return chatTypes, nil
}

func groupChatFilterCondition() (string, []any) {
	condition := `(contacts.phone_number LIKE ? OR contacts.metadata->>'is_group_chat' = 'true' OR EXISTS (` +
		`SELECT 1 FROM messages msg ` +
		`WHERE msg.contact_id = contacts.id ` +
		`AND msg.organization_id = contacts.organization_id ` +
		`AND (msg.conversation_id LIKE ? OR msg.metadata->>'is_group_chat' = 'true' OR msg.metadata->>'is_group' = 'true')` +
		`))`
	return condition, []any{"%@g.us", "%@g.us"}
}

func channelChatFilterCondition() (string, []any) {
	condition := `(contacts.phone_number LIKE ? OR contacts.metadata->>'is_channel_chat' = 'true' OR EXISTS (` +
		`SELECT 1 FROM messages msg ` +
		`WHERE msg.contact_id = contacts.id ` +
		`AND msg.organization_id = contacts.organization_id ` +
		`AND (msg.conversation_id LIKE ? OR msg.metadata->>'is_channel_chat' = 'true' OR msg.metadata->>'is_channel' = 'true')` +
		`))`
	return condition, []any{"%@newsletter", "%@newsletter"}
}

func privateChatFilterCondition() (string, []any) {
	contactLevelPrivateCondition := `(contacts.phone_number NOT LIKE ? AND contacts.phone_number NOT LIKE ? ` +
		`AND COALESCE(contacts.metadata->>'is_group_chat', 'false') != 'true' ` +
		`AND COALESCE(contacts.metadata->>'is_channel_chat', 'false') != 'true')`
	contactLevelPrivateArgs := []any{"%@g.us", "%@newsletter"}

	directMessageCondition := `(EXISTS (` +
		`SELECT 1 FROM messages msg ` +
		`WHERE msg.contact_id = contacts.id ` +
		`AND msg.organization_id = contacts.organization_id ` +
		`AND (msg.conversation_id IS NULL OR msg.conversation_id = '' OR (msg.conversation_id NOT LIKE ? AND msg.conversation_id NOT LIKE ?)) ` +
		`AND COALESCE(msg.metadata->>'is_group_chat', 'false') != 'true' ` +
		`AND COALESCE(msg.metadata->>'is_group', 'false') != 'true' ` +
		`AND COALESCE(msg.metadata->>'is_channel_chat', 'false') != 'true' ` +
		`AND COALESCE(msg.metadata->>'is_channel', 'false') != 'true'` +
		`))`
	directMessageArgs := []any{"%@g.us", "%@newsletter"}

	// A contact is private if it looks private at contact level OR it has at least one direct message.
	condition := fmt.Sprintf("((%s) OR %s)", contactLevelPrivateCondition, directMessageCondition)
	return condition, append(contactLevelPrivateArgs, directMessageArgs...)
}

func contactChatTypeCondition(chatType contactChatType) (string, []any, error) {
	switch chatType {
	case contactChatTypeGroup:
		condition, args := groupChatFilterCondition()
		return condition, args, nil
	case contactChatTypeChannel:
		condition, args := channelChatFilterCondition()
		return condition, args, nil
	case contactChatTypePrivate:
		condition, args := privateChatFilterCondition()
		return condition, args, nil
	default:
		return "", nil, errors.New("invalid chat type")
	}
}

func applyContactChatTypeFilters(query *gorm.DB, chatTypes []contactChatType) (*gorm.DB, error) {
	if len(chatTypes) == 0 {
		return query, nil
	}

	conditions := make([]string, 0, len(chatTypes))
	args := make([]any, 0, len(chatTypes)*4)

	for _, chatType := range chatTypes {
		condition, conditionArgs, err := contactChatTypeCondition(chatType)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
		args = append(args, conditionArgs...)
	}

	return query.Where("("+strings.Join(conditions, " OR ")+")", args...), nil
}

func (a *App) resolveContactInstanceID(orgID uuid.UUID, rawInstanceID string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(rawInstanceID)
	if trimmed == "" {
		return nil, nil
	}

	instanceID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid instance_id")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to validate instance_id")
	}

	return &instanceID, nil
}

func (a *App) canDeleteAnyChat(userID, orgID uuid.UUID) bool {
	return a.HasPermission(userID, models.ResourceContacts, models.ActionDelete, orgID)
}
