package testutil

import "encoding/base64"

// TestJWT constructs a fake JWT token with the given JSON payload.
// The header and signature are dummy values.
func TestJWT(payload string) string {
	return "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
}

// TestAuthJSON constructs a minimal auth.json body with the given account
// metadata embedded in a fake JWT id_token.
func TestAuthJSON(accountID, email, name string) string {
	payload := `{"email":"` + email + `","name":"` + name + `"}`
	return `{"tokens":{"account_id":"` + accountID + `","id_token":"header.` + base64.RawURLEncoding.EncodeToString([]byte(payload)) + `.signature"}}`
}
