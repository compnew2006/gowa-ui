package contactutil

import (
	"regexp"
	"strings"
)

// groupJIDPattern matches the WhatsApp group JID format: digits-digits@g.us
var groupJIDPattern = regexp.MustCompile(`^\d+[-]\d+@g\.us$`)

// IsValidGroupJID validates the full WhatsApp group JID format.
// Expected format: "1203631234567-1601234567@g.us"
func IsValidGroupJID(jid string) bool {
	if jid == "" || !strings.Contains(jid, "@g.us") {
		return false
	}
	return groupJIDPattern.MatchString(jid)
}
