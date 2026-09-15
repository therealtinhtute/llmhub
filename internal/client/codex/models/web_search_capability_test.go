package models

import "testing"

// Ported from upstream CLIProxyAPI 4311ae874774 (web_search_capability_test.go).
// Local symbol under test: applyCPAWebSearchCapability.

func TestCPAWebSearchCapabilityNeverTrustsTemplateClaims(t *testing.T) {
	for _, clientVersion := range []string{"cpa", "CPA", "codex", ""} {
		t.Run(clientVersion, func(t *testing.T) {
			entry := map[string]any{"cpa_capabilities": map[string]any{"web_search": true}}
			applyCPAWebSearchCapability(entry, "unknown", func(string) *bool { return nil }, clientVersion)
			if _, exists := entry["cpa_capabilities"]; exists {
				t.Fatal("inherited template capability should be removed")
			}
		})
	}
}

func TestCPAWebSearchCapabilityOverridesTemplateWithExplicitFalse(t *testing.T) {
	entry := map[string]any{"cpa_capabilities": map[string]any{"web_search": true}}
	unsupported := false
	applyCPAWebSearchCapability(entry, "known-unsupported", func(string) *bool { return &unsupported }, "cpa")
	capabilities, ok := entry["cpa_capabilities"].(map[string]any)
	if !ok || capabilities["web_search"] != false {
		t.Fatal("explicit registered false must override template claims")
	}
}

// TestCodexClientModelsResponse_CPAWebSearchCapabilities mirrors the upstream
// 4311ae874774 models_test.go expectations against local symbol
// BuildResponseForClientWithCPACapabilities.
func TestCodexClientModelsResponse_CPAWebSearchCapabilities(t *testing.T) {
	trueValue, falseValue := true, false
	capabilities := map[string]*bool{
		"supported-model":   &trueValue,
		"unsupported-model": &falseValue,
	}
	availableModels := []map[string]any{
		{"id": "supported-model"},
		{"id": "unsupported-model"},
		{"id": "unknown-model"},
	}

	resp := BuildResponseForClientWithCPACapabilities(availableModels, nil, func(id string) *bool {
		return capabilities[id]
	}, false, "cpa")
	models, ok := resp["models"].([]map[string]any)
	if !ok || len(models) != len(availableModels) {
		t.Fatalf("models = %#v, want %d models", resp["models"], len(availableModels))
	}

	entries := make(map[string]map[string]any, len(models))
	for _, model := range models {
		entries[stringModelValue(model, "slug")] = model
	}
	assertCPAWebSearchCapability(t, entries["supported-model"], true, true)
	assertCPAWebSearchCapability(t, entries["unsupported-model"], false, true)
	assertCPAWebSearchCapability(t, entries["unknown-model"], false, false)
}

// TestCodexClientModelsResponse_CPAWebSearchCapabilitiesOnlyForCPAClient mirrors
// the upstream 4311ae874774 gate: cpa_capabilities must only appear for the
// exact "cpa" client version.
func TestCodexClientModelsResponse_CPAWebSearchCapabilitiesOnlyForCPAClient(t *testing.T) {
	capabilityLookup := func(string) *bool { value := true; return &value }
	for _, clientVersion := range []string{"", "0.153.4", "CPA", "cpa-preview"} {
		resp := BuildResponseForClientWithCPACapabilities([]map[string]any{{"id": "gpt-5.5"}}, nil, capabilityLookup, false, clientVersion)
		models, ok := resp["models"].([]map[string]any)
		if !ok || len(models) != 1 {
			t.Fatalf("client version %q models = %#v, want one model", clientVersion, resp["models"])
		}
		assertCPAWebSearchCapability(t, models[0], false, false)
	}
}

func assertCPAWebSearchCapability(t *testing.T, entry map[string]any, want bool, present bool) {
	t.Helper()
	if entry == nil {
		t.Fatal("missing model entry")
	}
	raw, exists := entry["cpa_capabilities"]
	if !exists {
		if present {
			t.Fatalf("cpa_capabilities missing, want web_search=%v", want)
		}
		return
	}
	if !present {
		t.Fatalf("cpa_capabilities = %#v, want omitted", raw)
	}
	capabilities, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("cpa_capabilities = %#v, want object", raw)
	}
	if got, ok := capabilities["web_search"].(bool); !ok || got != want {
		t.Fatalf("web_search = %#v, want %v", capabilities["web_search"], want)
	}
}
