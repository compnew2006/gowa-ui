package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Status represents the state of a migration operation.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusComplete  Status = "complete"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"

	defaultBatchSize = 500
)

// Progress tracks the progress of a migration.
type Progress struct {
	OrgID            uuid.UUID `json:"organization_id"`
	OrgName          string    `json:"organization_name"`
	Status           Status    `json:"status"`
	AccountsTotal    int       `json:"accounts_total"`
	AccountsMigrated int       `json:"accounts_migrated"`
	ContactsTotal    int64     `json:"contacts_total"`
	ContactsMigrated int64     `json:"contacts_migrated"`
	MessagesTotal    int64     `json:"messages_total"`
	MessagesMigrated int64     `json:"messages_migrated"`
	StartedAt        time.Time `json:"started_at"`
	FinishedAt       time.Time `json:"finished_at,omitempty"`
	Error            string    `json:"error,omitempty"`
}

// Result is the overall migration result.
type Result struct {
	TotalOrgs    int        `json:"total_orgs"`
	SuccessOrgs  int        `json:"success_orgs"`
	FailedOrgs   int        `json:"failed_orgs"`
	OrgProgress  []Progress `json:"org_progress"`
	OverallStart time.Time  `json:"started_at"`
	OverallEnd   time.Time  `json:"finished_at"`
}

// Service performs data migration from WhatsAppAccount to WhatsAppInstance.
type Service struct {
	db        *gorm.DB
	logger    logf.Logger
	batchSize int

	mu       sync.RWMutex
	running  bool
	progress []Progress
}

// NewService creates a new migration service.
func NewService(db *gorm.DB, logger logf.Logger) *Service {
	return &Service{
		db:        db,
		logger:    logger,
		batchSize: defaultBatchSize,
	}
}

// IsRunning returns true if a migration is currently in progress.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetProgress returns the current migration progress for all orgs.
func (s *Service) GetProgress() []Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]Progress, len(s.progress))
	copy(cp, s.progress)
	return cp
}

// MigrateAll migrates all organizations that have WhatsAppAccounts
// but no corresponding WhatsAppInstances.
func (s *Service) MigrateAll(ctx context.Context) (*Result, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("migration already in progress")
	}
	s.running = true
	s.progress = nil
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	result := &Result{
		OverallStart: time.Now(),
	}

	// Find all orgs that have WhatsAppAccounts.
	var accounts []models.WhatsAppAccount
	if err := s.db.WithContext(ctx).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	// Group by org.
	orgAccounts := make(map[uuid.UUID][]models.WhatsAppAccount)
	for _, acc := range accounts {
		orgAccounts[acc.OrganizationID] = append(orgAccounts[acc.OrganizationID], acc)
	}

	result.TotalOrgs = len(orgAccounts)
	s.mu.Lock()
	for orgID := range orgAccounts {
		s.progress = append(s.progress, Progress{
			OrgID:         orgID,
			Status:        StatusPending,
			AccountsTotal: len(orgAccounts[orgID]),
		})
	}
	s.mu.Unlock()

	for i, p := range s.GetProgress() {
		if ctx.Err() != nil {
			s.updateProgress(i, func(pp *Progress) {
				pp.Status = StatusCancelled
			})
			continue
		}
		err := s.migrateOrg(ctx, p.OrgID, orgAccounts[p.OrgID], i)
		if err != nil {
			s.updateProgress(i, func(pp *Progress) {
				pp.Status = StatusFailed
				pp.Error = err.Error()
				pp.FinishedAt = time.Now()
			})
			result.FailedOrgs++
			s.logger.Error("Migration failed for org",
				"organization_id", p.OrgID,
				"error", err,
			)
		} else {
			result.SuccessOrgs++
		}
	}

	result.OverallEnd = time.Now()
	result.OrgProgress = s.GetProgress()
	return result, nil
}

// MigrateOrg migrates a single organization by ID.
func (s *Service) MigrateOrg(ctx context.Context, orgID uuid.UUID) (*Progress, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("migration already in progress")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	var accounts []models.WhatsAppAccount
	if err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to list accounts for org: %w", err)
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no WhatsApp accounts found for organization %s", orgID)
	}

	s.mu.Lock()
	s.progress = []Progress{{
		OrgID:         orgID,
		Status:        StatusPending,
		AccountsTotal: len(accounts),
	}}
	s.mu.Unlock()

	if err := s.migrateOrg(ctx, orgID, accounts, 0); err != nil {
		return nil, err
	}

	prog := s.GetProgress()
	if len(prog) > 0 {
		return &prog[0], nil
	}
	return nil, nil
}

func (s *Service) migrateOrg(ctx context.Context, orgID uuid.UUID, accounts []models.WhatsAppAccount, progressIdx int) error {
	s.updateProgress(progressIdx, func(p *Progress) {
		p.Status = StatusRunning
		p.StartedAt = time.Now()
	})

	s.logger.Info("Starting migration for organization",
		"organization_id", orgID,
		"accounts_count", len(accounts),
	)

	for _, acc := range accounts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check if an instance already exists for this account (idempotent).
		var existingCount int64
		if err := s.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
			Where("organization_id = ? AND name = ?", orgID, acc.Name).
			Count(&existingCount).Error; err != nil {
			return fmt.Errorf("failed to check existing instance for account %s: %w", acc.Name, err)
		}

		var instance models.WhatsAppInstance
		if existingCount > 0 {
			// Already migrated — load it.
			if err := s.db.WithContext(ctx).
				Where("organization_id = ? AND name = ?", orgID, acc.Name).
				First(&instance).Error; err != nil {
				return fmt.Errorf("failed to load existing instance for account %s: %w", acc.Name, err)
			}
			s.logger.Info("Instance already exists, reusing",
				"account_name", acc.Name,
				"instance_id", instance.ID,
			)
		} else {
			// Create new WhatsAppInstance from WhatsAppAccount.
			instance = models.WhatsAppInstance{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				OrganizationID:  orgID,
				Name:            acc.Name,
				Status:          models.InstanceStatusDisconnected,
				IsDefault:       acc.IsDefaultOutgoing,
				AutoReadReceipt: acc.AutoReadReceipt,
			}
			if err := s.db.WithContext(ctx).Create(&instance).Error; err != nil {
				return fmt.Errorf("failed to create instance for account %s: %w", acc.Name, err)
			}
			s.logger.Info("Created instance for account",
				"account_name", acc.Name,
				"instance_id", instance.ID,
			)
		}

		// Migrate contacts: set instance_id where whatsapp_account matches.
		if err := s.migrateContacts(ctx, orgID, acc.Name, instance.ID, progressIdx); err != nil {
			return fmt.Errorf("failed to migrate contacts for account %s: %w", acc.Name, err)
		}

		// Migrate messages: set instance_id where whatsapp_account matches.
		if err := s.migrateMessages(ctx, orgID, acc.Name, instance.ID, progressIdx); err != nil {
			return fmt.Errorf("failed to migrate messages for account %s: %w", acc.Name, err)
		}

		s.updateProgress(progressIdx, func(p *Progress) {
			p.AccountsMigrated++
		})
	}

	s.updateProgress(progressIdx, func(p *Progress) {
		p.Status = StatusComplete
		p.FinishedAt = time.Now()
	})

	s.logger.Info("Migration complete for organization",
		"organization_id", orgID,
	)
	return nil
}

func (s *Service) migrateContacts(ctx context.Context, orgID uuid.UUID, accountName string, instanceID uuid.UUID, progressIdx int) error {
	// Count total contacts to migrate.
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Contact{}).
		Where("organization_id = ? AND whatsapp_account = ? AND (instance_id IS NULL)", orgID, accountName).
		Count(&total).Error; err != nil {
		return err
	}

	s.updateProgress(progressIdx, func(p *Progress) {
		p.ContactsTotal += total
	})

	if total == 0 {
		return nil
	}

	// Batch update.
	var migrated int64
	for migrated < total {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		result := s.db.WithContext(ctx).
			Model(&models.Contact{}).
			Where("organization_id = ? AND whatsapp_account = ? AND instance_id IS NULL", orgID, accountName).
			Limit(s.batchSize).
			Update("instance_id", instanceID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		migrated += result.RowsAffected
		s.updateProgress(progressIdx, func(p *Progress) {
			p.ContactsMigrated += result.RowsAffected
		})
	}

	s.logger.Info("Contacts migrated",
		"account_name", accountName,
		"count", migrated,
	)
	return nil
}

func (s *Service) migrateMessages(ctx context.Context, orgID uuid.UUID, accountName string, instanceID uuid.UUID, progressIdx int) error {
	// Count total messages to migrate.
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Message{}).
		Where("organization_id = ? AND whatsapp_account = ? AND (instance_id IS NULL)", orgID, accountName).
		Count(&total).Error; err != nil {
		return err
	}

	s.updateProgress(progressIdx, func(p *Progress) {
		p.MessagesTotal += total
	})

	if total == 0 {
		return nil
	}

	// Batch update.
	var migrated int64
	for migrated < total {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		result := s.db.WithContext(ctx).
			Model(&models.Message{}).
			Where("organization_id = ? AND whatsapp_account = ? AND instance_id IS NULL", orgID, accountName).
			Limit(s.batchSize).
			Update("instance_id", instanceID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		migrated += result.RowsAffected
		s.updateProgress(progressIdx, func(p *Progress) {
			p.MessagesMigrated += result.RowsAffected
		})
	}

	s.logger.Info("Messages migrated",
		"account_name", accountName,
		"count", migrated,
	)
	return nil
}

func (s *Service) updateProgress(idx int, fn func(p *Progress)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx >= 0 && idx < len(s.progress) {
		fn(&s.progress[idx])
	}
}
