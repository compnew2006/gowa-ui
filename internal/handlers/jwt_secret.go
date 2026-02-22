package handlers

import (
	"fmt"
	"strings"
)

func (a *App) jwtSecretBytes() ([]byte, error) {
	if a == nil || a.Config == nil {
		return nil, fmt.Errorf("app config is not initialized")
	}

	secret := strings.TrimSpace(a.Config.JWT.Secret)
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is not configured")
	}

	return []byte(secret), nil
}
