package cliproxy

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type fakeQuotaAlertWaker struct {
	wakes atomic.Int32
}

func (f *fakeQuotaAlertWaker) Wake() { f.wakes.Add(1) }

func TestQuotaAlertResultHook(t *testing.T) {
	waker := &fakeQuotaAlertWaker{}
	hook := newQuotaAlertResultHook()
	base := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	now := base
	hook.now = func() time.Time { return now }

	quotaResult := coreauth.Result{
		AuthID:   "auth-1",
		Provider: "claude",
		Success:  false,
		Error:    &coreauth.Error{HTTPStatus: http.StatusTooManyRequests},
	}

	// No target bound yet: qualified evidence must not panic and must not wake.
	hook.OnResult(context.Background(), quotaResult)

	hook.SetTarget(nil)
	hook.OnResult(context.Background(), quotaResult)
	if waker.wakes.Load() != 0 {
		t.Fatalf("wakes = %d before target bound, want 0", waker.wakes.Load())
	}

	hook.mu.Lock()
	hook.target = waker
	hook.mu.Unlock()

	cases := []struct {
		name      string
		result    coreauth.Result
		wantWakes int32
	}{
		{name: "success ignored", result: coreauth.Result{AuthID: "auth-1", Provider: "claude", Success: true}},
		{name: "unsupported provider ignored", result: coreauth.Result{AuthID: "auth-2", Provider: "openai", Error: &coreauth.Error{HTTPStatus: http.StatusTooManyRequests}}},
		{name: "non-quota error ignored", result: coreauth.Result{AuthID: "auth-1", Provider: "claude", Error: &coreauth.Error{HTTPStatus: http.StatusInternalServerError}}},
		{name: "429 wakes", result: quotaResult, wantWakes: 1},
		{name: "credential scope wakes", result: coreauth.Result{AuthID: "auth-3", Provider: "codex", CredentialScope: true}, wantWakes: 1},
		{name: "quota code wakes", result: coreauth.Result{AuthID: "auth-4", Provider: "gemini", Error: &coreauth.Error{Code: "quota_exceeded"}}, wantWakes: 1},
		{name: "rate_limit code wakes", result: coreauth.Result{AuthID: "auth-5", Provider: "xai", Error: &coreauth.Error{Code: "rate_limit_exceeded"}}, wantWakes: 1},
	}
	for _, tc := range cases {
		before := waker.wakes.Load()
		hook.OnResult(context.Background(), tc.result)
		if got := waker.wakes.Load() - before; got != tc.wantWakes {
			t.Fatalf("%s: wakes delta = %d, want %d", tc.name, got, tc.wantWakes)
		}
	}

	// Min-interval dedup: same auth again within the window does not wake.
	before := waker.wakes.Load()
	hook.OnResult(context.Background(), quotaResult)
	if got := waker.wakes.Load() - before; got != 0 {
		t.Fatalf("min-interval dedup: wakes delta = %d, want 0", got)
	}

	// Past the min interval, the same auth wakes again.
	now = base.Add(2 * quotaAlertWakeMinInterval)
	hook.OnResult(context.Background(), quotaResult)
	if got := waker.wakes.Load() - before; got != 1 {
		t.Fatalf("post-interval wake delta = %d, want 1", got)
	}
}

type fakeRefreshExecutor struct {
	token string
	calls atomic.Int32
}

func (e *fakeRefreshExecutor) Identifier() string { return "claude" }

func (e *fakeRefreshExecutor) Execute(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *fakeRefreshExecutor) ExecuteStream(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, nil
}

func (e *fakeRefreshExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	e.calls.Add(1)
	updated := auth.Clone()
	updated.Metadata["access_token"] = e.token
	return updated, nil
}

func (e *fakeRefreshExecutor) CountTokens(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *fakeRefreshExecutor) HttpRequest(_ context.Context, _ *coreauth.Auth, req *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestQuotaAlertAuthRefresherReturnsRenewedSnapshot(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&fakeRefreshExecutor{token: "renewed-token"})
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       "auth-1",
		Provider: "claude",
		Metadata: map[string]any{"access_token": "stale-token", "refresh_token": "rt"},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	snapshots, err := NewQuotaAlertAuthSource(manager).ListQuotaAlertAuths(context.Background())
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("ListQuotaAlertAuths() = %v, %v", snapshots, err)
	}

	refresher := NewQuotaAlertAuthRefresher(manager)
	next, err := refresher(context.Background(), snapshots[0])
	if err != nil {
		t.Fatalf("refresh() error = %v", err)
	}
	if next.AuthID() != "auth-1" {
		t.Fatalf("refreshed auth id = %q, want auth-1", next.AuthID())
	}
	if token, ok := next.Metadata("access_token"); !ok || token != "renewed-token" {
		t.Fatalf("refreshed access_token = %v ok=%t, want renewed-token", token, ok)
	}

	if _, err := NewQuotaAlertAuthRefresher(nil)(context.Background(), snapshots[0]); err == nil {
		t.Fatal("nil manager refresh: want error")
	}
	if _, err := refresher(context.Background(), nil); err == nil {
		t.Fatal("nil snapshot refresh: want error")
	}
}
