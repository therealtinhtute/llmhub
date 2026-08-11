package config

import (
	"strings"
	"testing"
)

func comboYAML(combos string) *Config {
	cfg, err := ParseConfigBytes([]byte("combos:\n" + combos))
	if err != nil {
		return nil
	}
	return cfg
}

func TestValidateCombosEmptyName(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte("combos:\n  - name: \"\"\n    models: [claude/claude-opus-4-7]"))
	if err == nil {
		t.Fatalf("expected error for empty combo name, got config %+v", cfg)
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosDuplicateName(t *testing.T) {
	_, err := ParseConfigBytes([]byte("combos:\n  - name: daily\n    models: [claude/claude-opus-4-7]\n  - name: daily\n    models: [openrouter/deepseek-v4:free]"))
	if err == nil {
		t.Fatal("expected error for duplicate combo name")
	}
	if !strings.Contains(err.Error(), "duplicate name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosCollidesWithModelID(t *testing.T) {
	_, err := ParseConfigBytes([]byte("combos:\n  - name: claude-sonnet-4-6\n    models: [claude/claude-opus-4-7]"))
	if err == nil {
		t.Fatal("expected error for combo name colliding with a registered model id")
	}
	if !strings.Contains(err.Error(), "collides with a registered model id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosZeroCandidates(t *testing.T) {
	_, err := ParseConfigBytes([]byte("combos:\n  - name: daily\n    models: []"))
	if err == nil {
		t.Fatal("expected error for zero candidates")
	}
	if !strings.Contains(err.Error(), "at least one candidate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosMalformedCandidate(t *testing.T) {
	for _, bad := range []string{"no-slash", "provider/", "/model"} {
		_, err := ParseConfigBytes([]byte("combos:\n  - name: daily\n    models: [" + bad + "]"))
		if err == nil {
			t.Fatalf("expected error for candidate %q", bad)
		}
		if !strings.Contains(err.Error(), "must parse as provider/model") {
			t.Fatalf("candidate %q: unexpected error: %v", bad, err)
		}
	}
}

func TestValidateCombosCandidateIsComboName(t *testing.T) {
	_, err := ParseConfigBytes([]byte("combos:\n  - name: daily\n    models: [claude/claude-opus-4-7]\n  - name: backup\n    models: [openrouter/daily]"))
	if err == nil {
		t.Fatal("expected error when a candidate model is itself a combo name")
	}
	if !strings.Contains(err.Error(), "must not be another combo name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosInvalidStrategy(t *testing.T) {
	_, err := ParseConfigBytes([]byte("combos:\n  - name: daily\n    strategy: turbo\n    models: [claude/claude-opus-4-7]"))
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
	if !strings.Contains(err.Error(), "must be fallback or round-robin") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCombosNormalizations(t *testing.T) {
	cfg := comboYAML("  - name: daily\n    models: [claude/claude-opus-4-7, openrouter/deepseek-v4:free]\n  - name: rr\n    strategy: round-robin\n    sticky-limit: 0\n    models: [opencode/grok-code]\n  - name: explicit\n    strategy: fallback\n    models: [claude/claude-opus-4-7]")
	if cfg == nil {
		t.Fatal("expected valid combos config to parse")
	}
	if len(cfg.Combos) != 3 {
		t.Fatalf("expected 3 combos, got %d", len(cfg.Combos))
	}
	if cfg.Combos[0].Strategy != "fallback" {
		t.Fatalf("empty strategy should normalize to fallback, got %q", cfg.Combos[0].Strategy)
	}
	if cfg.Combos[1].StickyLimit != 1 {
		t.Fatalf("sticky-limit 0 should normalize to 1, got %d", cfg.Combos[1].StickyLimit)
	}
	if cfg.Combos[2].Strategy != "fallback" {
		t.Fatalf("explicit fallback strategy should be kept, got %q", cfg.Combos[2].Strategy)
	}
}
