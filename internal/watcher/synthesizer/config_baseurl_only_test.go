package synthesizer

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

// TestGeminiBaseURLOnlyCredentialDoesNotOrphanExistingIDs pins the R13 migration
// contract: existing api-key credentials keep byte-identical auth IDs after
// base_URL-only entries join the config, so cooldown and index mappings keyed by
// auth ID survive; base_URL-only entries synthesize as new auths with empty keys.
func TestGeminiBaseURLOnlyCredentialDoesNotOrphanExistingIDs(t *testing.T) {
	existing := config.GeminiKey{APIKey: "key-1", BaseURL: "https://proxy.example.com"}
	before := &config.Config{GeminiKey: []config.GeminiKey{existing}}
	sBefore := &ConfigSynthesizer{}
	cfgBefore := &SynthesisContext{Config: before, IDGenerator: NewStableIDGenerator()}
	authsBefore := sBefore.synthesizeGeminiKeys(cfgBefore)
	if len(authsBefore) != 1 {
		t.Fatalf("expected 1 auth before change, got %d", len(authsBefore))
	}

	after := &config.Config{GeminiKey: []config.GeminiKey{
		existing,
		{APIKey: "", BaseURL: "https://relay.example.net"},
	}}
	sAfter := &ConfigSynthesizer{}
	authsAfter := sAfter.synthesizeGeminiKeys(&SynthesisContext{Config: after, IDGenerator: NewStableIDGenerator()})
	if len(authsAfter) != 2 {
		t.Fatalf("expected 2 auths after adding base_URL-only credential, got %d", len(authsAfter))
	}

	if authsAfter[0].ID != authsBefore[0].ID {
		t.Fatalf("existing credential ID changed: %q -> %q (would orphan cooldown mappings)", authsBefore[0].ID, authsAfter[0].ID)
	}
	baseOnly := authsAfter[1]
	if v := baseOnly.Attributes["api_key"]; v != "" {
		t.Fatalf("base_URL-only credential must not fabricate an api key, got %q", v)
	}
	if baseOnly.Attributes["base_url"] != "https://relay.example.net" {
		t.Fatalf("base_url attribute missing: %#v", baseOnly.Attributes)
	}
	if kind := baseOnly.Attributes["auth_kind"]; kind != "apikey" {
		t.Fatalf("base_URL-only credential should carry auth_kind=apikey, got %q", kind)
	}
}

func TestSanitizeGeminiKeysKeepsBaseURLOnlyEntries(t *testing.T) {
	cfg := &config.Config{GeminiKey: []config.GeminiKey{
		{APIKey: "k1"},
		{APIKey: "", BaseURL: ""},           // dropped: neither key nor base URL
		{APIKey: "", BaseURL: "https://r"}, // kept: base_URL-only
	}}
	cfg.SanitizeGeminiKeys()
	if len(cfg.GeminiKey) != 2 {
		t.Fatalf("expected 2 surviving entries (dropped only fully-empty), got %d: %#v", len(cfg.GeminiKey), cfg.GeminiKey)
	}
	if cfg.GeminiKey[1].BaseURL != "https://r" {
		t.Fatalf("base_URL-only entry lost: %#v", cfg.GeminiKey[1])
	}
}
