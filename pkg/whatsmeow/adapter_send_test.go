package whatsmeow

import "testing"

func TestBuildTextMessageUsesConversationForPlainText(t *testing.T) {
	text := "زاهد : الرجاء تواصل مع قسم القرطاسية"

	msg := buildTextMessage(text)
	if msg == nil {
		t.Fatal("expected message")
	}
	if got := msg.GetConversation(); got != text {
		t.Fatalf("expected conversation text %q, got %q", text, got)
	}
	if msg.GetExtendedTextMessage() != nil {
		t.Fatal("expected no extended text message")
	}
}

func TestBuildTextMessageUsesExtendedTextForURLs(t *testing.T) {
	text := "قيم تجربتك / Rate your experience:\n5 ممتاز (😍Excellent)\nhttps://g.page/r/example/review"

	msg := buildTextMessage(text)
	if msg == nil {
		t.Fatal("expected message")
	}
	if msg.GetExtendedTextMessage() == nil {
		t.Fatal("expected extended text message")
	}
	if got := msg.GetExtendedTextMessage().GetText(); got != text {
		t.Fatalf("expected text %q, got %q", text, got)
	}
	if got := msg.GetConversation(); got != "" {
		t.Fatalf("expected empty conversation payload, got %q", got)
	}
}
