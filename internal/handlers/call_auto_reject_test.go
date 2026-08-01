package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCallAutoRejectSettings(t *testing.T) {
	defaults := callAutoRejectSettings{
		Message: defaultCallAutoRejectMessage,
	}

	tests := []struct {
		name  string
		in    models.JSONB
		check func(t *testing.T, s callAutoRejectSettings)
	}{
		{
			name: "missing block keeps defaults and stays disabled",
			in:   models.JSONB{},
			check: func(t *testing.T, s callAutoRejectSettings) {
				assert.False(t, s.Enabled)
				assert.Equal(t, defaultCallAutoRejectMessage, s.Message)
			},
		},
		{
			name: "full block applied",
			in: models.JSONB{"call_auto_reject": map[string]any{
				"enabled": true,
				"message": "مكالمات مش متاحة، ابعت رسالة",
			}},
			check: func(t *testing.T, s callAutoRejectSettings) {
				assert.True(t, s.Enabled)
				assert.Equal(t, "مكالمات مش متاحة، ابعت رسالة", s.Message)
			},
		},
		{
			name: "explicit empty message disables the automated text",
			in: models.JSONB{"call_auto_reject": map[string]any{
				"enabled": true,
				"message": "",
			}},
			check: func(t *testing.T, s callAutoRejectSettings) {
				assert.True(t, s.Enabled)
				assert.Equal(t, "", s.Message)
			},
		},
		{
			name: "whitespace-only message is treated as empty",
			in: models.JSONB{"call_auto_reject": map[string]any{
				"message": "   ",
			}},
			check: func(t *testing.T, s callAutoRejectSettings) {
				assert.Equal(t, "", s.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, parseCallAutoRejectSettings(tt.in, defaults))
		})
	}
}

// callRejectMock is a GOWA API mock that records every request path/body and
// can be told to fail POST /call/reject (call already ended or answered).
// It also serves the device webhook config endpoints so the call.offer
// subscription repair can be exercised.
type callRejectMock struct {
	server        *httptest.Server
	mu            sync.Mutex
	paths         []string
	rejectBody    map[string]any
	failReject    bool
	rejectCalled  bool
	webhookEvents string // served on GET /devices/{id}/webhook
	patchedEvents string // captured from PATCH /devices/{id}/webhook
	// lidMap maps a LID number (without suffix) to a phone number, simulating
	// GOWA's /user/info?phone=<lid>@lid resolved_phone field.
	lidMap map[string]string
}

func newCallRejectMock(t *testing.T) *callRejectMock {
	t.Helper()
	m := &callRejectMock{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.paths = append(m.paths, r.URL.Path)
		if strings.HasPrefix(r.URL.Path, "/devices/") && strings.HasSuffix(r.URL.Path, "/webhook") {
			events := m.webhookEvents
			if r.Method == http.MethodPatch {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if v, ok := body["webhook_events"].(string); ok {
					m.patchedEvents = v
					events = v
				}
			}
			m.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "SUCCESS",
				"message": "Success",
				"results": map[string]any{
					"webhook_url":    "http://localhost:8080/api/gowa/webhook",
					"webhook_secret": "secret",
					"webhook_events": events,
				},
			})
			return
		}
		if r.URL.Path == "/user/info" {
			phone := r.URL.Query().Get("phone")
			m.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(phone, "@lid") {
				bare := strings.TrimSuffix(phone, "@lid")
				if resolved, ok := m.lidMap[bare]; ok {
					_, _ = w.Write([]byte(`{"code":"SUCCESS","message":"Success get user info","results":{"resolved_phone":"` + resolved + `"}}`))
					return
				}
			}
			// Default: return a minimal success so the fallback path works.
			_, _ = w.Write([]byte(`{"code":"SUCCESS","message":"Success get user info","results":{}}`))
			return
		}
		if r.URL.Path == "/call/reject" {
			m.rejectCalled = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			m.rejectBody = body
			m.mu.Unlock()
			if m.failReject {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"code":"ERROR","message":"call not found"}`))
				return
			}
			// Real GOWA answers /call/reject with a GenericResponse that has
			// no message_id — the mock must match, or it hides parsing bugs.
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"SUCCESS","message":"Success","results":null}`))
			return
		}
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{"message_id": "GOWA_MSG_REJECT", "status": "ok"},
		})
	}))
	t.Cleanup(m.server.Close)
	return m
}

func (m *callRejectMock) calledPaths() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.paths...)
}

// newCallRejectTestApp wires an App whose GOWA provider talks to the given mock.
func newCallRejectTestApp(t *testing.T, mock *callRejectMock) *App {
	t.Helper()
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(mock.server.URL, username, password)
		},
	)

	app := &App{
		DB:         db,
		Log:        log,
		WARegistry: whatsapp.NewRegistry(log),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}
	return app
}

func callOfferEnvelope(t *testing.T, deviceID, from, callID string) *gowa.WebhookPayload {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"call_id": callID, "from": from})
	require.NoError(t, err)
	return &gowa.WebhookPayload{
		Event:    "call.offer",
		DeviceID: deviceID,
		Payload:  payload,
	}
}

// callOfferEnvelopeWithChat is like callOfferEnvelope but also includes chat_id
// in the payload — the phone-number-based conversation JID that GOWA may send
// alongside the internal call-signaling `from` address.
func callOfferEnvelopeWithChat(t *testing.T, deviceID, from, callID, chatID string) *gowa.WebhookPayload {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"call_id": callID,
		"from":    from,
		"chat_id": chatID,
	})
	require.NoError(t, err)
	return &gowa.WebhookPayload{
		Event:    "call.offer",
		DeviceID: deviceID,
		Payload:  payload,
	}
}

func TestProcessGowaCallOffer_RejectsAndSendsMessage(t *testing.T) {
	mock := newCallRejectMock(t)
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{
			"enabled": true,
			"message": "معلش، ابعت رسالة وهنرد عليك",
		},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	app.processGowaCallOffer(account, callOfferEnvelope(t, account.GowaDeviceID, "628123456789@s.whatsapp.net", "CALL_001"))

	// Rejection hit the GOWA API with the exact payload /call/reject expects.
	require.True(t, mock.rejectCalled, "must POST /call/reject")
	assert.Equal(t, "628123456789@s.whatsapp.net", mock.rejectBody["caller_jid"])
	assert.Equal(t, "CALL_001", mock.rejectBody["call_id"])

	// The caller got a contact stamped with the receiving account…
	var contact models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "628123456789").First(&contact).Error)
	assert.Equal(t, account.Name, contact.WhatsAppAccount)

	// …and the automated message was saved as an outgoing text.
	var msgs []models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND direction = ?", contact.ID, models.DirectionOutgoing).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "معلش، ابعت رسالة وهنرد عليك", msgs[0].Content)
}

func TestProcessGowaCallOffer_DisabledDoesNothing(t *testing.T) {
	mock := newCallRejectMock(t)
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	app.processGowaCallOffer(account, callOfferEnvelope(t, account.GowaDeviceID, "628123456789@s.whatsapp.net", "CALL_002"))

	assert.Empty(t, mock.calledPaths(), "disabled accounts must not touch the GOWA API")
	var count int64
	app.DB.Model(&models.Contact{}).Where("organization_id = ?", org.ID).Count(&count)
	assert.Zero(t, count, "no contact should be created")
}

func TestProcessGowaCallOffer_EmptyMessageRejectsSilently(t *testing.T) {
	mock := newCallRejectMock(t)
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{"enabled": true, "message": ""},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	app.processGowaCallOffer(account, callOfferEnvelope(t, account.GowaDeviceID, "628123456789@s.whatsapp.net", "CALL_003"))

	require.True(t, mock.rejectCalled, "rejection still happens with an empty message")
	assert.Equal(t, []string{"/call/reject"}, mock.calledPaths(), "no send endpoint should be called")
	var count int64
	app.DB.Model(&models.Message{}).Where("organization_id = ?", org.ID).Count(&count)
	assert.Zero(t, count, "no automated message should be saved")
}

func TestProcessGowaCallOffer_RejectFailureSkipsMessage(t *testing.T) {
	mock := newCallRejectMock(t)
	mock.failReject = true // call already ended/answered — GOWA errors
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{"enabled": true, "message": "معلش، ابعت رسالة"},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	app.processGowaCallOffer(account, callOfferEnvelope(t, account.GowaDeviceID, "628123456789@s.whatsapp.net", "CALL_004"))

	require.True(t, mock.rejectCalled)
	var count int64
	app.DB.Model(&models.Message{}).Where("organization_id = ?", org.ID).Count(&count)
	assert.Zero(t, count, "callers who hung up must not receive a rejection text")
}

func TestProcessGowaCallOffer_MalformedPayloadIgnored(t *testing.T) {
	mock := newCallRejectMock(t)
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{"enabled": true},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	// Missing call_id/from must be dropped without touching the GOWA API.
	app.processGowaCallOffer(account, &gowa.WebhookPayload{
		Event:    "call.offer",
		DeviceID: account.GowaDeviceID,
		Payload:  json.RawMessage(`{}`),
	})

	assert.Empty(t, mock.calledPaths())
}

// TestProcessGowaCallOffer_ChatIDPreferredForMessage verifies that when GOWA
// includes chat_id (the phone-number-based conversation JID) alongside an
// internal call-signaling from address, the rejection message is sent to the
// chat_id — not to the internal ID that /send/message can't reach.
func TestProcessGowaCallOffer_ChatIDPreferredForMessage(t *testing.T) {
	mock := newCallRejectMock(t)
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{
			"enabled": true,
			"message": "Sorry we can't take calls",
		},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	// from is an internal WhatsApp call-signaling ID; chat_id is the real phone.
	app.processGowaCallOffer(account, callOfferEnvelopeWithChat(t,
		account.GowaDeviceID,
		"149641526026409@s.whatsapp.net", // internal ID
		"CALL_005",
		"966561853319@s.whatsapp.net", // real phone JID
	))

	// Rejection still uses the exact signaling `from` value.
	require.True(t, mock.rejectCalled)
	assert.Equal(t, "149641526026409@s.whatsapp.net", mock.rejectBody["caller_jid"])

	// The automated message is stored against the chat_id phone number, not
	// the internal ID.
	var contact models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "966561853319").First(&contact).Error)
	var msgs []models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND direction = ?", contact.ID, models.DirectionOutgoing).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Sorry we can't take calls", msgs[0].Content)
}

// TestProcessGowaCallOffer_LIDResolvedToPhoneNumber verifies that when the
// call.offer `from` field is a WhatsApp LID (not a phone JID) and no chat_id
// is provided, the handler resolves the LID to a real phone number via GOWA's
// /user/info endpoint before sending the automated message.
func TestProcessGowaCallOffer_LIDResolvedToPhoneNumber(t *testing.T) {
	mock := newCallRejectMock(t)
	mock.lidMap = map[string]string{
		"149641526026409": "966561853319",
	}
	app := newCallRejectTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{
			"enabled": true,
			"message": "Please text us instead",
		},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)
	require.NoError(t, app.DB.First(account, account.ID).Error)

	// No chat_id — from is a LID disguised as @s.whatsapp.net.
	app.processGowaCallOffer(account, callOfferEnvelope(t,
		account.GowaDeviceID,
		"149641526026409@s.whatsapp.net",
		"CALL_006",
	))

	// Rejection still uses the exact `from` value.
	require.True(t, mock.rejectCalled)
	assert.Equal(t, "149641526026409@s.whatsapp.net", mock.rejectBody["caller_jid"])

	// The message was saved against the RESOLVED phone number, not the LID.
	var contact models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "966561853319").First(&contact).Error,
		"contact must be created with the resolved phone number")
	var msgs []models.Message
	require.NoError(t, app.DB.Where("contact_id = ? AND direction = ?", contact.ID, models.DirectionOutgoing).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Please text us instead", msgs[0].Content)
}

func TestEnsureCallOfferSubscription(t *testing.T) {
	t.Run("adds call.offer to a stale subscription", func(t *testing.T) {
		mock := newCallRejectMock(t)
		mock.webhookEvents = "message,message.ack"
		app := newCallRejectTestApp(t, mock)
		org := testutil.CreateTestOrganization(t, app.DB)
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		app.ensureCallOfferSubscription(account)

		assert.Equal(t, "message,message.ack,call.offer", mock.patchedEvents)
	})

	t.Run("already subscribed leaves the config untouched", func(t *testing.T) {
		mock := newCallRejectMock(t)
		mock.webhookEvents = "message,call.offer"
		app := newCallRejectTestApp(t, mock)
		org := testutil.CreateTestOrganization(t, app.DB)
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		app.ensureCallOfferSubscription(account)

		assert.Empty(t, mock.patchedEvents, "no PATCH expected")
	})

	t.Run("empty events means all events — nothing to repair", func(t *testing.T) {
		mock := newCallRejectMock(t)
		app := newCallRejectTestApp(t, mock)
		org := testutil.CreateTestOrganization(t, app.DB)
		account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

		app.ensureCallOfferSubscription(account)

		assert.Empty(t, mock.patchedEvents, "no PATCH expected")
	})
}
