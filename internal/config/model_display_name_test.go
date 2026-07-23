package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfiguredModelDisplayNameYAML(t *testing.T) {
	var cfg Config
	raw := []byte(`
claude-api-key:
  - api-key: claude-key
    models:
      - name: claude-upstream
        alias: claude-alias
        display-name: Claude Friendly
codex-api-key:
  - api-key: codex-key
    base-url: https://codex.example.com
    models:
      - name: codex-upstream
        alias: codex-alias
        display-name: Codex Friendly
gemini-api-key:
  - api-key: gemini-key
    models:
      - name: gemini-upstream
        alias: gemini-alias
        display-name: Gemini Friendly
vertex-api-key:
  - api-key: vertex-key
    models:
      - name: vertex-upstream
        alias: vertex-alias
        display-name: Vertex Friendly
openai-compatibility:
  - name: compat
    base-url: https://compat.example.com
    models:
      - name: compat-upstream
        alias: compat-alias
        display-name: Compat Friendly
oauth-model-alias:
  codex:
    - name: gpt-upstream
      alias: gpt-alias
      display-name: OAuth Friendly
`)

	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if got := cfg.ClaudeKey[0].Models[0].DisplayName; got != "Claude Friendly" {
		t.Fatalf("claude display name = %q", got)
	}
	if got := cfg.CodexKey[0].Models[0].DisplayName; got != "Codex Friendly" {
		t.Fatalf("codex display name = %q", got)
	}
	if got := cfg.GeminiKey[0].Models[0].DisplayName; got != "Gemini Friendly" {
		t.Fatalf("gemini display name = %q", got)
	}
	if got := cfg.VertexCompatAPIKey[0].Models[0].DisplayName; got != "Vertex Friendly" {
		t.Fatalf("vertex display name = %q", got)
	}
	if got := cfg.OpenAICompatibility[0].Models[0].DisplayName; got != "Compat Friendly" {
		t.Fatalf("openai compatibility display name = %q", got)
	}
	if got := cfg.OAuthModelAlias["codex"][0].DisplayName; got != "OAuth Friendly" {
		t.Fatalf("oauth alias display name = %q", got)
	}
}

func TestModelDisplayNameSurvivesSanitization(t *testing.T) {
	cfg := Config{
		ClaudeKey: []ClaudeKey{{
			APIKey: "claude-key",
			Models: []ClaudeModel{{Name: "claude-upstream", Alias: "claude-alias", DisplayName: "Claude Friendly"}},
		}},
		CodexKey: []CodexKey{{
			APIKey:  "codex-key",
			BaseURL: "https://codex.example.com",
			Models:  []CodexModel{{Name: "codex-upstream", Alias: "codex-alias", DisplayName: "Codex Friendly"}},
		}},
		GeminiKey: []GeminiKey{{
			APIKey: "gemini-key",
			Models: []GeminiModel{{Name: "gemini-upstream", Alias: "gemini-alias", DisplayName: "Gemini Friendly"}},
		}},
		VertexCompatAPIKey: []VertexCompatKey{{
			APIKey: "vertex-key",
			Models: []VertexCompatModel{{Name: "vertex-upstream", Alias: "vertex-alias", DisplayName: "Vertex Friendly"}},
		}},
		OpenAICompatibility: []OpenAICompatibility{{
			Name:    "compat",
			BaseURL: "https://compat.example.com",
			Models:  []OpenAICompatibilityModel{{Name: "compat-upstream", Alias: "compat-alias", DisplayName: "Compat Friendly"}},
		}},
		OAuthModelAlias: map[string][]OAuthModelAlias{
			" CoDeX ": {{Name: " gpt-upstream ", Alias: " gpt-alias ", Fork: true, DisplayName: " OAuth Friendly "}},
		},
	}

	cfg.SanitizeClaudeKeys()
	cfg.SanitizeCodexKeys()
	cfg.SanitizeGeminiKeys()
	cfg.SanitizeVertexCompatKeys()
	cfg.SanitizeOpenAICompatibility()
	cfg.SanitizeOAuthModelAlias()

	if got := cfg.ClaudeKey[0].Models[0].DisplayName; got != "Claude Friendly" {
		t.Fatalf("claude display name after sanitization = %q", got)
	}
	if got := cfg.CodexKey[0].Models[0].DisplayName; got != "Codex Friendly" {
		t.Fatalf("codex display name after sanitization = %q", got)
	}
	if got := cfg.GeminiKey[0].Models[0].DisplayName; got != "Gemini Friendly" {
		t.Fatalf("gemini display name after sanitization = %q", got)
	}
	if got := cfg.VertexCompatAPIKey[0].Models[0].DisplayName; got != "Vertex Friendly" {
		t.Fatalf("vertex display name after sanitization = %q", got)
	}
	if got := cfg.OpenAICompatibility[0].Models[0].DisplayName; got != "Compat Friendly" {
		t.Fatalf("openai compatibility display name after sanitization = %q", got)
	}
	alias := cfg.OAuthModelAlias["codex"][0]
	if alias.Name != "gpt-upstream" || alias.Alias != "gpt-alias" || !alias.Fork || alias.DisplayName != "OAuth Friendly" {
		t.Fatalf("sanitized oauth alias = %+v", alias)
	}
}
