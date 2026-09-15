package auth

import (
	"context"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
)

// Tests in this file cover the Cloudflare 520-526 transient-upstream handling
// ported from upstream CLIProxyAPI commit 9fad50550517 ("fix(auth): treat
// cloudflare 520-526 origin errors as transient upstream failures"). Local
// symbols under test: MarkResult, applyAuthFailureState,
// recoverableFailureRetryAfterWithHint, SetTransientErrorCooldownSeconds.
//
// Note: upstream also excluded HTTP >=500 responses from Cloudflare challenge
// classification. The local tree has no challenge-classification path
// (isCloudflareChallenge* helpers do not exist), so that exclusion is
// satisfied by absence and has nothing to port.

func withTransientCooldownSeconds(t *testing.T, seconds int) {
	t.Helper()
	previous := transientErrorCooldownSeconds.Load()
	SetTransientErrorCooldownSeconds(seconds)
	t.Cleanup(func() { transientErrorCooldownSeconds.Store(previous) })
}

// register520TestAuth registers an auth + model in the global registry so
// MarkResult can resolve the model state under test.
func register520TestAuth(t *testing.T, manager *Manager, provider, authID, model string) *Auth {
	t.Helper()
	auth := &Auth{ID: authID, Provider: provider}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(authID, provider, []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() { reg.UnregisterClient(authID) })
	if _, errRegister := manager.Register(WithSkipPersist(context.Background()), auth); errRegister != nil {
		t.Fatalf("Register returned error: %v", errRegister)
	}
	return auth
}

func TestMarkResult_520AppliesTransientCooldownToModelState(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 0)

	manager := NewManager(nil, nil, nil)
	model := "gpt-5-codex"
	auth := register520TestAuth(t, manager, "codex", "auth-520-model", model)

	manager.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: "codex",
		Model:    model,
		Success:  false,
		Error: &Error{
			HTTPStatus: 520,
			Message:    "web server returns an unknown error",
		},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok {
		t.Fatal("expected registered auth")
	}
	state := updated.ModelStates[model]
	if state == nil {
		t.Fatal("expected model state to be recorded")
	}
	low := time.Now().Add(transientErrorCooldown - 5*time.Second)
	if state.NextRetryAfter.Before(low) {
		t.Fatalf("model NextRetryAfter = %s, want >= %s (transient cooldown)", state.NextRetryAfter, low)
	}
	if !state.Unavailable {
		t.Fatal("expected model state to be marked unavailable during transient cooldown")
	}
}

func TestMarkResult_520HonorsRetryAfterHint(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 0)

	manager := NewManager(nil, nil, nil)
	model := "gpt-5-codex"
	auth := register520TestAuth(t, manager, "codex", "auth-520-hint", model)

	hint := 5 * time.Second
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth.ID,
		Provider:   "codex",
		Model:      model,
		Success:    false,
		RetryAfter: &hint,
		Error: &Error{
			HTTPStatus: 524,
			Message:    "origin time-out",
		},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok {
		t.Fatal("expected registered auth")
	}
	state := updated.ModelStates[model]
	if state == nil {
		t.Fatal("expected model state to be recorded")
	}
	low := time.Now().Add(hint - time.Second)
	high := time.Now().Add(hint + 30*time.Second)
	if state.NextRetryAfter.Before(low) || state.NextRetryAfter.After(high) {
		t.Fatalf("model NextRetryAfter = %s, want ~%s (provider RetryAfter hint)", state.NextRetryAfter, time.Now().Add(hint))
	}
}

func TestApplyAuthFailureState_520To526AreTransient(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 0)

	now := time.Now()
	for status := 520; status <= 526; status++ {
		auth := &Auth{ID: "auth-cf", Provider: "codex"}
		resultErr := &Error{HTTPStatus: status, Message: "cloudflare origin error"}
		applyAuthFailureState(auth, resultErr, nil, now, false)

		if !auth.Unavailable {
			t.Fatalf("status %d: expected auth to be marked unavailable", status)
		}
		if auth.StatusMessage != "transient upstream error" {
			t.Fatalf("status %d: StatusMessage = %q, want transient upstream error", status, auth.StatusMessage)
		}
		low := now.Add(transientErrorCooldown - time.Second)
		if auth.NextRetryAfter.Before(low) {
			t.Fatalf("status %d: NextRetryAfter = %s, want >= %s", status, auth.NextRetryAfter, low)
		}
	}
}

func TestApplyAuthFailureState_520RetryAfterHintBeatsDefaultCooldown(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 0)

	now := time.Now()
	hint := 3 * time.Second
	auth := &Auth{ID: "auth-cf-hint", Provider: "codex"}
	applyAuthFailureState(auth, &Error{HTTPStatus: 521, Message: "web server is down"}, &hint, now, false)

	if got := auth.NextRetryAfter.Sub(now); got != hint {
		t.Fatalf("NextRetryAfter delta = %s, want %s (provider hint)", got, hint)
	}
}

func TestApplyAuthFailureState_TransientCooldownSecondsOverride(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 90)

	now := time.Now()
	auth := &Auth{ID: "auth-cf-override", Provider: "codex"}
	applyAuthFailureState(auth, &Error{HTTPStatus: 502, Message: "bad gateway"}, nil, now, false)

	if got := auth.NextRetryAfter.Sub(now); got != 90*time.Second {
		t.Fatalf("NextRetryAfter delta = %s, want 90s override", got)
	}
}

func TestApplyAuthFailureState_NegativeTransientCooldownDisables(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, -1)

	now := time.Now()
	auth := &Auth{ID: "auth-cf-disabled", Provider: "codex"}
	applyAuthFailureState(auth, &Error{HTTPStatus: 525, Message: "ssl handshake failed"}, nil, now, false)

	if !auth.NextRetryAfter.IsZero() {
		t.Fatalf("NextRetryAfter = %s, want zero when transient cooldown disabled", auth.NextRetryAfter)
	}
	if auth.Unavailable {
		t.Fatal("expected auth to remain available when transient cooldown disabled")
	}
}

func TestApplyAuthFailureState_520DisableCoolingSkipsCooldown(t *testing.T) {
	withQuotaCooldownEnabled(t)
	withTransientCooldownSeconds(t, 0)

	now := time.Now()
	auth := &Auth{ID: "auth-cf-nocool", Provider: "codex"}
	applyAuthFailureState(auth, &Error{HTTPStatus: 523, Message: "origin unreachable"}, nil, now, true)

	if !auth.NextRetryAfter.IsZero() {
		t.Fatalf("NextRetryAfter = %s, want zero when disableCooling", auth.NextRetryAfter)
	}
	if auth.Unavailable {
		t.Fatal("expected auth to remain available when disableCooling")
	}
}

func TestRecoverableFailureRetryAfterWithHint_Priority(t *testing.T) {
	now := time.Now()
	hint := 7 * time.Second

	// disableCooling wins over everything.
	if got := recoverableFailureRetryAfterWithHint(now, &hint, true); !got.IsZero() {
		t.Fatalf("disableCooling: got %s, want zero", got)
	}
	// A positive hint beats the configured override and the default.
	withTransientCooldownSeconds(t, 42)
	if got := recoverableFailureRetryAfterWithHint(now, &hint, false); !got.Equal(now.Add(hint)) {
		t.Fatalf("hint precedence: got delta %s, want %s", got.Sub(now), hint)
	}
	// No hint: configured override applies.
	if got := recoverableFailureRetryAfterWithHint(now, nil, false); !got.Equal(now.Add(42 * time.Second)) {
		t.Fatalf("override: got delta %s, want 42s", got.Sub(now))
	}
}
