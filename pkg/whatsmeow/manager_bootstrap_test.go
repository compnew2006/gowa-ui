package whatsmeow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestAutoConnectLinkedInstancesOnFirstRun_ConnectsAndMarksDone(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Bootstrap Org",
		Slug:      "bootstrap-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Linked Instance",
		JID:            "15551234567@s.whatsapp.net",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	logger := logf.New(logf.Opts{})
	cm := NewConnectionManager(db, nil, logger, &config.WhatsmeowConfig{}, nil, "./uploads")

	connectCalls := 0
	cm.connectFn = func(_ context.Context, instanceID uuid.UUID) error {
		connectCalls++
		assert.Equal(t, instance.ID, instanceID)
		return nil
	}

	require.NoError(t, cm.AutoConnectLinkedInstancesOnFirstRun(context.Background()))
	assert.Equal(t, 1, connectCalls)

	var updatedOrg models.Organization
	require.NoError(t, db.First(&updatedOrg, "id = ?", org.ID).Error)

	done, ok := updatedOrg.Settings[orgAutoConnectBootstrapSettingsKey].(bool)
	require.True(t, ok)
	assert.True(t, done)

	// Should not run again after marker is set.
	connectCalls = 0
	require.NoError(t, cm.AutoConnectLinkedInstancesOnFirstRun(context.Background()))
	assert.Equal(t, 0, connectCalls)
}

func TestAutoConnectLinkedInstancesOnFirstRun_RetriesWhenConnectFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Bootstrap Retry Org",
		Slug:      "bootstrap-retry-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Retry Instance",
		JID:            "15557654321@s.whatsapp.net",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	logger := logf.New(logf.Opts{})
	cm := NewConnectionManager(db, nil, logger, &config.WhatsmeowConfig{}, nil, "./uploads")

	cm.connectFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("connect failed")
	}

	require.Error(t, cm.AutoConnectLinkedInstancesOnFirstRun(context.Background()))

	var updatedOrg models.Organization
	require.NoError(t, db.First(&updatedOrg, "id = ?", org.ID).Error)
	done, ok := updatedOrg.Settings[orgAutoConnectBootstrapSettingsKey].(bool)
	assert.False(t, ok && done)

	// Since marker was not set on failure, next run should retry.
	connectCalls := 0
	cm.connectFn = func(_ context.Context, instanceID uuid.UUID) error {
		connectCalls++
		assert.Equal(t, instance.ID, instanceID)
		return nil
	}

	require.NoError(t, cm.AutoConnectLinkedInstancesOnFirstRun(context.Background()))
	assert.Equal(t, 1, connectCalls)
}

func TestReconnectAll_ReconcilesStaleConnectingWithoutJID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Reconnect Reconcile Org",
		Slug:      "reconnect-reconcile-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Unpaired Connecting",
		Status:         models.InstanceStatusConnecting,
		JID:            "",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	logger := logf.New(logf.Opts{})
	cm := NewConnectionManager(db, nil, logger, &config.WhatsmeowConfig{}, nil, "./uploads")

	require.NoError(t, cm.ReconnectAll(context.Background()))

	var updated models.WhatsAppInstance
	require.NoError(t, db.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, models.InstanceStatusDisconnected, updated.Status)
}

func TestReconnectAll_SetsDisconnectedOnReconnectFailure(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Reconnect Failure Org",
		Slug:      "reconnect-failure-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Linked Connecting",
		Status:         models.InstanceStatusConnecting,
		JID:            "15559876543@s.whatsapp.net",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	logger := logf.New(logf.Opts{})
	cm := NewConnectionManager(db, nil, logger, &config.WhatsmeowConfig{}, nil, "./uploads")
	cm.connectFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("reconnect failed")
	}

	require.NoError(t, cm.ReconnectAll(context.Background()))

	var updated models.WhatsAppInstance
	require.NoError(t, db.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, models.InstanceStatusDisconnected, updated.Status)
}
