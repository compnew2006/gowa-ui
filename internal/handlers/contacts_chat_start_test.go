package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type stubWhatsmeowContactResolver struct {
	result         *ResolvedWhatsmeowDirectContact
	err            error
	receivedPhone  string
	receivedOrgID  uuid.UUID
	receivedUserID uuid.UUID
	receivedInstID uuid.UUID
}

func (s *stubWhatsmeowContactResolver) ResolveDirectContact(
	_ context.Context,
	instance *models.WhatsAppInstance,
	phone string,
) (*ResolvedWhatsmeowDirectContact, error) {
	s.receivedPhone = phone
	if instance != nil {
		s.receivedInstID = instance.ID
	}
	return s.result, s.err
}

func newContactsChatStartTestApp(t *testing.T) *App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set, skipping test")
	}

	return &App{
		Config: &config.Config{
			App: config.AppConfig{
				EncryptionKey: testutil.TestEncryptionKey,
			},
			JWT: config.JWTConfig{
				Secret:            testutil.TestJWTSecret,
				AccessExpiryMins:  15,
				RefreshExpiryDays: 7,
			},
			WhatsApp: config.WhatsAppConfig{
				Provider: "whatsmeow",
			},
		},
		DB:    db,
		Log:   log,
		Redis: redisClient,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func createTestWhatsmeowInstance(t *testing.T, app *App, orgID uuid.UUID) *models.WhatsAppInstance {
	t.Helper()

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "Primary Sender",
		PhoneNumber:    "15550001111",
		Status:         models.InstanceStatusConnected,
		IsDefault:      true,
	}
	require.NoError(t, app.DB.Create(instance).Error)
	return instance
}

func TestNormalizeWhatsmeowDirectChatPhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "keeps e164", input: "+12025550100", want: "+12025550100"},
		{name: "normalizes digits only", input: "12025550100", want: "+12025550100"},
		{name: "normalizes international prefix", input: "00442079460123", want: "+442079460123"},
		{name: "strips formatting", input: "+1 (202) 555-0100", want: "+12025550100"},
		{name: "rejects invalid local number", input: "05550100", wantErr: true},
		{name: "rejects empty", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeWhatsmeowDirectChatPhone(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, errWhatsmeowDirectChatInvalidPhone)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCreateContact_StartChatUsesWhatsmeowResolver(t *testing.T) {
	t.Parallel()

	app := newContactsChatStartTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	instance := createTestWhatsmeowInstance(t, app, org.ID)

	resolver := &stubWhatsmeowContactResolver{
		result: &ResolvedWhatsmeowDirectContact{
			CanonicalPhone: "12025550100",
			ProfileName:    "Acme Support",
		},
	}
	app.WhatsmeowContactResolver = resolver

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": "+1 (202) 555-0100",
		"instance_id":  instance.ID.String(),
		"start_chat":   true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response ContactResponse
	testutil.ParseEnvelopeResponse(t, req, &response)

	require.Equal(t, "12025550100", response.PhoneNumber)
	require.Equal(t, "Acme Support", response.ProfileName)
	require.Equal(t, "open", response.Status)
	require.NotNil(t, response.AssignedUserID)
	require.Equal(t, user.ID, *response.AssignedUserID)
	require.Equal(t, instance.PhoneNumber, response.WhatsAppAccount)
	require.NotNil(t, response.InstanceID)
	require.Equal(t, instance.ID.String(), *response.InstanceID)
	require.Equal(t, instance.ID, resolver.receivedInstID)
	require.Equal(t, "+1 (202) 555-0100", resolver.receivedPhone)

	var stored models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "12025550100").First(&stored).Error)
	require.Equal(t, models.ChatStatusOpen, stored.Status)
	require.NotNil(t, stored.AssignedUserID)
	require.Equal(t, user.ID, *stored.AssignedUserID)
	require.Equal(t, instance.ID, *stored.InstanceID)
	require.Equal(t, instance.PhoneNumber, stored.WhatsAppAccount)
}

func TestCreateContact_StartChatRejectsUnknownRecipient(t *testing.T) {
	t.Parallel()

	app := newContactsChatStartTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	instance := createTestWhatsmeowInstance(t, app, org.ID)

	app.WhatsmeowContactResolver = &stubWhatsmeowContactResolver{
		err: errWhatsmeowDirectChatNotFound,
	}

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": "+1 202 555 0100",
		"instance_id":  instance.ID.String(),
		"start_chat":   true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "phone number is not registered on WhatsApp")
}
