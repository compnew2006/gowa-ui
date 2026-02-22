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
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestHandleMessage_PersistsDeviceSentFromMeAsOutgoing(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Device Sent Org",
		Slug:      "device-sent-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	myJID, err := types.ParseJID("15550009999@s.whatsapp.net")
	require.NoError(t, err)
	cm.clients[instance.ID] = &waClient.Client{
		Store: &store.Device{ID: &myJID},
	}

	chatJID, err := types.ParseJID("15550001234@s.whatsapp.net")
	require.NoError(t, err)

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   myJID,
				IsFromMe: true,
			},
			ID:        "wamid-mobile-sync-1",
			Timestamp: time.Now().UTC(),
			DeviceSentMeta: &types.DeviceSentMeta{
				DestinationJID: chatJID.String(),
			},
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Sent from mobile device"),
		},
	}

	cm.handleMessage(context.Background(), evt, instance.ID, org.ID)

	var saved models.Message
	require.NoError(t, db.Where("organization_id = ? AND whats_app_message_id = ?", org.ID, "wamid-mobile-sync-1").First(&saved).Error)

	require.NotNil(t, saved.InstanceID)
	assert.Equal(t, instance.ID, *saved.InstanceID)
	assert.Equal(t, models.DirectionOutgoing, saved.Direction)
	assert.Equal(t, models.MessageStatusSent, saved.Status)
	assert.Equal(t, "Sent from mobile device", saved.Content)
	assert.Equal(t, chatJID.String(), saved.ConversationID)

	var contact models.Contact
	require.NoError(t, db.First(&contact, "id = ?", saved.ContactID).Error)
	assert.Equal(t, org.ID, contact.OrganizationID)
	assert.Equal(t, instance.ID, *contact.InstanceID)
	assert.Equal(t, "15550001234", contact.PhoneNumber)
}

func TestHandleMessage_SkipsFromMeWithoutDeviceSentMetadata(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Self Sent Org",
		Slug:      "self-sent-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	myJID, err := types.ParseJID("15550009999@s.whatsapp.net")
	require.NoError(t, err)
	cm.clients[instance.ID] = &waClient.Client{
		Store: &store.Device{ID: &myJID},
	}

	chatJID, err := types.ParseJID("15550008888@s.whatsapp.net")
	require.NoError(t, err)

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   myJID,
				IsFromMe: true,
			},
			ID:        "wamid-self-runtime-1",
			Timestamp: time.Now().UTC(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Sent by current runtime"),
		},
	}

	cm.handleMessage(context.Background(), evt, instance.ID, org.ID)

	var count int64
	require.NoError(t, db.Model(&models.Message{}).Where("organization_id = ?", org.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
