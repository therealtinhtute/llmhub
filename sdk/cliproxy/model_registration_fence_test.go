package cliproxy

import (
	"context"
	"testing"
	"time"

	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

// waitForClientModel polls until the registry publishes modelID for clientID,
// bounded so a missing publication fails deterministically instead of hanging.
// Adapted from upstream CLIProxyAPI commit 16b53545e006.
func waitForClientModel(t *testing.T, clientID, modelID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !GlobalModelRegistry().ClientSupportsModel(clientID, modelID) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for client %s to register model %s", clientID, modelID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestModelRegistrationFence_RejectsStaleRegistryEpoch(t *testing.T) {
	reg := GlobalModelRegistry()
	clientID := "test-epoch-fence-client"
	reg.UnregisterClient(clientID)
	t.Cleanup(func() {
		reg.UnregisterClient(clientID)
	})

	auth := &coreauth.Auth{
		ID:       clientID,
		Provider: "gemini",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"auth_kind": "apikey",
		},
	}

	cfg := &config.Config{}
	service := &Service{cfg: cfg}

	// 1. Capture fence at initial epoch (0)
	fence := newModelRegistrationFence(auth, cfg)

	// 2. Interleaving: another registration occurs on this client, advancing its epoch
	reg.RegisterClient(clientID, "gemini", []*ModelInfo{{ID: "interleaving-model"}})

	// 3. Stale fence publication must be rejected
	models := []*ModelInfo{{ID: "stale-model"}}
	published := fence.publishModels(service, clientID, "gemini", models)
	if published {
		t.Fatal("expected publishModels to return false when registry epoch was superseded")
	}

	// Verify that the stale model was not registered
	if reg.ClientSupportsModel(clientID, "stale-model") {
		t.Fatal("stale model should not be supported in registry")
	}
}

func TestModelRegistrationFence_RejectsWhenGlobalExclusionsChange(t *testing.T) {
	reg := GlobalModelRegistry()
	clientID := "test-global-excl-fence-client"
	reg.UnregisterClient(clientID)
	t.Cleanup(func() {
		reg.UnregisterClient(clientID)
	})

	cfg := &config.Config{
		OAuthExcludedModels: map[string][]string{
			"claude": {"claude-2.0"},
		},
	}
	service := &Service{cfg: cfg}

	auth := &coreauth.Auth{
		ID:       clientID,
		Provider: "claude",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"auth_kind": "oauth",
		},
	}

	// 1. Create fence observing "claude-2.0" excluded
	fence := newModelRegistrationFence(auth, cfg)

	// 2. Config update changes global exclusions to include another model
	service.cfgMu.Lock()
	service.cfg = &config.Config{
		OAuthExcludedModels: map[string][]string{
			"claude": {"claude-2.0", "claude-2.1"},
		},
	}
	service.cfgMu.Unlock()

	// 3. Publishing with the stale fence must be rejected
	models := []*ModelInfo{{ID: "claude-3-opus"}}
	published := fence.publishModels(service, clientID, "claude", models)
	if published {
		t.Fatal("expected publishModels to reject when global exclusions drifted mid-flight")
	}
}

// TestModelRegistrationSupersededReconcilesLatest_Issue6718 reproduces issue #6718:
// a registration that finishes after its auth record was updated mid-computation
// reconciles the latest auth snapshot so models are not dropped.
func TestModelRegistrationSupersededReconcilesLatest_Issue6718(t *testing.T) {
	clientID := "reconcile-test-client"
	reg := GlobalModelRegistry()
	reg.UnregisterClient(clientID)
	t.Cleanup(func() {
		reg.UnregisterClient(clientID)
	})

	cfg := &config.Config{
		SDKConfig: config.SDKConfig{ForceModelPrefix: false},
	}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	service := &Service{cfg: cfg, coreManager: manager}

	auth := &coreauth.Auth{
		ID:       clientID,
		Provider: "gemini",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"auth_kind": "apikey",
		},
	}

	registered, errRegister := manager.Register(context.Background(), auth)
	if errRegister != nil {
		t.Fatalf("failed to register auth: %v", errRegister)
	}

	// Interleave: update auth in manager before completion finishes
	updated := registered.Clone()
	updated.Attributes["note"] = "snapshot-updated"
	if _, errUpdate := manager.Update(context.Background(), updated); errUpdate != nil {
		t.Fatalf("failed to update auth: %v", errUpdate)
	}

	// Run completeModelRegistrationForAuth with the older registered reference
	ok := service.completeModelRegistrationForAuth(context.Background(), registered)
	if !ok {
		t.Fatal("expected completeModelRegistrationForAuth to succeed")
	}

	// Verify that models for the client are registered and available
	waitForClientModel(t, clientID, "gemini-2.5-flash")
	if !reg.ClientSupportsModel(clientID, "gemini-2.5-flash") {
		t.Fatal("expected gemini-2.5-flash to be available after reconciliation")
	}
}
