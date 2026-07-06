package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
)

func legacyDecodeIncomingMessage(msg any) (IncomingTextMessage, error) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return IncomingTextMessage{}, err
	}

	var parsed IncomingTextMessage
	if err := json.Unmarshal(msgBytes, &parsed); err != nil {
		return IncomingTextMessage{}, err
	}

	return parsed, nil
}

func directDecodeIncomingMessage(msg IncomingTextMessage) IncomingTextMessage {
	return msg
}

var benchmarkIncomingMessage = IncomingTextMessage{
	From:      "15551234567",
	ID:        "wamid.HBgMNTU1NTEyMzQ1NjcVAgASGBQzRUE5QTY4N0Q4Q0Y2Q0E3QjQ2AA==",
	Timestamp: "1700000000",
	Type:      "interactive",
	Text: &struct {
		Body string `json:"body"`
	}{
		Body: "hello",
	},
	Interactive: &struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
		ListReply *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"list_reply,omitempty"`
		NFMReply *struct {
			ResponseJSON string `json:"response_json"`
			Body         string `json:"body"`
			Name         string `json:"name"`
		} `json:"nfm_reply,omitempty"`
	}{
		Type: "button_reply",
		ButtonReply: &struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}{
			ID:    "btn_1",
			Title: "Yes",
		},
	},
	Context: &struct {
		From string `json:"from"`
		ID   string `json:"id"`
	}{
		From: "15557654321",
		ID:   "wamid.context123",
	},
}

func BenchmarkDecodeIncomingMessageLegacy(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := legacyDecodeIncomingMessage(benchmarkIncomingMessage); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeIncomingMessageDirect(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = directDecodeIncomingMessage(benchmarkIncomingMessage)
	}
}

func benchmarkWebhookStatuses(total int) []WebhookStatus {
	statuses := make([]WebhookStatus, total)
	for i := 0; i < total; i++ {
		messageIdx := i % 1000
		if i%10 == 0 {
			// Simulate missing records in the DB.
			messageIdx = 5000 + i
		}
		statuses[i] = WebhookStatus{
			ID:     fmt.Sprintf("wamid.%04d", messageIdx),
			Status: "delivered",
		}
	}
	return statuses
}

func benchmarkIncomingMessages(total int) []IncomingTextMessage {
	messages := make([]IncomingTextMessage, total)
	for i := 0; i < total; i++ {
		messageIdx := i % 1200
		messages[i] = IncomingTextMessage{
			ID:   fmt.Sprintf("wamid.in.%04d", messageIdx),
			From: "15551234567",
			Type: "text",
			Text: &struct {
				Body string `json:"body"`
			}{
				Body: "hello",
			},
		}
	}
	return messages
}

func benchmarkExistingSet(total int) map[string]struct{} {
	existing := make(map[string]struct{}, total)
	for i := 0; i < total; i++ {
		existing[fmt.Sprintf("wamid.%04d", i)] = struct{}{}
		existing[fmt.Sprintf("wamid.in.%04d", i)] = struct{}{}
	}
	return existing
}

func legacyStatusLookupCount(statuses []WebhookStatus, existing map[string]struct{}) (lookups, found int) {
	for _, status := range statuses {
		if status.ID == "" {
			continue
		}
		lookups++
		if _, ok := existing[status.ID]; ok {
			found++
		}
	}
	return lookups, found
}

func batchStatusLookupCount(statuses []WebhookStatus, existing map[string]struct{}) (lookups, found int) {
	uniqueIDs := make([]string, 0, len(statuses))
	seenIDs := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		if status.ID == "" {
			continue
		}
		if _, seen := seenIDs[status.ID]; seen {
			continue
		}
		seenIDs[status.ID] = struct{}{}
		uniqueIDs = append(uniqueIDs, status.ID)
	}

	if len(uniqueIDs) > 0 {
		lookups = 1
	}

	foundSet := make(map[string]struct{}, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if _, ok := existing[id]; ok {
			foundSet[id] = struct{}{}
		}
	}

	for _, status := range statuses {
		if _, ok := foundSet[status.ID]; ok {
			found++
		}
	}

	return lookups, found
}

func legacyIncomingDuplicateLookupCount(messages []IncomingTextMessage, existing map[string]struct{}) (lookups, accepted int) {
	for _, message := range messages {
		if message.ID == "" {
			accepted++
			continue
		}

		lookups++
		if _, exists := existing[message.ID]; exists {
			continue
		}

		accepted++
	}

	return lookups, accepted
}

func batchIncomingDuplicateLookupCount(messages []IncomingTextMessage, existing map[string]struct{}) (lookups, accepted int) {
	uniqueIDs := make([]string, 0, len(messages))
	seenIDs := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		if message.ID == "" {
			continue
		}
		if _, seen := seenIDs[message.ID]; seen {
			continue
		}
		seenIDs[message.ID] = struct{}{}
		uniqueIDs = append(uniqueIDs, message.ID)
	}

	if len(uniqueIDs) > 0 {
		lookups = 1
	}

	existingSet := make(map[string]struct{}, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if _, exists := existing[id]; exists {
			existingSet[id] = struct{}{}
		}
	}

	seenInPayload := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		if message.ID == "" {
			accepted++
			continue
		}

		if _, exists := existingSet[message.ID]; exists {
			continue
		}

		if _, seen := seenInPayload[message.ID]; seen {
			continue
		}
		seenInPayload[message.ID] = struct{}{}
		accepted++
	}

	return lookups, accepted
}

func BenchmarkWebhookStatusLookupLegacy(b *testing.B) {
	b.ReportAllocs()

	statuses := benchmarkWebhookStatuses(2000)
	existing := benchmarkExistingSet(800)

	var totalLookups int
	var totalFound int
	for i := 0; i < b.N; i++ {
		lookups, found := legacyStatusLookupCount(statuses, existing)
		totalLookups += lookups
		totalFound += found
	}

	b.ReportMetric(float64(totalLookups)/float64(b.N), "lookups/op")
	if totalFound == 0 {
		b.Fatal("unexpected empty match set")
	}
}

func BenchmarkWebhookStatusLookupBatch(b *testing.B) {
	b.ReportAllocs()

	statuses := benchmarkWebhookStatuses(2000)
	existing := benchmarkExistingSet(800)

	var totalLookups int
	var totalFound int
	for i := 0; i < b.N; i++ {
		lookups, found := batchStatusLookupCount(statuses, existing)
		totalLookups += lookups
		totalFound += found
	}

	b.ReportMetric(float64(totalLookups)/float64(b.N), "lookups/op")
	if totalFound == 0 {
		b.Fatal("unexpected empty match set")
	}
}

func BenchmarkIncomingDuplicateLookupLegacy(b *testing.B) {
	b.ReportAllocs()

	messages := benchmarkIncomingMessages(2000)
	existing := benchmarkExistingSet(1000)

	var totalLookups int
	var totalAccepted int
	for i := 0; i < b.N; i++ {
		lookups, accepted := legacyIncomingDuplicateLookupCount(messages, existing)
		totalLookups += lookups
		totalAccepted += accepted
	}

	b.ReportMetric(float64(totalLookups)/float64(b.N), "lookups/op")
	if totalAccepted == 0 {
		b.Fatal("unexpected empty accepted set")
	}
}

func BenchmarkIncomingDuplicateLookupBatch(b *testing.B) {
	b.ReportAllocs()

	messages := benchmarkIncomingMessages(2000)
	existing := benchmarkExistingSet(1000)

	var totalLookups int
	var totalAccepted int
	for i := 0; i < b.N; i++ {
		lookups, accepted := batchIncomingDuplicateLookupCount(messages, existing)
		totalLookups += lookups
		totalAccepted += accepted
	}

	b.ReportMetric(float64(totalLookups)/float64(b.N), "lookups/op")
	if totalAccepted == 0 {
		b.Fatal("unexpected empty accepted set")
	}
}
