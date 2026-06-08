package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestParseKiroUsageLimits_PaidTrialOverage(t *testing.T) {
	checkedAt := time.Date(2026, 6, 7, 8, 0, 0, 0, time.UTC)
	payload := []byte(`{
		"usageBreakdownList":[{"resourceType":"AGENTIC_REQUEST","currentUsage":75,"usageLimit":100}],
		"nextDateReset":"2026-06-08T00:00:00Z",
		"subscriptionInfo":{"subscriptionType":"paid","subscriptionTitle":"Kiro Pro"},
		"freeTrialInfo":{"currentUsage":2,"usageLimit":10,"status":"active","expiresAt":"2026-06-30T00:00:00Z"},
		"overageStatus":"enabled",
		"overageCap":25,
		"overageRate":0.04,
		"currentOverages":3
	}`)

	got, err := ParseKiroUsageLimits(payload, checkedAt)
	if err != nil {
		t.Fatalf("ParseKiroUsageLimits() error = %v", err)
	}
	if !got.ProviderQuotaAvailable {
		t.Fatal("ProviderQuotaAvailable = false, want true")
	}
	if got.Current == nil || *got.Current != 75 {
		t.Fatalf("Current = %#v, want 75", got.Current)
	}
	if got.Limit == nil || *got.Limit != 100 {
		t.Fatalf("Limit = %#v, want 100", got.Limit)
	}
	if got.Percent == nil || *got.Percent != 75 {
		t.Fatalf("Percent = %#v, want 75", got.Percent)
	}
	if got.Remaining == nil || *got.Remaining != 25 {
		t.Fatalf("Remaining = %#v, want 25", got.Remaining)
	}
	if len(got.Quotas) != 2 {
		t.Fatalf("Quotas length = %d, want provider + free trial rows: %#v", len(got.Quotas), got.Quotas)
	}
	if got.Quotas[0].ResourceType != "agentic_request" || got.Quotas[0].Used == nil || *got.Quotas[0].Used != 75 || got.Quotas[0].Total == nil || *got.Quotas[0].Total != 100 {
		t.Fatalf("provider quota row = %#v, want agentic_request 75/100", got.Quotas[0])
	}
	if !got.Quotas[1].FreeTrial || got.Quotas[1].Used == nil || *got.Quotas[1].Used != 2 || got.Quotas[1].Total == nil || *got.Quotas[1].Total != 10 {
		t.Fatalf("free trial quota row = %#v, want 2/10", got.Quotas[1])
	}
	if got.NextResetAt == nil || got.NextResetAt.Format(time.RFC3339) != "2026-06-08T00:00:00Z" {
		t.Fatalf("NextResetAt = %#v, want 2026-06-08T00:00:00Z", got.NextResetAt)
	}
	if got.SubscriptionType != "paid" || got.SubscriptionTitle != "Kiro Pro" {
		t.Fatalf("subscription = %q/%q, want paid/Kiro Pro", got.SubscriptionType, got.SubscriptionTitle)
	}
	if got.TrialCurrent == nil || *got.TrialCurrent != 2 || got.TrialLimit == nil || *got.TrialLimit != 10 {
		t.Fatalf("trial = %#v/%#v, want 2/10", got.TrialCurrent, got.TrialLimit)
	}
	if got.TrialPercent == nil || *got.TrialPercent != 20 {
		t.Fatalf("TrialPercent = %#v, want 20", got.TrialPercent)
	}
	if got.TrialStatus != "active" || got.TrialExpiresAt == nil {
		t.Fatalf("trial status/expires = %q/%#v, want active/date", got.TrialStatus, got.TrialExpiresAt)
	}
	if got.OverageStatus != "enabled" || got.OverageCap == nil || *got.OverageCap != 25 || got.OverageRate == nil || *got.OverageRate != 0.04 || got.CurrentOverages == nil || *got.CurrentOverages != 3 {
		t.Fatalf("overage fields not normalized: %#v", got)
	}
}

func TestParseKiroUsageLimits_MultipleBreakdownRowsWithPrecision(t *testing.T) {
	payload := []byte(`{
		"usageBreakdownList":[
			{"resourceType":"AGENTIC_REQUEST","currentUsageWithPrecision":10.5,"usageLimitWithPrecision":20},
			{"resourceType":"CHAT_REQUEST","currentUsageWithPrecision":3,"usageLimitWithPrecision":30,
			 "freeTrialInfo":{"currentUsageWithPrecision":1,"usageLimitWithPrecision":5,"freeTrialExpiry":"2026-06-30T00:00:00Z"}}
		],
		"nextDateReset": 1780272000000
	}`)

	got, err := ParseKiroUsageLimits(payload, time.Date(2026, 6, 7, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ParseKiroUsageLimits() error = %v", err)
	}
	if len(got.Quotas) != 3 {
		t.Fatalf("Quotas length = %d, want 3 rows: %#v", len(got.Quotas), got.Quotas)
	}
	if got.Current == nil || *got.Current != 10.5 || got.Limit == nil || *got.Limit != 20 {
		t.Fatalf("primary scalar = %#v/%#v, want first non-trial row 10.5/20", got.Current, got.Limit)
	}
	if got.NextResetAt == nil || got.NextResetAt.Format(time.RFC3339) != "2026-06-01T00:00:00Z" {
		t.Fatalf("NextResetAt = %#v, want unix-ms parsed 2026-06-01", got.NextResetAt)
	}
	if got.Quotas[1].ResourceType != "chat_request" || got.Quotas[1].RemainingPercent == nil || *got.Quotas[1].RemainingPercent != 90 {
		t.Fatalf("second quota row = %#v, want chat_request 90%% remaining", got.Quotas[1])
	}
	if !got.Quotas[2].FreeTrial || got.Quotas[2].ResetAt == nil || got.Quotas[2].ResetAt.Format(time.RFC3339) != "2026-06-30T00:00:00Z" {
		t.Fatalf("trial row = %#v, want free trial expiry", got.Quotas[2])
	}
}

func TestParseKiroUsageLimits_EmptyQuotaUnavailable(t *testing.T) {
	got, err := ParseKiroUsageLimits([]byte(`{"usageBreakdownList":[]}`), time.Time{})
	if err != nil {
		t.Fatalf("ParseKiroUsageLimits() error = %v", err)
	}
	if got.ProviderQuotaAvailable {
		t.Fatal("ProviderQuotaAvailable = true, want false")
	}
	if got.Current != nil || got.Limit != nil || got.Percent != nil || got.Remaining != nil {
		t.Fatalf("quota invented values: %#v", got)
	}
}

func TestKiroExecutorFetchQuota_UsesEndpointFallbackOrder(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch len(calls) {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/codewhisperer/getUsageLimits" {
				t.Fatalf("first request = %s %s, want codewhisperer GET", r.Method, r.URL.Path)
			}
			http.Error(w, "not here", http.StatusNotFound)
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/codewhisperer" {
				t.Fatalf("second request = %s %s, want codewhisperer POST", r.Method, r.URL.Path)
			}
			if got := r.Header.Get("x-amz-target"); got != kiroCodeWhispererGetUsageTarget {
				t.Fatalf("x-amz-target = %q, want %q", got, kiroCodeWhispererGetUsageTarget)
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		case 3:
			if r.Method != http.MethodGet || r.URL.Path != "/q/getUsageLimits" {
				t.Fatalf("third request = %s %s, want q GET", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("profileArn"); got == "" {
				t.Fatal("q fallback missing profileArn query")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"usageBreakdownList":[{"resourceType":"AGENTIC_REQUEST","currentUsageWithPrecision":4,"usageLimitWithPrecision":8}]}`))
		default:
			t.Fatalf("unexpected extra request %d: %s %s", len(calls), r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{
		ID:       "kiro-auth",
		Provider: "kiro",
		Metadata: map[string]any{
			"access_token":                 "access-token",
			"profile_arn":                  "arn:aws:codewhisperer:us-west-2:123:profile/test",
			"codewhisperer_usage_url":      server.URL + "/codewhisperer/getUsageLimits",
			"codewhisperer_usage_base_url": server.URL + "/codewhisperer",
			"q_usage_url":                  server.URL + "/q/getUsageLimits",
		},
	}

	got, updated, err := NewKiroExecutor(&config.Config{}).FetchQuota(context.Background(), auth)
	if err != nil {
		t.Fatalf("FetchQuota() error = %v", err)
	}
	if updated != nil {
		t.Fatalf("updated auth = %#v, want nil without refresh", updated)
	}
	if got.Current == nil || *got.Current != 4 || got.Limit == nil || *got.Limit != 8 {
		t.Fatalf("quota = %#v, want q fallback 4/8", got)
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %#v, want 3 attempts", calls)
	}
}

func TestKiroExecutorFetchQuota_RefreshesOnceOn401(t *testing.T) {
	var usageCalls int32
	var refreshCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case kiroUsageLimitsPath:
			call := atomic.AddInt32(&usageCalls, 1)
			if call == 1 {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer access-new" {
				t.Fatalf("Authorization = %q, want refreshed token", got)
			}
			if got := r.URL.Query().Get("origin"); got != "AI_EDITOR" {
				t.Fatalf("origin query = %q, want AI_EDITOR", got)
			}
			if got := r.URL.Query().Get("resourceType"); got != "AGENTIC_REQUEST" {
				t.Fatalf("resourceType query = %q, want AGENTIC_REQUEST", got)
			}
			if got := r.URL.Query().Get("isEmailRequired"); got != "true" {
				t.Fatalf("isEmailRequired query = %q, want true", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"usageBreakdownList":[{"currentUsage":1,"usageLimit":5}]}`))
		case "/refresh":
			atomic.AddInt32(&refreshCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessToken":"access-new","refreshToken":"refresh-new","profileArn":"arn:aws:iam:us-west-2:123:profile/test","expiresIn":3600}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{
		ID:       "kiro-auth",
		Provider: "kiro",
		Metadata: map[string]any{
			"access_token":  "access-old",
			"refresh_token": "refresh-old",
			"quota_url":     strings.TrimRight(server.URL, "/") + kiroUsageLimitsPath,
			"refresh_url":   strings.TrimRight(server.URL, "/") + "/refresh",
		},
	}

	got, updated, err := NewKiroExecutor(&config.Config{}).FetchQuota(context.Background(), auth)
	if err != nil {
		t.Fatalf("FetchQuota() error = %v", err)
	}
	if updated == nil {
		t.Fatal("updated auth = nil, want refreshed auth")
	}
	if updated.Metadata["access_token"] != "access-new" {
		t.Fatalf("updated access token = %#v, want access-new", updated.Metadata["access_token"])
	}
	if got.Current == nil || *got.Current != 1 || got.Limit == nil || *got.Limit != 5 {
		t.Fatalf("quota = %#v, want 1/5", got)
	}
	if atomic.LoadInt32(&usageCalls) != 2 {
		t.Fatalf("usage calls = %d, want 2", usageCalls)
	}
	if atomic.LoadInt32(&refreshCalls) != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshCalls)
	}
}
