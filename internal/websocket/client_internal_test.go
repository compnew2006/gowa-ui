package websocket

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestClientHandleSetContact_RespectsAccessValidator(t *testing.T) {
	t.Parallel()

	hub := NewHub(logf.New(logf.Opts{}))
	client := NewClient(hub, nil, uuid.New(), uuid.New())
	targetContact := uuid.New()

	client.SetContactAccessFn(func(userID, orgID, contactID uuid.UUID) bool {
		return false
	})

	client.handleSetContact(map[string]any{"contact_id": targetContact.String()})

	assert.Nil(t, client.getCurrentContact())
}

func TestClientHandleSetContact_SetsAndClearsCurrentContact(t *testing.T) {
	t.Parallel()

	hub := NewHub(logf.New(logf.Opts{}))
	client := NewClient(hub, nil, uuid.New(), uuid.New())
	targetContact := uuid.New()

	client.SetContactAccessFn(func(userID, orgID, contactID uuid.UUID) bool {
		return contactID == targetContact
	})

	client.handleSetContact(map[string]any{"contact_id": targetContact.String()})
	require.NotNil(t, client.getCurrentContact())
	assert.Equal(t, targetContact, *client.getCurrentContact())

	client.handleSetContact(map[string]any{"contact_id": ""})
	assert.Nil(t, client.getCurrentContact())
}
