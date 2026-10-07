package models

// A conversation is "internal" (the sidebar's Private tab) when it is with one
// of the org's own connected numbers, or when a user moved it there by hand.
// Internal conversations stay out of the customer queue and every customer
// automation (away reply, close-rating prompt, call auto-reject message, daily
// reset). The account-number half needs the org's accounts, so it is resolved
// by the caller (handlers.isInternalContact); this file owns the manual flag.

// MetaInternalChat is the contact metadata key set when a user moves a
// conversation into the Private tab by hand.
const MetaInternalChat = "internal_chat"

// IsMarkedInternal reports whether a user moved this conversation into the
// Private tab by hand.
func (c *Contact) IsMarkedInternal() bool {
	return c.Metadata != nil && c.Metadata[MetaInternalChat] == true
}

// SetMarkedInternal sets or clears the manual Private-tab flag.
func (c *Contact) SetMarkedInternal(on bool) {
	if c.Metadata == nil {
		c.Metadata = JSONB{}
	}
	if on {
		c.Metadata[MetaInternalChat] = true
	} else {
		delete(c.Metadata, MetaInternalChat)
	}
}

// ExcludeInternalContactsSQL is a WHERE fragment for queries on the contacts
// table that drops internal conversations: manually flagged ones and those
// whose phone is one of the org's connected numbers (gowa_jid minus the
// "@server" part and any ":<device>" suffix).
const ExcludeInternalContactsSQL = `COALESCE(contacts.metadata->>'internal_chat', '') <> 'true'
	AND contacts.phone_number NOT IN (
		SELECT split_part(split_part(wa.gowa_jid, '@', 1), ':', 1) FROM whatsapp_accounts wa
		WHERE wa.organization_id = contacts.organization_id AND wa.gowa_jid <> '' AND wa.deleted_at IS NULL)`
