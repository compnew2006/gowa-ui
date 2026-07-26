package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/utils"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// ContactResponse represents a contact with additional fields for the frontend
type ContactResponse struct {
	ID                 uuid.UUID             `json:"id"`
	PhoneNumber        string                `json:"phone_number"`
	Name               string                `json:"name"`
	ProfileName        string                `json:"profile_name"`
	AvatarURL          string                `json:"avatar_url"`
	Status             string                `json:"status"`
	Tags               []string              `json:"tags"`
	Metadata           any                   `json:"metadata"`
	LastMessageAt      *time.Time            `json:"last_message_at"`
	LastMessagePreview string                `json:"last_message_preview"`
	UnreadCount        int                   `json:"unread_count"`
	AssignedUserID     *uuid.UUID            `json:"assigned_user_id,omitempty"`
	AssignedUserName   string                `json:"assigned_user_name,omitempty"`
	WhatsAppAccount    string                `json:"whatsapp_account,omitempty"`
	LastInboundAt      *time.Time            `json:"last_inbound_at,omitempty"`
	ServiceWindowOpen  bool                  `json:"service_window_open"`
	MarketingOptOut    bool                  `json:"marketing_opt_out"`
	IsGroupChat        bool                  `json:"is_group_chat"`
	IsNewsletter       bool                  `json:"is_newsletter"`
	ChatStatus         string                `json:"chat_status,omitempty"`
	Collaborators      []models.Collaborator `json:"collaborators,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

// MessageResponse represents a message for the frontend
type MessageResponse struct {
	ID               uuid.UUID            `json:"id"`
	ContactID        uuid.UUID            `json:"contact_id"`
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
	IsGroupChat      bool                 `json:"is_group_chat"`
	IsNewsletter     bool                 `json:"is_newsletter"`
	SenderPhone      string               `json:"sender_phone,omitempty"`
	SenderPushName   string               `json:"sender_push_name,omitempty"`
	Metadata         models.JSONB         `json:"metadata,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// ReplyPreview contains a preview of the replied-to message
type ReplyPreview struct {
	ID          string             `json:"id"`
	Content     any                `json:"content"`
	MessageType models.MessageType `json:"message_type"`
	Direction   models.Direction   `json:"direction"`
}

// ReactionInfo represents a reaction on a message
type ReactionInfo struct {
	Emoji     string `json:"emoji"`
	FromPhone string `json:"from_phone,omitempty"`
	FromUser  string `json:"from_user,omitempty"`
}

// ListContacts returns all contacts for the organization
// Users without contacts:read permission only see contacts assigned to them
func (a *App) ListContacts(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Pagination
	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))
	tagsParam := string(r.RequestCtx.QueryArgs().Peek("tags"))
	// When has_messages=true, hide contacts with no messages (used by /chat to
	// distinguish real conversations from synced-but-empty contacts in /settings/contacts).
	hasMessages := string(r.RequestCtx.QueryArgs().Peek("has_messages"))

	var contacts []models.Contact
	query := a.ScopeToOrg(a.DB, userID, orgID)

	// Users without contacts:read permission can only see contacts assigned to them
	// or contacts with an active chat transfer to them
	query = a.scopeAssignedContact(query, userID, orgID)

	// Hide empty conversations: a contact is a "conversation" only if it has at
	// least one message. /chat uses this by default; /settings/contacts shows all.
	if hasMessages == "true" || hasMessages == "1" {
		query = query.Where("last_message_at IS NOT NULL")
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

	if err := query.Offset(pg.Offset).Limit(pg.Limit).Find(&contacts).Error; err != nil {
		a.Log.Error("Failed to list contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}

	// Check if phone masking is enabled
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)

	// Convert to response format
	response := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		// Count unread messages
		var unreadCount int64
		a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?", c.ID, models.DirectionIncoming, models.MessageStatusRead).
			Count(&unreadCount)

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
		if shouldMask {
			phoneNumber = utils.MaskPhoneNumber(phoneNumber)
			profileName = utils.MaskIfPhoneNumber(profileName)
		}

		serviceWindowOpen := c.LastInboundAt != nil && time.Since(*c.LastInboundAt) < 24*time.Hour

		response[i] = ContactResponse{
			ID:                 c.ID,
			PhoneNumber:        phoneNumber,
			Name:               profileName,
			ProfileName:        profileName,
			AvatarURL:          c.AvatarURL,
			Status:             "active",
			Tags:               tags,
			Metadata:           c.Metadata,
			LastMessageAt:      c.LastMessageAt,
			LastMessagePreview: c.LastMessagePreview,
			UnreadCount:        int(unreadCount),
			AssignedUserID:     c.AssignedUserID,
			AssignedUserName: func() string {
				if c.AssignedUserID == nil {
					return ""
				}
				var u models.User
				if a.DB.Select("full_name").First(&u, "id = ?", *c.AssignedUserID).Error == nil {
					return u.FullName
				}
				return ""
			}(),
			WhatsAppAccount:   c.WhatsAppAccount,
			LastInboundAt:     c.LastInboundAt,
			ServiceWindowOpen: serviceWindowOpen,
			MarketingOptOut:   c.MarketingOptOut,
			IsGroupChat:       c.Metadata != nil && c.Metadata["is_group_chat"] == true,
			IsNewsletter:      c.Metadata != nil && c.Metadata["is_newsletter"] == true,
			ChatStatus:        string(c.EffectiveStatus()),
			Collaborators:     a.filterCollaboratorsForViewer(c.GetCollaborators(), userID, orgID),
			CreatedAt:         c.CreatedAt,
			UpdatedAt:         c.UpdatedAt,
		}
	}

	return r.SendEnvelope(listEnvelope("contacts", response, total, pg))
}

// scopeAssignedContact narrows a contact query for users who lack the
// contacts:read permission: they may only access contacts assigned to them
// (assigned_user_id) or contacts with an active agent transfer to them. With
// the permission, the query is returned unchanged. Keeping this in one place
// ensures every contact endpoint enforces the same visibility — assignment
// via an active transfer counts even when assigned_user_id is unset (which it
// is unless the AssignToSameAgent setting is on).
func (a *App) scopeAssignedContact(query *gorm.DB, userID, orgID uuid.UUID) *gorm.DB {
	if a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		return query
	}
	// Agents (users without contacts:read) can only access contacts:
	//   1. assigned to them (assigned_user_id),
	//   2. with an active agent transfer to them, OR
	//   3. where they are listed as a collaborator in the contact's metadata.
	// Collaborators are stored in metadata.collaborators as a JSON array of
	// {user_id, name, role, joined_at}. The @> containment operator reuses the
	// same pattern as the tags filter above and leverages the GIN index.
	collaboratorJSON := fmt.Sprintf(`{"collaborators":[{"user_id":"%s"}]}`, userID.String())
	return query.Where(
		"assigned_user_id = ? OR id IN (?) OR metadata @> ?::jsonb",
		userID,
		a.DB.Model(&models.AgentTransfer{}).
			Select("contact_id").
			Where("agent_id = ? AND organization_id = ? AND status = ?", userID, orgID, models.TransferStatusActive),
		collaboratorJSON,
	)
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
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)

	// Users without contacts:read permission can only access their assigned contacts
	// or contacts with an active chat transfer to them
	query = a.scopeAssignedContact(query, userID, orgID)

	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	response := a.buildContactResponse(&contact, orgID, userID)

	return r.SendEnvelope(response)
}

// RefreshContactAvatar fetches the contact's current WhatsApp profile picture
// (or group icon) on demand and returns the freshly-cached avatar_url. This is
// the lazy refresh path for contacts created before a GOWA contact sync (e.g.
// via an inbound message) or whose picture changed after the last sync. The
// response is always 200 with the current avatar_url (possibly empty when the
// contact has no picture or no GOWA provider is available); the frontend
// initials fallback covers the empty case.
//
// GET /api/contacts/{id}/avatar
func (a *App) RefreshContactAvatar(r *fastglue.Request) error {
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// The avatar endpoint returns a (cached) profile picture URL — not private
	// conversation content — so it is scoped to the organization only. Applying
	// scopeAssignedContact here caused a 404 for any contact an agent can see
	// in their sidebar but isn't formally assigned/collaborating on (e.g. a
	// pending chat before claim, or a newsletter the admin surfaced). The
	// agent still sees the cached avatar; only the live GOWA re-fetch below is
	// gated by having a resolvable owning account.
	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Resolve the owning account. Without one we can't reach GOWA, so return
	// whatever avatar_url is already cached (may be empty).
	if contact.WhatsAppAccount == "" {
		return r.SendEnvelope(map[string]any{"avatar_url": contact.AvatarURL})
	}
	var account models.WhatsAppAccount
	if err := a.DB.Where("name = ? AND organization_id = ?", contact.WhatsAppAccount, orgID).First(&account).Error; err != nil {
		a.Log.Warn("Contact avatar refresh: account not found", "account", contact.WhatsAppAccount, "contact", contact.ID)
		return r.SendEnvelope(map[string]any{"avatar_url": contact.AvatarURL})
	}
	a.decryptAccountSecrets(&account)

	// GOWA accounts: fetch via the GOWA client. Meta Cloud API has no per-
	// contact profile-picture fetch, so non-GOWA accounts just return the
	// cached URL (empty today; Meta's business_profile picture is a different
	// field handled elsewhere).
	if account.IsGowa() {
		provider := a.resolveProvider(&account)
		gowaClient, _ := provider.(*gowa.Client)
		a.refreshContactAvatar(gowaClient, &account, &contact, account.GowaDeviceID, true)
	}

	return r.SendEnvelope(map[string]any{"avatar_url": contact.AvatarURL})
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

	hasContactsReadPermission := a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID)

	// Verify contact belongs to org (and to user if no contacts:read permission)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Managers/admins (contacts:write) can see any chat — no restrictions.
	// Agents can only see messages if: they own it, are a collaborator, or have collaborate permission.
	// Closed conversations are readable by everyone (read-only).
	hasContactsWritePermission := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
	hasCollaboratePermission := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	isAssigned := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	canViewContent := hasContactsWritePermission || isAssigned || isCollaborator || hasCollaboratePermission

	if !canViewContent && contact.EffectiveStatus() == models.ChatStatusPending {
		var pendingCount int64
		a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?",
				contactID, models.DirectionIncoming, models.MessageStatusRead).
			Count(&pendingCount)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden,
			"Claim this chat to view messages",
			map[string]any{"pending_message_count": pendingCount},
			"chat_not_claimed")
	}
	// ─── End privacy guard ───

	// Pagination parameters
	limit, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	beforeIDStr := string(r.RequestCtx.QueryArgs().Peek("before_id"))

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Build base query
	msgQuery := a.DB.Where("contact_id = ?", contactID)

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

		response := a.buildMessagesResponse(messages)
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
	a.markMessagesAsRead(orgID, contactID, &contact)

	response := a.buildMessagesResponse(messages)
	return r.SendEnvelope(map[string]any{
		"messages": response,
		"total":    total,
		"page":     page,
		"limit":    responseLimit,
		"has_more": offset > 0,
	})
}

// buildMessagesResponse converts messages to response format
func (a *App) buildMessagesResponse(messages []models.Message) []MessageResponse {
	response := make([]MessageResponse, len(messages))
	for i, m := range messages {
		var content any
		if m.MessageType == models.MessageTypeText {
			content = map[string]string{"body": m.Content}
		} else {
			content = map[string]string{"body": m.Content}
		}

		msgResp := MessageResponse{
			ID:              m.ID,
			ContactID:       m.ContactID,
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
			IsGroupChat:     m.Metadata != nil && m.Metadata["is_group_chat"] == true,
			IsNewsletter:    m.Metadata != nil && m.Metadata["is_newsletter"] == true,
			SenderPhone:     metadataString(m.Metadata, "sender_phone"),
			SenderPushName:  metadataString(m.Metadata, "sender_push_name"),
			Metadata:        m.Metadata,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}

		if m.IsReply && m.ReplyToMessageID != nil {
			replyToID := m.ReplyToMessageID.String()
			msgResp.ReplyToMessageID = &replyToID
			if m.ReplyToMessage != nil {
				msgResp.ReplyToMessage = &ReplyPreview{
					ID:          m.ReplyToMessage.ID.String(),
					Content:     map[string]string{"body": m.ReplyToMessage.Content},
					MessageType: m.ReplyToMessage.MessageType,
					Direction:   m.ReplyToMessage.Direction,
				}
			}
		}

		if m.Metadata != nil {
			if reactionsRaw, ok := m.Metadata["reactions"]; ok {
				if reactionsArray, ok := reactionsRaw.([]any); ok {
					for _, r := range reactionsArray {
						if rMap, ok := r.(map[string]any); ok {
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

		response[i] = msgResp
	}
	return response
}

// MarkContactRead marks all incoming messages from a contact as read.
// Called from the frontend when a new message arrives for the chat the
// user is currently viewing, so the sidebar unread badge stays at zero.
func (a *App) MarkContactRead(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	a.markMessagesAsRead(orgID, contactID, &contact)
	return r.SendEnvelope(map[string]any{"status": "ok"})
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
						if msg.WhatsAppMessageID != "" {
							provider := a.resolveProvider(account)
							// GOWA requires the chat JID for read receipts;
							// the Provider interface's MarkMessageRead lacks it.
							if account.IsGowa() {
								if gc, ok := provider.(*gowa.Client); ok {
									// Build the chat JID (handles group @g.us vs 1:1 suffix).
									chatJID := gowaChatJID(contact)
									if err := gc.MarkMessageReadWithJID(ctx, waAccount, msg.WhatsAppMessageID, chatJID); err != nil {
										a.Log.Error("Failed to send GOWA read receipt", "error", err, "message_id", msg.WhatsAppMessageID)
									}
								}
							} else if err := provider.MarkMessageRead(ctx, waAccount, msg.WhatsAppMessageID); err != nil {
								a.Log.Error("Failed to send read receipt", "error", err, "message_id", msg.WhatsAppMessageID)
							}
						}
					}
				}()
			}
		}
	}
}

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	Type    models.MessageType `json:"type"`
	Content struct {
		Body string `json:"body"`
		// Media fields (used on retry/resend of media messages)
		MediaData     string `json:"media_data,omitempty"`
		MediaMimeType string `json:"media_mime_type,omitempty"`
		MediaFilename string `json:"media_filename,omitempty"`
		MediaURL      string `json:"media_url,omitempty"`
	} `json:"content"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	WhatsAppAccount  string `json:"whatsapp_account,omitempty"`

	// Interactive message fields (for type="interactive")
	Interactive *InteractiveContent `json:"interactive,omitempty"`
}

// InteractiveContent holds interactive message data
type InteractiveContent struct {
	Type       string          `json:"type"`                  // "button", "list", "cta_url", "voice_call", "flow"
	Body       string          `json:"body"`                  // Body text
	Buttons    []ButtonContent `json:"buttons,omitempty"`     // For button type
	ButtonText string          `json:"button_text,omitempty"` // CTA label for cta_url and flow
	URL        string          `json:"url,omitempty"`         // For cta_url type
	// voice_call only: button face label and clickable TTL.
	// The payload (round-trip opaque string Meta echoes back on the incoming-
	// call webhook) is set server-side from the auth context — never from the
	// request body — to prevent agent-id spoofing.
	DisplayText string `json:"display_text,omitempty"`
	TTLMinutes  int    `json:"ttl_minutes,omitempty"`
	// flow only: the Meta flow to launch, an optional first screen, and an
	// optional header. Body holds the message text, ButtonText the CTA label.
	FlowID      string `json:"flow_id,omitempty"`
	FirstScreen string `json:"first_screen,omitempty"`
	Header      string `json:"header,omitempty"`
}

// ButtonContent represents a button in interactive messages
type ButtonContent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SendMessage sends a message to a contact
// Agents can only send messages to their assigned contacts
func (a *App) SendMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Parse request body
	var req SendMessageRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get WhatsApp account - prefer request-specified account over contact default
	accountName := contact.WhatsAppAccount
	if req.WhatsAppAccount != "" {
		accountName = req.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, accountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to resolve WhatsApp account", nil, "")
	}

	// Handle reply context
	var replyToMessage *models.Message
	if req.ReplyToMessageID != "" {
		replyToID, err := uuid.Parse(req.ReplyToMessageID)
		if err == nil {
			var replyTo models.Message
			if err := a.DB.Where("id = ? AND contact_id = ?", replyToID, contactID).First(&replyTo).Error; err == nil {
				replyToMessage = &replyTo
			}
		}
	}

	// Build request and send using unified sender
	msgReq := OutgoingMessageRequest{
		Account:        account,
		Contact:        &contact,
		Type:           req.Type,
		Content:        req.Content.Body,
		ReplyToMessage: replyToMessage,
	}

	// Wire media fields for image/video/audio/document sends and retries.
	if req.Content.MediaData != "" {
		if decoded, decErr := base64.StdEncoding.DecodeString(req.Content.MediaData); decErr == nil {
			msgReq.MediaData = decoded
		}
	}
	msgReq.MediaMimeType = req.Content.MediaMimeType
	msgReq.MediaFilename = req.Content.MediaFilename
	msgReq.MediaURL = req.Content.MediaURL

	// Handle interactive messages
	if req.Type == models.MessageTypeInteractive && req.Interactive != nil {
		msgReq.InteractiveType = req.Interactive.Type
		msgReq.BodyText = req.Interactive.Body
		msgReq.ButtonText = req.Interactive.ButtonText
		msgReq.URL = req.Interactive.URL

		// Convert buttons
		if len(req.Interactive.Buttons) > 0 {
			msgReq.Buttons = make([]whatsapp.Button, len(req.Interactive.Buttons))
			for i, btn := range req.Interactive.Buttons {
				msgReq.Buttons[i] = whatsapp.Button{
					ID:    btn.ID,
					Title: btn.Title,
				}
			}
		}

		if req.Interactive.Type == "flow" {
			if req.Interactive.FlowID == "" {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "flow_id is required to send a flow", nil, "")
			}
			// Ensure the flow belongs to this org so an agent can't send another
			// org's flow by supplying its Meta id.
			var waFlow models.WhatsAppFlow
			if err := a.DB.Where("meta_flow_id = ? AND organization_id = ?", req.Interactive.FlowID, orgID).First(&waFlow).Error; err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Flow not found for this organization", nil, "")
			}
			cta := req.Interactive.ButtonText
			if cta == "" {
				cta = "Open"
			}
			body := req.Interactive.Body
			if body == "" {
				body = req.Content.Body
			}
			msgReq.Type = models.MessageTypeFlow
			msgReq.FlowID = req.Interactive.FlowID
			msgReq.FlowCTA = cta
			msgReq.FlowHeader = req.Interactive.Header
			msgReq.FlowFirstScreen = req.Interactive.FirstScreen
			msgReq.BodyText = body
			msgReq.FlowToken = fmt.Sprintf("agent_%s_%d", contact.ID, time.Now().UnixNano())
		}

		if req.Interactive.Type == "voice_call" {
			if !account.BusinessCallingEnabled {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
					"This WhatsApp account is not enrolled in the Business Calling API. Enable it under Settings → Accounts before sending Call buttons.",
					nil, "")
			}
			msgReq.DisplayText = req.Interactive.DisplayText
			msgReq.TTLMinutes = req.Interactive.TTLMinutes
			// Stamp the payload server-side so the incoming-call webhook can
			// sticky-route the resulting call back to the agent who sent it.
			// Never trust a client-supplied payload — would let any agent
			// impersonate any other.
			msgReq.VoiceCallPayload = "agent:" + userID.String()
			// Pre-register the sticky-routing intent in Redis so that the
			// resulting incoming-call webhook can resolve the originating
			// agent in O(1) (Meta does not currently echo the payload).
			// TTL matches the button's clickable lifetime.
			a.MarkPendingStickyCall(context.Background(), orgID, contact.PhoneNumber, userID, req.Interactive.TTLMinutes)
		}
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		a.Log.Error("Failed to send message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	// Build response
	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
		Direction:       message.Direction,
		MessageType:     message.MessageType,
		Content:         map[string]string{"body": message.Content},
		InteractiveData: message.InteractiveData,
		Status:          message.Status,
		IsReply:         message.IsReply,
		WhatsAppAccount: message.WhatsAppAccount,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
	}

	// Add reply context to response
	if message.IsReply && message.ReplyToMessageID != nil && replyToMessage != nil {
		replyToID := message.ReplyToMessageID.String()
		response.ReplyToMessageID = &replyToID
		response.ReplyToMessage = &ReplyPreview{
			ID:          replyToMessage.ID.String(),
			Content:     map[string]string{"body": replyToMessage.Content},
			MessageType: replyToMessage.MessageType,
			Direction:   replyToMessage.Direction,
		}
	}

	return r.SendEnvelope(response)
}

// resolveWhatsAppAccount gets the WhatsApp account for sending messages
func (a *App) resolveWhatsAppAccount(orgID uuid.UUID, accountName string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount

	if accountName != "" {
		if err := a.DB.Where("name = ? AND organization_id = ?", accountName, orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("WhatsApp account not found")
		}
		a.decryptAccountSecrets(&account)
		return &account, nil
	}

	// Get default outgoing account
	if err := a.DB.Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).First(&account).Error; err != nil {
		// Fall back to any account
		if err := a.DB.Where("organization_id = ?", orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("no WhatsApp account configured")
		}
	}
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// resolveProvider returns the WhatsApp provider for the given account.
// When the registry is configured it resolves Meta vs GOWA based on the
// account's ProviderType. When the registry is nil it falls back to the
// default WhatsApp client (Meta).
func (a *App) resolveProvider(account *models.WhatsAppAccount) whatsapp.Provider {
	if a.WARegistry != nil && account != nil {
		return a.WARegistry.Get(account.ToWAAccount())
	}
	return a.WhatsApp
}

// resolveWhatsAppAccountByID fetches a WhatsApp account by UUID and org, decrypts secrets.
func (a *App) resolveWhatsAppAccountByID(r *fastglue.Request, id, orgID uuid.UUID) (*models.WhatsAppAccount, error) {
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil, err
	}
	a.decryptAccountSecrets(account)
	return account, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SendMediaMessage sends a media message (image, document, video, audio) to a contact
func (a *App) SendMediaMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Parse multipart form
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
	}

	// Get contact ID from form
	contactIDValues := form.Value["contact_id"]
	if len(contactIDValues) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "contact_id is required", nil, "")
	}
	contactID, err := uuid.Parse(contactIDValues[0])
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact ID", nil, "")
	}

	// Get media type (image, document, video, audio)
	mediaType := "image"
	if typeValues := form.Value["type"]; len(typeValues) > 0 {
		mediaType = typeValues[0]
	}

	// Get caption (optional)
	caption := ""
	if captionValues := form.Value["caption"]; len(captionValues) > 0 {
		caption = captionValues[0]
	}

	// Get WhatsApp account override (optional)
	formWhatsAppAccount := ""
	if accountValues := form.Value["whatsapp_account"]; len(accountValues) > 0 {
		formWhatsAppAccount = accountValues[0]
	}

	// Get uploaded file
	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "file is required", nil, "")
	}
	fileHeader := files[0]

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read file", nil, "")
	}
	defer func() { _ = file.Close() }()

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		a.Log.Error("Failed to read file data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file data", nil, "")
	}

	// Get MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get WhatsApp account - prefer form-specified account over contact default
	mediaAccountName := contact.WhatsAppAccount
	if formWhatsAppAccount != "" {
		mediaAccountName = formWhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, mediaAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Save file locally first
	localPath, err := a.saveMediaLocally(fileData, mimeType, fileHeader.Filename)
	if err != nil {
		a.Log.Error("Failed to save media locally", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media", nil, "")
	}

	// Build and send via unified message sender
	msgReq := OutgoingMessageRequest{
		Account:       account,
		Contact:       &contact,
		Type:          models.MessageType(mediaType),
		MediaData:     fileData,
		MediaURL:      localPath,
		MediaMimeType: mimeType,
		MediaFilename: fileHeader.Filename,
		Caption:       caption,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		a.Log.Error("Failed to send message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
		Direction:       message.Direction,
		MessageType:     message.MessageType,
		Content:         map[string]string{"body": message.Content},
		MediaURL:        message.MediaURL,
		MediaMimeType:   message.MediaMimeType,
		MediaFilename:   message.MediaFilename,
		Status:          message.Status,
		WhatsAppAccount: message.WhatsAppAccount,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
	}

	return r.SendEnvelope(response)
}

// saveMediaLocally saves media data to local storage and returns the relative path
func (a *App) saveMediaLocally(data []byte, mimeType, filename string) (string, error) {
	// Determine subdirectory based on MIME type
	var subdir string
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		subdir = "images"
	case strings.HasPrefix(mimeType, "video/"):
		subdir = "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		subdir = "audio"
	default:
		subdir = "documents"
	}

	// Ensure directory exists
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Get extension from MIME type or filename
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		// Try to get from filename
		if dotIdx := strings.LastIndex(filename, "."); dotIdx >= 0 {
			ext = filename[dotIdx:]
		} else {
			ext = ".bin"
		}
	}

	// Generate unique filename
	newFilename := uuid.New().String() + ext
	filePath := filepath.Join(a.getMediaStoragePath(), subdir, newFilename)

	// Save file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	// Return relative path
	relativePath := filepath.Join(subdir, newFilename)
	a.Log.Info("Media saved locally", "path", relativePath, "size", len(data))

	return relativePath, nil
}

// SendReactionRequest represents a request to send a reaction
type SendReactionRequest struct {
	Emoji string `json:"emoji"` // Empty string to remove reaction
}

// SendReaction sends a reaction to a message
func (a *App) SendReaction(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	messageIDStr := r.RequestCtx.UserValue("message_id").(string)

	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	// Parse request body
	var req SendReactionRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Get contact (users without full read permission can only react to messages in their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get message
	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Get WhatsApp account from the message being reacted to (not from contact, which may be stale)
	reactionAccountName := message.WhatsAppAccount
	if reactionAccountName == "" {
		reactionAccountName = contact.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, reactionAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Parse existing reactions from Metadata
	var metadata map[string]any
	if message.Metadata != nil {
		metadata = message.Metadata
	} else {
		metadata = make(map[string]any)
	}

	// Get or initialize reactions array
	type Reaction struct {
		Emoji     string `json:"emoji"`
		FromPhone string `json:"from_phone,omitempty"`
		FromUser  string `json:"from_user,omitempty"`
	}
	var reactions []Reaction
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]any); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]any); ok {
					emoji, _ := rMap["emoji"].(string)
					fromPhone, _ := rMap["from_phone"].(string)
					fromUser, _ := rMap["from_user"].(string)
					reactions = append(reactions, Reaction{
						Emoji:     emoji,
						FromPhone: fromPhone,
						FromUser:  fromUser,
					})
				}
			}
		}
	}

	// Remove existing reaction from this user (each user can only have one reaction)
	userIDStr := userID.String()
	var newReactions []Reaction
	for _, r := range reactions {
		if r.FromUser != userIDStr {
			newReactions = append(newReactions, r)
		}
	}

	// Add new reaction if emoji is not empty
	if req.Emoji != "" {
		newReactions = append(newReactions, Reaction{
			Emoji:    req.Emoji,
			FromUser: userIDStr,
		})
	}

	// Update metadata
	metadata["reactions"] = newReactions
	if err := a.DB.Model(&message).Update("metadata", metadata).Error; err != nil {
		a.Log.Error("Failed to update message reactions", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update reaction", nil, "")
	}

	// Send reaction to WhatsApp API
	go a.sendWhatsAppReaction(account, &contact, &message, req.Emoji)

	// Broadcast via WebSocket
	a.broadcastReactionUpdate(orgID, message.ID, contact.ID, newReactions)

	return r.SendEnvelope(map[string]any{
		"message_id": message.ID.String(),
		"reactions":  newReactions,
	})
}

// sendWhatsAppReaction sends a reaction to WhatsApp
func (a *App) sendWhatsAppReaction(account *models.WhatsAppAccount, contact *models.Contact, message *models.Message, emoji string) {
	if message.WhatsAppMessageID == "" {
		a.Log.Warn("Cannot send reaction - message has no WhatsApp ID", "message_id", message.ID)
		return
	}

	// GOWA accounts: use the GOWA client's reaction method (not Meta Graph API).
	if account.IsGowa() {
		provider := a.resolveProvider(account)
		gowaClient, ok := provider.(*gowa.Client)
		if !ok {
			a.Log.Error("GOWA provider not available for reaction", "account", account.Name)
			return
		}
		chatJID := gowaChatJID(contact)
		if err := gowaClient.SendReaction(context.Background(), account.ToWAAccount(), message.WhatsAppMessageID, chatJID, emoji); err != nil {
			a.Log.Error("GOWA reaction error", "error", err, "account", account.Name)
		}
		return
	}

	url := fmt.Sprintf("%s/%s/%s/messages", a.Config.WhatsApp.BaseURL, account.APIVersion, account.PhoneID)

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                contact.PhoneNumber,
		"type":              "reaction",
		"reaction": map[string]any{
			"message_id": message.WhatsAppMessageID,
			"emoji":      emoji, // Empty string removes the reaction
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		a.Log.Error("Failed to marshal reaction payload", "error", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		a.Log.Error("Failed to create reaction request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.Log.Error("Failed to send reaction", "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		a.Log.Error("WhatsApp API reaction error", "status", resp.StatusCode, "body", string(body))
		return
	}

	a.Log.Info("Reaction sent successfully", "message_id", message.WhatsAppMessageID, "emoji", emoji)
}

// TypingRequest is the body for the typing-indicator endpoint.
// gowaChatJID builds the WhatsApp JID for a GOWA API call from a contact.
// Group contacts (metadata is_group_chat, or phone_number starting with the
// WhatsApp group-ID prefix 120362/120363) need the "@g.us" suffix; newsletter
// contacts (metadata is_newsletter) need the "@newsletter" suffix; 1:1 chats
// use "@s.whatsapp.net". If the phone already contains "@" it is returned
// unchanged. Centralizing this fixes group/newsletter send/revoke/typing/
// reaction/read — previously each call site hardcoded "@s.whatsapp.net" and
// GOWA rejected group JIDs with "is not on whatsapp".
func gowaChatJID(contact *models.Contact) string {
	if contact == nil {
		return ""
	}
	phone := contact.PhoneNumber
	if phone == "" || strings.Contains(phone, "@") {
		return phone
	}
	isNewsletter := contact.Metadata != nil && contact.Metadata["is_newsletter"] == true
	if isNewsletter {
		return phone + "@newsletter"
	}
	isGroup := contact.Metadata != nil && contact.Metadata["is_group_chat"] == true
	if !isGroup && (strings.HasPrefix(phone, "120362") || strings.HasPrefix(phone, "120363")) {
		isGroup = true
	}
	if isGroup {
		return phone + "@g.us"
	}
	return phone + "@s.whatsapp.net"
}

// avatarFetchTimeout caps how long a single profile-picture lookup may take.
// GOWA's /user/avatar is usually fast, but a hung device shouldn't stall a
// whole contact sync or the lazy avatar endpoint.
const avatarFetchTimeout = 8 * time.Second

// refreshContactAvatar fetches the contact's WhatsApp profile picture (or group
// icon) via the GOWA client and persists it onto the contact row. It is
// best-effort: any error (no GOWA provider, network failure, the contact has
// hidden their picture, etc.) is logged and ignored so callers can keep
// processing other contacts. The fetched URL is cached on the contact's
// avatar_url column; callers pass force=false to skip contacts that already
// have an avatar (avoiding a GOWA round-trip on every sync).
//
// client may be nil for non-GOWA accounts (the caller decides whether to
// resolve one); in that case the function is a no-op.
func (a *App) refreshContactAvatar(client *gowa.Client, account *models.WhatsAppAccount, contact *models.Contact, deviceID string, force bool) bool {
	if client == nil || contact == nil || account == nil || !account.IsGowa() {
		return false
	}
	// Skip the round-trip when we already have a URL and the caller didn't ask
	// for a forced refresh (e.g. user clicked "refresh avatar").
	if !force && contact.AvatarURL != "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), avatarFetchTimeout)
	defer cancel()

	// GOWA's /user/avatar?phone= accepts either a bare phone or a full JID.
	// For groups/newsletters the JID (…@g.us / …@newsletter) is what resolves
	// the group icon, so we reuse the same JID builder as send/reaction paths.
	phone := contact.PhoneNumber
	if isGroupContact(contact) {
		phone = gowaChatJID(contact)
	}
	if phone == "" {
		return false
	}

	avatar, err := client.GetUserAvatar(ctx, deviceID, phone)
	if err != nil {
		a.Log.Debug("Could not fetch WhatsApp avatar",
			"contact_id", contact.ID, "phone", phone, "error", err)
		return false
	}
	if avatar == nil || avatar.URL == "" || avatar.URL == contact.AvatarURL {
		return false
	}

	if err := a.DB.Model(contact).Update("avatar_url", avatar.URL).Error; err != nil {
		a.Log.Error("Failed to persist contact avatar_url",
			"contact_id", contact.ID, "error", err)
		return false
	}
	contact.AvatarURL = avatar.URL
	return true
}

// isGroupContact reports whether the contact represents a WhatsApp group or
// newsletter (its phone_number carries the @g.us/@newsletter JID suffix or the
// 120362/120363 group-ID prefix, or it was flagged via metadata).
func isGroupContact(contact *models.Contact) bool {
	if contact == nil {
		return false
	}
	if contact.Metadata != nil && (contact.Metadata["is_group_chat"] == true || contact.Metadata["is_newsletter"] == true) {
		return true
	}
	p := contact.PhoneNumber
	if strings.Contains(p, "@g.us") || strings.Contains(p, "@newsletter") {
		return true
	}
	return strings.HasPrefix(p, "120362") || strings.HasPrefix(p, "120363")
}

// action is "start" or "stop".
type TypingRequest struct {
	Action string `json:"action"`
}

// SendTypingIndicator forwards a typing ("composing") presence to the chat's
// recipient via the GOWA send/chat-presence endpoint. This is GOWA-only: Meta
// Cloud API has no equivalent, so non-GOWA accounts get a clean 400.
// The indicator is outbound-only (it shows on the recipient's WhatsApp), so
// no WebSocket event is broadcast back to the Whatomate UI.
func (a *App) SendTypingIndicator(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req TypingRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	// Normalize the action and reject anything that is not start/stop.
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "start" && action != "stop" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, `action must be "start" or "stop"`, nil, "")
	}

	// Resolve contact (honoring per-user assignment scoping).
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	if !account.IsGowa() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Not supported for this account type", nil, "")
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not available", nil, "")
	}

	// Derive the chat JID (handles group @g.us vs 1:1 @s.whatsapp.net).
	chatJID := gowaChatJID(&contact)

	if err := gowaClient.SendChatPresence(context.Background(), account.ToWAAccount(), chatJID, action); err != nil {
		a.Log.Error("GOWA typing indicator error", "error", err, "account", account.Name, "action", action)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to send typing indicator", nil, "")
	}

	return r.SendEnvelope(map[string]any{"status": "ok", "action": action})
}

// RevokeMessageRequest is the (empty) body for the revoke endpoint. It exists
// so future fields can be added without changing the handler signature; the
// chat JID is derived server-side from the contact, never trusted from input.
type RevokeMessageRequest struct{}

// RevokeMessage unsends a message for everyone in the chat (GOWA-only) and
// marks the local message row as revoked so the UI shows a "[message revoked]"
// placeholder. It broadcasts a status_update over WebSocket so every open
// client reflects the revoked state in real time. The status and content set
// here mirror the inbound message.revoked webhook handler so the two paths
// stay consistent.
func (a *App) RevokeMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	messageIDStr := r.RequestCtx.UserValue("message_id").(string)
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	// Resolve the contact (assignment scoping applied).
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Load the message scoped to the contact (mirrors SendReaction).
	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Only outgoing messages can be revoked by the connected account.
	if message.Direction != models.DirectionOutgoing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Only outgoing messages can be revoked", nil, "")
	}
	if message.WhatsAppMessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message has no WhatsApp ID; cannot revoke", nil, "")
	}

	// Resolve the account from the message (authoritative) falling back to the
	// contact's account, exactly like SendReaction.
	revokeAccountName := message.WhatsAppAccount
	if revokeAccountName == "" {
		revokeAccountName = contact.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, revokeAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	if !account.IsGowa() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Not supported for this account type", nil, "")
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not available", nil, "")
	}

	chatJID := gowaChatJID(&contact)

	if err := gowaClient.RevokeMessage(context.Background(), account.ToWAAccount(), message.WhatsAppMessageID, chatJID); err != nil {
		a.Log.Error("GOWA revoke error", "error", err, "message_id", message.ID, "account", account.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to revoke message", nil, "")
	}

	// Persist the revoked status locally using the same values as the inbound
	// message.revoked webhook so outbound and inbound stay consistent.
	if err := a.DB.Model(&models.Message{}).Where("id = ?", message.ID).Updates(map[string]any{
		"status":  models.MessageStatusRevoked,
		"content": "[message revoked]",
	}).Error; err != nil {
		a.Log.Error("Failed to mark message as revoked after outbound revoke", "error", err, "message_id", message.ID)
	}

	// Broadcast a status_update so every open client swaps the bubble for the
	// revoked placeholder in real time.
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeStatusUpdate,
			Payload: map[string]any{
				"message_id": message.ID,
				"contact_id": message.ContactID,
				"status":     models.MessageStatusRevoked,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"message_id": message.ID.String(),
		"status":     models.MessageStatusRevoked,
	})
}

// AssignContactRequest represents the request to assign a contact to a user
type AssignContactRequest struct {
	UserID *uuid.UUID `json:"user_id"` // nil to unassign
}

// AssignContact assigns a contact to a user (agent)
// Only users with write permission can assign contacts
func (a *App) AssignContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Only users with write permission can assign contacts
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to assign contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req AssignContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// If assigning to a user, verify they exist in the same org
	if req.UserID != nil {
		var user models.User
		if err := a.DB.Where("id = ? AND organization_id = ?", req.UserID, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User not found", nil, "")
		}
	}

	// Update contact assignment
	if err := a.DB.Model(contact).Update("assigned_user_id", req.UserID).Error; err != nil {
		a.Log.Error("Failed to assign contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	// Maintain lifecycle consistency: assigning sets open, unassigning sets pending
	if req.UserID != nil {
		contact.AssignedUserID = req.UserID
		contact.SetStatus(models.ChatStatusOpen)
	} else {
		contact.AssignedUserID = nil
		contact.SetStatus(models.ChatStatusPending)
		contact.ClearCollaborators()
	}
	a.DB.Model(contact).Update("metadata", contact.Metadata)

	return r.SendEnvelope(map[string]any{
		"message":          "Contact assigned successfully",
		"assigned_user_id": req.UserID,
	})
}

// ContactSessionDataResponse represents the session data for a contact's info panel
type ContactSessionDataResponse struct {
	SessionID   *uuid.UUID     `json:"session_id,omitempty"`
	FlowID      *uuid.UUID     `json:"flow_id,omitempty"`
	FlowName    string         `json:"flow_name,omitempty"`
	SessionData map[string]any `json:"session_data"`
	PanelConfig map[string]any `json:"panel_config"`
}

// GetContactSessionData returns session data and panel configuration for a contact
// Used by the contact info panel in the chat view
func (a *App) GetContactSessionData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Verify contact belongs to org (users without full read permission can only access assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	response := ContactSessionDataResponse{
		SessionData: make(map[string]any),
		PanelConfig: map[string]any{"sections": []any{}},
	}

	// Get the most recent completed or active session for this contact
	var session models.ChatbotSession
	err = a.DB.Where("contact_id = ? AND organization_id = ?", contactID, orgID).
		Where("status IN ?", []models.SessionStatus{models.SessionStatusActive, models.SessionStatusCompleted}).
		Order("created_at DESC").
		First(&session).Error

	if err == nil {
		response.SessionID = &session.ID
		response.FlowID = session.CurrentFlowID

		// Get the flow to retrieve panel config
		// First try current_flow_id, then fall back to _flow_id in session_data
		var flowID *uuid.UUID
		if session.CurrentFlowID != nil {
			flowID = session.CurrentFlowID
		} else if flowIDStr, ok := session.SessionData["_flow_id"].(string); ok {
			if parsedID, err := uuid.Parse(flowIDStr); err == nil {
				flowID = &parsedID
			}
		}

		if flowID != nil {
			// Use cached flow to avoid DB query
			flow, err := a.getChatbotFlowByIDCached(orgID, *flowID)
			if err == nil && flow != nil {
				response.FlowName = flow.Name
				response.FlowID = flowID

				// Use panel config directly from flow (it's already JSONB/map)
				if len(flow.PanelConfig) > 0 {
					response.PanelConfig = flow.PanelConfig

					// Only include session data for configured fields (reduce payload)
					if session.SessionData != nil {
						configuredKeys := make(map[string]bool)
						if sections, ok := flow.PanelConfig["sections"].([]any); ok {
							for _, sec := range sections {
								if section, ok := sec.(map[string]any); ok {
									if fields, ok := section["fields"].([]any); ok {
										for _, f := range fields {
											if field, ok := f.(map[string]any); ok {
												if key, ok := field["key"].(string); ok {
													configuredKeys[key] = true
												}
											}
										}
									}
								}
							}
						}
						// Copy only configured fields to response
						for key := range configuredKeys {
							if val, exists := session.SessionData[key]; exists {
								response.SessionData[key] = val
							}
						}
					}
				}
			}
		}
	}

	return r.SendEnvelope(response)
}

// UpdateContactTagsRequest represents the request body for updating contact tags
type UpdateContactTagsRequest struct {
	Tags []string `json:"tags"`
}

// UpdateContactTags updates the tags on a contact
func (a *App) UpdateContactTags(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission - need contacts:write to update tags on contacts
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to update contact tags", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req UpdateContactTagsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// Convert tags to JSONBArray
	tagsArray := make(models.JSONBArray, len(req.Tags))
	for i, tag := range req.Tags {
		tagsArray[i] = tag
	}

	// Update contact tags
	if err := a.DB.Model(contact).Update("tags", tagsArray).Error; err != nil {
		a.Log.Error("Failed to update contact tags", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact tags", nil, "")
	}

	// Reload contact to get updated tags
	if err := a.DB.First(contact, contactID).Error; err != nil {
		a.Log.Error("Failed to reload contact", "error", err)
	}

	// Build response with tag details
	tags := []string{}
	if contact.Tags != nil {
		for _, t := range contact.Tags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	return r.SendEnvelope(map[string]any{
		"message": "Contact tags updated",
		"tags":    tags,
	})
}

// CreateContactRequest represents the request body for creating a contact
type CreateContactRequest struct {
	PhoneNumber     string         `json:"phone_number"`
	ProfileName     string         `json:"profile_name"`
	WhatsAppAccount string         `json:"whatsapp_account"`
	Tags            []string       `json:"tags"`
	Metadata        map[string]any `json:"metadata"`
}

// CreateContact creates a new contact or restores a soft-deleted one
func (a *App) CreateContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to create contacts", nil, "")
	}

	var req CreateContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.PhoneNumber == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_number is required", nil, "")
	}

	// Normalize phone number
	normalizedPhone := req.PhoneNumber
	if len(normalizedPhone) > 0 && normalizedPhone[0] == '+' {
		normalizedPhone = normalizedPhone[1:]
	}

	// Check if contact exists (including soft-deleted)
	var existingContact models.Contact
	if err := a.DB.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone).First(&existingContact).Error; err == nil {
		// Contact exists
		if existingContact.DeletedAt.Valid {
			// Restore soft-deleted contact
			a.DB.Unscoped().Model(&existingContact).Update("deleted_at", nil)
			existingContact.DeletedAt.Valid = false
			// Update fields
			updates := map[string]any{}
			if req.ProfileName != "" {
				updates["profile_name"] = req.ProfileName
			}
			if req.WhatsAppAccount != "" {
				updates["whats_app_account"] = req.WhatsAppAccount
			}
			if req.Tags != nil {
				tagsArray := make(models.JSONBArray, len(req.Tags))
				for i, tag := range req.Tags {
					tagsArray[i] = tag
				}
				updates["tags"] = tagsArray
			}
			if req.Metadata != nil {
				updates["metadata"] = models.JSONB(req.Metadata)
			}
			if len(updates) > 0 {
				a.DB.Model(&existingContact).Updates(updates)
			}
			// Reload contact
			a.DB.First(&existingContact, existingContact.ID)
			return r.SendEnvelope(a.buildContactResponse(&existingContact, orgID, userID))
		}
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Contact with this phone number already exists", nil, "")
	}

	// Create new contact
	contact := models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		PhoneNumber:     normalizedPhone,
		ProfileName:     req.ProfileName,
		WhatsAppAccount: req.WhatsAppAccount,
	}

	if req.Tags != nil {
		tagsArray := make(models.JSONBArray, len(req.Tags))
		for i, tag := range req.Tags {
			tagsArray[i] = tag
		}
		contact.Tags = tagsArray
	}

	if req.Metadata != nil {
		contact.Metadata = models.JSONB(req.Metadata)
	}

	if err := a.DB.Create(&contact).Error; err != nil {
		a.Log.Error("Failed to create contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create contact", nil, "")
	}

	a.logAudit(orgID, userID,
		"contact", contact.ID, models.AuditActionCreated, nil, &contact)

	return r.SendEnvelope(a.buildContactResponse(&contact, orgID, userID))
}

// UpdateContactRequest represents the request body for updating a contact.
// AssignedUserID uses *string so we can distinguish "not sent" (nil) from
// "sent as null" (pointer to empty string) to allow clearing the field.
type UpdateContactRequest struct {
	ProfileName        *string         `json:"profile_name"`
	WhatsAppAccount    *string         `json:"whatsapp_account"`
	Tags               []string        `json:"tags"`
	Metadata           *map[string]any `json:"metadata"`
	AssignedUserID     *uuid.UUID      `json:"assigned_user_id"`
	ClearAssignedAgent *bool           `json:"clear_assigned_agent"`
}

// UpdateContact updates an existing contact
func (a *App) UpdateContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to update contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req UpdateContactRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}
	oldContact := *contact

	// Build updates map
	updates := map[string]any{}

	if req.ProfileName != nil {
		updates["profile_name"] = *req.ProfileName
	}
	if req.WhatsAppAccount != nil {
		updates["whats_app_account"] = *req.WhatsAppAccount
	}
	if req.Tags != nil {
		tagsArray := make(models.JSONBArray, len(req.Tags))
		for i, tag := range req.Tags {
			tagsArray[i] = tag
		}
		updates["tags"] = tagsArray
	}
	if req.Metadata != nil {
		updates["metadata"] = models.JSONB(*req.Metadata)
	}
	if req.ClearAssignedAgent != nil && *req.ClearAssignedAgent {
		updates["assigned_user_id"] = nil
	} else if req.AssignedUserID != nil {
		var user models.User
		if err := a.DB.Where("id = ? AND organization_id = ?", req.AssignedUserID, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Assigned user not found", nil, "")
		}
		updates["assigned_user_id"] = req.AssignedUserID
	}

	if len(updates) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No fields to update", nil, "")
	}

	if err := a.DB.Model(contact).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update contact", nil, "")
	}

	// Reload contact
	a.DB.First(contact, contactID)

	a.logAudit(orgID, userID,
		"contact", contact.ID, models.AuditActionUpdated, &oldContact, contact)

	return r.SendEnvelope(a.buildContactResponse(contact, orgID, userID))
}

// DeleteContact soft-deletes a contact
func (a *App) DeleteContact(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionDelete, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to delete contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Get contact
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "Contact")
	if err != nil {
		return nil
	}

	// Soft delete the contact
	if err := a.DB.Delete(contact).Error; err != nil {
		a.Log.Error("Failed to delete contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete contact", nil, "")
	}

	a.logAudit(orgID, userID,
		"contact", contactID, models.AuditActionDeleted, contact, nil)

	return r.SendEnvelope(map[string]any{
		"message": "Contact deleted successfully",
	})
}

// filterCollaboratorsForViewer applies Ghost Mode: when the viewer is an agent
// (lacks contacts:write), admins/managers are stripped from the collaborators
// list so agents can never see that an admin is present in the chat. Admins and
// managers see the full list. The viewer's own entry is always preserved so a
// collaborator still sees themselves.
func (a *App) filterCollaboratorsForViewer(collabs []models.Collaborator, viewerID, orgID uuid.UUID) []models.Collaborator {
	if len(collabs) == 0 {
		return collabs
	}
	// Admins/managers see everyone.
	if a.HasPermission(viewerID, models.ResourceContacts, models.ActionWrite, orgID) {
		return collabs
	}
	// Agents: strip admin/manager collaborators (Ghost Mode), keep self + other agents.
	viewerIDStr := viewerID.String()
	filtered := make([]models.Collaborator, 0, len(collabs))
	for _, c := range collabs {
		if c.UserID == viewerIDStr {
			filtered = append(filtered, c)
			continue
		}
		uid, err := uuid.Parse(c.UserID)
		if err != nil {
			// Can't resolve — keep it rather than drop a legitimate agent.
			filtered = append(filtered, c)
			continue
		}
		if a.HasPermission(uid, models.ResourceContacts, models.ActionWrite, orgID) {
			continue // admin/manager → hide from agent
		}
		filtered = append(filtered, c)
	}
	return filtered
}

// buildContactResponse creates a ContactResponse from a Contact model
func (a *App) buildContactResponse(contact *models.Contact, orgID, viewerUserID uuid.UUID) ContactResponse {
	// Count unread messages
	var unreadCount int64
	a.DB.Model(&models.Message{}).
		Where("contact_id = ? AND direction = ? AND status != ?", contact.ID, models.DirectionIncoming, models.MessageStatusRead).
		Count(&unreadCount)

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
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	if shouldMask {
		phoneNumber = utils.MaskPhoneNumber(phoneNumber)
		profileName = utils.MaskIfPhoneNumber(profileName)
	}

	// 24-hour service window: open if customer messaged within the last 24 hours.
	serviceWindowOpen := contact.LastInboundAt != nil && time.Since(*contact.LastInboundAt) < 24*time.Hour

	// Load assigned user name
	assignedUserName := ""
	if contact.AssignedUserID != nil {
		var u models.User
		if a.DB.Select("full_name").First(&u, "id = ?", *contact.AssignedUserID).Error == nil {
			assignedUserName = u.FullName
		}
	}

	return ContactResponse{
		ID:                 contact.ID,
		PhoneNumber:        phoneNumber,
		Name:               profileName,
		ProfileName:        profileName,
		AvatarURL:          contact.AvatarURL,
		Status:             "active",
		Tags:               tags,
		Metadata:           contact.Metadata,
		LastMessageAt:      contact.LastMessageAt,
		LastMessagePreview: contact.LastMessagePreview,
		UnreadCount:        int(unreadCount),
		AssignedUserID:     contact.AssignedUserID,
		AssignedUserName:   assignedUserName,
		WhatsAppAccount:    contact.WhatsAppAccount,
		LastInboundAt:      contact.LastInboundAt,
		ServiceWindowOpen:  serviceWindowOpen,
		MarketingOptOut:    contact.MarketingOptOut,
		IsGroupChat:        contact.Metadata != nil && contact.Metadata["is_group_chat"] == true,
		IsNewsletter:       contact.Metadata != nil && contact.Metadata["is_newsletter"] == true,
		ChatStatus:         string(contact.EffectiveStatus()),
		Collaborators:      a.filterCollaboratorsForViewer(contact.GetCollaborators(), viewerUserID, orgID),
		CreatedAt:          contact.CreatedAt,
		UpdatedAt:          contact.UpdatedAt,
	}
}
