package models

import "strings"

// Canonical WhatsApp account connection states.
//
// The API and the UI speak exactly this enum: Connected / Disconnected /
// Reconnecting. It is the single vocabulary for "is this WhatsApp device
// live", end to end — stored value, API response, WebSocket payload and
// rendered badge.
//
// History: writers used to store ad-hoc lowercase strings ("active" for a
// connected device, "reconnecting", "disconnected"), and the API passed the
// stored string straight through. The UI then rendered the raw value when
// the live GOWA probe was unavailable, so an account showed a bare
// lowercase "active" instead of a Connected badge. New writers must only
// store these constants; NormalizeAccountStatus maps legacy rows on read.
const (
	// AccountStatusConnected is a live, connected WhatsApp device.
	AccountStatusConnected = "Connected"
	// AccountStatusDisconnected covers logged-out devices and accounts whose
	// session was never verified (including newly created ones).
	AccountStatusDisconnected = "Disconnected"
	// AccountStatusReconnecting is a transient reconnect attempt.
	AccountStatusReconnecting = "Reconnecting"
)

// NormalizeAccountStatus maps any stored or received account status onto the
// canonical enum. Legacy lowercase values map to their connected meaning
// ("active" was the old word for a connected device); empty or unrecognised
// values fail closed to Disconnected — the UI must never claim a device is
// connected on the basis of a value it does not recognise. No row is ever
// dropped or hidden because of its stored status.
func NormalizeAccountStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "connected", "active":
		return AccountStatusConnected
	case "reconnecting":
		return AccountStatusReconnecting
	case "disconnected", "":
		return AccountStatusDisconnected
	default:
		return AccountStatusDisconnected
	}
}
