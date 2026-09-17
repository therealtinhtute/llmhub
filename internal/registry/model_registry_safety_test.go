package registry

import (
	"testing"
	"time"
)

func TestGetModelInfoReturnsClone(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Min: 1, Max: 2, Levels: []string{"low", "high"}},
	}})

	first := r.GetModelInfo("m1", "gemini")
	if first == nil {
		t.Fatal("expected model info")
	}
	first.DisplayName = "mutated"
	first.Thinking.Levels[0] = "mutated"

	second := r.GetModelInfo("m1", "gemini")
	if second.DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second.DisplayName)
	}
	if second.Thinking == nil || len(second.Thinking.Levels) == 0 || second.Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second.Thinking)
	}
}

func TestGetModelsForClientReturnsClones(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Levels: []string{"low", "high"}},
	}})

	first := r.GetModelsForClient("client-1")
	if len(first) != 1 || first[0] == nil {
		t.Fatalf("expected one model, got %+v", first)
	}
	first[0].DisplayName = "mutated"
	first[0].Thinking.Levels[0] = "mutated"

	second := r.GetModelsForClient("client-1")
	if len(second) != 1 || second[0] == nil {
		t.Fatalf("expected one model on second fetch, got %+v", second)
	}
	if second[0].DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second[0].DisplayName)
	}
	if second[0].Thinking == nil || len(second[0].Thinking.Levels) == 0 || second[0].Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second[0].Thinking)
	}
}

func TestGetAvailableModelsByProviderReturnsClones(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Levels: []string{"low", "high"}},
	}})

	first := r.GetAvailableModelsByProvider("gemini")
	if len(first) != 1 || first[0] == nil {
		t.Fatalf("expected one model, got %+v", first)
	}
	first[0].DisplayName = "mutated"
	first[0].Thinking.Levels[0] = "mutated"

	second := r.GetAvailableModelsByProvider("gemini")
	if len(second) != 1 || second[0] == nil {
		t.Fatalf("expected one model on second fetch, got %+v", second)
	}
	if second[0].DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second[0].DisplayName)
	}
	if second[0].Thinking == nil || len(second[0].Thinking.Levels) == 0 || second[0].Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second[0].Thinking)
	}
}

func TestCleanupExpiredQuotasInvalidatesAvailableModelsCache(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "openai", []*ModelInfo{{ID: "m1", Created: 1}})
	r.SetModelQuotaExceeded("client-1", "m1")
	if models := r.GetAvailableModels("openai"); len(models) != 1 {
		t.Fatalf("expected cooldown model to remain listed before cleanup, got %d", len(models))
	}

	r.mutex.Lock()
	quotaTime := time.Now().Add(-6 * time.Minute)
	r.models["m1"].QuotaExceededClients["client-1"] = &quotaTime
	r.mutex.Unlock()

	r.CleanupExpiredQuotas()

	if count := r.GetModelCount("m1"); count != 1 {
		t.Fatalf("expected model count 1 after cleanup, got %d", count)
	}
	models := r.GetAvailableModels("openai")
	if len(models) != 1 {
		t.Fatalf("expected model to stay available after cleanup, got %d", len(models))
	}
	if got := models[0]["id"]; got != "m1" {
		t.Fatalf("expected model id m1, got %v", got)
	}
}

func TestGetAvailableModelsReturnsClonedSupportedParameters(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "openai", []*ModelInfo{{
		ID:                  "m1",
		DisplayName:         "Model One",
		SupportedParameters: []string{"temperature", "top_p"},
	}})

	first := r.GetAvailableModels("openai")
	if len(first) != 1 {
		t.Fatalf("expected one model, got %d", len(first))
	}
	params, ok := first[0]["supported_parameters"].([]string)
	if !ok || len(params) != 2 {
		t.Fatalf("expected supported_parameters slice, got %#v", first[0]["supported_parameters"])
	}
	params[0] = "mutated"

	second := r.GetAvailableModels("openai")
	params, ok = second[0]["supported_parameters"].([]string)
	if !ok || len(params) != 2 || params[0] != "temperature" {
		t.Fatalf("expected cloned supported_parameters, got %#v", second[0]["supported_parameters"])
	}
}

func TestLookupModelInfoReturnsCloneForStaticDefinitions(t *testing.T) {
	first := LookupModelInfo("claude-sonnet-4-6")
	if first == nil || first.Thinking == nil || len(first.Thinking.Levels) == 0 {
		t.Fatalf("expected static model with thinking levels, got %+v", first)
	}
	first.Thinking.Levels[0] = "mutated"

	second := LookupModelInfo("claude-sonnet-4-6")
	if second == nil || second.Thinking == nil || len(second.Thinking.Levels) == 0 || second.Thinking.Levels[0] == "mutated" {
		t.Fatalf("expected static lookup clone, got %+v", second)
	}
}

// The tests below are ported from upstream CLIProxyAPI commit 60e5b8bd432e
// (internal/registry/model_registry_safety_test.go). Upstream's dynamic
// SupportsWebSearch flag maps onto the fork's NativeCapabilities.WebSearch
// tri-state: explicit false stands in for "no support" and explicit true for a
// probed/declared capability.

func nativeCaps(webSearch bool) *NativeCapabilities {
	return &NativeCapabilities{WebSearch: &webSearch}
}

func webSearchSupported(info *ModelInfo) bool {
	return info != nil && info.NativeCapabilities != nil &&
		info.NativeCapabilities.WebSearch != nil && *info.NativeCapabilities.WebSearch
}

func TestApplyClientModelCapabilities_UpdatesAllViews(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("ag-client-1", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	epoch := r.ClientRegistrationEpoch("ag-client-1")
	applied := r.ApplyClientModelCapabilities("ag-client-1", epoch, func(modelID string, info *ModelInfo) {
		if modelID == "gemini-3.1-flash-lite" {
			info.NativeCapabilities = nativeCaps(true)
		}
	})
	if !applied {
		t.Fatal("expected capabilities to be applied")
	}

	// 1. Check GetModelsForClient
	clientModels := r.GetModelsForClient("ag-client-1")
	if len(clientModels) != 1 || !webSearchSupported(clientModels[0]) {
		t.Fatalf("GetModelsForClient: expected WebSearch=true, got %+v", clientModels)
	}

	// 2. Check GetModelInfo (both with provider and global)
	infoByProv := r.GetModelInfo("gemini-3.1-flash-lite", "antigravity")
	if !webSearchSupported(infoByProv) {
		t.Fatalf("GetModelInfo with provider: expected WebSearch=true, got %+v", infoByProv)
	}
	infoGlobal := r.GetModelInfo("gemini-3.1-flash-lite", "")
	if !webSearchSupported(infoGlobal) {
		t.Fatalf("GetModelInfo global: expected WebSearch=true, got %+v", infoGlobal)
	}

	// 3. Check the provider-scoped available-models view (fork's equivalent of
	// upstream's GetAvailableModelInfos check).
	availInfos := r.GetAvailableModelsByProvider("antigravity")
	foundAvail := false
	for _, m := range availInfos {
		if m.ID == "gemini-3.1-flash-lite" && webSearchSupported(m) {
			foundAvail = true
			break
		}
	}
	if !foundAvail {
		t.Fatal("GetAvailableModelsByProvider: expected gemini-3.1-flash-lite with WebSearch=true")
	}
}

func TestApplyClientModelCapabilities_EpochMismatchRejected(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("ag-client-1", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	if r.ApplyClientModelCapabilities("ag-client-1", 999, func(modelID string, info *ModelInfo) {
		info.NativeCapabilities = nativeCaps(true)
	}) {
		t.Fatal("expected stale epoch to be rejected")
	}
	if r.ApplyClientModelCapabilities("unknown-client", 0, func(modelID string, info *ModelInfo) {}) {
		t.Fatal("expected unknown client to be rejected")
	}
	if webSearchSupported(r.GetModelInfo("gemini-3.1-flash-lite", "antigravity")) {
		t.Fatal("rejected mutation must not change capability views")
	}
}

func TestMultiClientRegistration_PreservesProbedCapabilities(t *testing.T) {
	r := newTestModelRegistry()

	// 1. Client A registers model
	r.RegisterClient("client-A", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	// A probes successfully -> WebSearch = true
	epochA := r.ClientRegistrationEpoch("client-A")
	r.ApplyClientModelCapabilities("client-A", epochA, func(modelID string, info *ModelInfo) {
		if modelID == "gemini-3.1-flash-lite" {
			info.NativeCapabilities = nativeCaps(true)
		}
	})

	// 2. Client B registers the same model with static info (WebSearch = false)
	r.RegisterClient("client-B", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	// Check that global views preserve WebSearch=true because Client A is still registered and has the capability
	info := r.GetModelInfo("gemini-3.1-flash-lite", "antigravity")
	if !webSearchSupported(info) {
		t.Fatalf("GetModelInfo: expected WebSearch=true to be preserved after client B registered, got %+v", info)
	}

	// The global aggregate view (fork's equivalent of upstream's
	// GetAvailableModelInfos check) must also keep the merged capability.
	if global := r.GetModelInfo("gemini-3.1-flash-lite", ""); !webSearchSupported(global) {
		t.Fatalf("GetModelInfo global: expected WebSearch=true to be preserved after client B registered, got %+v", global)
	}

	// 3. Client A unregisters -> Now neither client supports web search, so global view should drop it
	r.UnregisterClient("client-A")
	infoAfterA := r.GetModelInfo("gemini-3.1-flash-lite", "antigravity")
	if infoAfterA == nil || webSearchSupported(infoAfterA) {
		t.Fatalf("GetModelInfo: expected WebSearch to drop after client A unregistered, got %+v", infoAfterA)
	}
}

func TestReRegisterClient_ClearsStaleProbedCapabilitiesWhenNoOtherClientSupports(t *testing.T) {
	r := newTestModelRegistry()

	// 1. Client A registers model and probes successfully (WebSearch=true)
	r.RegisterClient("client-A", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})
	epochA := r.ClientRegistrationEpoch("client-A")
	r.ApplyClientModelCapabilities("client-A", epochA, func(modelID string, info *ModelInfo) {
		if modelID == "gemini-3.1-flash-lite" {
			info.NativeCapabilities = nativeCaps(true)
		}
	})

	// 2. Client B registers the same model without web search support
	r.RegisterClient("client-B", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	// 3. Client A now re-registers with new model definitions where WebSearch=false
	// (e.g. config reload or model catalog update)
	r.RegisterClient("client-A", "antigravity", []*ModelInfo{{
		ID:                 "gemini-3.1-flash-lite",
		NativeCapabilities: nativeCaps(false),
	}})

	// Global view must NOT keep client A's stale pre-re-registration capability snapshot!
	// Since neither A nor B now has WebSearch=true, the global and provider view must be false.
	info := r.GetModelInfo("gemini-3.1-flash-lite", "antigravity")
	if info == nil || webSearchSupported(info) {
		t.Fatalf("GetModelInfo: expected WebSearch=false after client A re-registered with false, got %+v", info)
	}

	if global := r.GetModelInfo("gemini-3.1-flash-lite", ""); global == nil || webSearchSupported(global) {
		t.Fatalf("GetModelInfo global: expected WebSearch=false on gemini-3.1-flash-lite, got %+v", global)
	}

	clientAModels := r.GetModelsForClient("client-A")
	if len(clientAModels) != 1 || webSearchSupported(clientAModels[0]) {
		t.Fatalf("GetModelsForClient(A): expected WebSearch=false, got %+v", clientAModels)
	}
}
