package cliproxy

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestConfiguredModelsUseDisplayNameWithoutChangingID(t *testing.T) {
	tests := []struct {
		name        string
		modelID     string
		displayName string
		models      []*ModelInfo
	}{
		{
			name:        "claude",
			modelID:     "claude-alias",
			displayName: "Claude Friendly",
			models: buildClaudeConfigModels(&config.ClaudeKey{Models: []config.ClaudeModel{{
				Name: "claude-upstream", Alias: "claude-alias", DisplayName: " Claude Friendly ",
			}}}),
		},
		{
			name:        "gemini",
			modelID:     "gemini-alias",
			displayName: "Gemini Friendly",
			models: buildGeminiConfigModels(&config.GeminiKey{Models: []config.GeminiModel{{
				Name: "gemini-upstream", Alias: "gemini-alias", DisplayName: " Gemini Friendly ",
			}}}),
		},
		{
			name:        "vertex",
			modelID:     "vertex-alias",
			displayName: "Vertex Friendly",
			models: buildVertexCompatConfigModels(&config.VertexCompatKey{Models: []config.VertexCompatModel{{
				Name: "vertex-upstream", Alias: "vertex-alias", DisplayName: " Vertex Friendly ",
			}}}),
		},
		{
			name:        "openai compatibility",
			modelID:     "compat-alias",
			displayName: "Compat Friendly",
			models: buildOpenAICompatibilityConfigModels(&config.OpenAICompatibility{
				Name: "compat",
				Models: []config.OpenAICompatibilityModel{{
					Name: "compat-upstream", Alias: "compat-alias", DisplayName: " Compat Friendly ",
				}},
			}),
		},
		{
			name:        "codex builtin replacement",
			modelID:     "gpt-image-2",
			displayName: "Configured Image Two",
			models: buildCodexConfigModels(&config.CodexKey{Models: []config.CodexModel{{
				Name: "gpt-image-2", Alias: "gpt-image-2", DisplayName: " Configured Image Two ",
			}}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := findModelInfoByID(tt.models, tt.modelID)
			if model == nil {
				t.Fatalf("model %q not found", tt.modelID)
			}
			if model.ID != tt.modelID {
				t.Fatalf("model id = %q, want %q", model.ID, tt.modelID)
			}
			if model.DisplayName != tt.displayName {
				t.Fatalf("display name = %q, want %q", model.DisplayName, tt.displayName)
			}
		})
	}
}

func TestConfiguredModelDisplayNameSurvivesPrefixClone(t *testing.T) {
	models := buildGeminiConfigModels(&config.GeminiKey{Models: []config.GeminiModel{{
		Name: "gemini-upstream", Alias: "gemini-alias", DisplayName: "Gemini Friendly",
	}}})

	prefixed := applyModelPrefixes(models, "team", false)
	original := findModelInfoByID(prefixed, "gemini-alias")
	alias := findModelInfoByID(prefixed, "team/gemini-alias")
	if original == nil || alias == nil {
		t.Fatalf("expected original and prefixed models, got %+v", prefixed)
	}
	if original.DisplayName != "Gemini Friendly" || alias.DisplayName != "Gemini Friendly" {
		t.Fatalf("display names = %q and %q", original.DisplayName, alias.DisplayName)
	}
}

func TestOAuthModelAliasDisplayNameOnlyChangesPresentation(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {{
				Name: "gpt-upstream", Alias: "gpt-alias", Fork: true, DisplayName: "Alias Friendly",
			}},
		},
	}
	models := []*ModelInfo{{
		ID: "gpt-upstream", Name: "models/gpt-upstream", DisplayName: "Upstream Friendly", OwnedBy: "openai",
	}}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 2 {
		t.Fatalf("expected original and alias, got %d models", len(out))
	}
	original := findModelInfoByID(out, "gpt-upstream")
	alias := findModelInfoByID(out, "gpt-alias")
	if original == nil || alias == nil {
		t.Fatalf("expected original and alias models, got %+v", out)
	}
	if original.Name != "models/gpt-upstream" || original.DisplayName != "Upstream Friendly" {
		t.Fatalf("original model changed: %+v", original)
	}
	if alias.Name != "models/gpt-alias" || alias.DisplayName != "Alias Friendly" || alias.OwnedBy != "openai" {
		t.Fatalf("alias model metadata = %+v", alias)
	}
}

func findModelInfoByID(models []*ModelInfo, id string) *ModelInfo {
	for _, model := range models {
		if model != nil && model.ID == id {
			return model
		}
	}
	return nil
}
