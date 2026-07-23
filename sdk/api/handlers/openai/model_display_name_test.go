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

	var target map[string]any
	for _, model := range models {
		if stringModelValue(model, "slug") == "gpt-5.5" {
			target = model
			break
		}
	}
	if target == nil {
		t.Fatal("gpt-5.5 template model not found")
	}
	if got := stringModelValue(target, "slug"); got != "gpt-5.5" {
		t.Fatalf("slug = %q", got)
	}
	if got := stringModelValue(target, "display_name"); got != "Configured GPT 5.5" {
		t.Fatalf("display_name = %q", got)
	}
}
