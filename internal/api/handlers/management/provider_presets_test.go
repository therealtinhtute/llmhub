package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/config/presets"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestGetProviderPresets_ReturnsCatalog(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	manager := coreauth.NewManager(nil, nil, nil)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/provider-presets", nil)
	h.GetProviderPresets(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload struct {
		Presets []presets.Preset `json:"presets"`
	}
	if errUnmarshal := json.Unmarshal(rec.Body.Bytes(), &payload); errUnmarshal != nil {
		t.Fatalf("decode response: %v", errUnmarshal)
	}
	if len(payload.Presets) != len(presets.All()) {
		t.Fatalf("presets len = %d, want %d", len(payload.Presets), len(presets.All()))
	}
	for _, p := range payload.Presets {
		if p.ID == "" || p.BaseURL == "" {
			t.Fatalf("preset missing id/base_url: %+v", p)
		}
	}
}
