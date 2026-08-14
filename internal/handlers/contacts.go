package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/utils"
	"github.com/google/uuid"
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
// ReplyPreview contains a preview of the replied-to message
// ReactionInfo represents a reaction on a message
type ReactionInfo struct {
	Emoji     string `json:"emoji"`
	FromPhone string `json:"from_phone,omitempty"`
	FromUser  string `json:"from_user,omitempty"`
}

// ListContacts returns all contacts for the organization
// Users without contacts:read permission only see contacts assigned to them
func (a *App) ListContacts(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	// Pagination
	pg := parsePagination(r)
	search := string(r.RequestCtx.QueryArgs().Peek("search"))
	tagsParam := string(r.RequestCtx.QueryArgs().Peek("tags"))
	// When has_messages=true, hide contacts with no messages (used by /chat to
	// distinguish real conversations from synced-but-empty contacts in /settings/contacts).
	hasMessages := string(r.RequestCtx.QueryArgs().Peek("has_messages"))
	// Filter by WhatsApp account (references WhatsAppAccount.Name). Used by the
	// campaign "Add Recipients" picker to only show contacts synced under the
	// campaign's selected account.
	accountParam := string(r.RequestCtx.QueryArgs().Peek("account"))
	// When exclude_groups=true, hide group chats and newsletters. Campaign
	// recipients must be individual numbers — groups/newsletters can't receive
	// template broadcasts, so the recipient picker excludes them.
	excludeGroups := string(r.RequestCtx.QueryArgs().Peek("exclude_groups"))

	// Server-side sorting across the whole result set (not just the current
	// page). Allowed column names map to physical columns; the whatsapp_account
	// key maps to GORM's default column name whats_app_account.
	sortParam := string(r.RequestCtx.QueryArgs().Peek("sort"))
	sortDirParam := string(r.RequestCtx.QueryArgs().Peek("sort_dir"))

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

	// Scope to a single WhatsApp account when requested. The Contact model's
	// WhatsAppAccount field has no explicit column tag, so GORM's default
	// naming splits the acronym into the physical column whats_app_account.
	if accountParam != "" {
		query = query.Where("whats_app_account = ?", accountParam)
	}

	// Exclude group chats and newsletters. Group/newsletter status lives in the
	// metadata JSONB (is_group_chat/is_newsletter); older group rows are also
	// detectable by the WhatsApp group-ID phone prefixes 120362/120363.
	if excludeGroups == "true" || excludeGroups == "1" {
		query = query.Where(
			"COALESCE(metadata->>'is_group_chat', '') <> 'true' AND " +
				"COALESCE(metadata->>'is_newsletter', '') <> 'true' AND " +
				"phone_number NOT LIKE '120362%' AND phone_number NOT LIKE '120363%'")
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

	// Order by last message time (most recent first) by default. When a sort
	// column is requested, order the full result set server-side so pagination
	// stays consistent across pages.
	sortColumns := map[string]string{
		"profile_name": "profile_name",
		"phone_number": "phone_number",
		// Empty account strings (contacts never linked to an account) are
		// folded to NULL so NULLS LAST always pushes them to the bottom of
		// both directions, mirroring the "—" fallback shown in the UI.
		"whatsapp_account": "(CASE WHEN whats_app_account = '' THEN NULL ELSE whats_app_account END)",
		"last_message_at":  "last_message_at",
		"created_at":       "created_at",
	}
	orderClause := "last_message_at DESC NULLS LAST, created_at DESC"
	if col, ok := sortColumns[sortParam]; ok {
		dir := "ASC"
		if sortDirParam == "desc" {
			dir = "DESC"
		}
		orderClause = fmt.Sprintf("%s %s NULLS LAST, created_at DESC", col, dir)
	}
	query = query.Order(orderClause)

	var total int64
	query.Model(&models.Contact{}).Count(&total)

	if err := query.Offset(pg.Offset).Limit(pg.Limit).Find(&contacts).Error; err != nil {
		a.Log.Error("Failed to list contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}

	// Check if phone masking is enabled
	shouldMask := a.ShouldMaskPhoneNumbers(orgID)

	// Convert to response format (masking resolved once for the whole page)
	response := make([]ContactResponse, len(contacts))
	for i := range contacts {
		response[i] = a.buildContactResponseMasked(&contacts[i], orgID, userID, shouldMask)
	}

	return r.SendEnvelope(listEnvelope("contacts", response, total, pg))
}

// scopeAssignedContact narrows a contact query to what the user may see. It
// combines two independent gates, AND-combined:
//
//  1. Account scoping (scopeContactsByAssignedAccounts): users who have been
//     assigned specific WhatsApp accounts (user_whatsapp_accounts) only see
//     conversations belonging to those accounts. Super admins and users with
//     no assignments keep full org visibility (fallback). This is the fix for
//     scoped users seeing every account's conversations in /chat.
//
//  2. Assignment/collaborator scoping: users who lack the contacts:read
//     permission may only access contacts assigned to them
//     (assigned_user_id) or where they are a collaborator.
//
// Keeping this in one place ensures every contact endpoint enforces the same
// visibility (ListContacts, GetMessages, media serving, scheduled messages,
// … — every call site that already applied scopeAssignedContact).
func (a *App) scopeAssignedContact(query *gorm.DB, userID, orgID uuid.UUID) *gorm.DB {
	// Account scoping applies regardless of permissions: a user assigned to a
	// subset of accounts must not see other accounts' conversations even with
	// contacts:read.
	query = a.scopeContactsByAssignedAccounts(query, userID, orgID)

	if a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		return query
	}
	// Agents (users without contacts:read) can only access contacts:
	//   1. assigned to them (assigned_user_id), OR
	//   2. where they are listed as a collaborator in the contact's metadata.
	// Collaborators are stored in metadata.collaborators as a JSON array of
	// {user_id, name, role, joined_at}. The @> containment operator reuses the
	// same pattern as the tags filter above and leverages the GIN index.
	collaboratorJSON := fmt.Sprintf(`{"collaborators":[{"user_id":"%s"}]}`, userID.String())
	return query.Where(
		"assigned_user_id = ? OR metadata @> ?::jsonb",
		userID,
		collaboratorJSON,
	)
}

// scopeContactsByAssignedAccounts narrows a contacts query to the WhatsApp
// accounts explicitly assigned to the user via user_whatsapp_accounts. It is
// the contacts-table mirror of scopeAccountsToUser (accounts.go), which scopes
// the /settings/accounts picker. Because contacts reference accounts by Name
// (a soft string, not a FK — see AGENTS.md), the assigned account IDs are
// resolved to names before filtering.
//
// Bypass rules (identical to scopeAccountsToUser):
//   - super admins see every account;
//   - users with NO assignments fall back to full org visibility (so org
//     owners and pre-assignment users are unaffected).
//
// A load error fails CLOSED (returns a query that matches nothing): this is a
// visibility gate, and leaking every account's conversations during a DB
// hiccup is worse than showing an empty list until the error clears.
func (a *App) scopeContactsByAssignedAccounts(query *gorm.DB, userID, orgID uuid.UUID) *gorm.DB {
	if a.IsSuperAdmin(userID) {
		return query
	}
	ids, err := a.assignedAccountIDs(userID, orgID)
	if err != nil {
		a.Log.Error("Failed to load account assignments for contact scoping",
			"error", err, "user_id", userID)
		return query.Where("1 = 0")
	}
	if len(ids) == 0 {
		return query // no assignments — full org visibility (fallback)
	}
	var names []string
	if err := a.DB.Model(&models.WhatsAppAccount{}).
		Where("id IN ? AND organization_id = ?", ids, orgID).
		Pluck("name", &names).Error; err != nil {
		a.Log.Error("Failed to resolve assigned account names for contact scoping",
			"error", err, "user_id", userID)
		return query.Where("1 = 0")
	}
	if len(names) == 0 {
		// Assigned to accounts that no longer exist in this org — show nothing
		// rather than leaking unrelated conversations.
		return query.Where("1 = 0")
	}
	return query.Where("whats_app_account IN ?", names)
}

// findScopedContact loads a contact by ID through scopeAssignedContact so
// write endpoints (assign, tags, update, delete, lifecycle actions, custom
// actions) enforce exactly the same visibility as the read path. Sends a 404
// envelope on miss and returns errEnvelopeSent, mirroring findByIDAndOrg.
func (a *App) findScopedContact(r *fastglue.Request, id, userID, orgID uuid.UUID) (*models.Contact, error) {
	var contact models.Contact
	query := a.scopeAssignedContact(a.DB.Where("id = ? AND organization_id = ?", id, orgID), userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		return nil, errEnvelopeSent
	}
	return &contact, nil
}

// GetContact returns a single contact
// Users without contacts:read permission can only access contacts assigned to them
func (a *App) GetContact(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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

// GetMessages returns messages for a contact
// Agents can only access messages for their assigned contacts
// Supports cursor-based pagination with before_id for loading older messages
// AssignContactRequest represents the request to assign a contact to a user
type AssignContactRequest struct {
	UserID *uuid.UUID `json:"user_id"` // nil to unassign
}

// AssignContact assigns a contact to a user (agent)
// Only users with write permission can assign contacts
func (a *App) AssignContact(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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

	// Get contact (scoped: account-restricted users can only touch contacts
	// under their assigned accounts, matching what they can see)
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
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

	// Delegate to the lifecycle service: it persists the assignment, keeps
	// the status consistent (assign → open, unassign → release/pending),
	// writes the "X assigned this conversation to Y" system message + audit
	// entry, and broadcasts so all clients update without a refresh.
	if err := a.ChatLifecycle.Assign(r.RequestCtx, orgID, userID, contact, req.UserID); err != nil {
		a.Log.Error("Failed to assign contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to assign contact", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"message":          "Contact assigned successfully",
		"assigned_user_id": req.UserID,
	})
}

// UpdateContactTagsRequest represents the request body for updating contact tags
type UpdateContactTagsRequest struct {
	Tags []string `json:"tags"`
}

// UpdateContactTags updates the tags on a contact
func (a *App) UpdateContactTags(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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

	// Get contact (scoped to the user's visible accounts — see AssignContact)
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
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
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
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

	// Get contact (scoped to the user's visible accounts — see AssignContact)
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
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
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionDelete, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to delete contacts", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Get contact (scoped to the user's visible accounts — see AssignContact)
	contact, err := a.findScopedContact(r, contactID, userID, orgID)
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

// buildContactResponse creates a ContactResponse from a Contact model.
func (a *App) buildContactResponse(contact *models.Contact, orgID, viewerUserID uuid.UUID) ContactResponse {
	return a.buildContactResponseMasked(contact, orgID, viewerUserID, a.ShouldMaskPhoneNumbers(orgID))
}

// buildContactResponseMasked is buildContactResponse with the phone-masking
// decision supplied by the caller, so list endpoints can resolve it once per
// page instead of once per contact.
func (a *App) buildContactResponseMasked(contact *models.Contact, orgID, viewerUserID uuid.UUID, shouldMask bool) ContactResponse {
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
		AvatarURL:          contactAvatarURL(contact),
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
