package modelconfig

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	_ "github.com/therealtinhtute/llmhub/internal/thinking/provider/claude"
	"github.com/tidwall/gjson"
)

func TestResolveModelInfoUsesSuffixFreeStaticCapabilities(t *testing.T) {
	info := ResolveModelInfo("claude-opus-4-6(high)", "claude", nil)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want inherited thinking support", info)
	}
	if info.ID != "claude-opus-4-6(high)" {
		t.Fatalf("model ID = %q, want configured upstream name", info.ID)
	}
	if info.UserDefined {
		t.Fatal("resolved capability snapshot must not be user-defined")
	}
}

func TestResolveModelInfoExplicitThinkingOverridesAndClones(t *testing.T) {
	support := &registry.ThinkingSupport{Levels: []string{" XHIGH ", "xhigh", " High "}}
	info := ResolveModelInfo("custom-model", "codex", support)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want explicit thinking support", info)
	}
	if got := info.Thinking.Levels; len(got) != 2 || got[0] != "xhigh" || got[1] != "high" {
		t.Fatalf("normalized levels = %v, want [xhigh high]", got)
	}
	support.Levels[0] = "low"
	if info.Thinking.Levels[0] != "xhigh" {
		t.Fatal("resolved thinking support shares mutable config storage")
	}
}

func TestNormalizeThinkingSupportDerivesSpecialLevelFlags(t *testing.T) {
	support := NormalizeThinkingSupport(&registry.ThinkingSupport{Levels: []string{"low", "none", "auto"}})
	if support == nil {
		t.Fatal("NormalizeThinkingSupport() = nil")
	}
	if !support.ZeroAllowed {
		t.Fatal("none level did not enable ZeroAllowed")
	}
	if !support.DynamicAllowed {
		t.Fatal("auto level did not enable DynamicAllowed")
	}
}

// A claude-api-key model entry with its own thinking levels must keep the catalog
// way of turning thinking off: claude-sonnet-5-5 rejects "disabled".
func TestResolveModelInfoConfiguredThinkingKeepsCatalogDisableType(t *testing.T) {
	configured := &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "none"}}
	info := ResolveModelInfo("claude-sonnet-5-5", "claude", configured)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want thinking support", info)
	}
	if got := info.Thinking.DisableType; got != "between_tools" {
		t.Fatalf("DisableType = %q, want catalog between_tools", got)
	}
	if got := info.Thinking.Levels; len(got) != 4 || got[3] != "none" {
		t.Fatalf("levels = %v, want configured [low medium high none]", got)
	}

	body := []byte(`{"model":"claude-sonnet-5-5","messages":[{"role":"user","content":"hi"}],"thinking":{"type":"disabled"}}`)
	out, err := thinking.ApplyThinkingWithModelInfo(body, nil, "claude-sonnet-5-5", "claude", "claude", "claude", info)
	if err != nil {
		t.Fatalf("ApplyThinkingWithModelInfo: %v", err)
	}
	if got := gjson.GetBytes(out, "thinking").Raw; got != `{"type":"between_tools"}` {
		t.Fatalf("thinking = %s, want {\"type\":\"between_tools\"}; body=%s", got, out)
	}

	explicit := ResolveModelInfo("claude-sonnet-5-5", "claude", &registry.ThinkingSupport{Levels: []string{"low"}, DisableType: "disabled"})
	if got := explicit.Thinking.DisableType; got != "disabled" {
		t.Fatalf("explicit DisableType = %q, want configured disabled", got)
	}
	if other := ResolveModelInfo("claude-opus-5", "claude", configured); other.Thinking.DisableType != "" {
		t.Fatalf("claude-opus-5 DisableType = %q, want empty", other.Thinking.DisableType)
	}
	if variant := ResolveModelInfo("claude-sonnet-5-5[1m]", "claude", configured); variant.Thinking.DisableType != "between_tools" {
		t.Fatalf("claude-sonnet-5-5[1m] DisableType = %q, want catalog between_tools", variant.Thinking.DisableType)
	}
}

// A runtime registration for the same ID must not hide catalog capabilities it
// does not declare itself (P13: unlisted execution model lost Thinking).
func TestResolveModelInfoKeepsCatalogThinkingWhenRegistryEntryShadowsIt(t *testing.T) {
	const modelID = "gpt-6-luna"
	static := registry.LookupStaticModelInfo(modelID)
	if static == nil || static.Thinking == nil {
		t.Fatalf("catalog entry %q missing thinking support", modelID)
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient("shadow-"+modelID, "codex", []*registry.ModelInfo{{ID: modelID, SupportConfigurationUpdate: true}})
	t.Cleanup(func() { reg.UnregisterClient("shadow-" + modelID) })

	info := ResolveModelInfo(modelID, "codex", nil)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want catalog thinking support", info)
	}
	if len(info.Thinking.Levels) != len(static.Thinking.Levels) {
		t.Fatalf("thinking levels = %v, want catalog %v", info.Thinking.Levels, static.Thinking.Levels)
	}
	for i := range static.Thinking.Levels {
		if info.Thinking.Levels[i] != static.Thinking.Levels[i] {
			t.Fatalf("thinking levels = %v, want catalog %v", info.Thinking.Levels, static.Thinking.Levels)
		}
	}
}

// A shadowing runtime entry must not disable the family lookup that gives
// Claude bracket/dated variants their catalog capabilities.
func TestResolveModelInfoKeepsCatalogThinkingForShadowedClaudeVariant(t *testing.T) {
	const modelID = "claude-sonnet-5-5[1m]"
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient("variant-"+modelID, "claude", []*registry.ModelInfo{{ID: modelID, SupportConfigurationUpdate: true}})
	t.Cleanup(func() { reg.UnregisterClient("variant-" + modelID) })

	info := ResolveModelInfo(modelID, "claude", nil)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want family catalog thinking support", info)
	}
	if got := info.Thinking.DisableType; got != "between_tools" {
		t.Fatalf("DisableType = %q, want catalog between_tools", got)
	}
}

// A registry entry that declares its own thinking support still overrides the
// catalog entry instead of being replaced by it.
func TestResolveModelInfoPrefersRegistryThinkingOverCatalog(t *testing.T) {
	const modelID = "gpt-6-luna"
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient("thinking-"+modelID, "codex", []*registry.ModelInfo{{
		ID:       modelID,
		Thinking: &registry.ThinkingSupport{Levels: []string{"minimal"}},
	}})
	t.Cleanup(func() { reg.UnregisterClient("thinking-" + modelID) })

	info := ResolveModelInfo(modelID, "codex", nil)
	if info == nil || info.Thinking == nil {
		t.Fatalf("ResolveModelInfo() = %+v, want registry thinking support", info)
	}
	if got := info.Thinking.Levels; len(got) != 1 || got[0] != "minimal" {
		t.Fatalf("thinking levels = %v, want [minimal]", got)
	}
}

func TestResolveModelInfoUnknownModelKeepsMissingCapability(t *testing.T) {
	info := ResolveModelInfo("unknown-configured-model", "claude", nil)
	if info == nil {
		t.Fatal("ResolveModelInfo() = nil")
	}
	if info.Thinking != nil {
		t.Fatalf("unknown model thinking = %+v, want nil", info.Thinking)
	}
	if info.UserDefined {
		t.Fatal("unknown configured model must use its exact bound capability")
	}
}
