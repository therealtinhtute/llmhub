package quotaalert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestKimiCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_kimi_usage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer kimi-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.Path != kimiUsagePath {
			t.Fatalf("path = %q, want %q", r.URL.Path, kimiUsagePath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	observedAt := time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	collector := &KimiCollector{httpClient: client, now: func() time.Time { return observedAt }}
	auth := collectorTestAuth{
		id:       "kimi-auth-1",
		provider: ProviderKimi,
		label:    "Kimi Account",
		metadata: map[string]any{"access_token": "kimi-secret-token"},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	if len(byKey) != 3 {
		t.Fatalf("observations = %d (%v), want 3", len(byKey), byKey)
	}
	if got := byKey["summary/default"]; got.Remaining != 80 || !got.ResetKnown {
		t.Fatalf("summary = %#v", got)
	}
	if got := byKey["limit-0/default"]; got.Remaining != 20 || !got.ResetKnown || !got.ResetAt.Equal(observedAt.Add(time.Hour).Truncate(time.Second)) {
		t.Fatalf("limit-0 = %#v", got)
	}
	if got := byKey["limit-1/default"]; got.Remaining != 0 || !got.ExplicitlyExhausted {
		t.Fatalf("limit-1 = %#v", got)
	}
}

func TestKimiCollectorFailureIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota unavailable", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &KimiCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "kimi-auth-1",
		provider: ProviderKimi,
		label:    "Kimi Account",
		metadata: map[string]any{"access_token": "kimi-secret-token"},
	}
	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "kimi-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}

func TestXAICollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_xai_billing.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer xai-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.Path != xaiBillingPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, xaiBillingPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &XAICollector{httpClient: client, now: func() time.Time {
		return time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	}}
	auth := collectorTestAuth{
		id:       "xai-auth-1",
		provider: ProviderXAI,
		label:    "xAI Account",
		metadata: map[string]any{"access_token": "xai-secret-token"},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(observations))
	}
	observation := observations[0]
	if observation.Identity.Provider != ProviderXAI || observation.Identity.Resource != "monthly-credits" || observation.Identity.Window != "billing-period" {
		t.Fatalf("identity = %#v", observation.Identity)
	}
	if observation.Remaining != 75 || !observation.ResetKnown || observation.ExplicitlyExhausted {
		t.Fatalf("observation = %#v", observation)
	}
}

func TestXAICollectorFailureIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota unavailable", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &XAICollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "xai-auth-1",
		provider: ProviderXAI,
		label:    "xAI Account",
		metadata: map[string]any{"access_token": "xai-secret-token"},
	}
	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "xai-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
