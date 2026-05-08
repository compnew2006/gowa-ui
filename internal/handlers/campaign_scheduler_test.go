package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCampaignSchedulerBatchSize(t *testing.T) {
	assert.Equal(t, 100, campaignSchedulerBatchSize)
}

func TestCampaignSchedulerNewNil(t *testing.T) {
	scheduler := NewCampaignScheduler(nil, 0)
	assert.Nil(t, scheduler.app)
	assert.Equal(t, time.Duration(0), scheduler.interval)
}

func TestCampaignSchedulerStopIdempotent(t *testing.T) {
	scheduler := &CampaignScheduler{}
	scheduler.Stop()
	assert.Nil(t, scheduler.ticker)
}
