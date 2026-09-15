package registry

import (
	"encoding/json"
	"testing"
)

// TestV733ParityModelDefinitions covers the model additions ported from upstream
// CLIProxyAPI commits dacae5822842 (claude-fable-5-1, gemini-3.8-flash,
// gemini-3.8-flash-high), c77b13694318 (gpt-6-astra), and d48590a47d78
// (gemini-3.5-flash-lite on antigravity). Local symbols: GetClaudeModels,
// GetGeminiModels, GetGeminiVertexModels, GetAIStudioModels, GetAntigravityModels,
// GetCodexTeamModels, GetCodexPlusModels, GetCodexProModels.
func TestV733ParityModelDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		models  []*ModelInfo
		id      string
	}{
		// upstream dacae5822842: claude fable 5.1
		{name: "claude fable 5.1", channel: "claude", models: GetClaudeModels(), id: "claude-fable-5-1"},
		// upstream dacae5822842: gemini 3.8 flash across google surfaces
		{name: "gemini 3.8 flash", channel: "gemini", models: GetGeminiModels(), id: "gemini-3.8-flash"},
		{name: "vertex 3.8 flash", channel: "vertex", models: GetGeminiVertexModels(), id: "gemini-3.8-flash"},
		{name: "aistudio 3.8 flash", channel: "aistudio", models: GetAIStudioModels(), id: "gemini-3.8-flash"},
		{name: "antigravity 3.8 flash high", channel: "antigravity", models: GetAntigravityModels(), id: "gemini-3.8-flash-high"},
		// upstream c77b13694318: gpt-6-astra on codex paid tiers
		{name: "codex-team astra", channel: "codex-team", models: GetCodexTeamModels(), id: "gpt-6-astra"},
		{name: "codex-plus astra", channel: "codex-plus", models: GetCodexPlusModels(), id: "gpt-6-astra"},
		{name: "codex-pro astra", channel: "codex-pro", models: GetCodexProModels(), id: "gpt-6-astra"},
		// upstream d48590a47d78: gemini-3.5-flash-lite on antigravity
		{name: "antigravity 3.5 flash lite", channel: "antigravity", models: GetAntigravityModels(), id: "gemini-3.5-flash-lite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireModelDefinition(t, tt.models, tt.id)
		})
	}
}

// TestClaudeFable51DefinitionShape pins the field-level shape ported from
// upstream dacae5822842 for local symbol GetClaudeModels.
func TestClaudeFable51DefinitionShape(t *testing.T) {
	got := requireModelDefinition(t, GetClaudeModels(), "claude-fable-5-1")
	if got.Created != 1788220800 {
		t.Fatalf("claude-fable-5-1 created = %d, want 1788220800", got.Created)
	}
	if got.ContextLength != 1000000 || got.MaxCompletionTokens != 128000 {
		t.Fatalf("claude-fable-5-1 limits = %d/%d, want 1000000/128000", got.ContextLength, got.MaxCompletionTokens)
	}
	if got.Thinking == nil || !got.Thinking.ZeroAllowed || len(got.Thinking.Levels) != 5 {
		t.Fatalf("claude-fable-5-1 thinking = %#v, want zero-allowed 5-level config", got.Thinking)
	}
	if len(got.SupportedInputModalities) != 2 {
		t.Fatalf("claude-fable-5-1 input modalities = %#v, want [text image]", got.SupportedInputModalities)
	}
}

// TestGPT6AstraDefinitionShape pins the field-level shape ported from upstream
// c77b13694318 for local symbols GetCodexTeamModels/GetCodexPlusModels/GetCodexProModels.
func TestGPT6AstraDefinitionShape(t *testing.T) {
	for name, models := range map[string][]*ModelInfo{
		"codex-team": GetCodexTeamModels(),
		"codex-plus": GetCodexPlusModels(),
		"codex-pro":  GetCodexProModels(),
	} {
		got := requireModelDefinition(t, models, "gpt-6-astra")
		if got.ContextLength != 272000 || got.MaxCompletionTokens != 128000 {
			t.Fatalf("%s gpt-6-astra limits = %d/%d, want 272000/128000", name, got.ContextLength, got.MaxCompletionTokens)
		}
		if got.Thinking == nil || len(got.Thinking.Levels) != 5 || got.Thinking.Levels[0] != "low" {
			t.Fatalf("%s gpt-6-astra thinking = %#v, want 5 levels starting at low", name, got.Thinking)
		}
	}
}

// TestCodexClientCatalogIncludesGPT6Astra verifies the codex client template
// entry ported from upstream c77b13694318. Local symbol: GetCodexClientModelsJSON.
func TestCodexClientCatalogIncludesGPT6Astra(t *testing.T) {
	var payload struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(GetCodexClientModelsJSON(), &payload); err != nil {
		t.Fatalf("codex client catalog decode: %v", err)
	}
	var astra map[string]any
	for _, model := range payload.Models {
		if model["slug"] == "gpt-6-astra" {
			astra = model
			break
		}
	}
	if astra == nil {
		t.Fatal("codex client catalog missing gpt-6-astra entry")
	}
	if got, _ := astra["display_name"].(string); got != "GPT-6-Astra" {
		t.Fatalf("gpt-6-astra display_name = %q, want GPT-6-Astra", got)
	}
	if got, _ := astra["minimal_client_version"].(string); got != "0.153.0" {
		t.Fatalf("gpt-6-astra minimal_client_version = %q, want 0.153.0", got)
	}
	if got, _ := astra["max_context_window"].(float64); got != 872000 {
		t.Fatalf("gpt-6-astra max_context_window = %v, want 872000", got)
	}
	if _, ok := astra["base_instructions"].(string); !ok {
		t.Fatal("gpt-6-astra missing base_instructions")
	}
}
