package handlers

import (
	"github.com/compnew2006/whatomate/internal/models"
)

// IsFacebookUserCantDMError exposes the unexported helper for use from
// the external handlers_test package. Used by the 10903 unit test in
// fb_comments_test.go.
func IsFacebookUserCantDMError(err error) bool {
	return isFacebookUserCantDMError(err)
}

// NormalizeFacebookCommentForSave exposes the unexported helper for use from
// the external handlers_test package.
func NormalizeFacebookCommentForSave(comment *models.FacebookComment) {
	normalizeFacebookCommentForSave(comment)
}
