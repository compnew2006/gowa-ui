package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/worker"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wameow "go.mau.fi/whatsmeow"
)

// testServerTransport redirects Meta API requests to our httptest server.
type testServerTransport struct {
	serverURL string
}

func (t *testServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	testReq := req.Clone(req.Context())
	testReq.URL.Scheme = "http"
	testReq.URL.Host = t.serverURL[7:] // Strip "http://"
	return http.DefaultTransport.RoundTrip(testReq)
}

type failingWhatsmeowProvider struct {
	getClientCalled bool
	err             error
}

func (p *failingWhatsmeowProvider) GetClient(ctx context.Context, instanceID string) (*wameow.Client, error) {
	p.getClientCalled = true
	return nil, p.err
}

func (p *failingWhatsmeowProvider) SendText(ctx context.Context, instanceID string, to string, text string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string, caption string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) MarkRead(ctx context.Context, instanceID string, messageID string) error {
	return nil
}

func (p *failingWhatsmeowProvider) SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error {
	return nil
}

func (p *failingWhatsmeowProvider) RevokeMessage(ctx context.Context, instanceID string, messageID string) error {
	return nil
}

func (p *failingWhatsmeowProvider) GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error) {
	return "", nil
}

func (p *failingWhatsmeowProvider) DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error) {
	return nil, nil
}

func (p *failingWhatsmeowProvider) UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error) {
	return "", nil
}

func TestWorker_HandleWhatsAppFilterJob_MetaSuccess(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := testutil.CreateTestOrganization(t, db)

	// Create decrypted Meta WhatsApp Account
	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Meta Main Account",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test-encrypted-token", // decrypted to test-encrypted-token during fake decryption
		APIVersion:     "v21.0",
		Status:         "active",
	}
	require.NoError(t, db.Create(account).Error)

	// Seed Filter Batch
	batch := &models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "Meta Main Account",
		Status:          models.FilterStatusPending,
		TotalNumbers:    2,
	}
	require.NoError(t, db.Create(batch).Error)

	// Seed Filter Results (1 valid, 1 invalid)
	res1 := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batch.ID,
		PhoneNumber: "+1234567890",
		ContactName: "Alice",
	}
	require.NoError(t, db.Create(res1).Error)

	res2 := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batch.ID,
		PhoneNumber: "+9876543210",
		ContactName: "Bob",
	}
	require.NoError(t, db.Create(res2).Error)

	// Setup mock Meta Contacts API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/123456789/contacts")

		var reqBody whatsapp.ContactCheckRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Len(t, reqBody.Contacts, 2)

		resp := whatsapp.ContactCheckResponse{
			Contacts: []whatsapp.ContactCheckItem{
				{
					Input:  "+1234567890",
					Status: "valid",
					WaID:   "1234567890_jid",
				},
				{
					Input:  "+9876543210",
					Status: "invalid",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup Worker & dependencies
	log := testutil.NopLogger()
	cfg := &config.Config{
		App: config.AppConfig{
			EncryptionKey: testutil.TestEncryptionKey,
		},
		WhatsApp: config.WhatsAppConfig{
			Provider: "meta",
		},
	}

	waClient := whatsapp.NewWithTimeout(log, 5*time.Second)
	waClient.HTTPClient = &http.Client{
		Transport: &testServerTransport{serverURL: server.URL},
	}

	w := &worker.Worker{
		Config:   cfg,
		DB:       db,
		Log:      log,
		WhatsApp: waClient,
	}

	// Process validation job
	job := &queue.WhatsAppFilterJob{
		BatchID:           batch.ID,
		OrganizationID:    org.ID,
		WhatsAppAccountID: account.ID,
		EnqueuedAt:        time.Now(),
	}

	err := w.HandleWhatsAppFilterJob(context.Background(), job)
	require.NoError(t, err)

	// Verify database batch is completed
	var updatedBatch models.WhatsAppFilterBatch
	require.NoError(t, db.First(&updatedBatch, "id = ?", batch.ID).Error)
	assert.Equal(t, models.FilterStatusCompleted, updatedBatch.Status)
	assert.Equal(t, 1, updatedBatch.ValidNumbers)
	assert.Equal(t, 1, updatedBatch.InvalidNumbers)
	assert.NotNil(t, updatedBatch.CompletedAt)
	assert.Empty(t, updatedBatch.ErrorMessage)

	// Verify Alice is marked valid and registered name updated
	var updatedRes1 models.WhatsAppFilterResult
	require.NoError(t, db.First(&updatedRes1, "id = ?", res1.ID).Error)
	assert.True(t, updatedRes1.IsValid)
	assert.Equal(t, "1234567890_jid", updatedRes1.ContactName)
	assert.NotNil(t, updatedRes1.CheckedAt)
	assert.Empty(t, updatedRes1.ErrorMessage)

	// Verify Bob is marked invalid
	var updatedRes2 models.WhatsAppFilterResult
	require.NoError(t, db.First(&updatedRes2, "id = ?", res2.ID).Error)
	assert.False(t, updatedRes2.IsValid)
	assert.NotNil(t, updatedRes2.CheckedAt)
}

func TestWorker_HandleWhatsAppFilterJob_MetaFailure(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := testutil.CreateTestOrganization(t, db)

	// Create decrypted Meta WhatsApp Account
	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Meta Main Account",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test-encrypted-token",
		APIVersion:     "v21.0",
		Status:         "active",
	}
	require.NoError(t, db.Create(account).Error)

	// Seed Filter Batch
	batch := &models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "Meta Main Account",
		Status:          models.FilterStatusPending,
		TotalNumbers:    1,
	}
	require.NoError(t, db.Create(batch).Error)

	// Seed Filter Result
	res := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batch.ID,
		PhoneNumber: "+1234567890",
		ContactName: "Alice",
	}
	require.NoError(t, db.Create(res).Error)

	// Setup mock Meta Contacts API server returning HTTP 500 Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Meta Graph API Server Error",
				"code":    500,
			},
		})
	}))
	defer server.Close()

	// Setup Worker & dependencies
	log := testutil.NopLogger()
	cfg := &config.Config{
		App: config.AppConfig{
			EncryptionKey: testutil.TestEncryptionKey,
		},
		WhatsApp: config.WhatsAppConfig{
			Provider: "meta",
		},
	}

	waClient := whatsapp.NewWithTimeout(log, 5*time.Second)
	waClient.HTTPClient = &http.Client{
		Transport: &testServerTransport{serverURL: server.URL},
	}

	w := &worker.Worker{
		Config:   cfg,
		DB:       db,
		Log:      log,
		WhatsApp: waClient,
	}

	// Process validation job
	job := &queue.WhatsAppFilterJob{
		BatchID:           batch.ID,
		OrganizationID:    org.ID,
		WhatsAppAccountID: account.ID,
		EnqueuedAt:        time.Now(),
	}

	err := w.HandleWhatsAppFilterJob(context.Background(), job)
	require.Error(t, err)

	// Verify database batch is failed
	var updatedBatch models.WhatsAppFilterBatch
	require.NoError(t, db.First(&updatedBatch, "id = ?", batch.ID).Error)
	assert.Equal(t, models.FilterStatusFailed, updatedBatch.Status)
	assert.Contains(t, updatedBatch.ErrorMessage, "meta check contacts failed")
	assert.NotNil(t, updatedBatch.CompletedAt)
}

func TestWorker_HandleWhatsAppFilterJob_UsesExistingContactBeforeProviderLookup(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := testutil.CreateTestOrganization(t, db)

	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "14155552671",
		ProfileName:    "Existing Alice",
	}
	require.NoError(t, db.Create(contact).Error)

	batch := &models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "Meta Main Account",
		Status:          models.FilterStatusPending,
		TotalNumbers:    1,
	}
	require.NoError(t, db.Create(batch).Error)

	res := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batch.ID,
		PhoneNumber: "14155552671",
		ContactName: "Uploaded Alice",
	}
	require.NoError(t, db.Create(res).Error)

	w := &worker.Worker{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "meta"},
		},
		DB:       db,
		Log:      testutil.NopLogger(),
		WhatsApp: whatsapp.NewWithTimeout(testutil.NopLogger(), 5*time.Second),
	}

	job := &queue.WhatsAppFilterJob{
		BatchID:           batch.ID,
		OrganizationID:    org.ID,
		WhatsAppAccountID: uuid.New(),
		EnqueuedAt:        time.Now(),
	}

	err := w.HandleWhatsAppFilterJob(context.Background(), job)
	require.NoError(t, err)

	var updatedBatch models.WhatsAppFilterBatch
	require.NoError(t, db.First(&updatedBatch, "id = ?", batch.ID).Error)
	assert.Equal(t, models.FilterStatusCompleted, updatedBatch.Status)
	assert.Equal(t, 1, updatedBatch.ValidNumbers)
	assert.Equal(t, 0, updatedBatch.InvalidNumbers)

	var updatedResult models.WhatsAppFilterResult
	require.NoError(t, db.First(&updatedResult, "id = ?", res.ID).Error)
	assert.True(t, updatedResult.IsValid)
	assert.Equal(t, "Existing Alice", updatedResult.ContactName)
	assert.Equal(t, "Found in contacts", updatedResult.ErrorMessage)
	assert.NotNil(t, updatedResult.CheckedAt)
}

func TestWorker_HandleWhatsAppFilterJob_WhatsmeowUsesProviderClientGetter(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := testutil.CreateTestOrganization(t, db)
	instanceID := uuid.New()

	batch := &models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: "Primary",
		InstanceID:      &instanceID,
		Status:          models.FilterStatusPending,
		TotalNumbers:    1,
	}
	require.NoError(t, db.Create(batch).Error)

	res := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batch.ID,
		PhoneNumber: "+14155552671",
		ContactName: "Alice",
	}
	require.NoError(t, db.Create(res).Error)

	provider := &failingWhatsmeowProvider{err: errors.New("forced reconnect failed")}
	w := &worker.Worker{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:              db,
		Log:             testutil.NopLogger(),
		MessageProvider: provider,
	}

	job := &queue.WhatsAppFilterJob{
		BatchID:        batch.ID,
		OrganizationID: org.ID,
		InstanceID:     &instanceID,
		EnqueuedAt:     time.Now(),
	}

	err := w.HandleWhatsAppFilterJob(context.Background(), job)
	require.Error(t, err)
	assert.True(t, provider.getClientCalled)
	assert.True(t, strings.Contains(err.Error(), "forced reconnect failed"))

	var updatedBatch models.WhatsAppFilterBatch
	require.NoError(t, db.First(&updatedBatch, "id = ?", batch.ID).Error)
	assert.Equal(t, models.FilterStatusFailed, updatedBatch.Status)
	assert.Contains(t, updatedBatch.ErrorMessage, "forced reconnect failed")
}
