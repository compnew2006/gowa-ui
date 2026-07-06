package whatsmeow

import (
	"testing"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	waStore "go.mau.fi/whatsmeow/store"
	"google.golang.org/protobuf/proto"
)

func TestBuildLinkedDeviceName_UsesInstanceName(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("", "  Sales Team  ", instanceID)
	if got != "Sales Team" {
		t.Fatalf("expected trimmed instance name, got %q", got)
	}
}

func TestBuildLinkedDeviceName_UsesFallback(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("", "   ", instanceID)
	if got != "Whatomate de305d54" {
		t.Fatalf("unexpected fallback device name: %q", got)
	}
}

func TestBuildLinkedDeviceName_UsesIdentityPrefix(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("whats", "Support Team", instanceID)
	if got != "whats+Support Team" {
		t.Fatalf("unexpected prefixed device name: %q", got)
	}
}

func TestBuildLinkedDeviceName_UsesIdentityPrefixWithUnnamedInstance(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("whats", "   ", instanceID)
	if got != "whats+de305d54" {
		t.Fatalf("unexpected fallback prefixed device name: %q", got)
	}
}

func TestBuildLinkedDeviceName_UsesIdentityPrefixWithoutDuplicatingPlus(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("whats+", "Support Team", instanceID)
	if got != "whats+Support Team" {
		t.Fatalf("unexpected prefixed device name with + suffix: %q", got)
	}
}

func TestBuildLinkedDeviceName_TruncatesLongName(t *testing.T) {
	instanceID := uuid.MustParse("de305d54-75b4-431b-adb2-eb6b9e546014")
	got := buildLinkedDeviceName("", "12345678901234567890123456789012345678901234567890123456789012345", instanceID)
	if len([]rune(got)) != maxLinkedDeviceNameRunes {
		t.Fatalf("expected %d runes, got %d (%q)", maxLinkedDeviceNameRunes, len([]rune(got)), got)
	}
}

func TestApplyLinkedDeviceName_UpdatesPayload(t *testing.T) {
	propsBytes, err := proto.Marshal(proto.Clone(waStore.DeviceProps).(*waCompanionReg.DeviceProps))
	if err != nil {
		t.Fatalf("marshal default device props: %v", err)
	}

	payload := &waWa6.ClientPayload{
		UserAgent: &waWa6.ClientPayload_UserAgent{},
		DevicePairingData: &waWa6.ClientPayload_DevicePairingRegistrationData{
			DeviceProps: propsBytes,
		},
	}

	applyLinkedDeviceName(payload, "Support Team")

	if payload.GetUserAgent().GetDevice() != "Support Team" {
		t.Fatalf("expected user agent device to be updated, got %q", payload.GetUserAgent().GetDevice())
	}

	var props waCompanionReg.DeviceProps
	if err := proto.Unmarshal(payload.GetDevicePairingData().GetDeviceProps(), &props); err != nil {
		t.Fatalf("unmarshal updated device props: %v", err)
	}
	if props.GetOs() != "Support Team" {
		t.Fatalf("expected device props OS to be updated, got %q", props.GetOs())
	}
}
