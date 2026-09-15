package auth

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
)

// The following tests cover the bounded ForceRefreshAll worker pool ported
// from upstream CLIProxyAPI commit 6dce78673fbc ("fix(auth): bound force
// refresh all concurrency using worker pool"). Local symbols under test:
// Manager.refreshWorkers, Manager.ForceRefreshAll, Manager.ForceRefreshAuth.

type blockingRefreshExecutor struct {
	schedulerProviderTestExecutor
	inFlight atomic.Int32
	maxSeen  atomic.Int32
	release  chan struct{}
}

func (e *blockingRefreshExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	cur := e.inFlight.Add(1)
	for {
		prev := e.maxSeen.Load()
		if cur <= prev || e.maxSeen.CompareAndSwap(prev, cur) {
			break
		}
	}
	defer e.inFlight.Add(-1)
	select {
	case <-e.release:
		return auth, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestRefreshWorkers_ResolvesConfigAndFallback(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	if got := manager.refreshWorkers(); got != refreshMaxConcurrency {
		t.Fatalf("refreshWorkers() = %d, want fallback %d", got, refreshMaxConcurrency)
	}
	manager.runtimeConfig.Store(&internalconfig.Config{AuthAutoRefreshWorkers: 3})
	if got := manager.refreshWorkers(); got != 3 {
		t.Fatalf("refreshWorkers() = %d, want configured 3", got)
	}
	manager.runtimeConfig.Store(&internalconfig.Config{AuthAutoRefreshWorkers: 0})
	if got := manager.refreshWorkers(); got != refreshMaxConcurrency {
		t.Fatalf("refreshWorkers() = %d, want fallback %d for non-positive config", got, refreshMaxConcurrency)
	}
}

func TestForceRefreshAll_BoundsConcurrency(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, nil, nil)
	exec := &blockingRefreshExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
		release:                       make(chan struct{}),
	}
	manager.RegisterExecutor(exec)
	manager.runtimeConfig.Store(&internalconfig.Config{AuthAutoRefreshWorkers: 2})

	const total = 6
	for i := 0; i < total; i++ {
		auth := &Auth{
			ID:       "force-pool-" + strings.Repeat("a", i+1),
			Provider: "codex",
			Metadata: map[string]any{"refresh_token": "rt"},
		}
		if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
			t.Fatalf("register %d: %v", i, errRegister)
		}
	}

	done := make(chan []ForceRefreshResult, 1)
	go func() {
		done <- manager.ForceRefreshAll(ctx)
	}()

	// Let workers pick up jobs, then release all refreshes.
	deadline := time.Now().Add(5 * time.Second)
	for exec.inFlight.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(exec.release)

	var results []ForceRefreshResult
	select {
	case results = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ForceRefreshAll did not complete")
	}

	if len(results) != total {
		t.Fatalf("ForceRefreshAll returned %d results, want %d", len(results), total)
	}
	for _, res := range results {
		if !res.Success {
			t.Fatalf("ForceRefreshAll result for %s failed: %s", res.ID, res.Error)
		}
	}
	if got := exec.maxSeen.Load(); got > 2 {
		t.Fatalf("max concurrent refreshes = %d, want <= 2 workers", got)
	}
}

type cancelAwareRefreshExecutor struct {
	schedulerProviderTestExecutor
	calls atomic.Int32
}

func (e *cancelAwareRefreshExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	e.calls.Add(1)
	return auth, nil
}

func TestForceRefreshAll_PreCanceledContextFailsQueuedJobs(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	exec := &cancelAwareRefreshExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
	}
	manager.RegisterExecutor(exec)

	const total = 4
	wantIDs := make([]string, 0, total)
	for i := 0; i < total; i++ {
		id := "force-cancel-" + strings.Repeat("b", i+1)
		wantIDs = append(wantIDs, id)
		if _, errRegister := manager.Register(context.Background(), &Auth{
			ID:       id,
			Provider: "codex",
			Metadata: map[string]any{"refresh_token": "rt"},
		}); errRegister != nil {
			t.Fatalf("register %d: %v", i, errRegister)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := manager.ForceRefreshAll(ctx)

	if len(results) != total {
		t.Fatalf("ForceRefreshAll returned %d results, want %d", len(results), total)
	}
	// Result slots follow the original auth ID index ordering.
	seen := make(map[string]bool, total)
	for _, res := range results {
		seen[res.ID] = true
		if res.Success {
			t.Fatalf("expected canceled refresh to fail for %s", res.ID)
		}
		if !strings.Contains(res.Error, "context") {
			t.Fatalf("result error for %s = %q, want context cancellation", res.ID, res.Error)
		}
	}
	for _, id := range wantIDs {
		if !seen[id] {
			t.Fatalf("missing result for auth %s", id)
		}
	}
	// Queued jobs must fail fast without invoking the executor.
	if got := exec.calls.Load(); got != 0 {
		t.Fatalf("executor Refresh called %d times, want 0 for pre-canceled context", got)
	}
}

func TestForceRefreshAll_SkipsDisabledAndTokenless(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	exec := &cancelAwareRefreshExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
	}
	manager.RegisterExecutor(exec)

	ctx := context.Background()
	if _, err := manager.Register(ctx, &Auth{ID: "disabled-auth", Provider: "codex", Disabled: true, Metadata: map[string]any{"refresh_token": "rt"}}); err != nil {
		t.Fatalf("register disabled: %v", err)
	}
	if _, err := manager.Register(ctx, &Auth{ID: "no-refresh-token", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}); err != nil {
		t.Fatalf("register tokenless: %v", err)
	}

	results := manager.ForceRefreshAll(ctx)
	if len(results) != 0 {
		t.Fatalf("ForceRefreshAll returned %d results, want 0 (disabled/tokenless skipped)", len(results))
	}
	if got := exec.calls.Load(); got != 0 {
		t.Fatalf("executor Refresh called %d times, want 0", got)
	}
}

func TestForceRefreshAuth_UnknownID(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	if _, err := manager.ForceRefreshAuth(context.Background(), "missing"); err == nil {
		t.Fatal("ForceRefreshAuth(missing) error = nil, want error")
	}
	if _, err := manager.ForceRefreshAuth(context.Background(), "  "); err == nil {
		t.Fatal("ForceRefreshAuth(blank) error = nil, want error")
	}
}
