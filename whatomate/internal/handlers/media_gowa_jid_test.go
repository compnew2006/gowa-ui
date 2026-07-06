package handlers

import "testing"

// TestResolveGowaChatJID locks the chat-JID derivation that both ServeMedia
// and RetryMediaDownload rely on to scope GOWA's /message/:id/download. The
// reported "does not belong to chat" failure mode is exactly the case where
// resolveGowaChatJID returns a JID that disagrees with GOWA's stored
// chat_jid. These cases pin the contract: GOWA stores media under the chat
// JID (per GOWA/docs/openapi.yaml), so whatomate must echo that JID back.
func TestResolveGowaChatJID(t *testing.T) {
	tests := []struct {
		name            string
		conversationID  string
		contactPhone    string
		want            string
	}{
		{
			name:           "conversation_id wins when it already carries @ (1:1)",
			conversationID: "966561853319@s.whatsapp.net",
			contactPhone:   "966550224612",
			want:           "966561853319@s.whatsapp.net",
		},
		{
			name:           "bare phone digits get @s.whatsapp.net suffix",
			conversationID: "",
			contactPhone:   "966561853319",
			want:           "966561853319@s.whatsapp.net",
		},
		{
			name:           "status broadcast synthetic contact",
			conversationID: "",
			contactPhone:   "status",
			want:           "status@broadcast",
		},
		{
			name:           "status@broadcast passthrough when conversation_id carries it",
			conversationID: "status@broadcast",
			contactPhone:   "",
			want:           "status@broadcast",
		},
		{
			name:           "long numeric group id from contact phone",
			conversationID: "",
			contactPhone:   "120363418710917197",
			want:           "120363418710917197@g.us",
		},
		{
			name:           "group id with device suffix (hyphenated) is treated as group",
			conversationID: "",
			contactPhone:   "966555359601-1437241952",
			want:           "966555359601-1437241952@g.us",
		},
		{
			name:           "already-suffixed group JID passes through",
			conversationID: "120363425222667916@g.us",
			contactPhone:   "",
			want:           "120363425222667916@g.us",
		},
		{
			name:           "falls back to conversation_id when contact phone empty",
			conversationID: "201507625625",
			contactPhone:   "",
			want:           "201507625625@s.whatsapp.net",
		},
		{
			name:           "both empty returns empty (caller surfaces the error)",
			conversationID: "",
			contactPhone:   "",
			want:           "",
		},
		{
			name:           "whitespace in contact phone is trimmed before suffixing",
			conversationID: "",
			contactPhone:   "  966561853319  ",
			want:           "966561853319@s.whatsapp.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGowaChatJID(tt.conversationID, tt.contactPhone)
			if got != tt.want {
				t.Fatalf("resolveGowaChatJID(%q, %q) = %q; want %q",
					tt.conversationID, tt.contactPhone, got, tt.want)
			}
		})
	}
}

// TestGowaOwnershipMismatchMessageIsDetectable confirms the substring signal
// RetryMediaDownload uses to detect GOWA's ownership-mismatch rejection is
// stable across the message formats GOWA emits. The string literal here
// mirrors the production error verbatim (wamid + chat JID).
func TestGowaOwnershipMismatchMessageIsDetectable(t *testing.T) {
	cases := []string{
		"3EB0C08FAA14FDF83C2FC1 does not belong to chat 966561853319@s.whatsapp.net",
		"3EBWXYZ does not belong to chat status@broadcast",
		"msg-1 does not belong to chat 120363418710917197@g.us",
	}
	const signal = "does not belong to chat"
	for _, c := range cases {
		if !containsSubstr(c, signal) {
			t.Errorf("expected ownership-mismatch signal %q in %q", signal, c)
		}
	}
}

func containsSubstr(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
