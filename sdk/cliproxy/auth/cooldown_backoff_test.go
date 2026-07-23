package auth

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestManager_ModelCooldownBackoffEscalatesOncePerActiveWindow(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, nil, nil)
	const (
		authID       = "model-cooldown-backoff-auth"
		model        = "model-cooldown-backoff-model"
		backoffLevel = 3
	)
	deadline := time.Now().Add(time.Hour)

	if _, err := manager.Register(ctx, &Auth{
		ID:       authID,
		Provider: "gemini",
		ModelStates: map[string]*ModelState{
			model: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: deadline,
				Quota: QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: deadline,
					BackoffLevel:  backoffLevel,
				},
			},
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result := Result{
		AuthID:   authID,
		Provider: "gemini",
		Model:    model,
		Error:    &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	}
	manager.MarkResult(ctx, result)
	assertModelCooldownWindow(t, manager, authID, model, deadline, backoffLevel)

	const concurrentFailures = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrentFailures)
	for range concurrentFailures {
		go func() {
			defer wg.Done()
			<-start
			manager.MarkResult(ctx, result)
		}()
	}
	close(start)
	wg.Wait()

	assertModelCooldownWindow(t, manager, authID, model, deadline, backoffLevel)
}

func TestManager_AuthCooldownBackoffEscalatesOncePerActiveWindow(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, nil, nil)
	const (
		authID       = "auth-cooldown-backoff"
		backoffLevel = 4
	)
	deadline := time.Now().Add(time.Hour)

	if _, err := manager.Register(ctx, &Auth{
		ID:             authID,
		Provider:       "gemini",
		Status:         StatusError,
		Unavailable:    true,
		NextRetryAfter: deadline,
		Quota: QuotaState{
			Exceeded:      true,
			Reason:        "quota",
			NextRecoverAt: deadline,
			BackoffLevel:  backoffLevel,
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result := Result{
		AuthID:   authID,
		Provider: "gemini",
		Error:    &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	}
	manager.MarkResult(ctx, result)
	manager.MarkResult(ctx, result)

	auth := managerAuthSnapshot(t, manager, authID)
	if !auth.NextRetryAfter.Equal(deadline) {
		t.Fatalf("NextRetryAfter = %v, want %v", auth.NextRetryAfter, deadline)
	}
	if !auth.Quota.NextRecoverAt.Equal(deadline) {
		t.Fatalf("Quota.NextRecoverAt = %v, want %v", auth.Quota.NextRecoverAt, deadline)
	}
	if auth.Quota.BackoffLevel != backoffLevel {
		t.Fatalf("Quota.BackoffLevel = %d, want %d", auth.Quota.BackoffLevel, backoffLevel)
	}
}

func TestQuotaCooldownBackoffDisabled(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name      string
		level     int
		wantLevel int
	}{
		{name: "preserves non-negative level", level: 4, wantLevel: 4},
		{name: "clamps negative level", level: -1, wantLevel: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, level := quotaCooldownAfterFailure(QuotaState{
				NextRecoverAt: now.Add(time.Hour),
				BackoffLevel:  tt.level,
			}, true, now)
			if !next.IsZero() {
				t.Fatalf("quotaCooldownAfterFailure() next = %v, want zero", next)
			}
			if level != tt.wantLevel {
				t.Fatalf("quotaCooldownAfterFailure() level = %d, want %d", level, tt.wantLevel)
			}
		})
	}
}

func TestJitteredCooldownWait(t *testing.T) {
	tests := []struct {
		name string
		base time.Duration
		max  time.Duration
		min  time.Duration
		high time.Duration
	}{
		{name: "maximum zero", base: time.Second, max: 0, min: 0, high: 0},
		{name: "maximum negative", base: time.Second, max: -time.Second, min: 0, high: 0},
		{name: "base negative", base: -time.Second, max: 10 * time.Second, min: 0, high: 0},
		{name: "base zero", base: 0, max: 10 * time.Second, min: 0, high: 0},
		{name: "base below maximum", base: 4 * time.Second, max: 10 * time.Second, min: 4 * time.Second, high: 5 * time.Second},
		{name: "jitter capped at two seconds", base: 20 * time.Second, max: 30 * time.Second, min: 20 * time.Second, high: 22 * time.Second},
		{name: "base equal maximum", base: 10 * time.Second, max: 10 * time.Second, min: 10 * time.Second, high: 10 * time.Second},
		{name: "base above maximum", base: 12 * time.Second, max: 10 * time.Second, min: 10 * time.Second, high: 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 64 {
				got := jitteredCooldownWait(tt.base, tt.max)
				if got < tt.min || got > tt.high {
					t.Fatalf("jitteredCooldownWait(%v, %v) = %v, want [%v, %v]", tt.base, tt.max, got, tt.min, tt.high)
				}
			}
		})
	}
}

func assertModelCooldownWindow(t *testing.T, manager *Manager, authID, model string, deadline time.Time, backoffLevel int) {
	t.Helper()
	auth := managerAuthSnapshot(t, manager, authID)
	state := auth.ModelStates[model]
	if state == nil {
		t.Fatalf("ModelStates[%q] is nil", model)
	}
	if !state.NextRetryAfter.Equal(deadline) {
		t.Fatalf("ModelStates[%q].NextRetryAfter = %v, want %v", model, state.NextRetryAfter, deadline)
	}
	if !state.Quota.NextRecoverAt.Equal(deadline) {
		t.Fatalf("ModelStates[%q].Quota.NextRecoverAt = %v, want %v", model, state.Quota.NextRecoverAt, deadline)
	}
	if state.Quota.BackoffLevel != backoffLevel {
		t.Fatalf("ModelStates[%q].Quota.BackoffLevel = %d, want %d", model, state.Quota.BackoffLevel, backoffLevel)
	}
}

func managerAuthSnapshot(t *testing.T, manager *Manager, authID string) *Auth {
	t.Helper()
	for _, auth := range manager.List() {
		if auth != nil && auth.ID == authID {
			return auth
		}
	}
	t.Fatalf("auth %q not found", authID)
	return nil
}
