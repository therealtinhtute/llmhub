package openai

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/sdk/api/handlers"
	sdkconfig "github.com/therealtinhtute/llmhub/sdk/config"
)

// TestCodexClientModelsResponse_DevinDisplayName is ported from upstream
// CLIProxyAPI commit 28743473c11a ("feat(codex): append (Devin) suffix to
// Devin model display names").
func TestCodexClientModelsResponse_DevinDisplayName(t *testing.T) {
	devinClientID := "test-sdk-devin-models"
	openaiClientID := "test-sdk-openai-models"
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient(devinClientID, "devin", []*registry.ModelInfo{
		{
			ID:          "devin/swe-2",
			Object:      "model",
			OwnedBy:     "cognition",
			Type:        "devin",
			DisplayName: "SWE-2",
		},
		{
			ID:          "devin/gpt-6-astra",
			Object:      "model",
			OwnedBy:     "openai",
			Type:        "devin",
			DisplayName: "GPT-6 Astra",
		},
	})
	modelRegistry.RegisterClient(openaiClientID, "openai", []*registry.ModelInfo{
		{
			ID:          "standard-model",
			Object:      "model",
			OwnedBy:     "openai",
			Type:        "openai",
			DisplayName: "Standard Model",
		},
	})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(devinClientID)
		modelRegistry.UnregisterClient(openaiClientID)
	})

	base := handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil)
	handler := NewOpenAIAPIHandler(base)
	resp := handler.codexClientModelsResponse("0.153.4")
	models, ok := resp["models"].([]map[string]any)
	if !ok {
		t.Fatalf("models type = %T, want []map[string]any", resp["models"])
	}

	bySlug := make(map[string]map[string]any, len(models))
	for _, entry := range models {
		if slug, ok := entry["slug"].(string); ok {
			bySlug[slug] = entry
		}
	}

	if swe2 := bySlug["devin/swe-2"]; swe2 == nil {
		t.Fatal("missing devin/swe-2 entry")
	} else if got, _ := swe2["display_name"].(string); got != "SWE-2 (Devin)" {
		t.Errorf("devin/swe-2 display_name = %q, want SWE-2 (Devin)", got)
	}

	if astra := bySlug["devin/gpt-6-astra"]; astra == nil {
		t.Fatal("missing devin/gpt-6-astra entry")
	} else if got, _ := astra["display_name"].(string); got != "GPT-6 Astra (Devin)" {
		t.Errorf("devin/gpt-6-astra display_name = %q, want GPT-6 Astra (Devin)", got)
	}

	if std := bySlug["standard-model"]; std == nil {
		t.Fatal("missing standard-model entry")
	} else if got, _ := std["display_name"].(string); got != "Standard Model" {
		t.Errorf("standard-model display_name = %q, want Standard Model", got)
	}
}
