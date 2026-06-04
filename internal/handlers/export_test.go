package handlers

// IsFacebookUserCantDMError exposes the unexported helper for use from
// the external handlers_test package. Used by the 10903 unit test in
// fb_comments_test.go.
func IsFacebookUserCantDMError(err error) bool {
	return isFacebookUserCantDMError(err)
}
