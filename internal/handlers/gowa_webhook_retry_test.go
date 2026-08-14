package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failed inbox event must be gated by a retry backoff. Without it the
// drainAll loop re-claimed the just-failed row immediately and burned all 5
// attempts in milliseconds.
func TestGowaWebhookProcessor_RetryBackoff(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// RawBody is valid JSON but not an object — unmarshalling into the
	// envelope struct fails, giving a deterministic processing error.
	// EventKey is unique per run: the partial idempotency index collides
	// with rows left by previous test processes.
	evt := &models.GowaWebhookEvent{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		AccountID:      uuid.New(),
		DeviceID:       "dev-backoff",
		Event:          "message",
		EventKey:       "backoff-" + uuid.New().String()[:8],
		RawBody:        json.RawMessage(`"not-an-object"`),
		Status:         models.GowaWebhookEventPending,
	}
	require.NoError(t, app.DB.Create(evt).Error)

	proc := handlers.NewGowaWebhookProcessor(app, time.Hour)
	require.Equal(t, 1, proc.ProcessBatch(), "first batch claims the event")

	var stored models.GowaWebhookEvent
	require.NoError(t, app.DB.First(&stored, "id = ?", evt.ID).Error)
	assert.Equal(t, models.GowaWebhookEventPending, stored.Status, "failure under the attempt ceiling requeues as pending")
	assert.Equal(t, 1, stored.Attempts)
	require.NotNil(t, stored.NextAttemptAt, "requeued event must carry a next_attempt_at backoff")
	assert.True(t, stored.NextAttemptAt.After(time.Now()), "backoff must be in the future")

	// The backoff gate must stop an immediate re-claim.
	require.Equal(t, 0, proc.ProcessBatch(), "event still in backoff must not be re-claimed")
	assert.Equal(t, 1, stored.Attempts)
}
