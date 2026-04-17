package whatsmeow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestFindOrCreateContact_CreatesSeparateContactsPerInstance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Per Instance Org",
		Slug:      "per-instance-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instanceA := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance A",
		Settings:       models.JSONB{},
	}
	instanceB := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance B",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instanceA).Error)
	require.NoError(t, db.Create(&instanceB).Error)

	existing := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instanceA.ID,
		PhoneNumber:    "15550000001",
		ProfileName:    "Existing A",
		Metadata:       models.JSONB{"source": "a"},
	}
	require.NoError(t, db.Create(&existing).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instanceB.ID,
		"15550000001",
		"Existing B",
		models.JSONB{"source": "b"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	require.NotNil(t, contact.InstanceID)
	assert.Equal(t, instanceB.ID, *contact.InstanceID)
	assert.NotEqual(t, existing.ID, contact.ID)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).
		Where("organization_id = ? AND phone_number = ?", org.ID, "15550000001").
		Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestFindOrCreateContact_AdoptsLegacyContactWithoutInstance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Legacy Adopt Org",
		Slug:      "legacy-adopt-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance One",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	legacy := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "15550000002",
		ProfileName:    "Legacy Contact",
		Metadata:       models.JSONB{"legacy": true},
	}
	require.NoError(t, db.Create(&legacy).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"15550000002",
		"Legacy Contact",
		models.JSONB{"source": "inbound"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, legacy.ID, contact.ID)
	require.NotNil(t, contact.InstanceID)
	assert.Equal(t, instance.ID, *contact.InstanceID)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", legacy.ID).Error)
	require.NotNil(t, updated.InstanceID)
	assert.Equal(t, instance.ID, *updated.InstanceID)
	assert.Equal(t, "Legacy Contact", updated.ProfileName)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).
		Where("organization_id = ? AND phone_number = ?", org.ID, "15550000002").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestFindOrCreateContact_ReusesExistingContactForSameInstance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Reuse Org",
		Slug:      "reuse-instance-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Reuse Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	contactRow := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550000003",
		ProfileName:    "",
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&contactRow).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"15550000003",
		"Updated Name",
		models.JSONB{"source": "inbound"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, contactRow.ID, contact.ID)
	require.NotNil(t, contact.InstanceID)
	assert.Equal(t, instance.ID, *contact.InstanceID)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contactRow.ID).Error)
	assert.Equal(t, "Updated Name", updated.ProfileName)
}

func TestFindOrCreateContact_ReplacesLIDPlaceholderProfileName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "LID Name Replace Org",
		Slug:      "lid-name-replace-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "LID Name Replace Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	contactRow := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "966561853319",
		ProfileName:    "149641526026409@lid",
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&contactRow).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"966561853319",
		"Customer Profile",
		models.JSONB{"source": "inbound"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, contactRow.ID, contact.ID)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contactRow.ID).Error)
	assert.Equal(t, "Customer Profile", updated.ProfileName)
}

func TestFindOrCreateContact_DoesNotFallbackProfileNameToLIDIdentity(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "LID Fallback Org",
		Slug:      "lid-fallback-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "LID Fallback Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"149641526026409@lid",
		"",
		models.JSONB{},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, "", contact.ProfileName)
}

func TestMigrateContactPhoneFromLID_DoesNotCopyLIDProfileName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "LID Migration Org",
		Slug:      "lid-migration-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "LID Migration Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	lidPhone := "149641526026409@lid"
	pnPhone := "966561853319"

	lidContact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    lidPhone,
		ProfileName:    lidPhone,
	}
	require.NoError(t, db.Create(&lidContact).Error)

	pnContact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    pnPhone,
		ProfileName:    "",
	}
	require.NoError(t, db.Create(&pnContact).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	cm.migrateContactPhoneFromLID(context.Background(), org.ID, instance.ID, lidPhone, pnPhone)

	var updatedPN models.Contact
	require.NoError(t, db.First(&updatedPN, "id = ?", pnContact.ID).Error)
	assert.Equal(t, "", updatedPN.ProfileName)

	var lidCount int64
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", lidContact.ID).Count(&lidCount).Error)
	assert.Equal(t, int64(0), lidCount)
}

func TestFindOrCreateContact_RestoresSoftDeletedContactForSameInstance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Restore Org",
		Slug:      "restore-instance-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Restore Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	contactRow := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550000004",
		ProfileName:    "",
		Metadata:       models.JSONB{"old": true},
	}
	require.NoError(t, db.Create(&contactRow).Error)
	require.NoError(t, db.Delete(&contactRow).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"15550000004",
		"Restored Name",
		models.JSONB{"source": "inbound"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, contactRow.ID, contact.ID)
	require.NotNil(t, contact.InstanceID)
	assert.Equal(t, instance.ID, *contact.InstanceID)
	assert.Equal(t, "Restored Name", contact.ProfileName)

	var restored models.Contact
	require.NoError(t, db.Unscoped().First(&restored, "id = ?", contactRow.ID).Error)
	assert.False(t, restored.DeletedAt.Valid)
	assert.Equal(t, "Restored Name", restored.ProfileName)
	assert.Equal(t, "inbound", restored.Metadata["source"])
	assert.Equal(t, true, restored.Metadata["old"])

	var count int64
	require.NoError(t, db.Unscoped().Model(&models.Contact{}).
		Where("organization_id = ? AND phone_number = ? AND instance_id = ?", org.ID, "15550000004", instance.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestFindOrCreateContact_RestoresSoftDeletedLegacyContact(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Restore Legacy Org",
		Slug:      "restore-legacy-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Legacy Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	legacy := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "15550000005",
		ProfileName:    "",
		Metadata:       models.JSONB{"legacy": true},
	}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Delete(&legacy).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	contact, err := cm.findOrCreateContact(
		context.Background(),
		org.ID,
		instance.ID,
		"15550000005",
		"Legacy Restored",
		models.JSONB{"source": "inbound"},
	)
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, legacy.ID, contact.ID)
	require.NotNil(t, contact.InstanceID)
	assert.Equal(t, instance.ID, *contact.InstanceID)
	assert.Equal(t, "Legacy Restored", contact.ProfileName)

	var restored models.Contact
	require.NoError(t, db.Unscoped().First(&restored, "id = ?", legacy.ID).Error)
	assert.False(t, restored.DeletedAt.Valid)
	require.NotNil(t, restored.InstanceID)
	assert.Equal(t, instance.ID, *restored.InstanceID)
	assert.Equal(t, "Legacy Restored", restored.ProfileName)
	assert.Equal(t, true, restored.Metadata["legacy"])
	assert.Equal(t, "inbound", restored.Metadata["source"])

	var count int64
	require.NoError(t, db.Unscoped().Model(&models.Contact{}).
		Where("organization_id = ? AND phone_number = ?", org.ID, "15550000005").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestPersistParsedMessage_FailedInboundMediaEnqueuesRecoveryJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Inbound Queue Org",
		Slug:      "inbound-queue-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Inbound Queue Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{
		InboundMediaRetryCount:      2,
		InboundMediaRetryDelayMs:    0,
		InboundMediaRetryMaxDelayMs: 0,
	}, nil, t.TempDir())
	mockQueue := testutil.NewMockQueue()
	cm.SetInboundMediaQueue(mockQueue)

	evt := makeInboundDocumentEventForPersistTest(t, "wamid.inbound.media.queue.1")
	message, err := cm.persistParsedMessage(context.Background(), nil, evt, instance.ID, org.ID, persistMessageOptions{})
	require.NoError(t, err)
	require.NotNil(t, message)

	require.Len(t, mockQueue.InboundMediaJobs, 1)
	job := mockQueue.InboundMediaJobs[0]
	assert.Equal(t, message.ID, job.MessageID)
	assert.Equal(t, org.ID, job.OrganizationID)
	assert.Equal(t, instance.ID, job.InstanceID)
	assert.Equal(t, models.MessageTypeDocument, job.MessageType)
	assert.Equal(t, "document", job.MediaKind)
	assert.Equal(t, "application/pdf", job.MimeType)
	assert.Equal(t, "report.pdf", job.FallbackFilename)
	assert.NotEmpty(t, job.MediaPayloadBase64)
	assert.Equal(t, "client is nil", job.LastError)

	var saved models.Message
	require.NoError(t, db.First(&saved, "id = ?", message.ID).Error)
	assert.Equal(t, inboundMediaAsyncStatusQueued, saved.Metadata[inboundMediaAsyncStatusKey])
	assert.NotEmpty(t, saved.Metadata[inboundMediaAsyncEnqueuedAtKey])
	assert.Equal(t, "client is nil", saved.Metadata[inboundMediaAsyncLastErrorKey])
	assert.Contains(t, saved.ErrorMessage, "queued for async recovery")
	storedJob, err := decodeInboundMediaAsyncJobMetadata(saved.Metadata[inboundMediaAsyncJobKey])
	require.NoError(t, err)
	assert.Equal(t, job.MessageID, storedJob.MessageID)
	assert.Equal(t, job.OrganizationID, storedJob.OrganizationID)
	assert.Equal(t, job.InstanceID, storedJob.InstanceID)
	assert.Equal(t, job.MediaKind, storedJob.MediaKind)
	assert.Equal(t, job.MediaPayloadBase64, storedJob.MediaPayloadBase64)
}

func TestPersistParsedMessage_EnqueueFailureMarksMessageMetadata(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Inbound Queue Fail Org",
		Slug:      "inbound-queue-fail-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Inbound Queue Fail Instance",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{
		InboundMediaRetryCount:      2,
		InboundMediaRetryDelayMs:    0,
		InboundMediaRetryMaxDelayMs: 0,
	}, nil, t.TempDir())
	mockQueue := testutil.NewMockQueue()
	mockQueue.EnqueueInboundFunc = func(context.Context, *queue.InboundMediaJob) error {
		return errors.New("redis unavailable")
	}
	cm.SetInboundMediaQueue(mockQueue)

	evt := makeInboundDocumentEventForPersistTest(t, "wamid.inbound.media.queue.2")
	message, err := cm.persistParsedMessage(context.Background(), nil, evt, instance.ID, org.ID, persistMessageOptions{})
	require.NoError(t, err)
	require.NotNil(t, message)

	var saved models.Message
	require.NoError(t, db.First(&saved, "id = ?", message.ID).Error)
	assert.Equal(t, inboundMediaAsyncStatusEnqueueFail, saved.Metadata[inboundMediaAsyncStatusKey])
	assert.Contains(t, saved.Metadata[inboundMediaAsyncEnqueueErrorKey], "redis unavailable")
	assert.Equal(t, "client is nil", saved.Metadata[inboundMediaAsyncLastErrorKey])
	assert.Contains(t, saved.ErrorMessage, "async enqueue failed")
	storedJob, err := decodeInboundMediaAsyncJobMetadata(saved.Metadata[inboundMediaAsyncJobKey])
	require.NoError(t, err)
	assert.Equal(t, message.ID, storedJob.MessageID)
	assert.Equal(t, org.ID, storedJob.OrganizationID)
	assert.Equal(t, instance.ID, storedJob.InstanceID)
	assert.Equal(t, "document", storedJob.MediaKind)
}

func makeInboundDocumentEventForPersistTest(t *testing.T, waMessageID string) *events.Message {
	t.Helper()
	chatJID, err := types.ParseJID("15550001234@s.whatsapp.net")
	require.NoError(t, err)

	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: false,
				IsGroup:  false,
			},
			ID:        waMessageID,
			PushName:  "Customer",
			Timestamp: time.Now().UTC(),
		},
		Message: &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				FileName: proto.String("report.pdf"),
				Mimetype: proto.String("application/pdf"),
			},
		},
	}
}
