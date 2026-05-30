package models

import (
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

var variablePattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

var previewSampleValues = map[string]string{
	"name":              "John",
	"phone":             "+1234567890",
	"date":              "2026-05-30",
	"time":              "10:00 AM",
	"group_name":        "Marketing Team",
	"customer_name":     "Jane",
	"contact_name":      "Sam",
	"agent_name":        "Alex",
	"organization_name": "Acme Inc",
	"phone_number":      "+1987654321",
	"chat_id":           "12345",
}

func ExtractVariables(body string) StringArray {
	matches := variablePattern.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		v := "{" + m[1] + "}"
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	sort.Strings(result)
	if result == nil {
		result = []string{}
	}
	return result
}

func RenderPreview(body string) string {
	replacerPairs := make([]string, 0, len(previewSampleValues)*2)
	for key, value := range previewSampleValues {
		replacerPairs = append(replacerPairs, "{"+key+"}", value)
	}
	return strings.NewReplacer(replacerPairs...).Replace(body)
}

type SavedContent struct {
	BaseModel
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_saved_contents_org_name" json:"organization_id"`
	Name           string     `gorm:"size:255;not null;uniqueIndex:idx_saved_contents_org_name" json:"name"`
	Body           string     `gorm:"type:text;not null" json:"body"`
	Variables      StringArray `gorm:"type:jsonb;default:'[]'" json:"variables"`
	Category       string     `gorm:"size:100" json:"category"`
	CreatedByID    uuid.UUID  `gorm:"type:uuid" json:"created_by_id"`

	// Media attachment
	MediaID       string `gorm:"size:255" json:"media_id"`
	MediaFilename string `gorm:"size:255" json:"media_filename"`
	MediaMimeType string `gorm:"size:100" json:"media_mime_type"`
	MediaLocalPath string `gorm:"size:500" json:"media_local_path"`

	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	CreatedBy    *User         `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
}

func (SavedContent) TableName() string {
	return "saved_contents"
}
