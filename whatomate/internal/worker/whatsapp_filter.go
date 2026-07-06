package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	wameow "go.mau.fi/whatsmeow"
	"gorm.io/gorm"
)

type whatsmeowClientGetter interface {
	GetClient(ctx context.Context, instanceID string) (*wameow.Client, error)
}

const whatsmeowContactCheckTimeout = 60 * time.Second

// HandleWhatsAppFilterJob processes a background WhatsApp number validation job.
func (w *Worker) HandleWhatsAppFilterJob(ctx context.Context, job *queue.WhatsAppFilterJob) error {
	w.Log.Info("Processing whatsapp filter job", "batch_id", job.BatchID, "org_id", job.OrganizationID)

	// 1. Update batch status to processing
	err := w.DB.Model(&models.WhatsAppFilterBatch{}).
		Where("id = ? AND organization_id = ?", job.BatchID, job.OrganizationID).
		Update("status", models.FilterStatusProcessing).Error
	if err != nil {
		w.Log.Error("Failed to update filter batch status to processing", "error", err, "batch_id", job.BatchID)
		return err
	}

	// 2. Fetch pending result rows for the batch
	var results []models.WhatsAppFilterResult
	err = w.DB.Where("batch_id = ? AND checked_at IS NULL", job.BatchID).Find(&results).Error
	if err != nil {
		w.Log.Error("Failed to fetch pending results", "error", err, "batch_id", job.BatchID)
		w.failWhatsAppFilterBatch(job.BatchID, "Failed to retrieve numbers for verification")
		return err
	}

	if len(results) == 0 {
		w.Log.Info("No pending results for filter batch", "batch_id", job.BatchID)
		w.completeWhatsAppFilterBatch(job.BatchID)
		return nil
	}

	// Extract phone numbers
	phones := make([]string, len(results))
	resultMap := make(map[string]*models.WhatsAppFilterResult)
	for i, r := range results {
		phones[i] = r.PhoneNumber
		resultMap[r.PhoneNumber] = &results[i]
	}

	// 3. Process in sub-batches of 50
	const subBatchSize = 50
	validCountGlobal := 0
	invalidCountGlobal := 0

	for i := 0; i < len(phones); i += subBatchSize {
		end := i + subBatchSize
		if end > len(phones) {
			end = len(phones)
		}
		subBatch := phones[i:end]

		checkedResults, remainingPhones, err := w.checkExistingContacts(job.OrganizationID, subBatch)
		if err != nil {
			w.Log.Error("Failed to check existing contacts for filter batch", "error", err, "batch_id", job.BatchID)
			w.failWhatsAppFilterBatch(job.BatchID, "Failed to check existing contacts")
			return err
		}

		if len(remainingPhones) > 0 {
			var providerResults map[string]struct {
				IsValid bool
				Name    string
				Error   string
			}
			if job.InstanceID != nil {
				providerResults, err = w.checkWhatsmeowContacts(ctx, *job.InstanceID, remainingPhones)
			} else if w.isWhatsmeowProvider() {
				providerResults, err = w.checkWhatsmeowContacts(ctx, job.WhatsAppAccountID, remainingPhones)
			} else {
				providerResults, err = w.checkMetaContacts(ctx, job.WhatsAppAccountID, job.OrganizationID, remainingPhones)
			}
			for phone, result := range providerResults {
				checkedResults[phone] = result
			}
		}

		if err != nil {
			w.Log.Error("Sub-batch contact verification failed", "error", err, "batch_id", job.BatchID)
			// Return temporary error to retry, or continue with next batch
			// We choose to fail batch if it's a provider connection error
			w.failWhatsAppFilterBatch(job.BatchID, err.Error())
			return err
		}

		// Update database rows for this sub-batch
		tx := w.DB.Begin()
		validCount := 0
		invalidCount := 0

		for _, phone := range subBatch {
			checkRes, ok := checkedResults[phone]
			if !ok {
				checkRes.IsValid = false
				checkRes.Error = "No response from provider"
			}

			dbRow, ok := resultMap[phone]
			if !ok {
				continue
			}

			dbRow.IsValid = checkRes.IsValid
			dbRow.ContactName = checkRes.Name
			dbRow.ErrorMessage = checkRes.Error
			now := time.Now()
			dbRow.CheckedAt = &now

			if err := tx.Model(dbRow).Updates(map[string]any{
				"is_valid":      dbRow.IsValid,
				"contact_name":  dbRow.ContactName,
				"error_message": dbRow.ErrorMessage,
				"checked_at":    dbRow.CheckedAt,
			}).Error; err != nil {
				tx.Rollback()
				w.Log.Error("Failed to update filter result row", "error", err, "phone", phone)
				w.failWhatsAppFilterBatch(job.BatchID, "Database write failure")
				return err
			}

			if dbRow.IsValid {
				validCount++
			} else {
				invalidCount++
			}
		}

		if err := tx.Commit().Error; err != nil {
			w.Log.Error("Failed to commit filter results sub-batch transaction", "error", err)
			w.failWhatsAppFilterBatch(job.BatchID, "Transaction commit failure")
			return err
		}

		// Atomically increment counts
		w.incrementFilterBatchCount(job.BatchID, "valid_numbers", validCount)
		w.incrementFilterBatchCount(job.BatchID, "invalid_numbers", invalidCount)

		validCountGlobal += validCount
		invalidCountGlobal += invalidCount

		// Slight delay between sub-batches to respect rate limits
		if end < len(phones) {
			time.Sleep(200 * time.Millisecond)
		}
	}

	w.completeWhatsAppFilterBatch(job.BatchID)
	w.Log.Info("WhatsApp filter job completed successfully", "batch_id", job.BatchID, "valid", validCountGlobal, "invalid", invalidCountGlobal)
	return nil
}

func (w *Worker) checkWhatsmeowContacts(ctx context.Context, instanceID uuid.UUID, phones []string) (map[string]struct {
	IsValid bool
	Name    string
	Error   string
}, error) {
	results := make(map[string]struct {
		IsValid bool
		Name    string
		Error   string
	})

	client, err := w.getWhatsmeowFilterClient(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("whatsmeow client is not connected for instance %s", instanceID)
	}

	normalizedPhones := make([]string, 0, len(phones))
	phoneMap := make(map[string]string) // Normalized -> Original

	for _, phone := range phones {
		norm, err := normalizePhoneForWhatsmeow(phone)
		if err != nil {
			results[phone] = struct {
				IsValid bool
				Name    string
				Error   string
			}{IsValid: false, Error: "Invalid phone format"}
			continue
		}
		normalizedPhones = append(normalizedPhones, norm)
		phoneMap[norm] = phone
	}

	if len(normalizedPhones) == 0 {
		return results, nil
	}

	checkCtx := context.Background()
	if ctx != nil {
		checkCtx = context.WithoutCancel(ctx)
	}
	checkCtx, cancel := context.WithTimeout(checkCtx, whatsmeowContactCheckTimeout)
	defer cancel()

	// Call Whatsmeow IsOnWhatsApp API
	resp, err := client.IsOnWhatsApp(checkCtx, normalizedPhones)
	if err != nil {
		return nil, fmt.Errorf("whatsmeow IsOnWhatsApp failed: %w", err)
	}

	for _, item := range resp {
		originalPhone, ok := phoneMap[item.Query]
		if !ok {
			originalPhone = item.Query
		}

		name := ""
		if item.VerifiedName != nil && item.VerifiedName.Details != nil {
			name = strings.TrimSpace(item.VerifiedName.Details.GetVerifiedName())
		}

		results[originalPhone] = struct {
			IsValid bool
			Name    string
			Error   string
		}{
			IsValid: item.IsIn,
			Name:    name,
		}
	}

	return results, nil
}

func (w *Worker) getWhatsmeowFilterClient(ctx context.Context, instanceID uuid.UUID) (*wameow.Client, error) {
	if getter, ok := w.MessageProvider.(whatsmeowClientGetter); ok {
		client, err := getter.GetClient(ctx, instanceID.String())
		if err != nil {
			return nil, fmt.Errorf("whatsmeow client is not connected for instance %s: %w", instanceID, err)
		}
		return client, nil
	}

	mgr := w.whatsmeowMgr
	if mgr == nil {
		return nil, errors.New("whatsmeow provider is not available")
	}

	client := mgr.GetClient(instanceID)
	if client != nil && client.IsConnected() {
		return client, nil
	}

	reconnectCtx := ctx
	if reconnectCtx == nil {
		reconnectCtx = context.Background()
	}
	reconnectCtx, cancel := context.WithTimeout(context.WithoutCancel(reconnectCtx), 15*time.Second)
	defer cancel()

	if err := mgr.Connect(reconnectCtx, instanceID); err != nil {
		return nil, fmt.Errorf("whatsmeow client is not connected for instance %s: %w", instanceID, err)
	}

	return mgr.GetClient(instanceID), nil
}

func (w *Worker) checkExistingContacts(orgID uuid.UUID, phones []string) (map[string]struct {
	IsValid bool
	Name    string
	Error   string
}, []string, error) {
	results := make(map[string]struct {
		IsValid bool
		Name    string
		Error   string
	})
	if len(phones) == 0 {
		return results, nil, nil
	}

	candidateToPhone := make(map[string]string)
	candidates := make([]string, 0, len(phones)*4)
	addCandidate := func(original, candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, exists := candidateToPhone[candidate]; exists {
			return
		}
		candidateToPhone[candidate] = original
		candidates = append(candidates, candidate)
	}

	for _, phone := range phones {
		trimmed := strings.TrimSpace(phone)
		digits := strings.TrimPrefix(trimmed, "+")
		addCandidate(phone, trimmed)
		addCandidate(phone, digits)
		addCandidate(phone, "+"+digits)
		normalized := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, trimmed)
		addCandidate(phone, normalized)
		addCandidate(phone, "+"+normalized)
	}

	var contacts []models.Contact
	if err := w.DB.
		Select("phone_number", "profile_name").
		Where("organization_id = ? AND phone_number IN ?", orgID, candidates).
		Find(&contacts).Error; err != nil {
		return nil, nil, err
	}

	for _, contact := range contacts {
		original, ok := candidateToPhone[strings.TrimSpace(contact.PhoneNumber)]
		if !ok {
			continue
		}
		name := strings.TrimSpace(contact.ProfileName)
		if name == "" {
			name = strings.TrimSpace(contact.PhoneNumber)
		}
		results[original] = struct {
			IsValid bool
			Name    string
			Error   string
		}{
			IsValid: true,
			Name:    name,
			Error:   "Found in contacts",
		}
	}

	remaining := make([]string, 0, len(phones)-len(results))
	for _, phone := range phones {
		if _, ok := results[phone]; !ok {
			remaining = append(remaining, phone)
		}
	}

	return results, remaining, nil
}

func (w *Worker) checkMetaContacts(ctx context.Context, accountID uuid.UUID, orgID uuid.UUID, phones []string) (map[string]struct {
	IsValid bool
	Name    string
	Error   string
}, error) {
	results := make(map[string]struct {
		IsValid bool
		Name    string
		Error   string
	})

	// Fetch account
	var account models.WhatsAppAccount
	if err := w.DB.Where("id = ? AND organization_id = ?", accountID, orgID).First(&account).Error; err != nil {
		return nil, fmt.Errorf("whatsapp account not found: %w", err)
	}

	if err := w.decryptAccountSecrets(&account); err != nil {
		return nil, fmt.Errorf("failed to decrypt account credentials: %w", err)
	}

	waAccount := &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}

	// Form input format for Meta: must start with "+"
	normalizedPhones := make([]string, len(phones))
	phoneMap := make(map[string]string)

	for i, phone := range phones {
		norm := phone
		if !strings.HasPrefix(norm, "+") {
			norm = "+" + norm
		}
		normalizedPhones[i] = norm
		phoneMap[norm] = phone
	}

	metaResults, err := w.WhatsApp.CheckContacts(ctx, waAccount, normalizedPhones)
	if err != nil {
		return nil, fmt.Errorf("meta check contacts failed: %w", err)
	}

	for _, item := range metaResults {
		originalPhone, ok := phoneMap[item.Input]
		if !ok {
			originalPhone = item.Input
		}

		results[originalPhone] = struct {
			IsValid bool
			Name    string
			Error   string
		}{
			IsValid: item.Status == "valid",
			Name:    item.WaID, // Meta returns registered JID/WaID
		}
	}

	return results, nil
}

// Helpers

func (w *Worker) incrementFilterBatchCount(batchID uuid.UUID, column string, count int) {
	if count <= 0 {
		return
	}
	if err := w.DB.Model(&models.WhatsAppFilterBatch{}).
		Where("id = ?", batchID).
		Update(column, gorm.Expr(column+" + ?", count)).Error; err != nil {
		w.Log.Error("Failed to increment filter batch count", "error", err, "batch_id", batchID, "column", column)
	}
}

func (w *Worker) completeWhatsAppFilterBatch(batchID uuid.UUID) {
	now := time.Now()
	_ = w.DB.Model(&models.WhatsAppFilterBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]any{
			"status":       models.FilterStatusCompleted,
			"completed_at": &now,
		})
}

func (w *Worker) failWhatsAppFilterBatch(batchID uuid.UUID, errMsg string) {
	now := time.Now()
	_ = w.DB.Model(&models.WhatsAppFilterBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]any{
			"status":        models.FilterStatusFailed,
			"error_message": errMsg,
			"completed_at":  &now,
		})
}

func normalizePhoneForWhatsmeow(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("empty phone number")
	}

	normalized := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r == '+':
			return r
		default:
			return -1
		}
	}, trimmed)

	if !strings.HasPrefix(normalized, "+") {
		normalized = "+" + normalized
	}

	parsed, err := phonenumbers.Parse(normalized, "ZZ")
	if err != nil || !phonenumbers.IsValidNumber(parsed) {
		return "", errors.New("invalid international format")
	}

	return phonenumbers.Format(parsed, phonenumbers.E164), nil
}
