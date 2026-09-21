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

// TestOpenAIModels_DoesNotExposeMetadataModelID is ported from upstream
// CLIProxyAPI commit 8f23ad029144: internal metadata fields must not leak into
// public model-list responses.
func TestOpenAIModels_DoesNotExposeMetadataModelID(t *testing.T) {
	clientID := "openai-models-no-metadata-id-test"
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient(clientID, "codex", []*registry.ModelInfo{
		{
			ID:              "codex-main",
			MetadataModelID: "gpt-6-astra",
			DisplayName:     "GPT 6.0 Astra",
		},
	})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	openaiModels := modelRegistry.GetAvailableModels("openai")
	for _, m := range openaiModels {
		if _, exists := m["metadata_model_id"]; exists {
			t.Fatalf("model map exposed internal metadata_model_id: %#v", m)
		}
		if _, exists := m["MetadataModelID"]; exists {
			t.Fatalf("model map exposed internal MetadataModelID: %#v", m)
		}
	}
}

func testIntModelValue(model map[string]any, key string) int {
	if model == nil {
		return 0
	}
	switch value := model[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	}
	return 0
}

// TestCodexClientModelsResponse_OAuthAliasesIntegration is ported from upstream
// CLIProxyAPI commit 8f23ad029144: OAuth aliases must inherit the canonical
// template metadata in the Codex client models response.
func TestCodexClientModelsResponse_OAuthAliasesIntegration(t *testing.T) {
	clientID := "codex-client-models-integration-test"
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient(clientID, "codex", []*registry.ModelInfo{
		{
			ID:              "codex-main",
			MetadataModelID: "gpt-6-astra",
			DisplayName:     "GPT 6.0 Astra",
		},
		{
			ID:              "codex-luna",
			MetadataModelID: "gpt-5.6-luna",
			DisplayName:     "GPT 5.6 Luna",
		},
	})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	base := handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil)
	handler := NewOpenAIAPIHandler(base)
	resp := handler.codexClientModelsResponse("0.153.4")
	models, ok := resp["models"].([]map[string]any)
	if !ok {
		t.Fatalf("models type = %T, want []map[string]any", resp["models"])
	}

	entries := make(map[string]map[string]any, len(models))
	for _, entry := range models {
		if slug, ok := entry["slug"].(string); ok {
			entries[slug] = entry
		}
	}

	if mainEntry := entries["codex-main"]; mainEntry == nil {
		t.Fatal("missing codex-main entry")
	} else {
		if compHash, _ := mainEntry["comp_hash"].(string); compHash != "3000" {
			t.Errorf("codex-main comp_hash = %q, want 3000", compHash)
		}
		if maxContext := testIntModelValue(mainEntry, "max_context_window"); maxContext != 872000 {
			t.Errorf("codex-main max_context_window = %d, want 872000", maxContext)
		}
		if search, _ := mainEntry["supports_search_tool"].(bool); !search {
			t.Errorf("codex-main supports_search_tool = %v, want true", search)
		}
	}

	if lunaEntry := entries["codex-luna"]; lunaEntry == nil {
		t.Fatal("missing codex-luna entry")
	} else {
		if compHash, _ := lunaEntry["comp_hash"].(string); compHash != "3000" {
			t.Errorf("codex-luna comp_hash = %q, want 3000", compHash)
		}
		if maxContext := testIntModelValue(lunaEntry, "max_context_window"); maxContext != 872000 {
			t.Errorf("codex-luna max_context_window = %d, want 872000", maxContext)
		}
		if search, _ := lunaEntry["supports_search_tool"].(bool); !search {
			t.Errorf("codex-luna supports_search_tool = %v, want true", search)
		}
	}
}
