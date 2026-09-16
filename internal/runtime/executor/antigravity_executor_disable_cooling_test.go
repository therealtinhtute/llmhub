package executor

// Ported from upstream CLIProxyAPI commit 272c1cff4e4c
// (fix(antigravity): bypass quota cooldowns and credit hints when cooling is disabled).
// Local symbols under test: antigravityCoolingDisabled, antigravityIsInShortCooldown,
// markAntigravityShortCooldown, markAntigravityCreditsPermanentlyDisabled,
// AntigravityExecutor.maybeRefreshAntigravityCreditsHint.
// NOTE: upstream asserts against a Home KV client; locally short cooldowns live in
// the antigravityShortCooldownByAuth sync.Map, so assertions target that map.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

func TestAntigravityDisableCooling_ShortCooldownBypassedInHomeMode(t *testing.T) {
	resetAntigravityCreditsRetryState()
	t.Cleanup(resetAntigravityCreditsRetryState)

	cliproxyauth.SetQuotaCooldownDisabled(true)
	t.Cleanup(func() { cliproxyauth.SetQuotaCooldownDisabled(false) })

	cfg := &config.Config{
		DisableCooling: true,
		Home: config.HomeConfig{
			Enabled: true,
		},
	}
	exec := NewAntigravityExecutor(cfg)
	auth := &cliproxyauth.Auth{
		ID: "home-cooling-disabled-auth",
		Metadata: map[string]any{
			"access_token": "token",
			"expired":      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			"project_id":   "test-project",
		},
	}

	modelName := "claude-sonnet-4-5"
	now := time.Now()
	duration := 30 * time.Second

	// 1. With DisableCooling, marking short cooldown should be a no-op.
	markAntigravityShortCooldown(auth, modelName, now, duration)
	if _, loaded := antigravityShortCooldownByAuth.Load(antigravityShortCooldownKey(auth, modelName)); loaded {
		t.Fatal("antigravityShortCooldownByAuth stored key, want no-op when DisableCooling is true")
	}

	// 2. Pre-populate the cooldown map manually; read should still return false when cooling disabled.
	antigravityShortCooldownByAuth.Store(antigravityShortCooldownKey(auth, modelName), now.Add(time.Hour))
	inCooldown, remaining := antigravityIsInShortCooldown(auth, modelName, now)
	if inCooldown || remaining > 0 {
		t.Fatalf("inCooldown = %v, remaining = %v, want false/0 when DisableCooling is true", inCooldown, remaining)
	}

	// 3. In Execute, upstream 429 RATE_LIMIT_EXCEEDED should not record short cooldown.
	resetAntigravityCreditsRetryState()
	upstreamResp := `{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED","metadata":{"quotaResetDelay":"60s"}}]}}`
	var calls int
	execCtx := context.WithValue(context.Background(), "cliproxy.roundtripper", roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		h := make(http.Header)
		h.Set("Content-Type", "application/json")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     h,
			Body:       io.NopCloser(strings.NewReader(upstreamResp)),
		}, nil
	}))

	req := cliproxyexecutor.Request{
		Model:   modelName,
		Payload: []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	}
	opts := cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatClaude,
	}
	_, _ = exec.Execute(execCtx, auth, req, opts)
	if calls == 0 {
		t.Fatal("Execute did not reach upstream, want request to bypass pre-seeded cooldown when cooling disabled")
	}
	if _, loaded := antigravityShortCooldownByAuth.Load(antigravityShortCooldownKey(auth, modelName)); loaded {
		t.Fatal("Execute recorded short cooldown, want skipped when cooling disabled")
	}
}

func TestAntigravityDisableCooling_ExecuteStreamBypassesShortCooldown(t *testing.T) {
	resetAntigravityCreditsRetryState()
	t.Cleanup(resetAntigravityCreditsRetryState)

	cfg := &config.Config{
		DisableCooling: true,
		Home: config.HomeConfig{
			Enabled: true,
		},
	}
	exec := NewAntigravityExecutor(cfg)
	auth := &cliproxyauth.Auth{
		ID: "home-cooling-disabled-auth-stream",
		Metadata: map[string]any{
			"access_token": "token",
			"expired":      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			"project_id":   "test-project",
		},
	}

	modelName := "claude-sonnet-4-5"
	upstreamResp := `{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED","metadata":{"quotaResetDelay":"60s"}}]}}`
	var calls int
	execCtx := context.WithValue(context.Background(), "cliproxy.roundtripper", roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		h := make(http.Header)
		h.Set("Content-Type", "application/json")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     h,
			Body:       io.NopCloser(strings.NewReader(upstreamResp)),
		}, nil
	}))

	req := cliproxyexecutor.Request{
		Model:   modelName,
		Payload: []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	}
	opts := cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatClaude,
	}
	_, _ = exec.ExecuteStream(execCtx, auth, req, opts)
	if calls == 0 {
		t.Fatal("ExecuteStream did not reach upstream")
	}
	if _, loaded := antigravityShortCooldownByAuth.Load(antigravityShortCooldownKey(auth, modelName)); loaded {
		t.Fatal("ExecuteStream recorded short cooldown, want skipped when cooling disabled")
	}
}

func TestAntigravityDisableCooling_AuthOverrideBypassesShortCooldown(t *testing.T) {
	resetAntigravityCreditsRetryState()
	t.Cleanup(resetAntigravityCreditsRetryState)

	cfg := &config.Config{
		DisableCooling: false,
	}
	exec := NewAntigravityExecutor(cfg)
	auth := &cliproxyauth.Auth{
		ID: "override-cooling-disabled-auth",
		Metadata: map[string]any{
			"disable_cooling": true,
			"access_token":    "token",
			"expired":         time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			"project_id":      "test-project",
		},
	}

	modelName := "claude-sonnet-4-5"
	now := time.Now()
	duration := 30 * time.Second

	// Marking short cooldown should be skipped for auth with disable_cooling override.
	markAntigravityShortCooldown(auth, modelName, now, duration)
	if _, loaded := antigravityShortCooldownByAuth.Load(antigravityShortCooldownKey(auth, modelName)); loaded {
		t.Fatal("antigravityShortCooldownByAuth stored key, want skipped for auth with disable_cooling override")
	}

	// Pre-populate in-memory map; read should return false.
	antigravityShortCooldownByAuth.Store(antigravityShortCooldownKey(auth, modelName), now.Add(time.Hour))
	inCooldown, remaining := antigravityIsInShortCooldown(auth, modelName, now)
	if inCooldown || remaining > 0 {
		t.Fatalf("inCooldown = %v, remaining = %v, want false/0 when auth has disable_cooling override", inCooldown, remaining)
	}

	// In Execute, upstream 429 RATE_LIMIT_EXCEEDED should not record short cooldown.
	resetAntigravityCreditsRetryState()
	upstreamResp := `{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED","metadata":{"quotaResetDelay":"60s"}}]}}`
	var calls int
	execCtx := context.WithValue(context.Background(), "cliproxy.roundtripper", roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		h := make(http.Header)
		h.Set("Content-Type", "application/json")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     h,
			Body:       io.NopCloser(strings.NewReader(upstreamResp)),
		}, nil
	}))

	req := cliproxyexecutor.Request{
		Model:   modelName,
		Payload: []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	}
	opts := cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatClaude,
	}
	_, _ = exec.Execute(execCtx, auth, req, opts)
	if calls == 0 {
		t.Fatal("Execute did not reach upstream, want request to bypass pre-seeded cooldown when auth has disable_cooling override")
	}
	if _, loaded := antigravityShortCooldownByAuth.Load(antigravityShortCooldownKey(auth, modelName)); loaded {
		t.Fatal("Execute recorded short cooldown, want skipped when auth has disable_cooling override")
	}
}

func TestAntigravityDisableCooling_CreditsHintRefreshBypassedInHomeMode(t *testing.T) {
	resetAntigravityCreditsRetryState()
	t.Cleanup(resetAntigravityCreditsRetryState)

	cfg := &config.Config{
		DisableCooling: true,
		Home: config.HomeConfig{
			Enabled: true,
		},
		QuotaExceeded: config.QuotaExceeded{
			AntigravityCredits: true,
		},
	}
	exec := NewAntigravityExecutor(cfg)
	auth := &cliproxyauth.Auth{
		ID: "home-refresh-cooling-disabled-auth",
		Metadata: map[string]any{
			"access_token": "token",
			"expired":      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			"project_id":   "test-project",
		},
	}

	exec.maybeRefreshAntigravityCreditsHint(context.Background(), auth, "token")
	if _, loaded := antigravityCreditsHintRefreshByID.Load(auth.ID); loaded {
		t.Fatal("credits hint refresh state stored, want skipped when DisableCooling is true")
	}
	if cliproxyauth.HasKnownAntigravityCreditsHint(auth.ID) {
		t.Fatal("credits hint was stored for auth when DisableCooling is true")
	}
}

func TestAntigravityDisableCooling_CreditsPermanentlyDisabledBypassed(t *testing.T) {
	resetAntigravityCreditsRetryState()
	t.Cleanup(resetAntigravityCreditsRetryState)

	auth := &cliproxyauth.Auth{
		ID: "home-permanently-disabled-auth",
		Metadata: map[string]any{
			"disable_cooling": true,
		},
	}

	markAntigravityCreditsPermanentlyDisabled(auth)
	if _, loaded := antigravityCreditsFailureByAuth.Load(auth.ID); loaded {
		t.Fatal("credits failure state stored, want skipped when auth has disable_cooling override")
	}
	if cliproxyauth.HasKnownAntigravityCreditsHint(auth.ID) {
		t.Fatal("credits hint was stored for auth with DisableCooling true")
	}
}
