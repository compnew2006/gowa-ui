package handlers

import "fmt"

const minPasswordLength = 12

func validatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	var hasLower, hasUpper, hasDigit bool
	for _, ch := range password {
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit {
		return fmt.Errorf("password must include at least one uppercase letter, one lowercase letter, and one number")
	}

	return nil
}
