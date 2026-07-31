package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
)

func TestPostgresStoreRuntimeSettingsSchemaAndDefaults(t *testing.T) {
	ctx, store, _, schema := newPostgresRuntimeControlTestStore(t)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() second call error = %v", err)
	}

	var tableCount int
	if err := store.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_name = $2
	`, schema, runtimeControlSettingsTable).Scan(&tableCount); err != nil {
		t.Fatalf("count runtime control settings table: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("runtime control settings table count = %d, want 1", tableCount)
	}

	settings, err := store.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("LoadRuntimeSettings() error = %v", err)
	}
	if settings.Revision != 1 || settings.CredentialRouting.Strategy != runtimecontrol.RoutingRoundRobin {
		t.Fatalf("seeded settings = %#v", settings)
	}
	if settings.Home.Enabled || settings.CodexLive.Enabled || settings.CooldownPersistenceEnabled {
		t.Fatalf("seeded settings unexpectedly enable runtime behavior: %#v", settings)
	}

	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET settings = '[]'::jsonb WHERE id = $1
	`, store.fullTableName(runtimeControlSettingsTable)), runtimeControlSettingsID); err == nil {
		t.Fatal("non-object runtime settings bypassed database constraint")
	}
}

func TestPostgresRuntimeSettingsRevisionAndRestart(t *testing.T) {
	ctx, store, dsn, schema := newPostgresRuntimeControlTestStore(t)
	settings, err := store.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("LoadRuntimeSettings() error = %v", err)
	}
	settings.CredentialRouting.Strategy = runtimecontrol.RoutingWeightedRoundRobin
	settings.CredentialRouting.Weights = []runtimecontrol.CredentialWeight{{AuthID: "auth-1", Provider: "codex", Weight: 5}}
	settings.Cloaking.DisableClaudeModelList = true
	settings.Cloaking.DisableCodex = true
	settings.CodexLive = runtimecontrol.CodexLiveSettings{
		Enabled:     true,
		MaxSessions: 2,
		PublicIP:    "127.0.0.1",
		UDPPortMin:  5000,
		UDPPortMax:  5003,
		ICEServers: []runtimecontrol.ICEServer{{
			URLs:     []string{"turn:relay.example.com:3478"},
			Username: "relay-user",
		}},
	}
	settings.Home.DisableClusterDiscovery = true
	settings.CooldownPersistenceEnabled = true

	saved, err := store.SaveRuntimeSettings(ctx, settings.Revision, settings)
	if err != nil {
		t.Fatalf("SaveRuntimeSettings() error = %v", err)
	}
	if saved.Revision != 2 {
		t.Fatalf("saved revision = %d, want 2", saved.Revision)
	}
	if _, err = store.SaveRuntimeSettings(ctx, 1, settings); !errors.Is(err, runtimecontrol.ErrRevisionConflict) {
		t.Fatalf("stale SaveRuntimeSettings() error = %v, want revision conflict", err)
	}

	restarted, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(restart) error = %v", err)
	}
	defer restarted.Close()
	loaded, err := restarted.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("LoadRuntimeSettings(restart) error = %v", err)
	}
	if loaded.Revision != saved.Revision || loaded.CredentialRouting.Strategy != runtimecontrol.RoutingWeightedRoundRobin || len(loaded.CredentialRouting.Weights) != 1 {
		t.Fatalf("restarted settings = %#v", loaded)
	}
	if !loaded.CooldownPersistenceEnabled || !loaded.Cloaking.DisableCodex || !loaded.Cloaking.DisableClaudeModelList || !loaded.CodexLive.Enabled || !loaded.Home.DisableClusterDiscovery {
		t.Fatalf("restarted controls = %#v", loaded)
	}
	if got := loaded.CodexLive.ICEServers[0].Username; got != "relay-user" {
		t.Fatalf("restarted ICE username = %q", got)
	}
}

func TestPostgresRuntimeSettingsConcurrentRevision(t *testing.T) {
	ctx, store, _, _ := newPostgresRuntimeControlTestStore(t)
	settings, err := store.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("LoadRuntimeSettings() error = %v", err)
	}

	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var workers sync.WaitGroup
	for i := range 2 {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			candidate := settings.Clone()
			candidate.Cloaking.DisableCodex = index == 0
			<-start
			_, saveErr := store.SaveRuntimeSettings(ctx, settings.Revision, candidate)
			errorsCh <- saveErr
		}(i)
	}
	close(start)
	workers.Wait()
	close(errorsCh)

	var successes, conflicts int
	for saveErr := range errorsCh {
		switch {
		case saveErr == nil:
			successes++
		case errors.Is(saveErr, runtimecontrol.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("concurrent SaveRuntimeSettings() error = %v", saveErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent saves = %d success, %d conflicts; want 1 and 1", successes, conflicts)
	}
	loaded, err := store.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("LoadRuntimeSettings() after concurrent save error = %v", err)
	}
	if loaded.Revision != 2 {
		t.Fatalf("revision after concurrent save = %d, want 2", loaded.Revision)
	}
}

func newPostgresRuntimeControlTestStore(t *testing.T) (context.Context, *PostgresStore, string, string) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_runtime_control_test_%d", time.Now().UnixNano())
	store, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		dropTestSchema(t, dsn, schema)
	})
	if err = store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	return ctx, store, dsn, schema
}
