package facebookoauth

import "testing"

func TestOAuthStateTableName(t *testing.T) {
	t.Parallel()

	if got := (OAuthState{}).TableName(); got != "facebook_oauth_states" {
		t.Fatalf("OAuthState.TableName() = %q, want %q", got, "facebook_oauth_states")
	}
}
