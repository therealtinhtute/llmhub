package cliproxy

import (
	"testing"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

func TestRegisterComboModelsListsComboNames(t *testing.T) {
	GlobalModelRegistry().UnregisterClient(comboModelsClientID)
	defer GlobalModelRegistry().UnregisterClient(comboModelsClientID)

	s := &Service{}
	s.registerComboModels(&config.Config{
		Combos: []internalconfig.ComboConfig{
			{Name: "daily", Strategy: "fallback", Models: []string{"claude/claude-opus-4-7"}},
			{Name: "rr", Strategy: "round-robin", StickyLimit: 3, Models: []string{"claude/claude-opus-4-7", "openrouter/deepseek-v4:free"}},
		},
	})

	listed := GlobalModelRegistry().GetAvailableModelsByProvider("combo")
	if len(listed) != 2 {
		t.Fatalf("expected 2 combo models listed, got %d", len(listed))
	}
	seen := map[string]bool{}
	for _, model := range listed {
		seen[model.ID] = true
	}
	for _, name := range []string{"daily", "rr"} {
		if !seen[name] {
			t.Fatalf("combo %q not present in registry listing; got %v", name, listed)
		}
	}
}

func TestRegisterComboModelsEmptyUnregisters(t *testing.T) {
	GlobalModelRegistry().UnregisterClient(comboModelsClientID)
	defer GlobalModelRegistry().UnregisterClient(comboModelsClientID)

	s := &Service{}
	s.registerComboModels(&config.Config{
		Combos: []internalconfig.ComboConfig{{Name: "daily", Strategy: "fallback", Models: []string{"claude/claude-opus-4-7"}}},
	})
	if got := len(GlobalModelRegistry().GetAvailableModelsByProvider("combo")); got != 1 {
		t.Fatalf("expected 1 combo listed after register, got %d", got)
	}

	// Reload without combos (or nil config) must remove them from the listing.
	s.registerComboModels(&config.Config{})
	if got := len(GlobalModelRegistry().GetAvailableModelsByProvider("combo")); got != 0 {
		t.Fatalf("expected 0 combos listed after empty reload, got %d", got)
	}
	s.registerComboModels(nil)
	if got := len(GlobalModelRegistry().GetAvailableModelsByProvider("combo")); got != 0 {
		t.Fatalf("expected 0 combos listed after nil config, got %d", got)
	}
}
