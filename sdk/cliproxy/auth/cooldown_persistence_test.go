package auth

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/registry"
)

func TestManagerCooldownStatePersistsAndRestoresRestart(t *testing.T) {
	ctx := context.Background()
	store := &memoryCooldownStateStore{}
	authID := "cooldown-restart-auth"
	model := "cooldown-restart-model"

	manager := NewManager(nil, nil, nil)
	manager.SetCooldownStateStore(store)
	if _, err := manager.Register(ctx, &Auth{ID: authID, Provider: "gemini", Metadata: map[string]any{"type": "gemini"}}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	manager.MarkResult(ctx, Result{
		AuthID:   authID,
		Provider: "gemini",
		Model:    model,
		Success:  false,
		Error:    &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota exhausted"},
	})

	// Upstream snapshot shape (ed70aeaa1627): when every model is cooling, the
	// auth aggregate is also unavailable, so the store carries one auth-level
	// record plus the model-level record.
	snap := store.snapshot()
	if len(snap) != 2 {
		t.Fatalf("persisted cooldown snapshot = %#v, want auth + model records", snap)
	}
	var authRecord, modelRecord *CooldownStateRecord
	for i := range snap {
		if snap[i].Model == "" {
			authRecord = &snap[i]
		} else {
			modelRecord = &snap[i]
		}
	}
	if authRecord == nil || authRecord.AuthID != authID || !authRecord.Quota.Exceeded || authRecord.NextRetryAfter.IsZero() {
		t.Fatalf("persisted auth-level record = %#v, want active quota cooldown", authRecord)
	}
	if modelRecord == nil || modelRecord.AuthID != authID || modelRecord.Model != model || !modelRecord.Quota.Exceeded || modelRecord.NextRetryAfter.IsZero() {
		t.Fatalf("persisted model record = %#v, want quota model record", modelRecord)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(authID, "gemini", []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() { reg.UnregisterClient(authID) })

	restarted := NewManager(nil, nil, nil)
	restarted.SetCooldownStateStore(store)
	if _, err := restarted.Register(ctx, &Auth{ID: authID, Provider: "gemini", Metadata: map[string]any{"type": "gemini"}}); err != nil {
		t.Fatalf("Register(restarted) error = %v", err)
	}
	restarted.RefreshSchedulerEntry(authID)

	restored, err := restarted.RestoreCooldownState(ctx)
	if err != nil {
		t.Fatalf("RestoreCooldownState() error = %v", err)
	}
	if restored != 2 {
		t.Fatalf("RestoreCooldownState() restored = %d, want 2", restored)
	}

	updated, ok := restarted.GetByID(authID)
	if !ok || updated == nil {
		t.Fatalf("restored auth missing")
	}
	state := updated.ModelStates[model]
	if state == nil || !state.Unavailable || !state.NextRetryAfter.After(time.Now()) || !state.Quota.Exceeded {
		t.Fatalf("restored model state = %#v, want active cooldown", state)
	}
	if _, errAvailable := getAvailableAuths(restarted.List(), "gemini", model, time.Now()); errAvailable == nil {
		t.Fatalf("getAvailableAuths() error = nil, want restored cooldown to suppress credential")
	}
}

func TestManagerCooldownStateSuccessClearsSnapshot(t *testing.T) {
	ctx := context.Background()
	store := &memoryCooldownStateStore{}
	authID := "cooldown-clear-auth"
	model := "cooldown-clear-model"

	manager := NewManager(nil, nil, nil)
	manager.SetCooldownStateStore(store)
	if _, err := manager.Register(ctx, &Auth{ID: authID, Provider: "gemini", Metadata: map[string]any{"type": "gemini"}}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	manager.MarkResult(ctx, Result{AuthID: authID, Provider: "gemini", Model: model, Success: false, Error: &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota exhausted"}})
	// Auth-level + model-level records once the whole credential is unavailable.
	if got := store.snapshot(); len(got) != 2 {
		t.Fatalf("snapshot after failure len = %d, want 2", len(got))
	}

	manager.MarkResult(ctx, Result{AuthID: authID, Provider: "gemini", Model: model, Success: true})
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("snapshot after success = %#v, want cleared", got)
	}
}

func TestManagerCooldownStateRestoreSkipsExpiredRecords(t *testing.T) {
	ctx := context.Background()
	store := &memoryCooldownStateStore{records: []CooldownStateRecord{{
		AuthID:         "expired-auth",
		Provider:       "gemini",
		Model:          "expired-model",
		Status:         string(StatusError),
		NextRetryAfter: time.Now().Add(-time.Minute),
		UpdatedAt:      time.Now().Add(-time.Hour),
	}}}
	manager := NewManager(nil, nil, nil)
	manager.SetCooldownStateStore(store)
	if _, err := manager.Register(ctx, &Auth{ID: "expired-auth", Provider: "gemini", Metadata: map[string]any{"type": "gemini"}}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	restored, err := manager.RestoreCooldownState(ctx)
	if err != nil {
		t.Fatalf("RestoreCooldownState() error = %v", err)
	}
	if restored != 0 {
		t.Fatalf("RestoreCooldownState() restored = %d, want 0", restored)
	}
	updated, _ := manager.GetByID("expired-auth")
	if len(updated.ModelStates) != 0 || updated.Unavailable {
		t.Fatalf("expired restore mutated auth = %#v", updated)
	}
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("snapshot after expired restore = %#v, want stale record pruned", got)
	}
}

func TestManagerCooldownStateSkipsRequestScopedAndNeutralResults(t *testing.T) {
	ctx := context.Background()
	store := &memoryCooldownStateStore{}
	manager := NewManager(nil, nil, nil)
	manager.SetCooldownStateStore(store)
	if _, err := manager.Register(ctx, &Auth{ID: "neutral-auth", Provider: "gemini", Metadata: map[string]any{"type": "gemini"}}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	manager.MarkResult(ctx, Result{
		AuthID:        "neutral-auth",
		Provider:      "gemini",
		Model:         "neutral-model",
		Success:       false,
		RequestScoped: true,
		Error:         &Error{HTTPStatus: http.StatusTooManyRequests, Message: "request too large"},
	})
	manager.recordAvailabilityNeutralResult(ctx, Result{
		AuthID:   "neutral-auth",
		Provider: "gemini",
		Model:    "neutral-model",
		Success:  false,
		Error:    &Error{HTTPStatus: http.StatusNotFound, Message: "CountTokens not implemented"},
	})

	if store.saveCount() != 0 {
		t.Fatalf("cooldown Save calls = %d, want 0", store.saveCount())
	}
	updated, _ := manager.GetByID("neutral-auth")
	if len(updated.ModelStates) != 0 || updated.Unavailable || !updated.NextRetryAfter.IsZero() {
		t.Fatalf("neutral/request-scoped result mutated availability = %#v", updated)
	}
}

type memoryCooldownStateStore struct {
	mu      sync.Mutex
	records []CooldownStateRecord
	saves   int
}

func (s *memoryCooldownStateStore) Load(context.Context) ([]CooldownStateRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CooldownStateRecord, len(s.records))
	copy(out, s.records)
	return out, nil
}

func (s *memoryCooldownStateStore) Save(_ context.Context, records []CooldownStateRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make([]CooldownStateRecord, len(records))
	copy(s.records, records)
	s.saves++
	return nil
}

func (s *memoryCooldownStateStore) snapshot() []CooldownStateRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CooldownStateRecord, len(s.records))
	copy(out, s.records)
	return out
}

func (s *memoryCooldownStateStore) saveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saves
}
