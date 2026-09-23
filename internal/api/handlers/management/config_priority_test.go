package management

// Priority patch coverage for every credential PATCH endpoint, ported from
// upstream CLIProxyAPI commit cc77410866c2 ("feat(management): support priority
// field in credential patch handlers"). Upstream also covers XAI and
// interactions credentials; llmhub has no local equivalents for those handler
// types, so the matrix covers the six local credential families instead.
//
// Adaptation: persistence goes through the local ManagementConfigStore
// (recordingConfigStore) rather than a YAML file on disk, so the saved-YAML
// assertion inspects the captured store bytes instead of re-reading the file.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestPatchPriorityForEveryProvider(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*config.Config)
		patch func(*Handler, *gin.Context)
		get   func(*config.Config) int
	}{
		{
			name:  "claude",
			setup: func(cfg *config.Config) { cfg.ClaudeKey = []config.ClaudeKey{{APIKey: "key"}} },
			patch: (*Handler).PatchClaudeKey,
			get:   func(cfg *config.Config) int { return cfg.ClaudeKey[0].Priority },
		},
		{
			name: "meta",
			setup: func(cfg *config.Config) {
				cfg.MetaKey = []config.MetaKey{{APIKey: "key", BaseURL: "https://example.com"}}
			},
			patch: (*Handler).PatchMetaKey,
			get:   func(cfg *config.Config) int { return cfg.MetaKey[0].Priority },
		},
		{
			name: "codex",
			setup: func(cfg *config.Config) {
				cfg.CodexKey = []config.CodexKey{{APIKey: "key", BaseURL: "https://example.com"}}
			},
			patch: (*Handler).PatchCodexKey,
			get:   func(cfg *config.Config) int { return cfg.CodexKey[0].Priority },
		},
		{
			name:  "gemini",
			setup: func(cfg *config.Config) { cfg.GeminiKey = []config.GeminiKey{{APIKey: "key"}} },
			patch: (*Handler).PatchGeminiKey,
			get:   func(cfg *config.Config) int { return cfg.GeminiKey[0].Priority },
		},
		{
			name: "vertex",
			setup: func(cfg *config.Config) {
				cfg.VertexCompatAPIKey = []config.VertexCompatKey{{APIKey: "key", BaseURL: "https://example.com"}}
			},
			patch: (*Handler).PatchVertexCompatKey,
			get:   func(cfg *config.Config) int { return cfg.VertexCompatAPIKey[0].Priority },
		},
		{
			name: "openai compatibility",
			setup: func(cfg *config.Config) {
				cfg.OpenAICompatibility = []config.OpenAICompatibility{{
					Name:          "compat",
					BaseURL:       "https://compat.example.com",
					APIKeyEntries: []config.OpenAICompatibilityAPIKey{{APIKey: "key"}},
				}}
			},
			patch: (*Handler).PatchOpenAICompat,
			get:   func(cfg *config.Config) int { return cfg.OpenAICompatibility[0].Priority },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &recordingConfigStore{}
			cfg := &config.Config{}
			test.setup(cfg)
			h := &Handler{cfg: cfg, configFilePath: writeTestConfigFile(t), configStore: store}

			// Update priority to 7
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/credential",
				strings.NewReader(`{"index":0,"value":{"priority":7}}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			test.patch(h, ctx)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if got := test.get(cfg); got != 7 {
				t.Fatalf("priority = %d, want 7", got)
			}

			// Verify priority persisted through the config store
			savedText := string(store.data)
			if !strings.Contains(savedText, "priority: 7") {
				t.Fatalf("saved YAML missing 'priority: 7':\n%s", savedText)
			}

			// Omitting priority preserves existing priority
			rec = httptest.NewRecorder()
			ctx, _ = gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/credential",
				strings.NewReader(`{"index":0,"value":{"prefix":"team-a"}}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			test.patch(h, ctx)

			if rec.Code != http.StatusOK {
				t.Fatalf("omit priority: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if got := test.get(cfg); got != 7 {
				t.Fatalf("preserved priority = %d, want 7", got)
			}

			// Reset priority to 0 explicitly
			rec = httptest.NewRecorder()
			ctx, _ = gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/credential",
				strings.NewReader(`{"index":0,"value":{"priority":0}}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			test.patch(h, ctx)

			if rec.Code != http.StatusOK {
				t.Fatalf("reset priority: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if got := test.get(cfg); got != 0 {
				t.Fatalf("reset priority = %d, want 0", got)
			}
		})
	}
}
