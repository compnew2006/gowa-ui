package whatsmeow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	waClient "go.mau.fi/whatsmeow"
)

const phonePairingTimeoutSec = 160

// PairPhoneOptions configures phone-code pairing behavior.
type PairPhoneOptions struct {
	ShowPushNotification bool
	ClientType           waClient.PairClientType
	ClientDisplayName    string
}

// DefaultPairPhoneOptions returns production-safe defaults for phone-code pairing.
func DefaultPairPhoneOptions() PairPhoneOptions {
	return PairPhoneOptions{
		ShowPushNotification: true,
		ClientType:           waClient.PairClientChrome,
		ClientDisplayName:    "Chrome (Linux)",
	}
}

// PhonePairingTimeoutSec returns the nominal pairing window used for UI countdowns.
func PhonePairingTimeoutSec() int {
	return phonePairingTimeoutSec
}

// RequestPhonePairingCode requests a WhatsApp phone linking code for an unpaired instance.
func (cm *ConnectionManager) RequestPhonePairingCode(ctx context.Context, instanceID uuid.UUID, phoneNumber string, opts PairPhoneOptions) (string, error) {
	phoneDigits := normalizePhoneDigits(phoneNumber)
	if phoneDigits == "" {
		return "", fmt.Errorf("phone number is required")
	}

	if opts.ClientType == waClient.PairClientUnknown {
		opts.ClientType = waClient.PairClientChrome
	}
	opts.ClientDisplayName = strings.TrimSpace(opts.ClientDisplayName)
	if opts.ClientDisplayName == "" {
		opts.ClientDisplayName = "Chrome (Linux)"
	}

	client, err := cm.ensurePairingClient(ctx, instanceID)
	if err != nil {
		return "", err
	}

	if client.Store != nil && client.Store.ID != nil {
		return "", fmt.Errorf("instance is already paired")
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		if !client.IsConnected() {
			if time.Now().After(deadline) {
				return "", fmt.Errorf("client is not connected")
			}
			time.Sleep(250 * time.Millisecond)
			continue
		}

		code, pairErr := client.PairPhone(ctx, phoneDigits, opts.ShowPushNotification, opts.ClientType, opts.ClientDisplayName)
		if pairErr == nil {
			cm.logger.Info(
				"Phone pairing code generated",
				"component", "whatsmeow",
				"event", "pair_code_generated",
				"instance_id", instanceID,
			)
			return code, nil
		}

		if isRetryablePairPhoneError(pairErr) && time.Now().Before(deadline) {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		return "", pairErr
	}
}

func (cm *ConnectionManager) ensurePairingClient(ctx context.Context, instanceID uuid.UUID) (*waClient.Client, error) {
	client := cm.GetClient(instanceID)
	if client == nil {
		if err := cm.Connect(ctx, instanceID); err != nil {
			return nil, err
		}
		client = cm.GetClient(instanceID)
	}
	if client == nil {
		return nil, fmt.Errorf("instance client is not available")
	}

	if client.Store != nil && client.Store.ID != nil {
		return client, nil
	}

	if client.IsConnected() {
		return client, nil
	}

	if err := cm.updateInstanceStatus(ctx, instanceID, models.InstanceStatusConnecting); err != nil {
		cm.logger.Warn("Failed to set connecting status before phone pairing", "instance_id", instanceID, "error", err)
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect client for phone pairing: %w", err)
	}

	return client, nil
}

func isRetryablePairPhoneError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, waClient.ErrIQDisconnected) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "disconnected") ||
		strings.Contains(msg, "connection reset")
}

func normalizePhoneDigits(phone string) string {
	var b strings.Builder
	b.Grow(len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

