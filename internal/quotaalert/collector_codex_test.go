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

func TestCodexCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_codex_usage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var sawResetCredits bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer codex-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("Chatgpt-Account-Id"); got != "account-123" {
			t.Fatalf("Chatgpt-Account-Id = %q", got)
		}
		switch r.URL.Path {
		case codexUsagePath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture)
		case codexResetCreditsPath:
			sawResetCredits = true
			http.Error(w, "reset credits unavailable", http.StatusInternalServerError)
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
	collector := &CodexCollector{httpClient: client, now: func() time.Time {
		return time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	}}
	auth := collectorTestAuth{
		id:       "codex-auth-1",
		provider: ProviderCodex,
		label:    "Codex Account",
		attributes: map[string]string{
			"source": "oauth",
		},
		metadata: map[string]any{
			"access_token": "codex-secret-token",
			"id_token":     "codex-id-token",
			"account_id":   "account-123",
		},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if !sawResetCredits {
		t.Fatal("reset-credit enrichment was not attempted")
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	if len(byKey) != 6 {
		t.Fatalf("observations = %d (%v), want 6", len(byKey), byKey)
	}
	if got := byKey["code/five-hour"]; got.Remaining != 20 || !got.ResetKnown {
		t.Fatalf("code five-hour = %#v", got)
	}
	if got := byKey["code/weekly"]; got.Remaining != 70 || !got.ResetKnown {
		t.Fatalf("code weekly = %#v", got)
	}
	if got := byKey["code-review/five-hour"]; got.Remaining != 0 || !got.ExplicitlyExhausted {
		t.Fatalf("code-review five-hour = %#v, want exhausted", got)
	}
	if got := byKey["code-review/weekly"]; got.Remaining != 55 || got.ExplicitlyExhausted {
		t.Fatalf("code-review weekly = %#v, want 55 remaining", got)
	}
	if got := byKey["additional-tool-calls/five-hour"]; got.Remaining != 50 || !got.ResetKnown {
		t.Fatalf("additional primary = %#v", got)
	}
	if _, exists := byKey["additional-bare-denial/five-hour"]; exists {
		t.Fatal("bare denial without usage/reset evidence became an observation")
	}
}

func TestCodexCollectorClassifiesMonthlyWindows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case codexUsagePath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"rate_limit": {
					"allowed": true,
					"primary_window": {"used_percent": 40, "limit_window_seconds": 18000, "reset_after_seconds": 3600},
					"secondary_window": {"used_percent": 20, "limit_window_seconds": 2678400, "reset_at": 1788042000}
				},
				"additional_rate_limits": [
					{
						"limit_name": "Team Plan",
						"rate_limit": {
							"primary_window": {"used_percent": 10, "limit_window_seconds": 2419200, "reset_at": 1788042000},
							"secondary_window": {"used_percent": 5, "limit_window_seconds": 604800, "reset_at": 1785974400}
						}
					}
				]
			}`))
		case codexResetCreditsPath:
			_, _ = w.Write([]byte(`{"available_count":0}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &CodexCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "codex-auth-1",
		provider: ProviderCodex,
		label:    "Codex Account",
		metadata: map[string]any{"access_token": "codex-secret-token"},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	if got := byKey["code/five-hour"]; got.Remaining != 60 {
		t.Fatalf("code five-hour = %#v", got)
	}
	if got := byKey["code/monthly"]; got.Remaining != 80 || !got.ResetKnown {
		t.Fatalf("code monthly = %#v, want 80 remaining", got)
	}
	if _, exists := byKey["code/weekly"]; exists {
		t.Fatal("monthly-duration secondary was mislabeled weekly")
	}
	if got := byKey["additional-team-plan/monthly"]; got.Remaining != 90 {
		t.Fatalf("additional monthly = %#v, want 90 remaining", got)
	}
	if got := byKey["additional-team-plan/weekly"]; got.Remaining != 95 {
		t.Fatalf("additional weekly = %#v, want 95 remaining", got)
	}
	if _, exists := byKey["additional-team-plan/five-hour"]; exists {
		t.Fatal("monthly-duration primary was mislabeled five-hour")
	}
}

func TestCodexCollectorUsageFailureIsFatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == codexUsagePath {
			http.Error(w, "usage unavailable", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"available_count":1}`))
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
	collector := &CodexCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "codex-auth-1",
		provider: ProviderCodex,
		label:    "Codex Account",
		metadata: map[string]any{
			"access_token": "codex-secret-token",
		},
	}

	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "codex-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
