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
