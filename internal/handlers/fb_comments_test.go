package handlers_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type facebookCommentRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn facebookCommentRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestApp_ReplyFacebookComment_ByExternalID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.Path, "/external-comment-id/comments")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"graph-reply-id"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, account.ID, "external-comment-id")

	req := testutil.NewJSONRequest(t, map[string]any{
		"reply_text":           "Thanks for reaching out",
		"send_comment_reply":   true,
		"send_private_message": false,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", comment.ExternalID)

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "graph-reply-id", saved.GraphCommentReplyID)
	assert.Equal(t, models.FBCommentStatusReplied, reloadFacebookCommentStatus(t, app, comment.ID))
}

func TestApp_ReplyFacebookComment_IgnoresStaleScopedDB(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.Path, "/stale-scope-comment-id/comments")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"graph-reply-id"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, account.ID, "stale-scope-comment-id")

	req := testutil.NewJSONRequest(t, map[string]any{
		"reply_text":           "Thanks for reaching out",
		"send_comment_reply":   true,
		"send_private_message": false,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", comment.ID.String())
	tenant.SetScopedDB(req, tenant.ScopedDB(app.DB, otherOrg.ID))

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "graph-reply-id", saved.GraphCommentReplyID)
}

func TestApp_ReplyFacebookComment_PrivateReplyUsesPageMessagesEndpoint(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/v20.0/page-id/messages", req.URL.Path)
		assert.Equal(t, "page-token", req.URL.Query().Get("access_token"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		recipient, ok := body["recipient"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "122127202731122483_942592348778274", recipient["comment_id"])
		message, ok := body["message"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "How can I help?", message["text"])

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"message_id":"graph-private-reply-id"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, account.ID, "122127202731122483_942592348778274")

	req := testutil.NewJSONRequest(t, map[string]any{
		"private_message_text": "How can I help?",
		"send_comment_reply":   false,
		"send_private_message": true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", comment.ID.String())

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "graph-private-reply-id", saved.GraphPrivateReplyID)
	assert.Equal(t, "sent", saved.Status)
}

func TestApp_ReplyFacebookComment_PrivateReplyFallsBackToDirectMessenger(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	var paths []string
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		paths = append(paths, req.URL.Path)
		if req.URL.Path != "/v20.0/me/messages" {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"error": {
						"message": "Unsupported post request.",
						"type": "GraphMethodException",
						"code": 100,
						"error_subcode": 33
					}
				}`)),
			}, nil
		}

		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		assert.Equal(t, "RESPONSE", body["messaging_type"])
		recipient, ok := body["recipient"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "sender-id", recipient["id"])

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"recipient_id":"sender-id","message_id":"direct-message-id"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, account.ID, "122127202731122483_942592348778274")

	req := testutil.NewJSONRequest(t, map[string]any{
		"private_message_text": "How can I help?",
		"send_comment_reply":   false,
		"send_private_message": true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", comment.ID.String())

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, []string{"/v20.0/page-id/messages", "/v20.0/me/messages"}, paths)
	assert.Equal(t, "direct-message-id", saved.GraphPrivateReplyID)
	assert.Equal(t, "sent", saved.Status)
}

func TestApp_UpdateFacebookCommentStatus_ByExternalID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, account.ID, "status-external-comment-id")

	req := testutil.NewJSONRequest(t, map[string]any{"status": "closed"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", comment.ExternalID)

	err := app.UpdateFacebookCommentStatus(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, models.FBCommentStatusClosed, reloadFacebookCommentStatus(t, app, comment.ID))
}

func TestApp_ReceiveFacebookCommentsWebhook_PopulatesFromPayload(t *testing.T) {
	t.Parallel()

	pageID := fmt.Sprintf("page-from-%s", t.Name())
	extID := "wh-comment-1"

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	account := createFacebookCommentAccountWithPageID(t, app, org.ID, user.ID, pageID)

	payload := map[string]any{
		"object": "page",
		"entry": []map[string]any{
			{
				"id": pageID,
				"time": 1700000000,
				"changes": []map[string]any{
					{
						"field": "feed",
						"value": map[string]any{
							"item":        "comment",
							"verb":        "add",
							"comment_id":  extID,
							"post_id":     "wh-post-1",
							"created_time": 1700000000,
							"message":     "Where is my order?",
							"from": map[string]any{
								"id":   "PSID-12345",
								"name": "Waqas Ahmad",
							},
						},
					},
				},
			},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	err := app.ReceiveFacebookCommentsWebhook(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", extID).First(&saved).Error)
	assert.Equal(t, "PSID-12345", saved.FromID)
	assert.Equal(t, "Waqas Ahmad", saved.FromName)
	assert.Equal(t, "Where is my order?", saved.Message)
	assert.Equal(t, "wh-post-1", saved.PostID)
	assert.Equal(t, org.ID, saved.OrganizationID)
	assert.Equal(t, account.ID, saved.AccountID)
}

func TestApp_ReceiveFacebookCommentsWebhook_FallsBackToSenderFields(t *testing.T) {
	t.Parallel()

	pageID := fmt.Sprintf("page-sender-%s", t.Name())
	extID := "wh-comment-legacy"

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	createFacebookCommentAccountWithPageID(t, app, org.ID, user.ID, pageID)

	payload := map[string]any{
		"object": "page",
		"entry": []map[string]any{
			{
				"id": pageID,
				"time": 1700000000,
				"changes": []map[string]any{
					{
						"field": "feed",
						"value": map[string]any{
							"item":        "comment",
							"verb":        "add",
							"comment_id":  extID,
							"post_id":     "wh-post-legacy",
							"created_time": 1700000000,
							"message":     "Hi",
							"sender_id":   "PSID-LEGACY",
							"sender_name": "Legacy Sender",
						},
					},
				},
			},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	err := app.ReceiveFacebookCommentsWebhook(req)
	require.NoError(t, err)

	var saved models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", extID).First(&saved).Error)
	assert.Equal(t, "PSID-LEGACY", saved.FromID)
	assert.Equal(t, "Legacy Sender", saved.FromName)
}

func TestApp_ReceiveFacebookCommentsWebhook_AdminReplyTaggedAndNotAutoReplied(t *testing.T) {
	t.Parallel()

	pageID := fmt.Sprintf("page-admin-%s", t.Name())
	extID := "wh-comment-admin"

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	var graphCalls int
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		graphCalls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"should-not-be-called"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	_ = createFacebookCommentAccountWithPageID(t, app, org.ID, user.ID, pageID)

	settings := &models.FacebookCommentSettings{
		BaseModel:               models.BaseModel{ID: uuid.New()},
		OrganizationID:          org.ID,
		Enabled:                 true,
		AutoReplyEnabled:        true,
		AutoCommentReplyEnabled: true,
		AutoPrivateReplyEnabled: true,
		AutoCommentReplyText:    "auto public",
		AutoPrivateMessageText:  "auto private",
		OnlyAutoReplyUnanswered: true,
		IgnorePageAdminComments: true,
	}
	require.NoError(t, app.DB.Create(settings).Error)

	payload := map[string]any{
		"object": "page",
		"entry": []map[string]any{
			{
				"id": pageID,
				"changes": []map[string]any{
					{
						"field": "feed",
						"value": map[string]any{
							"item":        "comment",
							"verb":        "add",
							"comment_id":  extID,
							"post_id":     "wh-post-admin",
							"created_time": 1700000000,
							"message":     "Thanks for your question!",
							"from": map[string]any{
								"id":   pageID,
								"name": "Page Admin",
							},
						},
					},
				},
			},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	whBody := req.RequestCtx.PostBody()
	mac := hmac.New(sha256.New, []byte("app-secret"))
	_, _ = mac.Write(whBody)
	req.RequestCtx.Request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	err := app.ReceiveFacebookCommentsWebhook(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", extID).First(&saved).Error)
	assert.True(t, saved.IsAdminReply, "page-admin comment must be tagged IsAdminReply")
	assert.Equal(t, pageID, saved.FromID)

	var replies []models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", saved.ID).Find(&replies).Error)
	assert.Empty(t, replies, "no auto reply should be sent for an admin-authored comment")
	assert.Zero(t, graphCalls, "no outbound Graph API call should be made for an admin-authored comment")
}

func TestApp_ReceiveFacebookCommentsWebhook_NonAdminStillAutoReplies(t *testing.T) {
	t.Parallel()

	pageID := fmt.Sprintf("page-user-%s", t.Name())
	extID := "wh-comment-user"

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	var graphPaths []string
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		graphPaths = append(graphPaths, req.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"graph-reply-user"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	_ = createFacebookCommentAccountWithPageID(t, app, org.ID, user.ID, pageID)

	settings := &models.FacebookCommentSettings{
		BaseModel:               models.BaseModel{ID: uuid.New()},
		OrganizationID:          org.ID,
		Enabled:                 true,
		AutoReplyEnabled:        true,
		AutoCommentReplyEnabled: true,
		AutoPrivateReplyEnabled: false,
		AutoCommentReplyText:    "auto public",
		OnlyAutoReplyUnanswered: true,
		IgnorePageAdminComments: true,
	}
	require.NoError(t, app.DB.Create(settings).Error)

	payload := map[string]any{
		"object": "page",
		"entry": []map[string]any{
			{
				"id": pageID,
				"changes": []map[string]any{
					{
						"field": "feed",
						"value": map[string]any{
							"item":        "comment",
							"verb":        "add",
							"comment_id":  extID,
							"post_id":     "wh-post-user",
							"created_time": 1700000000,
							"message":     "I need help",
							"from": map[string]any{
								"id":   "PSID-USER-1",
								"name": "User One",
							},
						},
					},
				},
			},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	whBody := req.RequestCtx.PostBody()
	mac := hmac.New(sha256.New, []byte("app-secret"))
	_, _ = mac.Write(whBody)
	req.RequestCtx.Request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	err := app.ReceiveFacebookCommentsWebhook(req)
	require.NoError(t, err)

	var saved models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", extID).First(&saved).Error)
	assert.False(t, saved.IsAdminReply, "non-admin comment must not be tagged")

	var replies []models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", saved.ID).Find(&replies).Error)
	require.NotEmpty(t, replies, "non-admin comment must still trigger auto reply")
	assert.True(t, replies[0].IsAuto)
	assert.Contains(t, graphPaths, fmt.Sprintf("/v20.0/%s/comments", extID))
}

func TestApp_ReplyFacebookComment_PrivateReplyFallsBackToDirectMessenger_FromWebhook(t *testing.T) {
	t.Parallel()

	pageID := fmt.Sprintf("page-fb-%s", t.Name())
	extID := fmt.Sprintf("wh-comment-fb-%s", t.Name())

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	var paths []string
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		paths = append(paths, req.URL.Path)
		if req.URL.Path != "/v20.0/me/messages" {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"error": {
						"message": "Unsupported post request.",
						"type": "GraphMethodException",
						"code": 100,
						"error_subcode": 33
					}
				}`)),
			}, nil
		}

		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		recipient, ok := body["recipient"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "PSID-WEBHOOK-1", recipient["id"])

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"recipient_id":"PSID-WEBHOOK-1","message_id":"direct-message-id"}`)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	createFacebookCommentAccountWithPageID(t, app, org.ID, user.ID, pageID)

	payload := map[string]any{
		"object": "page",
		"entry": []map[string]any{
			{
				"id": pageID,
				"time": 1700000000,
				"changes": []map[string]any{
					{
						"field": "feed",
						"value": map[string]any{
							"item":        "comment",
							"verb":        "add",
							"comment_id":  extID,
							"post_id":     "wh-post-fb",
							"created_time": 1700000000,
							"message":     "Where is my order?",
							"from": map[string]any{
								"id":   "PSID-WEBHOOK-1",
								"name": "Waqas Ahmad",
							},
						},
					},
				},
			},
		},
	}

	whReq := testutil.NewJSONRequest(t, payload)
	whBody := whReq.RequestCtx.PostBody()
	mac := hmac.New(sha256.New, []byte("app-secret"))
	_, _ = mac.Write(whBody)
	whReq.RequestCtx.Request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	require.NoError(t, app.ReceiveFacebookCommentsWebhook(whReq))

	var comment models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", extID).First(&comment).Error)
	require.Equal(t, "PSID-WEBHOOK-1", comment.FromID, "webhook must populate FromID from value.from.id so private reply fallback can fire")

	replyReq := testutil.NewJSONRequest(t, map[string]any{
		"private_message_text": "How can I help?",
		"send_comment_reply":   false,
		"send_private_message": true,
	})
	testutil.SetAuthContext(replyReq, org.ID, user.ID)
	testutil.SetPathParam(replyReq, "id", comment.ExternalID)

	err := app.ReplyFacebookComment(replyReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(replyReq))

	assert.Equal(t, []string{fmt.Sprintf("/v20.0/%s/messages", pageID), "/v20.0/me/messages"}, paths)

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "direct-message-id", saved.GraphPrivateReplyID)
	assert.Equal(t, "sent", saved.Status)
}

func TestIsFacebookUserCantDMError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matches real 10903 OAuthException",
			err:  errors.New("Facebook Graph API returned status 400: This user can't reply to this activity: code=10903: subcode=1893049: type=OAuthException"),
			want: true,
		},
		{
			name: "matches without apostrophe (cant)",
			err:  errors.New("Facebook Graph API returned status 400: This user cant reply to this activity: code=10903"),
			want: true,
		},
		{
			name: "matches generic 400 with just code=10903",
			err:  errors.New("send failed: code=10903"),
			want: true,
		},
		{
			name: "ignores unrelated 400 error",
			err:  errors.New("Facebook Graph API returned status 400: Invalid OAuth access token: code=190"),
			want: false,
		},
		{
			name: "ignores nil",
			err:  nil,
			want: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, handlers.IsFacebookUserCantDMError(tc.err))
		})
	}
}

func TestApp_ReplyFacebookComment_DMOnly_10903MarksAsSkipped(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"message":"This user can't reply to this activity","code":10903,"error_subcode":1893049,"type":"OAuthException"}}`,
			)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	fbAccount := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, fbAccount.ID, "skipped-dm-comment-external-id")

	req := testutil.NewJSONRequest(t, map[string]any{
		"private_message_text": "How can I help?",
		"send_comment_reply":   false,
		"send_private_message": true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "skipped-dm-comment-external-id")

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "skipped", saved.Status, "DM-only 10903 must mark reply as skipped, not partial or failed")
	assert.True(t, readJSONBool(t, saved.Metadata, "dm_skipped"), "metadata.dm_skipped must be true")
	assert.Equal(t, "user_cant_be_dmed", readJSONString(t, saved.Metadata, "dm_skip_reason"))
	assert.Empty(t, saved.GraphPrivateReplyID, "no message ID should be persisted for skipped DM")
	assert.Equal(t, models.FBCommentStatusOpen, reloadFacebookCommentStatus(t, app, comment.ID), "comment must stay open when only DM was attempted and it was skipped")
}

func TestApp_ReplyFacebookComment_CommentAndDM_10903KeepsReplied(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	app.Config.FacebookOAuth.AppID = "app-id"
	app.Config.FacebookOAuth.AppSecret = "app-secret"
	app.Config.FacebookOAuth.BaseURL = "https://graph.test"
	app.Config.FacebookOAuth.APIVersion = "v20.0"
	app.HTTPClient = &http.Client{Transport: facebookCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		if strings.Contains(req.URL.Path, "/comments") {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"id":"graph-reply-id"}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"message":"This user can't reply to this activity","code":10903,"error_subcode":1893049,"type":"OAuthException"}}`,
			)),
		}, nil
	})}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	fbAccount := createFacebookCommentAccount(t, app, org.ID, user.ID)
	comment := createFacebookComment(t, app, org.ID, fbAccount.ID, "mixed-10903-comment-external-id")

	req := testutil.NewJSONRequest(t, map[string]any{
		"reply_text":           "Thanks for reaching out",
		"private_message_text": "How can I help?",
		"send_comment_reply":   true,
		"send_private_message": true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "mixed-10903-comment-external-id")

	err := app.ReplyFacebookComment(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var saved models.FacebookCommentReply
	require.NoError(t, app.DB.Where("comment_id = ?", comment.ID).First(&saved).Error)
	assert.Equal(t, "sent", saved.Status, "comment leg succeeded so overall status is sent; DM skip is recorded in metadata")
	assert.Equal(t, "graph-reply-id", saved.GraphCommentReplyID)
	assert.True(t, readJSONBool(t, saved.Metadata, "dm_skipped"), "metadata.dm_skipped must be true even when comment leg succeeded")
	assert.Equal(t, models.FBCommentStatusReplied, reloadFacebookCommentStatus(t, app, comment.ID), "comment must flip to replied when public comment reply succeeded")
}

func TestApp_UpdateFacebookCommentStatus_BroadcastsWSMessage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, withWSHub())
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	fbAccount := createFacebookCommentAccount(t, app, org.ID, user.ID)
	_ = createFacebookComment(t, app, org.ID, fbAccount.ID, "status-broadcast-comment-id")

	client := websocket.NewClient(app.WSHub, nil, user.ID, org.ID)
	app.WSHub.Register(client)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if app.WSHub.GetClientCount() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.Equal(t, 1, app.WSHub.GetClientCount(), "client must be registered before broadcast")

	req := testutil.NewJSONRequest(t, map[string]any{"status": "closed"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "status-broadcast-comment-id")

	err := app.UpdateFacebookCommentStatus(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "closed", reloadFacebookCommentStatus(t, app, _retrieveCommentIDByExternal(t, app, "status-broadcast-comment-id")), "comment status must be persisted as closed")
}

func _retrieveCommentIDByExternal(t *testing.T, app *handlers.App, externalID string) uuid.UUID {
	t.Helper()
	var c models.FacebookComment
	require.NoError(t, app.DB.Where("external_id = ?", externalID).First(&c).Error)
	return c.ID
}

func readJSONBool(t *testing.T, m models.JSONB, key string) bool {
	t.Helper()
	v, ok := m[key]
	if !ok {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true"
	default:
		return false
	}
}

func readJSONString(t *testing.T, m models.JSONB, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func createFacebookCommentAccount(t *testing.T, app *handlers.App, orgID, userID uuid.UUID) *models.FacebookAccount {
	t.Helper()
	return createFacebookCommentAccountWithPageID(t, app, orgID, userID, "page-id")
}

func createFacebookCommentAccountWithPageID(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, pageID string) *models.FacebookAccount {
	t.Helper()

	pageTokens, err := appcrypto.Encrypt(fmt.Sprintf(`{"%s":"page-token"}`, pageID), testutil.TestEncryptionKey)
	require.NoError(t, err)
	now := time.Now()
	account := &models.FacebookAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		UserID:         userID,
		Platform:       "facebook",
		Name:           "Facebook Test Account",
		AccountUID:     "account-uid",
		Status:         models.FBAccountStatusActive,
		Method:         models.FBAccountMethodOAuth,
		PageTokens:     pageTokens,
		ConnectedAt:    &now,
	}
	require.NoError(t, app.DB.Create(account).Error)
	return account
}

func createFacebookComment(t *testing.T, app *handlers.App, orgID, accountID uuid.UUID, externalID string) *models.FacebookComment {
	t.Helper()

	comment := &models.FacebookComment{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		AccountID:      accountID,
		PageID:         "page-id",
		PageName:       "Page",
		PostID:         "post-id",
		ExternalID:     externalID,
		FromID:         "sender-id",
		FromName:       "Sender",
		Message:        "Need help",
		Status:         models.FBCommentStatusOpen,
		Direction:      models.FBCommentDirectionIncoming,
		CommentedAt:    time.Now(),
		Metadata:       models.JSONB{},
	}
	require.NoError(t, app.DB.Create(comment).Error)
	return comment
}

func reloadFacebookCommentStatus(t *testing.T, app *handlers.App, commentID uuid.UUID) models.FacebookCommentStatus {
	t.Helper()

	var comment models.FacebookComment
	require.NoError(t, app.DB.First(&comment, "id = ?", commentID).Error)
	return comment.Status
}
