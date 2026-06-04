package whatsmeow

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	waClient "go.mau.fi/whatsmeow"
	"gorm.io/gorm"
)

func TestConnectionManager_HealthMonitorReconnectsDroppedClientExactlyOnce(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	cm := newHealthTestManager(db)
	instance := createHealthTestInstance(t, db, models.InstanceStatusConnected, "Support", "15550001111@s.whatsapp.net")
	require.NoError(t, cm.RegisterInstanceClient(instance, &waClient.Client{}))

	var reconnects int32
	reconnected := make(chan struct{}, 1)
	cm.connectFn = func(ctx context.Context, instanceID uuid.UUID) error {
		atomic.AddInt32(&reconnects, 1)
		select {
		case reconnected <- struct{}{}:
		default:
		}
		return nil
	}

	cm.StartHealthMonitor(context.Background())

	select {
	case <-reconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for health monitor reconnect attempt")
	}

	cm.StopHealthMonitor()
	assert.Equal(t, int32(1), atomic.LoadInt32(&reconnects))
}

func TestConnectionManager_HealthMonitorSkipsManuallyRemovedInstances(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	cm := newHealthTestManager(db)
	instance := createHealthTestInstance(t, db, models.InstanceStatusConnected, "Ops", "15550002222@s.whatsapp.net")
	require.NoError(t, cm.RegisterInstanceClient(instance, nil))
	require.NoError(t, cm.Disconnect(context.Background(), instance.ID))

	var reconnects int32
	cm.connectFn = func(ctx context.Context, instanceID uuid.UUID) error {
		atomic.AddInt32(&reconnects, 1)
		return nil
	}

	cm.StartHealthMonitor(context.Background())
	time.Sleep(200 * time.Millisecond)
	cm.StopHealthMonitor()

	assert.Equal(t, int32(0), atomic.LoadInt32(&reconnects))
	assert.Nil(t, cm.GetClient(instance.ID))
}

func TestConnectionManager_HealthMonitorSkipsPermanentStates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	cm := newHealthTestManager(db)
	loggedOut := createHealthTestInstance(t, db, models.InstanceStatusLoggedOut, "Logged Out", "15550003333@s.whatsapp.net")
	banned := createHealthTestInstance(t, db, models.InstanceStatusBanned, "Banned", "15550004444@s.whatsapp.net")

	require.NoError(t, cm.RegisterInstanceClient(loggedOut, &waClient.Client{}))
	require.NoError(t, cm.RegisterInstanceClient(banned, &waClient.Client{}))

	var reconnects int32
	cm.connectFn = func(ctx context.Context, instanceID uuid.UUID) error {
		atomic.AddInt32(&reconnects, 1)
		return nil
	}

	cm.StartHealthMonitor(context.Background())
	time.Sleep(200 * time.Millisecond)
	cm.StopHealthMonitor()

	assert.Equal(t, int32(0), atomic.LoadInt32(&reconnects))
	assert.Nil(t, cm.GetClient(loggedOut.ID))
	assert.Nil(t, cm.GetClient(banned.ID))
}

func newHealthTestManager(db *gorm.DB) *ConnectionManager {
	return NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{
		HealthMonitorIntervalSeconds: 1,
		ReconnectTimeoutSeconds:      1,
	}, nil, "./uploads")
}

// TestConnectionManager_ReconnectExistingClient_AllowsDispatcherAfterStop
// proves the contract that connectExistingClient must call AllowInstance on the
// event dispatcher so events are not silently dropped after a
// Disconnected → reconnect-existing sequence.
//
// Reproduces the bug: previously connectExistingClient did not call
// AllowInstance, so the dispatcher kept the instance ID in its `stopped` map
// and Dispatch returned false for all subsequent events.
func TestConnectionManager_ReconnectExistingClient_AllowsDispatcherAfterStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	instance := createHealthTestInstance(t, db, models.InstanceStatusConnected, "Reconnect", "15550005555@s.whatsapp.net")
	cm := newHealthTestManager(db)
	require.NotNil(t, cm.eventDispatcher)

	// Step 1: simulate the dispatcher state right after a Disconnected event.
	// stopEventDispatcherInstance -> StopInstance deletes the queue and adds
	// the instance to the dispatcher's `stopped` map.
	cm.eventDispatcher.StopInstance(instance.ID)

	// Sanity: while stopped, Dispatch must return false.
	require.False(t,
		cm.eventDispatcher.Dispatch("while-stopped", instance.ID, instance.OrganizationID),
		"sanity: Dispatch must return false while the instance is stopped",
	)

	// Step 2: short-circuit the real websocket dial so we can exercise
	// connectExistingClient without a real network or device store.
	cm.existingClientConnectFn = func(_ context.Context, _ *waClient.Client) error { return nil }

	// Step 3: invoke the reconnect-existing path directly (same package).
	// With the fix, this should call AllowInstance BEFORE the dial stub runs.
	require.NoError(t,
		cm.connectExistingClient(context.Background(), instance, &waClient.Client{}),
		"connectExistingClient must succeed when the dial stub returns nil",
	)

	// Step 4: the dispatcher must now accept events for this instance.
	require.True(t,
		cm.eventDispatcher.Dispatch("after-reconnect", instance.ID, instance.OrganizationID),
		"after connectExistingClient, the dispatcher must accept events for the reconnected instance",
	)
}

// TestConnectionManager_ReconnectExistingClientFailure_RestoresDispatcherStop
// proves the failure-side contract: when dialExistingClient returns an error,
// connectExistingClient must roll the dispatcher back to the stopped state so
// late events do not silently recreate the queue. Mirrors the rollback in
// newClient (client.Connect failure → stopEventDispatcherInstance).
func TestConnectionManager_ReconnectExistingClientFailure_RestoresDispatcherStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	instance := createHealthTestInstance(t, db, models.InstanceStatusConnected, "ReconnectFail", "15550006666@s.whatsapp.net")
	cm := newHealthTestManager(db)
	require.NotNil(t, cm.eventDispatcher)

	// Step 1: dispatcher enters the stopped state for this instance.
	cm.eventDispatcher.StopInstance(instance.ID)
	require.False(t,
		cm.eventDispatcher.Dispatch("while-stopped", instance.ID, instance.OrganizationID),
		"sanity: Dispatch must return false while the instance is stopped",
	)

	// Step 2: dial stub returns an error.
	dialErr := assert.AnError
	cm.existingClientConnectFn = func(_ context.Context, _ *waClient.Client) error { return dialErr }

	// Step 3: connectExistingClient must propagate the dial error.
	err := cm.connectExistingClient(context.Background(), instance, &waClient.Client{})
	require.ErrorIs(t, err, dialErr, "connectExistingClient must surface the dial error")

	// Step 4: dispatcher must be back in the stopped state. Without the
	// rollback, AllowInstance would have cleared the `stopped` entry and
	// late events would silently recreate the queue and be processed,
	// defeating the purpose of the original StopInstance.
	require.False(t,
		cm.eventDispatcher.Dispatch("after-failed-reconnect", instance.ID, instance.OrganizationID),
		"after a failed reconnect, the dispatcher must remain in the stopped state for the instance",
	)
}

func createHealthTestInstance(t *testing.T, db *gorm.DB, status models.InstanceStatus, name, jid string) models.WhatsAppInstance {
	t.Helper()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      name + " Org",
		Slug:      strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           name,
		Status:         status,
		JID:            jid,
		PhoneNumber:    strings.TrimSuffix(jid, "@s.whatsapp.net"),
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)
	return instance
}
