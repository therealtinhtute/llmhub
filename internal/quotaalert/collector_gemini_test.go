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

func TestGeminiCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_gemini_usage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var sawCodeAssist bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer gemini-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		switch r.URL.Path {
		case geminiCLIQuotaPath:
			if body["project"] != "project-1" {
				t.Fatalf("quota project = %v", body["project"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture)
		case geminiCLICodeAssistPath:
			sawCodeAssist = true
			http.Error(w, "supplement unavailable", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
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
	collector := &GeminiCLICollector{httpClient: client, now: func() time.Time {
		return time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	}}
	auth := collectorTestAuth{
		id:       "gemini-auth-1",
		provider: ProviderGeminiCLI,
		label:    "Gemini Account",
		metadata: map[string]any{
			"access_token": "gemini-secret-token",
			"project_id":   "project-1",
		},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if !sawCodeAssist {
		t.Fatal("code-assist enrichment was not attempted")
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	if len(byKey) != 3 {
		t.Fatalf("observations = %d (%v), want 3", len(byKey), byKey)
	}
	if got := byKey["gemini-flash-series/requests"]; got.Remaining != 20 || !got.ResetKnown {
		t.Fatalf("flash series = %#v", got)
	}
	if got := byKey["gemini-pro-series/tokens"]; got.Remaining != 0 || !got.ExplicitlyExhausted {
		t.Fatalf("pro series = %#v", got)
	}
	if got := byKey["custom-model/default"]; got.Remaining != 75 {
		t.Fatalf("custom model = %#v", got)
	}
}

func TestGeminiCollectorUsageFailureIsFatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == geminiCLIQuotaPath {
			http.Error(w, "quota unavailable", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"paidTier":{"id":"g1-ultra-tier"}}`))
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
	collector := &GeminiCLICollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "gemini-auth-1",
		provider: ProviderGeminiCLI,
		label:    "Gemini Account",
		metadata: map[string]any{
			"access_token": "gemini-secret-token",
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
	if strings.Contains(err.Error(), "gemini-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
