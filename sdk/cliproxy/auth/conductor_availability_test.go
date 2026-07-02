package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
)

func TestUpdateAggregatedAvailability_UnavailableWithoutNextRetryDoesNotBlockAuth(t *testing.T) {
	t.Parallel()

	now := time.Now()
	model := "test-model"
	auth := &Auth{
		ID: "a",
		ModelStates: map[string]*ModelState{
			model: {
				Status:      StatusError,
				Unavailable: true,
			},
		},
	}

	updateAggregatedAvailability(auth, now)

	if auth.Unavailable {
		t.Fatalf("auth.Unavailable = true, want false")
	}
	if !auth.NextRetryAfter.IsZero() {
		t.Fatalf("auth.NextRetryAfter = %v, want zero", auth.NextRetryAfter)
	}
}

func TestUpdateAggregatedAvailability_FutureNextRetryBlocksAuth(t *testing.T) {
	t.Parallel()

	now := time.Now()
	model := "test-model"
	next := now.Add(5 * time.Minute)
	auth := &Auth{
		ID: "a",
		ModelStates: map[string]*ModelState{
			model: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
			},
		},
	}

	updateAggregatedAvailability(auth, now)

	if !auth.Unavailable {
		t.Fatalf("auth.Unavailable = false, want true")
	}
	if auth.NextRetryAfter.IsZero() {
		t.Fatalf("auth.NextRetryAfter = zero, want %v", next)
	}
	if auth.NextRetryAfter.Sub(next) > time.Second || next.Sub(auth.NextRetryAfter) > time.Second {
		t.Fatalf("auth.NextRetryAfter = %v, want %v", auth.NextRetryAfter, next)
	}
}

func TestManager_ResetQuotaClearsRuntimeAndRegistryState(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	ctx := context.Background()
	authID := "reset-quota-auth"
	model := "reset-quota-model"
	next := time.Now().Add(time.Hour)

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(authID, "claude", []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() {
		reg.UnregisterClient(authID)
	})

	if _, errRegister := manager.Register(ctx, &Auth{
		ID:             authID,
		Provider:       "claude",
		Status:         StatusError,
		StatusMessage:  "quota exhausted",
		Unavailable:    true,
		NextRetryAfter: next,
		Quota:          QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, BackoffLevel: 2},
		ModelStates: map[string]*ModelState{
			model: {
				Status:         StatusError,
				StatusMessage:  "quota exhausted",
				Unavailable:    true,
				NextRetryAfter: next,
				Quota:          QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, BackoffLevel: 2},
				UpdatedAt:      next,
			},
		},
	}); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	reg.SetModelQuotaExceeded(authID, model)
	reg.SuspendClientModel(authID, model, "quota")
	if count := reg.GetModelCount(model); count != 0 {
		t.Fatalf("registry model count before reset = %d, want 0", count)
	}

	updated, models, errReset := manager.ResetQuota(ctx, authID)
	if errReset != nil {
		t.Fatalf("ResetQuota() error = %v", errReset)
	}
	if updated == nil {
		t.Fatalf("ResetQuota() updated auth is nil")
	}
	if len(models) != 1 || models[0] != model {
		t.Fatalf("ResetQuota() models = %v, want [%s]", models, model)
	}
	if updated.Status != StatusActive || updated.StatusMessage != "" || updated.Unavailable || !updated.NextRetryAfter.IsZero() {
		t.Fatalf("updated auth state = status %q message %q unavailable %v next %v", updated.Status, updated.StatusMessage, updated.Unavailable, updated.NextRetryAfter)
	}
	if updated.Quota.Exceeded || updated.Quota.Reason != "" || !updated.Quota.NextRecoverAt.IsZero() || updated.Quota.BackoffLevel != 0 {
		t.Fatalf("updated auth quota = %+v, want cleared", updated.Quota)
	}
	state := updated.ModelStates[model]
	if state == nil {
		t.Fatalf("updated model state missing")
	}
	if state.Status != StatusActive || state.StatusMessage != "" || state.Unavailable || !state.NextRetryAfter.IsZero() {
		t.Fatalf("updated model state = status %q message %q unavailable %v next %v", state.Status, state.StatusMessage, state.Unavailable, state.NextRetryAfter)
	}
	if state.Quota.Exceeded || state.Quota.Reason != "" || !state.Quota.NextRecoverAt.IsZero() || state.Quota.BackoffLevel != 0 {
		t.Fatalf("updated model quota = %+v, want cleared", state.Quota)
	}
	if count := reg.GetModelCount(model); count != 1 {
		t.Fatalf("registry model count after reset = %d, want 1", count)
	}
}

func TestManager_ModelCooldownPersistsThroughStore(t *testing.T) {
	ctx := context.Background()
	store := &captureStore{}
	manager := NewManager(store, nil, nil)
	authID := "cooldown-persist-auth"
	model := "cooldown-persist-model"

	if _, errRegister := manager.Register(ctx, &Auth{
		ID:       authID,
		Provider: "gemini",
		Metadata: map[string]any{
			"type": "gemini",
		},
	}); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	manager.MarkResult(ctx, Result{
		AuthID:   authID,
		Provider: "gemini",
		Model:    model,
		Success:  false,
		Error:    &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota exhausted"},
	})

	updated, ok := store.Get(authID)
	if !ok {
		t.Fatalf("expected store to capture auth %q", authID)
	}
	state := updated.ModelStates[model]
	if state == nil {
		t.Fatalf("expected persisted model state for %q", model)
	}
	if !state.Quota.Exceeded || state.Quota.Reason != "quota" || state.Quota.BackoffLevel == 0 || state.NextRetryAfter.IsZero() {
		t.Fatalf("persisted state quota = %+v next=%v, want quota cooldown", state.Quota, state.NextRetryAfter)
	}
	if !updated.Quota.Exceeded || updated.Quota.Reason != "quota" {
		t.Fatalf("persisted auth quota = %+v, want aggregate quota", updated.Quota)
	}
}

type captureStore struct {
	auths map[string]*Auth
}

func (s *captureStore) List(context.Context) ([]*Auth, error) {
	if s == nil || len(s.auths) == 0 {
		return nil, nil
	}
	out := make([]*Auth, 0, len(s.auths))
	for _, auth := range s.auths {
		out = append(out, auth.Clone())
	}
	return out, nil
}

func (s *captureStore) Save(_ context.Context, auth *Auth) (string, error) {
	if s.auths == nil {
		s.auths = make(map[string]*Auth)
	}
	s.auths[auth.ID] = auth.Clone()
	return auth.ID, nil
}

func (s *captureStore) Delete(_ context.Context, id string) error {
	delete(s.auths, id)
	return nil
}

func (s *captureStore) Get(id string) (*Auth, bool) {
	if s == nil {
		return nil, false
	}
	auth, ok := s.auths[id]
	if !ok {
		return nil, false
	}
	return auth.Clone(), true
}
