package worker

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/test/testutil"
)

func TestNewLeavesDisabledCampaignConsumerNil(t *testing.T) {
	t.Parallel()

	client := setupScalerRedis(t)
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			BaseURL: "https://graph.facebook.com",
		},
	}

	w, err := New(cfg, nil, client, testutil.NopLogger(), nil, nil, WorkerOptions{
		EnableCampaignConsumer: false,
		EnableInboundMedia:     true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.Consumer != nil {
		t.Fatal("Consumer should be nil when campaign consumer is disabled")
	}
	if w.InboundConsumer == nil {
		t.Fatal("InboundConsumer should be initialized when inbound media is enabled")
	}
}

func TestNewLeavesDisabledInboundConsumerNil(t *testing.T) {
	t.Parallel()

	client := setupScalerRedis(t)
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			BaseURL: "https://graph.facebook.com",
		},
	}

	w, err := New(cfg, nil, client, testutil.NopLogger(), nil, nil, WorkerOptions{
		EnableCampaignConsumer: true,
		EnableInboundMedia:     false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.Consumer == nil {
		t.Fatal("Consumer should be initialized when campaign consumer is enabled")
	}
	if w.InboundConsumer != nil {
		t.Fatal("InboundConsumer should be nil when inbound media is disabled")
	}
}
