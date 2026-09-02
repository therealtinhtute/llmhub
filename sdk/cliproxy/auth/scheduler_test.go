package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type schedulerTestExecutor struct{}

func (schedulerTestExecutor) Identifier() string { return "test" }

func (schedulerTestExecutor) Execute(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (schedulerTestExecutor) ExecuteStream(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, nil
}

func (schedulerTestExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	return auth, nil
}

func (schedulerTestExecutor) CountTokens(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (schedulerTestExecutor) HttpRequest(ctx context.Context, auth *Auth, req *http.Request) (*http.Response, error) {
	return nil, nil
}

type trackingSelector struct {
	calls      int
	lastAuthID []string
}

func (s *trackingSelector) Pick(ctx context.Context, provider, model string, opts cliproxyexecutor.Options, auths []*Auth) (*Auth, error) {
	s.calls++
	s.lastAuthID = s.lastAuthID[:0]
	for _, auth := range auths {
		s.lastAuthID = append(s.lastAuthID, auth.ID)
	}
	if len(auths) == 0 {
		return nil, nil
	}
	return auths[len(auths)-1], nil
}

func newSchedulerForTest(selector Selector, auths ...*Auth) *authScheduler {
	scheduler := newAuthScheduler(selector)
	scheduler.rebuild(auths)
	return scheduler
}

func registerSchedulerModels(t *testing.T, provider string, model string, authIDs ...string) {
	t.Helper()
	reg := registry.GetGlobalRegistry()
	for _, authID := range authIDs {
		reg.RegisterClient(authID, provider, []*registry.ModelInfo{{ID: model}})
	}
	t.Cleanup(func() {
		for _, authID := range authIDs {
			reg.UnregisterClient(authID)
		}
	})
}

func TestSchedulerPick_RoundRobinHighestPriority(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{ID: "low", Provider: "gemini", Attributes: map[string]string{"priority": "0"}},
		&Auth{ID: "high-b", Provider: "gemini", Attributes: map[string]string{"priority": "10"}},
		&Auth{ID: "high-a", Provider: "gemini", Attributes: map[string]string{"priority": "10"}},
	)

	want := []string{"high-a", "high-b", "high-a"}
	for index, wantID := range want {
		got, errPick := scheduler.pickSingle(context.Background(), "gemini", "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickSingle() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickSingle() #%d auth = nil", index)
		}
		if got.ID != wantID {
			t.Fatalf("pickSingle() #%d auth.ID = %q, want %q", index, got.ID, wantID)
		}
	}
}

func TestSchedulerPick_WeightedRoundRobin(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&WeightedRoundRobinSelector{},
		&Auth{ID: "a", Provider: "gemini", Attributes: map[string]string{AttributeWeight: "5"}},
		&Auth{ID: "b", Provider: "gemini", Attributes: map[string]string{AttributeWeight: "3"}},
		&Auth{ID: "c", Provider: "gemini", Attributes: map[string]string{AttributeWeight: "2"}},
	)

	counts := make(map[string]int)
	for index := 0; index < 100; index++ {
		got, errPick := scheduler.pickSingle(context.Background(), "gemini", "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickSingle() #%d error = %v", index, errPick)
		}
		counts[got.ID]++
	}
	want := map[string]int{"a": 50, "b": 30, "c": 20}
	for authID, wantCount := range want {
		if counts[authID] != wantCount {
			t.Fatalf("auth %q picks = %d, want %d", authID, counts[authID], wantCount)
		}
	}
}

func TestSchedulerPick_FillFirstSticksToFirstReady(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&FillFirstSelector{},
		&Auth{ID: "b", Provider: "gemini"},
		&Auth{ID: "a", Provider: "gemini"},
		&Auth{ID: "c", Provider: "gemini"},
	)

	for index := 0; index < 3; index++ {
		got, errPick := scheduler.pickSingle(context.Background(), "gemini", "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickSingle() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickSingle() #%d auth = nil", index)
		}
		if got.ID != "a" {
			t.Fatalf("pickSingle() #%d auth.ID = %q, want %q", index, got.ID, "a")
		}
	}
}

func TestSchedulerPick_PromotesExpiredCooldownBeforePick(t *testing.T) {
	t.Parallel()

	model := "gemini-2.5-pro"
	registerSchedulerModels(t, "gemini", model, "cooldown-expired")
	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{
			ID:       "cooldown-expired",
			Provider: "gemini",
			ModelStates: map[string]*ModelState{
				model: {
					Status:         StatusError,
					Unavailable:    true,
					NextRetryAfter: time.Now().Add(-1 * time.Second),
				},
			},
		},
	)

	got, errPick := scheduler.pickSingle(context.Background(), "gemini", model, cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle() error = %v", errPick)
	}
	if got == nil {
		t.Fatalf("pickSingle() auth = nil")
	}
	if got.ID != "cooldown-expired" {
		t.Fatalf("pickSingle() auth.ID = %q, want %q", got.ID, "cooldown-expired")
	}
}

func TestSchedulerPick_CodexWebsocketPrefersWebsocketEnabledSubset(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{ID: "codex-http", Provider: "codex"},
		&Auth{ID: "codex-ws-a", Provider: "codex", Attributes: map[string]string{"websockets": "true"}},
		&Auth{ID: "codex-ws-b", Provider: "codex", Attributes: map[string]string{"websockets": "true"}},
	)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	want := []string{"codex-ws-a", "codex-ws-b", "codex-ws-a"}
	for index, wantID := range want {
		got, errPick := scheduler.pickSingle(ctx, "codex", "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickSingle() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickSingle() #%d auth = nil", index)
		}
		if got.ID != wantID {
			t.Fatalf("pickSingle() #%d auth.ID = %q, want %q", index, got.ID, wantID)
		}
	}
}

func TestSchedulerPick_CodexWebsocketPrefersWebsocketEnabledAcrossPriorities(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{ID: "codex-http", Provider: "codex", Attributes: map[string]string{"priority": "10"}},
		&Auth{ID: "codex-ws-a", Provider: "codex", Attributes: map[string]string{"priority": "0", "websockets": "true"}},
		&Auth{ID: "codex-ws-b", Provider: "codex", Attributes: map[string]string{"priority": "0", "websockets": "true"}},
	)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	want := []string{"codex-ws-a", "codex-ws-b", "codex-ws-a"}
	for index, wantID := range want {
		got, errPick := scheduler.pickSingle(ctx, "codex", "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickSingle() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickSingle() #%d auth = nil", index)
		}
		if got.ID != wantID {
			t.Fatalf("pickSingle() #%d auth.ID = %q, want %q", index, got.ID, wantID)
		}
	}
}

func TestSchedulerPick_MixedProvidersUsesWeightedProviderRotationOverReadyCandidates(t *testing.T) {
	t.Parallel()

	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{ID: "gemini-a", Provider: "gemini"},
		&Auth{ID: "gemini-b", Provider: "gemini"},
		&Auth{ID: "claude-a", Provider: "claude"},
	)

	wantProviders := []string{"gemini", "gemini", "claude", "gemini"}
	wantIDs := []string{"gemini-a", "gemini-b", "claude-a", "gemini-a"}
	for index := range wantProviders {
		got, provider, errPick := scheduler.pickMixed(context.Background(), []string{"gemini", "claude"}, "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickMixed() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickMixed() #%d auth = nil", index)
		}
		if provider != wantProviders[index] {
			t.Fatalf("pickMixed() #%d provider = %q, want %q", index, provider, wantProviders[index])
		}
		if got.ID != wantIDs[index] {
			t.Fatalf("pickMixed() #%d auth.ID = %q, want %q", index, got.ID, wantIDs[index])
		}
	}
}

func TestSchedulerPick_MixedProvidersPrefersHighestPriorityTier(t *testing.T) {
	t.Parallel()

	model := "gpt-default"
	registerSchedulerModels(t, "provider-low", model, "low")
	registerSchedulerModels(t, "provider-high-a", model, "high-a")
	registerSchedulerModels(t, "provider-high-b", model, "high-b")

	scheduler := newSchedulerForTest(
		&RoundRobinSelector{},
		&Auth{ID: "low", Provider: "provider-low", Attributes: map[string]string{"priority": "4"}},
		&Auth{ID: "high-a", Provider: "provider-high-a", Attributes: map[string]string{"priority": "7"}},
		&Auth{ID: "high-b", Provider: "provider-high-b", Attributes: map[string]string{"priority": "7"}},
	)

	providers := []string{"provider-low", "provider-high-a", "provider-high-b"}
	wantProviders := []string{"provider-high-a", "provider-high-b", "provider-high-a", "provider-high-b"}
	wantIDs := []string{"high-a", "high-b", "high-a", "high-b"}
	for index := range wantProviders {
		got, provider, errPick := scheduler.pickMixed(context.Background(), providers, model, cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickMixed() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickMixed() #%d auth = nil", index)
		}
		if provider != wantProviders[index] {
			t.Fatalf("pickMixed() #%d provider = %q, want %q", index, provider, wantProviders[index])
		}
		if got.ID != wantIDs[index] {
			t.Fatalf("pickMixed() #%d auth.ID = %q, want %q", index, got.ID, wantIDs[index])
		}
	}
}

func TestManager_PickNextMixed_UsesWeightedProviderRotationBeforeCredentialRotation(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.executors["gemini"] = schedulerTestExecutor{}
	manager.executors["claude"] = schedulerTestExecutor{}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "gemini-a", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(gemini-a) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "gemini-b", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(gemini-b) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "claude-a", Provider: "claude"}); errRegister != nil {
		t.Fatalf("Register(claude-a) error = %v", errRegister)
	}

	wantProviders := []string{"gemini", "gemini", "claude", "gemini"}
	wantIDs := []string{"gemini-a", "gemini-b", "claude-a", "gemini-a"}
	for index := range wantProviders {
		got, _, provider, errPick := manager.pickNextMixed(context.Background(), []string{"gemini", "claude"}, "", cliproxyexecutor.Options{}, map[string]struct{}{})
		if errPick != nil {
			t.Fatalf("pickNextMixed() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickNextMixed() #%d auth = nil", index)
		}
		if provider != wantProviders[index] {
			t.Fatalf("pickNextMixed() #%d provider = %q, want %q", index, provider, wantProviders[index])
		}
		if got.ID != wantIDs[index] {
			t.Fatalf("pickNextMixed() #%d auth.ID = %q, want %q", index, got.ID, wantIDs[index])
		}
	}
}

func TestManager_PickNextMixed_DisallowFreeAuthSkipsCodexFreePlan(t *testing.T) {
	t.Parallel()

	model := "gpt-5.4-mini"
	registerSchedulerModels(t, "codex", model, "codex-a-free", "codex-b-plus")

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.executors["codex"] = schedulerTestExecutor{}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "codex-a-free", Provider: "codex", Attributes: map[string]string{"plan_type": "free"}}); errRegister != nil {
		t.Fatalf("Register(codex-a-free) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "codex-b-plus", Provider: "codex", Attributes: map[string]string{"plan_type": "plus"}}); errRegister != nil {
		t.Fatalf("Register(codex-b-plus) error = %v", errRegister)
	}

	opts := cliproxyexecutor.Options{
		Metadata: map[string]any{cliproxyexecutor.DisallowFreeAuthMetadataKey: true},
	}
	got, _, provider, errPick := manager.pickNextMixed(context.Background(), []string{"codex"}, model, opts, map[string]struct{}{})
	if errPick != nil {
		t.Fatalf("pickNextMixed() error = %v", errPick)
	}
	if got == nil {
		t.Fatalf("pickNextMixed() auth = nil")
	}
	if provider != "codex" {
		t.Fatalf("pickNextMixed() provider = %q, want %q", provider, "codex")
	}
	if got.ID != "codex-b-plus" {
		t.Fatalf("pickNextMixed() auth.ID = %q, want %q", got.ID, "codex-b-plus")
	}
}

func TestManagerCustomSelector_FallsBackToLegacyPath(t *testing.T) {
	t.Parallel()

	selector := &trackingSelector{}
	manager := NewManager(nil, selector, nil)
	manager.executors["gemini"] = schedulerTestExecutor{}
	manager.auths["auth-a"] = &Auth{ID: "auth-a", Provider: "gemini"}
	manager.auths["auth-b"] = &Auth{ID: "auth-b", Provider: "gemini"}

	got, _, errPick := manager.pickNext(context.Background(), "gemini", "", cliproxyexecutor.Options{}, map[string]struct{}{})
	if errPick != nil {
		t.Fatalf("pickNext() error = %v", errPick)
	}
	if got == nil {
		t.Fatalf("pickNext() auth = nil")
	}
	if selector.calls != 1 {
		t.Fatalf("selector.calls = %d, want %d", selector.calls, 1)
	}
	if len(selector.lastAuthID) != 2 {
		t.Fatalf("len(selector.lastAuthID) = %d, want %d", len(selector.lastAuthID), 2)
	}
	if got.ID != selector.lastAuthID[len(selector.lastAuthID)-1] {
		t.Fatalf("pickNext() auth.ID = %q, want selector-picked %q", got.ID, selector.lastAuthID[len(selector.lastAuthID)-1])
	}
}

func TestManager_InitializesSchedulerForBuiltInSelector(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	if manager.scheduler == nil {
		t.Fatalf("manager.scheduler = nil")
	}
	if manager.scheduler.strategy != schedulerStrategyRoundRobin {
		t.Fatalf("manager.scheduler.strategy = %v, want %v", manager.scheduler.strategy, schedulerStrategyRoundRobin)
	}

	manager.SetSelector(&FillFirstSelector{})
	if manager.scheduler.strategy != schedulerStrategyFillFirst {
		t.Fatalf("manager.scheduler.strategy = %v, want %v", manager.scheduler.strategy, schedulerStrategyFillFirst)
	}
}

func TestManager_SchedulerTracksRegisterAndUpdate(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "auth-b", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(auth-b) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "auth-a", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(auth-a) error = %v", errRegister)
	}

	got, errPick := manager.scheduler.pickSingle(context.Background(), "gemini", "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("scheduler.pickSingle() error = %v", errPick)
	}
	if got == nil || got.ID != "auth-a" {
		t.Fatalf("scheduler.pickSingle() auth = %v, want auth-a", got)
	}

	if _, errUpdate := manager.Update(context.Background(), &Auth{ID: "auth-a", Provider: "gemini", Disabled: true}); errUpdate != nil {
		t.Fatalf("Update(auth-a) error = %v", errUpdate)
	}

	got, errPick = manager.scheduler.pickSingle(context.Background(), "gemini", "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("scheduler.pickSingle() after update error = %v", errPick)
	}
	if got == nil || got.ID != "auth-b" {
		t.Fatalf("scheduler.pickSingle() after update auth = %v, want auth-b", got)
	}
}

func TestManager_PickNextMixed_UsesSchedulerRotation(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.executors["gemini"] = schedulerTestExecutor{}
	manager.executors["claude"] = schedulerTestExecutor{}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "gemini-a", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(gemini-a) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "gemini-b", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(gemini-b) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "claude-a", Provider: "claude"}); errRegister != nil {
		t.Fatalf("Register(claude-a) error = %v", errRegister)
	}

	wantProviders := []string{"gemini", "gemini", "claude", "gemini"}
	wantIDs := []string{"gemini-a", "gemini-b", "claude-a", "gemini-a"}
	for index := range wantProviders {
		got, _, provider, errPick := manager.pickNextMixed(context.Background(), []string{"gemini", "claude"}, "", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("pickNextMixed() #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("pickNextMixed() #%d auth = nil", index)
		}
		if provider != wantProviders[index] {
			t.Fatalf("pickNextMixed() #%d provider = %q, want %q", index, provider, wantProviders[index])
		}
		if got.ID != wantIDs[index] {
			t.Fatalf("pickNextMixed() #%d auth.ID = %q, want %q", index, got.ID, wantIDs[index])
		}
	}
}

func TestManager_PickNextMixed_SkipsProvidersWithoutExecutors(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.executors["claude"] = schedulerTestExecutor{}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "gemini-a", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(gemini-a) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "claude-a", Provider: "claude"}); errRegister != nil {
		t.Fatalf("Register(claude-a) error = %v", errRegister)
	}

	got, _, provider, errPick := manager.pickNextMixed(context.Background(), []string{"gemini", "claude"}, "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickNextMixed() error = %v", errPick)
	}
	if got == nil {
		t.Fatalf("pickNextMixed() auth = nil")
	}
	if provider != "claude" {
		t.Fatalf("pickNextMixed() provider = %q, want %q", provider, "claude")
	}
	if got.ID != "claude-a" {
		t.Fatalf("pickNextMixed() auth.ID = %q, want %q", got.ID, "claude-a")
	}
}

func TestManager_SchedulerTracksMarkResultCooldownAndRecovery(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient("auth-a", "gemini", []*registry.ModelInfo{{ID: "test-model"}})
	reg.RegisterClient("auth-b", "gemini", []*registry.ModelInfo{{ID: "test-model"}})
	t.Cleanup(func() {
		reg.UnregisterClient("auth-a")
		reg.UnregisterClient("auth-b")
	})
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "auth-a", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(auth-a) error = %v", errRegister)
	}
	if _, errRegister := manager.Register(context.Background(), &Auth{ID: "auth-b", Provider: "gemini"}); errRegister != nil {
		t.Fatalf("Register(auth-b) error = %v", errRegister)
	}

	manager.MarkResult(context.Background(), Result{
		AuthID:   "auth-a",
		Provider: "gemini",
		Model:    "test-model",
		Success:  false,
		Error:    &Error{HTTPStatus: 429, Message: "quota"},
	})

	got, errPick := manager.scheduler.pickSingle(context.Background(), "gemini", "test-model", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("scheduler.pickSingle() after cooldown error = %v", errPick)
	}
	if got == nil || got.ID != "auth-b" {
		t.Fatalf("scheduler.pickSingle() after cooldown auth = %v, want auth-b", got)
	}

	manager.MarkResult(context.Background(), Result{
		AuthID:   "auth-a",
		Provider: "gemini",
		Model:    "test-model",
		Success:  true,
	})

	seen := make(map[string]struct{}, 2)
	for index := 0; index < 2; index++ {
		got, errPick = manager.scheduler.pickSingle(context.Background(), "gemini", "test-model", cliproxyexecutor.Options{}, nil)
		if errPick != nil {
			t.Fatalf("scheduler.pickSingle() after recovery #%d error = %v", index, errPick)
		}
		if got == nil {
			t.Fatalf("scheduler.pickSingle() after recovery #%d auth = nil", index)
		}
		seen[got.ID] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("len(seen) = %d, want %d", len(seen), 2)
	}
}

func TestSchedulerPickMixed_RetryTriedFilterPreservesSmoothWeightedDistribution(t *testing.T) {
	t.Parallel()

	authA := &Auth{ID: "auth-a", Provider: "provider-a"}
	authB := &Auth{ID: "auth-b", Provider: "provider-b"}
	authC := &Auth{ID: "auth-c", Provider: "provider-c"}
	authD := &Auth{ID: "auth-d", Provider: "provider-d"}
	auths := []*Auth{authA, authB, authC, authD}

	scheduler := newSchedulerForTest(&WeightedRoundRobinSelector{}, auths...)
	providers := []string{"provider-a", "provider-b", "provider-c", "provider-d"}

	retryCounts := make(map[string]int)
	tried := map[string]struct{}{"auth-a": {}}
	for index := 0; index < 30; index++ {
		picked, _, errPick := scheduler.pickMixed(context.Background(), providers, "", cliproxyexecutor.Options{}, tried)
		if errPick != nil {
			t.Fatalf("pickMixed(tried) error = %v", errPick)
		}
		retryCounts[picked.ID]++
	}

	for _, authID := range []string{"auth-b", "auth-c", "auth-d"} {
		if retryCounts[authID] != 10 {
			t.Fatalf("auth %q retry picks = %d, want 10 (even distribution without alphabetical bias, counts=%#v)", authID, retryCounts[authID], retryCounts)
		}
	}
}

func TestReadyViewRoundRobinPreservesSuccessorAcrossRebuild(t *testing.T) {
	t.Parallel()

	entry := func(id string) *scheduledAuth {
		return &scheduledAuth{auth: &Auth{ID: id}}
	}

	t.Run("a cooling resumes at b", func(t *testing.T) {
		original := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C")},
		}
		if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != "A" {
			t.Fatalf("first pick = %v, want A", got)
		}

		state := snapshotReadyViewCursors(original)
		rebuilt := readyView{
			flat: []*scheduledAuth{entry("B"), entry("C")},
		}
		restoreReadyViewCursors(&rebuilt, state)

		got := rebuilt.pickRoundRobin(nil)
		if got == nil || got.auth.ID != "B" {
			t.Fatalf("pick after A cooldown = %v, want B", got)
		}
	})

	t.Run("b cooling resumes at c", func(t *testing.T) {
		original := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C")},
		}
		if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != "A" {
			t.Fatalf("first pick = %v, want A", got)
		}
		if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != "B" {
			t.Fatalf("second pick = %v, want B", got)
		}

		state := snapshotReadyViewCursors(original)
		rebuilt := readyView{
			flat: []*scheduledAuth{entry("A"), entry("C")},
		}
		restoreReadyViewCursors(&rebuilt, state)

		got := rebuilt.pickRoundRobin(nil)
		if got == nil || got.auth.ID != "C" {
			t.Fatalf("pick after B cooldown = %v, want C", got)
		}
	})

	t.Run("c cooling wraps to a", func(t *testing.T) {
		original := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C")},
		}
		for _, want := range []string{"A", "B", "C"} {
			if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != want {
				t.Fatalf("pick = %v, want %s", got, want)
			}
		}

		state := snapshotReadyViewCursors(original)
		rebuilt := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B")},
		}
		restoreReadyViewCursors(&rebuilt, state)

		got := rebuilt.pickRoundRobin(nil)
		if got == nil || got.auth.ID != "A" {
			t.Fatalf("pick after C cooldown = %v, want A", got)
		}
	})

	t.Run("recovery preserves successor", func(t *testing.T) {
		original := readyView{
			flat: []*scheduledAuth{entry("B"), entry("C")},
		}
		if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != "B" {
			t.Fatalf("first pick = %v, want B", got)
		}

		state := snapshotReadyViewCursors(original)
		rebuilt := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C")},
		}
		restoreReadyViewCursors(&rebuilt, state)

		got := rebuilt.pickRoundRobin(nil)
		if got == nil || got.auth.ID != "C" {
			t.Fatalf("pick after A recovery = %v, want C", got)
		}
	})

	t.Run("retry exclusion resumes without rebuild", func(t *testing.T) {
		view := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C")},
		}
		if got := view.pickRoundRobin(nil); got == nil || got.auth.ID != "A" {
			t.Fatalf("first pick = %v, want A", got)
		}
		got := view.pickRoundRobin(func(candidate *scheduledAuth) bool {
			return candidate.auth.ID != "B"
		})
		if got == nil || got.auth.ID != "C" {
			t.Fatalf("pick after excluding B = %v, want C", got)
		}
	})

	t.Run("multiple cooldown skips to first surviving successor", func(t *testing.T) {
		original := readyView{
			flat: []*scheduledAuth{entry("A"), entry("B"), entry("C"), entry("D")},
		}
		if got := original.pickRoundRobin(nil); got == nil || got.auth.ID != "A" {
			t.Fatalf("first pick = %v, want A", got)
		}

		state := snapshotReadyViewCursors(original)
		rebuilt := readyView{
			flat: []*scheduledAuth{entry("C"), entry("D")},
		}
		restoreReadyViewCursors(&rebuilt, state)

		got := rebuilt.pickRoundRobin(nil)
		if got == nil || got.auth.ID != "C" {
			t.Fatalf("pick after A and B cooldown = %v, want C", got)
		}
	})
}

func TestScheduledSuccessorIndex_WrapsAndSkipsFilteredCandidates(t *testing.T) {
	t.Parallel()

	entries := []*scheduledAuth{
		{auth: &Auth{ID: "aaa"}},
		{auth: &Auth{ID: "ccc"}},
		{auth: &Auth{ID: "eee"}},
	}
	tests := []struct {
		name   string
		lastID string
		want   int
	}{
		{name: "no previous pick starts at head", lastID: "", want: 0},
		{name: "resumes after previous pick", lastID: "aaa", want: 1},
		{name: "resumes after filtered-out pick", lastID: "bbb", want: 1},
		{name: "wraps at the end of the ring", lastID: "eee", want: 0},
		{name: "wraps for removed trailing pick", lastID: "zzz", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scheduledSuccessorIndex(entries, tt.lastID); got != tt.want {
				t.Fatalf("scheduledSuccessorIndex(%q) = %d, want %d", tt.lastID, got, tt.want)
			}
		})
	}
	if got := scheduledSuccessorIndex(nil, "aaa"); got != 0 {
		t.Fatalf("scheduledSuccessorIndex(nil, aaa) = %d, want 0", got)
	}
}

func TestManagerRoundRobinPreservesSuccessorAcrossCooldown(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	model := "test-successor-model"
	authIDs := []string{"successor-auth-a", "successor-auth-b", "successor-auth-c"}
	registerSchedulerModels(t, "gemini", model, authIDs...)

	for _, id := range authIDs {
		if _, errRegister := manager.Register(context.Background(), &Auth{ID: id, Provider: "gemini"}); errRegister != nil {
			t.Fatalf("Register(%s) error = %v", id, errRegister)
		}
	}

	got, errPick := manager.scheduler.pickSingle(context.Background(), "gemini", model, cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle #1 error = %v", errPick)
	}
	if got == nil || got.ID != "successor-auth-a" {
		t.Fatalf("pickSingle #1 = %v, want successor-auth-a", got)
	}

	manager.MarkResult(context.Background(), Result{
		AuthID:   "successor-auth-a",
		Provider: "gemini",
		Model:    model,
		Success:  false,
		Error:    &Error{HTTPStatus: 429, Message: "rate limit"},
	})

	got, errPick = manager.scheduler.pickSingle(context.Background(), "gemini", model, cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle #2 after successor-auth-a cooldown error = %v", errPick)
	}
	if got == nil || got.ID != "successor-auth-b" {
		t.Fatalf("pickSingle #2 after successor-auth-a cooldown = %v, want successor-auth-b", got)
	}

	manager.MarkResult(context.Background(), Result{
		AuthID:   "successor-auth-b",
		Provider: "gemini",
		Model:    model,
		Success:  false,
		Error:    &Error{HTTPStatus: 429, Message: "rate limit"},
	})

	got, errPick = manager.scheduler.pickSingle(context.Background(), "gemini", model, cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle #3 after successor-auth-b cooldown error = %v", errPick)
	}
	if got == nil || got.ID != "successor-auth-c" {
		t.Fatalf("pickSingle #3 after successor-auth-b cooldown = %v, want successor-auth-c", got)
	}
}

func TestSchedulerPick_RoundRobinPreservesWebsocketSuccessorAcrossCooldown(t *testing.T) {
	t.Parallel()

	wsA := &Auth{ID: "codex-ws-a", Provider: "codex", Attributes: map[string]string{"websockets": "true"}}
	wsB := &Auth{ID: "codex-ws-b", Provider: "codex", Attributes: map[string]string{"websockets": "true"}}
	wsC := &Auth{ID: "codex-ws-c", Provider: "codex", Attributes: map[string]string{"websockets": "true"}}
	httpOnly := &Auth{ID: "codex-http", Provider: "codex"}
	scheduler := newSchedulerForTest(&RoundRobinSelector{}, httpOnly, wsA, wsB, wsC)

	ctx := cliproxyexecutor.WithDownstreamWebsocket(context.Background())
	got, errPick := scheduler.pickSingle(ctx, "codex", "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle() first error = %v", errPick)
	}
	if got == nil || got.ID != "codex-ws-a" {
		t.Fatalf("pickSingle() first = %v, want codex-ws-a", got)
	}

	wsA.Unavailable = true
	wsA.NextRetryAfter = time.Now().Add(time.Hour)
	scheduler.upsertAuth(wsA)

	got, errPick = scheduler.pickSingle(ctx, "codex", "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickSingle() after ws-a cooldown error = %v", errPick)
	}
	if got == nil || got.ID != "codex-ws-b" {
		t.Fatalf("pickSingle() after ws-a cooldown = %v, want codex-ws-b", got)
	}
}
