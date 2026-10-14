package openai

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
)

// TestCodexClientModelsResponseFiltersExtendedReasoningByClientVersion covers
// the client_version forwarding path: older Codex CLIs (< 0.144.0) must not
// advertise max/ultra reasoning levels, while modern, empty, and unparseable
// versions keep them. Adapted from upstream 1cc72b9d1.
func TestCodexClientModelsResponseFiltersExtendedReasoningByClientVersion(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient("codex-version-filter-sdk-test", "openai", []*registry.ModelInfo{
		{
			ID:          "gpt-5.6-sol",
			Object:      "model",
			OwnedBy:     "openai",
			DisplayName: "GPT-5.6-Sol",
		},
	})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient("codex-version-filter-sdk-test")
	})

	handler := &OpenAIAPIHandler{}

	effortsFor := func(version string) []string {
		t.Helper()
		resp := handler.codexClientModelsResponse(version)
		models, ok := resp["models"].([]map[string]any)
		if !ok {
			t.Fatalf("version %q: models type = %T, want []map[string]any", version, resp["models"])
		}
		for _, entry := range models {
			if stringModelValue(entry, "slug") != "gpt-5.6-sol" {
				continue
			}
			levels, _ := entry["supported_reasoning_levels"].([]any)
			efforts := make([]string, 0, len(levels))
			for _, rawLevel := range levels {
				level, _ := rawLevel.(map[string]any)
				efforts = append(efforts, stringModelValue(level, "effort"))
			}
			return efforts
		}
		t.Fatalf("version %q: expected gpt-5.6-sol in catalog", version)
		return nil
	}

	contains := func(efforts []string, effort string) bool {
		for _, e := range efforts {
			if e == effort {
				return true
			}
		}
		return false
	}

	legacy := effortsFor("0.137.0")
	if contains(legacy, "max") || contains(legacy, "ultra") {
		t.Fatalf("0.137.0 advertised extended reasoning: %#v", legacy)
	}
	if len(legacy) != 4 {
		t.Fatalf("0.137.0 expected 4 legacy reasoning levels, got %#v", legacy)
	}

	modern := effortsFor("0.149.1")
	if !contains(modern, "ultra") || !contains(modern, "max") {
		t.Fatalf("0.149.1 expected max/ultra reasoning, got %#v", modern)
	}

	for _, version := range []string{"", "pi", "latest"} {
		unparsed := effortsFor(version)
		if !contains(unparsed, "ultra") {
			t.Fatalf("version %q expected ultra to be preserved, got %#v", version, unparsed)
		}
	}
}
