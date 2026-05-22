package whatsmeow

import "testing"

func TestBuildTextMessageUsesExtendedText(t *testing.T) {
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
