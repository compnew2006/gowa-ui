package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookRequest(t *testing.T) {
	t.Parallel()

	app := webhookTestApp(t)
	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "wh-sec-org",
		Slug:      "wh-sec-org-" + uuid.New().String()[:8],
	}
	require.NoError(t, app.DB.Create(&org).Error)

	account := models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "wh-sec-account",
		PhoneID:        "phone-sec-1",
		BusinessID:     "business-sec-1",
		AccessToken:    "token",
		AppSecret:      "app-secret-123",
	}
	require.NoError(t, app.DB.Create(&account).Error)

	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"business-sec-1","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"phone-sec-1"},"messages":[{"id":"wamid.1","from":"123","type":"text"}]}}]}]}`)
	var payload WebhookPayload
	require.NoError(t, json.Unmarshal(body, &payload))

	t.Run("missing signature", func(t *testing.T) {
		err := app.validateWebhookRequest(body, nil, &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("invalid signature", func(t *testing.T) {
		err := app.validateWebhookRequest(body, []byte("sha256=deadbeef"), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook signature")
	})

	t.Run("valid signature", func(t *testing.T) {
		mac := hmac.New(sha256.New, []byte(account.AppSecret))
		mac.Write(body)
		signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		err := app.validateWebhookRequest(body, []byte(signature), &payload)
		require.NoError(t, err)
	})
}

func TestValidateWebhookSignaturePayload(t *testing.T) {
	t.Parallel()

	app := webhookTestApp(t)
	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "wh-sec-org-signature",
		Slug:      "wh-sec-org-signature-" + uuid.New().String()[:8],
	}
	require.NoError(t, app.DB.Create(&org).Error)

	account := models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "wh-sec-signature-account",
		PhoneID:        "phone-signature-1",
		BusinessID:     "business-signature-1",
		AccessToken:    "token",
		AppSecret:      "signature-secret-123",
	}
	require.NoError(t, app.DB.Create(&account).Error)

	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"business-signature-1","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"phone-signature-1"},"messages":[{"id":"wamid.1"}]}}]}]}`)
	var payload webhookSignaturePayload
	require.NoError(t, json.Unmarshal(body, &payload))

	mac := hmac.New(sha256.New, []byte(account.AppSecret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	require.NoError(t, app.validateWebhookSignaturePayload(body, []byte(signature), &payload))
}

func TestCollectWebhookAppSecrets_BusinessIDFallback(t *testing.T) {
	t.Parallel()

	app := webhookTestApp(t)
	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "wh-sec-org-2",
		Slug:      "wh-sec-org-2-" + uuid.New().String()[:8],
	}
	require.NoError(t, app.DB.Create(&org).Error)

	account := models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "wh-sec-account-2",
		PhoneID:        "phone-sec-2",
		BusinessID:     "business-sec-2",
		AccessToken:    "token",
		AppSecret:      "app-secret-456",
	}
	require.NoError(t, app.DB.Create(&account).Error)

	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"business-sec-2","changes":[{"field":"message_template_status_update","value":{"event":"APPROVED","message_template_name":"hello","message_template_language":"en"}}]}]}`)
	var payload WebhookPayload
	require.NoError(t, json.Unmarshal(body, &payload))

	secrets, err := app.collectWebhookAppSecrets(&payload)
	require.NoError(t, err)
	assert.Contains(t, secrets, "app-secret-456")
}

func TestCountWebhookEvents(t *testing.T) {
	t.Parallel()

	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"biz-1","changes":[{"field":"messages","value":{"messages":[{"id":"1"},{"id":"2"}],"statuses":[{"id":"s1"}]}},{"field":"message_template_status_update","value":{"event":"APPROVED"}}]}]}`)
	var payload WebhookPayload
	require.NoError(t, json.Unmarshal(body, &payload))

	assert.Equal(t, 4, countWebhookEvents(&payload))
}
