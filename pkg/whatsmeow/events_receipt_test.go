package whatsmeow

import (
	"context"
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
	"gorm.io/gorm"
)

func TestHandleReceipt_UpdatesMessageStatusForNonStatusReceipts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000000@s.whatsapp.net")
	require.NoError(t, err)

	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{
			Chat: chatJID,
		},
		Type:       types.ReceiptTypeDelivered,
		MessageIDs: []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}, instance.ID, org.ID)

	var refreshed models.Message
	require.NoError(t, db.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, models.MessageStatusDelivered, refreshed.Status)
}

func TestHandleReceipt_SkipsEmptyMessageIDs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000001@s.whatsapp.net")
	require.NoError(t, err)

	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{
			Chat: chatJID,
		},
		Type:       types.ReceiptTypeDelivered,
		MessageIDs: []types.MessageID{""},
	}, instance.ID, org.ID)

	var refreshed models.Message
	require.NoError(t, db.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, models.MessageStatusSent, refreshed.Status)
}

func TestHandleReceipt_SkipsStatusBroadcastReceipts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{
			Chat: types.StatusBroadcastJID,
		},
		Type:       types.ReceiptTypeDelivered,
		MessageIDs: []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}, instance.ID, org.ID)

	var refreshed models.Message
	require.NoError(t, db.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, models.MessageStatusSent, refreshed.Status)
}

func TestHandleReceipt_SkipsMessageIDsTrackedAsStatuses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)

	status := models.WhatsAppStatus{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		OrganizationID:    org.ID,
		InstanceID:        instance.ID,
		WhatsAppAccount:   "15550001111",
		SenderJID:         "15550001111@s.whatsapp.net",
		WhatsAppMessageID: message.WhatsAppMessageID,
		StatusType:        models.WhatsAppStatusTypeText,
		Content:           "status body",
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
		Metadata:          models.JSONB{},
	}
	require.NoError(t, db.Create(&status).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000001@s.whatsapp.net")
	require.NoError(t, err)

	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{
			Chat: chatJID,
		},
		Type:       types.ReceiptTypeDelivered,
		MessageIDs: []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}, instance.ID, org.ID)

	var refreshed models.Message
	require.NoError(t, db.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, models.MessageStatusSent, refreshed.Status)
}

func seedReceiptTestData(t *testing.T, db *gorm.DB) (models.Organization, models.WhatsAppInstance, models.Contact, models.Message) {
	t.Helper()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Receipt Org",
		Slug:      "receipt-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Receipt Instance",
		PhoneNumber:    "15550001111",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550002222",
		ProfileName:    "Receipt Contact",
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&contact).Error)

	message := models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    org.ID,
		InstanceID:        &instance.ID,
		WhatsAppAccount:   "15550001111",
		ContactID:         contact.ID,
		WhatsAppMessageID: "wamid-receipt-1",
		ConversationID:    "15550002222@s.whatsapp.net",
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeText,
		Content:           "hello",
		Status:            models.MessageStatusSent,
		Metadata:          models.JSONB{},
	}
	require.NoError(t, db.Create(&message).Error)

	return org, instance, contact, message
}
