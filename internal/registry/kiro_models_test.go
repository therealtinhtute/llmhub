package registry

import "testing"

func TestGetKiroModelsUsesBuiltinsWhenCatalogMissingKiro(t *testing.T) {
	modelsCatalogStore.mu.Lock()
	previous := modelsCatalogStore.data
	modelsCatalogStore.data = &staticModelsJSON{}
	modelsCatalogStore.mu.Unlock()
	t.Cleanup(func() {
		modelsCatalogStore.mu.Lock()
		modelsCatalogStore.data = previous
		modelsCatalogStore.mu.Unlock()
	})

	models := GetKiroModels()
	if len(models) != 5 {
		t.Fatalf("GetKiroModels() len = %d, want 5 builtins", len(models))
	}
	if models[0].ID != kiroBuiltinAutoModelID {
		t.Fatalf("first Kiro model = %q, want %q", models[0].ID, kiroBuiltinAutoModelID)
	}

	foundThinkingAgentic := false
	for _, model := range models {
		if model.ID != "claude-sonnet-4.5-thinking-agentic" {
			continue
		}
		foundThinkingAgentic = true
		if model.Thinking == nil || len(model.Thinking.Levels) != 3 {
			t.Fatalf("thinking-agentic model missing thinking levels: %#v", model.Thinking)
		}
	}
	if !foundThinkingAgentic {
		t.Fatal("missing Kiro thinking-agentic builtin")
	}
}

func TestDetectChangedProvidersIncludesKiro(t *testing.T) {
	changed := detectChangedProviders(&staticModelsJSON{
		Kiro: []*ModelInfo{{ID: "auto"}},
	}, &staticModelsJSON{
		Kiro: []*ModelInfo{{ID: "auto"}, {ID: "claude-sonnet-4.5"}},
	})
	if len(changed) != 1 || changed[0] != "kiro" {
		t.Fatalf("changed providers = %#v, want [kiro]", changed)
	}
}
