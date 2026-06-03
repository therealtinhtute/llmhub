package auth

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type countingStore struct {
	saveCount atomic.Int32
}

func (s *countingStore) List(context.Context) ([]*Auth, error) { return nil, nil }

func (s *countingStore) Save(context.Context, *Auth) (string, error) {
	s.saveCount.Add(1)
	return "", nil
}

func (s *countingStore) Delete(context.Context, string) error { return nil }

func TestWithSkipPersist_DisablesUpdatePersistence(t *testing.T) {
	store := &countingStore{}
	mgr := NewManager(store, nil, nil)
	auth := &Auth{
		ID:       "auth-1",
		Provider: "antigravity",
		Metadata: map[string]any{"type": "antigravity"},
	}

	if _, err := mgr.Update(context.Background(), auth); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if got := store.saveCount.Load(); got != 1 {
		t.Fatalf("expected 1 Save call, got %d", got)
	}

	ctxSkip := WithSkipPersist(context.Background())
	if _, err := mgr.Update(ctxSkip, auth); err != nil {
		t.Fatalf("Update(skipPersist) returned error: %v", err)
	}
	if got := store.saveCount.Load(); got != 1 {
		t.Fatalf("expected Save call count to remain 1, got %d", got)
	}
}

func TestWithSkipPersist_DisablesRegisterPersistence(t *testing.T) {
	store := &countingStore{}
	mgr := NewManager(store, nil, nil)
	auth := &Auth{
		ID:       "auth-1",
		Provider: "antigravity",
		Metadata: map[string]any{"type": "antigravity"},
	}

	if _, err := mgr.Register(WithSkipPersist(context.Background()), auth); err != nil {
		t.Fatalf("Register(skipPersist) returned error: %v", err)
	}
	if got := store.saveCount.Load(); got != 0 {
		t.Fatalf("expected 0 Save calls, got %d", got)
	}
}

type failingStore struct {
	err error
}

func (s *failingStore) List(context.Context) ([]*Auth, error) { return nil, nil }

func (s *failingStore) Save(context.Context, *Auth) (string, error) {
	return "", s.err
}

func (s *failingStore) Delete(context.Context, string) error { return nil }

func TestManagerRegister_ReturnsPersistenceError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	mgr := NewManager(&failingStore{err: wantErr}, nil, nil)

	_, err := mgr.Register(context.Background(), &Auth{
		ID:       "auth-1",
		Provider: "codex",
		Metadata: map[string]any{"type": "codex"},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Register error = %v, want %v", err, wantErr)
	}
}

func TestManagerUpdate_ReturnsPersistenceError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	mgr := NewManager(&failingStore{err: wantErr}, nil, nil)

	_, err := mgr.Update(context.Background(), &Auth{
		ID:       "auth-1",
		Provider: "codex",
		Metadata: map[string]any{"type": "codex"},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Update error = %v, want %v", err, wantErr)
	}
}
