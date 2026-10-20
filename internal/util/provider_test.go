package util

import (
	"strings"
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

// TestMaskSensitiveQueryMasksValue covers the related half of issue #6666:
// deleting a client API key by value passes the secret as ?value=, so the
// request logger must mask it like other secret-bearing parameters.
func TestMaskSensitiveQueryMasksValue(t *testing.T) {
	secretKey := "sk-secret-client-key"

	t.Run("value parameter is masked", func(t *testing.T) {
		masked := MaskSensitiveQuery("value=" + secretKey)
		if strings.Contains(masked, secretKey) {
			t.Fatalf("value parameter leaked in full: %s", masked)
		}
		if masked != "value=sk-s...-key" {
			t.Fatalf("masked=%q want the HideAPIKey form", masked)
		}
	})

	t.Run("index parameter is untouched", func(t *testing.T) {
		masked := MaskSensitiveQuery("index=3")
		if masked != "index=3" {
			t.Fatalf("masked=%q want %q", masked, "index=3")
		}
	})

	t.Run("known secret parameters stay masked", func(t *testing.T) {
		masked := MaskSensitiveQuery("key=" + secretKey + "&access_token=" + secretKey)
		if strings.Contains(masked, secretKey) {
			t.Fatalf("secret parameter leaked in full: %s", masked)
		}
	})
}
