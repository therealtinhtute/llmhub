package claude

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
)

func TestClaudeModelsIncludesDisplayNameWithoutChangingID(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "test-claude-model-display-name"
	modelID := "test-claude-display-model"
	modelRegistry.RegisterClient(clientID, "claude", []*registry.ModelInfo{{
		ID: modelID, Object: "model", OwnedBy: "anthropic", Type: "claude", DisplayName: "Claude Friendly",
	}})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	(&ClaudeCodeAPIHandler{}).ClaudeModels(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	model := gjson.GetBytes(recorder.Body.Bytes(), `data.#(id=="test-claude-display-model")`)
	if !model.Exists() {
		t.Fatalf("model %q not found: %s", modelID, recorder.Body.String())
	}
	if got := model.Get("id").String(); got != modelID {
		t.Fatalf("id = %q, want %q", got, modelID)
	}
	if got := model.Get("display_name").String(); got != "Claude Friendly" {
		t.Fatalf("display_name = %q", got)
	}
}
