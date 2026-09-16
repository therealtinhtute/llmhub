package util

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
)

// TestGetProviderNameFallsBackToLowercaseModel verifies the mixed-case retry
// added by upstream CLIProxyAPI commit eed249072d57: when the exact model name
// has no registered providers, GetProviderName retries with the lowercased
// name so namespaced IDs such as devin/SWE-2 still resolve.
// Local symbols: GetProviderName, registry.ModelRegistry.RegisterClient.
func TestGetProviderNameFallsBackToLowercaseModel(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	const clientID = "util-test-devin-client"
	reg.RegisterClient(clientID, "devin", []*registry.ModelInfo{{ID: "devin/mixed-case-probe"}})
	t.Cleanup(func() {
		reg.UnregisterClient(clientID)
	})

	providers := GetProviderName("Devin/Mixed-Case-Probe")
	if len(providers) != 1 || providers[0] != "devin" {
		t.Fatalf("GetProviderName(mixed-case) = %v, want [devin]", providers)
	}

	// Exact-case lookup still resolves on the first pass.
	providers = GetProviderName("devin/mixed-case-probe")
	if len(providers) != 1 || providers[0] != "devin" {
		t.Fatalf("GetProviderName(exact) = %v, want [devin]", providers)
	}
}
