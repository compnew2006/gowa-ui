package handlers

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
)

const maxWebhookEventsPerRequest = 500
const maxWebhookBodyBytes = 5 * 1024 * 1024

type webhookSecretCandidate struct {
	businessID     string
	phoneNumberIDs []string
}

type webhookSignaturePayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func (a *App) validateWebhookRequest(body, signature []byte, payload *WebhookPayload) error {
	if len(signature) == 0 {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	secrets, err := a.collectWebhookAppSecrets(payload)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if verifyWebhookSignature(body, signature, []byte(secret)) {
			return nil
		}
	}

	return fmt.Errorf("invalid webhook signature")
}

func (a *App) collectWebhookAppSecrets(payload *WebhookPayload) ([]string, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload is nil")
	}
	return a.collectWebhookAppSecretsForCandidates(webhookSecretCandidatesFromPayload(payload))
}

func (a *App) validateWebhookSignaturePayload(body, signature []byte, payload *webhookSignaturePayload) error {
	if len(signature) == 0 {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	secrets, err := a.collectWebhookAppSecretsForCandidates(webhookSecretCandidatesFromSignaturePayload(payload))
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if verifyWebhookSignature(body, signature, []byte(secret)) {
			return nil
		}
	}

	return fmt.Errorf("invalid webhook signature")
}

func (a *App) collectWebhookAppSecretsForCandidates(candidates []webhookSecretCandidate) ([]string, error) {
	unique := make(map[string]struct{})

	for _, candidate := range candidates {
		if candidate.businessID != "" {
			secretsByBusiness, err := a.lookupAppSecretsByBusinessID(candidate.businessID)
			if err == nil {
				for _, secret := range secretsByBusiness {
					unique[secret] = struct{}{}
				}
			}
		}

		for _, phoneNumberID := range candidate.phoneNumberIDs {
			if phoneNumberID == "" {
				continue
			}

			secret, err := a.lookupAppSecretByPhoneID(phoneNumberID)
			if err != nil {
				continue
			}
			unique[secret] = struct{}{}
		}
	}

	if len(unique) == 0 {
		return nil, fmt.Errorf("no app secret configured for incoming webhook payload")
	}

	secrets := make([]string, 0, len(unique))
	for secret := range unique {
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

func webhookSecretCandidatesFromPayload(payload *WebhookPayload) []webhookSecretCandidate {
	candidates := make([]webhookSecretCandidate, 0, len(payload.Entry))
	for _, entry := range payload.Entry {
		candidate := webhookSecretCandidate{
			businessID:     entry.ID,
			phoneNumberIDs: make([]string, 0, len(entry.Changes)),
		}
		for _, change := range entry.Changes {
			candidate.phoneNumberIDs = append(candidate.phoneNumberIDs, change.Value.Metadata.PhoneNumberID)
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func webhookSecretCandidatesFromSignaturePayload(payload *webhookSignaturePayload) []webhookSecretCandidate {
	candidates := make([]webhookSecretCandidate, 0, len(payload.Entry))
	for _, entry := range payload.Entry {
		candidate := webhookSecretCandidate{
			businessID:     entry.ID,
			phoneNumberIDs: make([]string, 0, len(entry.Changes)),
		}
		for _, change := range entry.Changes {
			candidate.phoneNumberIDs = append(candidate.phoneNumberIDs, change.Value.Metadata.PhoneNumberID)
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (a *App) lookupAppSecretByPhoneID(phoneID string) (string, error) {
	account, err := a.getWhatsAppAccountCached(phoneID)
	if err != nil {
		return "", err
	}
	if account.AppSecret == "" {
		return "", fmt.Errorf("app secret is empty for phone_id %s", phoneID)
	}
	return account.AppSecret, nil
}

func (a *App) lookupAppSecretsByBusinessID(businessID string) ([]string, error) {
	var accounts []models.WhatsAppAccount
	if err := a.DB.Where("business_id = ? AND app_secret <> ''", businessID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no app secret configured for business_id %s", businessID)
	}

	secrets := make([]string, 0, len(accounts))
	for i := range accounts {
		if err := a.decryptAccountSecrets(&accounts[i]); err != nil {
			a.Log.Warn("Failed to decrypt app secret for webhook business lookup", "business_id", businessID, "account_id", accounts[i].ID, "error", err)
			continue
		}
		if accounts[i].AppSecret != "" {
			secrets = append(secrets, accounts[i].AppSecret)
		}
	}

	if len(secrets) == 0 {
		return nil, fmt.Errorf("no decryptable app secret configured for business_id %s", businessID)
	}

	return secrets, nil
}

func countWebhookEvents(payload *WebhookPayload) int {
	total := 0
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			total += len(change.Value.Messages)
			total += len(change.Value.Statuses)
			if change.Field == "message_template_status_update" {
				total++
			}
		}
	}
	return total
}
