package handlers_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

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
		assert.Equal(t, "active", resp.Data.Contacts[0].Status)
		assert.NotNil(t, resp.Data.Contacts[0].Tags)
	})

	t.Run("filter by whatsapp account", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		// Two contacts on account A, one on account B.
		match1 := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("account-a"))
		testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("account-a"))
		testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("account-b"))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "account", "account-a")

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
		for _, c := range resp.Data.Contacts {
			assert.Equal(t, "account-a", c.WhatsAppAccount)
		}
		_ = match1
	})

	t.Run("exclude groups and newsletters", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		// One individual contact and one group contact (WhatsApp group-ID prefix).
		individual := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("+15551230001"))
		testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber("120362123456789"))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "exclude_groups", "true")

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
		assert.Equal(t, individual.ID, resp.Data.Contacts[0].ID)
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
		assert.Equal(t, "active", resp.Data.Status)
		assert.NotNil(t, resp.Data.Tags)
		assert.Equal(t, 0, resp.Data.UnreadCount)
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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
}

// --- SendMessage Tests ---

func TestApp_SendMessage(t *testing.T) {
	t.Parallel()

	t.Run("success - text message", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockGowaServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		account := createTestAccount(t, app, org.ID)
		contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

		req := testutil.NewJSONRequest(t, map[string]any{
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

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		// Send non-JSON body
		ctx := &fasthttp.RequestCtx{}
		ctx.Init(&fasthttp.Request{}, nil, nil)
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
		mockServer := newMockGowaServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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
		mockServer := newMockGowaServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org1 := testutil.CreateTestOrganization(t, app.DB)
		org2 := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
		user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

		// Contact belongs to org2
		contact := testutil.CreateTestContact(t, app.DB, org2.ID)

		req := testutil.NewJSONRequest(t, map[string]any{
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
		mockServer := newMockGowaServer()
		defer mockServer.close()

		app := newMsgTestApp(t, mockServer)
		org := testutil.CreateTestOrganization(t, app.DB)
		adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
		// Contact with no WhatsApp account set and no accounts in org
		contact := testutil.CreateTestContact(t, app.DB, org.ID)

		req := testutil.NewJSONRequest(t, map[string]any{
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
		mockServer := newMockGowaServer()
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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
				"reactions": []any{
					map[string]any{
						"emoji":     "\U0001F44D",
						"from_user": user.ID.String(),
					},
				},
			},
		}
		require.NoError(t, app.DB.Create(msg).Error)

		// Send empty emoji to remove reaction
		req := testutil.NewJSONRequest(t, map[string]any{
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
				Reactions []any `json:"reactions"`
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

		req := testutil.NewJSONRequest(t, map[string]any{
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

// TestApp_GetMessages_AssignedUserOnly verifies the agent-visibility rules for
// a user without contacts:read:
//   - an unassigned pending contact is not visible,
//   - a persistent assigned_user_id grants access.
func TestApp_GetMessages_AssignedUserOnly(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Agent role WITHOUT contacts:read — agents only see assigned chats.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "agent-no-read",
		[]string{"chat:read", "chat:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))
	// assigned_user_id intentionally nil — the pending chat is unassigned.

	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "Hi",
		Status:          models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	getMessagesStatus := func() int {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, agent.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		require.NoError(t, app.GetMessages(req))
		return testutil.GetResponseStatusCode(req)
	}

	// Unassigned pending chat → agent without contacts:read cannot load it.
	assert.Equal(t, fasthttp.StatusNotFound, getMessagesStatus(),
		"an unassigned pending contact must not be visible without contacts:read")

	// Persistent assignment → agent gains access.
	require.NoError(t, app.DB.Model(contact).Update("assigned_user_id", agent.ID).Error)
	assert.Equal(t, fasthttp.StatusOK, getMessagesStatus(),
		"a persistent assigned_user_id should grant the agent access")
}

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

	// Create 1 incoming message the user has already read (read state is
	// per user: it is the user's own read position that covers it).
	readMsg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "Read message",
		Status:          models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(readMsg).Error)
	markReq := newMarkReadRequest(t)
	testutil.SetAuthContext(markReq, org.ID, user.ID)
	testutil.SetPathParam(markReq, "id", contact.ID.String())
	require.NoError(t, app.MarkContactRead(markReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(markReq))

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
	req := testutil.NewJSONRequest(t, map[string]any{
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

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": otherOrgUser.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	// User from a different org should not be found
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

// --- ListContacts server-side sort tests ---

func TestApp_ListContacts_SortByWhatsAppAccount(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// Contacts with distinct accounts plus one without any account.
	testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Zeta"))
	testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Alpha"))
	testutil.CreateTestContactWith(t, app.DB, org.ID) // no account

	listSorted := func(sort, sortDir string) []handlers.ContactResponse {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "sort", sort)
		testutil.SetQueryParam(req, "sort_dir", sortDir)

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
		assert.Equal(t, int64(3), resp.Data.Total)
		return resp.Data.Contacts
	}

	accounts := func(contacts []handlers.ContactResponse) []string {
		got := make([]string, len(contacts))
		for i, c := range contacts {
			got[i] = c.WhatsAppAccount
		}
		return got
	}

	// Ascending: NULLS LAST puts the account-less contact at the end.
	assert.Equal(t, []string{"Alpha", "Zeta", ""}, accounts(listSorted("whatsapp_account", "asc")),
		"asc sort must order by account with NULLS LAST")

	// Descending: NULLS LAST still puts the account-less contact at the end.
	assert.Equal(t, []string{"Zeta", "Alpha", ""}, accounts(listSorted("whatsapp_account", "desc")),
		"desc sort must order by account reversed with NULLS LAST")
}

// The Closed tab used to filter only the first page of the general list
// (newest 50 by last message). Closed chats stop receiving messages, so on a
// busy org that page held none and the tab stayed empty. chat_status=closed
// pages closed conversations straight from the server — and, like the tab,
// leaves internal conversations to the Private tab.
func TestApp_ListContacts_ChatStatusClosedFilter(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	withMeta := func(meta models.JSONB) *models.Contact {
		c := testutil.CreateTestContactWith(t, app.DB, org.ID)
		require.NoError(t, app.DB.Model(c).Update("metadata", meta).Error)
		return c
	}
	closedA := withMeta(models.JSONB{"chat_status": "closed"})
	closedB := withMeta(models.JSONB{"chat_status": "closed", "closed_by": map[string]any{"name": "Ramzy"}})
	withMeta(models.JSONB{"chat_status": "open"})
	withMeta(models.JSONB{"chat_status": "pending"})
	testutil.CreateTestContactWith(t, app.DB, org.ID) // legacy row without chat_status
	withMeta(models.JSONB{"chat_status": "closed", models.MetaInternalChat: true})

	list := func(status string) (int, []handlers.ContactResponse, int64) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "chat_status", status)
		require.NoError(t, app.ListContacts(req))
		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		code := testutil.GetResponseStatusCode(req)
		if code == fasthttp.StatusOK {
			require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		}
		return code, resp.Data.Contacts, resp.Data.Total
	}

	code, contacts, total := list("closed")
	require.Equal(t, fasthttp.StatusOK, code)
	assert.Equal(t, int64(2), total, "only customer conversations that are closed")
	ids := []uuid.UUID{}
	for _, c := range contacts {
		assert.Equal(t, "closed", c.ChatStatus)
		ids = append(ids, c.ID)
	}
	assert.ElementsMatch(t, []uuid.UUID{closedA.ID, closedB.ID}, ids)

	code, _, _ = list("archived")
	assert.Equal(t, fasthttp.StatusBadRequest, code, "unknown chat_status values are rejected")
}

func TestApp_ListContacts_UnknownSortKeyFallsBackToDefault(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	for i := 0; i < 3; i++ {
		testutil.CreateTestContact(t, app.DB, org.ID)
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "sort", "bogus_column")
	testutil.SetQueryParam(req, "sort_dir", "asc")

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
	assert.Equal(t, int64(3), resp.Data.Total)
	assert.Len(t, resp.Data.Contacts, 3)
}

// TestApp_ListContacts_LastMessageAccount pins the sidebar badge fix: the
// badge must reflect the account of the contact's most recent real message,
// not the stale whats_app_account column (only some write paths stamp it).
// A trailing system event (no account) must not blank the field.
func TestApp_ListContacts_LastMessageAccount(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// Contact stamped with a stale owning account.
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Adv-1926"))

	mk := func(account string, created time.Time) *models.Message {
		return &models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: created},
			OrganizationID:  org.ID,
			WhatsAppAccount: account,
			ContactID:       contact.ID,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeText,
			Content:         "hi",
			Status:          models.MessageStatusDelivered,
		}
	}
	base := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	require.NoError(t, app.DB.Create(mk("Adv-1926", base)).Error)
	require.NoError(t, app.DB.Create(mk("Print-Aser-6208", base.Add(time.Minute))).Error)
	// Trailing close-notice system message carries no account — must be skipped.
	sys := mk("", base.Add(2*time.Minute))
	sys.Direction = models.DirectionOutgoing
	sys.Metadata = models.JSONB{"is_system_message": true}
	require.NoError(t, app.DB.Create(sys).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.ListContacts(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.Contacts, 1)
	assert.Equal(t, "Print-Aser-6208", resp.Data.Contacts[0].LastMessageAccount,
		"badge source must be the account of the most recent real message")
}

// TestApp_ListContacts_ScopedByAssignedAccounts pins the fix for scoped users
// seeing every account's conversations in /chat: a user assigned a subset of
// WhatsApp accounts must only see conversations under those accounts, while
// super admins and users with no assignments keep full org visibility.
func TestApp_ListContacts_ScopedByAssignedAccounts(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID) // contacts:read + write

	// Two accounts, each with one conversation.
	assignedAcct := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID)
	otherAcct := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID)
	assignedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(assignedAcct.Name))
	otherContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(otherAcct.Name))
	_ = assignedContact
	_ = otherContact

	listContacts := func(userID uuid.UUID) ([]string, int64) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, userID)
		require.NoError(t, app.ListContacts(req))
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
				Total    int64                      `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		names := make([]string, 0, len(resp.Data.Contacts))
		for _, c := range resp.Data.Contacts {
			names = append(names, c.WhatsAppAccount)
		}
		return names, resp.Data.Total
	}

	// 1) User WITH an assignment (the regression case): sees only their account.
	scopedUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.AssignAccountToUser(t, app.DB, scopedUser.ID, assignedAcct.ID)
	got, total := listContacts(scopedUser.ID)
	assert.Equal(t, int64(1), total, "scoped user must see only their assigned account's conversations")
	assert.ElementsMatch(t, []string{assignedAcct.Name}, got,
		"scoped user must not see other accounts' conversations")

	// 2) User with NO assignments: full org visibility (fallback).
	noAssignUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	got, total = listContacts(noAssignUser.ID)
	assert.Equal(t, int64(2), total, "user without assignments must fall back to full org visibility")
	assert.ElementsMatch(t, []string{assignedAcct.Name, otherAcct.Name}, got)

	// 3) Super admin: full org visibility regardless of assignments.
	superAdmin := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithRoleID(&adminRole.ID), testutil.WithSuperAdmin())
	testutil.AssignAccountToUser(t, app.DB, superAdmin.ID, assignedAcct.ID) // assigned, but bypassed
	got, total = listContacts(superAdmin.ID)
	assert.Equal(t, int64(2), total, "super admin must see all accounts despite having assignments")
	assert.ElementsMatch(t, []string{assignedAcct.Name, otherAcct.Name}, got)
}

// seedContactMessage stores one inbound message on account for contact.
func seedContactMessage(t *testing.T, app *handlers.App, orgID uuid.UUID, contact *models.Contact, account string) *models.Message {
	t.Helper()

	m := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: account,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "hi",
		Status:          models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(m).Error)
	return m
}

// listContactIDs calls ListContacts as userID with the given query params and
// returns the contact IDs on the page plus the total count.
func listContactIDs(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, params map[string]string) ([]uuid.UUID, int64) {
	t.Helper()

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	for k, v := range params {
		testutil.SetQueryParam(req, k, v)
	}
	require.NoError(t, app.ListContacts(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
			Total    int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	ids := make([]uuid.UUID, 0, len(resp.Data.Contacts))
	for _, c := range resp.Data.Contacts {
		ids = append(ids, c.ID)
	}
	return ids, resp.Data.Total
}

// TestApp_ListContacts_AccountsFilter pins the accounts filter. A contact
// matches when it is owned by one of the named accounts OR has a live message
// on one of them: whats_app_account goes stale for contacts messaged on several
// accounts, so the message accounts must count too.
func TestApp_ListContacts_AccountsFilter(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	ownedAlpha := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Alpha"))
	crossBeta := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Beta"))
	betaOnly := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Beta"))
	deletedOnly := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Gamma"))
	gammaNoMessages := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Gamma"))

	// crossBeta is owned by Beta but messaged on Alpha: it matches accounts=Alpha.
	seedContactMessage(t, app, org.ID, crossBeta, "Alpha")
	seedContactMessage(t, app, org.ID, betaOnly, "Beta")
	// A soft-deleted message is not conversation activity on that account.
	deleted := seedContactMessage(t, app, org.ID, deletedOnly, "Alpha")
	require.NoError(t, app.DB.Delete(deleted).Error)

	// Fields the combined-filter rows depend on. betaOnly carries the same
	// fields but does not match accounts=Alpha, so each combined row only holds
	// when the accounts filter really applies.
	now := time.Now().UTC()
	require.NoError(t, app.DB.Model(crossBeta).Updates(map[string]any{
		"tags":            models.JSONBArray{"vip"},
		"profile_name":    "Layla Crossover",
		"last_message_at": now,
	}).Error)
	require.NoError(t, app.DB.Model(betaOnly).Updates(map[string]any{
		"tags":            models.JSONBArray{"vip"},
		"profile_name":    "Layla Beta",
		"last_message_at": now,
	}).Error)

	all := []uuid.UUID{ownedAlpha.ID, crossBeta.ID, betaOnly.ID, deletedOnly.ID, gammaNoMessages.ID}
	tests := []struct {
		name   string
		params map[string]string
		want   []uuid.UUID
	}{
		{
			name:   "one account matches owned contacts and contacts messaged on it",
			params: map[string]string{"accounts": "Alpha"},
			want:   []uuid.UUID{ownedAlpha.ID, crossBeta.ID},
		},
		{
			name:   "several accounts return the union",
			params: map[string]string{"accounts": "Alpha,Beta"},
			want:   []uuid.UUID{ownedAlpha.ID, crossBeta.ID, betaOnly.ID},
		},
		{
			name:   "names are trimmed and deduplicated",
			params: map[string]string{"accounts": " Alpha , ,Alpha "},
			want:   []uuid.UUID{ownedAlpha.ID, crossBeta.ID},
		},
		{
			name:   "blank list applies no filter",
			params: map[string]string{"accounts": " , "},
			want:   all,
		},
		{
			name:   "unknown account matches nothing",
			params: map[string]string{"accounts": "Delta"},
			want:   []uuid.UUID{},
		},
		{
			name:   "combines with tags",
			params: map[string]string{"accounts": "Alpha", "tags": "vip"},
			want:   []uuid.UUID{crossBeta.ID},
		},
		{
			name:   "combines with search",
			params: map[string]string{"accounts": "Alpha", "search": "Layla"},
			want:   []uuid.UUID{crossBeta.ID},
		},
		{
			name:   "combines with has_messages",
			params: map[string]string{"accounts": "Alpha", "has_messages": "true"},
			want:   []uuid.UUID{crossBeta.ID},
		},
		{
			name:   "combines with chat_status=pending",
			params: map[string]string{"accounts": "Alpha", "chat_status": "pending"},
			want:   []uuid.UUID{crossBeta.ID},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, total := listContactIDs(t, app, org.ID, user.ID, tc.params)
			assert.Equal(t, int64(len(tc.want)), total)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

// The accounts filter only narrows scopeAssignedContact. A scoped user sees
// contacts on their assigned account or ones they are involved in, and
// accounts=<other> must never reveal a contact outside that set, even when the
// contact's message history matches the filter.
func TestApp_ListContacts_AccountsFilterRespectsScoping(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID) // contacts:read + write

	assignedAcct := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("Beta"))

	ownedAlpha := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Alpha"))
	crossBeta := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Beta"))
	gammaOwned := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Gamma"))
	// Owned by Gamma with no Alpha message: must never match accounts=Alpha.
	testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("Gamma"))
	seedContactMessage(t, app, org.ID, crossBeta, "Alpha")
	seedContactMessage(t, app, org.ID, gammaOwned, "Alpha")

	scopedUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.AssignAccountToUser(t, app.DB, scopedUser.ID, assignedAcct.ID)

	got, total := listContactIDs(t, app, org.ID, scopedUser.ID, map[string]string{"accounts": "Alpha"})
	assert.Equal(t, int64(1), total, "scoped user must only see contacts inside their account grant")
	assert.ElementsMatch(t, []uuid.UUID{crossBeta.ID}, got,
		"accounts=Alpha must not reveal Alpha-owned or Gamma-owned contacts outside the grant")

	// Control: a user without assignments gets every match for the same filter.
	fullUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	got, total = listContactIDs(t, app, org.ID, fullUser.ID, map[string]string{"accounts": "Alpha"})
	assert.Equal(t, int64(3), total)
	assert.ElementsMatch(t, []uuid.UUID{ownedAlpha.ID, crossBeta.ID, gammaOwned.ID}, got)
}

// The accounts list keeps at most 50 distinct names, so the 51st name in a
// crafted query string is ignored and its contacts never match.
func TestApp_ListContacts_AccountsFilterCapsNames(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	within := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("acc-01"))
	beyondCap := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount("acc-51"))

	names := make([]string, 0, 51)
	for i := 1; i <= 51; i++ {
		names = append(names, fmt.Sprintf("acc-%02d", i))
	}
	got, total := listContactIDs(t, app, org.ID, user.ID, map[string]string{"accounts": strings.Join(names, ",")})
	assert.Equal(t, int64(1), total)
	assert.ElementsMatch(t, []uuid.UUID{within.ID}, got)
	assert.NotContains(t, got, beyondCap.ID)
}

// chat_status=pending is the unclaimed queue (unassigned, has messages, not
// closed); chat_status=internal is the Private tab. Each excludes conversations
// that belong to the other tabs.
func TestApp_ListContacts_ChatStatusPendingAndInternal(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// The org's own connected number: a contact whose phone matches it is
	// internal. gowa_jid is globally unique, so draw a number no other test uses.
	jidSeed := uuid.New()
	ownPhone := fmt.Sprintf("966%09d", binary.BigEndian.Uint32(jidSeed[:4])%1000000000)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(account).Update("gowa_jid", ownPhone+"@s.whatsapp.net").Error)

	recent := time.Now().UTC()
	withState := func(c *models.Contact, updates map[string]any) *models.Contact {
		require.NoError(t, app.DB.Model(c).Updates(updates).Error)
		return c
	}
	newContact := func(opts ...testutil.ContactOption) *models.Contact {
		return testutil.CreateTestContactWith(t, app.DB, org.ID, opts...)
	}

	pendingOK := withState(newContact(), map[string]any{"last_message_at": recent})
	withState(newContact(), map[string]any{"last_message_at": recent, "assigned_user_id": user.ID})
	withState(newContact(), map[string]any{
		"last_message_at": recent,
		"metadata":        models.JSONB{"chat_status": "closed"},
	})
	newContact() // no messages yet
	manualInternal := withState(newContact(), map[string]any{
		"last_message_at": recent,
		"metadata":        models.JSONB{models.MetaInternalChat: true},
	})
	accountNumber := withState(newContact(testutil.WithPhoneNumber(ownPhone)),
		map[string]any{"last_message_at": recent})

	tests := []struct {
		name   string
		status string
		want   []uuid.UUID
	}{
		{
			name:   "pending lists unassigned open conversations with messages",
			status: "pending",
			want:   []uuid.UUID{pendingOK.ID},
		},
		{
			name:   "internal lists manually marked and org-number conversations only",
			status: "internal",
			want:   []uuid.UUID{manualInternal.ID, accountNumber.ID},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, total := listContactIDs(t, app, org.ID, user.ID, map[string]string{"chat_status": tc.status})
			assert.Equal(t, int64(len(tc.want)), total)
			assert.ElementsMatch(t, tc.want, got)
		})
	}

	for _, status := range []string{"bogus", "open"} {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "chat_status", status)
		require.NoError(t, app.ListContacts(req))
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "chat_status=%s", status)
		assert.Contains(t, string(testutil.GetResponseBody(req)), "chat_status must be 'closed', 'pending' or 'internal'")
	}
}
