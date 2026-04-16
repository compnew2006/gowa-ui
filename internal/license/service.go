package license

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	StatusDisabled   = "disabled"
	StatusUnlicensed = "unlicensed"
	StatusActive     = "active"
	StatusGrace      = "grace"
	StatusLocked     = "locked"

	ResourceOrganizations = "organizations"
	ResourceUsers         = "users"
	ResourceEndpoints     = "whatsapp_endpoints"

	redisStateKey          = "whatomate:license:state"
	redisInvalidateChannel = "whatomate:license:invalidate"
	refreshInterval        = time.Minute
	expiringSoonDays       = 14
)

type State struct {
	Enabled                    bool           `json:"enabled"`
	Status                     string         `json:"status"`
	Locked                     bool           `json:"locked"`
	Reason                     string         `json:"reason,omitempty"`
	LicenseID                  string         `json:"license_id,omitempty"`
	LicenseFamilyID            string         `json:"license_family_id,omitempty"`
	Revision                   uint64         `json:"revision"`
	KeyID                      string         `json:"key_id,omitempty"`
	HWIDFull                   string         `json:"hwid_full"`
	HWIDShort                  string         `json:"hwid_short"`
	HWIDHash                   string         `json:"hwid_hash"`
	Tier                       string         `json:"tier,omitempty"`
	LicenseKind                string         `json:"license_kind,omitempty"`
	TrialDays                  int            `json:"trial_days,omitempty"`
	DurationLabel              string         `json:"duration_label,omitempty"`
	MaxOrganizations           int            `json:"max_organizations"`
	MaxUsersPerOrg             int            `json:"max_users_per_org"`
	MaxWhatsAppEndpointsPerOrg int            `json:"max_whatsapp_endpoints_per_org"`
	MaxWorkers                 int            `json:"max_workers"`
	ExpiresAt                  *time.Time     `json:"expires_at,omitempty"`
	GraceDeadline              *time.Time     `json:"grace_deadline,omitempty"`
	DaysUntilExpiry            *int           `json:"days_until_expiry,omitempty"`
	ExpiringSoon               bool           `json:"expiring_soon"`
	QuotaOverages              map[string]int `json:"quota_overages,omitempty"`
	UpdatedAt                  time.Time      `json:"updated_at"`
}

type MetricUsage struct {
	Current   int  `json:"current"`
	Limit     int  `json:"limit"`
	OverQuota bool `json:"over_quota"`
	Overage   int  `json:"overage"`
}

type OrganizationUsage struct {
	OrganizationID        uuid.UUID `json:"organization_id"`
	OrganizationName      string    `json:"organization_name"`
	UserCount             int       `json:"user_count"`
	WhatsAppEndpointCount int       `json:"whatsapp_endpoint_count"`
}

type UsageSnapshot struct {
	Organizations           MetricUsage         `json:"organizations"`
	UsersPerOrg             MetricUsage         `json:"users_per_org"`
	WhatsAppEndpointsPerOrg MetricUsage         `json:"whatsapp_endpoints_per_org"`
	OrganizationDetails     []OrganizationUsage `json:"organization_details"`
}

type BootstrapResponse struct {
	State
	Usage UsageSnapshot `json:"usage"`
}

type QuotaCheck struct {
	Allowed   bool   `json:"allowed"`
	Resource  string `json:"resource"`
	Current   int    `json:"current"`
	Limit     int    `json:"limit"`
	OverQuota bool   `json:"over_quota"`
}

type ActivationError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *ActivationError) Error() string {
	return e.Message
}

type Service struct {
	cfg     *config.Config
	db      *gorm.DB
	redis   *redis.Client
	log     logf.Logger
	keyRing map[string]ed25519.PublicKey
	now     func() time.Time

	hwidFull  string
	hwidShort string
	hwidHash  string

	state atomic.Pointer[State]
	once  sync.Once
}

func NewService(cfg *config.Config, db *gorm.DB, redisClient *redis.Client, log logf.Logger) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if err := config.ValidateLicenseConfig(cfg); err != nil {
		return nil, err
	}

	full, short, hash, err := BuildHWID(&cfg.License, log)
	if err != nil {
		return nil, err
	}

	keyRing, err := buildKeyRing(cfg)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		cfg:       cfg,
		db:        db,
		redis:     redisClient,
		log:       log,
		keyRing:   keyRing,
		now:       time.Now,
		hwidFull:  full,
		hwidShort: short,
		hwidHash:  hash,
	}

	if !cfg.License.Enabled {
		svc.storeState(State{
			Enabled:       false,
			Status:        StatusDisabled,
			Locked:        false,
			HWIDFull:      full,
			HWIDShort:     short,
			HWIDHash:      hash,
			UpdatedAt:     svc.now().UTC(),
			QuotaOverages: map[string]int{},
		})
		return svc, nil
	}

	if len(keyRing) == 0 {
		return nil, fmt.Errorf("license is enabled but no usable public keys are configured or embedded")
	}
	if db == nil {
		return nil, fmt.Errorf("license is enabled but database is nil")
	}

	if _, err := svc.RefreshState(context.Background()); err != nil {
		return nil, err
	}

	return svc, nil
}

func buildKeyRing(cfg *config.Config) (map[string]ed25519.PublicKey, error) {
	embedded, err := ParseEmbeddedKeyRing()
	if err != nil {
		return nil, err
	}
	override := strings.TrimSpace(cfg.License.PublicKey)
	if override == "" {
		return embedded, nil
	}
	if !cfg.License.AllowUnsafePublicKeyOverride {
		if strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production") {
			return nil, fmt.Errorf("license.public_key override is disabled in production unless license.allow_unsafe_public_key_override=true")
		}
		return nil, fmt.Errorf("license.public_key override requires license.allow_unsafe_public_key_override=true outside production")
	}

	overrideKey, err := DecodePublicKey(override)
	if err != nil {
		return nil, err
	}
	if embedded == nil {
		embedded = make(map[string]ed25519.PublicKey)
	}
	kid := strings.TrimSpace(cfg.License.PublicKeyKID)
	if kid == "" {
		kid = "config-override"
	}
	embedded[kid] = overrideKey
	return embedded, nil
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || !s.cfg.License.Enabled {
		return
	}
	s.once.Do(func() {
		go s.runPeriodicRefresh(ctx)
		if s.redis != nil {
			go s.runInvalidationSubscriber(ctx)
		}
	})
}

func (s *Service) runPeriodicRefresh(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RefreshState(ctx); err != nil {
				s.log.Warn("License state refresh failed", "error", err)
			}
		}
	}
}

func (s *Service) runInvalidationSubscriber(ctx context.Context) {
	pubsub := s.redis.Subscribe(ctx, redisInvalidateChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if _, err := s.RefreshState(ctx); err != nil {
				s.log.Warn("License invalidation refresh failed", "error", err)
			}
		}
	}
}

func (s *Service) CurrentState() State {
	ptr := s.state.Load()
	if ptr == nil {
		return State{
			Enabled:       s.cfg != nil && s.cfg.License.Enabled,
			Status:        StatusUnlicensed,
			Locked:        s.cfg != nil && s.cfg.License.Enabled,
			HWIDFull:      s.hwidFull,
			HWIDShort:     s.hwidShort,
			HWIDHash:      s.hwidHash,
			UpdatedAt:     s.now().UTC(),
			QuotaOverages: map[string]int{},
		}
	}
	return cloneState(*ptr)
}

func (s *Service) IsLocked() bool {
	return s.cfg != nil && s.cfg.License.Enabled && s.CurrentState().Locked
}

func (s *Service) RequiresQuotaCleanup() bool {
	if s == nil || s.cfg == nil || !s.cfg.License.Enabled {
		return false
	}
	state := s.CurrentState()
	return !state.Locked && len(state.QuotaOverages) > 0
}

func (s *Service) BlockValueDelivery() bool {
	return s.IsLocked() || s.RequiresQuotaCleanup()
}

func (s *Service) WorkersEnforced() bool {
	if s == nil || s.cfg == nil || s.cfg.License.EnforceOnWorkers == nil {
		return true
	}
	return *s.cfg.License.EnforceOnWorkers
}

func (s *Service) WaitUntilOperational(ctx context.Context) error {
	if s == nil || !s.cfg.License.Enabled || !s.WorkersEnforced() {
		return nil
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		if !s.BlockValueDelivery() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := s.RefreshState(ctx); err != nil {
				s.log.Warn("Failed to refresh license state while waiting", "error", err)
			}
		}
	}
}

func (s *Service) Bootstrap(ctx context.Context) (BootstrapResponse, error) {
	if s == nil {
		return BootstrapResponse{}, fmt.Errorf("license service is nil")
	}
	state, err := s.RefreshState(ctx)
	if err != nil {
		return BootstrapResponse{}, err
	}
	usage, err := s.computeUsage(ctx, state)
	if err != nil {
		return BootstrapResponse{}, err
	}
	return BootstrapResponse{
		State: state,
		Usage: usage,
	}, nil
}

func (s *Service) Activate(ctx context.Context, rawToken string) (BootstrapResponse, error) {
	if !s.cfg.License.Enabled {
		return BootstrapResponse{}, &ActivationError{StatusCode: 400, Code: "license_disabled", Message: "Licensing is disabled"}
	}
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return BootstrapResponse{}, &ActivationError{StatusCode: 400, Code: "missing_token", Message: "Security key is required"}
	}

	now := s.now().UTC()
	claims, kid, err := VerifyToken(token, s.keyRing, now)
	if err != nil {
		return BootstrapResponse{}, mapActivationError(err)
	}
	if claims.HWIDHash != s.hwidHash {
		return BootstrapResponse{}, &ActivationError{StatusCode: 403, Code: "hwid_mismatch", Message: "The provided security key does not match this server HWID"}
	}
	if claims.NotBefore != nil && now.Before(claims.NotBefore.Time) {
		return BootstrapResponse{}, &ActivationError{StatusCode: 403, Code: "not_yet_valid", Message: "The provided security key is not valid yet"}
	}
	if claims.ExpiresAt != nil && now.After(claims.ExpiresAt.Time) {
		return BootstrapResponse{}, &ActivationError{StatusCode: 403, Code: "expired", Message: "The provided security key is expired"}
	}

	existing, err := s.loadRecord(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return BootstrapResponse{}, err
	}
	if err == nil && existing != nil && existing.LicenseFamilyID == claims.LicenseFamilyID && existing.Revision >= claims.Revision {
		return BootstrapResponse{}, &ActivationError{StatusCode: 409, Code: "stale_revision", Message: "A newer or equal license revision is already installed"}
	}

	overages, err := s.computeOverages(ctx, claims.MaxOrganizations, claims.MaxUsersPerOrg, claims.MaxWhatsAppEndpointsPerOrg)
	if err != nil {
		return BootstrapResponse{}, err
	}

	encryptedToken, err := appcrypto.Encrypt(token, s.cfg.App.EncryptionKey)
	if err != nil {
		return BootstrapResponse{}, err
	}

	record := &models.LicenseRecord{
		ActivationToken:            encryptedToken,
		LicenseFamilyID:            claims.LicenseFamilyID,
		LicenseID:                  claims.LicenseID,
		Revision:                   claims.Revision,
		KeyID:                      kid,
		Issuer:                     claims.Issuer,
		Audience:                   TokenAudience,
		Product:                    claims.Product,
		HWIDFull:                   s.hwidFull,
		HWIDHash:                   s.hwidHash,
		Tier:                       claims.Tier,
		LicenseKind:                claims.LicenseKind,
		TrialDays:                  claims.TrialDays,
		MaxOrganizations:           claims.MaxOrganizations,
		MaxUsersPerOrg:             claims.MaxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: claims.MaxWhatsAppEndpointsPerOrg,
		MaxWorkers:                 claims.MaxWorkers,
		Status:                     StatusActive,
		Overages:                   intMapToJSONB(overages),
		IssuedAt:                   claims.IssuedAt.Time.UTC(),
		NotBefore:                  claims.NotBefore.Time.UTC(),
		ExpiresAt:                  toTimePtr(claims.ExpiresAt),
		LastSeenAt:                 now,
		ActivatedAt:                now,
	}
	if record.ExpiresAt != nil {
		graceDeadline := record.ExpiresAt.AddDate(0, 0, s.cfg.License.GracePeriodDays)
		record.GraceDeadline = &graceDeadline
	}
	record.IntegrityHMAC = s.recordHMAC(record)

	if err := s.persistRecord(ctx, record); err != nil {
		return BootstrapResponse{}, err
	}

	if err := s.recordEvent(ctx, "license_activated", "", record.Status, record.LicenseFamilyID, record.LicenseID, record.HWIDHash, models.JSONB{
		"tier":                           record.Tier,
		"license_kind":                   record.LicenseKind,
		"trial_days":                     record.TrialDays,
		"max_organizations":              record.MaxOrganizations,
		"max_users_per_org":              record.MaxUsersPerOrg,
		"max_whatsapp_endpoints_per_org": record.MaxWhatsAppEndpointsPerOrg,
		"max_workers":                    record.MaxWorkers,
	}); err != nil {
		s.log.Warn("Failed to record license activation event", "error", err)
	}

	state, err := s.RefreshState(ctx)
	if err != nil {
		return BootstrapResponse{}, err
	}
	usage, err := s.computeUsage(ctx, state)
	if err != nil {
		return BootstrapResponse{}, err
	}
	return BootstrapResponse{State: state, Usage: usage}, nil
}

func (s *Service) RefreshState(ctx context.Context) (State, error) {
	previous := s.CurrentState()

	if !s.cfg.License.Enabled {
		state := State{
			Enabled:       false,
			Status:        StatusDisabled,
			Locked:        false,
			HWIDFull:      s.hwidFull,
			HWIDShort:     s.hwidShort,
			HWIDHash:      s.hwidHash,
			UpdatedAt:     s.now().UTC(),
			QuotaOverages: map[string]int{},
		}
		s.storeState(state)
		return state, nil
	}

	record, err := s.loadRecord(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state := State{
				Enabled:       true,
				Status:        StatusUnlicensed,
				Locked:        true,
				Reason:        "license_required",
				HWIDFull:      s.hwidFull,
				HWIDShort:     s.hwidShort,
				HWIDHash:      s.hwidHash,
				UpdatedAt:     s.now().UTC(),
				QuotaOverages: map[string]int{},
			}
			s.storeState(state)
			if err := s.publishStateIfChanged(ctx, previous, state); err != nil {
				s.log.Warn("Failed to publish license state", "error", err)
			}
			return state, nil
		}
		return State{}, err
	}

	now := s.now().UTC()
	claims, kid, err := s.verifyStoredActivationToken(record)
	if err != nil {
		state := State{
			Enabled:         true,
			Status:          StatusLocked,
			Locked:          true,
			Reason:          "stored_token_invalid",
			LicenseID:       strings.TrimSpace(record.LicenseID),
			LicenseFamilyID: strings.TrimSpace(record.LicenseFamilyID),
			Revision:        record.Revision,
			HWIDFull:        s.hwidFull,
			HWIDShort:       s.hwidShort,
			HWIDHash:        s.hwidHash,
			UpdatedAt:       now,
			QuotaOverages:   map[string]int{},
		}
		s.storeState(state)
		if err := s.publishStateIfChanged(ctx, previous, state); err != nil {
			s.log.Warn("Failed to publish license state", "error", err)
		}
		return state, nil
	}

	claimUpdates := s.applySignedClaimsToRecord(record, claims, kid)
	state := State{
		Enabled:                    true,
		Status:                     StatusActive,
		Locked:                     false,
		LicenseID:                  record.LicenseID,
		LicenseFamilyID:            record.LicenseFamilyID,
		Revision:                   record.Revision,
		KeyID:                      record.KeyID,
		HWIDFull:                   s.hwidFull,
		HWIDShort:                  s.hwidShort,
		HWIDHash:                   s.hwidHash,
		Tier:                       record.Tier,
		LicenseKind:                record.LicenseKind,
		TrialDays:                  record.TrialDays,
		DurationLabel:              licenseDurationLabel(record),
		MaxOrganizations:           record.MaxOrganizations,
		MaxUsersPerOrg:             record.MaxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: record.MaxWhatsAppEndpointsPerOrg,
		MaxWorkers:                 record.MaxWorkers,
		ExpiresAt:                  record.ExpiresAt,
		GraceDeadline:              record.GraceDeadline,
		QuotaOverages:              jsonbToIntMap(record.Overages),
		UpdatedAt:                  now,
	}

	if record.HWIDHash != s.hwidHash {
		state.Status = StatusLocked
		state.Locked = true
		state.Reason = "hwid_mismatch"
	} else if now.Before(record.LastSeenAt.Add(-time.Duration(s.cfg.License.RollbackToleranceSeconds) * time.Second)) {
		state.Status = StatusLocked
		state.Locked = true
		state.Reason = "time_rollback"
	} else if record.ExpiresAt != nil && now.After(record.ExpiresAt.UTC()) {
		if record.GraceDeadline != nil && !now.After(record.GraceDeadline.UTC()) {
			state.Status = StatusGrace
		} else {
			state.Status = StatusLocked
			state.Locked = true
			state.Reason = "expired"
		}
	}

	if state.ExpiresAt != nil {
		days := int(state.ExpiresAt.UTC().Sub(now).Hours() / 24)
		state.DaysUntilExpiry = &days
		state.ExpiringSoon = !state.Locked && state.Status == StatusActive && days <= expiringSoonDays
	}

	overages, err := s.computeOverages(ctx, record.MaxOrganizations, record.MaxUsersPerOrg, record.MaxWhatsAppEndpointsPerOrg)
	if err != nil {
		return State{}, err
	}
	state.QuotaOverages = overages

	updates := map[string]any{
		"status":   state.Status,
		"overages": intMapToJSONB(overages),
	}
	for key, value := range claimUpdates {
		updates[key] = value
	}
	if !state.Locked && now.After(record.LastSeenAt) {
		record.LastSeenAt = now
		updates["last_seen_at"] = now
	}
	if record.GraceDeadline == nil && record.ExpiresAt != nil {
		graceDeadline := record.ExpiresAt.AddDate(0, 0, s.cfg.License.GracePeriodDays)
		record.GraceDeadline = &graceDeadline
		state.GraceDeadline = &graceDeadline
		updates["grace_deadline"] = graceDeadline
	}
	record.Status = state.Status
	record.Overages = intMapToJSONB(overages)
	record.IntegrityHMAC = s.recordHMAC(record)
	updates["integrity_hmac"] = record.IntegrityHMAC

	if err := s.db.WithContext(ctx).Model(&models.LicenseRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		return State{}, err
	}

	s.storeState(state)
	if err := s.publishStateIfChanged(ctx, previous, state); err != nil {
		s.log.Warn("Failed to publish license state", "error", err)
	}
	return state, nil
}

func (s *Service) CheckQuota(ctx context.Context, resource string, orgID uuid.UUID) (QuotaCheck, error) {
	state := s.CurrentState()
	if !state.Enabled {
		return QuotaCheck{Allowed: true, Resource: resource}, nil
	}
	if state.Locked {
		return QuotaCheck{Allowed: false, Resource: resource, Current: 0, Limit: 0, OverQuota: true}, nil
	}

	switch resource {
	case ResourceOrganizations:
		current, err := s.countOrganizations(ctx)
		if err != nil {
			return QuotaCheck{}, err
		}
		return quotaFromCount(resource, current, state.MaxOrganizations), nil
	case ResourceUsers:
		current, err := s.countUsersForOrg(ctx, orgID)
		if err != nil {
			return QuotaCheck{}, err
		}
		return quotaFromCount(resource, current, state.MaxUsersPerOrg), nil
	case ResourceEndpoints:
		current, err := s.countWhatsAppEndpointsForOrg(ctx, orgID)
		if err != nil {
			return QuotaCheck{}, err
		}
		return quotaFromCount(resource, current, state.MaxWhatsAppEndpointsPerOrg), nil
	default:
		return QuotaCheck{}, fmt.Errorf("unknown resource %q", resource)
	}
}

func quotaFromCount(resource string, current, limit int) QuotaCheck {
	return QuotaCheck{
		Allowed:   current < limit,
		Resource:  resource,
		Current:   current,
		Limit:     limit,
		OverQuota: current >= limit,
	}
}

func (s *Service) loadRecord(ctx context.Context) (*models.LicenseRecord, error) {
	var record models.LicenseRecord
	result := s.db.WithContext(ctx).Order("id DESC").Limit(1).Find(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &record, nil
}

func (s *Service) persistRecord(ctx context.Context, record *models.LicenseRecord) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM license_records").Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (s *Service) publishState(ctx context.Context, state State) error {
	if s.redis == nil {
		return nil
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	envelope := map[string]any{
		"state": string(payload),
		"hmac":  s.stateHMAC(payload),
	}
	if envBytes, err := json.Marshal(envelope); err == nil {
		if err := s.redis.Set(ctx, redisStateKey, envBytes, 2*time.Minute).Err(); err != nil {
			return err
		}
	}
	return s.redis.Publish(ctx, redisInvalidateChannel, state.Status).Err()
}

func (s *Service) verifyStoredActivationToken(record *models.LicenseRecord) (*LicenseClaims, string, error) {
	if s == nil || s.cfg == nil {
		return nil, "", fmt.Errorf("license service is not configured")
	}
	if record == nil {
		return nil, "", fmt.Errorf("license record is nil")
	}

	rawToken, err := appcrypto.DecryptStrict(record.ActivationToken, s.cfg.App.EncryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("decrypt stored activation token: %w", err)
	}

	claims, kid, err := VerifyTokenSignatureOnly(rawToken, s.keyRing)
	if err != nil {
		return nil, "", fmt.Errorf("verify stored activation token: %w", err)
	}
	if claims.HWIDHash != s.hwidHash {
		return nil, "", fmt.Errorf("stored activation token HWID mismatch")
	}

	return claims, kid, nil
}

func (s *Service) applySignedClaimsToRecord(record *models.LicenseRecord, claims *LicenseClaims, kid string) map[string]any {
	updates := map[string]any{}
	if record == nil || claims == nil {
		return updates
	}

	setString := func(current *string, column, next string) {
		next = strings.TrimSpace(next)
		if *current == next {
			return
		}
		*current = next
		updates[column] = next
	}
	setUint64 := func(current *uint64, column string, next uint64) {
		if *current == next {
			return
		}
		*current = next
		updates[column] = next
	}
	setInt := func(current *int, column string, next int) {
		if *current == next {
			return
		}
		*current = next
		updates[column] = next
	}
	setTime := func(current *time.Time, column string, next time.Time) {
		next = next.UTC()
		if sameMoment(*current, next) {
			return
		}
		*current = next
		updates[column] = next
	}
	setTimePtr := func(current **time.Time, column string, next *time.Time) {
		if timePtrEqual(*current, next) {
			return
		}
		if next == nil {
			*current = nil
			updates[column] = nil
			return
		}
		value := next.UTC()
		*current = &value
		updates[column] = value
	}

	setString(&record.LicenseFamilyID, "license_family_id", claims.LicenseFamilyID)
	setString(&record.LicenseID, "license_id", claims.LicenseID)
	setUint64(&record.Revision, "revision", claims.Revision)
	setString(&record.KeyID, "key_id", kid)
	setString(&record.Issuer, "issuer", claims.Issuer)
	setString(&record.Audience, "audience", TokenAudience)
	setString(&record.Product, "product", claims.Product)
	setString(&record.HWIDFull, "hwid_full", s.hwidFull)
	setString(&record.HWIDHash, "hwid_hash", claims.HWIDHash)
	setString(&record.Tier, "tier", claims.Tier)
	setString(&record.LicenseKind, "license_kind", claims.LicenseKind)
	setInt(&record.TrialDays, "trial_days", claims.TrialDays)
	setInt(&record.MaxOrganizations, "max_organizations", claims.MaxOrganizations)
	setInt(&record.MaxUsersPerOrg, "max_users_per_org", claims.MaxUsersPerOrg)
	setInt(&record.MaxWhatsAppEndpointsPerOrg, "max_whatsapp_endpoints_per_org", claims.MaxWhatsAppEndpointsPerOrg)
	setInt(&record.MaxWorkers, "max_workers", claims.MaxWorkers)
	setTime(&record.IssuedAt, "issued_at", numericDateTime(claims.IssuedAt))
	setTime(&record.NotBefore, "not_before", numericDateTime(claims.NotBefore))
	setTimePtr(&record.ExpiresAt, "expires_at", toTimePtr(claims.ExpiresAt))

	return updates
}

func (s *Service) publishStateIfChanged(ctx context.Context, previous, next State) error {
	if s.redis == nil {
		return nil
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return err
	}
	envelope := map[string]any{
		"state": string(payload),
		"hmac":  s.stateHMAC(payload),
	}
	if envBytes, err := json.Marshal(envelope); err == nil {
		if err := s.redis.Set(ctx, redisStateKey, envBytes, 2*time.Minute).Err(); err != nil {
			return err
		}
	}

	if statesEquivalent(previous, next) {
		return nil
	}

	return s.redis.Publish(ctx, redisInvalidateChannel, next.Status).Err()
}

func statesEquivalent(left, right State) bool {
	left.UpdatedAt = time.Time{}
	right.UpdatedAt = time.Time{}
	return reflect.DeepEqual(left, right)
}

func (s *Service) recordHMAC(record *models.LicenseRecord) string {
	tokenDigest := sha256.Sum256([]byte(record.ActivationToken))
	payload := map[string]any{
		"activation_token_sha256":        hex.EncodeToString(tokenDigest[:]),
		"license_family_id":              record.LicenseFamilyID,
		"license_id":                     record.LicenseID,
		"revision":                       record.Revision,
		"key_id":                         record.KeyID,
		"issuer":                         record.Issuer,
		"audience":                       record.Audience,
		"product":                        record.Product,
		"hwid_full":                      record.HWIDFull,
		"hwid_hash":                      record.HWIDHash,
		"tier":                           record.Tier,
		"license_kind":                   record.LicenseKind,
		"trial_days":                     record.TrialDays,
		"max_organizations":              record.MaxOrganizations,
		"max_users_per_org":              record.MaxUsersPerOrg,
		"max_whatsapp_endpoints_per_org": record.MaxWhatsAppEndpointsPerOrg,
		"max_workers":                    record.MaxWorkers,
		"status":                         record.Status,
		"overages":                       record.Overages,
		"issued_at":                      record.IssuedAt.UTC().Format(time.RFC3339Nano),
		"not_before":                     record.NotBefore.UTC().Format(time.RFC3339Nano),
		"expires_at":                     formatTime(record.ExpiresAt),
		"grace_deadline":                 formatTime(record.GraceDeadline),
		"last_seen_at":                   record.LastSeenAt.UTC().Format(time.RFC3339Nano),
		"activated_at":                   record.ActivatedAt.UTC().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(payload)
	return s.stateHMAC(data)
}

func (s *Service) stateHMAC(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.App.EncryptionKey))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func licenseDurationLabel(record *models.LicenseRecord) string {
	if record == nil {
		return ""
	}
	if record.LicenseKind == KindTrial && record.TrialDays > 0 {
		return fmt.Sprintf("%dd", record.TrialDays)
	}
	if record.ExpiresAt == nil {
		return "lifetime"
	}
	if record.IssuedAt.IsZero() {
		return ""
	}
	days := int(record.ExpiresAt.UTC().Sub(record.IssuedAt.UTC()).Hours() / 24)
	if days <= 0 {
		return ""
	}
	return fmt.Sprintf("%dd", days)
}

func (s *Service) recordEvent(ctx context.Context, eventType, reason, status, familyID, licenseID, hwidHash string, details models.JSONB) error {
	if s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx).Create(&models.LicenseEvent{
		EventType:       eventType,
		Reason:          reason,
		Status:          status,
		LicenseFamilyID: familyID,
		LicenseID:       licenseID,
		HWIDHash:        hwidHash,
		Details:         details,
	}).Error
}

func (s *Service) storeState(state State) {
	cloned := cloneState(state)
	s.state.Store(&cloned)
}

func cloneState(state State) State {
	cloned := state
	if state.QuotaOverages != nil {
		cloned.QuotaOverages = make(map[string]int, len(state.QuotaOverages))
		for key, value := range state.QuotaOverages {
			cloned.QuotaOverages[key] = value
		}
	}
	if state.ExpiresAt != nil {
		expiresAt := state.ExpiresAt.UTC()
		cloned.ExpiresAt = &expiresAt
	}
	if state.GraceDeadline != nil {
		graceDeadline := state.GraceDeadline.UTC()
		cloned.GraceDeadline = &graceDeadline
	}
	if state.DaysUntilExpiry != nil {
		days := *state.DaysUntilExpiry
		cloned.DaysUntilExpiry = &days
	}
	return cloned
}

func jsonbToIntMap(input models.JSONB) map[string]int {
	result := make(map[string]int)
	for key, value := range input {
		switch typed := value.(type) {
		case float64:
			result[key] = int(typed)
		case int:
			result[key] = typed
		case int64:
			result[key] = int(typed)
		}
	}
	return result
}

func intMapToJSONB(input map[string]int) models.JSONB {
	if len(input) == 0 {
		return models.JSONB{}
	}
	result := make(models.JSONB, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func toTimePtr(date *jwt.NumericDate) *time.Time {
	if date == nil {
		return nil
	}
	value := date.Time.UTC()
	return &value
}

func numericDateTime(date *jwt.NumericDate) time.Time {
	if date == nil {
		return time.Time{}
	}
	return date.Time.UTC()
}

func sameMoment(left, right time.Time) bool {
	if left.IsZero() || right.IsZero() {
		return left.IsZero() && right.IsZero()
	}
	return left.UTC().Equal(right.UTC())
}

func timePtrEqual(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return sameMoment(*left, *right)
	}
}

func mapActivationError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return &ActivationError{StatusCode: 403, Code: "expired", Message: "The provided security key is expired"}
	default:
		return &ActivationError{StatusCode: 403, Code: "invalid_token", Message: "The provided security key is invalid"}
	}
}

func (s *Service) computeUsage(ctx context.Context, state State) (UsageSnapshot, error) {
	orgs, err := s.listOrganizations(ctx)
	if err != nil {
		return UsageSnapshot{}, err
	}

	usage := UsageSnapshot{
		Organizations: MetricUsage{
			Current: len(orgs),
			Limit:   state.MaxOrganizations,
		},
		UsersPerOrg: MetricUsage{
			Limit: state.MaxUsersPerOrg,
		},
		WhatsAppEndpointsPerOrg: MetricUsage{
			Limit: state.MaxWhatsAppEndpointsPerOrg,
		},
		OrganizationDetails: make([]OrganizationUsage, 0, len(orgs)),
	}

	maxUsers := 0
	maxEndpoints := 0
	for _, org := range orgs {
		userCount, err := s.countUsersForOrg(ctx, org.ID)
		if err != nil {
			return UsageSnapshot{}, err
		}
		endpointCount, err := s.countWhatsAppEndpointsForOrg(ctx, org.ID)
		if err != nil {
			return UsageSnapshot{}, err
		}
		if userCount > maxUsers {
			maxUsers = userCount
		}
		if endpointCount > maxEndpoints {
			maxEndpoints = endpointCount
		}
		usage.OrganizationDetails = append(usage.OrganizationDetails, OrganizationUsage{
			OrganizationID:        org.ID,
			OrganizationName:      org.Name,
			UserCount:             userCount,
			WhatsAppEndpointCount: endpointCount,
		})
	}

	usage.Organizations.OverQuota = usage.Organizations.Limit > 0 && usage.Organizations.Current > usage.Organizations.Limit
	if usage.Organizations.OverQuota {
		usage.Organizations.Overage = usage.Organizations.Current - usage.Organizations.Limit
	}
	usage.UsersPerOrg.Current = maxUsers
	usage.UsersPerOrg.OverQuota = usage.UsersPerOrg.Limit > 0 && maxUsers > usage.UsersPerOrg.Limit
	if usage.UsersPerOrg.OverQuota {
		usage.UsersPerOrg.Overage = maxUsers - usage.UsersPerOrg.Limit
	}
	usage.WhatsAppEndpointsPerOrg.Current = maxEndpoints
	usage.WhatsAppEndpointsPerOrg.OverQuota = usage.WhatsAppEndpointsPerOrg.Limit > 0 && maxEndpoints > usage.WhatsAppEndpointsPerOrg.Limit
	if usage.WhatsAppEndpointsPerOrg.OverQuota {
		usage.WhatsAppEndpointsPerOrg.Overage = maxEndpoints - usage.WhatsAppEndpointsPerOrg.Limit
	}
	return usage, nil
}

func (s *Service) computeOverages(ctx context.Context, maxOrganizations, maxUsersPerOrg, maxEndpointsPerOrg int) (map[string]int, error) {
	usage, err := s.computeUsage(ctx, State{
		MaxOrganizations:           maxOrganizations,
		MaxUsersPerOrg:             maxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: maxEndpointsPerOrg,
	})
	if err != nil {
		return nil, err
	}

	overages := map[string]int{}
	if usage.Organizations.Overage > 0 {
		overages[ResourceOrganizations] = usage.Organizations.Overage
	}
	if usage.UsersPerOrg.Overage > 0 {
		overages[ResourceUsers] = usage.UsersPerOrg.Overage
	}
	if usage.WhatsAppEndpointsPerOrg.Overage > 0 {
		overages[ResourceEndpoints] = usage.WhatsAppEndpointsPerOrg.Overage
	}
	return overages, nil
}

func (s *Service) listOrganizations(ctx context.Context) ([]models.Organization, error) {
	var orgs []models.Organization
	if err := s.db.WithContext(ctx).Order("created_at ASC").Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

func (s *Service) countOrganizations(ctx context.Context) (int, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Organization{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Service) countUsersForOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.UserOrganization{}).
		Distinct("user_id").
		Where("organization_id = ?", orgID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Service) countWhatsAppEndpointsForOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var accounts int64
	if err := s.db.WithContext(ctx).Model(&models.WhatsAppAccount{}).Where("organization_id = ?", orgID).Count(&accounts).Error; err != nil {
		return 0, err
	}
	var instances int64
	if err := s.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).Where("organization_id = ?", orgID).Count(&instances).Error; err != nil {
		return 0, err
	}
	return int(accounts + instances), nil
}

func BuildHWID(cfg *config.LicenseConfig, log logf.Logger) (full, short, hash string, err error) {
	values := make([]string, 0, 4)
	seen := make(map[string]struct{})

	addValue := func(label, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		entry := label + ":" + trimmed
		if _, exists := seen[entry]; exists {
			return
		}
		seen[entry] = struct{}{}
		values = append(values, entry)
	}

	if cfg != nil && strings.TrimSpace(cfg.HostMachineIDPath) != "" {
		data, readErr := os.ReadFile(strings.TrimSpace(cfg.HostMachineIDPath))
		if readErr != nil {
			return "", "", "", fmt.Errorf("configured license.host_machine_id_path is missing or unreadable: %w", readErr)
		}
		addValue("machine-id", string(data))
	}

	if cfg != nil {
		for _, source := range cfg.FingerprintSources {
			path := strings.TrimSpace(source)
			if path == "" {
				continue
			}
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				label := filepathLabel(path)
				addValue(label, string(data))
			}
		}
	}

	if len(values) == 0 {
		for _, mac := range stableMACAddresses() {
			addValue("mac", mac)
		}
	}

	if len(values) == 0 {
		return "", "", "", fmt.Errorf("unable to derive stable host identity for licensing")
	}

	sort.Strings(values)
	canonical := strings.Join(values, "|")
	sum := sha256.Sum256([]byte(canonical))
	hash = hex.EncodeToString(sum[:])
	full = hash
	short = hash
	if len(short) > 12 {
		short = short[:12]
	}

	if runningInContainer() && (cfg == nil || strings.TrimSpace(cfg.HostMachineIDPath) == "") {
		log.Warn("Docker/container environment detected without license.host_machine_id_path; host-bound licensing may drift on container recreation")
	}

	return full, short, hash, nil
}

func filepathLabel(path string) string {
	switch filepath := strings.ToLower(path); {
	case strings.Contains(filepath, "machine-id"):
		return "machine-id"
	case strings.Contains(filepath, "product_uuid"):
		return "product-uuid"
	default:
		return path
	}
}

func stableMACAddresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	blockedPrefixes := []string{"lo", "docker", "veth", "br-", "virbr", "cni", "flannel", "tun", "tap"}
	values := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		name := strings.ToLower(strings.TrimSpace(iface.Name))
		if name == "" {
			continue
		}
		skip := false
		for _, prefix := range blockedPrefixes {
			if strings.HasPrefix(name, prefix) {
				skip = true
				break
			}
		}
		if skip || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := strings.TrimSpace(iface.HardwareAddr.String())
		if mac == "" {
			continue
		}
		values = append(values, mac)
	}
	sort.Strings(values)
	return values
}

func runningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.Contains(text, "docker") || strings.Contains(text, "kubepods") || strings.Contains(text, "containerd")
}
