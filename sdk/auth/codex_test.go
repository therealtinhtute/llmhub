package auth

import (
	"testing"
	"time"
)

// Upstream 9812b1e76872 reduced the Codex refresh lead from five days to 24
// hours. Local symbol under test: CodexAuthenticator.RefreshLead.
func TestCodexAuthenticator_RefreshLeadIs24Hours(t *testing.T) {
	lead := NewCodexAuthenticator().RefreshLead()
	if lead == nil {
		t.Fatal("RefreshLead() = nil, want non-nil")
	}
	if *lead != 24*time.Hour {
		t.Fatalf("RefreshLead() = %v, want 24h", *lead)
	}
}
