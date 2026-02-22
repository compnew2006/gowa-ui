package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/pkg/whatsapp"
)

func (a *App) getConfiguredWhatsAppClient() *whatsapp.Client {
	if a.WhatsApp != nil {
		return a.WhatsApp
	}

	baseURL := whatsapp.BaseURL
	if a.Config != nil {
		if configured := strings.TrimSpace(a.Config.WhatsApp.BaseURL); configured != "" {
			baseURL = configured
		}
	}

	return whatsapp.NewWithBaseURL(a.Log, baseURL)
}
