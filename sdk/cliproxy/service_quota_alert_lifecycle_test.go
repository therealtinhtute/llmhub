package cliproxy

import (
	"bytes"
	"context"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

func TestServiceQuotaAlertBuilderOptionIsOptional(t *testing.T) {
	builder := NewBuilder().WithConfig(&config.Config{}).WithConfigPath("config.yaml")
	service, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if service.quotaAlertService != nil {
		t.Fatal("quota alert service should be nil without explicit option")
	}

	quotaSvc, err := quotaalert.NewService(quotaalert.ServiceConfig{Store: newQuotaAlertLifecycleStore()})
	if err != nil {
		t.Fatalf("quotaalert.NewService() error = %v", err)
	}
	service, err = builder.WithQuotaAlertService(quotaSvc).Build()
	if err != nil {
		t.Fatalf("Build(with quota service) error = %v", err)
	}
	if service.quotaAlertService != quotaSvc {
		t.Fatal("quota alert service was not attached")
	}

	cipher, err := quotaalert.NewSecretCipher("runtime", bytes.Repeat([]byte{3}, quotaalert.SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	service, err = NewBuilder().
		WithConfig(&config.Config{}).
		WithConfigPath("config.yaml").
		WithQuotaAlertStore(newQuotaAlertLifecycleStore()).
		WithQuotaAlertSecretCipher(cipher).
		Build()
	if err != nil {
		t.Fatalf("Build(with quota store and cipher) error = %v", err)
	}
	if service.quotaAlertService == nil {
		t.Fatal("quota alert service was not built from store and cipher")
	}
}

func TestQuotaAlertAuthSourceListsSupportedCoreAuths(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	manager.Register(context.Background(), &coreauth.Auth{ID: "auth-1", Provider: "gemini", Label: "Gemini Primary", ProxyURL: "http://proxy", Attributes: map[string]string{"access_token": "secret"}})
	manager.Register(context.Background(), &coreauth.Auth{ID: "auth-2", Provider: "claude", FileName: "claude.json"})
	manager.Register(context.Background(), &coreauth.Auth{ID: "disabled", Provider: "codex", Disabled: true})
	manager.Register(context.Background(), &coreauth.Auth{ID: "unsupported", Provider: "openai"})

	source := NewQuotaAlertAuthSource(manager)
	auths, err := source.ListQuotaAlertAuths(context.Background())
	if err != nil {
		t.Fatalf("ListQuotaAlertAuths() error = %v", err)
	}
	if len(auths) != 2 {
		t.Fatalf("len(auths) = %d, want 2", len(auths))
	}
	authByID := make(map[string]quotaalert.AuthSnapshot, len(auths))
	for _, auth := range auths {
		authByID[auth.AuthID()] = auth
	}
	gemini := authByID["auth-1"]
	if gemini == nil || gemini.Provider() != quotaalert.ProviderGeminiCLI || gemini.RedactedLabel() != "Gemini Primary" || gemini.ProxyURL() != "http://proxy" {
		t.Fatalf("gemini auth = %#v", gemini)
	}
	if value, ok := gemini.Attribute("access_token"); !ok || value != "secret" {
		t.Fatalf("attribute clone = %q ok=%t", value, ok)
	}
	claude := authByID["auth-2"]
	if claude == nil || claude.Provider() != quotaalert.ProviderClaude || claude.RedactedLabel() != "claude.json" {
		t.Fatalf("claude auth = %#v", claude)
	}
}

type quotaAlertLifecycleStore struct{ quotaalert.Store }

func newQuotaAlertLifecycleStore() *quotaAlertLifecycleStore { return &quotaAlertLifecycleStore{} }

func (s *quotaAlertLifecycleStore) LoadSettings(context.Context) (quotaalert.Settings, error) {
	return quotaalert.DefaultSettings(), nil
}
