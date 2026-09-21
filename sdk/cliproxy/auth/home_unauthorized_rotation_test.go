package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/executionregistry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Ported from upstream CLIProxyAPI commit 9a2201c36a0a
// (TestHomeStreamOAuthUnauthorizedRotatesWithoutRefreshRetry), adapted to the
// local Home dispatch generation: a Home-owned credential that fails with an
// upstream 401 is not refreshed or retried locally — the conductor ends the
// selection and asks Home for a different credential.

type unauthorizedRotationDispatcher struct {
	calls   atomic.Int32
	counts  []int
	mu      sync.Mutex
	repeatA bool // return home-retry-a on every call
}

func (*unauthorizedRotationDispatcher) HeartbeatOK() bool { return true }

func (d *unauthorizedRotationDispatcher) RPopAuth(_ context.Context, model, _ string, _ http.Header, count int) ([]byte, error) {
	call := d.calls.Add(1)
	d.mu.Lock()
	d.counts = append(d.counts, count)
	d.mu.Unlock()
	if d.repeatA {
		return json.Marshal(homeAuthDispatchResponse{
			Model:     model,
			AuthIndex: "home-retry-a",
			Auth:      Auth{ID: "home-retry-a", Provider: "codex", Status: StatusActive},
		})
	}
	ids := []string{"home-retry-a", "home-retry-b"}
	if int(call) > len(ids) {
		return nil, &Error{Code: "auth_not_found", Message: "no more auths"}
	}
	id := ids[call-1]
	return json.Marshal(homeAuthDispatchResponse{
		Model:     model,
		AuthIndex: id,
		Auth:      Auth{ID: id, Provider: "codex", Status: StatusActive},
	})
}

func (*unauthorizedRotationDispatcher) AbortAmbiguousDispatch() {}

type unauthorizedRotationExecutor struct {
	schedulerTestExecutor
	calls        []string
	mu           sync.Mutex
	refreshCalls atomic.Int32
}

func (*unauthorizedRotationExecutor) Identifier() string { return "codex" }

func (e *unauthorizedRotationExecutor) Execute(_ context.Context, auth *Auth, _ cliproxyexecutor.Request, _ cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	e.mu.Lock()
	e.calls = append(e.calls, auth.ID)
	e.mu.Unlock()
	if auth.ID == "home-retry-a" {
		return cliproxyexecutor.Response{}, &Error{HTTPStatus: http.StatusUnauthorized, Message: "access token expired"}
	}
	return cliproxyexecutor.Response{Payload: []byte("ok")}, nil
}

func (e *unauthorizedRotationExecutor) ExecuteStream(_ context.Context, auth *Auth, _ cliproxyexecutor.Request, _ cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	e.mu.Lock()
	e.calls = append(e.calls, auth.ID)
	e.mu.Unlock()
	if auth.ID == "home-retry-a" {
		return nil, &Error{HTTPStatus: http.StatusUnauthorized, Message: "access token expired"}
	}
	ch := make(chan cliproxyexecutor.StreamChunk, 1)
	ch <- cliproxyexecutor.StreamChunk{Payload: []byte("ok")}
	close(ch)
	return &cliproxyexecutor.StreamResult{Chunks: ch}, nil
}

// Refresh counts calls to prove the conductor never refreshes a Home-owned
// credential after an upstream 401.
func (e *unauthorizedRotationExecutor) Refresh(_ context.Context, auth *Auth) (*Auth, error) {
	e.refreshCalls.Add(1)
	return auth, nil
}

func (e *unauthorizedRotationExecutor) Calls() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.calls...)
}

func TestHomeStreamOAuthUnauthorizedRotatesWithoutRefreshRetry(t *testing.T) {
	dispatcher := &unauthorizedRotationDispatcher{}
	executor := &unauthorizedRotationExecutor{}
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.SetRetryConfig(0, time.Second, 2)
	manager.PublishHomeDispatch(dispatcher, executionregistry.New(), 1)
	manager.RegisterExecutor(executor)

	result, errExecute := manager.ExecuteStream(context.Background(), []string{"codex"}, cliproxyexecutor.Request{Model: "gpt"}, cliproxyexecutor.Options{Stream: true})
	if errExecute != nil {
		t.Fatalf("ExecuteStream() error = %v", errExecute)
	}
	for range result.Chunks {
	}
	if got := executor.Calls(); len(got) != 2 || got[0] != "home-retry-a" || got[1] != "home-retry-b" {
		t.Fatalf("executor calls = %v, want [home-retry-a home-retry-b] (no refresh retry of the unauthorized credential)", got)
	}
	if got := dispatcher.calls.Load(); got != 2 {
		t.Fatalf("Home RPOP calls = %d, want 2", got)
	}
	dispatcher.mu.Lock()
	counts := append([]int(nil), dispatcher.counts...)
	dispatcher.mu.Unlock()
	if len(counts) != 2 || counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("Home dispatch attempt counts = %v, want [1 2]", counts)
	}
}

// Ported from upstream CLIProxyAPI commit 9a2201c36a0a
// (TestHomeUnauthorizedReturnsOriginalErrorWithoutRefresh), adapted: when Home
// re-pops the already-tried credential the conductor returns the original
// upstream 401 instead of refreshing or replaying it.
func TestHomeUnauthorizedReturnsOriginalErrorWithoutRefresh(t *testing.T) {
	dispatcher := &unauthorizedRotationDispatcher{repeatA: true}
	executor := &unauthorizedRotationExecutor{}
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.SetRetryConfig(0, time.Second, 2)
	manager.PublishHomeDispatch(dispatcher, executionregistry.New(), 1)
	manager.RegisterExecutor(executor)

	_, errExecute := manager.Execute(context.Background(), []string{"codex"}, cliproxyexecutor.Request{Model: "gpt"}, cliproxyexecutor.Options{})
	if errExecute == nil || errExecute.Error() != "access token expired" || statusCodeFromError(errExecute) != http.StatusUnauthorized {
		t.Fatalf("Execute() error = %v, want original 401", errExecute)
	}
	if got := executor.refreshCalls.Load(); got != 0 {
		t.Fatalf("refresh calls = %d, want 0", got)
	}
	if got := executor.Calls(); len(got) != 1 || got[0] != "home-retry-a" {
		t.Fatalf("executor calls = %v, want single attempt on home-retry-a", got)
	}
}
