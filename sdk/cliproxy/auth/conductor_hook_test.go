package auth

import (
	"context"
	"sync/atomic"
	"testing"
)

type recordingHook struct {
	NoopHook
	results atomic.Int32
	auths   atomic.Int32
}

func (h *recordingHook) OnResult(context.Context, Result) {
	h.results.Add(1)
}

func (h *recordingHook) OnAuthRegistered(context.Context, *Auth) {
	h.auths.Add(1)
}

func TestManagerAddHookComposesAfterConstructionHook(t *testing.T) {
	first := &recordingHook{}
	second := &recordingHook{}
	manager := NewManager(nil, nil, first)
	manager.AddHook(second)

	manager.MarkResult(context.Background(), Result{
		AuthID:        "auth-1",
		Provider:      "claude",
		RequestScoped: true,
		Success:       false,
	})
	if first.results.Load() != 1 || second.results.Load() != 1 {
		t.Fatalf("hook results = %d/%d, want 1/1", first.results.Load(), second.results.Load())
	}
	if _, err := manager.Register(context.Background(), &Auth{ID: "auth-1", Provider: "claude"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if first.auths.Load() != 1 || second.auths.Load() != 1 {
		t.Fatalf("hook registrations = %d/%d, want 1/1", first.auths.Load(), second.auths.Load())
	}

	manager.AddHook(nil)
	third := &recordingHook{}
	manager.AddHook(third)
	manager.MarkResult(context.Background(), Result{AuthID: "auth-1", Provider: "claude", RequestScoped: true, Success: false})
	if first.results.Load() != 2 || second.results.Load() != 2 || third.results.Load() != 1 {
		t.Fatalf("chained hook results = %d/%d/%d, want 2/2/1", first.results.Load(), second.results.Load(), third.results.Load())
	}
}
