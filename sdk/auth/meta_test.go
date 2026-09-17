package auth

import (
	"testing"
)

// TestMetaAuthenticator covers Provider and RefreshLead.
// Ported from upstream CLIProxyAPI sdk/auth/meta_test.go (54d4f4c0193c, d09042a54810).
func TestMetaAuthenticator(t *testing.T) {
	authenticator := NewMetaAuthenticator()
	if authenticator.Provider() != "meta" {
		t.Errorf("expected provider 'meta', got '%s'", authenticator.Provider())
	}
	if lead := authenticator.RefreshLead(); lead != nil {
		t.Errorf("expected nil refresh lead, got %v", lead)
	}
}
