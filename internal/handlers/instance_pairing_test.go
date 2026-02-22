package handlers

import (
	"testing"

	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
)

func TestNormalizePairingPhoneNumber(t *testing.T) {
	t.Parallel()

	got := normalizePairingPhoneNumber("+1 (555) 123-4567")
	if got != "15551234567" {
		t.Fatalf("expected normalized digits, got %q", got)
	}
}

func TestNormalizePairClientDisplayName(t *testing.T) {
	t.Parallel()

	if got := normalizePairClientDisplayName("  ", "Whatomate Support (abc12345)"); got != "Whatomate Support (abc12345)" {
		t.Fatalf("expected fallback display name, got %q", got)
	}
	if got := normalizePairClientDisplayName("  ", ""); got != "Chrome (Linux)" {
		t.Fatalf("expected default display name, got %q", got)
	}
	if got := normalizePairClientDisplayName("Safari (MacOS)", "Whatomate Support (abc12345)"); got != "Safari (MacOS)" {
		t.Fatalf("expected explicit display name, got %q", got)
	}
}

func TestBuildPairClientDisplayName(t *testing.T) {
	t.Parallel()

	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildPairClientDisplayName("Support Team", instanceID)
	if got != "Whatomate de305d54-75b4-431b-adb2-eb6b9e546014 - Support Team" {
		t.Fatalf("unexpected display name: %q", got)
	}
}

func TestParsePairClientType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   waClient.PairClientType
		wantOK bool
	}{
		{name: "default empty", input: "", want: waClient.PairClientChrome, wantOK: true},
		{name: "chrome", input: "chrome", want: waClient.PairClientChrome, wantOK: true},
		{name: "firefox case insensitive", input: "FireFox", want: waClient.PairClientFirefox, wantOK: true},
		{name: "other alias", input: "other_web_client", want: waClient.PairClientOtherWebClient, wantOK: true},
		{name: "invalid", input: "netscape", want: waClient.PairClientUnknown, wantOK: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parsePairClientType(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("input %q => got (%v,%v), want (%v,%v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
