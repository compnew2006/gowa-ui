package worker

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	defaultWorkerScalerInterval = 15 * time.Second
	workerStopTimeout           = 10 * time.Second
	failureFreezeThreshold      = 3
)

type scalerManagedWorker interface {
	Run(ctx context.Context) error
	Close() error
}

type managedWorkerFactory func(orgID uuid.UUID) (scalerManagedWorker, error)

type managedWorkerHandle struct {
	id        string
	worker    scalerManagedWorker
	cancel    context.CancelFunc
	done      chan error
	closeOnce sync.Once
}

func (h *managedWorkerHandle) close() error {
	if h == nil || h.worker == nil {
		return nil
	}

	var closeErr error
	h.closeOnce.Do(func() {
		closeErr = h.worker.Close()
	})
	return closeErr
}

type workerRuntimeEvent struct {
	orgID     uuid.UUID
	workerID  string
	err       error
	timestamp time.Time
}

// TenantWorkerRuntime stores the dynamic worker state for one organization.
type TenantWorkerRuntime struct {
	Workers         []*managedWorkerHandle
	LastScaleUp     time.Time
	LastScaleDown   time.Time
	HealthySince    time.Time
	Frozen          bool
	FreezeReason    string
	FailureStreak   int
	FailureThisTick bool
}

type tenantPlan struct {
	OrganizationID uuid.UUID
	Config         OrganizationWorkerConfig
	Depth          int64
	BacklogRatio   float64
	Desired        int
	Current        int
	Frozen         bool
	StartCount     int
	StopHandles    []*managedWorkerHandle
}

type tenantHealthState struct {
	Healthy bool
	Reason  string
}

// WorkerScaler dynamically grows and shrinks tenant-scoped workers based on queue depth and health.
type WorkerScaler struct {
	cfg             *config.Config
	db              *gorm.DB
	redis           *redis.Client
	log             logf.Logger
	messageProvider provider.MessageProvider
	license         *license.Service
	globalBudget    int
	interval        time.Duration
	now             func() time.Time

	newWorker managedWorkerFactory

	mu       sync.Mutex
	runtimes map[uuid.UUID]*TenantWorkerRuntime
	events   chan workerRuntimeEvent
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWorkerScaler creates a new tenant worker scaler.
func NewWorkerScaler(
	cfg *config.Config,
	db *gorm.DB,
	rdb *redis.Client,
	log logf.Logger,
	messageProvider provider.MessageProvider,
	licenseService *license.Service,
	globalBudget int,
) *WorkerScaler {
	scaler := &WorkerScaler{
		cfg:             cfg,
		db:              db,
		redis:           rdb,
		log:             log,
		messageProvider: messageProvider,
		license:         licenseService,
		globalBudget:    max(0, globalBudget),
		interval:        defaultWorkerScalerInterval,
		now:             time.Now,
		runtimes:        make(map[uuid.UUID]*TenantWorkerRuntime),
		events:          make(chan workerRuntimeEvent, 256),
	}
	scaler.newWorker = func(orgID uuid.UUID) (scalerManagedWorker, error) {
		return New(cfg, db, rdb, log, messageProvider, licenseService, WorkerOptions{
			CampaignOrganizationID: orgID,
			EnableCampaignConsumer: true,
			EnableInboundMedia:     false,
		})
	}
	return scaler
}

// Start begins the reconcile loop and blocks until the context is cancelled.
func (s *WorkerScaler) Start(ctx context.Context) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if s.ctx != nil {
		s.mu.Unlock()
		return fmt.Errorf("worker scaler already started")
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	runCtx := s.ctx
	s.mu.Unlock()

	if err := s.reconcile(runCtx); err != nil && !errors.Is(err, context.Canceled) {
		s.log.Error("Initial worker scaler reconcile failed", "error", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer s.Stop()

	for {
		select {
		case <-runCtx.Done():
			return runCtx.Err()
		case <-ticker.C:
			if err := s.reconcile(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				s.log.Error("Worker scaler reconcile failed", "error", err)
			}
		}
	}
}

// Stop cancels the scaler loop and drains all tenant workers.
func (s *WorkerScaler) Stop() {
	if s == nil {
		return
	}

	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.ctx = nil
	handles := make([]*managedWorkerHandle, 0)
	for _, runtime := range s.runtimes {
		handles = append(handles, runtime.Workers...)
		runtime.Workers = nil
	}
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, handle := range handles {
		s.stopWorkerHandle(handle)
	}
}

func (s *WorkerScaler) reconcile(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.db == nil || s.redis == nil {
		return fmt.Errorf("worker scaler requires database and redis")
	}

	s.handleWorkerEvents()

	orgRows, err := s.loadOrganizations(ctx)
	if err != nil {
		return err
	}
	healthByOrg, err := s.loadTenantHealth(ctx)
	if err != nil {
		return err
	}
	licenseState := license.State{}
	if s.license != nil {
		licenseState = s.license.CurrentState()
	}

	now := s.now().UTC()
	depthByOrg := make(map[uuid.UUID]int64, len(orgRows))
	for _, org := range orgRows {
		depth, depthErr := s.redis.XLen(ctx, queue.CampaignStreamName(org.ID)).Result()
		if depthErr != nil && depthErr != redis.Nil {
			s.log.Warn("Failed to read tenant queue depth", "organization_id", org.ID, "error", depthErr)
			continue
		}
		depthByOrg[org.ID] = depth
	}

	plannedStops := make([]*managedWorkerHandle, 0)
	plannedStarts := make([]uuid.UUID, 0)
	plannedBudgets := make(map[uuid.UUID]int)

	s.mu.Lock()
	for _, org := range orgRows {
		if _, ok := s.runtimes[org.ID]; !ok {
			s.runtimes[org.ID] = &TenantWorkerRuntime{}
		}
	}

	orgSet := make(map[uuid.UUID]struct{}, len(orgRows))
	plans := make([]tenantPlan, 0, len(orgRows))

	for _, org := range orgRows {
		orgSet[org.ID] = struct{}{}
		runtime := s.runtimes[org.ID]
		health := healthByOrg[org.ID]
		cfg := applyLicensedWorkerCap(LoadOrganizationWorkerConfig(org.Settings), licenseState)
		depth := depthByOrg[org.ID]

		if runtime.FailureThisTick {
			runtime.FailureStreak++
			runtime.FailureThisTick = false
		} else {
			runtime.FailureStreak = 0
		}

		if runtime.FailureStreak >= failureFreezeThreshold {
			s.freezeRuntime(runtime, org.ID, "worker_start_failures")
		}

		if !health.Healthy && cfg.PauseOnDisconnect {
			runtime.HealthySince = time.Time{}
			s.freezeRuntime(runtime, org.ID, stringsTrimOr(health.Reason, "whatsapp_disconnected"))
		} else if health.Healthy {
			if runtime.Frozen {
				if runtime.HealthySince.IsZero() {
					runtime.HealthySince = now
				} else if now.Sub(runtime.HealthySince) >= s.interval {
					s.unfreezeRuntime(runtime, org.ID)
				}
			} else {
				runtime.HealthySince = now
			}
		}

		current := len(runtime.Workers)
		desired := 0
		if !runtime.Frozen {
			desired = desiredWorkersForTenant(cfg, depth)
			desired = applyScaleDownCooldown(now, runtime, cfg, depth, desired)
		}

		backlogRatio := 0.0
		if cfg.JobsPerWorker > 0 {
			backlogRatio = float64(depth) / float64(cfg.JobsPerWorker)
		}

		plans = append(plans, tenantPlan{
			OrganizationID: org.ID,
			Config:         cfg,
			Depth:          depth,
			BacklogRatio:   backlogRatio,
			Desired:        desired,
			Current:        current,
			Frozen:         runtime.Frozen,
		})
	}

	for orgID, runtime := range s.runtimes {
		if _, ok := orgSet[orgID]; ok {
			continue
		}
		plannedStops = append(plannedStops, runtime.Workers...)
		delete(s.runtimes, orgID)
	}

	allocated := allocateWorkerBudget(plans, s.globalBudget)

	for i := range plans {
		plan := &plans[i]
		runtime := s.runtimes[plan.OrganizationID]
		plan.Desired = allocated[plan.OrganizationID]
		plannedBudgets[plan.OrganizationID] = plan.Desired

		if plan.Current > plan.Desired {
			excess := plan.Current - plan.Desired
			stopHandles := slices.Clone(runtime.Workers[len(runtime.Workers)-excess:])
			runtime.Workers = runtime.Workers[:len(runtime.Workers)-excess]
			runtime.LastScaleDown = now
			plan.StopHandles = stopHandles
			plannedStops = append(plannedStops, stopHandles...)
			continue
		}

		if plan.Current < plan.Desired && !runtime.Frozen {
			if now.Sub(runtime.LastScaleUp) >= time.Duration(plan.Config.ScaleUpCooldownSeconds)*time.Second {
				plan.StartCount = 1
				runtime.LastScaleUp = now
				plannedStarts = append(plannedStarts, plan.OrganizationID)
			}
		}
	}

	s.mu.Unlock()
	for _, handle := range plannedStops {
		s.stopWorkerHandle(handle)
	}
	for _, orgID := range plannedStarts {
		if err := s.startWorkerForOrganization(orgID); err != nil {
			s.log.Error("Failed to start tenant worker", "organization_id", orgID, "error", err)
			s.recordWorkerFailure(orgID)
		}
	}
	s.mu.Lock()

	for orgID, budget := range plannedBudgets {
		s.log.Debug("Tenant worker allocation", "organization_id", orgID, "desired_workers", budget)
	}
	s.mu.Unlock()

	return nil
}

func (s *WorkerScaler) loadOrganizations(ctx context.Context) ([]models.Organization, error) {
	var orgs []models.Organization
	if err := s.db.WithContext(ctx).
		Select("id", "settings").
		Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("load organizations: %w", err)
	}
	return orgs, nil
}

func (s *WorkerScaler) loadTenantHealth(ctx context.Context) (map[uuid.UUID]tenantHealthState, error) {
	var instances []models.WhatsAppInstance
	if err := s.db.WithContext(ctx).
		Select("organization_id", "status", "send_blocked_until", "send_block_reason").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("load tenant health: %w", err)
	}

	now := s.now().UTC()
	healthByOrg := make(map[uuid.UUID]tenantHealthState)
	for _, instance := range instances {
		state := healthByOrg[instance.OrganizationID]
		if instance.Status == models.InstanceStatusConnected &&
			(instance.SendBlockedUntil == nil || now.After(instance.SendBlockedUntil.UTC())) {
			state.Healthy = true
			state.Reason = ""
			healthByOrg[instance.OrganizationID] = state
			continue
		}
		if state.Healthy {
			continue
		}
		if instance.SendBlockedUntil != nil && now.Before(instance.SendBlockedUntil.UTC()) {
			reason := stringsTrimOr(instance.SendBlockReason, "instance_send_blocked")
			state.Reason = reason
		} else {
			state.Reason = "whatsapp_disconnected"
		}
		healthByOrg[instance.OrganizationID] = state
	}
	return healthByOrg, nil
}

func (s *WorkerScaler) freezeRuntime(runtime *TenantWorkerRuntime, orgID uuid.UUID, reason string) {
	if runtime == nil {
		return
	}
	if runtime.Frozen {
		return
	}
	runtime.Frozen = true
	runtime.FreezeReason = reason
	s.log.Warn("Tenant worker allocation frozen", "organization_id", orgID, "reason", reason)
}

func (s *WorkerScaler) unfreezeRuntime(runtime *TenantWorkerRuntime, orgID uuid.UUID) {
	if runtime == nil || !runtime.Frozen {
		return
	}
	runtime.Frozen = false
	runtime.FreezeReason = ""
	runtime.FailureStreak = 0
	runtime.HealthySince = s.now().UTC()
	s.log.Info("Tenant worker allocation resumed", "organization_id", orgID)
}

func (s *WorkerScaler) startWorkerForOrganization(orgID uuid.UUID) error {
	s.mu.Lock()
	runCtx := s.ctx
	runtime := s.runtimes[orgID]
	if runtime == nil {
		runtime = &TenantWorkerRuntime{}
		s.runtimes[orgID] = runtime
	}
	s.mu.Unlock()

	if runCtx == nil {
		return fmt.Errorf("worker scaler is not running")
	}

	workerInstance, err := s.newWorker(orgID)
	if err != nil {
		return err
	}

	workerCtx, cancel := context.WithCancel(runCtx)
	handle := &managedWorkerHandle{
		id:     uuid.NewString(),
		worker: workerInstance,
		cancel: cancel,
		done:   make(chan error, 1),
	}

	s.mu.Lock()
	runtime.Workers = append(runtime.Workers, handle)
	s.mu.Unlock()

	go func() {
		err := workerInstance.Run(workerCtx)
		_ = handle.close()
		handle.done <- err
		close(handle.done)

		select {
		case s.events <- workerRuntimeEvent{
			orgID:     orgID,
			workerID:  handle.id,
			err:       err,
			timestamp: s.now().UTC(),
		}:
		default:
			s.log.Warn("Dropping worker runtime event due to full channel", "organization_id", orgID, "worker_id", handle.id)
		}
	}()

	return nil
}

func (s *WorkerScaler) stopWorkerHandle(handle *managedWorkerHandle) {
	if handle == nil {
		return
	}

	handle.cancel()
	select {
	case <-handle.done:
	case <-time.After(workerStopTimeout):
		s.log.Warn("Timed out waiting for tenant worker to stop", "worker_id", handle.id)
	}
	_ = handle.close()
}

func (s *WorkerScaler) handleWorkerEvents() {
	for {
		select {
		case event := <-s.events:
			s.mu.Lock()
			runtime := s.runtimes[event.orgID]
			if runtime != nil {
				for idx, handle := range runtime.Workers {
					if handle.id != event.workerID {
						continue
					}
					runtime.Workers = append(runtime.Workers[:idx], runtime.Workers[idx+1:]...)
					break
				}
				if event.err != nil && !errors.Is(event.err, context.Canceled) {
					runtime.FailureThisTick = true
				}
			}
			s.mu.Unlock()
		default:
			return
		}
	}
}

func (s *WorkerScaler) recordWorkerFailure(orgID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runtime := s.runtimes[orgID]
	if runtime == nil {
		runtime = &TenantWorkerRuntime{}
		s.runtimes[orgID] = runtime
	}
	runtime.FailureThisTick = true
}

func desiredWorkersForTenant(cfg OrganizationWorkerConfig, depth int64) int {
	if depth <= 0 {
		return 0
	}

	desired := int((depth + int64(cfg.JobsPerWorker) - 1) / int64(cfg.JobsPerWorker))
	if desired < cfg.MinWorkers {
		desired = cfg.MinWorkers
	}
	if desired > cfg.MaxWorkers {
		desired = cfg.MaxWorkers
	}
	return desired
}

func applyScaleDownCooldown(now time.Time, runtime *TenantWorkerRuntime, cfg OrganizationWorkerConfig, depth int64, desired int) int {
	if runtime == nil || depth > 0 {
		return desired
	}
	lastChange := runtime.LastScaleUp
	if runtime.LastScaleDown.After(lastChange) {
		lastChange = runtime.LastScaleDown
	}
	if lastChange.IsZero() {
		return desired
	}
	if now.Sub(lastChange) < time.Duration(cfg.ScaleDownCooldownSeconds)*time.Second {
		return len(runtime.Workers)
	}
	return desired
}

func allocateWorkerBudget(plans []tenantPlan, budget int) map[uuid.UUID]int {
	allocation := make(map[uuid.UUID]int, len(plans))
	if budget <= 0 {
		return allocation
	}

	totalDesired := 0
	for _, plan := range plans {
		totalDesired += plan.Desired
	}
	if totalDesired <= budget {
		for _, plan := range plans {
			allocation[plan.OrganizationID] = plan.Desired
		}
		return allocation
	}

	ordered := slices.Clone(plans)
	slices.SortFunc(ordered, func(a, b tenantPlan) int {
		switch {
		case a.BacklogRatio > b.BacklogRatio:
			return -1
		case a.BacklogRatio < b.BacklogRatio:
			return 1
		case a.Depth > b.Depth:
			return -1
		case a.Depth < b.Depth:
			return 1
		default:
			return stringsCompareUUID(a.OrganizationID, b.OrganizationID)
		}
	})

	remaining := budget
	for _, plan := range ordered {
		preserved := min(plan.Current, plan.Desired)
		if preserved <= 0 || remaining <= 0 {
			continue
		}
		if preserved > remaining {
			preserved = remaining
		}
		allocation[plan.OrganizationID] = preserved
		remaining -= preserved
	}

	if remaining <= 0 {
		return allocation
	}

	for _, plan := range ordered {
		current := allocation[plan.OrganizationID]
		for current < plan.Desired && remaining > 0 {
			current++
			remaining--
		}
		allocation[plan.OrganizationID] = current
		if remaining == 0 {
			break
		}
	}

	return allocation
}

func stringsTrimOr(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func stringsCompareUUID(a, b uuid.UUID) int {
	switch {
	case a.String() < b.String():
		return -1
	case a.String() > b.String():
		return 1
	default:
		return 0
	}
}
