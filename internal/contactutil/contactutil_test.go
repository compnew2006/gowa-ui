package contactutil

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateContact_CreatesNew(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	contact, isNew, err := GetOrCreateContact(db, org.ID, "1234567890", "Alice")
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.Equal(t, "1234567890", contact.PhoneNumber)
	assert.Equal(t, "Alice", contact.ProfileName)
}

func TestGetOrCreateContact_FindsExisting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	existing := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "1234567890",
		ProfileName:    "Alice",
	}
	require.NoError(t, db.Create(&existing).Error)

	contact, isNew, err := GetOrCreateContact(db, org.ID, "1234567890", "Alice")
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, existing.ID, contact.ID)
}

func TestGetOrCreateContact_NormalizesPlus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	existing := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "1234567890",
		ProfileName:    "Bob",
	}
	require.NoError(t, db.Create(&existing).Error)

	contact, isNew, err := GetOrCreateContact(db, org.ID, "+1234567890", "Bob")
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, existing.ID, contact.ID)
}

func TestGetOrCreateContact_FindsPlusPrefix(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	existing := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "+1234567890",
		ProfileName:    "Charlie",
	}
	require.NoError(t, db.Create(&existing).Error)

	contact, isNew, err := GetOrCreateContact(db, org.ID, "1234567890", "Charlie")
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, existing.ID, contact.ID)
}

func TestGetOrCreateContact_UpdatesProfileName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	existing := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "1234567890",
		ProfileName:    "Old Name",
	}
	require.NoError(t, db.Create(&existing).Error)

	contact, isNew, err := GetOrCreateContact(db, org.ID, "1234567890", "New Name")
	require.NoError(t, err)
	assert.False(t, isNew)

	var reloaded models.Contact
	require.NoError(t, db.First(&reloaded, contact.ID).Error)
	assert.Equal(t, "New Name", reloaded.ProfileName)
}

func TestStampChatCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	newContact := func(phone string, meta models.JSONB) *models.Contact {
		c := models.Contact{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			PhoneNumber:    phone,
			Metadata:       meta,
		}
		require.NoError(t, db.Create(&c).Error)
		return &c
	}

	t.Run("marks group chat", func(t *testing.T) {
		c := newContact("g-"+uid+"@g.us", nil)
		require.NoError(t, StampChatCategory(db, c, true, false))

		var reloaded models.Contact
		require.NoError(t, db.First(&reloaded, c.ID).Error)
		assert.Equal(t, true, reloaded.Metadata["is_group_chat"])
		_, hasNewsletter := reloaded.Metadata["is_newsletter"]
		assert.False(t, hasNewsletter)
	})

	t.Run("marks newsletter and clears stale group flag", func(t *testing.T) {
		c := newContact("n-"+uid+"@newsletter", models.JSONB{"is_group_chat": true})
		require.NoError(t, StampChatCategory(db, c, false, true))

		var reloaded models.Contact
		require.NoError(t, db.First(&reloaded, c.ID).Error)
		assert.Equal(t, true, reloaded.Metadata["is_newsletter"])
		_, hasGroup := reloaded.Metadata["is_group_chat"]
		assert.False(t, hasGroup)
	})

	t.Run("no-op for 1:1 chats", func(t *testing.T) {
		c := newContact("111"+uid, nil)
		require.NoError(t, StampChatCategory(db, c, false, false))

		var reloaded models.Contact
		require.NoError(t, db.First(&reloaded, c.ID).Error)
		_, hasGroup := reloaded.Metadata["is_group_chat"]
		_, hasNewsletter := reloaded.Metadata["is_newsletter"]
		assert.False(t, hasGroup)
		assert.False(t, hasNewsletter)
	})

	t.Run("no-op when flag already correct", func(t *testing.T) {
		c := newContact("g2-"+uid+"@g.us", models.JSONB{"is_group_chat": true, "other": "kept"})
		require.NoError(t, StampChatCategory(db, c, true, false))

		var reloaded models.Contact
		require.NoError(t, db.First(&reloaded, c.ID).Error)
		assert.Equal(t, true, reloaded.Metadata["is_group_chat"])
		assert.Equal(t, "kept", reloaded.Metadata["other"])
	})
}

func TestStampAccountName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uid := uuid.New().String()[:8]
	org := models.Organization{BaseModel: models.BaseModel{ID: uuid.New()}, Name: "test-" + uid, Slug: "test-" + uid}
	require.NoError(t, db.Create(&org).Error)

	c := models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		PhoneNumber:     "555" + uid,
		WhatsAppAccount: "stale-account",
	}
	require.NoError(t, db.Create(&c).Error)

	require.NoError(t, StampAccountName(db, &c, "fresh-account"))
	assert.Equal(t, "fresh-account", c.WhatsAppAccount)

	var reloaded models.Contact
	require.NoError(t, db.First(&reloaded, c.ID).Error)
	assert.Equal(t, "fresh-account", reloaded.WhatsAppAccount)

	// Stamping the same name again is a no-op and must not error.
	require.NoError(t, StampAccountName(db, &c, "fresh-account"))
}
