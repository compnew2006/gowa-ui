package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingManagedWorker struct{}

func (w *blockingManagedWorker) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (w *blockingManagedWorker) Close() error {
	return nil
}

func setupScalerRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestLoadOrganizationWorkerConfig_Defaults(t *testing.T) {
	t.Parallel()

	config := LoadOrganizationWorkerConfig(nil)
	assert.Equal(t, DefaultOrganizationWorkerConfig(), config)
}

func TestApplyLicensedWorkerCap(t *testing.T) {
	t.Parallel()

	config := OrganizationWorkerConfig{
		MinWorkers: 3,
		MaxWorkers: 8,
	}

	capped := applyLicensedWorkerCap(config, license.State{
		MaxWorkersPerOrg: 2,
	})

	assert.Equal(t, 2, capped.MinWorkers)
	assert.Equal(t, 2, capped.MaxWorkers)
}

func TestAllocateWorkerBudget_PrefersBusyTenants(t *testing.T) {
	t.Parallel()

	orgA := uuid.New()
	orgB := uuid.New()
	plans := []tenantPlan{
		{OrganizationID: orgA, Desired: 3, Current: 2, Depth: 90, BacklogRatio: 3.6},
		{OrganizationID: orgB, Desired: 3, Current: 2, Depth: 30, BacklogRatio: 1.2},
	}

	allocation := allocateWorkerBudget(plans, 3)
	assert.Equal(t, 2, allocation[orgA])
	assert.Equal(t, 1, allocation[orgB])
}

func TestApplyScaleDownCooldown_KeepsWarmWorkersUntilCooldownExpires(t *testing.T) {
	t.Parallel()

	cfg := DefaultOrganizationWorkerConfig()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	runtime := &TenantWorkerRuntime{
		Workers:       []*managedWorkerHandle{{id: "w1"}},
		LastScaleUp:   now.Add(-30 * time.Second),
		LastScaleDown: time.Time{},
	}

	desired := applyScaleDownCooldown(now, runtime, cfg, 0, 0)
	assert.Equal(t, 1, desired)

	desired = applyScaleDownCooldown(now.Add(2*time.Minute), runtime, cfg, 0, 0)
	assert.Equal(t, 0, desired)
}

func TestWorkerScaler_FreezesDisconnectedTenantAndResumesAfterRecovery(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	client := setupScalerRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Scaler Freeze Org",
		Slug:      "scaler-freeze-" + uuid.NewString()[:8],
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(org).Error)

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Tenant Instance",
		Status:         models.InstanceStatusDisconnected,
	}
	require.NoError(t, db.Create(instance).Error)

	jobQueue := queue.NewRedisQueue(client, log)
	require.NoError(t, jobQueue.EnqueueRecipient(ctx, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: org.ID,
		PhoneNumber:    "201234567890",
		RecipientName:  "Tenant User",
	}))

	scaler := NewWorkerScaler(nil, db, client, log, nil, nil, 4)
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	scaler.now = func() time.Time { return now }

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scaler.ctx = runCtx
	scaler.cancel = cancel

	started := 0
	scaler.newWorker = func(orgID uuid.UUID) (scalerManagedWorker, error) {
		started++
		return &blockingManagedWorker{}, nil
	}
	defer scaler.Stop()

	require.NoError(t, scaler.reconcile(ctx))
	assert.Equal(t, 0, started)
	runtime := scaler.runtimes[org.ID]
	require.NotNil(t, runtime)
	assert.True(t, runtime.Frozen)

	require.NoError(t, db.Model(instance).Update("status", models.InstanceStatusConnected).Error)

	now = now.Add(5 * time.Second)
	require.NoError(t, scaler.reconcile(ctx))
	assert.Equal(t, 0, started)
	assert.True(t, runtime.Frozen)

	now = now.Add(scaler.interval)
	require.NoError(t, scaler.reconcile(ctx))
	assert.Equal(t, 1, started)
	assert.False(t, runtime.Frozen)
}

func TestWorkerScaler_FreezesAfterRepeatedStartFailures(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	client := setupScalerRedis(t)
	log := testutil.NopLogger()
	ctx := context.Background()

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Scaler Fail Org",
		Slug:      "scaler-fail-" + uuid.NewString()[:8],
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(org).Error)

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Healthy Instance",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, db.Create(instance).Error)

	jobQueue := queue.NewRedisQueue(client, log)
	require.NoError(t, jobQueue.EnqueueRecipient(ctx, &queue.RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: org.ID,
		PhoneNumber:    "201234567890",
		RecipientName:  "Tenant User",
	}))

	scaler := NewWorkerScaler(nil, db, client, log, nil, nil, 2)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scaler.ctx = runCtx
	scaler.cancel = cancel

	attempts := 0
	scaler.newWorker = func(orgID uuid.UUID) (scalerManagedWorker, error) {
		attempts++
		return nil, fmt.Errorf("boom")
	}

	require.NoError(t, scaler.reconcile(ctx))
	require.NoError(t, scaler.reconcile(ctx))
	require.NoError(t, scaler.reconcile(ctx))
	require.NoError(t, scaler.reconcile(ctx))

	runtime := scaler.runtimes[org.ID]
	require.NotNil(t, runtime)
	assert.True(t, runtime.Frozen)
	assert.Equal(t, "worker_start_failures", runtime.FreezeReason)
	assert.Equal(t, 3, attempts)
}
