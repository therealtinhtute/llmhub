package executor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

// Ported from upstream claude_executor_ratelimit_test.go +
// claude_executor_fable_ratelimit_test.go (end state at v7.3.9 covering
// 44eaef0009f8 model-level cooling + scope-overage rate limits and
// 1cce9325738f skip-Retry-After-on-overage-only). Adapted to local symbols:
//   - config.Claude.ModelLevelCooling -> config.Config.ClaudeModelLevelCooling
//   - executor.countTokensUpstream    -> executor.CountTokens
//   - upstream-attempt tracker assertions dropped (API absent locally)

type retryAfterProvider interface {
	RetryAfter() *time.Duration
}

func TestClaudeExecutor_HonorsAnthropicRateLimitHeaders_Execute(t *testing.T) {
	now := time.Now()
	sevenDayReset := now.Add(7 * 24 * time.Hour).Unix()
	fiveHourReset := now.Add(5 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "allowed")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Reset", strconv.FormatInt(fiveHourReset, 10))
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Reset", strconv.FormatInt(sevenDayReset, 10))
		w.Header().Set("Anthropic-Ratelimit-Unified-Representative-Claim", "seven_day")
		w.Header().Set("Anthropic-Ratelimit-Unified-Reset", strconv.FormatInt(sevenDayReset, 10))
		w.Header().Set("Retry-After", strconv.FormatInt(7*24*3600, 10))
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Number of requests has exceeded your 7-day rate limit."}}`))
	}))
	defer server.Close()

	executor := NewClaudeExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "claude-auth-1",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	}

	payload := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected error from Execute, got nil")
	}

	var rap retryAfterProvider
	if !errors.As(err, &rap) || rap == nil {
		t.Fatalf("expected error %T to implement RetryAfter() *time.Duration", err)
	}

	retryAfter := rap.RetryAfter()
	if retryAfter == nil {
		t.Fatalf("expected non-nil RetryAfter, got nil")
	}

	// Should be at least 7 days (reported reset) and at most 7 days + 35s (fuzz upper bound).
	minExpected := 7*24*time.Hour - 5*time.Second
	maxExpected := 7*24*time.Hour + 35*time.Second
	if *retryAfter < minExpected || *retryAfter > maxExpected {
		t.Fatalf("RetryAfter = %v, want between %v and %v", *retryAfter, minExpected, maxExpected)
	}

	// Verify one-time fuzz stability: repeat calls return exact same value
	if second := rap.RetryAfter(); second == nil || *second != *retryAfter {
		t.Fatalf("RetryAfter changed across calls: %v vs %v", *second, *retryAfter)
	}
}

func TestClaudeExecutor_HonorsAnthropicRateLimitHeaders_ExecuteStream(t *testing.T) {
	now := time.Now()
	fiveHourReset := now.Add(5 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Reset", strconv.FormatInt(fiveHourReset, 10))
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Reset", strconv.FormatInt(now.Add(7*24*time.Hour).Unix(), 10))
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"5-hour limit exceeded."}}`))
	}))
	defer server.Close()

	executor := NewClaudeExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "claude-auth-1",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	}

	payload := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected error from ExecuteStream, got nil")
	}

	var rap retryAfterProvider
	if !errors.As(err, &rap) || rap == nil {
		t.Fatalf("expected error %T to implement RetryAfter() *time.Duration", err)
	}

	retryAfter := rap.RetryAfter()
	if retryAfter == nil {
		t.Fatalf("expected non-nil RetryAfter, got nil")
	}

	minExpected := 5*time.Hour - 5*time.Second
	maxExpected := 5*time.Hour + 35*time.Second
	if *retryAfter < minExpected || *retryAfter > maxExpected {
		t.Fatalf("RetryAfter = %v, want between %v and %v (5h window)", *retryAfter, minExpected, maxExpected)
	}
}

func TestClaudeExecutor_RateLimit_BothRejectedUsesLongest(t *testing.T) {
	now := time.Now()
	fiveHourReset := now.Add(5 * time.Hour).Unix()
	sevenDayReset := now.Add(7 * 24 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Reset", strconv.FormatInt(fiveHourReset, 10))
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Reset", strconv.FormatInt(sevenDayReset, 10))
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Both limits exceeded."}}`))
	}))
	defer server.Close()

	executor := NewClaudeExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "claude-auth-1",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	}

	payload := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rap retryAfterProvider
	if !errors.As(err, &rap) || rap == nil {
		t.Fatalf("expected error %T to implement RetryAfter() *time.Duration", err)
	}

	retryAfter := rap.RetryAfter()
	if retryAfter == nil {
		t.Fatalf("expected non-nil RetryAfter, got nil")
	}

	minExpected := 7*24*time.Hour - 5*time.Second
	maxExpected := 7*24*time.Hour + 35*time.Second
	if *retryAfter < minExpected || *retryAfter > maxExpected {
		t.Fatalf("RetryAfter = %v, want between %v and %v", *retryAfter, minExpected, maxExpected)
	}
}

func TestClaudeExecutor_RateLimit_CountTokensHonorsRateLimitReset(t *testing.T) {
	now := time.Now()
	sevenDayReset := now.Add(7 * 24 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Reset", strconv.FormatInt(sevenDayReset, 10))
		w.Header().Set("Retry-After", "604800")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"7-day rate limit exceeded."}}`))
	}))
	defer server.Close()

	executor := NewClaudeExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "claude-auth-1",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	}

	payload := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := executor.CountTokens(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected error from CountTokens, got nil")
	}

	var rap retryAfterProvider
	if !errors.As(err, &rap) || rap == nil || rap.RetryAfter() == nil {
		t.Fatalf("expected CountTokens rate limit error to implement RetryAfter, got %v", err)
	}

	type credentialScopedProvider interface {
		IsCredentialScoped() bool
	}
	var csp credentialScopedProvider
	if !errors.As(err, &csp) || csp == nil || !csp.IsCredentialScoped() {
		t.Fatalf("expected CountTokens rate limit error to be credential-scoped, got %v", err)
	}

	minExpected := 7*24*time.Hour - 5*time.Second
	maxExpected := 7*24*time.Hour + 35*time.Second
	if *rap.RetryAfter() < minExpected || *rap.RetryAfter() > maxExpected {
		t.Fatalf("RetryAfter = %v, want between %v and %v", *rap.RetryAfter(), minExpected, maxExpected)
	}
}

func TestClaudeExecutor_RateLimit_FastModeAuthoritativeRejectionHeadersOverrideBody(t *testing.T) {
	var attemptsCred1 atomic.Int32

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptsCred1.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Reset", strconv.FormatInt(time.Now().Add(7*24*time.Hour).Unix(), 10))
		w.Header().Set("Retry-After", "604800")
		w.WriteHeader(http.StatusTooManyRequests)
		// Body text mentioning fast request rejected, but headers explicitly reject unified quota
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fast request rejected"}}`))
	}))
	defer server1.Close()

	cfg := &config.Config{DisableCooling: false}
	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 2)

	executor := NewClaudeExecutor(cfg)
	manager.RegisterExecutor(executor)

	baseID := uuid.NewString()
	auth1 := &cliproxyauth.Auth{ID: baseID + "-fast-override-1", Provider: "claude", Attributes: map[string]string{"api_key": "k1", "base_url": server1.URL}}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth1.ID, "claude", []*registry.ModelInfo{{ID: "claude-3-5-sonnet-20241022"}})
	t.Cleanup(func() {
		reg.UnregisterClient(auth1.ID)
	})

	if _, err := manager.Register(context.Background(), auth1); err != nil {
		t.Fatalf("register auth1: %v", err)
	}

	payload := []byte(`{"speed":"fast","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected rate limit error from cred1, got nil")
	}
	if attemptsCred1.Load() != 1 {
		t.Fatalf("attempts on cred1 = %d, want 1", attemptsCred1.Load())
	}

	// The authoritative unified-window headers must win over the fast-mode body
	// text: the error is credential-scoped (not a request-scoped entitlement
	// refusal) and cred1 is cooled down at credential level. NOTE: upstream also
	// fails over to a second credential inside the same round; the local
	// conductor returns credential-scoped errors immediately instead
	// (conductor ownership belongs to the management slice), so failover is
	// asserted here only via the cooling outcome.
	registeredAuth, ok := manager.GetByID(auth1.ID)
	if !ok || registeredAuth == nil {
		t.Fatal("auth1 not found")
	}
	if !registeredAuth.Unavailable || !registeredAuth.Quota.Exceeded {
		t.Fatalf("cred1 was not cooled down: unavailable=%v quota=%+v", registeredAuth.Unavailable, registeredAuth.Quota)
	}
}

func TestClaudeExecutor_RateLimit_FastEntitlementWithRetryAfterRemainsRequestScoped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Usage credits are required for fast mode."}}`))
	}))
	defer server.Close()

	cfg := &config.Config{DisableCooling: false}
	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 2)

	executor := NewClaudeExecutor(cfg)
	manager.RegisterExecutor(executor)

	baseID := uuid.NewString()
	auth := &cliproxyauth.Auth{ID: baseID + "-fast-entitlement", Provider: "claude", Attributes: map[string]string{"api_key": "k1", "base_url": server.URL}}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-3-5-sonnet-20241022"}})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})

	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	payload := []byte(`{"speed":"fast","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, err := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet-20241022",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	registeredAuth, ok := manager.GetByID(auth.ID)
	if !ok || registeredAuth == nil {
		t.Fatal("auth not found")
	}
	if registeredAuth.Unavailable || registeredAuth.Quota.Exceeded {
		t.Fatalf("fast entitlement refusal incorrectly cooled down the credential: unavailable=%v quota=%+v", registeredAuth.Unavailable, registeredAuth.Quota)
	}
}

// --- 44eaef0009f8: model-level cooling + scope-overage rate limits ---

func TestClassifyClaudeUpstreamError_FableOnlyRejectionIsModelScoped(t *testing.T) {
	// Given
	headers := http.Header{
		"Anthropic-Ratelimit-Unified-Status":       []string{"rejected"},
		"Anthropic-Ratelimit-Unified-5h-Status":    []string{"allowed"},
		"Anthropic-Ratelimit-Unified-7d-Status":    []string{"allowed"},
		"Anthropic-Ratelimit-Unified-7d_oi-Status": []string{"rejected"},
		"Retry-After": []string{"120"},
	}

	// When
	err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))

	// Then
	var scoped interface{ IsCredentialScoped() bool }
	if !errors.As(err, &scoped) || scoped == nil {
		t.Fatalf("expected %T to expose credential scope", err)
	}
	if scoped.IsCredentialScoped() {
		t.Fatal("Fable-only 7d_oi rejection was credential-scoped; want model-scoped")
	}
}

func TestClassifyClaudeUpstreamError_FableOnlyAllowedWarningIsModelScoped(t *testing.T) {
	// Given: 7d window is in allowed_warning, 7d_oi is rejected
	headers := http.Header{
		"Anthropic-Ratelimit-Unified-Status":       []string{"rejected"},
		"Anthropic-Ratelimit-Unified-5h-Status":    []string{"allowed"},
		"Anthropic-Ratelimit-Unified-7d-Status":    []string{"allowed_warning"},
		"Anthropic-Ratelimit-Unified-7d_oi-Status": []string{"rejected"},
		"Retry-After": []string{"120"},
	}

	// When
	err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))

	// Then
	var scoped interface{ IsCredentialScoped() bool }
	if !errors.As(err, &scoped) || scoped == nil {
		t.Fatalf("expected %T to expose credential scope", err)
	}
	if scoped.IsCredentialScoped() {
		t.Fatal("Fable-only 7d_oi rejection with allowed_warning was credential-scoped; want model-scoped")
	}
}

func TestClassifyClaudeUpstreamError_OverageRejectionIsModelScoped_Issue5915(t *testing.T) {
	// Given: exact headers from Issue #5915 (7d allowed, 5h status omitted, overage rejected)
	headers := http.Header{
		"Anthropic-Ratelimit-Unified-Status":                  []string{"rejected"},
		"Anthropic-Ratelimit-Unified-Representative-Claim":    []string{"seven_day_overage_included"},
		"Anthropic-Ratelimit-Unified-7d-Status":               []string{"allowed"},
		"Anthropic-Ratelimit-Unified-7d-Utilization":          []string{"0.69"},
		"Anthropic-Ratelimit-Unified-5h-Utilization":          []string{"0.00"},
		"Anthropic-Ratelimit-Unified-7d_oi-Status":            []string{"rejected"},
		"Anthropic-Ratelimit-Unified-7d_oi-Utilization":       []string{"1.02"},
		"Anthropic-Ratelimit-Unified-Overage-Status":          []string{"rejected"},
		"Anthropic-Ratelimit-Unified-Overage-Disabled-Reason": []string{"org_spend_cap_reached"},
		"Retry-After": []string{"121180"},
	}

	// When
	err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))

	// Then
	var scoped interface{ IsCredentialScoped() bool }
	if !errors.As(err, &scoped) || scoped == nil {
		t.Fatalf("expected %T to expose credential scope", err)
	}
	if scoped.IsCredentialScoped() {
		t.Fatal("overage rejection with healthy subscription allowance was credential-scoped; want model-scoped (Issue #5915)")
	}
}

func TestClassifyClaudeUpstreamError_ModelLevelCoolingForcesModelScope(t *testing.T) {
	headers := http.Header{
		"Anthropic-Ratelimit-Unified-Status":    []string{"rejected"},
		"Anthropic-Ratelimit-Unified-5h-Status": []string{"rejected"},
		"Retry-After":                           []string{"60"},
	}

	// Default cooling treats explicit 5h rejection as credential-scoped
	errDefault := classifyClaudeUpstreamErrorWithCooling(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error"}}`), false)
	var scopedDefault interface{ IsCredentialScoped() bool }
	if !errors.As(errDefault, &scopedDefault) || !scopedDefault.IsCredentialScoped() {
		t.Fatal("expected default cooling to treat 5h rejection as credential-scoped")
	}

	// With modelLevelCooling=true, it is model-scoped
	errModelLevel := classifyClaudeUpstreamErrorWithCooling(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error"}}`), true)
	var scopedModelLevel interface{ IsCredentialScoped() bool }
	if !errors.As(errModelLevel, &scopedModelLevel) || scopedModelLevel.IsCredentialScoped() {
		t.Fatal("expected modelLevelCooling=true to treat 5h rejection as model-scoped")
	}
}

func TestClassifyClaudeUpstreamError_SharedOrAmbiguousRejectionRemainsCredentialScoped(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
	}{
		{
			name: "explicit 5h rejection",
			headers: http.Header{
				"Anthropic-Ratelimit-Unified-5h-Status": []string{"rejected"},
				"Anthropic-Ratelimit-Unified-7d-Status": []string{"allowed"},
			},
		},
		{
			name: "explicit shared 7d rejection",
			headers: http.Header{
				"Anthropic-Ratelimit-Unified-5h-Status": []string{"allowed"},
				"Anthropic-Ratelimit-Unified-7d-Status": []string{"rejected"},
			},
		},
		{
			name: "aggregate rejection with shared statuses missing",
			headers: http.Header{
				"Anthropic-Ratelimit-Unified-Status": []string{"rejected"},
			},
		},
		{
			name: "aggregate rejection with shared statuses malformed",
			headers: http.Header{
				"Anthropic-Ratelimit-Unified-Status":    []string{"rejected"},
				"Anthropic-Ratelimit-Unified-5h-Status": []string{"unknown"},
				"Anthropic-Ratelimit-Unified-7d-Status": []string{"invalid"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			err := classifyClaudeUpstreamError(http.StatusTooManyRequests, tt.headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Shared usage window rejected."}}`))

			// Then
			var scoped interface{ IsCredentialScoped() bool }
			if !errors.As(err, &scoped) || scoped == nil {
				t.Fatalf("expected %T to expose credential scope", err)
			}
			if !scoped.IsCredentialScoped() {
				t.Fatal("shared or ambiguous rejection was model-scoped; want credential-scoped")
			}
		})
	}
}

func TestClassifyClaudeUpstreamError_FableRetryDuration(t *testing.T) {
	t.Run("retry-after header is skipped for overage rejection to avoid global cooldown", func(t *testing.T) {
		headers := http.Header{
			"Anthropic-Ratelimit-Unified-Status":       []string{"rejected"},
			"Anthropic-Ratelimit-Unified-5h-Status":    []string{"allowed"},
			"Anthropic-Ratelimit-Unified-7d-Status":    []string{"allowed"},
			"Anthropic-Ratelimit-Unified-7d_oi-Status": []string{"rejected"},
			"Anthropic-Ratelimit-Unified-7d_oi-Reset":  []string{strconv.FormatInt(time.Now().Add(7*24*time.Hour).Unix(), 10)},
			"Retry-After": []string{"120"},
		}

		err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))

		var retry retryAfterProvider
		if errors.As(err, &retry) && retry != nil && retry.RetryAfter() != nil {
			t.Fatalf("expected overage Retry-After to yield nil RetryAfter for exponential backoff, got %v", *retry.RetryAfter())
		}
	})

	t.Run("7d_oi reset only does not set week-long retry duration", func(t *testing.T) {
		headers := http.Header{
			"Anthropic-Ratelimit-Unified-Status":       []string{"rejected"},
			"Anthropic-Ratelimit-Unified-5h-Status":    []string{"allowed"},
			"Anthropic-Ratelimit-Unified-7d-Status":    []string{"allowed"},
			"Anthropic-Ratelimit-Unified-7d_oi-Status": []string{"rejected"},
			"Anthropic-Ratelimit-Unified-7d_oi-Reset":  []string{strconv.FormatInt(time.Now().Add(7*24*time.Hour).Unix(), 10)},
		}

		err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))

		var retry retryAfterProvider
		if errors.As(err, &retry) && retry != nil && retry.RetryAfter() != nil {
			t.Fatalf("expected Fable 7d_oi-only reset to yield nil RetryAfter, got %v", *retry.RetryAfter())
		}
	})
}

func TestClaudeExecutor_AuthManager_FableOnlyRejectionDoesNotBlockOpus(t *testing.T) {
	var fableAttempts, opusAttempts atomic.Int32
	reset := time.Now().Add(7 * 24 * time.Hour).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, "failed to read sanitized test request", http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(string(body), `"model":"claude-fable-5"`):
			fableAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Reset", strconv.FormatInt(reset, 10))
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))
		case strings.Contains(string(body), `"model":"claude-opus-5"`):
			opusAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"msg-opus-ok","type":"message","model":"claude-opus-5","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
		default:
			http.Error(w, "unexpected sanitized test model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 0)
	manager.RegisterExecutor(NewClaudeExecutor(&config.Config{DisableCooling: false}))

	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-fable-model-scope",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-fable-5"}, {ID: "claude-opus-5"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	payloadFable := []byte(`{"model":"claude-fable-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errFable := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-fable-5",
		Payload: payloadFable,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errFable == nil {
		t.Fatal("expected Fable request to be rate limited")
	}
	if got := fableAttempts.Load(); got != 1 {
		t.Fatalf("Fable upstream attempts = %d, want 1", got)
	}

	// Verify that Fable model state cooldown is driven by exponential backoff and not 7 days.
	updatedAuth, ok := manager.GetByID(auth.ID)
	if !ok || updatedAuth == nil {
		t.Fatal("auth not found")
	}
	fableState := updatedAuth.ModelStates["claude-fable-5"]
	if fableState == nil {
		t.Fatal("fable model state not found")
	}
	if fableState.Quota.NextRecoverAt.After(time.Now().Add(5 * time.Minute)) {
		t.Fatalf("fable model state cooldown too long: NextRecoverAt = %v (want short backoff, not 7 days)", fableState.Quota.NextRecoverAt)
	}

	payloadOpus := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errOpus := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpus != nil {
		t.Fatalf("expected Opus to reach upstream on the same credential, got: %v", errOpus)
	}
	if got := opusAttempts.Load(); got != 1 {
		t.Fatalf("Opus upstream attempts = %d, want 1", got)
	}
}

func TestClaudeExecutor_AuthManager_OverageSpendCapReachedDoesNotBlockOpus_Issue5915(t *testing.T) {
	var fableAttempts, opusAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, "failed to read sanitized test request", http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(string(body), `"model":"claude-fable-5-1"`):
			fableAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Representative-Claim", "seven_day_overage_included")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Utilization", "0.69")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Utilization", "0.00")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Utilization", "1.02")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Disabled-Reason", "org_spend_cap_reached")
			w.Header().Set("Retry-After", "121180")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))
		case strings.Contains(string(body), `"model":"claude-opus-5"`):
			opusAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"msg-opus-ok","type":"message","model":"claude-opus-5","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
		default:
			http.Error(w, "unexpected sanitized test model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 0)
	manager.RegisterExecutor(NewClaudeExecutor(&config.Config{DisableCooling: false}))

	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-overage-model-scope",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-fable-5-1"}, {ID: "claude-opus-5"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	// 1. A healthy subscription model (Opus) succeeds first.
	payloadOpus := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errOpusFirst := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpusFirst != nil {
		t.Fatalf("initial Opus request failed: %v", errOpusFirst)
	}

	// 2. An overage-only model (Fable) fails with spend cap reached.
	payloadFable := []byte(`{"model":"claude-fable-5-1","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errFable := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-fable-5-1",
		Payload: payloadFable,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errFable == nil {
		t.Fatal("expected Fable request to be rate limited")
	}
	if got := fableAttempts.Load(); got != 1 {
		t.Fatalf("Fable upstream attempts = %d, want 1", got)
	}

	// 3. Verify that the auth credential itself remains available (not credential-scoped).
	updatedAuth, ok := manager.GetByID(auth.ID)
	if !ok || updatedAuth == nil {
		t.Fatal("auth not found")
	}
	if updatedAuth.Unavailable {
		t.Fatalf("auth was marked unavailable; want unavailable = false")
	}
	if updatedAuth.Quota.Reason == "credential_quota" {
		t.Fatalf("auth Quota.Reason = credential_quota; want non-credential_quota")
	}
	if !updatedAuth.NextRetryAfter.IsZero() {
		t.Fatalf("auth NextRetryAfter = %v; want zero time", updatedAuth.NextRetryAfter)
	}

	// 4. Verify that Fable model state cooldown is set.
	fableState := updatedAuth.ModelStates["claude-fable-5-1"]
	if fableState == nil {
		t.Fatal("fable model state not found")
	}
	if !fableState.Unavailable {
		t.Fatal("fable model state was not marked unavailable")
	}
	if fableState.NextRetryAfter.IsZero() {
		t.Fatal("fable model state NextRetryAfter should be set")
	}

	// 5. Verify sibling Opus model state remains healthy.
	opusState := updatedAuth.ModelStates["claude-opus-5"]
	if opusState == nil {
		t.Fatal("opus model state not found")
	}
	if opusState.Unavailable {
		t.Fatalf("sibling opus model state was marked unavailable")
	}

	// 6. Verify Opus on the same credential can still be served cleanly.
	_, errOpus := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpus != nil {
		t.Fatalf("expected Opus to reach upstream on the same credential, got: %v", errOpus)
	}
	if got := opusAttempts.Load(); got != 2 {
		t.Fatalf("Opus upstream attempts = %d, want 2", got)
	}
}

func TestClaudeExecutor_AuthManager_ModelLevelCoolingConfigScopesSharedRejection(t *testing.T) {
	var sonnetAttempts, opusAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, "failed to read test request", http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(string(body), `"model":"claude-3-5-sonnet"`):
			sonnetAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "rejected")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"5h rate limit exceeded."}}`))
		case strings.Contains(string(body), `"model":"claude-opus-5"`):
			opusAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"msg-opus-ok","type":"message","model":"claude-opus-5","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
		default:
			http.Error(w, "unexpected test model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 0)
	// Enable model-level cooling via the local flat config field
	// (upstream cfg.Claude.ModelLevelCooling -> local cfg.ClaudeModelLevelCooling).
	manager.RegisterExecutor(NewClaudeExecutor(&config.Config{
		DisableCooling:          false,
		ClaudeModelLevelCooling: true,
	}))

	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-claude-model-cooling-cfg",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-3-5-sonnet"}, {ID: "claude-opus-5"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	payloadSonnet := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errSonnet := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-3-5-sonnet",
		Payload: payloadSonnet,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errSonnet == nil {
		t.Fatal("expected Sonnet request to be rate limited")
	}
	if got := sonnetAttempts.Load(); got != 1 {
		t.Fatalf("Sonnet upstream attempts = %d, want 1", got)
	}

	// Verify auth credential is not credential-scoped because model-level cooling is on.
	updatedAuth, ok := manager.GetByID(auth.ID)
	if !ok || updatedAuth == nil {
		t.Fatal("auth not found")
	}
	if updatedAuth.Quota.Reason == "credential_quota" {
		t.Fatal("auth Quota.Reason = credential_quota; want non-credential_quota when model-level cooling is enabled")
	}

	// Sibling Opus request still succeeds on the same credential
	payloadOpus := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errOpus := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpus != nil {
		t.Fatalf("expected Opus to succeed when model-level cooling is enabled, got: %v", errOpus)
	}
	if got := opusAttempts.Load(); got != 1 {
		t.Fatalf("Opus upstream attempts = %d, want 1", got)
	}
}

func TestClaudeExecutor_ExecuteStream_OverageRejectionIsModelScoped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-Representative-Claim", "seven_day_overage_included")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d-Utilization", "0.69")
		w.Header().Set("Anthropic-Ratelimit-Unified-5h-Utilization", "0.00")
		w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Status", "rejected")
		w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Disabled-Reason", "org_spend_cap_reached")
		w.Header().Set("Retry-After", "121180")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Fable usage window rejected."}}`))
	}))
	defer server.Close()

	executor := NewClaudeExecutor(&config.Config{DisableCooling: false})
	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-stream-overage",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}

	payload := []byte(`{"model":"claude-fable-5-1","stream":true,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	_, errStream := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "claude-fable-5-1",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errStream == nil {
		t.Fatal("expected stream request to fail with rate limit error")
	}

	var scoped interface{ IsCredentialScoped() bool }
	if !errors.As(errStream, &scoped) || scoped == nil {
		t.Fatalf("expected stream error to expose credential scope, got %v", errStream)
	}
	if scoped.IsCredentialScoped() {
		t.Fatal("stream overage rejection was credential-scoped; want model-scoped")
	}
}

// --- 1cce9325738f: skip Retry-After on overage-only rejections ---

func TestClassifyClaudeUpstreamError_OverageRejectionWithRetryAfter_Issue5920(t *testing.T) {
	// Given: exact headers from Issue #5920 (7d allowed, 5h allowed, 7d_oi rejected, Retry-After present)
	headers := http.Header{
		"Anthropic-Ratelimit-Unified-Status":                  []string{"rejected"},
		"Anthropic-Ratelimit-Unified-Representative-Claim":    []string{"seven_day_overage_included"},
		"Anthropic-Ratelimit-Unified-7d-Status":               []string{"allowed"},
		"Anthropic-Ratelimit-Unified-7d-Utilization":          []string{"0.73"},
		"Anthropic-Ratelimit-Unified-5h-Status":               []string{"allowed"},
		"Anthropic-Ratelimit-Unified-5h-Utilization":          []string{"0.23"},
		"Anthropic-Ratelimit-Unified-7d_oi-Status":            []string{"rejected"},
		"Anthropic-Ratelimit-Unified-7d_oi-Utilization":       []string{"1.02"},
		"Anthropic-Ratelimit-Unified-Overage-Status":          []string{"rejected"},
		"Anthropic-Ratelimit-Unified-Overage-Disabled-Reason": []string{"org_spend_cap_reached"},
		"Retry-After": []string{"97273"},
	}

	err := classifyClaudeUpstreamError(http.StatusTooManyRequests, headers, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your account's monthly spend limit. Please try again later."}}`))

	var scoped interface{ IsCredentialScoped() bool }
	if !errors.As(err, &scoped) || scoped == nil {
		t.Fatalf("expected error to expose credential scope, got %v", err)
	}
	if scoped.IsCredentialScoped() {
		t.Fatal("overage rejection with healthy subscription allowance was credential-scoped; want model-scoped (Issue #5920)")
	}

	var retry retryAfterProvider
	if errors.As(err, &retry) && retry != nil && retry.RetryAfter() != nil {
		t.Fatalf("expected overage Retry-After to be skipped so credential is not cooled, got %v", *retry.RetryAfter())
	}
}

func TestClaudeExecutor_AuthManager_OverageSpendCapWithRetryAfter_Issue5920(t *testing.T) {
	var fableAttempts, opusAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, "failed to read sanitized test request", http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(string(body), `"model":"claude-fable-5-1"`):
			fableAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Representative-Claim", "seven_day_overage_included")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Utilization", "0.73")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Utilization", "0.23")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Utilization", "1.02")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Disabled-Reason", "org_spend_cap_reached")
			w.Header().Set("Retry-After", "97273")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your account's monthly spend limit. Please try again later."}}`))
		case strings.Contains(string(body), `"model":"claude-opus-5"`):
			opusAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"msg-opus-ok","type":"message","model":"claude-opus-5","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
		default:
			http.Error(w, "unexpected sanitized test model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 0)
	manager.RegisterExecutor(NewClaudeExecutor(&config.Config{DisableCooling: false}))

	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-issue-5920-overage",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-fable-5-1"}, {ID: "claude-opus-5"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	// 1. Initial healthy subscription request succeeds first.
	payloadOpus := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errOpusFirst := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpusFirst != nil {
		t.Fatalf("initial Opus request failed: %v", errOpusFirst)
	}

	// 2. Fable request fails with 429 carrying Retry-After: 97273.
	payloadFable := []byte(`{"model":"claude-fable-5-1","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errFable := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-fable-5-1",
		Payload: payloadFable,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errFable == nil {
		t.Fatal("expected Fable request to be rate limited")
	}
	if got := fableAttempts.Load(); got != 1 {
		t.Fatalf("Fable upstream attempts = %d, want 1", got)
	}

	// 3. Verify that the auth credential itself remains available (not benched globally for 27h).
	updatedAuth, ok := manager.GetByID(auth.ID)
	if !ok || updatedAuth == nil {
		t.Fatal("auth not found")
	}
	if updatedAuth.Unavailable {
		t.Fatalf("auth was marked unavailable; want unavailable = false")
	}
	if !updatedAuth.NextRetryAfter.IsZero() {
		t.Fatalf("auth NextRetryAfter = %v; want zero time", updatedAuth.NextRetryAfter)
	}

	// 4. Verify that Fable model state cooldown is NOT 27 hours (Retry-After was skipped).
	fableState := updatedAuth.ModelStates["claude-fable-5-1"]
	if fableState == nil {
		t.Fatal("fable model state not found")
	}
	if fableState.Quota.NextRecoverAt.IsZero() || !fableState.Quota.NextRecoverAt.After(time.Now()) {
		t.Fatal("fable model cooldown should be set in the future via exponential backoff")
	}
	if fableState.Quota.NextRecoverAt.After(time.Now().Add(10 * time.Minute)) {
		t.Fatalf("fable model cooldown was %v; want exponential backoff (<10m), not 27h spend cap", fableState.Quota.NextRecoverAt)
	}

	// 5. Verify sibling Opus request still succeeds on the same credential.
	_, errOpus := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpus != nil {
		t.Fatalf("expected Opus to succeed on the same credential, got: %v", errOpus)
	}
	if got := opusAttempts.Load(); got != 2 {
		t.Fatalf("Opus upstream attempts = %d, want 2", got)
	}
}

func TestClaudeExecutor_AuthManager_OverageSpendCapFirstRequestAllowsOpus_Issue5920(t *testing.T) {
	var fableAttempts, opusAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, "failed to read sanitized test request", http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(string(body), `"model":"claude-fable-5-1"`):
			fableAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Anthropic-Ratelimit-Unified-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Representative-Claim", "seven_day_overage_included")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d-Utilization", "0.73")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Status", "allowed")
			w.Header().Set("Anthropic-Ratelimit-Unified-5h-Utilization", "0.23")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-7d_oi-Utilization", "1.02")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Status", "rejected")
			w.Header().Set("Anthropic-Ratelimit-Unified-Overage-Disabled-Reason", "org_spend_cap_reached")
			w.Header().Set("Retry-After", "97273")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your account's monthly spend limit. Please try again later."}}`))
		case strings.Contains(string(body), `"model":"claude-opus-5"`):
			opusAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"msg-opus-ok","type":"message","model":"claude-opus-5","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
		default:
			http.Error(w, "unexpected sanitized test model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetRetryConfig(0, 0, 0)
	manager.RegisterExecutor(NewClaudeExecutor(&config.Config{DisableCooling: false}))

	auth := &cliproxyauth.Auth{
		ID:       uuid.NewString() + "-issue-5920-first-fable",
		Provider: "claude",
		Attributes: map[string]string{
			"api_key":  "sanitized-test-key",
			"base_url": server.URL,
		},
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "claude", []*registry.ModelInfo{{ID: "claude-fable-5-1"}, {ID: "claude-opus-5"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	// 1. Initial request on the fresh credential is for the overage-only model (Fable).
	// No prior Opus request has created any active model state.
	payloadFable := []byte(`{"model":"claude-fable-5-1","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errFable := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-fable-5-1",
		Payload: payloadFable,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errFable == nil {
		t.Fatal("expected Fable request to be rate limited")
	}
	if got := fableAttempts.Load(); got != 1 {
		t.Fatalf("Fable upstream attempts = %d, want 1", got)
	}

	// 2. Fable model state must not have a 27h cooldown.
	updatedAuth, ok := manager.GetByID(auth.ID)
	if !ok || updatedAuth == nil {
		t.Fatal("auth not found")
	}
	fableState := updatedAuth.ModelStates["claude-fable-5-1"]
	if fableState == nil {
		t.Fatal("fable model state not found")
	}
	if fableState.Quota.NextRecoverAt.IsZero() || !fableState.Quota.NextRecoverAt.After(time.Now()) {
		t.Fatal("fable model cooldown should be set in the future via exponential backoff")
	}
	if fableState.Quota.NextRecoverAt.After(time.Now().Add(10 * time.Minute)) {
		t.Fatalf("fable model cooldown was %v; want exponential backoff (<10m), not 27h spend cap", fableState.Quota.NextRecoverAt)
	}

	// 3. Subsequent Opus request on the same credential can still be served cleanly.
	payloadOpus := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"test"}]}]}`)
	_, errOpus := manager.Execute(context.Background(), []string{"claude"}, cliproxyexecutor.Request{
		Model:   "claude-opus-5",
		Payload: payloadOpus,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude})
	if errOpus != nil {
		t.Fatalf("expected Opus to succeed on the same credential, got: %v", errOpus)
	}
	if got := opusAttempts.Load(); got != 1 {
		t.Fatalf("Opus upstream attempts = %d, want 1", got)
	}
}
