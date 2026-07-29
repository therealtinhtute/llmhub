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

func TestKiroCollector(t *testing.T) {
	fixture, err := os.ReadFile("testdata/collector_kiro_usage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer kiro-secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		switch len(calls) {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != kiroUsageLimitsPath {
				t.Fatalf("first request = %s %s, want getUsageLimits GET", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("profileArn"); got != "" {
				t.Fatalf("codewhisperer GET profileArn = %q, want empty", got)
			}
			http.Error(w, "not here", http.StatusNotFound)
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/" {
				t.Fatalf("second request = %s %s, want root POST", r.Method, r.URL.Path)
			}
			if got := r.Header.Get("x-amz-target"); got != kiroCodeWhispererGetUsageTarget {
				t.Fatalf("x-amz-target = %q, want %q", got, kiroCodeWhispererGetUsageTarget)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture)
		default:
			t.Fatalf("unexpected extra request %d: %s %s", len(calls), r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	observedAt := time.Date(2026, 7, 29, 3, 0, 0, 123456789, time.UTC)
	collector := &KiroCollector{httpClients: []*CollectorHTTPClient{client}, now: func() time.Time { return observedAt }}
	auth := collectorTestAuth{
		id:       "kiro-auth-1",
		provider: ProviderKiro,
		label:    "Kiro Account",
		metadata: map[string]any{
			"access_token": "kiro-secret-token",
			"profile_arn":  "arn:aws:codewhisperer:us-west-2:123:profile/test",
			"quota_url":    "https://attacker.example/getUsageLimits",
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
	if got := byKey["agentic-request/default"]; got.Remaining != 25 || !got.ResetKnown || !got.ResetAt.Equal(time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("agentic-request = %#v", got)
	}
	if got := byKey["chat-request/default"]; got.Remaining != 50 || !got.ResetKnown || !got.ResetAt.Equal(time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("chat-request = %#v", got)
	}
	if got := byKey["chat-request-freetrial/default"]; got.Remaining != 80 || !got.ResetKnown || !got.ResetAt.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("chat-request-freetrial = %#v", got)
	}
	if got := byKey["credit/default"]; got.Remaining != 0 || !got.ExplicitlyExhausted {
		t.Fatalf("credit = %#v, want exhausted", got)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %#v, want two fixed-host attempts", calls)
	}
}

func TestKiroCollectorFallsBackToQHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != kiroUsageLimitsPath {
			t.Fatalf("request = %s %s, want q GET", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("profileArn"); got == "" {
			t.Fatal("q fallback missing profileArn query")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"usageBreakdownList":[{"resourceType":"AGENTIC_REQUEST","currentUsage":1,"usageLimit":5}]}`))
	}))
	defer server.Close()
	codeWhispererServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not here", http.StatusNotFound)
	}))
	defer codeWhispererServer.Close()
	codeWhispererClient, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: codeWhispererServer.URL, Timeout: time.Second, Client: codeWhispererServer.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient(codewhisperer) error = %v", err)
	}
	qClient, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient(q) error = %v", err)
	}
	collector := &KiroCollector{httpClients: []*CollectorHTTPClient{codeWhispererClient, qClient}, now: time.Now}
	auth := collectorTestAuth{
		id:       "kiro-auth-1",
		provider: ProviderKiro,
		label:    "Kiro Account",
		metadata: map[string]any{
			"access_token": "kiro-secret-token",
			"profile_arn":  "arn:aws:codewhisperer:us-west-2:123:profile/test",
		},
	}
	observations, err := collector.Collect(context.Background(), auth)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(observations) != 1 || observations[0].Remaining != 80 {
		t.Fatalf("observations = %#v, want fallback success", observations)
	}
}

func TestKiroCollectorFailureIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota unavailable", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{BaseURL: server.URL, Timeout: time.Second, Client: server.Client()})
	if err != nil {
		t.Fatalf("NewCollectorHTTPClient() error = %v", err)
	}
	collector := &KiroCollector{httpClients: []*CollectorHTTPClient{client, client}, now: time.Now}
	auth := collectorTestAuth{
		id:       "kiro-auth-1",
		provider: ProviderKiro,
		label:    "Kiro Account",
		metadata: map[string]any{"access_token": "kiro-secret-token"},
	}
	_, err = collector.Collect(context.Background(), auth)
	if err == nil {
		t.Fatal("Collect() error = nil")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("Collect() error = %q, want HTTP 429", err.Error())
	}
	if strings.Contains(err.Error(), "kiro-secret-token") {
		t.Fatalf("Collect() leaked token: %q", err.Error())
	}
}
