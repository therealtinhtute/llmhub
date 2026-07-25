package auth

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/executionregistry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

func TestPickNextViaHomeReusesPinnedWebsocketAuthWithoutHomeDispatch(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.RegisterExecutor(schedulerTestExecutor{})

	auth := &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Status:   StatusActive,
		Attributes: map[string]string{
			"websockets":                  "true",
			homeUpstreamModelAttributeKey: "upstream-model",
		},
		Metadata: map[string]any{"email": "home@example.com"},
	}
	auth.EnsureIndex()
	manager.rememberHomeRuntimeAuth("session-1", auth)
	cachedAuth, ok := manager.GetExecutionSessionAuthByID("session-1", "home-auth-1")
	if !ok || cachedAuth == nil || !authWebsocketsEnabled(cachedAuth) {
		t.Fatalf("GetExecutionSessionAuthByID() did not expose remembered websocket home auth: auth=%#v ok=%v", cachedAuth, ok)
	}

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	opts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
		Headers: http.Header{"Authorization": {"Bearer client-key"}},
	}

	got, executor, provider, errPick := manager.pickNextViaHome(ctx, "gpt-5.4", opts, nil)
	if errPick != nil {
		t.Fatalf("pickNextViaHome() error = %v", errPick)
	}
	if got == nil || got.ID != "home-auth-1" {
		t.Fatalf("pickNextViaHome() auth = %#v, want home-auth-1", got)
	}
	if executor == nil {
		t.Fatal("pickNextViaHome() executor is nil")
	}
	if provider != "test" {
		t.Fatalf("pickNextViaHome() provider = %q, want test", provider)
	}
}

func TestPickNextViaHomeKeepsSameAuthIDPayloadSessionScoped(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.RegisterExecutor(schedulerTestExecutor{})

	manager.rememberHomeRuntimeAuth("session-1", &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Status:   StatusActive,
		Attributes: map[string]string{
			"websockets":                  "true",
			homeUpstreamModelAttributeKey: "upstream-model-a",
		},
	})
	manager.rememberHomeRuntimeAuth("session-2", &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Status:   StatusActive,
		Attributes: map[string]string{
			"websockets":                  "true",
			homeUpstreamModelAttributeKey: "upstream-model-b",
		},
	})

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	optsSession1 := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
	}
	optsSession2 := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-2",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
	}

	gotSession1, _, _, errSession1 := manager.pickNextViaHome(ctx, "gpt-5.4", optsSession1, nil)
	if errSession1 != nil {
		t.Fatalf("pickNextViaHome(session-1) error = %v", errSession1)
	}
	if got := gotSession1.Attributes[homeUpstreamModelAttributeKey]; got != "upstream-model-a" {
		t.Fatalf("pickNextViaHome(session-1) upstream model = %q, want upstream-model-a", got)
	}

	gotSession2, _, _, errSession2 := manager.pickNextViaHome(ctx, "gpt-5.4", optsSession2, nil)
	if errSession2 != nil {
		t.Fatalf("pickNextViaHome(session-2) error = %v", errSession2)
	}
	if got := gotSession2.Attributes[homeUpstreamModelAttributeKey]; got != "upstream-model-b" {
		t.Fatalf("pickNextViaHome(session-2) upstream model = %q, want upstream-model-b", got)
	}
}

func TestPickNextViaHomeDoesNotReuseTriedPinnedWebsocketAuth(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.RegisterExecutor(schedulerTestExecutor{})

	auth := &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Status:   StatusActive,
		Attributes: map[string]string{
			"websockets": "true",
		},
	}
	manager.rememberHomeRuntimeAuth("session-1", auth)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	opts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
	}
	tried := map[string]struct{}{"home-auth-1": {}}

	got, executor, provider, errPick := manager.pickNextViaHome(ctx, "gpt-5.4", opts, tried)
	if errPick == nil {
		t.Fatal("pickNextViaHome() error is nil, want home unavailable error")
	}
	var authErr *Error
	if !errors.As(errPick, &authErr) || authErr.Code != "home_unavailable" {
		t.Fatalf("pickNextViaHome() error = %v, want home_unavailable", errPick)
	}
	if got != nil || executor != nil || provider != "" {
		t.Fatalf("pickNextViaHome() reused tried auth: auth=%#v executor=%#v provider=%q", got, executor, provider)
	}
}

func TestPickNextViaHomeDoesNotReusePinnedWebsocketAuthAfterFirstHomeAttempt(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.RegisterExecutor(schedulerTestExecutor{})

	auth := &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Status:   StatusActive,
		Attributes: map[string]string{
			"websockets": "true",
		},
	}
	manager.rememberHomeRuntimeAuth("session-1", auth)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	opts := withHomeAuthCount(cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
	}, 2)

	got, executor, provider, errPick := manager.pickNextViaHome(ctx, "gpt-5.4", opts, nil)
	if errPick == nil {
		t.Fatal("pickNextViaHome() error is nil, want home unavailable error")
	}
	var authErr *Error
	if !errors.As(errPick, &authErr) || authErr.Code != "home_unavailable" {
		t.Fatalf("pickNextViaHome() error = %v, want home_unavailable", errPick)
	}
	if got != nil || executor != nil || provider != "" {
		t.Fatalf("pickNextViaHome() reused auth after first home attempt: auth=%#v executor=%#v provider=%q", got, executor, provider)
	}
}

func TestPickNextViaHomeDoesNotReusePinnedNonWebsocketAuth(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.RegisterExecutor(schedulerTestExecutor{})

	manager.mu.Lock()
	manager.homeRuntimeAuths["session-1"] = map[string]*Auth{
		"home-auth-1": &Auth{
			ID:       "home-auth-1",
			Provider: "test",
			Status:   StatusActive,
		},
	}
	manager.mu.Unlock()

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	opts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
			cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
		},
		Headers: http.Header{"Authorization": {"Bearer client-key"}},
	}

	got, executor, provider, errPick := manager.pickNextViaHome(ctx, "gpt-5.4", opts, nil)
	if errPick == nil {
		t.Fatal("pickNextViaHome() error is nil, want home unavailable error")
	}
	var authErr *Error
	if !errors.As(errPick, &authErr) || authErr.Code != "home_unavailable" {
		t.Fatalf("pickNextViaHome() error = %v, want home_unavailable", errPick)
	}
	if got != nil || executor != nil || provider != "" {
		t.Fatalf("pickNextViaHome() reused non-websocket auth: auth=%#v executor=%#v provider=%q", got, executor, provider)
	}
}

func TestHomeRuntimeAuthsClearWhenHomeDisabled(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.rememberHomeRuntimeAuth("session-1", &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Attributes: map[string]string{
			"websockets": "true",
		},
	})

	if _, ok := manager.GetExecutionSessionAuthByID("session-1", "home-auth-1"); !ok {
		t.Fatal("expected remembered home auth before disabling home")
	}

	manager.SetConfig(&internalconfig.Config{})
	if _, ok := manager.GetExecutionSessionAuthByID("session-1", "home-auth-1"); ok {
		t.Fatal("remembered home auth was not cleared when home was disabled")
	}
}

func TestCloseExecutionSessionClearsHomeRuntimeAuthForSession(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "home-auth-1",
		Provider: "test",
		Attributes: map[string]string{
			"websockets": "true",
		},
	}

	manager.rememberHomeRuntimeAuth("session-1", auth)
	manager.rememberHomeRuntimeAuth("session-2", auth)

	manager.CloseExecutionSession("session-1")
	if _, ok := manager.GetExecutionSessionAuthByID("session-1", "home-auth-1"); ok {
		t.Fatal("home auth for closed session was not cleared")
	}
	if _, ok := manager.GetExecutionSessionAuthByID("session-2", "home-auth-1"); !ok {
		t.Fatal("home auth for another session was cleared")
	}

	manager.CloseExecutionSession("session-2")
	if _, ok := manager.GetExecutionSessionAuthByID("session-2", "home-auth-1"); ok {
		t.Fatal("home auth was not cleared when its last session closed")
	}
}

type retainingHomeSessionExecutor struct {
	schedulerTestExecutor
	calls atomic.Int32
}

func (*retainingHomeSessionExecutor) Identifier() string { return "retaining-test" }

func (e *retainingHomeSessionExecutor) Execute(_ context.Context, _ *Auth, _ cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	e.calls.Add(1)
	if lifecycle, ok := opts.ExecutionLifecycle.(interface{ Retain() }); ok {
		lifecycle.Retain()
	}
	return cliproxyexecutor.Response{Payload: []byte("ok")}, nil
}

func TestHomeWebsocketSessionRetainsOneAccountedSelectionUntilClose(t *testing.T) {
	payload := []byte(`{"concurrency":{"accounted":true,"credential_id":"home-auth-1","model":"model-a"},"auth_index":"home-auth-1","auth":{"id":"home-auth-1","provider":"retaining-test","status":"active","attributes":{"websockets":"true"}},"model":"model-a"}`)
	dispatcher := &fixtureHomeDispatcher{payloads: [][]byte{payload, payload}}
	registry := executionregistry.New()
	releases := make(chan executionregistry.ReleaseGroup, 2)
	registry.SetReleaseSink(func(group executionregistry.ReleaseGroup, _ int64) *executionregistry.ReleaseTicket {
		releases <- group
		return nil
	})

	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.PublishHomeDispatch(dispatcher, registry, 1)
	executor := &retainingHomeSessionExecutor{}
	manager.RegisterExecutor(executor)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.ExecutionSessionMetadataKey: "session-1",
		cliproxyexecutor.PinnedAuthMetadataKey:       "home-auth-1",
	}}
	for range 2 {
		if _, errExecute := manager.Execute(ctx, []string{"retaining-test"}, cliproxyexecutor.Request{Model: "model-a"}, opts); errExecute != nil {
			t.Fatalf("Execute() error = %v", errExecute)
		}
	}
	if got := dispatcher.calls; got != 1 {
		t.Fatalf("Home RPOP calls = %d, want 1", got)
	}
	if got := executor.calls.Load(); got != 2 {
		t.Fatalf("executor calls = %d, want 2", got)
	}
	select {
	case group := <-releases:
		t.Fatalf("selection released before session close: %#v", group)
	default:
	}

	manager.CloseExecutionSession("session-1")
	select {
	case group := <-releases:
		want := executionregistry.ReleaseGroup{CredentialID: "home-auth-1", Model: "model-a"}
		if group != want {
			t.Fatalf("release group = %#v, want %#v", group, want)
		}
	case <-time.After(time.Second):
		t.Fatal("session close did not release accounted selection")
	}
}
