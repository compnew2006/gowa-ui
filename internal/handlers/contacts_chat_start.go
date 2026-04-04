package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	whatsmeowpkg "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/nyaruka/phonenumbers"
	waTypes "go.mau.fi/whatsmeow/types"
)

var (
	errWhatsmeowDirectChatUnavailable  = errors.New("whatsmeow contact lookup is unavailable")
	errWhatsmeowDirectChatInvalidPhone = errors.New("phone_number must be a valid international phone number with country code")
	errWhatsmeowDirectChatNotFound     = errors.New("phone number is not registered on WhatsApp")
)

// ResolvedWhatsmeowDirectContact contains the canonical recipient details for a new direct chat.
type ResolvedWhatsmeowDirectContact struct {
	CanonicalPhone string
	ProfileName    string
}

// WhatsmeowContactResolver resolves recipient details before opening a new direct chat.
type WhatsmeowContactResolver interface {
	ResolveDirectContact(ctx context.Context, instance *models.WhatsAppInstance, phone string) (*ResolvedWhatsmeowDirectContact, error)
}

type liveWhatsmeowContactResolver struct {
	manager *whatsmeowpkg.ConnectionManager
}

func (a *App) resolveWhatsmeowContactResolver() WhatsmeowContactResolver {
	if a != nil && a.WhatsmeowContactResolver != nil {
		return a.WhatsmeowContactResolver
	}
	return &liveWhatsmeowContactResolver{manager: a.WhatsmeowManager}
}

func (r *liveWhatsmeowContactResolver) ResolveDirectContact(
	ctx context.Context,
	instance *models.WhatsAppInstance,
	phone string,
) (*ResolvedWhatsmeowDirectContact, error) {
	if r == nil || r.manager == nil || instance == nil {
		return nil, errWhatsmeowDirectChatUnavailable
	}

	client := r.manager.GetClient(instance.ID)
	if client == nil || !client.IsConnected() {
		return nil, errWhatsmeowDirectChatUnavailable
	}

	normalizedPhone, err := normalizeWhatsmeowDirectChatPhone(phone)
	if err != nil {
		return nil, err
	}

	results, err := client.IsOnWhatsApp(ctx, []string{normalizedPhone})
	if err != nil {
		return nil, fmt.Errorf("failed to verify phone number on WhatsApp: %w", err)
	}
	if len(results) == 0 || !results[0].IsIn {
		return nil, errWhatsmeowDirectChatNotFound
	}

	canonicalPhone := strings.TrimSpace(results[0].JID.User)
	if canonicalPhone == "" {
		canonicalPhone = strings.TrimPrefix(normalizedPhone, "+")
	}

	return &ResolvedWhatsmeowDirectContact{
		CanonicalPhone: canonicalPhone,
		ProfileName:    resolveVerifiedBusinessName(results[0].VerifiedName),
	}, nil
}

func normalizeWhatsmeowDirectChatPhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errWhatsmeowDirectChatInvalidPhone
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
	if normalized == "" {
		return "", errWhatsmeowDirectChatInvalidPhone
	}

	if strings.Count(normalized, "+") > 1 || strings.Contains(normalized[1:], "+") {
		return "", errWhatsmeowDirectChatInvalidPhone
	}
	if strings.HasPrefix(normalized, "00") {
		normalized = "+" + normalized[2:]
	}
	if !strings.HasPrefix(normalized, "+") {
		normalized = "+" + normalized
	}

	parsed, err := phonenumbers.Parse(normalized, "ZZ")
	if err != nil || !phonenumbers.IsValidNumber(parsed) {
		return "", errWhatsmeowDirectChatInvalidPhone
	}

	return phonenumbers.Format(parsed, phonenumbers.E164), nil
}

func resolveVerifiedBusinessName(verifiedName *waTypes.VerifiedName) string {
	if verifiedName == nil || verifiedName.Details == nil {
		return ""
	}
	return strings.TrimSpace(verifiedName.Details.GetVerifiedName())
}
