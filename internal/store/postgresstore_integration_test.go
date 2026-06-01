package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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
	if len(items) != 1 || string(items[0]) != `{"id":1}` {
		t.Fatalf("first popped usage = %q, want id 1", items)
	}
	items, err = store.PopUsage(ctx, 10)
	if err != nil {
		t.Fatalf("PopUsage second: %v", err)
	}
	if len(items) != 1 || string(items[0]) != `{"id":2}` {
		t.Fatalf("second popped usage = %q, want id 2", items)
	}
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
