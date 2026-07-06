package handlers

import (
	"strings"

	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
)

func normalizePairingPhoneNumber(phone string) string {
	var b strings.Builder
	b.Grow(len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizePairClientDisplayName(displayName, fallback string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		fallback = strings.TrimSpace(fallback)
		if fallback != "" {
			return fallback
		}
		return "Chrome (Linux)"
	}
	return displayName
}

func buildPairClientDisplayName(instanceName string, instanceID uuid.UUID) string {
	name := strings.TrimSpace(instanceName)
	prefix := "Whatomate " + instanceID.String()
	if name == "" {
		return prefix
	}

	label := prefix + " - " + name
	if len(label) <= 80 {
		return label
	}

	allowedNameLen := 80 - len(prefix) - 3 // " - "
	if allowedNameLen <= 0 {
		if len(prefix) > 80 {
			return prefix[:80]
		}
		return prefix
	}
	return prefix + " - " + name[:allowedNameLen]
}

func parsePairClientType(raw string) (waClient.PairClientType, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "chrome":
		return waClient.PairClientChrome, true
	case "edge":
		return waClient.PairClientEdge, true
	case "firefox":
		return waClient.PairClientFirefox, true
	case "ie":
		return waClient.PairClientIE, true
	case "opera":
		return waClient.PairClientOpera, true
	case "safari":
		return waClient.PairClientSafari, true
	case "electron":
		return waClient.PairClientElectron, true
	case "uwp":
		return waClient.PairClientUWP, true
	case "other", "other_web", "other_web_client":
		return waClient.PairClientOtherWebClient, true
	default:
		return waClient.PairClientUnknown, false
	}
}
