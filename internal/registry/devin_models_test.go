package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestValidateDevinModelsJSON covers the catalog validation ported from
// upstream CLIProxyAPI commits 982cd124f9fe (envelope/array forms),
// 61741744889d (auto-namespacing of bare IDs), and db0b957c4831
// (case-insensitive duplicate rejection). Local symbols: ValidateDevinModelsJSON.
func TestValidateDevinModelsJSON(t *testing.T) {
	t.Run("valid envelope devin", func(t *testing.T) {
		data := []byte(`{
			"devin": [
				{
					"id": "devin/swe-2",
					"display_name": "SWE-2",
					"owned_by": "cognition",
					"context_length": 262000
				}
			]
		}`)
		models, err := ValidateDevinModelsJSON(data)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if len(models) != 1 || models[0].ID != "devin/swe-2" {
			t.Fatalf("unexpected models: %+v", models)
		}
		if models[0].Type != "devin" {
			t.Errorf("expected type 'devin', got %q", models[0].Type)
		}
	})

	t.Run("valid envelope models", func(t *testing.T) {
		data := []byte(`{
			"models": [
				{
					"id": "devin/glm-5-2",
					"display_name": "GLM-5.2"
				}
			]
		}`)
		models, err := ValidateDevinModelsJSON(data)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if len(models) != 1 || models[0].ID != "devin/glm-5-2" {
			t.Fatalf("unexpected models: %+v", models)
		}
	})

	t.Run("valid direct array", func(t *testing.T) {
		data := []byte(`[
			{
				"id": "devin/deepseek-v4-flash",
				"display_name": "DeepSeek V4 Flash"
			}
		]`)
		models, err := ValidateDevinModelsJSON(data)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if len(models) != 1 || models[0].ID != "devin/deepseek-v4-flash" {
			t.Fatalf("unexpected models: %+v", models)
		}
	})

	// Upstream 61741744889d: bare IDs are namespaced under devin/.
	t.Run("clean id without devin prefix automatically namespaced", func(t *testing.T) {
		data := []byte(`{
			"devin": [
				{
					"id": "swe-2",
					"display_name": "SWE-2"
				}
			]
		}`)
		models, err := ValidateDevinModelsJSON(data)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if len(models) != 1 || models[0].ID != "devin/swe-2" {
			t.Fatalf("expected auto-namespaced to devin/swe-2, got: %q", models[0].ID)
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		_, err := ValidateDevinModelsJSON([]byte(`   `))
		if err == nil {
			t.Fatal("expected error on empty payload, got nil")
		}
	})

	t.Run("empty model id", func(t *testing.T) {
		data := []byte(`{"devin": [{"id": "", "display_name": "No ID"}]}`)
		_, err := ValidateDevinModelsJSON(data)
		if err == nil {
			t.Fatal("expected error on empty model id, got nil")
		}
	})

	// Upstream db0b957c4831: duplicate detection is case-insensitive.
	for name, data := range map[string][]byte{
		"exact duplicate": []byte(`{"devin": [{"id": "devin/swe-2"}, {"id": "devin/swe-2"}]}`),
		"case duplicate":  []byte(`{"devin": [{"id": "devin/SWE-2"}, {"id": "devin/swe-2"}]}`),
	} {
		t.Run("duplicate model id/"+name, func(t *testing.T) {
			_, err := ValidateDevinModelsJSON(data)
			if err == nil {
				t.Fatal("expected error on duplicate model id, got nil")
			}
		})
	}
}

// TestValidateDevinModelsJSONGeminiDefaults covers the auto-populated Gemini
// token limits and generation methods from upstream 0aedd05d31c8.
// Local symbol: sanitizeAndValidateDevinModels via ValidateDevinModelsJSON.
func TestValidateDevinModelsJSONGeminiDefaults(t *testing.T) {
	data := []byte(`{"devin": [{"id": "swe-2", "context_length": 262000, "max_completion_tokens": 128000}]}`)
	models, err := ValidateDevinModelsJSON(data)
	if err != nil {
		t.Fatalf("expected valid, got error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	m := models[0]
	if m.InputTokenLimit != 262000 {
		t.Errorf("InputTokenLimit = %d, want 262000 (from context_length)", m.InputTokenLimit)
	}
	if m.OutputTokenLimit != 128000 {
		t.Errorf("OutputTokenLimit = %d, want 128000 (from max_completion_tokens)", m.OutputTokenLimit)
	}
	if len(m.SupportedGenerationMethods) != 2 ||
		m.SupportedGenerationMethods[0] != "generateContent" ||
		m.SupportedGenerationMethods[1] != "countTokens" {
		t.Errorf("SupportedGenerationMethods = %v, want [generateContent countTokens]", m.SupportedGenerationMethods)
	}
	if m.Object != "model" {
		t.Errorf("Object = %q, want model", m.Object)
	}
	if len(m.SupportedInputModalities) != 1 || m.SupportedInputModalities[0] != "text" {
		t.Errorf("SupportedInputModalities = %v, want [text]", m.SupportedInputModalities)
	}
}

// TestEmbeddedDevinModelsLoadedOnStartup verifies the embedded
// devin_models.json catalog loads at init and that every ID is namespaced
// under devin/ (upstream eed249072d57, 982cd124f9fe).
func TestEmbeddedDevinModelsLoadedOnStartup(t *testing.T) {
	models := GetDevinModels()
	if len(models) < 30 {
		t.Fatalf("expected at least 30 embedded Devin models, got %d", len(models))
	}

	foundMap := make(map[string]bool)
	for _, m := range models {
		foundMap[m.ID] = true
	}

	expectedIDs := []string{
		"devin/swe-2",
		"devin/glm-5-2",
		"devin/glm-5-3",
		"devin/deepseek-v4-flash",
		"devin/deepseek-v4-1-flash",
		"devin/gemini-3-8-flash",
		"devin/grok-4-6",
		"devin/claude-fable-5-1",
		"devin/gpt-6-astra",
	}

	for _, id := range expectedIDs {
		if !foundMap[id] {
			t.Errorf("expected embedded catalog to contain %q", id)
		}
	}
}

// TestGetDevinModelsChannelAndLookup verifies channel dispatch and static
// lookup for namespaced Devin IDs (upstream eed249072d57).
// Local symbols: GetStaticModelDefinitionsByChannel, LookupStaticModelInfo.
func TestGetDevinModelsChannelAndLookup(t *testing.T) {
	byChannel := GetStaticModelDefinitionsByChannel("devin")
	if len(byChannel) == 0 {
		t.Fatal("GetStaticModelDefinitionsByChannel(\"devin\") returned empty list")
	}

	info := LookupStaticModelInfo("devin/swe-2")
	if info == nil {
		t.Fatal("LookupStaticModelInfo(\"devin/swe-2\") = nil, want valid model")
	}
	if info.DisplayName != "SWE-2" {
		t.Errorf("info.DisplayName = %q, want SWE-2", info.DisplayName)
	}
	if info.Type != "devin" {
		t.Errorf("info.Type = %q, want devin", info.Type)
	}
}

// TestLookupDevinModel verifies bare and namespaced lookups against the active
// Devin catalog (upstream 926450e87b5c).
// Local symbol: LookupDevinModel.
func TestLookupDevinModel(t *testing.T) {
	for _, id := range []string{"devin/swe-2", "swe-2", "Devin/SWE-2"} {
		info := LookupDevinModel(id)
		if info == nil {
			t.Fatalf("LookupDevinModel(%q) = nil, want valid model", id)
		}
		if info.ID != "devin/swe-2" {
			t.Errorf("LookupDevinModel(%q).ID = %q, want devin/swe-2", id, info.ID)
		}
	}
	if got := LookupDevinModel("devin/nonexistent-model"); got != nil {
		t.Errorf("LookupDevinModel(nonexistent) = %+v, want nil", got)
	}
	if got := LookupDevinModel(""); got != nil {
		t.Errorf("LookupDevinModel(empty) = %+v, want nil", got)
	}
}

// TestDevinModelsRemoteFetchFallback verifies the remote updater keeps the
// embedded catalog on failure and swaps in validated remote data on success
// (upstream 982cd124f9fe). Local symbols: tryRefreshDevinModels,
// loadDevinModelsFromBytes, devinModelsURLs.
func TestDevinModelsRemoteFetchFallback(t *testing.T) {
	// Test remote failure maintains existing embedded data
	origURLs := devinModelsURLs
	defer func() { devinModelsURLs = origURLs }()

	// Point to failing server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	devinModelsURLs = []string{ts.URL + "/devin_models.json"}

	initialCount := len(GetDevinModels())
	if initialCount == 0 {
		t.Fatal("expected non-empty initial Devin models")
	}

	// Attempt refresh from failing remote
	tryRefreshDevinModels(context.Background(), "test failing refresh")

	afterCount := len(GetDevinModels())
	if afterCount != initialCount {
		t.Fatalf("expected catalog to remain intact with %d models, got %d", initialCount, afterCount)
	}

	// Point to succeeding server with valid update
	validUpdate := []byte(`{
		"devin": [
			{
				"id": "devin/custom-test-model",
				"display_name": "Custom Test Model",
				"owned_by": "custom"
			}
		]
	}`)

	tsValid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(validUpdate)
	}))
	defer tsValid.Close()

	devinModelsURLs = []string{tsValid.URL + "/devin_models.json"}
	tryRefreshDevinModels(context.Background(), "test succeeding refresh")

	updatedModels := GetDevinModels()
	if len(updatedModels) != 1 || updatedModels[0].ID != "devin/custom-test-model" {
		t.Fatalf("expected catalog to be updated to custom-test-model, got: %+v", updatedModels)
	}

	// Restore original embedded data for following tests
	_, _ = loadDevinModelsFromBytes(embeddedDevinModelsJSON, "restore-embed")
}
