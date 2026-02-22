package handlers

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
)

const maxWebhookEventsPerRequest = 500

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
	unique := make(map[string]struct{})

	for _, entry := range payload.Entry {
		if entry.ID != "" {
			secretsByBusiness, err := a.lookupAppSecretsByBusinessID(entry.ID)
			if err == nil {
				for _, secret := range secretsByBusiness {
					unique[secret] = struct{}{}
				}
			}
		}

		for _, change := range entry.Changes {
			phoneNumberID := change.Value.Metadata.PhoneNumberID
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
