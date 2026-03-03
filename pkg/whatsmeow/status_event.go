package whatsmeow

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
)

func isStatusJID(jid types.JID) bool {
	return jid.ToNonAD() == types.StatusBroadcastJID
}

func isStatusMessageSource(source types.MessageSource) bool {
	if isStatusJID(source.Chat) {
		return true
	}
	if isStatusJID(source.RecipientAlt) {
		return true
	}
	if isStatusJID(source.BroadcastListOwner) {
		return true
	}

	return false
}

func isStatusMessageInfo(info types.MessageInfo) bool {
	if isStatusMessageSource(info.MessageSource) {
		return true
	}

	if info.DeviceSentMeta != nil {
		destination := strings.TrimSpace(info.DeviceSentMeta.DestinationJID)
		if destination == types.StatusBroadcastJID.String() {
			return true
		}
	}

	if strings.EqualFold(strings.TrimSpace(info.Category), "status") {
		return true
	}

	return false
}
