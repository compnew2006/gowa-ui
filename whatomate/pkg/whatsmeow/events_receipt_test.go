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

func TestHandleReceipt_UpdatesCampaignDeliveredStatsIdempotently(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)
	campaign, recipient := seedReceiptCampaignData(t, db, org.ID, message.ID, message.WhatsAppMessageID, models.MessageStatusSent)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000000@s.whatsapp.net")
	require.NoError(t, err)
	receipt := &events.Receipt{
		MessageSource: types.MessageSource{Chat: chatJID},
		Type:          types.ReceiptTypeDelivered,
		MessageIDs:    []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}

	cm.handleReceipt(context.Background(), receipt, instance.ID, org.ID)
	cm.handleReceipt(context.Background(), receipt, instance.ID, org.ID)

	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, db.First(&updatedRecipient, "id = ?", recipient.ID).Error)
	assert.Equal(t, models.MessageStatusDelivered, updatedRecipient.Status)
	assert.NotNil(t, updatedRecipient.DeliveredAt)
	assert.Nil(t, updatedRecipient.ReadAt)

	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, db.First(&updatedCampaign, "id = ?", campaign.ID).Error)
	assert.Equal(t, 1, updatedCampaign.DeliveredCount)
	assert.Equal(t, 0, updatedCampaign.ReadCount)
}

func TestHandleReceipt_UpdatesCampaignReadStatsOnceWithoutDeliveredIncrement(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)
	campaign, recipient := seedReceiptCampaignData(t, db, org.ID, message.ID, message.WhatsAppMessageID, models.MessageStatusDelivered)
	require.NoError(t, db.Model(&message).Updates(map[string]any{
		"status": models.MessageStatusDelivered,
		"metadata": models.JSONB{
			"campaign_id": campaign.ID.String(),
		},
	}).Error)
	require.NoError(t, db.Model(campaign).Update("delivered_count", 1).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000000@s.whatsapp.net")
	require.NoError(t, err)
	receipt := &events.Receipt{
		MessageSource: types.MessageSource{Chat: chatJID},
		Type:          types.ReceiptTypeRead,
		MessageIDs:    []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}

	cm.handleReceipt(context.Background(), receipt, instance.ID, org.ID)
	cm.handleReceipt(context.Background(), receipt, instance.ID, org.ID)

	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, db.First(&updatedRecipient, "id = ?", recipient.ID).Error)
	assert.Equal(t, models.MessageStatusRead, updatedRecipient.Status)
	assert.NotNil(t, updatedRecipient.ReadAt)

	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, db.First(&updatedCampaign, "id = ?", campaign.ID).Error)
	assert.Equal(t, 1, updatedCampaign.DeliveredCount)
	assert.Equal(t, 1, updatedCampaign.ReadCount)
}

func TestHandleReceipt_NonCampaignMessageDoesNotCreateCampaignStats(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000000@s.whatsapp.net")
	require.NoError(t, err)

	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{Chat: chatJID},
		Type:          types.ReceiptTypeDelivered,
		MessageIDs:    []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}, instance.ID, org.ID)

	var campaigns int64
	require.NoError(t, db.Model(&models.BulkMessageCampaign{}).Count(&campaigns).Error)
	assert.Zero(t, campaigns)
}

func TestHandleReceipt_CampaignStatsRespectMessageOrganization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org, instance, _, message := seedReceiptTestData(t, db)
	otherOrg := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Other Receipt Org",
		Slug:      "other-receipt-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&otherOrg).Error)
	campaign, recipient := seedReceiptCampaignData(t, db, otherOrg.ID, message.ID, message.WhatsAppMessageID, models.MessageStatusSent)
	require.NoError(t, db.Model(&message).Update("metadata", models.JSONB{
		"campaign_id": campaign.ID.String(),
	}).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	chatJID, err := types.ParseJID("201000000000@s.whatsapp.net")
	require.NoError(t, err)

	cm.handleReceipt(context.Background(), &events.Receipt{
		MessageSource: types.MessageSource{Chat: chatJID},
		Type:          types.ReceiptTypeDelivered,
		MessageIDs:    []types.MessageID{types.MessageID(message.WhatsAppMessageID)},
	}, instance.ID, org.ID)

	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, db.First(&updatedRecipient, "id = ?", recipient.ID).Error)
	assert.Equal(t, models.MessageStatusSent, updatedRecipient.Status)

	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, db.First(&updatedCampaign, "id = ?", campaign.ID).Error)
	assert.Zero(t, updatedCampaign.DeliveredCount)
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

func seedReceiptCampaignData(t *testing.T, db *gorm.DB, orgID, messageID uuid.UUID, waMessageID string, status models.MessageStatus) (*models.BulkMessageCampaign, *models.BulkMessageRecipient) {
	t.Helper()

	template := models.Template{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "receipt-template-" + uuid.NewString(),
		Language:       "en",
		Category:       string(models.TemplateCategoryUtility),
		Status:         string(models.TemplateStatusApproved),
		BodyContent:    "hello",
	}
	require.NoError(t, db.Create(&template).Error)

	user := models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Email:          "receipt-user-" + uuid.NewString() + "@example.com",
		FullName:       "Receipt User",
		PasswordHash:   "hash",
		IsActive:       true,
	}
	require.NoError(t, db.Create(&user).Error)

	campaign := &models.BulkMessageCampaign{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: "15550001111",
		Name:            "Receipt Campaign",
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		TotalRecipients: 1,
		SentCount:       1,
		DeliveredCount:  0,
		ReadCount:       0,
		FailedCount:     0,
		CreatedBy:       user.ID,
		MinDelaySeconds: 10,
		MaxDelaySeconds: 10,
	}
	require.NoError(t, db.Create(campaign).Error)

	recipient := &models.BulkMessageRecipient{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		CampaignID:        campaign.ID,
		PhoneNumber:       "201000000000",
		PhoneNormalized:   "201000000000",
		RecipientName:     "Receipt Contact",
		Status:            status,
		WhatsAppMessageID: waMessageID,
		MessageID:         &messageID,
		TemplateParams:    models.JSONB{},
	}
	require.NoError(t, db.Create(recipient).Error)

	require.NoError(t, db.Model(&models.Message{}).
		Where("id = ?", messageID).
		Update("metadata", models.JSONB{"campaign_id": campaign.ID.String()}).Error)

	return campaign, recipient
}
