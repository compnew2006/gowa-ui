package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type stubMessageProvider struct{}

func (m *stubMessageProvider) SendText(context.Context, string, string, string) (string, error) {
	return "wamid.stub." + uuid.NewString(), nil
}

func (m *stubMessageProvider) SendImage(context.Context, string, string, string, string) (string, error) {
	return "wamid.stub." + uuid.NewString(), nil
}

func (m *stubMessageProvider) SendDocument(context.Context, string, string, string, string) (string, error) {
	return "wamid.stub." + uuid.NewString(), nil
}

func (m *stubMessageProvider) SendVideo(context.Context, string, string, string, string) (string, error) {
	return "wamid.stub." + uuid.NewString(), nil
}

func (m *stubMessageProvider) SendAudio(context.Context, string, string, string) (string, error) {
	return "wamid.stub." + uuid.NewString(), nil
}

func (m *stubMessageProvider) MarkRead(context.Context, string, string) error {
	return nil
}

func (m *stubMessageProvider) SendReaction(context.Context, string, string, string) error {
	return nil
}

func (m *stubMessageProvider) RevokeMessage(context.Context, string, string) error {
	return nil
}

func (m *stubMessageProvider) GetMediaURL(context.Context, string, string) (string, error) {
	return "", nil
}

func (m *stubMessageProvider) DownloadMedia(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (m *stubMessageProvider) UploadMedia(context.Context, string, string, []byte) (string, error) {
	return "", nil
}

// --- ListContacts Tests ---

func TestApp_ListContacts(t *testing.T) {
	t.Parallel()

	t.Run("success with pagination", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		// Create 3 contacts
		for i := 0; i < 3; i++ {
			testutil.CreateTestContact(t, app.DB, org.ID)
		}

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "page", 1)
		testutil.SetQueryParam(req, "limit", 2)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
				Page     int                        `json:"page"`
				Limit    int                        `json:"limit"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(3), resp.Data.Total)
		assert.Len(t, resp.Data.Contacts, 2)
		assert.Equal(t, 1, resp.Data.Page)
		assert.Equal(t, 2, resp.Data.Limit)
	})

	t.Run("empty list", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(0), resp.Data.Total)
		assert.Empty(t, resp.Data.Contacts)
	})

	t.Run("filter by search on phone number", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		// Create contacts with distinct phone numbers
		uniquePhone := "+9998887776"
		testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber(uniquePhone))
		testutil.CreateTestContact(t, app.DB, org.ID) // different phone

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "search", "9998887776")

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(1), resp.Data.Total)
		assert.Len(t, resp.Data.Contacts, 1)
		assert.Equal(t, uniquePhone, resp.Data.Contacts[0].PhoneNumber)
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		// Create a contact in org2
		testutil.CreateTestContact(t, app.DB, org2.ID)

		// User from org1 should see no contacts
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org1.ID, user1.ID)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(0), resp.Data.Total)
		assert.Empty(t, resp.Data.Contacts)
	})

	t.Run("returns contact fields correctly", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Contacts, 1)
		assert.Equal(t, contact.ID, resp.Data.Contacts[0].ID)
		assert.Equal(t, contact.PhoneNumber, resp.Data.Contacts[0].PhoneNumber)
		assert.Equal(t, contact.ProfileName, resp.Data.Contacts[0].ProfileName)
		assert.Equal(t, "pending", resp.Data.Contacts[0].Status)
		assert.NotNil(t, resp.Data.Contacts[0].Tags)
	})

	t.Run("repairs private contact phone from conversation jid", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		legacyPhone := "219103612641296"
		canonicalPhone := "201000183177"
		contact := testutil.CreateTestContactWith(
			t,
			app.DB,
			org.ID,
			testutil.WithPhoneNumber(legacyPhone),
			testutil.WithContactAccount(account.Name),
		)
		require.NoError(t, app.DB.Model(contact).Update("profile_name", "Issam Accountant Egypt").Error)

		require.NoError(t, app.DB.Create(&models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			ConversationID:  canonicalPhone + "@s.whatsapp.net",
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "hello",
			Status:          models.MessageStatusDelivered,
		}).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Contacts, 1)
		assert.Equal(t, canonicalPhone, resp.Data.Contacts[0].PhoneNumber)

		var refreshed models.Contact
		require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
		assert.Equal(t, canonicalPhone, refreshed.PhoneNumber)
	})

	t.Run("default pagination with no params", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
				Page     int                        `json:"page"`
				Limit    int                        `json:"limit"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		// Default pagination: page=1, limit=50
		assert.Equal(t, 1, resp.Data.Page)
		assert.Equal(t, 50, resp.Data.Limit)
	})

	t.Run("admin role bypass can see assigned chats without contacts read permission", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)

		allPerms := testutil.GetOrCreateTestPermissions(t, app.DB)
		var chatReadPerms []models.Permission
		for _, perm := range allPerms {
			if perm.Resource == models.ResourceChat && perm.Action == models.ActionRead {
				chatReadPerms = append(chatReadPerms, perm)
			}
		}
		adminRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "admin", false, false, chatReadPerms)
		adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		assignee := testutil.CreateTestUser(t, app.DB, org.ID)
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
			"status":           models.ChatStatusOpen,
			"assigned_user_id": assignee.ID,
		}).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, adminUser.ID)
		testutil.SetQueryParam(req, "status", "open")

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Contacts, 1)
		assert.Equal(t, int64(1), resp.Data.Total)
		assert.Equal(t, contact.ID, resp.Data.Contacts[0].ID)
		require.NotNil(t, resp.Data.Contacts[0].AssignedUserID)
		assert.Equal(t, assignee.ID, *resp.Data.Contacts[0].AssignedUserID)
	})
}

// --- GetContact Tests ---

func TestApp_GetContact(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, contact.ID, resp.Data.ID)
		assert.Equal(t, contact.PhoneNumber, resp.Data.PhoneNumber)
		assert.Equal(t, contact.ProfileName, resp.Data.ProfileName)
		assert.Equal(t, "pending", resp.Data.Status)
		assert.NotNil(t, resp.Data.Tags)
		assert.Equal(t, 0, resp.Data.UnreadCount)
	})

	t.Run("closed chat remains readable", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		closedAt := time.Now().UTC()
		require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
			"status":            models.ChatStatusClosed,
			"assigned_user_id":  user.ID,
			"closed_at":         closedAt,
			"closed_by_user_id": user.ID,
		}).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, "closed", resp.Data.Status)
		assert.NotNil(t, resp.Data.ClosedAt)
		assert.NotNil(t, resp.Data.ClosedByUserID)
	})

	t.Run("resolves newsletter channel name from latest message metadata", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		channelID := "120363163799333272"
		channelJID := channelID + "@newsletter"
		channelName := "Daily Offers"

		contact := testutil.CreateTestContactWith(
			t,
			app.DB,
			org.ID,
			testutil.WithPhoneNumber(channelID),
			testutil.WithContactAccount(account.Name),
		)
		require.NoError(t, app.DB.Model(contact).Update("profile_name", channelID).Error)

		msg := &models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			ConversationID:  channelJID,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "Channel update",
			Status:          models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_channel_chat": true,
				"channel_jid":     channelJID,
				"channel_name":    channelName,
			},
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, channelName, resp.Data.ProfileName)
		assert.Equal(t, channelID, resp.Data.PhoneNumber)
	})

	t.Run("not found", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid ID", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		// Create contact in org2
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		// User from org1 should not access org2's contact
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("returns unread count", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Create an incoming unread message
		msg := &models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "Hello",
			Status:          models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, 1, resp.Data.UnreadCount)
	})
}

// --- GetContactSessionData Tests ---

func TestApp_GetContactSessionData(t *testing.T) {
	t.Parallel()

	t.Run("success with no session", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContactSessionData(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactSessionDataResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Nil(t, resp.Data.SessionID)
		assert.NotNil(t, resp.Data.SessionData)
		assert.NotNil(t, resp.Data.PanelConfig)
	})

	t.Run("success with active session", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Create an active chatbot session
		session := &models.ChatbotSession{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  org.ID,
			ContactID:       contact.ID,
			WhatsAppAccount: account.Name,
			PhoneNumber:     contact.PhoneNumber,
			Status:          models.SessionStatusActive,
			SessionData:     models.JSONB{"name": "Test User", "email": "test@example.com"},
			StartedAt:       time.Now(),
			LastActivityAt:  time.Now(),
		}
		require.NoError(t, app.DB.Create(session).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContactSessionData(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactSessionDataResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.NotNil(t, resp.Data.SessionID)
		assert.Equal(t, session.ID, *resp.Data.SessionID)
	})

	t.Run("not found - contact does not exist", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())

		err := app.GetContactSessionData(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid contact ID", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")

		err := app.GetContactSessionData(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetContactSessionData(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})
}

// --- AssignContact Tests ---

func TestApp_AssignContact(t *testing.T) {
	t.Parallel()

	t.Run("success - assign to user", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Create another user to assign to
		assignee := testutil.CreateTestUser(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": assignee.ID.String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Message        string     `json:"message"`
				AssignedUserID *uuid.UUID `json:"assigned_user_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Contains(t, resp.Data.Message, "assigned successfully")
		assert.NotNil(t, resp.Data.AssignedUserID)
		assert.Equal(t, assignee.ID, *resp.Data.AssignedUserID)

		// Verify in database
		var updatedContact models.Contact
		require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&updatedContact).Error)
		require.NotNil(t, updatedContact.AssignedUserID)
		assert.Equal(t, assignee.ID, *updatedContact.AssignedUserID)
	})

	t.Run("success - unassign", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		assignee := testutil.CreateTestUser(t, app.DB, org.ID)
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Pre-assign the contact
		require.NoError(t, app.DB.Model(&contact).Update("assigned_user_id", assignee.ID).Error)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": nil,
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Message        string     `json:"message"`
				AssignedUserID *uuid.UUID `json:"assigned_user_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Contains(t, resp.Data.Message, "assigned successfully")
		assert.Nil(t, resp.Data.AssignedUserID)

		// Verify in database
		var updatedContact models.Contact
		require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&updatedContact).Error)
		assert.Nil(t, updatedContact.AssignedUserID)
	})

	t.Run("forbidden - user without write permission", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)

		// Create a role with only contacts:read (no contacts:write)
		readOnlyRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "readonly", []string{
			"contacts:read",
		})
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readOnlyRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)
		assignee := testutil.CreateTestUser(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": assignee.ID.String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
	})

	t.Run("contact not found", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		assignee := testutil.CreateTestUser(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": assignee.ID.String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid contact ID", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": uuid.New().String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("assign to non-existent user", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": uuid.New().String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation - cannot assign contact from another org", func(t *testing.T) {
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))
		assignee := testutil.CreateTestUser(t, app.DB, org1.ID)

		// Contact belongs to org2
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"user_id": assignee.ID.String(),
		})
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.AssignContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})
}

// --- GetMessages Tests ---

func TestApp_GetMessages(t *testing.T) {
	t.Parallel()

	t.Run("success with messages", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		// Create messages with staggered timestamps
		now := time.Now()
		for i := 0; i < 3; i++ {
			msg := &models.Message{
				BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: now.Add(time.Duration(i) * time.Minute)},
				OrganizationID:  org.ID,
				WhatsAppAccount: account.Name,
				ContactID:       contact.ID,
				Direction:       models.DirectionIncoming,
				MessageType:     models.MessageTypeText,
				Content:         "Hello " + string(rune('A'+i)),
				Status:          models.MessageStatusDelivered,
			}
			require.NoError(t, app.DB.Create(msg).Error)
		}

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
				HasMore  bool                       `json:"has_more"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(3), resp.Data.Total)
		assert.Len(t, resp.Data.Messages, 3)
		assert.False(t, resp.Data.HasMore)
	})

	t.Run("empty messages", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(0), resp.Data.Total)
		assert.Empty(t, resp.Data.Messages)
	})

	t.Run("contact not found", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid contact ID", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		// Contact belongs to org2
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("default pagination limit", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		// No limit set - should default to 50

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
				Page     int                        `json:"page"`
				Limit    int                        `json:"limit"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, 1, resp.Data.Page)
		assert.Equal(t, 50, resp.Data.Limit)
	})

	t.Run("cursor-based pagination with before_id", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		// Create messages with staggered timestamps
		now := time.Now()
		var msgIDs []uuid.UUID
		for i := 0; i < 5; i++ {
			msg := &models.Message{
				BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: now.Add(time.Duration(i) * time.Minute)},
				OrganizationID:  org.ID,
				WhatsAppAccount: account.Name,
				ContactID:       contact.ID,
				Direction:       models.DirectionIncoming,
				MessageType:     models.MessageTypeText,
				Content:         "Message " + string(rune('A'+i)),
				Status:          models.MessageStatusDelivered,
			}
			require.NoError(t, app.DB.Create(msg).Error)
			msgIDs = append(msgIDs, msg.ID)
		}

		// Use before_id pointing to the 4th message (index 3), limit 2
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetQueryParam(req, "before_id", msgIDs[3].String())
		testutil.SetQueryParam(req, "limit", 2)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
				HasMore  bool                       `json:"has_more"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		// Should return messages before the 4th (so messages at index 1,2)
		assert.Len(t, resp.Data.Messages, 2)
		assert.True(t, resp.Data.HasMore)
	})

	t.Run("marks messages as read on page-based fetch", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		// Create an unread incoming message
		msg := &models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "Hello",
			Status:          models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		// Verify message was marked as read in the database
		var updatedMsg models.Message
		require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&updatedMsg).Error)
		assert.Equal(t, models.MessageStatusRead, updatedMsg.Status)
	})

	t.Run("message response includes correct fields", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.test123",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Test message content",
			Status:            models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 1)

		m := resp.Data.Messages[0]
		assert.Equal(t, msg.ID, m.ID)
		assert.Equal(t, contact.ID, m.ContactID)
		assert.Equal(t, models.DirectionIncoming, m.Direction)
		assert.Equal(t, models.MessageTypeText, m.MessageType)
		assert.Equal(t, "wamid.test123", m.WAMID)
		assert.NotNil(t, m.Content)
	})

	t.Run("group message response includes sender phone and group context", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		groupJID := "120363123456789012@g.us"
		groupContact := testutil.CreateTestContactWith(
			t,
			app.DB,
			org.ID,
			testutil.WithContactAccount(account.Name),
			testutil.WithPhoneNumber(groupJID),
		)
		require.NoError(t, app.DB.Model(groupContact).Updates(map[string]any{
			"profile_name": "Support Group",
			"metadata":     models.JSONB{"is_group_chat": true, "group_jid": groupJID},
		}).Error)

		senderPhone := "15551234567"
		incomingMsg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         groupContact.ID,
			WhatsAppMessageID: "wamid.group.inbound",
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Hello from group member",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_group":         true,
				"is_group_chat":    true,
				"group_jid":        groupJID,
				"sender_phone":     senderPhone,
				"sender_push_name": "Alice",
			},
		}
		require.NoError(t, app.DB.Create(incomingMsg).Error)

		replyMsg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(time.Minute)},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         groupContact.ID,
			WhatsAppMessageID: "wamid.group.reply",
			ConversationID:    groupJID,
			Direction:         models.DirectionOutgoing,
			MessageType:       models.MessageTypeText,
			Content:           "Reply from agent",
			Status:            models.MessageStatusSent,
			IsReply:           true,
			ReplyToMessageID:  &incomingMsg.ID,
		}
		require.NoError(t, app.DB.Create(replyMsg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", groupContact.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 2)

		var incomingResponse *handlers.MessageResponse
		var outgoingResponse *handlers.MessageResponse
		for i := range resp.Data.Messages {
			msg := resp.Data.Messages[i]
			if msg.Direction == models.DirectionIncoming {
				incomingResponse = &msg
			}
			if msg.Direction == models.DirectionOutgoing {
				outgoingResponse = &msg
			}
		}

		require.NotNil(t, incomingResponse)
		assert.True(t, incomingResponse.IsGroupChat)
		assert.Equal(t, groupJID, incomingResponse.ConversationID)
		assert.Equal(t, senderPhone, incomingResponse.SenderPhone)
		assert.Equal(t, "Alice", incomingResponse.SenderPushName)

		require.NotNil(t, outgoingResponse)
		require.NotNil(t, outgoingResponse.ReplyToMessage)
		assert.Equal(t, senderPhone, outgoingResponse.ReplyToMessage.SenderPhone)
	})

	t.Run("reply preview falls back to metadata when quoted row is unavailable", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.reply.metadata",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Reply without local quote row",
			Status:            models.MessageStatusDelivered,
			IsReply:           true,
			Metadata: models.JSONB{
				"reply_to_wamid":     "wamid.missing.original",
				"reply_sender_phone": "15551112222",
				"reply_preview_type": "image",
				"reply_preview_body": "",
				"reply_direction":    "incoming",
			},
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 1)
		require.NotNil(t, resp.Data.Messages[0].ReplyToMessage)
		assert.Equal(t, models.MessageTypeImage, resp.Data.Messages[0].ReplyToMessage.MessageType)
		assert.Equal(t, "15551112222", resp.Data.Messages[0].ReplyToMessage.SenderPhone)
	})

	t.Run("group conversation fetch aggregates messages across participant contacts", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		groupJID := "120363777777777777@g.us"
		participantContact1 := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
		participantContact2 := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		msg1 := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         participantContact1.ID,
			WhatsAppMessageID: "wamid.group.aggregate.1",
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "First participant message",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_group_chat": true,
				"group_jid":     groupJID,
				"sender_phone":  "15550000001",
			},
		}
		msg2 := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(time.Minute)},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         participantContact2.ID,
			WhatsAppMessageID: "wamid.group.aggregate.2",
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Second participant message",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_group_chat": true,
				"group_jid":     groupJID,
				"sender_phone":  "15550000002",
			},
		}
		require.NoError(t, app.DB.Create(msg1).Error)
		require.NoError(t, app.DB.Create(msg2).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", participantContact1.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(2), resp.Data.Total)
		require.Len(t, resp.Data.Messages, 2)

		assert.Equal(t, groupJID, resp.Data.Messages[0].ConversationID)
		assert.Equal(t, groupJID, resp.Data.Messages[1].ConversationID)
		assert.True(t, resp.Data.Messages[0].IsGroupChat)
		assert.True(t, resp.Data.Messages[1].IsGroupChat)
	})

	t.Run("group conversation fetch scopes messages by instance", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		groupJID := "120363888888888888@g.us"
		instanceA := uuid.New()
		instanceB := uuid.New()

		contactA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
		contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
		require.NoError(t, app.DB.Model(contactA).Update("instance_id", instanceA).Error)
		require.NoError(t, app.DB.Model(contactB).Update("instance_id", instanceB).Error)

		msgA := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:    org.ID,
			InstanceID:        &instanceA,
			WhatsAppAccount:   account.Name,
			ContactID:         contactA.ID,
			WhatsAppMessageID: "wamid.group.instance.a",
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Message from instance A",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_group_chat": true,
				"group_jid":     groupJID,
				"sender_phone":  "15551110001",
			},
		}
		msgB := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(time.Minute)},
			OrganizationID:    org.ID,
			InstanceID:        &instanceB,
			WhatsAppAccount:   account.Name,
			ContactID:         contactB.ID,
			WhatsAppMessageID: "wamid.group.instance.b",
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Message from instance B",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"is_group_chat": true,
				"group_jid":     groupJID,
				"sender_phone":  "15551110002",
			},
		}
		require.NoError(t, app.DB.Create(msgA).Error)
		require.NoError(t, app.DB.Create(msgB).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contactA.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 1)
		assert.Equal(t, "Message from instance A", resp.Data.Messages[0].Content.(map[string]interface{})["body"])
		assert.Equal(t, groupJID, resp.Data.Messages[0].ConversationID)
		assert.Equal(t, "wamid.group.instance.a", resp.Data.Messages[0].WAMID)
	})

	t.Run("group conversation fetch includes legacy system events without conversation id", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		groupJID := "120363999000000000@g.us"
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
		require.NoError(t, app.DB.Model(contact).Update("metadata", models.JSONB{
			"is_group_chat": true,
			"group_jid":     groupJID,
		}).Error)

		require.NoError(t, app.DB.Create(&models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().UTC().Add(-time.Minute)},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			ConversationID:    groupJID,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Group message",
			Status:            models.MessageStatusDelivered,
			WhatsAppMessageID: "wamid.group.legacy.seed",
			Metadata: models.JSONB{
				"is_group_chat": true,
				"group_jid":     groupJID,
				"sender_phone":  "15550000003",
			},
		}).Error)
		require.NoError(t, app.DB.Create(&models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().UTC()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			Direction:       models.DirectionOutgoing,
			MessageType:     models.MessageTypeText,
			Content:         "System: Claim Agent claimed this chat.",
			Status:          models.MessageStatusSent,
			Metadata: models.JSONB{
				"system_event":  true,
				"event_type":    "chat_claimed",
				"is_group_chat": true,
			},
		}).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetQueryParam(req, "limit", 50)

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(2), resp.Data.Total)
		require.Len(t, resp.Data.Messages, 2)

		foundClaimMessage := false
		for _, msg := range resp.Data.Messages {
			contentMap, ok := msg.Content.(map[string]interface{})
			if !ok {
				continue
			}
			body, _ := contentMap["body"].(string)
			if strings.Contains(body, "claimed this chat") {
				foundClaimMessage = true
				assert.Equal(t, true, msg.Metadata["system_event"])
				assert.Equal(t, "chat_claimed", msg.Metadata["event_type"])
			}
		}
		assert.True(t, foundClaimMessage)
	})

	t.Run("suppresses synthetic placeholder rows when media companion exists", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		wamid := "wamid.synthetic.placeholder"
		placeholderMessage := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: wamid,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "[Unsupported message type]",
			Status:            models.MessageStatusRead,
		}
		mediaMessage := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(time.Second)},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: wamid,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeImage,
			Content:           "Image caption",
			MediaURL:          "images/sample.jpg",
			MediaMimeType:     "image/jpeg",
			MediaFilename:     "sample.jpg",
			Status:            models.MessageStatusRead,
		}
		require.NoError(t, app.DB.Create(placeholderMessage).Error)
		require.NoError(t, app.DB.Create(mediaMessage).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 1)
		assert.Equal(t, models.MessageTypeImage, resp.Data.Messages[0].MessageType)
		assert.Equal(t, wamid, resp.Data.Messages[0].WAMID)
	})

	t.Run("keeps unsupported placeholders as unsupported in response", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.revoke.legacy",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "[Unsupported message type]",
			Status:            models.MessageStatusRead,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.GetMessages(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		require.Len(t, resp.Data.Messages, 1)
		bodyMap, ok := resp.Data.Messages[0].Content.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "[Unsupported message type]", bodyMap["body"])
	})
}

// --- SendMessage Tests ---

func TestApp_SendMessage(t *testing.T) {
	t.Parallel()

	t.Run("success - text message", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockWhatsAppServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := createTestAccount(t, app, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "Hello from agent!",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.MessageResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, contact.ID, resp.Data.ContactID)
		assert.Equal(t, models.DirectionOutgoing, resp.Data.Direction)
		assert.Equal(t, models.MessageTypeText, resp.Data.MessageType)
	})

	t.Run("success - whatsmeow persists outbound account for filtered history", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		app.Config.WhatsApp.Provider = "whatsmeow"
		app.MessageProvider = &stubMessageProvider{}

		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		instance := models.WhatsAppInstance{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			Name:           "Primary",
			PhoneNumber:    "201007181781",
			Status:         models.InstanceStatusConnected,
			IsDefault:      true,
			Settings:       models.JSONB{},
		}
		require.NoError(t, app.DB.Create(&instance).Error)

		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(""))

		sendReq := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "visible with account filter",
			},
		})
		testutil.SetAuthContext(sendReq, org.ID, user.ID)
		testutil.SetPathParam(sendReq, "id", contact.ID.String())

		err := app.SendMessage(sendReq)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(sendReq))

		var sendResp struct {
			Data handlers.MessageResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(sendReq), &sendResp))
		assert.Equal(t, instance.PhoneNumber, sendResp.Data.WhatsAppAccount)

		var stored models.Message
		require.NoError(t, app.DB.Where("id = ?", sendResp.Data.ID).First(&stored).Error)
		assert.Equal(t, instance.PhoneNumber, stored.WhatsAppAccount)

		messagesReq := testutil.NewGETRequest(t)
		testutil.SetAuthContext(messagesReq, org.ID, user.ID)
		testutil.SetPathParam(messagesReq, "id", contact.ID.String())
		testutil.SetQueryParam(messagesReq, "account", instance.PhoneNumber)
		testutil.SetQueryParam(messagesReq, "limit", 50)

		err = app.GetMessages(messagesReq)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(messagesReq))

		var messagesResp struct {
			Data struct {
				Messages []handlers.MessageResponse `json:"messages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(messagesReq), &messagesResp))

		found := false
		for _, message := range messagesResp.Data.Messages {
			if message.ID == sendResp.Data.ID {
				found = true
				assert.Equal(t, instance.PhoneNumber, message.WhatsAppAccount)
				break
			}
		}
		assert.True(t, found, "expected sent message to be included when filtering by whatsapp account")
	})

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Send non-JSON body
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetBody([]byte("not-json"))
		req := &fastglue.Request{RequestCtx: ctx}
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("contact not found", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockWhatsAppServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "Hello!",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid contact ID", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "Hello!",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockWhatsAppServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		// Contact belongs to org2
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "Hello!",
			},
		})
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("no whatsapp account configured", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockWhatsAppServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		// Contact with no WhatsApp account set and no accounts in org
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "Hello!",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("success with reply context", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockWhatsAppServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := createTestAccount(t, app, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		// Create an original message to reply to
		origMsg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.original123",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Original message",
			Status:            models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(origMsg).Error)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"type": "text",
			"content": map[string]string{
				"body": "This is a reply",
			},
			"reply_to_message_id": origMsg.ID.String(),
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.MessageResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.True(t, resp.Data.IsReply)
		assert.NotNil(t, resp.Data.ReplyToMessageID)
		assert.NotNil(t, resp.Data.ReplyToMessage)
	})
}

// --- SendReaction Tests ---

func TestApp_SendReaction(t *testing.T) {
	t.Parallel()

	t.Run("success - add reaction", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, withHTTPClient(&http.Client{}))
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.react123",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Hello",
			Status:            models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetPathParam(req, "message_id", msg.ID.String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				MessageID string `json:"message_id"`
				Reactions []struct {
					Emoji    string `json:"emoji"`
					FromUser string `json:"from_user"`
				} `json:"reactions"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, msg.ID.String(), resp.Data.MessageID)
		require.Len(t, resp.Data.Reactions, 1)
		assert.Equal(t, "\U0001F44D", resp.Data.Reactions[0].Emoji)
		assert.Equal(t, user.ID.String(), resp.Data.Reactions[0].FromUser)
	})

	t.Run("success - remove reaction", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, withHTTPClient(&http.Client{}))
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		// Create message with an existing reaction
		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.remove-react",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Hello",
			Status:            models.MessageStatusDelivered,
			Metadata: models.JSONB{
				"reactions": []interface{}{
					map[string]interface{}{
						"emoji":     "\U0001F44D",
						"from_user": user.ID.String(),
					},
				},
			},
		}
		require.NoError(t, app.DB.Create(msg).Error)

		// Send empty emoji to remove reaction
		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetPathParam(req, "message_id", msg.ID.String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Reactions []interface{} `json:"reactions"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		// Reaction should be removed (empty or nil)
		assert.Empty(t, resp.Data.Reactions)
	})

	t.Run("contact not found", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", uuid.New().String())
		testutil.SetPathParam(req, "message_id", uuid.New().String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetPathParam(req, "message_id", uuid.New().String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid contact ID", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", "not-a-uuid")
		testutil.SetPathParam(req, "message_id", uuid.New().String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("invalid message ID", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetPathParam(req, "message_id", "not-a-uuid")

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	})

	t.Run("cross-org isolation", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org2.ID)
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org2.ID,
			WhatsAppAccount:   account.Name,
			ContactID:         contact.ID,
			WhatsAppMessageID: "wamid.cross-org",
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "Hello",
			Status:            models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)

		req := testutil.NewJSONRequest(t, map[string]interface{}{
			"emoji": "\U0001F44D",
		})
		testutil.SetAuthContext(req, org1.ID, user1.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		testutil.SetPathParam(req, "message_id", msg.ID.String())

		err := app.SendReaction(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})
}

// --- ListContacts additional tests ---

func TestApp_ListContacts_SearchByProfileName(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// Create contact with a unique profile name
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "+1111111111",
		ProfileName:    "UniqueAlphaName",
	}
	require.NoError(t, app.DB.Create(contact).Error)

	// Create another contact with a different name
	testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "search", "UniqueAlpha")

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.Contacts, 1)
	assert.Equal(t, "UniqueAlphaName", resp.Data.Contacts[0].ProfileName)
}

func TestApp_ListContacts_Page2(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// Create 5 contacts
	for i := 0; i < 5; i++ {
		testutil.CreateTestContact(t, app.DB, org.ID)
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "page", 2)
	testutil.SetQueryParam(req, "limit", 2)

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
			Page     int                        `json:"page"`
			Limit    int                        `json:"limit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(5), resp.Data.Total)
	assert.Len(t, resp.Data.Contacts, 2)
	assert.Equal(t, 2, resp.Data.Page)
	assert.Equal(t, 2, resp.Data.Limit)
}

// --- GetContact additional tests ---

func TestApp_GetContact_WithAssignedUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Assign the contact
	require.NoError(t, app.DB.Model(&contact).Update("assigned_user_id", assignee.ID).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.NotNil(t, resp.Data.AssignedUserID)
	assert.Equal(t, assignee.ID, *resp.Data.AssignedUserID)
}

func TestApp_GetContact_MultipleUnreadMessages(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create 3 unread incoming messages
	for i := 0; i < 3; i++ {
		msg := &models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "Unread message",
			Status:          models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)
	}

	// Create 1 read incoming message
	readMsg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "Read message",
		Status:          models.MessageStatusRead,
	}
	require.NoError(t, app.DB.Create(readMsg).Error)

	// Create 1 outgoing message (should not count as unread)
	outMsg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionOutgoing,
		MessageType:     models.MessageTypeText,
		Content:         "Outgoing message",
		Status:          models.MessageStatusSent,
	}
	require.NoError(t, app.DB.Create(outMsg).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	// Only 3 incoming delivered (not read) messages should be counted
	assert.Equal(t, 3, resp.Data.UnreadCount)
}

// --- GetContactSessionData additional tests ---

func TestApp_GetContactSessionData_CompletedSession(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	completedAt := time.Now()
	session := &models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		PhoneNumber:     contact.PhoneNumber,
		Status:          models.SessionStatusCompleted,
		SessionData:     models.JSONB{"order_id": "ORD-123", "amount": 99.99},
		StartedAt:       time.Now().Add(-1 * time.Hour),
		LastActivityAt:  time.Now(),
		CompletedAt:     &completedAt,
	}
	require.NoError(t, app.DB.Create(session).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetContactSessionData(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ContactSessionDataResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.NotNil(t, resp.Data.SessionID)
	assert.Equal(t, session.ID, *resp.Data.SessionID)
}

func TestApp_GetContactSessionData_MostRecentSessionReturned(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create an older completed session
	oldSession := &models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(-2 * time.Hour)},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		PhoneNumber:     contact.PhoneNumber,
		Status:          models.SessionStatusCompleted,
		SessionData:     models.JSONB{"key": "old"},
		StartedAt:       time.Now().Add(-2 * time.Hour),
		LastActivityAt:  time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, app.DB.Create(oldSession).Error)

	// Create a newer active session
	newSession := &models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		PhoneNumber:     contact.PhoneNumber,
		Status:          models.SessionStatusActive,
		SessionData:     models.JSONB{"key": "new"},
		StartedAt:       time.Now(),
		LastActivityAt:  time.Now(),
	}
	require.NoError(t, app.DB.Create(newSession).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetContactSessionData(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ContactSessionDataResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	// Should return the most recent session
	require.NotNil(t, resp.Data.SessionID)
	assert.Equal(t, newSession.ID, *resp.Data.SessionID)
}

// --- AssignContact additional tests ---

func TestApp_AssignContact_ReassignToAnotherUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee1 := testutil.CreateTestUser(t, app.DB, org.ID)
	assignee2 := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Pre-assign to assignee1
	require.NoError(t, app.DB.Model(&contact).Update("assigned_user_id", assignee1.ID).Error)

	// Reassign to assignee2
	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"user_id": assignee2.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			AssignedUserID *uuid.UUID `json:"assigned_user_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.NotNil(t, resp.Data.AssignedUserID)
	assert.Equal(t, assignee2.ID, *resp.Data.AssignedUserID)

	// Verify in database
	var updatedContact models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&updatedContact).Error)
	require.NotNil(t, updatedContact.AssignedUserID)
	assert.Equal(t, assignee2.ID, *updatedContact.AssignedUserID)
}

func TestApp_AssignContact_AssignUserFromDifferentOrg(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
	user := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org1.ID)

	// Create a user in a different org
	otherOrgUser := testutil.CreateTestUser(t, app.DB, org2.ID)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"user_id": otherOrgUser.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	// User from a different org should not be found
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_CreateContact_WithInstanceID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Sales Instance",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instance).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": "15551234567",
		"profile_name": "Alice",
		"instance_id":  instance.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.NotNil(t, resp.Data.InstanceID)
	assert.Equal(t, instance.ID.String(), *resp.Data.InstanceID)

	var created models.Contact
	require.NoError(t, app.DB.Where("id = ?", resp.Data.ID).First(&created).Error)
	require.NotNil(t, created.InstanceID)
	assert.Equal(t, instance.ID, *created.InstanceID)
}

func TestApp_CreateContact_InvalidInstanceID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": "15550001111",
		"instance_id":  "not-a-uuid",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_ListContacts_FilterByInstanceAndChatType(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	instanceA := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance A",
		Status:         models.InstanceStatusConnected,
	}
	instanceB := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance B",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instanceA).Error)
	require.NoError(t, app.DB.Create(instanceB).Error)

	privateContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15550000001"))
	groupContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("120363000000000001@g.us"))
	channelContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("120363000000000002@newsletter"))

	require.NoError(t, app.DB.Model(privateContact).Update("instance_id", instanceA.ID).Error)
	require.NoError(t, app.DB.Model(groupContact).Updates(map[string]any{
		"instance_id": instanceB.ID,
		"metadata":    models.JSONB{"is_group_chat": true},
	}).Error)
	require.NoError(t, app.DB.Model(channelContact).Updates(map[string]any{
		"instance_id": instanceA.ID,
		"metadata":    models.JSONB{"is_channel_chat": true},
	}).Error)

	t.Run("filters by instance and chat types together", func(t *testing.T) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "instance_id", instanceA.ID.String())
		testutil.SetQueryParam(req, "chat_types", "private,channel")

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(2), resp.Data.Total)
		assert.Len(t, resp.Data.Contacts, 2)
	})

	t.Run("filters groups only", func(t *testing.T) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "chat_types", "group")

		err := app.ListContacts(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, int64(1), resp.Data.Total)
		require.Len(t, resp.Data.Contacts, 1)
		assert.Equal(t, groupContact.ID, resp.Data.Contacts[0].ID)
	})
}

func TestApp_ListContacts_RestrictedUserFiltersByAllowedInstanceEvenWhenOrgStrictDisabled(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	instanceA := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance A",
		Status:         models.InstanceStatusConnected,
	}
	instanceB := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance B",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instanceA).Error)
	require.NoError(t, app.DB.Create(instanceB).Error)

	contactA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15550000101"))
	contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15550000102"))
	require.NoError(t, app.DB.Model(contactA).Update("instance_id", instanceA.ID).Error)
	require.NoError(t, app.DB.Model(contactB).Update("instance_id", instanceB.ID).Error)

	enableRestrictedInstanceVisibilityWithStrict(t, app, org.ID, user.ID, instanceA.ID, false)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.Contacts, 1)
	assert.Equal(t, contactA.ID, resp.Data.Contacts[0].ID)
}

func TestApp_ListContacts_RestrictedUserFiltersByAllowedInstanceWhenRestrictionsDisabled(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	instanceA := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance A",
		Status:         models.InstanceStatusConnected,
	}
	instanceB := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance B",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instanceA).Error)
	require.NoError(t, app.DB.Create(instanceB).Error)

	contactA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15550000201"))
	contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15550000202"))
	require.NoError(t, app.DB.Model(contactA).Update("instance_id", instanceA.ID).Error)
	require.NoError(t, app.DB.Model(contactB).Update("instance_id", instanceB.ID).Error)

	enableRestrictedInstanceVisibilityWithStrictAndEnabled(t, app, org.ID, user.ID, instanceA.ID, false, false)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.Contacts, 1)
	assert.Equal(t, contactA.ID, resp.Data.Contacts[0].ID)
}

func TestApp_ListContacts_FilterPrivateChats_WithGroupHistory(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	contactWithDirectHistory := testutil.CreateTestContactWith(
		t,
		app.DB,
		org.ID,
		testutil.WithContactAccount(account.Name),
		testutil.WithPhoneNumber("+15551000001"),
	)
	require.NoError(t, app.DB.Model(contactWithDirectHistory).Update("metadata", models.JSONB{
		"is_group_chat": true,
		"group_jid":     "120363555555555555@g.us",
	}).Error)

	groupOnlyContact := testutil.CreateTestContactWith(
		t,
		app.DB,
		org.ID,
		testutil.WithContactAccount(account.Name),
		testutil.WithPhoneNumber("+15551000002"),
	)
	require.NoError(t, app.DB.Model(groupOnlyContact).Update("metadata", models.JSONB{
		"is_group_chat": true,
		"group_jid":     "120363666666666666@g.us",
	}).Error)

	groupJID := "120363555555555555@g.us"
	directConversationID := "15551000001@s.whatsapp.net"
	now := time.Now().UTC()

	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: now.Add(-2 * time.Minute)},
		OrganizationID:    org.ID,
		WhatsAppAccount:   account.Name,
		ContactID:         contactWithDirectHistory.ID,
		WhatsAppMessageID: "wamid.private.filter.group.history",
		ConversationID:    groupJID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "Group history",
		Status:            models.MessageStatusDelivered,
		Metadata:          models.JSONB{"is_group_chat": true, "group_jid": groupJID},
	}).Error)

	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: now.Add(-1 * time.Minute)},
		OrganizationID:    org.ID,
		WhatsAppAccount:   account.Name,
		ContactID:         contactWithDirectHistory.ID,
		WhatsAppMessageID: "wamid.private.filter.direct.history",
		ConversationID:    directConversationID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "Direct history",
		Status:            models.MessageStatusDelivered,
		Metadata:          models.JSONB{},
	}).Error)

	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: now},
		OrganizationID:    org.ID,
		WhatsAppAccount:   account.Name,
		ContactID:         groupOnlyContact.ID,
		WhatsAppMessageID: "wamid.private.filter.group.only",
		ConversationID:    "120363666666666666@g.us",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "Group only",
		Status:            models.MessageStatusDelivered,
		Metadata:          models.JSONB{"is_group_chat": true, "group_jid": "120363666666666666@g.us"},
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "chat_types", "private")

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	require.Len(t, resp.Data.Contacts, 1)
	assert.Equal(t, int64(1), resp.Data.Total)
	assert.Equal(t, contactWithDirectHistory.ID, resp.Data.Contacts[0].ID)
}

func TestApp_ReopenChat(t *testing.T) {
	t.Parallel()

	t.Run("success reopens closed chat as pending and unassigned", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		closedAt := time.Now().UTC()
		require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
			"status":            models.ChatStatusClosed,
			"assigned_user_id":  assignee.ID,
			"closed_at":         &closedAt,
			"closed_by_user_id": assignee.ID,
		}).Error)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("PUT")
		testutil.SetAuthContext(req, org.ID, adminUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.ReopenChat(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ContactResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		assert.Equal(t, "pending", resp.Data.Status)
		assert.Nil(t, resp.Data.AssignedUserID)
		assert.Nil(t, resp.Data.ClosedAt)
		assert.Nil(t, resp.Data.ClosedByUserID)

		var refreshed models.Contact
		require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
		assert.Equal(t, models.ChatStatusPending, refreshed.EffectiveStatus())
		assert.Nil(t, refreshed.AssignedUserID)
		assert.Nil(t, refreshed.ClosedAt)
		assert.Nil(t, refreshed.ClosedByUserID)
	})

	t.Run("returns conflict when chat is not closed", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("PUT")
		testutil.SetAuthContext(req, org.ID, adminUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.ReopenChat(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusConflict, testutil.GetResponseStatusCode(req))
	})
}

func TestApp_ClaimChat_CreatesSystemMessage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Claim Agent"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":               models.ChatStatusPending,
		"assigned_user_id":     nil,
		"last_message_at":      nil,
		"last_message_preview": "",
	}).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var claimResp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &claimResp))
	require.NotNil(t, claimResp.Data.AssignedUserID)
	assert.Equal(t, adminUser.ID, *claimResp.Data.AssignedUserID)
	assert.Equal(t, "Claim Agent", claimResp.Data.AssignedUserName)

	var systemMessage models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_claimed").Order("created_at DESC").First(&systemMessage).Error)
	assert.Equal(t, models.DirectionOutgoing, systemMessage.Direction)
	assert.Equal(t, models.MessageTypeText, systemMessage.MessageType)
	assert.Equal(t, models.MessageStatusSent, systemMessage.Status)
	assert.Equal(t, true, systemMessage.Metadata["system_event"])
	assert.Contains(t, systemMessage.Content, "claimed this chat")
}

func TestApp_CloseChat_CreatesSystemMessage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Close Agent"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":           models.ChatStatusOpen,
		"assigned_user_id": adminUser.ID,
	}).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var systemMessage models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_closed").Order("created_at DESC").First(&systemMessage).Error)
	assert.Equal(t, models.DirectionOutgoing, systemMessage.Direction)
	assert.Equal(t, models.MessageTypeText, systemMessage.MessageType)
	assert.Equal(t, models.MessageStatusSent, systemMessage.Status)
	assert.Equal(t, true, systemMessage.Metadata["system_event"])
	assert.Contains(t, systemMessage.Content, "closed this chat")
	assert.Equal(t, "Close Agent", systemMessage.Metadata["closed_by_user_name"])
}

func TestApp_ClaimChat_AlreadyAssignedToCurrentUser_StillCreatesSystemMessage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Claim Agent"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":               models.ChatStatusOpen,
		"assigned_user_id":     adminUser.ID,
		"last_message_at":      nil,
		"last_message_preview": "",
	}).Error)

	var beforeCount int64
	require.NoError(t, app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_claimed").
		Count(&beforeCount).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var afterCount int64
	require.NoError(t, app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_claimed").
		Count(&afterCount).Error)
	assert.Equal(t, beforeCount+1, afterCount)

	var systemMessage models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_claimed").Order("created_at DESC").First(&systemMessage).Error)
	assert.Equal(t, models.DirectionOutgoing, systemMessage.Direction)
	assert.Equal(t, models.MessageTypeText, systemMessage.MessageType)
	assert.Equal(t, models.MessageStatusSent, systemMessage.Status)
	assert.Equal(t, true, systemMessage.Metadata["system_event"])
	assert.Contains(t, systemMessage.Content, "claimed this chat")
}

func TestApp_ClaimChat_SystemMessageUsesResolvedAccountWhenContactAccountMissing(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Claim Agent"))

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "201007181781",
		Status:         models.InstanceStatusConnected,
		Settings:       models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&instance).Error)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(""))
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":            models.ChatStatusPending,
		"assigned_user_id":  nil,
		"whats_app_account": "",
		"instance_id":       instance.ID,
	}).Error)

	conversationID := strings.TrimSpace(contact.PhoneNumber) + "@s.whatsapp.net"
	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().UTC().Add(-time.Minute)},
		OrganizationID:  org.ID,
		InstanceID:      &instance.ID,
		WhatsAppAccount: instance.PhoneNumber,
		ContactID:       contact.ID,
		ConversationID:  conversationID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "Incoming seed",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"is_group_chat": false,
		},
	}).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var systemMessage models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND metadata->>'event_type' = ?", contact.ID, "chat_claimed").Order("created_at DESC").First(&systemMessage).Error)
	assert.Equal(t, instance.PhoneNumber, systemMessage.WhatsAppAccount)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, instance.PhoneNumber, refreshed.WhatsAppAccount)
}

func TestApp_ClaimChat_GroupConversation_SystemMessageVisibleInHistory(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Noie Many"))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
	groupJID := "120363999999999999@g.us"

	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":           models.ChatStatusPending,
		"assigned_user_id": nil,
		"metadata": models.JSONB{
			"is_group_chat": true,
			"group_jid":     groupJID,
		},
	}).Error)

	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().UTC().Add(-time.Minute)},
		OrganizationID:    org.ID,
		WhatsAppAccount:   account.Name,
		ContactID:         contact.ID,
		ConversationID:    groupJID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "Incoming group message",
		Status:            models.MessageStatusDelivered,
		WhatsAppMessageID: "wamid.group.claim.seed",
		Metadata: models.JSONB{
			"is_group_chat": true,
			"group_jid":     groupJID,
			"sender_phone":  "15550000001",
		},
	}).Error)

	claimReq := testutil.NewRequest(t)
	claimReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(claimReq, org.ID, adminUser.ID)
	testutil.SetPathParam(claimReq, "id", contact.ID.String())

	err := app.ClaimChat(claimReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(claimReq))

	messagesReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(messagesReq, org.ID, adminUser.ID)
	testutil.SetPathParam(messagesReq, "id", contact.ID.String())
	testutil.SetQueryParam(messagesReq, "limit", 50)

	err = app.GetMessages(messagesReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(messagesReq))

	var resp struct {
		Data struct {
			Messages []handlers.MessageResponse `json:"messages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(messagesReq), &resp))
	require.NotEmpty(t, resp.Data.Messages)

	foundClaimMessage := false
	for _, msg := range resp.Data.Messages {
		contentMap, ok := msg.Content.(map[string]interface{})
		if !ok {
			continue
		}
		body, _ := contentMap["body"].(string)
		if !strings.Contains(body, "claimed this chat") {
			continue
		}

		foundClaimMessage = true
		assert.Equal(t, groupJID, msg.ConversationID)
		assert.True(t, msg.IsGroupChat)
		assert.Equal(t, true, msg.Metadata["system_event"])
		assert.Equal(t, "chat_claimed", msg.Metadata["event_type"])
	}
	assert.True(t, foundClaimMessage)
}

func TestApp_ListContacts_IncludesAssignedUserName(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(
		t,
		app.DB,
		org.ID,
		testutil.WithRoleID(&adminRole.ID),
		testutil.WithFullName("Assigned Agent"),
	)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": assignee.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetQueryParam(req, "status", "open")

	err := app.ListContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.Contacts, 1)
	require.NotNil(t, resp.Data.Contacts[0].AssignedUserID)
	assert.Equal(t, assignee.ID, *resp.Data.Contacts[0].AssignedUserID)
	assert.Equal(t, "Assigned Agent", resp.Data.Contacts[0].AssignedUserName)
}

func TestApp_DeleteContact_PermissionBased(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	managerRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "manager", []string{"contacts:read", "contacts:delete"})
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)

	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	managerUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&managerRole.ID))
	superAdminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID), testutil.WithSuperAdmin())

	t.Run("manager with contacts delete permission can delete", func(t *testing.T) {
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("DELETE")
		testutil.SetAuthContext(req, org.ID, managerUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.DeleteContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var deleted models.Contact
		require.NoError(t, app.DB.Unscoped().Where("id = ?", contact.ID).First(&deleted).Error)
		assert.True(t, deleted.DeletedAt.Valid)
	})

	t.Run("agent without contacts delete permission is blocked", func(t *testing.T) {
		contact := testutil.CreateTestContact(t, app.DB, org.ID)
		agentUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("DELETE")
		testutil.SetAuthContext(req, org.ID, agentUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.DeleteContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

		var stillExists models.Contact
		require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&stillExists).Error)
	})

	t.Run("admin can delete contact", func(t *testing.T) {
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("DELETE")
		testutil.SetAuthContext(req, org.ID, adminUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.DeleteContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var deleted models.Contact
		require.NoError(t, app.DB.Unscoped().Where("id = ?", contact.ID).First(&deleted).Error)
		assert.True(t, deleted.DeletedAt.Valid)
	})

	t.Run("super admin can delete contact", func(t *testing.T) {
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("DELETE")
		testutil.SetAuthContext(req, org.ID, superAdminUser.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())

		err := app.DeleteContact(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	})
}

func TestApp_DeleteContact_PreservesConversationData(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	now := time.Now().UTC()
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":               models.ChatStatusOpen,
		"assigned_user_id":     assignee.ID,
		"last_message_at":      now,
		"last_message_preview": "old message",
	}).Error)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "whatsmeow",
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "old history",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var deletedContact models.Contact
	require.NoError(t, app.DB.Unscoped().Where("id = ?", contact.ID).First(&deletedContact).Error)
	assert.True(t, deletedContact.DeletedAt.Valid)
	assert.Equal(t, models.ChatStatusOpen, deletedContact.Status)
	require.NotNil(t, deletedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *deletedContact.AssignedUserID)
	assert.Equal(t, "old message", deletedContact.LastMessagePreview)

	var persistedMessage models.Message
	require.NoError(t, app.DB.Where("id = ?", message.ID).First(&persistedMessage).Error)
	assert.False(t, persistedMessage.DeletedAt.Valid)
}
