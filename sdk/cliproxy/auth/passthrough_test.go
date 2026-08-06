package auth

import (
	"context"
	"testing"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
)

func newPassthroughEligibilityTestManager(t *testing.T, passthrough bool) (*Manager, *Auth) {
	t.Helper()
	cfg := &internalconfig.Config{
		OpenAICompatibility: []internalconfig.OpenAICompatibility{{
			Name:        "compat",
			Passthrough: passthrough,
		}},
	}
	m := NewManager(nil, nil, nil)
	m.SetConfig(cfg)

	auth := &Auth{
		ID:       "compat-auth-" + t.Name(),
		Provider: "compat",
		Status:   StatusActive,
		Attributes: map[string]string{
			"api_key":      "test-key",
			"compat_name":  "compat",
			"provider_key": "compat",
		},
	}
	if _, err := m.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}
	return m, auth
}

func TestAuthSupportsRouteModel_PassthroughAllowsUnlistedModel(t *testing.T) {
	m, auth := newPassthroughEligibilityTestManager(t, true)
	registered, ok := m.GetByID(auth.ID)
	if !ok {
		t.Fatalf("expected auth to be registered")
	}

	reg := registry.GetGlobalRegistry()
	if !m.authSupportsRouteModel(reg, registered, "some-unlisted-model") {
		t.Fatalf("expected passthrough auth to support an unlisted model")
	}
}

func TestAuthSupportsRouteModel_NonPassthroughRejectsUnlistedModel(t *testing.T) {
	m, auth := newPassthroughEligibilityTestManager(t, false)
	registered, ok := m.GetByID(auth.ID)
	if !ok {
		t.Fatalf("expected auth to be registered")
	}

	reg := registry.GetGlobalRegistry()
	if m.authSupportsRouteModel(reg, registered, "some-unlisted-model") {
		t.Fatalf("expected non-passthrough auth to reject an unlisted model")
	}
}

func TestRewriteModelForAuth_StripsPrefixBeforeForwarding(t *testing.T) {
	auth := &Auth{Prefix: "compat"}
	if got := rewriteModelForAuth("compat/some-unlisted-model", auth); got != "some-unlisted-model" {
		t.Fatalf("rewriteModelForAuth = %q, want %q", got, "some-unlisted-model")
	}
}

func TestRewriteModelForAuth_NoPrefixLeavesModelUnchanged(t *testing.T) {
	auth := &Auth{}
	if got := rewriteModelForAuth("some-unlisted-model", auth); got != "some-unlisted-model" {
		t.Fatalf("rewriteModelForAuth = %q, want %q", got, "some-unlisted-model")
	}
}

func TestAuthSupportsRouteModel_PassthroughStillHonorsRegisteredModels(t *testing.T) {
	m, auth := newPassthroughEligibilityTestManager(t, true)
	registered, ok := m.GetByID(auth.ID)
	if !ok {
		t.Fatalf("expected auth to be registered")
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "compat", []*registry.ModelInfo{{ID: "listed-model"}})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})

	if !m.authSupportsRouteModel(reg, registered, "listed-model") {
		t.Fatalf("expected registry-listed model to remain supported")
	}
}
