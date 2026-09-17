package auth

import "testing"

// Ported from upstream CLIProxyAPI commit be7323f3bf66 (meta_refresh_test.go).
func TestMetaDCARefreshCredentialIsProviderScoped(t *testing.T) {
	for _, tc := range []struct {
		name string
		auth *Auth
		want bool
	}{
		{"meta", &Auth{Provider: "meta", Metadata: map[string]any{"dca_token": "dca:valid"}}, true},
		{"meta attributes", &Auth{Provider: "meta", Attributes: map[string]string{"dca_token": "dca:valid"}}, true},
		{"non-meta", &Auth{Provider: "codex", Metadata: map[string]any{"dca_token": "dca:valid"}}, false},
		{"empty", &Auth{Provider: "meta", Metadata: map[string]any{"dca_token": " "}}, false},
		{"existing oauth", &Auth{Provider: "codex", Metadata: map[string]any{"refresh_token": "refresh"}}, true},
		{"nil", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := authHasRefreshCredential(tc.auth); got != tc.want {
				t.Fatalf("refresh eligible = %v, want %v", got, tc.want)
			}
		})
	}
}

// A new device login can succeed while API key minting fails. Its DCA credential
// must not inherit the previous login's API key.
// Ported from upstream CLIProxyAPI commit 4a0131c062cd (metadata_merge_test.go).
func TestMergeExistingAuthMetadataMetaDoesNotRestoreOldKey(t *testing.T) {
	auth := &Auth{Provider: "meta", Metadata: map[string]any{"access_token": "dca:new", "dca_token": "dca:new"}}
	MergeExistingAuthMetadata(auth, map[string]any{"api_key": "LLM|old", "dca_expired": "old expiry", "dca_expires_at": 42, "priority": 3})
	for _, key := range []string{"api_key", "dca_expired", "dca_expires_at"} {
		if _, exists := auth.Metadata[key]; exists {
			t.Fatalf("restored old Meta credential field %s", key)
		}
	}
	if auth.Metadata["priority"] != 3 || auth.Metadata["dca_token"] != "dca:new" {
		t.Fatalf("incorrect merged metadata: %#v", auth.Metadata)
	}
}
