package sanitizer

import (
	"testing"
)

func TestSanitizeMessageContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text", "Hello world", "Hello world"},
		{"script tag", `<script>alert("xss")</script>`, ""},
		{"img onerror", `<img src=x onerror=alert(1)>`, ""},
		{"html tags stripped", "<b>bold</b> text", "bold text"},
		{"href stripped", `<a href="http://evil.com">click</a>`, "click"},
		{"iframe stripped", `<iframe src="http://evil.com"></iframe>`, ""},
		{"event handler attr", `<div onmouseover="alert(1)">text</div>`, "text"},
		{"mixed content", `Hello <script>alert(1)</script> world`, "Hello  world"},
		{"javascript protocol", `<a href="javascript:alert(1)">click</a>`, "click"},
		{"svg xss", `<svg onload="alert(1)">`, ""},
		{"style tag", `<style>body{display:none}</style>`, ""},
		{"nested script", `<<script>script>alert(1)</script>`, "&lt;"},
		{"unicode text", "こんにちは世界", "こんにちは世界"},
		{"emoji text", "Hello 🌍", "Hello 🌍"},
		{"whitespace preserved", "Hello  world", "Hello  world"},
		{"newline preserved", "line1\nline2", "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeMessageContent(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeMessageContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
