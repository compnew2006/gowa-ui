package handlers

import (
	"sort"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Internal conversations (the sidebar's Private tab) are chats between two of
// the org's OWN connected WhatsApp numbers (e.g. the Saudi account messaging
// the Egypt account), plus any conversation a user moved there by hand
// (models.MetaInternalChat). Each side of an account-to-account chat stores
// the other number as an ordinary contact, so without this marker they mix
// into the customer queue and trigger customer automations against a
// colleague's number. Every customer automation must check isInternalContact.

// accountPhoneFromJID extracts the bare phone from an account's connected
// JID, dropping any ":<device>" suffix ("9665...:12@s.whatsapp.net").
func accountPhoneFromJID(jid string) string {
	phone := gowa.PhoneFromJID(jid)
	if idx := strings.Index(phone, ":"); idx > 0 {
		phone = phone[:idx]
	}
	return phone
}

// orgAccountPhones maps each connected org account's phone number to its
// account Name. Accounts that never reported a JID are skipped.
func (a *App) orgAccountPhones(orgID uuid.UUID) map[string]string {
	var accounts []models.WhatsAppAccount
	if err := a.DB.Select("name", "gowa_jid").
		Where("organization_id = ? AND gowa_jid <> ''", orgID).
		Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to load org account phones", "error", err, "org_id", orgID)
		return nil
	}
	phones := make(map[string]string, len(accounts))
	for _, acc := range accounts {
		if phone := accountPhoneFromJID(acc.GowaJID); phone != "" {
			phones[phone] = acc.Name
		}
	}
	return phones
}

// isInternalPhone reports whether phone is one of the org's own connected
// numbers, i.e. the conversation is with a colleague, not a customer.
func (a *App) isInternalPhone(orgID uuid.UUID, phone string) bool {
	if phone == "" {
		return false
	}
	_, ok := a.orgAccountPhones(orgID)[phone]
	return ok
}

// isInternalContact is the single check every customer automation uses:
// the conversation was moved to the Private tab by hand, or its phone is one
// of the org's own connected numbers.
func (a *App) isInternalContact(orgID uuid.UUID, contact *models.Contact) bool {
	return contact.IsMarkedInternal() || a.isInternalPhone(orgID, contact.PhoneNumber)
}

// markInternalContact flags resp as an internal conversation when the contact
// was moved to the Private tab by hand or its (unmasked) phone is one of the
// org's own numbers.
func markInternalContact(resp *ContactResponse, contact *models.Contact, accountPhones map[string]string) {
	if name, ok := accountPhones[contact.PhoneNumber]; ok {
		resp.IsInternal = true
		resp.InternalAccountName = name
	}
	if contact.IsMarkedInternal() {
		resp.IsInternal = true
		resp.InternalMarked = true
	}
}

// SetContactInternalRequest moves a conversation into (or out of) the
// Private tab by hand.
type SetContactInternalRequest struct {
	Internal bool `json:"internal"`
}

// SetContactInternal flags a conversation as internal by hand, so it is
// treated like a chat with one of the org's own numbers: Private tab only, no
// customer automations. Any user who can write to the conversation may do it
// (historical read-only grant holders cannot). Clearing the flag on an org
// number's contact leaves it internal — that half comes from the phone.
// Route: PUT /api/contacts/{id}/internal  Permission: chat:write
func (a *App) SetContactInternal(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	var req SetContactInternalRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	contact, err := a.findScopedMutableContact(r, contactID, userID, orgID)
	if err != nil {
		return nil
	}

	// Patch the one key in SQL so a concurrent metadata write (chat_status,
	// collaborators) is not overwritten with a stale copy.
	was := contact.IsMarkedInternal()
	metadata := gorm.Expr("COALESCE(metadata, '{}'::jsonb) - ?::text", models.MetaInternalChat)
	if req.Internal {
		metadata = gorm.Expr("COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(?::text, true)", models.MetaInternalChat)
	}
	if err := a.DB.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", metadata).Error; err != nil {
		a.Log.Error("Failed to update contact internal flag", "error", err, "contact_id", contact.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update conversation", nil, "")
	}
	if err := a.DB.First(contact, "id = ?", contact.ID).Error; err != nil {
		a.Log.Error("Failed to reload contact", "error", err, "contact_id", contact.ID)
	}

	if was != req.Internal {
		a.logAudit(orgID, userID, "contact", contact.ID, models.AuditActionUpdated,
			map[string]any{models.MetaInternalChat: was}, map[string]any{models.MetaInternalChat: req.Internal})
		if a.WSHub != nil {
			a.WSHub.BroadcastToUsers(orgID, a.wsContactRecipients(contact, orgID), websocket.WSMessage{
				Type: websocket.TypeContactUpdate,
				Payload: map[string]any{
					"contact_id":      contact.ID.String(),
					"internal_marked": req.Internal,
				},
			})
		}
	}
	return r.SendEnvelope(a.buildContactResponse(contact, orgID, userID))
}

// InternalConversationSide is one account's copy of an internal conversation.
// Every message between two org numbers is stored twice — once per account —
// and Account's copies of its chat with PeerAccount live on ContactID (the
// contact whose phone is PeerAccount's connected number).
type InternalConversationSide struct {
	ContactID          uuid.UUID  `json:"contact_id"`
	Account            string     `json:"account"`
	PeerAccount        string     `json:"peer_account"`
	LastMessageAt      *time.Time `json:"last_message_at"`
	LastMessagePreview string     `json:"last_message_preview"`
	UnreadCount        int        `json:"unread_count"`
}

// InternalConversation is ONE chat between two org accounts, merging both
// sides so the sidebar lists it once. Accounts is sorted; Key joins it.
type InternalConversation struct {
	Key                string                     `json:"key"`
	Accounts           [2]string                  `json:"accounts"`
	LastMessageAt      *time.Time                 `json:"last_message_at"`
	LastMessagePreview string                     `json:"last_message_preview"`
	UnreadCount        int                        `json:"unread_count"`
	Sides              []InternalConversationSide `json:"sides"`
}

// ListInternalConversations returns one entry per pair of org accounts that
// have messaged each other. Visibility follows ListContacts: each side's
// contact goes through scopeAssignedContact, so a viewer who can reach only
// one side gets a single-sided entry.
// Route: GET /api/contacts/internal-conversations
func (a *App) ListInternalConversations(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	convs, err := a.internalConversations(orgID, userID)
	if err != nil {
		a.Log.Error("Failed to list internal conversations", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list internal conversations", nil, "")
	}
	return r.SendEnvelope(map[string]any{"conversations": convs})
}

func (a *App) internalConversations(orgID, userID uuid.UUID) ([]InternalConversation, error) {
	convs := []InternalConversation{}
	accountPhones := a.orgAccountPhones(orgID)
	if len(accountPhones) < 2 {
		return convs, nil
	}
	phones := make([]string, 0, len(accountPhones))
	accountNames := make([]string, 0, len(accountPhones))
	for phone, name := range accountPhones {
		phones = append(phones, phone)
		accountNames = append(accountNames, name)
	}

	var contacts []models.Contact
	query := a.scopeAssignedContact(a.ScopeToOrg(a.DB, userID, orgID), userID, orgID)
	if err := query.Where("phone_number IN ?", phones).Find(&contacts).Error; err != nil {
		return nil, err
	}
	if len(contacts) == 0 {
		return convs, nil
	}
	peerByContact := make(map[uuid.UUID]string, len(contacts))
	contactIDs := make([]uuid.UUID, 0, len(contacts))
	for _, c := range contacts {
		peerByContact[c.ID] = accountPhones[c.PhoneNumber]
		contactIDs = append(contactIDs, c.ID)
	}

	// Latest message per (contact, account). System messages carry no
	// account, so the account filter drops them.
	type latestRow struct {
		ContactID       uuid.UUID
		WhatsAppAccount string
		CreatedAt       time.Time
		MessageType     models.MessageType
		Content         string
	}
	var latestRows []latestRow
	latest := a.DB.Model(&models.Message{}).
		Select("contact_id, whats_app_account, created_at, message_type, content, "+
			"ROW_NUMBER() OVER (PARTITION BY contact_id, whats_app_account ORDER BY created_at DESC) AS row_num").
		Where("organization_id = ? AND contact_id IN ? AND whats_app_account IN ?", orgID, contactIDs, accountNames)
	if err := a.DB.Table("(?) AS latest_internal_messages", latest).
		Select("contact_id, whats_app_account, created_at, message_type, content").
		Where("row_num = ?", 1).
		Scan(&latestRows).Error; err != nil {
		return nil, err
	}

	type unreadRow struct {
		ContactID       uuid.UUID
		WhatsAppAccount string
		UnreadCount     int
	}
	var unreadRows []unreadRow
	if err := a.DB.Model(&models.Message{}).
		Select("contact_id, whats_app_account, COUNT(*) AS unread_count").
		Where("organization_id = ? AND contact_id IN ? AND whats_app_account IN ? AND direction = ? AND status NOT IN ?",
			orgID, contactIDs, accountNames, models.DirectionIncoming,
			[]models.MessageStatus{models.MessageStatusRead, models.MessageStatusRevoked, models.MessageStatusFailed}).
		Group("contact_id, whats_app_account").
		Scan(&unreadRows).Error; err != nil {
		return nil, err
	}
	type sideKey struct {
		contactID uuid.UUID
		account   string
	}
	unread := make(map[sideKey]int, len(unreadRows))
	for _, row := range unreadRows {
		unread[sideKey{row.ContactID, row.WhatsAppAccount}] = row.UnreadCount
	}

	byKey := map[string]*InternalConversation{}
	for _, row := range latestRows {
		peer := peerByContact[row.ContactID]
		if peer == "" || peer == row.WhatsAppAccount {
			continue // an account messaging its own number is not a pair
		}
		at := row.CreatedAt
		side := InternalConversationSide{
			ContactID:          row.ContactID,
			Account:            row.WhatsAppAccount,
			PeerAccount:        peer,
			LastMessageAt:      &at,
			LastMessagePreview: getMessagePreviewFromContent(row.MessageType, row.Content),
			UnreadCount:        unread[sideKey{row.ContactID, row.WhatsAppAccount}],
		}
		pair := [2]string{side.Account, side.PeerAccount}
		if pair[0] > pair[1] {
			pair[0], pair[1] = pair[1], pair[0]
		}
		key := pair[0] + "|" + pair[1]
		conv, ok := byKey[key]
		if !ok {
			conv = &InternalConversation{Key: key, Accounts: pair}
			byKey[key] = conv
		}
		conv.Sides = append(conv.Sides, side)
		conv.UnreadCount += side.UnreadCount
		if conv.LastMessageAt == nil || at.After(*conv.LastMessageAt) {
			conv.LastMessageAt = &at
			conv.LastMessagePreview = side.LastMessagePreview
		}
	}

	for _, conv := range byKey {
		sort.Slice(conv.Sides, func(i, j int) bool { return conv.Sides[i].Account < conv.Sides[j].Account })
		convs = append(convs, *conv)
	}
	sort.Slice(convs, func(i, j int) bool {
		if !convs[i].LastMessageAt.Equal(*convs[j].LastMessageAt) {
			return convs[i].LastMessageAt.After(*convs[j].LastMessageAt)
		}
		return convs[i].Key < convs[j].Key
	})
	return convs, nil
}
