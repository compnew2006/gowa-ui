package whatsmeow

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
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
