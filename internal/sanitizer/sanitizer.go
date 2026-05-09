package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

var messagePolicy *bluemonday.Policy

func init() {
	messagePolicy = bluemonday.StrictPolicy()
}

func SanitizeMessageContent(input string) string {
	if input == "" {
		return ""
	}
	return messagePolicy.Sanitize(input)
}
