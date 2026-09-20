package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// Internal-package tests for the media-retention processor: they exercise the
// unexported claim → refcheck → delete state machine directly (the external
// handlers_test package can only see the HTTP surface).

// newRetentionTestApp builds a minimal App (DB + Redis + config + logger),
// mirroring the handlers_test newTestApp but inside package handlers so the
// unexported retention methods are reachable.
func newRetentionTestApp(t *testing.T) *App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set, skipping test")
	}

	return &App{
		Config: &config.Config{
			App: config.AppConfig{
				EncryptionKey: "test-encryption-key-for-handlers-longer-than-32-chars",
			},
			JWT: config.JWTConfig{Secret: testutil.TestJWTSecret},
		},
		DB:    db,
		Log:   testutil.NopLogger(),
		Redis: redisClient,
	}
}

// retentionAdminUser creates a super-admin user (full visibility through
// scopeAssignedContact). Replica of the handlers_test createAdminUser, which
// is not visible from this package.
func retentionAdminUser(t *testing.T, app *App, orgID uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateAdminRole(t, app.DB, orgID)
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))
}

// enableRetention writes the media_retention block onto an account and
// reloads it so the in-memory struct carries the settings (processAccount
// reads retention_days from the struct it is handed).
func enableRetention(t *testing.T, app *App, account *models.WhatsAppAccount, days int) {
	t.Helper()
	require.NoError(t, app.DB.Model(account).Update("settings", models.JSONB{
		"media_retention": map[string]any{"enabled": true, "retention_days": days},
	}).Error)
	require.NoError(t, app.DB.First(account, "id = ?", account.ID).Error)
}

// writeRetentionFile creates a real file under the app's storage root.
func writeRetentionFile(t *testing.T, app *App, rel string, content string) string {
	t.Helper()
	abs := filepath.Join(app.getMediaStoragePath(), rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0644))
	return abs
}

// createRetentionMessage inserts a message row and backdates created_at.
func createRetentionMessage(t *testing.T, app *App, orgID, contactID uuid.UUID, accountName, msgType, rel string, age time.Duration) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: accountName,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageType(msgType),
		MediaURL:        rel,
		MediaFilename:   "file.jpg",
		MediaMimeType:   "image/jpeg",
		Status:          models.MessageStatusDelivered,
		Metadata:        models.JSONB{},
	}
	require.NoError(t, app.DB.Create(msg).Error)
	if age > 0 {
		require.NoError(t, app.DB.Model(msg).Update("created_at", time.Now().Add(-age)).Error)
	}
	return msg
}

func reloadMessage(t *testing.T, app *App, id uuid.UUID) models.Message {
	t.Helper()
	var msg models.Message
	require.NoError(t, app.DB.Unscoped().First(&msg, "id = ?", id).Error)
	return msg
}

// --- Claim → delete happy path ---

func TestRetentionPurgeMessage_DeletesFileAndTombstones(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rel := filepath.Join("images", "old.jpg")
	writeRetentionFile(t, app, rel, "old-bytes")
	msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, 40*24*time.Hour)

	stats := mediaRetentionPassStats{}
	app.retentionPurgeMessage(msg, time.Now(), &stats)

	updated := reloadMessage(t, app, msg.ID)
	assert.Equal(t, 1, stats.purged)
	assert.Empty(t, updated.MediaURL, "reference must be cleared")
	assert.Equal(t, "purged", updated.Metadata[retentionStateKey])
	assert.NotEmpty(t, updated.Metadata[retentionOriginalPathKey])
	assert.Equal(t, rel, updated.Metadata[retentionOriginalPathKey])
	assert.NotEmpty(t, updated.Metadata[retentionPurgedAtKey])
	_, err := os.Stat(filepath.Join(app.getMediaStoragePath(), rel))
	assert.True(t, os.IsNotExist(err), "file must be gone from disk")
}

// A newer message (within the window) must never be handed to the purge path
// by the scan — but if it were, the claim is what it is; the guarantee tested
// here is the caller-side one via processAccount below.

// --- Shared-file protection ---

func TestRetentionPurgeMessage_SharedFileKeptForOtherOwners(t *testing.T) {
	cases := []struct {
		name   string
		setUp  func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string)
		assert func(t *testing.T, app *App, rel string)
	}{
		{
			name: "second message row",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, time.Hour)
			},
			assert: func(t *testing.T, app *App, rel string) {},
		},
		{
			name: "soft-deleted second message row",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				other := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, time.Hour)
				require.NoError(t, app.DB.Delete(other).Error)
			},
			assert: func(t *testing.T, app *App, rel string) {},
		},
		{
			name: "pending scheduled message",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				creator := retentionAdminUser(t, app, org.ID)
				require.NoError(t, app.DB.Create(&models.ScheduledMessage{
					BaseModel:       models.BaseModel{ID: uuid.New()},
					OrganizationID:  org.ID,
					ContactID:       contact.ID,
					WhatsAppAccount: "acct",
					MessageType:     models.MessageTypeImage,
					MediaURL:        rel,
					ScheduledAt:     time.Now().Add(time.Hour),
					Status:          models.ScheduledMessageStatusPending,
					CreatedBy:       creator.ID,
				}).Error)
			},
			assert: func(t *testing.T, app *App, rel string) {},
		},
		{
			name: "sent scheduled message does NOT protect (fire-time copy is its own message row)",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				creator := retentionAdminUser(t, app, org.ID)
				require.NoError(t, app.DB.Create(&models.ScheduledMessage{
					BaseModel:       models.BaseModel{ID: uuid.New()},
					OrganizationID:  org.ID,
					ContactID:       contact.ID,
					WhatsAppAccount: "acct",
					MessageType:     models.MessageTypeImage,
					MediaURL:        rel,
					ScheduledAt:     time.Now().Add(-time.Hour),
					Status:          models.ScheduledMessageStatusSent,
					CreatedBy:       creator.ID,
				}).Error)
			},
			assert: func(t *testing.T, app *App, rel string) {
				_, err := os.Stat(filepath.Join(app.getMediaStoragePath(), rel))
				assert.True(t, os.IsNotExist(err), "file with only a SENT scheduled reference must be deleted")
			},
		},
		{
			name: "draft campaign header media",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				creator := retentionAdminUser(t, app, org.ID)
				template := &models.Template{
					BaseModel:       models.BaseModel{ID: uuid.New()},
					OrganizationID:  org.ID,
					WhatsAppAccount: "acct",
					Name:            "tpl-" + uuid.NewString()[:8],
					Language:        "ar",
					BodyContent:     "hello",
				}
				require.NoError(t, app.DB.Create(template).Error)
				require.NoError(t, app.DB.Create(&models.BulkMessageCampaign{
					BaseModel:            models.BaseModel{ID: uuid.New()},
					OrganizationID:       org.ID,
					WhatsAppAccount:      "acct",
					Name:                 "camp-" + uuid.NewString()[:8],
					TemplateID:           template.ID,
					HeaderMediaLocalPath: rel,
					Status:               "draft",
					CreatedBy:            creator.ID,
				}).Error)
			},
			assert: func(t *testing.T, app *App, rel string) {},
		},
		{
			name: "contact avatar",
			setUp: func(t *testing.T, app *App, org *models.Organization, contact *models.Contact, rel string) {
				require.NoError(t, app.DB.Model(contact).
					Update("avatar_local_path", rel).Error)
			},
			assert: func(t *testing.T, app *App, rel string) {},
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newRetentionTestApp(t)
			app.Config.Storage.LocalPath = t.TempDir()
			org := testutil.CreateTestOrganization(t, app.DB)
			contact := testutil.CreateTestContact(t, app.DB, org.ID)

			// Unique path per subtest: the refcheck is GLOBAL across the
			// shared test DB, so a leftover row from a previous subtest with
			// the same path would wrongly protect this subtest's file.
			rel := filepath.Join("images", fmt.Sprintf("shared-%d.jpg", i))
			writeRetentionFile(t, app, rel, "shared-bytes")
			msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, 40*24*time.Hour)
			tc.setUp(t, app, org, contact, rel)

			stats := mediaRetentionPassStats{}
			app.retentionPurgeMessage(msg, time.Now(), &stats)

			updated := reloadMessage(t, app, msg.ID)
			// The expired message's own reference is ALWAYS cleared…
			assert.Empty(t, updated.MediaURL)
			assert.Equal(t, "purged", updated.Metadata[retentionStateKey])
			// …but the file survives when another owner still references it.
			if tc.name != "sent scheduled message does NOT protect (fire-time copy is its own message row)" {
				assert.Equal(t, 1, stats.purgedShared)
				assert.Zero(t, stats.purged)
				_, err := os.Stat(filepath.Join(app.getMediaStoragePath(), rel))
				assert.NoError(t, err, "shared file must stay on disk")
			}
			tc.assert(t, app, rel)
		})
	}
}

// --- Race / missing-file / delete-failure edges ---

func TestRetentionPurgeMessage_LostRaceNeverTouchesFile(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rel := filepath.Join("images", "raced.jpg")
	writeRetentionFile(t, app, rel, "bytes")
	msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, 40*24*time.Hour)

	// Simulate a concurrent winner: media_url changed between the scan and
	// the claim (e.g. redownload restored a different path). Update via a
	// fresh model so the in-hand scan struct keeps the OLD value.
	require.NoError(t, app.DB.Model(&models.Message{}).Where("id = ?", msg.ID).
		Update("media_url", "images/new.jpg").Error)

	stats := mediaRetentionPassStats{}
	app.retentionPurgeMessage(msg, time.Now(), &stats)

	assert.Equal(t, 1, stats.lostRace)
	assert.Zero(t, stats.purged)
	updated := reloadMessage(t, app, msg.ID)
	assert.Equal(t, "images/new.jpg", updated.MediaURL)
	assert.NotContains(t, updated.Metadata, retentionStateKey)
	_, err := os.Stat(filepath.Join(app.getMediaStoragePath(), rel))
	assert.NoError(t, err, "loser of the race must not delete the file")
}

func TestRetentionPurgeMessage_MissingFileIsSuccess(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rel := filepath.Join("images", "never-existed.jpg")
	msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, 40*24*time.Hour)

	stats := mediaRetentionPassStats{}
	app.retentionPurgeMessage(msg, time.Now(), &stats)

	assert.Equal(t, 1, stats.purged)
	updated := reloadMessage(t, app, msg.ID)
	assert.Equal(t, "purged", updated.Metadata[retentionStateKey])
}

func TestRetentionPurgeMessage_DeleteFailureMarkedAndLeased(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// A non-empty DIRECTORY at the media path: resolveMediaPath accepts it,
	// os.Remove refuses to delete non-empty dirs → deterministic failure.
	rel := filepath.Join("documents", "stuck")
	absDir := filepath.Join(app.getMediaStoragePath(), rel)
	require.NoError(t, os.MkdirAll(absDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(absDir, "child.txt"), []byte("x"), 0644))

	msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "document", rel, 40*24*time.Hour)

	stats := mediaRetentionPassStats{}
	app.retentionPurgeMessage(msg, time.Now(), &stats)

	assert.Equal(t, 1, stats.deleteFailed)
	updated := reloadMessage(t, app, msg.ID)
	assert.Empty(t, updated.MediaURL)
	assert.Equal(t, "delete_failed", updated.Metadata[retentionStateKey])
	assert.Equal(t, rel, updated.Metadata[retentionOriginalPathKey])
}

// --- Stuck sweep (crash recovery) ---

func TestRetentionSweepStuck_FinishesCrashedClaims(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rel := filepath.Join("images", "crashed.jpg")
	writeRetentionFile(t, app, rel, "bytes")

	// A claim stranded mid-flight: state purging, old lease, file still there.
	stuck := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, 40*24*time.Hour)
	oldLease := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	require.NoError(t, app.DB.Exec(
		`UPDATE messages SET media_url = '', metadata = ?::jsonb WHERE id = ?`,
		`{"retention_state":"purging","retention_purged_at":"`+oldLease+`","retention_original_path":"`+rel+`"}`,
		stuck.ID).Error)

	// A fresh claim (lease < 1h) that must be LEFT ALONE this pass.
	freshRel := filepath.Join("images", "fresh.jpg")
	writeRetentionFile(t, app, freshRel, "bytes")
	fresh := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", freshRel, 40*24*time.Hour)
	freshLease := time.Now().UTC().Format(time.RFC3339)
	require.NoError(t, app.DB.Exec(
		`UPDATE messages SET media_url = '', metadata = ?::jsonb WHERE id = ?`,
		`{"retention_state":"purging","retention_purged_at":"`+freshLease+`","retention_original_path":"`+freshRel+`"}`,
		fresh.ID).Error)

	stats := mediaRetentionPassStats{}
	processed := app.retentionSweepStuck(time.Now(), 10, &stats)

	assert.Equal(t, 1, processed, "only the lease-expired row is swept")

	done := reloadMessage(t, app, stuck.ID)
	assert.Equal(t, "purged", done.Metadata[retentionStateKey])
	_, err := os.Stat(filepath.Join(app.getMediaStoragePath(), rel))
	assert.True(t, os.IsNotExist(err), "crashed claim's file must be finished")

	stillFresh := reloadMessage(t, app, fresh.ID)
	assert.Equal(t, "purging", stillFresh.Metadata[retentionStateKey])
	_, err = os.Stat(filepath.Join(app.getMediaStoragePath(), freshRel))
	assert.NoError(t, err, "fresh-lease claim must not be retried yet")
}

// --- Backfill exclusion (the ping-pong guard) ---

func TestPendingBackfillQuery_ExcludesAllRetentionStates(t *testing.T) {
	app := newRetentionTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	mk := func(state string) uuid.UUID {
		msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", "", 40*24*time.Hour)
		require.NoError(t, app.DB.Model(msg).Updates(map[string]any{
			"whats_app_message_id": "wamid-" + uuid.NewString()[:8],
			"media_url":            "",
			"metadata":             models.JSONB{retentionStateKey: state},
		}).Error)
		return msg.ID
	}
	purged := mk("purged")
	failed := mk("delete_failed")
	purging := mk("purging")

	// A normal pending row (no retention state) stays eligible.
	normal := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", "", 40*24*time.Hour)
	require.NoError(t, app.DB.Model(normal).Updates(map[string]any{
		"whats_app_message_id": "wamid-" + uuid.NewString()[:8],
		"media_url":            "",
		"metadata":             models.JSONB{},
	}).Error)

	var selected []models.Message
	require.NoError(t, app.pendingBackfillQuery().Find(&selected).Error)
	ids := map[uuid.UUID]bool{}
	for _, m := range selected {
		ids[m.ID] = true
	}
	assert.True(t, ids[normal.ID], "normal pending row stays eligible")
	assert.False(t, ids[purged], "purged row must not be backfilled")
	assert.False(t, ids[failed], "delete_failed row must not be backfilled")
	assert.False(t, ids[purging], "in-flight purge must not be backfilled")
}

// --- ServeMedia 410 ---

func TestServeMedia_RetentionPurgedReturns410(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	user := retentionAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rel := filepath.Join("images", "gone.jpg")
	writeRetentionFile(t, app, rel, "bytes") // file present: 410 must not even serve it
	msg := createRetentionMessage(t, app, org.ID, contact.ID, "acct", "image", rel, time.Hour)
	require.NoError(t, app.DB.Model(msg).Updates(map[string]any{
		"media_url": "",
		"metadata":  models.JSONB{retentionStateKey: "purged", retentionOriginalPathKey: rel},
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", msg.ID.String())
	require.NoError(t, app.ServeMedia(req))
	assert.Equal(t, fasthttp.StatusGone, testutil.GetResponseStatusCode(req))
}

// --- Full account pass (scan filters) ---

func TestMediaRetentionProcessor_ProcessAccount(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("retail"))
	enableRetention(t, app, account, 30)

	oldRel := filepath.Join("images", "old.jpg")
	writeRetentionFile(t, app, oldRel, "old")
	old := createRetentionMessage(t, app, org.ID, contact.ID, "retail", "image", oldRel, 40*24*time.Hour)

	// New message: inside the window.
	newRel := filepath.Join("images", "new.jpg")
	writeRetentionFile(t, app, newRel, "new")
	createRetentionMessage(t, app, org.ID, contact.ID, "retail", "image", newRel, 5*24*time.Hour)

	// Template message: excluded by type even when old.
	tplRel := filepath.Join("images", "tpl.jpg")
	writeRetentionFile(t, app, tplRel, "tpl")
	createRetentionMessage(t, app, org.ID, contact.ID, "retail", "template", tplRel, 40*24*time.Hour)

	// Restored media: keep-until in the future protects it.
	keepRel := filepath.Join("images", "kept.jpg")
	writeRetentionFile(t, app, keepRel, "kept")
	createRetentionMessage(t, app, org.ID, contact.ID, "retail", "image", keepRel, 40*24*time.Hour)
	require.NoError(t, app.DB.Exec(
		`UPDATE messages SET metadata = metadata || ?::jsonb WHERE media_url = ?`,
		`{"retention_keep_until":"`+time.Now().Add(10*24*time.Hour).UTC().Format(time.RFC3339)+`"}`,
		keepRel).Error)

	// Soft-deleted old message: file purged, row stays soft-deleted.
	softRel := filepath.Join("images", "soft.jpg")
	writeRetentionFile(t, app, softRel, "soft")
	soft := createRetentionMessage(t, app, org.ID, contact.ID, "retail", "image", softRel, 40*24*time.Hour)
	require.NoError(t, app.DB.Delete(soft).Error)

	// Another org, SAME account name: isolation — not this account's org.
	otherOrg := testutil.CreateTestOrganization(t, app.DB)
	otherContact := testutil.CreateTestContact(t, app.DB, otherOrg.ID)
	otherRel := filepath.Join("images", "other-org.jpg")
	writeRetentionFile(t, app, otherRel, "other")
	createRetentionMessage(t, app, otherOrg.ID, otherContact.ID, "retail", "image", otherRel, 40*24*time.Hour)

	processor := NewMediaRetentionProcessor(app, time.Hour)
	stats := mediaRetentionPassStats{}
	processor.processAccount(account, &stats, time.Now().Add(time.Hour))

	assert.Equal(t, 2, stats.scanned, "old + soft-deleted only")
	assert.Equal(t, 2, stats.purged)

	gone := reloadMessage(t, app, old.ID)
	assert.Empty(t, gone.MediaURL)
	assert.Equal(t, "purged", gone.Metadata[retentionStateKey])
	assert.NoFileExists(t, filepath.Join(app.getMediaStoragePath(), oldRel))

	softGone := reloadMessage(t, app, soft.ID)
	assert.Empty(t, softGone.MediaURL)
	assert.NotNil(t, softGone.DeletedAt, "row must stay soft-deleted")
	assert.NoFileExists(t, filepath.Join(app.getMediaStoragePath(), softRel))

	// Untouched set. Fresh struct per lookup: GORM's First() adds the
	// destination's existing primary key as a condition otherwise.
	for _, rel := range []string{newRel, tplRel, keepRel, otherRel} {
		var kept models.Message
		require.NoError(t, app.DB.Unscoped().Where("media_url = ?", rel).First(&kept).Error, "message for %s must survive", rel)
		assert.FileExists(t, filepath.Join(app.getMediaStoragePath(), rel), rel)
	}
}

// --- Advisory lock ---

func TestMediaRetentionProcessor_AdvisoryLockBlocksSecondPass(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	enableRetention(t, app, account, 30)

	rel := filepath.Join("images", "locked.jpg")
	writeRetentionFile(t, app, rel, "bytes")
	msg := createRetentionMessage(t, app, org.ID, contact.ID, account.Name, "image", rel, 40*24*time.Hour)

	// Hold the pass lock on another open transaction.
	holder := app.DB.Begin()
	var ok bool
	require.NoError(t, holder.Raw("SELECT pg_try_advisory_xact_lock(?)", mediaRetentionAdvisoryKey).Scan(&ok).Error)
	require.True(t, ok)
	defer holder.Rollback()

	processor := NewMediaRetentionProcessor(app, time.Hour)
	processor.runPass(context.Background())

	stillThere := reloadMessage(t, app, msg.ID)
	assert.NotEmpty(t, stillThere.MediaURL, "locked-out pass must not purge anything")
	assert.FileExists(t, filepath.Join(app.getMediaStoragePath(), rel))
}

// --- Redownload restore ---

func TestApplyRetentionRestore_ClearsTombstoneAndSetsKeepUntil(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("restore-acct"))
	enableRetention(t, app, account, 30)

	// A purged row.
	msg := createRetentionMessage(t, app, org.ID, contact.ID, "restore-acct", "image", "", 40*24*time.Hour)
	require.NoError(t, app.DB.Exec(
		`UPDATE messages SET metadata = ?::jsonb WHERE id = ?`,
		`{"retention_state":"purged","retention_purged_at":"`+time.Now().UTC().Format(time.RFC3339)+`","retention_original_path":"images/x.jpg"}`,
		msg.ID).Error)

	newRel := filepath.Join("images", "restored.jpg")
	writeRetentionFile(t, app, newRel, "bytes")

	require.NoError(t, app.applyRetentionRestore(msg, newRel, "image/jpeg", time.Now()))

	updated := reloadMessage(t, app, msg.ID)
	assert.Equal(t, newRel, updated.MediaURL)
	assert.NotContains(t, updated.Metadata, retentionStateKey, "tombstone must be cleared so ServeMedia stops 410-ing")
	assert.NotContains(t, updated.Metadata, retentionOriginalPathKey)
	assert.NotEmpty(t, updated.Metadata[retentionRestoredAtKey])

	keepUntil := asString(updated.Metadata[retentionKeepUntilKey])
	require.NotEmpty(t, keepUntil, "account with retention must grant a keep-until window")
	parsed, err := time.Parse(time.RFC3339, keepUntil)
	require.NoError(t, err)
	assert.True(t, parsed.After(time.Now().Add(29*24*time.Hour)), "keep-until ≈ now + retention_days")
	assert.True(t, parsed.Before(time.Now().Add(31*24*time.Hour)))
}

func TestApplyRetentionRestore_NoRetentionMeansNoKeepUntil(t *testing.T) {
	app := newRetentionTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("no-retention"))

	msg := createRetentionMessage(t, app, org.ID, contact.ID, "no-retention", "image", "", 40*24*time.Hour)
	require.NoError(t, app.applyRetentionRestore(msg, "images/back.jpg", "image/jpeg", time.Now()))

	updated := reloadMessage(t, app, msg.ID)
	assert.Equal(t, "images/back.jpg", updated.MediaURL)
	assert.NotContains(t, updated.Metadata, retentionKeepUntilKey)
	assert.NotEmpty(t, updated.Metadata[retentionRestoredAtKey])
}
