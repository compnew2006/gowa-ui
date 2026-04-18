package whatsmeow

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// getClient validates and returns a connected WhatsApp client for the given instance.
func (a *WhatsmeowAdapter) getClient(ctx context.Context, instanceID string) (*whatsmeow.Client, error) {
	uid, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}

	client := a.runtimeClient(uid)
	if client != nil && client.IsConnected() {
		return client, nil
	}

	if reconnectErr := a.reconnectClient(ctx, uid, client); reconnectErr != nil {
		a.logger.Warn(
			"Whatsmeow adapter reconnect attempt failed",
			"instance_id", uid,
			"error", reconnectErr,
		)
	}

	client = a.runtimeClient(uid)
	if client == nil {
		return nil, fmt.Errorf("instance not connected")
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("instance disconnected")
	}

	return client, nil
}

func (a *WhatsmeowAdapter) runtimeClient(instanceID uuid.UUID) *whatsmeow.Client {
	if a == nil {
		return nil
	}
	if a.getRuntimeClient != nil {
		return a.getRuntimeClient(instanceID)
	}
	if a.manager == nil {
		return nil
	}
	return a.manager.GetClient(instanceID)
}

func (a *WhatsmeowAdapter) reconnectClient(ctx context.Context, instanceID uuid.UUID, current *whatsmeow.Client) error {
	if a == nil {
		return fmt.Errorf("whatsmeow adapter not initialized")
	}
	if a.connectRuntime == nil {
		if a.manager == nil {
			return fmt.Errorf("whatsmeow manager not initialized")
		}
		a.connectRuntime = a.manager.Connect
	}

	a.logger.Warn(
		"Whatsmeow adapter forcing reconnect for unavailable runtime client",
		"instance_id", instanceID,
		"had_client", current != nil,
		"was_connected", current != nil && current.IsConnected(),
	)

	reconnectCtx := ctx
	if reconnectCtx == nil {
		reconnectCtx = context.Background()
	}
	timeout := a.reconnectTimeout
	if timeout <= 0 {
		timeout = defaultAdapterReconnectTimeout
	}
	reconnectCtx, cancel := context.WithTimeout(context.WithoutCancel(reconnectCtx), timeout)
	defer cancel()

	return a.connectRuntime(reconnectCtx, instanceID)
}

// parseJID handles individual (@s.whatsapp.net), group (@g.us), and newsletter (@newsletter) JIDs.
// Plain phone numbers are treated as individual contacts.
func (a *WhatsmeowAdapter) parseJID(to string) (types.JID, error) {
	if strings.Contains(to, "@") {
		jid, err := types.ParseJID(to)
		if err != nil {
			return types.JID{}, fmt.Errorf("invalid JID %q: %w", to, err)
		}
		return jid, nil
	}

	jid, err := types.ParseJID(to + "@s.whatsapp.net")
	if err != nil {
		return types.JID{}, fmt.Errorf("failed to parse JID for %q: %w", to, err)
	}
	return jid, nil
}

// isGroupJID returns true if the given string looks like a group JID.
func isGroupJID(jidStr string) bool {
	return strings.HasSuffix(jidStr, "@g.us")
}
