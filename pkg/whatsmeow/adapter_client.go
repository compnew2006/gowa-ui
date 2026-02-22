package whatsmeow

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// getClient validates and returns a connected WhatsApp client for the given instance.
func (a *WhatsmeowAdapter) getClient(instanceID string) (*whatsmeow.Client, error) {
	uid, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}
	client := a.manager.GetClient(uid)
	if client == nil {
		return nil, fmt.Errorf("instance not connected")
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("instance disconnected")
	}
	return client, nil
}

// parseJID handles individual (@s.whatsapp.net), group (@g.us), and newsletter (@newsletter) JIDs.
// Plain phone numbers are treated as individual contacts.
func (a *WhatsmeowAdapter) parseJID(to string) (types.JID, error) {
	if strings.Contains(to, "@") {
		jid, err := types.ParseJID(to)
		if err != nil {
			return types.JID{}, fmt.Errorf("invalid JID %q: %w", to, err)
		}
		return jid, nil
	}

	jid, err := types.ParseJID(to + "@s.whatsapp.net")
	if err != nil {
		return types.JID{}, fmt.Errorf("failed to parse JID for %q: %w", to, err)
	}
	return jid, nil
}

// isGroupJID returns true if the given string looks like a group JID.
func isGroupJID(jidStr string) bool {
	return strings.HasSuffix(jidStr, "@g.us")
}
