package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// TestExecute_RetryRoundsExcludeCappedCredentials pins the R14 contract: when a
// retry round exhausts max-retry-credentials, the tried set carries into the
// next round's exclusion metadata so later rounds pick fresh credentials instead
// of replaying failed ones; single-credential cooldown retries are unaffected.
func TestExecute_RetryRoundsExcludeCappedCredentials(t *testing.T) {
	prev := quotaCooldownDisabled.Load()
	quotaCooldownDisabled.Store(true)
	t.Cleanup(func() { quotaCooldownDisabled.Store(prev) })

	m := NewManager(nil, nil, nil)
	m.SetRetryConfig(2, 10*time.Millisecond, 2)

	executor := &authFallbackExecutor{
		id: "claude",
		executeErrors: map[string]error{
			"rr-a": &retryAfterStatusError{status: http.StatusTooManyRequests, message: "quota", retryAfter: time.Millisecond},
			"rr-b": &retryAfterStatusError{status: http.StatusTooManyRequests, message: "quota", retryAfter: time.Millisecond},
			"rr-c": &retryAfterStatusError{status: http.StatusTooManyRequests, message: "quota", retryAfter: time.Millisecond},
		},
	}
	m.RegisterExecutor(executor)

	reg := registry.GetGlobalRegistry()
	for _, id := range []string{"rr-a", "rr-b", "rr-c"} {
		auth := &Auth{ID: id, Provider: "claude"}
		if _, errRegister := m.Register(context.Background(), auth); errRegister != nil {
			t.Fatalf("register %s: %v", id, errRegister)
		}
		reg.RegisterClient(id, "claude", []*registry.ModelInfo{{ID: "rr-model"}})
		t.Cleanup(func() { reg.UnregisterClient(id) })
	}

	req := cliproxyexecutor.Request{Model: "rr-model"}
	_, errExecute := m.Execute(context.Background(), []string{"claude"}, req, cliproxyexecutor.Options{})
	if errExecute == nil {
		t.Fatal("expected execute error after exhausting the pool")
	}
	if statusCodeFromError(errExecute) != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", statusCodeFromError(errExecute))
	}

	calls := executor.ExecuteCalls()
	seen := make(map[string]int, len(calls))
	for _, id := range calls {
		seen[id]++
	}
	t.Logf("call order: %v", calls)
	if len(seen) != 3 {
		t.Fatalf("expected all three distinct credentials to be attempted across rounds, got calls=%v", calls)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("credential %s executed %d times, want exactly 1 (round exclusion must prevent replay)", id, n)
		}
	}
}
