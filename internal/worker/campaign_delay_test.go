package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResolveCampaignDelayScopeKey_PrefersInstanceID(t *testing.T) {
	instanceID := "instance-123"
	campaignID := uuid.New()

	scope := resolveCampaignDelayScopeKey(instanceID, campaignID)

	assert.Equal(t, instanceID, scope)
	assert.Equal(t, campaignDelayKeyPrefix+instanceID, campaignDelayRedisKey(scope))
}

func TestResolveCampaignDelayScopeKey_FallsBackToCampaignID(t *testing.T) {
	campaignID := uuid.New()

	scope := resolveCampaignDelayScopeKey("   ", campaignID)

	assert.Equal(t, campaignID.String(), scope)
}

func TestResolveCampaignDelayScopeKey_DefaultWhenMissingIDs(t *testing.T) {
	scope := resolveCampaignDelayScopeKey("", uuid.Nil)

	assert.Equal(t, "default", scope)
	assert.Equal(t, campaignDelayKeyPrefix+"default", campaignDelayRedisKey(scope))
}

func TestCampaignDelayRedisKey_UsesInstanceScopeAcrossCampaigns(t *testing.T) {
	instanceID := "instance-a"
	campaignOne := uuid.New()
	campaignTwo := uuid.New()

	keyOne := campaignDelayRedisKey(resolveCampaignDelayScopeKey(instanceID, campaignOne))
	keyTwo := campaignDelayRedisKey(resolveCampaignDelayScopeKey(instanceID, campaignTwo))

	assert.Equal(t, keyOne, keyTwo)
	assert.Equal(t, campaignDelayKeyPrefix+instanceID, keyOne)
}
