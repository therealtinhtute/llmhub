package store

import (
	"context"
	"testing"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestPostgresCooldownRestartExpiryAndStaleWrite(t *testing.T) {
	ctx, store, dsn, schema := newPostgresRuntimeControlTestStore(t)
	cooldowns := store.CooldownStateStore()
	if _, err := cooldowns.Load(ctx); err != nil {
		t.Fatalf("initial Load() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	active := cliproxyauth.CooldownStateRecord{
		Provider:       "codex",
		AuthID:         "auth-active",
		Model:          "gpt-test",
		NextRetryAfter: now.Add(time.Hour),
		UpdatedAt:      now,
	}
	expired := cliproxyauth.CooldownStateRecord{
		Provider:       "claude",
		AuthID:         "auth-expired",
		NextRetryAfter: now.Add(-time.Minute),
		UpdatedAt:      now,
	}
	if err := cooldowns.Save(ctx, []cliproxyauth.CooldownStateRecord{active, expired}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	restarted, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(restart) error = %v", err)
	}
	defer restarted.Close()
	restartedCooldowns := restarted.CooldownStateStore()
	loaded, err := restartedCooldowns.Load(ctx)
	if err != nil {
		t.Fatalf("Load(restart) error = %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load(restart) returned %d records, want 2", len(loaded))
	}
	current := make([]cliproxyauth.CooldownStateRecord, 0, len(loaded))
	for _, record := range loaded {
		if !record.Expired(now) {
			current = append(current, record)
		}
	}
	if err = restartedCooldowns.Save(ctx, current); err != nil {
		t.Fatalf("Save(non-expired snapshot) error = %v", err)
	}

	stale, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(stale) error = %v", err)
	}
	defer stale.Close()
	staleCooldowns := stale.CooldownStateStore()
	staleRecords, err := staleCooldowns.Load(ctx)
	if err != nil {
		t.Fatalf("stale Load() error = %v", err)
	}
	if len(staleRecords) != 1 || staleRecords[0].AuthID != active.AuthID {
		t.Fatalf("stale Load() = %#v, want only active record", staleRecords)
	}

	newer := active
	newer.UpdatedAt = now.Add(time.Minute)
	newer.NextRetryAfter = now.Add(2 * time.Hour)
	if err = restartedCooldowns.Save(ctx, []cliproxyauth.CooldownStateRecord{newer}); err != nil {
		t.Fatalf("Save(newer) error = %v", err)
	}
	if err = staleCooldowns.Save(ctx, staleRecords); err != nil {
		t.Fatalf("Save(stale) error = %v", err)
	}
	if err = staleCooldowns.Save(context.Background(), nil); err != nil {
		t.Fatalf("Save(stale delete) error = %v", err)
	}

	verifier, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(verifier) error = %v", err)
	}
	defer verifier.Close()
	finalRecords, err := verifier.CooldownStateStore().Load(ctx)
	if err != nil {
		t.Fatalf("Load(final) error = %v", err)
	}
	if len(finalRecords) != 1 || finalRecords[0].AuthID != active.AuthID || !finalRecords[0].UpdatedAt.Equal(newer.UpdatedAt) || !finalRecords[0].NextRetryAfter.Equal(newer.NextRetryAfter) {
		t.Fatalf("final cooldown records = %#v, want newer active record", finalRecords)
	}
}
