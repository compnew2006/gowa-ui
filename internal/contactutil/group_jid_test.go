package contactutil

import "testing"

func TestIsValidGroupJID(t *testing.T) {
	tests := []struct {
		jid    string
		expect bool
	}{
		{"1203631234567-1601234567@g.us", true},
		{"1234567890-1234567890@g.us", true},
		{"0-0@g.us", true},
		{"", false},
		{"not-a-jid", false},
		{"1203631234567-1601234567@s.whatsapp.net", false},
		{"12036312345671601234567@g.us", false},    // missing hyphen
		{"1203631234567-@g.us", false},              // empty server part after hyphen
		{"-1601234567@g.us", false},                     // empty before hyphen
		{"12345@g.us", false},                              // no hyphen
		{"1203631234567-1601234567@g.u", false},      // truncated
	{"1203631234567-1601234567@g.use", false},     // extra char
	}

	for _, tt := range tests {
		t.Run(tt.jid, func(t *testing.T) {
			got := IsValidGroupJID(tt.jid)
			if got != tt.expect {
				t.Errorf("IsValidGroupJID(%q) = %v, want %v", tt.jid, got, tt.expect)
			}
		})
	}
}
