package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	waTypes "go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
)

// ContactResponse represents a contact with additional fields for the frontend
type ContactResponse struct {
	ID                 uuid.UUID  `json:"id"`
	InstanceID         *string    `json:"instance_id,omitempty"`
	ConversationID     string     `json:"conversation_id,omitempty"`
	IsGroupChat        bool       `json:"is_group_chat,omitempty"`
	PhoneNumber        string     `json:"phone_number"`
	Name               string     `json:"name"`
	ProfileName        string     `json:"profile_name"`
	AvatarURL          string     `json:"avatar_url"`
	Status             string     `json:"status"`
	Tags               []string   `json:"tags"`
	Metadata           any        `json:"metadata"`
	LastMessageAt      *time.Time `json:"last_message_at"`
	LastMessagePreview string     `json:"last_message_preview"`
	UnreadCount        int        `json:"unread_count"`
	AssignedUserID     *uuid.UUID `json:"assigned_user_id,omitempty"`
	AssignedUserName   string     `json:"assigned_user_name,omitempty"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
	ClosedByUserID     *uuid.UUID `json:"closed_by_user_id,omitempty"`
	ClosedByName       string     `json:"closed_by_name,omitempty"`
	WhatsAppAccount    string     `json:"whatsapp_account,omitempty"`
	LastInboundAt      *time.Time `json:"last_inbound_at,omitempty"`
	ServiceWindowOpen  bool       `json:"service_window_open"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// MessageResponse represents a message for the frontend
type MessageResponse struct {
	ID               uuid.UUID            `json:"id"`
	ContactID        uuid.UUID            `json:"contact_id"`
	InstanceID       *string              `json:"instance_id,omitempty"`
	ConversationID   string               `json:"conversation_id,omitempty"`
	IsGroupChat      bool                 `json:"is_group_chat,omitempty"`
	SenderPhone      string               `json:"sender_phone,omitempty"`
	SenderPushName   string               `json:"sender_push_name,omitempty"`
	Direction        models.Direction     `json:"direction"`
	MessageType      models.MessageType   `json:"message_type"`
	Content          any                  `json:"content"`
	MediaURL         string               `json:"media_url,omitempty"`
	MediaMimeType    string               `json:"media_mime_type,omitempty"`
	MediaFilename    string               `json:"media_filename,omitempty"`
	InteractiveData  models.JSONB         `json:"interactive_data,omitempty"`
	Status           models.MessageStatus `json:"status"`
	WAMID            string               `json:"wamid"`
	Error            string               `json:"error_message"`
	IsReply          bool                 `json:"is_reply"`
	ReplyToMessageID *string              `json:"reply_to_message_id,omitempty"`
	ReplyToMessage   *ReplyPreview        `json:"reply_to_message,omitempty"`
	Reactions        []ReactionInfo       `json:"reactions,omitempty"`
	WhatsAppAccount  string               `json:"whatsapp_account,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// ReplyPreview contains a preview of the replied-to message
type ReplyPreview struct {
	ID          string             `json:"id"`
	Content     any                `json:"content"`
	MessageType models.MessageType `json:"message_type"`
	Direction   models.Direction   `json:"direction"`
	SenderPhone string             `json:"sender_phone,omitempty"`
}

// ReactionInfo represents a reaction on a message
type ReactionInfo struct {
	Emoji     string `json:"emoji"`
	FromPhone string `json:"from_phone,omitempty"`
	FromUser  string `json:"from_user,omitempty"`
}

const (
	unsupportedMessageBody = "[Unsupported message type]"
	deletedMessageBody     = "(This message was deleted)"
	legacyDeletedBody      = "This message was deleted"
)

func normalizeDeletedMessageBody(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.EqualFold(trimmed, legacyDeletedBody) {
		return deletedMessageBody
	}
	return content
}

func appendDeletedMessageCaption(content string) string {
	trimmed := strings.TrimSpace(content)
	switch trimmed {
	case "":
		return deletedMessageBody
	case deletedMessageBody, legacyDeletedBody:
		return deletedMessageBody
	case unsupportedMessageBody:
		return deletedMessageBody
	}

	if strings.Contains(content, deletedMessageBody) {
		return content
	}

	return strings.TrimRight(content, "\n") + "\n" + deletedMessageBody
}

func messageMetadataBool(metadata models.JSONB, key string) bool {
	if metadata == nil {
		return false
	}
	value, ok := metadata[key]
	if !ok {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}

func isPlaceholderMessageBody(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == unsupportedMessageBody ||
		trimmed == deletedMessageBody ||
		strings.EqualFold(trimmed, legacyDeletedBody)
}

func isPlaceholderTextMessage(message models.Message) bool {
	return message.MessageType == models.MessageTypeText && isPlaceholderMessageBody(message.Content)
}

func isSyntheticPlaceholderMessage(message models.Message, hasCompanionByWAMID map[string]bool) bool {
	if !isPlaceholderTextMessage(message) || messageMetadataBool(message.Metadata, "revoked") {
		return false
	}

	wamid := strings.TrimSpace(message.WhatsAppMessageID)
	if wamid == "" {
		return false
	}

	return hasCompanionByWAMID[wamid]
}

func buildConversationScopeQuery(db *gorm.DB, orgID uuid.UUID, conversationID string, instanceID *uuid.UUID) *gorm.DB {
	query := db.Where("organization_id = ? AND conversation_id = ?", orgID, conversationID)
	if instanceID != nil {
		query = query.Where("instance_id = ?", *instanceID)
	}
	return query
}

func stringifyInstanceID(instanceID *uuid.UUID) *string {
	if instanceID == nil {
		return nil
	}
	value := instanceID.String()
	return &value
}

func cloneJSONB(metadata models.JSONB) models.JSONB {
	if metadata == nil {
		return models.JSONB{}
	}
	cloned := make(models.JSONB, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func contactAvatarURL(metadata models.JSONB) string {
	if metadata == nil {
		return ""
	}
	raw, ok := metadata["avatar_url"]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func (a *App) scheduleContactAvatarRefresh(contact *models.Contact) {
	if a == nil || a.WhatsmeowManager == nil || contact == nil || contact.InstanceID == nil {
		return
	}
	a.WhatsmeowManager.ScheduleContactAvatarRefresh(*contact.InstanceID, contact)
}

func (a *App) resolveChannelNameFromWhatsmeow(contact models.Contact, conversationID string) string {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || a.WhatsmeowManager == nil || contact.InstanceID == nil {
		return ""
	}

	client := a.WhatsmeowManager.GetClient(*contact.InstanceID)
	if client == nil {
		return ""
	}

	channelJID, err := waTypes.ParseJID(conversationID)
	if err != nil && !strings.Contains(conversationID, "@") {
		channelJID, err = waTypes.ParseJID(conversationID + newsletterJIDSuffix)
	}
	if err != nil || channelJID.Server != waTypes.NewsletterServer {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	newsletterInfo, err := client.GetNewsletterInfo(ctx, channelJID)
	if err != nil || newsletterInfo == nil {
		return ""
	}

	channelName := strings.TrimSpace(newsletterInfo.ThreadMeta.Name.Text)
	if channelName == "" {
		return ""
	}

	metadata := cloneJSONB(contact.Metadata)
	metadata["is_channel_chat"] = true
	metadata["channel_jid"] = channelJID.String()
	metadata["channel_name"] = channelName

	if err := a.DB.Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Updates(map[string]any{
			"profile_name": channelName,
			"metadata":     metadata,
		}).Error; err != nil {
		a.Log.Warn("Failed to persist resolved channel name", "contact_id", contact.ID, "error", err)
	}

	return channelName
}

type contactConversationContext struct {
	ConversationID string
	IsGroupChat    bool
	IsChannelChat  bool
	DisplayName    string
}

func (a *App) resolveContactConversationContext(orgID uuid.UUID, contact models.Contact) contactConversationContext {
	if isGroupContact(&contact) {
		conversationID := strings.TrimSpace(contact.PhoneNumber)
		if conversationID == "" {
			conversationID = messageMetadataString(contact.Metadata, "group_jid")
		}
		groupName := messageMetadataString(contact.Metadata, "group_name")
		if groupName == "" {
			groupName = strings.TrimSpace(contact.ProfileName)
		}
		return contactConversationContext{
			ConversationID: conversationID,
			IsGroupChat:    conversationID != "",
			DisplayName:    groupName,
		}
	}

	if isChannelContact(&contact) {
		conversationID := messageMetadataString(contact.Metadata, "channel_jid")
		if conversationID == "" {
			conversationID = strings.TrimSpace(contact.PhoneNumber)
		}
		channelName := messageMetadataString(contact.Metadata, "channel_name")
		if channelName == "" {
			channelName = strings.TrimSpace(contact.ProfileName)
		}
		if channelName == "" || channelName == strings.TrimSpace(contact.PhoneNumber) {
			if resolvedName := a.resolveChannelNameFromWhatsmeow(contact, conversationID); resolvedName != "" {
				channelName = resolvedName
			}
		}
		return contactConversationContext{
			ConversationID: conversationID,
			IsChannelChat:  conversationID != "" || channelName != "",
			DisplayName:    channelName,
		}
	}

	var latestMessage models.Message
	if err := a.DB.WithContext(context.Background()).
		Select("conversation_id", "metadata").
		Where("organization_id = ? AND contact_id = ?", orgID, contact.ID).
		Order("created_at DESC").
		First(&latestMessage).Error; err != nil {
		return contactConversationContext{}
	}

	if isGroupMessage(latestMessage) {
		conversationID := strings.TrimSpace(latestMessage.ConversationID)
		if conversationID == "" {
			conversationID = messageMetadataString(latestMessage.Metadata, "group_jid")
		}
		displayName := messageMetadataString(latestMessage.Metadata, "group_name")
		if displayName == "" {
			displayName = strings.TrimSpace(contact.ProfileName)
		}
		return contactConversationContext{
			ConversationID: conversationID,
			IsGroupChat:    conversationID != "",
			DisplayName:    displayName,
		}
	}

	if isChannelMessage(latestMessage) {
		conversationID := strings.TrimSpace(latestMessage.ConversationID)
		if conversationID == "" {
			conversationID = messageMetadataString(latestMessage.Metadata, "channel_jid")
		}
		displayName := messageMetadataString(latestMessage.Metadata, "channel_name")
		if displayName == "" {
			displayName = messageMetadataString(contact.Metadata, "channel_name")
		}
		if displayName == "" {
			displayName = strings.TrimSpace(contact.ProfileName)
		}
		if displayName == "" || displayName == strings.TrimSpace(contact.PhoneNumber) {
			if resolvedName := a.resolveChannelNameFromWhatsmeow(contact, conversationID); resolvedName != "" {
				displayName = resolvedName
			}
		}
		return contactConversationContext{
			ConversationID: conversationID,
			IsChannelChat:  conversationID != "" || displayName != "",
			DisplayName:    displayName,
		}
	}

	conversationID := strings.TrimSpace(latestMessage.ConversationID)
	if conversationID != "" {
		return contactConversationContext{
			ConversationID: conversationID,
		}
	}

	return contactConversationContext{}
}

// repairDirectContactPhoneFromConversation keeps private-chat contacts on canonical PN identity.
// It uses the latest direct conversation JID (`<pn>@s.whatsapp.net`) as source of truth.
func (a *App) repairDirectContactPhoneFromConversation(contact *models.Contact, conversationID string) {
	if a == nil || contact == nil {
		return
	}
	if isGroupContact(contact) || isChannelContact(contact) {
		return
	}

	canonicalPhone := strings.TrimSpace(directUserFromConversationID(conversationID))
	if canonicalPhone == "" || canonicalPhone == strings.TrimSpace(contact.PhoneNumber) {
		return
	}

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		target := models.Contact{}
		targetQuery := tx.Where("organization_id = ? AND phone_number = ?", contact.OrganizationID, canonicalPhone)
		if contact.InstanceID != nil {
			targetQuery = targetQuery.Where("instance_id = ?", *contact.InstanceID)
		} else {
			targetQuery = targetQuery.Where("instance_id IS NULL")
		}

		if err := targetQuery.First(&target).Error; err == nil {
			if target.ID == contact.ID {
				contact.PhoneNumber = canonicalPhone
				return nil
			}

			if err := tx.Model(&models.Message{}).
				Where("organization_id = ? AND contact_id = ?", contact.OrganizationID, contact.ID).
				Update("contact_id", target.ID).Error; err != nil {
				return err
			}

			profileName := strings.TrimSpace(contact.ProfileName)
			if (target.ProfileName == "" || target.ProfileName == target.PhoneNumber) && profileName != "" && profileName != canonicalPhone {
				if err := tx.Model(&models.Contact{}).Where("id = ?", target.ID).Update("profile_name", profileName).Error; err != nil {
					return err
				}
				target.ProfileName = profileName
			}

			if err := tx.Delete(&models.Contact{}, "id = ?", contact.ID).Error; err != nil {
				return err
			}

			*contact = target
			contact.PhoneNumber = canonicalPhone
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&models.Contact{}).Where("id = ?", contact.ID).Update("phone_number", canonicalPhone).Error; err != nil {
			return err
		}
		contact.PhoneNumber = canonicalPhone
		return nil
	})
	if err != nil {
		a.Log.Warn("Failed to repair direct contact phone", "contact_id", contact.ID, "conversation_id", conversationID, "canonical_phone", canonicalPhone, "error", err)
	}
}

// ListContacts returns all contacts for the organization
// Users without contacts:read permission see pending queue + contacts assigned to them
func (a *App) ListContacts(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Pagination
	pg := parsePaginationWithDefaults(r, 50, 500)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))
	createdFromParam := string(r.RequestCtx.QueryArgs().Peek("created_from"))
	createdToParam := string(r.RequestCtx.QueryArgs().Peek("created_to"))
	tagsParam := string(r.RequestCtx.QueryArgs().Peek("tags"))
	instanceIDParam := string(r.RequestCtx.QueryArgs().Peek("instance_id"))
	chatTypesParam := string(r.RequestCtx.QueryArgs().Peek("chat_types"))
	statusParam := string(r.RequestCtx.QueryArgs().Peek("status"))
	assignedToParam := string(r.RequestCtx.QueryArgs().Peek("assigned_to"))

	var contacts []models.Contact
	query := a.ScopeToOrg(a.DB, userID, orgID)
	statusFilter, parseStatusErr := parseChatStatusFilter(statusParam)
	if parseStatusErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseStatusErr.Error(), nil, "status")
	}
	assignedToUserID, hasAssignedToFilter, parseAssignedToErr := parseAssignedToFilter(assignedToParam, userID)
	if parseAssignedToErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseAssignedToErr.Error(), nil, "assigned_to")
	}

	hasContactsReadPermission := a.canReadAllContacts(userID, orgID)
	// Users without contacts:read can still see pending queue + their assigned chats.
	if !hasContactsReadPermission {
		query = query.Where(
			"assigned_user_id = ? OR ((status IS NULL OR status = '' OR status = ?) AND assigned_user_id IS NULL)",
			userID,
			models.ChatStatusPending,
		)
	}

	restrictedInstanceID, err := a.getRestrictedInstanceForUser(orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for contact list", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}
	if restrictedInstanceID != nil {
		query = query.Where("instance_id = ?", *restrictedInstanceID)
	}

	if statusFilter != nil {
		query = applyChatStatusFilter(query, *statusFilter)
	} else {
		query = applyDefaultActiveChatFilter(query)
	}

	if hasAssignedToFilter {
		if assignedToUserID == nil {
			query = query.Where("assigned_user_id IS NULL")
		} else {
			query = query.Where("assigned_user_id = ?", *assignedToUserID)
		}
	}

	if createdFromParam != "" {
		createdFrom, parseErr := time.Parse("2006-01-02", createdFromParam)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid created_from format. Use YYYY-MM-DD", nil, "created_from")
		}
		query = query.Where("created_at >= ?", createdFrom)
	}
	if createdToParam != "" {
		createdTo, parseErr := time.Parse("2006-01-02", createdToParam)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid created_to format. Use YYYY-MM-DD", nil, "created_to")
		}
		query = query.Where("created_at <= ?", endOfDay(createdTo))
	}

	if search != "" {
		// Limit search string length to prevent abuse
		if len(search) > 1000 {
			search = search[:1000]
		}
		searchPattern := "%" + search + "%"
		// Use ILIKE for case-insensitive search on profile_name
		query = query.Where("phone_number LIKE ? OR profile_name ILIKE ?", searchPattern, searchPattern)
	}

	if instanceIDParam != "" {
		instanceID, resolveErr := a.resolveContactInstanceID(orgID, instanceIDParam)
		if resolveErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		if instanceID != nil {
			query = query.Where("instance_id = ?", *instanceID)
		}
	}

	chatTypes, parseErr := parseContactChatTypes(chatTypesParam)
	if parseErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseErr.Error(), nil, "chat_types")
	}
	query, err = applyContactChatTypeFilters(query, chatTypes)
	if err != nil {
		a.Log.Error("Failed to apply chat type filters", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid chat_types filter", nil, "chat_types")
	}

	// Filter by tags (comma-separated, matches contacts that have ANY of the specified tags)
	if tagsParam != "" {
		tagList := strings.Split(tagsParam, ",")
		// Trim whitespace from each tag and build OR conditions
		// Using @> operator which leverages the GIN index on tags
		conditions := make([]string, 0, len(tagList))
		args := make([]any, 0, len(tagList))
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				// Use proper JSONB containment with explicit cast
				conditions = append(conditions, "tags @> ?::jsonb")
				tagJSON, _ := json.Marshal([]string{tag})
				args = append(args, string(tagJSON))
			}
		}
		if len(conditions) > 0 {
			query = query.Where("("+strings.Join(conditions, " OR ")+")", args...)
		}
	}

	// Order by last message time (most recent first)
	query = query.Order("last_message_at DESC NULLS LAST, created_at DESC")

	var total int64
	query.Model(&models.Contact{}).Count(&total)

	if err := query.Preload("ClosedByUser").Preload("AssignedUser").Offset(pg.Offset).Limit(pg.Limit).Find(&contacts).Error; err != nil {
		a.Log.Error("Failed to list contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}

	// Check if phone masking is enabled
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)

	// Convert to response format
	response := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		status := normalizeContactStatus(&c)
		conversationContext := a.resolveContactConversationContext(orgID, c)
		a.repairDirectContactPhoneFromConversation(&c, conversationContext.ConversationID)
		a.scheduleContactAvatarRefresh(&c)

		// Count unread messages
		var unreadCount int64
		if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
			buildConversationScopeQuery(a.DB, orgID, conversationContext.ConversationID, c.InstanceID).
				Model(&models.Message{}).
				Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead).
				Count(&unreadCount)
		} else {
			a.DB.Model(&models.Message{}).
				Where("contact_id = ? AND direction = ? AND status != ?", c.ID, models.DirectionIncoming, models.MessageStatusRead).
				Count(&unreadCount)
		}

		tags := []string{}
		if c.Tags != nil {
			for _, t := range c.Tags {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		phoneNumber := c.PhoneNumber
		profileName := c.ProfileName
		if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
			phoneNumber = conversationContext.ConversationID
		}
		if conversationContext.DisplayName != "" {
			profileName = conversationContext.DisplayName
		}
		if shouldMask {
			phoneNumber = MaskPhoneNumber(phoneNumber)
			profileName = MaskIfPhoneNumber(profileName)
		}

		closedAt := c.ClosedAt
		closedByUserID := c.ClosedByUserID
		if status == models.ChatStatusClosed {
			if closedAt == nil {
				closedAt = &c.UpdatedAt
			}
			if closedByUserID == nil && c.AssignedUserID != nil {
				closedByUserID = c.AssignedUserID
			}
		}
		serviceWindowOpen := c.LastInboundAt != nil && time.Since(*c.LastInboundAt) < 24*time.Hour

		response[i] = ContactResponse{
			ID:                 c.ID,
			InstanceID:         stringifyInstanceID(c.InstanceID),
			ConversationID:     conversationContext.ConversationID,
			IsGroupChat:        conversationContext.IsGroupChat,
			PhoneNumber:        phoneNumber,
			Name:               profileName,
			ProfileName:        profileName,
			AvatarURL:          contactAvatarURL(c.Metadata),
			Status:             status.String(),
			Tags:               tags,
			Metadata:           c.Metadata,
			LastMessageAt:      c.LastMessageAt,
			LastMessagePreview: c.LastMessagePreview,
			UnreadCount:        int(unreadCount),
			AssignedUserID:     c.AssignedUserID,
			AssignedUserName:   strings.TrimSpace(userFullName(c.AssignedUser)),
			ClosedAt:           closedAt,
			ClosedByUserID:     closedByUserID,
			ClosedByName:       strings.TrimSpace(userFullName(c.ClosedByUser)),
			WhatsAppAccount:    c.WhatsAppAccount,
			LastInboundAt:      c.LastInboundAt,
			ServiceWindowOpen:  serviceWindowOpen,
			CreatedAt:          c.CreatedAt,
			UpdatedAt:          c.UpdatedAt,
		}
	}

	return r.SendEnvelope(map[string]any{
		"contacts": response,
		"total":    total,
		"page":     pg.Page,
		"limit":    pg.Limit,
	})
}

// GetContact returns a single contact
// Users without contacts:read permission can only access contacts assigned to them
func (a *App) GetContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	query := a.DB.Preload("ClosedByUser").Preload("AssignedUser").Where("id = ? AND organization_id = ?", contactID, orgID)

	// Users without contacts:read permission can only access their assigned contacts
	if !a.canReadAllContacts(userID, orgID) {
		query = query.Where("assigned_user_id = ?", userID)
	}
	restrictedInstanceID, restrictedErr := a.getRestrictedInstanceForUser(orgID, userID)
	if restrictedErr != nil {
		a.Log.Error("Failed to resolve restricted instance for contact read", "error", restrictedErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load contact", nil, "")
	}
	if restrictedInstanceID != nil {
		query = query.Where("instance_id = ?", *restrictedInstanceID)
	}

	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	// Count unread messages
	conversationContext := a.resolveContactConversationContext(orgID, contact)
	a.repairDirectContactPhoneFromConversation(&contact, conversationContext.ConversationID)
	a.scheduleContactAvatarRefresh(&contact)
	var unreadCount int64
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		buildConversationScopeQuery(a.DB, orgID, conversationContext.ConversationID, contact.InstanceID).
			Model(&models.Message{}).
			Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)
	} else {
		a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?", contact.ID, models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)
	}

	tags := []string{}
	if contact.Tags != nil {
		for _, t := range contact.Tags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	phoneNumber := contact.PhoneNumber
	profileName := contact.ProfileName
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		phoneNumber = conversationContext.ConversationID
	}
	if conversationContext.DisplayName != "" {
		profileName = conversationContext.DisplayName
	}
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	if shouldMask {
		phoneNumber = MaskPhoneNumber(phoneNumber)
		profileName = MaskIfPhoneNumber(profileName)
	}
	status := normalizeContactStatus(&contact)
	closedAt := contact.ClosedAt
	closedByUserID := contact.ClosedByUserID
	if status == models.ChatStatusClosed {
		if closedAt == nil {
			closedAt = &contact.UpdatedAt
		}
		if closedByUserID == nil && contact.AssignedUserID != nil {
			closedByUserID = contact.AssignedUserID
		}
	}
	serviceWindowOpen := contact.LastInboundAt != nil && time.Since(*contact.LastInboundAt) < 24*time.Hour

	response := ContactResponse{
		ID:                 contact.ID,
		InstanceID:         stringifyInstanceID(contact.InstanceID),
		ConversationID:     conversationContext.ConversationID,
		IsGroupChat:        conversationContext.IsGroupChat,
		PhoneNumber:        phoneNumber,
		Name:               profileName,
		ProfileName:        profileName,
		AvatarURL:          contactAvatarURL(contact.Metadata),
		Status:             status.String(),
		Tags:               tags,
		Metadata:           contact.Metadata,
		LastMessageAt:      contact.LastMessageAt,
		LastMessagePreview: contact.LastMessagePreview,
		UnreadCount:        int(unreadCount),
		AssignedUserID:     contact.AssignedUserID,
		AssignedUserName:   strings.TrimSpace(userFullName(contact.AssignedUser)),
		ClosedAt:           closedAt,
		ClosedByUserID:     closedByUserID,
		ClosedByName:       strings.TrimSpace(userFullName(contact.ClosedByUser)),
		WhatsAppAccount:    contact.WhatsAppAccount,
		LastInboundAt:      contact.LastInboundAt,
		ServiceWindowOpen:  serviceWindowOpen,
		CreatedAt:          contact.CreatedAt,
		UpdatedAt:          contact.UpdatedAt,
	}

	return r.SendEnvelope(response)
}

// GetMessages returns messages for a contact
// Agents can only access messages for their assigned contacts
// Supports cursor-based pagination with before_id for loading older messages
func (a *App) GetMessages(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	hasContactsReadPermission := a.canReadAllContacts(userID, orgID)
	restrictedInstanceID, restrictedErr := a.getRestrictedInstanceForUser(orgID, userID)
	if restrictedErr != nil {
		a.Log.Error("Failed to resolve restricted instance for messages", "error", restrictedErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load messages", nil, "")
	}

	// Verify contact belongs to org (and to user if no contacts:read permission)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !hasContactsReadPermission {
		query = query.Where("assigned_user_id = ?", userID)
	}
	if restrictedInstanceID != nil {
		query = query.Where("instance_id = ?", *restrictedInstanceID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	normalizeContactStatus(&contact)
	if isChatRestrictedForMessageRead(contact) && !a.canBypassPendingChatRestriction(userID, orgID) {
		return r.SendErrorEnvelope(
			fasthttp.StatusForbidden,
			"This chat is currently unassigned. Claim it before viewing messages.",
			nil,
			"",
		)
	}
	shouldMaskPhoneNumbers := a.ShouldMaskPhoneNumbers(orgID)
	conversationContext := a.resolveContactConversationContext(orgID, contact)

	// Pagination parameters
	limit, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	beforeIDStr := string(r.RequestCtx.QueryArgs().Peek("before_id"))

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Build base query
	msgQuery := a.DB.Where("contact_id = ?", contactID)
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		msgQuery = buildConversationScopeQuery(a.DB, orgID, conversationContext.ConversationID, contact.InstanceID)
	}

	// Filter by WhatsApp account if specified
	accountFilter := string(r.RequestCtx.QueryArgs().Peek("account"))
	if accountFilter != "" {
		msgQuery = msgQuery.Where("whats_app_account = ?", accountFilter)
	}

	// Check if user without contacts:read should only see current conversation
	if !hasContactsReadPermission {
		settings, err := a.getChatbotSettingsCached(orgID, "")
		if err == nil {
			if settings.AgentAssignment.CurrentConversationOnly {
				// Find the most recent session for this contact
				var session models.ChatbotSession
				if err := a.DB.Where("contact_id = ? AND organization_id = ?", contactID, orgID).
					Order("started_at DESC").First(&session).Error; err == nil {
					// Filter messages to only those from this session onwards
					msgQuery = msgQuery.Where("created_at >= ?", session.StartedAt)
				}
			}
		}
	}

	// Count total messages (with session filter if applied)
	var total int64
	msgQuery.Model(&models.Message{}).Count(&total)

	// Cursor-based pagination: load messages before a specific ID
	if beforeIDStr != "" {
		beforeID, err := uuid.Parse(beforeIDStr)
		if err == nil {
			// Get the created_at of the before_id message
			var beforeMsg models.Message
			if err := a.DB.Where("id = ?", beforeID).First(&beforeMsg).Error; err == nil {
				msgQuery = msgQuery.Where("created_at < ?", beforeMsg.CreatedAt)
			}
		}
		// For loading older messages, order DESC and limit, then reverse
		var messages []models.Message
		if err := msgQuery.Preload("ReplyToMessage").Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
			a.Log.Error("Failed to list messages", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list messages", nil, "")
		}
		// Reverse to get chronological order
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		response := a.buildMessagesResponse(messages, shouldMaskPhoneNumbers)
		return r.SendEnvelope(map[string]any{
			"messages": response,
			"total":    total,
			"has_more": len(messages) == limit,
		})
	}

	// Default: load most recent messages (page 1)
	page, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	if page < 1 {
		page = 1
	}

	// For chat, we want the most recent messages
	// Calculate offset from the end for pagination
	// Preserve the original limit for the response; adjust a query-specific limit
	// when the remaining messages are fewer than the requested page size.
	responseLimit := limit
	queryLimit := limit
	offset := int(total) - (page * limit)
	if offset < 0 {
		queryLimit = limit + offset // Adjust limit if we're on the last page
		offset = 0
	}

	var messages []models.Message
	if err := msgQuery.Preload("ReplyToMessage").Order("created_at ASC").Offset(offset).Limit(queryLimit).Find(&messages).Error; err != nil {
		a.Log.Error("Failed to list messages", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list messages", nil, "")
	}

	// Mark messages as read
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		a.markGroupMessagesAsRead(orgID, conversationContext.ConversationID, contact.InstanceID)
	} else {
		a.markMessagesAsRead(orgID, contactID, &contact)
	}

	response := a.buildMessagesResponse(messages, shouldMaskPhoneNumbers)
	return r.SendEnvelope(map[string]any{
		"messages": response,
		"total":    total,
		"page":     page,
		"limit":    responseLimit,
		"has_more": offset > 0,
	})
}

// buildMessagesResponse converts messages to response format
func (a *App) buildMessagesResponse(messages []models.Message, shouldMaskPhoneNumbers bool) []MessageResponse {
	hasCompanionByWAMID := make(map[string]bool)
	for _, m := range messages {
		wamid := strings.TrimSpace(m.WhatsAppMessageID)
		if wamid == "" || isPlaceholderTextMessage(m) {
			continue
		}
		hasCompanionByWAMID[wamid] = true
	}

	response := make([]MessageResponse, 0, len(messages))
	for _, m := range messages {
		if isSyntheticPlaceholderMessage(m, hasCompanionByWAMID) {
			continue
		}

		normalizedContent := normalizeDeletedMessageBody(m.Content)
		var content any
		if m.MessageType == models.MessageTypeText {
			content = map[string]string{"body": normalizedContent}
		} else {
			content = map[string]string{"body": normalizedContent}
		}

		senderPhone := extractMessageSenderPhone(m.Metadata)
		if shouldMaskPhoneNumbers && senderPhone != "" {
			senderPhone = MaskPhoneNumber(senderPhone)
		}

		msgResp := MessageResponse{
			ID:              m.ID,
			ContactID:       m.ContactID,
			ConversationID:  m.ConversationID,
			IsGroupChat:     isGroupMessage(m),
			SenderPhone:     senderPhone,
			SenderPushName:  extractMessageSenderPushName(m.Metadata),
			Direction:       m.Direction,
			MessageType:     m.MessageType,
			Content:         content,
			MediaURL:        m.MediaURL,
			MediaMimeType:   m.MediaMimeType,
			MediaFilename:   m.MediaFilename,
			InteractiveData: m.InteractiveData,
			Status:          m.Status,
			WAMID:           m.WhatsAppMessageID,
			Error:           m.ErrorMessage,
			IsReply:         m.IsReply,
			WhatsAppAccount: m.WhatsAppAccount,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}

		if m.InstanceID != nil {
			instanceIDStr := m.InstanceID.String()
			msgResp.InstanceID = &instanceIDStr
		}

		if m.IsReply {
			if m.ReplyToMessageID != nil {
				replyToID := m.ReplyToMessageID.String()
				msgResp.ReplyToMessageID = &replyToID
			}

			if m.ReplyToMessage != nil {
				replySenderPhone := extractMessageSenderPhone(m.ReplyToMessage.Metadata)
				if shouldMaskPhoneNumbers && replySenderPhone != "" {
					replySenderPhone = MaskPhoneNumber(replySenderPhone)
				}
				msgResp.ReplyToMessage = &ReplyPreview{
					ID:          m.ReplyToMessage.ID.String(),
					Content:     map[string]string{"body": normalizeDeletedMessageBody(m.ReplyToMessage.Content)},
					MessageType: m.ReplyToMessage.MessageType,
					Direction:   m.ReplyToMessage.Direction,
					SenderPhone: replySenderPhone,
				}
			}

			if msgResp.ReplyToMessage == nil {
				msgResp.ReplyToMessage = buildReplyPreviewFromMetadata(m.Metadata)
				if msgResp.ReplyToMessage != nil && shouldMaskPhoneNumbers && msgResp.ReplyToMessage.SenderPhone != "" {
					msgResp.ReplyToMessage.SenderPhone = MaskPhoneNumber(msgResp.ReplyToMessage.SenderPhone)
				}
			}
		}

		if m.Metadata != nil {
			if reactionsRaw, ok := m.Metadata["reactions"]; ok {
				if reactionsArray, ok := reactionsRaw.([]interface{}); ok {
					for _, r := range reactionsArray {
						if rMap, ok := r.(map[string]interface{}); ok {
							emoji, _ := rMap["emoji"].(string)
							fromPhone, _ := rMap["from_phone"].(string)
							fromUser, _ := rMap["from_user"].(string)
							msgResp.Reactions = append(msgResp.Reactions, ReactionInfo{
								Emoji:     emoji,
								FromPhone: fromPhone,
								FromUser:  fromUser,
							})
						}
					}
				}
			}
		}

		response = append(response, msgResp)
	}
	return response
}

// markMessagesAsRead marks messages as read and sends read receipts
func (a *App) markMessagesAsRead(orgID uuid.UUID, contactID uuid.UUID, contact *models.Contact) {
	var unreadMessages []models.Message
	a.DB.Where("contact_id = ? AND direction = ? AND status != ?", contactID, models.DirectionIncoming, models.MessageStatusRead).
		Find(&unreadMessages)

	a.DB.Model(&models.Message{}).
		Where("contact_id = ? AND direction = ?", contactID, models.DirectionIncoming).
		Update("status", models.MessageStatusRead)

	a.DB.Model(contact).Update("is_read", true)

	if len(unreadMessages) > 0 && contact.WhatsAppAccount != "" {
		if account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount); err == nil {
			if account.AutoReadReceipt {
				a.wg.Add(1)
				go func() {
					defer a.wg.Done()
					// Use timeout context for external API calls
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					waAccount := a.toWhatsAppAccount(account)
					for _, msg := range unreadMessages {
						// Check if context was cancelled
						if ctx.Err() != nil {
							a.Log.Warn("Read receipt sending cancelled", "reason", ctx.Err())
							return
						}
						if msg.WhatsAppMessageID == "" {
							continue
						}

						var err error
						if a.isWhatsmeowProvider() && a.MessageProvider != nil && msg.InstanceID != nil {
							err = a.MessageProvider.MarkRead(ctx, msg.InstanceID.String(), msg.WhatsAppMessageID)
						} else if !a.isWhatsmeowProvider() && a.WhatsApp != nil {
							err = a.WhatsApp.MarkMessageRead(ctx, waAccount, msg.WhatsAppMessageID)
						}

						if err != nil {
							a.Log.Error("Failed to send read receipt", "error", err, "message_id", msg.WhatsAppMessageID)
						}
					}
				}()
			}
		}
	}
}

// markGroupMessagesAsRead marks all incoming messages in a group conversation as read.
func (a *App) markGroupMessagesAsRead(orgID uuid.UUID, conversationID string, instanceID *uuid.UUID) {
	if strings.TrimSpace(conversationID) == "" {
		return
	}

	var unreadMessages []models.Message
	buildConversationScopeQuery(a.DB, orgID, conversationID, instanceID).
		Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead).
		Find(&unreadMessages)

	buildConversationScopeQuery(a.DB, orgID, conversationID, instanceID).
		Model(&models.Message{}).
		Where("direction = ?", models.DirectionIncoming).
		Update("status", models.MessageStatusRead)

	var conversationContactIDs []uuid.UUID
	buildConversationScopeQuery(a.DB, orgID, conversationID, instanceID).
		Model(&models.Message{}).
		Distinct("contact_id").
		Pluck("contact_id", &conversationContactIDs)
	if len(conversationContactIDs) > 0 {
		a.DB.Model(&models.Contact{}).
			Where("id IN ?", conversationContactIDs).
			Update("is_read", true)
	}

	if len(unreadMessages) == 0 {
		return
	}

	type accountEntry struct {
		Account *models.WhatsAppAccount
		Found   bool
	}
	accountCache := make(map[string]accountEntry)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, msg := range unreadMessages {
			if ctx.Err() != nil || msg.WhatsAppMessageID == "" {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			entry, ok := accountCache[msg.WhatsAppAccount]
			if !ok {
				account, err := a.resolveWhatsAppAccount(orgID, msg.WhatsAppAccount)
				if err != nil {
					accountCache[msg.WhatsAppAccount] = accountEntry{Found: false}
					continue
				}
				entry = accountEntry{Account: account, Found: true}
				accountCache[msg.WhatsAppAccount] = entry
			}
			if !entry.Found || entry.Account == nil || !entry.Account.AutoReadReceipt {
				continue
			}

			var err error
			if a.isWhatsmeowProvider() && a.MessageProvider != nil && msg.InstanceID != nil {
				err = a.MessageProvider.MarkRead(ctx, msg.InstanceID.String(), msg.WhatsAppMessageID)
			} else if !a.isWhatsmeowProvider() && a.WhatsApp != nil {
				err = a.WhatsApp.MarkMessageRead(ctx, a.toWhatsAppAccount(entry.Account), msg.WhatsAppMessageID)
			}
			if err != nil {
				a.Log.Error("Failed to send group read receipt", "error", err, "message_id", msg.WhatsAppMessageID)
			}
		}
	}()
}
