package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Tests in this file pin the per-model quota fairness semantics ported from
// upstream CLIProxyAPI commit 09471dd9daba ("fix(auth): prevent individual model
// quota cooldowns from blocking credential"). Local symbols under test:
// isAuthBlockedForModel, updateAggregatedAvailability,
// availableAuthsForRouteModelWithPriorityMode.
//
// The aggregated auth.Quota produced by updateAggregatedAvailability is a summary
// of single-model quota cooldowns (reason "quota"); it must not block the whole
// credential while at least one model remains available.

func TestManagerExecute_ModelAliasRequestNotBlockedByOtherModelQuotaCooldown(t *testing.T) {
	const (
		provider     = "antigravity"
		requestModel = "gemini-3.6-flash"
		targetModel  = "gemini-3.6-flash-high"
		imageModel   = "gemini-3.1-flash-image"
	)

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	executor := &aliasRoutingExecutor{id: provider}
	manager.RegisterExecutor(executor)
	manager.SetOAuthModelAlias(map[string][]internalconfig.OAuthModelAlias{
		provider: {{
			Name:  targetModel,
			Alias: requestModel,
			Fork:  true,
		}},
	})

	now := time.Now()
	next := now.Add(1 * time.Hour)

	auth := &Auth{
		ID:       "antigravity-auth-1",
		Provider: provider,
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			targetModel: {
				Status: StatusActive,
			},
			imageModel: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
				Quota: QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: next,
				},
			},
		},
	}
	updateAggregatedAvailability(auth, now)
	if !auth.Quota.Exceeded {
		t.Fatalf("precondition failed: auth.Quota.Exceeded should be true after updateAggregatedAvailability")
	}
	if auth.Unavailable {
		t.Fatalf("precondition failed: auth.Unavailable should be false since targetModel is active")
	}

	if _, errRegister := manager.Register(WithSkipPersist(context.Background()), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{
		{ID: requestModel},
		{ID: targetModel},
		{ID: imageModel},
	})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})
	manager.RefreshSchedulerEntry(auth.ID)

	resp, errExecute := manager.Execute(
		context.Background(),
		[]string{provider},
		cliproxyexecutor.Request{Model: requestModel},
		cliproxyexecutor.Options{},
	)
	if errExecute != nil {
		t.Fatalf("Execute() error = %v, want success", errExecute)
	}
	if string(resp.Payload) != targetModel {
		t.Fatalf("Execute() payload = %q, want %q", string(resp.Payload), targetModel)
	}
}

func TestManagerPickNext_ModelAliasRequestNotBlockedByOtherModelQuotaCooldown(t *testing.T) {
	const (
		provider     = "antigravity"
		requestModel = "gemini-3.6-flash"
		targetModel  = "gemini-3.6-flash-high"
		imageModel   = "gemini-3.1-flash-image"
	)

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	executor := &aliasRoutingExecutor{id: provider}
	manager.RegisterExecutor(executor)
	manager.SetOAuthModelAlias(map[string][]internalconfig.OAuthModelAlias{
		provider: {{
			Name:  targetModel,
			Alias: requestModel,
			Fork:  true,
		}},
	})

	now := time.Now()
	next := now.Add(1 * time.Hour)

	auth := &Auth{
		ID:       "antigravity-auth-2",
		Provider: provider,
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			targetModel: {
				Status: StatusActive,
			},
			imageModel: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
				Quota: QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: next,
				},
			},
		},
	}
	updateAggregatedAvailability(auth, now)
	if !auth.Quota.Exceeded {
		t.Fatalf("precondition failed: auth.Quota.Exceeded should be true after updateAggregatedAvailability")
	}
	if auth.Unavailable {
		t.Fatalf("precondition failed: auth.Unavailable should be false since targetModel is active")
	}

	if _, errRegister := manager.Register(WithSkipPersist(context.Background()), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{
		{ID: requestModel},
		{ID: targetModel},
		{ID: imageModel},
	})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})
	manager.RefreshSchedulerEntry(auth.ID)

	selected, _, errPick := manager.pickNext(
		context.Background(),
		provider,
		requestModel,
		cliproxyexecutor.Options{},
		nil,
	)
	if errPick != nil {
		t.Fatalf("pickNext() error = %v, want success", errPick)
	}
	if selected == nil || selected.ID != auth.ID {
		t.Fatalf("pickNext() selected = %#v, want %s", selected, auth.ID)
	}
}

func TestIsAuthBlockedForModel_EmptyModelIgnoresAggregatedModelQuota(t *testing.T) {
	// The credential-level availability check (empty model) must not treat the
	// aggregate of single-model quota cooldowns as a credential-wide block while
	// the credential itself is not unavailable.
	now := time.Now()
	next := now.Add(1 * time.Hour)
	auth := &Auth{
		ID:       "aggregate-quota-auth",
		Provider: "antigravity",
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			"gemini-3.6-flash-high": {
				Status: StatusActive,
			},
			"gemini-3.1-flash-image": {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
				Quota: QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: next,
				},
			},
		},
	}
	updateAggregatedAvailability(auth, now)
	if !auth.Quota.Exceeded || auth.Unavailable {
		t.Fatalf("precondition failed: want aggregate quota with available credential, got quota=%v unavailable=%v", auth.Quota.Exceeded, auth.Unavailable)
	}

	blocked, reason, _ := isAuthBlockedForModel(auth, "", now)
	if blocked {
		t.Fatalf("isAuthBlockedForModel(auth, \"\") blocked = %v reason = %v, want not blocked by aggregate single-model quota", blocked, reason)
	}
}

func TestManagerExecute_ModelAliasRequestBlockedWhenTargetModelInQuotaCooldown(t *testing.T) {
	const (
		provider     = "antigravity"
		requestModel = "gemini-3.6-flash"
		targetModel  = "gemini-3.6-flash-high"
	)

	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	executor := &aliasRoutingExecutor{id: provider}
	manager.RegisterExecutor(executor)
	manager.SetOAuthModelAlias(map[string][]internalconfig.OAuthModelAlias{
		provider: {{
			Name:  targetModel,
			Alias: requestModel,
			Fork:  true,
		}},
	})

	now := time.Now()
	next := now.Add(1 * time.Hour)

	auth := &Auth{
		ID:       "antigravity-auth-3",
		Provider: provider,
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			targetModel: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
				Quota: QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: next,
				},
			},
		},
	}
	updateAggregatedAvailability(auth, now)

	if _, errRegister := manager.Register(WithSkipPersist(context.Background()), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{
		{ID: requestModel},
		{ID: targetModel},
	})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})
	manager.RefreshSchedulerEntry(auth.ID)

	_, errExecute := manager.Execute(
		context.Background(),
		[]string{provider},
		cliproxyexecutor.Request{Model: requestModel},
		cliproxyexecutor.Options{},
	)
	if errExecute == nil {
		t.Fatal("Execute() error = nil, want cooldown error")
	}
	var cooldownErr *modelCooldownError
	if !errors.As(errExecute, &cooldownErr) {
		t.Fatalf("Execute() error = %T (%v), want *modelCooldownError", errExecute, errExecute)
	}
	if cooldownErr.model != requestModel {
		t.Fatalf("cooldown model = %q, want %q", cooldownErr.model, requestModel)
	}
}
