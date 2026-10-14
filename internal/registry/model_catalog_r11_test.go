package registry

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// R11 (Wave 1 stream P8): registry/catalog additions adapted from upstream
// checkpoint 0d18d3b365be — Claude Haiku 5.5 (f62d4d28ceb4), Gemini 4 / Argon
// variants (6143cc11c9e1, 866d84d08674), Muse Wonder models (4c159069e201),
// web_search native capability (4f9630df230b), and model-specific max-token
// defaults consumed via the existing GetModelInfo/ensureModelMaxTokens path.

func TestR11ClaudeHaiku55Definition(t *testing.T) {
	got := requireModelDefinition(t, GetClaudeModels(), "claude-haiku-5-5")
	want := &ModelInfo{
		ID:                  "claude-haiku-5-5",
		Object:              "model",
		Created:             1790553600,
		OwnedBy:             "anthropic",
		Type:                "claude",
		DisplayName:         "Claude Haiku 5.5",
		Description:         "For high-volume, latency-sensitive tasks such as classification, extraction, and routing.",
		ContextLength:       1000000,
		MaxCompletionTokens: 128000,
		Thinking: &ThinkingSupport{
			ZeroAllowed:    true,
			DynamicAllowed: true,
			Levels:         []string{"low", "medium", "high", "xhigh", "max"},
		},
		SupportedInputModalities:  []string{"text", "image"},
		SupportedOutputModalities: []string{"text"},
		NativeCapabilities:        &NativeCapabilities{WebSearch: boolPtr(true)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("claude-haiku-5-5 definition = %#v, want %#v", got, want)
	}
	if !got.HasNativeWebSearch() {
		t.Fatal("claude-haiku-5-5 should report native web search support")
	}
}

func TestR11GeminiArgonVariants(t *testing.T) {
	argon := requireModelDefinition(t, GetGeminiVertexModels(), "gemini-4-argon")
	if argon.ContextLength != 1048576 {
		t.Fatalf("gemini-4-argon context_length = %d, want 1048576", argon.ContextLength)
	}
	if argon.MaxCompletionTokens != 1048576 {
		t.Fatalf("gemini-4-argon max_completion_tokens = %d, want 1048576", argon.MaxCompletionTokens)
	}
	if !argon.HasNativeWebSearch() {
		t.Fatal("gemini-4-argon should report native web search support")
	}

	high := requireModelDefinition(t, GetAntigravityModels(), "gemini-4-argon-high")
	if !high.SupportsWebSearch {
		t.Fatal("gemini-4-argon-high should carry supports_web_search")
	}
	if high.MaxCompletionTokens != 1048576 {
		t.Fatalf("gemini-4-argon-high max_completion_tokens = %d, want 1048576", high.MaxCompletionTokens)
	}
	if !high.HasNativeWebSearch() {
		t.Fatal("gemini-4-argon-high should report native web search support")
	}
}

func TestR11MuseWonderModels(t *testing.T) {
	for _, id := range []string{"muse-wonder-1.0", "muse-wonder-1.0-contributor"} {
		got := requireModelDefinition(t, GetMetaModels(), id)
		if got.Type != "meta" {
			t.Fatalf("%s type = %q, want meta", id, got.Type)
		}
		if got.ContextLength != 1048576 || got.MaxCompletionTokens != 65536 {
			t.Fatalf("%s context/max = %d/%d, want 1048576/65536", id, got.ContextLength, got.MaxCompletionTokens)
		}
		if got.Thinking == nil || len(got.Thinking.Levels) != 6 {
			t.Fatalf("%s thinking levels = %#v, want 6 levels", id, got.Thinking)
		}
		if lookup := LookupStaticModelInfo(id); lookup == nil || lookup.ID != id {
			t.Fatalf("LookupStaticModelInfo(%q) missed", id)
		}
	}

	for _, channel := range []string{"meta", "muse", "META"} {
		models := GetStaticModelDefinitionsByChannel(channel)
		for _, id := range []string{"muse-wonder-1.0", "muse-wonder-1.0-contributor"} {
			requireModelDefinition(t, models, id)
		}
	}
}

func TestR11NativeCapabilitiesParsing(t *testing.T) {
	raw := []byte(`{"id":"cap-test","object":"model","supports_web_search":true,"native_capabilities":{"web_search":true}}`)
	var info ModelInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("unmarshal ModelInfo: %v", err)
	}
	if !info.SupportsWebSearch {
		t.Fatal("SupportsWebSearch should parse from JSON")
	}
	if !info.HasNativeWebSearch() {
		t.Fatal("HasNativeWebSearch should be true")
	}

	// Native capabilities stay internal: model listings must not expose them.
	marshaled, err := json.Marshal(&info)
	if err != nil {
		t.Fatalf("marshal ModelInfo: %v", err)
	}
	if strings.Contains(string(marshaled), "native_capabilities") {
		t.Fatalf("marshaled ModelInfo should not expose native_capabilities: %s", marshaled)
	}
}

func TestR11ClonePreservesNativeCapabilities(t *testing.T) {
	original := &ModelInfo{
		ID:                "clone-cap-test",
		SupportsWebSearch: true,
		NativeCapabilities: &NativeCapabilities{
			WebSearch: boolPtr(true),
		},
	}
	cloned := cloneModelInfo(original)
	if !reflect.DeepEqual(cloned, original) {
		t.Fatalf("cloned = %#v, want %#v", cloned, original)
	}
	*original.NativeCapabilities.WebSearch = false
	if !*cloned.NativeCapabilities.WebSearch {
		t.Fatal("clone must deep-copy native capabilities")
	}
}

func TestR11LookupModelWebSearchSupport(t *testing.T) {
	for _, id := range []string{"claude-haiku-5-5", "gemini-4-argon", "gemini-4-argon-high"} {
		support := LookupModelWebSearchSupport(id)
		if support == nil || !*support {
			t.Fatalf("LookupModelWebSearchSupport(%q) = %v, want true", id, support)
		}
	}
	if support := LookupModelWebSearchSupport("claude-3-5-haiku-20241022"); support != nil {
		t.Fatalf("LookupModelWebSearchSupport(claude-3-5-haiku-20241022) = %v, want nil", *support)
	}
	if support := LookupModelWebSearchSupport("no-such-model-r11"); support != nil {
		t.Fatalf("LookupModelWebSearchSupport(unknown) = %v, want nil", *support)
	}
}

func TestR11ModelSpecificMaxTokenDefaults(t *testing.T) {
	// ensureModelMaxTokens (claude executor) resolves per-model defaults through
	// GetModelInfo; the static catalog must therefore carry max_completion_tokens
	// for the newly added models.
	for id, want := range map[string]int{
		"claude-haiku-5-5":            128000,
		"muse-wonder-1.0":             65536,
		"muse-wonder-1.0-contributor": 65536,
	} {
		info := LookupStaticModelInfo(id)
		if info == nil {
			t.Fatalf("LookupStaticModelInfo(%q) = nil", id)
		}
		if info.MaxCompletionTokens != want {
			t.Fatalf("%s max_completion_tokens = %d, want %d", id, info.MaxCompletionTokens, want)
		}
	}
	if info := LookupStaticModelInfo("gemini-4-argon-high"); info == nil || info.MaxCompletionTokens != 1048576 {
		t.Fatalf("gemini-4-argon-high max_completion_tokens = %#v, want 1048576", info)
	}
}

func TestR11LocalAliasesRemainIntact(t *testing.T) {
	// Additive changes only: Kiro fallbacks and legacy Gemini entries stay.
	for _, id := range []string{"auto", "claude-sonnet-4.5", "claude-sonnet-4.5-thinking"} {
		requireModelDefinition(t, GetKiroModels(), id)
	}
	for _, models := range map[string][]*ModelInfo{
		"gemini":   GetGeminiModels(),
		"vertex":   GetGeminiVertexModels(),
		"aistudio": GetAIStudioModels(),
	} {
		requireModelDefinition(t, models, "gemini-3.7-flash")
	}
	requireModelDefinition(t, GetAntigravityModels(), "gemini-3.6-flash-high")
}

func boolPtr(value bool) *bool {
	return &value
}
