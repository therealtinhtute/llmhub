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

	// Run live registerModelsForAuth with the older registered reference
	service.registerModelsForAuth(registered)

	// Verify that models for the client are registered and available via reconciliation
	waitForClientModel(t, clientID, "gemini-2.5-flash")
	if !reg.ClientSupportsModel(clientID, "gemini-2.5-flash") {
		t.Fatal("expected gemini-2.5-flash to be available after reconciliation")
	}
}

func TestRegisterResolvedModelsForAuth_LiveFencingReconcilesSuperseded(t *testing.T) {
	clientID := "live-fencing-test-client"
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

	// Initial registration through live registerModelsForAuth
	service.registerModelsForAuth(registered)
	waitForClientModel(t, clientID, "gemini-2.5-flash")

	// Interleaving: manager auth is updated to disabled
	disabledAuth := registered.Clone()
	disabledAuth.Disabled = true
	disabledAuth.Status = coreauth.StatusDisabled
	if _, errUpdate := manager.Update(context.Background(), disabledAuth); errUpdate != nil {
		t.Fatalf("failed to update auth: %v", errUpdate)
	}

	// A stale concurrent resolution pass tries to publish models using the old registered auth and old fence
	staleFence := newModelRegistrationFence(registered, service.cfg)
	staleModels := []*ModelInfo{{ID: "stale-model"}}
	service.registerResolvedModelsForAuth(registered, staleFence, "gemini", staleModels)

	// The stale publication must be rejected and latest auth reconciled (client unregistered)
	if reg.ClientSupportsModel(clientID, "stale-model") {
		t.Fatal("stale model should not be registered when auth identity was superseded in manager")
	}
	if reg.ClientSupportsModel(clientID, "gemini-2.5-flash") {
		t.Fatal("expected gemini-2.5-flash to be unregistered after reconcile of disabled latest auth")
	}
}

func TestRegisterResolvedModelsForAuth_LiveFencingReconcilesEnabledLatest(t *testing.T) {
	clientID := "live-fencing-enabled-test-client"
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

	// Capture old fence
	oldFence := newModelRegistrationFence(registered, cfg)

	// Interleaving: manager updates auth with an exclusion attribute
	updatedAuth := registered.Clone()
	updatedAuth.Attributes["excluded_models"] = "gemini-2.5-flash"
	if _, errUpdate := manager.Update(context.Background(), updatedAuth); errUpdate != nil {
		t.Fatalf("failed to update auth: %v", errUpdate)
	}

	// Stale resolution tries to publish gemini-2.5-flash using old fence
	staleModels := []*ModelInfo{{ID: "gemini-2.5-flash"}}
	service.registerResolvedModelsForAuth(registered, oldFence, "gemini", staleModels)

	// Stale publication was superseded; latest was reconciled with excluded_models="gemini-2.5-flash".
	// So gemini-2.5-flash must NOT be supported!
	if reg.ClientSupportsModel(clientID, "gemini-2.5-flash") {
		t.Fatal("gemini-2.5-flash should be excluded after reconciliation of latest auth")
	}
	// But gemini-2.5-pro should be supported!
	if !reg.ClientSupportsModel(clientID, "gemini-2.5-pro") {
		t.Fatal("expected gemini-2.5-pro to be registered after reconciliation of latest auth")
	}
}
