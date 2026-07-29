package quotaalert

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type collectorTestAuth struct {
	id         string
	provider   Provider
	label      string
	proxyURL   string
	attributes map[string]string
	metadata   map[string]any
}

func (a collectorTestAuth) AuthID() string        { return a.id }
func (a collectorTestAuth) Provider() Provider    { return a.provider }
func (a collectorTestAuth) RedactedLabel() string { return a.label }
func (a collectorTestAuth) ProxyURL() string      { return a.proxyURL }
func (a collectorTestAuth) Attribute(key string) (string, bool) {
	value, ok := a.attributes[key]
	return value, ok
}
func (a collectorTestAuth) Metadata(key string) (any, bool) {
	value, ok := a.metadata[key]
	return value, ok
}

func TestDefaultCollectorRegistry(t *testing.T) {
	registry, err := NewDefaultCollectorRegistry()
	if err != nil {
		t.Fatalf("NewDefaultCollectorRegistry() error = %v", err)
	}
	providers := registry.Providers()
	want := SupportedProviders()
	sort.Slice(want, func(left, right int) bool { return want[left] < want[right] })
	if len(providers) != len(want) {
		t.Fatalf("Providers() = %v, want %v", providers, want)
	}
	for index, provider := range want {
		if providers[index] != provider {
			t.Fatalf("Providers()[%d] = %q, want %q", index, providers[index], provider)
		}
		collector, err := registry.Collector(provider, CollectorDependencies{})
		if err != nil {
			t.Fatalf("Collector(%q) error = %v", provider, err)
		}
		if collector == nil {
			t.Fatalf("Collector(%q) = nil", provider)
		}
	}
}

func TestCollectorRegistry(t *testing.T) {
	registry := NewCollectorRegistry()
	collector := CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
		return nil, nil
	})
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return collector, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return collector, nil
	}); err == nil {
		t.Fatal("Register(duplicate) error = nil")
	}
	if err := registry.Register(Provider("unknown"), func(CollectorDependencies) (Collector, error) {
		return collector, nil
	}); err == nil {
		t.Fatal("Register(unknown provider) error = nil")
	}
	got, err := registry.Collector(ProviderClaude, CollectorDependencies{})
	if err != nil {
		t.Fatalf("Collector() error = %v", err)
	}
	if got == nil {
		t.Fatal("Collector() = nil")
	}
	if _, err = registry.Collector(ProviderCodex, CollectorDependencies{}); err == nil {
		t.Fatal("Collector(unregistered) error = nil")
	}
	providers := registry.Providers()
	if len(providers) != 1 || providers[0] != ProviderClaude {
		t.Fatalf("Providers() = %v, want [claude]", providers)
	}
}

func TestCollectorRegistryClonesAuthSnapshot(t *testing.T) {
	auth := collectorTestAuth{
		id:       "auth-1",
		provider: ProviderClaude,
		label:    "Account 1",
		attributes: map[string]string{
			"access_token": "secret-access-token",
			"ignored":      "ignored",
		},
		metadata: map[string]any{
			"refresh_token": "secret-refresh-token",
			"ignored":       "ignored",
		},
	}
	clone, err := CloneAuthSnapshot(auth, []string{"access_token"}, []string{"refresh_token"})
	if err != nil {
		t.Fatalf("CloneAuthSnapshot() error = %v", err)
	}
	auth.attributes["access_token"] = "mutated"
	auth.metadata["refresh_token"] = "mutated"
	if value, _ := clone.Attribute("access_token"); value != "secret-access-token" {
		t.Fatalf("cloned access token = %q", value)
	}
	if value, _ := clone.Metadata("refresh_token"); value != "secret-refresh-token" {
		t.Fatalf("cloned refresh token = %q", value)
	}
	if _, ok := clone.Attribute("ignored"); ok {
		t.Fatal("unexpected ignored attribute in clone")
	}
}

func TestCollectorHTTP(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_known.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quota" {
			t.Fatalf("request path = %q, want /quota", r.URL.Path)
		}
		if attempts.Add(1) == 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
		BaseURL:      server.URL,
		AllowedHosts: []string{strings.TrimPrefix(server.URL, "http://")},
		Timeout:      time.Second,
		MaxBodyBytes: int64(len(fixture) + 8),
		Client:       server.Client(),
	})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	var refreshed atomic.Int32
	var payload struct {
		Remaining int       `json:"remaining"`
		ResetAt   time.Time `json:"reset_at"`
	}
	if err = client.JSON(
		context.Background(),
		collectorTestAuth{},
		http.MethodGet,
		"/quota",
		map[string]string{"Authorization": "Bearer secret"},
		&payload,
		func(context.Context, AuthSnapshot) error {
			refreshed.Add(1)
			return nil
		},
	); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if attempts.Load() != 2 || refreshed.Load() != 1 {
		t.Fatalf("attempts = %d refreshes = %d, want 2 and 1", attempts.Load(), refreshed.Load())
	}
	if payload.Remaining != 42 || payload.ResetAt.IsZero() {
		t.Fatalf("payload = %#v", payload)
	}

	var ignored any
	if err = client.JSON(context.Background(), nil, http.MethodGet, "https://attacker.example/quota", nil, &ignored, nil); err == nil {
		t.Fatal("JSON(absolute URL) error = nil")
	}
	if err = client.JSON(context.Background(), nil, http.MethodGet, "quota", nil, &ignored, nil); err == nil {
		t.Fatal("JSON(non-root path) error = nil")
	}
}

func TestCollectorHTTPSanitization(t *testing.T) {
	auth := collectorTestAuth{
		attributes: map[string]string{
			"access_token": "secret-access-token",
		},
		metadata: map[string]any{
			"refresh_token": "secret-refresh-token",
		},
	}
	message := RedactCollectorError(
		errors.New("failed with secret-access-token and secret-refresh-token"),
		auth,
	)
	if strings.Contains(message, "secret-access-token") || strings.Contains(message, "secret-refresh-token") {
		t.Fatalf("redacted message still contains secret: %q", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("redacted message = %q, want redaction marker", message)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"too":"large"}`))
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		MaxBodyBytes: 4,
		Client:       server.Client(),
	})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	var payload any
	if err = client.JSON(context.Background(), auth, http.MethodGet, "/quota", nil, &payload, nil); err == nil {
		t.Fatal("JSON(oversized body) error = nil")
	}
}
