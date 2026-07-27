package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Campaign recipients must be individual phone numbers — group/newsletter
// JIDs (…@g.us / …@newsletter) and their bare 12036x IDs are rejected.
func TestNormalizeRecipientPhone(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		valid bool
	}{
		{"plain number", "1234567890", "1234567890", true},
		{"plus prefix stripped", "+201234567890", "201234567890", true},
		{"surrounding spaces", " 1234567890 ", "1234567890", true},
		{"group jid", "120363322157268559@g.us", "", false},
		{"newsletter jid", "120363163799333272@newsletter", "", false},
		{"individual jid", "201234567890@s.whatsapp.net", "", false},
		{"bare group id (18 digits)", "120363322157268559", "", false},
		{"group prefix within length", "120363999999999", "", false},
		{"legacy group prefix", "120362999999999", "", false},
		{"letters", "not-a-number", "", false},
		{"too short", "12345", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeRecipientPhone(tt.in)
			assert.Equal(t, tt.valid, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
