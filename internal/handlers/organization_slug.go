package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	errInvalidOrganizationSlug = errors.New("invalid organization slug")
	errOrganizationSlugTaken   = errors.New("organization slug is already in use")
)

func normalizeOrganizationSlug(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	prevDash := false
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			prevDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if builder.Len() == 0 || prevDash {
				continue
			}
			builder.WriteByte('-')
			prevDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func resolveOrganizationSlug(db *gorm.DB, requestedSlug, name string, excludeID uuid.UUID) (string, error) {
	baseSlug := normalizeOrganizationSlug(requestedSlug)
	if baseSlug != "" {
		if err := ensureOrganizationSlugAvailable(db, baseSlug, excludeID); err != nil {
			return "", err
		}
		return baseSlug, nil
	}

	baseSlug = normalizeOrganizationSlug(name)
	if baseSlug == "" {
		return "", errInvalidOrganizationSlug
	}

	slug := baseSlug
	for suffix := 2; ; suffix++ {
		err := ensureOrganizationSlugAvailable(db, slug, excludeID)
		if err == nil {
			return slug, nil
		}
		if !errors.Is(err, errOrganizationSlugTaken) {
			return "", err
		}
		slug = baseSlug + "-" + strconv.Itoa(suffix)
	}
}

func ensureOrganizationSlugAvailable(db *gorm.DB, slug string, excludeID uuid.UUID) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	normalizedSlug := normalizeOrganizationSlug(slug)
	if normalizedSlug == "" {
		return errInvalidOrganizationSlug
	}

	query := db.Model(&struct {
		ID uuid.UUID
	}{}).Table("organizations").Where("slug = ? AND deleted_at IS NULL", normalizedSlug)
	if excludeID != uuid.Nil {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errOrganizationSlugTaken
	}
	return nil
}
