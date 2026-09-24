package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestPostgresStoreIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}

	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_test_%d", time.Now().UnixNano())
	store, err := NewPostgresStore(ctx, PostgresStoreConfig{
		DSN:    dsn,
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer store.Close()
	t.Cleanup(func() { dropTestSchema(t, dsn, schema) })

	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema first: %v", err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema second: %v", err)
	}

	version, err := store.SaveConfig(ctx, []byte("port: 8317\n"))
	if err != nil {
		t.Fatalf("SaveConfig first: %v", err)
	}
	if version != 1 {
		t.Fatalf("first config version = %d, want 1", version)
	}
	version, err = store.SaveConfig(ctx, []byte("port: 8318\n"))
	if err != nil {
		t.Fatalf("SaveConfig second: %v", err)
	}
	if version != 2 {
		t.Fatalf("second config version = %d, want 2", version)
	}
	snapshot, err := store.LoadConfig(ctx)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if string(snapshot.Content) != "port: 8318\n" || snapshot.Version != 2 {
		t.Fatalf("config snapshot = %q v%d, want port 8318 v2", snapshot.Content, snapshot.Version)
	}

	auth := &cliproxyauth.Auth{
		ID:       "gemini-user.json",
		Provider: "gemini",
		Metadata: map[string]any{
			"type":  "gemini",
			"email": "user@example.com",
		},
		Disabled:       true,
		Status:         cliproxyauth.StatusDisabled,
		StatusMessage:  "disabled in test",
		NextRetryAfter: time.Date(2026, 6, 1, 1, 2, 3, 0, time.UTC),
	}
	if _, err := store.Save(ctx, auth); err != nil {
		t.Fatalf("Save auth: %v", err)
	}
	auths, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List auth: %v", err)
	}
	if len(auths) != 1 {
		t.Fatalf("auth count = %d, want 1", len(auths))
	}
	if got := auths[0]; got.ID != auth.ID || !got.Disabled || got.Status != cliproxyauth.StatusDisabled || got.NextRetryAfter.IsZero() {
		t.Fatalf("auth = %+v, want persisted runtime fields", got)
	}
	raw, err := store.LoadAuthContent(ctx, auth.ID)
	if err != nil {
		t.Fatalf("LoadAuthContent: %v", err)
	}
	if !strings.Contains(string(raw), `"email":`) {
		t.Fatalf("raw auth content = %s, want email payload", string(raw))
	}
	if err := store.Delete(ctx, auth.ID); err != nil {
		t.Fatalf("Delete auth: %v", err)
	}
	auths, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List auth after delete: %v", err)
	}
	if len(auths) != 0 {
		t.Fatalf("auth count after delete = %d, want 0", len(auths))
	}

	if err := store.AppendUsage(ctx, []byte(`{"id":1}`), time.Date(2026, 6, 1, 0, 0, 1, 0, time.UTC)); err != nil {
		t.Fatalf("AppendUsage first: %v", err)
	}
	if err := store.AppendUsage(ctx, []byte(`{"id":2}`), time.Date(2026, 6, 1, 0, 0, 2, 0, time.UTC)); err != nil {
		t.Fatalf("AppendUsage second: %v", err)
	}
	items, err := store.PopUsage(ctx, 1)
	if err != nil {
		t.Fatalf("PopUsage first: %v", err)
	}
	// payload is stored as JSONB; Postgres normalizes whitespace/key order, so
	// popped bytes are compared semantically rather than byte-for-byte.
	if len(items) != 1 || !jsonUsageEqual(items[0], `{"id":1}`) {
		t.Fatalf("first popped usage = %q, want id 1", items)
	}
	items, err = store.PopUsage(ctx, 10)
	if err != nil {
		t.Fatalf("PopUsage second: %v", err)
	}
	if len(items) != 1 || !jsonUsageEqual(items[0], `{"id":2}`) {
		t.Fatalf("second popped usage = %q, want id 2", items)
	}
}

func jsonUsageEqual(raw []byte, want string) bool {
	var gotValue, wantValue any
	if err := json.Unmarshal(raw, &gotValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		return false
	}
	return reflect.DeepEqual(gotValue, wantValue)
}

func dropTestSchema(t *testing.T, dsn, schema string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Logf("open cleanup db: %v", err)
		return
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdentifier(schema))); err != nil {
		t.Logf("drop schema %s: %v", schema, err)
	}
}

func TestPostgresStorageRevisionsAndUsageBatch(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}

	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_test_%d", time.Now().UnixNano())
	st, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer st.Close()
	t.Cleanup(func() { dropTestSchema(t, dsn, schema) })

	if err := st.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	cfgV, aCount, _, aHash, rtRev, _, err := st.StorageRevisions(ctx)
	if err != nil {
		t.Fatalf("StorageRevisions empty: %v", err)
	}
	// runtime_control_settings is seeded with a default row (revision 1) by
	// EnsureSchema; config/auth/native revisions stay zero until written.
	if cfgV != 0 || aCount != 0 || aHash != "" || rtRev != 1 {
		t.Fatalf("empty revisions = cfg:%d auths:%d hash:%q rt:%d, want 0/0/''/1", cfgV, aCount, aHash, rtRev)
	}

	if _, err := st.SaveConfig(ctx, []byte("port: 8317\n")); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if _, err := st.Save(ctx, &cliproxyauth.Auth{ID: "a.json", Provider: "gemini", Metadata: map[string]any{"email": "a@b.c"}}); err != nil {
		t.Fatalf("Save auth: %v", err)
	}

	cfgV, aCount, _, aHash, _, _, err = st.StorageRevisions(ctx)
	if err != nil {
		t.Fatalf("StorageRevisions populated: %v", err)
	}
	if cfgV != 1 || aCount != 1 || aHash == "" {
		t.Fatalf("revisions = cfg:%d auths:%d hash:%q, want 1/1/non-empty", cfgV, aCount, aHash)
	}

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := st.AppendUsageBatch(ctx, [][]byte{[]byte(`{"id":1}`), []byte(`{"id":2}`), []byte(`{"id":3}`)},
		[]time.Time{base.Add(time.Second), base.Add(2 * time.Second), base.Add(3 * time.Second)}); err != nil {
		t.Fatalf("AppendUsageBatch: %v", err)
	}
	items, err := st.PopUsage(ctx, 10)
	if err != nil {
		t.Fatalf("PopUsage: %v", err)
	}
	if len(items) != 3 || !jsonUsageEqual(items[0], `{"id":1}`) || !jsonUsageEqual(items[2], `{"id":3}`) {
		t.Fatalf("popped batch = %q, want ordered ids 1,2,3", items)
	}

	if err := st.AppendUsageBatch(ctx, [][]byte{[]byte(`{"id":1}`)}, []time.Time{base, base}); err == nil {
		t.Fatalf("AppendUsageBatch length mismatch error = nil, want non-nil")
	}
}

// TestPostgresRevisionKeyedCaches proves the revision-keyed read caches:
// a write through a second store instance stays invisible to the first until
// ObserveRevisions delivers the new revisions (watcher probe path), and a
// local write is visible immediately (write-through).
func TestPostgresRevisionKeyedCaches(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_test_%d", time.Now().UnixNano())
	st1, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore st1: %v", err)
	}
	defer st1.Close()
	t.Cleanup(func() { dropTestSchema(t, dsn, schema) })
	if err := st1.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	st2, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore st2: %v", err)
	}
	defer st2.Close()

	// --- runtime settings ---
	settings, err := st1.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("st1 LoadRuntimeSettings: %v", err)
	}
	settings.CooldownPersistenceEnabled = !settings.CooldownPersistenceEnabled
	saved, err := st2.SaveRuntimeSettings(ctx, settings.Revision, settings)
	if err != nil {
		t.Fatalf("st2 SaveRuntimeSettings: %v", err)
	}
	stale, err := st1.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("st1 cached LoadRuntimeSettings: %v", err)
	}
	if stale.CooldownPersistenceEnabled != !saved.CooldownPersistenceEnabled {
		t.Fatalf("st1 served uncached value %v, want stale cached %v", stale.CooldownPersistenceEnabled, !saved.CooldownPersistenceEnabled)
	}
	cfgV, _, _, _, rtRev, _, err := st1.StorageRevisions(ctx)
	if err != nil {
		t.Fatalf("StorageRevisions: %v", err)
	}
	st1.ObserveRevisions(cfgV, rtRev)
	fresh, err := st1.LoadRuntimeSettings(ctx)
	if err != nil {
		t.Fatalf("st1 post-invalidation LoadRuntimeSettings: %v", err)
	}
	if fresh.CooldownPersistenceEnabled != saved.CooldownPersistenceEnabled {
		t.Fatalf("st1 fresh value %v, want %v", fresh.CooldownPersistenceEnabled, saved.CooldownPersistenceEnabled)
	}

	// --- config bytes ---
	if _, err := st2.SaveConfig(ctx, []byte("port: 9999\n")); err != nil {
		t.Fatalf("st2 SaveConfig: %v", err)
	}
	got, err := st1.LoadConfigBytes(ctx)
	if err != nil {
		t.Fatalf("st1 LoadConfigBytes: %v", err)
	}
	if !strings.Contains(string(got), "9999") {
		t.Fatalf("st1 first LoadConfigBytes = %q, want port 9999", got)
	}
	if _, err := st2.SaveConfig(ctx, []byte("port: 7777\n")); err != nil {
		t.Fatalf("st2 SaveConfig 2: %v", err)
	}
	got, err = st1.LoadConfigBytes(ctx)
	if err != nil {
		t.Fatalf("st1 cached LoadConfigBytes: %v", err)
	}
	if !strings.Contains(string(got), "9999") {
		t.Fatalf("st1 cached config = %q, want stale 9999", got)
	}
	cfgV, _, _, _, rtRev, _, _ = st1.StorageRevisions(ctx)
	st1.ObserveRevisions(cfgV, rtRev)
	got, err = st1.LoadConfigBytes(ctx)
	if err != nil {
		t.Fatalf("st1 post-invalidation LoadConfigBytes: %v", err)
	}
	if !strings.Contains(string(got), "7777") {
		t.Fatalf("st1 fresh config = %q, want 7777", got)
	}
}
