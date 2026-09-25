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
			"source": "oauth",
		},
		metadata: map[string]any{
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

func TestClaudeCollectorCollectsFableLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case claudeUsagePath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"five_hour": {"utilization": 25, "resets_at": "2026-07-29T04:00:00Z"},
				"iguana_necktie": {"utilization": 99, "resets_at": "2026-07-30T00:00:00Z"},
				"limits": [
					{"kind": "weekly_scoped", "scope": {"model": {"display_name": "other-model"}}, "percent": 10},
					{"kind": "weekly_scoped", "scope": {"model": {"display_name": "fable"}}, "is_active": false, "percent": 30, "resets_at": "2026-08-01T00:00:00Z"},
					{"kind": "weekly_scoped", "scope": {"model": {"display_name": "Fable 5"}}, "is_active": true, "percent": 60, "resets_at": "2026-08-02T00:00:00Z"},
					{"kind": "daily", "scope": {"model": {"display_name": "fable"}}, "percent": 5}
				]
			}`))
		case claudeProfilePath:
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &ClaudeCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "claude-auth-1",
		provider: ProviderClaude,
		label:    "Claude Account",
		metadata: map[string]any{"access_token": "claude-secret-token"},
	}

	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v — limits[] payload must decode", err)
	}
	byKey := map[string]Observation{}
	for _, observation := range observations {
		byKey[observation.Identity.Resource+"/"+observation.Identity.Window] = observation
	}
	// Active fable candidate wins over the first-listed inactive one.
	if got := byKey["fable/seven-day"]; got.Remaining != 40 || !got.ResetKnown {
		t.Fatalf("fable observation = %#v, want 40 remaining", got)
	}
	if _, exists := byKey["iguana-necktie/default"]; exists {
		t.Fatal("iguana_necktie collected alongside the fable limit — screens would double-count")
	}
	if got := byKey["messages/five-hour"]; got.Remaining != 75 {
		t.Fatalf("five-hour observation = %#v", got)
	}
}

func TestClaudeCollectorRejectsDriftedPayloadShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case claudeUsagePath:
			w.Header().Set("Content-Type", "application/json")
			// 2026-07-05 endpoint drift: only the new rate_limit_info shape.
			_, _ = w.Write([]byte(`{"rate_limit_info": {"isUsingOverage": false, "overageResetsAt": "2026-08-01T00:00:00Z", "overageStatus": "none", "rateLimitType": "default", "resetsAt": "2026-07-30T00:00:00Z"}}`))
		case claudeProfilePath:
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &ClaudeCollector{httpClient: client, now: time.Now}
	auth := collectorTestAuth{
		id:       "claude-auth-1",
		provider: ProviderClaude,
		label:    "Claude Account",
		metadata: map[string]any{"access_token": "claude-secret-token"},
	}

	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil — drifted payload must not report silent healthy")
	}
	if got := classifyCollectorError(err); got != FailureDecode {
		t.Fatalf("classifyCollectorError = %q, want %q", got, FailureDecode)
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
		metadata: map[string]any{
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
