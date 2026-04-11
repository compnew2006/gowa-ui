package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
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
	IsPublic           bool       `json:"is_public"`
	IsCollaborator     bool       `json:"is_collaborator,omitempty"`
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
	Metadata         models.JSONB         `json:"metadata,omitempty"`
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
	ID            string             `json:"id"`
	Content       any                `json:"content"`
	MessageType   models.MessageType `json:"message_type"`
	Direction     models.Direction   `json:"direction"`
	SenderPhone   string             `json:"sender_phone,omitempty"`
	MediaURL      string             `json:"media_url,omitempty"`
	MediaMimeType string             `json:"media_mime_type,omitempty"`
	MediaFilename string             `json:"media_filename,omitempty"`
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

func buildGroupConversationMessagesQuery(db *gorm.DB, orgID uuid.UUID, conversationID string, contactID uuid.UUID, instanceID *uuid.UUID) *gorm.DB {
	// Include canonical conversation messages plus legacy system events that were
	// stored before conversation_id was persisted for claim/reset events.
	conversationClause := "organization_id = ? AND conversation_id = ?"
	conversationArgs := []any{orgID, conversationID}
	if instanceID != nil {
		conversationClause += " AND instance_id = ?"
		conversationArgs = append(conversationArgs, *instanceID)
	}

	legacyClause := "organization_id = ? AND contact_id = ? AND COALESCE(conversation_id, '') = '' AND metadata->>'system_event' = 'true'"
	legacyArgs := []any{orgID, contactID}
	if instanceID != nil {
		legacyClause += " AND instance_id = ?"
		legacyArgs = append(legacyArgs, *instanceID)
	}

	args := append(conversationArgs, legacyArgs...)
	return db.Where("(("+conversationClause+") OR ("+legacyClause+"))", args...)
}

func stringifyInstanceID(instanceID *uuid.UUID) *string {
	if instanceID == nil {
		return nil
	}
	value := instanceID.String()
	return &value
}

func conversationUnreadKey(conversationID string, instanceID *uuid.UUID) string {
	if instanceID == nil {
		return conversationID + "|"
	}
	return conversationID + "|" + instanceID.String()
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

func (a *App) resolveContactConversationContext(ctx context.Context, orgID uuid.UUID, contact models.Contact) contactConversationContext {
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

	if ctx == nil {
		ctx = context.Background()
	}

	var latestMessage models.Message
	if err := a.DB.WithContext(ctx).
		Select("conversation_id", "metadata", "direction").
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
		displayName := ""
		if latestMessage.Direction == models.DirectionIncoming {
			displayName = messageMetadataString(latestMessage.Metadata, "sender_push_name")
			if displayName == "" {
				displayName = messageMetadataString(latestMessage.Metadata, "push_name")
			}
		}
		return contactConversationContext{
			ConversationID: conversationID,
			DisplayName:    displayName,
		}
	}

	return contactConversationContext{}
}

func applyDirectContactPhoneFromConversation(contact *models.Contact, conversationID string) {
	if contact == nil {
		return
	}
	if isGroupContact(contact) || isChannelContact(contact) {
		return
	}

	canonicalPhone := strings.TrimSpace(directUserFromConversationID(conversationID))
	if canonicalPhone == "" {
		return
	}

	contact.PhoneNumber = canonicalPhone
}

func (a *App) repairDirectContactPhoneFromConversation(contact *models.Contact, conversationID string) {
	applyDirectContactPhoneFromConversation(contact, conversationID)
	if a == nil {
		return
	}
	a.enqueueDirectContactRepair(contact, conversationID)
}

func (a *App) enqueueDirectContactRepair(contact *models.Contact, conversationID string) {
	if a == nil || a.Queue == nil || contact == nil {
		return
	}
	if isGroupContact(contact) || isChannelContact(contact) {
		return
	}

	canonicalPhone := strings.TrimSpace(directUserFromConversationID(conversationID))
	if canonicalPhone == "" || canonicalPhone == strings.TrimSpace(contact.PhoneNumber) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job := &queue.ContactRepairJob{
		ContactID:      contact.ID,
		OrganizationID: contact.OrganizationID,
		ConversationID: conversationID,
	}
	if err := a.Queue.EnqueueContactRepair(ctx, job); err != nil {
		a.Log.Warn("Failed to enqueue contact phone repair", "contact_id", contact.ID, "conversation_id", conversationID, "canonical_phone", canonicalPhone, "error", err)
	}
}

// ListContacts returns all contacts for the organization
// Users without contacts:read permission see pending queue + contacts assigned to them
func (a *App) ListContacts(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 5*time.Second)
	defer cancel()
	ctxDB := requestDB.WithContext(ctx)

	// Pagination
	pg := parsePaginationWithDefaults(r, 50, 500)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))
	createdFromParam := string(r.RequestCtx.QueryArgs().Peek("created_from"))
	createdToParam := string(r.RequestCtx.QueryArgs().Peek("created_to"))
	dateBasisParam := string(r.RequestCtx.QueryArgs().Peek("date_basis"))
	dateFromParam := string(r.RequestCtx.QueryArgs().Peek("date_from"))
	dateToParam := string(r.RequestCtx.QueryArgs().Peek("date_to"))
	tagsParam := string(r.RequestCtx.QueryArgs().Peek("tags"))
	instanceIDParam := string(r.RequestCtx.QueryArgs().Peek("instance_id"))
	chatTypesParam := string(r.RequestCtx.QueryArgs().Peek("chat_types"))
	statusParam := string(r.RequestCtx.QueryArgs().Peek("status"))
	assignedToParam := string(r.RequestCtx.QueryArgs().Peek("assigned_to"))

	var contacts []models.Contact
	// Use explicit contacts table qualification so later JOINs (closed chat filters)
	// cannot make organization_id references ambiguous.
	query := ctxDB.Model(&models.Contact{}).Where("contacts.organization_id = ?", orgID)
	query = query.Joins(
		"LEFT JOIN contact_user_deletions cud ON cud.contact_id = contacts.id AND cud.organization_id = ? AND cud.user_id = ?",
		orgID,
		userID,
	).Where("(cud.id IS NULL OR COALESCE(contacts.last_message_at, contacts.created_at) > cud.deleted_at)")
	statusFilter, parseStatusErr := parseChatStatusFilter(statusParam)
	if parseStatusErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseStatusErr.Error(), nil, "status")
	}
	closedChatFilters, closedFiltersField, parseClosedFiltersErr := parseClosedChatFilters(r)
	if parseClosedFiltersErr != nil {
		errorType := fastglue.ErrorType("")
		if closedFiltersField == "closed_from" {
			errorType = "closed_from"
		}
		if closedFiltersField == "closed_to" {
			errorType = "closed_to"
		}
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseClosedFiltersErr.Error(), nil, errorType)
	}
	assignedToUserID, hasAssignedToFilter, parseAssignedToErr := parseAssignedToFilter(assignedToParam, userID)
	if parseAssignedToErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseAssignedToErr.Error(), nil, "assigned_to")
	}
	dateBasis, parseDateBasisErr := parseContactDateBasis(dateBasisParam)
	if parseDateBasisErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, parseDateBasisErr.Error(), nil, "date_basis")
	}

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatListFilter(query, userID)
	}

	restrictedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for contact list", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}
	query = applyRestrictedInstanceVisibilityFilter(query, restrictedInstanceIDs)

	var explicitInstanceID *uuid.UUID
	if instanceIDParam != "" {
		instanceID, resolveErr := a.resolveContactInstanceID(orgID, instanceIDParam)
		if resolveErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		if instanceID != nil {
			explicitInstanceID = instanceID
			query = query.Where("instance_id = ?", *instanceID)
		}
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
			if *assignedToUserID == userID {
				query = query.Where("(assigned_user_id = ? OR EXISTS (SELECT 1 FROM contact_collaborators cc WHERE cc.contact_id = contacts.id AND cc.user_id = ? AND cc.status IN ? AND cc.deleted_at IS NULL))",
					*assignedToUserID,
					userID,
					collaboratorAccessStatuses(),
				)
			} else {
				query = query.Where("assigned_user_id = ?", *assignedToUserID)
			}
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

	hasDateBasisFilters := strings.TrimSpace(dateBasisParam) != "" ||
		strings.TrimSpace(dateFromParam) != "" ||
		strings.TrimSpace(dateToParam) != ""
	if hasDateBasisFilters {
		var dateFrom *time.Time
		if dateFromParam != "" {
			parsedDateFrom, parseErr := time.Parse("2006-01-02", dateFromParam)
			if parseErr != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid date_from format. Use YYYY-MM-DD", nil, "date_from")
			}
			dateFrom = &parsedDateFrom
		}

		var dateTo *time.Time
		if dateToParam != "" {
			parsedDateTo, parseErr := time.Parse("2006-01-02", dateToParam)
			if parseErr != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid date_to format. Use YYYY-MM-DD", nil, "date_to")
			}
			dateTo = &parsedDateTo
		}

		query = applyContactDateBasisFilter(query, orgID, dateBasis, dateFrom, dateTo, explicitInstanceID)
	}

	if len(search) > 1000 {
		search = search[:1000]
	}

	isClosedChatList := statusFilter != nil && *statusFilter == models.ChatStatusClosed
	if isClosedChatList {
		query = applyClosedChatFilters(query, search, closedChatFilters)
	} else if search != "" {
		searchPattern := "%" + search + "%"
		// Use ILIKE for case-insensitive search on profile_name
		query = query.Where("phone_number LIKE ? OR profile_name ILIKE ?", searchPattern, searchPattern)
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

	// Public chats are pinned first, then most recent activity.
	query = query.Order("is_public DESC, last_message_at DESC NULLS LAST, created_at DESC")

	var total int64
	query.Model(&models.Contact{}).Count(&total)

	if err := query.Preload("ClosedByUser").Preload("AssignedUser").Offset(pg.Offset).Limit(pg.Limit).Find(&contacts).Error; err != nil {
		a.Log.Error("Failed to list contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}

	// Check if phone masking is enabled
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	collaboratorContactIDs, collabErr := a.listCollaboratorContactIDs(orgID, userID)
	if collabErr != nil {
		a.Log.Error("Failed to load collaborator contacts", "error", collabErr, "org_id", orgID, "user_id", userID)
	}
	contactIDs := make([]uuid.UUID, 0, len(contacts))
	for _, c := range contacts {
		contactIDs = append(contactIDs, c.ID)
	}
	deletionMap, deletionErr := a.getContactUserDeletionMap(ctx, orgID, userID, contactIDs)
	if deletionErr != nil {
		a.Log.Error("Failed to load chat deletions", "error", deletionErr, "org_id", orgID, "user_id", userID)
		deletionMap = map[uuid.UUID]time.Time{}
	}

	conversationContexts := make([]contactConversationContext, len(contacts))
	directContactIDs := make([]uuid.UUID, 0, len(contacts))
	groupConversationIDs := make([]string, 0, len(contacts))
	groupConversationIDSet := make(map[string]struct{}, len(contacts))
	groupKeyByContactID := make(map[uuid.UUID]string, len(contacts))

	for i, c := range contacts {
		conversationContext := a.resolveContactConversationContext(ctx, orgID, c)
		conversationContexts[i] = conversationContext
		if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
			groupKeyByContactID[c.ID] = conversationUnreadKey(conversationContext.ConversationID, c.InstanceID)
			if _, ok := groupConversationIDSet[conversationContext.ConversationID]; !ok {
				groupConversationIDSet[conversationContext.ConversationID] = struct{}{}
				groupConversationIDs = append(groupConversationIDs, conversationContext.ConversationID)
			}
		} else {
			directContactIDs = append(directContactIDs, c.ID)
		}
	}

	directUnreadCounts := map[uuid.UUID]int64{}
	var directUnreadErr error
	if len(directContactIDs) > 0 {
		type directUnreadRow struct {
			ContactID   uuid.UUID `gorm:"column:contact_id"`
			UnreadCount int64     `gorm:"column:unread_count"`
		}
		var rows []directUnreadRow
		directUnreadErr = ctxDB.Model(&models.Message{}).
			Select("messages.contact_id, COUNT(*) as unread_count").
			Joins("LEFT JOIN contact_user_deletions cud ON cud.organization_id = ? AND cud.user_id = ? AND cud.contact_id = messages.contact_id", orgID, userID).
			Where("messages.organization_id = ? AND messages.direction = ? AND messages.status != ? AND messages.contact_id IN ?",
				orgID, models.DirectionIncoming, models.MessageStatusRead, directContactIDs).
			Where("cud.deleted_at IS NULL OR messages.created_at > cud.deleted_at").
			Group("messages.contact_id").
			Scan(&rows).Error
		if directUnreadErr != nil {
			a.Log.Error("Failed to precompute unread counts for contacts", "error", directUnreadErr, "org_id", orgID, "user_id", userID)
		} else {
			for _, row := range rows {
				directUnreadCounts[row.ContactID] = row.UnreadCount
			}
		}
	}

	groupUnreadCounts := map[string]int64{}
	var groupUnreadErr error
	if len(groupConversationIDs) > 0 {
		type groupUnreadRow struct {
			ConversationID string     `gorm:"column:conversation_id"`
			InstanceID     *uuid.UUID `gorm:"column:instance_id"`
			UnreadCount    int64      `gorm:"column:unread_count"`
		}
		var rows []groupUnreadRow
		groupUnreadErr = ctxDB.Model(&models.Message{}).
			Select("messages.conversation_id, messages.instance_id, COUNT(*) as unread_count").
			Joins("LEFT JOIN contact_user_deletions cud ON cud.organization_id = ? AND cud.user_id = ? AND cud.contact_id = messages.contact_id", orgID, userID).
			Where("messages.organization_id = ? AND messages.direction = ? AND messages.status != ? AND messages.conversation_id IN ?",
				orgID, models.DirectionIncoming, models.MessageStatusRead, groupConversationIDs).
			Where("cud.deleted_at IS NULL OR messages.created_at > cud.deleted_at").
			Group("messages.conversation_id, messages.instance_id").
			Scan(&rows).Error
		if groupUnreadErr != nil {
			a.Log.Error("Failed to precompute unread counts for group chats", "error", groupUnreadErr, "org_id", orgID, "user_id", userID)
		} else {
			for _, row := range rows {
				groupUnreadCounts[conversationUnreadKey(row.ConversationID, row.InstanceID)] = row.UnreadCount
			}
		}
	}

	// Convert to response format
	response := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		status := normalizeContactStatus(&c)
		conversationContext := conversationContexts[i]
		applyDirectContactPhoneFromConversation(&c, conversationContext.ConversationID)
		a.enqueueDirectContactRepair(&c, conversationContext.ConversationID)
		a.scheduleContactAvatarRefresh(&c)

		// Count unread messages
		var unreadCount int64
		if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
			if groupUnreadErr == nil {
				if key, ok := groupKeyByContactID[c.ID]; ok {
					unreadCount = groupUnreadCounts[key]
				}
			} else {
				deletedAt, hasDeletion := deletionMap[c.ID]
				msgQuery := buildConversationScopeQuery(ctxDB, orgID, conversationContext.ConversationID, c.InstanceID).
					Model(&models.Message{}).
					Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead)
				if hasDeletion {
					msgQuery = msgQuery.Where("created_at > ?", deletedAt)
				}
				msgQuery.Count(&unreadCount)
			}
		} else {
			if directUnreadErr == nil {
				unreadCount = directUnreadCounts[c.ID]
			} else {
				deletedAt, hasDeletion := deletionMap[c.ID]
				msgQuery := ctxDB.Model(&models.Message{}).
					Where("contact_id = ? AND direction = ? AND status != ?", c.ID, models.DirectionIncoming, models.MessageStatusRead)
				if hasDeletion {
					msgQuery = msgQuery.Where("created_at > ?", deletedAt)
				}
				msgQuery.Count(&unreadCount)
			}
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
		if !conversationContext.IsGroupChat && !conversationContext.IsChannelChat && isDirectIdentityValue(profileName) {
			if fallbackName := fallbackDirectContactDisplayName(phoneNumber, conversationContext.ConversationID); fallbackName != "" {
				profileName = fallbackName
			}
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
		isCollaborator := false
		if _, ok := collaboratorContactIDs[c.ID]; ok {
			isCollaborator = true
		}

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
			IsPublic:           c.IsPublic,
			IsCollaborator:     isCollaborator,
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

// GetContact returns a single contact.
// Agent-role users stay scoped to visible chats (pending queue, public chats, and their own assignments).
func (a *App) GetContact(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	ctx, cancel := context.WithTimeout(r.RequestCtx, 5*time.Second)
	defer cancel()
	ctxDB := requestDB.WithContext(ctx)
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	query := ctxDB.Preload("ClosedByUser").Preload("AssignedUser").Where("id = ? AND organization_id = ?", contactID, orgID)

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}
	restrictedInstanceIDs, restrictedErr := a.getRestrictedInstancesForUser(orgID, userID)
	if restrictedErr != nil {
		a.Log.Error("Failed to resolve restricted instance for contact read", "error", restrictedErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load contact", nil, "")
	}
	query = applyRestrictedInstanceVisibilityFilter(query, restrictedInstanceIDs)

	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	// Count unread messages
	conversationContext := a.resolveContactConversationContext(ctx, orgID, contact)
	applyDirectContactPhoneFromConversation(&contact, conversationContext.ConversationID)
	a.enqueueDirectContactRepair(&contact, conversationContext.ConversationID)
	a.scheduleContactAvatarRefresh(&contact)
	var unreadCount int64
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		buildConversationScopeQuery(ctxDB, orgID, conversationContext.ConversationID, contact.InstanceID).
			Model(&models.Message{}).
			Where("direction = ? AND status != ?", models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)
	} else {
		ctxDB.Model(&models.Message{}).
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
	if !conversationContext.IsGroupChat && !conversationContext.IsChannelChat && isDirectIdentityValue(profileName) {
		if fallbackName := fallbackDirectContactDisplayName(phoneNumber, conversationContext.ConversationID); fallbackName != "" {
			profileName = fallbackName
		}
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
	isCollaborator := a.isContactCollaborator(orgID, contact.ID, userID)

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
		IsPublic:           contact.IsPublic,
		IsCollaborator:     isCollaborator,
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

// GetMessages returns messages for a contact.
// Agent-role users stay scoped to visible chats (pending queue, public chats, and their own assignments).
// Supports cursor-based pagination with before_id for loading older messages.
func (a *App) GetMessages(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	ctx, cancel := context.WithTimeout(r.RequestCtx, 5*time.Second)
	defer cancel()
	ctxDB := requestDB.WithContext(ctx)
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	hasContactsReadPermission := a.canReadAllContacts(userID, orgID)
	limitChatVisibilityToAgentScope := a.shouldRestrictChatVisibilityToAgentScope(userID, orgID)
	restrictedInstanceIDs, restrictedErr := a.getRestrictedInstancesForUser(orgID, userID)
	if restrictedErr != nil {
		a.Log.Error("Failed to resolve restricted instance for messages", "error", restrictedErr, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load messages", nil, "")
	}

	// Verify contact belongs to org and stays inside the current user's chat scope.
	var contact models.Contact
	query := ctxDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if limitChatVisibilityToAgentScope {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}
	query = applyRestrictedInstanceVisibilityFilter(query, restrictedInstanceIDs)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	normalizeContactStatus(&contact)
	isCollaborator := a.isContactCollaborator(orgID, contact.ID, userID)
	if isCollaborator {
		hasContactsReadPermission = true
	}
	if isChatRestrictedForMessageRead(contact) && !a.canAccessRestrictedChatWithoutClaim(contact, userID, orgID) {
		return r.SendErrorEnvelope(
			fasthttp.StatusForbidden,
			"This chat is currently unassigned. Claim it before viewing messages.",
			nil,
			"",
		)
	}
	shouldMaskPhoneNumbers := a.ShouldMaskPhoneNumbers(orgID)
	conversationContext := a.resolveContactConversationContext(ctx, orgID, contact)
	deletedAt, deletionErr := a.getContactUserDeletionTimestamp(ctx, orgID, contact.ID, userID)
	if deletionErr != nil {
		a.Log.Error("Failed to resolve chat deletion timestamp", "error", deletionErr, "contact_id", contact.ID, "user_id", userID)
	}

	// Pagination parameters
	limit, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	beforeIDStr := string(r.RequestCtx.QueryArgs().Peek("before_id"))

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Build base query
	msgQuery := ctxDB.Where("contact_id = ?", contactID)
	if conversationContext.IsGroupChat && conversationContext.ConversationID != "" {
		msgQuery = buildGroupConversationMessagesQuery(ctxDB, orgID, conversationContext.ConversationID, contact.ID, contact.InstanceID)
	}
	if deletedAt != nil {
		msgQuery = msgQuery.Where("created_at > ?", *deletedAt)
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
				if err := requestDB.Where("contact_id = ? AND organization_id = ?", contactID, orgID).
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
			if err := requestDB.Where("id = ?", beforeID).First(&beforeMsg).Error; err == nil {
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
		if shouldMaskPhoneNumbers {
			normalizedContent = MaskPhoneNumbersInText(normalizedContent)
		}
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
			Metadata:        m.Metadata,
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
				replyContent := normalizeDeletedMessageBody(m.ReplyToMessage.Content)
				if shouldMaskPhoneNumbers {
					replyContent = MaskPhoneNumbersInText(replyContent)
				}
				msgResp.ReplyToMessage = &ReplyPreview{
					ID:            m.ReplyToMessage.ID.String(),
					Content:       map[string]string{"body": replyContent},
					MessageType:   m.ReplyToMessage.MessageType,
					Direction:     m.ReplyToMessage.Direction,
					SenderPhone:   replySenderPhone,
					MediaURL:      m.ReplyToMessage.MediaURL,
					MediaMimeType: m.ReplyToMessage.MediaMimeType,
					MediaFilename: m.ReplyToMessage.MediaFilename,
				}
			}

			if msgResp.ReplyToMessage == nil {
				msgResp.ReplyToMessage = buildReplyPreviewFromMetadata(a.DB, m.OrganizationID, m.InstanceID, m.Metadata)
				if msgResp.ReplyToMessage != nil && shouldMaskPhoneNumbers {
					if msgResp.ReplyToMessage.SenderPhone != "" {
						msgResp.ReplyToMessage.SenderPhone = MaskPhoneNumber(msgResp.ReplyToMessage.SenderPhone)
					}
					if cMap, ok := msgResp.ReplyToMessage.Content.(map[string]interface{}); ok {
						if bodyAny, ok := cMap["body"]; ok {
							if bodyStr, ok := bodyAny.(string); ok {
								cMap["body"] = MaskPhoneNumbersInText(bodyStr)
							}
						}
					} else if cMapStr, ok := msgResp.ReplyToMessage.Content.(map[string]string); ok {
						if bodyStr, ok := cMapStr["body"]; ok {
							cMapStr["body"] = MaskPhoneNumbersInText(bodyStr)
						}
					}
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
	const maxReadReceiptMessages = 1000

	type receiptCandidate struct {
		WhatsAppMessageID string
		InstanceID        *uuid.UUID
	}

	var receiptCandidates []receiptCandidate
	a.DB.Model(&models.Message{}).
		Select("whats_app_message_id", "instance_id").
		Where("contact_id = ? AND direction = ? AND status != ?", contactID, models.DirectionIncoming, models.MessageStatusRead).
		Order("created_at DESC").
		Limit(maxReadReceiptMessages).
		Scan(&receiptCandidates)

	a.DB.Model(&models.Message{}).
		Where("contact_id = ? AND direction = ?", contactID, models.DirectionIncoming).
		Update("status", models.MessageStatusRead)

	a.DB.Model(contact).Update("is_read", true)

	if len(receiptCandidates) > 0 && contact.WhatsAppAccount != "" {
		if account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount); err == nil {
			if account.AutoReadReceipt {
				a.wg.Add(1)
				go func() {
					defer a.wg.Done()
					// Use timeout context for external API calls
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					waAccount := a.toWhatsAppAccount(account)
					for _, msg := range receiptCandidates {
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
