package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClientCodexSectionParses covers the additive client.codex section
// adapted from upstream 63c04b4bd4: both flags parse and default to false.
func TestClientCodexSectionParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	payload := "client:\n  codex:\n    optimize-multi-agent-v2: true\n    enable-apply-patch: true\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Client.Codex.OptimizeMultiAgentV2 {
		t.Fatal("client.codex.optimize-multi-agent-v2 = false, want true")
	}
	if !cfg.Client.Codex.EnableApplyPatch {
		t.Fatal("client.codex.enable-apply-patch = false, want true")
	}
	if !cfg.CodexMultiAgentV2Enabled() {
		t.Fatal("CodexMultiAgentV2Enabled() = false, want true")
	}
	if !cfg.CodexApplyPatchEnabled() {
		t.Fatal("CodexApplyPatchEnabled() = false, want true")
	}
}

// TestClientCodexSectionDefaultsOff ensures the section is zero-valued when
// absent so existing configs keep current behavior.
func TestClientCodexSectionDefaultsOff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Client.Codex.OptimizeMultiAgentV2 || cfg.Client.Codex.EnableApplyPatch {
		t.Fatalf("client.codex defaults = %+v, want all false", cfg.Client.Codex)
	}
	if cfg.CodexMultiAgentV2Enabled() || cfg.CodexApplyPatchEnabled() {
		t.Fatal("effective codex client flags = true, want false")
	}
}

// TestFlatCodexOptimizeMultiAgentV2KeepsWorking ensures the legacy flat field
// still enables the behavior additively alongside the new section.
func TestFlatCodexOptimizeMultiAgentV2KeepsWorking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	payload := "codex-optimize-multi-agent-v2: true\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.CodexOptimizeMultiAgentV2 {
		t.Fatal("flat codex-optimize-multi-agent-v2 = false, want true")
	}
	if !cfg.CodexMultiAgentV2Enabled() {
		t.Fatal("CodexMultiAgentV2Enabled() = false, want true via legacy flat field")
	}
	if cfg.CodexApplyPatchEnabled() {
		t.Fatal("CodexApplyPatchEnabled() = true, want false (flat field must not enable it)")
	}
}

// TestClientCodexSectionSurvivesRuntimeClone ensures CloneForRuntime preserves
// the new section for concurrent request-handler snapshots.
func TestClientCodexSectionSurvivesRuntimeClone(t *testing.T) {
	cfg := &Config{}
	cfg.Client.Codex.OptimizeMultiAgentV2 = true
	cfg.Client.Codex.EnableApplyPatch = true
	cloned := cfg.CloneForRuntime()
	if cloned == nil {
		t.Fatal("CloneForRuntime() = nil")
	}
	if !cloned.CodexMultiAgentV2Enabled() || !cloned.CodexApplyPatchEnabled() {
		t.Fatalf("cloned client.codex = %+v, want both true", cloned.Client.Codex)
	}
}
