package whatsmeow

import "testing"

func TestShouldSuppressClientError_StatusRetryNotFound(t *testing.T) {
	msg := "Failed to handle retry receipt for %s/%s from %s: %v"
	args := []interface{}{
		"status@broadcast",
		"3EB049249816077766BACF",
		"42687998775377@lid",
		"couldn't find message 3EB049249816077766BACF",
	}

	if !shouldSuppressClientError(msg, args...) {
		t.Fatal("expected status retry not found error to be suppressed")
	}
}

func TestShouldSuppressClientError_NonStatusRetryNotSuppressed(t *testing.T) {
	msg := "Failed to handle retry receipt for %s/%s from %s: %v"
	args := []interface{}{
		"201022491228@s.whatsapp.net",
		"A5EFEA120134E7444E16039D43F609B1",
		"42687998775377@lid",
		"couldn't find message A5EFEA120134E7444E16039D43F609B1",
	}

	if shouldSuppressClientError(msg, args...) {
		t.Fatal("did not expect non-status retry error to be suppressed")
	}
}

func TestShouldSuppressClientError_StatusOtherErrorNotSuppressed(t *testing.T) {
	msg := "Failed to handle retry receipt for %s/%s from %s: %v"
	args := []interface{}{
		"status@broadcast",
		"3EB049249816077766BACF",
		"42687998775377@lid",
		"failed to fetch keys",
	}

	if shouldSuppressClientError(msg, args...) {
		t.Fatal("did not expect non-not-found status retry error to be suppressed")
	}
}

func TestShouldDemoteClientError_WebSocketEOF(t *testing.T) {
	msg := "Error reading from websocket: %v"
	args := []interface{}{
		"failed to get reader: failed to read frame header: EOF",
	}

	if !shouldDemoteClientError(msg, args...) {
		t.Fatal("expected websocket EOF to be demoted")
	}
}

func TestShouldDemoteClientError_NonEOFNotDemoted(t *testing.T) {
	msg := "Error reading from websocket: %v"
	args := []interface{}{
		"failed to decrypt frame",
	}

	if shouldDemoteClientError(msg, args...) {
		t.Fatal("did not expect non-EOF websocket error to be demoted")
	}
}
