package handlers_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	tinyPNGBase64  = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+nmZ0AAAAASUVORK5CYII="
	tinyJPEGBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkGBxAQEBUQEBAVFhUVFRUVFRUVFRUVFRUVFRUWFhUVFRUYHSggGBolGxUVITEhJSkrLi4uFx8zODMsNygtLisBCgoKDg0OGhAQGyslICYtLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLf/AABEIAAEAAQMBIgACEQEDEQH/xAAXAAEBAQEAAAAAAAAAAAAAAAAAAQID/8QAFhEBAQEAAAAAAAAAAAAAAAAAABEh/9oADAMBAAIQAxAAAAH/AP/EABQQAQAAAAAAAAAAAAAAAAAAACD/2gAIAQEAAQUCcf/EABQRAQAAAAAAAAAAAAAAAAAAACD/2gAIAQMBAT8Bp//EABQRAQAAAAAAAAAAAAAAAAAAACD/2gAIAQIBAT8Bp//Z"
	tinyWEBPBase64 = "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEAAUAmJaQAA3AA/v89WAAAAA=="
)

func newChatBackgroundTestApp(t *testing.T) *handlers.App {
	t.Helper()

	app := newTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	return app
}

func decodeBase64Fixture(t *testing.T, raw string) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString(raw)
	require.NoError(t, err)
	return data
}

func newChatBackgroundMultipartRequest(
	t *testing.T,
	fileName string,
	fileMIME string,
	fileData []byte,
) *fastglue.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	if fileMIME != "" {
		partHeader.Set("Content-Type", fileMIME)
	}

	part, err := writer.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	req.RequestCtx.Request.Header.SetContentType(writer.FormDataContentType())
	req.RequestCtx.Request.SetBody(body.Bytes())

	return req
}

func chatBackgroundAssetPath(baseDir string, userID string, asset handlers.UserChatBackground) string {
	return filepath.Join(
		baseDir,
		"chat-backgrounds",
		userID,
		asset.CustomAssetID+filepath.Ext(asset.CustomFilename),
	)
}

func TestApp_UpdateCurrentUserSettings_PartialUpdatePreservesExistingKeys(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("settings-partial")),
	)

	user.Settings = models.JSONB{
		"email_notifications": true,
		"campaign_updates":    true,
		"notification_sound":  "notification2",
		"chat_background": map[string]any{
			"kind":      "preset",
			"preset_id": "aurora-veil",
		},
		"send_restrictions": map[string]any{
			"enabled": true,
		},
	}
	require.NoError(t, app.DB.Save(user).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"new_message_alerts": false,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateCurrentUserSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Settings map[string]any `json:"settings"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Equal(t, true, resp.Settings["email_notifications"])
	assert.Equal(t, false, resp.Settings["new_message_alerts"])
	assert.Equal(t, true, resp.Settings["campaign_updates"])
	assert.Equal(t, "notification2", resp.Settings["notification_sound"])
	assert.Equal(t, map[string]any{
		"kind":      "preset",
		"preset_id": "aurora-veil",
	}, resp.Settings["chat_background"])
	require.Contains(t, resp.Settings, "send_restrictions")
}

func TestApp_UpdateCurrentUserSettings_ChatBackgroundPresetValidation(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid preset id", func(t *testing.T) {
		app := newChatBackgroundTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		user := testutil.CreateTestUser(t, app.DB, org.ID,
			testutil.WithEmail(testutil.UniqueEmail("settings-preset-valid")),
		)

		req := testutil.NewJSONRequest(t, map[string]any{
			"chat_background": map[string]any{
				"kind":      "preset",
				"preset_id": "aurora-veil",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.UpdateCurrentUserSettings(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Settings map[string]any `json:"settings"`
		}
		testutil.ParseEnvelopeResponse(t, req, &resp)

		chatBackground, ok := resp.Settings["chat_background"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "preset", chatBackground["kind"])
		assert.Equal(t, "aurora-veil", chatBackground["preset_id"])
	})

	t.Run("rejects invalid preset id", func(t *testing.T) {
		app := newChatBackgroundTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		user := testutil.CreateTestUser(t, app.DB, org.ID,
			testutil.WithEmail(testutil.UniqueEmail("settings-preset-invalid")),
		)

		req := testutil.NewJSONRequest(t, map[string]any{
			"chat_background": map[string]any{
				"kind":      "preset",
				"preset_id": "not-real",
			},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.UpdateCurrentUserSettings(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "invalid chat background preset")
	})
}

func TestApp_UploadCurrentUserChatBackground_ValidFormatsAndValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		filename string
		mimeType string
		data     []byte
	}{
		{name: "png", filename: "background.png", mimeType: "image/png", data: decodeBase64Fixture(t, tinyPNGBase64)},
		{name: "jpeg", filename: "background.jpg", mimeType: "image/jpeg", data: decodeBase64Fixture(t, tinyJPEGBase64)},
		{name: "webp", filename: "background.webp", mimeType: "image/webp", data: decodeBase64Fixture(t, tinyWEBPBase64)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app := newChatBackgroundTestApp(t)
			org := testutil.CreateTestOrganization(t, app.DB)
			user := testutil.CreateTestUser(t, app.DB, org.ID,
				testutil.WithEmail(testutil.UniqueEmail("chat-bg-"+tc.name)),
			)

			req := newChatBackgroundMultipartRequest(t, tc.filename, tc.mimeType, tc.data)
			testutil.SetAuthContext(req, org.ID, user.ID)

			err := app.UploadCurrentUserChatBackground(req)
			require.NoError(t, err)
			assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

			var resp struct {
				ChatBackground handlers.UserChatBackground `json:"chat_background"`
			}
			testutil.ParseEnvelopeResponse(t, req, &resp)

			assert.Equal(t, "custom", resp.ChatBackground.Kind)
			assert.Equal(t, tc.mimeType, resp.ChatBackground.CustomMimeType)

			assetPath := chatBackgroundAssetPath(app.Config.Storage.LocalPath, user.ID.String(), resp.ChatBackground)
			if _, err := os.Stat(assetPath); err != nil {
				t.Fatalf("expected asset to exist at %s: %v", assetPath, err)
			}
		})
	}

	t.Run("rejects oversize upload", func(t *testing.T) {
		app := newChatBackgroundTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		user := testutil.CreateTestUser(t, app.DB, org.ID,
			testutil.WithEmail(testutil.UniqueEmail("chat-bg-too-large")),
		)

		oversize := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte("x"), (5*1024*1024)+1)...)
		req := newChatBackgroundMultipartRequest(t, "too-large.png", "image/png", oversize)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.UploadCurrentUserChatBackground(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Maximum size is 5MB")
	})

	t.Run("rejects unsupported mime type", func(t *testing.T) {
		app := newChatBackgroundTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		user := testutil.CreateTestUser(t, app.DB, org.ID,
			testutil.WithEmail(testutil.UniqueEmail("chat-bg-bad-type")),
		)

		req := newChatBackgroundMultipartRequest(t, "notes.txt", "text/plain", []byte("plain text"))
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.UploadCurrentUserChatBackground(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Use JPG, PNG, or WebP")
	})
}

func TestApp_UploadCurrentUserChatBackground_ReplacesPreviousAsset(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-replace")),
	)

	firstReq := newChatBackgroundMultipartRequest(t, "first.png", "image/png", decodeBase64Fixture(t, tinyPNGBase64))
	testutil.SetAuthContext(firstReq, org.ID, user.ID)
	require.NoError(t, app.UploadCurrentUserChatBackground(firstReq))

	var firstResp struct {
		ChatBackground handlers.UserChatBackground `json:"chat_background"`
	}
	testutil.ParseEnvelopeResponse(t, firstReq, &firstResp)
	firstPath := chatBackgroundAssetPath(app.Config.Storage.LocalPath, user.ID.String(), firstResp.ChatBackground)

	secondReq := newChatBackgroundMultipartRequest(t, "second.webp", "image/webp", decodeBase64Fixture(t, tinyWEBPBase64))
	testutil.SetAuthContext(secondReq, org.ID, user.ID)
	require.NoError(t, app.UploadCurrentUserChatBackground(secondReq))

	var secondResp struct {
		ChatBackground handlers.UserChatBackground `json:"chat_background"`
	}
	testutil.ParseEnvelopeResponse(t, secondReq, &secondResp)

	_, err := os.Stat(firstPath)
	assert.ErrorIs(t, err, os.ErrNotExist)

	secondPath := chatBackgroundAssetPath(app.Config.Storage.LocalPath, user.ID.String(), secondResp.ChatBackground)
	_, err = os.Stat(secondPath)
	require.NoError(t, err)
}

func TestApp_UpdateCurrentUserSettings_SwitchingToPresetRemovesOldCustomAsset(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-preset-switch")),
	)

	uploadReq := newChatBackgroundMultipartRequest(t, "before.png", "image/png", decodeBase64Fixture(t, tinyPNGBase64))
	testutil.SetAuthContext(uploadReq, org.ID, user.ID)
	require.NoError(t, app.UploadCurrentUserChatBackground(uploadReq))

	var uploadResp struct {
		ChatBackground handlers.UserChatBackground `json:"chat_background"`
	}
	testutil.ParseEnvelopeResponse(t, uploadReq, &uploadResp)
	oldPath := chatBackgroundAssetPath(app.Config.Storage.LocalPath, user.ID.String(), uploadResp.ChatBackground)

	updateReq := testutil.NewJSONRequest(t, map[string]any{
		"chat_background": map[string]any{
			"kind":      "preset",
			"preset_id": "linen-grid",
		},
	})
	testutil.SetAuthContext(updateReq, org.ID, user.ID)

	require.NoError(t, app.UpdateCurrentUserSettings(updateReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(updateReq))

	_, err := os.Stat(oldPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestApp_UpdateCurrentUserSettings_ClearingChatBackgroundRemovesMetadataAndCustomAsset(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-clear")),
	)

	uploadReq := newChatBackgroundMultipartRequest(t, "before.png", "image/png", decodeBase64Fixture(t, tinyPNGBase64))
	testutil.SetAuthContext(uploadReq, org.ID, user.ID)
	require.NoError(t, app.UploadCurrentUserChatBackground(uploadReq))

	var uploadResp struct {
		ChatBackground handlers.UserChatBackground `json:"chat_background"`
	}
	testutil.ParseEnvelopeResponse(t, uploadReq, &uploadResp)
	oldPath := chatBackgroundAssetPath(app.Config.Storage.LocalPath, user.ID.String(), uploadResp.ChatBackground)

	updateReq := testutil.NewJSONRequest(t, map[string]any{
		"chat_background": nil,
	})
	testutil.SetAuthContext(updateReq, org.ID, user.ID)

	require.NoError(t, app.UpdateCurrentUserSettings(updateReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(updateReq))

	var resp struct {
		Settings map[string]any `json:"settings"`
	}
	testutil.ParseEnvelopeResponse(t, updateReq, &resp)
	assert.NotContains(t, resp.Settings, "chat_background")

	_, err := os.Stat(oldPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestApp_GetCurrentUserChatBackground_AuthAndIsolation(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	owner := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-owner")),
	)
	otherUser := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-other")),
	)

	uploadReq := newChatBackgroundMultipartRequest(t, "owner.png", "image/png", decodeBase64Fixture(t, tinyPNGBase64))
	testutil.SetAuthContext(uploadReq, org.ID, owner.ID)
	require.NoError(t, app.UploadCurrentUserChatBackground(uploadReq))

	var uploadResp struct {
		ChatBackground handlers.UserChatBackground `json:"chat_background"`
	}
	testutil.ParseEnvelopeResponse(t, uploadReq, &uploadResp)

	t.Run("requires auth", func(t *testing.T) {
		req := testutil.NewGETRequest(t)
		err := app.GetCurrentUserChatBackground(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
	})

	t.Run("serves current user's asset", func(t *testing.T) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, owner.ID)

		err := app.GetCurrentUserChatBackground(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		assert.Equal(t, "private", string(req.RequestCtx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "image/png", string(req.RequestCtx.Response.Header.Peek("Content-Type")))

		expectedBody := decodeBase64Fixture(t, tinyPNGBase64)
		assert.Equal(t, expectedBody, req.RequestCtx.Response.Body())
	})

	t.Run("does not expose another user's asset", func(t *testing.T) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, otherUser.ID)

		err := app.GetCurrentUserChatBackground(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})
}

func TestApp_GetCurrentUserSettings_IncludesChatBackgroundMetadata(t *testing.T) {
	t.Parallel()

	app := newChatBackgroundTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("chat-bg-me")),
	)

	user.Settings = models.JSONB{
		"chat_background": map[string]any{
			"kind":      "preset",
			"preset_id": "dot-orbit",
		},
	}
	require.NoError(t, app.DB.Save(user).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.GetCurrentUser(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &envelope))

	var resp struct {
		Settings map[string]any `json:"settings"`
	}
	require.NoError(t, json.Unmarshal(envelope.Data, &resp))
	require.Contains(t, resp.Settings, "chat_background")
}
