package handlers

import (
	"fmt"
	"strings"
)

const minJWTSecretLength = 32

func (a *App) jwtSecretBytes() ([]byte, error) {
	if a == nil || a.Config == nil {
		return nil, fmt.Errorf("app config is not initialized")
	}

	secret := strings.TrimSpace(a.Config.JWT.Secret)
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is not configured")
	}
	if len(secret) < minJWTSecretLength {
		return nil, fmt.Errorf("jwt secret must be at least %d characters", minJWTSecretLength)
	}

	return []byte(secret), nil
}
