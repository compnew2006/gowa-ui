package whatsmeow

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func setupManager(t *testing.T) (*ConnectionManager, *models.Organization, *models.WhatsAppInstance) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Event Test Org " + uuid.NewString()[:8],
		Slug:      "event-org-" + uuid.NewString()[:8],
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(org).Error)

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Event Test Instance",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	cm.disableAvatarSync = true

	return cm, org, instance
}

func callMeta(callID string) types.BasicCallMeta {
	return types.BasicCallMeta{CallID: callID}
}

func TestHandleEvent_UnknownEvent_NoPanic(t *testing.T) {
	cm, org, instance := setupManager(t)

	assert.NotPanics(t, func() {
		cm.handleEvent("unknown_event_type", instance.ID, org.ID)
	})
}

func TestHandleEvent_NilEvent_NoPanic(t *testing.T) {
	cm, org, instance := setupManager(t)

	assert.NotPanics(t, func() {
		cm.handleEvent(nil, instance.ID, org.ID)
	})
}

func TestHandleEvent_QRCode_CachesCode(t *testing.T) {
	cm, org, instance := setupManager(t)

	cm.handleEvent(&events.QR{Codes: []string{"QR-TEST-123"}}, instance.ID, org.ID)

	snap, ok := cm.GetCachedQRCode(instance.ID)
	require.True(t, ok)
	assert.Equal(t, "QR-TEST-123", snap.Code)
}

func TestHandleEvent_QRCode_EmptyCodes_NoPanic(t *testing.T) {
	cm, org, instance := setupManager(t)

	assert.NotPanics(t, func() {
		cm.handleEvent(&events.QR{Codes: []string{}}, instance.ID, org.ID)
	})
}

func TestHandleEvent_Disconnected_UpdatesStatus(t *testing.T) {
	cm, org, instance := setupManager(t)

	require.NoError(t, cm.db.Model(instance).Update("status", models.InstanceStatusConnected).Error)

	cm.handleEvent(&events.Disconnected{}, instance.ID, org.ID)

	var updated models.WhatsAppInstance
	require.NoError(t, cm.db.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, models.InstanceStatusDisconnected, updated.Status)
}

func TestHandleEvent_CallOfferThenTerminate_CountsCorrectly(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))

	cm.handleEvent(&events.CallTerminate{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 0, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_CallReject_ClearsCallState(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))

	cm.handleEvent(&events.CallReject{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 0, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_CallPreAccept_MarksActive(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_CallAccept_MarksActive(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallAccept{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_CallTransport_MarksActive(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallTransport{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_Connected_UpdatesStatusAndClearsQR(t *testing.T) {
	cm, org, instance := setupManager(t)

	cm.CacheQRCode(instance.ID, "pre-connect-qr", 20)
	_, ok := cm.GetCachedQRCode(instance.ID)
	require.True(t, ok)

	cm.handleEvent(&events.Connected{}, instance.ID, org.ID)

	var updated models.WhatsAppInstance
	require.NoError(t, cm.db.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, models.InstanceStatusConnected, updated.Status)

	_, ok = cm.GetCachedQRCode(instance.ID)
	assert.False(t, ok)
}

func TestHandleEvent_Disconnected_ClearsActiveCalls(t *testing.T) {
	cm, org, instance := setupManager(t)
	callID := "call-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callID)}, instance.ID, org.ID)
	require.Equal(t, 1, cm.activeCallCount(instance.ID))

	cm.handleEvent(&events.Disconnected{}, instance.ID, org.ID)
	assert.Equal(t, 0, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_MultipleCallsPerInstance(t *testing.T) {
	cm, org, instance := setupManager(t)
	callA := "call-A-" + uuid.NewString()[:8]
	callB := "call-B-" + uuid.NewString()[:8]

	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callA)}, instance.ID, org.ID)
	cm.handleEvent(&events.CallPreAccept{BasicCallMeta: callMeta(callB)}, instance.ID, org.ID)
	assert.Equal(t, 2, cm.activeCallCount(instance.ID))

	cm.handleEvent(&events.CallTerminate{BasicCallMeta: callMeta(callA)}, instance.ID, org.ID)
	assert.Equal(t, 1, cm.activeCallCount(instance.ID))
}

func TestHandleEvent_CompletesWithinTimeout(t *testing.T) {
	cm, org, instance := setupManager(t)

	start := time.Now()
	cm.handleEvent(&events.QR{Codes: []string{"timeout-test"}}, instance.ID, org.ID)
	assert.Less(t, time.Since(start), 5*time.Second)
}
