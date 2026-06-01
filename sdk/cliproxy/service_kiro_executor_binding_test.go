package cliproxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	internalregistry "github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

func TestEnsureExecutorsForAuth_KiroBindsKiroExecutor(t *testing.T) {
	service := &Service{
		cfg:         &config.Config{},
		coreManager: coreauth.NewManager(nil, nil, nil),
	}
	auth := &coreauth.Auth{
		ID:       "kiro-auth-1",
		Provider: "kiro",
		Status:   coreauth.StatusActive,
	}

	service.ensureExecutorsForAuth(auth)
	resolved, ok := service.coreManager.Executor("kiro")
	if !ok || resolved == nil {
		t.Fatal("expected kiro executor after bind")
	}
	if _, isKiro := resolved.(*executor.KiroExecutor); !isKiro {
		t.Fatalf("executor type = %T, want *executor.KiroExecutor", resolved)
	}
}

func TestRegisterModelsForAuth_KiroUsesLiveCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ListAvailableModels" {
			t.Fatalf("path = %s, want /ListAvailableModels", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"models":[{"modelId":"LIVE_MODEL","modelName":"Live Model"}]}`)
	}))
	defer server.Close()

	service := &Service{cfg: &config.Config{}}
	auth := &coreauth.Auth{
		ID:       "kiro-live-models",
		Provider: "kiro",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{
			"type":         "kiro",
			"access_token": "access-1",
			"models_url":   server.URL + "/ListAvailableModels",
		},
	}
	registry := internalregistry.GetGlobalRegistry()
	registry.UnregisterClient(auth.ID)
	t.Cleanup(func() {
		registry.UnregisterClient(auth.ID)
	})

	service.registerModelsForAuth(auth)

	models := registry.GetModelsForClient(auth.ID)
	if len(models) != 4 {
		t.Fatalf("models len = %d, want 4 live variants", len(models))
	}
	foundStaticFallback := false
	foundLive := false
	for _, model := range models {
		if model.ID == "claude-sonnet-4.5" {
			foundStaticFallback = true
		}
		if model.ID == "LIVE_MODEL-thinking-agentic" {
			foundLive = true
		}
	}
	if foundStaticFallback {
		t.Fatal("static fallback model registered despite live catalog")
	}
	if !foundLive {
		t.Fatalf("live thinking-agentic variant not registered: %#v", models)
	}
}

func TestRegisterLoadedCoreAuths_PopulatesKiroFallbackModels(t *testing.T) {
	auth := &coreauth.Auth{
		ID:       "kiro-loaded-startup",
		Provider: "kiro",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{
			"type": "kiro",
		},
	}
	manager := coreauth.NewManager(nil, nil, nil)
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service := &Service{
		cfg:         &config.Config{},
		coreManager: manager,
	}
	registry := internalregistry.GetGlobalRegistry()
	registry.UnregisterClient(auth.ID)
	t.Cleanup(func() {
		registry.UnregisterClient(auth.ID)
	})

	service.registerLoadedCoreAuths(context.Background())

	models := registry.GetModelsForClient(auth.ID)
	if len(models) == 0 {
		t.Fatal("expected startup registration to populate Kiro fallback models")
	}
}
