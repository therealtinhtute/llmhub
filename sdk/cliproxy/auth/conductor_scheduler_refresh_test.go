package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type schedulerProviderTestExecutor struct {
	provider string
}

func (e schedulerProviderTestExecutor) Identifier() string { return e.provider }

func (e schedulerProviderTestExecutor) Execute(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e schedulerProviderTestExecutor) ExecuteStream(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, nil
}

func (e schedulerProviderTestExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	return auth, nil
}

func (e schedulerProviderTestExecutor) CountTokens(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e schedulerProviderTestExecutor) HttpRequest(ctx context.Context, auth *Auth, req *http.Request) (*http.Response, error) {
	return nil, nil
}

type unauthorizedRefreshTestExecutor struct {
	schedulerProviderTestExecutor
}

func (e unauthorizedRefreshTestExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	return nil, errors.New("token refresh failed with status 401: invalid_grant")
}

func TestManager_RefreshAuthUnauthorizedFailureStopsAutoRefreshRetry(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(unauthorizedRefreshTestExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
	})

	auth := &Auth{
		ID:       "unauthorized-refresh",
		Provider: "codex",
		Metadata: map[string]any{
			"email": "x@example.com",
		},
	}
	if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	manager.refreshAuth(ctx, auth.ID)

	updated, ok := manager.GetByID(auth.ID)
	if !ok {
		t.Fatalf("expected auth %q after refresh", auth.ID)
	}
	if updated.LastError == nil {
		t.Fatal("expected unauthorized refresh failure to be recorded")
	}
	if got := updated.LastError.StatusCode(); got != http.StatusUnauthorized {
		t.Fatalf("LastError.StatusCode() = %d, want %d", got, http.StatusUnauthorized)
	}
	if updated.LastError.Code != "unauthorized" {
		t.Fatalf("LastError.Code = %q, want unauthorized", updated.LastError.Code)
	}
	if !updated.NextRefreshAfter.IsZero() {
		t.Fatalf("NextRefreshAfter = %s, want zero for unauthorized refresh failure", updated.NextRefreshAfter)
	}
	now := time.Now()
	if manager.shouldRefresh(updated, now) {
		t.Fatal("expected unauthorized auth to stop refresh attempts")
	}
	if _, shouldSchedule := nextRefreshCheckAt(now, updated, time.Second); shouldSchedule {
		t.Fatal("expected unauthorized auth to be removed from the auto-refresh schedule")
	}
}

// The following tests cover the terminal-auth classification ported from
// upstream CLIProxyAPI commit aedc9e6a3987 ("fix(auth): classify terminal
// upstream auth failures as non-retryable errors"). Local symbols under test:
// IsTerminalAuthError, NewTerminalAuthError, hasUnauthorizedAuthFailure,
// availableAuthsForRouteModelWithPriorityMode (legacy path),
// modelScheduler.unavailableErrorLocked / authScheduler.mixedUnavailableErrorLocked
// (scheduler fast path), reached via Manager.pickNextMixed.

func TestManager_TerminalOAuthFailure_ReturnsTerminalAuthError(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name     string
		selector Selector
	}{
		{name: "fast_path_round_robin", selector: &RoundRobinSelector{}},
		// WeightedRoundRobinSelector is not a built-in scheduler selector, so
		// pickNextMixed falls back to pickNextMixedLegacy ->
		// availableAuthsForRouteModel, covering the conductor-selection terminal
		// classification. (Upstream's nil-selector subtest resolves to
		// RoundRobinSelector and never leaves the fast path.)
		{name: "legacy_path_weighted_selector", selector: &WeightedRoundRobinSelector{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager(nil, tc.selector, nil)
			manager.RegisterExecutor(unauthorizedRefreshTestExecutor{
				schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
			})

			authID := "unauthorized-terminal-" + tc.name
			healthyID := "healthy-" + tc.name
			auth := &Auth{
				ID:       authID,
				Provider: "codex",
				Metadata: map[string]any{
					"email": "x@example.com",
				},
			}
			if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
				t.Fatalf("register auth: %v", errRegister)
			}
			registry.GetGlobalRegistry().RegisterClient(authID, "codex", []*registry.ModelInfo{{ID: "any-model"}})
			t.Cleanup(func() {
				registry.GetGlobalRegistry().UnregisterClient(authID)
				registry.GetGlobalRegistry().UnregisterClient(healthyID)
			})

			manager.refreshAuth(ctx, auth.ID)

			_, _, _, errPick := manager.pickNextMixed(ctx, []string{"codex"}, "any-model", cliproxyexecutor.Options{}, nil)
			if errPick == nil {
				t.Fatal("expected pick error for terminal unauthorized auth")
			}
			if !IsTerminalAuthError(errPick) {
				t.Fatalf("expected IsTerminalAuthError to be true, got %T: %v", errPick, errPick)
			}
			var authErr *Error
			if !errors.As(errPick, &authErr) || authErr == nil {
				t.Fatalf("expected *Error, got %T: %v", errPick, errPick)
			}
			if authErr.Retryable {
				t.Fatal("expected Retryable to be false for terminal unauthorized auth")
			}
			if authErr.StatusCode() != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", authErr.StatusCode(), http.StatusServiceUnavailable)
			}

			// Negative case: adding a healthy credential allows selection to succeed.
			healthy := &Auth{ID: healthyID, Provider: "codex", Status: StatusActive}
			if _, errRegisterHealthy := manager.Register(ctx, healthy); errRegisterHealthy != nil {
				t.Fatalf("register healthy auth: %v", errRegisterHealthy)
			}
			registry.GetGlobalRegistry().RegisterClient(healthyID, "codex", []*registry.ModelInfo{{ID: "any-model"}})
			picked, _, _, errPickHealthy := manager.pickNextMixed(ctx, []string{"codex"}, "any-model", cliproxyexecutor.Options{}, nil)
			if errPickHealthy != nil {
				t.Fatalf("expected healthy pick to succeed, got: %v", errPickHealthy)
			}
			if picked == nil || picked.ID != healthyID {
				t.Fatalf("expected picked auth %q, got: %v", healthyID, picked)
			}
		})
	}
}

func TestManager_QuotaCooldown_DoesNotClassifyAsTerminalAuth(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(schedulerProviderTestExecutor{provider: "codex"})

	auth := &Auth{
		ID:          "quota-cooldown-auth",
		Provider:    "codex",
		Unavailable: true,
		Quota: QuotaState{
			Exceeded:      true,
			Reason:        "credential_quota",
			NextRecoverAt: time.Now().Add(time.Minute),
		},
	}
	if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}
	registry.GetGlobalRegistry().RegisterClient(auth.ID, "codex", []*registry.ModelInfo{{ID: "model-cooldown"}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(auth.ID)
	})

	_, _, _, errPick := manager.pickNextMixed(ctx, []string{"codex"}, "model-cooldown", cliproxyexecutor.Options{}, nil)
	if errPick == nil {
		t.Fatal("expected pick error for cooling auth")
	}
	if IsTerminalAuthError(errPick) {
		t.Fatalf("expected IsTerminalAuthError to be false for quota cooldown, got true")
	}
}

func TestManager_TerminalOAuthFailure_OverridesPreexistingModelStateError(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name     string
		selector Selector
	}{
		{name: "fast_path_round_robin", selector: &RoundRobinSelector{}},
		{name: "legacy_path_weighted_selector", selector: &WeightedRoundRobinSelector{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager(nil, tc.selector, nil)
			manager.RegisterExecutor(unauthorizedRefreshTestExecutor{
				schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
			})

			authID := "unauthorized-stale-model-" + tc.name
			auth := &Auth{
				ID:       authID,
				Provider: "codex",
				ModelStates: map[string]*ModelState{
					"any-model": {
						LastError: &Error{Code: "rate_limit_exceeded", Message: "historical rate limit", HTTPStatus: http.StatusTooManyRequests},
						UpdatedAt: time.Now().Add(-10 * time.Minute),
					},
				},
				Metadata: map[string]any{
					"email": "x@example.com",
				},
			}
			if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
				t.Fatalf("register auth: %v", errRegister)
			}
			registry.GetGlobalRegistry().RegisterClient(authID, "codex", []*registry.ModelInfo{{ID: "any-model"}})
			t.Cleanup(func() {
				registry.GetGlobalRegistry().UnregisterClient(authID)
			})

			// Trigger unauthorized refresh failure.
			manager.refreshAuth(ctx, auth.ID)

			_, _, _, errPick := manager.pickNextMixed(ctx, []string{"codex"}, "any-model", cliproxyexecutor.Options{}, nil)
			if errPick == nil {
				t.Fatal("expected pick error for terminal unauthorized auth")
			}
			if !IsTerminalAuthError(errPick) {
				t.Fatalf("expected IsTerminalAuthError to be true, got %T: %v", errPick, errPick)
			}
			cause := errors.Unwrap(errPick)
			if cause == nil {
				t.Fatal("expected non-nil cause")
			}
			if !strings.Contains(cause.Error(), "unauthorized") {
				t.Fatalf("expected cause to contain unauthorized OAuth error, got: %v", cause)
			}
			if strings.Contains(cause.Error(), "historical rate limit") {
				t.Fatalf("expected cause NOT to be masked by historical model error, got: %v", cause)
			}
		})
	}
}

func TestManager_RefreshSchedulerEntry_RebuildsSupportedModelSetAfterModelRegistration(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		prime func(*Manager, *Auth) error
	}{
		{
			name: "register",
			prime: func(manager *Manager, auth *Auth) error {
				_, errRegister := manager.Register(ctx, auth)
				return errRegister
			},
		},
		{
			name: "update",
			prime: func(manager *Manager, auth *Auth) error {
				_, errRegister := manager.Register(ctx, auth)
				if errRegister != nil {
					return errRegister
				}
				updated := auth.Clone()
				updated.Metadata = map[string]any{"updated": true}
				_, errUpdate := manager.Update(ctx, updated)
				return errUpdate
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			manager := NewManager(nil, &RoundRobinSelector{}, nil)
			auth := &Auth{
				ID:       "refresh-entry-" + testCase.name,
				Provider: "gemini",
			}
			if errPrime := testCase.prime(manager, auth); errPrime != nil {
				t.Fatalf("prime auth %s: %v", testCase.name, errPrime)
			}

			registerSchedulerModels(t, "gemini", "scheduler-refresh-model", auth.ID)

			got, errPick := manager.scheduler.pickSingle(ctx, "gemini", "scheduler-refresh-model", cliproxyexecutor.Options{}, nil)
			var authErr *Error
			if !errors.As(errPick, &authErr) || authErr == nil {
				t.Fatalf("pickSingle() before refresh error = %v, want auth_not_found", errPick)
			}
			if authErr.Code != "auth_not_found" {
				t.Fatalf("pickSingle() before refresh code = %q, want %q", authErr.Code, "auth_not_found")
			}
			if got != nil {
				t.Fatalf("pickSingle() before refresh auth = %v, want nil", got)
			}

			manager.RefreshSchedulerEntry(auth.ID)

			got, errPick = manager.scheduler.pickSingle(ctx, "gemini", "scheduler-refresh-model", cliproxyexecutor.Options{}, nil)
			if errPick != nil {
				t.Fatalf("pickSingle() after refresh error = %v", errPick)
			}
			if got == nil || got.ID != auth.ID {
				t.Fatalf("pickSingle() after refresh auth = %v, want %q", got, auth.ID)
			}
		})
	}
}

func TestManager_PickNext_RebuildsSchedulerAfterModelCooldownError(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(schedulerProviderTestExecutor{provider: "gemini"})

	registerSchedulerModels(t, "gemini", "scheduler-cooldown-rebuild-model", "cooldown-stale-old")

	oldAuth := &Auth{
		ID:       "cooldown-stale-old",
		Provider: "gemini",
	}
	if _, errRegister := manager.Register(ctx, oldAuth); errRegister != nil {
		t.Fatalf("register old auth: %v", errRegister)
	}

	manager.MarkResult(ctx, Result{
		AuthID:   oldAuth.ID,
		Provider: "gemini",
		Model:    "scheduler-cooldown-rebuild-model",
		Success:  false,
		Error:    &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})

	newAuth := &Auth{
		ID:       "cooldown-stale-new",
		Provider: "gemini",
	}
	if _, errRegister := manager.Register(ctx, newAuth); errRegister != nil {
		t.Fatalf("register new auth: %v", errRegister)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(newAuth.ID, "gemini", []*registry.ModelInfo{{ID: "scheduler-cooldown-rebuild-model"}})
	t.Cleanup(func() {
		reg.UnregisterClient(newAuth.ID)
	})

	got, errPick := manager.scheduler.pickSingle(ctx, "gemini", "scheduler-cooldown-rebuild-model", cliproxyexecutor.Options{}, nil)
	var cooldownErr *modelCooldownError
	if !errors.As(errPick, &cooldownErr) {
		t.Fatalf("pickSingle() before sync error = %v, want modelCooldownError", errPick)
	}
	if got != nil {
		t.Fatalf("pickSingle() before sync auth = %v, want nil", got)
	}

	got, executor, errPick := manager.pickNext(ctx, "gemini", "scheduler-cooldown-rebuild-model", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickNext() error = %v", errPick)
	}
	if executor == nil {
		t.Fatal("pickNext() executor = nil")
	}
	if got == nil || got.ID != newAuth.ID {
		t.Fatalf("pickNext() auth = %v, want %q", got, newAuth.ID)
	}
}
