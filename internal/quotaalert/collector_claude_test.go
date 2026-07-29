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

func TestClaudeCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_claude_usage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var sawProfile bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer claude-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case claudeUsagePath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture)
		case claudeProfilePath:
			sawProfile = true
			http.Error(w, "profile unavailable", http.StatusInternalServerError)
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
	collector := &ClaudeCollector{httpClient: client, now: func() time.Time {
		return time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	}}
	auth := collectorTestAuth{
		id:       "claude-auth-1",
		provider: ProviderClaude,
		label:    "Claude Account",
		attributes: map[string]string{
			"access_token": "claude-secret-token",
		},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if !sawProfile {
		t.Fatal("profile enrichment was not attempted")
	}
	if len(observations) != 2 {
		t.Fatalf("observations = %d, want 2", len(observations))
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	fiveHour := byKey["messages/five-hour"]
	if fiveHour.Identity.Provider != ProviderClaude || fiveHour.Remaining != 75 || !fiveHour.ResetKnown {
		t.Fatalf("five-hour observation = %#v", fiveHour)
	}
	opus := byKey["opus/seven-day"]
	if opus.Remaining != 0 || !opus.ExplicitlyExhausted {
		t.Fatalf("opus observation = %#v, want exhausted zero remaining", opus)
	}
}

func TestClaudeCollectorUsageFailureIsFatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == claudeUsagePath {
			http.Error(w, "usage unavailable", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"account":{"has_claude_pro":true}}`))
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
	collector := &ClaudeCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "claude-auth-1",
		provider: ProviderClaude,
		label:    "Claude Account",
		attributes: map[string]string{
			"access_token": "claude-secret-token",
		},
	}

	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "claude-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
