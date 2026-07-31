package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
)

func TestOpenAIModelsIncludesDisplayNameWithoutChangingID(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "test-openai-model-display-name"
	modelID := "test-openai-display-model"
	modelRegistry.RegisterClient(clientID, "openai", []*registry.ModelInfo{{
		ID: modelID, Object: "model", OwnedBy: "test", DisplayName: "OpenAI Friendly",
	}})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	(&OpenAIAPIHandler{}).OpenAIModels(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	model := gjson.GetBytes(recorder.Body.Bytes(), `data.#(id=="test-openai-display-model")`)
	if !model.Exists() {
		t.Fatalf("model %q not found: %s", modelID, recorder.Body.String())
	}
	if got := model.Get("id").String(); got != modelID {
		t.Fatalf("id = %q, want %q", got, modelID)
	}
	if got := model.Get("display_name").String(); got != "OpenAI Friendly" {
		t.Fatalf("display_name = %q", got)
	}
}

func TestCodexClientModelsUsesConfiguredDisplayNameForTemplate(t *testing.T) {
	models := buildCodexClientModels([]map[string]any{{
		"id":           "gpt-5.5",
		"display_name": "Configured GPT 5.5",
	}})

	target := requireCodexClientModel(t, models, "gpt-5.5")
	if got := stringModelValue(target, "display_name"); got != "Configured GPT 5.5" {
		t.Fatalf("display_name = %q", got)
	}
}

func TestCodexClientModelsIncludesCheckpointTemplates(t *testing.T) {
	models := buildCodexClientModels([]map[string]any{
		{"id": "gpt-5.6-sol"},
		{"id": "gpt-5.6-terra"},
		{"id": "gpt-5.6-luna"},
		{"id": "gpt-5.3-codex-spark"},
	})

	tests := []struct {
		slug        string
		displayName string
	}{
		{slug: "gpt-5.6-sol", displayName: "GPT-5.6-Sol"},
		{slug: "gpt-5.6-terra", displayName: "GPT-5.6-Terra"},
		{slug: "gpt-5.6-luna", displayName: "GPT-5.6-Luna"},
		{slug: "gpt-5.3-codex-spark", displayName: "GPT-5.3-Codex-Spark"},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			model := requireCodexClientModel(t, models, tt.slug)
			if got := stringModelValue(model, "display_name"); got != tt.displayName {
				t.Fatalf("display_name = %q, want %q", got, tt.displayName)
			}
			if got := stringModelValue(model, "visibility"); got != "list" {
				t.Fatalf("visibility = %q, want list", got)
			}
		})
	}
}

func requireCodexClientModel(t *testing.T, models []map[string]any, slug string) map[string]any {
	t.Helper()
	for _, model := range models {
		if stringModelValue(model, "slug") == slug {
			return model
		}
	}
	t.Fatalf("codex client model %q not found", slug)
	return nil
}
