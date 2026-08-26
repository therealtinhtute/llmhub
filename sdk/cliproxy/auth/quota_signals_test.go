package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	internallogging "github.com/therealtinhtute/llmhub/internal/logging"
)

func TestQuotaStateObserveResponseHeadersKeepsProviderScopedSignals(t *testing.T) {
	observedAt := time.Unix(123, 0)
	var quota QuotaState
	if !quota.ObserveResponseHeadersForProvider("codex", http.Header{
		"X-Codex-Primary-Used-Percent": []string{"2"},
		"X-Codex-Plan-Type": []string{"pro"},
	}, observedAt) {
		t.Fatal("expected observation to be recorded")
	}
	if quota.Signals["X-Codex-Primary-Used-Percent"] != "2" {
		t.Fatalf("unexpected signals %#v", quota.Signals)
	}
	if !quota.ObservedAt.Equal(observedAt) {
		t.Fatalf("ObservedAt = %v want %v", quota.ObservedAt, observedAt)
	}
	// non-codex provider should not observe
	var quota2 QuotaState
	if quota2.ObserveResponseHeadersForProvider("gemini", http.Header{"X-Codex-Primary-Used-Percent": []string{"2"}}, observedAt) {
		t.Fatal("gemini should not observe codex headers")
	}
}

func TestProviderSupportsQuotaObservation(t *testing.T) {
	if !ProviderSupportsQuotaObservation("codex") || !ProviderSupportsQuotaObservation("claude") {
		t.Fatal("codex/claude should support")
	}
	if ProviderSupportsQuotaObservation("gemini") || ProviderSupportsQuotaObservation("openai") {
		t.Fatal("gemini/openai should not support")
	}
}

func TestQuotaStateCloneCopiesSignals(t *testing.T) {
	orig := QuotaState{
		ObservedAt: time.Unix(10, 0),
		Signals: map[string]string{"X-Codex-Primary-Used-Percent": "51"},
	}
	clone := orig.Clone()
	if clone.Signals["X-Codex-Primary-Used-Percent"] != "51" {
		t.Fatal("clone failed")
	}
	clone.Signals["new"] = "1"
	if orig.Signals["new"] == "1" {
		t.Fatal("clone not deep")
	}
}

func TestApplyCooldownFieldsPreservesObservation(t *testing.T) {
	orig := QuotaState{
		ObservedAt: time.Unix(10, 0),
		Signals: map[string]string{"X-Codex-Primary-Used-Percent": "51"},
		Exceeded: false,
	}
	cooldown := QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: time.Unix(20, 0), BackoffLevel: 2}
	applyCooldownFields(&orig, cooldown)
	if !orig.Exceeded || orig.Reason != "quota" {
		t.Fatalf("cooldown fields not applied %#v", orig)
	}
	if orig.Signals["X-Codex-Primary-Used-Percent"] != "51" || !orig.ObservedAt.Equal(time.Unix(10, 0)) {
		t.Fatalf("observation was overwritten %#v", orig)
	}
}

func TestManagerMarkResultRecordsResponseQuotaSignalsInMemory(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	auth, errRegister := manager.Register(context.Background(), &Auth{
		ID:       "quota-signal-auth",
		Provider: "codex",
	})
	if errRegister != nil || auth == nil {
		t.Fatalf("Register() auth=%#v err=%v", auth, errRegister)
	}
	ctx := internallogging.WithResponseHeadersHolder(context.Background())
	internallogging.SetResponseHeaders(ctx, http.Header{
		"X-Codex-Active-Limit":           []string{"codex_bengalfox"},
		"X-Codex-Primary-Used-Percent":   []string{"2"},
		"X-Codex-Primary-Window-Minutes": []string{"10080"},
		"X-Codex-Primary-Reset-At":       []string{"1782951970"},
	})
	manager.MarkResult(ctx, Result{
		AuthID:   auth.ID,
		Provider: "codex",
		Model:    "gpt-5.3-codex",
		Success:  true,
	})
	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("auth not found after MarkResult")
	}
	if updated.Quota.Signals["X-Codex-Active-Limit"] != "codex_bengalfox" ||
		updated.Quota.Signals["X-Codex-Primary-Used-Percent"] != "2" {
		t.Fatalf("in-memory quota signals = %#v", updated.Quota.Signals)
	}
}

func TestMarkResultCountTokensDoesNotReplaceObservation(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	auth, errRegister := manager.Register(context.Background(), &Auth{
		ID:       "quota-count-tokens-auth",
		Provider: "claude",
		Quota: QuotaState{
			ObservedAt: time.Unix(10, 0),
			Signals:    map[string]string{"Anthropic-Ratelimit-Unified-Status": "allowed"},
		},
	})
	if errRegister != nil || auth == nil {
		t.Fatalf("Register() auth=%#v err=%v", auth, errRegister)
	}
	ctx := internallogging.WithResponseHeadersHolder(context.Background())
	internallogging.SetResponseHeaders(ctx, http.Header{
		"Anthropic-Ratelimit-Unified-Status": []string{"rejected"},
	})
	manager.MarkResult(ctx, Result{
		AuthID:               auth.ID,
		Provider:             "claude",
		Model:                "claude-opus-4-6",
		Success:              true,
		SkipQuotaObservation: true,
	})
	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("auth not found after MarkResult")
	}
	if updated.Quota.Signals["Anthropic-Ratelimit-Unified-Status"] != "allowed" || !updated.Quota.ObservedAt.Equal(time.Unix(10, 0)) {
		t.Fatalf("count_tokens replaced the last generation snapshot: %#v", updated.Quota)
	}
}
