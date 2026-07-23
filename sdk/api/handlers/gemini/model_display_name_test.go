package gemini

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
)

func TestGeminiModelsIncludesDisplayNameWithoutChangingName(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "test-gemini-model-display-name"
	modelID := "test-gemini-display-model"
	modelRegistry.RegisterClient(clientID, "gemini", []*registry.ModelInfo{{
		ID: modelID, Name: modelID, DisplayName: "Gemini Friendly",
	}})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/models", nil)

	(&GeminiAPIHandler{}).GeminiModels(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	model := gjson.GetBytes(recorder.Body.Bytes(), `models.#(name=="models/test-gemini-display-model")`)
	if !model.Exists() {
		t.Fatalf("model %q not found: %s", modelID, recorder.Body.String())
	}
	if got := model.Get("name").String(); got != "models/"+modelID {
		t.Fatalf("name = %q, want %q", got, "models/"+modelID)
	}
	if got := model.Get("displayName").String(); got != "Gemini Friendly" {
		t.Fatalf("displayName = %q", got)
	}
}

func TestGeminiCLIModelsIncludesDisplayNameWithoutChangingName(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "test-gemini-cli-model-display-name"
	modelID := "test-gemini-cli-display-model"
	modelRegistry.RegisterClient(clientID, "gemini-cli", []*registry.ModelInfo{{
		ID: modelID, Name: modelID, DisplayName: "Gemini CLI Friendly",
	}})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	models := (&GeminiCLIAPIHandler{}).Models()
	var target map[string]any
	for _, model := range models {
		if name, _ := model["name"].(string); name == modelID {
			target = model
			break
		}
	}
	if target == nil {
		t.Fatalf("model %q not found", modelID)
	}
	if got, _ := target["name"].(string); got != modelID {
		t.Fatalf("name = %q, want %q", got, modelID)
	}
	if got, _ := target["displayName"].(string); got != "Gemini CLI Friendly" {
		t.Fatalf("displayName = %q", got)
	}
}
