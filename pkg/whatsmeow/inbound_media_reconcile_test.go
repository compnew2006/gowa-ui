package whatsmeow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
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
