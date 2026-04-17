package whatsmeow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestInboundMediaQueueGroupStateValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		state       inboundMediaQueueGroupState
		allowActive bool
		wantErr     bool
	}{
		{
			name:    "missing group",
			state:   inboundMediaQueueGroupState{},
			wantErr: true,
		},
		{
			name: "pending jobs with zero lag do not block reconcile",
			state: inboundMediaQueueGroupState{
				Found:   true,
				Pending: 1,
				Lag:     0,
			},
			wantErr: false,
		},
		{
			name: "lag blocks reconcile",
			state: inboundMediaQueueGroupState{
				Found:   true,
				Pending: 0,
				Lag:     3,
			},
			wantErr: true,
		},
		{
			name: "unknown lag blocks reconcile",
			state: inboundMediaQueueGroupState{
				Found:   true,
				Pending: 0,
				Lag:     -1,
			},
			wantErr: true,
		},
		{
			name: "idle queue allows reconcile",
			state: inboundMediaQueueGroupState{
				Found:   true,
				Pending: 0,
				Lag:     0,
			},
			wantErr: false,
		},
		{
			name: "force active queue bypasses safety check",
			state: inboundMediaQueueGroupState{
				Found:   true,
				Pending: 8,
				Lag:     4,
			},
			allowActive: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.state.validate(tt.allowActive)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestBuildStaleInboundMediaFailure(t *testing.T) {
	t.Parallel()

	t.Run("preserves the last known recovery error", func(t *testing.T) {
		t.Parallel()

		metadata := models.JSONB{
			inboundMediaAsyncStatusKey:       inboundMediaAsyncStatusQueued,
			inboundMediaAsyncLastErrorKey:    "create media directory: permission denied",
			inboundMediaAsyncRecoveredAtKey:  "2026-04-06T19:31:00Z",
			inboundMediaAsyncEnqueueErrorKey: "old enqueue error",
		}

		nextMetadata, reason, errorMessage := buildStaleInboundMediaFailure(metadata)

		if reason != "create media directory: permission denied" {
			t.Fatalf("unexpected reason: %q", reason)
		}
		if nextMetadata[inboundMediaAsyncStatusKey] != inboundMediaAsyncStatusFailed {
			t.Fatalf("expected failed status, got %#v", nextMetadata[inboundMediaAsyncStatusKey])
		}
		if nextMetadata[inboundMediaAsyncLastErrorKey] != reason {
			t.Fatalf("expected last error to be preserved")
		}
		if nextMetadata[inboundMediaAsyncRecoveredAtKey] != nil {
			t.Fatalf("expected recovered_at to be cleared")
		}
		if _, exists := nextMetadata[inboundMediaAsyncEnqueueErrorKey]; exists {
			t.Fatalf("expected enqueue error marker to be removed")
		}
		if errorMessage != "Inbound media async recovery failed: create media directory: permission denied" {
			t.Fatalf("unexpected error message: %q", errorMessage)
		}
	})

	t.Run("falls back to the stale queue reason when last error is missing", func(t *testing.T) {
		t.Parallel()

		nextMetadata, reason, errorMessage := buildStaleInboundMediaFailure(models.JSONB{
			inboundMediaAsyncStatusKey: inboundMediaAsyncStatusQueued,
		})

		if reason != staleInboundMediaQueueReason {
			t.Fatalf("unexpected fallback reason: %q", reason)
		}
		if nextMetadata[inboundMediaAsyncLastErrorKey] != staleInboundMediaQueueReason {
			t.Fatalf("expected fallback reason to be written into metadata")
		}
		if errorMessage != "Inbound media async recovery failed: "+staleInboundMediaQueueReason {
			t.Fatalf("unexpected error message: %q", errorMessage)
		}
	})
}

func TestLoadActivePendingInboundMediaMessageIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	firstJob := queue.InboundMediaJob{
		MessageID:      uuid.New(),
		OrganizationID: uuid.New(),
		InstanceID:     uuid.New(),
		MessageType:    models.MessageTypeImage,
		MediaKind:      "image",
		MimeType:       "image/png",
		EnqueuedAt:     time.Now().UTC(),
	}
	secondJob := queue.InboundMediaJob{
		MessageID:      uuid.New(),
		OrganizationID: uuid.New(),
		InstanceID:     uuid.New(),
		MessageType:    models.MessageTypeDocument,
		MediaKind:      "document",
		MimeType:       "application/pdf",
		EnqueuedAt:     time.Now().UTC(),
	}

	addJob := func(job queue.InboundMediaJob) string {
		payload, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		streamID, err := client.XAdd(ctx, &redis.XAddArgs{
			Stream: queue.InboundMediaStreamName,
			Values: map[string]any{
				"type":    string(queue.JobTypeInboundMedia),
				"payload": string(payload),
			},
		}).Result()
		if err != nil {
			t.Fatalf("xadd inbound media job: %v", err)
		}
		return streamID
	}

	firstStreamID := addJob(firstJob)
	secondStreamID := addJob(secondJob)

	if err := client.XGroupCreateMkStream(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("xgroup create: %v", err)
	}

	messages, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    queue.InboundMediaConsumerGroup,
		Consumer: "test-consumer",
		Streams:  []string{queue.InboundMediaStreamName, ">"},
		Count:    2,
	}).Result()
	if err != nil {
		t.Fatalf("xreadgroup: %v", err)
	}
	if len(messages) != 1 || len(messages[0].Messages) != 2 {
		t.Fatalf("expected 2 pending messages, got %#v", messages)
	}

	activeIDs, err := loadActivePendingInboundMediaMessageIDs(ctx, client, inboundMediaQueueGroupState{
		Found:   true,
		Pending: 2,
		Lag:     0,
	})
	if err != nil {
		t.Fatalf("load active pending ids: %v", err)
	}

	if len(activeIDs) != 2 {
		t.Fatalf("expected 2 active pending ids, got %d", len(activeIDs))
	}
	if _, ok := activeIDs[firstJob.MessageID]; !ok {
		t.Fatalf("expected first message id %s to be active", firstJob.MessageID)
	}
	if _, ok := activeIDs[secondJob.MessageID]; !ok {
		t.Fatalf("expected second message id %s to be active", secondJob.MessageID)
	}

	for _, streamID := range []string{firstStreamID, secondStreamID} {
		if err := client.XAck(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup, streamID).Err(); err != nil {
			t.Fatalf("xack %s: %v", streamID, err)
		}
	}
}

func TestFilterQueuedInboundMediaRows(t *testing.T) {
	t.Parallel()

	keepFirst := inboundMediaQueuedMessage{ID: uuid.New()}
	skip := inboundMediaQueuedMessage{ID: uuid.New()}
	keepLast := inboundMediaQueuedMessage{ID: uuid.New()}

	filtered, skipped := filterQueuedInboundMediaRows(
		[]inboundMediaQueuedMessage{keepFirst, skip, keepLast},
		map[uuid.UUID]struct{}{skip.ID: {}},
	)

	if skipped != 1 {
		t.Fatalf("expected 1 skipped row, got %d", skipped)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered rows, got %d", len(filtered))
	}
	if filtered[0].ID != keepFirst.ID || filtered[1].ID != keepLast.ID {
		t.Fatalf("unexpected filtered order: %#v", filtered)
	}
}

func TestReconcileStaleQueuedInboundMedia_RequeuesRecoverableRows(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	if err := client.XGroupCreateMkStream(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("xgroup create: %v", err)
	}

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Inbound Reconcile Org",
		Slug:      "inbound-reconcile-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Inbound Reconcile Instance",
		Settings:       models.JSONB{},
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550009999",
		ProfileName:    "Recoverable Contact",
		Metadata:       models.JSONB{},
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("create contact: %v", err)
	}

	messageID := uuid.New()
	staleTime := time.Now().UTC().Add(-time.Hour)
	job := &queue.InboundMediaJob{
		MessageID:          messageID,
		OrganizationID:     org.ID,
		InstanceID:         instance.ID,
		WhatsAppMessageID:  "wamid.requeue.1",
		MessageType:        models.MessageTypeDocument,
		MediaKind:          "document",
		MimeType:           "application/pdf",
		FallbackFilename:   "report.pdf",
		MediaPayloadBase64: "R0lGODlhAQABAIAAAAUEBA==",
		LastError:          "client is nil",
		EnqueuedAt:         staleTime,
	}

	metadata := models.JSONB{
		inboundMediaAsyncStatusKey:      inboundMediaAsyncStatusQueued,
		inboundMediaAsyncEnqueuedAtKey:  staleTime.Format(time.RFC3339Nano),
		inboundMediaAsyncLastErrorKey:   job.LastError,
		inboundMediaAsyncRecoveredAtKey: nil,
	}
	setInboundMediaAsyncJobMetadata(metadata, job)

	message := models.Message{
		BaseModel:         models.BaseModel{ID: messageID, CreatedAt: staleTime, UpdatedAt: staleTime},
		OrganizationID:    org.ID,
		InstanceID:        &instance.ID,
		WhatsAppAccount:   "instance-account",
		ContactID:         contact.ID,
		WhatsAppMessageID: job.WhatsAppMessageID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeDocument,
		MediaMimeType:     "application/pdf",
		MediaFilename:     "report.pdf",
		Status:            models.MessageStatusReceived,
		Metadata:          metadata,
	}
	if err := db.Create(&message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	summary, err := ReconcileStaleQueuedInboundMedia(
		ctx,
		db,
		client,
		InboundMediaReconcileOptions{
			Apply:     true,
			OlderThan: 15 * time.Minute,
			Now:       time.Now().UTC(),
		},
		testutil.NopLogger(),
	)
	if err != nil {
		t.Fatalf("reconcile stale inbound media: %v", err)
	}
	if summary.Requeued != 1 {
		t.Fatalf("expected 1 requeued row, got %d", summary.Requeued)
	}
	if summary.MarkedFailed != 0 {
		t.Fatalf("expected 0 failed rows, got %d", summary.MarkedFailed)
	}

	streams, err := client.XRangeN(ctx, queue.InboundMediaStreamName, "-", "+", 10).Result()
	if err != nil {
		t.Fatalf("xrange inbound_media: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("expected 1 requeued redis job, got %d", len(streams))
	}

	payload, ok := streams[0].Values["payload"].(string)
	if !ok {
		t.Fatalf("expected payload string, got %#v", streams[0].Values["payload"])
	}
	var queuedJob queue.InboundMediaJob
	if err := json.Unmarshal([]byte(payload), &queuedJob); err != nil {
		t.Fatalf("decode queued payload: %v", err)
	}
	if queuedJob.MessageID != messageID {
		t.Fatalf("queued message id mismatch: got %s want %s", queuedJob.MessageID, messageID)
	}

	var saved models.Message
	if err := db.First(&saved, "id = ?", messageID).Error; err != nil {
		t.Fatalf("reload message: %v", err)
	}
	if saved.Metadata[inboundMediaAsyncStatusKey] != inboundMediaAsyncStatusQueued {
		t.Fatalf("expected queued status, got %#v", saved.Metadata[inboundMediaAsyncStatusKey])
	}
	if saved.ErrorMessage == "" {
		t.Fatal("expected message error to explain stale requeue")
	}
}

func TestReconcileStaleQueuedInboundMedia_MarksFailedWithoutStoredJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	if err := client.XGroupCreateMkStream(ctx, queue.InboundMediaStreamName, queue.InboundMediaConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("xgroup create: %v", err)
	}

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Inbound Reconcile Fail Org",
		Slug:      "inbound-reconcile-fail-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Inbound Reconcile Fail Instance",
		Settings:       models.JSONB{},
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550008888",
		ProfileName:    "Unrecoverable Contact",
		Metadata:       models.JSONB{},
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("create contact: %v", err)
	}

	staleTime := time.Now().UTC().Add(-time.Hour)
	message := models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: staleTime, UpdatedAt: staleTime},
		OrganizationID:    org.ID,
		InstanceID:        &instance.ID,
		WhatsAppAccount:   "instance-account",
		ContactID:         contact.ID,
		WhatsAppMessageID: "wamid.requeue.fail.1",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeDocument,
		MediaMimeType:     "application/pdf",
		MediaFilename:     "report.pdf",
		Status:            models.MessageStatusReceived,
		Metadata: models.JSONB{
			inboundMediaAsyncStatusKey:     inboundMediaAsyncStatusQueued,
			inboundMediaAsyncLastErrorKey:  "download failed with status code 403",
			inboundMediaAsyncEnqueuedAtKey: staleTime.Format(time.RFC3339Nano),
		},
	}
	if err := db.Create(&message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	summary, err := ReconcileStaleQueuedInboundMedia(
		ctx,
		db,
		client,
		InboundMediaReconcileOptions{
			Apply:     true,
			OlderThan: 15 * time.Minute,
			Now:       time.Now().UTC(),
		},
		testutil.NopLogger(),
	)
	if err != nil {
		t.Fatalf("reconcile stale inbound media: %v", err)
	}
	if summary.MarkedFailed != 1 {
		t.Fatalf("expected 1 failed row, got %d", summary.MarkedFailed)
	}
	if summary.Requeued != 0 {
		t.Fatalf("expected 0 requeued rows, got %d", summary.Requeued)
	}

	var saved models.Message
	if err := db.First(&saved, "id = ?", message.ID).Error; err != nil {
		t.Fatalf("reload message: %v", err)
	}
	if saved.Metadata[inboundMediaAsyncStatusKey] != inboundMediaAsyncStatusFailed {
		t.Fatalf("expected failed status, got %#v", saved.Metadata[inboundMediaAsyncStatusKey])
	}
	if saved.ErrorMessage == "" {
		t.Fatal("expected failure error message to be set")
	}
}
