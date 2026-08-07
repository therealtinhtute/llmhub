package nativeproviders

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

type testStore struct {
	records map[string][]ResourceRecord
}

func (s *testStore) ListNativeProviderResources(_ context.Context, provider string) ([]ResourceRecord, error) {
	return append([]ResourceRecord(nil), s.records[provider]...), nil
}

func (s *testStore) SaveNativeProviderResource(_ context.Context, provider, id string, payload []byte) error {
	s.records[provider] = append(s.records[provider], ResourceRecord{Provider: provider, ID: id, Payload: payload})
	return nil
}

func (s *testStore) DeleteNativeProviderResource(_ context.Context, provider, id string) error {
	return nil
}

func TestValidateResourceRequiresOpenRouterAPIKey(t *testing.T) {
	t.Parallel()

	resource := &OpenRouterResource{ID: "openrouter-1"}
	if err := ValidateResource(ProviderOpenRouter, resource); err == nil {
		t.Fatal("expected OpenRouter API key validation error")
	}
}

func TestOpenCodeAllowsZeroKeyResource(t *testing.T) {
	t.Parallel()

	payload, err := EncodeResource(ProviderOpenCode, &OpenCodeResource{
		ID:      "opencode-1",
		Enabled: true,
		Models:  []Model{{Name: "free-model"}},
	})
	if err != nil {
		t.Fatalf("EncodeResource() error = %v", err)
	}

	var decoded OpenCodeResource
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode encoded resource: %v", err)
	}
	if decoded.APIKey != "" {
		t.Fatalf("expected zero-key resource, got API key %q", decoded.APIKey)
	}
	if len(decoded.Models) != 1 || decoded.Models[0].Alias != "free-model" {
		t.Fatalf("expected normalized model alias, got %#v", decoded.Models)
	}
}

func TestToPublicResourceRedactsAPIKey(t *testing.T) {
	t.Parallel()

	resource := &OpenRouterResource{ID: "openrouter-1", Enabled: true, APIKey: "sk-secret-value"}
	publicResource, err := ToPublicResource(ProviderOpenRouter, resource)
	if err != nil {
		t.Fatalf("ToPublicResource() error = %v", err)
	}
	if !publicResource.APIKeyPresent || publicResource.APIKeyPreview != "sk-s...alue" {
		t.Fatalf("unexpected redacted resource: %#v", publicResource)
	}
	encoded, err := json.Marshal(publicResource)
	if err != nil {
		t.Fatalf("marshal public resource: %v", err)
	}
	if strings.Contains(string(encoded), resource.APIKey) {
		t.Fatalf("public resource leaked API key: %s", encoded)
	}
}

func TestHydrateConfigReplacesNativeProjections(t *testing.T) {
	t.Parallel()

	openRouterPayload, err := EncodeResource(ProviderOpenRouter, &OpenRouterResource{
		ID:      "router-1",
		Enabled: true,
		APIKey:  "sk-router",
		Models:  []Model{{Name: "router-model", Alias: "router"}},
	})
	if err != nil {
		t.Fatalf("encode OpenRouter resource: %v", err)
	}
	openCodePayload, err := EncodeResource(ProviderOpenCode, &OpenCodeResource{
		ID:      "code-1",
		Enabled: false,
		Models:  []Model{{Name: "code-model"}},
	})
	if err != nil {
		t.Fatalf("encode OpenCode resource: %v", err)
	}

	cfg := &config.Config{OpenAICompatibility: []config.OpenAICompatibility{
		{Name: "generic", BaseURL: "https://generic.example/v1"},
		{Name: ProjectedName(ProviderOpenRouter, "old")},
	}}
	store := &testStore{records: map[string][]ResourceRecord{
		ProviderOpenRouter: {{Provider: ProviderOpenRouter, ID: "router-1", Payload: openRouterPayload}},
		ProviderOpenCode:   {{Provider: ProviderOpenCode, ID: "code-1", Payload: openCodePayload}},
	}}

	if err := HydrateConfig(context.Background(), cfg, store); err != nil {
		t.Fatalf("HydrateConfig() error = %v", err)
	}
	if len(cfg.OpenAICompatibility) != 3 {
		t.Fatalf("expected generic plus two native projections, got %d", len(cfg.OpenAICompatibility))
	}
	if cfg.OpenAICompatibility[0].Name != "generic" {
		t.Fatalf("generic config was not preserved: %#v", cfg.OpenAICompatibility)
	}
	router := cfg.OpenAICompatibility[1]
	if router.Name != "openrouter:router-1" || router.APIKeyEntries[0].APIKey != "sk-router" || !router.Passthrough {
		t.Fatalf("unexpected OpenRouter projection: %#v", router)
	}
	code := cfg.OpenAICompatibility[2]
	if code.Name != "opencode:code-1" || len(code.APIKeyEntries) != 0 || !code.Disabled || !code.Passthrough {
		t.Fatalf("unexpected OpenCode zero-key projection: %#v", code)
	}
}

func TestStripProjectedEntries(t *testing.T) {
	t.Parallel()

	entries := []config.OpenAICompatibility{
		{Name: "generic"},
		{Name: "openrouter:one"},
		{Name: "opencode:two"},
	}
	stripped := StripProjectedEntries(entries)
	if len(stripped) != 1 || stripped[0].Name != "generic" {
		t.Fatalf("unexpected stripped entries: %#v", stripped)
	}
}

func TestFetchRemoteModelsOpenRouterWithoutKeyUsesFallback(t *testing.T) {
	t.Parallel()

	models, source, err := FetchRemoteModels(context.Background(), ProviderOpenRouter, "")
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if source != "fallback" || len(models) == 0 {
		t.Fatalf("expected fallback catalog, source=%q models=%d", source, len(models))
	}
}
