package quotaalert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAntigravityCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_antigravity_models.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer antigravity-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.Path != antigravityQuotaPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["project"] != "project-1" {
			t.Fatalf("project = %v", body["project"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
		BaseURL: server.URL,
		Timeout: time.Second,
		Client:  server.Client(),
	})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &AntigravityCollector{httpClients: []*CollectorHTTPClient{client}, now: func() time.Time {
		return time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	}}
	auth := collectorTestAuth{
		id:       "antigravity-auth-1",
		provider: ProviderAntigravity,
		label:    "Antigravity Account",
		metadata: map[string]any{
			"access_token": "antigravity-secret-token",
			"project_id":   "project-1",
		},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	if len(byKey) != 4 {
		t.Fatalf("observations = %d (%v), want 4", len(byKey), byKey)
	}
	if got := byKey["claude-gpt/default"]; got.Remaining != 30 || !got.ResetKnown {
		t.Fatalf("claude/gpt = %#v", got)
	}
	if got := byKey["gemini-3-1-pro-series/default"]; got.Remaining != 40 || !got.ResetKnown {
		t.Fatalf("gemini pro = %#v", got)
	}
	if got := byKey["gemini-2-5-flash-lite/default"]; got.Remaining != 25 || !got.ResetKnown {
		t.Fatalf("flash-lite = %#v", got)
	}
	if got := byKey["gemini-image-preview/default"]; got.Remaining != 80 || !got.ResetKnown {
		t.Fatalf("image = %#v", got)
	}
}

func TestAntigravityCollectorFallsBackAcrossFixedHosts(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not here", http.StatusNotFound)
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":{"claude-sonnet-4-6":{"quotaInfo":{"remainingFraction":0.5}}}}`))
	}))
	defer second.Close()
	firstClient, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: first.URL, Timeout: time.Second, Client: first.Client()})
	if err != nil {
		t.Fatalf("first NewCollectorHTTPClient() error = %v", err)
	}
	secondClient, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: second.URL, Timeout: time.Second, Client: second.Client()})
	if err != nil {
		t.Fatalf("second NewCollectorHTTPClient() error = %v", err)
	}
	collector := &AntigravityCollector{httpClients: []*CollectorHTTPClient{firstClient, secondClient}, now: time.Now}
	auth := collectorTestAuth{
		id:       "antigravity-auth-1",
		provider: ProviderAntigravity,
		label:    "Antigravity Account",
		metadata: map[string]any{
			"access_token": "antigravity-secret-token",
			"project_id":   "project-1",
		},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(observations) != 1 || observations[0].Remaining != 50 {
		t.Fatalf("observations = %#v, want fallback success", observations)
	}
}

func TestAntigravityCollectorFailureIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota unavailable", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
		BaseURL: server.URL,
		Timeout: time.Second,
		Client:  server.Client(),
	})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &AntigravityCollector{httpClients: []*CollectorHTTPClient{client}, now: time.Now}
	auth := collectorTestAuth{
		id:       "antigravity-auth-1",
		provider: ProviderAntigravity,
		label:    "Antigravity Account",
		metadata: map[string]any{
			"access_token": "antigravity-secret-token",
			"project_id":   "project-1",
		},
	}

	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "antigravity-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
